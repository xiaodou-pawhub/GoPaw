// Package feishu implements the Feishu (Lark) channel plugin for GoPaw.
package feishu

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
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/renderer"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"
)

const (
	tokenEndpoint    = "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal"
	uploadEndpoint   = "https://open.feishu.cn/open-apis/im/v1/images"
	sendEndpoint     = "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"
	messageEndpoint  = "https://open.feishu.cn/open-apis/im/v1/messages"
	tokenRefreshSkew = 5 * time.Minute
	defaultTimeout   = 30 * time.Second
)

type Config struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Domain    string `json:"domain"`
}

type Plugin struct {
	cfg        Config
	logger     *zap.Logger
	httpClient *http.Client
	deduper    *Deduper
	store      plugin.MediaStore

	// reactionCache maps "messageID:reactionType" -> "reactionID"
	reactionCache sync.Map

	inbound    chan *types.Message
	ctx        context.Context
	cancelFunc context.CancelFunc

	mu        sync.RWMutex
	connected bool
	started   time.Time

	tokenMu     sync.RWMutex
	cachedToken string
	tokenExpiry time.Time

	configured bool
}

// 确保实现所有必需的接口
var _ plugin.MediaStoreReceiver = (*Plugin)(nil)
var _ plugin.TypingCapable = (*Plugin)(nil)
var _ plugin.ReactionCapable = (*Plugin)(nil)
var _ plugin.MessageEditor = (*Plugin)(nil)
var _ plugin.PlaceholderCapable = (*Plugin)(nil)
var _ plugin.ApprovalUI = (*Plugin)(nil)

func New() *Plugin {
	return &Plugin{
		inbound: make(chan *types.Message, 100),
		deduper: NewDeduper(30 * time.Minute),
	}
}

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 100),
		deduper: NewDeduper(30 * time.Minute),
	})
}

func (p *Plugin) Name() string        { return "feishu" }
func (p *Plugin) DisplayName() string { return "飞书" }

func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.feishu")
	p.httpClient = &http.Client{Timeout: defaultTimeout}
	if p.deduper == nil {
		p.deduper = NewDeduper(30 * time.Minute)
	}

	if len(cfg) > 0 && string(cfg) != "{}" {
		if err := json.Unmarshal(cfg, &p.cfg); err != nil {
			p.logger.Warn("feishu: failed to parse config", zap.Error(err))
		}
	}
	if p.cfg.AppID == "" || p.cfg.AppSecret == "" {
		return plugin.ErrMissingCredentials
	}
	if p.cfg.Domain == "" {
		p.cfg.Domain = "feishu.cn"
	}
	p.configured = true
	return nil
}

func (p *Plugin) SetMediaStore(s plugin.MediaStore) {
	p.mu.Lock()
	p.store = s
	p.mu.Unlock()
}

func (p *Plugin) Start(ctx context.Context) error {
	if !p.configured {
		return nil
	}
	p.started = time.Now()
	p.ctx, p.cancelFunc = context.WithCancel(ctx)

	eventHandler := dispatcher.NewEventDispatcher("", "")

	// 1. Handle regular messages
	eventHandler.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		p.handleIncomingMessage(event)
		return nil
	})

	// 2. Handle card action callbacks - Following 2025-03-17 Official Docs
	eventHandler.OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
		p.logger.Info("feishu: card action received", zap.String("id", event.Event.Context.OpenMessageID))

		actionData := event.Event.Action.Value
		actionType, _ := actionData["action"].(string)

		if actionType == "tool_approve" {
			reqID, _ := actionData["request_id"].(string)
			verdict, _ := actionData["verdict"].(string)

			p.logger.Info("feishu: tool approval resolve", zap.String("req_id", reqID), zap.String("verdict", verdict))

			if err := tool.GlobalApprovalStore.Resolve(reqID, tool.ApprovalVerdict(verdict)); err != nil {
				p.logger.Warn("feishu: failed to resolve approval", zap.Error(err))
			}

			// Get approval context for user-friendly display
			approvalCtx := tool.GlobalApprovalStore.GetApprovalContext(reqID)
			
			// Return updated card in callback response (Card 2.0 raw type).
			// Per Feishu docs, PATCH API must NOT be called before responding to the callback —
			// doing so causes the update to fail/revert. The response IS the update.
			statusCard := buildApprovalStatusCard(verdict, reqID, approvalCtx)
			toastType, toastContent := "success", "已批准"
			if verdict == string(tool.VerdictDenied) {
				toastType, toastContent = "error", "已拒绝"
			}
			return &callback.CardActionTriggerResponse{
				Card: &callback.Card{
					Type: "raw",
					Data: statusCard,
				},
				Toast: &callback.Toast{
					Type:    toastType,
					Content: toastContent,
				},
			}, nil
		}

		return nil, nil
	})

	wsClient := larkws.NewClient(p.cfg.AppID, p.cfg.AppSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)

	go func() {
		p.mu.Lock()
		p.connected = true
		p.mu.Unlock()

		if err := wsClient.Start(p.ctx); err != nil && p.ctx.Err() == nil {
			p.logger.Error("feishu ws client failed", zap.Error(err))
		}

		p.mu.Lock()
		p.connected = false
		p.mu.Unlock()
	}()

	return nil
}

func (p *Plugin) Stop() error {
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
	return nil
}

func (p *Plugin) Receive() <-chan *types.Message {
	return p.inbound
}

func (p *Plugin) Send(msg *types.Message) error {
	_, err := p.sendInternal(context.Background(), msg, false)
	return err
}

// SendWithMessageID sends a message and returns the message ID.
// This is useful for later editing the message (e.g., approval cards).
func (p *Plugin) SendWithMessageID(msg *types.Message) (string, error) {
	return p.sendInternal(context.Background(), msg, false)
}

func (p *Plugin) replyText(messageID, text string) {
	msg := &types.Message{
		Channel: "feishu",
		ChatID:  messageID,
		Content: text,
		MsgType: types.MsgTypeText,
	}
	_, _ = p.sendInternal(context.Background(), msg, false)
}

func (p *Plugin) sendInternal(ctx context.Context, msg *types.Message, isCard bool) (string, error) {
	token, err := p.getToken()
	if err != nil {
		return "", err
	}

	var cardMsgID string
	if msg.Content != "" || isCard || msg.MsgType == types.MsgTypeMarkdown {
		content := msg.Content
		if !isCard && msg.MsgType != types.MsgTypeMarkdown {
			blocks := renderer.ParseMarkdown(msg.Content)
			card, _ := BuildCard("🤖 智能回复", blocks, "success")
			content = card
		}

		payload := map[string]interface{}{
			"receive_id": msg.ChatID,
			"msg_type":   "interactive",
			"content":    content,
		}

		body, _ := json.Marshal(payload)
		resp, err := p.postWithToken(ctx, sendEndpoint, body, token)
		if err != nil {
			p.logger.Error("feishu send message failed", zap.Error(err))
			return "", err
		}
		
		var res struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				MessageId string `json:"message_id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &res); err != nil {
			p.logger.Error("feishu send response parse failed", zap.Error(err), zap.String("resp", string(resp)))
			return "", err
		}
		
		if res.Code != 0 {
			p.logger.Error("feishu send api error", zap.Int("code", res.Code), zap.String("msg", res.Msg))
			return "", fmt.Errorf("feishu api error: code=%d, msg=%s", res.Code, res.Msg)
		}
		
		cardMsgID = res.Data.MessageId
		p.logger.Info("feishu message sent successfully", zap.String("msg_id", cardMsgID))
	}

	for _, f := range msg.Files {
		localPath, err := p.store.Resolve(f.URL)
		if err != nil {
			continue
		}

		isImage := strings.Contains(strings.ToLower(f.Name), ".png") ||
			strings.Contains(strings.ToLower(f.Name), ".jpg")

		var msgType string
		var contentMap map[string]string

		if isImage {
			key, err := p.uploadImage(ctx, localPath)
			if err != nil {
				continue
			}
			msgType = "image"
			contentMap = map[string]string{"image_key": key}
		} else {
			key, err := p.uploadFile(ctx, localPath, f.Name)
			if err != nil {
				continue
			}
			msgType = "file"
			contentMap = map[string]string{"file_key": key}
		}

		contentBody, _ := json.Marshal(contentMap)
		payload := map[string]interface{}{
			"receive_id": msg.ChatID,
			"msg_type":   msgType,
			"content":    string(contentBody),
		}

		sendBody, _ := json.Marshal(payload)
		_, _ = p.postWithToken(ctx, sendEndpoint, sendBody, token)
	}

	return cardMsgID, nil
}

func (p *Plugin) postWithToken(ctx context.Context, url string, body []byte, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return respBody, nil
}

func (p *Plugin) uploadImage(ctx context.Context, path string) (string, error) {
	token, err := p.getToken()
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.WriteField("image_type", "message")
	part, _ := w.CreateFormFile("image", filepath.Base(path))
	io.Copy(part, file)
	w.Close()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, uploadEndpoint, &b)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Code int `json:"code"`
		Data struct {
			ImageKey string `json:"image_key"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	if res.Code != 0 {
		return "", fmt.Errorf("upload failed: %d", res.Code)
	}
	return res.Data.ImageKey, nil
}

func (p *Plugin) uploadFile(ctx context.Context, path, filename string) (string, error) {
	token, _ := p.getToken()
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.WriteField("file_type", "stream")
	w.WriteField("file_name", filename)
	part, _ := w.CreateFormFile("file", filename)
	io.Copy(part, file)
	w.Close()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://open.feishu.cn/open-apis/im/v1/files", &b)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Code int `json:"code"`
		Data struct {
			FileKey string `json:"file_key"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	if res.Code != 0 {
		return "", fmt.Errorf("file upload failed: %d", res.Code)
	}
	return res.Data.FileKey, nil
}

func (p *Plugin) getToken() (string, error) {
	p.tokenMu.RLock()
	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry.Add(-tokenRefreshSkew)) {
		t := p.cachedToken
		p.tokenMu.RUnlock()
		return t, nil
	}
	p.tokenMu.RUnlock()

	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	if p.cachedToken != "" && time.Now().Before(p.tokenExpiry.Add(-tokenRefreshSkew)) {
		return p.cachedToken, nil
	}

	payload := map[string]string{"app_id": p.cfg.AppID, "app_secret": p.cfg.AppSecret}
	b, _ := json.Marshal(payload)
	resp, err := p.httpClient.Post(tokenEndpoint, "application/json", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		AppAccessToken string `json:"app_access_token"`
		Expire         int    `json:"expire"`
		Code           int    `json:"code"`
		Msg            string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to decode token response: %w (body: %s)", err, string(body))
	}

	if res.Code != 0 {
		return "", fmt.Errorf("token request failed: code=%d, msg=%s", res.Code, res.Msg)
	}

	if res.AppAccessToken == "" {
		return "", fmt.Errorf("empty app_access_token in response")
	}

	p.cachedToken = res.AppAccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(res.Expire) * time.Second)
	return p.cachedToken, nil
}

func (p *Plugin) Health() plugin.HealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return plugin.HealthStatus{Running: p.connected, Since: p.started}
}

func (p *Plugin) Test(ctx context.Context) plugin.TestResult {
	_, err := p.getToken()
	if err != nil {
		return plugin.TestResult{Success: false, Message: err.Error()}
	}
	return plugin.TestResult{Success: true, Message: "OK"}
}

func (p *Plugin) handleIncomingMessage(event *larkim.P2MessageReceiveV1) {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return
	}

	msgData := event.Event.Message
	msgID := strVal(msgData.MessageId)
	
	// 使用 deduper 去重
	if p.deduper.Seen(msgID) {
		return
	}

	msgType := strVal(msgData.MessageType)
	
	var text string
	var files []types.FileAttachment

	switch msgType {
	case "text":
		var content map[string]string
		if err := json.Unmarshal([]byte(strVal(msgData.Content)), &content); err == nil {
			text = content["text"]
		}
	case "image", "file":
		p.mu.RLock()
		store := p.store
		p.mu.RUnlock()

		if store != nil {
			localPath, meta, err := p.downloadResource(msgID, msgData)
			if err == nil {
				ref, storeErr := store.Store(localPath, plugin.MediaMeta{
					Filename:    meta.Filename,
					ContentType: meta.ContentType,
					Source:      "feishu",
				}, strVal(msgData.ChatId))
				if storeErr == nil {
					files = append(files, types.FileAttachment{
						Name: meta.Filename,
						URL:  ref,
					})
					text = fmt.Sprintf("[媒体消息：%s]", meta.Filename)
				}
			}
		}
	}

	peerKind := types.PeerDirect
	if msgData.ChatType != nil && *msgData.ChatType == "group" {
		peerKind = types.PeerGroup
	}

	userID := ""
	if event.Event.Sender != nil && event.Event.Sender.SenderId != nil {
		userID = strVal(event.Event.Sender.SenderId.OpenId)
	}

	msg := &types.Message{
		ID:          msgID,
		ChatID:      strVal(msgData.ChatId),
		UserID:      userID,
		Channel:     p.Name(),
		Content:     text,
		MsgType:     types.MessageType(msgType),
		Timestamp:   time.Now().UnixMilli(),
		IsMentioned: len(msgData.Mentions) > 0,
		ThreadID:    strVal(msgData.ThreadId),
		PeerKind:    peerKind,
		Files:       files,
	}

	select {
	case p.inbound <- msg:
	default:
		p.logger.Warn("feishu: inbound queue full")
	}
}

// strVal safely dereferences a *string to a string, returning "" if nil.
func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ── Capabilities Implementation ─────────────────────────────────────────────

// StartTyping is a no-op for Feishu as it doesn't support typing indicators.
func (p *Plugin) StartTyping(ctx context.Context, chatID string) (func(), error) {
	return func() {}, nil
}

// AddReaction adds an emoji reaction to a message.
func (p *Plugin) AddReaction(ctx context.Context, chatID, messageID, reactionType string) error {
	emoji := p.mapReaction(reactionType)
	if emoji == "" {
		return nil
	}
	
	token, err := p.getToken()
	if err != nil {
		return err
	}
	
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/reactions", messageID)
	
	payload := map[string]interface{}{
		"reaction_type": map[string]string{
			"emoji_type": emoji,
		},
	}
	body, _ := json.Marshal(payload)
	
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("feishu add reaction failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var res struct {
			Data struct {
				ReactionId string `json:"reaction_id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && res.Data.ReactionId != "" {
			// Store the reaction_id so we can remove it later
			key := fmt.Sprintf("%s:%s", messageID, reactionType)
			p.reactionCache.Store(key, res.Data.ReactionId)
		}
	} else {
		respBody, _ := io.ReadAll(resp.Body)
		p.logger.Warn("feishu add reaction api error", 
			zap.Int("status", resp.StatusCode), 
			zap.String("body", string(respBody)))
	}
	return nil
}

// RemoveReaction removes a previously added reaction.
func (p *Plugin) RemoveReaction(ctx context.Context, chatID, messageID, reactionType string) error {
	key := fmt.Sprintf("%s:%s", messageID, reactionType)
	val, ok := p.reactionCache.LoadAndDelete(key)
	if !ok {
		return nil // Nothing to remove
	}
	reactionID := val.(string)

	token, err := p.getToken()
	if err != nil {
		return err
	}
	
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/reactions/%s", messageID, reactionID)
	
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("feishu remove reaction failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		p.logger.Warn("feishu remove reaction api error", 
			zap.Int("status", resp.StatusCode), 
			zap.String("body", string(respBody)))
	}
	return nil
}

// SendPlaceholder sends a placeholder "thinking" card.
func (p *Plugin) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	card, _ := BuildPlaceholderCard()
	msg := &types.Message{
		ChatID:  chatID,
		Channel: p.Name(),
		Content: card,
		MsgType: types.MsgTypeMarkdown,
	}
	return p.sendInternal(ctx, msg, true)
}

// EditMessage edits a previously sent message.
func (p *Plugin) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	if messageID == "" {
		return fmt.Errorf("feishu edit: empty messageID")
	}
	
	blocks := renderer.ParseMarkdown(content)
	p.processOutboundImages(ctx, blocks)

	card, err := BuildCard("🤖 智能回复", blocks, "success")
	if err != nil {
		return err
	}

	token, err := p.getToken()
	if err != nil {
		return err
	}
	
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s", messageID)
	// Feishu PATCH expects the card JSON inside the "content" field
	payload := map[string]string{"content": card}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Error("feishu edit request failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		p.logger.Warn("feishu edit api error", 
			zap.Int("status", resp.StatusCode), 
			zap.String("body", string(respBody)))
		return fmt.Errorf("feishu api returned %d", resp.StatusCode)
	}

	p.logger.Info("feishu message updated successfully", zap.String("msg_id", messageID))
	return nil
}

// processOutboundImages converts media:// URLs to Feishu image_keys.
func (p *Plugin) processOutboundImages(ctx context.Context, blocks []renderer.MessageBlock) {
	for i, b := range blocks {
		if b.Type == renderer.BlockImage && strings.HasPrefix(b.Content, "media://") {
			localPath, err := p.store.Resolve(b.Content)
			if err == nil {
				imageKey, err := p.uploadImage(ctx, localPath)
				if err == nil {
					blocks[i].Content = imageKey
				}
			}
		}
	}
}

// downloadResource downloads a file or image from Feishu.
func (p *Plugin) downloadResource(msgID string, msgData *larkim.EventMessage) (string, plugin.MediaMeta, error) {
	msgType := strVal(msgData.MessageType)
	var fileKey string
	var filename string
	
	var content map[string]string
	if err := json.Unmarshal([]byte(strVal(msgData.Content)), &content); err != nil {
		return "", plugin.MediaMeta{}, err
	}

	if msgType == "image" {
		fileKey = content["image_key"]
		filename = "image.png"
	} else {
		fileKey = content["file_key"]
		filename = content["file_name"]
	}

	token, _ := p.getToken()
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s/resources/%s?type=%s", 
		msgID, fileKey, msgType)
	
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := p.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", plugin.MediaMeta{}, fmt.Errorf("feishu api error")
	}
	defer resp.Body.Close()

	tmpPath := p.store.TempPath(filepath.Ext(filename))
	f, _ := os.Create(tmpPath)
	defer f.Close()
	io.Copy(f, resp.Body)

	return tmpPath, plugin.MediaMeta{Filename: filename, ContentType: resp.Header.Get("Content-Type")}, nil
}

// mapReaction maps standard reaction types to Feishu-specific emoji.
func (p *Plugin) mapReaction(rt string) string {
	switch rt {
	case plugin.ReactionWait:
		return "Get" // 飞书中的"了解/Get"表情
	case plugin.ReactionSuccess:
		return "DONE" // 飞书中的"完成"表情
	case plugin.ReactionError:
		return "WRONG" // 飞书中的"错误"表情
	default:
		return ""
	}
}

// buildApprovalStatusCard builds a status card shown after user approves/denies/times out.
func buildApprovalStatusCard(verdict string, reqID string, approvalCtx *tool.ApprovalContext) map[string]interface{} {
	var title, template, content string

	switch verdict {
	case string(tool.VerdictAllowed):
		title = "✅ 已批准"
		template = "green"
		if approvalCtx != nil {
			content = fmt.Sprintf(
				"**%s**\n\n%s\n状态：正在执行中...",
				title,
				approvalCtx.ToolDisplay,
			)
		} else {
			content = fmt.Sprintf("**工具执行请求已批准**\n\n请求 ID: `%s`\n正在执行工具，请稍候...", reqID)
		}
	case string(tool.VerdictTimeout):
		title = "⏱️ 超时未响应"
		template = "orange"
		if approvalCtx != nil {
			content = fmt.Sprintf(
				"**%s**\n\n%s\n超过 5 分钟未响应，已自动拒绝。",
				title,
				approvalCtx.ToolDisplay,
			)
		} else {
			content = fmt.Sprintf("**工具执行请求已超时**\n\n请求 ID: `%s`\n超过 5 分钟未响应，已自动拒绝。", reqID)
		}
	default: // denied
		title = "❌ 已拒绝"
		template = "red"
		if approvalCtx != nil {
			content = fmt.Sprintf(
				"**%s**\n\n%s\n操作已根据用户要求取消。",
				title,
				approvalCtx.ToolDisplay,
			)
		} else {
			content = fmt.Sprintf("**工具执行请求已拒绝**\n\n请求 ID: `%s`\n用户已手动拒绝该请求。", reqID)
		}
	}

	// Cleanup context after displaying
	if approvalCtx != nil {
		tool.GlobalApprovalStore.CleanupApprovalContext(reqID)
	}

	return map[string]interface{}{
		"schema": "2.0",
		"header": map[string]interface{}{
			"title":    map[string]string{"tag": "plain_text", "content": title},
			"template": template,
		},
		"body": map[string]interface{}{
			"elements": []interface{}{
				map[string]interface{}{
					"tag":     "markdown",
					"content": content,
				},
			},
		},
	}
}
