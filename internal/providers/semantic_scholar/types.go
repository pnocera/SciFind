package semantic_scholar

import "strings"

// SemanticScholarSearchResponse represents the search response from Semantic Scholar API
type SemanticScholarSearchResponse struct {
	Total  int                     `json:"total"`
	Offset int                     `json:"offset"`
	Next   *int                    `json:"next"`
	Data   []SemanticScholarPaper  `json:"data"`
}

// SemanticScholarPaper represents a paper from Semantic Scholar API
type SemanticScholarPaper struct {
	PaperID        string                    `json:"paperId"`
	ExternalIDs    *ExternalIDs              `json:"externalIds"`
	Title          string                    `json:"title"`
	Abstract       string                    `json:"abstract"`
	Authors        []SemanticScholarAuthor   `json:"authors"`
	Venue          string                    `json:"venue"`
	Year           int                       `json:"year"`
	CitationCount  int                       `json:"citationCount"`
	ReferenceCount int                       `json:"referenceCount"`
	FieldsOfStudy  []FieldOfStudy            `json:"fieldsOfStudy"`
	URL            string                    `json:"url"`
	OpenAccessPDF  *OpenAccessPDF            `json:"openAccessPdf"`
}

// ExternalIDs represents external identifiers for a paper
type ExternalIDs struct {
	ArxivID      string `json:"ArXiv"`
	DOI          string `json:"DOI"`
	MAG          string `json:"MAG"`
	PubMed       string `json:"PubMed"`
	PubMedCentral string `json:"PubMedCentral"`
	DBLP         string `json:"DBLP"`
}

// SemanticScholarAuthor represents an author from Semantic Scholar API
type SemanticScholarAuthor struct {
	AuthorID   string `json:"authorId"`
	Name       string `json:"name"`
	URL        string `json:"url"`
}

// FieldOfStudy represents a field of study classification
type FieldOfStudy struct {
	Category string `json:"category"`
	Source   string `json:"source"`
}

// OpenAccessPDF represents open access PDF information
type OpenAccessPDF struct {
	URL    string `json:"url"`
	Status string `json:"status"`
}

// SemanticScholarBulkResponse represents bulk paper retrieval response
type SemanticScholarBulkResponse []SemanticScholarPaper

// SemanticScholarAuthorResponse represents author details response
type SemanticScholarAuthorResponse struct {
	AuthorID      string                    `json:"authorId"`
	ExternalIDs   map[string]string         `json:"externalIds"`
	Name          string                    `json:"name"`
	Aliases       []string                  `json:"aliases"`
	Affiliations  []string                  `json:"affiliations"`
	Homepage      string                    `json:"homepage"`
	PaperCount    int                       `json:"paperCount"`
	CitationCount int                       `json:"citationCount"`
	HIndex        int                       `json:"hIndex"`
	Papers        []SemanticScholarPaper    `json:"papers"`
}

// SemanticScholarCitationsResponse represents citations response
type SemanticScholarCitationsResponse struct {
	Offset int                     `json:"offset"`
	Next   *int                    `json:"next"`
	Data   []CitationContext       `json:"data"`
}

// CitationContext represents citation context information
type CitationContext struct {
	PaperID      string                `json:"paperId"`
	CorpusID     int                   `json:"corpusId"`
	Title        string                `json:"title"`
	Venue        string                `json:"venue"`
	Year         int                   `json:"year"`
	Authors      []SemanticScholarAuthor `json:"authors"`
	Intent       []string              `json:"intent"`
	IsInfluential bool                 `json:"isInfluential"`
	Contexts     []string              `json:"contexts"`
}

// SemanticScholarReferencesResponse represents references response
type SemanticScholarReferencesResponse struct {
	Offset int                     `json:"offset"`
	Next   *int                    `json:"next"`
	Data   []ReferenceContext      `json:"data"`
}

// ReferenceContext represents reference context information
type ReferenceContext struct {
	PaperID      string                `json:"paperId"`
	CorpusID     int                   `json:"corpusId"`
	Title        string                `json:"title"`
	Venue        string                `json:"venue"`
	Year         int                   `json:"year"`
	Authors      []SemanticScholarAuthor `json:"authors"`
	Intent       []string              `json:"intent"`
	IsInfluential bool                 `json:"isInfluential"`
	Contexts     []string              `json:"contexts"`
}

// SemanticScholarError represents an error response from the API
type SemanticScholarError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SemanticScholarQuery represents query parameters for the API
type SemanticScholarQuery struct {
	Query         string   `json:"query"`
	Year          string   `json:"year,omitempty"`
	Venue         string   `json:"venue,omitempty"`
	FieldsOfStudy []string `json:"fieldsOfStudy,omitempty"`
	Offset        int      `json:"offset"`
	Limit         int      `json:"limit"`
	Fields        []string `json:"fields"`
}

// Standard field sets for different query types
var (
	// BasicFields contains essential paper information
	BasicFields = []string{
		"paperId", "title", "authors", "venue", "year", "citationCount",
	}

	// DetailedFields contains comprehensive paper information
	DetailedFields = []string{
		"paperId", "externalIds", "title", "abstract", "authors",
		"venue", "year", "citationCount", "referenceCount",
		"fieldsOfStudy", "url", "openAccessPdf",
	}

	// AuthorFields contains author-specific information
	AuthorFields = []string{
		"authorId", "name", "aliases", "affiliations", "homepage",
		"paperCount", "citationCount", "hIndex",
	}

	// CitationFields contains citation context information
	CitationFields = []string{
		"paperId", "title", "authors", "venue", "year",
		"intent", "isInfluential", "contexts",
	}
)

// SemanticScholarMetrics represents provider-specific metrics
type SemanticScholarMetrics struct {
	APIKeyUsed       bool    `json:"api_key_used"`
	RateLimitHits    int64   `json:"rate_limit_hits"`
	AvgResultsPerQuery float64 `json:"avg_results_per_query"`
	PopularFields    map[string]int `json:"popular_fields"`
}

// GetFieldSet returns appropriate field set based on query type
func GetFieldSet(queryType string) []string {
	switch queryType {
	case "basic":
		return BasicFields
	case "detailed":
		return DetailedFields
	case "author":
		return AuthorFields
	case "citation":
		return CitationFields
	default:
		return DetailedFields
	}
}

// ValidateFieldsOfStudy checks if fields of study are valid
func ValidateFieldsOfStudy(fields []string) []string {
	validFields := map[string]bool{
		"Computer Science": true,
		"Medicine": true,
		"Chemistry": true,
		"Biology": true,
		"Materials Science": true,
		"Physics": true,
		"Geology": true,
		"Psychology": true,
		"Art": true,
		"History": true,
		"Geography": true,
		"Sociology": true,
		"Business": true,
		"Political Science": true,
		"Economics": true,
		"Philosophy": true,
		"Mathematics": true,
		"Engineering": true,
		"Environmental Science": true,
		"Agricultural and Food Sciences": true,
		"Education": true,
		"Law": true,
		"Linguistics": true,
	}

	var valid []string
	for _, field := range fields {
		if validFields[field] {
			valid = append(valid, field)
		}
	}

	return valid
}

// BuildSemanticScholarQuery constructs a query for the API
func BuildSemanticScholarQuery(query string, filters map[string]string, limit, offset int) *SemanticScholarQuery {
	ssQuery := &SemanticScholarQuery{
		Query:  query,
		Offset: offset,
		Limit:  limit,
		Fields: DetailedFields,
	}

	// Add filters
	if year, ok := filters["year"]; ok {
		ssQuery.Year = year
	}

	if venue, ok := filters["venue"]; ok {
		ssQuery.Venue = venue
	}

	if fields, ok := filters["fieldsOfStudy"]; ok {
		ssQuery.FieldsOfStudy = strings.Split(fields, ",")
		ssQuery.FieldsOfStudy = ValidateFieldsOfStudy(ssQuery.FieldsOfStudy)
	}

	return ssQuery
}