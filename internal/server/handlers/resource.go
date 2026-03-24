// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/audit"
	"github.com/gopaw/gopaw/internal/resource"
	"go.uber.org/zap"
)

// ResourceHandler handles resource package management.
type ResourceHandler struct {
	svc      *resource.Service
	auditMgr *audit.Manager
	logger   *zap.Logger
}

// NewResourceHandler creates a new resource handler.
func NewResourceHandler(svc *resource.Service, auditMgr *audit.Manager, logger *zap.Logger) *ResourceHandler {
	return &ResourceHandler{
		svc:      svc,
		auditMgr: auditMgr,
		logger:   logger.Named("resource_handler"),
	}
}

// CreatePackageRequest represents the request to create a package.
type CreatePackageRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsGlobal    bool   `json:"is_global"`
}

// CreatePackage handles POST /api/resource-packages.
func (h *ResourceHandler) CreatePackage(c *gin.Context) {
	var req CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "资源包名称不能为空"})
		return
	}

	userID, _ := c.Get("gopaw_user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	pkg, err := h.svc.CreatePackage(c.Request.Context(), req.Name, req.Description, userID.(string), req.IsGlobal)
	if err != nil {
		h.logger.Error("failed to create package", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	// Log audit
	h.auditMgr.Log(&audit.Log{
		Timestamp:    time.Now(),
		Category:     audit.CategoryPermission,
		Action:       audit.ActionResourceCreate,
		UserID:       userID.(string),
		UserIP:       c.ClientIP(),
		ResourceType: "resource_package",
		ResourceID:   pkg.ID,
		Status:       audit.StatusSuccess,
		Details: map[string]interface{}{
			"name":      req.Name,
			"is_global": req.IsGlobal,
		},
	})

	c.JSON(http.StatusCreated, gin.H{"package": pkg})
}

// ListPackages handles GET /api/resource-packages.
func (h *ResourceHandler) ListPackages(c *gin.Context) {
	packages, err := h.svc.ListPackages(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to list packages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"packages": packages})
}

// GetPackage handles GET /api/resource-packages/:id.
func (h *ResourceHandler) GetPackage(c *gin.Context) {
	id := c.Param("id")
	pkg, err := h.svc.GetPackage(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资源包不存在"})
		return
	}

	items, err := h.svc.GetItems(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get package items", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"package": pkg,
		"items":   items,
	})
}

// UpdatePackage handles PUT /api/resource-packages/:id.
func (h *ResourceHandler) UpdatePackage(c *gin.Context) {
	id := c.Param("id")
	var req CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	err := h.svc.UpdatePackage(c.Request.Context(), id, req.Name, req.Description, req.IsGlobal)
	if err != nil {
		h.logger.Error("failed to update package", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeletePackage handles DELETE /api/resource-packages/:id.
func (h *ResourceHandler) DeletePackage(c *gin.Context) {
	id := c.Param("id")
	err := h.svc.DeletePackage(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to delete package", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// AddItemRequest represents the request to add an item to a package.
type AddItemRequest struct {
	ResourceType string `json:"resource_type"` // agent, skill, knowledge, model
	ResourceID   string `json:"resource_id"`
}

// AddItem handles POST /api/resource-packages/:id/items.
func (h *ResourceHandler) AddItem(c *gin.Context) {
	id := c.Param("id")
	var req AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	err := h.svc.AddItem(c.Request.Context(), id, resource.ResourceType(req.ResourceType), req.ResourceID)
	if err != nil {
		h.logger.Error("failed to add item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RemoveItem handles DELETE /api/resource-packages/:id/items/:type/:id.
func (h *ResourceHandler) RemoveItem(c *gin.Context) {
	packageID := c.Param("id")
	resourceType := c.Param("type")
	resourceID := c.Param("resource_id")

	err := h.svc.RemoveItem(c.Request.Context(), packageID, resource.ResourceType(resourceType), resourceID)
	if err != nil {
		h.logger.Error("failed to remove item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GrantToUserRequest represents the request to grant a package to a user.
type GrantToUserRequest struct {
	UserID string `json:"user_id"`
}

// GrantToUser handles POST /api/resource-packages/:id/grant.
func (h *ResourceHandler) GrantToUser(c *gin.Context) {
	id := c.Param("id")
	var req GrantToUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	grantedBy, _ := c.Get("gopaw_user_id")
	if grantedBy == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	err := h.svc.GrantToUser(c.Request.Context(), req.UserID, id, grantedBy.(string))
	if err != nil {
		h.logger.Error("failed to grant package", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "授权失败"})
		return
	}

	// Log audit
	h.auditMgr.Log(&audit.Log{
		Timestamp:    time.Now(),
		Category:     audit.CategoryPermission,
		Action:       audit.ActionResourceGrant,
		UserID:       grantedBy.(string),
		UserIP:       c.ClientIP(),
		ResourceType: "resource_package",
		ResourceID:   id,
		Status:       audit.StatusSuccess,
		Details: map[string]interface{}{
			"granted_to": req.UserID,
		},
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RevokeGrant handles DELETE /api/resource-packages/:id/grant/:user_id.
func (h *ResourceHandler) RevokeGrant(c *gin.Context) {
	id := c.Param("id")
	userID := c.Param("user_id")
	grantedBy, _ := c.Get("gopaw_user_id")

	err := h.svc.RevokeUserGrant(c.Request.Context(), userID, id)
	if err != nil {
		h.logger.Error("failed to revoke grant", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "撤销失败"})
		return
	}

	// Log audit
	h.auditMgr.Log(&audit.Log{
		Timestamp:    time.Now(),
		Category:     audit.CategoryPermission,
		Action:       audit.ActionResourceRevoke,
		UserID:       grantedBy.(string),
		UserIP:       c.ClientIP(),
		ResourceType: "resource_package",
		ResourceID:   id,
		Status:       audit.StatusSuccess,
		Details: map[string]interface{}{
			"revoked_from": userID,
		},
	})

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetPackageGrants handles GET /api/resource-packages/:id/grants.
func (h *ResourceHandler) GetPackageGrants(c *gin.Context) {
	// id := c.Param("id")
	// This would need a store method to get grants for a package
	// For now, return empty list
	c.JSON(http.StatusOK, gin.H{"grants": []interface{}{}})
}

// GetUserPackages handles GET /api/users/:id/packages.
func (h *ResourceHandler) GetUserPackages(c *gin.Context) {
	userID := c.Param("id")
	packages, err := h.svc.GetUserPackages(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user packages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"packages": packages})
}

// SetAgentPermissionRequest represents the request to set agent permissions.
type SetAgentPermissionRequest struct {
	UserID     string `json:"user_id"`
	CanUse     bool   `json:"can_use"`
	CanModify  bool   `json:"can_modify"`
	CanDelete  bool   `json:"can_delete"`
}

// SetAgentPermission handles POST /api/agents/:id/permissions.
func (h *ResourceHandler) SetAgentPermission(c *gin.Context) {
	agentID := c.Param("id")
	var req SetAgentPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	err := h.svc.SetAgentPermission(c.Request.Context(), req.UserID, agentID, req.CanUse, req.CanModify, req.CanDelete)
	if err != nil {
		h.logger.Error("failed to set agent permission", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SetAgentVisibilityRequest represents the request to set agent visibility.
type SetAgentVisibilityRequest struct {
	Visibility string  `json:"visibility"` // global, private, shared
	OwnerID    *string `json:"owner_id"`
}

// SetAgentVisibility handles PUT /api/agents/:id/visibility.
func (h *ResourceHandler) SetAgentVisibility(c *gin.Context) {
	agentID := c.Param("id")
	var req SetAgentVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	err := h.svc.SetAgentVisibility(c.Request.Context(), agentID, req.Visibility, req.OwnerID)
	if err != nil {
		h.logger.Error("failed to set agent visibility", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "设置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// CheckAgentPermission handles GET /api/agents/:id/permission?user_id=xxx.
func (h *ResourceHandler) CheckAgentPermission(c *gin.Context) {
	agentID := c.Param("id")
	userID := c.Query("user_id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	canUse, err := h.svc.CanUseAgent(c.Request.Context(), userID, agentID)
	if err != nil {
		h.logger.Error("failed to check agent permission", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查失败"})
		return
	}

	canModify, err := h.svc.CanModifyAgent(c.Request.Context(), userID, agentID)
	if err != nil {
		h.logger.Error("failed to check agent modify permission", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"can_use":     strconv.FormatBool(canUse),
		"can_modify":  strconv.FormatBool(canModify),
	})
}
