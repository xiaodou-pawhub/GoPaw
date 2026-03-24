// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/auth"
	"github.com/gopaw/gopaw/internal/mode"
	"github.com/gopaw/gopaw/internal/user"
)

const (
	sessionCookieName = "gopaw_session"
	sessionMaxAge     = 7 * 24 * 60 * 60 // 7 天（秒）
)

// AuthHandler handles login / logout for the Web UI.
// It supports two modes:
//   - solo/legacy: accepts a single admin token.
//   - team:  accepts username + password, returns a JWT stored in the session cookie.
type AuthHandler struct {
	adminToken string
	mode       mode.Mode
	authSvc    *auth.Service  // nil in solo mode
	userSvc    *user.Service  // nil in solo mode
}

// NewAuthHandler creates an AuthHandler with the resolved admin token.
// Pass authSvc and userSvc non-nil to enable username+password login for team mode.
func NewAuthHandler(adminToken string, m mode.Mode, authSvc *auth.Service, userSvc *user.Service) *AuthHandler {
	return &AuthHandler{
		adminToken: adminToken,
		mode:       m,
		authSvc:    authSvc,
		userSvc:    userSvc,
	}
}

type loginRequest struct {
	// Token is used in solo mode (backward compatibility).
	Token string `json:"token"`
	// Username and Password are used in team mode.
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	if h.mode.RequireAuth() && h.authSvc != nil && h.userSvc != nil {
		// Team mode: username + password authentication.
		if req.Username == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username 和 password 不能为空"})
			return
		}
		u, err := h.userSvc.Authenticate(req.Username, req.Password)
		if errors.Is(err, user.ErrBadCredentials) || errors.Is(err, user.ErrInactive) {
			time.Sleep(500 * time.Millisecond)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败"})
			return
		}
		tokens, err := h.authSvc.GenerateToken(u.ID, u.Username, u.Email, u.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 Token 失败"})
			return
		}
		// Store JWT in the session cookie so existing WebAuth middleware works.
		c.SetCookie(sessionCookieName, tokens.AccessToken, sessionMaxAge, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{
			"ok":           true,
			"user_id":      u.ID,
			"username":     u.Username,
			"role":         u.Role,
			"access_token": tokens.AccessToken,
		})
		return
	}

	// Solo / legacy mode: single admin token.
	if req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token 不能为空"})
		return
	}
	if req.Token != h.adminToken {
		time.Sleep(500 * time.Millisecond)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token 不正确"})
		return
	}
	c.SetCookie(sessionCookieName, h.adminToken, sessionMaxAge, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Logout handles POST /api/auth/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Status handles GET /api/auth/status.
// Returns whether the current request is authenticated (WebAuth middleware verifies before this).
func (h *AuthHandler) Status(c *gin.Context) {
	resp := gin.H{"authenticated": true, "mode": h.mode.String()}
	if userID, exists := c.Get("gopaw_user_id"); exists {
		resp["user_id"] = userID
		resp["username"] = c.GetString("gopaw_username")
	}
	c.JSON(http.StatusOK, resp)
}

// Me handles GET /api/auth/me — returns the current user's profile (team mode).
func (h *AuthHandler) Me(c *gin.Context) {
	if h.userSvc == nil {
		c.JSON(http.StatusOK, gin.H{"mode": "solo"})
		return
	}
	userID, _ := c.Get("gopaw_user_id")
	uid, _ := userID.(string)
	if uid == "" {
		c.JSON(http.StatusOK, gin.H{"mode": h.mode.String()})
		return
	}
	u, err := h.userSvc.GetByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":       u.ID,
		"username": u.Username,
		"email":    u.Email,
		"role":     u.Role,
		"mode":     h.mode.String(),
	})
}
