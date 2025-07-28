package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

// CorsConfig contains CORS configuration
type CorsConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// DefaultCorsConfig returns default CORS configuration
func DefaultCorsConfig() CorsConfig {
	return CorsConfig{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:8080",
			"https://scifind.ai",
		},
		AllowedMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"HEAD",
			"OPTIONS",
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"X-API-Key",
			"X-Request-ID",
			"X-Forwarded-For",
			"User-Agent",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

// CorsMiddleware returns CORS middleware with configuration
func CorsMiddleware(config CorsConfig) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowOrigins:     config.AllowedOrigins,
		AllowMethods:     config.AllowedMethods,
		AllowHeaders:     config.AllowedHeaders,
		ExposeHeaders:    config.ExposedHeaders,
		AllowCredentials: config.AllowCredentials,
		MaxAge:           config.MaxAge,
	}

	// Allow all origins in development
	if gin.Mode() == gin.DebugMode {
		corsConfig.AllowAllOrigins = true
	}

	return cors.New(corsConfig)
}