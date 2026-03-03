// Package feishu implements the Feishu (Lark) channel plugin for GoPaw.
// It uses Feishu Stream Mode (WebSocket) to receive events, eliminating the need
// for public IP or reverse proxy.
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"
)

// ── 常量定义 / Constants ───────────────────────────────────────────────────

const (
	tokenEndpoint   = "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal"
	sendEndpoint    = "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"
	tokenRefreshSkew = 5 * time.Minute // 提前 5 分钟刷新
	defaultTimeout  = 10 * time.Second
)

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 256),
	})
}

type feishuConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// Plugin implements the Feishu channel using Stream Mode (WebSocket).
type Plugin struct {
	cfg        feishuConfig
	inbound    chan *types.Message
	started    time.Time
	configured bool
	logger     *zap.Logger
	httpClient *http.Client
	
	// 状态控制 / State Control
	ctx        context.Context
	cancelFunc context.CancelFunc
	connected  bool
	mu         sync.RWMutex

	// Token 缓存 / Token Cache
	tokenMu      sync.RWMutex
	cachedToken  string
	tokenExpiry  time.Time
}

func (p *Plugin) Name() string        { return "feishu" }
func (p *Plugin) DisplayName() string { return "飞书" }

// Init parses Feishu credentials.
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.feishu")
	p.httpClient = &http.Client{Timeout: defaultTimeout}
	
	if len(cfg) > 0 && string(cfg) != "{}" {
		if err := json.Unmarshal(cfg, &p.cfg); err != nil {
			p.logger.Warn("feishu: failed to parse config", zap.Error(err))
			return nil
		}
	}
	if p.cfg.AppID == "" || p.cfg.AppSecret == "" {
		return plugin.ErrMissingCredentials
	}
	p.configured = true
	return nil
}

// Start establishes the WebSocket connection to Feishu.
func (p *Plugin) Start(ctx context.Context) error {
	if !p.configured {
		return nil
	}
	p.started = time.Now()
	
	// 中文：绑定生命周期上下文
	// English: Bind lifecycle context
	p.ctx, p.cancelFunc = context.WithCancel(ctx)

	// 中文：创建事件处理器
	// English: Create event dispatcher
	eventHandler := dispatcher.NewEventDispatcher("", "")
	eventHandler.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		p.handleIncomingMessage(event)
		return nil
	})

	// 中文：初始化 WebSocket 客户端
	// English: Initialize WebSocket client
	wsClient := larkws.NewClient(p.cfg.AppID, p.cfg.AppSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)

	// 中文：启动长连接
	// English: Start long connection
	go func() {
		// 中文：启动时先标记为已连接，SDK Start 会阻塞直到连接断开
		// English: Mark as connected on start, SDK Start blocks until disconnected
		p.mu.Lock()
		p.connected = true
		p.mu.Unlock()
		
		p.logger.Info("feishu stream mode connecting")
		
		err := wsClient.Start(p.ctx)
		
		// 中文：连接断开，标记状态
		// English: Connection closed, update status
		p.mu.Lock()
		p.connected = false
		p.mu.Unlock()
		
		if err != nil && p.ctx.Err() == nil {
			p.logger.Error("feishu ws client failed", zap.Error(err))
		} else {
			p.logger.Info("feishu ws client stopped")
		}
	}()

	p.logger.Info("feishu stream mode started")
	return nil
}

// Stop closes the WebSocket connection.
func (p *Plugin) Stop() error {
	p.logger.Info("feishu channel stopping")
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
	return nil
}

// Receive returns the inbound message channel.
func (p *Plugin) Receive() <-chan *types.Message {
	return p.inbound
}

// handleIncomingMessage converts a Feishu event to GoPaw Message.
func (p *Plugin) handleIncomingMessage(event *larkim.P2MessageReceiveV1) {
	// 中文：严格的 nil 检查，防止 SDK 字段解引用 panic
	// English: Strict nil checks to prevent SDK field dereference panic
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return
	}
	
	msgData := event.Event.Message
	if msgData.MessageType == nil || *msgData.MessageType != "text" {
		return
	}

	if msgData.Content == nil {
		return
	}

	var content map[string]string
	if err := json.Unmarshal([]byte(*msgData.Content), &content); err != nil {
		p.logger.Error("feishu: failed to unmarshal message content", zap.Error(err))
		return
	}
	text := content["text"]

	if event.Event.Sender == nil || event.Event.Sender.SenderId == nil || event.Event.Sender.SenderId.OpenId == nil {
		return
	}

	userID := *event.Event.Sender.SenderId.OpenId
	chatID := ""
	if msgData.ChatId != nil {
		chatID = *msgData.ChatId
	}
	msgID := ""
	if msgData.MessageId != nil {
		msgID = *msgData.MessageId
	}

	msg := &types.Message{
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
	case p.inbound <- msg:
	default:
		p.logger.Warn("feishu: inbound queue full, dropping message")
	}
}

// Send delivers a message via the Feishu REST API.
func (p *Plugin) Send(msg *types.Message) error {
	if !p.configured {
		return fmt.Errorf("feishu: channel not configured")
	}
	receiveID := msg.SessionID
	if receiveID == "" {
		receiveID = msg.UserID
	}

	token, err := p.getToken()
	if err != nil {
		return fmt.Errorf("feishu: get token: %w", err)
	}

	payload := map[string]interface{}{
		"receive_id": receiveID,
		"msg_type":   "text",
		"content":    fmt.Sprintf(`{"text":%q}`, msg.Content),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("feishu: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		sendEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("feishu: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("feishu: http send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu api error (status %d): %s", resp.StatusCode, string(b))
	}
	return nil
}

// Health returns the current operational status.
func (p *Plugin) Health() plugin.HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.configured {
		return plugin.HealthStatus{Running: false, Message: "未配置", Since: p.started}
	}
	if !p.connected {
		return plugin.HealthStatus{Running: false, Message: "长连接未建立或已断开", Since: p.started}
	}
	return plugin.HealthStatus{Running: true, Message: "长连接运行中", Since: p.started}
}

// Test validates the Feishu connection and credentials.
// It checks: 1) configuration completeness, 2) token validity, 3) WebSocket connection status.
func (p *Plugin) Test(ctx context.Context) plugin.TestResult {
	// 检查配置是否完整
	if !p.configured || p.cfg.AppID == "" || p.cfg.AppSecret == "" {
		return plugin.TestResult{
			Success: false,
			Message: "请先配置 app_id 和 app_secret",
		}
	}

	// 尝试获取 token（验证凭证有效性）
	_, err := p.getToken()
	if err != nil {
		return plugin.TestResult{
			Success: false,
			Message: "凭证验证失败，请检查 app_id 和 app_secret",
			Details: err.Error(),
		}
	}

	// 检查 WebSocket 连接状态
	p.mu.RLock()
	connected := p.connected
	p.mu.RUnlock()

	if !connected {
		return plugin.TestResult{
			Success: false,
			Message: "长连接未建立，请稍后重试或检查网络",
		}
	}

	return plugin.TestResult{
		Success: true,
		Message: "连接正常，凭证有效",
	}
}

// getToken handles thread-safe token retrieval and lazy refreshing.
func (p *Plugin) getToken() (string, error) {
	p.tokenMu.RLock()
	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry.Add(-tokenRefreshSkew)) {
		token := p.cachedToken
		p.tokenMu.RUnlock()
		return token, nil
	}
	p.tokenMu.RUnlock()

	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry.Add(-tokenRefreshSkew)) {
		return p.cachedToken, nil
	}

	if err := p.refreshToken(); err != nil {
		return "", err
	}
	return p.cachedToken, nil
}

func (p *Plugin) refreshToken() error {
	payload := map[string]string{
		"app_id":     p.cfg.AppID,
		"app_secret": p.cfg.AppSecret,
	}
	body, _ := json.Marshal(payload)

	resp, err := p.httpClient.Post(tokenEndpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("feishu: token http post: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code           int    `json:"code"`
		AppAccessToken string `json:"app_access_token"`
		Expire         int    `json:"expire"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("feishu: decode token response: %w", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("feishu: token error code %d", result.Code)
	}

	p.cachedToken = result.AppAccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(result.Expire) * time.Second)
	return nil
}
