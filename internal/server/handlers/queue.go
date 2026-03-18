// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/queue"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// QueueHandler handles queue-related HTTP requests.
type QueueHandler struct {
	mgr    *queue.Manager
	logger *zap.Logger
}

// NewQueueHandler creates a new queue handler.
func NewQueueHandler(mgr *queue.Manager, logger *zap.Logger) *QueueHandler {
	return &QueueHandler{
		mgr:    mgr,
		logger: logger.Named("queue_handler"),
	}
}

// ListQueues returns all queues.
func (h *QueueHandler) ListQueues(c *gin.Context) {
	queues, err := h.mgr.ListQueues()
	if err != nil {
		h.logger.Error("failed to list queues", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to list queues", err)
		return
	}

	// Get stats for each queue
	result := make([]gin.H, len(queues))
	for i, q := range queues {
		stats, err := h.mgr.GetStats(q)
		if err != nil {
			continue
		}
		result[i] = gin.H{
			"name":               q,
			"pending_count":      stats.PendingCount,
			"processing_count":   stats.ProcessingCount,
			"completed_count":    stats.CompletedCount,
			"failed_count":       stats.FailedCount,
			"delayed_count":      stats.DelayedCount,
			"total_count":        stats.TotalCount,
		}
	}

	api.Success(c, result)
}

// GetQueueStats returns statistics for a queue.
func (h *QueueHandler) GetQueueStats(c *gin.Context) {
	queueName := c.Param("name")
	stats, err := h.mgr.GetStats(queueName)
	if err != nil {
		h.logger.Error("failed to get queue stats", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get queue stats", err)
		return
	}

	api.Success(c, stats)
}

// ListMessages returns messages in a queue.
func (h *QueueHandler) ListMessages(c *gin.Context) {
	queueName := c.Param("name")
	status := c.Query("status")
	limit := 50

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	messages, err := h.mgr.ListMessages(queueName, status, limit)
	if err != nil {
		h.logger.Error("failed to list messages", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to list messages", err)
		return
	}

	api.Success(c, messages)
}

// PublishMessageRequest represents a request to publish a message.
type PublishMessageRequest struct {
	Type       string                 `json:"type" binding:"required"`
	Payload    map[string]interface{} `json:"payload" binding:"required"`
	Priority   int                    `json:"priority,omitempty"`
	MaxRetries int                    `json:"max_retries,omitempty"`
	Delay      int                    `json:"delay_seconds,omitempty"` // seconds
}

// PublishMessage publishes a message to a queue.
func (h *QueueHandler) PublishMessage(c *gin.Context) {
	queueName := c.Param("name")

	var req PublishMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request body", err)
		return
	}

	opts := &queue.PublishOptions{
		Priority:   req.Priority,
		MaxRetries: req.MaxRetries,
	}

	if req.Delay > 0 {
		opts.Delay = time.Duration(req.Delay) * time.Second
	}

	msg, err := h.mgr.Publish(queueName, req.Type, req.Payload, opts)
	if err != nil {
		h.logger.Error("failed to publish message", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to publish message", err)
		return
	}

	api.Created(c, msg)
}

// GetMessage returns a message by ID.
func (h *QueueHandler) GetMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.mgr.GetMessage(id)
	if err != nil {
		api.NotFound(c, "message")
		return
	}

	api.Success(c, msg)
}

// RetryMessage retries a failed message.
func (h *QueueHandler) RetryMessage(c *gin.Context) {
	id := c.Param("id")

	// Get message first
	msg, err := h.mgr.GetMessage(id)
	if err != nil {
		api.NotFound(c, "message")
		return
	}

	if !msg.IsRetryable() {
		api.BadRequest(c, "message cannot be retried")
		return
	}

	if err := h.mgr.Retry(id); err != nil {
		h.logger.Error("failed to retry message", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to retry message", err)
		return
	}

	api.Success(c, gin.H{"message": "message queued for retry"})
}

// DeleteMessage deletes a message.
func (h *QueueHandler) DeleteMessage(c *gin.Context) {
	id := c.Param("id")

	if err := h.mgr.Delete(id); err != nil {
		h.logger.Error("failed to delete message", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to delete message", err)
		return
	}

	api.Success(c, gin.H{"message": "message deleted"})
}

// PauseQueue pauses a queue (not implemented yet).
func (h *QueueHandler) PauseQueue(c *gin.Context) {
	// TODO: Implement queue pause/resume
	api.Success(c, gin.H{"message": "queue pause not implemented yet"})
}

// ResumeQueue resumes a queue (not implemented yet).
func (h *QueueHandler) ResumeQueue(c *gin.Context) {
	// TODO: Implement queue pause/resume
	api.Success(c, gin.H{"message": "queue resume not implemented yet"})
}

// CleanupQueue deletes old messages from a queue.
func (h *QueueHandler) CleanupQueue(c *gin.Context) {
	queueName := c.Param("name")
	status := c.Query("status")

	if status == "" {
		api.BadRequest(c, "status parameter required")
		return
	}

	// Delete messages with the specified status
	// This is a simple implementation - in production, you might want to add date filtering
	messages, err := h.mgr.ListMessages(queueName, status, 10000)
	if err != nil {
		h.logger.Error("failed to list messages for cleanup", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to list messages for cleanup", err)
		return
	}

	deletedCount := 0
	for _, msg := range messages {
		if err := h.mgr.Delete(msg.ID); err == nil {
			deletedCount++
		}
	}

	api.Success(c, gin.H{
		"message":         "queue cleaned up",
		"deleted_count":   deletedCount,
		"status":          status,
	})
}
