// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/agent/message"
	"go.uber.org/zap"
)

// AgentMessageHandler handles agent-to-agent message HTTP requests.
type AgentMessageHandler struct {
	manager *message.Manager
	logger  *zap.Logger
}

// NewAgentMessageHandler creates a new agent message handler.
func NewAgentMessageHandler(manager *message.Manager, logger *zap.Logger) *AgentMessageHandler {
	return &AgentMessageHandler{
		manager: manager,
		logger:  logger.Named("agent_message_handler"),
	}
}

// SendMessageRequest represents a request to send a message.
type SendMessageRequest struct {
	Type      string                 `json:"type" binding:"required"` // task/response/notify/query/result
	FromAgent string                 `json:"from_agent" binding:"required"`
	ToAgent   string                 `json:"to_agent" binding:"required"`
	Content   string                 `json:"content" binding:"required"`
	Payload   map[string]interface{} `json:"payload"`
	ParentID  string                 `json:"parent_id"`
}

// SendMessage sends a message from one agent to another.
func (h *AgentMessageHandler) SendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msgType := message.MessageType(req.Type)
	if !isValidMessageType(msgType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message type"})
		return
	}

	msg, err := message.NewMessage(msgType, req.FromAgent, req.ToAgent, req.Content, req.Payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	msg.ParentID = req.ParentID

	if err := h.manager.Send(msg); err != nil {
		h.logger.Error("failed to send message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// SendTaskRequest represents a request to send a task.
type SendTaskRequest struct {
	FromAgent   string                 `json:"from_agent" binding:"required"`
	ToAgent     string                 `json:"to_agent" binding:"required"`
	Description string                 `json:"description" binding:"required"`
	TaskID      string                 `json:"task_id"`
	Priority    string                 `json:"priority"` // low, normal, high, urgent
	Data        map[string]interface{} `json:"data"`
}

// SendTask sends a task message.
func (h *AgentMessageHandler) SendTask(c *gin.Context) {
	var req SendTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payload := &message.TaskPayload{
		TaskID:      req.TaskID,
		Description: req.Description,
		Priority:    req.Priority,
		Data:        req.Data,
	}

	msg, err := h.manager.SendTask(req.FromAgent, req.ToAgent, req.Description, payload)
	if err != nil {
		h.logger.Error("failed to send task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// SendResponseRequest represents a request to send a response.
type SendResponseRequest struct {
	FromAgent string                 `json:"from_agent" binding:"required"`
	ToAgent   string                 `json:"to_agent" binding:"required"`
	InReplyTo string                 `json:"in_reply_to" binding:"required"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
}

// SendResponse sends a response message.
func (h *AgentMessageHandler) SendResponse(c *gin.Context) {
	var req SendResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.manager.SendResponse(req.FromAgent, req.ToAgent, req.InReplyTo, req.Success, req.Message, req.Data)
	if err != nil {
		h.logger.Error("failed to send response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// SendNotifyRequest represents a request to send a notification.
type SendNotifyRequest struct {
	FromAgent string                 `json:"from_agent" binding:"required"`
	ToAgent   string                 `json:"to_agent" binding:"required"`
	Event     string                 `json:"event" binding:"required"`
	Details   map[string]interface{} `json:"details"`
}

// SendNotify sends a notification message.
func (h *AgentMessageHandler) SendNotify(c *gin.Context) {
	var req SendNotifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.manager.SendNotify(req.FromAgent, req.ToAgent, req.Event, req.Details)
	if err != nil {
		h.logger.Error("failed to send notify", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// SendQueryRequest represents a request to send a query.
type SendQueryRequest struct {
	FromAgent string                 `json:"from_agent" binding:"required"`
	ToAgent   string                 `json:"to_agent" binding:"required"`
	Question  string                 `json:"question" binding:"required"`
	Context   map[string]interface{} `json:"context"`
}

// SendQuery sends a query message.
func (h *AgentMessageHandler) SendQuery(c *gin.Context) {
	var req SendQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.manager.SendQuery(req.FromAgent, req.ToAgent, req.Question, req.Context)
	if err != nil {
		h.logger.Error("failed to send query", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// GetMessage returns a specific message.
func (h *AgentMessageHandler) GetMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.manager.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msg)
}

// ListMessages lists messages for an agent.
func (h *AgentMessageHandler) ListMessages(c *gin.Context) {
	agentID := c.Param("agent_id")
	limit := 50
	status := c.Query("status")

	messages, err := h.manager.ListMessages(agentID, limit, status)
	if err != nil {
		h.logger.Error("failed to list messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// ListSentMessages lists messages sent by an agent.
func (h *AgentMessageHandler) ListSentMessages(c *gin.Context) {
	agentID := c.Param("agent_id")
	limit := 50

	messages, err := h.manager.ListSentMessages(agentID, limit)
	if err != nil {
		h.logger.Error("failed to list sent messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// ListConversation lists messages in a conversation.
func (h *AgentMessageHandler) ListConversation(c *gin.Context) {
	parentID := c.Param("parent_id")

	messages, err := h.manager.ListConversation(parentID)
	if err != nil {
		h.logger.Error("failed to list conversation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// UpdateStatusRequest represents a request to update message status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"` // pending/processing/completed/failed
	Error  string `json:"error"`
}

// UpdateStatus updates the status of a message.
func (h *AgentMessageHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := message.MessageStatus(req.Status)
	if !isValidMessageStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	if err := h.manager.UpdateStatus(id, status, req.Error); err != nil {
		h.logger.Error("failed to update status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

// GetPendingMessages returns pending messages for an agent.
func (h *AgentMessageHandler) GetPendingMessages(c *gin.Context) {
	agentID := c.Param("agent_id")
	limit := 50

	messages, err := h.manager.GetPendingMessages(agentID, limit)
	if err != nil {
		h.logger.Error("failed to get pending messages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// GetStats returns message statistics for an agent.
func (h *AgentMessageHandler) GetStats(c *gin.Context) {
	agentID := c.Param("agent_id")

	stats, err := h.manager.GetStats(agentID)
	if err != nil {
		h.logger.Error("failed to get stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ListConversations lists conversations for an agent.
func (h *AgentMessageHandler) ListConversations(c *gin.Context) {
	agentID := c.Param("agent_id")
	limit := 50

	conversations, err := h.manager.ListConversations(agentID, limit)
	if err != nil {
		h.logger.Error("failed to list conversations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conversations)
}

// isValidMessageType checks if a message type is valid.
func isValidMessageType(t message.MessageType) bool {
	switch t {
	case message.MessageTypeTask, message.MessageTypeResponse, message.MessageTypeNotify, message.MessageTypeQuery, message.MessageTypeResult:
		return true
	}
	return false
}

// isValidMessageStatus checks if a message status is valid.
func isValidMessageStatus(s message.MessageStatus) bool {
	switch s {
	case message.MessageStatusPending, message.MessageStatusProcessing, message.MessageStatusCompleted, message.MessageStatusFailed:
		return true
	}
	return false
}
