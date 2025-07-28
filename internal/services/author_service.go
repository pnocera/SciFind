package services

import (
	"context"
	"fmt"
	"log/slog"

	"scifind-backend/internal/messaging"
	"scifind-backend/internal/models"
	"scifind-backend/internal/repository"
)

// AuthorService handles author-related business logic
type AuthorService struct {
	repo      repository.AuthorRepository
	paperRepo repository.PaperRepository
	messaging *messaging.Client
	logger    *slog.Logger
}

// NewAuthorService creates a new author service
func NewAuthorService(
	repo repository.AuthorRepository,
	paperRepo repository.PaperRepository,
	messaging *messaging.Client,
	logger *slog.Logger,
) AuthorServiceInterface {
	return &AuthorService{
		repo:      repo,
		paperRepo: paperRepo,
		messaging: messaging,
		logger:    logger,
	}
}

// GetByID retrieves an author by their ID
func (s *AuthorService) GetByID(ctx context.Context, id string) (*models.Author, error) {
	author, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get author by ID", slog.String("id", id), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get author: %w", err)
	}
	return author, nil
}

// Create creates a new author
func (s *AuthorService) Create(ctx context.Context, author *models.Author) error {
	if err := s.repo.Create(ctx, author); err != nil {
		s.logger.Error("Failed to create author", slog.String("name", author.Name), slog.String("error", err.Error()))
		return fmt.Errorf("failed to create author: %w", err)
	}
	
	// Publish author created event
	if s.messaging != nil {
		event := map[string]interface{}{
			"type":      "author_created",
			"author_id": author.ID,
			"name":      author.Name,
			"timestamp": author.CreatedAt,
		}
		if err := s.messaging.Publish(ctx, "authors.created", event); err != nil {
			s.logger.Warn("Failed to publish author created event", slog.String("error", err.Error()))
		}
	}
	
	return nil
}

// Update updates an existing author
func (s *AuthorService) Update(ctx context.Context, author *models.Author) error {
	if err := s.repo.Update(ctx, author); err != nil {
		s.logger.Error("Failed to update author", slog.String("id", author.ID), slog.String("error", err.Error()))
		return fmt.Errorf("failed to update author: %w", err)
	}
	
	// Publish author updated event
	if s.messaging != nil {
		event := map[string]interface{}{
			"type":      "author_updated",
			"author_id": author.ID,
			"name":      author.Name,
			"timestamp": author.UpdatedAt,
		}
		if err := s.messaging.Publish(ctx, "authors.updated", event); err != nil {
			s.logger.Warn("Failed to publish author updated event", slog.String("error", err.Error()))
		}
	}
	
	return nil
}

// Delete deletes an author by ID
func (s *AuthorService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete author", slog.String("id", id), slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete author: %w", err)
	}
	
	// Publish author deleted event
	if s.messaging != nil {
		event := map[string]interface{}{
			"type":      "author_deleted",
			"author_id": id,
			"timestamp": ctx.Value("timestamp"),
		}
		if err := s.messaging.Publish(ctx, "authors.deleted", event); err != nil {
			s.logger.Warn("Failed to publish author deleted event", slog.String("error", err.Error()))
		}
	}
	
	return nil
}

// List retrieves authors with optional filters
func (s *AuthorService) List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*models.Author, int, error) {
	// Simplified implementation - use search with empty query
	authors, total, err := s.repo.Search(ctx, "", nil, nil, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list authors", slog.Any("filters", filters), slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("failed to list authors: %w", err)
	}
	
	// Convert []models.Author to []*models.Author
	authorPtrs := make([]*models.Author, len(authors))
	for i := range authors {
		authorPtrs[i] = &authors[i]
	}
	return authorPtrs, int(total), nil
}

// Search searches for authors by query
func (s *AuthorService) Search(ctx context.Context, query string, limit, offset int) ([]*models.Author, int, error) {
	authors, total, err := s.repo.Search(ctx, query, nil, nil, limit, offset)
	if err != nil {
		s.logger.Error("Failed to search authors", slog.String("query", query), slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("failed to search authors: %w", err)
	}
	
	// Convert []models.Author to []*models.Author
	authorPtrs := make([]*models.Author, len(authors))
	for i := range authors {
		authorPtrs[i] = &authors[i]
	}
	return authorPtrs, int(total), nil
}

// GetPapers retrieves papers by an author
func (s *AuthorService) GetPapers(ctx context.Context, authorID string, limit, offset int) ([]*models.Paper, int, error) {
	// Simplified implementation - use author paper relationship
	papers, err := s.paperRepo.GetAuthorPapers(ctx, authorID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get papers by author", slog.String("authorID", authorID), slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("failed to get papers by author: %w", err)
	}
	
	// Convert []models.Paper to []*models.Paper
	paperPtrs := make([]*models.Paper, len(papers))
	for i := range papers {
		paperPtrs[i] = &papers[i]
	}
	return paperPtrs, len(papers), nil
}

// Health checks the health of the author service
func (s *AuthorService) Health(ctx context.Context) error {
	// Basic health check - service is operational
	if s.repo == nil {
		return fmt.Errorf("author repository not initialized")
	}
	
	// Check messaging health if available
	if s.messaging != nil && !s.messaging.IsConnected() {
		s.logger.Warn("Messaging not connected")
		// Don't fail the health check for messaging issues
	}
	
	return nil
}