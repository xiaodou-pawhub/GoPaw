// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/skill"
	"go.uber.org/zap"
)

// SkillsHandler handles /api/skills routes.
type SkillsHandler struct {
	manager *skill.Manager
	logger  *zap.Logger
}

// NewSkillsHandler creates a SkillsHandler.
func NewSkillsHandler(m *skill.Manager, logger *zap.Logger) *SkillsHandler {
	return &SkillsHandler{manager: m, logger: logger}
}

type skillInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Level       int    `json:"level"`
	Enabled     bool   `json:"enabled"`
}

// List handles GET /api/skills.
func (h *SkillsHandler) List(c *gin.Context) {
	entries := h.manager.Registry().All()
	out := make([]skillInfo, 0, len(entries))
	for _, e := range entries {
		out = append(out, skillInfo{
			Name:        e.Manifest.Name,
			DisplayName: e.Manifest.DisplayName,
			Description: e.Manifest.Description,
			Author:      e.Manifest.Author,
			Version:     e.Manifest.Version,
			Level:       int(e.Manifest.Level),
			Enabled:     e.Enabled,
		})
	}
	c.JSON(http.StatusOK, gin.H{"skills": out})
}

// SetEnabled handles PUT /api/skills/:name/enabled.
func (h *SkillsHandler) SetEnabled(c *gin.Context) {
	name := c.Param("name")

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.manager.Registry().SetEnabled(name, body.Enabled); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("skill enabled state changed",
		zap.String("name", name), zap.Bool("enabled", body.Enabled))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
