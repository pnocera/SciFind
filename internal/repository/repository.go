package repository

import (
	"context"
	"fmt"

	"scifind-backend/internal/config"
	"scifind-backend/internal/errors"

	"gorm.io/gorm"
	"log/slog"
)

// repository implements the Repository interface
type repository struct {
	db               *Database
	paperRepo        PaperRepository
	authorRepo       AuthorRepository
	categoryRepo     CategoryRepository
	searchRepo       SearchRepository
	logger           *slog.Logger
}

// NewRepository creates a new repository instance
func NewRepository(cfg *config.Config, logger *slog.Logger) (Repository, error) {
	db, err := NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}
	
	return &repository{
		db:               db,
		paperRepo:        NewPaperRepository(db.DB, logger),
		authorRepo:       NewAuthorRepository(db.DB, logger),
		categoryRepo:     NewCategoryRepository(db.DB, logger),
		searchRepo:       NewSearchRepository(db.DB, logger),
		logger:           logger,
	}, nil
}

// Papers returns the paper repository
func (r *repository) Papers() PaperRepository {
	return r.paperRepo
}

// Authors returns the author repository
func (r *repository) Authors() AuthorRepository {
	return r.authorRepo
}

// Categories returns the category repository
func (r *repository) Categories() CategoryRepository {
	return r.categoryRepo
}

// Search returns the search repository
func (r *repository) Search() SearchRepository {
	return r.searchRepo
}

// Transaction executes a function within a database transaction
func (r *repository) Transaction(ctx context.Context, fn func(Transaction) error) error {
	return r.db.Transaction(ctx, func(tx *gorm.DB) error {
		txRepo := &transactionRepository{
			tx:               tx,
			paperRepo:        NewPaperRepository(tx, r.logger),
			authorRepo:       NewAuthorRepository(tx, r.logger),
			categoryRepo:     NewCategoryRepository(tx, r.logger),
			searchRepo:       NewSearchRepository(tx, r.logger),
		}
		return fn(txRepo)
	})
}

// Ping checks the database connection
func (r *repository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

// Close closes the database connection
func (r *repository) Close() error {
	return r.db.Close()
}

// GetStats returns database statistics
func (r *repository) GetStats() (map[string]interface{}, error) {
	return r.db.GetStats()
}

// transactionRepository implements the Transaction interface
type transactionRepository struct {
	tx               *gorm.DB
	paperRepo        PaperRepository
	authorRepo       AuthorRepository
	categoryRepo     CategoryRepository
	searchRepo       SearchRepository
}

// Begin starts a new transaction (not needed for GORM as it's already in transaction)
func (t *transactionRepository) Begin(ctx context.Context) (Transaction, error) {
	// GORM automatically handles nested transactions
	return t, nil
}

// Commit commits the transaction
func (t *transactionRepository) Commit() error {
	// GORM handles this automatically when the parent transaction function returns
	return nil
}

// Rollback rolls back the transaction
func (t *transactionRepository) Rollback() error {
	// GORM handles this automatically when the parent transaction function returns an error
	return nil
}

// Papers returns the paper repository within the transaction
func (t *transactionRepository) Papers() PaperRepository {
	return t.paperRepo
}

// Authors returns the author repository within the transaction
func (t *transactionRepository) Authors() AuthorRepository {
	return t.authorRepo
}

// Categories returns the category repository within the transaction
func (t *transactionRepository) Categories() CategoryRepository {
	return t.categoryRepo
}

// Search returns the search repository within the transaction
func (t *transactionRepository) Search() SearchRepository {
	return t.searchRepo
}

// RepositoryManager provides additional repository management functionality
type RepositoryManager struct {
	repo   Repository
	logger *slog.Logger
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(repo Repository, logger *slog.Logger) *RepositoryManager {
	return &RepositoryManager{
		repo:   repo,
		logger: logger,
	}
}

// HealthCheck performs a comprehensive health check of all repositories
func (rm *RepositoryManager) HealthCheck(ctx context.Context) error {
	// Check database connection
	if err := rm.repo.Ping(ctx); err != nil {
		return errors.NewHealthCheckError("database ping failed: " + err.Error(), "database")
	}
	
	// Test basic operations on each repository
	if err := rm.testPaperRepository(ctx); err != nil {
		return errors.NewHealthCheckError("paper repository test failed: " + err.Error(), "repository")
	}
	
	if err := rm.testAuthorRepository(ctx); err != nil {
		return errors.NewHealthCheckError("author repository test failed: " + err.Error(), "repository")
	}
	
	if err := rm.testCategoryRepository(ctx); err != nil {
		return errors.NewHealthCheckError("category repository test failed: " + err.Error(), "repository")
	}
	
	if err := rm.testSearchRepository(ctx); err != nil {
		return errors.NewHealthCheckError("search repository test failed: " + err.Error(), "repository")
	}
	
	rm.logger.Info("Repository health check passed")
	return nil
}

// GetDetailedStats returns detailed statistics from all repositories
func (rm *RepositoryManager) GetDetailedStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Database stats
	dbStats, err := rm.repo.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}
	stats["database"] = dbStats
	
	// Paper stats
	paperStats, err := rm.repo.Papers().GetStats(ctx, nil)
	if err != nil {
		rm.logger.Warn("Failed to get paper stats", slog.String("error", err.Error()))
	} else {
		stats["papers"] = paperStats
	}
	
	// Author stats
	authorStats, err := rm.repo.Authors().GetStats(ctx, nil)
	if err != nil {
		rm.logger.Warn("Failed to get author stats", slog.String("error", err.Error()))
	} else {
		stats["authors"] = authorStats
	}
	
	// Category stats
	categoryStats, err := rm.repo.Categories().GetStats(ctx, nil)
	if err != nil {
		rm.logger.Warn("Failed to get category stats", slog.String("error", err.Error()))
	} else {
		stats["categories"] = categoryStats
	}
	
	// Cache stats
	cacheStats, err := rm.repo.Search().GetCacheStats(ctx)
	if err != nil {
		rm.logger.Warn("Failed to get cache stats", slog.String("error", err.Error()))
	} else {
		stats["cache"] = cacheStats
	}
	
	return stats, nil
}

// CleanupExpiredData performs cleanup of expired data across all repositories
func (rm *RepositoryManager) CleanupExpiredData(ctx context.Context) error {
	rm.logger.Info("Starting expired data cleanup")
	
	// Cleanup expired search cache
	if err := rm.repo.Search().CleanupExpiredCache(ctx); err != nil {
		rm.logger.Error("Failed to cleanup expired cache", slog.String("error", err.Error()))
		return fmt.Errorf("failed to cleanup expired cache: %w", err)
	}
	
	// Additional cleanup operations could be added here
	// For example: cleanup old search history, temporary data, etc.
	
	rm.logger.Info("Expired data cleanup completed")
	return nil
}

// RecalculateMetrics recalculates derived metrics across all repositories
func (rm *RepositoryManager) RecalculateMetrics(ctx context.Context) error {
	rm.logger.Info("Starting metrics recalculation")
	
	// Recalculate category paper counts
	if err := rm.repo.Categories().RecalculatePaperCounts(ctx); err != nil {
		rm.logger.Error("Failed to recalculate category paper counts", slog.String("error", err.Error()))
		return fmt.Errorf("failed to recalculate category paper counts: %w", err)
	}
	
	// Could add more metric recalculations here:
	// - Author metrics (H-index, citation counts)
	// - Paper quality scores
	// - Trending calculations
	
	rm.logger.Info("Metrics recalculation completed")
	return nil
}

// Helper methods for health checks

func (rm *RepositoryManager) testPaperRepository(ctx context.Context) error {
	// Test getting processing stats (lightweight operation)
	_, err := rm.repo.Papers().GetProcessingStats(ctx)
	return err
}

func (rm *RepositoryManager) testAuthorRepository(ctx context.Context) error {
	// Test getting top authors (lightweight operation)
	_, err := rm.repo.Authors().GetTopAuthors(ctx, "h_index", 1)
	return err
}

func (rm *RepositoryManager) testCategoryRepository(ctx context.Context) error {
	// Test getting active categories (lightweight operation)
	_, err := rm.repo.Categories().GetActiveCategories(ctx, "")
	return err
}

func (rm *RepositoryManager) testSearchRepository(ctx context.Context) error {
	// Test getting cache stats (lightweight operation)
	_, err := rm.repo.Search().GetCacheStats(ctx)
	return err
}