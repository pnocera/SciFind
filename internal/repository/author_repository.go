package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"scifind-backend/internal/errors"
	"scifind-backend/internal/models"

	"gorm.io/gorm"
)

// authorRepository implements AuthorRepository interface
type authorRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewAuthorRepository creates a new author repository
func NewAuthorRepository(db *gorm.DB, logger *slog.Logger) AuthorRepository {
	return &authorRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new author
func (r *authorRepository) Create(ctx context.Context, author *models.Author) error {
	if err := r.db.WithContext(ctx).Create(author).Error; err != nil {
		if errors.IsDuplicateKeyError(err) {
			return errors.NewDuplicateError("Author already exists", "author")
		}
		return errors.NewDatabaseError("create_author", err)
	}
	return nil
}

// GetByID retrieves an author by ID
func (r *authorRepository) GetByID(ctx context.Context, id string) (*models.Author, error) {
	var author models.Author
	err := r.db.WithContext(ctx).
		Preload("Papers").
		First(&author, "id = ?", id).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Author not found", "author")
		}
		return nil, errors.NewDatabaseError("get_author", err)
	}
	return &author, nil
}

// GetByEmail retrieves an author by email
func (r *authorRepository) GetByEmail(ctx context.Context, email string) (*models.Author, error) {
	var author models.Author
	err := r.db.WithContext(ctx).
		Preload("Papers").
		First(&author, "email = ?", email).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Author not found", "email")
		}
		return nil, errors.NewDatabaseError("get_author_by_email", err)
	}
	return &author, nil
}

// GetByORCID retrieves an author by ORCID
func (r *authorRepository) GetByORCID(ctx context.Context, orcid string) (*models.Author, error) {
	var author models.Author
	err := r.db.WithContext(ctx).
		Preload("Papers").
		First(&author, "orcid = ?", orcid).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Author not found", "orcid")
		}
		return nil, errors.NewDatabaseError("get_author_by_orcid", err)
	}
	return &author, nil
}

// Update updates an author
func (r *authorRepository) Update(ctx context.Context, author *models.Author) error {
	result := r.db.WithContext(ctx).Save(author)
	if result.Error != nil {
		return errors.NewDatabaseError("update_author", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Author not found", "author")
	}
	return nil
}

// Delete deletes an author
func (r *authorRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Author{}, "id = ?", id)
	if result.Error != nil {
		return errors.NewDatabaseError("delete_author", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Author not found", "author")
	}
	return nil
}

// Search searches for authors with filters and sorting
func (r *authorRepository) Search(ctx context.Context, query string, filters *models.AuthorFilter, sort *models.AuthorSort, limit, offset int) ([]models.Author, int64, error) {
	db := r.db.WithContext(ctx)
	
	// Apply search query
	if query != "" {
		db = db.Where("name ILIKE ? OR email ILIKE ? OR affiliation ILIKE ?", 
			"%"+query+"%", "%"+query+"%", "%"+query+"%")
	}
	
	// Apply filters
	db = r.applyAuthorFilters(db, filters)
	
	// Count total results
	var total int64
	if err := db.Model(&models.Author{}).Count(&total).Error; err != nil {
		return nil, 0, errors.NewDatabaseError("count_authors", err)
	}
	
	// Apply sorting
	db = r.applyAuthorSorting(db, sort)
	
	// Apply pagination
	var authors []models.Author
	err := db.Limit(limit).Offset(offset).Find(&authors).Error
	if err != nil {
		return nil, 0, errors.NewDatabaseError("search_authors", err)
	}
	
	return authors, total, nil
}

// SearchByName searches for authors by name using fuzzy matching
func (r *authorRepository) SearchByName(ctx context.Context, name string, limit int) ([]models.Author, error) {
	var authors []models.Author
	err := r.db.WithContext(ctx).
		Where("name % ?", name). // PostgreSQL trigram similarity
		Order("similarity(name, '" + name + "') DESC").
		Limit(limit).
		Find(&authors).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("search_authors_by_name", err)
	}
	
	return authors, nil
}

// SearchByAffiliation searches for authors by affiliation
func (r *authorRepository) SearchByAffiliation(ctx context.Context, affiliation string, limit int) ([]models.Author, error) {
	var authors []models.Author
	err := r.db.WithContext(ctx).
		Where("affiliation ILIKE ?", "%"+affiliation+"%").
		Order("h_index DESC, citation_count DESC").
		Limit(limit).
		Find(&authors).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("search_authors_by_affiliation", err)
	}
	
	return authors, nil
}

// CreateBatch creates multiple authors in a batch
func (r *authorRepository) CreateBatch(ctx context.Context, authors []models.Author) error {
	if len(authors) == 0 {
		return nil
	}
	
	batchSize := 100
	for i := 0; i < len(authors); i += batchSize {
		end := i + batchSize
		if end > len(authors) {
			end = len(authors)
		}
		
		batch := authors[i:end]
		if err := r.db.WithContext(ctx).CreateInBatches(batch, len(batch)).Error; err != nil {
			return errors.NewDatabaseError("create_authors_batch", err)
		}
	}
	
	return nil
}

// UpdateBatch updates multiple authors in a batch
func (r *authorRepository) UpdateBatch(ctx context.Context, authors []models.Author) error {
	if len(authors) == 0 {
		return nil
	}
	
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	for _, author := range authors {
		if err := tx.Save(&author).Error; err != nil {
			tx.Rollback()
			return errors.NewDatabaseError("update_authors_batch", err)
		}
	}
	
	if err := tx.Commit().Error; err != nil {
		return errors.NewDatabaseError("commit_authors_batch", err)
	}
	
	return nil
}

// GetStats returns author statistics
func (r *authorRepository) GetStats(ctx context.Context, filters *models.AuthorFilter) (*AuthorStats, error) {
	var stats AuthorStats
	
	db := r.db.WithContext(ctx).Model(&models.Author{})
	db = r.applyAuthorFilters(db, filters)
	
	// Total count
	if err := db.Count(&stats.TotalCount).Error; err != nil {
		return nil, errors.NewDatabaseError("get_author_stats_total", err)
	}
	
	// Average metrics
	var avgStats struct {
		AvgPaperCount    float64
		AvgCitationCount float64
		AvgHIndex        float64
	}
	if err := db.Select("AVG(paper_count) as avg_paper_count, AVG(citation_count) as avg_citation_count, AVG(h_index) as avg_h_index").Scan(&avgStats).Error; err != nil {
		return nil, errors.NewDatabaseError("get_author_stats_avg", err)
	}
	stats.AvgPaperCount = avgStats.AvgPaperCount
	stats.AvgCitationCount = avgStats.AvgCitationCount
	stats.AvgHIndex = avgStats.AvgHIndex
	
	// Top affiliations
	var affiliations []struct {
		Affiliation string
		Count       int64
	}
	err := r.db.WithContext(ctx).
		Model(&models.Author{}).
		Select("affiliation, COUNT(*) as count").
		Where("affiliation IS NOT NULL AND affiliation != ''").
		Group("affiliation").
		Order("count DESC").
		Limit(10).
		Scan(&affiliations).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_author_stats_affiliations", err)
	}
	
	stats.TopAffiliations = make([]AffiliationCount, len(affiliations))
	for i, aff := range affiliations {
		stats.TopAffiliations[i] = AffiliationCount{
			Affiliation: aff.Affiliation,
			Count:       aff.Count,
		}
	}
	
	// For research areas, we'd need to unnest the JSON array
	// For now, returning empty slice
	stats.TopResearchAreas = []ResearchAreaCount{}
	
	return &stats, nil
}

// GetTopAuthors returns top authors by specified criteria
func (r *authorRepository) GetTopAuthors(ctx context.Context, criteria string, limit int) ([]models.Author, error) {
	var authors []models.Author
	
	var orderBy string
	switch criteria {
	case "h_index":
		orderBy = "h_index DESC"
	case "citations":
		orderBy = "citation_count DESC"
	case "papers":
		orderBy = "paper_count DESC"
	default:
		orderBy = "h_index DESC"
	}
	
	err := r.db.WithContext(ctx).
		Order(orderBy).
		Limit(limit).
		Find(&authors).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_top_authors", err)
	}
	
	return authors, nil
}

// GetProductiveAuthors returns authors with at least the specified number of papers
func (r *authorRepository) GetProductiveAuthors(ctx context.Context, minPapers int, limit int) ([]models.Author, error) {
	var authors []models.Author
	err := r.db.WithContext(ctx).
		Where("paper_count >= ?", minPapers).
		Order("h_index DESC, citation_count DESC").
		Limit(limit).
		Find(&authors).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_productive_authors", err)
	}
	
	return authors, nil
}

// GetCollaborators returns authors who have collaborated with the given author
func (r *authorRepository) GetCollaborators(ctx context.Context, authorID string, limit int) ([]models.Author, error) {
	var authors []models.Author
	
	// Find authors who have co-authored papers with the given author
	err := r.db.WithContext(ctx).
		Table("authors").
		Select("DISTINCT authors.*").
		Joins("JOIN paper_authors pa1 ON authors.id = pa1.author_id").
		Joins("JOIN paper_authors pa2 ON pa1.paper_id = pa2.paper_id").
		Where("pa2.author_id = ? AND authors.id != ?", authorID, authorID).
		Order("authors.h_index DESC").
		Limit(limit).
		Find(&authors).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_collaborators", err)
	}
	
	return authors, nil
}

// GetAuthorsByPaper returns all authors of a specific paper
func (r *authorRepository) GetAuthorsByPaper(ctx context.Context, paperID string) ([]models.Author, error) {
	var authors []models.Author
	err := r.db.WithContext(ctx).
		Joins("JOIN paper_authors ON authors.id = paper_authors.author_id").
		Where("paper_authors.paper_id = ?", paperID).
		Find(&authors).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_authors_by_paper", err)
	}
	
	return authors, nil
}

// UpdateMetrics updates author metrics manually
func (r *authorRepository) UpdateMetrics(ctx context.Context, authorID string, paperCount, citationCount, hIndex int) error {
	result := r.db.WithContext(ctx).
		Model(&models.Author{}).
		Where("id = ?", authorID).
		Updates(map[string]interface{}{
			"paper_count":    paperCount,
			"citation_count": citationCount,
			"h_index":        hIndex,
		})
	
	if result.Error != nil {
		return errors.NewDatabaseError("update_author_metrics", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Author not found", "author")
	}
	
	return nil
}

// RecalculateMetrics recalculates author metrics based on their papers
func (r *authorRepository) RecalculateMetrics(ctx context.Context, authorID string) error {
	// Get author with papers
	var author models.Author
	err := r.db.WithContext(ctx).
		Preload("Papers").
		First(&author, "id = ?", authorID).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.NewNotFoundError("Author not found", "author")
		}
		return errors.NewDatabaseError("get_author_for_metrics", err)
	}
	
	// Update metrics based on papers
	author.UpdateMetrics(author.Papers)
	
	// Save updated author
	if err := r.db.WithContext(ctx).Save(&author).Error; err != nil {
		return errors.NewDatabaseError("save_author_metrics", err)
	}
	
	return nil
}

// Helper methods

// applyAuthorFilters applies filters to a GORM query
func (r *authorRepository) applyAuthorFilters(db *gorm.DB, filters *models.AuthorFilter) *gorm.DB {
	if filters == nil {
		return db
	}
	
	if len(filters.IDs) > 0 {
		db = db.Where("id IN ?", filters.IDs)
	}
	
	if filters.Name != "" {
		db = db.Where("name ILIKE ?", "%"+filters.Name+"%")
	}
	
	if filters.Email != "" {
		db = db.Where("email ILIKE ?", "%"+filters.Email+"%")
	}
	
	if filters.Affiliation != "" {
		db = db.Where("affiliation ILIKE ?", "%"+filters.Affiliation+"%")
	}
	
	if filters.ORCID != "" {
		db = db.Where("orcid = ?", filters.ORCID)
	}
	
	if len(filters.ResearchAreas) > 0 {
		// For JSON array contains query
		for _, area := range filters.ResearchAreas {
			db = db.Where("research_areas::jsonb ? ?", area)
		}
	}
	
	if filters.MinPapers != nil {
		db = db.Where("paper_count >= ?", *filters.MinPapers)
	}
	
	if filters.MaxPapers != nil {
		db = db.Where("paper_count <= ?", *filters.MaxPapers)
	}
	
	if filters.MinCitations != nil {
		db = db.Where("citation_count >= ?", *filters.MinCitations)
	}
	
	if filters.MaxCitations != nil {
		db = db.Where("citation_count <= ?", *filters.MaxCitations)
	}
	
	if filters.MinHIndex != nil {
		db = db.Where("h_index >= ?", *filters.MinHIndex)
	}
	
	if filters.MaxHIndex != nil {
		db = db.Where("h_index <= ?", *filters.MaxHIndex)
	}
	
	if filters.CreatedFrom != nil {
		db = db.Where("created_at >= ?", *filters.CreatedFrom)
	}
	
	if filters.CreatedTo != nil {
		db = db.Where("created_at <= ?", *filters.CreatedTo)
	}
	
	return db
}

// applyAuthorSorting applies sorting to a GORM query
func (r *authorRepository) applyAuthorSorting(db *gorm.DB, sort *models.AuthorSort) *gorm.DB {
	if sort == nil {
		sort = &models.AuthorSort{Field: "name", Order: "asc"}
	}
	
	orderClause := fmt.Sprintf("%s %s", sort.Field, strings.ToUpper(sort.Order))
	return db.Order(orderClause)
}