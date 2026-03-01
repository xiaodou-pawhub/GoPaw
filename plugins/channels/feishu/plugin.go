// Package feishu implements the Feishu (Lark) channel plugin for GoPaw.
// It receives events via the Feishu Event Subscription mechanism (HTTP callback)
// and sends messages using the Feishu Open Platform messaging API.
package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

type feishuConfig struct {
	AppID             string `json:"app_id"`
	AppSecret         string `json:"app_secret"`
	VerificationToken string `json:"verification_token"`
	EncryptKey        string `json:"encrypt_key"`
}

// Plugin implements the Feishu channel.
type Plugin struct {
	cfg     feishuConfig
	inbound chan *types.Message
	started time.Time
	token   string // app_access_token, refreshed periodically
	logger  *zap.Logger
}

func (p *Plugin) Name() string        { return "feishu" }
func (p *Plugin) DisplayName() string { return "飞书" }

// Init parses Feishu credentials.
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.feishu")
	if err := json.Unmarshal(cfg, &p.cfg); err != nil {
		return fmt.Errorf("feishu: parse config: %w", err)
	}
	if p.cfg.AppID == "" || p.cfg.AppSecret == "" {
		return fmt.Errorf("feishu: app_id and app_secret are required")
	}
	return nil
}

// Start fetches the initial app access token.
func (p *Plugin) Start(_ context.Context) error {
	p.started = time.Now()
	if err := p.refreshToken(); err != nil {
		p.logger.Warn("feishu: initial token fetch failed, will retry", zap.Error(err))
	}
	p.logger.Info("feishu channel started")
	return nil
}

// Stop is a no-op for the Feishu channel.
func (p *Plugin) Stop() error {
	p.logger.Info("feishu channel stopped")
	return nil
}

// Receive returns the inbound message channel.
func (p *Plugin) Receive() <-chan *types.Message {
	return p.inbound
}

// Send delivers a message to a Feishu chat via the open API.
func (p *Plugin) Send(msg *types.Message) error {
	receiveID := msg.SessionID
	if receiveID == "" {
		receiveID = msg.UserID
	}

	payload := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    fmt.Sprintf(`{"text":%q}`, msg.Content),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost,
		"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("feishu: build send request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("feishu: send http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu: send api error: %s", string(b))
	}
	return nil
}

// Health returns the plugin's operational status.
func (p *Plugin) Health() plugin.HealthStatus {
	return plugin.HealthStatus{
		Running: p.token != "",
		Message: "ok",
		Since:   p.started,
	}
}

// HandleEventRequest processes a raw HTTP request from the Feishu event subscription.
// Returns the response body to write and the HTTP status code.
func (p *Plugin) HandleEventRequest(r *http.Request) (interface{}, int) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return map[string]string{"error": "read body"}, http.StatusBadRequest
	}

	var event map[string]interface{}
	if err := json.Unmarshal(body, &event); err != nil {
		return map[string]string{"error": "invalid json"}, http.StatusBadRequest
	}

	// Handle URL verification challenge.
	if challenge, ok := event["challenge"].(string); ok {
		if p.cfg.VerificationToken != "" {
			token, _ := event["token"].(string)
			if token != p.cfg.VerificationToken {
				return map[string]string{"error": "invalid token"}, http.StatusUnauthorized
			}
		}
		return map[string]string{"challenge": challenge}, http.StatusOK
	}

	// Extract message from the event envelope.
	header, _ := event["header"].(map[string]interface{})
	if header == nil {
		return nil, http.StatusOK
	}
	eventType, _ := header["event_type"].(string)
	if eventType != "im.message.receive_v1" {
		return nil, http.StatusOK
	}

	eventData, _ := event["event"].(map[string]interface{})
	if eventData == nil {
		return nil, http.StatusOK
	}

	msgData, _ := eventData["message"].(map[string]interface{})
	sender, _ := eventData["sender"].(map[string]interface{})
	if msgData == nil || sender == nil {
		return nil, http.StatusOK
	}

	contentStr, _ := msgData["content"].(string)
	var content map[string]string
	json.Unmarshal([]byte(contentStr), &content) //nolint:errcheck
	text := content["text"]

	senderID, _ := sender["sender_id"].(map[string]interface{})
	userID, _ := senderID["open_id"].(string)
	chatID, _ := msgData["chat_id"].(string)
	msgID, _ := msgData["message_id"].(string)
	if msgID == "" {
		msgID = uuid.New().String()
	}

	inMsg := &types.Message{
		ID:        msgID,
		SessionID: chatID,
		UserID:    userID,
		Channel:   p.Name(),
		Content:   text,
		MsgType:   types.MsgTypeText,
		Timestamp: time.Now().UnixMilli(),
		Metadata:  map[string]string{"chat_id": chatID},
	}

	select {
	case p.inbound <- inMsg:
	default:
		p.logger.Warn("feishu: inbound queue full, dropping message")
	}

	return nil, http.StatusOK
}

// refreshToken fetches a new app_access_token from Feishu.
func (p *Plugin) refreshToken() error {
	payload := map[string]string{
		"app_id":     p.cfg.AppID,
		"app_secret": p.cfg.AppSecret,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(
		"https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("feishu token: http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code           int    `json:"code"`
		AppAccessToken string `json:"app_access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("feishu token: decode: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("feishu token: api code %d", result.Code)
	}
	p.token = result.AppAccessToken
	return nil
}

// signFeishu computes the Feishu signature for encrypted event payloads.
// This is reserved for future encrypt_key support.
func signFeishu(timestamp, nonce, encryptKey, body string) string {
	h := hmac.New(sha256.New, []byte(encryptKey))
	h.Write([]byte(timestamp + nonce + body))
	return hex.EncodeToString(h.Sum(nil))
}
