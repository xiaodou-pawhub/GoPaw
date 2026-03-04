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
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"go.uber.org/zap"
)

const (
	tokenEndpoint    = "https://open.feishu.cn/open-apis/auth/v3/app_access_token/internal"
	sendEndpoint     = "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id"
	uploadEndpoint   = "https://open.feishu.cn/open-apis/im/v1/images"
	tokenRefreshSkew = 5 * time.Minute
	defaultTimeout   = 30 * time.Second
)

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 256),
		deduper: NewDeduper(30 * time.Minute),
	})
}

type feishuConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	Domain    string `json:"domain"` // feishu.cn or larksuite.com
}

type Plugin struct {
	cfg        feishuConfig
	inbound    chan *types.Message
	started    time.Time
	configured bool
	logger     *zap.Logger
	httpClient *http.Client
	deduper    *Deduper
	store      plugin.MediaStore

	// reactionCache maps "messageID:reactionType" -> "reactionID"
	reactionCache sync.Map

	ctx        context.Context
	cancelFunc context.CancelFunc
	connected  bool
	mu         sync.RWMutex

	tokenMu     sync.RWMutex
	cachedToken string
	tokenExpiry time.Time
}

var _ plugin.MediaStoreReceiver = (*Plugin)(nil)
var _ plugin.TypingCapable = (*Plugin)(nil)
var _ plugin.ReactionCapable = (*Plugin)(nil)
var _ plugin.MessageEditor = (*Plugin)(nil)
var _ plugin.PlaceholderCapable = (*Plugin)(nil)

func (p *Plugin) Name() string        { return "feishu" }
func (p *Plugin) DisplayName() string { return "飞书" }

func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.feishu")
	p.httpClient = &http.Client{Timeout: defaultTimeout}

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
	eventHandler.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		p.handleIncomingMessage(event)
		return nil
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

func (p *Plugin) handleIncomingMessage(event *larkim.P2MessageReceiveV1) {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return
	}

	msgData := event.Event.Message
	msgID := strVal(msgData.MessageId)
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
					text = fmt.Sprintf("[媒体消息: %s]", meta.Filename)
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

func (p *Plugin) StartTyping(ctx context.Context, chatID string) (func(), error) {
	return func() {}, nil
}

func (p *Plugin) AddReaction(ctx context.Context, chatID, messageID, reactionType string) error {
	emoji := p.mapReaction(reactionType)
	if emoji == "" {
		return nil
	}
	token, _ := p.getToken()
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

func (p *Plugin) RemoveReaction(ctx context.Context, chatID, messageID, reactionType string) error {
	key := fmt.Sprintf("%s:%s", messageID, reactionType)
	val, ok := p.reactionCache.LoadAndDelete(key)
	if !ok {
		return nil // Nothing to remove
	}
	reactionID := val.(string)

	token, _ := p.getToken()
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

func (p *Plugin) EditMessage(ctx context.Context, chatID, messageID, content string) error {
	blocks := renderer.ParseMarkdown(content)
	p.processOutboundImages(ctx, blocks)

	card, err := BuildCard("🤖 智能回复", blocks, "success")
	if err != nil {
		return err
	}

	token, _ := p.getToken()
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/im/v1/messages/%s", messageID)
	payload := map[string]string{"content": card}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (p *Plugin) Send(msg *types.Message) error {
	_, err := p.sendInternal(context.Background(), msg, false)
	return err
}

func (p *Plugin) sendInternal(ctx context.Context, msg *types.Message, isCard bool) (string, error) {
	token, err := p.getToken()
	if err != nil {
		return "", err
	}

	var content string
	msgType := "interactive"

	if isCard || msg.MsgType == types.MsgTypeMarkdown {
		content = msg.Content
	} else {
		blocks := renderer.ParseMarkdown(msg.Content)
		p.processOutboundImages(ctx, blocks)
		card, _ := BuildCard("🤖 智能回复", blocks, "success")
		content = card
	}

	payload := map[string]interface{}{
		"receive_id": msg.ChatID,
		"msg_type":   msgType,
		"content":    content,
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, sendEndpoint, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Data struct {
			MessageId string `json:"message_id"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	return res.Data.MessageId, nil
}

// ── Multi-modal Support ─────────────────────────────────────────────────────

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

func (p *Plugin) uploadImage(ctx context.Context, path string) (string, error) {
	token, _ := p.getToken()
	file, _ := os.Open(path)
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

// ── Helpers ────────────────────────────────────────────────────────────────

func (p *Plugin) mapReaction(rt string) string {
	switch rt {
	case plugin.ReactionWait:
		return "Get" // 飞书中的“了解/Get”表情
	case plugin.ReactionSuccess:
		return "DONE" // 飞书中的“完成”表情
	case plugin.ReactionError:
		return "WRONG" // 飞书中的“错误”表情
	default:
		return ""
	}
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
	if err != nil { return "", err }
	defer resp.Body.Close()

	var res struct {
		AppAccessToken string `json:"app_access_token"`
		Expire         int    `json:"expire"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
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
	if err != nil { return plugin.TestResult{Success: false, Message: err.Error()} }
	return plugin.TestResult{Success: true, Message: "OK"}
}
