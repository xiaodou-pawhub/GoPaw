// Package api provides common API utilities for the GoPaw platform.
package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Pagination represents pagination parameters.
type Pagination struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`
	Offset   int `json:"-"`
}

// DefaultPagination returns default pagination parameters.
func DefaultPagination() Pagination {
	return Pagination{
		Page:     1,
		PageSize: 20,
	}
}

// ParsePagination parses pagination parameters from the request.
func ParsePagination(c *gin.Context) Pagination {
	pagination := DefaultPagination()

	if page := c.Query("page"); page != "" {
		if p := parseInt(page); p > 0 {
			pagination.Page = p
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps := parseInt(pageSize); ps > 0 && ps <= 100 {
			pagination.PageSize = ps
		}
	}

	// Calculate offset
	pagination.Offset = (pagination.Page - 1) * pagination.PageSize

	return pagination
}

// parseInt parses a string to int safely.
func parseInt(s string) int {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return 0
	}
	return result
}

// PaginatedResponse is the standard response for paginated endpoints.
type PaginatedResponse struct {
	Code       int         `json:"code"`
	Message    string      `json:"message,omitempty"`
	Data       interface{} `json:"data,omitempty"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int         `json:"total"`
	TotalPages int         `json:"total_pages"`
}

// ListWithPagination returns a paginated list response.
func ListWithPagination(c *gin.Context, data interface{}, total int, pagination Pagination) {
	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Code:       http.StatusOK,
		Message:    "success",
		Data:       data,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// FilterParams represents common filter parameters.
type FilterParams struct {
	Query    string            `form:"q"`
	Status   string            `form:"status"`
	Category string            `form:"category"`
	Tags     []string          `form:"tags"`
	Metadata map[string]string `form:"metadata"`
}

// ParseFilters parses filter parameters from the request.
func ParseFilters(c *gin.Context) FilterParams {
	filters := FilterParams{
		Metadata: make(map[string]string),
	}

	filters.Query = c.Query("q")
	filters.Status = c.Query("status")
	filters.Category = c.Query("category")

	// Parse tags
	tags := c.QueryArray("tags")
	if len(tags) > 0 {
		filters.Tags = tags
	}

	// Parse metadata (format: key=value)
	metadataParams := c.QueryArray("metadata")
	for _, param := range metadataParams {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 2 {
			filters.Metadata[parts[0]] = parts[1]
		}
	}

	return filters
}

// SortParams represents sort parameters.
type SortParams struct {
	Field     string `form:"sort_by"`
	Direction string `form:"sort_order"` // asc or desc
}

// ParseSort parses sort parameters from the request.
func ParseSort(c *gin.Context) SortParams {
	sort := SortParams{
		Direction: "desc",
	}

	sort.Field = c.Query("sort_by")

	if order := c.Query("sort_order"); order != "" {
		if order == "asc" || order == "desc" {
			sort.Direction = order
		}
	}

	return sort
}
