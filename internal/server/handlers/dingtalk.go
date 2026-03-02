// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/channel"
)

// DingTalkHandler handles DingTalk channel HTTP requests.
type DingTalkHandler struct {
	channelMgr *channel.Manager
}

// NewDingTalkHandler creates a DingTalkHandler.
func NewDingTalkHandler(m *channel.Manager) *DingTalkHandler {
	return &DingTalkHandler{channelMgr: m}
}

// Event handles POST /dingtalk/event — DingTalk pushes events to this endpoint.
func (h *DingTalkHandler) Event(c *gin.Context) {
	// Use GetActivePlugin to ensure plugin is started
	p, err := h.channelMgr.GetActivePlugin("dingtalk")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "dingtalk channel not available"})
		return
	}

	// Interface assertion for HTTP handling
	handler, ok := p.(HTTPHandler)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "plugin does not support HTTP handling"})
		return
	}
	handler.HandleReceive(c.Writer, c.Request, "")
}
