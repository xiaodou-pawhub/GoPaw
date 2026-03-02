// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// AgentHandler handles /api/agent routes.
type AgentHandler struct {
	agent  *agent.ReActAgent
	mem    *memory.Manager
	logger *zap.Logger
}

// NewAgentHandler creates an AgentHandler.
func NewAgentHandler(a *agent.ReActAgent, mem *memory.Manager, logger *zap.Logger) *AgentHandler {
	return &AgentHandler{agent: a, mem: mem, logger: logger}
}

type chatRequest struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content" binding:"required"`
	MsgType   string `json:"msg_type"`
}

type chatResponse struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	MsgType   string `json:"msg_type"`
	ElapsedMs int64  `json:"elapsed_ms"`
}

// Chat handles POST /api/agent/chat.
func (h *AgentHandler) Chat(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msgType := types.MsgTypeText
	if req.MsgType != "" {
		msgType = types.MessageType(req.MsgType)
	}

	agentReq := &types.Request{
		SessionID: req.SessionID,
		Content:   req.Content,
		MsgType:   msgType,
		Channel:   "console",
	}

	start := time.Now()
	resp, err := h.agent.Process(c.Request.Context(), agentReq)
	if err != nil {
		h.logger.Error("agent chat error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chatResponse{
		SessionID: req.SessionID,
		Content:   resp.Content,
		MsgType:   string(resp.MsgType),
		ElapsedMs: time.Since(start).Milliseconds(),
	})
}

// ChatStream handles GET /api/agent/chat/stream using Server-Sent Events.
// Deprecated: Use ChatStreamPost for large content support.
// 中文：已废弃，请使用 ChatStreamPost 支持大内容
// English: Deprecated, use ChatStreamPost for large content support
func (h *AgentHandler) ChatStream(c *gin.Context) {
	sessionID := c.Query("session_id")
	content := c.Query("content")
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	h.processStream(c, sessionID, content)
}

// ChatStreamPost handles POST /api/agent/chat/stream using Server-Sent Events.
// Supports large content in request body (e.g., file attachments).
// 中文：POST 流式对话，支持大内容（如文件附件）
// English: POST streaming chat, supports large content (e.g., file attachments)
func (h *AgentHandler) ChatStreamPost(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.processStream(c, req.SessionID, req.Content)
}

// processStream is the common logic for streaming responses.
// 中文：流式响应的通用逻辑
// English: Common logic for streaming responses
func (h *AgentHandler) processStream(c *gin.Context, sessionID, content string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	agentReq := &types.Request{
		SessionID: sessionID,
		Content:   content,
		MsgType:   types.MsgTypeText,
		Channel:   "console",
	}

	resp, err := h.agent.Process(c.Request.Context(), agentReq)
	if err != nil {
		fmt.Fprintf(c.Writer, "data: {\"error\":%q}\n\n", err.Error())
		c.Writer.Flush()
		return
	}

	// Simulate streaming by sending the full response as a single delta.
	// Real streaming requires the LLM client's Stream method to be wired here.
	fmt.Fprintf(c.Writer, "data: {\"delta\":%q}\n\n", resp.Content)
	fmt.Fprintf(c.Writer, "data: {\"done\":true}\n\n")
	c.Writer.Flush()
}

// ListSessions handles GET /api/agent/sessions.
func (h *AgentHandler) ListSessions(c *gin.Context) {
	sessions := h.agent.Sessions().All()
	type sessionInfo struct {
		ID        string `json:"id"`
		UserID    string `json:"user_id"`
		Channel   string `json:"channel"`
		CreatedAt int64  `json:"created_at"`
		UpdatedAt int64  `json:"updated_at"`
	}
	out := make([]sessionInfo, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, sessionInfo{
			ID:        s.ID,
			UserID:    s.UserID,
			Channel:   s.Channel,
			CreatedAt: s.CreatedAt.UnixMilli(),
			UpdatedAt: s.UpdatedAt.UnixMilli(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"sessions": out})
}

// DeleteSession handles DELETE /api/agent/sessions/:id.
func (h *AgentHandler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}
	if err := h.mem.Clear(sessionID); err != nil {
		h.logger.Error("failed to delete session", zap.String("id", sessionID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除会话失败 / Failed to delete session"})
		return
	}
	h.agent.Sessions().Delete(sessionID)
	c.JSON(http.StatusOK, gin.H{"ok": true, "deleted_session_id": sessionID})
}

// GetSessionStats handles GET /api/agent/sessions/:id/stats.
func (h *AgentHandler) GetSessionStats(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}

	count, total, user, assist, err := h.mem.GetSessionStats(sessionID)
	if err != nil {
		h.logger.Error("failed to get session stats", zap.String("id", sessionID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计数据失败 / Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":    sessionID,
		"message_count": count,
		"total_tokens":  total,
		"user_tokens":   user,
		"assist_tokens": assist,
	})
}


// GetSessionMessages handles GET /api/agent/sessions/:id/messages.
// Returns the message history for a given session, oldest-first.
// Optional query param: limit (default 100, max 500).
func (h *AgentHandler) GetSessionMessages(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session id is required"})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			if v > 500 {
				v = 500
			}
			limit = v
		}
	}

	msgs, err := h.mem.GetContext(sessionID, limit)
	if err != nil {
		h.logger.Error("agent: get session messages", zap.String("session_id", sessionID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type messageItem struct {
		Role      string `json:"role"`
		Content   string `json:"content"`
		CreatedAt int64  `json:"created_at"`
	}
	out := make([]messageItem, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, messageItem{
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt.UnixMilli(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"session_id": sessionID, "messages": out, "total": len(out)})
}
