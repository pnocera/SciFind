package services

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"scifind-backend/internal/messaging"
	"scifind-backend/internal/repository"
)

// HealthService handles health checks for all system components
type HealthService struct {
	repos     *repository.Container
	messaging *messaging.Client
	logger    *slog.Logger
	startTime time.Time
}

// NewHealthService creates a new health service
func NewHealthService(repos *repository.Container, messaging *messaging.Client, logger *slog.Logger) HealthServiceInterface {
	return &HealthService{
		repos:     repos,
		messaging: messaging,
		logger:    logger,
		startTime: time.Now(),
	}
}

// Health checks the health of the health service itself
func (s *HealthService) Health(ctx context.Context) error {
	// Basic health check - service is running
	return nil
}

// DatabaseHealth checks the health of the database connection
func (s *HealthService) DatabaseHealth(ctx context.Context) error {
	if s.repos == nil {
		return fmt.Errorf("database repositories not initialized")
	}
	
	// Check basic repository availability
	var errors []string
	
	if s.repos.Paper == nil {
		errors = append(errors, "paper repository not initialized")
	}
	
	if s.repos.Author == nil {
		errors = append(errors, "author repository not initialized")
	}
	
	if s.repos.Category == nil {
		errors = append(errors, "category repository not initialized")
	}
	
	if s.repos.Search == nil {
		errors = append(errors, "search repository not initialized")
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("database health check failed: %v", errors)
	}
	
	return nil
}

// MessagingHealth checks the health of the messaging system
func (s *HealthService) MessagingHealth(ctx context.Context) error {
	if s.messaging == nil {
		return fmt.Errorf("messaging client not initialized")
	}
	
	// Check if NATS connection is alive
	if !s.messaging.IsConnected() {
		return fmt.Errorf("NATS connection is not established")
	}
	
	return nil
}

// ExternalServicesHealth checks the health of external services
func (s *HealthService) ExternalServicesHealth(ctx context.Context) map[string]error {
	results := make(map[string]error)
	
	// Add checks for external services like provider APIs
	results["arxiv"] = s.checkExternalService(ctx, "arxiv")
	results["semantic_scholar"] = s.checkExternalService(ctx, "semantic_scholar")
	results["exa"] = s.checkExternalService(ctx, "exa")
	results["tavily"] = s.checkExternalService(ctx, "tavily")
	
	return results
}

// GetSystemInfo returns comprehensive system information
func (s *HealthService) GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	memInfo := MemoryInfo{
		Allocated: m.Alloc,
		Total:     m.TotalAlloc,
		System:    m.Sys,
		GCRuns:    m.NumGC,
	}
	
	dbInfo := DatabaseInfo{
		Connected: s.repos != nil,
		Type:      "postgresql", // Default, should be configurable
		Connections: map[string]int{
			"active": 0, // Would need to get from database pool
			"idle":   0,
		},
	}
	
	// Check database health
	if dbErr := s.DatabaseHealth(ctx); dbErr != nil {
		dbInfo.Connected = false
	}
	
	services := map[string]bool{
		"database":  dbInfo.Connected,
		"messaging": s.messaging != nil && s.messaging.IsConnected(),
		"health":    true,
	}
	
	return &SystemInfo{
		Version:   "1.0.0", // Should be injected at build time
		Uptime:    time.Since(s.startTime),
		Memory:    memInfo,
		Database:  dbInfo,
		Services:  services,
		Timestamp: time.Now(),
	}, nil
}

// checkExternalService performs a basic health check on external services
func (s *HealthService) checkExternalService(ctx context.Context, serviceName string) error {
	// This is a placeholder implementation
	// In a real implementation, you would make actual HTTP requests to check service availability
	switch serviceName {
	case "arxiv":
		// Could make a simple request to ArXiv API
		return nil
	case "semantic_scholar":
		// Could make a simple request to Semantic Scholar API
		return nil
	case "exa", "tavily":
		// These require API keys, so might just check configuration
		return nil
	default:
		return fmt.Errorf("unknown service: %s", serviceName)
	}
}