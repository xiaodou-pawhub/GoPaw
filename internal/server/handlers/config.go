// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/config"
	"go.uber.org/zap"
)

// ConfigHandler handles /api/config routes.
// Note: LLM provider config, channel secrets, and agent persona are managed
// via /api/settings — see SettingsHandler.
type ConfigHandler struct {
	manager *config.Manager
	logger  *zap.Logger
}

// NewConfigHandler creates a ConfigHandler.
func NewConfigHandler(m *config.Manager, logger *zap.Logger) *ConfigHandler {
	return &ConfigHandler{manager: m, logger: logger}
}

// Get handles GET /api/config — returns the current static startup configuration.
func (h *ConfigHandler) Get(c *gin.Context) {
	cfg := h.manager.Get()
	c.JSON(http.StatusOK, gin.H{
		"app":       cfg.App,
		"server":    cfg.Server,
		"workspace": cfg.Workspace,
		"agent":     cfg.Agent,
		"skills":    cfg.Skills,
		"log":       cfg.Log,
	})
}
