package providers

import (
	"context"
	"fmt"
	"time"

	"scifind-backend/internal/models"
)

// SearchProvider defines the interface for all search providers
type SearchProvider interface {
	// Provider information
	Name() string
	IsEnabled() bool
	GetCapabilities() ProviderCapabilities
	
	// Search operations
	Search(ctx context.Context, query *SearchQuery) (*SearchResult, error)
	GetPaper(ctx context.Context, id string) (*models.Paper, error)
	
	// Health and status
	HealthCheck(ctx context.Context) error
	GetStatus() ProviderStatus
	GetMetrics() ProviderMetrics
	
	// Configuration
	Configure(config ProviderConfig) error
	ValidateConfig(config ProviderConfig) error
}

// ProviderManager manages multiple search providers
type ProviderManager interface {
	// Provider management
	RegisterProvider(name string, provider SearchProvider) error
	GetProvider(name string) (SearchProvider, error)
	GetEnabledProviders() []SearchProvider
	GetAllProviders() map[string]SearchProvider
	
	// Search operations
	SearchAll(ctx context.Context, query *SearchQuery) (*AggregatedResult, error)
	SearchProviders(ctx context.Context, query *SearchQuery, providerNames []string) (*AggregatedResult, error)
	
	// Health and monitoring
	HealthCheckAll(ctx context.Context) map[string]error
	GetProviderMetrics() map[string]ProviderMetrics
	
	// Configuration
	UpdateProviderConfig(name string, config ProviderConfig) error
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// RateLimiter defines rate limiting interface
type RateLimiter interface {
	Allow(ctx context.Context, provider string) bool
	Wait(ctx context.Context, provider string) error
	GetLimits(provider string) RateLimitInfo
	UpdateLimits(provider string, limits RateLimitConfig) error
}

// CacheManager defines caching interface for providers
type CacheManager interface {
	Get(ctx context.Context, key string) (*CachedResult, error)
	Set(ctx context.Context, key string, result *SearchResult, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context, pattern string) error
	GetStats() CacheStats
}

// CircuitBreaker defines circuit breaker interface for providers
type CircuitBreaker interface {
	Execute(ctx context.Context, provider string, fn func() error) error
	GetState(provider string) CircuitState
	ForceOpen(provider string)
	ForceClose(provider string)
	Reset(provider string)
}

// Data Structures

// SearchQuery represents a search query to providers
type SearchQuery struct {
	// Core query
	Query    string            `json:"query"`
	Filters  map[string]string `json:"filters,omitempty"`
	
	// Pagination
	Limit    int `json:"limit"`
	Offset   int `json:"offset"`
	
	// Sorting
	SortBy   string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
	
	// Date range
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`
	
	// Categories/fields
	Categories []string `json:"categories,omitempty"`
	Authors    []string `json:"authors,omitempty"`
	
	// Additional options
	IncludeFullText bool              `json:"include_full_text"`
	Language        string            `json:"language,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	
	// Request context
	RequestID string `json:"request_id"`
	UserID    *string `json:"user_id,omitempty"`
}

// SearchResult represents search results from a provider
type SearchResult struct {
	// Results
	Papers      []models.Paper `json:"papers"`
	TotalCount  int            `json:"total_count"`
	ResultCount int            `json:"result_count"`
	
	// Query info
	Query     string `json:"query"`
	Provider  string `json:"provider"`
	
	// Performance
	Duration    time.Duration `json:"duration"`
	CacheHit    bool          `json:"cache_hit"`
	
	// Metadata
	RequestID   string                 `json:"request_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	
	// Pagination
	HasMore     bool   `json:"has_more"`
	NextCursor  *string `json:"next_cursor,omitempty"`
	
	// Status
	Success     bool   `json:"success"`
	Error       error  `json:"error,omitempty"`
}

// AggregatedResult represents aggregated results from multiple providers
type AggregatedResult struct {
	// Aggregated results
	Papers          []models.Paper     `json:"papers"`
	TotalCount      int                `json:"total_count"`
	ProviderResults map[string]*SearchResult `json:"provider_results"`
	
	// Query info
	Query           string             `json:"query"`
	RequestedProviders []string        `json:"requested_providers"`
	SuccessfulProviders []string       `json:"successful_providers"`
	FailedProviders  []string          `json:"failed_providers"`
	
	// Performance
	TotalDuration   time.Duration      `json:"total_duration"`
	CacheHits       int                `json:"cache_hits"`
	
	// Metadata
	RequestID       string             `json:"request_id"`
	Timestamp       time.Time          `json:"timestamp"`
	AggregationStrategy string         `json:"aggregation_strategy"`
	
	// Status
	PartialFailure  bool               `json:"partial_failure"`
	Errors          []ProviderError    `json:"errors,omitempty"`
}

// ProviderCapabilities describes what a provider can do
type ProviderCapabilities struct {
	// Search features
	SupportsFullText    bool     `json:"supports_full_text"`
	SupportsDateFilter  bool     `json:"supports_date_filter"`
	SupportsAuthFilter  bool     `json:"supports_author_filter"`
	SupportsCategoryFilter bool  `json:"supports_category_filter"`
	SupportsSort        bool     `json:"supports_sort"`
	
	// Content types
	SupportedFields     []string `json:"supported_fields"`
	SupportedLanguages  []string `json:"supported_languages"`
	SupportedFormats    []string `json:"supported_formats"`
	
	// Technical limits
	MaxResults          int      `json:"max_results"`
	MaxQueryLength      int      `json:"max_query_length"`
	RateLimit          int      `json:"rate_limit_per_minute"`
	
	// Features
	SupportsRealtime    bool     `json:"supports_realtime"`
	SupportsExactMatch  bool     `json:"supports_exact_match"`
	SupportsFuzzySearch bool     `json:"supports_fuzzy_search"`
	SupportsWildcards   bool     `json:"supports_wildcards"`
}

// ProviderStatus represents the current status of a provider
type ProviderStatus struct {
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	Healthy     bool      `json:"healthy"`
	LastCheck   time.Time `json:"last_check"`
	LastError   error     `json:"last_error,omitempty"`
	CircuitState string   `json:"circuit_state"`
	
	// Rate limiting
	RateLimited bool      `json:"rate_limited"`
	ResetTime   *time.Time `json:"reset_time,omitempty"`
	
	// Performance
	AvgResponseTime time.Duration `json:"avg_response_time"`
	SuccessRate     float64       `json:"success_rate"`
	
	// Versioning
	APIVersion  string    `json:"api_version,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

// ProviderMetrics contains performance metrics for a provider
type ProviderMetrics struct {
	// Request statistics
	TotalRequests     int64         `json:"total_requests"`
	SuccessfulRequests int64        `json:"successful_requests"`
	FailedRequests    int64         `json:"failed_requests"`
	CachedRequests    int64         `json:"cached_requests"`
	
	// Performance metrics
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	MinResponseTime   time.Duration `json:"min_response_time"`
	MaxResponseTime   time.Duration `json:"max_response_time"`
	P95ResponseTime   time.Duration `json:"p95_response_time"`
	
	// Error statistics
	TimeoutErrors     int64         `json:"timeout_errors"`
	RateLimitErrors   int64         `json:"rate_limit_errors"`
	NetworkErrors     int64         `json:"network_errors"`
	ParseErrors       int64         `json:"parse_errors"`
	
	// Rate limiting
	RateLimitHits     int64         `json:"rate_limit_hits"`
	RateLimitResets   int64         `json:"rate_limit_resets"`
	
	// Circuit breaker
	CircuitOpenCount  int64         `json:"circuit_open_count"`
	CircuitCloseCount int64         `json:"circuit_close_count"`
	
	// Result statistics
	TotalResults      int64         `json:"total_results"`
	AvgResultsPerQuery float64      `json:"avg_results_per_query"`
	
	// Time window
	WindowStart       time.Time     `json:"window_start"`
	WindowEnd         time.Time     `json:"window_end"`
}

// Configuration structures

// ProviderConfig contains configuration for a provider
type ProviderConfig struct {
	// Basic config
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	
	// API configuration
	BaseURL     string            `json:"base_url"`
	APIKey      string            `json:"api_key,omitempty"`
	APISecret   string            `json:"api_secret,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	
	// Timeouts
	Timeout     time.Duration     `json:"timeout"`
	RetryDelay  time.Duration     `json:"retry_delay"`
	MaxRetries  int               `json:"max_retries"`
	
	// Rate limiting
	RateLimit   RateLimitConfig   `json:"rate_limit"`
	
	// Circuit breaker
	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`
	
	// Caching
	Cache       CacheConfig       `json:"cache"`
	
	// Custom settings
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

// RateLimitConfig configures rate limiting for a provider
type RateLimitConfig struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	RequestsPerMinute int           `json:"requests_per_minute"`
	RequestsPerHour   int           `json:"requests_per_hour"`
	BurstSize         int           `json:"burst_size"`
	BackoffDuration   time.Duration `json:"backoff_duration"`
}

// RateLimitInfo contains current rate limit information
type RateLimitInfo struct {
	Remaining     int       `json:"remaining"`
	Limit         int       `json:"limit"`
	ResetTime     time.Time `json:"reset_time"`
	RetryAfter    time.Duration `json:"retry_after"`
}

// CircuitBreakerConfig configures circuit breaker for a provider
type CircuitBreakerConfig struct {
	FailureThreshold  int           `json:"failure_threshold"`
	SuccessThreshold  int           `json:"success_threshold"`
	Timeout           time.Duration `json:"timeout"`
	MaxRequests       int           `json:"max_requests"`
	Interval          time.Duration `json:"interval"`
}

// CircuitState represents circuit breaker state
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

// CacheConfig configures caching for a provider
type CacheConfig struct {
	Enabled   bool          `json:"enabled"`
	TTL       time.Duration `json:"ttl"`
	MaxSize   int           `json:"max_size"`
	KeyPrefix string        `json:"key_prefix"`
}

// CachedResult represents a cached search result
type CachedResult struct {
	Result    *SearchResult `json:"result"`
	CachedAt  time.Time     `json:"cached_at"`
	ExpiresAt time.Time     `json:"expires_at"`
	HitCount  int           `json:"hit_count"`
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	Size        int     `json:"size"`
	MaxSize     int     `json:"max_size"`
	Evictions   int64   `json:"evictions"`
}

// Error types

// ProviderError represents an error from a specific provider
type ProviderError struct {
	Provider string `json:"provider"`
	Error    error  `json:"error"`
	Type     string `json:"type"` // timeout, rate_limit, network, parse, etc.
	Retryable bool  `json:"retryable"`
}

// AggregationStrategy defines how to aggregate results from multiple providers
type AggregationStrategy string

const (
	// Merge all results with deduplication
	StrategyMerge AggregationStrategy = "merge"
	
	// Use first successful provider
	StrategyFirst AggregationStrategy = "first"
	
	// Use fastest provider
	StrategyFastest AggregationStrategy = "fastest"
	
	// Use most accurate provider (based on quality scores)
	StrategyBestQuality AggregationStrategy = "best_quality"
	
	// Round-robin across providers
	StrategyRoundRobin AggregationStrategy = "round_robin"
)

// Provider names (constants for consistency)
const (
	ProviderArxiv           = "arxiv"
	ProviderSemanticScholar = "semantic_scholar"
	ProviderExa             = "exa"
	ProviderTavily          = "tavily"
	ProviderCrossRef        = "crossref"
	ProviderPubMed          = "pubmed"
)

// Common search filters
const (
	FilterDateFrom      = "date_from"
	FilterDateTo        = "date_to"
	FilterAuthor        = "author"
	FilterCategory      = "category"
	FilterJournal       = "journal"
	FilterLanguage      = "language"
	FilterMinCitations  = "min_citations"
	FilterMaxCitations  = "max_citations"
	FilterPaperType     = "paper_type"
	FilterOpenAccess    = "open_access"
)

// Sort options
const (
	SortRelevance   = "relevance"
	SortDate        = "date"
	SortCitations   = "citations"
	SortQuality     = "quality"
	SortTitle       = "title"
	SortAuthor      = "author"
)

// Helper functions

// NewSearchQuery creates a new search query with defaults
func NewSearchQuery(query string) *SearchQuery {
	return &SearchQuery{
		Query:           query,
		Limit:           20,
		Offset:          0,
		SortBy:          SortRelevance,
		SortOrder:       "desc",
		IncludeFullText: false,
		Language:        "en",
		Filters:         make(map[string]string),
		Metadata:        make(map[string]interface{}),
		RequestID:       generateRequestID(),
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%s", time.Now().UnixNano(), generateRandomString(8))
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