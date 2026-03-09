package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
)

// Plugin 企业微信频道插件
type Plugin struct {
	cfg *Config

	// HTTP 客户端
	httpClient *http.Client

	// 消息通道
	inbound chan *types.Message

	// 媒体存储
	store plugin.MediaStore

	// Token 管理
	tokenMu     sync.RWMutex
	cachedToken string
	tokenExpiry time.Time

	// 状态
	running bool
	started time.Time

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	logger *zap.Logger
}

// Config 企业微信配置
type Config struct {
	Enabled   bool   `json:"enabled"`
	CorpID    string `json:"corp_id"`
	AgentID   int    `json:"agent_id"`
	Secret    string `json:"secret"`
	Token     string `json:"token"`
	EncodingKey string `json:"encoding_key"`
	MediaDir  string `json:"media_dir"`
}

// API 响应
type APIResponse struct {
	ErrCode int             `json:"errcode"`
	ErrMsg  string          `json:"errmsg"`
	OK      bool            `json:"ok"`
	Result  json.RawMessage `json:"result,omitempty"`
}

// Token 响应
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// 消息接收响应
type ReceiveResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	Total       int    `json:"total"`
	NextCursor  string `json:"next_cursor"`
	Messages    []Message `json:"messages"`
}

// Message 企业微信消息
type Message struct {
	MsgID     string `json:"msgid"`
	FromUser string `json:"fromuser"`
	ToUser   string `json:"touser"`
	ChatID   string `json:"chatid"`
	MsgType  string `json:"msgtype"`
	Content  string `json:"content,omitempty"`
	AgentID  int    `json:"agentid"`
	Time     int64  `json:"createtime"`
}

// 发送消息请求
type SendMessageRequest struct {
	ToUser  string `json:"touser,omitempty"`
	ToParty string `json:"toparty,omitempty"`
	ToTag   string `json:"totag,omitempty"`
	MsgType string `json:"msgtype"`
	AgentID int    `json:"agentid"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text,omitempty"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown,omitempty"`
}

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 100),
	})
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "wecom"
}

// DisplayName 返回显示名称
func (p *Plugin) DisplayName() string {
	return "企业微信"
}

// Init 初始化插件
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.cfg = &Config{}
	if err := json.Unmarshal(cfg, p.cfg); err != nil {
		return fmt.Errorf("parse wecom config: %w", err)
	}

	p.logger = zap.L().Named("wecom")
	p.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	return nil
}

// SetMediaStore 设置媒体存储器
func (p *Plugin) SetMediaStore(store plugin.MediaStore) {
	p.store = store
}

// AddReaction 添加表情反应（实现 ReactionCapable 接口）
func (p *Plugin) AddReaction(channelID, messageTS, reaction string) error {
	// 企业微信暂不支持表情反应
	p.logger.Debug("wecom does not support reactions")
	return nil
}

// RemoveReaction 移除表情反应（实现 ReactionCapable 接口）
func (p *Plugin) RemoveReaction(channelID, messageTS, reaction string) error {
	return nil
}

// EditMessage 编辑消息（实现 MessageEditor 接口）
func (p *Plugin) EditMessage(channelID, messageTS, newContent string) error {
	// 企业微信暂不支持消息编辑
	p.logger.Debug("wecom does not support message editing")
	return nil
}

// DeleteMessage 删除消息（实现 MessageEditor 接口）
func (p *Plugin) DeleteMessage(channelID, messageTS string) error {
	// 企业微信暂不支持消息删除
	p.logger.Debug("wecom does not support message deletion")
	return nil
}

// Start 启动插件
func (p *Plugin) Start(ctx context.Context) error {
	if !p.cfg.Enabled {
		p.logger.Info("wecom plugin disabled")
		return nil
	}

	if p.cfg.CorpID == "" || p.cfg.Secret == "" {
		return fmt.Errorf("wecom corp_id and secret required")
	}

	p.ctx, p.cancel = context.WithCancel(ctx)
	p.running = true
	p.started = time.Now()

	p.logger.Info("starting wecom plugin",
		zap.String("corp_id", p.cfg.CorpID),
		zap.Int("agent_id", p.cfg.AgentID))

	// 测试 Token 获取
	_, err := p.GetAccessToken()
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	// 启动消息轮询
	go p.pollingLoop()

	return nil
}

// Stop 停止插件
func (p *Plugin) Stop() error {
	if !p.running {
		return nil
	}

	p.running = false
	p.cancel()

	if p.inbound != nil {
		close(p.inbound)
	}

	p.logger.Info("wecom plugin stopped")
	return nil
}

// Receive 接收消息
func (p *Plugin) Receive() <-chan *types.Message {
	return p.inbound
}

// Send 发送消息
func (p *Plugin) Send(msg *types.Message) error {
	if !p.running {
		return fmt.Errorf("plugin not running")
	}

	token, err := p.GetAccessToken()
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	// 从 Metadata 中获取收件人
	recipient := msg.Metadata["recipient"]
	if recipient == "" {
		recipient = msg.UserID
	}

	if recipient == "" {
		return fmt.Errorf("no recipient specified")
	}

	// 发送 Markdown 消息
	return p.sendMarkdown(token, recipient, msg.Content)
}

// Health 返回健康状态
func (p *Plugin) Health() plugin.HealthStatus {
	if !p.running {
		return plugin.HealthStatus{
			Running: false,
			Message: "stopped",
		}
	}

	return plugin.HealthStatus{
		Running: true,
		Message: "running",
		Since:   p.started,
	}
}

// Test 测试连接
func (p *Plugin) Test(ctx context.Context) plugin.TestResult {
	if !p.running {
		return plugin.TestResult{
			Success: false,
			Message: "plugin not running",
		}
	}

	// 测试 Token 获取
	token, err := p.GetAccessToken()
	if err != nil {
		return plugin.TestResult{
			Success: false,
			Message: "failed to get access token: " + err.Error(),
		}
	}

	// 测试发送消息（可选）
	// 这里只测试 Token 获取

	return plugin.TestResult{
		Success: true,
		Message: "connected",
		Details: fmt.Sprintf("corp_id: %s, token: %s...", p.cfg.CorpID, token[:min(10, len(token))]),
	}
}

// pollingLoop 消息轮询循环
func (p *Plugin) pollingLoop() {
	cursor := ""
	for p.running {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		messages, nextCursor, err := p.fetchMessages(cursor)
		if err != nil {
			p.logger.Error("fetch messages failed", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}

		for _, msg := range messages {
			p.handleMessage(&msg)
		}

		if nextCursor != "" {
			cursor = nextCursor
		}

		// 无新消息时等待
		if len(messages) == 0 {
			time.Sleep(3 * time.Second)
		}
	}
}

// handleMessage 处理消息
func (p *Plugin) handleMessage(msg *Message) {
	// 跳过空消息
	if msg.Content == "" {
		return
	}

	p.logger.Debug("received message",
		zap.String("from", msg.FromUser),
		zap.String("content", msg.Content))

	// 构建消息
	message := &types.Message{
		ID:        fmt.Sprintf("wecom_%s", msg.MsgID),
		Channel:   "wecom",
		UserID:    msg.FromUser,
		ChatID:    msg.ChatID,
		Content:   msg.Content,
		Timestamp: msg.Time,
		SessionID: fmt.Sprintf("wecom:%s", msg.ChatID),
		Metadata: map[string]string{
			"msg_id":   msg.MsgID,
			"msg_type": msg.MsgType,
			"agent_id": fmt.Sprintf("%d", msg.AgentID),
		},
	}

	// 发送到消息通道
	select {
	case p.inbound <- message:
		p.logger.Debug("message sent to channel")
	case <-time.After(5 * time.Second):
		p.logger.Warn("send message timeout")
	case <-p.ctx.Done():
		return
	}
}

// fetchMessages 获取消息
func (p *Plugin) fetchMessages(cursor string) ([]Message, string, error) {
	token, err := p.GetAccessToken()
	if err != nil {
		return nil, "", err
	}

	// 构建请求体
	body := map[string]interface{}{
		"agentid": p.cfg.AgentID,
		"limit":   100,
	}
	if cursor != "" {
		body["cursor"] = cursor
	}

	jsonBody, _ := json.Marshal(body)

	// 发送请求
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/get?access_token=%s", token)
	resp, err := p.httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read response: %w", err)
	}

	var result ReceiveResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, "", fmt.Errorf("unmarshal response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, "", fmt.Errorf("wecom API error (%d): %s", result.ErrCode, result.ErrMsg)
	}

	return result.Messages, result.NextCursor, nil
}

// sendMarkdown 发送 Markdown 消息
func (p *Plugin) sendMarkdown(token, recipient, content string) error {
	req := SendMessageRequest{
		ToUser:  recipient,
		MsgType: "markdown",
		AgentID: p.cfg.AgentID,
	}
	req.Markdown.Content = content

	jsonBody, _ := json.Marshal(req)

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	resp, err := p.httpClient.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.ErrCode != 0 {
		return fmt.Errorf("wecom API error (%d): %s", apiResp.ErrCode, apiResp.ErrMsg)
	}

	p.logger.Debug("message sent",
		zap.String("recipient", recipient),
		zap.String("content", content[:min(50, len(content))]))

	return nil
}

// GetAccessToken 获取访问令牌
func (p *Plugin) GetAccessToken() (string, error) {
	// 检查缓存
	p.tokenMu.RLock()
	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry) {
		token := p.cachedToken
		p.tokenMu.RUnlock()
		return token, nil
	}
	p.tokenMu.RUnlock()

	// 双重检查 + 获取新令牌
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	// 再次检查（避免并发请求）
	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry) {
		return p.cachedToken, nil
	}

	// 请求新令牌
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		p.cfg.CorpID, p.cfg.Secret)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result TokenResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("token error (%d): %s", result.ErrCode, result.ErrMsg)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response")
	}

	// 缓存令牌（提前 5 分钟刷新）
	p.cachedToken = result.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	p.logger.Debug("access token refreshed",
		zap.Int("expires_in", result.ExpiresIn))

	return result.AccessToken, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
