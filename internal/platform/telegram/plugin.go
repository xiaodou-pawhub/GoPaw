package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
)

// Plugin Telegram 频道插件
type Plugin struct {
	cfg *Config

	// HTTP 客户端
	httpClient *http.Client

	// 消息通道
	inbound chan *types.Message

	// 媒体存储
	store plugin.MediaStore

	// 状态
	running   bool
	started   time.Time
	lastUpdateID int

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	logger *zap.Logger
}

// Config Telegram 配置
type Config struct {
	Enabled   bool   `json:"enabled"`
	BotToken  string `json:"bot_token"`
	HTTPProxy string `json:"http_proxy"`
	MediaDir  string `json:"media_dir"`
}

// Update Telegram 更新
type Update struct {
	UpdateID int       `json:"update_id"`
	Message  *Message  `json:"message,omitempty"`
}

// Message Telegram 消息
type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text,omitempty"`
	Photo     []PhotoSize `json:"photo,omitempty"`
	Document  *Document `json:"document,omitempty"`
	Video     *Video    `json:"video,omitempty"`
	Audio     *Audio    `json:"audio,omitempty"`
	Voice     *Voice    `json:"voice,omitempty"`
}

// User Telegram 用户
type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

// Chat Telegram 聊天
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// PhotoSize 照片尺寸
type PhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FileSize int    `json:"file_size,omitempty"`
}

// Document 文档
type Document struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// Video 视频
type Video struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Duration int    `json:"duration"`
}

// Audio 音频
type Audio struct {
	FileID   string `json:"file_id"`
	Duration int    `json:"duration"`
}

// Voice 语音
type Voice struct {
	FileID   string `json:"file_id"`
	Duration int    `json:"duration"`
}

// File Telegram 文件
type File struct {
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	FileSize int    `json:"file_size"`
}

// API 响应
type APIResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Description string          `json:"description,omitempty"`
}

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 100),
	})
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "telegram"
}

// DisplayName 返回显示名称
func (p *Plugin) DisplayName() string {
	return "Telegram"
}

// Init 初始化插件
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.cfg = &Config{}
	if err := json.Unmarshal(cfg, p.cfg); err != nil {
		return fmt.Errorf("parse telegram config: %w", err)
	}

	p.logger = zap.L().Named("telegram")
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
	// Telegram 暂不支持表情反应（需要 Bot API 6.1+）
	p.logger.Debug("telegram reaction not implemented")
	return nil
}

// RemoveReaction 移除表情反应（实现 ReactionCapable 接口）
func (p *Plugin) RemoveReaction(channelID, messageTS, reaction string) error {
	return nil
}

// EditMessage 编辑消息（实现 MessageEditor 接口）
func (p *Plugin) EditMessage(channelID, messageTS, newContent string) error {
	token := p.cfg.BotToken
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token)

	params := map[string]string{
		"chat_id":    channelID,
		"message_id": messageTS,
		"text":       newContent,
	}

	_, err := p.callAPI(url, params)
	return err
}

// DeleteMessage 删除消息（实现 MessageEditor 接口）
func (p *Plugin) DeleteMessage(channelID, messageTS string) error {
	token := p.cfg.BotToken
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", token)

	params := map[string]string{
		"chat_id":    channelID,
		"message_id": messageTS,
	}

	_, err := p.callAPI(url, params)
	return err
}

// SendMarkdown 发送 Markdown 消息（实现 RichTextCapable 接口）
func (p *Plugin) SendMarkdown(channelID, markdown string) error {
	token := p.cfg.BotToken
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	params := map[string]string{
		"chat_id":      channelID,
		"text":         markdown,
		"parse_mode":   "MarkdownV2",
	}

	_, err := p.callAPI(url, params)
	return err
}

// SendHTML 发送 HTML 消息（实现 RichTextCapable 接口）
func (p *Plugin) SendHTML(channelID, html string) error {
	token := p.cfg.BotToken
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	params := map[string]string{
		"chat_id":      channelID,
		"text":         html,
		"parse_mode":   "HTML",
	}

	_, err := p.callAPI(url, params)
	return err
}

// SendFile 发送文件（实现 FileCapable 接口）
func (p *Plugin) SendFile(channelID, filePath, caption string) error {
	return p.sendMediaFile(channelID, filePath, caption, "document")
}

// SendImage 发送图片（实现 FileCapable 接口）
func (p *Plugin) SendImage(channelID, imagePath, caption string) error {
	return p.sendMediaFile(channelID, imagePath, caption, "photo")
}

// SendVideo 发送视频（实现 FileCapable 接口）
func (p *Plugin) SendVideo(channelID, videoPath, caption string) error {
	return p.sendMediaFile(channelID, videoPath, caption, "video")
}

// SendAudio 发送音频（实现 FileCapable 接口）
func (p *Plugin) SendAudio(channelID, audioPath, caption string) error {
	return p.sendMediaFile(channelID, audioPath, caption, "audio")
}

// sendMediaFile 发送媒体文件（内部方法）
func (p *Plugin) sendMediaFile(channelID, filePath, caption, mediaType string) error {
	token := p.cfg.BotToken
	url := fmt.Sprintf("https://api.telegram.org/bot%s/send%s", token, capitalize(mediaType))

	// 创建 multipart form
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// 添加 chat_id
	if err := w.WriteField("chat_id", channelID); err != nil {
		return fmt.Errorf("write chat_id: %w", err)
	}

	// 添加 caption
	if caption != "" {
		if err := w.WriteField("caption", caption); err != nil {
			return fmt.Errorf("write caption: %w", err)
		}
	}

	// 添加文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	part, err := w.CreateFormFile(mediaType, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	// 发送请求
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("telegram API error: %s", apiResp.Description)
	}

	return nil
}

// Start 启动插件
func (p *Plugin) Start(ctx context.Context) error {
	if !p.cfg.Enabled {
		p.logger.Info("telegram plugin disabled")
		return nil
	}

	if p.cfg.BotToken == "" {
		return fmt.Errorf("telegram bot_token required")
	}

	p.ctx, p.cancel = context.WithCancel(ctx)
	p.running = true
	p.started = time.Now()

	p.logger.Info("starting telegram plugin")

	// 获取 Bot 信息
	botInfo, err := p.getMe()
	if err != nil {
		return fmt.Errorf("getMe: %w", err)
	}
	p.logger.Info("telegram bot info",
		zap.String("username", botInfo.Username),
		zap.String("first_name", botInfo.FirstName))

	// 创建媒体目录
	if p.cfg.MediaDir != "" {
		if err := os.MkdirAll(p.cfg.MediaDir, 0755); err != nil {
			p.logger.Warn("create media dir failed", zap.Error(err))
		}
	}

	// 启动长轮询
	go p.longPollingLoop()

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

	p.logger.Info("telegram plugin stopped")
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

	chatID := msg.ChatID
	if chatID == "" {
		return fmt.Errorf("no chat_id specified")
	}

	// 发送文本消息
	if msg.Content != "" {
		if err := p.sendMessage(chatID, msg.Content); err != nil {
			return fmt.Errorf("send message: %w", err)
		}
	}

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

	// 测试 getMe
	bot, err := p.getMe()
	if err != nil {
		return plugin.TestResult{
			Success: false,
			Message: "failed to get bot info: " + err.Error(),
		}
	}

	return plugin.TestResult{
		Success: true,
		Message: "connected",
		Details: fmt.Sprintf("bot: @%s", bot.Username),
	}
}

// longPollingLoop 长轮询循环
func (p *Plugin) longPollingLoop() {
	for p.running {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		updates, err := p.getUpdates(p.lastUpdateID + 1)
		if err != nil {
			p.logger.Error("get updates failed", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.UpdateID > p.lastUpdateID {
				p.lastUpdateID = update.UpdateID
			}

			if update.Message != nil {
				p.handleMessage(update.Message)
			}
		}
	}
}

// handleMessage 处理消息
func (p *Plugin) handleMessage(msg *Message) {
	// 跳过机器人自己的消息
	if msg.From != nil && msg.From.Username != "" {
		// 检查是否是机器人自己
		// TODO: 缓存 Bot 用户名
	}

	// 跳过空消息
	if msg.Text == "" && len(msg.Photo) == 0 && msg.Document == nil {
		return
	}

	p.logger.Debug("received message",
		zap.Int64("chat_id", msg.Chat.ID),
		zap.String("from", msg.From.Username),
		zap.String("text", msg.Text))

	// 构建消息
	message := &types.Message{
		ID:        fmt.Sprintf("telegram_%d", msg.MessageID),
		Channel:   "telegram",
		UserID:    fmt.Sprintf("%d", msg.From.ID),
		ChatID:    fmt.Sprintf("%d", msg.Chat.ID),
		Content:   msg.Text,
		Timestamp: time.Now().Unix(),
		SessionID: fmt.Sprintf("telegram:%d", msg.Chat.ID),
		Metadata: map[string]string{
			"message_id": fmt.Sprintf("%d", msg.MessageID),
			"chat_type":  msg.Chat.Type,
			"username":   msg.From.Username,
		},
	}

	// 处理媒体文件
	if len(msg.Photo) > 0 {
		// 获取最大尺寸的照片
		photo := msg.Photo[len(msg.Photo)-1]
		if err := p.downloadAndStoreMedia(photo.FileID, message); err != nil {
			p.logger.Warn("download photo failed", zap.Error(err))
		}
	}

	if msg.Document != nil {
		if err := p.downloadAndStoreMedia(msg.Document.FileID, message); err != nil {
			p.logger.Warn("download document failed", zap.Error(err))
		}
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

// downloadAndStoreMedia 下载并存储媒体文件
func (p *Plugin) downloadAndStoreMedia(fileID string, msg *types.Message) error {
	if p.store == nil {
		return fmt.Errorf("media store not configured")
	}

	// 获取文件信息
	file, err := p.getFile(fileID)
	if err != nil {
		return fmt.Errorf("get file: %w", err)
	}

	// 下载文件
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", p.cfg.BotToken, file.FilePath)
	resp, err := p.httpClient.Get(fileURL)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	// 保存到临时文件
	if p.cfg.MediaDir == "" {
		p.cfg.MediaDir = "~/.gopaw/media/telegram"
	}
	if err := os.MkdirAll(p.cfg.MediaDir, 0755); err != nil {
		return fmt.Errorf("create media dir: %w", err)
	}

	filename := fmt.Sprintf("%s_%s", msg.ID, filepath.Base(file.FilePath))
	localPath := filepath.Join(p.cfg.MediaDir, filename)

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	// 存储到媒体库
	mediaURL, err := p.store.Store(localPath, plugin.MediaMeta{
		Filename:    filename,
		ContentType: "application/octet-stream",
		Source:      "telegram",
	}, "telegram")
	if err != nil {
		return fmt.Errorf("store media: %w", err)
	}

	// 添加到消息内容
	msg.Content += fmt.Sprintf("\n[Media: %s]", mediaURL)

	return nil
}

// getMe 获取 Bot 信息
func (p *Plugin) getMe() (*User, error) {
	resp, err := p.callAPI("getMe", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(resp, &user); err != nil {
		return nil, fmt.Errorf("unmarshal user: %w", err)
	}

	return &user, nil
}

// getUpdates 获取更新
func (p *Plugin) getUpdates(offset int) ([]Update, error) {
	params := map[string]string{
		"offset":  fmt.Sprintf("%d", offset),
		"timeout": "30",
		"limit":   "100",
	}

	resp, err := p.callAPI("getUpdates", params)
	if err != nil {
		return nil, err
	}

	var updates []Update
	if err := json.Unmarshal(resp, &updates); err != nil {
		return nil, fmt.Errorf("unmarshal updates: %w", err)
	}

	return updates, nil
}

// getFile 获取文件信息
func (p *Plugin) getFile(fileID string) (*File, error) {
	params := map[string]string{
		"file_id": fileID,
	}

	resp, err := p.callAPI("getFile", params)
	if err != nil {
		return nil, err
	}

	var file File
	if err := json.Unmarshal(resp, &file); err != nil {
		return nil, fmt.Errorf("unmarshal file: %w", err)
	}

	return &file, nil
}

// sendMessage 发送消息
func (p *Plugin) sendMessage(chatID, text string) error {
	params := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}

	_, err := p.callAPI("sendMessage", params)
	return err
}

// callAPI 调用 Telegram API
func (p *Plugin) callAPI(method string, params map[string]string) (json.RawMessage, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", p.cfg.BotToken, method)

	var req *http.Request
	var err error

	if params != nil {
		// POST form
		form := make(map[string][]string)
		for k, v := range params {
			form[k] = []string{v}
		}
		
		// 手动构建 query string
		query := make([]string, 0, len(params))
		for k, v := range params {
			query = append(query, fmt.Sprintf("%s=%s", k, v))
		}
		body := strings.Join(query, "&")

		req, err = http.NewRequest("POST", url, strings.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("telegram API error (%d): %s", apiResp.ErrorCode, apiResp.Description)
	}

	return apiResp.Result, nil
}

// capitalize 首字母大写
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// sendMedia 发送媒体消息
func (p *Plugin) sendMedia(chatID, filePath, caption string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", p.cfg.BotToken)

	// 创建 multipart form
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// 添加 chat_id
	if err := w.WriteField("chat_id", chatID); err != nil {
		return fmt.Errorf("write chat_id: %w", err)
	}

	// 添加 caption
	if caption != "" {
		if err := w.WriteField("caption", caption); err != nil {
			return fmt.Errorf("write caption: %w", err)
		}
	}

	// 添加文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	part, err := w.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close writer: %w", err)
	}

	// 发送请求
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("telegram API error: %s", apiResp.Description)
	}

	return nil
}
