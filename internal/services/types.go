package services

import (
	"time"

	"scifind-backend/internal/models"
	"scifind-backend/internal/providers"
)

// SearchRequest represents a search request from the API layer
type SearchRequest struct {
	RequestID string            `json:"request_id" validate:"required"`
	Query     string            `json:"query" validate:"required,min=1,max=1000"`
	Limit     int               `json:"limit,omitempty" validate:"min=1,max=100"`
	Offset    int               `json:"offset,omitempty" validate:"min=0"`
	Providers []string          `json:"providers,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
	DateFrom  *time.Time        `json:"date_from,omitempty"`
	DateTo    *time.Time        `json:"date_to,omitempty"`
	UserID    *string           `json:"user_id,omitempty"`
}

// SearchResponse represents a search response to the API layer
type SearchResponse struct {
	RequestID            string                   `json:"request_id"`
	Query                string                   `json:"query"`
	Papers               []models.Paper           `json:"papers"`
	TotalCount          int                      `json:"total_count"`
	ResultCount         int                      `json:"result_count"`
	ProvidersUsed       []string                 `json:"providers_used"`
	ProvidersFailed     []string                 `json:"providers_failed,omitempty"`
	Duration            time.Duration            `json:"duration"`
	AggregationStrategy string                   `json:"aggregation_strategy"`
	CacheHits           int                      `json:"cache_hits"`
	PartialFailure      bool                     `json:"partial_failure"`
	Errors              []providers.ProviderError `json:"errors,omitempty"`
	Timestamp           time.Time                `json:"timestamp"`
}

// PaperRequest represents a request to get a specific paper
type PaperRequest struct {
	ProviderName string `json:"provider_name" validate:"required"`
	PaperID      string `json:"paper_id" validate:"required"`
	UserID       *string `json:"user_id,omitempty"`
}

// PaperResponse represents a response containing a specific paper
type PaperResponse struct {
	Paper     *models.Paper `json:"paper"`
	Source    string        `json:"source"`
	Timestamp time.Time     `json:"timestamp"`
}

// ProviderStatusRequest represents a request for provider status
type ProviderStatusRequest struct {
	ProviderName *string `json:"provider_name,omitempty"` // If nil, return all providers
}

// ProviderStatusResponse represents provider status information
type ProviderStatusResponse struct {
	Providers map[string]providers.ProviderStatus `json:"providers"`
	Timestamp time.Time                           `json:"timestamp"`
}

// ProviderMetricsRequest represents a request for provider metrics
type ProviderMetricsRequest struct {
	ProviderName *string   `json:"provider_name,omitempty"` // If nil, return all providers
	TimeRange    *string   `json:"time_range,omitempty"`    // e.g., "1h", "24h", "7d"
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
}

// ProviderMetricsResponse represents provider metrics information
type ProviderMetricsResponse struct {
	Providers map[string]providers.ProviderMetrics `json:"providers"`
	TimeRange string                               `json:"time_range,omitempty"`
	StartTime *time.Time                           `json:"start_time,omitempty"`
	EndTime   *time.Time                           `json:"end_time,omitempty"`
	Timestamp time.Time                            `json:"timestamp"`
}

// ProviderConfigRequest represents a request to update provider configuration
type ProviderConfigRequest struct {
	ProviderName string                     `json:"provider_name" validate:"required"`
	Config       providers.ProviderConfig   `json:"config" validate:"required"`
}

// ProviderConfigResponse represents a response after updating provider configuration
type ProviderConfigResponse struct {
	ProviderName string                   `json:"provider_name"`
	Status       providers.ProviderStatus `json:"status"`
	Message      string                   `json:"message"`
	Timestamp    time.Time                `json:"timestamp"`
}

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct {
	Component *string `json:"component,omitempty"` // If nil, check all components
	Deep      bool    `json:"deep,omitempty"`      // Perform deep health checks
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status     string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Components map[string]ComponentHealth `json:"components"`
	Timestamp  time.Time              `json:"timestamp"`
	Duration   time.Duration          `json:"duration"`
}

// ComponentHealth represents the health status of a single component
type ComponentHealth struct {
	Status      string        `json:"status"` // "healthy", "degraded", "unhealthy"
	Error       *string       `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	LastCheck   time.Time     `json:"last_check"`
	Duration    time.Duration `json:"duration"`
}

// AnalyticsRequest represents a request for search analytics
type AnalyticsRequest struct {
	TimeRange   string     `json:"time_range" validate:"required"` // e.g., "1h", "24h", "7d", "30d"
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Granularity string     `json:"granularity,omitempty"` // "hour", "day", "week"
	Filters     map[string]string `json:"filters,omitempty"`
}

// AnalyticsResponse represents analytics data
type AnalyticsResponse struct {
	TimeRange   string                    `json:"time_range"`
	StartTime   time.Time                 `json:"start_time"`
	EndTime     time.Time                 `json:"end_time"`
	Granularity string                    `json:"granularity"`
	Metrics     AnalyticsMetrics          `json:"metrics"`
	Trends      []AnalyticsTrend          `json:"trends"`
	TopQueries  []PopularQuery            `json:"top_queries"`
	Providers   map[string]ProviderStats  `json:"providers"`
	Timestamp   time.Time                 `json:"timestamp"`
}

// AnalyticsMetrics represents aggregate analytics metrics
type AnalyticsMetrics struct {
	TotalSearches        int64         `json:"total_searches"`
	UniqueUsers          int64         `json:"unique_users"`
	AvgResponseTime      time.Duration `json:"avg_response_time"`
	SuccessRate          float64       `json:"success_rate"`
	TotalResults         int64         `json:"total_results"`
	AvgResultsPerSearch  float64       `json:"avg_results_per_search"`
	CacheHitRate         float64       `json:"cache_hit_rate"`
	PopularTimeRanges    map[string]int64 `json:"popular_time_ranges"`
}

// AnalyticsTrend represents a trend data point
type AnalyticsTrend struct {
	Timestamp    time.Time `json:"timestamp"`
	Searches     int64     `json:"searches"`
	Users        int64     `json:"users"`
	ResponseTime float64   `json:"response_time_ms"`
	SuccessRate  float64   `json:"success_rate"`
}

// PopularQuery represents a popular search query
type PopularQuery struct {
	Query      string  `json:"query"`
	Count      int64   `json:"count"`
	SuccessRate float64 `json:"success_rate"`
	AvgResults float64 `json:"avg_results"`
}

// ProviderStats represents provider-specific statistics
type ProviderStats struct {
	Requests     int64         `json:"requests"`
	Successes    int64         `json:"successes"`
	Failures     int64         `json:"failures"`
	SuccessRate  float64       `json:"success_rate"`
	AvgResponse  time.Duration `json:"avg_response_time"`
	TotalResults int64         `json:"total_results"`
	AvgResults   float64       `json:"avg_results"`
}

// Validation helpers

// ValidateSearchRequest validates a search request
func (r *SearchRequest) ValidateSearchRequest() error {
	if r.Query == "" {
		return NewValidationError("query is required")
	}

	if len(r.Query) > 1000 {
		return NewValidationError("query too long (max 1000 characters)")
	}

	if r.Limit < 0 {
		return NewValidationError("limit must be non-negative")
	}

	if r.Limit > 100 {
		return NewValidationError("limit cannot exceed 100")
	}

	if r.Offset < 0 {
		return NewValidationError("offset must be non-negative")
	}

	// Validate date range
	if r.DateFrom != nil && r.DateTo != nil {
		if r.DateFrom.After(*r.DateTo) {
			return NewValidationError("date_from must be before date_to")
		}
	}

	// Validate providers
	validProviders := map[string]bool{
		"arxiv":            true,
		"semantic_scholar": true,
		"exa":              true,
		"tavily":           true,
	}

	for _, provider := range r.Providers {
		if !validProviders[provider] {
			return NewValidationError("invalid provider: " + provider)
		}
	}

	return nil
}

// SetDefaults sets default values for a search request
func (r *SearchRequest) SetDefaults() {
	if r.Limit <= 0 {
		r.Limit = 20
	}

	if r.Offset < 0 {
		r.Offset = 0
	}

	if r.Filters == nil {
		r.Filters = make(map[string]string)
	}
}

// GetValidProviders returns the list of valid provider names
func GetValidProviders() []string {
	return []string{"arxiv", "semantic_scholar", "exa", "tavily"}
}

// GetValidTimeRanges returns the list of valid time ranges for analytics
func GetValidTimeRanges() []string {
	return []string{"1h", "6h", "24h", "7d", "30d", "90d"}
}

// GetValidGranularities returns the list of valid granularities for analytics
func GetValidGranularities() []string {
	return []string{"hour", "day", "week"}
}

// Helper function to create validation errors
func NewValidationError(message string) error {
	return &ValidationError{Message: message}
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}