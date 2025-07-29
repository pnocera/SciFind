package models

import (
	"time"

	"scifind-backend/internal/errors"
)

// SearchRequest represents a search query request
type SearchRequest struct {
	Query       string            `json:"query" validate:"required,min=1,max=1000"`
	Providers   []string          `json:"providers" validate:"omitempty,dive,oneof=arxiv semantic_scholar exa tavily"`
	Categories  []string          `json:"categories" validate:"omitempty,dive,min=1,max=100"`
	DateRange   *DateRange        `json:"date_range,omitempty" validate:"omitempty"`
	Language    string            `json:"language" validate:"omitempty,len=2"`
	Limit       int               `json:"limit" validate:"min=1,max=100"`
	Offset      int               `json:"offset" validate:"min=0"`
	SortBy      string            `json:"sort_by" validate:"omitempty,oneof=relevance date citations quality"`
	SortOrder   string            `json:"sort_order" validate:"omitempty,oneof=asc desc"`
	Filters     map[string]string `json:"filters" validate:"omitempty"`
	IncludeText bool              `json:"include_text"`
}

// DateRange represents a date filtering range
type DateRange struct {
	From *time.Time `json:"from,omitempty" validate:"omitempty"`
	To   *time.Time `json:"to,omitempty" validate:"omitempty,gtfield=From"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Query       string        `json:"query"`
	TotalCount  int           `json:"total_count"`
	ResultCount int           `json:"result_count"`
	Limit       int           `json:"limit"`
	Offset      int           `json:"offset"`
	Duration    string        `json:"duration"`
	Papers      []Paper       `json:"papers"`
	Facets      *SearchFacets `json:"facets,omitempty"`
	Suggestions []string      `json:"suggestions,omitempty"`
	NextCursor  *string       `json:"next_cursor,omitempty"`
	Stats       *SearchStats  `json:"stats,omitempty"`
}

// SearchFacets represents search result facets
type SearchFacets struct {
	Categories map[string]int `json:"categories"`
	Authors    map[string]int `json:"authors"`
	Journals   map[string]int `json:"journals"`
	Years      map[string]int `json:"years"`
	Languages  map[string]int `json:"languages"`
	Providers  map[string]int `json:"providers"`
}

// SearchStats represents search operation statistics
type SearchStats struct {
	QueryID      string          `json:"query_id"`
	Query        string          `json:"query"`
	Providers    []ProviderStats `json:"providers"`
	TotalTime    time.Duration   `json:"total_time"`
	CacheHit     bool            `json:"cache_hit"`
	ResultsFound int             `json:"results_found"`
	Timestamp    time.Time       `json:"timestamp"`
}

// ProviderStats represents per-provider search statistics
type ProviderStats struct {
	Name         string        `json:"name"`
	ResponseTime time.Duration `json:"response_time"`
	ResultCount  int           `json:"result_count"`
	Error        *string       `json:"error,omitempty"`
	CacheHit     bool          `json:"cache_hit"`
	StatusCode   int           `json:"status_code"`
}

// SearchQuery represents an internal search query
type SearchQuery struct {
	ID          string            `json:"id"`
	Text        string            `json:"text"`
	Filters     map[string]string `json:"filters"`
	Providers   []string          `json:"providers"`
	Limit       int               `json:"limit"`
	Offset      int               `json:"offset"`
	SortBy      string            `json:"sort_by"`
	SortOrder   string            `json:"sort_order"`
	RequestedAt time.Time         `json:"requested_at"`
}

// SearchResult represents search results from a provider
type SearchResult struct {
	Provider    string        `json:"provider"`
	Query       string        `json:"query"`
	Papers      []Paper       `json:"papers"`
	TotalCount  int           `json:"total_count"`
	Duration    time.Duration `json:"duration"`
	Error       error         `json:"error,omitempty"`
	CacheHit    bool          `json:"cache_hit"`
	RequestedAt time.Time     `json:"requested_at"`
}

// AggregatedResult represents aggregated search results from multiple providers
type AggregatedResult struct {
	Query         string          `json:"query"`
	Papers        []Paper         `json:"papers"`
	TotalCount    int             `json:"total_count"`
	ProviderStats []ProviderStats `json:"provider_stats"`
	Duration      time.Duration   `json:"duration"`
	CacheHits     int             `json:"cache_hits"`
	Errors        []string        `json:"errors,omitempty"`
}

// SearchSuggestion represents a search suggestion
type SearchSuggestion struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"`
	Type  string  `json:"type"` // query, author, journal, category
}

// DefaultSearchRequest returns a default search request
func DefaultSearchRequest() SearchRequest {
	return SearchRequest{
		Limit:     20,
		Offset:    0,
		SortBy:    "relevance",
		SortOrder: "desc",
		Language:  "en",
		Filters:   make(map[string]string),
	}
}

// Validate validates the search request
func (sr *SearchRequest) Validate() error {
	if sr.Query == "" {
		return errors.NewValidationError("Query is required", "query", sr.Query)
	}

	if sr.Limit <= 0 {
		sr.Limit = 20
	}
	if sr.Limit > 100 {
		sr.Limit = 100
	}

	if sr.Offset < 0 {
		sr.Offset = 0
	}

	if sr.SortBy == "" {
		sr.SortBy = "relevance"
	}

	if sr.SortOrder == "" {
		sr.SortOrder = "desc"
	}

	return nil
}

// GetEnabledProviders returns the list of enabled providers or default ones
func (sr *SearchRequest) GetEnabledProviders() []string {
	if len(sr.Providers) > 0 {
		return sr.Providers
	}

	// Default enabled providers
	return []string{"arxiv", "semantic_scholar"}
}

// HasDateFilter returns true if date filtering is requested
func (sr *SearchRequest) HasDateFilter() bool {
	return sr.DateRange != nil && (sr.DateRange.From != nil || sr.DateRange.To != nil)
}

// HasCategoryFilter returns true if category filtering is requested
func (sr *SearchRequest) HasCategoryFilter() bool {
	return len(sr.Categories) > 0
}

// HasProviderFilter returns true if specific providers are requested
func (sr *SearchRequest) HasProviderFilter() bool {
	return len(sr.Providers) > 0
}

// GetFilter returns a filter value by key
func (sr *SearchRequest) GetFilter(key string) (string, bool) {
	if sr.Filters == nil {
		return "", false
	}
	value, exists := sr.Filters[key]
	return value, exists
}

// SetFilter sets a filter value
func (sr *SearchRequest) SetFilter(key, value string) {
	if sr.Filters == nil {
		sr.Filters = make(map[string]string)
	}
	sr.Filters[key] = value
}

// ToSearchQuery converts SearchRequest to SearchQuery
func (sr *SearchRequest) ToSearchQuery() SearchQuery {
	return SearchQuery{
		ID:          generateSearchQueryID(),
		Text:        sr.Query,
		Filters:     sr.Filters,
		Providers:   sr.GetEnabledProviders(),
		Limit:       sr.Limit,
		Offset:      sr.Offset,
		SortBy:      sr.SortBy,
		SortOrder:   sr.SortOrder,
		RequestedAt: time.Now(),
	}
}

// generateSearchQueryID generates a unique search query ID
func generateSearchQueryID() string {
	return "sq_" + time.Now().Format("20060102150405") + "_" + generateRandomString(8)
}

// SearchHistory represents a stored search query
type SearchHistory struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(50)"`
	Query       string    `json:"query" gorm:"type:text;not null"`
	UserID      *string   `json:"user_id,omitempty" gorm:"type:varchar(50);index"`
	ResultCount int       `json:"result_count" gorm:"default:0"`
	Duration    int64     `json:"duration"` // milliseconds
	Providers   []string  `json:"providers" gorm:"serializer:json"`
	Filters     string    `json:"filters" gorm:"type:text"` // SQLite compatible - no jsonb
	RequestedAt time.Time `json:"requested_at" gorm:"index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns the table name for GORM
func (SearchHistory) TableName() string {
	return "search_history"
}

// SearchCache represents cached search results
type SearchCache struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(50)"`
	QueryHash   string    `json:"query_hash" gorm:"type:varchar(64);uniqueIndex"`
	Query       string    `json:"query" gorm:"type:text;not null"`
	Results     string    `json:"results" gorm:"type:text;not null"` // SQLite compatible - no jsonb
	ResultCount int       `json:"result_count" gorm:"default:0;index"`
	Provider    string    `json:"provider" gorm:"type:varchar(50);index"`
	ExpiresAt   time.Time `json:"expires_at" gorm:"index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;index"`
	AccessCount int       `json:"access_count" gorm:"default:0"`
	LastAccess  time.Time `json:"last_access" gorm:"autoUpdateTime"`
}

// TableName returns the table name for GORM
func (SearchCache) TableName() string {
	return "search_cache"
}

// IsExpired returns true if the cache entry is expired
func (sc *SearchCache) IsExpired() bool {
	return time.Now().After(sc.ExpiresAt)
}

// IncrementAccess increments the access count
func (sc *SearchCache) IncrementAccess() {
	sc.AccessCount++
	sc.LastAccess = time.Now()
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
