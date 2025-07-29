package api

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	swaggerFiles "github.com/swaggo/files"
	
	_ "scifind-backend/docs"
	"scifind-backend/internal/api/handlers"
	"scifind-backend/internal/api/middleware"
	"scifind-backend/internal/services"
)

// Router creates and configures the HTTP router
func NewRouter(
	searchService *services.SearchService,
	paperService *services.PaperService,
	authorService *services.AuthorService,
	healthHandler *handlers.HealthHandler,
	logger *slog.Logger,
) *gin.Engine {
	// Set Gin mode based on environment
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()

	// Global middleware
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.CorsMiddleware(middleware.DefaultCorsConfig()))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.StructuredLoggingMiddleware(logger))
	router.Use(gin.Recovery())

	// Register health endpoints first (without auth)
	healthHandler.RegisterRoutes(router)

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Search endpoints
		search := v1.Group("/search")
		{
			searchHandler := handlers.NewSearchHandler(searchService, logger)
			search.GET("", searchHandler.Search)
			search.GET("/papers/:provider/:id", searchHandler.GetPaper)
			search.GET("/providers", searchHandler.GetProviders)
			search.GET("/providers/metrics", searchHandler.GetProviderMetrics)
			search.PUT("/providers/:provider/configure", searchHandler.ConfigureProvider)
		}

		// Paper endpoints
		papers := v1.Group("/papers")
		{
			paperHandler := handlers.NewPaperHandler(paperService, logger)
			papers.GET("", paperHandler.ListPapers)
			papers.POST("", paperHandler.CreatePaper)
			papers.GET("/:id", paperHandler.GetPaper)
			papers.PUT("/:id", paperHandler.UpdatePaper)
			papers.DELETE("/:id", paperHandler.DeletePaper)
		}

		// Author endpoints
		authors := v1.Group("/authors")
		{
			authorHandler := handlers.NewAuthorHandler(authorService, logger)
			authors.GET("", authorHandler.ListAuthors)
			authors.GET("/:id", authorHandler.GetAuthor)
			authors.GET("/:id/papers", authorHandler.GetAuthorPapers)
		}
	}

	// Swagger documentation endpoints
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(301, "/swagger/index.html")
	})
	
	// Legacy documentation endpoint (redirect to Swagger)
	router.GET("/docs", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "SciFIND API Documentation",
			"version": "1.0.0",
			"swagger_ui": "/swagger/index.html",
			"openapi_spec": "/swagger/doc.json",
			"endpoints": gin.H{
				"health":  "/health",
				"search":  "/v1/search",
				"papers":  "/v1/papers",
				"authors": "/v1/authors",
			},
			"mcp_server": gin.H{
				"description": "This server also supports Model Context Protocol",
				"methods": []string{"search", "get_paper", "list_capabilities", "get_schema", "ping"},
			},
		})
	})

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "SciFIND Backend",
			"version": "1.0.0",
			"status":  "running",
			"docs":    "/docs",
			"health":  "/health",
		})
	})

	return router
}

// SetupHandlers creates and returns all HTTP handlers
func SetupHandlers(
	searchService *services.SearchService,
	paperService *services.PaperService,
	authorService *services.AuthorService,
	healthService services.HealthServiceInterface,
	logger *slog.Logger,
) (handlers.SearchHandlerInterface, handlers.PaperHandlerInterface, *handlers.AuthorHandler, *handlers.HealthHandler) {
	
	searchHandler := handlers.NewSearchHandler(searchService, logger)
	paperHandler := handlers.NewPaperHandler(paperService, logger)
	authorHandler := handlers.NewAuthorHandler(authorService, logger)
	healthHandler := handlers.NewHealthHandler(healthService, logger)
	
	return searchHandler, paperHandler, authorHandler, healthHandler
}