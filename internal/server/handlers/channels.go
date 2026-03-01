// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/channel"
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
	c.JSON(http.StatusOK, gin.H{"channels": out})
}
