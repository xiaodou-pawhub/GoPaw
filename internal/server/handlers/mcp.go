// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/mcp"
	"go.uber.org/zap"
)

// MCPHandler handles /api/mcp routes.
type MCPHandler struct {
	manager *mcp.Manager
	logger  *zap.Logger
}

// NewMCPHandler creates an MCPHandler.
func NewMCPHandler(manager *mcp.Manager, logger *zap.Logger) *MCPHandler {
	return &MCPHandler{manager: manager, logger: logger}
}

// mcpServerCreateRequest represents a request to create an MCP server.
type mcpServerCreateRequest struct {
	ID          string   `json:"id" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Command     string   `json:"command" binding:"required"`
	Args        []string `json:"args"`
	Env         []string `json:"env"`
	Transport   string   `json:"transport"`
	URL         string   `json:"url"`
}

// mcpServerUpdateRequest represents a request to update an MCP server.
type mcpServerUpdateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Env         []string `json:"env"`
	Transport   string   `json:"transport"`
	URL         string   `json:"url"`
}

// List handles GET /api/mcp/servers
func (h *MCPHandler) List(c *gin.Context) {
	servers := h.manager.List()
	c.JSON(http.StatusOK, gin.H{"servers": servers})
}

// Get handles GET /api/mcp/servers/:id
func (h *MCPHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	server, err := h.manager.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

// Create handles POST /api/mcp/servers
func (h *MCPHandler) Create(c *gin.Context) {
	var req mcpServerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default transport
	if req.Transport == "" {
		req.Transport = "stdio"
	}

	server := &mcp.Server{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Command:     req.Command,
		Args:        req.Args,
		Env:         req.Env,
		Transport:   req.Transport,
		URL:         req.URL,
		IsActive:    false,
		IsBuiltin:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := h.manager.Create(server); err != nil {
		h.logger.Error("failed to create mcp server", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, server)
}

// Update handles PUT /api/mcp/servers/:id
func (h *MCPHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	var req mcpServerUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing server
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
	if req.Command != "" {
		existing.Command = req.Command
	}
	if req.Args != nil {
		existing.Args = req.Args
	}
	if req.Env != nil {
		existing.Env = req.Env
	}
	if req.Transport != "" {
		existing.Transport = req.Transport
	}
	if req.URL != "" {
		existing.URL = req.URL
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := h.manager.Update(id, existing); err != nil {
		h.logger.Error("failed to update mcp server", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// Delete handles DELETE /api/mcp/servers/:id
func (h *MCPHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	if err := h.manager.Delete(id); err != nil {
		h.logger.Error("failed to delete mcp server", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// SetActive handles POST /api/mcp/servers/:id/active
func (h *MCPHandler) SetActive(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	var req struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.manager.SetActive(id, req.Active); err != nil {
		h.logger.Error("failed to set mcp server active status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "active": req.Active})
}

// GetTools handles GET /api/mcp/servers/:id/tools
func (h *MCPHandler) GetTools(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server id is required"})
		return
	}

	client, err := h.manager.GetClient(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	tools := client.GetTools()
	c.JSON(http.StatusOK, gin.H{"tools": tools})
}

// GetAllTools handles GET /api/mcp/tools
func (h *MCPHandler) GetAllTools(c *gin.Context) {
	tools := h.manager.GetTools()
	c.JSON(http.StatusOK, gin.H{"tools": tools})
}
