// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/settings"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// SettingsHandler handles /api/settings routes for runtime configuration
// (LLM providers and channel secrets).
type SettingsHandler struct {
	store      *settings.Store
	logger     *zap.Logger
	channelMgr *channel.Manager
}

// NewSettingsHandler creates a SettingsHandler.
func NewSettingsHandler(store *settings.Store, channelMgr *channel.Manager, logger *zap.Logger) *SettingsHandler {
	return &SettingsHandler{store: store, channelMgr: channelMgr, logger: logger}
}

// ── LLM Providers ──────────────────────────────────────────────────────────

// ListProviders handles GET /api/settings/providers
func (h *SettingsHandler) ListProviders(c *gin.Context) {
	// Use new priority-based listing
	list, err := h.store.ListProvidersByPriority()
	if err != nil {
		h.logger.Error("settings: list providers by priority", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to list providers", err)
		return
	}
	api.Success(c, gin.H{"providers": list})
}

// ToggleProvider handles POST /api/settings/providers/:id/toggle
func (h *SettingsHandler) ToggleProvider(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.BadRequestWithError(c, "invalid request body", err)
		return
	}

	if err := h.store.SetProviderEnabled(id, body.Enabled); err != nil {
		h.logger.Error("settings: toggle provider", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to toggle provider", err)
		return
	}

	api.Success(c, gin.H{"id": id, "enabled": body.Enabled})
}

// ReorderProviders handles POST /api/settings/providers/reorder
func (h *SettingsHandler) ReorderProviders(c *gin.Context) {
	var body struct {
		ProviderIDs []string `json:"provider_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.BadRequestWithError(c, "invalid request body", err)
		return
	}

	if err := h.store.ReorderProviders(body.ProviderIDs); err != nil {
		h.logger.Error("settings: reorder providers", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to reorder providers", err)
		return
	}

	api.Success(c, gin.H{"success": true})
}

// GetCapableProviders handles GET /api/settings/providers/capable/:capability
func (h *SettingsHandler) GetCapableProviders(c *gin.Context) {
	capability := c.Param("capability") // e.g., "vision", "multimodal"

	list, err := h.store.GetProvidersByCapability(capability)
	if err != nil {
		h.logger.Error("settings: get capable providers", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get capable providers", err)
		return
	}

	api.Success(c, gin.H{"providers": list})
}

// ListBuiltinProviders handles GET /api/settings/builtin-providers
func (h *SettingsHandler) ListBuiltinProviders(c *gin.Context) {
	api.Success(c, gin.H{"providers": llm.BuiltinProviders})
}

// GetProvidersHealth handles GET /api/settings/providers/health
func (h *SettingsHandler) GetProvidersHealth(c *gin.Context) {
	list, err := h.store.ListProviders()
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list providers", err)
		return
	}

	type healthInfo struct {
		ID            string             `json:"id"`
		Status        llm.ProviderStatus `json:"status"`
		LastError     string             `json:"last_error"`
		CooldownUntil int64              `json:"cooldown_until"` // unix ms
	}

	results := make([]healthInfo, 0, len(list))
	for _, p := range list {
		status, lastErr, until := llm.GlobalHealthTracker.GetStatus(p.ID)
		results = append(results, healthInfo{
			ID:            p.ID,
			Status:        status,
			LastError:     lastErr,
			CooldownUntil: until.UnixMilli(),
		})
	}

	api.Success(c, gin.H{"health": results})
}

// SaveProvider handles POST /api/settings/providers (create or update)
func (h *SettingsHandler) SaveProvider(c *gin.Context) {
	var p settings.ProviderConfig
	if err := c.ShouldBindJSON(&p); err != nil {
		api.BadRequestWithError(c, "invalid request body", err)
		return
	}
	if p.Name == "" || p.BaseURL == "" || p.APIKey == "" || p.Model == "" {
		api.BadRequest(c, "name, base_url, api_key and model are required")
		return
	}

	// 自动推断标签逻辑：如果用户没有手动设置标签，则根据模型名推断
	if len(p.Tags) == 0 {
		p.Tags = llm.InferTags(p.Model)
	}

	if err := h.store.SaveProvider(&p); err != nil {
		h.logger.Error("settings: save provider", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to save provider", err)
		return
	}
	api.Success(c, gin.H{"id": p.ID})
}

// SetActiveProvider handles PUT /api/settings/providers/:id/active
func (h *SettingsHandler) SetActiveProvider(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.SetActiveProvider(id); err != nil {
		h.logger.Error("settings: set active provider", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to set active provider", err)
		return
	}
	api.Success(c, gin.H{"active": id})
}

// DeleteProvider handles DELETE /api/settings/providers/:id
func (h *SettingsHandler) DeleteProvider(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.DeleteProvider(id); err != nil {
		h.logger.Error("settings: delete provider", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to delete provider", err)
		return
	}
	api.Success(c, gin.H{"deleted": id})
}

// ── Channel Configs ────────────────────────────────────────────────────────

// GetChannelConfig handles GET /api/settings/channels/:name
func (h *SettingsHandler) GetChannelConfig(c *gin.Context) {
	name := c.Param("name")
	cfg, err := h.store.GetChannelConfig(name)
	if err != nil {
		h.logger.Error("settings: get channel config", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get channel config", err)
		return
	}
	// Return raw JSON as string to avoid double-encoding
	c.Header("Content-Type", "application/json")
	c.String(200, `{"channel":%q,"config":%s}`, name, cfg)
}

// SetChannelConfig handles PUT /api/settings/channels/:name
func (h *SettingsHandler) SetChannelConfig(c *gin.Context) {
	name := c.Param("name")
	var body struct {
		Config string `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.BadRequestWithError(c, "invalid request body", err)
		return
	}
	if err := h.store.SetChannelConfig(name, body.Config); err != nil {
		h.logger.Error("settings: set channel config", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to set channel config", err)
		return
	}

	// 热重载插件：仅针对已注册的真实频道插件进行重载
	if h.channelMgr != nil {
		// 检查插件是否真的存在，防止 email 等工具类配置触发报错
		if _, err := h.channelMgr.GetPlugin(name); err == nil {
			if err := h.channelMgr.Reinit(name, []byte(body.Config)); err != nil {
				h.logger.Error("channel reinit failed after config save",
					zap.String("name", name),
					zap.Error(err),
				)
			}
		}
	}

	api.Success(c, gin.H{"channel": name})
}

// SetupStatus handles GET /api/settings/setup-status
// Returns whether initial setup (LLM configuration) is still required.
func (h *SettingsHandler) SetupStatus(c *gin.Context) {
	llmConfigured := !h.store.IsSetupRequired()
	api.Success(c, gin.H{
		"llm_configured": llmConfigured,
		"setup_required": !llmConfigured,
		"hint":           map[bool]string{true: "", false: "请在 Web UI → 模型 中配置 LLM 才能开始对话"}[llmConfigured],
	})
}
