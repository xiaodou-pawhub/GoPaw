package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"go.uber.org/zap"

	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
)

// Plugin 钉钉频道插件
type Plugin struct {
	cfg *Config

	// Stream 客户端
	streamClient *client.StreamClient

	// 消息通道
	inbound chan *types.Message

	// Session Webhook 存储 (用于主动推送)
	sessionWebhooks sync.Map // sessionID -> webhook

	// Token 管理
	tokenMu     sync.RWMutex
	cachedToken string
	tokenExpiry time.Time

	// 状态
	running bool
	started time.Time
	logger  *zap.Logger
}

// Config 钉钉配置
type Config struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	BotPrefix    string `json:"bot_prefix"`
	MediaDir     string `json:"media_dir"`
}

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 100),
	})
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "dingtalk"
}

// DisplayName 返回显示名称
func (p *Plugin) DisplayName() string {
	return "钉钉"
}

// Init 初始化插件
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.cfg = &Config{}
	if err := json.Unmarshal(cfg, p.cfg); err != nil {
		return fmt.Errorf("parse dingtalk config: %w", err)
	}

	p.logger = zap.L().Named("dingtalk")
	return nil
}

// Start 启动插件
func (p *Plugin) Start(ctx context.Context) error {
	if !p.cfg.Enabled {
		p.logger.Info("dingtalk plugin disabled")
		return nil
	}

	if p.cfg.ClientID == "" || p.cfg.ClientSecret == "" {
		return fmt.Errorf("dingtalk client_id and client_secret required")
	}

	p.logger.Info("starting dingtalk plugin",
		zap.String("client_id", p.cfg.ClientID))

	// 创建凭证
	cred := client.NewAppCredentialConfig(p.cfg.ClientID, p.cfg.ClientSecret)

	// 创建 Stream 客户端
	p.streamClient = client.NewStreamClient(
		client.WithAppCredential(cred),
		client.WithAutoReconnect(true),
	)

	// 注册消息处理器
	handler := p.createMessageHandler()
	p.streamClient.RegisterChatBotCallbackRouter(handler)

	// 启动客户端
	if err := p.streamClient.Start(ctx); err != nil {
		return fmt.Errorf("start stream client: %w", err)
	}

	p.running = true
	p.started = time.Now()
	p.logger.Info("dingtalk plugin started")

	return nil
}

// Stop 停止插件
func (p *Plugin) Stop() error {
	if !p.running {
		return nil
	}

	p.running = false

	// 关闭消息通道
	close(p.inbound)

	// 关闭 Stream 客户端
	if p.streamClient != nil {
		p.streamClient.Close()
	}

	p.logger.Info("dingtalk plugin stopped")
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

	// 尝试使用 Session Webhook 发送
	if webhook, ok := p.sessionWebhooks.Load(msg.SessionID); ok {
		if err := p.sendViaWebhook(webhook.(string), msg); err == nil {
			return nil
		}
		// Webhook 失败，回退到 Open API
	}

	// 回退到 Open API 发送
	token, err := p.GetAccessToken()
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	// 从 Metadata 中获取收件人信息
	recipient := msg.Metadata["recipient"]
	isGroup := msg.Metadata["is_group"] == "true"

	if recipient == "" {
		return fmt.Errorf("no recipient specified")
	}

	return p.SendViaOpenAPI(token, recipient, msg.Content, isGroup)
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
	_, err := p.GetAccessToken()
	if err != nil {
		return plugin.TestResult{
			Success: false,
			Message: "failed to get access token: " + err.Error(),
		}
	}

	return plugin.TestResult{
		Success: true,
		Message: "connected",
		Details: fmt.Sprintf("client_id: %s", p.cfg.ClientID),
	}
}

// SetMediaStore 设置媒体存储器（实现 MediaStoreReceiver 接口）
func (p *Plugin) SetMediaStore(store plugin.MediaStore) {
	p.logger.Debug("media store configured")
	// TODO: 使用 store 处理媒体文件
}

// AddReaction 添加表情反应（实现 ReactionCapable 接口）
func (p *Plugin) AddReaction(channelID, messageTS, reaction string) error {
	// 钉钉暂不支持表情反应
	p.logger.Debug("dingtalk does not support reactions")
	return nil
}

// RemoveReaction 移除表情反应（实现 ReactionCapable 接口）
func (p *Plugin) RemoveReaction(channelID, messageTS, reaction string) error {
	// 钉钉暂不支持表情反应
	return nil
}

// EditMessage 编辑消息（实现 MessageEditor 接口）
func (p *Plugin) EditMessage(channelID, messageTS, newContent string) error {
	// 钉钉暂不支持消息编辑
	p.logger.Debug("dingtalk does not support message editing")
	return nil
}

// DeleteMessage 删除消息（实现 MessageEditor 接口）
func (p *Plugin) DeleteMessage(channelID, messageTS string) error {
	// 钉钉暂不支持消息删除
	p.logger.Debug("dingtalk does not support message deletion")
	return nil
}

// SendMarkdown 发送 Markdown 消息（实现 RichTextCapable 接口）
func (p *Plugin) SendMarkdown(channelID, markdown string) error {
	token, err := p.GetAccessToken()
	if err != nil {
		return err
	}

	// 钉钉使用 sampleMarkdown 模板
	url := "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
	payload := map[string]interface{}{
		"robotCode": p.cfg.ClientID,
		"userIds":   []string{channelID},
		"msgKey":    "sampleMarkdown",
		"msgParam":  fmt.Sprintf(`{"text":"%s","title":"GoPaw"}`, markdown),
	}

	return p.sendDingTalkRequest(token, url, payload)
}

// SendHTML 发送 HTML 消息（实现 RichTextCapable 接口）
func (p *Plugin) SendHTML(channelID, html string) error {
	// 钉钉暂不支持纯 HTML，使用 Markdown 替代
	return p.SendMarkdown(channelID, html)
}

// SendBlockKit 发送 Block Kit 消息（实现 RichTextCapable 接口）
func (p *Plugin) SendBlockKit(channelID string, blocks []plugin.Block) error {
	// 钉钉暂不支持 Block Kit
	p.logger.Debug("dingtalk does not support Block Kit")
	return nil
}

// SendFile 发送文件（实现 FileCapable 接口）
func (p *Plugin) SendFile(channelID, filePath, caption string) error {
	// 钉钉暂不支持文件上传
	p.logger.Debug("dingtalk does not support file upload yet")
	return nil
}

// SendImage 发送图片（实现 FileCapable 接口）
func (p *Plugin) SendImage(channelID, imagePath, caption string) error {
	// 钉钉暂不支持图片上传
	p.logger.Debug("dingtalk does not support image upload yet")
	return nil
}

// SendVideo 发送视频（实现 FileCapable 接口）
func (p *Plugin) SendVideo(channelID, videoPath, caption string) error {
	// 钉钉暂不支持视频上传
	p.logger.Debug("dingtalk does not support video upload yet")
	return nil
}

// SendAudio 发送音频（实现 FileCapable 接口）
func (p *Plugin) SendAudio(channelID, audioPath, caption string) error {
	// 钉钉暂不支持音频上传
	p.logger.Debug("dingtalk does not support audio upload yet")
	return nil
}

// sendDingTalkRequest 发送钉钉请求（内部方法）
func (p *Plugin) sendDingTalkRequest(token, url string, payload map[string]interface{}) error {
	jsonBody, _ := json.Marshal(payload)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dingtalk API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// createMessageHandler 创建消息处理器
func (p *Plugin) createMessageHandler() chatbot.IChatBotMessageHandler {
	return func(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
		p.logger.Debug("received message",
			zap.String("conversation_id", data.ConversationId),
			zap.String("sender_id", data.SenderId),
			zap.String("content", data.Text.Content))

		// 存储 Session Webhook (用于主动推送)
		if data.SessionWebhook != "" {
			sessionID := fmt.Sprintf("dingtalk:%s", data.ConversationId)
			p.sessionWebhooks.Store(sessionID, data.SessionWebhook)
			p.logger.Debug("stored session webhook",
				zap.String("session_id", sessionID))
		}

		// 构建消息
		message := &types.Message{
			ID:        data.MsgId,
			Channel:   "dingtalk",
			UserID:    data.SenderId,
			Content:   data.Text.Content,
			Timestamp: time.Now().Unix(),
			SessionID: fmt.Sprintf("dingtalk:%s", data.ConversationId),
			Metadata: map[string]string{
				"conversation_id": data.ConversationId,
				"sender_id":       data.SenderId,
				"msg_type":        data.Msgtype,
			},
		}

		// 发送到消息通道
		select {
		case p.inbound <- message:
			p.logger.Debug("message sent to channel",
				zap.String("message_id", message.ID))
		case <-time.After(5 * time.Second):
			p.logger.Warn("send message timeout")
			return []byte("skip"), nil
		}

		return []byte("success"), nil
	}
}

// sendViaWebhook 通过 Session Webhook 发送消息
func (p *Plugin) sendViaWebhook(webhook string, msg *types.Message) error {
	if webhook == "" {
		return fmt.Errorf("webhook is empty")
	}

	// 构建 Markdown 内容
	markdownText := normalizeDingTalkMarkdown(msg.Content)

	// 构建请求体
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "GoPaw",
			"text":  markdownText,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// 发送请求
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", webhook, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook response status %d: %s", resp.StatusCode, string(respBody))
	}

	p.logger.Debug("message sent via webhook",
		zap.String("session_id", msg.SessionID))

	return nil
}

// SendViaOpenAPI 通过 Open API 发送消息（主动推送）
func (p *Plugin) SendViaOpenAPI(token, recipient, content string, isGroup bool) error {
	if token == "" {
		return fmt.Errorf("access token is empty")
	}

	// 构建 Markdown 内容
	markdownText := normalizeDingTalkMarkdown(content)

	// 构建请求体
	var url string
	var payload map[string]interface{}

	if isGroup {
		// 群聊消息
		url = "https://api.dingtalk.com/v1.0/robot/groupMessages/send"
		payload = map[string]interface{}{
			"robotCode":         p.cfg.ClientID,
			"openConversationId": recipient,
			"msgKey":            "sampleMarkdown",
			"msgParam":          fmt.Sprintf(`{"text":"%s","title":"GoPaw"}`, markdownText),
		}
	} else {
		// 私聊消息
		url = "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
		payload = map[string]interface{}{
			"robotCode": p.cfg.ClientID,
			"userIds":   []string{recipient},
			"msgKey":    "sampleMarkdown",
			"msgParam":  fmt.Sprintf(`{"text":"%s","title":"GoPaw"}`, markdownText),
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// 发送请求
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-acs-dingtalk-access-token", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api response status %d: %s", resp.StatusCode, string(respBody))
	}

	p.logger.Debug("message sent via open api",
		zap.String("recipient", recipient),
		zap.Bool("is_group", isGroup))

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
	url := "https://api.dingtalk.com/v1.0/oauth2/accessToken"
	payload := map[string]string{
		"appKey":    p.cfg.ClientID,
		"appSecret": p.cfg.ClientSecret,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token response status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireIn    int    `json:"expireIn"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response")
	}

	// 缓存令牌（提前 5 分钟刷新）
	p.cachedToken = result.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(result.ExpireIn-300) * time.Second)

	p.logger.Debug("access token refreshed",
		zap.Int("expire_in", result.ExpireIn))

	return result.AccessToken, nil
}

// normalizeDingTalkMarkdown 钉钉 Markdown 规范化
func normalizeDingTalkMarkdown(text string) string {
	// 1. 确保编号列表前有空行
	text = ensureListSpacing(text)

	// 2. 去除代码块的不必要缩进
	text = dedentCodeBlocks(text)

	return text
}

// ensureListSpacing 确保编号列表项前有空行
func ensureListSpacing(text string) string {
	lines := strings.Split(text, "\n")
	var out []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		isNumbered := len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9'
		if isNumbered && i > 0 {
			prev := strings.TrimSpace(lines[i-1])
			prevIsNumbered := len(prev) > 0 && prev[0] >= '0' && prev[0] <= '9'
			if prev != "" && !prevIsNumbered {
				out = append(out, "") // 添加空行
			}
		}
		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

// dedentCodeBlocks 去除代码块的不必要缩进
func dedentCodeBlocks(text string) string {
	// TODO: 实现代码块缩进处理
	// 目前保持原样
	return text
}
