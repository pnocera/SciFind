//go:build wireinject
// +build wireinject

package wire

import (
	"context"
	"log/slog"

	"scifind-backend/internal/api/handlers"
	"scifind-backend/internal/config"
	"scifind-backend/internal/messaging"
	"scifind-backend/internal/repository"
	"scifind-backend/internal/services"

	"github.com/google/wire"
)

// Application represents the complete application with all dependencies
type Application struct {
	Config     *config.Config
	Database   *repository.Database
	Messaging  *messaging.Client
	Services   *services.Container
	Handlers   *handlers.Container
	Logger     *slog.Logger
}

// InitializeApplication creates a fully configured application instance
func InitializeApplication(ctx context.Context) (*Application, func(), error) {
	wire.Build(
		// Configuration
		config.LoadConfig,
		
		// Logger
		ProvideLogger,
		
		// Database
		ProvideDatabase,
		
		// Messaging
		ProvideMessaging,
		
		// Repositories
		ProvideRepositories,
		
		// Services
		ProvideServices,
		
		// Handlers
		ProvideHandlers,
		
		// Application
		ProvideApplication,
		
	)
	return nil, nil, nil
}

// InitializeDevelopmentApplication creates an application instance for development
func InitializeDevelopmentApplication(ctx context.Context) (*Application, func(), error) {
	wire.Build(
		// Configuration with development overrides
		ProvideDevelopmentConfig,
		
		// Development logger (more verbose)
		ProvideDevelopmentLogger,
		
		// SQLite database for development
		ProvideDevelopmentDatabase,
		
		// Local NATS for development
		ProvideDevelopmentMessaging,
		
		// Mock providers for development
		ProvideDevelopmentRepositories,
		
		// Services with development configuration
		ProvideDevelopmentServices,
		
		// Development handlers
		ProvideHandlers,
		
		// Application
		ProvideApplication,
		
	)
	return nil, nil, nil
}

// InitializeTestApplication creates an application instance for testing
func InitializeTestApplication(ctx context.Context) (*Application, func(), error) {
	wire.Build(
		// Test configuration
		ProvideTestConfig,
		
		// Test logger (silent)
		ProvideTestLogger,
		
		// In-memory database for testing
		ProvideTestDatabase,
		
		// Mock messaging for testing
		ProvideTestMessaging,
		
		// Mock repositories for testing
		ProvideTestRepositories,
		
		// Mock services for testing
		ProvideTestServices,
		
		// Test handlers
		ProvideTestHandlers,
		
		// Application
		ProvideApplication,
		
	)
	return nil, nil, nil
}

// InitializeDatabaseOnly creates only database dependencies for migrations
func InitializeDatabaseOnly(ctx context.Context) (*repository.Database, func(), error) {
	wire.Build(
		config.LoadConfig,
		ProvideLogger,
		ProvideDatabase,
	)
	return nil, nil, nil
}

// InitializeMessagingOnly creates only messaging dependencies for testing
func InitializeMessagingOnly(ctx context.Context) (*messaging.Client, func(), error) {
	wire.Build(
		config.LoadConfig,
		ProvideLogger,
		ProvideMessaging,
	)
	return nil, nil, nil
}