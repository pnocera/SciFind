package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"scifind-backend/internal/services"
)

// PaperHandler handles paper-related HTTP requests
type PaperHandler struct {
	paperService services.PaperServiceInterface
	logger       *slog.Logger
}

// NewPaperHandler creates a new paper handler
func NewPaperHandler(paperService services.PaperServiceInterface, logger *slog.Logger) *PaperHandler {
	return &PaperHandler{
		paperService: paperService,
		logger:       logger,
	}
}

// ListPapers handles GET /v1/papers
// @Summary List papers
// @Description Get a paginated list of papers
// @Tags papers
// @Accept json
// @Produce json
// @Param limit query int false "Number of results to return (default: 20, max: 100)"
// @Param offset query int false "Number of results to skip (default: 0)"
// @Success 200 {string} string "List of papers with pagination info"
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /v1/papers [get]
func (h *PaperHandler) ListPapers(c *gin.Context) {
	// Parse query parameters
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

	// Get papers
	papers, total, err := h.paperService.List(c.Request.Context(), nil, limit, offset)
	if err != nil {
		h.logger.Error("failed to list papers", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve papers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"papers": papers,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetPaper handles GET /v1/papers/:id
// @Summary Get a paper by ID
// @Description Retrieve a specific paper by its ID
// @Tags papers
// @Accept json
// @Produce json
// @Param id path string true "Paper ID"
// @Success 200 {string} string "Paper details"
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /v1/papers/{id} [get]
func (h *PaperHandler) GetPaper(c *gin.Context) {
	paperID := c.Param("id")
	if paperID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "paper ID is required",
		})
		return
	}

	paper, err := h.paperService.GetByID(c.Request.Context(), paperID)
	if err != nil {
		h.logger.Error("failed to get paper", 
			slog.String("paper_id", paperID),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "paper not found",
		})
		return
	}

	c.JSON(http.StatusOK, paper)
}

// CreatePaper handles POST /v1/papers
// @Summary Create a new paper
// @Description Create a new paper (currently not implemented)
// @Tags papers
// @Accept json
// @Produce json
// @Param paper body string true "Paper data"
// @Success 201 {string} string "Created paper"
// @Failure 400 {object} object{error=string}
// @Failure 501 {object} object{error=string,message=string}
// @Router /v1/papers [post]
func (h *PaperHandler) CreatePaper(c *gin.Context) {
	// TODO: Implement paper creation
	// This would typically be used for manual paper entry
	// or data import from external sources
	
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "paper creation not yet implemented",
		"message": "papers are currently created through search providers",
	})
}

// UpdatePaper handles PUT /v1/papers/:id
// @Summary Update a paper
// @Description Update an existing paper (currently not implemented)
// @Tags papers
// @Accept json
// @Produce json
// @Param id path string true "Paper ID"
// @Param paper body string true "Updated paper data"
// @Success 200 {string} string "Updated paper"
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 501 {object} object{error=string}
// @Router /v1/papers/{id} [put]
func (h *PaperHandler) UpdatePaper(c *gin.Context) {
	// TODO: Implement paper update
	// This would allow updating paper metadata, quality scores, etc.
	
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "paper update not yet implemented",
	})
}

// DeletePaper handles DELETE /v1/papers/:id
// @Summary Delete a paper
// @Description Delete a paper (currently not implemented)
// @Tags papers
// @Accept json
// @Produce json
// @Param id path string true "Paper ID"
// @Success 204 "No Content"
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 501 {object} object{error=string}
// @Router /v1/papers/{id} [delete]
func (h *PaperHandler) DeletePaper(c *gin.Context) {
	// TODO: Implement paper deletion
	// This should be restricted and logged carefully
	
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "paper deletion not yet implemented",
	})
}