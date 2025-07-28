package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"scifind-backend/internal/models"
	"scifind-backend/internal/repository"
)

// MockPaperRepository is a mock implementation of PaperRepository
type MockPaperRepository struct {
	mock.Mock
}

func (m *MockPaperRepository) Create(ctx context.Context, paper *models.Paper) error {
	args := m.Called(ctx, paper)
	return args.Error(0)
}

func (m *MockPaperRepository) GetByID(ctx context.Context, id string) (*models.Paper, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Paper), args.Error(1)
}

func (m *MockPaperRepository) GetByDOI(ctx context.Context, doi string) (*models.Paper, error) {
	args := m.Called(ctx, doi)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Paper), args.Error(1)
}

func (m *MockPaperRepository) GetByArxivID(ctx context.Context, arxivID string) (*models.Paper, error) {
	args := m.Called(ctx, arxivID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Paper), args.Error(1)
}

func (m *MockPaperRepository) Update(ctx context.Context, paper *models.Paper) error {
	args := m.Called(ctx, paper)
	return args.Error(0)
}

func (m *MockPaperRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPaperRepository) Search(ctx context.Context, query string, filters *models.PaperFilter, sort *models.PaperSort, limit, offset int) ([]models.Paper, int64, error) {
	args := m.Called(ctx, query, filters, sort, limit, offset)
	return args.Get(0).([]models.Paper), args.Get(1).(int64), args.Error(2)
}

func (m *MockPaperRepository) SearchByVector(ctx context.Context, embedding []float32, filters *models.PaperFilter, limit int) ([]models.Paper, error) {
	args := m.Called(ctx, embedding, filters, limit)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) SearchFullText(ctx context.Context, query string, filters *models.PaperFilter, limit, offset int) ([]models.Paper, int64, error) {
	args := m.Called(ctx, query, filters, limit, offset)
	return args.Get(0).([]models.Paper), args.Get(1).(int64), args.Error(2)
}

func (m *MockPaperRepository) CreateBatch(ctx context.Context, papers []models.Paper) error {
	args := m.Called(ctx, papers)
	return args.Error(0)
}

func (m *MockPaperRepository) UpdateBatch(ctx context.Context, papers []models.Paper) error {
	args := m.Called(ctx, papers)
	return args.Error(0)
}

func (m *MockPaperRepository) GetStats(ctx context.Context, filters *models.PaperFilter) (*repository.PaperStats, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PaperStats), args.Error(1)
}

func (m *MockPaperRepository) GetTrendingPapers(ctx context.Context, since time.Time, limit int) ([]models.Paper, error) {
	args := m.Called(ctx, since, limit)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) GetPopularPapers(ctx context.Context, limit int) ([]models.Paper, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) GetPendingPapers(ctx context.Context, limit int) ([]models.Paper, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) UpdateProcessingState(ctx context.Context, paperID string, state string) error {
	args := m.Called(ctx, paperID, state)
	return args.Error(0)
}

func (m *MockPaperRepository) GetProcessingStats(ctx context.Context) (*repository.ProcessingStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ProcessingStats), args.Error(1)
}

func (m *MockPaperRepository) GetAuthorPapers(ctx context.Context, authorID string, limit, offset int) ([]models.Paper, error) {
	args := m.Called(ctx, authorID, limit, offset)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) GetCategoryPapers(ctx context.Context, categoryID string, limit, offset int) ([]models.Paper, error) {
	args := m.Called(ctx, categoryID, limit, offset)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) GetSimilarPapers(ctx context.Context, paperID string, limit int) ([]models.Paper, error) {
	args := m.Called(ctx, paperID, limit)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) GetCitations(ctx context.Context, paperID string) ([]models.Paper, error) {
	args := m.Called(ctx, paperID)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) GetReferences(ctx context.Context, paperID string) ([]models.Paper, error) {
	args := m.Called(ctx, paperID)
	return args.Get(0).([]models.Paper), args.Error(1)
}

func (m *MockPaperRepository) UpdateCitationCount(ctx context.Context, paperID string, count int) error {
	args := m.Called(ctx, paperID, count)
	return args.Error(0)
}

// MockAuthorRepository is a mock implementation of AuthorRepository
type MockAuthorRepository struct {
	mock.Mock
}

func (m *MockAuthorRepository) Create(ctx context.Context, author *models.Author) error {
	args := m.Called(ctx, author)
	return args.Error(0)
}

func (m *MockAuthorRepository) GetByID(ctx context.Context, id string) (*models.Author, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Author), args.Error(1)
}

func (m *MockAuthorRepository) GetByEmail(ctx context.Context, email string) (*models.Author, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Author), args.Error(1)
}

func (m *MockAuthorRepository) GetByORCID(ctx context.Context, orcid string) (*models.Author, error) {
	args := m.Called(ctx, orcid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Author), args.Error(1)
}

func (m *MockAuthorRepository) Update(ctx context.Context, author *models.Author) error {
	args := m.Called(ctx, author)
	return args.Error(0)
}

func (m *MockAuthorRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAuthorRepository) Search(ctx context.Context, query string, filters *models.AuthorFilter, sort *models.AuthorSort, limit, offset int) ([]models.Author, int64, error) {
	args := m.Called(ctx, query, filters, sort, limit, offset)
	return args.Get(0).([]models.Author), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuthorRepository) SearchByName(ctx context.Context, name string, limit int) ([]models.Author, error) {
	args := m.Called(ctx, name, limit)
	return args.Get(0).([]models.Author), args.Error(1)
}

func (m *MockAuthorRepository) SearchByAffiliation(ctx context.Context, affiliation string, limit int) ([]models.Author, error) {
	args := m.Called(ctx, affiliation, limit)
	return args.Get(0).([]models.Author), args.Error(1)
}

func (m *MockAuthorRepository) CreateBatch(ctx context.Context, authors []models.Author) error {
	args := m.Called(ctx, authors)
	return args.Error(0)
}

func (m *MockAuthorRepository) UpdateBatch(ctx context.Context, authors []models.Author) error {
	args := m.Called(ctx, authors)
	return args.Error(0)
}

func (m *MockAuthorRepository) GetStats(ctx context.Context, filters *models.AuthorFilter) (*repository.AuthorStats, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.AuthorStats), args.Error(1)
}

func (m *MockAuthorRepository) GetTopAuthors(ctx context.Context, criteria string, limit int) ([]models.Author, error) {
	args := m.Called(ctx, criteria, limit)
	return args.Get(0).([]models.Author), args.Error(1)
}

func (m *MockAuthorRepository) GetProductiveAuthors(ctx context.Context, minPapers int, limit int) ([]models.Author, error) {
	args := m.Called(ctx, minPapers, limit)
	return args.Get(0).([]models.Author), args.Error(1)
}

func (m *MockAuthorRepository) GetCollaborators(ctx context.Context, authorID string, limit int) ([]models.Author, error) {
	args := m.Called(ctx, authorID, limit)
	return args.Get(0).([]models.Author), args.Error(1)
}

func (m *MockAuthorRepository) GetAuthorsByPaper(ctx context.Context, paperID string) ([]models.Author, error) {
	args := m.Called(ctx, paperID)
	return args.Get(0).([]models.Author), args.Error(1)
}

func (m *MockAuthorRepository) UpdateMetrics(ctx context.Context, authorID string, paperCount, citationCount, hIndex int) error {
	args := m.Called(ctx, authorID, paperCount, citationCount, hIndex)
	return args.Error(0)
}

func (m *MockAuthorRepository) RecalculateMetrics(ctx context.Context, authorID string) error {
	args := m.Called(ctx, authorID)
	return args.Error(0)
}

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
	paperRepo    *MockPaperRepository
	authorRepo   *MockAuthorRepository
	categoryRepo *MockCategoryRepository
	searchRepo   *MockSearchRepository
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		paperRepo:    &MockPaperRepository{},
		authorRepo:   &MockAuthorRepository{},
		categoryRepo: &MockCategoryRepository{},
		searchRepo:   &MockSearchRepository{},
	}
}

func (m *MockRepository) Papers() repository.PaperRepository {
	return m.paperRepo
}

func (m *MockRepository) Authors() repository.AuthorRepository {
	return m.authorRepo
}

func (m *MockRepository) Categories() repository.CategoryRepository {
	return m.categoryRepo
}

func (m *MockRepository) Search() repository.SearchRepository {
	return m.searchRepo
}

func (m *MockRepository) Transaction(ctx context.Context, fn func(repository.Transaction) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockRepository) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRepository) GetStats() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// GetMockPaperRepo returns the mock paper repository for setting expectations
func (m *MockRepository) GetMockPaperRepo() *MockPaperRepository {
	return m.paperRepo
}

// GetMockAuthorRepo returns the mock author repository for setting expectations
func (m *MockRepository) GetMockAuthorRepo() *MockAuthorRepository {
	return m.authorRepo
}

// GetMockCategoryRepo returns the mock category repository for setting expectations  
func (m *MockRepository) GetMockCategoryRepo() *MockCategoryRepository {
	return m.categoryRepo
}

// GetMockSearchRepo returns the mock search repository for setting expectations
func (m *MockRepository) GetMockSearchRepo() *MockSearchRepository {
	return m.searchRepo
}

// MockCategoryRepository is a mock implementation of CategoryRepository
type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Create(ctx context.Context, category *models.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetByID(ctx context.Context, id string) (*models.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetBySourceCode(ctx context.Context, source, sourceCode string) (*models.Category, error) {
	args := m.Called(ctx, source, sourceCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Category), args.Error(1)
}

func (m *MockCategoryRepository) Update(ctx context.Context, category *models.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func (m *MockCategoryRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetRootCategories(ctx context.Context, source string) ([]models.Category, error) {
	args := m.Called(ctx, source)
	return args.Get(0).([]models.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetChildren(ctx context.Context, parentID string) ([]models.Category, error) {
	args := m.Called(ctx, parentID)
	return args.Get(0).([]models.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetParent(ctx context.Context, categoryID string) (*models.Category, error) {
	args := m.Called(ctx, categoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetAncestors(ctx context.Context, categoryID string) ([]models.Category, error) {
	args := m.Called(ctx, categoryID)
	return args.Get(0).([]models.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetDescendants(ctx context.Context, categoryID string) ([]models.Category, error) {
	args := m.Called(ctx, categoryID)
	return args.Get(0).([]models.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetCategoryTree(ctx context.Context, source string) ([]models.CategoryTree, error) {
	args := m.Called(ctx, source)
	return args.Get(0).([]models.CategoryTree), args.Error(1)
}

func (m *MockCategoryRepository) Search(ctx context.Context, query string, filters *models.CategoryFilter, sort *models.CategorySort, limit, offset int) ([]models.Category, int64, error) {
	args := m.Called(ctx, query, filters, sort, limit, offset)
	return args.Get(0).([]models.Category), args.Get(1).(int64), args.Error(2)
}

func (m *MockCategoryRepository) GetActiveCategories(ctx context.Context, source string) ([]models.Category, error) {
	args := m.Called(ctx, source)
	return args.Get(0).([]models.Category), args.Error(1)
}

func (m *MockCategoryRepository) GetPopularCategories(ctx context.Context, limit int) ([]models.Category, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]models.Category), args.Error(1)
}

func (m *MockCategoryRepository) CreateBatch(ctx context.Context, categories []models.Category) error {
	args := m.Called(ctx, categories)
	return args.Error(0)
}

func (m *MockCategoryRepository) UpdateBatch(ctx context.Context, categories []models.Category) error {
	args := m.Called(ctx, categories)
	return args.Error(0)
}

func (m *MockCategoryRepository) GetStats(ctx context.Context, filters *models.CategoryFilter) (*repository.CategoryStats, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.CategoryStats), args.Error(1)
}

func (m *MockCategoryRepository) UpdatePaperCount(ctx context.Context, categoryID string, count int) error {
	args := m.Called(ctx, categoryID, count)
	return args.Error(0)
}

func (m *MockCategoryRepository) RecalculatePaperCounts(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockSearchRepository is a mock implementation of SearchRepository
type MockSearchRepository struct {
	mock.Mock
}

func (m *MockSearchRepository) CreateSearchHistory(ctx context.Context, history *models.SearchHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockSearchRepository) GetSearchHistory(ctx context.Context, userID *string, limit, offset int) ([]models.SearchHistory, error) {
	args := m.Called(ctx, userID, limit, offset)
	return args.Get(0).([]models.SearchHistory), args.Error(1)
}

func (m *MockSearchRepository) GetPopularQueries(ctx context.Context, since time.Time, limit int) ([]repository.QueryStats, error) {
	args := m.Called(ctx, since, limit)
	return args.Get(0).([]repository.QueryStats), args.Error(1)
}

func (m *MockSearchRepository) GetUserSearchStats(ctx context.Context, userID string) (*repository.UserSearchStats, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.UserSearchStats), args.Error(1)
}

func (m *MockSearchRepository) GetCachedSearch(ctx context.Context, queryHash string) (*models.SearchCache, error) {
	args := m.Called(ctx, queryHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SearchCache), args.Error(1)
}

func (m *MockSearchRepository) SetSearchCache(ctx context.Context, cache *models.SearchCache) error {
	args := m.Called(ctx, cache)
	return args.Error(0)
}

func (m *MockSearchRepository) InvalidateCache(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockSearchRepository) CleanupExpiredCache(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSearchRepository) GetCacheStats(ctx context.Context) (*repository.CacheStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.CacheStats), args.Error(1)
}

func (m *MockSearchRepository) GetSearchSuggestions(ctx context.Context, query string, limit int) ([]models.SearchSuggestion, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]models.SearchSuggestion), args.Error(1)
}

func (m *MockSearchRepository) UpdateSearchSuggestions(ctx context.Context, query string, resultCount int) error {
	args := m.Called(ctx, query, resultCount)
	return args.Error(0)
}

func (m *MockSearchRepository) GetSearchAnalytics(ctx context.Context, from, to time.Time) (*repository.SearchAnalytics, error) {
	args := m.Called(ctx, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.SearchAnalytics), args.Error(1)
}

func (m *MockSearchRepository) GetProviderPerformance(ctx context.Context, provider string, from, to time.Time) (*repository.ProviderPerformance, error) {
	args := m.Called(ctx, provider, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ProviderPerformance), args.Error(1)
}