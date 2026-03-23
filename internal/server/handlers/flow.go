// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/flow"
	"go.uber.org/zap"
)

// FlowHandler 流程 API 处理器
type FlowHandler struct {
	service *flow.Service
	logger  *zap.Logger
}

// NewFlowHandler 创建流程处理器
func NewFlowHandler(service *flow.Service, logger *zap.Logger) *FlowHandler {
	return &FlowHandler{
		service: service,
		logger:  logger.Named("flow_handler"),
	}
}

// RegisterRoutes 注册路由
func (h *FlowHandler) RegisterRoutes(r *gin.RouterGroup) {
	flows := r.Group("/flows")
	{
		flows.GET("", h.ListFlows)
		flows.POST("", h.CreateFlow)
		flows.GET("/node-types", h.GetNodeTypes)
		flows.GET("/:id", h.GetFlow)
		flows.PUT("/:id", h.UpdateFlow)
		flows.DELETE("/:id", h.DeleteFlow)
		flows.POST("/:id/execute", h.ExecuteFlow)
		flows.POST("/:id/activate", h.ActivateFlow)
		flows.POST("/:id/deactivate", h.DeactivateFlow)

		// 执行记录
		flows.GET("/:id/executions", h.ListExecutions)
		flows.GET("/executions/:execId", h.GetExecution)
		flows.POST("/executions/:execId/continue", h.ContinueExecution)
	}
}

// ListFlows 列出流程
func (h *FlowHandler) ListFlows(c *gin.Context) {
	flowType := flow.FlowType(c.Query("type"))
	status := flow.FlowStatus(c.Query("status"))

	flows, err := h.service.ListFlows(flowType, status)
	if err != nil {
		h.logger.Error("failed to list flows", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, flows)
}

// CreateFlow 创建流程
func (h *FlowHandler) CreateFlow(c *gin.Context) {
	var req flow.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := h.service.CreateFlow(req)
	if err != nil {
		h.logger.Error("failed to create flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, f)
}

// GetFlow 获取流程
func (h *FlowHandler) GetFlow(c *gin.Context) {
	id := c.Param("id")

	f, err := h.service.GetFlow(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, f)
}

// UpdateFlow 更新流程
func (h *FlowHandler) UpdateFlow(c *gin.Context) {
	id := c.Param("id")

	var req flow.UpdateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := h.service.UpdateFlow(id, req)
	if err != nil {
		h.logger.Error("failed to update flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, f)
}

// DeleteFlow 删除流程
func (h *FlowHandler) DeleteFlow(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteFlow(id); err != nil {
		h.logger.Error("failed to delete flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ExecuteFlow 执行流程
func (h *FlowHandler) ExecuteFlow(c *gin.Context) {
	id := c.Param("id")

	var req flow.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Execute(id, req)
	if err != nil {
		h.logger.Error("failed to execute flow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ActivateFlow 激活流程
func (h *FlowHandler) ActivateFlow(c *gin.Context) {
	id := c.Param("id")

	f, err := h.service.UpdateFlow(id, flow.UpdateFlowRequest{Status: flow.FlowStatusActive})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, f)
}

// DeactivateFlow 停用流程
func (h *FlowHandler) DeactivateFlow(c *gin.Context) {
	id := c.Param("id")

	f, err := h.service.UpdateFlow(id, flow.UpdateFlowRequest{Status: flow.FlowStatusDisabled})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, f)
}

// ListExecutions 列出执行记录
func (h *FlowHandler) ListExecutions(c *gin.Context) {
	id := c.Param("id")

	executions, err := h.service.ListExecutions(id, 50)
	if err != nil {
		h.logger.Error("failed to list executions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetExecution 获取执行记录
func (h *FlowHandler) GetExecution(c *gin.Context) {
	execID := c.Param("execId")

	exec, err := h.service.GetExecution(execID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, exec)
}

// ContinueExecution 继续执行
func (h *FlowHandler) ContinueExecution(c *gin.Context) {
	execID := c.Param("execId")

	var req struct {
		Input string `json:"input"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ContinueExecution(execID, req.Input)
	if err != nil {
		h.logger.Error("failed to continue execution", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetNodeTypes 获取节点类型列表
func (h *FlowHandler) GetNodeTypes(c *gin.Context) {
	types := flow.GetNodeTypes()
	c.JSON(http.StatusOK, types)
}