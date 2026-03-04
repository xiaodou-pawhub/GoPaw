package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	sessionCookieName = "gopaw_session"
	sessionMaxAge     = 7 * 24 * 60 * 60 // 7 天（秒）
)

// AuthHandler handles login / logout for the Web UI.
type AuthHandler struct {
	adminToken string
}

// NewAuthHandler creates an AuthHandler with the resolved admin token.
func NewAuthHandler(adminToken string) *AuthHandler {
	return &AuthHandler{adminToken: adminToken}
}

type loginRequest struct {
	Token string `json:"token" binding:"required"`
}

// Login handles POST /api/auth/login.
// Validates the token and writes a session cookie on success.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token 不能为空"})
		return
	}

	if req.Token != h.adminToken {
		// 错误时稍作延迟，防止暴力枚举
		time.Sleep(500 * time.Millisecond)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token 不正确"})
		return
	}

	// 写 httpOnly cookie，7天有效
	c.SetCookie(
		sessionCookieName,
		h.adminToken,
		sessionMaxAge,
		"/",
		"",    // domain: 空 = 当前域
		false, // secure: 开发环境不强制 HTTPS；生产环境建议改为 true
		true,  // httpOnly: 前端 JS 无法读取，防 XSS
	)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Logout handles POST /api/auth/logout.
// Clears the session cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Status handles GET /api/auth/status.
// Returns whether the current request is authenticated (checked by WebAuth middleware before this).
func (h *AuthHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"authenticated": true})
}
