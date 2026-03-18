// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/workflow"
	"github.com/gopaw/gopaw/pkg/api"
	"github.com/gopaw/gopaw/pkg/handler"
	"go.uber.org/zap"
)

// WorkflowHandler handles workflow-related HTTP requests.
type WorkflowHandler struct {
	engine      *workflow.Engine
	logger      *zap.Logger
	crudHandler *handler.CRUDHandler[workflow.Workflow, workflow.CreateRequest, workflow.UpdateRequest]
}

// NewWorkflowHandler creates a new workflow handler.
func NewWorkflowHandler(engine *workflow.Engine, logger *zap.Logger) *WorkflowHandler {
	adapter := workflow.NewCRUDAdapter(engine)
	return &WorkflowHandler{
		engine:      engine,
		logger:      logger.Named("workflow_handler"),
		crudHandler: handler.NewCRUDHandler(adapter, "workflow"),
	}
}

// ExecuteWorkflowRequest represents a request to execute a workflow.
type ExecuteWorkflowRequest struct {
	Input map[string]interface{} `json:"input"`
}

// ListWorkflows returns all workflows.
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	status := c.Query("status")

	// 如果有过滤条件，使用原有逻辑
	if status != "" {
		workflows, err := h.engine.ListByStatus(workflow.WorkflowStatus(status))
		if err != nil {
			h.logger.Error("failed to list workflows", zap.Error(err))
			api.InternalErrorWithDetails(c, "failed to list workflows", err)
			return
		}
		api.Success(c, workflows)
		return
	}

	// 使用通用 CRUD
	h.crudHandler.List(c)
}

// GetWorkflow returns a specific workflow.
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	h.crudHandler.Get(c)
}

// CreateWorkflow creates a new workflow.
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	h.crudHandler.Create(c)
}

// UpdateWorkflow updates an existing workflow.
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	h.crudHandler.Update(c)
}

// DeleteWorkflow deletes a workflow.
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	h.crudHandler.Delete(c)
}

// ExecuteWorkflow executes a workflow.
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	id := c.Param("id")

	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	execution, err := h.engine.Execute(id, req.Input, "user")
	if err != nil {
		h.logger.Error("failed to execute workflow", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to execute workflow", err)
		return
	}

	api.Created(c, execution)
}

// GetExecution returns a workflow execution.
func (h *WorkflowHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	execution, err := h.engine.GetExecution(id)
	if err != nil {
		api.NotFound(c, "execution")
		return
	}
	api.Success(c, execution)
}

// ListExecutions returns executions for a workflow.
func (h *WorkflowHandler) ListExecutions(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	limit := 50

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	executions, err := h.engine.ListExecutions(workflowID, limit)
	if err != nil {
		h.logger.Error("failed to list executions", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to list executions", err)
		return
	}

	api.Success(c, executions)
}

// CancelExecution cancels a workflow execution.
func (h *WorkflowHandler) CancelExecution(c *gin.Context) {
	id := c.Param("id")
	if err := h.engine.Cancel(id); err != nil {
		h.logger.Error("failed to cancel execution", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to cancel execution", err)
		return
	}
	api.SuccessWithMessage(c, "execution cancelled", nil)
}

// GetStats returns workflow statistics.
func (h *WorkflowHandler) GetStats(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	stats, err := h.engine.GetStats(workflowID)
	if err != nil {
		h.logger.Error("failed to get stats", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get stats", err)
		return
	}
	api.Success(c, stats)
}

// ValidateWorkflowRequest represents a request to validate a workflow definition.
type ValidateWorkflowRequest struct {
	Definition workflow.WorkflowDef `json:"definition" binding:"required"`
}

// ValidateWorkflowResponse represents the response for workflow validation.
type ValidateWorkflowResponse struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
}

// ValidateWorkflow validates a workflow definition.
func (h *WorkflowHandler) ValidateWorkflow(c *gin.Context) {
	var req ValidateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	resp := ValidateWorkflowResponse{
		Valid:  true,
		Errors: []string{},
	}

	// Validate steps
	if len(req.Definition.Steps) == 0 {
		resp.Valid = false
		resp.Errors = append(resp.Errors, "workflow must have at least one step")
	}

	// Check for duplicate step IDs
	stepIDs := make(map[string]bool)
	for _, step := range req.Definition.Steps {
		if step.ID == "" {
			resp.Valid = false
			resp.Errors = append(resp.Errors, "step ID is required")
			continue
		}
		if stepIDs[step.ID] {
			resp.Valid = false
			resp.Errors = append(resp.Errors, "duplicate step ID: "+step.ID)
		}
		stepIDs[step.ID] = true

		if step.Agent == "" {
			resp.Valid = false
			resp.Errors = append(resp.Errors, "step agent is required: "+step.ID)
		}
		if step.Action == "" {
			resp.Valid = false
			resp.Errors = append(resp.Errors, "step action is required: "+step.ID)
		}
	}

	// Check all dependencies exist
	for _, step := range req.Definition.Steps {
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				resp.Valid = false
				resp.Errors = append(resp.Errors, "step "+step.ID+" depends on non-existent step: "+dep)
			}
		}
	}

	api.Success(c, resp)
}
