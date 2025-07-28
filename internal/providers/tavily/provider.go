package tavily

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
	defaultBaseURL = "https://api.tavily.com"
	providerName   = "tavily"
	maxResults     = 100 // Tavily API limit per search
)

// Provider implements the Tavily web search provider
type Provider struct {
	config     providers.ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	metrics    *providers.ProviderMetrics
	enabled    bool
}

// NewProvider creates a new Tavily provider
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
		SupportsDateFilter:     false, // Tavily doesn't support direct date filtering
		SupportsAuthFilter:     false, // Tavily doesn't support direct author filtering
		SupportsCategoryFilter: false, // Tavily doesn't support category filtering
		SupportsSort:          true,

		SupportedFields:    []string{"title", "content", "url"},
		SupportedLanguages: []string{"en", "es", "fr", "de", "it", "pt", "ru", "ja", "ko", "zh", "ar", "hi"},
		SupportedFormats:   []string{"json", "markdown"},

		MaxResults:         maxResults,
		MaxQueryLength:     1000,
		RateLimit:         1000, // 1000 requests per month for free tier

		SupportsRealtime:    true,
		SupportsExactMatch:  true,
		SupportsFuzzySearch: true,
		SupportsWildcards:   false,
	}
}

// Search performs a search using the Tavily API
func (p *Provider) Search(ctx context.Context, query *providers.SearchQuery) (*providers.SearchResult, error) {
	start := time.Now()

	// Build search request
	searchReq := BuildTavilySearchRequest(query.Query, query.Filters, query.Limit, query.Offset)

	// Make API request
	response, err := p.makeSearchRequest(ctx, searchReq)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("Tavily API search failed: %w", err)
	}

	// Parse response
	papers, err := p.parseSearchResponse(response)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to parse Tavily response: %w", err)
	}

	duration := time.Since(start)
	p.updateMetrics(true, duration, nil)

	result := &providers.SearchResult{
		Papers:      papers,
		TotalCount:  len(papers), // Tavily doesn't provide total count
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

	p.logger.Debug("Tavily search completed",
		slog.String("query", query.Query),
		slog.Int("results", len(papers)),
		slog.Duration("duration", duration))

	return result, nil
}

// GetPaper retrieves a specific paper by ID (URL in Tavily's case)
func (p *Provider) GetPaper(ctx context.Context, id string) (*models.Paper, error) {
	start := time.Now()

	// Tavily uses URLs as IDs, so we can extract content directly
	extractReq := &TavilyExtractRequest{
		URLs: []string{id},
	}

	response, err := p.makeExtractRequest(ctx, extractReq)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("Tavily API extract request failed: %w", err)
	}

	if len(response.Results) == 0 {
		return nil, errors.NewNotFoundError("paper", id)
	}

	paper, err := p.convertExtractToPaper(response.Results[0])
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to convert extracted content: %w", err)
	}

	p.updateMetrics(true, time.Since(start), nil)
	return paper, nil
}

// HealthCheck checks if the Tavily API is accessible
func (p *Provider) HealthCheck(ctx context.Context) error {
	start := time.Now()

	// Make a simple test search
	testReq := &TavilySearchRequest{
		Query:      "test",
		SearchDepth: SearchDepthBasic,
		MaxResults: 1,
	}

	_, err := p.makeSearchRequest(ctx, testReq)
	if err != nil {
		return errors.NewHealthCheckError("Health check failed: " + err.Error(), "tavily")
	}

	p.logger.Debug("Tavily health check passed", slog.Duration("duration", time.Since(start)))
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

	p.logger.Info("Tavily provider configured",
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
		return fmt.Errorf("api_key is required for Tavily provider")
	}

	return nil
}

// makeSearchRequest makes a search request to Tavily API
func (p *Provider) makeSearchRequest(ctx context.Context, searchReq *TavilySearchRequest) (*TavilySearchResponse, error) {
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
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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

	var response TavilySearchResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// makeExtractRequest makes an extract request to Tavily API
func (p *Provider) makeExtractRequest(ctx context.Context, extractReq *TavilyExtractRequest) (*TavilyExtractResponse, error) {
	url := p.config.BaseURL + "/extract"

	body, err := json.Marshal(extractReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal extract request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SciFIND-Backend/1.0")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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

	var response TavilyExtractResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// handleAPIError handles API error responses
func (p *Provider) handleAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Tavily API returned status %d", resp.StatusCode)
	}

	var tavilyError TavilyError
	if err := json.Unmarshal(body, &tavilyError); err != nil {
		return fmt.Errorf("Tavily API returned status %d: %s", resp.StatusCode, string(body))
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return errors.NewRateLimitError(tavilyError.Message, time.Minute)
	case http.StatusUnauthorized:
		return errors.NewAuthenticationError(tavilyError.Message)
	case http.StatusBadRequest:
		return errors.NewValidationError(tavilyError.Message, "query", "unknown")
	default:
		return fmt.Errorf("Tavily API error (%d): %s", resp.StatusCode, tavilyError.Message)
	}
}

// parseSearchResponse converts Tavily search response to papers
func (p *Provider) parseSearchResponse(response *TavilySearchResponse) ([]models.Paper, error) {
	papers := make([]models.Paper, 0, len(response.Results))

	for _, result := range response.Results {
		paper, err := p.convertResultToPaper(result)
		if err != nil {
			p.logger.Warn("Failed to convert Tavily result to paper",
				slog.String("url", result.URL),
				slog.String("error", err.Error()))
			continue
		}
		papers = append(papers, *paper)
	}

	return papers, nil
}

// convertResultToPaper converts a Tavily search result to our Paper model
func (p *Provider) convertResultToPaper(result TavilyResult) (*models.Paper, error) {
	if result.URL == "" {
		return nil, fmt.Errorf("result URL is required")
	}

	// Extract metadata from content
	metadata := ExtractMetadata(result.Content)

	// Parse published date if available
	var publishedAt *time.Time
	if dateStr, exists := metadata["published_date"]; exists {
		if parsed, err := ParsePublishedDate(dateStr); err == nil {
			publishedAt = parsed
		}
	}

	// Create author if available
	var authors []models.Author
	if authorStr, exists := metadata["author"]; exists {
		authors = append(authors, models.Author{
			Name: authorStr,
		})
	}

	// Use content as abstract, but limit length
	abstract := result.Content
	if len(abstract) > 1000 {
		abstract = abstract[:1000] + "..."
	}

	// Extract DOI if available
	var doi *string
	if doiStr, exists := metadata["doi"]; exists {
		doi = &doiStr
	}

	// Extract journal if available
	var journal *string
	if journalStr, exists := metadata["journal"]; exists {
		journal = &journalStr
	}

	// Build paper
	paper := &models.Paper{
		ID:              "tavily_" + fmt.Sprintf("%x", result.URL), // Use URL hash as ID
		DOI:             doi,
		Title:           result.Title,
		Abstract:        &abstract,
		Authors:         authors,
		Journal:         journal,
		PublishedAt:     publishedAt,
		URL:             &result.URL,
		Language:        "en", // Assume English for now
		SourceProvider:  providerName,
		SourceID:        result.URL,
		SourceURL:       &result.URL,
		ProcessingState: "completed",
		QualityScore:    EstimateQualityScore(result),
	}

	return paper, nil
}

// convertExtractToPaper converts Tavily extracted content to our Paper model
func (p *Provider) convertExtractToPaper(extract TavilyExtractResult) (*models.Paper, error) {
	if extract.URL == "" {
		return nil, fmt.Errorf("extract URL is required")
	}

	// Extract metadata from raw content
	metadata := ExtractMetadata(extract.RawContent)

	// Parse published date if available
	var publishedAt *time.Time
	if dateStr, exists := metadata["published_date"]; exists {
		if parsed, err := ParsePublishedDate(dateStr); err == nil {
			publishedAt = parsed
		}
	}

	// Create author if available
	var authors []models.Author
	if authorStr, exists := metadata["author"]; exists {
		authors = append(authors, models.Author{
			Name: authorStr,
		})
	}

	// Use raw content as abstract, but limit length
	abstract := extract.RawContent
	if len(abstract) > 1500 {
		abstract = abstract[:1500] + "..."
	}

	// Extract DOI if available
	var doi *string
	if doiStr, exists := metadata["doi"]; exists {
		doi = &doiStr
	}

	// Extract journal if available
	var journal *string
	if journalStr, exists := metadata["journal"]; exists {
		journal = &journalStr
	}

	// Build paper
	paper := &models.Paper{
		ID:              "tavily_" + fmt.Sprintf("%x", extract.URL), // Use URL hash as ID
		DOI:             doi,
		Title:           "Extracted Content", // Default title if not found
		Abstract:        &abstract,
		Authors:         authors,
		Journal:         journal,
		PublishedAt:     publishedAt,
		URL:             &extract.URL,
		Language:        "en",
		SourceProvider:  providerName,
		SourceID:        extract.URL,
		SourceURL:       &extract.URL,
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