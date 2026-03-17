// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/workflow"
	"go.uber.org/zap"
)

// WorkflowHandler handles workflow-related HTTP requests.
type WorkflowHandler struct {
	engine *workflow.Engine
	logger *zap.Logger
}

// NewWorkflowHandler creates a new workflow handler.
func NewWorkflowHandler(engine *workflow.Engine, logger *zap.Logger) *WorkflowHandler {
	return &WorkflowHandler{
		engine: engine,
		logger: logger.Named("workflow_handler"),
	}
}

// CreateWorkflowRequest represents a request to create a workflow.
type CreateWorkflowRequest struct {
	ID          string                `json:"id" binding:"required"`
	Name        string                `json:"name" binding:"required"`
	Description string                `json:"description"`
	Definition  workflow.WorkflowDef  `json:"definition" binding:"required"`
	Version     string                `json:"version"`
}

// UpdateWorkflowRequest represents a request to update a workflow.
type UpdateWorkflowRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Definition  *workflow.WorkflowDef `json:"definition"`
	Version     string                `json:"version"`
	Status      string                `json:"status"`
}

// ExecuteWorkflowRequest represents a request to execute a workflow.
type ExecuteWorkflowRequest struct {
	Input map[string]interface{} `json:"input"`
}

// ListWorkflows returns all workflows.
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	status := c.Query("status")

	var workflows []*workflow.Workflow
	var err error

	if status != "" {
		workflows, err = h.engine.ListByStatus(workflow.WorkflowStatus(status))
	} else {
		workflows, err = h.engine.List()
	}

	if err != nil {
		h.logger.Error("failed to list workflows", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflows)
}

// GetWorkflow returns a specific workflow.
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	id := c.Param("id")
	wf, err := h.engine.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, wf)
}

// CreateWorkflow creates a new workflow.
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	var req CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wf := &workflow.Workflow{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Definition:  &req.Definition,
		Version:     req.Version,
		Status:      workflow.WorkflowStatusDraft,
	}

	if err := h.engine.Create(wf); err != nil {
		h.logger.Error("failed to create workflow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wf)
}

// UpdateWorkflow updates an existing workflow.
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	id := c.Param("id")

	existing, err := h.engine.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var req UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Definition != nil {
		existing.Definition = req.Definition
	}
	if req.Version != "" {
		existing.Version = req.Version
	}
	if req.Status != "" {
		existing.Status = workflow.WorkflowStatus(req.Status)
	}

	if err := h.engine.Update(id, existing); err != nil {
		h.logger.Error("failed to update workflow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteWorkflow deletes a workflow.
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	id := c.Param("id")
	if err := h.engine.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "workflow deleted"})
}

// ExecuteWorkflow executes a workflow.
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	id := c.Param("id")

	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	execution, err := h.engine.Execute(id, req.Input, "user")
	if err != nil {
		h.logger.Error("failed to execute workflow", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, execution)
}

// GetExecution returns a workflow execution.
func (h *WorkflowHandler) GetExecution(c *gin.Context) {
	id := c.Param("id")
	execution, err := h.engine.GetExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, execution)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// CancelExecution cancels a workflow execution.
func (h *WorkflowHandler) CancelExecution(c *gin.Context) {
	id := c.Param("id")
	if err := h.engine.Cancel(id); err != nil {
		h.logger.Error("failed to cancel execution", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "execution cancelled"})
}

// GetStats returns workflow statistics.
func (h *WorkflowHandler) GetStats(c *gin.Context) {
	workflowID := c.Param("workflow_id")
	stats, err := h.engine.GetStats(workflowID)
	if err != nil {
		h.logger.Error("failed to get stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

		// Validate dependencies
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				// Dependency might be defined later
				// We'll check this after processing all steps
			}
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

	c.JSON(http.StatusOK, resp)
}
