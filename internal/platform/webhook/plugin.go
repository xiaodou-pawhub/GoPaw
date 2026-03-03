// Package webhook implements a simple Webhook channel plugin for GoPaw.
// It pushes messages to a configured webhook URL.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// webhookConfig holds the webhook configuration.
type webhookConfig struct {
	// URL is the target webhook endpoint to push messages to.
	URL string `json:"url"`
}

// Plugin implements the Webhook channel.
type Plugin struct {
	cfg     webhookConfig
	inbound chan *types.Message
	started time.Time
	logger  *zap.Logger
}

func (p *Plugin) Name() string        { return "webhook" }
func (p *Plugin) DisplayName() string { return "Webhook" }

// Init parses the webhook configuration.
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.webhook")
	p.inbound = make(chan *types.Message, 256)
	
	if err := json.Unmarshal(cfg, &p.cfg); err != nil {
		return fmt.Errorf("webhook: parse config: %w", err)
	}
	if p.cfg.URL == "" {
		return plugin.ErrMissingCredentials
	}
	return nil
}

// Start activates the webhook channel.
func (p *Plugin) Start(_ context.Context) error {
	p.started = time.Now()
	p.logger.Info("webhook channel started", zap.String("url", p.cfg.URL))
	return nil
}

// Stop gracefully shuts down the plugin.
func (p *Plugin) Stop() error {
	p.logger.Info("webhook channel stopped")
	return nil
}

// Receive returns the inbound message channel (not used for webhook).
func (p *Plugin) Receive() <-chan *types.Message {
	return p.inbound
}

// Send pushes a message to the configured webhook URL.
func (p *Plugin) Send(msg *types.Message) error {
	return p.pushWebhook(msg)
}

// Health returns the current status.
func (p *Plugin) Health() plugin.HealthStatus {
	return plugin.HealthStatus{
		Running: true,
		Message: "ok",
		Since:   p.started,
	}
}

// Test validates the webhook configuration by sending a test message.
func (p *Plugin) Test(ctx context.Context) plugin.TestResult {
	// 检查 URL 配置
	if p.cfg.URL == "" {
		return plugin.TestResult{
			Success: false,
			Message: "请先配置 Webhook URL",
		}
	}

	// 发送测试消息
	if err := p.sendTestMessage(ctx); err != nil {
		return plugin.TestResult{
			Success: false,
			Message: "发送测试消息失败",
			Details: err.Error(),
		}
	}
	
	return plugin.TestResult{
		Success: true,
		Message: "测试消息已发送，请检查 Webhook 接收端是否收到",
	}
}

// sendTestMessage sends a test message to the webhook URL.
func (p *Plugin) sendTestMessage(ctx context.Context) error {
	testMsg := &types.Message{
		ID:        uuid.New().String(),
		SessionID: "test-session",
		UserID:    "system",
		Channel:   p.Name(),
		Content:   "这是一条来自 GoPaw 的测试消息，用于验证 Webhook 配置是否正确。",
		MsgType:   types.MsgTypeText,
		Timestamp: time.Now().UnixMilli(),
	}
	return p.pushWebhook(testMsg)
}

// pushWebhook sends the message to the configured webhook URL.
func (p *Plugin) pushWebhook(msg *types.Message) error {
	// 智能适配不同平台的 Webhook 格式
	var payload []byte
	var err error
	
	if isFeishuWebhook(p.cfg.URL) {
		// 飞书群机器人格式
		payload, err = p.buildFeishuPayload(msg)
	} else {
		// 通用格式
		payload, err = p.buildGenericPayload(msg)
	}
	
	if err != nil {
		return fmt.Errorf("webhook: build payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: http request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体用于错误信息
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: returned status %d, body: %s", resp.StatusCode, string(body))
	}
	
	// 尝试解析响应检查是否有错误（飞书格式）
	var respBody map[string]interface{}
	if err := json.Unmarshal(body, &respBody); err == nil {
		// 飞书：{"code":0,"msg":"ok","data":{}}
		if code, ok := respBody["code"].(float64); ok && code != 0 {
			return fmt.Errorf("webhook: api error code %d, msg: %v", int(code), respBody["msg"])
		}
		// 钉钉：{"errcode":0,"errmsg":"ok"}
		if errcode, ok := respBody["errcode"].(float64); ok && errcode != 0 {
			return fmt.Errorf("webhook: api error code %d, msg: %v", int(errcode), respBody["errmsg"])
		}
	}

	p.logger.Debug("webhook message delivered",
		zap.String("url", p.cfg.URL),
		zap.Int("status", resp.StatusCode),
	)
	
	return nil
}

// buildFeishuPayload builds payload for Feishu webhook.
func (p *Plugin) buildFeishuPayload(msg *types.Message) ([]byte, error) {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": msg.Content,
		},
	}
	return json.Marshal(payload)
}

// buildGenericPayload builds generic JSON payload.
func (p *Plugin) buildGenericPayload(msg *types.Message) ([]byte, error) {
	return json.Marshal(msg)
}

// isFeishuWebhook checks if the URL is a Feishu webhook.
func isFeishuWebhook(url string) bool {
	// 飞书 Webhook URL 特征：https://open.feishu.cn/open-apis/bot/v2/hook/
	return strings.Contains(url, "feishu.cn") || 
		   strings.Contains(url, "open.feishu.cn") ||
		   strings.Contains(url, "/bot/v2/hook/")
}
