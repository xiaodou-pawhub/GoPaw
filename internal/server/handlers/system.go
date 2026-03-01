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
