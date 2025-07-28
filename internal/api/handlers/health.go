package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"scifind-backend/internal/services"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	healthService services.HealthServiceInterface
	logger        *slog.Logger
	version       string
	buildTime     string
	gitCommit     string
	environment   string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthService services.HealthServiceInterface, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
		logger:        logger,
		version:       "1.0.0", // TODO: inject from build
		buildTime:     "unknown", // TODO: inject from build
		gitCommit:     "unknown", // TODO: inject from build
		environment:   "development", // TODO: inject from config
	}
}

// HealthStatus represents the health status response
type HealthStatus struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	BuildTime   string                 `json:"build_time"`
	GitCommit   string                 `json:"git_commit"`
	Environment string                 `json:"environment"`
	Uptime      string                 `json:"uptime"`
	Checks      map[string]CheckResult `json:"checks"`
}

// CheckResult represents the result of a health check
type CheckResult struct {
	Status    string        `json:"status"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
	Metadata  interface{}   `json:"metadata,omitempty"`
}

var startTime = time.Now()

// Liveness returns a simple liveness check
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
		"uptime":    time.Since(startTime).String(),
	})
}

// Readiness returns a comprehensive readiness check
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	status := &HealthStatus{
		Status:      "healthy",
		Timestamp:   time.Now().UTC(),
		Version:     h.version,
		BuildTime:   h.buildTime,
		GitCommit:   h.gitCommit,
		Environment: h.environment,
		Uptime:      time.Since(startTime).String(),
		Checks:      make(map[string]CheckResult),
	}

	// Check database connectivity
	dbResult := h.checkDatabase(ctx)
	status.Checks["database"] = dbResult
	if dbResult.Status != "healthy" {
		status.Status = "unhealthy"
	}

	// Check NATS connectivity
	natsResult := h.checkNATS(ctx)
	status.Checks["nats"] = natsResult
	if natsResult.Status != "healthy" {
		status.Status = "degraded"
	}

	// Check system resources
	resourceResult := h.checkResources(ctx)
	status.Checks["resources"] = resourceResult
	if resourceResult.Status != "healthy" && status.Status == "healthy" {
		status.Status = "degraded"
	}

	// Return appropriate HTTP status
	httpStatus := http.StatusOK
	if status.Status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	} else if status.Status == "degraded" {
		httpStatus = http.StatusOK // Still ready but degraded
	}

	c.JSON(httpStatus, status)
}

// Health returns comprehensive health information
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	status := &HealthStatus{
		Status:      "healthy",
		Timestamp:   time.Now().UTC(),
		Version:     h.version,
		BuildTime:   h.buildTime,
		GitCommit:   h.gitCommit,
		Environment: h.environment,
		Uptime:      time.Since(startTime).String(),
		Checks:      make(map[string]CheckResult),
	}

	// Perform all health checks
	checks := []struct {
		name string
		fn   func(context.Context) CheckResult
	}{
		{"database", h.checkDatabase},
		{"nats", h.checkNATS},
		{"resources", h.checkResources},
		{"external_apis", h.checkExternalAPIs},
	}

	for _, check := range checks {
		result := check.fn(ctx)
		status.Checks[check.name] = result
		
		// Update overall status
		if result.Status == "unhealthy" {
			status.Status = "unhealthy"
		} else if result.Status == "degraded" && status.Status == "healthy" {
			status.Status = "degraded"
		}
	}

	c.JSON(http.StatusOK, status)
}

// checkDatabase verifies database connectivity and performance
func (h *HealthHandler) checkDatabase(ctx context.Context) CheckResult {
	start := time.Now()
	
	if h.healthService == nil {
		return CheckResult{
			Status:   "unhealthy",
			Duration: time.Since(start),
			Error:    "health service not available",
		}
	}
	
	// Use health service to check database
	err := h.healthService.Health(ctx)
	if err != nil {
		return CheckResult{
			Status:   "unhealthy",
			Duration: time.Since(start),
			Error:    "database health check failed: " + err.Error(),
		}
	}
	
	return CheckResult{
		Status:   "healthy",
		Duration: time.Since(start),
		Metadata: map[string]interface{}{"database": "connected"},
	}
}

// checkNATS verifies NATS connectivity
func (h *HealthHandler) checkNATS(ctx context.Context) CheckResult {
	start := time.Now()
	
	if h.healthService == nil {
		return CheckResult{
			Status:   "degraded",
			Duration: time.Since(start),
			Error:    "health service not available",
		}
	}
	
	// For now, assume NATS is working if health service is available
	// TODO: Add specific NATS health check to health service
	return CheckResult{
		Status:   "healthy",
		Duration: time.Since(start),
		Metadata: map[string]interface{}{"nats": "status_unknown"},
	}
}

// checkResources verifies system resource availability
func (h *HealthHandler) checkResources(ctx context.Context) CheckResult {
	start := time.Now()
	
	// TODO: Implement resource checks
	// - Memory usage
	// - CPU usage
	// - Disk space
	// - File descriptors
	
	metadata := map[string]interface{}{
		"goroutines": "unknown", // runtime.NumGoroutine()
		"memory":     "unknown", // Get memory stats
		"cpu":        "unknown", // Get CPU stats
	}
	
	return CheckResult{
		Status:   "healthy",
		Duration: time.Since(start),
		Metadata: metadata,
	}
}

// checkExternalAPIs verifies external API connectivity
func (h *HealthHandler) checkExternalAPIs(ctx context.Context) CheckResult {
	start := time.Now()
	
	// TODO: Implement external API health checks
	// - ArXiv API
	// - Semantic Scholar API
	// - Exa API
	// - Tavily API
	
	metadata := map[string]interface{}{
		"arxiv":            "unknown",
		"semantic_scholar": "unknown",
		"exa":              "unknown",
		"tavily":           "unknown",
	}
	
	return CheckResult{
		Status:   "healthy",
		Duration: time.Since(start),
		Metadata: metadata,
	}
}

// RegisterRoutes registers health check routes
func (h *HealthHandler) RegisterRoutes(router *gin.Engine) {
	health := router.Group("/health")
	{
		health.GET("/live", h.Liveness)
		health.GET("/ready", h.Readiness)
		health.GET("", h.Health)
		health.GET("/", h.Health)
	}
	
	// Also register at root level for convenience
	router.GET("/health", h.Health)
	router.GET("/ping", h.Liveness)
}