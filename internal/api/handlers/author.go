package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"scifind-backend/internal/models"
	"scifind-backend/internal/services"
)

// AuthorHandler handles author-related HTTP requests
type AuthorHandler struct {
	authorService *services.AuthorService
	logger        *slog.Logger
}

// NewAuthorHandler creates a new author handler
func NewAuthorHandler(authorService *services.AuthorService, logger *slog.Logger) *AuthorHandler {
	return &AuthorHandler{
		authorService: authorService,
		logger:        logger,
	}
}

// ListAuthors handles GET /v1/authors
func (h *AuthorHandler) ListAuthors(c *gin.Context) {
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	query := c.Query("q")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit parameter",
		})
		return
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset parameter",
		})
		return
	}

	// Search or list authors
	var authors []*models.Author
	var total int
	if query != "" {
		authors, total, err = h.authorService.Search(c.Request.Context(), query, limit, offset)
	} else {
		authors, total, err = h.authorService.List(c.Request.Context(), nil, limit, offset)
	}
	
	if err != nil {
		h.logger.Error("failed to list/search authors", 
			slog.String("query", query),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve authors",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"authors": authors,
		"total":   total,
		"query":   query,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetAuthor handles GET /v1/authors/:id
func (h *AuthorHandler) GetAuthor(c *gin.Context) {
	authorID := c.Param("id")
	if authorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "author ID is required",
		})
		return
	}

	author, err := h.authorService.GetByID(c.Request.Context(), authorID)
	if err != nil {
		h.logger.Error("failed to get author", 
			slog.String("author_id", authorID),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "author not found",
		})
		return
	}

	c.JSON(http.StatusOK, author)
}

// GetAuthorPapers handles GET /v1/authors/:id/papers
func (h *AuthorHandler) GetAuthorPapers(c *gin.Context) {
	authorID := c.Param("id")
	if authorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "author ID is required",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit parameter",
		})
		return
	}
	
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset parameter",
		})
		return
	}

	papers, total, err := h.authorService.GetPapers(c.Request.Context(), authorID, limit, offset)
	if err != nil {
		h.logger.Error("failed to get author papers", 
			slog.String("author_id", authorID),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve author papers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"author_id": authorID,
		"papers":    papers,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}