// Package api provides common API utilities for the GoPaw platform.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard API response structure.
type Response struct {
	Code    int         `json:"code"`              // HTTP status code
	Message string      `json:"message,omitempty"` // Human-readable message
	Data    interface{} `json:"data,omitempty"`    // Response data
}

// ListResponse is the standard response for list endpoints.
type ListResponse struct {
	Code    int         `json:"code"`              // HTTP status code
	Message string      `json:"message,omitempty"` // Human-readable message
	Data    interface{} `json:"data,omitempty"`    // List data
	Total   int         `json:"total,omitempty"`   // Total count for pagination
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Code    int    `json:"code"`              // HTTP status code
	Message string `json:"message"`           // Error message
	Error   string `json:"error,omitempty"`   // Detailed error info
}

// Success returns a successful response with data.
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage returns a successful response with custom message.
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
	})
}

// Created returns a successful creation response.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    http.StatusCreated,
		Message: "created",
		Data:    data,
	})
}

// List returns a list response with total count.
func List(c *gin.Context, data interface{}, total int) {
	c.JSON(http.StatusOK, ListResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
		Total:   total,
	})
}

// NoContent returns a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest returns a 400 Bad Request error.
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:    http.StatusBadRequest,
		Message: message,
	})
}

// BadRequestWithError returns a 400 Bad Request error with detailed error.
func BadRequestWithError(c *gin.Context, message string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:    http.StatusBadRequest,
		Message: message,
		Error:   errStr,
	})
}

// Unauthorized returns a 401 Unauthorized error.
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Code:    http.StatusUnauthorized,
		Message: message,
	})
}

// Forbidden returns a 403 Forbidden error.
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, ErrorResponse{
		Code:    http.StatusForbidden,
		Message: message,
	})
}

// NotFound returns a 404 Not Found error.
func NotFound(c *gin.Context, resource string) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Code:    http.StatusNotFound,
		Message: resource + " not found",
	})
}

// Conflict returns a 409 Conflict error.
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, ErrorResponse{
		Code:    http.StatusConflict,
		Message: message,
	})
}

// InternalError returns a 500 Internal Server Error.
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    http.StatusInternalServerError,
		Message: message,
	})
}

// InternalErrorWithDetails returns a 500 error with detailed error info.
func InternalErrorWithDetails(c *gin.Context, message string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    http.StatusInternalServerError,
		Message: message,
		Error:   errStr,
	})
}

// ValidationError returns a 422 Unprocessable Entity error.
func ValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
		Code:    http.StatusUnprocessableEntity,
		Message: message,
	})
}

// BadGateway returns a 502 Bad Gateway error.
func BadGateway(c *gin.Context, message string) {
	c.JSON(http.StatusBadGateway, ErrorResponse{
		Code:    http.StatusBadGateway,
		Message: message,
	})
}

// BadGatewayWithError returns a 502 error with detailed error info.
func BadGatewayWithError(c *gin.Context, message string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	c.JSON(http.StatusBadGateway, ErrorResponse{
		Code:    http.StatusBadGateway,
		Message: message,
		Error:   errStr,
	})
}
