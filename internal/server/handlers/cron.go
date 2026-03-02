// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/scheduler"
	"go.uber.org/zap"
)

// CronHandler handles /api/cron routes.
type CronHandler struct {
	manager *scheduler.Manager
	logger  *zap.Logger
}

// NewCronHandler creates a CronHandler.
func NewCronHandler(m *scheduler.Manager, logger *zap.Logger) *CronHandler {
	return &CronHandler{manager: m, logger: logger}
}

type cronJobRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	CronExpr    string `json:"cron_expr" binding:"required"`
	Channel     string `json:"channel" binding:"required"`
	SessionID   string `json:"session_id"`
	Prompt      string `json:"prompt" binding:"required"`
	Enabled     bool   `json:"enabled"`
	ActiveFrom  string `json:"active_from"`
	ActiveUntil string `json:"active_until"`
}

// cronRunResponse is the JSON response structure for cron run history.
// 中文：Cron 执行历史响应结构，使用 snake_case 字段名和 Unix 秒时间戳
// English: Cron run history response structure with snake_case field names and Unix seconds timestamps
type cronRunResponse struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	TriggeredAt int64  `json:"triggered_at"`  // Unix seconds / Unix 秒时间戳
	FinishedAt  *int64 `json:"finished_at"`   // Unix seconds, null if running / Unix 秒时间戳，运行中为 null
	Status      string `json:"status"`        // "success" | "error" | "running"
	Output      string `json:"output"`
	ErrorMsg    string `json:"error_msg"`
}

// toCronRunResponse converts scheduler.CronRun to cronRunResponse.
// 中文：将 scheduler.CronRun 转换为 cronRunResponse
// English: Convert scheduler.CronRun to cronRunResponse
func toCronRunResponse(r scheduler.CronRun) cronRunResponse {
	resp := cronRunResponse{
		ID:          r.ID,
		JobID:       r.JobID,
		TriggeredAt: r.TriggeredAt.Unix(),
		Status:      r.Status,
		Output:      r.Output,
		ErrorMsg:    r.ErrorMessage,
	}
	if !r.FinishedAt.IsZero() {
		finished := r.FinishedAt.Unix()
		resp.FinishedAt = &finished
	}
	return resp
}

// List handles GET /api/cron.
func (h *CronHandler) List(c *gin.Context) {
	jobs, err := h.manager.ListJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// Create handles POST /api/cron.
func (h *CronHandler) Create(c *gin.Context) {
	var req cronJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job := &scheduler.CronJob{
		Name:        req.Name,
		Description: req.Description,
		CronExpr:    req.CronExpr,
		Channel:     req.Channel,
		SessionID:   req.SessionID,
		Prompt:      req.Prompt,
		Enabled:     req.Enabled,
		ActiveFrom:  req.ActiveFrom,
		ActiveUntil: req.ActiveUntil,
	}

	id, err := h.manager.AddJob(c.Request.Context(), job)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("cron job created", zap.String("id", id), zap.String("name", req.Name))
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// Delete handles DELETE /api/cron/:id.
func (h *CronHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.RemoveJob(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	h.logger.Info("cron job deleted", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Update handles PUT /api/cron/:id.
func (h *CronHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req scheduler.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.manager.UpdateJob(c.Request.Context(), id, req); err != nil {
		// 中文：使用 errors.Is 判断错误类型，cron 表达式无效返回 400
		// English: Use errors.Is to distinguish error types, invalid cron returns 400
		if errors.Is(err, scheduler.ErrInvalidCronExpr) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("cron job updated", zap.String("id", id))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Trigger handles POST /api/cron/:id/trigger — runs the job immediately.
func (h *CronHandler) Trigger(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.TriggerJob(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListRuns handles GET /api/cron/:id/runs — returns execution history.
// 中文：返回任务执行历史
// English: Return task execution history
func (h *CronHandler) ListRuns(c *gin.Context) {
	id := c.Param("id")
	limit := 20
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	runs, err := h.manager.ListRuns(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 中文：转换为响应格式
	// English: Convert to response format
	resp := make([]cronRunResponse, len(runs))
	for i, r := range runs {
		resp[i] = toCronRunResponse(r)
	}

	c.JSON(http.StatusOK, gin.H{"runs": resp})
}
