package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"scifind-backend/internal/services"
)

// Container holds all handler instances
type Container struct {
	Paper     PaperHandlerInterface
	Search    SearchHandlerInterface
	Analytics AnalyticsHandlerInterface
	Health    HealthHandlerInterface
}

// NewContainer creates a new handler container
func NewContainer(services *services.Container, logger *slog.Logger) *Container {
	return &Container{
		Paper:     NewPaperHandler(services.Paper, logger),
		Search:    NewSearchHandler(services.Search, logger),
		Analytics: NewAnalyticsHandler(services.Analytics, logger),
		Health:    NewHealthHandler(services.Health, logger),
	}
}

// Handler interfaces for dependency injection

type PaperHandlerInterface interface {
	ListPapers(c *gin.Context)
	GetPaper(c *gin.Context)
	CreatePaper(c *gin.Context)
	UpdatePaper(c *gin.Context)
	DeletePaper(c *gin.Context)
}

type SearchHandlerInterface interface {
	Search(c *gin.Context)
	GetPaper(c *gin.Context)
	GetProviders(c *gin.Context)
	GetProviderMetrics(c *gin.Context)
	ConfigureProvider(c *gin.Context)
}

type AnalyticsHandlerInterface interface {
	// Add analytics handler methods here
}

type HealthHandlerInterface interface {
	// Add health handler methods here
}