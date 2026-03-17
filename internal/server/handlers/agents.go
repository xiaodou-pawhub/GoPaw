// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/agent"
	"go.uber.org/zap"
)

// AgentsHandler handles /api/agents routes.
type AgentsHandler struct {
	manager *agent.Manager
	logger  *zap.Logger
}

// NewAgentsHandler creates an AgentsHandler.
func NewAgentsHandler(manager *agent.Manager, logger *zap.Logger) *AgentsHandler {
	return &AgentsHandler{manager: manager, logger: logger}
}

// agentCreateRequest represents a request to create an agent.
type agentCreateRequest struct {
	ID          string         `json:"id" binding:"required"`
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Avatar      string         `json:"avatar"`
	Config      *agent.AgentConfig  `json:"config"`
}

// agentUpdateRequest represents a request to update an agent.
type agentUpdateRequest struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Avatar      string        `json:"avatar"`
	IsActive    *bool         `json:"is_active"`
	Config      *agent.AgentConfig `json:"config"`
}

// List handles GET /api/agents
func (h *AgentsHandler) List(c *gin.Context) {
	agents := h.manager.List()
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// Get handles GET /api/agents/:id
func (h *AgentsHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent id is required"})
		return
	}

	def, err := h.manager.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, def)
}

// GetDefault handles GET /api/agents/default
func (h *AgentsHandler) GetDefault(c *gin.Context) {
	def, err := h.manager.GetDefault()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, def)
}

// Create handles POST /api/agents
func (h *AgentsHandler) Create(c *gin.Context) {
	var req agentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate config if provided
	if req.Config != nil {
		req.Config.MergeWithDefault()
		if err := req.Config.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		req.Config = agent.DefaultAgentConfig()
	}

	def := &agent.Definition{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Config:      req.Config,
		IsActive:    true,
		IsDefault:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := h.manager.Create(def); err != nil {
		h.logger.Error("failed to create agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, def)
}

// Update handles PUT /api/agents/:id
func (h *AgentsHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent id is required"})
		return
	}

	var req agentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing agent
	existing, err := h.manager.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Avatar != "" {
		existing.Avatar = req.Avatar
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.Config != nil {
		req.Config.MergeWithDefault()
		if err := req.Config.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		existing.Config = req.Config
	}

	if err := h.manager.Update(id, existing); err != nil {
		h.logger.Error("failed to update agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/agents/:id
func (h *AgentsHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent id is required"})
		return
	}

	if err := h.manager.Delete(id); err != nil {
		h.logger.Error("failed to delete agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// SetDefault handles POST /api/agents/:id/default
func (h *AgentsHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent id is required"})
		return
	}

	if err := h.manager.SetDefault(id); err != nil {
		h.logger.Error("failed to set default agent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"default": id})
}

// GetConfig handles GET /api/agents/:id/config
func (h *AgentsHandler) GetConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent id is required"})
		return
	}

	def, err := h.manager.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if def.Config == nil {
		c.JSON(http.StatusOK, agent.DefaultAgentConfig())
		return
	}

	c.JSON(http.StatusOK, def.Config)
}

// UpdateConfig handles PUT /api/agents/:id/config
func (h *AgentsHandler) UpdateConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent id is required"})
		return
	}

	var config agent.AgentConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.MergeWithDefault()
	if err := config.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing agent
	existing, err := h.manager.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	existing.Config = &config

	if err := h.manager.Update(id, existing); err != nil {
		h.logger.Error("failed to update agent config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing.Config)
}
