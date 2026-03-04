// Package server provides the HTTP server, WebSocket handler and middleware for GoPaw.
package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZapLogger returns a Gin middleware that logs HTTP requests using zap.
// 2xx/3xx → Debug（正常请求不打扰控制台）
// 4xx     → Warn
// 5xx     → Error
func ZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := c.Writer.Status()
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		}

		switch {
		case status >= 500:
			logger.Error("http request", fields...)
		case status >= 400:
			logger.Warn("http request", fields...)
		default:
			// 2xx/3xx: debug only, keeps info log clean during normal operation
			logger.Debug("http request", fields...)
		}
	}
}

// Recovery returns a Gin middleware that recovers from panics and returns HTTP 500.
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.Any("recover", r),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

// WebAuth returns a middleware that enforces session-cookie authentication for the Web UI.
// Requests carrying a valid gopaw_session cookie are allowed through.
// Returns 401 for unauthenticated API calls so the frontend can redirect to the login overlay.
func WebAuth(adminToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("gopaw_session")
		if err != nil || cookie != adminToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录，请输入访问 Token"})
			return
		}
		c.Next()
	}
}

// CORS returns a permissive CORS middleware suitable for local development.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
