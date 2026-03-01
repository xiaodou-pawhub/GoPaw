// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// AgentHandler handles /api/agent routes.
type AgentHandler struct {
	agent  *agent.ReActAgent
	logger *zap.Logger
}

// NewAgentHandler creates an AgentHandler.
func NewAgentHandler(a *agent.ReActAgent, logger *zap.Logger) *AgentHandler {
	return &AgentHandler{agent: a, logger: logger}
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
func (h *AgentHandler) ChatStream(c *gin.Context) {
	sessionID := c.Query("session_id")
	content := c.Query("content")
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

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
