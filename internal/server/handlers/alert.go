// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/alert"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	service *alert.Service
	logger  *zap.Logger
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler(service *alert.Service, logger *zap.Logger) *AlertHandler {
	return &AlertHandler{
		service: service,
		logger:  logger.Named("alert_handler"),
	}
}

// RegisterRoutes 注册路由
func (h *AlertHandler) RegisterRoutes(router *gin.RouterGroup) {
	// 告警规则
	rules := router.Group("/alert/rules")
	{
		rules.GET("", h.ListRules)
		rules.POST("", h.CreateRule)
		rules.GET("/:id", h.GetRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
	}

	// 通知渠道
	channels := router.Group("/alert/channels")
	{
		channels.GET("", h.ListChannels)
		channels.POST("", h.CreateChannel)
		channels.GET("/:id", h.GetChannel)
		channels.PUT("/:id", h.UpdateChannel)
		channels.DELETE("/:id", h.DeleteChannel)
		channels.POST("/:id/test", h.TestChannel)
	}

	// 告警历史
	history := router.Group("/alert/history")
	{
		history.GET("", h.ListHistory)
	}
}

// ========== 告警规则 ==========

// ListRules 列出告警规则
func (h *AlertHandler) ListRules(c *gin.Context) {
	rules, err := h.service.ListRules(c.Request.Context())
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list rules", err)
		return
	}

	api.Success(c, rules)
}

// CreateRule 创建告警规则
func (h *AlertHandler) CreateRule(c *gin.Context) {
	var rule alert.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.service.CreateRule(c.Request.Context(), &rule); err != nil {
		api.InternalErrorWithDetails(c, "failed to create rule", err)
		return
	}

	api.Created(c, rule)
}

// GetRule 获取告警规则
func (h *AlertHandler) GetRule(c *gin.Context) {
	id := c.Param("id")

	rule, err := h.service.GetRule(c.Request.Context(), id)
	if err != nil {
		api.NotFound(c, "rule")
		return
	}

	api.Success(c, rule)
}

// UpdateRule 更新告警规则
func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id := c.Param("id")

	var rule alert.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.service.UpdateRule(c.Request.Context(), id, &rule); err != nil {
		api.InternalErrorWithDetails(c, "failed to update rule", err)
		return
	}

	api.Success(c, rule)
}

// DeleteRule 删除告警规则
func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteRule(c.Request.Context(), id); err != nil {
		api.InternalErrorWithDetails(c, "failed to delete rule", err)
		return
	}

	api.Success(c, gin.H{"status": "ok"})
}

// ========== 通知渠道 ==========

// ListChannels 列出通知渠道
func (h *AlertHandler) ListChannels(c *gin.Context) {
	channels, err := h.service.ListChannels(c.Request.Context())
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list channels", err)
		return
	}

	api.Success(c, channels)
}

// CreateChannel 创建通知渠道
func (h *AlertHandler) CreateChannel(c *gin.Context) {
	var channel alert.NotificationChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.service.CreateChannel(c.Request.Context(), &channel); err != nil {
		api.InternalErrorWithDetails(c, "failed to create channel", err)
		return
	}

	api.Created(c, channel)
}

// GetChannel 获取通知渠道
func (h *AlertHandler) GetChannel(c *gin.Context) {
	id := c.Param("id")

	channel, err := h.service.GetChannel(c.Request.Context(), id)
	if err != nil {
		api.NotFound(c, "channel")
		return
	}

	api.Success(c, channel)
}

// UpdateChannel 更新通知渠道
func (h *AlertHandler) UpdateChannel(c *gin.Context) {
	id := c.Param("id")

	var channel alert.NotificationChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.service.UpdateChannel(c.Request.Context(), id, &channel); err != nil {
		api.InternalErrorWithDetails(c, "failed to update channel", err)
		return
	}

	api.Success(c, channel)
}

// DeleteChannel 删除通知渠道
func (h *AlertHandler) DeleteChannel(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteChannel(c.Request.Context(), id); err != nil {
		api.InternalErrorWithDetails(c, "failed to delete channel", err)
		return
	}

	api.Success(c, gin.H{"status": "ok"})
}

// TestChannel 测试通知渠道
func (h *AlertHandler) TestChannel(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.SendNotification(c.Request.Context(), id, "测试通知", "这是一条测试通知消息，用于验证通知渠道配置是否正确。"); err != nil {
		api.InternalErrorWithDetails(c, "failed to send test notification", err)
		return
	}

	api.SuccessWithMessage(c, "test notification sent", nil)
}

// ========== 告警历史 ==========

// ListHistory 列出告警历史
func (h *AlertHandler) ListHistory(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.service.ListHistory(c.Request.Context(), limit)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list history", err)
		return
	}

	api.Success(c, history)
}