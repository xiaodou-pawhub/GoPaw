// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/cron"
	"go.uber.org/zap"
)

// CronHandler handles /api/cron routes.
type CronHandler struct {
	service *cron.CronService
	logger  *zap.Logger
}

// NewCronHandler creates a CronHandler.
func NewCronHandler(s *cron.CronService, logger *zap.Logger) *CronHandler {
	return &CronHandler{service: s, logger: logger}
}

type createJobRequest struct {
	Name     string `json:"name" binding:"required"`
	Schedule string `json:"schedule" binding:"required"`
	Task     string `json:"task" binding:"required"`
	Channel  string `json:"channel"`
	TargetID string `json:"target_id"`
	AgentID  string `json:"agent_id"` // alias for target_id
}

type updateJobRequest struct {
	Name     string `json:"name" binding:"required"`
	Schedule string `json:"schedule" binding:"required"`
	Task     string `json:"task" binding:"required"`
	TargetID string `json:"target_id"`
	AgentID  string `json:"agent_id"`
	Enabled  *bool  `json:"enabled"`
}

// List handles GET /api/cron.
func (h *CronHandler) List(c *gin.Context) {
	jobs := h.service.ListJobs()
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// Create handles POST /api/cron.
func (h *CronHandler) Create(c *gin.Context) {
	var req createJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetID := req.TargetID
	if targetID == "" {
		targetID = req.AgentID
	}
	job, err := h.service.AddJob(req.Name, req.Schedule, req.Task, req.Channel, targetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("cron job created", zap.String("id", job.ID), zap.String("name", job.Name))
	c.JSON(http.StatusCreated, job)
}

// Delete handles DELETE /api/cron/:id.
func (h *CronHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.RemoveJob(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("cron job deleted", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Update handles PUT /api/cron/:id.
func (h *CronHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req updateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetID := req.TargetID
	if targetID == "" {
		targetID = req.AgentID
	}

	job, err := h.service.UpdateJob(id, req.Name, req.Schedule, req.Task, targetID, req.Enabled)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("cron job updated", zap.String("id", id), zap.String("name", req.Name))
	c.JSON(http.StatusOK, job)
}

// Trigger handles POST /api/cron/:id/trigger.
func (h *CronHandler) Trigger(c *gin.Context) {
	id := c.Param("id")

	run, err := h.service.TriggerJob(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("cron job triggered", zap.String("id", id))
	c.JSON(http.StatusOK, run)
}

// ListRuns handles GET /api/cron/:id/runs.
func (h *CronHandler) ListRuns(c *gin.Context) {
	id := c.Param("id")
	runs := h.service.GetRuns(id)
	// If no runs found, it returns empty slice, which is JSON []
	c.JSON(http.StatusOK, gin.H{"runs": runs})
}
