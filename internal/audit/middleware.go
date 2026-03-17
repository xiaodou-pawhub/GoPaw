// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package audit

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Middleware creates a Gin middleware for audit logging.
func Middleware(mgr *Manager, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Skip WebSocket and static files
		if c.Request.URL.Path == "/ws" || c.Request.URL.Path == "/ws/approval" {
			return
		}

		// Get user ID from context (set by auth middleware)
		userID, _ := c.Get("user_id")
		userIDStr, _ := userID.(string)

		// Determine status
		status := StatusSuccess
		if c.Writer.Status() >= 400 {
			status = StatusFailed
		}

		// Determine action from path and method
		action := determineAction(c.Request.Method, c.Request.URL.Path)

		// Build details
		details := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"user_agent": c.Request.UserAgent(),
		}

		// Add error details if failed
		var errStr string
		if len(c.Errors) > 0 {
			errStr = c.Errors.String()
			details["errors"] = errStr
		}

		// Log the request
		mgr.LogAsync(&Log{
			Category:     CategoryHTTP,
			Action:       action,
			UserID:       userIDStr,
			UserIP:       c.ClientIP(),
			ResourceType: "api",
			ResourceID:   c.Request.URL.Path,
			Status:       status,
			Details:      details,
			Error:        errStr,
			Duration:     int(time.Since(start).Milliseconds()),
			RequestID:    c.GetString("request_id"),
		})
	}
}

// determineAction determines the audit action from HTTP method and path.
func determineAction(method, path string) Action {
	// Map HTTP methods to actions
	switch method {
	case "POST":
		if contains(path, "/execute") {
			return ActionHTTPRequest
		}
		return ActionHTTPRequest
	case "PUT", "PATCH":
		return ActionHTTPRequest
	case "DELETE":
		return ActionHTTPRequest
	default:
		return ActionHTTPRequest
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

// containsSubstring checks if s contains substr anywhere.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// AuthMiddleware creates a middleware for logging authentication events.
func AuthMiddleware(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only process login/logout endpoints
		if c.Request.URL.Path != "/api/auth/login" && c.Request.URL.Path != "/api/auth/logout" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		var action Action
		if c.Request.URL.Path == "/api/auth/login" {
			action = ActionLogin
		} else {
			action = ActionLogout
		}

		status := StatusSuccess
		if c.Writer.Status() >= 400 {
			status = StatusFailed
		}

		// Get user ID from context if login was successful
		userID, _ := c.Get("user_id")
		userIDStr, _ := userID.(string)

		var errStr string
		if len(c.Errors) > 0 {
			errStr = c.Errors.String()
		}

		mgr.LogAsync(&Log{
			Category: CategoryAuth,
			Action:   action,
			UserID:   userIDStr,
			UserIP:   c.ClientIP(),
			Status:   status,
			Details: map[string]interface{}{
				"path":   c.Request.URL.Path,
				"status": c.Writer.Status(),
			},
			Error:    errStr,
			Duration: int(time.Since(start).Milliseconds()),
		})
	}
}
