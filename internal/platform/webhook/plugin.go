// Package webhook implements the generic Webhook channel plugin for GoPaw.
// It accepts arbitrary HTTP POST requests and can deliver responses via callback URL
// or allow the caller to poll for responses.
package webhook

import (
	"bytes"
	"context"
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

func init() {
	channel.Register(&Plugin{
		inbound:  make(chan *types.Message, 256),
		outbound: make(map[string][]*types.Message),
	})
}

type webhookConfig struct {
	Token       string `json:"token"`
	CallbackURL string `json:"callback_url"`
}

// Plugin implements the Webhook channel.
type Plugin struct {
	cfg      webhookConfig
	inbound  chan *types.Message
	started  time.Time
	logger   *zap.Logger

	// outbound stores responses for polling (keyed by token).
	mu       sync.Mutex
	outbound map[string][]*types.Message
}

func (p *Plugin) Name() string        { return "webhook" }
func (p *Plugin) DisplayName() string { return "Webhook" }

// Init parses the webhook configuration.
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.webhook")
	if err := json.Unmarshal(cfg, &p.cfg); err != nil {
		return fmt.Errorf("webhook: parse config: %w", err)
	}
	if p.cfg.Token == "" {
		return plugin.ErrMissingCredentials
	}
	return nil
}

// Start activates the webhook channel.
func (p *Plugin) Start(_ context.Context) error {
	p.started = time.Now()
	// Token 脱敏：只显示前后各 4 个字符
	maskedToken := maskToken(p.cfg.Token)
	p.logger.Info("webhook channel started", zap.String("token", maskedToken))
	return nil
}

// Stop gracefully shuts down the plugin.
func (p *Plugin) Stop() error {
	p.logger.Info("webhook channel stopped")
	return nil
}

// Receive returns the inbound message channel.
func (p *Plugin) Receive() <-chan *types.Message {
	return p.inbound
}

// Send delivers a message. If callback_url is configured, it POSTs to the URL.
// Otherwise, the message is queued for polling via the /webhook/{token}/messages endpoint.
func (p *Plugin) Send(msg *types.Message) error {
	if p.cfg.CallbackURL != "" {
		return p.pushCallback(msg)
	}
	// Store for polling.
	p.mu.Lock()
	defer p.mu.Unlock()
	p.outbound[p.cfg.Token] = append(p.outbound[p.cfg.Token], msg)
	return nil
}

// Health returns the current status.
func (p *Plugin) Health() plugin.HealthStatus {
	return plugin.HealthStatus{
		Running: true,
		Message: "ok",
		Since:   p.started,
	}
}

// Test validates the webhook configuration.
// If callback_url is configured, it sends a test request to verify reachability.
func (p *Plugin) Test(ctx context.Context) plugin.TestResult {
	// 检查 token 配置
	if p.cfg.Token == "" {
		return plugin.TestResult{
			Success: false,
			Message: "请先配置 token",
		}
	}

	// 如果配置了 callback_url，测试可达性
	if p.cfg.CallbackURL != "" {
		if err := p.testCallbackURL(ctx); err != nil {
			return plugin.TestResult{
				Success: false,
				Message: "回调地址不可达",
				Details: err.Error(),
			}
		}
		return plugin.TestResult{
			Success: true,
			Message: "配置有效，回调地址可达",
		}
	}

	// 没有配置 callback_url，使用轮询模式
	return plugin.TestResult{
		Success: true,
		Message: "配置有效（轮询模式）",
	}
}

// testCallbackURL sends a test request to the callback URL.
func (p *Plugin) testCallbackURL(ctx context.Context) error {
	testPayload := map[string]interface{}{
		"test":     "connection",
		"time":     time.Now().Format(time.RFC3339),
		"channel":  p.Name(),
	}
	body, _ := json.Marshal(testPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.CallbackURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// 接受 2xx 和 3xx 响应
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}

// HandleReceive processes POST /webhook/{token} requests.
func (p *Plugin) HandleReceive(w http.ResponseWriter, r *http.Request, token string) {
	if token != p.cfg.Token {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	var payload struct {
		UserID    string `json:"user_id"`
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		MsgType   string `json:"msg_type"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = "webhook-" + payload.UserID
	}
	msgType := types.MsgTypeText
	if payload.MsgType != "" {
		msgType = types.MessageType(payload.MsgType)
	}

	msg := &types.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		UserID:    payload.UserID,
		Channel:   p.Name(),
		Content:   payload.Content,
		MsgType:   msgType,
		Timestamp: time.Now().UnixMilli(),
	}

	select {
	case p.inbound <- msg:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": msg.ID, "status": "queued"}) //nolint:errcheck
	default:
		// 队列满，返回 503 Service Unavailable
		p.logger.Warn("webhook: inbound queue full, rejecting message")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"id": msg.ID, "status": "rejected", "error": "queue full"}) //nolint:errcheck
	}
}

// HandlePoll serves GET /webhook/{token}/messages — returns queued outbound messages.
func (p *Plugin) HandlePoll(w http.ResponseWriter, r *http.Request, token string) {
	if token != p.cfg.Token {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	p.mu.Lock()
	msgs := p.outbound[token]
	p.outbound[token] = nil
	p.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"messages": msgs}) //nolint:errcheck
}

// pushCallback sends the response to the configured callback URL.
func (p *Plugin) pushCallback(msg *types.Message) error {
	payload, _ := json.Marshal(msg)
	req, err := http.NewRequest(http.MethodPost, p.cfg.CallbackURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("webhook: build callback request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: callback http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: callback returned status %d", resp.StatusCode)
	}
	return nil
}

// maskToken returns a masked version of the token for logging.
// Shows first 4 and last 4 characters, with *** in between.
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
