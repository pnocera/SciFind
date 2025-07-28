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

// categoryRepository implements CategoryRepository interface
type categoryRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewCategoryRepository creates a new category repository
func NewCategoryRepository(db *gorm.DB, logger *slog.Logger) CategoryRepository {
	return &categoryRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new category
func (r *categoryRepository) Create(ctx context.Context, category *models.Category) error {
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		if errors.IsDuplicateKeyError(err) {
			return errors.NewDuplicateError("Category already exists", "category")
		}
		return errors.NewDatabaseError("create_category", err)
	}
	return nil
}

// GetByID retrieves a category by ID
func (r *categoryRepository) GetByID(ctx context.Context, id string) (*models.Category, error) {
	var category models.Category
	err := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children").
		First(&category, "id = ?", id).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Category not found", "category")
		}
		return nil, errors.NewDatabaseError("get_category", err)
	}
	return &category, nil
}

// GetBySourceCode retrieves a category by source and source code
func (r *categoryRepository) GetBySourceCode(ctx context.Context, source, sourceCode string) (*models.Category, error) {
	var category models.Category
	err := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children").
		First(&category, "source = ? AND source_code = ?", source, sourceCode).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Category not found", "source_code")
		}
		return nil, errors.NewDatabaseError("get_category_by_source_code", err)
	}
	return &category, nil
}

// Update updates a category
func (r *categoryRepository) Update(ctx context.Context, category *models.Category) error {
	result := r.db.WithContext(ctx).Save(category)
	if result.Error != nil {
		return errors.NewDatabaseError("update_category", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Category not found", "category")
	}
	return nil
}

// Delete deletes a category
func (r *categoryRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Category{}, "id = ?", id)
	if result.Error != nil {
		return errors.NewDatabaseError("delete_category", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Category not found", "category")
	}
	return nil
}

// GetRootCategories returns all root categories for a given source
func (r *categoryRepository) GetRootCategories(ctx context.Context, source string) ([]models.Category, error) {
	var categories []models.Category
	query := r.db.WithContext(ctx).
		Preload("Children").
		Where("(parent_id IS NULL OR parent_id = '') AND is_active = ?", true)
	
	if source != "" {
		query = query.Where("source = ?", source)
	}
	
	err := query.Order("name ASC").Find(&categories).Error
	if err != nil {
		return nil, errors.NewDatabaseError("get_root_categories", err)
	}
	
	return categories, nil
}

// GetChildren returns all direct children of a category
func (r *categoryRepository) GetChildren(ctx context.Context, parentID string) ([]models.Category, error) {
	var categories []models.Category
	err := r.db.WithContext(ctx).
		Preload("Children").
		Where("parent_id = ? AND is_active = ?", parentID, true).
		Order("name ASC").
		Find(&categories).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_children_categories", err)
	}
	
	return categories, nil
}

// GetParent returns the parent category
func (r *categoryRepository) GetParent(ctx context.Context, categoryID string) (*models.Category, error) {
	var category models.Category
	err := r.db.WithContext(ctx).
		Select("parent_id").
		First(&category, "id = ?", categoryID).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Category not found", "category")
		}
		return nil, errors.NewDatabaseError("get_category_parent_id", err)
	}
	
	if category.ParentID == nil || *category.ParentID == "" {
		return nil, nil // No parent
	}
	
	var parent models.Category
	err = r.db.WithContext(ctx).
		Preload("Parent").
		First(&parent, "id = ?", *category.ParentID).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFoundError("Parent category not found", "category")
		}
		return nil, errors.NewDatabaseError("get_parent_category", err)
	}
	
	return &parent, nil
}

// GetAncestors returns all ancestor categories up to the root
func (r *categoryRepository) GetAncestors(ctx context.Context, categoryID string) ([]models.Category, error) {
	var ancestors []models.Category
	currentID := categoryID
	
	for {
		parent, err := r.GetParent(ctx, currentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			break
		}
		
		ancestors = append([]models.Category{*parent}, ancestors...)
		currentID = parent.ID
	}
	
	return ancestors, nil
}

// GetDescendants returns all descendant categories recursively
func (r *categoryRepository) GetDescendants(ctx context.Context, categoryID string) ([]models.Category, error) {
	var descendants []models.Category
	
	// Get direct children
	children, err := r.GetChildren(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	
	// Add children to descendants
	descendants = append(descendants, children...)
	
	// Recursively get descendants of each child
	for _, child := range children {
		childDescendants, err := r.GetDescendants(ctx, child.ID)
		if err != nil {
			return nil, err
		}
		descendants = append(descendants, childDescendants...)
	}
	
	return descendants, nil
}

// GetCategoryTree returns the complete category tree for a source
func (r *categoryRepository) GetCategoryTree(ctx context.Context, source string) ([]models.CategoryTree, error) {
	var categories []models.Category
	query := r.db.WithContext(ctx).Where("is_active = ?", true)
	
	if source != "" {
		query = query.Where("source = ?", source)
	}
	
	err := query.Order("level ASC, name ASC").Find(&categories).Error
	if err != nil {
		return nil, errors.NewDatabaseError("get_categories_for_tree", err)
	}
	
	return models.BuildCategoryTree(categories), nil
}

// Search searches for categories with filters and sorting
func (r *categoryRepository) Search(ctx context.Context, query string, filters *models.CategoryFilter, sort *models.CategorySort, limit, offset int) ([]models.Category, int64, error) {
	db := r.db.WithContext(ctx).
		Preload("Parent").
		Preload("Children")
	
	// Apply search query
	if query != "" {
		db = db.Where("name ILIKE ? OR description ILIKE ?", 
			"%"+query+"%", "%"+query+"%")
	}
	
	// Apply filters
	db = r.applyCategoryFilters(db, filters)
	
	// Count total results
	var total int64
	if err := db.Model(&models.Category{}).Count(&total).Error; err != nil {
		return nil, 0, errors.NewDatabaseError("count_categories", err)
	}
	
	// Apply sorting
	db = r.applyCategorySorting(db, sort)
	
	// Apply pagination
	var categories []models.Category
	err := db.Limit(limit).Offset(offset).Find(&categories).Error
	if err != nil {
		return nil, 0, errors.NewDatabaseError("search_categories", err)
	}
	
	return categories, total, nil
}

// GetActiveCategories returns all active categories for a source
func (r *categoryRepository) GetActiveCategories(ctx context.Context, source string) ([]models.Category, error) {
	var categories []models.Category
	query := r.db.WithContext(ctx).Where("is_active = ?", true)
	
	if source != "" {
		query = query.Where("source = ?", source)
	}
	
	err := query.Order("level ASC, name ASC").Find(&categories).Error
	if err != nil {
		return nil, errors.NewDatabaseError("get_active_categories", err)
	}
	
	return categories, nil
}

// GetPopularCategories returns categories with the most papers
func (r *categoryRepository) GetPopularCategories(ctx context.Context, limit int) ([]models.Category, error) {
	var categories []models.Category
	err := r.db.WithContext(ctx).
		Where("is_active = ? AND paper_count > 0", true).
		Order("paper_count DESC").
		Limit(limit).
		Find(&categories).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_popular_categories", err)
	}
	
	return categories, nil
}

// CreateBatch creates multiple categories in a batch
func (r *categoryRepository) CreateBatch(ctx context.Context, categories []models.Category) error {
	if len(categories) == 0 {
		return nil
	}
	
	batchSize := 100
	for i := 0; i < len(categories); i += batchSize {
		end := i + batchSize
		if end > len(categories) {
			end = len(categories)
		}
		
		batch := categories[i:end]
		if err := r.db.WithContext(ctx).CreateInBatches(batch, len(batch)).Error; err != nil {
			return errors.NewDatabaseError("create_categories_batch", err)
		}
	}
	
	return nil
}

// UpdateBatch updates multiple categories in a batch
func (r *categoryRepository) UpdateBatch(ctx context.Context, categories []models.Category) error {
	if len(categories) == 0 {
		return nil
	}
	
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	for _, category := range categories {
		if err := tx.Save(&category).Error; err != nil {
			tx.Rollback()
			return errors.NewDatabaseError("update_categories_batch", err)
		}
	}
	
	if err := tx.Commit().Error; err != nil {
		return errors.NewDatabaseError("commit_categories_batch", err)
	}
	
	return nil
}

// GetStats returns category statistics
func (r *categoryRepository) GetStats(ctx context.Context, filters *models.CategoryFilter) (*CategoryStats, error) {
	var stats CategoryStats
	
	db := r.db.WithContext(ctx).Model(&models.Category{})
	db = r.applyCategoryFilters(db, filters)
	
	// Total count
	if err := db.Count(&stats.TotalCount).Error; err != nil {
		return nil, errors.NewDatabaseError("get_category_stats_total", err)
	}
	
	// Active count
	if err := db.Where("is_active = ?", true).Count(&stats.ActiveCount).Error; err != nil {
		return nil, errors.NewDatabaseError("get_category_stats_active", err)
	}
	
	// Average paper count
	var avgPapers float64
	if err := db.Select("AVG(paper_count)").Scan(&avgPapers).Error; err != nil {
		return nil, errors.NewDatabaseError("get_category_stats_avg_papers", err)
	}
	stats.AvgPaperCount = avgPapers
	
	// Top categories by paper count
	var topCategories []struct {
		ID         string
		Name       string
		PaperCount int64
	}
	err := r.db.WithContext(ctx).
		Model(&models.Category{}).
		Select("id, name, paper_count").
		Where("is_active = ? AND paper_count > 0", true).
		Order("paper_count DESC").
		Limit(10).
		Scan(&topCategories).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_category_stats_top", err)
	}
	
	stats.TopCategories = make([]CategoryCount, len(topCategories))
	for i, cat := range topCategories {
		stats.TopCategories[i] = CategoryCount{
			CategoryID: cat.ID,
			Name:       cat.Name,
			Count:      cat.PaperCount,
		}
	}
	
	// Source breakdown
	var sources []struct {
		Source string
		Count  int64
	}
	err = r.db.WithContext(ctx).
		Model(&models.Category{}).
		Select("source, COUNT(*) as count").
		Where("is_active = ?", true).
		Group("source").
		Order("count DESC").
		Scan(&sources).Error
	
	if err != nil {
		return nil, errors.NewDatabaseError("get_category_stats_sources", err)
	}
	
	stats.SourceBreakdown = make([]SourceCount, len(sources))
	for i, src := range sources {
		stats.SourceBreakdown[i] = SourceCount{
			Source: src.Source,
			Count:  src.Count,
		}
	}
	
	return &stats, nil
}

// UpdatePaperCount updates the paper count for a category
func (r *categoryRepository) UpdatePaperCount(ctx context.Context, categoryID string, count int) error {
	result := r.db.WithContext(ctx).
		Model(&models.Category{}).
		Where("id = ?", categoryID).
		Update("paper_count", count)
	
	if result.Error != nil {
		return errors.NewDatabaseError("update_category_paper_count", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NewNotFoundError("Category not found", "category")
	}
	
	return nil
}

// RecalculatePaperCounts recalculates paper counts for all categories
func (r *categoryRepository) RecalculatePaperCounts(ctx context.Context) error {
	// Update paper counts based on actual paper-category relationships
	query := `
		UPDATE categories 
		SET paper_count = (
			SELECT COUNT(*) 
			FROM paper_categories 
			WHERE paper_categories.category_id = categories.id
		)
	`
	
	if err := r.db.WithContext(ctx).Exec(query).Error; err != nil {
		return errors.NewDatabaseError("recalculate_paper_counts", err)
	}
	
	return nil
}

// Helper methods

// applyCategoryFilters applies filters to a GORM query
func (r *categoryRepository) applyCategoryFilters(db *gorm.DB, filters *models.CategoryFilter) *gorm.DB {
	if filters == nil {
		return db
	}
	
	if len(filters.IDs) > 0 {
		db = db.Where("id IN ?", filters.IDs)
	}
	
	if len(filters.Names) > 0 {
		db = db.Where("name IN ?", filters.Names)
	}
	
	if filters.Source != "" {
		db = db.Where("source = ?", filters.Source)
	}
	
	if len(filters.SourceCodes) > 0 {
		db = db.Where("source_code IN ?", filters.SourceCodes)
	}
	
	if filters.ParentID != nil {
		if *filters.ParentID == "" {
			db = db.Where("parent_id IS NULL OR parent_id = ''")
		} else {
			db = db.Where("parent_id = ?", *filters.ParentID)
		}
	}
	
	if filters.Level != nil {
		db = db.Where("level = ?", *filters.Level)
	}
	
	if filters.IsActive != nil {
		db = db.Where("is_active = ?", *filters.IsActive)
	}
	
	if filters.MinPapers != nil {
		db = db.Where("paper_count >= ?", *filters.MinPapers)
	}
	
	if filters.MaxPapers != nil {
		db = db.Where("paper_count <= ?", *filters.MaxPapers)
	}
	
	if filters.CreatedFrom != nil {
		db = db.Where("created_at >= ?", *filters.CreatedFrom)
	}
	
	if filters.CreatedTo != nil {
		db = db.Where("created_at <= ?", *filters.CreatedTo)
	}
	
	return db
}

// applyCategorySorting applies sorting to a GORM query
func (r *categoryRepository) applyCategorySorting(db *gorm.DB, sort *models.CategorySort) *gorm.DB {
	if sort == nil {
		sort = &models.CategorySort{Field: "name", Order: "asc"}
	}
	
	orderClause := fmt.Sprintf("%s %s", sort.Field, strings.ToUpper(sort.Order))
	return db.Order(orderClause)
}