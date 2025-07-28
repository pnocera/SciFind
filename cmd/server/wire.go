//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"scifind-backend/internal/api"
	"scifind-backend/internal/api/handlers"
	"scifind-backend/internal/config"
	"scifind-backend/internal/messaging"
	"scifind-backend/internal/messaging/embedded"
	"scifind-backend/internal/repository"
	"scifind-backend/internal/services"
)

// Application represents the complete application with all dependencies
type Application struct {
	Config           *config.Config
	Database         *repository.Database
	Messaging        *messaging.Client
	EmbeddedManager  *embedded.Manager
	Services         *services.Container
	Handlers         *handlers.Container
	Router           *gin.Engine
	Logger           *slog.Logger
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

// ProvideServices creates service instances
func ProvideServices(repos *repository.Container, messaging *messaging.Client, logger *slog.Logger) *services.Container {
	return services.NewContainer(repos, messaging, logger)
}

// ProvideHandlers creates HTTP handler instances
func ProvideHandlers(services *services.Container, logger *slog.Logger) *handlers.Container {
	return handlers.NewContainer(services, logger)
}

// ProvideConcreteSearchService creates a concrete search service 
func ProvideConcreteSearchService(repos *repository.Container, messaging *messaging.Client, logger *slog.Logger) *services.SearchService {
	return services.NewSearchService(repos.Search, repos.Paper, messaging, logger).(*services.SearchService)
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

// InitializeApplication creates a fully configured application using Wire
func InitializeApplication(ctx context.Context) (*Application, func(), error) {
	wire.Build(ApplicationProviderSet)
	return &Application{}, func() {}, nil
}
