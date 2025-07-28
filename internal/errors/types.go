package errors

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// Transient errors - retry with backoff
	ErrorTypeTransient ErrorType = "transient"

	// Permanent errors - fail fast, no retry
	ErrorTypePermanent ErrorType = "permanent"

	// Circuit breaker errors - gradual recovery
	ErrorTypeCircuitBreaker ErrorType = "circuit_breaker"

	// Rate limit errors - specific backoff strategy
	ErrorTypeRateLimit ErrorType = "rate_limit"

	// Authentication errors - immediate escalation
	ErrorTypeAuth ErrorType = "authentication"

	// Validation errors - client-side fix required
	ErrorTypeValidation ErrorType = "validation"

	// Timeout errors - increase timeout or circuit break
	ErrorTypeTimeout ErrorType = "timeout"

	// Network errors - connection-specific handling
	ErrorTypeNetwork ErrorType = "network"

	// Resource exhaustion - backpressure application
	ErrorTypeResource ErrorType = "resource"
)

// SciFindError represents a structured error with context
type SciFindError struct {
	Type       ErrorType              `json:"type"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	Stack      string                 `json:"stack,omitempty"`
	Component  string                 `json:"component"`
	Operation  string                 `json:"operation"`
	Timestamp  time.Time              `json:"timestamp"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	Retryable  bool                   `json:"retryable"`
	StatusCode int                    `json:"status_code"`
}

// Error implements the error interface
func (e *SciFindError) Error() string {
	return fmt.Sprintf("[%s:%s] %s", e.Component, e.Code, e.Message)
}

// Is implements error matching for Go 1.13+ error handling
func (e *SciFindError) Is(target error) bool {
	if t, ok := target.(*SciFindError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// Unwrap implements error unwrapping for Go 1.13+ error handling
func (e *SciFindError) Unwrap() error {
	return e.Cause
}

// String returns a string representation of the error
func (e *SciFindError) String() string {
	return e.Error()
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *SciFindError) HTTPStatus() int {
	if e.StatusCode != 0 {
		return e.StatusCode
	}

	switch e.Type {
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeAuth:
		return http.StatusUnauthorized
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeTimeout:
		return http.StatusRequestTimeout
	case ErrorTypeNetwork, ErrorTypeTransient, ErrorTypeCircuitBreaker:
		return http.StatusServiceUnavailable
	case ErrorTypeResource:
		return http.StatusInsufficientStorage
	default:
		return http.StatusInternalServerError
	}
}

// ErrorBuilder helps build SciFindError instances
type ErrorBuilder struct {
	err *SciFindError
}

// NewError creates a new ErrorBuilder
func NewError(errorType ErrorType, code, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &SciFindError{
			Type:      errorType,
			Code:      code,
			Message:   message,
			Details:   make(map[string]interface{}),
			Timestamp: time.Now(),
			Retryable: errorType == ErrorTypeTransient || errorType == ErrorTypeTimeout || errorType == ErrorTypeNetwork,
		},
	}
}

// WithCause sets the underlying cause
func (b *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	b.err.Cause = cause
	return b
}

// WithComponent sets the component where the error occurred
func (b *ErrorBuilder) WithComponent(component string) *ErrorBuilder {
	b.err.Component = component
	return b
}

// WithOperation sets the operation that failed
func (b *ErrorBuilder) WithOperation(operation string) *ErrorBuilder {
	b.err.Operation = operation
	return b
}

// WithDetail adds a detail to the error
func (b *ErrorBuilder) WithDetail(key string, value interface{}) *ErrorBuilder {
	b.err.Details[key] = value
	return b
}

// WithDetails sets multiple details
func (b *ErrorBuilder) WithDetails(details map[string]interface{}) *ErrorBuilder {
	for k, v := range details {
		b.err.Details[k] = v
	}
	return b
}

// WithRequestID sets the request ID
func (b *ErrorBuilder) WithRequestID(requestID string) *ErrorBuilder {
	b.err.RequestID = requestID
	return b
}

// WithUserID sets the user ID
func (b *ErrorBuilder) WithUserID(userID string) *ErrorBuilder {
	b.err.UserID = userID
	return b
}

// WithStatusCode sets the HTTP status code
func (b *ErrorBuilder) WithStatusCode(statusCode int) *ErrorBuilder {
	b.err.StatusCode = statusCode
	return b
}

// WithStack captures the current stack trace
func (b *ErrorBuilder) WithStack() *ErrorBuilder {
	b.err.Stack = captureStack()
	return b
}

// Retryable sets whether the error is retryable
func (b *ErrorBuilder) Retryable(retryable bool) *ErrorBuilder {
	b.err.Retryable = retryable
	return b
}

// Build returns the constructed error
func (b *ErrorBuilder) Build() *SciFindError {
	return b.err
}

// Predefined error constructors

// NewValidationError creates a validation error
func NewValidationError(message string, field string, value interface{}) *SciFindError {
	return NewError(ErrorTypeValidation, "VALIDATION_ERROR", message).
		WithDetail("field", field).
		WithDetail("rejected_value", value).
		WithStatusCode(http.StatusBadRequest).
		Build()
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string, id string) *SciFindError {
	return NewError(ErrorTypePermanent, "NOT_FOUND", fmt.Sprintf("%s not found", resource)).
		WithDetail("resource", resource).
		WithDetail("id", id).
		WithStatusCode(http.StatusNotFound).
		Retryable(false).
		Build()
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(message string) *SciFindError {
	return NewError(ErrorTypeAuth, "AUTHENTICATION_FAILED", message).
		WithStatusCode(http.StatusUnauthorized).
		Retryable(false).
		Build()
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(message string, retryAfter time.Duration) *SciFindError {
	return NewError(ErrorTypeRateLimit, "RATE_LIMIT_EXCEEDED", message).
		WithDetail("retry_after", retryAfter.String()).
		WithStatusCode(http.StatusTooManyRequests).
		Build()
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(operation string, timeout time.Duration) *SciFindError {
	return NewError(ErrorTypeTimeout, "OPERATION_TIMEOUT", fmt.Sprintf("Operation %s timed out", operation)).
		WithOperation(operation).
		WithDetail("timeout", timeout.String()).
		WithStatusCode(http.StatusRequestTimeout).
		Build()
}

// NewNetworkError creates a network error
func NewNetworkError(message string, cause error) *SciFindError {
	return NewError(ErrorTypeNetwork, "NETWORK_ERROR", message).
		WithCause(cause).
		WithStatusCode(http.StatusServiceUnavailable).
		Build()
}

// NewCircuitBreakerError creates a circuit breaker error
func NewCircuitBreakerError(service string) *SciFindError {
	return NewError(ErrorTypeCircuitBreaker, "CIRCUIT_OPEN", fmt.Sprintf("Circuit breaker open for %s", service)).
		WithDetail("service", service).
		WithStatusCode(http.StatusServiceUnavailable).
		Build()
}

// NewDatabaseError creates a database error
func NewDatabaseError(operation string, cause error) *SciFindError {
	return NewError(ErrorTypeTransient, "DATABASE_ERROR", "Database operation failed").
		WithOperation(operation).
		WithCause(cause).
		WithComponent("database").
		WithStatusCode(http.StatusInternalServerError).
		Build()
}

// NewProviderError creates a provider error
func NewProviderError(provider string, message string, cause error) *SciFindError {
	return NewError(ErrorTypeTransient, "PROVIDER_ERROR", message).
		WithComponent(fmt.Sprintf("%s_provider", provider)).
		WithCause(cause).
		WithDetail("provider", provider).
		WithStatusCode(http.StatusServiceUnavailable).
		Build()
}

// captureStack captures the current stack trace
func captureStack() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])

	var buf strings.Builder
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		fmt.Fprintf(&buf, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}

	return buf.String()
}

// Common error variables for easy reuse
var (
	ErrInvalidInput = NewError(ErrorTypeValidation, "INVALID_INPUT", "Invalid input provided").Build()
	ErrUnauthorized = NewError(ErrorTypeAuth, "UNAUTHORIZED", "Authentication required").WithStatusCode(http.StatusUnauthorized).Build()
	ErrForbidden    = NewError(ErrorTypeAuth, "FORBIDDEN", "Access denied").WithStatusCode(http.StatusForbidden).Build()
	ErrInternal     = NewError(ErrorTypePermanent, "INTERNAL_ERROR", "Internal server error").WithStatusCode(http.StatusInternalServerError).Build()
)

// NewMessagingError creates a messaging error
func NewMessagingError(message string, details map[string]interface{}) *SciFindError {
	return NewError(ErrorTypeTransient, "MESSAGING_ERROR", message).
		WithDetails(details).
		WithStatusCode(http.StatusServiceUnavailable).
		Build()
}

// NewSerializationError creates a serialization error
func NewSerializationError(message string, data interface{}) *SciFindError {
	return NewError(ErrorTypePermanent, "SERIALIZATION_ERROR", message).
		WithDetail("data", data).
		WithStatusCode(http.StatusBadRequest).
		Build()
}

// NewDuplicateError creates a duplicate error
func NewDuplicateError(message string, key string) *SciFindError {
	return NewError(ErrorTypePermanent, "DUPLICATE_ERROR", message).
		WithDetail("key", key).
		WithStatusCode(http.StatusConflict).
		Build()
}

// NewInternalError creates an internal error
func NewInternalError(message string, err error) *SciFindError {
	builder := NewError(ErrorTypePermanent, "INTERNAL_ERROR", message).
		WithStatusCode(http.StatusInternalServerError)

	if err != nil {
		builder = builder.WithCause(err)
	}

	return builder.Build()
}

// IsDuplicateKeyError checks if error is a duplicate key error
func IsDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return strings.Contains(errMsg, "duplicate key") ||
		strings.Contains(errMsg, "UNIQUE constraint") ||
		strings.Contains(errMsg, "already exists")
}

// NewHealthCheckError creates a health check error
func NewHealthCheckError(message string, component string) *SciFindError {
	return NewError(ErrorTypeTransient, "HEALTH_CHECK_ERROR", message).
		WithComponent(component).
		WithOperation("health_check").
		WithStatusCode(http.StatusServiceUnavailable).
		Build()
}
