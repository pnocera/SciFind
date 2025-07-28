package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"scifind-backend/internal/errors"
	"scifind-backend/internal/models"

	"gorm.io/gorm"
)

// paperRepository implements PaperRepository interface
type paperRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewPaperRepository creates a new paper repository
func NewPaperRepository(db *gorm.DB, logger *slog.Logger) PaperRepository {
	return &paperRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new paper
func (r *paperRepository) Create(ctx context.Context, paper *models.Paper) error {
	if err := r.db.WithContext(ctx).Create(paper).Error; err != nil {
		if errors.IsDuplicateKeyError(err) {
			return errors.NewDuplicateError("Paper already exists", "paper")
		}
		return errors.NewDatabaseError("create_paper", err)
	}
	return nil
}

// GetByID retrieves a paper by ID
func (r *paperRepository) GetByID(ctx context.Context, id string) (*models.Paper, error) {
	var paper models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		First(&paper, "id = ?", id).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Paper not found", "paper")
		}
		return nil, errors.NewDatabaseError("get_paper", err)
	}
	return &paper, nil
}

// GetByDOI retrieves a paper by DOI
func (r *paperRepository) GetByDOI(ctx context.Context, doi string) (*models.Paper, error) {
	var paper models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		First(&paper, "doi = ?", doi).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Paper not found", "doi")
		}
		return nil, errors.NewDatabaseError("get_paper_by_doi", err)
	}
	return &paper, nil
}

// GetByArxivID retrieves a paper by ArXiv ID
func (r *paperRepository) GetByArxivID(ctx context.Context, arxivID string) (*models.Paper, error) {
	var paper models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		First(&paper, "arxiv_id = ?", arxivID).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Paper not found", "arxiv_id")
		}
		return nil, errors.NewDatabaseError("get_paper_by_arxiv", err)
	}
	return &paper, nil
}

// Update updates a paper
func (r *paperRepository) Update(ctx context.Context, paper *models.Paper) error {
	result := r.db.WithContext(ctx).Save(paper)
	if result.Error != nil {
		return errors.NewDatabaseError("update_paper", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Paper not found", "paper")
	}
	return nil
}

// Delete deletes a paper
func (r *paperRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Paper{}, "id = ?", id)
	if result.Error != nil {
		return errors.NewDatabaseError("delete_paper", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Paper not found", "paper")
	}
	return nil
}

// Search searches for papers with filters and sorting
func (r *paperRepository) Search(ctx context.Context, query string, filters *models.PaperFilter, sort *models.PaperSort, limit, offset int) ([]models.Paper, int64, error) {
	db := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories")
	
	// Apply search query
	if query != "" {
		db = db.Where("to_tsvector('english', title || ' ' || COALESCE(abstract, '')) @@ plainto_tsquery('english', ?)", query)
	}
	
	// Apply filters
	db = r.applyPaperFilters(db, filters)
	
	// Count total results
	var total int64
	if err := db.Model(&models.Paper{}).Count(&total).Error; err != nil {
		return nil, 0, errors.NewDatabaseError("count_papers", err)
	}
	
	// Apply sorting
	db = r.applyPaperSorting(db, sort)
	
	// Apply pagination
	var papers []models.Paper
	err := db.Limit(limit).Offset(offset).Find(&papers).Error
	if err != nil {
		return nil, 0, errors.NewDatabaseError("search_papers", err)
	}
	
	return papers, total, nil
}

// SearchByVector searches for papers using vector similarity
func (r *paperRepository) SearchByVector(ctx context.Context, embedding []float32, filters *models.PaperFilter, limit int) ([]models.Paper, error) {
	// Convert float32 slice to PostgreSQL array format
	embeddingStr := fmt.Sprintf("[%s]", strings.Join(func() []string {
		strs := make([]string, len(embedding))
		for i, v := range embedding {
			strs[i] = fmt.Sprintf("%f", v)
		}
		return strs
	}(), ","))
	
	db := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories")
	
	// Apply filters
	db = r.applyPaperFilters(db, filters)
	
	// Order by vector similarity (cosine distance)
	db = db.Order(fmt.Sprintf("embedding <=> '%s'", embeddingStr))
	
	var papers []models.Paper
	err := db.Limit(limit).Find(&papers).Error
	if err != nil {
		return nil, errors.NewDatabaseError("search_papers_vector", err)
	}
	
	return papers, nil
}

// SearchFullText performs full-text search on papers
func (r *paperRepository) SearchFullText(ctx context.Context, query string, filters *models.PaperFilter, limit, offset int) ([]models.Paper, int64, error) {
	db := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories")
	
	// Full-text search with ranking
	if query != "" {
		db = db.Where("to_tsvector('english', title || ' ' || COALESCE(abstract, '') || ' ' || COALESCE(full_text, '')) @@ plainto_tsquery('english', ?)", query).
			Order("ts_rank(to_tsvector('english', title || ' ' || COALESCE(abstract, '') || ' ' || COALESCE(full_text, '')), plainto_tsquery('english', '" + query + "')) DESC")
	}
	
	// Apply filters
	db = r.applyPaperFilters(db, filters)
	
	// Count total results
	var total int64
	if err := db.Model(&models.Paper{}).Count(&total).Error; err != nil {
		return nil, 0, errors.NewDatabaseError("count_papers_fulltext", err)
	}
	
	// Get results
	var papers []models.Paper
	err := db.Limit(limit).Offset(offset).Find(&papers).Error
	if err != nil {
		return nil, 0, errors.NewDatabaseError("search_papers_fulltext", err)
	}
	
	return papers, total, nil
}

// CreateBatch creates multiple papers in a batch
func (r *paperRepository) CreateBatch(ctx context.Context, papers []models.Paper) error {
	if len(papers) == 0 {
		return nil
	}
	
	batchSize := 100
	for i := 0; i < len(papers); i += batchSize {
		end := i + batchSize
		if end > len(papers) {
			end = len(papers)
		}
		
		batch := papers[i:end]
		if err := r.db.WithContext(ctx).CreateInBatches(batch, len(batch)).Error; err != nil {
			return errors.NewDatabaseError("create_papers_batch", err)
		}
	}
	
	return nil
}

// UpdateBatch updates multiple papers in a batch
func (r *paperRepository) UpdateBatch(ctx context.Context, papers []models.Paper) error {
	if len(papers) == 0 {
		return nil
	}
	
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	for _, paper := range papers {
		if err := tx.Save(&paper).Error; err != nil {
			tx.Rollback()
			return errors.NewDatabaseError("update_papers_batch", err)
		}
	}
	
	if err := tx.Commit().Error; err != nil {
		return errors.NewDatabaseError("commit_papers_batch", err)
	}
	
	return nil
}

// GetStats returns paper statistics
func (r *paperRepository) GetStats(ctx context.Context, filters *models.PaperFilter) (*PaperStats, error) {
	var stats PaperStats
	
	db := r.db.WithContext(ctx).Model(&models.Paper{})
	db = r.applyPaperFilters(db, filters)
	
	// Total count
	if err := db.Count(&stats.TotalCount).Error; err != nil {
		return nil, errors.NewDatabaseError("get_paper_stats_total", err)
	}
	
	// Published vs unpublished count
	if err := db.Where("published_at IS NOT NULL").Count(&stats.PublishedCount).Error; err != nil {
		return nil, errors.NewDatabaseError("get_paper_stats_published", err)
	}
	stats.UnpublishedCount = stats.TotalCount - stats.PublishedCount
	
	// Average citations and quality score
	var avgStats struct {
		AvgCitations    float64
		AvgQualityScore float64
	}
	if err := db.Select("AVG(citation_count) as avg_citations, AVG(quality_score) as avg_quality_score").Scan(&avgStats).Error; err != nil {
		return nil, errors.NewDatabaseError("get_paper_stats_avg", err)
	}
	stats.AvgCitations = avgStats.AvgCitations
	stats.AvgQualityScore = avgStats.AvgQualityScore
	
	// Top categories - this would need a more complex query with joins
	// For now, returning empty slice
	stats.TopCategories = []CategoryCount{}
	stats.TopJournals = []JournalCount{}
	stats.YearDistribution = []YearCount{}
	
	return &stats, nil
}

// GetTrendingPapers returns trending papers since a given time
func (r *paperRepository) GetTrendingPapers(ctx context.Context, since time.Time, limit int) ([]models.Paper, error) {
	var papers []models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		Where("created_at >= ?", since).
		Order("citation_count DESC, quality_score DESC").
		Limit(limit).
		Find(&papers).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_trending_papers", err)
	}
	
	return papers, nil
}

// GetPopularPapers returns most popular papers by citation count
func (r *paperRepository) GetPopularPapers(ctx context.Context, limit int) ([]models.Paper, error) {
	var papers []models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		Order("citation_count DESC, quality_score DESC").
		Limit(limit).
		Find(&papers).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_popular_papers", err)
	}
	
	return papers, nil
}

// GetPendingPapers returns papers with pending processing state
func (r *paperRepository) GetPendingPapers(ctx context.Context, limit int) ([]models.Paper, error) {
	var papers []models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		Where("processing_state = ?", "pending").
		Order("created_at ASC").
		Limit(limit).
		Find(&papers).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_pending_papers", err)
	}
	
	return papers, nil
}

// UpdateProcessingState updates the processing state of a paper
func (r *paperRepository) UpdateProcessingState(ctx context.Context, paperID string, state string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Paper{}).
		Where("id = ?", paperID).
		Update("processing_state", state)
	
	if result.Error != nil {
		return errors.NewDatabaseError("update_processing_state", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Paper not found", "paper")
	}
	
	return nil
}

// GetProcessingStats returns processing statistics
func (r *paperRepository) GetProcessingStats(ctx context.Context) (*ProcessingStats, error) {
	var stats ProcessingStats
	
	// Total count
	if err := r.db.WithContext(ctx).Model(&models.Paper{}).Count(&stats.TotalCount).Error; err != nil {
		return nil, errors.NewDatabaseError("get_processing_stats_total", err)
	}
	
	// Count by processing state
	var stateCounts []struct {
		ProcessingState string
		Count           int64
	}
	
	err := r.db.WithContext(ctx).
		Model(&models.Paper{}).
		Select("processing_state, COUNT(*) as count").
		Group("processing_state").
		Scan(&stateCounts).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_processing_stats_states", err)
	}
	
	for _, sc := range stateCounts {
		switch sc.ProcessingState {
		case "pending":
			stats.PendingCount = sc.Count
		case "processing":
			stats.ProcessingCount = sc.Count
		case "completed":
			stats.CompletedCount = sc.Count
		case "failed":
			stats.FailedCount = sc.Count
		}
	}
	
	return &stats, nil
}

// GetAuthorPapers returns papers by a specific author
func (r *paperRepository) GetAuthorPapers(ctx context.Context, authorID string, limit, offset int) ([]models.Paper, error) {
	var papers []models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		Joins("JOIN paper_authors ON papers.id = paper_authors.paper_id").
		Where("paper_authors.author_id = ?", authorID).
		Order("published_at DESC NULLS LAST").
		Limit(limit).
		Offset(offset).
		Find(&papers).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_author_papers", err)
	}
	
	return papers, nil
}

// GetCategoryPapers returns papers in a specific category
func (r *paperRepository) GetCategoryPapers(ctx context.Context, categoryID string, limit, offset int) ([]models.Paper, error) {
	var papers []models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		Joins("JOIN paper_categories ON papers.id = paper_categories.paper_id").
		Where("paper_categories.category_id = ?", categoryID).
		Order("quality_score DESC, citation_count DESC").
		Limit(limit).
		Offset(offset).
		Find(&papers).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_category_papers", err)
	}
	
	return papers, nil
}

// GetSimilarPapers returns papers similar to a given paper
func (r *paperRepository) GetSimilarPapers(ctx context.Context, paperID string, limit int) ([]models.Paper, error) {
	// Get the paper's embedding and categories
	var paper models.Paper
	err := r.db.WithContext(ctx).
		Preload("Categories").
		First(&paper, "id = ?", paperID).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Paper not found", "paper")
		}
		return nil, errors.NewDatabaseError("get_paper_for_similarity", err)
	}
	
	db := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		Where("id != ?", paperID)
	
	// If paper has categories, find papers in same categories
	if len(paper.Categories) > 0 {
		categoryIDs := make([]string, len(paper.Categories))
		for i, cat := range paper.Categories {
			categoryIDs[i] = cat.ID
		}
		
		db = db.Joins("JOIN paper_categories ON papers.id = paper_categories.paper_id").
			Where("paper_categories.category_id IN ?", categoryIDs)
	}
	
	var papers []models.Paper
	err = db.Order("quality_score DESC, citation_count DESC").
		Limit(limit).
		Find(&papers).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_similar_papers", err)
	}
	
	return papers, nil
}

// GetCitations returns papers that cite the given paper
func (r *paperRepository) GetCitations(ctx context.Context, paperID string) ([]models.Paper, error) {
	var papers []models.Paper
	err := r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		Where("? = ANY(references)", paperID).
		Order("published_at DESC").
		Find(&papers).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_citations", err)
	}
	
	return papers, nil
}

// GetReferences returns papers referenced by the given paper
func (r *paperRepository) GetReferences(ctx context.Context, paperID string) ([]models.Paper, error) {
	// Get the paper's references
	var paper models.Paper
	err := r.db.WithContext(ctx).
		Select("references").
		First(&paper, "id = ?", paperID).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Paper not found", "paper")
		}
		return nil, errors.NewDatabaseError("get_paper_references", err)
	}
	
	if len(paper.References) == 0 {
		return []models.Paper{}, nil
	}
	
	var papers []models.Paper
	err = r.db.WithContext(ctx).
		Preload("Authors").
		Preload("Categories").
		Where("id IN ?", paper.References).
		Find(&papers).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_references", err)
	}
	
	return papers, nil
}

// UpdateCitationCount updates the citation count of a paper
func (r *paperRepository) UpdateCitationCount(ctx context.Context, paperID string, count int) error {
	result := r.db.WithContext(ctx).
		Model(&models.Paper{}).
		Where("id = ?", paperID).
		Update("citation_count", count)
	
	if result.Error != nil {
		return errors.NewDatabaseError("update_citation_count", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Paper not found", "paper")
	}
	
	return nil
}

// Helper methods

// applyPaperFilters applies filters to a GORM query
func (r *paperRepository) applyPaperFilters(db *gorm.DB, filters *models.PaperFilter) *gorm.DB {
	if filters == nil {
		return db
	}
	
	if len(filters.IDs) > 0 {
		db = db.Where("id IN ?", filters.IDs)
	}
	
	if len(filters.DOIs) > 0 {
		db = db.Where("doi IN ?", filters.DOIs)
	}
	
	if len(filters.ArxivIDs) > 0 {
		db = db.Where("arxiv_id IN ?", filters.ArxivIDs)
	}
	
	if filters.Title != "" {
		db = db.Where("title ILIKE ?", "%"+filters.Title+"%")
	}
	
	if filters.Journal != "" {
		db = db.Where("journal ILIKE ?", "%"+filters.Journal+"%")
	}
	
	if filters.Language != "" {
		db = db.Where("language = ?", filters.Language)
	}
	
	if filters.SourceProvider != "" {
		db = db.Where("source_provider = ?", filters.SourceProvider)
	}
	
	if filters.MinCitations != nil {
		db = db.Where("citation_count >= ?", *filters.MinCitations)
	}
	
	if filters.MaxCitations != nil {
		db = db.Where("citation_count <= ?", *filters.MaxCitations)
	}
	
	if filters.MinQuality != nil {
		db = db.Where("quality_score >= ?", *filters.MinQuality)
	}
	
	if filters.MaxQuality != nil {
		db = db.Where("quality_score <= ?", *filters.MaxQuality)
	}
	
	if filters.PublishedFrom != nil {
		db = db.Where("published_at >= ?", *filters.PublishedFrom)
	}
	
	if filters.PublishedTo != nil {
		db = db.Where("published_at <= ?", *filters.PublishedTo)
	}
	
	if filters.CreatedFrom != nil {
		db = db.Where("created_at >= ?", *filters.CreatedFrom)
	}
	
	if filters.CreatedTo != nil {
		db = db.Where("created_at <= ?", *filters.CreatedTo)
	}
	
	if len(filters.States) > 0 {
		db = db.Where("processing_state IN ?", filters.States)
	}
	
	if filters.HasFullText != nil {
		if *filters.HasFullText {
			db = db.Where("full_text IS NOT NULL AND full_text != ''")
		} else {
			db = db.Where("full_text IS NULL OR full_text = ''")
		}
	}
	
	if filters.HasPDF != nil {
		if *filters.HasPDF {
			db = db.Where("pdf_url IS NOT NULL AND pdf_url != ''")
		} else {
			db = db.Where("pdf_url IS NULL OR pdf_url = ''")
		}
	}
	
	return db
}

// applyPaperSorting applies sorting to a GORM query
func (r *paperRepository) applyPaperSorting(db *gorm.DB, sort *models.PaperSort) *gorm.DB {
	if sort == nil {
		sort = &models.PaperSort{Field: "created_at", Order: "desc"}
	}
	
	orderClause := fmt.Sprintf("%s %s", sort.Field, strings.ToUpper(sort.Order))
	
	// Handle special sorting cases
	switch sort.Field {
	case "published_at":
		orderClause = fmt.Sprintf("%s %s NULLS LAST", sort.Field, strings.ToUpper(sort.Order))
	case "relevance":
		// This would be handled in the search query with ts_rank
		orderClause = "quality_score DESC, citation_count DESC"
	}
	
	return db.Order(orderClause)
}