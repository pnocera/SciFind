package repository

import (
	"context"
	"time"

	"scifind-backend/internal/models"
)

// PaperRepository defines the interface for paper database operations
type PaperRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, paper *models.Paper) error
	GetByID(ctx context.Context, id string) (*models.Paper, error)
	GetByDOI(ctx context.Context, doi string) (*models.Paper, error)
	GetByArxivID(ctx context.Context, arxivID string) (*models.Paper, error)
	Update(ctx context.Context, paper *models.Paper) error
	Delete(ctx context.Context, id string) error
	
	// Search and filtering
	Search(ctx context.Context, query string, filters *models.PaperFilter, sort *models.PaperSort, limit, offset int) ([]models.Paper, int64, error)
	SearchByVector(ctx context.Context, embedding []float32, filters *models.PaperFilter, limit int) ([]models.Paper, error)
	SearchFullText(ctx context.Context, query string, filters *models.PaperFilter, limit, offset int) ([]models.Paper, int64, error)
	
	// Bulk operations
	CreateBatch(ctx context.Context, papers []models.Paper) error
	UpdateBatch(ctx context.Context, papers []models.Paper) error
	
	// Statistics and analytics
	GetStats(ctx context.Context, filters *models.PaperFilter) (*PaperStats, error)
	GetTrendingPapers(ctx context.Context, since time.Time, limit int) ([]models.Paper, error)
	GetPopularPapers(ctx context.Context, limit int) ([]models.Paper, error)
	
	// Processing state management
	GetPendingPapers(ctx context.Context, limit int) ([]models.Paper, error)
	UpdateProcessingState(ctx context.Context, paperID string, state string) error
	GetProcessingStats(ctx context.Context) (*ProcessingStats, error)
	
	// Relationships
	GetAuthorPapers(ctx context.Context, authorID string, limit, offset int) ([]models.Paper, error)
	GetCategoryPapers(ctx context.Context, categoryID string, limit, offset int) ([]models.Paper, error)
	GetSimilarPapers(ctx context.Context, paperID string, limit int) ([]models.Paper, error)
	
	// Citation analysis
	GetCitations(ctx context.Context, paperID string) ([]models.Paper, error)
	GetReferences(ctx context.Context, paperID string) ([]models.Paper, error)
	UpdateCitationCount(ctx context.Context, paperID string, count int) error
}

// AuthorRepository defines the interface for author database operations
type AuthorRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, author *models.Author) error
	GetByID(ctx context.Context, id string) (*models.Author, error)
	GetByEmail(ctx context.Context, email string) (*models.Author, error)
	GetByORCID(ctx context.Context, orcid string) (*models.Author, error)
	Update(ctx context.Context, author *models.Author) error
	Delete(ctx context.Context, id string) error
	
	// Search and filtering
	Search(ctx context.Context, query string, filters *models.AuthorFilter, sort *models.AuthorSort, limit, offset int) ([]models.Author, int64, error)
	SearchByName(ctx context.Context, name string, limit int) ([]models.Author, error)
	SearchByAffiliation(ctx context.Context, affiliation string, limit int) ([]models.Author, error)
	
	// Bulk operations
	CreateBatch(ctx context.Context, authors []models.Author) error
	UpdateBatch(ctx context.Context, authors []models.Author) error
	
	// Statistics and analytics
	GetStats(ctx context.Context, filters *models.AuthorFilter) (*AuthorStats, error)
	GetTopAuthors(ctx context.Context, criteria string, limit int) ([]models.Author, error)
	GetProductiveAuthors(ctx context.Context, minPapers int, limit int) ([]models.Author, error)
	
	// Relationships
	GetCollaborators(ctx context.Context, authorID string, limit int) ([]models.Author, error)
	GetAuthorsByPaper(ctx context.Context, paperID string) ([]models.Author, error)
	
	// Metrics management
	UpdateMetrics(ctx context.Context, authorID string, paperCount, citationCount, hIndex int) error
	RecalculateMetrics(ctx context.Context, authorID string) error
}

// CategoryRepository defines the interface for category database operations
type CategoryRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, category *models.Category) error
	GetByID(ctx context.Context, id string) (*models.Category, error)
	GetBySourceCode(ctx context.Context, source, sourceCode string) (*models.Category, error)
	Update(ctx context.Context, category *models.Category) error
	Delete(ctx context.Context, id string) error
	
	// Hierarchy operations
	GetRootCategories(ctx context.Context, source string) ([]models.Category, error)
	GetChildren(ctx context.Context, parentID string) ([]models.Category, error)
	GetParent(ctx context.Context, categoryID string) (*models.Category, error)
	GetAncestors(ctx context.Context, categoryID string) ([]models.Category, error)
	GetDescendants(ctx context.Context, categoryID string) ([]models.Category, error)
	GetCategoryTree(ctx context.Context, source string) ([]models.CategoryTree, error)
	
	// Search and filtering
	Search(ctx context.Context, query string, filters *models.CategoryFilter, sort *models.CategorySort, limit, offset int) ([]models.Category, int64, error)
	GetActiveCategories(ctx context.Context, source string) ([]models.Category, error)
	GetPopularCategories(ctx context.Context, limit int) ([]models.Category, error)
	
	// Bulk operations
	CreateBatch(ctx context.Context, categories []models.Category) error
	UpdateBatch(ctx context.Context, categories []models.Category) error
	
	// Statistics
	GetStats(ctx context.Context, filters *models.CategoryFilter) (*CategoryStats, error)
	UpdatePaperCount(ctx context.Context, categoryID string, count int) error
	RecalculatePaperCounts(ctx context.Context) error
}

// SearchRepository defines the interface for search history and cache operations
type SearchRepository interface {
	// Search history
	CreateSearchHistory(ctx context.Context, history *models.SearchHistory) error
	GetSearchHistory(ctx context.Context, userID *string, limit, offset int) ([]models.SearchHistory, error)
	GetPopularQueries(ctx context.Context, since time.Time, limit int) ([]QueryStats, error)
	GetUserSearchStats(ctx context.Context, userID string) (*UserSearchStats, error)
	
	// Search cache
	GetCachedSearch(ctx context.Context, queryHash string) (*models.SearchCache, error)
	SetSearchCache(ctx context.Context, cache *models.SearchCache) error
	InvalidateCache(ctx context.Context, pattern string) error
	CleanupExpiredCache(ctx context.Context) error
	GetCacheStats(ctx context.Context) (*CacheStats, error)
	
	// Search suggestions
	GetSearchSuggestions(ctx context.Context, query string, limit int) ([]models.SearchSuggestion, error)
	UpdateSearchSuggestions(ctx context.Context, query string, resultCount int) error
	
	// Analytics
	GetSearchAnalytics(ctx context.Context, from, to time.Time) (*SearchAnalytics, error)
	GetProviderPerformance(ctx context.Context, provider string, from, to time.Time) (*ProviderPerformance, error)
}

// Transaction defines the interface for database transactions
type Transaction interface {
	// Transaction management
	Begin(ctx context.Context) (Transaction, error)
	Commit() error
	Rollback() error
	
	// Repository access within transaction
	Papers() PaperRepository
	Authors() AuthorRepository
	Categories() CategoryRepository
	Search() SearchRepository
}

// Repository aggregates all repository interfaces
type Repository interface {
	// Individual repositories
	Papers() PaperRepository
	Authors() AuthorRepository
	Categories() CategoryRepository
	Search() SearchRepository
	
	// Transaction management
	Transaction(ctx context.Context, fn func(Transaction) error) error
	
	// Health and maintenance
	Ping(ctx context.Context) error
	Close() error
	GetStats() (map[string]interface{}, error)
}

// Statistics structures

// PaperStats represents paper statistics
type PaperStats struct {
	TotalCount       int64   `json:"total_count"`
	PublishedCount   int64   `json:"published_count"`
	UnpublishedCount int64   `json:"unpublished_count"`
	AvgCitations     float64 `json:"avg_citations"`
	AvgQualityScore  float64 `json:"avg_quality_score"`
	TopCategories    []CategoryCount `json:"top_categories"`
	TopJournals      []JournalCount  `json:"top_journals"`
	YearDistribution []YearCount     `json:"year_distribution"`
}

// AuthorStats represents author statistics
type AuthorStats struct {
	TotalCount         int64   `json:"total_count"`
	AvgPaperCount      float64 `json:"avg_paper_count"`
	AvgCitationCount   float64 `json:"avg_citation_count"`
	AvgHIndex          float64 `json:"avg_h_index"`
	TopAffiliations    []AffiliationCount `json:"top_affiliations"`
	TopResearchAreas   []ResearchAreaCount `json:"top_research_areas"`
}

// CategoryStats represents category statistics
type CategoryStats struct {
	TotalCount      int64 `json:"total_count"`
	ActiveCount     int64 `json:"active_count"`
	AvgPaperCount   float64 `json:"avg_paper_count"`
	TopCategories   []CategoryCount `json:"top_categories"`
	SourceBreakdown []SourceCount   `json:"source_breakdown"`
}

// ProcessingStats represents paper processing statistics
type ProcessingStats struct {
	TotalCount      int64 `json:"total_count"`
	PendingCount    int64 `json:"pending_count"`
	ProcessingCount int64 `json:"processing_count"`
	CompletedCount  int64 `json:"completed_count"`
	FailedCount     int64 `json:"failed_count"`
}

// QueryStats represents search query statistics
type QueryStats struct {
	Query       string    `json:"query"`
	Count       int64     `json:"count"`
	LastQueried time.Time `json:"last_queried"`
}

// UserSearchStats represents user search statistics
type UserSearchStats struct {
	UserID       string    `json:"user_id"`
	TotalQueries int64     `json:"total_queries"`
	UniqueQueries int64    `json:"unique_queries"`
	LastSearch   time.Time `json:"last_search"`
	TopQueries   []QueryStats `json:"top_queries"`
}

// CacheStats represents search cache statistics
type CacheStats struct {
	TotalEntries   int64   `json:"total_entries"`
	ExpiredEntries int64   `json:"expired_entries"`
	HitRate        float64 `json:"hit_rate"`
	AvgAge         float64 `json:"avg_age_hours"`
	SizeBytes      int64   `json:"size_bytes"`
}

// SearchAnalytics represents search analytics
type SearchAnalytics struct {
	TotalSearches    int64   `json:"total_searches"`
	UniqueQueries    int64   `json:"unique_queries"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	TopQueries       []QueryStats `json:"top_queries"`
	ProviderUsage    []ProviderUsageStats `json:"provider_usage"`
	ErrorRate        float64 `json:"error_rate"`
}

// ProviderPerformance represents provider performance metrics
type ProviderPerformance struct {
	Provider        string  `json:"provider"`
	TotalRequests   int64   `json:"total_requests"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	TotalResults    int64   `json:"total_results"`
	AvgResultsPerRequest float64 `json:"avg_results_per_request"`
}

// Count structures for statistics

type CategoryCount struct {
	CategoryID string `json:"category_id"`
	Name       string `json:"name"`
	Count      int64  `json:"count"`
}

type JournalCount struct {
	Journal string `json:"journal"`
	Count   int64  `json:"count"`
}

type YearCount struct {
	Year  int   `json:"year"`
	Count int64 `json:"count"`
}

type AffiliationCount struct {
	Affiliation string `json:"affiliation"`
	Count       int64  `json:"count"`
}

type ResearchAreaCount struct {
	Area  string `json:"area"`
	Count int64  `json:"count"`
}

type SourceCount struct {
	Source string `json:"source"`
	Count  int64  `json:"count"`
}

type ProviderUsageStats struct {
	Provider string `json:"provider"`
	Usage    int64  `json:"usage"`
}