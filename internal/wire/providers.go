package wire

import (
	"log/slog"
	"os"
	"time"

	"scifind-backend/internal/api/handlers"
	"scifind-backend/internal/config"
	"scifind-backend/internal/messaging"
	"scifind-backend/internal/repository"
	"scifind-backend/internal/services"
)

// Configuration Providers

// ProvideLogger creates a structured logger instance
func ProvideLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	// Configure log output
	var output *os.File
	switch cfg.Logging.Output {
	case "stderr":
		output = os.Stderr
	case "file":
		if cfg.Logging.FilePath != "" {
			if f, err := os.OpenFile(cfg.Logging.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
				output = f
			} else {
				output = os.Stdout // fallback
			}
		} else {
			output = os.Stdout
		}
	default:
		output = os.Stdout
	}

	// Configure log level
	var level slog.Level
	switch cfg.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.Logging.AddSource,
	}

	// Create handler based on format
	switch cfg.Logging.Format {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	return slog.New(handler)
}

// ProvideDevelopmentLogger creates a development logger
func ProvideDevelopmentLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// ProvideTestLogger creates a test logger (silent)
func ProvideTestLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelError, // Only errors in tests
		AddSource: false,
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}

// Database Providers

// ProvideDatabase creates a database instance
func ProvideDatabase(cfg *config.Config, logger *slog.Logger) *repository.Database {
	db, err := repository.NewDatabase(cfg, logger)
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		// Return nil to allow graceful degradation
		return nil
	}
	return db
}

// ProvideDevelopmentDatabase creates a development database (SQLite)
func ProvideDevelopmentDatabase(logger *slog.Logger) *repository.Database {
	cfg := &config.Config{}
	cfg.Database.Type = "sqlite"
	cfg.Database.SQLite.Path = "./dev-scifind.db"
	cfg.Database.SQLite.AutoMigrate = true

	db, err := repository.NewDatabase(cfg, logger)
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		// Return nil to allow graceful degradation
		return nil
	}
	return db
}

// ProvideTestDatabase creates a test database (in-memory SQLite)
func ProvideTestDatabase(logger *slog.Logger) *repository.Database {
	cfg := &config.Config{}
	cfg.Database.Type = "sqlite"
	cfg.Database.SQLite.Path = ":memory:"
	cfg.Database.SQLite.AutoMigrate = true

	db, err := repository.NewDatabase(cfg, logger)
	if err != nil {
		logger.Error("Failed to connect to database", slog.String("error", err.Error()))
		// Return nil to allow graceful degradation
		return nil
	}
	return db
}

// Messaging Providers

// ProvideMessaging creates a NATS messaging client
func ProvideMessaging(cfg *config.Config, logger *slog.Logger) *messaging.Client {
	client, err := messaging.NewClient(cfg.NATS, logger)
	if err != nil {
		logger.Error("Failed to connect to NATS", slog.String("error", err.Error()))
		// Return nil client to allow graceful degradation
		return nil
	}

	return client
}

// ProvideDevelopmentMessaging creates a development NATS client
func ProvideDevelopmentMessaging(logger *slog.Logger) *messaging.Client {
	natsConfig := &config.NATSConfig{
		URL:           "nats://localhost:4222",
		ClusterID:     "scifind-dev",
		ClientID:      "scifind-backend-dev",
		MaxReconnects: 5,
		ReconnectWait: "2s",
		Timeout:       "5s",
		PingInterval:  30,
		MaxPingsOut:   2,
	}

	// Set JetStream configuration
	natsConfig.JetStream.Enabled = true
	natsConfig.JetStream.Domain = ""
	natsConfig.JetStream.MaxMemory = "1GB"
	natsConfig.JetStream.MaxStorage = "10GB"

	client, err := messaging.NewClient(*natsConfig, logger)
	if err != nil {
		logger.Error("Failed to connect to NATS", slog.String("error", err.Error()))
		// Return nil to allow graceful degradation
		return nil
	}

	return client
}

// ProvideTestMessaging creates a mock messaging client for testing
func ProvideTestMessaging(logger *slog.Logger) *messaging.Client {
	// For testing, we can return a mock client or skip NATS entirely
	// This would typically implement the same interface but with in-memory behavior
	return nil // TODO: Implement mock messaging client
}

// Repository Providers

// ProvideRepositories creates repository instances
func ProvideRepositories(db *repository.Database, logger *slog.Logger) *repository.Container {
	return &repository.Container{
		Paper:    repository.NewPaperRepository(db.DB, logger),
		Author:   repository.NewAuthorRepository(db.DB, logger),
		Category: repository.NewCategoryRepository(db.DB, logger),
		Search:   repository.NewSearchRepository(db.DB, logger),
	}
}

// ProvideDevelopmentRepositories creates development repositories
func ProvideDevelopmentRepositories(db *repository.Database, logger *slog.Logger) *repository.Container {
	// Same as production but could include development-specific features
	return ProvideRepositories(db, logger)
}

// ProvideTestRepositories creates test repositories
func ProvideTestRepositories(db *repository.Database, logger *slog.Logger) *repository.Container {
	// Could return mock repositories for testing
	return ProvideRepositories(db, logger)
}

// Service Providers

// ProvideServices creates service instances
func ProvideServices(repos *repository.Container, messaging *messaging.Client, logger *slog.Logger) *services.Container {
	return &services.Container{
		Paper:     services.NewPaperService(repos.Paper, messaging, logger),
		Search:    services.NewSearchService(repos.Search, repos.Paper, messaging, logger),
		Analytics: services.NewAnalyticsService(repos.Search, messaging, logger),
		Health:    services.NewHealthService(repos, messaging, logger),
	}
}

// ProvideDevelopmentServices creates development services
func ProvideDevelopmentServices(repos *repository.Container, messaging *messaging.Client, logger *slog.Logger) *services.Container {
	// Development services might have different configurations
	return ProvideServices(repos, messaging, logger)
}

// ProvideTestServices creates test services
func ProvideTestServices(repos *repository.Container, messaging *messaging.Client, logger *slog.Logger) *services.Container {
	// Test services might use mocks
	return ProvideServices(repos, messaging, logger)
}

// Handler Providers

// ProvideHandlers creates HTTP handler instances
func ProvideHandlers(services *services.Container, logger *slog.Logger) *handlers.Container {
	return &handlers.Container{
		Paper:     handlers.NewPaperHandler(services.Paper, logger),
		Search:    handlers.NewSearchHandler(services.Search, logger),
		Analytics: handlers.NewAnalyticsHandler(services.Analytics, logger),
		Health:    handlers.NewHealthHandler(services.Health, logger),
	}
}

// ProvideTestHandlers creates test handlers
func ProvideTestHandlers(services *services.Container, logger *slog.Logger) *handlers.Container {
	return ProvideHandlers(services, logger)
}

// Application Providers

// ProvideApplication creates the main application instance
func ProvideApplication(
	cfg *config.Config,
	db *repository.Database,
	messaging *messaging.Client,
	services *services.Container,
	handlers *handlers.Container,
	logger *slog.Logger,
) *Application {
	return &Application{
		Config:    cfg,
		Database:  db,
		Messaging: messaging,
		Services:  services,
		Handlers:  handlers,
		Logger:    logger,
	}
}

// Cleanup Providers

// ProvideCleanup creates a cleanup function for the application
func ProvideCleanup(db *repository.Database, messaging *messaging.Client) func() {
	return func() {
		if messaging != nil {
			messaging.Close()
		}
		if db != nil {
			db.Close()
		}
	}
}

// ProvideDevelopmentCleanup creates a development cleanup function
func ProvideDevelopmentCleanup(db *repository.Database, messaging *messaging.Client) func() {
	return ProvideCleanup(db, messaging)
}

// ProvideTestCleanup creates a test cleanup function
func ProvideTestCleanup(db *repository.Database, messaging *messaging.Client) func() {
	return ProvideCleanup(db, messaging)
}

// ProvideDatabaseCleanup creates a database-only cleanup function
func ProvideDatabaseCleanup(db *repository.Database) func() {
	return func() {
		if db != nil {
			db.Close()
		}
	}
}

// ProvideMessagingCleanup creates a messaging-only cleanup function
func ProvideMessagingCleanup(messaging *messaging.Client) func() {
	return func() {
		if messaging != nil {
			messaging.Close()
		}
	}
}

// Development Configuration Providers

// ProvideDevelopmentConfig creates a development configuration
func ProvideDevelopmentConfig() *config.Config {
	cfg, err := config.LoadConfig()
	if err != nil {
		// Fallback to development defaults
		cfg = &config.Config{}
		cfg.Server.Mode = "debug"
		cfg.Server.Port = 8080
		cfg.Database.Type = "sqlite"
		cfg.Database.SQLite.Path = "./dev-scifind.db"
		cfg.Database.SQLite.AutoMigrate = true
		cfg.NATS.URL = "nats://localhost:4222"
		cfg.Logging.Level = "debug"
		cfg.Logging.Format = "text"
	}
	return cfg
}

// ProvideTestConfig creates a test configuration
func ProvideTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Server.Mode = "test"
	cfg.Server.Port = 0 // Random port for testing
	cfg.Database.Type = "sqlite"
	cfg.Database.SQLite.Path = ":memory:"
	cfg.Database.SQLite.AutoMigrate = true
	cfg.Logging.Level = "error"
	cfg.Logging.Format = "text"
	return cfg
}

// Helper functions

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second // default fallback
	}
	return d
}
