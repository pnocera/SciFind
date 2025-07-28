package exa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"scifind-backend/internal/errors"
	"scifind-backend/internal/models"
	"scifind-backend/internal/providers"
)

const (
	defaultBaseURL = "https://api.exa.ai"
	providerName   = "exa"
	maxResults     = 1000 // Exa API limit
)

// Provider implements the Exa neural search provider
type Provider struct {
	config     providers.ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	metrics    *providers.ProviderMetrics
	enabled    bool
}

// NewProvider creates a new Exa provider
func NewProvider(config providers.ProviderConfig, logger *slog.Logger) *Provider {
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	if config.BaseURL == "" {
		config.BaseURL = defaultBaseURL
	}

	return &Provider{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
		metrics:    &providers.ProviderMetrics{},
		enabled:    config.Enabled,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return providerName
}

// IsEnabled returns whether the provider is enabled
func (p *Provider) IsEnabled() bool {
	return p.enabled
}

// GetCapabilities returns provider capabilities
func (p *Provider) GetCapabilities() providers.ProviderCapabilities {
	return providers.ProviderCapabilities{
		SupportsFullText:       true,
		SupportsDateFilter:     true,
		SupportsAuthFilter:     false, // Exa doesn't support direct author filtering
		SupportsCategoryFilter: true,
		SupportsSort:           true,

		SupportedFields:    []string{"title", "text", "summary", "url", "published_date"},
		SupportedLanguages: []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh"},
		SupportedFormats:   []string{"json"},

		MaxResults:     maxResults,
		MaxQueryLength: 2000,
		RateLimit:      1000, // 1000 requests per month for free tier

		SupportsRealtime:    true,
		SupportsExactMatch:  false, // Exa is semantic/neural search
		SupportsFuzzySearch: true,
		SupportsWildcards:   false,
	}
}

// Search performs a search using the Exa API
func (p *Provider) Search(ctx context.Context, query *providers.SearchQuery) (*providers.SearchResult, error) {
	start := time.Now()

	// Build search request
	searchReq := BuildExaSearchRequest(query.Query, query.Filters, query.Limit, query.Offset)

	// Make API request
	response, err := p.makeSearchRequest(ctx, searchReq)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("Exa API search failed: %w", err)
	}

	// Parse response
	papers, err := p.parseSearchResponse(response)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to parse Exa response: %w", err)
	}

	duration := time.Since(start)
	p.updateMetrics(true, duration, nil)

	result := &providers.SearchResult{
		Papers:      papers,
		TotalCount:  len(papers), // Exa doesn't provide total count
		ResultCount: len(papers),
		Query:       query.Query,
		Provider:    providerName,
		Duration:    duration,
		CacheHit:    false,
		RequestID:   query.RequestID,
		Timestamp:   time.Now(),
		Success:     true,
		HasMore:     len(papers) == query.Limit, // Estimate if there are more results
	}

	p.logger.Debug("Exa search completed",
		slog.String("query", query.Query),
		slog.Int("results", len(papers)),
		slog.Duration("duration", duration))

	return result, nil
}

// GetPaper retrieves a specific paper by ID (URL in Exa's case)
func (p *Provider) GetPaper(ctx context.Context, id string) (*models.Paper, error) {
	start := time.Now()

	// Exa uses URLs as IDs, so we can request content directly
	contentsReq := &ExaContentsRequest{
		IDs:        []string{id},
		Text:       true,
		Summary:    true,
		Highlights: true,
		LiveCrawl:  false,
	}

	response, err := p.makeContentsRequest(ctx, contentsReq)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("Exa API contents request failed: %w", err)
	}

	if len(response.Results) == 0 {
		return nil, errors.NewNotFoundError("paper", id)
	}

	paper, err := p.convertContentToPaper(response.Results[0])
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to convert content: %w", err)
	}

	p.updateMetrics(true, time.Since(start), nil)
	return paper, nil
}

// HealthCheck checks if the Exa API is accessible
func (p *Provider) HealthCheck(ctx context.Context) error {
	start := time.Now()

	// Make a simple test search
	testReq := &ExaSearchRequest{
		Query:      "test",
		Type:       SearchTypeNeural,
		NumResults: 1,
	}

	_, err := p.makeSearchRequest(ctx, testReq)
	if err != nil {
		return errors.NewHealthCheckError("Health check failed: " + err.Error(), "exa")
	}

	p.logger.Debug("Exa health check passed", slog.Duration("duration", time.Since(start)))
	return nil
}

// GetStatus returns the current provider status
func (p *Provider) GetStatus() providers.ProviderStatus {
	return providers.ProviderStatus{
		Name:            providerName,
		Enabled:         p.enabled,
		Healthy:         true,
		LastCheck:       time.Now(),
		CircuitState:    "closed",
		RateLimited:     false,
		AvgResponseTime: p.calculateAvgResponseTime(),
		SuccessRate:     p.calculateSuccessRate(),
		APIVersion:      "v1",
		LastUpdated:     time.Now(),
	}
}

// GetMetrics returns provider metrics
func (p *Provider) GetMetrics() providers.ProviderMetrics {
	return *p.metrics
}

// Configure updates the provider configuration
func (p *Provider) Configure(config providers.ProviderConfig) error {
	if err := p.ValidateConfig(config); err != nil {
		return err
	}

	p.config = config
	p.enabled = config.Enabled

	// Update HTTP client timeout
	p.httpClient.Timeout = config.Timeout

	p.logger.Info("Exa provider configured",
		slog.Bool("enabled", config.Enabled),
		slog.Duration("timeout", config.Timeout))

	return nil
}

// ValidateConfig validates the provider configuration
func (p *Provider) ValidateConfig(config providers.ProviderConfig) error {
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative")
	}

	if config.APIKey == "" {
		return fmt.Errorf("api_key is required for Exa provider")
	}

	return nil
}

// makeSearchRequest makes a search request to Exa API
func (p *Provider) makeSearchRequest(ctx context.Context, searchReq *ExaSearchRequest) (*ExaSearchResponse, error) {
	url := p.config.BaseURL + "/search"

	body, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SciFIND-Backend/1.0")
	req.Header.Set("x-api-key", p.config.APIKey)

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleAPIError(resp)
	}

	// Read and parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response ExaSearchResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// makeContentsRequest makes a contents request to Exa API
func (p *Provider) makeContentsRequest(ctx context.Context, contentsReq *ExaContentsRequest) (*ExaContentsResponse, error) {
	url := p.config.BaseURL + "/contents"

	body, err := json.Marshal(contentsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal contents request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SciFIND-Backend/1.0")
	req.Header.Set("x-api-key", p.config.APIKey)

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleAPIError(resp)
	}

	// Read and parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response ExaContentsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// handleAPIError handles API error responses
func (p *Provider) handleAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Exa API returned status %d", resp.StatusCode)
	}

	var exaError ExaError
	if err := json.Unmarshal(body, &exaError); err != nil {
		return fmt.Errorf("Exa API returned status %d: %s", resp.StatusCode, string(body))
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return errors.NewRateLimitError(exaError.Message, time.Minute)
	case http.StatusUnauthorized:
		return errors.NewAuthenticationError(exaError.Message)
	case http.StatusBadRequest:
		return errors.NewValidationError(exaError.Message, "query", "unknown")
	default:
		return fmt.Errorf("Exa API error (%d): %s", resp.StatusCode, exaError.Message)
	}
}

// parseSearchResponse converts Exa search response to papers
func (p *Provider) parseSearchResponse(response *ExaSearchResponse) ([]models.Paper, error) {
	papers := make([]models.Paper, 0, len(response.Results))

	for _, result := range response.Results {
		paper, err := p.convertResultToPaper(result)
		if err != nil {
			p.logger.Warn("Failed to convert Exa result to paper",
				slog.String("url", result.URL),
				slog.String("error", err.Error()))
			continue
		}
		papers = append(papers, *paper)
	}

	return papers, nil
}

// convertResultToPaper converts an Exa search result to our Paper model
func (p *Provider) convertResultToPaper(result ExaResult) (*models.Paper, error) {
	if result.URL == "" {
		return nil, fmt.Errorf("result URL is required")
	}

	// Parse published date
	var publishedAt *time.Time
	if result.PublishedDate != nil && *result.PublishedDate != "" {
		if parsed, err := time.Parse("2006-01-02", *result.PublishedDate); err == nil {
			publishedAt = &parsed
		}
	}

	// Create author if available
	var authors []models.Author
	if result.Author != nil && *result.Author != "" {
		authors = append(authors, models.Author{
			Name: *result.Author,
		})
	}

	// Use text or summary as abstract
	var abstract *string
	if result.Summary != nil && *result.Summary != "" {
		abstract = result.Summary
	} else if result.Text != nil && *result.Text != "" {
		// Truncate text to reasonable abstract length
		text := *result.Text
		if len(text) > 500 {
			text = text[:500] + "..."
		}
		abstract = &text
	}

	// Build paper
	paper := &models.Paper{
		ID:              "exa_" + result.ID,
		Title:           result.Title,
		Abstract:        abstract,
		Authors:         authors,
		PublishedAt:     publishedAt,
		URL:             &result.URL,
		Language:        "en", // Assume English for now
		SourceProvider:  providerName,
		SourceID:        result.ID,
		SourceURL:       &result.URL,
		ProcessingState: "completed",
		QualityScore:    result.Score, // Use Exa's relevance score as quality score
	}

	return paper, nil
}

// convertContentToPaper converts Exa content to our Paper model
func (p *Provider) convertContentToPaper(content ExaContentResult) (*models.Paper, error) {
	if content.URL == "" {
		return nil, fmt.Errorf("content URL is required")
	}

	// Use summary or text as abstract
	var abstract *string
	if content.Summary != nil && *content.Summary != "" {
		abstract = content.Summary
	} else if content.Text != nil && *content.Text != "" {
		// Truncate text to reasonable abstract length
		text := *content.Text
		if len(text) > 1000 {
			text = text[:1000] + "..."
		}
		abstract = &text
	}

	// Build paper
	paper := &models.Paper{
		ID:              "exa_" + content.ID,
		Title:           content.Title,
		Abstract:        abstract,
		URL:             &content.URL,
		Language:        "en",
		SourceProvider:  providerName,
		SourceID:        content.ID,
		SourceURL:       &content.URL,
		ProcessingState: "completed",
	}

	// Calculate quality score
	paper.UpdateQualityScore()

	return paper, nil
}

// Helper methods
func (p *Provider) updateMetrics(success bool, duration time.Duration, err error) {
	p.metrics.TotalRequests++

	if success {
		p.metrics.SuccessfulRequests++
	} else {
		p.metrics.FailedRequests++

		// Categorize errors
		if err != nil {
			switch {
			case errors.IsTimeoutError(err):
				p.metrics.TimeoutErrors++
			case errors.IsRateLimitError(err):
				p.metrics.RateLimitErrors++
			case errors.IsNetworkError(err):
				p.metrics.NetworkErrors++
			default:
				p.metrics.ParseErrors++
			}
		}
	}

	// Update response time statistics
	if p.metrics.MinResponseTime == 0 || duration < p.metrics.MinResponseTime {
		p.metrics.MinResponseTime = duration
	}
	if duration > p.metrics.MaxResponseTime {
		p.metrics.MaxResponseTime = duration
	}

	// Simple moving average for response time
	if p.metrics.AvgResponseTime == 0 {
		p.metrics.AvgResponseTime = duration
	} else {
		p.metrics.AvgResponseTime = (p.metrics.AvgResponseTime + duration) / 2
	}
}

func (p *Provider) calculateAvgResponseTime() time.Duration {
	return p.metrics.AvgResponseTime
}

func (p *Provider) calculateSuccessRate() float64 {
	if p.metrics.TotalRequests == 0 {
		return 1.0
	}
	return float64(p.metrics.SuccessfulRequests) / float64(p.metrics.TotalRequests)
}
