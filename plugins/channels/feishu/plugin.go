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
	"strconv"
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
		inbound: make(chan *types.Message, 256),
	})
}

type feishuConfig struct {
	AppID             string `json:"app_id"`
	AppSecret         string `json:"app_secret"`
	VerificationToken string `json:"verification_token"`
	EncryptKey        string `json:"encrypt_key"`
}

// feishuErrors 定义了飞书 API 错误码及用户友好的错误信息
var feishuErrors = map[int]string{
	99991663: "app_id 或 app_secret 错误",
	99991664: "access_token 无效或已过期",
	99991665: "access_token 缺少权限",
	99991403: "请求频率超限",
	99991404: "资源不存在",
	99999999: "服务内部错误",
}

// FeishuError 封装飞书 API 错误，提供用户友好的错误信息
type FeishuError struct {
	Code       int
	Msg        string
	InternalMsg string
}

func (e *FeishuError) Error() string {
	return e.Msg
}

// newFeishuError 根据飞书返回的 code 创建错误实例
func newFeishuError(code int, internalMsg string) *FeishuError {
	msg, ok := feishuErrors[code]
	if !ok {
		msg = fmt.Sprintf("飞书 API 错误 (%d)", code)
	}
	return &FeishuError{
		Code:        code,
		Msg:         msg,
		InternalMsg: internalMsg,
	}
}

// Plugin implements the Feishu channel.
type Plugin struct {
	cfg        feishuConfig
	inbound    chan *types.Message
	started    time.Time
	configured bool   // true when app_id and app_secret have been provided
	logger     *zap.Logger
	httpClient *http.Client // 带超时的 HTTP 客户端

	// Token 缓存（替换原来的裸 token 字段）
	tokenMu      sync.RWMutex
	cachedToken  string
	tokenExpiry  time.Time
}

func (p *Plugin) Name() string        { return "feishu" }
func (p *Plugin) DisplayName() string { return "飞书" }

// Init parses Feishu credentials.
// An empty or missing config is accepted — the plugin starts in unconfigured mode
// and will log a warning. Configure credentials via the Web UI → Settings → Channels.
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.feishu")
	// 初始化带超时配置的 HTTP 客户端
	p.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	if len(cfg) > 0 && string(cfg) != "{}" {
		if err := json.Unmarshal(cfg, &p.cfg); err != nil {
			p.logger.Warn("feishu: failed to parse config, running unconfigured", zap.Error(err))
			return nil
		}
	}
	if p.cfg.AppID == "" || p.cfg.AppSecret == "" {
		p.logger.Warn("feishu: app_id / app_secret not set — configure via Web UI → Settings → Channels")
		return nil
	}
	p.configured = true
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
	if !p.configured {
		return fmt.Errorf("feishu: channel not configured — add credentials via Web UI")
	}
	receiveID := msg.SessionID
	if receiveID == "" {
		receiveID = msg.UserID
	}

	// 获取 token（自动刷新）
	token, err := p.getToken()
	if err != nil {
		return fmt.Errorf("feishu: get token: %w", err)
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
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &FeishuError{Msg: fmt.Sprintf("网络请求失败: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return newFeishuError(resp.StatusCode, string(b))
	}
	return nil
}

// Health returns the plugin's operational status.
func (p *Plugin) Health() plugin.HealthStatus {
	if !p.configured {
		return plugin.HealthStatus{
			Running: false,
			Message: "not configured — add credentials via Web UI → Settings → Channels",
			Since:   p.started,
		}
	}
	// 检查 token 是否有效
	p.tokenMu.RLock()
	hasToken := p.cachedToken != "" && time.Now().Before(p.tokenExpiry)
	p.tokenMu.RUnlock()
	return plugin.HealthStatus{
		Running: hasToken,
		Message: "ok",
		Since:   p.started,
	}
}

// HandleEventRequest processes a raw HTTP request from the Feishu event subscription.
// Returns the response body to write and the HTTP status code.
// If encrypt_key is configured, it validates the request signature and timestamp (anti-replay).
// If encrypt_key is not configured, it skips signature validation.
func (p *Plugin) HandleEventRequest(r *http.Request) (interface{}, int) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return map[string]string{"error": "read body"}, http.StatusBadRequest
	}

	// 可选的签名校验：如果配置了 encrypt_key，则验证请求
	if p.cfg.EncryptKey != "" {
		timestamp := r.Header.Get("X-Lark-Request-Timestamp")
		signature := r.Header.Get("X-Lark-Signature")

		if timestamp == "" || signature == "" {
			p.logger.Warn("feishu: missing signature headers, rejecting request")
			return map[string]string{"error": "missing signature"}, http.StatusUnauthorized
		}

		// 检查时间戳是否在 5 分钟内（防重放）
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return map[string]string{"error": "invalid timestamp"}, http.StatusBadRequest
		}
		reqTime := time.Unix(ts, 0)
		if time.Since(reqTime) > 5*time.Minute {
			p.logger.Warn("feishu: request timestamp expired, possible replay attack")
			return map[string]string{"error": "request expired"}, http.StatusUnauthorized
		}

		// 计算签名：HMAC-SHA256(timestamp + body)
		expectedSig := signFeishu(timestamp, p.cfg.EncryptKey, string(body))
		if expectedSig != signature {
			p.logger.Warn("feishu: signature mismatch, rejecting request")
			return map[string]string{"error": "invalid signature"}, http.StatusUnauthorized
		}
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

	resp, err := p.httpClient.Post(
		"https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return &FeishuError{Msg: fmt.Sprintf("获取 token 失败: %v", err)}
	}
	defer resp.Body.Close()

	var result struct {
		Code           int    `json:"code"`
		AppAccessToken string `json:"app_access_token"`
		Expire         int    `json:"expire"` // token 有效期（秒），飞书通常返回 7200
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &FeishuError{Msg: "解析 token 响应失败"}
	}
	if result.Code != 0 {
		return newFeishuError(result.Code, "")
	}

	// 保存 token 和过期时间
	p.tokenMu.Lock()
	p.cachedToken = result.AppAccessToken
	if result.Expire > 0 {
		p.tokenExpiry = time.Now().Add(time.Duration(result.Expire) * time.Second)
	} else {
		// 默认 2 小时
		p.tokenExpiry = time.Now().Add(2 * time.Hour)
	}
	p.tokenMu.Unlock()

	return nil
}

// getToken returns the cached token, refreshing if expired or about to expire.
// It uses double-check locking pattern under write lock to avoid concurrent refreshes.
func (p *Plugin) getToken() (string, error) {
	p.tokenMu.RLock()
	// 第一次检查：快速路径
	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry.Add(-5*time.Minute)) {
		token := p.cachedToken
		p.tokenMu.RUnlock()
		return token, nil
	}
	p.tokenMu.RUnlock()

	// 需要刷新：加写锁，进入临界区
	p.tokenMu.Lock()
	// 第二次检查：确保其他协程没有已经刷新
	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry.Add(-5*time.Minute)) {
		token := p.cachedToken
		p.tokenMu.Unlock()
		return token, nil
	}

	// 确实需要刷新
	if err := p.refreshToken(); err != nil {
		p.tokenMu.Unlock()
		return "", err
	}

	token := p.cachedToken
	p.tokenMu.Unlock()
	return token, nil
}

// signFeishu computes the Feishu signature for event callback verification.
// The signature is HMAC-SHA256(timestamp + body), encoded as hex string.
// encrypt_key is configured in Feishu Open Platform → Event Subscription → Callback URL.
func signFeishu(timestamp, encryptKey, body string) string {
	h := hmac.New(sha256.New, []byte(encryptKey))
	h.Write([]byte(timestamp + body))
	return hex.EncodeToString(h.Sum(nil))
}
