package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/orchestration"
	"github.com/gopaw/gopaw/pkg/api"
)

// OrchestrationHandler 编排处理器
type OrchestrationHandler struct {
	service *orchestration.Service
}

// NewOrchestrationHandler 创建编排处理器
func NewOrchestrationHandler(service *orchestration.Service) *OrchestrationHandler {
	return &OrchestrationHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *OrchestrationHandler) RegisterRoutes(router *gin.RouterGroup) {
	orch := router.Group("/orchestrations")
	{
		orch.POST("", h.CreateOrchestration)
		orch.GET("", h.ListOrchestrations)
		orch.GET("/:id", h.GetOrchestration)
		orch.PUT("/:id", h.UpdateOrchestration)
		orch.DELETE("/:id", h.DeleteOrchestration)
		orch.POST("/:id/execute", h.ExecuteOrchestration)
		orch.POST("/validate", h.ValidateOrchestration)

		// 执行记录
		orch.GET("/:id/executions", h.ListExecutions)
	}

	// 执行相关（不需要编排 ID）
	router.GET("/executions/:executionId", h.GetExecution)
	router.GET("/executions/:executionId/messages", h.GetExecutionMessages)
	router.POST("/executions/:executionId/human-input", h.SubmitHumanInput)
}

// CreateOrchestration 创建编排
func (h *OrchestrationHandler) CreateOrchestration(c *gin.Context) {
	var req orchestration.CreateOrchestrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	orch, err := h.service.CreateOrchestration(c.Request.Context(), req)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to create orchestration", err)
		return
	}

	api.Created(c, orch)
}

// ListOrchestrations 列出编排
func (h *OrchestrationHandler) ListOrchestrations(c *gin.Context) {
	orchestrations, err := h.service.ListOrchestrations(c.Request.Context())
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list orchestrations", err)
		return
	}

	api.Success(c, orchestrations)
}

// GetOrchestration 获取编排
func (h *OrchestrationHandler) GetOrchestration(c *gin.Context) {
	id := c.Param("id")

	orch, err := h.service.GetOrchestration(c.Request.Context(), id)
	if err != nil {
		api.NotFound(c, "orchestration")
		return
	}

	api.Success(c, orch)
}

// UpdateOrchestration 更新编排
func (h *OrchestrationHandler) UpdateOrchestration(c *gin.Context) {
	id := c.Param("id")

	var req orchestration.UpdateOrchestrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.service.UpdateOrchestration(c.Request.Context(), id, req); err != nil {
		api.InternalErrorWithDetails(c, "failed to update orchestration", err)
		return
	}

	api.SuccessWithMessage(c, "updated", nil)
}

// DeleteOrchestration 删除编排
func (h *OrchestrationHandler) DeleteOrchestration(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteOrchestration(c.Request.Context(), id); err != nil {
		api.InternalErrorWithDetails(c, "failed to delete orchestration", err)
		return
	}

	api.SuccessWithMessage(c, "deleted", nil)
}

// ExecuteOrchestration 执行编排
func (h *OrchestrationHandler) ExecuteOrchestration(c *gin.Context) {
	id := c.Param("id")

	var req orchestration.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	execCtx, err := h.service.ExecuteOrchestration(c.Request.Context(), id, req)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to execute orchestration", err)
		return
	}

	api.Success(c, orchestration.ExecuteResponse{
		ExecutionID: execCtx.ID,
		Status:      execCtx.Status,
	})
}

// ValidateOrchestration 验证编排
func (h *OrchestrationHandler) ValidateOrchestration(c *gin.Context) {
	var def orchestration.OrchestrationDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.service.ValidateOrchestration(def); err != nil {
		api.ValidationError(c, err.Error())
		return
	}

	api.Success(c, gin.H{"valid": true})
}

// ListExecutions 列出执行记录
func (h *OrchestrationHandler) ListExecutions(c *gin.Context) {
	id := c.Param("id")

	executions, err := h.service.ListExecutions(c.Request.Context(), id)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list executions", err)
		return
	}

	api.Success(c, executions)
}

// GetExecution 获取执行记录
func (h *OrchestrationHandler) GetExecution(c *gin.Context) {
	executionID := c.Param("executionId")

	execCtx, err := h.service.GetExecution(c.Request.Context(), executionID)
	if err != nil {
		api.NotFound(c, "execution")
		return
	}

	api.Success(c, execCtx)
}

// GetExecutionMessages 获取执行消息
func (h *OrchestrationHandler) GetExecutionMessages(c *gin.Context) {
	executionID := c.Param("executionId")

	messages, err := h.service.GetExecutionMessages(c.Request.Context(), executionID)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to get execution messages", err)
		return
	}

	api.Success(c, messages)
}

// SubmitHumanInput 提交人工输入
func (h *OrchestrationHandler) SubmitHumanInput(c *gin.Context) {
	executionID := c.Param("executionId")

	var req struct {
		Input string `json:"input" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.service.SubmitHumanInput(c.Request.Context(), executionID, req.Input); err != nil {
		api.BadRequest(c, err.Error())
		return
	}

	api.SuccessWithMessage(c, "submitted", nil)
}
