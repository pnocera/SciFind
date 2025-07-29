package semantic_scholar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"scifind-backend/internal/errors"
	"scifind-backend/internal/models"
	"scifind-backend/internal/providers"
)

const (
	defaultBaseURL = "https://api.semanticscholar.org/graph/v1"
	providerName   = "semantic_scholar"
	maxResults     = 10000 // Semantic Scholar API limit
)

// Provider implements the Semantic Scholar search provider
type Provider struct {
	config     providers.ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	metrics    *providers.ProviderMetrics
	enabled    bool
}

// NewProvider creates a new Semantic Scholar provider
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
		SupportsAuthFilter:     true,
		SupportsCategoryFilter: true,
		SupportsSort:           true,

		SupportedFields:    []string{"title", "abstract", "author", "venue", "year", "fieldsOfStudy"},
		SupportedLanguages: []string{"en"},
		SupportedFormats:   []string{"json"},

		MaxResults:     maxResults,
		MaxQueryLength: 1000,
		RateLimit:      100, // 100 requests per second with API key

		SupportsRealtime:    true,
		SupportsExactMatch:  true,
		SupportsFuzzySearch: true,
		SupportsWildcards:   false,
	}
}

// Search performs a search using the Semantic Scholar API
func (p *Provider) Search(ctx context.Context, query *providers.SearchQuery) (*providers.SearchResult, error) {
	start := time.Now()

	// Build request URL
	reqURL, err := p.buildURL("/paper/search", query)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Make HTTP request
	response, err := p.makeRequest(ctx, reqURL)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("Semantic Scholar API request failed: %w", err)
	}

	// Parse response
	papers, totalCount, err := p.parseSearchResponse(response)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to parse Semantic Scholar response: %w", err)
	}

	duration := time.Since(start)
	p.updateMetrics(true, duration, nil)

	result := &providers.SearchResult{
		Papers:      papers,
		TotalCount:  totalCount,
		ResultCount: len(papers),
		Query:       query.Query,
		Provider:    providerName,
		Duration:    duration,
		CacheHit:    false,
		RequestID:   query.RequestID,
		Timestamp:   time.Now(),
		Success:     true,
		HasMore:     query.Offset+len(papers) < totalCount,
	}

	p.logger.Debug("Semantic Scholar search completed",
		slog.String("query", query.Query),
		slog.Int("results", len(papers)),
		slog.Int("total", totalCount),
		slog.Duration("duration", duration))

	return result, nil
}

// GetPaper retrieves a specific paper by ID
func (p *Provider) GetPaper(ctx context.Context, id string) (*models.Paper, error) {
	start := time.Now()

	// Build URL for specific paper
	reqURL := fmt.Sprintf("%s/paper/%s?fields=paperId,externalIds,title,abstract,authors,venue,year,citationCount,referenceCount,fieldsOfStudy,url,openAccessPdf", p.config.BaseURL, id)

	response, err := p.makeRequest(ctx, reqURL)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("Semantic Scholar API request failed: %w", err)
	}

	var paperData SemanticScholarPaper
	if err := json.Unmarshal(response, &paperData); err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to parse paper response: %w", err)
	}

	paper, err := p.convertPaper(paperData)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to convert paper: %w", err)
	}

	p.updateMetrics(true, time.Since(start), nil)
	return paper, nil
}

// HealthCheck checks if the Semantic Scholar API is accessible
func (p *Provider) HealthCheck(ctx context.Context) error {
	start := time.Now()

	// Make a simple test query
	testURL := fmt.Sprintf("%s/paper/search?query=test&limit=1", p.config.BaseURL)
	_, err := p.makeRequest(ctx, testURL)
	if err != nil {
		return errors.NewHealthCheckError("Health check failed: "+err.Error(), "semantic_scholar")
	}

	p.logger.Debug("Semantic Scholar health check passed", slog.Duration("duration", time.Since(start)))
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

	p.logger.Info("Semantic Scholar provider configured",
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

	return nil
}

// buildURL builds the request URL with query parameters
func (p *Provider) buildURL(endpoint string, query *providers.SearchQuery) (string, error) {
	baseURL := p.config.BaseURL + endpoint

	params := url.Values{}
	params.Set("query", query.Query)
	params.Set("limit", strconv.Itoa(query.Limit))
	params.Set("offset", strconv.Itoa(query.Offset))

	// Add fields to retrieve
	fields := []string{
		"paperId", "externalIds", "title", "abstract", "authors",
		"venue", "year", "citationCount", "referenceCount",
		"fieldsOfStudy", "url", "openAccessPdf",
	}
	params.Set("fields", strings.Join(fields, ","))

	// Add filters
	if authors, ok := query.Filters[providers.FilterAuthor]; ok {
		// Semantic Scholar doesn't support direct author filtering in the query
		// We'll filter after getting results
		_ = authors // Will be used for post-processing filtering
	}

	if journal, ok := query.Filters[providers.FilterJournal]; ok {
		// Add venue filter
		currentQuery := params.Get("query")
		params.Set("query", fmt.Sprintf("%s venue:%s", currentQuery, journal))
	}

	// Date filters
	if query.DateFrom != nil {
		currentQuery := params.Get("query")
		params.Set("query", fmt.Sprintf("%s year>=%d", currentQuery, query.DateFrom.Year()))
	}

	if query.DateTo != nil {
		currentQuery := params.Get("query")
		params.Set("query", fmt.Sprintf("%s year<=%d", currentQuery, query.DateTo.Year()))
	}

	return baseURL + "?" + params.Encode(), nil
}

// makeRequest makes an HTTP request to the Semantic Scholar API
func (p *Provider) makeRequest(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "SciFIND-Backend/1.0")
	if p.config.APIKey != "" {
		req.Header.Set("x-api-key", p.config.APIKey)
	}

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle different HTTP status codes
	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusTooManyRequests:
		// Rate limiting
		retryAfter := resp.Header.Get("Retry-After")
		var retryDuration time.Duration
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				retryDuration = time.Duration(seconds) * time.Second
			}
		}
		return nil, errors.NewRateLimitError(
			fmt.Sprintf("Semantic Scholar API rate limit exceeded. Retry after: %s", retryAfter),
			retryDuration,
		)
	case http.StatusUnauthorized:
		return nil, errors.NewAuthenticationError(
			"Invalid API key for Semantic Scholar",
		)
	case http.StatusForbidden:
		return nil, errors.NewValidationError(
			"Access forbidden to Semantic Scholar API",
			"access",
			"forbidden",
		)
	case http.StatusNotFound:
		return nil, errors.NewNotFoundError(
			"Resource not found in Semantic Scholar",
			reqURL,
		)
	case http.StatusInternalServerError:
		return nil, errors.NewInternalError(
			"Semantic Scholar API internal server error",
			nil,
		)
	case http.StatusServiceUnavailable:
		return nil, errors.NewNetworkError(
			"Semantic Scholar API service unavailable",
			nil,
		)
	default:
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// Try to parse error response
			var apiError SemanticScholarError
			if err := json.Unmarshal(body, &apiError); err == nil && apiError.Message != "" {
				return nil, errors.NewValidationError(
					fmt.Sprintf("Semantic Scholar API error: %s", apiError.Message),
					"api",
					"validation",
				)
			}
			return nil, errors.NewValidationError(
				fmt.Sprintf("Semantic Scholar API client error: %d %s", resp.StatusCode, resp.Status),
				"status",
				resp.StatusCode,
			)
		}
		if resp.StatusCode >= 500 {
			return nil, errors.NewInternalError(
				fmt.Sprintf("Semantic Scholar API server error: %d %s", resp.StatusCode, resp.Status),
				nil,
			)
		}
		return nil, errors.NewInternalError(
			fmt.Sprintf("Semantic Scholar API returned unexpected status %d", resp.StatusCode),
			nil,
		)
	}
}

// parseSearchResponse parses the search response
func (p *Provider) parseSearchResponse(data []byte) ([]models.Paper, int, error) {
	var response SemanticScholarSearchResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	papers := make([]models.Paper, 0, len(response.Data))

	for _, paperData := range response.Data {
		paper, err := p.convertPaper(paperData)
		if err != nil {
			p.logger.Warn("Failed to convert Semantic Scholar paper",
				slog.String("id", paperData.PaperID),
				slog.String("error", err.Error()))
			continue
		}
		papers = append(papers, *paper)
	}

	// Semantic Scholar provides total count in response
	totalCount := response.Total
	if totalCount == 0 {
		totalCount = len(papers)
	}

	return papers, totalCount, nil
}

// convertPaper converts a Semantic Scholar paper to our Paper model
func (p *Provider) convertPaper(paperData SemanticScholarPaper) (*models.Paper, error) {
	if paperData.PaperID == "" {
		return nil, fmt.Errorf("paper ID is required")
	}

	// Parse published date
	var publishedAt *time.Time
	if paperData.Year > 0 {
		// Create a date with January 1st of the given year
		date := time.Date(paperData.Year, 1, 1, 0, 0, 0, 0, time.UTC)
		publishedAt = &date
	}

	// Convert authors
	authors := make([]models.Author, 0, len(paperData.Authors))
	for _, authorData := range paperData.Authors {
		author := models.Author{
			Name: authorData.Name,
		}
		authors = append(authors, author)
	}

	// Convert fields of study to categories
	categories := make([]models.Category, 0, len(paperData.FieldsOfStudy))
	for _, field := range paperData.FieldsOfStudy {
		category := models.Category{
			ID:         "ss_" + strings.ToLower(strings.ReplaceAll(field.Category, " ", "_")),
			Name:       field.Category,
			Source:     "semantic_scholar",
			SourceCode: field.Category,
			IsActive:   true,
		}
		categories = append(categories, category)
	}

	// Extract DOI from external IDs
	var doi *string
	var arxivID *string
	if paperData.ExternalIDs != nil {
		if paperData.ExternalIDs.DOI != "" {
			doi = &paperData.ExternalIDs.DOI
		}
		if paperData.ExternalIDs.ArxivID != "" {
			arxivID = &paperData.ExternalIDs.ArxivID
		}
	}

	// Get PDF URL
	var pdfURL *string
	if paperData.OpenAccessPDF != nil && paperData.OpenAccessPDF.URL != "" {
		pdfURL = &paperData.OpenAccessPDF.URL
	}

	// Get main URL
	var url *string
	if paperData.URL != "" {
		url = &paperData.URL
	}

	// Build paper
	paper := &models.Paper{
		ID:              "ss_" + paperData.PaperID,
		DOI:             doi,
		ArxivID:         arxivID,
		Title:           paperData.Title,
		Abstract:        &paperData.Abstract,
		Authors:         authors,
		Categories:      categories,
		Journal:         &paperData.Venue,
		PublishedAt:     publishedAt,
		URL:             url,
		PDFURL:          pdfURL,
		Language:        "en",
		CitationCount:   paperData.CitationCount,
		SourceProvider:  providerName,
		SourceID:        paperData.PaperID,
		SourceURL:       url,
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
