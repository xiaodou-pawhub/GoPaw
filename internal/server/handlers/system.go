// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

const version = "0.1.0"

var startTime = time.Now()

// SystemHandler handles /api/system routes.
type SystemHandler struct{}

// NewSystemHandler creates a SystemHandler.
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// Health handles GET /api/system/health.
func (h *SystemHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"uptime_s": int64(time.Since(startTime).Seconds()),
	})
}

// Version handles GET /api/system/version.
func (h *SystemHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    version,
		"go_version": runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	})
}

// AdminAuth returns a middleware that checks for a simple admin token.
func (h *SystemHandler) AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 中文：简单实现的 Token 鉴权，生产环境应从配置读取
		// English: Simple token auth, production should read from config.
		const adminToken = "gopaw-admin-secret"
		
		token := c.GetHeader("X-Admin-Token")
		if token == "" {
			token = c.Query("token")
		}

		if token != adminToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未授权访问 / Unauthorized"})
			return
		}
		c.Next()
	}
}


