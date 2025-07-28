package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Prevent XSS attacks
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Force HTTPS (if enabled)
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		// Content Security Policy
		csp := strings.Join([]string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: https:",
			"font-src 'self'",
			"connect-src 'self'",
			"frame-ancestors 'none'",
			"base-uri 'self'",
			"form-action 'self'",
		}, "; ")
		c.Header("Content-Security-Policy", csp)
		
		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Remove server information
		c.Header("Server", "")
		
		c.Next()
	}
}

// APIKeyAuthConfig contains API key authentication configuration
type APIKeyAuthConfig struct {
	ValidKeys    map[string]bool
	HeaderName   string
	SkipPaths    []string
	ErrorMessage string
}

// APIKeyAuthMiddleware provides API key authentication
func APIKeyAuthMiddleware(config APIKeyAuthConfig) gin.HandlerFunc {
	// Default header name
	if config.HeaderName == "" {
		config.HeaderName = "X-API-Key"
	}
	
	// Default error message
	if config.ErrorMessage == "" {
		config.ErrorMessage = "Invalid or missing API key"
	}
	
	// Create skip paths map for faster lookup
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}
	
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		
		// Extract API key from header
		apiKey := c.GetHeader(config.HeaderName)
		
		// Check if API key is provided
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "authentication_required",
				"message":    "API key required",
				"request_id": GetRequestID(c),
			})
			c.Abort()
			return
		}
		
		// Validate API key
		if !config.ValidKeys[apiKey] {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "invalid_api_key",
				"message":    config.ErrorMessage,
				"request_id": GetRequestID(c),
			})
			c.Abort()
			return
		}
		
		// Store API key in context for later use
		c.Set("api_key", apiKey)
		
		c.Next()
	}
}

// BasicAuthConfig contains basic authentication configuration
type BasicAuthConfig struct {
	Users     map[string]string // username -> password
	Realm     string
	SkipPaths []string
}

// BasicAuthMiddleware provides basic HTTP authentication
func BasicAuthMiddleware(config BasicAuthConfig) gin.HandlerFunc {
	// Default realm
	if config.Realm == "" {
		config.Realm = "SciFIND API"
	}
	
	// Create skip paths map
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}
	
	return func(c *gin.Context) {
		// Skip authentication for certain paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		
		// Get credentials from request
		username, password, hasAuth := c.Request.BasicAuth()
		
		// Check if credentials are provided
		if !hasAuth {
			c.Header("WWW-Authenticate", `Basic realm="`+config.Realm+`"`)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "authentication_required",
				"message":    "Basic authentication required",
				"request_id": GetRequestID(c),
			})
			c.Abort()
			return
		}
		
		// Validate credentials
		expectedPassword, exists := config.Users[username]
		if !exists || expectedPassword != password {
			c.Header("WWW-Authenticate", `Basic realm="`+config.Realm+`"`)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "invalid_credentials",
				"message":    "Invalid username or password",
				"request_id": GetRequestID(c),
			})
			c.Abort()
			return
		}
		
		// Store username in context
		c.Set("username", username)
		
		c.Next()
	}
}