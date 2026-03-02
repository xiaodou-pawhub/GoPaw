// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/settings"
	"go.uber.org/zap"
)

// SettingsHandler handles /api/settings routes for runtime configuration
// (LLM providers, channel secrets, agent persona).
type SettingsHandler struct {
	store       *settings.Store
	agentMDPath string
	logger      *zap.Logger
	channelMgr  *channel.Manager
}

// NewSettingsHandler creates a SettingsHandler.
func NewSettingsHandler(store *settings.Store, agentMDPath string, channelMgr *channel.Manager, logger *zap.Logger) *SettingsHandler {
	return &SettingsHandler{store: store, agentMDPath: agentMDPath, channelMgr: channelMgr, logger: logger}
}

// ── LLM Providers ──────────────────────────────────────────────────────────

// ListProviders handles GET /api/settings/providers
func (h *SettingsHandler) ListProviders(c *gin.Context) {
	list, err := h.store.ListProviders()
	if err != nil {
		h.logger.Error("settings: list providers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"providers": list})
}

// SaveProvider handles POST /api/settings/providers (create or update)
func (h *SettingsHandler) SaveProvider(c *gin.Context) {
	var p settings.ProviderConfig
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if p.Name == "" || p.BaseURL == "" || p.APIKey == "" || p.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, base_url, api_key and model are required"})
		return
	}
	if err := h.store.SaveProvider(&p); err != nil {
		h.logger.Error("settings: save provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": p.ID})
}

// SetActiveProvider handles PUT /api/settings/providers/:id/active
func (h *SettingsHandler) SetActiveProvider(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.SetActiveProvider(id); err != nil {
		h.logger.Error("settings: set active provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"active": id})
}

// DeleteProvider handles DELETE /api/settings/providers/:id
func (h *SettingsHandler) DeleteProvider(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.DeleteProvider(id); err != nil {
		h.logger.Error("settings: delete provider", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// ── Channel Configs ────────────────────────────────────────────────────────

// GetChannelConfig handles GET /api/settings/channels/:name
func (h *SettingsHandler) GetChannelConfig(c *gin.Context) {
	name := c.Param("name")
	cfg, err := h.store.GetChannelConfig(name)
	if err != nil {
		h.logger.Error("settings: get channel config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Return raw JSON as string to avoid double-encoding
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, `{"channel":%q,"config":%s}`, name, cfg)
}

// SetChannelConfig handles PUT /api/settings/channels/:name
func (h *SettingsHandler) SetChannelConfig(c *gin.Context) {
	name := c.Param("name")
	var body struct {
		Config string `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.store.SetChannelConfig(name, body.Config); err != nil {
		h.logger.Error("settings: set channel config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 热重载插件：用新配置重新初始化
	// Note: Reinit uses process-level context, not request context
	if h.channelMgr != nil {
		if err := h.channelMgr.Reinit(name, []byte(body.Config)); err != nil {
			// 重载失败只记录日志，不影响配置保存的成功响应
			h.logger.Error("channel reinit failed after config save",
				zap.String("name", name),
				zap.Error(err),
			)
		}
	}

	c.JSON(http.StatusOK, gin.H{"channel": name})
}

// ── Agent Persona (AGENT.md) ───────────────────────────────────────────────

// GetAgentMD handles GET /api/settings/agent
func (h *SettingsHandler) GetAgentMD(c *gin.Context) {
	content, err := settings.ReadAgentMD(h.agentMDPath)
	if err != nil {
		h.logger.Error("settings: read AGENT.md", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": content})
}

// SetAgentMD handles PUT /api/settings/agent
func (h *SettingsHandler) SetAgentMD(c *gin.Context) {
	var body struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := settings.WriteAgentMD(h.agentMDPath, body.Content); err != nil {
		h.logger.Error("settings: write AGENT.md", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"saved": true})
}

// SetupStatus handles GET /api/settings/setup-status
// Returns whether initial setup (LLM configuration) is still required.
func (h *SettingsHandler) SetupStatus(c *gin.Context) {
	llmConfigured := !h.store.IsSetupRequired()
	c.JSON(http.StatusOK, gin.H{
		"llm_configured": llmConfigured,
		"setup_required": !llmConfigured,
		"hint":           map[bool]string{true: "", false: "请在 Web UI → 设置 → LLM 提供商 中配置 LLM 才能开始对话"}[llmConfigured],
	})
}
