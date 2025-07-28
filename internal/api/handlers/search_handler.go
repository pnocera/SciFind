package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"scifind-backend/internal/errors"
	"scifind-backend/internal/providers"
	"scifind-backend/internal/services"
)

// SearchHandler handles search-related HTTP requests
type SearchHandler struct {
	service services.SearchServiceInterface
	logger  *slog.Logger
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(service services.SearchServiceInterface, logger *slog.Logger) SearchHandlerInterface {
	return &SearchHandler{
		service: service,
		logger:  logger,
	}
}

// Search performs a search across providers
// @Summary Search for academic papers
// @Description Search for academic papers across multiple providers
// @Tags search
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param limit query int false "Number of results to return (default: 20, max: 100)"
// @Param offset query int false "Number of results to skip (default: 0)"
// @Param providers query string false "Comma-separated list of providers (arxiv,semantic_scholar,exa,tavily)"
// @Param date_from query string false "Start date filter (YYYY-MM-DD)"
// @Param date_to query string false "End date filter (YYYY-MM-DD)"
// @Param author query string false "Author filter"
// @Param journal query string false "Journal filter"
// @Param category query string false "Category filter"
// @Success 200 {object} services.SearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /search [get]
func (h *SearchHandler) Search(c *gin.Context) {
	// Generate request ID for tracking
	requestID := uuid.New().String()

	// Parse query parameters
	searchReq, err := h.parseSearchRequest(c, requestID)
	if err != nil {
		h.logger.Warn("Invalid search request",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "Invalid request",
			Message:   err.Error(),
			RequestID: requestID,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Log search request
	h.logger.Info("Search request received",
		slog.String("request_id", requestID),
		slog.String("query", searchReq.Query),
		slog.Int("limit", searchReq.Limit),
		slog.Any("providers", searchReq.Providers))

	// Execute search
	response, err := h.service.Search(c.Request.Context(), searchReq)
	if err != nil {
		h.logger.Error("Search failed",
			slog.String("request_id", requestID),
			slog.String("error", err.Error()))

		statusCode := http.StatusInternalServerError
		if errors.IsValidationError(err) {
			statusCode = http.StatusBadRequest
		} else if errors.IsTimeoutError(err) {
			statusCode = http.StatusRequestTimeout
		} else if errors.IsRateLimitError(err) {
			statusCode = http.StatusTooManyRequests
		}

		c.JSON(statusCode, ErrorResponse{
			Error:     "Search failed",
			Message:   err.Error(),
			RequestID: requestID,
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Log successful search
	h.logger.Info("Search completed successfully",
		slog.String("request_id", requestID),
		slog.Int("results", response.ResultCount),
		slog.Duration("duration", response.Duration))

	c.JSON(http.StatusOK, response)
}

// GetPaper retrieves a specific paper by provider and ID
// @Summary Get a specific paper
// @Description Retrieve a specific paper by provider name and paper ID
// @Tags search
// @Accept json
// @Produce json
// @Param provider path string true "Provider name" Enums(arxiv,semantic_scholar,exa,tavily)
// @Param id path string true "Paper ID"
// @Success 200 {object} services.PaperResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /search/papers/{provider}/{id} [get]
func (h *SearchHandler) GetPaper(c *gin.Context) {
	provider := c.Param("provider")
	paperID := c.Param("id")

	// Validate parameters
	if provider == "" || paperID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:     "Invalid request",
			Message:   "Provider and paper ID are required",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	// Get paper from service
	paper, err := h.service.GetPaper(c.Request.Context(), provider, paperID)
	if err != nil {
		h.logger.Error("Failed to get paper",
			slog.String("provider", provider),
			slog.String("paper_id", paperID),
			slog.String("error", err.Error()))

		statusCode := http.StatusInternalServerError
		if err != nil && strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if errors.IsValidationError(err) {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   "Failed to get paper",
			Message: err.Error(),
		})
		return
	}

	response := &services.PaperResponse{
		Paper:     paper,
		Source:    provider,
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetProviders returns information about available search providers
// @Summary Get available search providers
// @Description Get information about available search providers and their status
// @Tags search
// @Accept json
// @Produce json
// @Success 200 {object} services.ProviderStatusResponse
// @Failure 500 {object} ErrorResponse
// @Router /search/providers [get]
func (h *SearchHandler) GetProviders(c *gin.Context) {
	status, err := h.service.GetProviderStatus(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get provider status", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get provider status",
			Message: err.Error(),
		})
		return
	}

	response := gin.H{
		"providers": status,
		"timestamp": time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetProviderMetrics returns metrics for search providers
// @Summary Get search provider metrics
// @Description Get performance metrics for search providers
// @Tags search
// @Accept json
// @Produce json
// @Param provider query string false "Specific provider name"
// @Success 200 {object} services.ProviderMetricsResponse
// @Failure 500 {object} ErrorResponse
// @Router /search/providers/metrics [get]
func (h *SearchHandler) GetProviderMetrics(c *gin.Context) {
	metrics, err := h.service.GetProviderMetrics(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get provider metrics", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get provider metrics",
			Message: err.Error(),
		})
		return
	}

	// Filter by provider if specified
	providerName := c.Query("provider")
	if providerName != "" {
		if providerMetrics, exists := metrics[providerName]; exists {
			if pm, ok := providerMetrics.(providers.ProviderMetrics); ok {
				metrics = map[string]interface{}{providerName: pm}
			}
		} else {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Provider not found",
				Message: fmt.Sprintf("Provider '%s' not found", providerName),
			})
			return
		}
	}

	response := gin.H{
		"providers": metrics,
		"timestamp": time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// ConfigureProvider updates provider configuration
// @Summary Configure a search provider
// @Description Update the configuration of a specific search provider
// @Tags search
// @Accept json
// @Produce json
// @Param provider path string true "Provider name" Enums(arxiv,semantic_scholar,exa,tavily)
// @Param config body providers.ProviderConfig true "Provider configuration"
// @Success 200 {object} services.ProviderConfigResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /search/providers/{provider}/configure [put]
func (h *SearchHandler) ConfigureProvider(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Provider name is required",
		})
		return
	}

	var config providers.ProviderConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Update provider configuration
	err := h.service.ConfigureProvider(c.Request.Context(), provider, config)
	if err != nil {
		h.logger.Error("Failed to configure provider",
			slog.String("provider", provider),
			slog.String("error", err.Error()))

		statusCode := http.StatusInternalServerError
		if err != nil && strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if errors.IsValidationError(err) {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   "Failed to configure provider",
			Message: err.Error(),
		})
		return
	}

	// Get updated status
	status, err := h.service.GetProviderStatus(c.Request.Context())
	if err != nil {
		h.logger.Warn("Failed to get updated provider status", slog.String("error", err.Error()))
	}

	var providerStatus providers.ProviderStatus
	if status != nil {
		if ps, exists := status[provider]; exists {
			if pStatus, ok := ps.(providers.ProviderStatus); ok {
				providerStatus = pStatus
			}
		}
	}

	response := &services.ProviderConfigResponse{
		ProviderName: provider,
		Status:       providerStatus,
		Message:      "Provider configuration updated successfully",
		Timestamp:    time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// Helper methods

func (h *SearchHandler) parseSearchRequest(c *gin.Context, requestID string) (*services.SearchRequest, error) {
	req := &services.SearchRequest{
		RequestID: requestID,
		Query:     c.Query("query"),
		Filters:   make(map[string]string),
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %v", err)
		}
		req.Limit = limit
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return nil, fmt.Errorf("invalid offset: %v", err)
		}
		req.Offset = offset
	}

	// Parse providers
	if providersStr := c.Query("providers"); providersStr != "" {
		providers := splitAndTrim(providersStr, ",")
		req.Providers = providers
	}

	// Parse date filters
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		dateFrom, err := time.Parse("2006-01-02", dateFromStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date_from format: %v", err)
		}
		req.DateFrom = &dateFrom
	}

	if dateToStr := c.Query("date_to"); dateToStr != "" {
		dateTo, err := time.Parse("2006-01-02", dateToStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date_to format: %v", err)
		}
		req.DateTo = &dateTo
	}

	// Add other filters
	if author := c.Query("author"); author != "" {
		req.Filters["author"] = author
	}

	if journal := c.Query("journal"); journal != "" {
		req.Filters["journal"] = journal
	}

	if category := c.Query("category"); category != "" {
		req.Filters["category"] = category
	}

	if subject := c.Query("subject"); subject != "" {
		req.Filters["subject"] = subject
	}

	// Set defaults and validate
	req.SetDefaults()
	if err := req.ValidateSearchRequest(); err != nil {
		return nil, err
	}

	return req, nil
}

// Helper function to split and trim strings
func splitAndTrim(s, sep string) []string {
	parts := []string{}
	for _, part := range strings.Split(s, sep) {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}