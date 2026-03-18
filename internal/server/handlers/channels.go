// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// ChannelsHandler handles /api/channels routes.
type ChannelsHandler struct {
	manager *channel.Manager
	logger  *zap.Logger
}

// NewChannelsHandler creates a ChannelsHandler.
func NewChannelsHandler(m *channel.Manager, logger *zap.Logger) *ChannelsHandler {
	return &ChannelsHandler{manager: m, logger: logger}
}

type channelStatus struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
	Message string `json:"message"`
	Since   int64  `json:"since"`
}

// Health handles GET /api/channels/health.
func (h *ChannelsHandler) Health(c *gin.Context) {
	statuses := h.manager.Health()
	out := make([]channelStatus, 0, len(statuses))
	for name, hs := range statuses {
		var since int64
		if !hs.Since.IsZero() {
			since = hs.Since.UnixMilli()
		} else {
			since = time.Now().UnixMilli()
		}
		out = append(out, channelStatus{
			Name:    name,
			Running: hs.Running,
			Message: hs.Message,
			Since:   since,
		})
	}
	api.Success(c, gin.H{"channels": out})
}

// Test handles POST /api/channels/:name/test.
// It triggers a connection test for the specified channel plugin.
func (h *ChannelsHandler) Test(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		api.BadRequest(c, "channel name is required")
		return
	}

	result, err := h.manager.Test(c.Request.Context(), name)
	if err != nil {
		api.NotFound(c, "channel")
		return
	}

	// 中文：Details 包含敏感错误信息，只写日志不返回前端
	// English: Details contains sensitive error info, log only, don't return to frontend
	if result.Details != "" {
		h.logger.Warn("channel test failed",
			zap.String("channel", name),
			zap.String("message", result.Message),
			zap.String("details", result.Details),
		)
	}

	api.Success(c, gin.H{
		"success": result.Success,
		"message": result.Message,
	})
}
