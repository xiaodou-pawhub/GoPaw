package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/pkg/api"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// UserIDKey is the context key for user ID.
	UserIDKey contextKey = "user_id"
	// UsernameKey is the context key for username.
	UsernameKey contextKey = "username"
	// TokenKey is the context key for token.
	TokenKey contextKey = "token"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	// SkipPaths defines paths that don't require authentication.
	SkipPaths []string
	// TokenHeader is the header name for the token (default: "Authorization").
	TokenHeader string
	// TokenPrefix is the token prefix (default: "Bearer ").
	TokenPrefix string
	// ValidateToken is a function to validate the token.
	ValidateToken func(token string) (userID string, username string, err error)
}

// AuthMiddleware returns a middleware that handles authentication.
func AuthMiddleware(config AuthConfig) gin.HandlerFunc {
	// Set defaults
	if config.TokenHeader == "" {
		config.TokenHeader = "Authorization"
	}
	if config.TokenPrefix == "" {
		config.TokenPrefix = "Bearer "
	}

	// Create skip paths map
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip if path is in skip list
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Get token from header
		token := c.GetHeader(config.TokenHeader)
		if token == "" {
			api.Unauthorized(c, "missing authorization token")
			c.Abort()
			return
		}

		// Remove prefix if present
		if config.TokenPrefix != "" && strings.HasPrefix(token, config.TokenPrefix) {
			token = strings.TrimPrefix(token, config.TokenPrefix)
		}

		// Validate token
		if config.ValidateToken != nil {
			userID, username, err := config.ValidateToken(token)
			if err != nil {
				api.Unauthorized(c, "invalid authorization token")
				c.Abort()
				return
			}

			// Store user info in context
			c.Set(string(UserIDKey), userID)
			c.Set(string(UsernameKey), username)
			c.Set(string(TokenKey), token)
		}

		c.Next()
	}
}

// GetUserID returns the user ID from context.
func GetUserID(c *gin.Context) string {
	if uid, ok := c.Get(string(UserIDKey)); ok {
		if uidStr, ok := uid.(string); ok {
			return uidStr
		}
	}
	return ""
}

// GetUsername returns the username from context.
func GetUsername(c *gin.Context) string {
	if username, ok := c.Get(string(UsernameKey)); ok {
		if usernameStr, ok := username.(string); ok {
			return usernameStr
		}
	}
	return ""
}

// GetToken returns the token from context.
func GetToken(c *gin.Context) string {
	if token, ok := c.Get(string(TokenKey)); ok {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	return ""
}

// RequireAdmin returns a middleware that requires admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is admin (implementation depends on your auth system)
		isAdmin := c.GetBool("is_admin")
		if !isAdmin {
			api.Forbidden(c, "admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// CORS returns a middleware that handles CORS.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimitConfig holds rate limit configuration.
type RateLimitConfig struct {
	// RequestsPerMinute is the maximum number of requests per minute.
	RequestsPerMinute int
	// SkipPaths defines paths that are not rate limited.
	SkipPaths []string
}

// RateLimit returns a middleware that limits request rate.
// Note: This is a simple implementation. For production, use redis or similar.
func RateLimit(config RateLimitConfig) gin.HandlerFunc {
	// Simple in-memory rate limiting (for demo purposes)
	// In production, use redis or a dedicated rate limiting service
	return func(c *gin.Context) {
		// Skip if path is in skip list
		for _, path := range config.SkipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// For now, just pass through
		// Implement proper rate limiting with redis in production
		c.Next()
	}
}
