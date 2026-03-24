// Package server provides the HTTP server, WebSocket handler and middleware for GoPaw.
package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/auth"
	"github.com/gopaw/gopaw/internal/mode"
	"go.uber.org/zap"
)

// contextKeyUserID is the gin context key for the authenticated user ID (team mode).
const contextKeyUserID = "gopaw_user_id"
const contextKeyUsername = "gopaw_username"

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

// WebAuth returns a middleware that enforces authentication based on the current mode.
//
//   - solo:        no authentication — UI is immediately accessible.
//   - team:  JWT validation (Bearer header or gopaw_session cookie).
//     Falls back to admin token cookie for backward compatibility.
func WebAuth(adminToken string, m mode.Mode, authSvc ...*auth.Service) gin.HandlerFunc {
	var svc *auth.Service
	if len(authSvc) > 0 {
		svc = authSvc[0]
	}
	return func(c *gin.Context) {
		// Solo mode: open access, no token required.
		if !m.RequireAuth() {
			c.Next()
			return
		}

		// Attempt JWT from Authorization: Bearer header.
		if svc != nil {
			if bearer := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "); bearer != "" {
				if claims, err := svc.ValidateToken(bearer); err == nil {
					c.Set(contextKeyUserID, claims.UserID)
					c.Set(contextKeyUsername, claims.Username)
					c.Next()
					return
				}
			}
			// Attempt JWT stored in session cookie.
			if cookie, err := c.Cookie("gopaw_session"); err == nil && cookie != "" {
				if claims, err := svc.ValidateToken(cookie); err == nil {
					c.Set(contextKeyUserID, claims.UserID)
					c.Set(contextKeyUsername, claims.Username)
					c.Next()
					return
				}
			}
		}

		// Fallback: legacy admin token cookie (team mode on fresh install).
		cookie, err := c.Cookie("gopaw_session")
		if err == nil && cookie == adminToken {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录，请输入访问 Token"})
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
