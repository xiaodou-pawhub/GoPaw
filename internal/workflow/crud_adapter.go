package workflow

import (
	"context"
)

// CRUDAdapter adapts Engine to the CRUDService interface.
type CRUDAdapter struct {
	engine *Engine
}

// NewCRUDAdapter creates a new CRUD adapter for the workflow engine.
func NewCRUDAdapter(engine *Engine) *CRUDAdapter {
	return &CRUDAdapter{engine: engine}
}

// CreateRequest represents a request to create a workflow.
type CreateRequest struct {
	ID          string       `json:"id" binding:"required"`
	Name        string       `json:"name" binding:"required"`
	Description string       `json:"description"`
	Definition  *WorkflowDef `json:"definition" binding:"required"`
	Version     string       `json:"version"`
}

// UpdateRequest represents a request to update a workflow.
type UpdateRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Definition  *WorkflowDef `json:"definition"`
	Version     string       `json:"version"`
	Status      string       `json:"status"`
}

// Create creates a new workflow.
func (a *CRUDAdapter) Create(ctx context.Context, req CreateRequest) (*Workflow, error) {
	wf := &Workflow{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Definition:  req.Definition,
		Version:     req.Version,
		Status:      WorkflowStatusDraft,
	}

	if err := a.engine.Create(wf); err != nil {
		return nil, err
	}

	return wf, nil
}

// Get retrieves a workflow by ID.
func (a *CRUDAdapter) Get(ctx context.Context, id string) (*Workflow, error) {
	return a.engine.Get(id)
}

// List retrieves all workflows.
func (a *CRUDAdapter) List(ctx context.Context) ([]Workflow, error) {
	workflows, err := a.engine.List()
	if err != nil {
		return nil, err
	}

	// Convert []*Workflow to []Workflow
	result := make([]Workflow, len(workflows))
	for i, wf := range workflows {
		result[i] = *wf
	}
	return result, nil
}

// Update updates a workflow.
func (a *CRUDAdapter) Update(ctx context.Context, id string, req UpdateRequest) error {
	existing, err := a.engine.Get(id)
	if err != nil {
		return err
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
		existing.Status = WorkflowStatus(req.Status)
	}

	return a.engine.Update(id, existing)
}

// Delete deletes a workflow.
func (a *CRUDAdapter) Delete(ctx context.Context, id string) error {
	return a.engine.Delete(id)
}
