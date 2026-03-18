package mcp

import (
	"context"
	"time"
)

// CRUDAdapter adapts Manager to the CRUDService interface.
type CRUDAdapter struct {
	manager *Manager
}

// NewCRUDAdapter creates a new CRUD adapter for the MCP manager.
func NewCRUDAdapter(manager *Manager) *CRUDAdapter {
	return &CRUDAdapter{manager: manager}
}

// CreateRequest represents a request to create an MCP server.
type CreateRequest struct {
	ID          string   `json:"id" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Command     string   `json:"command" binding:"required"`
	Args        []string `json:"args"`
	Env         []string `json:"env"`
	Transport   string   `json:"transport"`
	URL         string   `json:"url"`
}

// UpdateRequest represents a request to update an MCP server.
type UpdateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Env         []string `json:"env"`
	Transport   string   `json:"transport"`
	URL         string   `json:"url"`
}

// Create creates a new MCP server.
func (a *CRUDAdapter) Create(ctx context.Context, req CreateRequest) (*Server, error) {
	// Set default transport
	if req.Transport == "" {
		req.Transport = "stdio"
	}

	server := &Server{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Command:     req.Command,
		Args:        req.Args,
		Env:         req.Env,
		Transport:   req.Transport,
		URL:         req.URL,
		IsActive:    false,
		IsBuiltin:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := a.manager.Create(server); err != nil {
		return nil, err
	}

	return server, nil
}

// Get retrieves an MCP server by ID.
func (a *CRUDAdapter) Get(ctx context.Context, id string) (*Server, error) {
	return a.manager.Get(id)
}

// List retrieves all MCP servers.
func (a *CRUDAdapter) List(ctx context.Context) ([]Server, error) {
	servers := a.manager.List()

	// Convert []*Server to []Server
	result := make([]Server, len(servers))
	for i, server := range servers {
		result[i] = *server
	}
	return result, nil
}

// Update updates an MCP server.
func (a *CRUDAdapter) Update(ctx context.Context, id string, req UpdateRequest) error {
	// Get existing server
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
	if req.Command != "" {
		existing.Command = req.Command
	}
	if req.Args != nil {
		existing.Args = req.Args
	}
	if req.Env != nil {
		existing.Env = req.Env
	}
	if req.Transport != "" {
		existing.Transport = req.Transport
	}
	if req.URL != "" {
		existing.URL = req.URL
	}
	existing.UpdatedAt = time.Now().UTC()

	return a.manager.Update(id, existing)
}

// Delete deletes an MCP server.
func (a *CRUDAdapter) Delete(ctx context.Context, id string) error {
	return a.manager.Delete(id)
}
