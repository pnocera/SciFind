package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"scifind-backend/internal/models"
)

// DatabaseTestUtil provides database testing utilities
type DatabaseTestUtil struct {
	container  *postgres.PostgresContainer
	db         *gorm.DB
	cleanup    func()
	isPostgres bool
}

// SetupTestDatabase creates a test database (PostgreSQL in container or SQLite in memory)
func SetupTestDatabase(t *testing.T, usePostgres bool) *DatabaseTestUtil {
	ctx := context.Background()
	
	if usePostgres {
		return setupPostgresContainer(t, ctx)
	}
	return setupSQLiteInMemory(t)
}

// setupPostgresContainer creates a PostgreSQL container for testing
func setupPostgresContainer(t *testing.T, ctx context.Context) *DatabaseTestUtil {
	// Create PostgreSQL container
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect with GORM
	db, err := gorm.Open(pgdriver.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.Paper{},
		&models.Author{},
		&models.Category{},
		&models.SearchHistory{},
		&models.SearchCache{},
		&models.SearchSuggestion{},
	)
	require.NoError(t, err)

	return &DatabaseTestUtil{
		container:  pgContainer,
		db:         db,
		isPostgres: true,
		cleanup: func() {
			if err := pgContainer.Terminate(ctx); err != nil {
				t.Logf("failed to terminate container: %s", err)
			}
		},
	}
}

// setupSQLiteInMemory creates an in-memory SQLite database for testing
func setupSQLiteInMemory(t *testing.T) *DatabaseTestUtil {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.Paper{},
		&models.Author{},
		&models.Category{},
		&models.SearchHistory{},
		&models.SearchCache{},
		&models.SearchSuggestion{},
	)
	require.NoError(t, err)

	return &DatabaseTestUtil{
		db:         db,
		isPostgres: false,
		cleanup:    func() {}, // Nothing to cleanup for in-memory SQLite
	}
}

// DB returns the GORM database instance
func (d *DatabaseTestUtil) DB() *gorm.DB {
	return d.db
}

// Cleanup cleans up the test database
func (d *DatabaseTestUtil) Cleanup() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

// TruncateAllTables truncates all tables for clean test state
func (d *DatabaseTestUtil) TruncateAllTables(t *testing.T) {
	tables := []string{
		"paper_authors",
		"paper_categories", 
		"papers",
		"authors",
		"categories",
		"search_histories",
		"search_caches",
		"search_suggestions",
	}

	if d.isPostgres {
		// For PostgreSQL, use TRUNCATE CASCADE
		for _, table := range tables {
			err := d.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)).Error
			if err != nil {
				// Table might not exist, which is fine
				continue
			}
		}
	} else {
		// For SQLite, delete all records
		for _, table := range tables {
			err := d.db.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error
			if err != nil {
				// Table might not exist, which is fine
				continue
			}
		}
	}
}

// Transaction executes a function within a database transaction
func (d *DatabaseTestUtil) Transaction(t *testing.T, fn func(*gorm.DB) error) {
	tx := d.db.Begin()
	require.NoError(t, tx.Error)

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			t.Fatalf("Transaction panicked: %v", r)
		}
	}()

	err := fn(tx)
	if err != nil {
		tx.Rollback()
		require.NoError(t, err)
	}

	require.NoError(t, tx.Commit().Error)
}

// AssertTableCount asserts the count of records in a table
func (d *DatabaseTestUtil) AssertTableCount(t *testing.T, table string, expected int64) {
	var count int64
	err := d.db.Table(table).Count(&count).Error
	require.NoError(t, err)
	require.Equal(t, expected, count, "Table %s should have %d records", table, expected)
}

// SeedBasicData seeds the database with basic test data
func (d *DatabaseTestUtil) SeedBasicData(t *testing.T) {
	// Create test categories
	categories := []models.Category{
		{
			ID:       "cs.AI",
			Name:     "Artificial Intelligence",
			Source:   "arxiv",
			SourceCode: "cs.AI",
			IsActive: true,
		},
		{
			ID:       "cs.ML",
			Name:     "Machine Learning",
			Source:   "arxiv",
			SourceCode: "cs.ML",
			ParentID: stringPtr("cs.AI"),
			IsActive: true,
		},
	}

	for _, category := range categories {
		err := d.db.Create(&category).Error
		require.NoError(t, err)
	}

	// Create test authors
	authors := []models.Author{
		{
			ID:          "auth_1",
			Name:        "John Doe",
			Email:       stringPtr("john.doe@university.edu"),
			Affiliation: stringPtr("University of AI"),
			ORCID:       stringPtr("0000-0000-0000-0001"),
		},
		{
			ID:          "auth_2", 
			Name:        "Jane Smith",
			Email:       stringPtr("jane.smith@tech.com"),
			Affiliation: stringPtr("Tech Institute"),
			ORCID:       stringPtr("0000-0000-0000-0002"),
		},
	}

	for _, author := range authors {
		err := d.db.Create(&author).Error
		require.NoError(t, err)
	}

	// Create test papers
	publishedAt := time.Now().AddDate(-1, 0, 0)
	papers := []models.Paper{
		{
			ID:              "arxiv_2301.00001",
			Title:           "Advances in Machine Learning",
			Abstract:        stringPtr("This paper discusses recent advances in machine learning techniques."),
			DOI:             stringPtr("10.1000/test.001"),
			ArxivID:         stringPtr("2301.00001"),
			Journal:         stringPtr("Journal of AI Research"),
			PublishedAt:     &publishedAt,
			URL:             stringPtr("https://arxiv.org/abs/2301.00001"),
			PDFURL:          stringPtr("https://arxiv.org/pdf/2301.00001.pdf"),
			Keywords:        []string{"machine learning", "artificial intelligence"},
			Language:        "en",
			CitationCount:   42,
			SourceProvider:  "arxiv",
			SourceID:        "2301.00001",
			QualityScore:    0.85,
			ProcessingState: "completed",
		},
	}

	for _, paper := range papers {
		err := d.db.Create(&paper).Error
		require.NoError(t, err)

		// Associate with authors and categories
		err = d.db.Model(&paper).Association("Authors").Append(authors)
		require.NoError(t, err)

		err = d.db.Model(&paper).Association("Categories").Append(categories)
		require.NoError(t, err)
	}
}

// CreateTestPaper creates a test paper with minimal required fields
func (d *DatabaseTestUtil) CreateTestPaper(t *testing.T, overrides *models.Paper) *models.Paper {
	paper := &models.Paper{
		ID:              fmt.Sprintf("test_%d", time.Now().UnixNano()),
		Title:           "Test Paper",
		SourceProvider:  "test",
		SourceID:        fmt.Sprintf("test_%d", time.Now().UnixNano()),
		Language:        "en",
		ProcessingState: "pending",
	}

	if overrides != nil {
		// Apply overrides
		if overrides.ID != "" {
			paper.ID = overrides.ID
		}
		if overrides.Title != "" {
			paper.Title = overrides.Title
		}
		if overrides.Abstract != nil {
			paper.Abstract = overrides.Abstract
		}
		if overrides.DOI != nil {
			paper.DOI = overrides.DOI
		}
		if overrides.ArxivID != nil {
			paper.ArxivID = overrides.ArxivID
		}
		if overrides.SourceProvider != "" {
			paper.SourceProvider = overrides.SourceProvider
		}
		if overrides.SourceID != "" {
			paper.SourceID = overrides.SourceID
		}
		if overrides.ProcessingState != "" {
			paper.ProcessingState = overrides.ProcessingState
		}
	}

	err := d.db.Create(paper).Error
	require.NoError(t, err)

	return paper
}

// CreateTestAuthor creates a test author
func (d *DatabaseTestUtil) CreateTestAuthor(t *testing.T, overrides *models.Author) *models.Author {
	author := &models.Author{
		ID:   fmt.Sprintf("auth_%d", time.Now().UnixNano()),
		Name: "Test Author",
	}

	if overrides != nil {
		if overrides.ID != "" {
			author.ID = overrides.ID
		}
		if overrides.Name != "" {
			author.Name = overrides.Name
		}
		if overrides.Email != nil {
			author.Email = overrides.Email
		}
		if overrides.Affiliation != nil {
			author.Affiliation = overrides.Affiliation
		}
		if overrides.ORCID != nil {
			author.ORCID = overrides.ORCID
		}
	}

	err := d.db.Create(author).Error
	require.NoError(t, err)

	return author
}

// CreateTestCategory creates a test category
func (d *DatabaseTestUtil) CreateTestCategory(t *testing.T, overrides *models.Category) *models.Category {
	category := &models.Category{
		ID:         fmt.Sprintf("cat_%d", time.Now().UnixNano()),
		Name:       "Test Category",
		Source:     "test",
		SourceCode: fmt.Sprintf("test.%d", time.Now().UnixNano()),
		IsActive:   true,
	}

	if overrides != nil {
		if overrides.ID != "" {
			category.ID = overrides.ID
		}
		if overrides.Name != "" {
			category.Name = overrides.Name
		}
		if overrides.Source != "" {
			category.Source = overrides.Source
		}
		if overrides.SourceCode != "" {
			category.SourceCode = overrides.SourceCode
		}
		if overrides.ParentID != nil {
			category.ParentID = overrides.ParentID
		}
	}

	err := d.db.Create(category).Error
	require.NoError(t, err)

	return category
}

// GetPostgresConnectionForRawSQL returns raw SQL connection for PostgreSQL
func (d *DatabaseTestUtil) GetPostgresConnectionForRawSQL(t *testing.T) *sql.DB {
	require.True(t, d.isPostgres, "This method is only available for PostgreSQL containers")
	
	sqlDB, err := d.db.DB()
	require.NoError(t, err)
	
	return sqlDB
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}