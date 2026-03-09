package dingtalk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"go.uber.org/zap"

	"github.com/gopaw/gopaw/internal/channel"
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
	channel.RegisterChannelPlugin("dingtalk", func() channel.ChannelPlugin {
		return &Plugin{
			inbound: make(chan *types.Message, 100),
		}
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
	p.logger.Info("dingtalk plugin started")

	return nil
}

// Stop 停止插件
func (p *Plugin) Stop() error {
	if !p.running {
		return nil
	}

	p.running = false
	close(p.inbound)

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
func (p *Plugin) Health() channel.ChannelHealth {
	if !p.running {
		return channel.ChannelHealth{
			Healthy: false,
			Status:  "stopped",
		}
	}

	return channel.ChannelHealth{
		Healthy: true,
		Status:  "running",
	}
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
			ID:        data.MessageId,
			Channel:   "dingtalk",
			UserID:    data.SenderId,
			Content:   data.Text.Content,
			Timestamp: time.Now().Unix(),
			SessionID: fmt.Sprintf("dingtalk:%s", data.ConversationId),
			Metadata: map[string]string{
				"conversation_id": data.ConversationId,
				"sender_id":       data.SenderId,
				"message_type":    data.MessageType,
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

// StoreSessionWebhook 存储 Session Webhook
func (p *Plugin) StoreSessionWebhook(sessionID, webhook string) {
	p.sessionWebhooks.Store(sessionID, webhook)
	p.logger.Debug("stored session webhook",
		zap.String("session_id", sessionID),
		zap.String("webhook", truncateString(webhook, 50)+"..."))
}

// GetSessionWebhook 获取 Session Webhook
func (p *Plugin) GetSessionWebhook(sessionID string) (string, bool) {
	webhook, ok := p.sessionWebhooks.Load(sessionID)
	if !ok {
		return "", false
	}
	return webhook.(string), true
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
