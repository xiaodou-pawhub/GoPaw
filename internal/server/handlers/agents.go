// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
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
