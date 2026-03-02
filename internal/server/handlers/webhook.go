// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/plugins/channels/webhook"
)

// WebhookHandler handles Webhook channel HTTP requests.
type WebhookHandler struct {
	channelMgr *channel.Manager
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(m *channel.Manager) *WebhookHandler {
	return &WebhookHandler{channelMgr: m}
}

// Receive handles POST /webhook/:token — external systems push messages to Agent.
func (h *WebhookHandler) Receive(c *gin.Context) {
	token := c.Param("token")
	p, err := h.channelMgr.GetPlugin("webhook")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook channel not available"})
		return
	}
	wp, ok := p.(*webhook.Plugin)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid plugin type"})
		return
	}
	wp.HandleReceive(c.Writer, c.Request, token)
}

// Poll handles GET /webhook/:token/messages — external systems poll for Agent responses.
func (h *WebhookHandler) Poll(c *gin.Context) {
	token := c.Param("token")
	p, err := h.channelMgr.GetPlugin("webhook")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook channel not available"})
		return
	}
	wp, ok := p.(*webhook.Plugin)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid plugin type"})
		return
	}
	wp.HandlePoll(c.Writer, c.Request, token)
}
