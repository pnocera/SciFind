//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"scifind-backend/internal/api"
	"scifind-backend/internal/api/handlers"
	"scifind-backend/internal/config"
	"scifind-backend/internal/messaging"
	"scifind-backend/internal/messaging/embedded"
	"scifind-backend/internal/providers"
	"scifind-backend/internal/providers/arxiv"
	"scifind-backend/internal/providers/exa"
	"scifind-backend/internal/providers/semantic_scholar"
	"scifind-backend/internal/providers/tavily"
	"scifind-backend/internal/repository"
	"scifind-backend/internal/services"
)

// Application represents the complete application with all dependencies
type Application struct {
	Config          *config.Config
	Database        *repository.Database
	Messaging       *messaging.Client
	EmbeddedManager *embedded.Manager
	Services        *services.Container
	Handlers        *handlers.Container
	Router          *gin.Engine
	Logger          *slog.Logger
}

// NewApplication creates the main application instance
func NewApplication(
	cfg *config.Config,
	db *repository.Database,
	messaging *messaging.Client,
	embeddedManager *embedded.Manager,
	services *services.Container,
	handlers *handlers.Container,
	router *gin.Engine,
	logger *slog.Logger,
) *Application {
	return &Application{
		Config:          cfg,
		Database:        db,
		Messaging:       messaging,
		EmbeddedManager: embeddedManager,
		Services:        services,
		Handlers:        handlers,
		Router:          router,
		Logger:          logger,
	}
}

// Provider sets for Wire dependency injection
var ConfigProviderSet = wire.NewSet(
	config.LoadConfig,
	ProvideLogger,
)

var DatabaseProviderSet = wire.NewSet(
	ProvideDatabase,
	ProvideRepositories,
)

var MessagingProviderSet = wire.NewSet(
	ProvideEmbeddedManager,
	ProvideMessagingFromEmbedded,
)

var ServicesProviderSet = wire.NewSet(
	ProvideServices,
	ProvideProviderManager,
)

var HandlersProviderSet = wire.NewSet(
	ProvideHandlers,
)

var APIProviderSet = wire.NewSet(
	ProvideConcreteSearchService,
	ProvideConcretePaperService,
	ProvideConcreteAuthorService,
	ProvideConcreteHealthHandler,
	ProvideRouter,
)

// ApplicationProviderSet combines all provider sets
var ApplicationProviderSet = wire.NewSet(
	ConfigProviderSet,
	DatabaseProviderSet,
	MessagingProviderSet,
	ServicesProviderSet,
	HandlersProviderSet,
	APIProviderSet,
	NewApplication,
)

// Provider functions

// ProvideLogger creates a structured logger instance
func ProvideLogger(cfg *config.Config) (*slog.Logger, error) {
	logger, err := config.NewLogger(cfg)
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// ProvideDatabase creates a database instance
func ProvideDatabase(cfg *config.Config, logger *slog.Logger) (*repository.Database, error) {
	return repository.NewDatabase(cfg, logger)
}

// ProvideRepositories creates repository instances
func ProvideRepositories(db *repository.Database, logger *slog.Logger) *repository.Container {
	return repository.NewContainer(db.DB, logger)
}

// ProvideEmbeddedManager creates an embedded NATS manager
func ProvideEmbeddedManager(cfg *config.Config, logger *slog.Logger) (*embedded.Manager, error) {
	return embedded.NewManager(&cfg.NATS, logger)
}

// ProvideMessagingFromEmbedded provides messaging client from embedded manager
func ProvideMessagingFromEmbedded(embeddedManager *embedded.Manager) *messaging.Client {
	return embeddedManager.GetClient()
}

// ProvideProviderManager creates a provider manager instance
func ProvideProviderManager(logger *slog.Logger) providers.ProviderManager {
	managerConfig := providers.ManagerConfig{
		AggregationStrategy: providers.StrategyMerge,
		MaxConcurrency:      5,
		Timeout:             30 * time.Second,
	}
	manager := providers.NewManager(logger, managerConfig)

	// Initialize providers
	initializeProviders(manager, logger)
	return manager
}

// initializeProviders sets up all search providers
func initializeProviders(manager providers.ProviderManager, logger *slog.Logger) {
	// Initialize ArXiv provider
	arxivConfig := providers.ProviderConfig{
		Enabled:    true,
		BaseURL:    "https://export.arxiv.org/api/query",
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}
	arxivProvider := arxiv.NewProvider(arxivConfig, logger)
	manager.RegisterProvider("arxiv", arxivProvider)

	// Initialize Semantic Scholar provider
	ssConfig := providers.ProviderConfig{
		Enabled:    true,
		BaseURL:    "https://api.semanticscholar.org/graph/v1",
		Timeout:    15 * time.Second,
		MaxRetries: 3,
		APIKey:     "", // Optional for basic usage
	}
	ssProvider := semantic_scholar.NewProvider(ssConfig, logger)
	manager.RegisterProvider("semantic_scholar", ssProvider)

	// Initialize Exa provider (requires API key)
	exaConfig := providers.ProviderConfig{
		Enabled:    false, // Disabled by default, enable when API key is available
		BaseURL:    "https://api.exa.ai",
		Timeout:    20 * time.Second,
		MaxRetries: 3,
		APIKey:     "", // Must be configured
	}
	exaProvider := exa.NewProvider(exaConfig, logger)
	manager.RegisterProvider("exa", exaProvider)

	// Initialize Tavily provider (requires API key)
	tavilyConfig := providers.ProviderConfig{
		Enabled:    false, // Disabled by default, enable when API key is available
		BaseURL:    "https://api.tavily.com",
		Timeout:    25 * time.Second,
		MaxRetries: 3,
		APIKey:     "", // Must be configured
	}
	tavilyProvider := tavily.NewProvider(tavilyConfig, logger)
	manager.RegisterProvider("tavily", tavilyProvider)

	logger.Info("Search providers initialized",
		slog.Int("total_providers", len(manager.GetAllProviders())),
		slog.Int("enabled_providers", len(manager.GetEnabledProviders())))
}

// ProvideServices creates service instances
func ProvideServices(repos *repository.Container, messaging *messaging.Client, providerManager providers.ProviderManager, logger *slog.Logger) *services.Container {
	return services.NewContainer(repos, messaging, providerManager, logger)
}

// ProvideHandlers creates HTTP handler instances
func ProvideHandlers(services *services.Container, logger *slog.Logger) *handlers.Container {
	return handlers.NewContainer(services, logger)
}

// ProvideConcreteSearchService creates a concrete search service
func ProvideConcreteSearchService(repos *repository.Container, messaging *messaging.Client, providerManager providers.ProviderManager, logger *slog.Logger) *services.SearchService {
	return services.NewSearchService(repos.Search, repos.Paper, messaging, providerManager, logger).(*services.SearchService)
}

// ProvideConcretePaperService creates a concrete paper service
func ProvideConcretePaperService(repos *repository.Container, messaging *messaging.Client, logger *slog.Logger) *services.PaperService {
	return services.NewPaperService(repos.Paper, messaging, logger).(*services.PaperService)
}

// ProvideConcreteAuthorService creates a concrete author service
func ProvideConcreteAuthorService(repos *repository.Container, messaging *messaging.Client, logger *slog.Logger) *services.AuthorService {
	return services.NewAuthorService(repos.Author, repos.Paper, messaging, logger).(*services.AuthorService)
}

// ProvideConcreteHealthHandler creates a concrete health handler
func ProvideConcreteHealthHandler(services *services.Container, logger *slog.Logger) *handlers.HealthHandler {
	return handlers.NewHealthHandler(services.Health, logger)
}

// ProvideRouter creates the HTTP router
func ProvideRouter(
	searchService *services.SearchService,
	paperService *services.PaperService,
	authorService *services.AuthorService,
	healthHandler *handlers.HealthHandler,
	providerManager providers.ProviderManager,
	logger *slog.Logger,
) *gin.Engine {
	return api.NewRouter(
		searchService,
		paperService,
		authorService,
		healthHandler,
		logger,
	)
}

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
		cfg.NATS.Embedded.Enabled = true
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

// InitializeApplication creates a fully configured application using Wire
func InitializeApplication(ctx context.Context) (*Application, func(), error) {
	wire.Build(ApplicationProviderSet)
	return &Application{}, func() {}, nil
}

// InitializeDevelopmentApplication creates an application instance for development
func InitializeDevelopmentApplication(ctx context.Context) (*Application, func(), error) {
	wire.Build(
		ProvideDevelopmentConfig,
		ProvideLogger,
		ProvideDatabase,
		ProvideEmbeddedManager,
		ProvideMessagingFromEmbedded,
		ProvideRepositories,
		ProvideProviderManager,
		ProvideServices,
		ProvideHandlers,
		ProvideConcreteSearchService,
		ProvideConcretePaperService,
		ProvideConcreteAuthorService,
		ProvideConcreteHealthHandler,
		ProvideRouter,
		NewApplication,
	)
	return &Application{}, func() {}, nil
}

// InitializeTestApplication creates an application instance for testing
func InitializeTestApplication(ctx context.Context) (*Application, func(), error) {
	wire.Build(
		ProvideTestConfig,
		ProvideLogger,
		ProvideDatabase,
		ProvideEmbeddedManager,
		ProvideMessagingFromEmbedded,
		ProvideRepositories,
		ProvideProviderManager,
		ProvideServices,
		ProvideHandlers,
		ProvideConcreteSearchService,
		ProvideConcretePaperService,
		ProvideConcreteAuthorService,
		ProvideConcreteHealthHandler,
		ProvideRouter,
		NewApplication,
	)
	return &Application{}, func() {}, nil
}
