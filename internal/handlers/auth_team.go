// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/gopaw/gopaw/internal/auth"
	"github.com/gopaw/gopaw/internal/team"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService *auth.Service
	teamManager *team.Manager
	logger      *zap.Logger
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authService *auth.Service, teamManager *team.Manager, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		teamManager: teamManager,
		logger:      logger.Named("auth_handler"),
	}
}

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register handles user registration.
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	// Check if username already exists
	if _, err := h.teamManager.GetUserByUsername(req.Username); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "username already exists",
		})
		return
	}

	// Check if email already exists
	if _, err := h.teamManager.GetUserByEmail(req.Email); err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "email already exists",
		})
		return
	}

	// Hash password
	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to create user",
		})
		return
	}

	// Create user
	user := &team.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		Status:       "active",
	}

	if user.DisplayName == "" {
		user.DisplayName = user.Username
	}

	if err := h.teamManager.CreateUser(user); err != nil {
		h.logger.Error("failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to create user",
		})
		return
	}

	// Generate token
	tokenPair, err := h.authService.GenerateToken(user.ID, user.Username, user.Email, user.DisplayName)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "user created successfully",
		"data": gin.H{
			"user":   sanitizeUser(user),
			"tokens": tokenPair,
		},
	})
}

// Login handles user login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	// Get user by username or email
	user, err := h.teamManager.GetUserByUsername(req.Username)
	if err != nil {
		user, err = h.teamManager.GetUserByEmail(req.Username)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid credentials",
			})
			return
		}
	}

	// Check password
	if !h.authService.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "invalid credentials",
		})
		return
	}

	// Check if user is active
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "user account is not active",
		})
		return
	}

	// Update last login
	if err := h.teamManager.UpdateLastLogin(user.ID); err != nil {
		h.logger.Warn("failed to update last login", zap.Error(err))
	}

	// Generate token
	tokenPair, err := h.authService.GenerateToken(user.ID, user.Username, user.Email, user.DisplayName)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "login successful",
		"data": gin.H{
			"user":   sanitizeUser(user),
			"tokens": tokenPair,
		},
	})
}

// Refresh handles token refresh.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	tokenPair, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "invalid or expired refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "token refreshed",
		"data": gin.H{
			"tokens": tokenPair,
		},
	})
}

// GetProfile returns the current user's profile.
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	user, err := h.teamManager.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    sanitizeUser(user),
	})
}

// UpdateProfileRequest represents a profile update request.
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar"`
}

// UpdateProfile updates the current user's profile.
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	user, err := h.teamManager.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "user not found",
		})
		return
	}

	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	if err := h.teamManager.UpdateUser(user); err != nil {
		h.logger.Error("failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to update profile",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "profile updated",
		"data":    sanitizeUser(user),
	})
}

// ChangePasswordRequest represents a password change request.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword changes the current user's password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	user, err := h.teamManager.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "user not found",
		})
		return
	}

	// Verify current password
	if !h.authService.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "current password is incorrect",
		})
		return
	}

	// Hash new password
	passwordHash, err := h.authService.HashPassword(req.NewPassword)
	if err != nil {
		h.logger.Error("failed to hash password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to change password",
		})
		return
	}

	// Update password
	user.PasswordHash = passwordHash
	if err := h.teamManager.UpdateUser(user); err != nil {
		h.logger.Error("failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to change password",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "password changed successfully",
	})
}

// sanitizeUser removes sensitive information from user.
func sanitizeUser(user *team.User) gin.H {
	return gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"display_name":  user.DisplayName,
		"avatar":        user.Avatar,
		"status":        user.Status,
		"last_login_at": user.LastLoginAt,
		"created_at":    user.CreatedAt,
	}
}

// TeamHandler handles team-related HTTP requests.
type TeamHandler struct {
	teamManager *team.Manager
	authService *auth.Service
	logger      *zap.Logger
}

// NewTeamHandler creates a new team handler.
func NewTeamHandler(teamManager *team.Manager, authService *auth.Service, logger *zap.Logger) *TeamHandler {
	return &TeamHandler{
		teamManager: teamManager,
		authService: authService,
		logger:      logger.Named("team_handler"),
	}
}

// CreateTeamRequest represents a team creation request.
type CreateTeamRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
}

// CreateTeam creates a new team.
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	var req CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	team := &team.Team{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Avatar:      req.Avatar,
		OwnerID:     userID,
		Status:      "active",
	}

	if err := h.teamManager.CreateTeam(team); err != nil {
		h.logger.Error("failed to create team", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to create team",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    201,
		"message": "team created successfully",
		"data":    team,
	})
}

// ListTeams lists all teams the user belongs to.
func (h *TeamHandler) ListTeams(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	teams, err := h.teamManager.ListUserTeams(userID)
	if err != nil {
		h.logger.Error("failed to list teams", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to list teams",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    teams,
	})
}

// GetTeam returns a team by ID.
func (h *TeamHandler) GetTeam(c *gin.Context) {
	teamID := c.Param("team_id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "team_id is required",
		})
		return
	}

	team, err := h.teamManager.GetTeam(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "team not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    team,
	})
}

// UpdateTeamRequest represents a team update request.
type UpdateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Avatar      string `json:"avatar"`
	Settings    string `json:"settings"`
}

// UpdateTeam updates a team.
func (h *TeamHandler) UpdateTeam(c *gin.Context) {
	teamID := c.Param("team_id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "team_id is required",
		})
		return
	}

	var req UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	team, err := h.teamManager.GetTeam(teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "team not found",
		})
		return
	}

	if req.Name != "" {
		team.Name = req.Name
	}
	if req.Description != "" {
		team.Description = req.Description
	}
	if req.Avatar != "" {
		team.Avatar = req.Avatar
	}
	if req.Settings != "" {
		team.Settings = req.Settings
	}

	if err := h.teamManager.UpdateTeam(team); err != nil {
		h.logger.Error("failed to update team", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to update team",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "team updated",
		"data":    team,
	})
}

// DeleteTeam deletes a team.
func (h *TeamHandler) DeleteTeam(c *gin.Context) {
	teamID := c.Param("team_id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "team_id is required",
		})
		return
	}

	if err := h.teamManager.DeleteTeam(teamID); err != nil {
		h.logger.Error("failed to delete team", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to delete team",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "team deleted",
	})
}

// GetTeamMembers returns all members of a team.
func (h *TeamHandler) GetTeamMembers(c *gin.Context) {
	teamID := c.Param("team_id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "team_id is required",
		})
		return
	}

	members, err := h.teamManager.GetTeamMembers(teamID)
	if err != nil {
		h.logger.Error("failed to get team members", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to get team members",
		})
		return
	}

	// Enrich members with user info
	type MemberWithUser struct {
		*team.TeamMember
		User *team.User `json:"user"`
	}

	var result []MemberWithUser
	for _, member := range members {
		user, err := h.teamManager.GetUser(member.UserID)
		if err != nil {
			h.logger.Warn("failed to get user", zap.String("user_id", member.UserID), zap.Error(err))
			continue
		}
		user.PasswordHash = "" // Remove sensitive data
		result = append(result, MemberWithUser{
			TeamMember: member,
			User:       user,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    result,
	})
}

// AddMemberRequest represents a request to add a team member.
type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=owner admin member guest"`
}

// AddMember adds a user to a team.
func (h *TeamHandler) AddMember(c *gin.Context) {
	teamID := c.Param("team_id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "team_id is required",
		})
		return
	}

	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	if err := h.teamManager.AddTeamMember(teamID, req.UserID, req.Role, userID); err != nil {
		h.logger.Error("failed to add team member", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to add team member",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "member added successfully",
	})
}

// RemoveMember removes a user from a team.
func (h *TeamHandler) RemoveMember(c *gin.Context) {
	teamID := c.Param("team_id")
	memberUserID := c.Param("user_id")
	if teamID == "" || memberUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "team_id and user_id are required",
		})
		return
	}

	if err := h.teamManager.RemoveTeamMember(teamID, memberUserID); err != nil {
		h.logger.Error("failed to remove team member", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to remove team member",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "member removed successfully",
	})
}

// InviteMemberRequest represents a request to invite a member.
type InviteMemberRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Role      string `json:"role" binding:"required,oneof=owner admin member guest"`
	ExpiresIn int    `json:"expires_in"` // hours
}

// InviteMember invites a user to join a team.
func (h *TeamHandler) InviteMember(c *gin.Context) {
	teamID := c.Param("team_id")
	if teamID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "team_id is required",
		})
		return
	}

	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	var req InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		expiresAt = &t
	}

	inv, err := h.teamManager.CreateInvitation(teamID, req.Email, req.Role, userID, expiresAt)
	if err != nil {
		h.logger.Error("failed to create invitation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "failed to create invitation",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "invitation sent successfully",
		"data": gin.H{
			"invitation_id": inv.ID,
			"token":         inv.Token,
			"expires_at":    inv.ExpiresAt,
		},
	})
}

// AcceptInvitation accepts a team invitation.
func (h *TeamHandler) AcceptInvitation(c *gin.Context) {
	userID := auth.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "user not authenticated",
		})
		return
	}

	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "token is required",
		})
		return
	}

	if err := h.teamManager.AcceptInvitation(token, userID); err != nil {
		h.logger.Error("failed to accept invitation", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "invitation accepted successfully",
	})
}