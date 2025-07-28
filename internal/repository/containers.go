package repository

import (
	"log/slog"

	"gorm.io/gorm"
)

// Container holds all repository instances
type Container struct {
	Paper    PaperRepository
	Author   AuthorRepository
	Category CategoryRepository
	Search   SearchRepository
}

// NewContainer creates a new repository container
func NewContainer(db *gorm.DB, logger *slog.Logger) *Container {
	return &Container{
		Paper:    NewPaperRepository(db, logger),
		Author:   NewAuthorRepository(db, logger),
		Category: NewCategoryRepository(db, logger),
		Search:   NewSearchRepository(db, logger),
	}
}

// Health checks all repositories
func (c *Container) Health() map[string]bool {
	return map[string]bool{
		"paper":    c.Paper != nil,
		"author":   c.Author != nil,
		"category": c.Category != nil,
		"search":   c.Search != nil,
	}
}