package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader is the header name for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for request ID
	RequestIDKey = "request_id"
)

// RequestIDMiddleware generates and sets a unique request ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header
		requestID := c.GetHeader(RequestIDHeader)
		
		// Generate new request ID if not present
		if requestID == "" {
			requestID = generateRequestID()
		}
		
		// Set request ID in context and response header
		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)
		
		// Continue processing
		c.Next()
	}
}

// generateRequestID creates a unique request ID
func generateRequestID() string {
	// Use timestamp + random bytes for uniqueness
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 4)
	
	// Generate random bytes
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp only if random generation fails
		return fmt.Sprintf("req_%d", timestamp)
	}
	
	return fmt.Sprintf("req_%d_%s", timestamp, hex.EncodeToString(randomBytes))
}

// GetRequestID extracts request ID from Gin context
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return "unknown"
}