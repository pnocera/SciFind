package handlers

import (
	"log/slog"

	"scifind-backend/internal/services"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	service services.AnalyticsServiceInterface
	logger  *slog.Logger
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(service services.AnalyticsServiceInterface, logger *slog.Logger) AnalyticsHandlerInterface {
	return &AnalyticsHandler{
		service: service,
		logger:  logger,
	}
}