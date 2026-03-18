package agent

import (
	"context"
	"time"
)

// CRUDAdapter adapts Manager to the CRUDService interface.
type CRUDAdapter struct {
	manager *Manager
}

// NewCRUDAdapter creates a new CRUD adapter for the agent manager.
func NewCRUDAdapter(manager *Manager) *CRUDAdapter {
	return &CRUDAdapter{manager: manager}
}

// CreateRequest represents a request to create an agent.
type CreateRequest struct {
	ID          string       `json:"id" binding:"required"`
	Name        string       `json:"name" binding:"required"`
	Description string       `json:"description"`
	Avatar      string       `json:"avatar"`
	Config      *AgentConfig `json:"config"`
}

// UpdateRequest represents a request to update an agent.
type UpdateRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Avatar      string       `json:"avatar"`
	IsActive    *bool        `json:"is_active"`
	Config      *AgentConfig `json:"config"`
}

// Create creates a new agent.
func (a *CRUDAdapter) Create(ctx context.Context, req CreateRequest) (*Definition, error) {
	// Validate config if provided
	if req.Config != nil {
		req.Config.MergeWithDefault()
		if err := req.Config.Validate(); err != nil {
			return nil, err
		}
	} else {
		req.Config = DefaultAgentConfig()
	}

	def := &Definition{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Config:      req.Config,
		IsActive:    true,
		IsDefault:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := a.manager.Create(def); err != nil {
		return nil, err
	}

	return def, nil
}

// Get retrieves an agent by ID.
func (a *CRUDAdapter) Get(ctx context.Context, id string) (*Definition, error) {
	return a.manager.Get(id)
}

// List retrieves all agents.
func (a *CRUDAdapter) List(ctx context.Context) ([]Definition, error) {
	defs := a.manager.List()
	// Convert []*Definition to []Definition
	result := make([]Definition, len(defs))
	for i, def := range defs {
		result[i] = *def
	}
	return result, nil
}

// Update updates an agent.
func (a *CRUDAdapter) Update(ctx context.Context, id string, req UpdateRequest) error {
	// Get existing agent
	existing, err := a.manager.Get(id)
	if err != nil {
		return err
	}

	// Update fields
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Avatar != "" {
		existing.Avatar = req.Avatar
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.Config != nil {
		req.Config.MergeWithDefault()
		if err := req.Config.Validate(); err != nil {
			return err
		}
		existing.Config = req.Config
	}

	existing.UpdatedAt = time.Now().UTC()

	return a.manager.Update(id, existing)
}

// Delete deletes an agent.
func (a *CRUDAdapter) Delete(ctx context.Context, id string) error {
	return a.manager.Delete(id)
}
