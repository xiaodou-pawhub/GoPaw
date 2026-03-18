// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/mcp"
	"github.com/gopaw/gopaw/pkg/api"
	"github.com/gopaw/gopaw/pkg/handler"
	"go.uber.org/zap"
)

// MCPHandler handles /api/mcp routes.
type MCPHandler struct {
	manager     *mcp.Manager
	logger      *zap.Logger
	crudHandler *handler.CRUDHandler[mcp.Server, mcp.CreateRequest, mcp.UpdateRequest]
}

// NewMCPHandler creates an MCPHandler.
func NewMCPHandler(manager *mcp.Manager, logger *zap.Logger) *MCPHandler {
	adapter := mcp.NewCRUDAdapter(manager)
	return &MCPHandler{
		manager:     manager,
		logger:      logger,
		crudHandler: handler.NewCRUDHandler(adapter, "mcp server"),
	}
}

// List handles GET /api/mcp/servers
func (h *MCPHandler) List(c *gin.Context) {
	h.crudHandler.List(c)
}

// Get handles GET /api/mcp/servers/:id
func (h *MCPHandler) Get(c *gin.Context) {
	h.crudHandler.Get(c)
}

// Create handles POST /api/mcp/servers
func (h *MCPHandler) Create(c *gin.Context) {
	h.crudHandler.Create(c)
}

// Update handles PUT /api/mcp/servers/:id
func (h *MCPHandler) Update(c *gin.Context) {
	h.crudHandler.Update(c)
}

// Delete handles DELETE /api/mcp/servers/:id
func (h *MCPHandler) Delete(c *gin.Context) {
	h.crudHandler.Delete(c)
}

// SetActive handles POST /api/mcp/servers/:id/active
func (h *MCPHandler) SetActive(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "server id is required")
		return
	}

	var req struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.manager.SetActive(id, req.Active); err != nil {
		h.logger.Error("failed to set mcp server active status", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to set active status", err)
		return
	}

	api.Success(c, gin.H{"id": id, "active": req.Active})
}

// GetTools handles GET /api/mcp/servers/:id/tools
func (h *MCPHandler) GetTools(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "server id is required")
		return
	}

	client, err := h.manager.GetClient(id)
	if err != nil {
		api.NotFound(c, "mcp client")
		return
	}

	tools := client.GetTools()
	api.Success(c, gin.H{"tools": tools})
}

// GetAllTools handles GET /api/mcp/tools
func (h *MCPHandler) GetAllTools(c *gin.Context) {
	tools := h.manager.GetTools()
	api.Success(c, gin.H{"tools": tools})
}
