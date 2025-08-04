// Package main SciFIND Backend API
//
//	@title			SciFIND Backend API
//	@version		1.0.0
//	@description	This is the main API server for SciFIND, a scientific literature discovery platform. It provides endpoints for searching academic papers across multiple providers, managing papers and authors, and retrieving scientific literature metadata.
//	@termsOfService	https://scifind.ai/terms
//
//	@contact.name	SciFIND Support
//	@contact.email	support@scifind.ai
//	@contact.url	https://scifind.ai/support
//
//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT
//
//	@host		localhost:8080
//	@BasePath	/
//	@schemes	http https
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				API key for authentication
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"
	
	"scifind-backend/internal/mcp"
	"scifind-backend/internal/services"
	_ "scifind-backend/docs" // Import generated Swagger docs
)

//go:generate wire

func main() {
	// Create base context
	ctx := context.Background()

	// Initialize application using Wire dependency injection
	app, cleanup, err := InitializeApplication(ctx)
	if err != nil {
		slog.Error("Failed to initialize application", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer cleanup()

	logger := app.Logger
	config := app.Config
	embeddedManager := app.EmbeddedManager

	// Start embedded NATS manager if available and enabled
	if embeddedManager != nil && config.NATS.Embedded.Enabled {
		logger.Info("Starting embedded NATS manager...")
		if err := embeddedManager.Start(ctx); err != nil {
			logger.Error("Failed to start embedded NATS manager",
				slog.String("error", err.Error()),
				slog.String("configured_host", config.NATS.Embedded.Host),
				slog.Int("configured_port", config.NATS.Embedded.Port))
			logger.Error("Server startup failed: embedded NATS is enabled but could not start")
			os.Exit(1)
		} else {
			logger.Info("Embedded NATS manager started successfully")
		}
	}

	// Set server address
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	if addr == ":0" {
		addr = ":8080" // Default fallback
	}

	// Create HTTP server with the complete router
	server := &http.Server{
		Addr:           addr,
		Handler:        app.Router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Check if embedded NATS server is running (already started by Wire initialization)
	var embeddedServerRunning bool
	if embeddedManager != nil {
		embeddedServerRunning = embeddedManager.IsEmbeddedServerEnabled()
		logger.Info("Embedded NATS manager status",
			slog.Bool("embedded_server", embeddedServerRunning))
	}

	// Initialize simple MCP server
	var mcpServer *mcp.SimpleMCPServer
	mcpEnabled := true
	if mcpEnabled {
		mcpServer = mcp.NewSimpleMCPServer(
			app.Services.Search.(*services.SearchService),
			app.Services.Paper.(*services.PaperService), 
			app.Services.Author.(*services.AuthorService),
			logger,
		)
		logger.Info("MCP server initialized with 2 core tools (KISS approach)")
		
		// Start MCP server in separate goroutine for stdio
		go func() {
			logger.Info("Starting MCP server on stdio...")
			if err := mcpServer.ServeStdio(); err != nil {
				logger.Error("MCP server failed", slog.String("error", err.Error()))
			}
		}()
	}

	// Start HTTP server in goroutine
	go func() {
		logger.Info("Starting SciFIND Backend server",
			slog.String("addr", server.Addr),
			slog.String("mode", config.Server.Mode),
			slog.String("version", "1.0.0"))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Log successful startup
	logger.Info("SciFIND Backend startup complete",
		slog.String("http_addr", server.Addr),
		slog.Bool("database_connected", app.Database != nil),
		slog.Bool("messaging_connected", app.Messaging != nil && app.Messaging.IsConnected()),
		slog.Bool("embedded_nats_server", embeddedServerRunning),
		slog.Bool("mcp_enabled", mcpEnabled))

	// Log available endpoints
	logger.Info("Available endpoints",
		slog.String("health", "/health, /health/live, /health/ready"),
		slog.String("search", "/v1/search, /v1/search/papers/{provider}/{id}"),
		slog.String("papers", "/v1/papers, /v1/papers/{id}"),
		slog.String("authors", "/v1/authors, /v1/authors/{id}/papers"),
		slog.String("docs", "/docs"))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down SciFIND Backend...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server forced to shutdown", slog.String("error", err.Error()))
	} else {
		logger.Info("HTTP server shutdown gracefully")
	}

	// MCP server shutdown
	if mcpEnabled && mcpServer != nil {
		logger.Info("MCP server shutdown - stdio connection will close automatically")
	}

	// Close database connection
	if app.Database != nil {
		app.Database.Close()
		logger.Info("Database connection closed")
	}

	// Stop embedded NATS manager
	if embeddedManager != nil {
		if err := embeddedManager.Stop(shutdownCtx); err != nil {
			logger.Error("Failed to stop embedded NATS manager", slog.String("error", err.Error()))
		} else {
			logger.Info("Embedded NATS manager stopped")
		}
	}

	// Close NATS connection (if not using embedded manager)
	if app.Messaging != nil && embeddedManager == nil {
		app.Messaging.Close()
		logger.Info("NATS connection closed")
	}

	logger.Info("SciFIND Backend shutdown complete")
}
