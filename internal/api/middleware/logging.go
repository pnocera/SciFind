package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggingConfig contains logging middleware configuration
type LoggingConfig struct {
	Logger       *slog.Logger
	SkipPaths    []string
	LogLatency   bool
	LogUserAgent bool
	LogReferer   bool
}

// LoggingMiddleware returns request logging middleware
func LoggingMiddleware(config LoggingConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			// Skip certain paths like health checks
			if skipPaths[param.Path] {
				return ""
			}

			// Extract request ID from context
			requestID := param.Keys["request_id"]
			if requestID == nil {
				requestID = "unknown"
			}

			// Log structured request
			fields := []any{
				slog.String("method", param.Method),
				slog.String("path", param.Path),
				slog.Int("status", param.StatusCode),
				slog.String("client_ip", param.ClientIP),
				slog.String("request_id", requestID.(string)),
				slog.Duration("latency", param.Latency),
				slog.Int("body_size", param.BodySize),
			}

			if config.LogUserAgent && param.Request != nil {
				fields = append(fields, slog.String("user_agent", param.Request.UserAgent()))
			}

			if config.LogReferer && param.Request != nil {
				fields = append(fields, slog.String("referer", param.Request.Referer()))
			}

			if param.ErrorMessage != "" {
				fields = append(fields, slog.String("error", param.ErrorMessage))
				config.Logger.Error("HTTP request completed with error", fields...)
			} else {
				config.Logger.Info("HTTP request completed", fields...)
			}

			return ""
		},
		SkipPaths: config.SkipPaths,
	})
}

// StructuredLoggingMiddleware provides structured logging with slog
func StructuredLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Get request ID from context
		requestID, exists := c.Get("request_id")
		if !exists {
			requestID = "unknown"
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Build log entry
		fields := []any{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", raw),
			slog.Int("status", c.Writer.Status()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("request_id", requestID.(string)),
			slog.Duration("latency", latency),
			slog.Int("body_size", c.Writer.Size()),
		}

		// Add error information if present
		if len(c.Errors) > 0 {
			fields = append(fields, slog.String("errors", c.Errors.String()))
		}

		// Log based on status code
		switch {
		case c.Writer.Status() >= 500:
			logger.Error("Server error", fields...)
		case c.Writer.Status() >= 400:
			logger.Warn("Client error", fields...)
		default:
			logger.Info("Request completed", fields...)
		}
	}
}