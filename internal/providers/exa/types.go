package exa

import (
	"fmt"
	"time"
)

// ExaSearchResponse represents the search response from Exa API
type ExaSearchResponse struct {
	Results        []ExaResult `json:"results"`
	RequestID      string      `json:"requestId"`
	ProcessingTime int         `json:"processingTimeMs"`
	Autoprompt     *string     `json:"autoprompt,omitempty"`
}

// ExaResult represents a single search result from Exa
type ExaResult struct {
	ID              string    `json:"id"`
	URL             string    `json:"url"`
	Title           string    `json:"title"`
	Score           float64   `json:"score"`
	PublishedDate   *string   `json:"publishedDate,omitempty"`
	Author          *string   `json:"author,omitempty"`
	Text            *string   `json:"text,omitempty"`
	Summary         *string   `json:"summary,omitempty"`
	Highlights      []string  `json:"highlights,omitempty"`
	HighlightScores []float64 `json:"highlightScores,omitempty"`
}

// ExaSearchRequest represents a search request to Exa API
type ExaSearchRequest struct {
	Query              string   `json:"query"`
	Type               string   `json:"type"` // "neural", "keyword", "auto"
	UseAutoprompt      bool     `json:"useAutoprompt"`
	NumResults         int      `json:"numResults"`
	IncludeDomains     []string `json:"includeDomains,omitempty"`
	ExcludeDomains     []string `json:"excludeDomains,omitempty"`
	StartCrawlDate     *string  `json:"startCrawlDate,omitempty"`
	EndCrawlDate       *string  `json:"endCrawlDate,omitempty"`
	StartPublishedDate *string  `json:"startPublishedDate,omitempty"`
	EndPublishedDate   *string  `json:"endPublishedDate,omitempty"`
	IncludeText        bool     `json:"includeText"`
	IncludeSummary     bool     `json:"includeSummary"`
	IncludeHighlights  bool     `json:"includeHighlights"`
	Category           string   `json:"category,omitempty"`
	SimilarityTopK     int      `json:"similarityTopK,omitempty"`
}

// ExaContentsRequest represents a request to get full content for URLs
type ExaContentsRequest struct {
	IDs        []string `json:"ids"`
	Text       bool     `json:"text"`
	Summary    bool     `json:"summary"`
	Highlights bool     `json:"highlights"`
	LiveCrawl  bool     `json:"liveCrawl"`
}

// ExaContentsResponse represents the contents response from Exa API
type ExaContentsResponse struct {
	Results []ExaContentResult `json:"results"`
}

// ExaContentResult represents content for a specific URL
type ExaContentResult struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	Title      string   `json:"title"`
	Text       *string  `json:"text,omitempty"`
	Summary    *string  `json:"summary,omitempty"`
	Highlights []string `json:"highlights,omitempty"`
}

// ExaFindSimilarRequest represents a request to find similar content
type ExaFindSimilarRequest struct {
	URL                string   `json:"url"`
	NumResults         int      `json:"numResults"`
	IncludeDomains     []string `json:"includeDomains,omitempty"`
	ExcludeDomains     []string `json:"excludeDomains,omitempty"`
	StartCrawlDate     *string  `json:"startCrawlDate,omitempty"`
	EndCrawlDate       *string  `json:"endCrawlDate,omitempty"`
	StartPublishedDate *string  `json:"startPublishedDate,omitempty"`
	EndPublishedDate   *string  `json:"endPublishedDate,omitempty"`
	Category           string   `json:"category,omitempty"`
}

// ExaError represents an error response from the Exa API
type ExaError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// ExaMetrics represents Exa-specific metrics
type ExaMetrics struct {
	NeuralSearches     int64   `json:"neural_searches"`
	KeywordSearches    int64   `json:"keyword_searches"`
	AutoPromptUsage    int64   `json:"autoprompt_usage"`
	AvgScore           float64 `json:"avg_score"`
	HighlightHits      int64   `json:"highlight_hits"`
	SimilaritySearches int64   `json:"similarity_searches"`
}

// Academic domain mappings for better research results
var AcademicDomains = []string{
	"arxiv.org",
	"pubmed.ncbi.nlm.nih.gov",
	"scholar.google.com",
	"researchgate.net",
	"academia.edu",
	"semanticscholar.org",
	"jstor.org",
	"ieee.org",
	"acm.org",
	"springer.com",
	"sciencedirect.com",
	"nature.com",
	"science.org",
	"plos.org",
	"biorxiv.org",
	"medrxiv.org",
	"psyarxiv.com",
	"osf.io",
}

// Research categories for filtering
var ResearchCategories = map[string]string{
	"computer_science": "computer science",
	"ai_ml":            "artificial intelligence machine learning",
	"physics":          "physics",
	"biology":          "biology",
	"chemistry":        "chemistry",
	"medicine":         "medicine",
	"mathematics":      "mathematics",
	"engineering":      "engineering",
	"psychology":       "psychology",
	"economics":        "economics",
	"sociology":        "sociology",
	"history":          "history",
	"literature":       "literature",
	"philosophy":       "philosophy",
}

// BuildExaSearchRequest creates a search request optimized for academic content
func BuildExaSearchRequest(query string, filters map[string]string, limit, offset int) *ExaSearchRequest {
	req := &ExaSearchRequest{
		Query:             query,
		Type:              "neural", // Use neural search for better semantic understanding
		UseAutoprompt:     true,     // Let Exa optimize the query
		NumResults:        limit,
		IncludeDomains:    AcademicDomains,
		IncludeText:       true,
		IncludeSummary:    true,
		IncludeHighlights: true,
	}

	// Add date filters
	if startDate, ok := filters["start_date"]; ok {
		req.StartPublishedDate = &startDate
	}
	if endDate, ok := filters["end_date"]; ok {
		req.EndPublishedDate = &endDate
	}

	// Add category filter
	if category, ok := filters["category"]; ok {
		if categoryQuery, exists := ResearchCategories[category]; exists {
			req.Category = categoryQuery
		}
	}

	// Add domain filters
	if domains, ok := filters["include_domains"]; ok {
		// Parse comma-separated domains
		req.IncludeDomains = append(req.IncludeDomains, domains)
	}

	if excludeDomains, ok := filters["exclude_domains"]; ok {
		// Parse comma-separated domains
		req.ExcludeDomains = []string{excludeDomains}
	}

	return req
}

// FormatDateForExa formats a time.Time for Exa API (YYYY-MM-DD format)
func FormatDateForExa(t time.Time) string {
	return t.Format("2006-01-02")
}

// ValidateExaConfig validates Exa provider configuration
func ValidateExaConfig(config map[string]interface{}) error {
	if apiKey, ok := config["api_key"].(string); !ok || apiKey == "" {
		return fmt.Errorf("exa api_key is required")
	}
	return nil
}

// ExaSearchType constants
const (
	SearchTypeNeural  = "neural"
	SearchTypeKeyword = "keyword"
	SearchTypeAuto    = "auto"
)

// ExaCategory constants for common academic fields
const (
	CategoryComputerScience = "computer science"
	CategoryAI              = "artificial intelligence"
	CategoryPhysics         = "physics"
	CategoryBiology         = "biology"
	CategoryChemistry       = "chemistry"
	CategoryMedicine        = "medicine"
	CategoryMathematics     = "mathematics"
	CategoryEngineering     = "engineering"
)
