// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gopaw/gopaw/internal/team"
)

// PermissionMiddleware provides permission checking middleware.
type PermissionMiddleware struct {
	teamManager *team.Manager
	logger      *zap.Logger
}

// NewPermissionMiddleware creates a new permission middleware.
func NewPermissionMiddleware(teamManager *team.Manager, logger *zap.Logger) *PermissionMiddleware {
	return &PermissionMiddleware{
		teamManager: teamManager,
		logger:      logger.Named("permission_middleware"),
	}
}

// RequirePermission creates a middleware that checks if the user has a specific permission.
func (m *PermissionMiddleware) RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "user not authenticated",
			})
			c.Abort()
			return
		}

		teamID := GetTeamID(c)
		if teamID == "" {
			// Try to get team ID from query parameter
			teamID = c.Query("team_id")
			if teamID == "" {
				// Try to get team ID from request body for POST/PUT
				if c.Request.Method == "POST" || c.Request.Method == "PUT" {
					var body struct {
						TeamID string `json:"team_id"`
					}
					if err := c.ShouldBindJSON(&body); err == nil && body.TeamID != "" {
						teamID = body.TeamID
					}
				}
			}
		}

		if teamID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "team_id is required",
			})
			c.Abort()
			return
		}

		// Check permission
		hasPermission, err := m.teamManager.HasPermission(userID, teamID, resource, action)
		if err != nil {
			m.logger.Error("failed to check permission",
				zap.String("user_id", userID),
				zap.String("team_id", teamID),
				zap.String("resource", resource),
				zap.String("action", action),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to check permission",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			m.logger.Debug("permission denied",
				zap.String("user_id", userID),
				zap.String("team_id", teamID),
				zap.String("resource", resource),
				zap.String("action", action),
			)
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "permission denied",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireTeamMember creates a middleware that checks if the user is a member of the team.
func (m *PermissionMiddleware) RequireTeamMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "user not authenticated",
			})
			c.Abort()
			return
		}

		teamID := c.Param("team_id")
		if teamID == "" {
			teamID = c.Query("team_id")
		}

		if teamID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "team_id is required",
			})
			c.Abort()
			return
		}

		// Check if user is a team member
		members, err := m.teamManager.GetTeamMembers(teamID)
		if err != nil {
			m.logger.Error("failed to get team members",
				zap.String("team_id", teamID),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to check team membership",
			})
			c.Abort()
			return
		}

		isMember := false
		for _, member := range members {
			if member.UserID == userID {
				isMember = true
				break
			}
		}

		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "you are not a member of this team",
			})
			c.Abort()
			return
		}

		// Set team ID in context for later use
		SetTeamID(c, teamID)
		c.Next()
	}
}

// RequireTeamRole creates a middleware that checks if the user has a specific role in the team.
func (m *PermissionMiddleware) RequireTeamRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "user not authenticated",
			})
			c.Abort()
			return
		}

		teamID := c.Param("team_id")
		if teamID == "" {
			teamID = c.Query("team_id")
		}

		if teamID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "team_id is required",
			})
			c.Abort()
			return
		}

		// Check if user has the required role
		members, err := m.teamManager.GetTeamMembers(teamID)
		if err != nil {
			m.logger.Error("failed to get team members",
				zap.String("team_id", teamID),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "failed to check team role",
			})
			c.Abort()
			return
		}

		hasRole := false
		for _, member := range members {
			if member.UserID == userID {
				for _, role := range roles {
					if member.Role == role {
						hasRole = true
						break
					}
				}
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "you don't have the required role",
			})
			c.Abort()
			return
		}

		// Set team ID in context for later use
		SetTeamID(c, teamID)
		c.Next()
	}
}