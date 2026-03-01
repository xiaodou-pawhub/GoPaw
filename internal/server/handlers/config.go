// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/config"
	"go.uber.org/zap"
)

// ConfigHandler handles /api/config routes.
type ConfigHandler struct {
	manager *config.Manager
	logger  *zap.Logger
}

// NewConfigHandler creates a ConfigHandler.
func NewConfigHandler(m *config.Manager, logger *zap.Logger) *ConfigHandler {
	return &ConfigHandler{manager: m, logger: logger}
}

// Get handles GET /api/config — returns the current (sanitised) configuration.
func (h *ConfigHandler) Get(c *gin.Context) {
	cfg := h.manager.Get()

	// Return a sanitised view that omits secrets.
	safe := gin.H{
		"app":     cfg.App,
		"server":  cfg.Server,
		"storage": cfg.Storage,
		"llm": gin.H{
			"provider":   cfg.LLM.Provider,
			"base_url":   cfg.LLM.BaseURL,
			"model":      cfg.LLM.Model,
			"max_tokens": cfg.LLM.MaxTokens,
			"timeout":    cfg.LLM.TimeoutSeconds,
			// api_key is intentionally omitted
		},
		"agent":   cfg.Agent,
		"plugins": cfg.Plugins,
		"skills":  cfg.Skills,
		"log":     cfg.Log,
	}
	c.JSON(http.StatusOK, safe)
}

// GetLLM handles GET /api/config/llm.
func (h *ConfigHandler) GetLLM(c *gin.Context) {
	cfg := h.manager.Get()
	c.JSON(http.StatusOK, gin.H{
		"provider":   cfg.LLM.Provider,
		"base_url":   cfg.LLM.BaseURL,
		"model":      cfg.LLM.Model,
		"max_tokens": cfg.LLM.MaxTokens,
		"timeout":    cfg.LLM.TimeoutSeconds,
	})
}

// GetAgent handles GET /api/config/agent.
func (h *ConfigHandler) GetAgent(c *gin.Context) {
	cfg := h.manager.Get()
	c.JSON(http.StatusOK, cfg.Agent)
}
