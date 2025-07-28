package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"scifind-backend/internal/config"
	"scifind-backend/internal/errors"
	"scifind-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database represents the database connection and operations
type Database struct {
	*gorm.DB
	config *config.Config
	logger *slog.Logger
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type        string
	DSN         string
	MaxConns    int
	MaxIdle     int
	MaxLifetime time.Duration
	MaxIdleTime time.Duration
	AutoMigrate bool
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.Config, logger *slog.Logger) (*Database, error) {
	dbConfig, err := buildDatabaseConfig(cfg)
	if err != nil {
		return nil, errors.NewDatabaseError("config_validation", err)
	}

	var dialector gorm.Dialector
	
	switch dbConfig.Type {
	case "postgres":
		dialector = postgres.Open(dbConfig.DSN)
	case "sqlite":
		dialector = sqlite.Open(dbConfig.DSN)
	default:
		return nil, errors.NewValidationError("Unsupported database type", "type", dbConfig.Type)
	}

	gormConfig := &gorm.Config{
		Logger: NewGormLogger(logger),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: true, // Enable prepared statements for better performance
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, errors.NewDatabaseError("connection", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.NewDatabaseError("connection_pool", err)
	}

	sqlDB.SetMaxOpenConns(dbConfig.MaxConns)
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdle)
	sqlDB.SetConnMaxLifetime(dbConfig.MaxLifetime)
	sqlDB.SetConnMaxIdleTime(dbConfig.MaxIdleTime)

	database := &Database{
		DB:     db,
		config: cfg,
		logger: logger,
	}

	// Auto-migrate if configured
	if dbConfig.AutoMigrate {
		if err := database.Migrate(); err != nil {
			return nil, errors.NewDatabaseError("migration", err)
		}
	}

	logger.Info("Database connection established",
		slog.String("type", dbConfig.Type),
		slog.Int("max_conns", dbConfig.MaxConns),
		slog.Int("max_idle", dbConfig.MaxIdle))

	return database, nil
}

// Migrate runs database migrations
func (d *Database) Migrate() error {
	models := []interface{}{
		&models.Author{},
		&models.Category{},
		&models.Paper{},
		&models.SearchHistory{},
		&models.SearchCache{},
	}

	for _, model := range models {
		if err := d.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// Create custom indexes
	if err := d.createCustomIndexes(); err != nil {
		return fmt.Errorf("failed to create custom indexes: %w", err)
	}

	// Seed predefined data
	if err := d.seedPredefinedData(); err != nil {
		return fmt.Errorf("failed to seed data: %w", err)
	}

	d.logger.Info("Database migration completed successfully")
	return nil
}

// createCustomIndexes creates custom database indexes
func (d *Database) createCustomIndexes() error {
	var indexes []string
	
	switch d.config.Database.Type {
	case "postgres":
		indexes = []string{
			// Papers indexes - PostgreSQL specific
			"CREATE INDEX IF NOT EXISTS idx_papers_search_text ON papers USING gin(to_tsvector('english', title || ' ' || COALESCE(abstract, '')))",
			"CREATE INDEX IF NOT EXISTS idx_papers_provider_source ON papers (source_provider, source_id)",
			"CREATE INDEX IF NOT EXISTS idx_papers_published_quality ON papers (published_at DESC NULLS LAST, quality_score DESC)",
			"CREATE INDEX IF NOT EXISTS idx_papers_citations_date ON papers (citation_count DESC, published_at DESC NULLS LAST)",
			"CREATE INDEX IF NOT EXISTS idx_papers_processing_state ON papers (processing_state) WHERE processing_state != 'completed'",
			
			// Authors indexes - PostgreSQL specific
			"CREATE INDEX IF NOT EXISTS idx_authors_name_trgm ON authors USING gin(name gin_trgm_ops)",
			"CREATE INDEX IF NOT EXISTS idx_authors_metrics ON authors (h_index DESC, citation_count DESC, paper_count DESC)",
			"CREATE INDEX IF NOT EXISTS idx_authors_affiliation ON authors USING gin(affiliation gin_trgm_ops) WHERE affiliation IS NOT NULL",
			
			// Categories indexes
			"CREATE INDEX IF NOT EXISTS idx_categories_source_code ON categories (source, source_code)",
			"CREATE INDEX IF NOT EXISTS idx_categories_hierarchy ON categories (parent_id, level)",
			"CREATE INDEX IF NOT EXISTS idx_categories_active ON categories (is_active) WHERE is_active = true",
			
			// Search cache indexes - PostgreSQL specific
			"CREATE INDEX IF NOT EXISTS idx_search_cache_expires ON search_cache (expires_at) WHERE expires_at > NOW()",
			"CREATE INDEX IF NOT EXISTS idx_search_cache_provider ON search_cache (provider, created_at DESC)",
			
			// Search history indexes - PostgreSQL specific
			"CREATE INDEX IF NOT EXISTS idx_search_history_user ON search_history (user_id, requested_at DESC) WHERE user_id IS NOT NULL",
			"CREATE INDEX IF NOT EXISTS idx_search_history_query ON search_history USING gin(to_tsvector('english', query))",
		}
	case "sqlite":
		indexes = []string{
			// Papers indexes - SQLite compatible
			"CREATE INDEX IF NOT EXISTS idx_papers_title ON papers (title)",
			"CREATE INDEX IF NOT EXISTS idx_papers_abstract ON papers (abstract)",
			"CREATE INDEX IF NOT EXISTS idx_papers_provider_source ON papers (source_provider, source_id)",
			"CREATE INDEX IF NOT EXISTS idx_papers_published_quality ON papers (published_at DESC, quality_score DESC)",
			"CREATE INDEX IF NOT EXISTS idx_papers_citations_date ON papers (citation_count DESC, published_at DESC)",
			"CREATE INDEX IF NOT EXISTS idx_papers_processing_state ON papers (processing_state)",
			
			// Authors indexes - SQLite compatible
			"CREATE INDEX IF NOT EXISTS idx_authors_name ON authors (name)",
			"CREATE INDEX IF NOT EXISTS idx_authors_metrics ON authors (h_index DESC, citation_count DESC, paper_count DESC)",
			"CREATE INDEX IF NOT EXISTS idx_authors_affiliation ON authors (affiliation)",
			
			// Categories indexes
			"CREATE INDEX IF NOT EXISTS idx_categories_source_code ON categories (source, source_code)",
			"CREATE INDEX IF NOT EXISTS idx_categories_hierarchy ON categories (parent_id, level)",
			"CREATE INDEX IF NOT EXISTS idx_categories_active ON categories (is_active)",
			
			// Search cache indexes - SQLite compatible
			"CREATE INDEX IF NOT EXISTS idx_search_cache_expires ON search_cache (expires_at)",
			"CREATE INDEX IF NOT EXISTS idx_search_cache_provider ON search_cache (provider, created_at DESC)",
			
			// Search history indexes - SQLite compatible
			"CREATE INDEX IF NOT EXISTS idx_search_history_user ON search_history (user_id, requested_at DESC)",
			"CREATE INDEX IF NOT EXISTS idx_search_history_query ON search_history (query)",
		}
	default:
		// Basic indexes that work on most databases
		indexes = []string{
			"CREATE INDEX IF NOT EXISTS idx_papers_provider_source ON papers (source_provider, source_id)",
			"CREATE INDEX IF NOT EXISTS idx_papers_published ON papers (published_at)",
			"CREATE INDEX IF NOT EXISTS idx_authors_name ON authors (name)",
			"CREATE INDEX IF NOT EXISTS idx_categories_source_code ON categories (source, source_code)",
		}
	}

	for _, indexSQL := range indexes {
		if err := d.Exec(indexSQL).Error; err != nil {
			d.logger.Warn("Failed to create index", slog.String("sql", indexSQL), slog.String("error", err.Error()))
			// Continue with other indexes even if one fails
		}
	}

	return nil
}

// seedPredefinedData seeds predefined categories and other data
func (d *Database) seedPredefinedData() error {
	// Seed predefined categories
	for _, category := range models.PredefinedCategories {
		var existing models.Category
		if err := d.Where("id = ?", category.ID).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := d.Create(&category).Error; err != nil {
					d.logger.Warn("Failed to seed category", slog.String("id", category.ID), slog.String("error", err.Error()))
				}
			}
		}
	}

	return nil
}

// Ping checks the database connection
func (d *Database) Ping(ctx context.Context) error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// WithContext returns a new DB instance with context
func (d *Database) WithContext(ctx context.Context) *gorm.DB {
	return d.DB.WithContext(ctx)
}

// Transaction executes a function within a database transaction
func (d *Database) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return d.WithContext(ctx).Transaction(fn)
}

// GetStats returns database statistics
func (d *Database) GetStats() (map[string]interface{}, error) {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return nil, err
	}

	stats := sqlDB.Stats()
	
	return map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration.String(),
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}, nil
}

// buildDatabaseConfig builds database configuration from app config
func buildDatabaseConfig(cfg *config.Config) (*DatabaseConfig, error) {
	dbConfig := &DatabaseConfig{
		Type: cfg.Database.Type,
	}

	switch cfg.Database.Type {
	case "postgres":
		dbConfig.DSN = cfg.Database.PostgreSQL.DSN
		dbConfig.MaxConns = cfg.Database.PostgreSQL.MaxConns
		dbConfig.MaxIdle = cfg.Database.PostgreSQL.MaxIdle
		dbConfig.AutoMigrate = cfg.Database.PostgreSQL.AutoMigrate
		
		if cfg.Database.PostgreSQL.MaxLifetime != "" {
			duration, err := time.ParseDuration(cfg.Database.PostgreSQL.MaxLifetime)
			if err != nil {
				return nil, fmt.Errorf("invalid max_lifetime: %w", err)
			}
			dbConfig.MaxLifetime = duration
		} else {
			dbConfig.MaxLifetime = time.Hour
		}
		
		if cfg.Database.PostgreSQL.MaxIdleTime != "" {
			duration, err := time.ParseDuration(cfg.Database.PostgreSQL.MaxIdleTime)
			if err != nil {
				return nil, fmt.Errorf("invalid max_idle_time: %w", err)
			}
			dbConfig.MaxIdleTime = duration
		} else {
			dbConfig.MaxIdleTime = 30 * time.Minute
		}
		
	case "sqlite":
		dbConfig.DSN = cfg.Database.SQLite.Path
		dbConfig.MaxConns = 1  // SQLite is single-writer
		dbConfig.MaxIdle = 1
		dbConfig.MaxLifetime = 0 // No limit for SQLite
		dbConfig.MaxIdleTime = 0
		dbConfig.AutoMigrate = cfg.Database.SQLite.AutoMigrate
		
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	if dbConfig.DSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	return dbConfig, nil
}

// GormLogger adapts slog to gorm logger interface
type GormLogger struct {
	logger *slog.Logger
}

// NewGormLogger creates a new GORM logger
func NewGormLogger(logger *slog.Logger) logger.Interface {
	return &GormLogger{
		logger: logger,
	}
}

// LogMode sets the log mode (ignored, using slog level)
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

// Info logs info level messages
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.InfoContext(ctx, fmt.Sprintf(msg, data...))
}

// Warn logs warning level messages
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.WarnContext(ctx, fmt.Sprintf(msg, data...))
}

// Error logs error level messages
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.ErrorContext(ctx, fmt.Sprintf(msg, data...))
}

// Trace logs SQL traces
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	
	args := []any{
		slog.Duration("elapsed", elapsed),
		slog.Int64("rows", rows),
		slog.String("sql", sql),
	}
	
	if err != nil {
		args = append(args, slog.String("error", err.Error()))
		l.logger.ErrorContext(ctx, "SQL query failed", args...)
	} else {
		l.logger.DebugContext(ctx, "SQL query executed", args...)
	}
}