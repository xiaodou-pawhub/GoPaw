// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/pkg/api"
	"github.com/gopaw/gopaw/pkg/handler"
	"go.uber.org/zap"
)

// AgentsHandler handles /api/agents routes.
type AgentsHandler struct {
	manager    *agent.Manager
	logger     *zap.Logger
	crudHandler *handler.CRUDHandler[agent.Definition, agent.CreateRequest, agent.UpdateRequest]
}

// NewAgentsHandler creates an AgentsHandler.
func NewAgentsHandler(manager *agent.Manager, logger *zap.Logger) *AgentsHandler {
	adapter := agent.NewCRUDAdapter(manager)
	return &AgentsHandler{
		manager:     manager,
		logger:      logger,
		crudHandler: handler.NewCRUDHandler(adapter, "agent"),
	}
}

// List handles GET /api/agents
func (h *AgentsHandler) List(c *gin.Context) {
	h.crudHandler.List(c)
}

// Get handles GET /api/agents/:id
func (h *AgentsHandler) Get(c *gin.Context) {
	h.crudHandler.Get(c)
}

// GetDefault handles GET /api/agents/default
func (h *AgentsHandler) GetDefault(c *gin.Context) {
	def, err := h.manager.GetDefault()
	if err != nil {
		api.NotFound(c, "default agent")
		return
	}

	api.Success(c, def)
}

// Create handles POST /api/agents
func (h *AgentsHandler) Create(c *gin.Context) {
	h.crudHandler.Create(c)
}

// Update handles PUT /api/agents/:id
func (h *AgentsHandler) Update(c *gin.Context) {
	h.crudHandler.Update(c)
}

// Delete handles DELETE /api/agents/:id
func (h *AgentsHandler) Delete(c *gin.Context) {
	h.crudHandler.Delete(c)
}

// SetDefault handles POST /api/agents/:id/default
func (h *AgentsHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "agent id is required")
		return
	}

	if err := h.manager.SetDefault(id); err != nil {
		h.logger.Error("failed to set default agent", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to set default agent", err)
		return
	}

	api.SuccessWithMessage(c, "default set", gin.H{"id": id})
}

// GetConfig handles GET /api/agents/:id/config
func (h *AgentsHandler) GetConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "agent id is required")
		return
	}

	def, err := h.manager.Get(id)
	if err != nil {
		api.NotFound(c, "agent")
		return
	}

	if def.Config == nil {
		api.Success(c, agent.DefaultAgentConfig())
		return
	}

	api.Success(c, def.Config)
}

// UpdateConfig handles PUT /api/agents/:id/config
func (h *AgentsHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "agent id is required")
		return
	}

	var config agent.AgentConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	config.MergeWithDefault()
	if err := config.Validate(); err != nil {
		api.ValidationError(c, err.Error())
		return
	}

	// Get existing agent
	existing, err := h.manager.Get(id)
	if err != nil {
		api.NotFound(c, "agent")
		return
	}

	existing.Config = &config

	if err := h.manager.Update(id, existing); err != nil {
		h.logger.Error("failed to update agent config", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to update agent config", err)
		return
	}

	api.Success(c, existing.Config)
}

// ========== 版本管理 ==========

// ListVersions handles GET /api/agents/:id/versions
func (h *AgentsHandler) ListVersions(c *gin.Context) {
	agentID := c.Param("id")

	versions, err := h.manager.ListVersions(agentID)
	if err != nil {
		h.logger.Error("failed to list versions", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to list versions", err)
		return
	}

	api.Success(c, versions)
}

// CreateVersion handles POST /api/agents/:id/versions
func (h *AgentsHandler) CreateVersion(c *gin.Context) {
	agentID := c.Param("id")

	var req struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	version, err := h.manager.CreateVersion(agentID, req.Name, "")
	if err != nil {
		h.logger.Error("failed to create version", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to create version", err)
		return
	}

	api.Created(c, version)
}

// GetVersion handles GET /api/agents/:id/versions/:version
func (h *AgentsHandler) GetVersion(c *gin.Context) {
	agentID := c.Param("id")
	versionStr := c.Param("version")

	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		api.BadRequest(c, "invalid version number")
		return
	}

	v, err := h.manager.GetVersion(agentID, version)
	if err != nil {
		api.NotFound(c, "version")
		return
	}

	api.Success(c, v)
}

// RollbackVersion handles POST /api/agents/:id/versions/:version/rollback
func (h *AgentsHandler) RollbackVersion(c *gin.Context) {
	agentID := c.Param("id")
	versionStr := c.Param("version")

	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		api.BadRequest(c, "invalid version number")
		return
	}

	def, err := h.manager.RollbackVersion(agentID, version)
	if err != nil {
		h.logger.Error("failed to rollback version", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to rollback version", err)
		return
	}

	api.SuccessWithMessage(c, "rolled back successfully", def)
}

// DeleteVersion handles DELETE /api/agents/:id/versions/:version
func (h *AgentsHandler) DeleteVersion(c *gin.Context) {
	agentID := c.Param("id")
	versionStr := c.Param("version")

	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		api.BadRequest(c, "invalid version number")
		return
	}

	if err := h.manager.DeleteVersion(agentID, version); err != nil {
		h.logger.Error("failed to delete version", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to delete version", err)
		return
	}

	api.Success(c, gin.H{"status": "ok"})
}

// GetVersionStats handles GET /api/agents/:id/versions/stats
func (h *AgentsHandler) GetVersionStats(c *gin.Context) {
	agentID := c.Param("id")

	stats, err := h.manager.GetVersionStats(agentID)
	if err != nil {
		h.logger.Error("failed to get version stats", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get version stats", err)
		return
	}

	api.Success(c, stats)
}

// ========== 性能分析 ==========

// GetAgentStats handles GET /api/agents/:id/stats
func (h *AgentsHandler) GetAgentStats(c *gin.Context) {
	agentID := c.Param("id")

	stats, err := h.manager.GetAgentStats(agentID)
	if err != nil {
		h.logger.Error("failed to get agent stats", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get agent stats", err)
		return
	}

	api.Success(c, stats)
}

// GetAgentDailyStats handles GET /api/agents/:id/stats/daily
func (h *AgentsHandler) GetAgentDailyStats(c *gin.Context) {
	agentID := c.Param("id")
	days := 7
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	stats, err := h.manager.GetAgentDailyStats(agentID, days)
	if err != nil {
		h.logger.Error("failed to get daily stats", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get daily stats", err)
		return
	}

	api.Success(c, stats)
}

// GetAgentErrorStats handles GET /api/agents/:id/stats/errors
func (h *AgentsHandler) GetAgentErrorStats(c *gin.Context) {
	agentID := c.Param("id")
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	stats, err := h.manager.GetAgentErrorStats(agentID, limit)
	if err != nil {
		h.logger.Error("failed to get error stats", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get error stats", err)
		return
	}

	api.Success(c, stats)
}

// GetAllAgentsStats handles GET /api/agents/stats
func (h *AgentsHandler) GetAllAgentsStats(c *gin.Context) {
	stats, err := h.manager.GetAllAgentsStats()
	if err != nil {
		h.logger.Error("failed to get all agents stats", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get all agents stats", err)
		return
	}

	api.Success(c, stats)
}
