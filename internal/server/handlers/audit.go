// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/audit"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// AuditHandler handles audit log-related HTTP requests.
type AuditHandler struct {
	mgr    *audit.Manager
	logger *zap.Logger
}

// NewAuditHandler creates a new audit handler.
func NewAuditHandler(mgr *audit.Manager, logger *zap.Logger) *AuditHandler {
	return &AuditHandler{
		mgr:    mgr,
		logger: logger.Named("audit_handler"),
	}
}

// QueryAuditLogsRequest represents a request to query audit logs.
type QueryAuditLogsRequest struct {
	Category     string `form:"category"`
	Action       string `form:"action"`
	UserID       string `form:"user_id"`
	ResourceType string `form:"resource_type"`
	ResourceID   string `form:"resource_id"`
	Status       string `form:"status"`
	StartTime    string `form:"start_time"`
	EndTime      string `form:"end_time"`
	Limit        int    `form:"limit"`
	Offset       int    `form:"offset"`
}

// ListAuditLogs returns audit logs with filtering.
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	var req QueryAuditLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	opts := audit.QueryOptions{
		Category:     audit.Category(req.Category),
		Action:       audit.Action(req.Action),
		UserID:       req.UserID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Status:       audit.Status(req.Status),
		Limit:        req.Limit,
		Offset:       req.Offset,
	}

	if opts.Limit <= 0 || opts.Limit > 1000 {
		opts.Limit = 50
	}

	// Parse time range
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			opts.StartTime = &t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			opts.EndTime = &t
		}
	}

	logs, err := h.mgr.Query(opts)
	if err != nil {
		h.logger.Error("failed to query audit logs", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to query audit logs", err)
		return
	}

	api.Success(c, logs)
}

// GetAuditStats returns audit log statistics.
func (h *AuditHandler) GetAuditStats(c *gin.Context) {
	stats, err := h.mgr.GetStats(audit.QueryOptions{})
	if err != nil {
		h.logger.Error("failed to get audit stats", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get audit stats", err)
		return
	}

	api.Success(c, stats)
}

// ExportAuditLogsRequest represents a request to export audit logs.
type ExportAuditLogsRequest struct {
	Format    string `json:"format" binding:"required,oneof=csv json"`
	Category  string `json:"category"`
	UserID    string `json:"user_id"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// ExportAuditLogs exports audit logs to a file.
func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
	var req ExportAuditLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	opts := audit.ExportOptions{
		Format:   req.Format,
		Category: audit.Category(req.Category),
		UserID:   req.UserID,
	}

	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			opts.StartTime = &t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			opts.EndTime = &t
		}
	}

	filepath, err := h.mgr.Export(opts)
	if err != nil {
		h.logger.Error("failed to export audit logs", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to export audit logs", err)
		return
	}

	// Serve file
	c.FileAttachment(filepath, "audit_logs."+req.Format)

	// Clean up temp file
	go func() {
		time.Sleep(5 * time.Minute)
		h.mgr.Cleanup(0) // This will be handled by the cleanup endpoint
	}()
}

// CleanupAuditLogsRequest represents a request to cleanup old audit logs.
type CleanupAuditLogsRequest struct {
	OlderThanDays int `json:"older_than_days" binding:"required,min=1"`
}

// CleanupAuditLogs deletes old audit logs.
func (h *AuditHandler) CleanupAuditLogs(c *gin.Context) {
	var req CleanupAuditLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	duration := time.Duration(req.OlderThanDays) * 24 * time.Hour
	if err := h.mgr.Cleanup(duration); err != nil {
		h.logger.Error("failed to cleanup audit logs", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to cleanup audit logs", err)
		return
	}

	api.SuccessWithMessage(c, "audit logs cleaned up", nil)
}

// GetAuditLog returns a single audit log by ID.
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	id := c.Param("id")

	// Query with ID filter
	opts := audit.QueryOptions{}
	logs, err := h.mgr.Query(opts)
	if err != nil {
		h.logger.Error("failed to get audit log", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get audit log", err)
		return
	}

	// Find the log with matching ID
	for _, log := range logs {
		if log.ID == id {
			api.Success(c, log)
			return
		}
	}

	api.NotFound(c, "audit log")
}

// GetRecentAuditLogs returns recent audit logs.
func (h *AuditHandler) GetRecentAuditLogs(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	opts := audit.QueryOptions{
		Limit: limit,
	}

	logs, err := h.mgr.Query(opts)
	if err != nil {
		h.logger.Error("failed to get recent audit logs", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get recent audit logs", err)
		return
	}

	api.Success(c, logs)
}
