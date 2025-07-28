package models

import (
	"time"

	"gorm.io/gorm"
)

// Paper represents a scientific paper with comprehensive metadata
type Paper struct {
	// Primary identifiers
	ID      string  `json:"id" gorm:"primaryKey;type:varchar(50)" validate:"required"`
	DOI     *string `json:"doi,omitempty" gorm:"uniqueIndex;type:varchar(255)" validate:"omitempty,doi"`
	ArxivID *string `json:"arxiv_id,omitempty" gorm:"uniqueIndex;type:varchar(50)" validate:"omitempty,arxiv_id"`
	
	// Core metadata
	Title    string   `json:"title" gorm:"type:text;not null" validate:"required,min=1,max=1000"`
	Abstract *string  `json:"abstract,omitempty" gorm:"type:text" validate:"omitempty,max=10000"`
	Authors  []Author `json:"authors" gorm:"many2many:paper_authors;" validate:"required,min=1,dive"`
	
	// Publication details
	Journal     *string    `json:"journal,omitempty" gorm:"type:varchar(500)" validate:"omitempty,max=500"`
	Volume      *string    `json:"volume,omitempty" gorm:"type:varchar(50)" validate:"omitempty,max=50"`
	Issue       *string    `json:"issue,omitempty" gorm:"type:varchar(50)" validate:"omitempty,max=50"`
	Pages       *string    `json:"pages,omitempty" gorm:"type:varchar(100)" validate:"omitempty,max=100"`
	PublishedAt *time.Time `json:"published_at,omitempty" gorm:"index"`
	
	// URLs and access
	URL    *string `json:"url,omitempty" gorm:"type:varchar(2048)" validate:"omitempty,url,max=2048"`
	PDFURL *string `json:"pdf_url,omitempty" gorm:"type:varchar(2048)" validate:"omitempty,url,max=2048"`
	
	// Classification and metrics
	Categories []Category `json:"categories" gorm:"many2many:paper_categories;"`
	Keywords   []string   `json:"keywords" gorm:"serializer:json" validate:"omitempty,dive,min=1,max=100"`
	Language   string     `json:"language" gorm:"type:varchar(10);default:'en'" validate:"required,len=2"`
	
	// Citation metrics
	CitationCount int      `json:"citation_count" gorm:"default:0;index" validate:"min=0"`
	References    []string `json:"references" gorm:"serializer:json" validate:"omitempty,dive,min=1"`
	Citations     []string `json:"citations" gorm:"serializer:json" validate:"omitempty,dive,min=1"`
	
	// Content analysis
	FullText      *string `json:"full_text,omitempty" gorm:"type:text"`
	ExtractedData *string `json:"extracted_data,omitempty" gorm:"type:jsonb"`
	
	// Indexing and search
	SearchVector *string   `json:"-" gorm:"type:tsvector;index:,type:gin"`
	Embedding    []float32 `json:"-" gorm:"serializer:json"`
	
	// Source tracking
	SourceProvider string  `json:"source_provider" gorm:"type:varchar(100);not null;index" validate:"required,oneof=arxiv semantic_scholar exa tavily manual"`
	SourceID       string  `json:"source_id" gorm:"type:varchar(255);not null;index" validate:"required"`
	SourceURL      *string `json:"source_url,omitempty" gorm:"type:varchar(2048)" validate:"omitempty,url"`
	
	// Quality and processing
	QualityScore    float64 `json:"quality_score" gorm:"default:0;index" validate:"min=0,max=1"`
	ProcessingState string  `json:"processing_state" gorm:"type:varchar(50);default:'pending';index" validate:"oneof=pending processing completed failed"`
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName returns the table name for GORM
func (Paper) TableName() string {
	return "papers"
}

// BeforeCreate hook to set default values
func (p *Paper) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = generatePaperID(p.SourceProvider, p.SourceID)
	}
	return nil
}

// IsPublished returns true if the paper has a publication date
func (p *Paper) IsPublished() bool {
	return p.PublishedAt != nil
}

// GetYear returns the publication year
func (p *Paper) GetYear() int {
	if p.PublishedAt != nil {
		return p.PublishedAt.Year()
	}
	return 0
}

// HasFullText returns true if full text is available
func (p *Paper) HasFullText() bool {
	return p.FullText != nil && *p.FullText != ""
}

// HasPDF returns true if PDF URL is available
func (p *Paper) HasPDF() bool {
	return p.PDFURL != nil && *p.PDFURL != ""
}

// GetPrimaryAuthor returns the first author
func (p *Paper) GetPrimaryAuthor() *Author {
	if len(p.Authors) > 0 {
		return &p.Authors[0]
	}
	return nil
}

// GetAuthorNames returns a slice of author names
func (p *Paper) GetAuthorNames() []string {
	names := make([]string, len(p.Authors))
	for i, author := range p.Authors {
		names[i] = author.Name
	}
	return names
}

// GetCategoryNames returns a slice of category names
func (p *Paper) GetCategoryNames() []string {
	names := make([]string, len(p.Categories))
	for i, category := range p.Categories {
		names[i] = category.Name
	}
	return names
}

// IsCompleted returns true if processing is completed
func (p *Paper) IsCompleted() bool {
	return p.ProcessingState == "completed"
}

// IsPending returns true if processing is pending
func (p *Paper) IsPending() bool {
	return p.ProcessingState == "pending"
}

// IsProcessing returns true if currently processing
func (p *Paper) IsProcessing() bool {
	return p.ProcessingState == "processing"
}

// IsFailed returns true if processing failed
func (p *Paper) IsFailed() bool {
	return p.ProcessingState == "failed"
}

// SetProcessingState sets the processing state
func (p *Paper) SetProcessingState(state string) {
	p.ProcessingState = state
}

// AddKeyword adds a keyword if it doesn't exist
func (p *Paper) AddKeyword(keyword string) {
	for _, k := range p.Keywords {
		if k == keyword {
			return
		}
	}
	p.Keywords = append(p.Keywords, keyword)
}

// RemoveKeyword removes a keyword
func (p *Paper) RemoveKeyword(keyword string) {
	for i, k := range p.Keywords {
		if k == keyword {
			p.Keywords = append(p.Keywords[:i], p.Keywords[i+1:]...)
			return
		}
	}
}

// AddReference adds a reference if it doesn't exist
func (p *Paper) AddReference(paperID string) {
	for _, ref := range p.References {
		if ref == paperID {
			return
		}
	}
	p.References = append(p.References, paperID)
}

// AddCitation adds a citation if it doesn't exist
func (p *Paper) AddCitation(paperID string) {
	for _, cit := range p.Citations {
		if cit == paperID {
			return
		}
	}
	p.Citations = append(p.Citations, paperID)
	p.CitationCount = len(p.Citations)
}

// UpdateQualityScore updates the quality score based on various factors
func (p *Paper) UpdateQualityScore() {
	score := 0.0
	
	// Base score for having title and abstract
	if p.Title != "" {
		score += 0.1
	}
	if p.Abstract != nil && *p.Abstract != "" {
		score += 0.2
	}
	
	// Author information
	if len(p.Authors) > 0 {
		score += 0.1
		if len(p.Authors) >= 2 {
			score += 0.1
		}
	}
	
	// Publication information
	if p.Journal != nil && *p.Journal != "" {
		score += 0.1
	}
	if p.PublishedAt != nil {
		score += 0.1
	}
	
	// Citation count (normalized)
	if p.CitationCount > 0 {
		score += 0.2 * (1.0 - 1.0/float64(1+p.CitationCount))
	}
	
	// Full text availability
	if p.HasFullText() {
		score += 0.1
	}
	
	// PDF availability
	if p.HasPDF() {
		score += 0.1
	}
	
	p.QualityScore = score
}

// generatePaperID generates a unique paper ID
func generatePaperID(provider, sourceID string) string {
	return provider + "_" + sourceID
}

// PaperFilter represents filters for paper queries
type PaperFilter struct {
	IDs           []string    `json:"ids,omitempty"`
	DOIs          []string    `json:"dois,omitempty"`
	ArxivIDs      []string    `json:"arxiv_ids,omitempty"`
	Title         string      `json:"title,omitempty"`
	Authors       []string    `json:"authors,omitempty"`
	Journal       string      `json:"journal,omitempty"`
	Categories    []string    `json:"categories,omitempty"`
	Keywords      []string    `json:"keywords,omitempty"`
	Language      string      `json:"language,omitempty"`
	SourceProvider string     `json:"source_provider,omitempty"`
	MinCitations  *int        `json:"min_citations,omitempty"`
	MaxCitations  *int        `json:"max_citations,omitempty"`
	MinQuality    *float64    `json:"min_quality,omitempty"`
	MaxQuality    *float64    `json:"max_quality,omitempty"`
	PublishedFrom *time.Time  `json:"published_from,omitempty"`
	PublishedTo   *time.Time  `json:"published_to,omitempty"`
	CreatedFrom   *time.Time  `json:"created_from,omitempty"`
	CreatedTo     *time.Time  `json:"created_to,omitempty"`
	States        []string    `json:"states,omitempty"`
	HasFullText   *bool       `json:"has_full_text,omitempty"`
	HasPDF        *bool       `json:"has_pdf,omitempty"`
}

// PaperSort represents sorting options for papers
type PaperSort struct {
	Field string `json:"field" validate:"oneof=created_at updated_at published_at citation_count quality_score title"`
	Order string `json:"order" validate:"oneof=asc desc"`
}

// DefaultPaperSort returns the default sort order
func DefaultPaperSort() PaperSort {
	return PaperSort{
		Field: "created_at",
		Order: "desc",
	}
}