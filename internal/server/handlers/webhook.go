// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/channel"
)

// HTTPHandler is the interface for channel plugins that handle HTTP requests directly.
// This provides interface isolation - handlers depend on an interface, not concrete types.
type HTTPHandler interface {
	Name() string
	HandleReceive(w http.ResponseWriter, r *http.Request, token string)
	HandlePoll(w http.ResponseWriter, r *http.Request, token string)
}

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

	// Use GetActivePlugin to ensure plugin is started
	p, err := h.channelMgr.GetActivePlugin("webhook")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook channel not available"})
		return
	}

	// Interface assertion instead of concrete type
	handler, ok := p.(HTTPHandler)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "plugin does not support HTTP handling"})
		return
	}
	handler.HandleReceive(c.Writer, c.Request, token)
}

// Poll handles GET /webhook/:token/messages — external systems poll for Agent responses.
func (h *WebhookHandler) Poll(c *gin.Context) {
	token := c.Param("token")

	// Use GetActivePlugin to ensure plugin is started
	p, err := h.channelMgr.GetActivePlugin("webhook")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook channel not available"})
		return
	}

	// Interface assertion instead of concrete type
	handler, ok := p.(HTTPHandler)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "plugin does not support HTTP handling"})
		return
	}
	handler.HandlePoll(c.Writer, c.Request, token)
}
