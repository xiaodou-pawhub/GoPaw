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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// ── 常量定义 / Constants ───────────────────────────────────────────────────

const (
	defaultTimeout = 10 * time.Second
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
	logger     *zap.Logger
	httpClient *http.Client
	cancel     context.CancelFunc

	// Token 缓存与并发保护
	tokenMu     sync.RWMutex
	cachedToken string
	tokenExpiry time.Time
	configured  bool // true when client_id and client_secret have been provided
}

func (p *Plugin) Name() string        { return "dingtalk" }
func (p *Plugin) DisplayName() string { return "钉钉" }

// Init parses DingTalk credentials.
// An empty or missing config is accepted — the plugin starts in unconfigured mode.
// Configure credentials via the Web UI → Settings → Channels.
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.dingtalk")
	p.httpClient = &http.Client{Timeout: defaultTimeout}
	
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

// Start fetches the initial access token and starts the refresh loop.
func (p *Plugin) Start(ctx context.Context) error {
	p.started = time.Now()

	// 创建可取消的 context
	ctx, p.cancel = context.WithCancel(ctx)

	if p.configured {
		if err := p.refreshAndCache(ctx); err != nil {
			p.logger.Warn("dingtalk: initial token fetch failed", zap.Error(err))
		}
		// 启动后台 Token 刷新（钉钉 Token 有效期约 2 小时）
		go p.tokenRefreshLoop(ctx)
	}

	p.logger.Info("dingtalk channel started")
	return nil
}

// tokenRefreshLoop periodically refreshes the access token.
// DingTalk tokens expire after ~2 hours, so we refresh every 90 minutes.
func (p *Plugin) tokenRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(90 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.refreshAndCache(ctx); err != nil {
				p.logger.Error("dingtalk: token refresh failed", zap.Error(err))
			}
		}
	}
}

// refreshAndCache fetches a new token and updates the cache.
func (p *Plugin) refreshAndCache(ctx context.Context) error {
	payload := map[string]string{
		"clientId":     p.cfg.ClientID,
		"clientSecret": p.cfg.ClientSecret,
		"grantType":    "client_credentials",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal token payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.dingtalk.com/v1.0/oauth2/accessToken",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int64  `json:"expireIn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}
	if result.AccessToken == "" {
		return fmt.Errorf("empty access token in response")
	}

	p.tokenMu.Lock()
	p.cachedToken = result.AccessToken
	// 默认 2 小时过期，使用返回值或默认值
	expiry := time.Now().Add(2 * time.Hour)
	if result.ExpireIn > 0 {
		expiry = time.Now().Add(time.Duration(result.ExpireIn) * time.Second)
	}
	// 提前 5 分钟过期，避免临界时刻失效
	p.tokenExpiry = expiry.Add(-5 * time.Minute)
	p.tokenMu.Unlock()

	p.logger.Debug("dingtalk: token refreshed")
	return nil
}

// getToken returns the cached token, refreshing if necessary.
func (p *Plugin) getToken() (string, error) {
	p.tokenMu.RLock()
	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry) {
		token := p.cachedToken
		p.tokenMu.RUnlock()
		return token, nil
	}
	p.tokenMu.RUnlock()

	// Token 过期或不存在，需要刷新
	// 使用 background context 因为这是在请求处理中
	if err := p.refreshAndCache(context.Background()); err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}

	p.tokenMu.RLock()
	defer p.tokenMu.RUnlock()
	if p.cachedToken == "" {
		return "", fmt.Errorf("token not available")
	}
	return p.cachedToken, nil
}

// Stop gracefully shuts down the plugin.
func (p *Plugin) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
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

	token, err := p.getToken()
	if err != nil {
		return fmt.Errorf("dingtalk: get token: %w", err)
	}

	payload := map[string]interface{}{
		"robotCode": p.cfg.ClientID,
		"userIds":   []string{msg.UserID},
		"msgKey":    "sampleText",
		"msgParam":  fmt.Sprintf(`{"content":%q}`, msg.Content),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("dingtalk: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost,
		"https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("dingtalk: build send request: %w", err)
	}
	req.Header.Set("x-acs-dingtalk-access-token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk: send http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("dingtalk: send api error (status %d)", resp.StatusCode)
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
	p.tokenMu.RLock()
	hasToken := p.cachedToken != ""
	p.tokenMu.RUnlock()
	return plugin.HealthStatus{
		Running: hasToken,
		Message: "ok",
		Since:   p.started,
	}
}

// Test validates the DingTalk credentials by attempting to get an access token.
func (p *Plugin) Test(ctx context.Context) plugin.TestResult {
	if !p.configured || p.cfg.ClientID == "" || p.cfg.ClientSecret == "" {
		return plugin.TestResult{
			Success: false,
			Message: "请先配置 client_id 和 client_secret",
		}
	}

	// 尝试获取 token
	if err := p.refreshAndCache(ctx); err != nil {
		return plugin.TestResult{
			Success: false,
			Message: "凭证验证失败，请检查 client_id 和 client_secret",
			Details: err.Error(),
		}
	}

	return plugin.TestResult{
		Success: true,
		Message: "连接正常，凭证有效",
	}
}

// HandleWebhook processes an incoming DingTalk webhook event.
// It should be registered as POST /dingtalk/event by the server.
func (p *Plugin) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	p.HandleReceive(w, r, "")
}

// HandleReceive implements HTTPHandler interface for DingTalk.
// The token parameter is ignored (DingTalk uses signature validation instead).
func (p *Plugin) HandleReceive(w http.ResponseWriter, r *http.Request, _ string) {
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
		w.WriteHeader(http.StatusOK)
	default:
		// 队列满，返回 503
		p.logger.Warn("dingtalk: inbound queue full, rejecting message")
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// HandlePoll implements HTTPHandler interface (not used for DingTalk).
func (p *Plugin) HandlePoll(w http.ResponseWriter, r *http.Request, _ string) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
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
