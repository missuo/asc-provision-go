package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/missuo/asc-provision-go/config"
)

// APIKeyAuth is a middleware that checks for a valid API key
func APIKeyAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth check for health check endpoint
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Skip auth for preflight requests
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Get API key from Authorization header
		authHeader := c.GetHeader("Authorization")

		// Support API key in query parameters for direct downloads
		apiKey := c.Query("api_key")

		// If API key is not in query param, try to extract from header
		if apiKey == "" && authHeader != "" {
			// Check if authorization header format is "Bearer <api-key>"
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				apiKey = parts[1]
			} else {
				// If not in Bearer format, use the entire header as the key
				apiKey = authHeader
			}
		}

		// Validate API key
		if apiKey == "" || apiKey != cfg.APIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing API key",
			})
			return
		}

		c.Next()
	}
}
