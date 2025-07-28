package providers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"scifind-backend/internal/errors"
	"scifind-backend/internal/models"
)

// Manager implements the ProviderManager interface
type Manager struct {
	providers      map[string]SearchProvider
	enabled        map[string]bool
	rateLimit      RateLimiter
	cache          CacheManager
	circuitBreaker CircuitBreaker
	logger         *slog.Logger
	mu             sync.RWMutex

	// Configuration
	aggregationStrategy AggregationStrategy
	maxConcurrency     int
	timeout            time.Duration
}

// NewManager creates a new provider manager
func NewManager(logger *slog.Logger, config ManagerConfig) *Manager {
	return &Manager{
		providers:           make(map[string]SearchProvider),
		enabled:             make(map[string]bool),
		logger:              logger,
		aggregationStrategy: config.AggregationStrategy,
		maxConcurrency:      config.MaxConcurrency,
		timeout:             config.Timeout,
	}
}

// RegisterProvider registers a search provider
func (m *Manager) RegisterProvider(name string, provider SearchProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	m.providers[name] = provider
	m.enabled[name] = provider.IsEnabled()

	m.logger.Info("Provider registered",
		slog.String("name", name),
		slog.Bool("enabled", provider.IsEnabled()))

	return nil
}

// GetProvider returns a specific provider
func (m *Manager) GetProvider(name string) (SearchProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, exists := m.providers[name]
	if !exists {
		return nil, errors.NewNotFoundError("Provider not found", name)
	}

	return provider, nil
}

// GetEnabledProviders returns all enabled providers
func (m *Manager) GetEnabledProviders() []SearchProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var enabled []SearchProvider
	for name, provider := range m.providers {
		if m.enabled[name] && provider.IsEnabled() {
			enabled = append(enabled, provider)
		}
	}

	return enabled
}

// GetAllProviders returns all registered providers
func (m *Manager) GetAllProviders() map[string]SearchProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	copy := make(map[string]SearchProvider)
	for name, provider := range m.providers {
		copy[name] = provider
	}

	return copy
}

// SearchAll searches across all enabled providers
func (m *Manager) SearchAll(ctx context.Context, query *SearchQuery) (*AggregatedResult, error) {
	enabledProviders := m.GetEnabledProviders()
	if len(enabledProviders) == 0 {
		return nil, errors.NewValidationError("No enabled providers available", "providers", "none")
	}

	// Get provider names
	providerNames := make([]string, len(enabledProviders))
	for i, provider := range enabledProviders {
		providerNames[i] = provider.Name()
	}

	return m.SearchProviders(ctx, query, providerNames)
}

// SearchProviders searches specific providers
func (m *Manager) SearchProviders(ctx context.Context, query *SearchQuery, providerNames []string) (*AggregatedResult, error) {
	_ = time.Now() // Removed unused searchStart variable

	// Validate providers
	validProviders := make([]SearchProvider, 0, len(providerNames))
	for _, name := range providerNames {
		provider, err := m.GetProvider(name)
		if err != nil {
			m.logger.Warn("Provider not found, skipping",
				slog.String("name", name),
				slog.String("error", err.Error()))
			continue
		}
		if !provider.IsEnabled() {
			m.logger.Debug("Provider disabled, skipping", slog.String("name", name))
			continue
		}
		validProviders = append(validProviders, provider)
	}

	if len(validProviders) == 0 {
		return nil, errors.NewValidationError("No valid providers available", "providers", providerNames)
	}

	// Execute searches based on aggregation strategy
	switch m.aggregationStrategy {
	case StrategyFirst:
		return m.searchFirst(ctx, query, validProviders)
	case StrategyFastest:
		return m.searchFastest(ctx, query, validProviders)
	case StrategyBestQuality:
		return m.searchBestQuality(ctx, query, validProviders)
	case StrategyRoundRobin:
		return m.searchRoundRobin(ctx, query, validProviders)
	default: // StrategyMerge
		return m.searchMerge(ctx, query, validProviders)
	}
}

// searchMerge executes searches across all providers and merges results
func (m *Manager) searchMerge(ctx context.Context, query *SearchQuery, providers []SearchProvider) (*AggregatedResult, error) {
	searchStart := time.Now()
	resultChan := make(chan *providerResult, len(providers))
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// Launch searches concurrently
	for _, provider := range providers {
		go m.executeProviderSearch(ctx, provider, query, resultChan)
	}

	// Collect results
	providerResults := make(map[string]*SearchResult)
	var allPapers []models.Paper
	var totalCount int
	successfulProviders := make([]string, 0)
	failedProviders := make([]string, 0)
	errors := make([]ProviderError, 0)
	cacheHits := 0

	for i := 0; i < len(providers); i++ {
		select {
		case result := <-resultChan:
			providerResults[result.providerName] = result.result
			if result.err != nil {
				failedProviders = append(failedProviders, result.providerName)
				errors = append(errors, ProviderError{
					Provider:  result.providerName,
					Error:     result.err,
					Type:      classifyError(result.err),
					Retryable: isRetryableError(result.err),
				})
			} else {
				successfulProviders = append(successfulProviders, result.providerName)
				allPapers = append(allPapers, result.result.Papers...)
				totalCount += result.result.TotalCount
				if result.result.CacheHit {
					cacheHits++
				}
			}
		case <-ctx.Done():
			m.logger.Warn("Provider search timeout", slog.String("remaining", fmt.Sprintf("%d", len(providers)-i)))
			break
		}
	}

	// Deduplicate papers
	deduplicatedPapers := m.deduplicatePapers(allPapers)

	result := &AggregatedResult{
		Papers:              deduplicatedPapers,
		TotalCount:          len(deduplicatedPapers),
		ProviderResults:     providerResults,
		Query:               query.Query,
		RequestedProviders:  getProviderNames(providers),
		SuccessfulProviders: successfulProviders,
		FailedProviders:     failedProviders,
		TotalDuration:       time.Since(searchStart),
		CacheHits:           cacheHits,
		RequestID:           query.RequestID,
		Timestamp:           time.Now(),
		AggregationStrategy: string(m.aggregationStrategy),
		PartialFailure:      len(failedProviders) > 0 && len(successfulProviders) > 0,
		Errors:              errors,
	}

	m.logger.Info("Search completed",
		slog.String("query", query.Query),
		slog.Int("total_results", len(deduplicatedPapers)),
		slog.Int("successful_providers", len(successfulProviders)),
		slog.Int("failed_providers", len(failedProviders)),
		slog.Duration("duration", time.Since(searchStart)))

	return result, nil
}

// searchFirst returns the first successful result
func (m *Manager) searchFirst(ctx context.Context, query *SearchQuery, providers []SearchProvider) (*AggregatedResult, error) {
	for _, provider := range providers {
		result, err := provider.Search(ctx, query)
		if err == nil {
			return m.wrapSingleResult(result, provider.Name(), query), nil
		}
		m.logger.Debug("Provider search failed, trying next",
			slog.String("provider", provider.Name()),
			slog.String("error", err.Error()))
	}

	return nil, errors.NewInternalError("All providers failed", nil)
}

// searchFastest returns the fastest result
func (m *Manager) searchFastest(ctx context.Context, query *SearchQuery, providers []SearchProvider) (*AggregatedResult, error) {
	resultChan := make(chan *providerResult, 1)
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// Launch all searches
	for _, provider := range providers {
		go func(p SearchProvider) {
			result, err := p.Search(ctx, query)
			select {
			case resultChan <- &providerResult{p.Name(), result, err}:
			case <-ctx.Done():
			}
		}(provider)
	}

	// Return first successful result
	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		return m.wrapSingleResult(result.result, result.providerName, query), nil
	case <-ctx.Done():
		return nil, errors.NewTimeoutError("Search timeout", 30*time.Second)
	}
}

// searchBestQuality returns the highest quality result
func (m *Manager) searchBestQuality(ctx context.Context, query *SearchQuery, providers []SearchProvider) (*AggregatedResult, error) {
	// For now, treat this the same as merge and select best papers
	result, err := m.searchMerge(ctx, query, providers)
	if err != nil {
		return nil, err
	}

	// Sort papers by quality score and take top results
	m.sortPapersByQuality(result.Papers)
	if len(result.Papers) > query.Limit {
		result.Papers = result.Papers[:query.Limit]
	}

	return result, nil
}

// searchRoundRobin cycles through providers
func (m *Manager) searchRoundRobin(ctx context.Context, query *SearchQuery, providers []SearchProvider) (*AggregatedResult, error) {
	// Simple implementation: use the first provider for now
	// In a real implementation, you'd maintain state about which provider to use next
	if len(providers) == 0 {
		return nil, errors.NewValidationError("No providers available", "providers", "none")
	}

	result, err := providers[0].Search(ctx, query)
	if err != nil {
		return nil, err
	}

	return m.wrapSingleResult(result, providers[0].Name(), query), nil
}

// HealthCheckAll performs health checks on all providers
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)
	for name, provider := range m.providers {
		err := provider.HealthCheck(ctx)
		results[name] = err
	}

	return results
}

// GetProviderMetrics returns metrics for all providers
func (m *Manager) GetProviderMetrics() map[string]ProviderMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]ProviderMetrics)
	for name, provider := range m.providers {
		metrics[name] = provider.GetMetrics()
	}

	return metrics
}

// UpdateProviderConfig updates a provider's configuration
func (m *Manager) UpdateProviderConfig(name string, config ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider, exists := m.providers[name]
	if !exists {
		return errors.NewNotFoundError("Provider not found", name)
	}

	err := provider.Configure(config)
	if err != nil {
		return err
	}

	m.enabled[name] = config.Enabled
	m.logger.Info("Provider configuration updated",
		slog.String("name", name),
		slog.Bool("enabled", config.Enabled))

	return nil
}

// Start initializes all providers
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting provider manager")

	// Perform initial health checks
	healthResults := m.HealthCheckAll(ctx)
	for name, err := range healthResults {
		if err != nil {
			m.logger.Warn("Provider health check failed",
				slog.String("name", name),
				slog.String("error", err.Error()))
		} else {
			m.logger.Debug("Provider health check passed", slog.String("name", name))
		}
	}

	m.logger.Info("Provider manager started",
		slog.Int("total_providers", len(m.providers)),
		slog.Int("enabled_providers", len(m.GetEnabledProviders())))

	return nil
}

// Stop shuts down all providers
func (m *Manager) Stop(ctx context.Context) error {
	m.logger.Info("Stopping provider manager")
	// In a real implementation, you might need to clean up resources
	return nil
}

// Helper methods

type providerResult struct {
	providerName string
	result       *SearchResult
	err          error
}

type ManagerConfig struct {
	AggregationStrategy AggregationStrategy
	MaxConcurrency      int
	Timeout             time.Duration
}

func (m *Manager) executeProviderSearch(ctx context.Context, provider SearchProvider, query *SearchQuery, resultChan chan<- *providerResult) {
	searchStart := time.Now()
	result, err := provider.Search(ctx, query)

	m.logger.Debug("Provider search completed",
		slog.String("provider", provider.Name()),
		slog.Duration("duration", time.Since(searchStart)),
		slog.Bool("success", err == nil))

	select {
	case resultChan <- &providerResult{provider.Name(), result, err}:
	case <-ctx.Done():
	}
}

func (m *Manager) wrapSingleResult(result *SearchResult, providerName string, query *SearchQuery) *AggregatedResult {
	return &AggregatedResult{
		Papers:              result.Papers,
		TotalCount:          result.TotalCount,
		ProviderResults:     map[string]*SearchResult{providerName: result},
		Query:               query.Query,
		RequestedProviders:  []string{providerName},
		SuccessfulProviders: []string{providerName},
		FailedProviders:     []string{},
		TotalDuration:       result.Duration,
		CacheHits:           0,
		RequestID:           query.RequestID,
		Timestamp:           time.Now(),
		AggregationStrategy: string(m.aggregationStrategy),
		PartialFailure:      false,
		Errors:              []ProviderError{},
	}
}

func (m *Manager) deduplicatePapers(papers []models.Paper) []models.Paper {
	seen := make(map[string]bool)
	var unique []models.Paper

	for _, paper := range papers {
		// Use DOI for deduplication if available, otherwise use title
		key := paper.Title
		if paper.DOI != nil && *paper.DOI != "" {
			key = *paper.DOI
		} else if paper.ArxivID != nil && *paper.ArxivID != "" {
			key = *paper.ArxivID
		}

		if !seen[key] {
			seen[key] = true
			unique = append(unique, paper)
		}
	}

	return unique
}

func (m *Manager) sortPapersByQuality(papers []models.Paper) {
	// Simple bubble sort by quality score (descending)
	n := len(papers)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if papers[j].QualityScore < papers[j+1].QualityScore {
				papers[j], papers[j+1] = papers[j+1], papers[j]
			}
		}
	}
}

func getProviderNames(providers []SearchProvider) []string {
	names := make([]string, len(providers))
	for i, provider := range providers {
		names[i] = provider.Name()
	}
	return names
}

func classifyError(err error) string {
	switch {
	case errors.IsTimeoutError(err):
		return "timeout"
	case errors.IsRateLimitError(err):
		return "rate_limit"
	case errors.IsNetworkError(err):
		return "network"
	case errors.IsValidationError(err):
		return "validation"
	default:
		return "unknown"
	}
}

func isRetryableError(err error) bool {
	return errors.IsTimeoutError(err) || errors.IsRateLimitError(err) || errors.IsNetworkError(err)
}
