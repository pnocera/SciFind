package tavily

import (
	"fmt"
	"strings"
	"time"
)

// TavilySearchResponse represents the search response from Tavily API
type TavilySearchResponse struct {
	Query           string         `json:"query"`
	FollowUpQueries []string       `json:"follow_up_queries"`
	Answer          string         `json:"answer"`
	Images          []TavilyImage  `json:"images"`
	Results         []TavilyResult `json:"results"`
	ResponseTime    float64        `json:"response_time"`
}

// TavilyResult represents a single search result from Tavily
type TavilyResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
	RawContent *string `json:"raw_content,omitempty"`
}

// TavilyImage represents an image result from Tavily
type TavilyImage struct {
	URL string `json:"url"`
}

// TavilySearchRequest represents a search request to Tavily API
type TavilySearchRequest struct {
	Query              string   `json:"query"`
	SearchDepth        string   `json:"search_depth"`        // "basic" or "advanced"
	IncludeImages      bool     `json:"include_images"`
	IncludeAnswer      bool     `json:"include_answer"`
	IncludeRawContent  bool     `json:"include_raw_content"`
	MaxResults         int      `json:"max_results"`
	IncludeDomains     []string `json:"include_domains,omitempty"`
	ExcludeDomains     []string `json:"exclude_domains,omitempty"`
	Format             string   `json:"format,omitempty"`     // "json" or "markdown"
}

// TavilyExtractRequest represents a request to extract content from URLs
type TavilyExtractRequest struct {
	URLs []string `json:"urls"`
}

// TavilyExtractResponse represents the extract response from Tavily API
type TavilyExtractResponse struct {
	Results []TavilyExtractResult `json:"results"`
}

// TavilyExtractResult represents extracted content from a URL
type TavilyExtractResult struct {
	URL        string `json:"url"`
	RawContent string `json:"raw_content"`
}

// TavilyError represents an error response from the Tavily API
type TavilyError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// TavilyMetrics represents Tavily-specific metrics
type TavilyMetrics struct {
	BasicSearches    int64   `json:"basic_searches"`
	AdvancedSearches int64   `json:"advanced_searches"`
	AvgScore        float64 `json:"avg_score"`
	AnswerHits      int64   `json:"answer_hits"`
	ImageResults    int64   `json:"image_results"`
	ExtractRequests int64   `json:"extract_requests"`
}

// Academic and research-focused domains for better scholarly results
var ResearchDomains = []string{
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
	"ncbi.nlm.nih.gov",
	"wiley.com",
	"tandfonline.com",
	"sagepub.com",
	"cambridge.org",
	"oxfordjournals.org",
	"acs.org",
	"aps.org",
	"iop.org",
	"rsc.org",
}

// Low-quality domains to exclude for academic searches
var ExcludedDomains = []string{
	"facebook.com",
	"twitter.com",
	"instagram.com",
	"tiktok.com",
	"pinterest.com",
	"reddit.com",
	"quora.com",
	"yahoo.com",
	"ask.com",
	"answers.com",
	"ehow.com",
	"wikihow.com",
}

// BuildTavilySearchRequest creates a search request optimized for academic research
func BuildTavilySearchRequest(query string, filters map[string]string, limit, offset int) *TavilySearchRequest {
	req := &TavilySearchRequest{
		Query:             query,
		SearchDepth:       "advanced", // Use advanced search for better academic results
		IncludeImages:     false,      // Focus on text content for academic research
		IncludeAnswer:     true,       // Include AI-generated answer for context
		IncludeRawContent: true,       // Include raw content for better text extraction
		MaxResults:        limit,
		IncludeDomains:    ResearchDomains,
		ExcludeDomains:    ExcludedDomains,
		Format:           "json",
	}

	// Add custom domain filters if specified
	if domains, ok := filters["include_domains"]; ok {
		// Parse comma-separated domains and add to existing research domains
		customDomains := strings.Split(domains, ",")
		req.IncludeDomains = append(req.IncludeDomains, customDomains...)
	}

	if excludeDomains, ok := filters["exclude_domains"]; ok {
		// Parse comma-separated domains
		customExcluded := strings.Split(excludeDomains, ",")
		req.ExcludeDomains = append(req.ExcludeDomains, customExcluded...)
	}

	// Adjust search depth based on query complexity
	if searchDepth, ok := filters["search_depth"]; ok {
		if searchDepth == "basic" || searchDepth == "advanced" {
			req.SearchDepth = searchDepth
		}
	}

	return req
}

// ValidateTavilyConfig validates Tavily provider configuration
func ValidateTavilyConfig(config map[string]interface{}) error {
	if apiKey, ok := config["api_key"].(string); !ok || apiKey == "" {
		return fmt.Errorf("tavily api_key is required")
	}
	return nil
}

// TavilySearchDepth constants
const (
	SearchDepthBasic    = "basic"
	SearchDepthAdvanced = "advanced"
)

// Format constants
const (
	FormatJSON     = "json"
	FormatMarkdown = "markdown"
)

// ExtractMetadata extracts metadata from Tavily result content
func ExtractMetadata(content string) map[string]string {
	metadata := make(map[string]string)
	
	// Simple heuristics to extract common metadata
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Published:") {
			metadata["published_date"] = strings.TrimSpace(strings.Split(line, "Published:")[1])
		} else if strings.Contains(line, "Author:") {
			metadata["author"] = strings.TrimSpace(strings.Split(line, "Author:")[1])
		} else if strings.Contains(line, "DOI:") {
			metadata["doi"] = strings.TrimSpace(strings.Split(line, "DOI:")[1])
		} else if strings.Contains(line, "Journal:") {
			metadata["journal"] = strings.TrimSpace(strings.Split(line, "Journal:")[1])
		}
	}
	
	return metadata
}

// IsAcademicURL checks if a URL is from an academic or research source
func IsAcademicURL(url string) bool {
	for _, domain := range ResearchDomains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

// EstimateQualityScore estimates a quality score based on Tavily result properties
func EstimateQualityScore(result TavilyResult) float64 {
	score := result.Score
	
	// Boost score for academic domains
	if IsAcademicURL(result.URL) {
		score *= 1.3
	}
	
	// Boost score for longer, more detailed content
	contentLength := len(result.Content)
	if contentLength > 1000 {
		score *= 1.2
	} else if contentLength > 500 {
		score *= 1.1
	}
	
	// Normalize to 0-1 range
	if score > 1.0 {
		score = 1.0
	}
	
	return score
}

// ParsePublishedDate attempts to parse a published date from various formats
func ParsePublishedDate(dateStr string) (*time.Time, error) {
	// Common date formats found in academic content
	formats := []string{
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006/01/02",
		"02/01/2006",
		"01-02-2006",
		"2006",
	}
	
	for _, format := range formats {
		if parsed, err := time.Parse(format, dateStr); err == nil {
			return &parsed, nil
		}
	}
	
	return nil, fmt.Errorf("unable to parse date: %s", dateStr)
}