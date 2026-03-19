// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Context keys for storing user information.
const (
	ContextKeyUserID      = "user_id"
	ContextKeyUsername    = "username"
	ContextKeyEmail       = "email"
	ContextKeyDisplayName = "display_name"
	ContextKeyTeamID      = "team_id"
)

// Middleware provides authentication middleware for Gin.
type Middleware struct {
	authService *Service
	logger      *zap.Logger
	skipPaths   map[string]bool
}

// NewMiddleware creates a new authentication middleware.
func NewMiddleware(authService *Service, logger *zap.Logger, skipPaths []string) *Middleware {
	skipMap := make(map[string]bool)
	for _, path := range skipPaths {
		skipMap[path] = true
	}
	return &Middleware{
		authService: authService,
		logger:      logger.Named("auth_middleware"),
		skipPaths:   skipMap,
	}
}

// RequireAuth is a middleware that requires authentication.
func (m *Middleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		if m.skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization header",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := m.authService.ValidateToken(tokenString)
		if err != nil {
			m.logger.Debug("token validation failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid or expired token",
			})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyDisplayName, claims.DisplayName)

		c.Next()
	}
}

// OptionalAuth is a middleware that optionally extracts user info if token is present.
func (m *Middleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims, err := m.authService.ValidateToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyDisplayName, claims.DisplayName)

		c.Next()
	}
}

// GetUserID extracts the user ID from the context.
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get(ContextKeyUserID); exists {
		return userID.(string)
	}
	return ""
}

// GetUsername extracts the username from the context.
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get(ContextKeyUsername); exists {
		return username.(string)
	}
	return ""
}

// GetEmail extracts the email from the context.
func GetEmail(c *gin.Context) string {
	if email, exists := c.Get(ContextKeyEmail); exists {
		return email.(string)
	}
	return ""
}

// GetDisplayName extracts the display name from the context.
func GetDisplayName(c *gin.Context) string {
	if displayName, exists := c.Get(ContextKeyDisplayName); exists {
		return displayName.(string)
	}
	return ""
}

// GetTeamID extracts the team ID from the context.
func GetTeamID(c *gin.Context) string {
	if teamID, exists := c.Get(ContextKeyTeamID); exists {
		return teamID.(string)
	}
	return ""
}

// SetTeamID sets the team ID in the context.
func SetTeamID(c *gin.Context, teamID string) {
	c.Set(ContextKeyTeamID, teamID)
}