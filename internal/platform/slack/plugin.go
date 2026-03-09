package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"

	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
)

// Plugin Slack 频道插件
type Plugin struct {
	cfg *Config

	// Slack 客户端
	client *slack.Client

	// 消息通道
	inbound chan *types.Message

	// 媒体存储
	store plugin.MediaStore

	// 用户缓存
	userCache sync.Map // userID -> displayName

	// Token 管理（Socket Mode）
	appToken   string
	botToken   string

	// 状态
	running bool
	started time.Time

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	logger *zap.Logger
}

// Config Slack 配置
type Config struct {
	Enabled     bool   `json:"enabled"`
	BotToken    string `json:"bot_token"`
	AppToken    string `json:"app_token"`  // Socket Mode 需要
	HTTPProxy   string `json:"http_proxy"`
	MediaDir    string `json:"media_dir"`
}

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 100),
	})
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "slack"
}

// DisplayName 返回显示名称
func (p *Plugin) DisplayName() string {
	return "Slack"
}

// Init 初始化插件
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.cfg = &Config{}
	if err := json.Unmarshal(cfg, p.cfg); err != nil {
		return fmt.Errorf("parse slack config: %w", err)
	}

	p.logger = zap.L().Named("slack")
	return nil
}

// SetMediaStore 设置媒体存储器
func (p *Plugin) SetMediaStore(store plugin.MediaStore) {
	p.store = store
}

// Start 启动插件
func (p *Plugin) Start(ctx context.Context) error {
	if !p.cfg.Enabled {
		p.logger.Info("slack plugin disabled")
		return nil
	}

	if p.cfg.BotToken == "" {
		return fmt.Errorf("slack bot_token required")
	}

	p.ctx, p.cancel = context.WithCancel(ctx)
	p.running = true
	p.started = time.Now()
	p.botToken = p.cfg.BotToken
	p.appToken = p.cfg.AppToken

	p.logger.Info("starting slack plugin",
		zap.Bool("socket_mode", p.cfg.AppToken != ""))

	// 创建 Slack 客户端
	p.client = slack.New(p.cfg.BotToken)

	// 测试连接
	auth, err := p.client.AuthTest()
	if err != nil {
		return fmt.Errorf("auth test: %w", err)
	}
	p.logger.Info("slack connected",
		zap.String("team", auth.Team),
		zap.String("user", auth.User))

	// 创建媒体目录
	if p.cfg.MediaDir != "" {
		if err := os.MkdirAll(p.cfg.MediaDir, 0755); err != nil {
			p.logger.Warn("create media dir failed", zap.Error(err))
		}
	}

	// 启动消息接收
	if p.cfg.AppToken != "" {
		// Socket Mode
		go p.socketModeLoop()
	} else {
		// HTTP Polling
		go p.pollingLoop()
	}

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

	p.logger.Info("slack plugin stopped")
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

	channelID := msg.ChatID
	if channelID == "" {
		return fmt.Errorf("no channel_id specified")
	}

	// 发送消息
	_, _, err := p.client.PostMessage(
		channelID,
		slack.MsgOptionText(msg.Content, false),
	)
	if err != nil {
		return fmt.Errorf("post message: %w", err)
	}

	p.logger.Debug("message sent",
		zap.String("channel", channelID),
		zap.String("content", msg.Content[:min(50, len(msg.Content))]))

	return nil
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

	// 测试连接
	auth, err := p.client.AuthTest()
	if err != nil {
		return plugin.TestResult{
			Success: false,
			Message: "auth test failed: " + err.Error(),
		}
	}

	return plugin.TestResult{
		Success: true,
		Message: "connected",
		Details: fmt.Sprintf("team: %s, user: %s", auth.Team, auth.User),
	}
}

// socketModeLoop Socket Mode 消息循环（需要额外依赖，这里使用简化实现）
func (p *Plugin) socketModeLoop() {
	p.logger.Info("socket mode not fully implemented, falling back to polling")
	// TODO: 完整实现需要 github.com/slack-go/slack/socketmode
	// 这里暂时使用 polling 方式
	p.pollingLoop()
}

// pollingLoop HTTP Polling 消息循环
func (p *Plugin) pollingLoop() {
	// 获取所有频道
	channelIDs, err := p.getChannelIDs()
	if err != nil {
		p.logger.Error("get channels failed", zap.Error(err))
		return
	}

	p.logger.Info("polling channels", zap.Int("count", len(channelIDs)))

	// 记录最后消息时间
	lastMessageTime := make(map[string]float64)

	for p.running {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		// 轮询每个频道
		for _, channelID := range channelIDs {
			messages, err := p.getRecentMessages(channelID, lastMessageTime[channelID])
			if err != nil {
				p.logger.Error("get messages failed",
					zap.String("channel", channelID),
					zap.Error(err))
				continue
			}

			for _, msg := range messages {
				msgTs := parseSlackTimestamp(msg.Msg.Timestamp)
				if msgTs > lastMessageTime[channelID] {
					lastMessageTime[channelID] = msgTs
					p.handleSlackMessage(&msg, channelID)
				}
			}
		}

		// 无新消息时等待
		time.Sleep(3 * time.Second)
	}
}

// getChannelIDs 获取频道 ID 列表
func (p *Plugin) getChannelIDs() ([]string, error) {
	var channelIDs []string

	cursor := ""
	for {
		params := &slack.GetConversationsParameters{
			Types: []string{"public_channel", "private_channel", "im"},
			Limit: 100,
			Cursor: cursor,
		}

		channels, nextCursor, err := p.client.GetConversations(params)
		if err != nil {
			return nil, fmt.Errorf("get conversations: %w", err)
		}

		for _, ch := range channels {
			channelIDs = append(channelIDs, ch.ID)
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return channelIDs, nil
}

// getRecentMessages 获取最近消息
func (p *Plugin) getRecentMessages(channelID string, since float64) ([]slack.Message, error) {
	params := &slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Limit:     100,
		Inclusive: true,
	}

	if since > 0 {
		params.Oldest = fmt.Sprintf("%.6f", since)
	}

	result, err := p.client.GetConversationHistory(params)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	if !result.Ok {
		return nil, fmt.Errorf("get history failed: %s", result.Error)
	}

	// 转换消息格式
	messages := make([]slack.Message, len(result.Messages))
	for i, msg := range result.Messages {
		messages[i] = msg
	}

	return messages, nil
}

// handleSlackMessage 处理 Slack 消息
func (p *Plugin) handleSlackMessage(msg *slack.Message, channelID string) {
	// 跳过机器人自己的消息
	if msg.SubType == "bot_message" || msg.BotID != "" {
		return
	}

	// 跳过空消息
	if msg.Msg.Text == "" {
		return
	}

	// 获取用户显示名
	username := p.resolveUsername(msg.Msg.User)

	p.logger.Debug("received message",
		zap.String("channel", channelID),
		zap.String("user", msg.Msg.User),
		zap.String("text", msg.Msg.Text))

	// 构建消息
	message := &types.Message{
		ID:        fmt.Sprintf("slack_%s", msg.Msg.Timestamp),
		Channel:   "slack",
		UserID:    msg.Msg.User,
		ChatID:    channelID,
		Content:   msg.Msg.Text,
		Timestamp: int64(parseSlackTimestamp(msg.Msg.Timestamp)),
		SessionID: fmt.Sprintf("slack:%s", channelID),
		Metadata: map[string]string{
			"ts":       msg.Msg.Timestamp,
			"username": username,
			"thread_ts": msg.Msg.ThreadTimestamp,
		},
	}

	// 处理线程消息
	if msg.Msg.ThreadTimestamp != "" && msg.Msg.ThreadTimestamp != msg.Msg.Timestamp {
		message.Metadata["is_thread_reply"] = "true"
		message.Metadata["thread_ts"] = msg.Msg.ThreadTimestamp
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

// resolveUsername 解析用户显示名
func (p *Plugin) resolveUsername(userID string) string {
	if userID == "" {
		return "unknown"
	}

	// 检查缓存
	if cached, ok := p.userCache.Load(userID); ok {
		return cached.(string)
	}

	// 获取用户信息
	user, err := p.client.GetUserInfo(userID)
	if err != nil {
		p.logger.Debug("get user info failed",
			zap.String("user_id", userID),
			zap.Error(err))
		return userID
	}

	username := user.Profile.DisplayName
	if username == "" {
		username = user.Profile.RealName
	}
	if username == "" {
		username = user.Name
	}
	if username == "" {
		username = userID
	}

	// 缓存
	p.userCache.Store(userID, username)

	return username
}

// parseSlackTimestamp 解析 Slack 时间戳
func parseSlackTimestamp(ts string) float64 {
	var t float64
	fmt.Sscanf(ts, "%f", &t)
	return t
}

// sendThreadMessage 发送线程消息
func (p *Plugin) sendThreadMessage(channelID, threadTS, content string) error {
	_, _, err := p.client.PostMessage(
		channelID,
		slack.MsgOptionText(content, false),
		slack.MsgOptionTS(threadTS),
	)
	return err
}

// sendEphemeralMessage 发送仅用户可见消息
func (p *Plugin) sendEphemeralMessage(channelID, userID, content string) error {
	_, err := p.client.PostEphemeral(
		channelID,
		userID,
		slack.MsgOptionText(content, false),
	)
	return err
}

// addReaction 添加表情反应
func (p *Plugin) addReaction(channelID, ts, reaction string) error {
	item := slack.NewRefToMessage(channelID, ts)
	return p.client.AddReaction(reaction, item)
}

// removeReaction 移除表情反应
func (p *Plugin) removeReaction(channelID, ts, reaction string) error {
	item := slack.NewRefToMessage(channelID, ts)
	return p.client.RemoveReaction(reaction, item)
}

// downloadFile 下载 Slack 文件
func (p *Plugin) downloadFile(file *slack.File) (string, error) {
	if p.store == nil {
		return "", fmt.Errorf("media store not configured")
	}

	// 下载文件
	resp, err := http.Get(file.URLPrivateDownload)
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	// 保存到本地
	if p.cfg.MediaDir == "" {
		p.cfg.MediaDir = "~/.gopaw/media/slack"
	}
	if err := os.MkdirAll(p.cfg.MediaDir, 0755); err != nil {
		return "", fmt.Errorf("create media dir: %w", err)
	}

	localPath := fmt.Sprintf("%s/%s_%s", p.cfg.MediaDir, file.ID, file.Name)
	out, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", fmt.Errorf("save file: %w", err)
	}

	// 存储到媒体库
	mediaURL, err := p.store.Store(localPath, plugin.MediaMeta{
		Filename:    file.Name,
		ContentType: file.Mimetype,
		Source:      "slack",
	}, "slack")
	if err != nil {
		return "", fmt.Errorf("store media: %w", err)
	}

	return mediaURL, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetFile 下载文件（供外部调用）
func (p *Plugin) GetFile(fileID string) (*slack.File, error) {
	file, _, _, err := p.client.GetFileInfo(fileID, 0, 0)
	return file, err
}

// SendMessage 发送消息（带选项）
func (p *Plugin) SendMessage(channelID, content string, options ...slack.MsgOption) error {
	opts := []slack.MsgOption{slack.MsgOptionText(content, false)}
	opts = append(opts, options...)

	_, _, err := p.client.PostMessage(channelID, opts...)
	return err
}
