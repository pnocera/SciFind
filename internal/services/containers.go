package services

import (
	"context"
	"log/slog"

	"scifind-backend/internal/messaging"
	"scifind-backend/internal/providers"
	"scifind-backend/internal/repository"
)

// Container holds all service instances
type Container struct {
	Paper     PaperServiceInterface
	Search    SearchServiceInterface
	Analytics AnalyticsServiceInterface
	Health    HealthServiceInterface
	Author    AuthorServiceInterface
}

// NewContainer creates a new service container
func NewContainer(repos *repository.Container, messaging *messaging.Client, providerManager providers.ProviderManager, logger *slog.Logger) *Container {
	return &Container{
		Paper:     NewPaperService(repos.Paper, messaging, logger),
		Search:    NewSearchService(repos.Search, repos.Paper, messaging, providerManager, logger),
		Analytics: NewAnalyticsService(repos.Search, messaging, logger),
		Health:    NewHealthService(repos, messaging, logger),
		Author:    NewAuthorService(repos.Author, repos.Paper, messaging, logger),
	}
}

// HealthCheck checks all services
func (c *Container) HealthCheck(ctx context.Context) map[string]error {
	return map[string]error{
		"paper":     c.checkServiceHealth(ctx, "paper"),
		"search":    c.checkServiceHealth(ctx, "search"),
		"analytics": c.checkServiceHealth(ctx, "analytics"),
		"health":    c.checkServiceHealth(ctx, "health"),
	}
}

func (c *Container) checkServiceHealth(ctx context.Context, serviceName string) error {
	// Basic service availability check
	switch serviceName {
	case "paper":
		return c.Paper.Health(ctx)
	case "search":
		return c.Search.Health(ctx)
	case "analytics":
		return c.Analytics.Health(ctx)
	case "health":
		return c.Health.Health(ctx)
	default:
		return nil
	}
}

// Note: Service interfaces are defined in interfaces.go
