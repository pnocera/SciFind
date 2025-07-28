package services

import (
	"context"
	"fmt"
	"log/slog"

	"scifind-backend/internal/messaging"
	"scifind-backend/internal/models"
	"scifind-backend/internal/repository"
)

// PaperService handles paper-related business logic
type PaperService struct {
	repo      repository.PaperRepository
	messaging *messaging.Client
	logger    *slog.Logger
}

// NewPaperService creates a new paper service
func NewPaperService(repo repository.PaperRepository, messaging *messaging.Client, logger *slog.Logger) PaperServiceInterface {
	return &PaperService{
		repo:      repo,
		messaging: messaging,
		logger:    logger,
	}
}

// GetByID retrieves a paper by its ID
func (s *PaperService) GetByID(ctx context.Context, id string) (*models.Paper, error) {
	paper, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get paper by ID", slog.String("id", id), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get paper: %w", err)
	}
	return paper, nil
}

// Create creates a new paper
func (s *PaperService) Create(ctx context.Context, paper *models.Paper) error {
	if err := s.repo.Create(ctx, paper); err != nil {
		s.logger.Error("Failed to create paper", slog.String("title", paper.Title), slog.String("error", err.Error()))
		return fmt.Errorf("failed to create paper: %w", err)
	}
	
	// Publish paper created event
	if s.messaging != nil {
		event := map[string]interface{}{
			"type":      "paper_created",
			"paper_id":  paper.ID,
			"title":     paper.Title,
			"timestamp": paper.CreatedAt,
		}
		if err := s.messaging.Publish(ctx, "papers.created", event); err != nil {
			s.logger.Warn("Failed to publish paper created event", slog.String("error", err.Error()))
		}
	}
	
	return nil
}

// Update updates an existing paper
func (s *PaperService) Update(ctx context.Context, paper *models.Paper) error {
	if err := s.repo.Update(ctx, paper); err != nil {
		s.logger.Error("Failed to update paper", slog.String("id", paper.ID), slog.String("error", err.Error()))
		return fmt.Errorf("failed to update paper: %w", err)
	}
	
	// Publish paper updated event
	if s.messaging != nil {
		event := map[string]interface{}{
			"type":      "paper_updated",
			"paper_id":  paper.ID,
			"title":     paper.Title,
			"timestamp": paper.UpdatedAt,
		}
		if err := s.messaging.Publish(ctx, "papers.updated", event); err != nil {
			s.logger.Warn("Failed to publish paper updated event", slog.String("error", err.Error()))
		}
	}
	
	return nil
}

// Delete deletes a paper by ID
func (s *PaperService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete paper", slog.String("id", id), slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete paper: %w", err)
	}
	
	// Publish paper deleted event
	if s.messaging != nil {
		event := map[string]interface{}{
			"type":      "paper_deleted",
			"paper_id":  id,
			"timestamp": ctx.Value("timestamp"),
		}
		if err := s.messaging.Publish(ctx, "papers.deleted", event); err != nil {
			s.logger.Warn("Failed to publish paper deleted event", slog.String("error", err.Error()))
		}
	}
	
	return nil
}

// List retrieves papers with optional filters
func (s *PaperService) List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.Paper, int, error) {
	// Simplified implementation - would use a proper list method when available
	papers, total, err := s.repo.Search(ctx, "", nil, nil, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list papers", slog.Any("filters", filters), slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("failed to list papers: %w", err)
	}
	
	// Convert []models.Paper to []*models.Paper
	paperPtrs := make([]*models.Paper, len(papers))
	for i := range papers {
		paperPtrs[i] = &papers[i]
	}
	return paperPtrs, int(total), nil
}

// Search searches for papers by query
func (s *PaperService) Search(ctx context.Context, query string, limit, offset int) ([]*models.Paper, int, error) {
	papers, total, err := s.repo.Search(ctx, query, nil, nil, limit, offset)
	if err != nil {
		s.logger.Error("Failed to search papers", slog.String("query", query), slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("failed to search papers: %w", err)
	}
	
	// Convert []models.Paper to []*models.Paper
	paperPtrs := make([]*models.Paper, len(papers))
	for i := range papers {
		paperPtrs[i] = &papers[i]
	}
	return paperPtrs, int(total), nil
}

// GetByProvider retrieves a paper by provider and source ID
func (s *PaperService) GetByProvider(ctx context.Context, provider, sourceID string) (*models.Paper, error) {
	// Simplified implementation - would search by provider-specific ID when available
	// For now, try to find by ArXiv ID if it's ArXiv
	if provider == "arxiv" {
		return s.repo.GetByArxivID(ctx, sourceID)
	}
	
	s.logger.Warn("GetByProvider not fully implemented", slog.String("provider", provider), slog.String("sourceID", sourceID))
	return nil, fmt.Errorf("provider %s not supported yet", provider)
}

// Health checks the health of the paper service
func (s *PaperService) Health(ctx context.Context) error {
	// Basic health check - service is operational
	// TODO: Add proper repository health check when available
	if s.repo == nil {
		return fmt.Errorf("paper repository not initialized")
	}
	
	// Check messaging health if available
	if s.messaging != nil && !s.messaging.IsConnected() {
		s.logger.Warn("Messaging not connected")
		// Don't fail the health check for messaging issues
	}
	
	return nil
}