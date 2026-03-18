// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/workspace"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// WorkspaceHandler handles /api/workspace routes for reading/writing agent files.
type WorkspaceHandler struct {
	paths  *workspace.Paths
	logger *zap.Logger
}

// NewWorkspaceHandler creates a WorkspaceHandler.
func NewWorkspaceHandler(paths *workspace.Paths, logger *zap.Logger) *WorkspaceHandler {
	return &WorkspaceHandler{paths: paths, logger: logger}
}

// readFile returns file content or empty string if not exists.
func (h *WorkspaceHandler) readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeFile writes content to file, creating parent dirs if needed.
func (h *WorkspaceHandler) writeFile(path, content string) error {
	if err := os.MkdirAll(h.paths.Root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// response wraps file content in JSON.
type fileResponse struct {
	Content string `json:"content"`
}

// request for writing file content.
type fileRequest struct {
	Content string `json:"content"`
}

// GetAgent handles GET /api/workspace/agent — reads AGENT.md.
func (h *WorkspaceHandler) GetAgent(c *gin.Context) {
	content, err := h.readFile(h.paths.AgentMDFile)
	if err != nil {
		h.logger.Error("failed to read AGENT.md", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to read AGENT.md", err)
		return
	}
	api.Success(c, fileResponse{Content: content})
}

// PutAgent handles PUT /api/workspace/agent — writes AGENT.md.
func (h *WorkspaceHandler) PutAgent(c *gin.Context) {
	var req fileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}
	if err := h.writeFile(h.paths.AgentMDFile, req.Content); err != nil {
		h.logger.Error("failed to write AGENT.md", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to write AGENT.md", err)
		return
	}
	api.Success(c, gin.H{"message": "ok"})
}

// GetPersona handles GET /api/workspace/persona — reads PERSONA.md.
func (h *WorkspaceHandler) GetPersona(c *gin.Context) {
	content, err := h.readFile(h.paths.PersonaMDFile)
	if err != nil {
		h.logger.Error("failed to read PERSONA.md", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to read PERSONA.md", err)
		return
	}
	api.Success(c, fileResponse{Content: content})
}

// PutPersona handles PUT /api/workspace/persona — writes PERSONA.md.
func (h *WorkspaceHandler) PutPersona(c *gin.Context) {
	var req fileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}
	if err := h.writeFile(h.paths.PersonaMDFile, req.Content); err != nil {
		h.logger.Error("failed to write PERSONA.md", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to write PERSONA.md", err)
		return
	}
	api.Success(c, gin.H{"message": "ok"})
}

// GetContext handles GET /api/workspace/context — reads CONTEXT.md.
func (h *WorkspaceHandler) GetContext(c *gin.Context) {
	content, err := h.readFile(h.paths.ContextMDFile)
	if err != nil {
		h.logger.Error("failed to read CONTEXT.md", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to read CONTEXT.md", err)
		return
	}
	api.Success(c, fileResponse{Content: content})
}

// PutContext handles PUT /api/workspace/context — writes CONTEXT.md.
func (h *WorkspaceHandler) PutContext(c *gin.Context) {
	var req fileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}
	if err := h.writeFile(h.paths.ContextMDFile, req.Content); err != nil {
		h.logger.Error("failed to write CONTEXT.md", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to write CONTEXT.md", err)
		return
	}
	api.Success(c, gin.H{"message": "ok"})
}

// GetMemory handles GET /api/workspace/memory — reads MEMORY.md.
func (h *WorkspaceHandler) GetMemory(c *gin.Context) {
	content, err := h.readFile(h.paths.MemoryMDFile)
	if err != nil {
		h.logger.Error("failed to read MEMORY.md", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to read MEMORY.md", err)
		return
	}
	api.Success(c, fileResponse{Content: content})
}

// PutMemory handles PUT /api/workspace/memory — writes MEMORY.md.
func (h *WorkspaceHandler) PutMemory(c *gin.Context) {
	var req fileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}
	if err := h.writeFile(h.paths.MemoryMDFile, req.Content); err != nil {
		h.logger.Error("failed to write MEMORY.md", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to write MEMORY.md", err)
		return
	}
	api.Success(c, gin.H{"message": "ok"})
}
