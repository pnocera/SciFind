package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// LogLevel represents logging levels
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return "info"
	}
}

// ToSlogLevel converts LogLevel to slog.Level
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// RequestContext contains request-specific information for logging
type RequestContext struct {
	RequestID string            `json:"request_id"`
	UserID    string            `json:"user_id"`
	Operation string            `json:"operation"`
	StartTime time.Time         `json:"start_time"`
	Metadata  map[string]string `json:"metadata"`
	TraceID   string            `json:"trace_id"`
	SpanID    string            `json:"span_id"`
}

type contextKey string

const requestContextKey contextKey = "request_context"

// NewLogger creates a new structured logger based on configuration
func NewLogger(config *Config) (*slog.Logger, error) {
	var handler slog.Handler
	
	level := parseLogLevel(config.Logging.Level)
	
	opts := &slog.HandlerOptions{
		Level:     level.ToSlogLevel(),
		AddSource: config.Logging.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	var output *os.File
	switch config.Logging.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "file":
		if config.Logging.FilePath == "" {
			return nil, fmt.Errorf("file path required when output is file")
		}
		file, err := os.OpenFile(config.Logging.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	default:
		output = os.Stdout
	}

	if config.Logging.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	logger := slog.New(handler)
	
	// Set as default logger
	slog.SetDefault(logger)
	
	return logger, nil
}

// WithRequestContext adds request context to the context
func WithRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	return context.WithValue(ctx, requestContextKey, reqCtx)
}

// GetRequestContext retrieves request context from the context
func GetRequestContext(ctx context.Context) (*RequestContext, bool) {
	reqCtx, ok := ctx.Value(requestContextKey).(*RequestContext)
	return reqCtx, ok
}

// NewRequestContext creates a new request context
func NewRequestContext(operation string) *RequestContext {
	return &RequestContext{
		RequestID: generateRequestID(),
		Operation: operation,
		StartTime: time.Now(),
		Metadata:  make(map[string]string),
		TraceID:   generateTraceID(),
		SpanID:    generateSpanID(),
	}
}

// LogWithContext logs with request context if available
func LogWithContext(ctx context.Context, logger *slog.Logger, level slog.Level, msg string, args ...any) {
	if reqCtx, ok := GetRequestContext(ctx); ok {
		args = append(args,
			slog.String("request_id", reqCtx.RequestID),
			slog.String("trace_id", reqCtx.TraceID),
			slog.String("span_id", reqCtx.SpanID),
			slog.String("operation", reqCtx.Operation),
			slog.Duration("duration", time.Since(reqCtx.StartTime)),
		)
		
		if reqCtx.UserID != "" {
			args = append(args, slog.String("user_id", reqCtx.UserID))
		}
	}
	
	logger.Log(ctx, level, msg, args...)
}

// Helper functions for different log levels with context
func DebugWithContext(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	LogWithContext(ctx, logger, slog.LevelDebug, msg, args...)
}

func InfoWithContext(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	LogWithContext(ctx, logger, slog.LevelInfo, msg, args...)
}

func WarnWithContext(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	LogWithContext(ctx, logger, slog.LevelWarn, msg, args...)
}

func ErrorWithContext(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	LogWithContext(ctx, logger, slog.LevelError, msg, args...)
}

// parseLogLevel parses string log level to LogLevel
func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%s", time.Now().Unix(), generateRandomString(8))
}

// generateTraceID generates a trace ID
func generateTraceID() string {
	return fmt.Sprintf("trace_%s", generateRandomString(16))
}

// generateSpanID generates a span ID
func generateSpanID() string {
	return fmt.Sprintf("span_%s", generateRandomString(8))
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}