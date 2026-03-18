// Package handler provides common handler utilities for the GoPaw platform.
package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/pkg/api"
)

// CRUDService defines the interface for CRUD operations.
// Implement this interface to use the generic CRUD handlers.
type CRUDService[T any, CreateReq any, UpdateReq any] interface {
	// Create creates a new entity
	Create(ctx context.Context, req CreateReq) (*T, error)

	// Get retrieves an entity by ID
	Get(ctx context.Context, id string) (*T, error)

	// List retrieves all entities
	List(ctx context.Context) ([]T, error)

	// Update updates an entity
	Update(ctx context.Context, id string, req UpdateReq) error

	// Delete deletes an entity
	Delete(ctx context.Context, id string) error
}

// CRUDHandler provides generic CRUD HTTP handlers.
type CRUDHandler[T any, CreateReq any, UpdateReq any] struct {
	Service    CRUDService[T, CreateReq, UpdateReq]
	ResourceName string // e.g., "agent", "workflow"
}

// NewCRUDHandler creates a new CRUD handler.
func NewCRUDHandler[T any, CreateReq any, UpdateReq any](
	service CRUDService[T, CreateReq, UpdateReq],
	resourceName string,
) *CRUDHandler[T, CreateReq, UpdateReq] {
	return &CRUDHandler[T, CreateReq, UpdateReq]{
		Service:      service,
		ResourceName: resourceName,
	}
}

// Create handles POST /{resource}
func (h *CRUDHandler[T, CreateReq, UpdateReq]) Create(c *gin.Context) {
	var req CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	entity, err := h.Service.Create(c.Request.Context(), req)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to create "+h.ResourceName, err)
		return
	}

	api.Created(c, entity)
}

// Get handles GET /{resource}/:id
func (h *CRUDHandler[T, CreateReq, UpdateReq]) Get(c *gin.Context) {
	id := c.Param("id")

	entity, err := h.Service.Get(c.Request.Context(), id)
	if err != nil {
		api.NotFound(c, h.ResourceName)
		return
	}

	api.Success(c, entity)
}

// List handles GET /{resource}
func (h *CRUDHandler[T, CreateReq, UpdateReq]) List(c *gin.Context) {
	entities, err := h.Service.List(c.Request.Context())
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list "+h.ResourceName+"s", err)
		return
	}

	api.Success(c, entities)
}

// Update handles PUT /{resource}/:id
func (h *CRUDHandler[T, CreateReq, UpdateReq]) Update(c *gin.Context) {
	id := c.Param("id")

	var req UpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.Service.Update(c.Request.Context(), id, req); err != nil {
		api.InternalErrorWithDetails(c, "failed to update "+h.ResourceName, err)
		return
	}

	api.SuccessWithMessage(c, "updated", nil)
}

// Delete handles DELETE /{resource}/:id
func (h *CRUDHandler[T, CreateReq, UpdateReq]) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.Service.Delete(c.Request.Context(), id); err != nil {
		api.InternalErrorWithDetails(c, "failed to delete "+h.ResourceName, err)
		return
	}

	api.SuccessWithMessage(c, "deleted", nil)
}

// RegisterRoutes registers standard CRUD routes.
func (h *CRUDHandler[T, CreateReq, UpdateReq]) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("", h.Create)
	router.GET("", h.List)
	router.GET("/:id", h.Get)
	router.PUT("/:id", h.Update)
	router.DELETE("/:id", h.Delete)
}
