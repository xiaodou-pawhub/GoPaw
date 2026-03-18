// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/config"
	"github.com/gopaw/gopaw/pkg/api"
)

const version = "0.1.0"

var startTime = time.Now()

// SystemHandler handles /api/system routes.
type SystemHandler struct {
	cfg     *config.Config
	logFile string // absolute path to the workspace log file
}

// NewSystemHandler creates a SystemHandler.
// logFile is the workspace log file path (e.g. wp.LogFile).
func NewSystemHandler(cfg *config.Config, logFile string) *SystemHandler {
	return &SystemHandler{cfg: cfg, logFile: logFile}
}

// Health handles GET /api/system/health.
func (h *SystemHandler) Health(c *gin.Context) {
	api.Success(c, gin.H{
		"status":   "ok",
		"uptime_s": int64(time.Since(startTime).Seconds()),
	})
}

// Version handles GET /api/system/version.
func (h *SystemHandler) Version(c *gin.Context) {
	api.Success(c, gin.H{
		"version":    version,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	})
}

// AdminAuth returns a middleware that checks for a simple admin token.
func (h *SystemHandler) AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 中文：从配置读取 Token 鉴权
		// English: Read token auth from config.
		adminToken := h.cfg.App.AdminToken
		if adminToken == "" {
			// 中文：如果未配置，出于安全考虑拒绝所有请求
			// English: If not configured, deny all requests for security.
			api.Forbidden(c, "管理员 Token 未配置，请在 config.yaml 中设置 / Admin token not configured")
			c.Abort()
			return
		}

		token := c.GetHeader("X-Admin-Token")
		if token != adminToken {
			api.Unauthorized(c, "未授权访问 / Unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}
