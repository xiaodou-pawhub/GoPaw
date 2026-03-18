package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger returns a middleware that logs HTTP requests.
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get client IP
		clientIP := c.ClientIP()

		// Get status code
		statusCode := c.Writer.Status()

		// Get error message if any
		errMsg := ""
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		// Log request
		if logger != nil {
			logger.Info("http request",
				zap.Int("status", statusCode),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", clientIP),
				zap.Duration("latency", latency),
				zap.String("user_agent", c.Request.UserAgent()),
				zap.String("error", errMsg),
			)
		}
	}
}

// LoggerWithConfig returns a middleware with custom config.
type LoggerConfig struct {
	Logger       *zap.Logger
	SkipPaths    []string
	ShowQuery    bool
	ShowUserAgent bool
}

func LoggerWithConfig(config LoggerConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip if path is in skip list
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()

		if config.Logger != nil {
			fields := []zap.Field{
				zap.Int("status", statusCode),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("ip", clientIP),
				zap.Duration("latency", latency),
			}

			if config.ShowQuery && c.Request.URL.RawQuery != "" {
				fields = append(fields, zap.String("query", c.Request.URL.RawQuery))
			}

			if config.ShowUserAgent && c.Request.UserAgent() != "" {
				fields = append(fields, zap.String("user_agent", c.Request.UserAgent()))
			}

			config.Logger.Info("http request", fields...)
		}
	}
}
