// Package middleware provides HTTP middleware for the GoPaw platform.
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// ErrorHandler returns a middleware that handles panics and errors.
func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic with stack trace
				if logger != nil {
					logger.Error("panic recovered",
						zap.Any("error", err),
						zap.String("stack", string(debug.Stack())),
					)
				}

				// Return error response
				api.InternalError(c, fmt.Sprintf("internal server error: %v", err))
				c.Abort()
			}
		}()

		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			lastErr := c.Errors[len(c.Errors)-1]
			if lastErr.Err != nil {
				api.InternalErrorWithDetails(c, lastErr.Error(), lastErr.Err)
				c.Abort()
			}
		}
	}
}

// RecoveryMiddleware returns a middleware that recovers from panics.
// This is a wrapper around gin.Recovery with custom logging.
func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		if logger != nil {
			logger.Error("panic recovered",
				zap.Any("error", err),
				zap.String("stack", string(debug.Stack())),
			)
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("internal server error: %v", err),
		})
	})
}
