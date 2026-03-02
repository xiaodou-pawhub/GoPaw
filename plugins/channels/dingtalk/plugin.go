// Package dingtalk implements the DingTalk channel plugin for GoPaw.
// It receives events via the DingTalk Stream mode or HTTP callback,
// and sends messages using the DingTalk Open Platform API.
package dingtalk

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 256),
	})
}

type dingtalkConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Plugin implements the DingTalk channel.
type Plugin struct {
	cfg        dingtalkConfig
	inbound    chan *types.Message
	started    time.Time
	token      string
	configured bool // true when client_id and client_secret have been provided
	logger     *zap.Logger
}

func (p *Plugin) Name() string        { return "dingtalk" }
func (p *Plugin) DisplayName() string { return "钉钉" }

// Init parses DingTalk credentials.
// An empty or missing config is accepted — the plugin starts in unconfigured mode.
// Configure credentials via the Web UI → Settings → Channels.
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.dingtalk")
	if len(cfg) > 0 && string(cfg) != "{}" {
		if err := json.Unmarshal(cfg, &p.cfg); err != nil {
			p.logger.Warn("dingtalk: failed to parse config, running unconfigured", zap.Error(err))
			return nil
		}
	}
	if p.cfg.ClientID == "" || p.cfg.ClientSecret == "" {
		p.logger.Warn("dingtalk: client_id / client_secret not set — configure via Web UI → Settings → Channels")
		return nil
	}
	p.configured = true
	return nil
}

// Start fetches the initial access token.
func (p *Plugin) Start(_ context.Context) error {
	p.started = time.Now()
	if err := p.refreshToken(); err != nil {
		p.logger.Warn("dingtalk: initial token fetch failed", zap.Error(err))
	}
	p.logger.Info("dingtalk channel started")
	return nil
}

// Stop gracefully shuts down the plugin.
func (p *Plugin) Stop() error {
	p.logger.Info("dingtalk channel stopped")
	return nil
}

// Receive returns the inbound message channel.
func (p *Plugin) Receive() <-chan *types.Message {
	return p.inbound
}

// Send delivers a message to a DingTalk conversation.
func (p *Plugin) Send(msg *types.Message) error {
	if !p.configured {
		return fmt.Errorf("dingtalk: channel not configured — add credentials via Web UI")
	}
	payload := map[string]interface{}{
		"robotCode": p.cfg.ClientID,
		"userIds":   []string{msg.UserID},
		"msgKey":    "sampleText",
		"msgParam":  fmt.Sprintf(`{"content":%q}`, msg.Content),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost,
		"https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("dingtalk: build send request: %w", err)
	}
	req.Header.Set("x-acs-dingtalk-access-token", p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk: send http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk: send api error: %s", string(b))
	}
	return nil
}

// Health returns the current health status.
func (p *Plugin) Health() plugin.HealthStatus {
	if !p.configured {
		return plugin.HealthStatus{
			Running: false,
			Message: "not configured — add credentials via Web UI → Settings → Channels",
			Since:   p.started,
		}
	}
	return plugin.HealthStatus{
		Running: p.token != "",
		Message: "ok",
		Since:   p.started,
	}
}

// HandleWebhook processes an incoming DingTalk webhook event.
// It should be registered as POST /dingtalk/event by the server.
func (p *Plugin) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	// Validate DingTalk request signature.
	timestamp := r.Header.Get("timestamp")
	sign := r.Header.Get("sign")
	if !p.verifySign(timestamp, sign) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	msgType, _ := event["msgtype"].(string)
	if msgType != "text" {
		w.WriteHeader(http.StatusOK)
		return
	}

	textObj, _ := event["text"].(map[string]interface{})
	content, _ := textObj["content"].(string)

	senderStaff, _ := event["senderStaffId"].(string)
	conversationID, _ := event["conversationId"].(string)
	msgID, _ := event["msgId"].(string)
	if msgID == "" {
		msgID = uuid.New().String()
	}

	msg := &types.Message{
		ID:        msgID,
		SessionID: conversationID,
		UserID:    senderStaff,
		Channel:   p.Name(),
		Content:   content,
		MsgType:   types.MsgTypeText,
		Timestamp: time.Now().UnixMilli(),
	}

	select {
	case p.inbound <- msg:
	default:
		p.logger.Warn("dingtalk: inbound queue full, dropping message")
	}

	w.WriteHeader(http.StatusOK)
}

// refreshToken fetches a DingTalk access token using client credentials.
func (p *Plugin) refreshToken() error {
	payload := map[string]string{
		"clientId":     p.cfg.ClientID,
		"clientSecret": p.cfg.ClientSecret,
		"grantType":    "client_credentials",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(
		"https://api.dingtalk.com/v1.0/oauth2/accessToken",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("dingtalk token: http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("dingtalk token: decode: %w", err)
	}
	if result.AccessToken == "" {
		return fmt.Errorf("dingtalk token: empty access token in response")
	}
	p.token = result.AccessToken
	return nil
}

// verifySign validates the HMAC-SHA256 signature on a DingTalk webhook request.
func (p *Plugin) verifySign(timestamp, sign string) bool {
	if p.cfg.ClientSecret == "" {
		return true // no secret configured — skip validation
	}
	stringToSign := timestamp + "\n" + p.cfg.ClientSecret
	mac := hmac.New(sha256.New, []byte(p.cfg.ClientSecret))
	mac.Write([]byte(stringToSign))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return sign == expected
}
