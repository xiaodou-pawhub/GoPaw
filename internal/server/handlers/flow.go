// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/gopaw/gopaw/internal/flow"
	"go.uber.org/zap"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源
	},
}

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

		// 版本管理
		flows.GET("/:id/versions", h.ListVersions)
		flows.POST("/:id/versions", h.CreateVersion)
		flows.GET("/:id/versions/:version", h.GetVersion)
		flows.POST("/:id/versions/:version/rollback", h.RollbackVersion)
		flows.DELETE("/:id/versions/:version", h.DeleteVersion)

		// 模板管理
		flows.GET("/templates", h.ListTemplates)
		flows.GET("/templates/categories", h.GetTemplateCategories)
		flows.GET("/templates/:id", h.GetTemplate)
		flows.POST("/templates", h.CreateTemplate)
		flows.POST("/templates/:id/use", h.UseTemplate)
		flows.DELETE("/templates/:id", h.DeleteTemplate)

		// 执行记录
		flows.GET("/:id/executions", h.ListExecutions)
		flows.GET("/executions", h.ListAllExecutions)
		flows.GET("/executions/:execId", h.GetExecution)
		flows.POST("/executions/:execId/continue", h.ContinueExecution)
		flows.POST("/executions/:execId/step", h.StepExecution)
		flows.POST("/executions/:execId/breakpoints", h.SetBreakpoints)
		flows.POST("/executions/:execId/retry/:nodeId", h.RetryFromNode)
	}

	// Webhook 触发流程执行（不需要认证）
	r.POST("/webhooks/flow/:flowId", h.TriggerFlowByWebhook)

	// Webhook 回调（用于恢复等待中的执行，不需要认证）
	r.POST("/webhooks/:webhookId", h.HandleWebhook)

	// WebSocket 实时执行状态
	r.GET("/ws/flow/:flowId", h.HandleWebSocket)
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

// ListAllExecutions 列出所有执行记录（支持状态过滤）
func (h *FlowHandler) ListAllExecutions(c *gin.Context) {
	status := c.Query("status")

	executions, err := h.service.ListExecutionsByStatus(flow.ExecutionStatus(status), 100)
	if err != nil {
		h.logger.Error("failed to list all executions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 为每个执行记录添加流程名称
	type ExecutionWithFlowName struct {
		*flow.Execution
		FlowName string `json:"flow_name,omitempty"`
	}

	result := make([]ExecutionWithFlowName, 0, len(executions))
	for _, exec := range executions {
		item := ExecutionWithFlowName{Execution: exec}
		// 获取流程名称
		if f, err := h.service.GetFlow(exec.FlowID); err == nil {
			item.FlowName = f.Name
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, gin.H{"executions": result})
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

// StepExecution 单步执行（调试模式）
func (h *FlowHandler) StepExecution(c *gin.Context) {
	execID := c.Param("execId")

	resp, err := h.service.Step(execID)
	if err != nil {
		h.logger.Error("failed to step execution", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SetBreakpoints 设置断点
func (h *FlowHandler) SetBreakpoints(c *gin.Context) {
	execID := c.Param("execId")

	var req struct {
		Breakpoints []string `json:"breakpoints"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.SetBreakpoints(execID, req.Breakpoints); err != nil {
		h.logger.Error("failed to set breakpoints", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RetryFromNode 从特定节点重试执行
func (h *FlowHandler) RetryFromNode(c *gin.Context) {
	execID := c.Param("execId")
	nodeID := c.Param("nodeId")

	resp, err := h.service.RetryFromNode(execID, nodeID)
	if err != nil {
		h.logger.Error("failed to retry from node", zap.Error(err))
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

// TriggerFlowByWebhook 通过 Webhook 触发流程执行
func (h *FlowHandler) TriggerFlowByWebhook(c *gin.Context) {
	flowID := c.Param("flowId")

	// 获取流程
	f, err := h.service.GetFlow(flowID)
	if err != nil {
		h.logger.Error("flow not found", zap.String("flow_id", flowID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "flow not found"})
		return
	}

	// 检查流程状态
	if f.Status != flow.FlowStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow is not active"})
		return
	}

	// 检查触发器类型
	if f.Trigger == nil || f.Trigger.Type != "webhook" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow does not have webhook trigger"})
		return
	}

	// 解析请求体
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		// 允许空 payload
		payload = make(map[string]interface{})
	}

	// 添加 webhook 元信息
	payload["webhook_method"] = c.Request.Method
	payload["webhook_headers"] = c.Request.Header
	payload["webhook_query"] = c.Request.URL.Query()
	payload["webhook_triggered_at"] = c.Request.Header.Get("X-Request-Time")
	if payload["webhook_triggered_at"] == "" {
		payload["webhook_triggered_at"] = c.Request.Header.Get("Date")
	}

	// 构建输入
	input := ""
	if inputVal, ok := payload["input"]; ok {
		input = fmt.Sprintf("%v", inputVal)
	} else {
		// 将整个 payload 作为输入
		inputBytes, _ := json.Marshal(payload)
		input = string(inputBytes)
	}

	// 执行流程
	exec, err := h.service.Execute(flowID, flow.ExecuteRequest{
		Input:    input,
		Context:  payload,
		Async:    true,
		Trigger:  "webhook",
	})
	if err != nil {
		h.logger.Error("failed to execute flow via webhook",
			zap.String("flow_id", flowID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("flow triggered via webhook",
		zap.String("flow_id", flowID),
		zap.String("execution_id", exec.ExecutionID))

	c.JSON(http.StatusOK, gin.H{
		"status":       "triggered",
		"execution_id": exec.ExecutionID,
		"flow_id":      flowID,
	})
}

// HandleWebhook 处理 Webhook 回调
func (h *FlowHandler) HandleWebhook(c *gin.Context) {
	webhookID := c.Param("webhookId")

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		// 允许空 payload
		payload = make(map[string]interface{})
	}

	// 添加请求信息到 payload
	payload["webhook_method"] = c.Request.Method
	payload["webhook_headers"] = c.Request.Header
	payload["webhook_query"] = c.Request.URL.Query()

	if err := h.service.HandleWebhook(webhookID, payload); err != nil {
		h.logger.Error("failed to handle webhook",
			zap.String("webhook_id", webhookID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "webhook received"})
}

// ========== 版本管理 API ==========

// ListVersions 列出流程版本
func (h *FlowHandler) ListVersions(c *gin.Context) {
	flowID := c.Param("id")

	versions, err := h.service.ListVersions(flowID)
	if err != nil {
		h.logger.Error("failed to list versions", zap.String("flow_id", flowID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}

// CreateVersion 创建流程版本
func (h *FlowHandler) CreateVersion(c *gin.Context) {
	flowID := c.Param("id")

	var req flow.CreateVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户信息（如果有认证）
	createdBy := c.GetString("user_id")
	if createdBy == "" {
		createdBy = "system"
	}

	version, err := h.service.CreateVersion(flowID, req, createdBy)
	if err != nil {
		h.logger.Error("failed to create version", zap.String("flow_id", flowID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, version)
}

// GetVersion 获取特定版本
func (h *FlowHandler) GetVersion(c *gin.Context) {
	flowID := c.Param("id")
	versionStr := c.Param("version")

	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}

	v, err := h.service.GetVersion(flowID, version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, v)
}

// RollbackVersion 回滚到指定版本
func (h *FlowHandler) RollbackVersion(c *gin.Context) {
	flowID := c.Param("id")
	versionStr := c.Param("version")

	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}

	f, err := h.service.RollbackVersion(flowID, version)
	if err != nil {
		h.logger.Error("failed to rollback version",
			zap.String("flow_id", flowID),
			zap.Int("version", version),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("flow rolled back",
		zap.String("flow_id", flowID),
		zap.Int("version", version))

	c.JSON(http.StatusOK, f)
}

// DeleteVersion 删除指定版本
func (h *FlowHandler) DeleteVersion(c *gin.Context) {
	flowID := c.Param("id")
	versionStr := c.Param("version")

	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}

	if err := h.service.DeleteVersion(flowID, version); err != nil {
		h.logger.Error("failed to delete version",
			zap.String("flow_id", flowID),
			zap.Int("version", version),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "version deleted"})
}

// ========== 模板管理 API ==========

// ListTemplates 列出模板
func (h *FlowHandler) ListTemplates(c *gin.Context) {
	category := c.Query("category")
	publicOnly := c.Query("public") != "false"

	templates, err := h.service.ListTemplates(category, publicOnly)
	if err != nil {
		h.logger.Error("failed to list templates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, templates)
}

// GetTemplateCategories 获取模板分类
func (h *FlowHandler) GetTemplateCategories(c *gin.Context) {
	categories := h.service.GetTemplateCategories()
	c.JSON(http.StatusOK, categories)
}

// GetTemplate 获取模板
func (h *FlowHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")

	template, err := h.service.GetTemplate(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// CreateTemplate 创建模板
func (h *FlowHandler) CreateTemplate(c *gin.Context) {
	var req flow.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	author := c.GetString("user_id")
	if author == "" {
		author = "system"
	}

	template, err := h.service.CreateTemplate(req, author)
	if err != nil {
		h.logger.Error("failed to create template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// UseTemplate 使用模板创建流程
func (h *FlowHandler) UseTemplate(c *gin.Context) {
	templateID := c.Param("id")

	var req flow.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空请求，使用模板默认值
		req = flow.CreateFlowRequest{}
	}

	f, err := h.service.UseTemplate(templateID, req)
	if err != nil {
		h.logger.Error("failed to use template", zap.String("template_id", templateID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, f)
}

// DeleteTemplate 删除模板
func (h *FlowHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteTemplate(id); err != nil {
		h.logger.Error("failed to delete template", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "template deleted"})
}

// HandleWebSocket 处理 WebSocket 连接
func (h *FlowHandler) HandleWebSocket(c *gin.Context) {
	flowID := c.Param("flowId")

	// 获取 WebSocket Hub
	hub := h.service.GetWebSocketHub()
	if hub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "WebSocket not available"})
		return
	}

	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade websocket", zap.Error(err))
		return
	}

	// 创建客户端
	client := flow.NewWebSocketClient(conn, flowID, hub, h.logger)
	hub.RegisterClient(client)

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()
}