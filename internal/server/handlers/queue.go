// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/queue"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	c.JSON(http.StatusOK, result)
}

// GetQueueStats returns statistics for a queue.
func (h *QueueHandler) GetQueueStats(c *gin.Context) {
	queueName := c.Param("name")
	stats, err := h.mgr.GetStats(queueName)
	if err != nil {
		h.logger.Error("failed to get queue stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// GetMessage returns a message by ID.
func (h *QueueHandler) GetMessage(c *gin.Context) {
	id := c.Param("id")
	msg, err := h.mgr.GetMessage(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, msg)
}

// RetryMessage retries a failed message.
func (h *QueueHandler) RetryMessage(c *gin.Context) {
	id := c.Param("id")

	// Get message first
	msg, err := h.mgr.GetMessage(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if !msg.IsRetryable() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message cannot be retried"})
		return
	}

	if err := h.mgr.Retry(id); err != nil {
		h.logger.Error("failed to retry message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message queued for retry"})
}

// DeleteMessage deletes a message.
func (h *QueueHandler) DeleteMessage(c *gin.Context) {
	id := c.Param("id")

	if err := h.mgr.Delete(id); err != nil {
		h.logger.Error("failed to delete message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message deleted"})
}

// PauseQueue pauses a queue (not implemented yet).
func (h *QueueHandler) PauseQueue(c *gin.Context) {
	// TODO: Implement queue pause/resume
	c.JSON(http.StatusOK, gin.H{"message": "queue pause not implemented yet"})
}

// ResumeQueue resumes a queue (not implemented yet).
func (h *QueueHandler) ResumeQueue(c *gin.Context) {
	// TODO: Implement queue pause/resume
	c.JSON(http.StatusOK, gin.H{"message": "queue resume not implemented yet"})
}

// CleanupQueue deletes old messages from a queue.
func (h *QueueHandler) CleanupQueue(c *gin.Context) {
	queueName := c.Param("name")
	status := c.Query("status")

	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status parameter required"})
		return
	}

	// Delete messages with the specified status
	// This is a simple implementation - in production, you might want to add date filtering
	messages, err := h.mgr.ListMessages(queueName, status, 10000)
	if err != nil {
		h.logger.Error("failed to list messages for cleanup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	deletedCount := 0
	for _, msg := range messages {
		if err := h.mgr.Delete(msg.ID); err == nil {
			deletedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "queue cleaned up",
		"deleted_count":   deletedCount,
		"status":          status,
	})
}
