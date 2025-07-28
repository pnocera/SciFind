package fixtures

import (
	"fmt"
	"time"
	"scifind-backend/internal/models"
)

// PaperFixtures provides test paper data
type PaperFixtures struct{}

// NewPaperFixtures creates a new paper fixtures instance
func NewPaperFixtures() *PaperFixtures {
	return &PaperFixtures{}
}

// BasicPaper returns a basic test paper
func (pf *PaperFixtures) BasicPaper() *models.Paper {
	publishedAt := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	
	return &models.Paper{
		ID:              "arxiv_2301.00001",
		Title:           "Advances in Machine Learning: A Comprehensive Survey",
		Abstract:        stringPtr("This paper provides a comprehensive survey of recent advances in machine learning techniques, including deep learning, reinforcement learning, and transfer learning."),
		DOI:             stringPtr("10.1000/test.001"),
		ArxivID:         stringPtr("2301.00001"),
		Journal:         stringPtr("Journal of Machine Learning Research"),
		Volume:          stringPtr("24"),
		Issue:           stringPtr("1"),
		Pages:           stringPtr("1-50"),
		PublishedAt:     &publishedAt,
		URL:             stringPtr("https://arxiv.org/abs/2301.00001"),
		PDFURL:          stringPtr("https://arxiv.org/pdf/2301.00001.pdf"),
		Keywords:        []string{"machine learning", "deep learning", "artificial intelligence", "survey"},
		Language:        "en",
		CitationCount:   125,
		References:      []string{"arxiv_2201.00001", "arxiv_2201.00002"},
		Citations:       []string{"arxiv_2302.00001", "arxiv_2302.00002"},
		SourceProvider:  "arxiv",
		SourceID:        "2301.00001",
		SourceURL:       stringPtr("https://arxiv.org/abs/2301.00001"),
		QualityScore:    0.92,
		ProcessingState: "completed",
	}
}

// PaperWithoutDOI returns a paper without DOI
func (pf *PaperFixtures) PaperWithoutDOI() *models.Paper {
	paper := pf.BasicPaper()
	paper.ID = "arxiv_2301.00002"
	paper.DOI = nil
	paper.ArxivID = stringPtr("2301.00002")
	paper.SourceID = "2301.00002"
	paper.Title = "Novel Approaches in Natural Language Processing"
	paper.QualityScore = 0.75
	return paper
}

// UnpublishedPaper returns an unpublished paper
func (pf *PaperFixtures) UnpublishedPaper() *models.Paper {
	paper := pf.BasicPaper()
	paper.ID = "arxiv_2301.00003"
	paper.ArxivID = stringPtr("2301.00003")
	paper.SourceID = "2301.00003"
	paper.PublishedAt = nil
	paper.Journal = nil
	paper.Volume = nil
	paper.Issue = nil
	paper.Pages = nil
	paper.Title = "Preprint: Emerging Trends in Computer Vision"
	paper.ProcessingState = "pending"
	paper.QualityScore = 0.60
	return paper
}

// PaperWithMultipleAuthors returns a paper with multiple authors
func (pf *PaperFixtures) PaperWithMultipleAuthors() *models.Paper {
	paper := pf.BasicPaper()
	paper.ID = "arxiv_2301.00004"
	paper.ArxivID = stringPtr("2301.00004")
	paper.SourceID = "2301.00004"
	paper.Title = "Collaborative Research in Quantum Computing"
	return paper
}

// PaperWithMinimalData returns a paper with minimal required data
func (pf *PaperFixtures) PaperWithMinimalData() *models.Paper {
	return &models.Paper{
		ID:              "test_minimal",
		Title:           "Minimal Test Paper",
		SourceProvider:  "test",
		SourceID:        "minimal",
		Language:        "en",
		ProcessingState: "pending",
	}
}

// HighQualityPaper returns a high-quality paper
func (pf *PaperFixtures) HighQualityPaper() *models.Paper {
	paper := pf.BasicPaper()
	paper.ID = "nature_001"
	paper.Title = "Breakthrough in Quantum Machine Learning"
	paper.Journal = stringPtr("Nature")
	paper.CitationCount = 500
	paper.QualityScore = 0.98
	paper.SourceProvider = "manual"
	paper.SourceID = "nature_001"
	return paper
}

// LowQualityPaper returns a low-quality paper
func (pf *PaperFixtures) LowQualityPaper() *models.Paper {
	paper := pf.BasicPaper()
	paper.ID = "preprint_001"
	paper.Title = "Preliminary Results"
	paper.Abstract = stringPtr("Short abstract.")
	paper.Journal = nil
	paper.CitationCount = 0
	paper.QualityScore = 0.25
	paper.ProcessingState = "pending"
	return paper
}

// FailedProcessingPaper returns a paper with failed processing
func (pf *PaperFixtures) FailedProcessingPaper() *models.Paper {
	paper := pf.BasicPaper()
	paper.ID = "failed_001"
	paper.Title = "Failed Processing Paper"
	paper.ProcessingState = "failed"
	paper.QualityScore = 0.0
	return paper
}

// PaperWithFullText returns a paper with full text
func (pf *PaperFixtures) PaperWithFullText() *models.Paper {
	paper := pf.BasicPaper()
	paper.ID = "fulltext_001"
	paper.Title = "Paper with Full Text Available"
	fullText := "This is the full text of the paper. It contains detailed information about the research methodology, results, and conclusions."
	paper.FullText = &fullText
	return paper
}

// PaperList returns a list of test papers
func (pf *PaperFixtures) PaperList() []*models.Paper {
	return []*models.Paper{
		pf.BasicPaper(),
		pf.PaperWithoutDOI(),
		pf.UnpublishedPaper(),
		pf.PaperWithMultipleAuthors(),
		pf.HighQualityPaper(),
		pf.LowQualityPaper(),
	}
}

// RecentPapers returns a list of recently published papers
func (pf *PaperFixtures) RecentPapers() []*models.Paper {
	papers := make([]*models.Paper, 5)
	base := pf.BasicPaper()
	
	for i := 0; i < 5; i++ {
		paper := *base // Copy struct
		paper.ID = fmt.Sprintf("recent_%d", i+1)
		paper.SourceID = fmt.Sprintf("recent_%d", i+1)
		paper.Title = fmt.Sprintf("Recent Paper %d", i+1)
		
		// Set different publication dates
		publishedAt := time.Now().AddDate(0, -i, 0)
		paper.PublishedAt = &publishedAt
		
		papers[i] = &paper
	}
	
	return papers
}

// PapersFromDifferentProviders returns papers from different providers
func (pf *PaperFixtures) PapersFromDifferentProviders() []*models.Paper {
	papers := []*models.Paper{}
	
	// ArXiv paper
	arxivPaper := pf.BasicPaper()
	papers = append(papers, arxivPaper)
	
	// Semantic Scholar paper
	ssPaper := pf.BasicPaper()
	ssPaper.ID = "ss_001"
	ssPaper.SourceProvider = "semantic_scholar"
	ssPaper.SourceID = "ss_001"
	ssPaper.Title = "Semantic Scholar Paper"
	papers = append(papers, ssPaper)
	
	// Exa paper
	exaPaper := pf.BasicPaper()
	exaPaper.ID = "exa_001"
	exaPaper.SourceProvider = "exa"
	exaPaper.SourceID = "exa_001"
	exaPaper.Title = "Exa Search Result"
	papers = append(papers, exaPaper)
	
	// Manual entry
	manualPaper := pf.BasicPaper()
	manualPaper.ID = "manual_001"
	manualPaper.SourceProvider = "manual"
	manualPaper.SourceID = "manual_001"
	manualPaper.Title = "Manually Added Paper"
	papers = append(papers, manualPaper)
	
	return papers
}

// PaperFilter returns a test paper filter
func (pf *PaperFixtures) PaperFilter() *models.PaperFilter {
	minCitations := 10
	maxCitations := 1000
	minQuality := 0.5
	
	return &models.PaperFilter{
		Authors:       []string{"John Doe", "Jane Smith"},
		Categories:    []string{"cs.AI", "cs.ML"},
		Keywords:      []string{"machine learning", "deep learning"},
		Language:      "en",
		MinCitations:  &minCitations,
		MaxCitations:  &maxCitations,
		MinQuality:    &minQuality,
		States:        []string{"completed"},
		HasFullText:   boolPtr(false),
		HasPDF:        boolPtr(true),
	}
}

// PaperSort returns a test paper sort
func (pf *PaperFixtures) PaperSort() *models.PaperSort {
	return &models.PaperSort{
		Field: "citation_count",
		Order: "desc",
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

