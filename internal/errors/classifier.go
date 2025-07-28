package errors

import (
	"net/http"
	"strings"
)

// ErrorClassifier determines error type and handling strategy
type ErrorClassifier struct {
	transientCodes  map[int]bool
	permanentCodes  map[int]bool
	timeoutPatterns []string
	networkPatterns []string
	rateLimitPatterns []string
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		transientCodes: map[int]bool{
			http.StatusInternalServerError: true,
			http.StatusBadGateway:          true,
			http.StatusServiceUnavailable:  true,
			http.StatusGatewayTimeout:      true,
		},
		permanentCodes: map[int]bool{
			http.StatusBadRequest:          true,
			http.StatusUnauthorized:        true,
			http.StatusForbidden:           true,
			http.StatusNotFound:            true,
			http.StatusMethodNotAllowed:    true,
			http.StatusConflict:            true,
			http.StatusUnprocessableEntity: true,
		},
		timeoutPatterns: []string{
			"timeout",
			"deadline exceeded",
			"context canceled",
			"connection reset",
		},
		networkPatterns: []string{
			"connection refused",
			"no such host",
			"network unreachable",
			"connection reset",
			"broken pipe",
			"connection closed",
		},
		rateLimitPatterns: []string{
			"rate limit",
			"too many requests",
			"quota exceeded",
			"throttled",
		},
	}
}

// Classify determines the error type and creates a SciFindError
func (ec *ErrorClassifier) Classify(err error) *SciFindError {
	if err == nil {
		return nil
	}
	
	// Check if already classified
	if sciErr, ok := err.(*SciFindError); ok {
		return sciErr
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Classify based on error content
	switch {
	case ec.isTimeoutError(errStr):
		return NewError(ErrorTypeTimeout, "OPERATION_TIMEOUT", "Unknown operation timed out").
			WithCause(err).
			WithStack().
			Build()
	case ec.isNetworkError(errStr):
		return NewNetworkError("Network connectivity issue", err)
	case ec.isRateLimitError(errStr):
		return NewError(ErrorTypeRateLimit, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded").
			WithCause(err).
			WithStack().
			Build()
	case ec.isDatabaseError(errStr):
		return NewDatabaseError("database operation", err)
	default:
		return NewError(ErrorTypeTransient, "UNKNOWN", "Unknown error occurred").
			WithCause(err).
			WithStatusCode(http.StatusInternalServerError).
			WithStack().
			Retryable(false).
			Build()
	}
}

// ClassifyHTTPError classifies HTTP response errors
func (ec *ErrorClassifier) ClassifyHTTPError(statusCode int, body string) *SciFindError {
	switch {
	case ec.transientCodes[statusCode]:
		return NewError(ErrorTypeTransient, "HTTP_ERROR", "HTTP request failed").
			WithDetail("status_code", statusCode).
			WithDetail("response_body", body).
			WithStatusCode(statusCode).
			Build()
	case ec.permanentCodes[statusCode]:
		return NewError(ErrorTypePermanent, "HTTP_ERROR", "HTTP request failed").
			WithDetail("status_code", statusCode).
			WithDetail("response_body", body).
			WithStatusCode(statusCode).
			Retryable(false).
			Build()
	case statusCode == http.StatusTooManyRequests:
		return NewError(ErrorTypeRateLimit, "HTTP_RATE_LIMIT", "HTTP rate limit exceeded").
			WithDetail("status_code", statusCode).
			WithDetail("response_body", body).
			Build()
	case statusCode == http.StatusRequestTimeout:
		return NewError(ErrorTypeTimeout, "HTTP_TIMEOUT", "HTTP request timed out").
			WithDetail("status_code", statusCode).
			WithDetail("response_body", body).
			Build()
	default:
		return NewError(ErrorTypeTransient, "HTTP_ERROR", "HTTP request failed").
			WithDetail("status_code", statusCode).
			WithDetail("response_body", body).
			WithStatusCode(statusCode).
			Build()
	}
}

// isTimeoutError checks if the error is a timeout error
func (ec *ErrorClassifier) isTimeoutError(errStr string) bool {
	for _, pattern := range ec.timeoutPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isNetworkError checks if the error is a network error
func (ec *ErrorClassifier) isNetworkError(errStr string) bool {
	for _, pattern := range ec.networkPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isRateLimitError checks if the error is a rate limit error
func (ec *ErrorClassifier) isRateLimitError(errStr string) bool {
	for _, pattern := range ec.rateLimitPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isDatabaseError checks if the error is a database error
func (ec *ErrorClassifier) isDatabaseError(errStr string) bool {
	dbPatterns := []string{
		"database",
		"sql",
		"connection pool",
		"deadlock",
		"constraint",
		"foreign key",
		"duplicate key",
		"table doesn't exist",
		"column doesn't exist",
	}
	
	for _, pattern := range dbPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isProviderError checks if the error is from an external provider
func (ec *ErrorClassifier) isProviderError(errStr string, provider string) bool {
	providerPatterns := map[string][]string{
		"arxiv": {
			"arxiv",
			"export.arxiv.org",
		},
		"semantic_scholar": {
			"semantic scholar",
			"semanticscholar.org",
		},
		"exa": {
			"exa",
			"api.exa.ai",
		},
		"tavily": {
			"tavily",
			"api.tavily.com",
		},
	}
	
	if patterns, exists := providerPatterns[provider]; exists {
		for _, pattern := range patterns {
			if strings.Contains(errStr, pattern) {
				return true
			}
		}
	}
	
	return false
}

// ClassifyProviderError classifies provider-specific errors
func (ec *ErrorClassifier) ClassifyProviderError(provider string, err error) *SciFindError {
	if err == nil {
		return nil
	}
	
	errStr := strings.ToLower(err.Error())
	
	switch provider {
	case "arxiv":
		return ec.classifyArxivError(err, errStr)
	case "semantic_scholar":
		return ec.classifySemanticScholarError(err, errStr)
	case "exa":
		return ec.classifyExaError(err, errStr)
	case "tavily":
		return ec.classifyTavilyError(err, errStr)
	default:
		return NewProviderError(provider, "Provider error occurred", err)
	}
}

// classifyArxivError classifies ArXiv-specific errors
func (ec *ErrorClassifier) classifyArxivError(err error, errStr string) *SciFindError {
	switch {
	case strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429"):
		return NewError(ErrorTypeRateLimit, "ARXIV_RATE_LIMIT", "ArXiv API rate limit exceeded").
			WithComponent("arxiv_provider").
			WithCause(err).
			WithDetail("rate_limit", "1 request per 3 seconds").
			WithStack().
			Build()
	case ec.isTimeoutError(errStr):
		return NewError(ErrorTypeTimeout, "ARXIV_TIMEOUT", "ArXiv API request timed out").
			WithComponent("arxiv_provider").
			WithCause(err).
			WithStack().
			Build()
	case ec.isNetworkError(errStr):
		return NewNetworkError("Failed to connect to ArXiv API", err)
	default:
		return NewProviderError("arxiv", "ArXiv API error", err)
	}
}

// classifySemanticScholarError classifies Semantic Scholar-specific errors
func (ec *ErrorClassifier) classifySemanticScholarError(err error, errStr string) *SciFindError {
	switch {
	case strings.Contains(errStr, "quota exceeded") || strings.Contains(errStr, "rate limit"):
		return NewError(ErrorTypeRateLimit, "SS_RATE_LIMIT", "Semantic Scholar API rate limit exceeded").
			WithComponent("semantic_scholar_provider").
			WithCause(err).
			WithStack().
			Build()
	case strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "401"):
		return NewAuthenticationError("Semantic Scholar API authentication failed")
	default:
		return NewProviderError("semantic_scholar", "Semantic Scholar API error", err)
	}
}

// classifyExaError classifies Exa-specific errors
func (ec *ErrorClassifier) classifyExaError(err error, errStr string) *SciFindError {
	switch {
	case strings.Contains(errStr, "insufficient credits"):
		return NewError(ErrorTypeResource, "EXA_INSUFFICIENT_CREDITS", "Exa API insufficient credits").
			WithComponent("exa_provider").
			WithCause(err).
			WithStatusCode(http.StatusPaymentRequired).
			WithDetail("action_required", "check billing and credit balance").
			Retryable(false).
			WithStack().
			Build()
	case strings.Contains(errStr, "invalid api key"):
		return NewAuthenticationError("Exa API key is invalid")
	default:
		return NewProviderError("exa", "Exa API error", err)
	}
}

// classifyTavilyError classifies Tavily-specific errors
func (ec *ErrorClassifier) classifyTavilyError(err error, errStr string) *SciFindError {
	switch {
	case ec.isRateLimitError(errStr):
		return NewError(ErrorTypeRateLimit, "TAVILY_RATE_LIMIT", "Tavily API rate limit exceeded").
			WithComponent("tavily_provider").
			WithCause(err).
			WithStack().
			Build()
	default:
		return NewProviderError("tavily", "Tavily API error", err)
	}
}

// Error Classification Helper Functions

// IsTimeoutError checks if an error is a timeout error
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	
	if sciErr, ok := err.(*SciFindError); ok {
		return sciErr.Type == ErrorTypeTimeout
	}
	
	classifier := NewErrorClassifier()
	classifiedErr := classifier.Classify(err)
	return classifiedErr.Type == ErrorTypeTimeout
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	
	if sciErr, ok := err.(*SciFindError); ok {
		return sciErr.Type == ErrorTypeRateLimit
	}
	
	classifier := NewErrorClassifier()
	classifiedErr := classifier.Classify(err)
	return classifiedErr.Type == ErrorTypeRateLimit
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	
	if sciErr, ok := err.(*SciFindError); ok {
		return sciErr.Type == ErrorTypeNetwork
	}
	
	classifier := NewErrorClassifier()
	classifiedErr := classifier.Classify(err)
	return classifiedErr.Type == ErrorTypeNetwork
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	
	if sciErr, ok := err.(*SciFindError); ok {
		return sciErr.Type == ErrorTypeValidation
	}
	
	classifier := NewErrorClassifier()
	classifiedErr := classifier.Classify(err)
	return classifiedErr.Type == ErrorTypeValidation
}