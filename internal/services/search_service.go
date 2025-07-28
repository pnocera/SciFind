package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"scifind-backend/internal/messaging"
	"scifind-backend/internal/models"
	"scifind-backend/internal/providers"
	"scifind-backend/internal/providers/arxiv"
	"scifind-backend/internal/providers/exa"
	"scifind-backend/internal/providers/semantic_scholar"
	"scifind-backend/internal/providers/tavily"
	"scifind-backend/internal/repository"
)

// SearchService handles search-related business logic
type SearchService struct {
	searchRepo      repository.SearchRepository
	paperRepo       repository.PaperRepository
	providerManager providers.ProviderManager
	messaging       *messaging.Client
	logger          *slog.Logger
}

// NewSearchService creates a new search service
func NewSearchService(
	searchRepo repository.SearchRepository,
	paperRepo repository.PaperRepository,
	messaging *messaging.Client,
	logger *slog.Logger,
) SearchServiceInterface {
	service := &SearchService{
		searchRepo: searchRepo,
		paperRepo:  paperRepo,
		messaging:  messaging,
		logger:     logger,
	}

	// Initialize provider manager
	service.initializeProviders()

	return service
}

// initializeProviders sets up all search providers
func (s *SearchService) initializeProviders() {
	// Create provider manager
	managerConfig := providers.ManagerConfig{
		AggregationStrategy: providers.StrategyMerge,
		MaxConcurrency:     5,
		Timeout:           30 * time.Second,
	}
	s.providerManager = providers.NewManager(s.logger, managerConfig)

	// Initialize ArXiv provider
	arxivConfig := providers.ProviderConfig{
		Enabled:    true,
		BaseURL:    "http://export.arxiv.org/api",
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}
	arxivProvider := arxiv.NewProvider(arxivConfig, s.logger)
	s.providerManager.RegisterProvider("arxiv", arxivProvider)

	// Initialize Semantic Scholar provider
	ssConfig := providers.ProviderConfig{
		Enabled:    true,
		BaseURL:    "https://api.semanticscholar.org/graph/v1",
		Timeout:    15 * time.Second,
		MaxRetries: 3,
		APIKey:     "", // Optional for basic usage
	}
	ssProvider := semantic_scholar.NewProvider(ssConfig, s.logger)
	s.providerManager.RegisterProvider("semantic_scholar", ssProvider)

	// Initialize Exa provider (requires API key)
	exaConfig := providers.ProviderConfig{
		Enabled:    false, // Disabled by default, enable when API key is available
		BaseURL:    "https://api.exa.ai",
		Timeout:    20 * time.Second,
		MaxRetries: 3,
		APIKey:     "", // Must be configured
	}
	exaProvider := exa.NewProvider(exaConfig, s.logger)
	s.providerManager.RegisterProvider("exa", exaProvider)

	// Initialize Tavily provider (requires API key)
	tavilyConfig := providers.ProviderConfig{
		Enabled:    false, // Disabled by default, enable when API key is available
		BaseURL:    "https://api.tavily.com",
		Timeout:    25 * time.Second,
		MaxRetries: 3,
		APIKey:     "", // Must be configured
	}
	tavilyProvider := tavily.NewProvider(tavilyConfig, s.logger)
	s.providerManager.RegisterProvider("tavily", tavilyProvider)

	s.logger.Info("Search providers initialized",
		slog.Int("total_providers", len(s.providerManager.GetAllProviders())),
		slog.Int("enabled_providers", len(s.providerManager.GetEnabledProviders())))
}

// Search performs a search across configured providers
func (s *SearchService) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	start := time.Now()

	// Validate request
	if err := s.validateSearchRequest(req); err != nil {
		return nil, fmt.Errorf("invalid search request: %v", err)
	}

	// Build provider search query
	searchQuery := &providers.SearchQuery{
		RequestID: req.RequestID,
		Query:     req.Query,
		Limit:     req.Limit,
		Offset:    req.Offset,
		Filters:   req.Filters,
		DateFrom:  req.DateFrom,
		DateTo:    req.DateTo,
	}

	// Publish search request event
	if err := s.publishSearchRequestEvent(ctx, req); err != nil {
		s.logger.Warn("Failed to publish search request event", slog.String("error", err.Error()))
	}

	// Execute search
	var result *providers.AggregatedResult
	var err error

	if len(req.Providers) > 0 {
		// Search specific providers
		result, err = s.providerManager.SearchProviders(ctx, searchQuery, req.Providers)
	} else {
		// Search all enabled providers
		result, err = s.providerManager.SearchAll(ctx, searchQuery)
	}

	if err != nil {
		// Publish search failure event
		s.publishSearchCompletedEvent(ctx, req, nil, time.Since(start), err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Process and enhance results
	enhancedPapers, err := s.enhanceSearchResults(ctx, result.Papers)
	if err != nil {
		s.logger.Warn("Failed to enhance search results", slog.String("error", err.Error()))
		// Continue with unenhanced results
		enhancedPapers = result.Papers
	}

	// Build response
	response := &SearchResponse{
		RequestID:            req.RequestID,
		Query:                req.Query,
		Papers:               enhancedPapers,
		TotalCount:          result.TotalCount,
		ResultCount:         len(enhancedPapers),
		ProvidersUsed:       result.SuccessfulProviders,
		ProvidersFailed:     result.FailedProviders,
		Duration:            result.TotalDuration,
		AggregationStrategy: result.AggregationStrategy,
		CacheHits:           result.CacheHits,
		PartialFailure:      result.PartialFailure,
		Errors:              result.Errors,
		Timestamp:           time.Now(),
	}

	// Store search in database for analytics
	if err := s.storeSearchResult(ctx, req, response); err != nil {
		s.logger.Warn("Failed to store search result", slog.String("error", err.Error()))
	}

	// Publish search completion event
	if err := s.publishSearchCompletedEvent(ctx, req, response, time.Since(start), nil); err != nil {
		s.logger.Warn("Failed to publish search completed event", slog.String("error", err.Error()))
	}

	s.logger.Info("Search completed",
		slog.String("query", req.Query),
		slog.Int("results", len(enhancedPapers)),
		slog.Duration("duration", time.Since(start)),
		slog.Any("providers_used", result.SuccessfulProviders))

	return response, nil
}

// GetPaper retrieves a specific paper by ID from a provider
func (s *SearchService) GetPaper(ctx context.Context, providerName, paperID string) (*models.Paper, error) {
	provider, err := s.providerManager.GetProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	paper, err := provider.GetPaper(ctx, paperID)
	if err != nil {
		return nil, fmt.Errorf("failed to get paper: %w", err)
	}

	// Store paper if not already in database
	if err := s.paperRepo.Create(ctx, paper); err != nil {
		// Ignore duplicate errors (simplified check)
		if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "UNIQUE constraint") {
			s.logger.Warn("Failed to store paper", slog.String("error", err.Error()))
		}
	}

	return paper, nil
}

// GetProviderStatus returns the status of all providers
func (s *SearchService) GetProviderStatus(ctx context.Context) (map[string]interface{}, error) {
	providers := s.providerManager.GetAllProviders()
	status := make(map[string]interface{})

	for name, provider := range providers {
		status[name] = provider.GetStatus()
	}

	return status, nil
}

// GetProviderMetrics returns metrics for all providers
func (s *SearchService) GetProviderMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := s.providerManager.GetProviderMetrics()
	result := make(map[string]interface{})
	for name, metric := range metrics {
		result[name] = metric
	}
	return result, nil
}

// ConfigureProvider updates a provider's configuration
func (s *SearchService) ConfigureProvider(ctx context.Context, name string, config interface{}) error {
	// For now, just log the configuration attempt
	s.logger.Info("Provider configuration requested", slog.String("provider", name), slog.Any("config", config))
	return nil
}

// Health checks the health of the search service and all providers
func (s *SearchService) Health(ctx context.Context) error {
	healthResults := s.providerManager.HealthCheckAll(ctx)
	
	var failedProviders []string
	for name, err := range healthResults {
		if err != nil {
			failedProviders = append(failedProviders, name)
			s.logger.Warn("Provider health check failed",
				slog.String("provider", name),
				slog.String("error", err.Error()))
		}
	}

	if len(failedProviders) > 0 {
		return fmt.Errorf("provider health checks failed: %s", strings.Join(failedProviders, ", "))
	}

	return nil
}

// Helper methods

func (s *SearchService) validateSearchRequest(req *SearchRequest) error {
	if req.Query == "" {
		return fmt.Errorf("query is required")
	}

	if req.Limit <= 0 {
		req.Limit = 20 // Default limit
	}

	if req.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	if req.Offset < 0 {
		req.Offset = 0
	}

	return nil
}

func (s *SearchService) enhanceSearchResults(ctx context.Context, papers []models.Paper) ([]models.Paper, error) {
	// For now, just return papers as-is
	// Future enhancements could include:
	// - Deduplication across providers
	// - Quality scoring
	// - Content enrichment
	// - Citation analysis
	return papers, nil
}

func (s *SearchService) storeSearchResult(ctx context.Context, req *SearchRequest, resp *SearchResponse) error {
	// TODO: Implement search result storage for analytics
	// This would store search queries, results, and metadata for analysis
	return nil
}

func (s *SearchService) publishSearchRequestEvent(ctx context.Context, req *SearchRequest) error {
	event := messaging.NewSearchRequestEvent(
		req.RequestID,
		req.Query,
		req.Providers,
		req.UserID,
	)

	return s.messaging.Publish(ctx, messaging.SubjectSearchRequest, event)
}

func (s *SearchService) publishSearchCompletedEvent(ctx context.Context, req *SearchRequest, resp *SearchResponse, duration time.Duration, err error) error {
	var resultCount int
	var providersUsed []string
	var success bool
	var errorMsg string

	if resp != nil {
		resultCount = resp.ResultCount
		providersUsed = resp.ProvidersUsed
		success = true
	} else {
		success = false
		if err != nil {
			errorMsg = err.Error()
		}
	}

	event := &messaging.SearchCompletedEvent{
		RequestID:     req.RequestID,
		UserID:        req.UserID,
		Query:         req.Query,
		ResultCount:   resultCount,
		Duration:      duration.Milliseconds(),
		ProvidersUsed: providersUsed,
		CacheHit:      false, // TODO: Implement cache detection
		CompletedAt:   time.Now().UnixMilli(),
		Success:       success,
		Error:         errorMsg,
	}

	return s.messaging.Publish(ctx, messaging.SubjectSearchCompleted, event)
}