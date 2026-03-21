package knowledge

import (
	"context"
)

// CRUDAdapter adapts Service to the CRUDService interface.
type CRUDAdapter struct {
	service *Service
}

// NewCRUDAdapter creates a new CRUD adapter for the knowledge service.
func NewCRUDAdapter(service *Service) *CRUDAdapter {
	return &CRUDAdapter{service: service}
}

// CreateRequest represents a request to create a knowledge base.
type CreateRequest struct {
	ID          string `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateRequest represents a request to update a knowledge base.
type UpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// Create creates a new knowledge base.
func (a *CRUDAdapter) Create(ctx context.Context, req CreateRequest) (*KnowledgeBase, error) {
	return a.service.CreateKnowledgeBase(ctx, CreateKnowledgeBaseRequest{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
	})
}

// Get retrieves a knowledge base by ID.
func (a *CRUDAdapter) Get(ctx context.Context, id string) (*KnowledgeBase, error) {
	return a.service.GetKnowledgeBase(ctx, id)
}

// List retrieves all knowledge bases.
func (a *CRUDAdapter) List(ctx context.Context) ([]KnowledgeBase, error) {
	return a.service.ListKnowledgeBases(ctx)
}

// Update updates a knowledge base.
func (a *CRUDAdapter) Update(ctx context.Context, id string, req UpdateRequest) error {
	return a.service.UpdateKnowledgeBase(ctx, id, UpdateKnowledgeBaseRequest{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	})
}

// Delete deletes a knowledge base.
func (a *CRUDAdapter) Delete(ctx context.Context, id string) error {
	return a.service.DeleteKnowledgeBase(ctx, id)
}
