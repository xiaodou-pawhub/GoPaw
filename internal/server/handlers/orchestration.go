package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/orchestration"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orch, err := h.service.CreateOrchestration(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, orch)
}

// ListOrchestrations 列出编排
func (h *OrchestrationHandler) ListOrchestrations(c *gin.Context) {
	orchestrations, err := h.service.ListOrchestrations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orchestrations)
}

// GetOrchestration 获取编排
func (h *OrchestrationHandler) GetOrchestration(c *gin.Context) {
	id := c.Param("id")

	orch, err := h.service.GetOrchestration(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "orchestration not found"})
		return
	}

	c.JSON(http.StatusOK, orch)
}

// UpdateOrchestration 更新编排
func (h *OrchestrationHandler) UpdateOrchestration(c *gin.Context) {
	id := c.Param("id")

	var req orchestration.UpdateOrchestrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateOrchestration(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DeleteOrchestration 删除编排
func (h *OrchestrationHandler) DeleteOrchestration(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteOrchestration(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ExecuteOrchestration 执行编排
func (h *OrchestrationHandler) ExecuteOrchestration(c *gin.Context) {
	id := c.Param("id")

	var req orchestration.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	execCtx, err := h.service.ExecuteOrchestration(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orchestration.ExecuteResponse{
		ExecutionID: execCtx.ID,
		Status:      execCtx.Status,
	})
}

// ValidateOrchestration 验证编排
func (h *OrchestrationHandler) ValidateOrchestration(c *gin.Context) {
	var def orchestration.OrchestrationDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ValidateOrchestration(def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// ListExecutions 列出执行记录
func (h *OrchestrationHandler) ListExecutions(c *gin.Context) {
	id := c.Param("id")

	executions, err := h.service.ListExecutions(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// GetExecution 获取执行记录
func (h *OrchestrationHandler) GetExecution(c *gin.Context) {
	executionID := c.Param("executionId")

	execCtx, err := h.service.GetExecution(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		return
	}

	c.JSON(http.StatusOK, execCtx)
}

// GetExecutionMessages 获取执行消息
func (h *OrchestrationHandler) GetExecutionMessages(c *gin.Context) {
	executionID := c.Param("executionId")

	messages, err := h.service.GetExecutionMessages(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

// SubmitHumanInput 提交人工输入
func (h *OrchestrationHandler) SubmitHumanInput(c *gin.Context) {
	executionID := c.Param("executionId")

	var req struct {
		Input string `json:"input" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.SubmitHumanInput(c.Request.Context(), executionID, req.Input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "submitted"})
}
