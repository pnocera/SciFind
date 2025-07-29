package arxiv

import (
	"context"
	"encoding/xml"
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
	defaultBaseURL = "https://export.arxiv.org/api/query"
	providerName   = "arxiv"
	maxResults     = 2000 // ArXiv API limit
)

// Provider implements the ArXiv search provider
type Provider struct {
	config     providers.ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	metrics    *providers.ProviderMetrics
	enabled    bool
}

// NewProvider creates a new ArXiv provider
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
		SupportsFullText:       false, // ArXiv doesn't provide full text search
		SupportsDateFilter:     true,
		SupportsAuthFilter:     true,
		SupportsCategoryFilter: true,
		SupportsSort:           true,

		SupportedFields:    []string{"title", "abstract", "author", "category", "id"},
		SupportedLanguages: []string{"en"},
		SupportedFormats:   []string{"pdf", "ps", "html"},

		MaxResults:     maxResults,
		MaxQueryLength: 1000,
		RateLimit:      180, // 3 requests per second, be conservative

		SupportsRealtime:    true,
		SupportsExactMatch:  true,
		SupportsFuzzySearch: false,
		SupportsWildcards:   false,
	}
}

// Search performs a search using the ArXiv API
func (p *Provider) Search(ctx context.Context, query *providers.SearchQuery) (*providers.SearchResult, error) {
	p.logger.Error("ArXiv search started", 
		slog.String("query", query.Query),
		slog.Bool("enabled", p.enabled),
		slog.String("base_url", p.config.BaseURL))
		
	if !p.enabled {
		p.logger.Error("ArXiv provider is disabled")
		return nil, fmt.Errorf("ArXiv provider is disabled")
	}
		
	start := time.Now()

	// Build ArXiv query
	arxivQuery, err := p.buildQuery(query)
	if err != nil {
		p.logger.Error("Failed to build ArXiv query", slog.String("error", err.Error()))
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to build ArXiv query: %w", err)
	}

	p.logger.Error("ArXiv query built", slog.String("arxiv_query", arxivQuery))

	// Make HTTP request
	response, err := p.makeRequest(ctx, arxivQuery, query.Limit, query.Offset)
	if err != nil {
		p.logger.Error("ArXiv API request failed", slog.String("error", err.Error()))
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("ArXiv API request failed: %w", err)
	}

	// Parse response
	papers, totalCount, err := p.parseResponse(response)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to parse ArXiv response: %w", err)
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

	p.logger.Debug("ArXiv search completed",
		slog.String("query", query.Query),
		slog.Int("results", len(papers)),
		slog.Int("total", totalCount),
		slog.Duration("duration", duration))

	return result, nil
}

// GetPaper retrieves a specific paper by ArXiv ID
func (p *Provider) GetPaper(ctx context.Context, id string) (*models.Paper, error) {
	start := time.Now()

	// Clean ArXiv ID (remove any prefix)
	arxivID := strings.TrimPrefix(id, "arxiv:")
	arxivID = strings.TrimPrefix(arxivID, "arXiv:")

	// Build query to get specific paper
	query := fmt.Sprintf("id:%s", arxivID)

	response, err := p.makeRequest(ctx, query, 1, 0)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("ArXiv API request failed: %w", err)
	}

	papers, _, err := p.parseResponse(response)
	if err != nil {
		p.updateMetrics(false, time.Since(start), err)
		return nil, fmt.Errorf("failed to parse ArXiv response: %w", err)
	}

	if len(papers) == 0 {
		p.updateMetrics(false, time.Since(start), fmt.Errorf("paper not found"))
		return nil, errors.NewNotFoundError("Paper not found in ArXiv", id)
	}

	p.updateMetrics(true, time.Since(start), nil)
	return &papers[0], nil
}

// HealthCheck checks if the ArXiv API is accessible
func (p *Provider) HealthCheck(ctx context.Context) error {
	start := time.Now()

	// Make a simple test query
	testQuery := "cat:cs.AI"
	response, err := p.makeRequest(ctx, testQuery, 1, 0)
	if err != nil {
		return errors.NewHealthCheckError("Health check failed: "+err.Error(), "arxiv")
	}

	// Try to parse the response
	_, _, err = p.parseResponse(response)
	if err != nil {
		return errors.NewHealthCheckError("Parse error: "+err.Error(), "arxiv")
	}

	p.logger.Debug("ArXiv health check passed", slog.Duration("duration", time.Since(start)))
	return nil
}

// GetStatus returns the current provider status
func (p *Provider) GetStatus() providers.ProviderStatus {
	return providers.ProviderStatus{
		Name:            providerName,
		Enabled:         p.enabled,
		Healthy:         true, // Would be updated by health checks
		LastCheck:       time.Now(),
		CircuitState:    "closed",
		RateLimited:     false,
		AvgResponseTime: p.calculateAvgResponseTime(),
		SuccessRate:     p.calculateSuccessRate(),
		APIVersion:      "1.0",
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

	p.logger.Info("ArXiv provider configured",
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

// buildQuery converts a generic search query to ArXiv format
func (p *Provider) buildQuery(query *providers.SearchQuery) (string, error) {
	var parts []string

	// Main search query
	if query.Query != "" {
		// Search in title and abstract by default
		parts = append(parts, fmt.Sprintf("(ti:\"%s\" OR abs:\"%s\")", query.Query, query.Query))
	}

	// Author filter
	if authors, ok := query.Filters[providers.FilterAuthor]; ok {
		for _, author := range strings.Split(authors, ",") {
			if author = strings.TrimSpace(author); author != "" {
				parts = append(parts, fmt.Sprintf("au:\"%s\"", author))
			}
		}
	}

	// Category filter
	if categories, ok := query.Filters[providers.FilterCategory]; ok {
		for _, category := range strings.Split(categories, ",") {
			if category = strings.TrimSpace(category); category != "" {
				parts = append(parts, fmt.Sprintf("cat:%s", category))
			}
		}
	}

	// Date filters
	if query.DateFrom != nil {
		parts = append(parts, fmt.Sprintf("submittedDate:[%s TO *]", query.DateFrom.Format("20060102")))
	}

	if query.DateTo != nil {
		parts = append(parts, fmt.Sprintf("submittedDate:[* TO %s]", query.DateTo.Format("20060102")))
	}

	// If no query parts, search everything (limited)
	if len(parts) == 0 {
		parts = append(parts, "cat:cs.*") // Default to computer science
	}

	return strings.Join(parts, " AND "), nil
}

// makeRequest makes an HTTP request to the ArXiv API
func (p *Provider) makeRequest(ctx context.Context, query string, maxresults, start int) ([]byte, error) {
	// Limit results to API maximum
	if maxresults > maxResults {
		maxresults = maxResults
	}

	// Build URL
	params := url.Values{}
	params.Set("search_query", query)
	params.Set("start", strconv.Itoa(start))
	params.Set("max_results", strconv.Itoa(maxresults))
	params.Set("sortBy", "submittedDate")
	params.Set("sortOrder", "descending")

	reqURL := p.config.BaseURL + "?" + params.Encode()

	// Log the full URL for debugging  
	p.logger.Error("ArXiv API request URL", slog.String("url", reqURL))

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "SciFIND-Backend/1.0")

	// Make request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ArXiv API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// parseResponse parses the ArXiv API XML response
func (p *Provider) parseResponse(data []byte) ([]models.Paper, int, error) {
	var feed ArxivFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	papers := make([]models.Paper, 0, len(feed.Entries))

	for _, entry := range feed.Entries {
		paper, err := p.convertEntry(entry)
		if err != nil {
			p.logger.Warn("Failed to convert ArXiv entry",
				slog.String("id", entry.ID),
				slog.String("error", err.Error()))
			continue
		}
		papers = append(papers, *paper)
	}

	// ArXiv doesn't provide total count in the response, estimate it
	totalCount := len(papers)
	if len(papers) == maxResults {
		totalCount = maxResults * 2 // Conservative estimate
	}

	return papers, totalCount, nil
}

// convertEntry converts an ArXiv entry to a Paper model
func (p *Provider) convertEntry(entry ArxivEntry) (*models.Paper, error) {
	// Extract ArXiv ID from the entry ID
	arxivID := extractArxivID(entry.ID)
	if arxivID == "" {
		return nil, fmt.Errorf("invalid ArXiv ID: %s", entry.ID)
	}

	// Parse published date
	var publishedAt *time.Time
	if entry.Published != "" {
		if parsed, err := time.Parse(time.RFC3339, entry.Published); err == nil {
			publishedAt = &parsed
		}
	}

	// Convert authors
	authors := make([]models.Author, 0, len(entry.Authors))
	for _, authorData := range entry.Authors {
		author := models.Author{
			Name: authorData.Name,
		}
		authors = append(authors, author)
	}

	// Extract categories
	categories := make([]models.Category, 0, len(entry.Categories))
	for _, catData := range entry.Categories {
		category := models.Category{
			ID:         "arxiv_" + catData.Term,
			Name:       p.getCategoryName(catData.Term),
			Source:     "arxiv",
			SourceCode: catData.Term,
			IsActive:   true,
		}
		categories = append(categories, category)
	}

	// Extract PDF URL
	var pdfURL *string
	for _, link := range entry.Links {
		if link.Type == "application/pdf" {
			pdfURL = &link.Href
			break
		}
	}

	// Build paper
	paper := &models.Paper{
		ID:              "arxiv_" + arxivID,
		ArxivID:         &arxivID,
		Title:           entry.Title,
		Abstract:        &entry.Summary,
		Authors:         authors,
		Categories:      categories,
		PublishedAt:     publishedAt,
		URL:             &entry.ID,
		PDFURL:          pdfURL,
		Language:        "en",
		SourceProvider:  providerName,
		SourceID:        arxivID,
		SourceURL:       &entry.ID,
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

func extractArxivID(entryID string) string {
	// ArXiv entry IDs are in format: http://arxiv.org/abs/1234.5678v1
	parts := strings.Split(entryID, "/")
	if len(parts) == 0 {
		return ""
	}

	id := parts[len(parts)-1]

	// Remove version suffix if present
	if idx := strings.LastIndex(id, "v"); idx > 0 {
		id = id[:idx]
	}

	return id
}

func (p *Provider) getCategoryName(term string) string {
	// Map ArXiv category codes to readable names
	categoryNames := map[string]string{
		"cs.AI":            "Artificial Intelligence",
		"cs.CL":            "Computation and Language",
		"cs.CV":            "Computer Vision and Pattern Recognition",
		"cs.LG":            "Machine Learning",
		"cs.DS":            "Data Structures and Algorithms",
		"cs.DB":            "Databases",
		"cs.DC":            "Distributed, Parallel, and Cluster Computing",
		"cs.CR":            "Cryptography and Security",
		"cs.GT":            "Computer Science and Game Theory",
		"cs.HC":            "Human-Computer Interaction",
		"cs.IR":            "Information Retrieval",
		"cs.IT":            "Information Theory",
		"cs.LO":            "Logic in Computer Science",
		"cs.NE":            "Neural and Evolutionary Computing",
		"cs.NI":            "Networking and Internet Architecture",
		"cs.OH":            "Other Computer Science",
		"cs.OS":            "Operating Systems",
		"cs.PF":            "Performance",
		"cs.PL":            "Programming Languages",
		"cs.RO":            "Robotics",
		"cs.SE":            "Software Engineering",
		"cs.SY":            "Systems and Control",
		"math.AC":          "Commutative Algebra",
		"math.AG":          "Algebraic Geometry",
		"math.AP":          "Analysis of PDEs",
		"math.AT":          "Algebraic Topology",
		"math.CA":          "Classical Analysis and ODEs",
		"math.CO":          "Combinatorics",
		"math.CT":          "Category Theory",
		"math.CV":          "Complex Variables",
		"math.DG":          "Differential Geometry",
		"math.DS":          "Dynamical Systems",
		"math.FA":          "Functional Analysis",
		"math.GM":          "General Mathematics",
		"math.GN":          "General Topology",
		"math.GR":          "Group Theory",
		"math.GT":          "Geometric Topology",
		"math.HO":          "History and Overview",
		"math.IT":          "Information Theory",
		"math.KT":          "K-Theory and Homology",
		"math.LO":          "Logic",
		"math.MG":          "Metric Geometry",
		"math.MP":          "Mathematical Physics",
		"math.NA":          "Numerical Analysis",
		"math.NT":          "Number Theory",
		"math.OA":          "Operator Algebras",
		"math.OC":          "Optimization and Control",
		"math.PR":          "Probability",
		"math.QA":          "Quantum Algebra",
		"math.RA":          "Rings and Algebras",
		"math.RT":          "Representation Theory",
		"math.SG":          "Symplectic Geometry",
		"math.SP":          "Spectral Theory",
		"math.ST":          "Statistics Theory",
		"physics.acc-ph":   "Accelerator Physics",
		"physics.ao-ph":    "Atmospheric and Oceanic Physics",
		"physics.app-ph":   "Applied Physics",
		"physics.atm-clus": "Atomic and Molecular Clusters",
		"physics.atom-ph":  "Atomic Physics",
		"physics.bio-ph":   "Biological Physics",
		"physics.chem-ph":  "Chemical Physics",
		"physics.class-ph": "Classical Physics",
		"physics.comp-ph":  "Computational Physics",
		"physics.data-an":  "Data Analysis, Statistics and Probability",
		"physics.ed-ph":    "Physics Education",
		"physics.flu-dyn":  "Fluid Dynamics",
		"physics.gen-ph":   "General Physics",
		"physics.geo-ph":   "Geophysics",
		"physics.hist-ph":  "History and Philosophy of Physics",
		"physics.ins-det":  "Instrumentation and Detectors",
		"physics.med-ph":   "Medical Physics",
		"physics.optics":   "Optics",
		"physics.plasm-ph": "Plasma Physics",
		"physics.pop-ph":   "Popular Physics",
		"physics.soc-ph":   "Physics and Society",
		"physics.space-ph": "Space Physics",
		"stat.AP":          "Applications",
		"stat.CO":          "Computation",
		"stat.ME":          "Methodology",
		"stat.ML":          "Machine Learning",
		"stat.OT":          "Other Statistics",
		"stat.TH":          "Statistics Theory",
		"q-bio.BM":         "Biomolecules",
		"q-bio.CB":         "Cell Behavior",
		"q-bio.GN":         "Genomics",
		"q-bio.MN":         "Molecular Networks",
		"q-bio.NC":         "Neurons and Cognition",
		"q-bio.OT":         "Other Quantitative Biology",
		"q-bio.PE":         "Populations and Evolution",
		"q-bio.QM":         "Quantitative Methods",
		"q-bio.SC":         "Subcellular Processes",
		"q-bio.TO":         "Tissues and Organs",
		"q-fin.CP":         "Computational Finance",
		"q-fin.EC":         "Economics",
		"q-fin.GN":         "General Finance",
		"q-fin.MF":         "Mathematical Finance",
		"q-fin.PM":         "Portfolio Management",
		"q-fin.PR":         "Pricing of Securities",
		"q-fin.RM":         "Risk Management",
		"q-fin.ST":         "Statistical Finance",
		"q-fin.TR":         "Trading and Market Microstructure",
		"econ.EM":          "Econometrics",
		"econ.GN":          "General Economics",
		"econ.TH":          "Theoretical Economics",
		"eess.AS":          "Audio and Speech Processing",
		"eess.IV":          "Image and Video Processing",
		"eess.SP":          "Signal Processing",
		"eess.SY":          "Systems and Control",
	}

	if name, exists := categoryNames[term]; exists {
		return name
	}

	return term // Return the term itself if no mapping found
}
