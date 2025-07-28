package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scifind-backend/internal/models"
	"scifind-backend/test/fixtures"
)

func TestPaper_IsPublished(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()

	t.Run("published paper", func(t *testing.T) {
		paper := paperFixtures.BasicPaper()
		assert.True(t, paper.IsPublished())
	})

	t.Run("unpublished paper", func(t *testing.T) {
		paper := paperFixtures.UnpublishedPaper()
		assert.False(t, paper.IsPublished())
	})
}

func TestPaper_GetYear(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()

	t.Run("published paper", func(t *testing.T) {
		paper := paperFixtures.BasicPaper()
		expected := paper.PublishedAt.Year()
		assert.Equal(t, expected, paper.GetYear())
	})

	t.Run("unpublished paper", func(t *testing.T) {
		paper := paperFixtures.UnpublishedPaper()
		assert.Equal(t, 0, paper.GetYear())
	})
}

func TestPaper_HasFullText(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()

	t.Run("paper with full text", func(t *testing.T) {
		paper := paperFixtures.PaperWithFullText()
		assert.True(t, paper.HasFullText())
	})

	t.Run("paper without full text", func(t *testing.T) {
		paper := paperFixtures.BasicPaper()
		assert.False(t, paper.HasFullText())
	})

	t.Run("paper with empty full text", func(t *testing.T) {
		paper := paperFixtures.BasicPaper()
		emptyText := ""
		paper.FullText = &emptyText
		assert.False(t, paper.HasFullText())
	})
}

func TestPaper_HasPDF(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()

	t.Run("paper with PDF URL", func(t *testing.T) {
		paper := paperFixtures.BasicPaper()
		assert.True(t, paper.HasPDF())
	})

	t.Run("paper without PDF URL", func(t *testing.T) {
		paper := paperFixtures.PaperWithMinimalData()
		assert.False(t, paper.HasPDF())
	})

	t.Run("paper with empty PDF URL", func(t *testing.T) {
		paper := paperFixtures.BasicPaper()
		emptyURL := ""
		paper.PDFURL = &emptyURL
		assert.False(t, paper.HasPDF())
	})
}

func TestPaper_GetPrimaryAuthor(t *testing.T) {
	t.Run("paper with authors", func(t *testing.T) {
		authorFixtures := fixtures.NewAuthorFixtures()
		authors := authorFixtures.AuthorsForPaper()
		
		paper := &models.Paper{
			Authors: []models.Author{*authors[0], *authors[1]},
		}
		
		primaryAuthor := paper.GetPrimaryAuthor()
		require.NotNil(t, primaryAuthor)
		assert.Equal(t, authors[0].ID, primaryAuthor.ID)
	})

	t.Run("paper without authors", func(t *testing.T) {
		paper := &models.Paper{
			Authors: []models.Author{},
		}
		
		primaryAuthor := paper.GetPrimaryAuthor()
		assert.Nil(t, primaryAuthor)
	})
}

func TestPaper_GetAuthorNames(t *testing.T) {
	t.Run("paper with multiple authors", func(t *testing.T) {
		authorFixtures := fixtures.NewAuthorFixtures()
		authors := authorFixtures.AuthorsForPaper()
		
		paper := &models.Paper{
			Authors: []models.Author{*authors[0], *authors[1], *authors[2]},
		}
		
		names := paper.GetAuthorNames()
		expectedNames := []string{authors[0].Name, authors[1].Name, authors[2].Name}
		assert.Equal(t, expectedNames, names)
	})

	t.Run("paper without authors", func(t *testing.T) {
		paper := &models.Paper{
			Authors: []models.Author{},
		}
		
		names := paper.GetAuthorNames()
		assert.Empty(t, names)
	})
}

func TestPaper_ProcessingStateCheckers(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()

	t.Run("completed paper", func(t *testing.T) {
		paper := paperFixtures.BasicPaper()
		assert.True(t, paper.IsCompleted())
		assert.False(t, paper.IsPending())
		assert.False(t, paper.IsProcessing())
		assert.False(t, paper.IsFailed())
	})

	t.Run("pending paper", func(t *testing.T) {
		paper := paperFixtures.UnpublishedPaper()
		assert.False(t, paper.IsCompleted())
		assert.True(t, paper.IsPending())
		assert.False(t, paper.IsProcessing())
		assert.False(t, paper.IsFailed())
	})

	t.Run("processing paper", func(t *testing.T) {
		paper := paperFixtures.BasicPaper()
		paper.ProcessingState = "processing"
		assert.False(t, paper.IsCompleted())
		assert.False(t, paper.IsPending())
		assert.True(t, paper.IsProcessing())
		assert.False(t, paper.IsFailed())
	})

	t.Run("failed paper", func(t *testing.T) {
		paper := paperFixtures.FailedProcessingPaper()
		assert.False(t, paper.IsCompleted())
		assert.False(t, paper.IsPending())
		assert.False(t, paper.IsProcessing())
		assert.True(t, paper.IsFailed())
	})
}

func TestPaper_SetProcessingState(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()
	paper := paperFixtures.BasicPaper()

	paper.SetProcessingState("processing")
	assert.Equal(t, "processing", paper.ProcessingState)
	assert.True(t, paper.IsProcessing())

	paper.SetProcessingState("failed")
	assert.Equal(t, "failed", paper.ProcessingState)
	assert.True(t, paper.IsFailed())
}

func TestPaper_AddKeyword(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()
	paper := paperFixtures.BasicPaper()
	
	originalLength := len(paper.Keywords)
	
	t.Run("add new keyword", func(t *testing.T) {
		paper.AddKeyword("new keyword")
		assert.Len(t, paper.Keywords, originalLength+1)
		assert.Contains(t, paper.Keywords, "new keyword")
	})

	t.Run("add existing keyword", func(t *testing.T) {
		existingKeyword := paper.Keywords[0]
		paper.AddKeyword(existingKeyword)
		assert.Len(t, paper.Keywords, originalLength+1) // Should not increase
	})
}

func TestPaper_RemoveKeyword(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()
	paper := paperFixtures.BasicPaper()
	
	originalLength := len(paper.Keywords)
	existingKeyword := paper.Keywords[0]
	
	t.Run("remove existing keyword", func(t *testing.T) {
		paper.RemoveKeyword(existingKeyword)
		assert.Len(t, paper.Keywords, originalLength-1)
		assert.NotContains(t, paper.Keywords, existingKeyword)
	})

	t.Run("remove non-existing keyword", func(t *testing.T) {
		currentLength := len(paper.Keywords)
		paper.RemoveKeyword("non-existing keyword")
		assert.Len(t, paper.Keywords, currentLength) // Should not change
	})
}

func TestPaper_AddReference(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()
	paper := paperFixtures.BasicPaper()
	
	originalLength := len(paper.References)
	
	t.Run("add new reference", func(t *testing.T) {
		paper.AddReference("new_paper_id")
		assert.Len(t, paper.References, originalLength+1)
		assert.Contains(t, paper.References, "new_paper_id")
	})

	t.Run("add existing reference", func(t *testing.T) {
		existingRef := paper.References[0]
		paper.AddReference(existingRef)
		assert.Len(t, paper.References, originalLength+1) // Should not increase
	})
}

func TestPaper_AddCitation(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()
	paper := paperFixtures.BasicPaper()
	
	originalLength := len(paper.Citations)
	_ = paper.CitationCount // originalCount not used in this test
	
	t.Run("add new citation", func(t *testing.T) {
		paper.AddCitation("citing_paper_id")
		assert.Len(t, paper.Citations, originalLength+1)
		assert.Contains(t, paper.Citations, "citing_paper_id")
		assert.Equal(t, len(paper.Citations), paper.CitationCount)
	})

	t.Run("add existing citation", func(t *testing.T) {
		currentLength := len(paper.Citations)
		existingCitation := paper.Citations[0]
		paper.AddCitation(existingCitation)
		assert.Len(t, paper.Citations, currentLength) // Should not increase
		assert.Equal(t, len(paper.Citations), paper.CitationCount)
	})
}

func TestPaper_UpdateQualityScore(t *testing.T) {
	t.Run("high quality paper", func(t *testing.T) {
		paperFixtures := fixtures.NewPaperFixtures()
		paper := paperFixtures.HighQualityPaper()
		paper.QualityScore = 0.0 // Reset to test calculation
		
		paper.UpdateQualityScore()
		
		// High quality paper should have high score
		assert.Greater(t, paper.QualityScore, 0.8)
	})

	t.Run("minimal paper", func(t *testing.T) {
		paperFixtures := fixtures.NewPaperFixtures()
		paper := paperFixtures.PaperWithMinimalData()
		
		paper.UpdateQualityScore()
		
		// Minimal paper should have low score
		assert.Less(t, paper.QualityScore, 0.3)
	})

	t.Run("paper with all quality factors", func(t *testing.T) {
		authorFixtures := fixtures.NewAuthorFixtures()
		authors := authorFixtures.AuthorsForPaper()
		
		publishedAt := time.Now()
		fullText := "Full text content"
		abstract := "Detailed abstract"
		journal := "Nature"
		pdfURL := "https://example.com/paper.pdf"
		
		paper := &models.Paper{
			Title:         "Complete Paper",
			Abstract:      &abstract,
			Authors:       []models.Author{*authors[0], *authors[1]},
			Journal:       &journal,
			PublishedAt:   &publishedAt,
			CitationCount: 100,
			FullText:      &fullText,
			PDFURL:        &pdfURL,
		}
		
		paper.UpdateQualityScore()
		
		// Paper with all factors should have very high score
		assert.Greater(t, paper.QualityScore, 0.9)
	})
}

func TestPaperFilter_Validation(t *testing.T) {
	paperFixtures := fixtures.NewPaperFixtures()
	filter := paperFixtures.PaperFilter()

	// Test that filter has expected values
	assert.NotEmpty(t, filter.Authors)
	assert.NotEmpty(t, filter.Categories)
	assert.NotEmpty(t, filter.Keywords)
	assert.Equal(t, "en", filter.Language)
	assert.NotNil(t, filter.MinCitations)
	assert.NotNil(t, filter.MaxCitations)
	assert.NotNil(t, filter.MinQuality)
	assert.NotEmpty(t, filter.States)
}

func TestPaperSort_DefaultSort(t *testing.T) {
	defaultSort := models.DefaultPaperSort()
	
	assert.Equal(t, "created_at", defaultSort.Field)
	assert.Equal(t, "desc", defaultSort.Order)
}

func TestPaper_BeforeCreateHook(t *testing.T) {
	paper := &models.Paper{
		Title:          "Test Paper",
		SourceProvider: "test",
		SourceID:       "123",
	}

	// Simulate GORM BeforeCreate hook
	err := paper.BeforeCreate(nil)
	require.NoError(t, err)
	
	// Should generate ID if empty
	assert.NotEmpty(t, paper.ID)
	assert.Equal(t, "test_123", paper.ID)
}

func TestPaper_TableName(t *testing.T) {
	paper := &models.Paper{}
	assert.Equal(t, "papers", paper.TableName())
}