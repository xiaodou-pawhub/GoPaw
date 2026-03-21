// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/user"
)

// UsersHandler manages user accounts (team/cloud mode only).
type UsersHandler struct {
	svc *user.Service
}

// NewUsersHandler creates a handler for user management.
func NewUsersHandler(svc *user.Service) *UsersHandler {
	return &UsersHandler{svc: svc}
}

// List handles GET /api/users — admin only.
func (h *UsersHandler) List(c *gin.Context) {
	users, err := h.svc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}
	// Strip password hashes before sending.
	type safeUser struct {
		ID       string    `json:"id"`
		Username string    `json:"username"`
		Email    string    `json:"email"`
		Role     user.Role `json:"role"`
		IsActive bool      `json:"is_active"`
	}
	out := make([]safeUser, len(users))
	for i, u := range users {
		out[i] = safeUser{ID: u.ID, Username: u.Username, Email: u.Email, Role: u.Role, IsActive: u.IsActive}
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

type createUserRequest struct {
	Username string    `json:"username" binding:"required"`
	Email    string    `json:"email"`
	Password string    `json:"password" binding:"required,min=8"`
	Role     user.Role `json:"role"`
}

// Create handles POST /api/users — admin only.
func (h *UsersHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := req.Role
	if role == "" {
		role = user.RoleMember
	}
	u, err := h.svc.CreateUser(req.Username, req.Email, req.Password, role)
	if errors.Is(err, user.ErrDuplicate) {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "username": u.Username, "role": u.Role})
}

// Delete handles DELETE /api/users/:id — admin only.
func (h *UsersHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(id); errors.Is(err, user.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type setActiveRequest struct {
	Active bool `json:"active"`
}

// SetActive handles PUT /api/users/:id/active — admin only.
func (h *UsersHandler) SetActive(c *gin.Context) {
	var req setActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.SetActive(c.Param("id"), req.Active); errors.Is(err, user.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
