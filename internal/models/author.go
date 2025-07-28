package models

import (
	"time"

	"gorm.io/gorm"
)

// Author represents a paper author with detailed information
type Author struct {
	ID          string  `json:"id" gorm:"primaryKey;type:varchar(50)" validate:"required"`
	Name        string  `json:"name" gorm:"type:varchar(255);not null;index" validate:"required,min=1,max=255"`
	Email       *string `json:"email,omitempty" gorm:"type:varchar(255);uniqueIndex" validate:"omitempty,email,max=255"`
	Affiliation *string `json:"affiliation,omitempty" gorm:"type:varchar(500)" validate:"omitempty,max=500"`
	ORCID       *string `json:"orcid,omitempty" gorm:"type:varchar(19);uniqueIndex" validate:"omitempty,orcid"`
	
	// Research profile
	ResearchAreas []string `json:"research_areas" gorm:"serializer:json" validate:"omitempty,dive,min=1,max=100"`
	Website       *string  `json:"website,omitempty" gorm:"type:varchar(2048)" validate:"omitempty,url,max=2048"`
	
	// Metrics
	PaperCount    int `json:"paper_count" gorm:"default:0;index" validate:"min=0"`
	CitationCount int `json:"citation_count" gorm:"default:0;index" validate:"min=0"`
	HIndex        int `json:"h_index" gorm:"default:0;index" validate:"min=0"`
	
	// Relationships
	Papers []Paper `json:"papers,omitempty" gorm:"many2many:paper_authors;"`
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName returns the table name for GORM
func (Author) TableName() string {
	return "authors"
}

// BeforeCreate hook to set default values
func (a *Author) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = generateAuthorID(a.Name, a.Email)
	}
	return nil
}

// HasEmail returns true if author has an email
func (a *Author) HasEmail() bool {
	return a.Email != nil && *a.Email != ""
}

// HasORCID returns true if author has an ORCID
func (a *Author) HasORCID() bool {
	return a.ORCID != nil && *a.ORCID != ""
}

// HasAffiliation returns true if author has an affiliation
func (a *Author) HasAffiliation() bool {
	return a.Affiliation != nil && *a.Affiliation != ""
}

// HasWebsite returns true if author has a website
func (a *Author) HasWebsite() bool {
	return a.Website != nil && *a.Website != ""
}

// AddResearchArea adds a research area if it doesn't exist
func (a *Author) AddResearchArea(area string) {
	for _, ra := range a.ResearchAreas {
		if ra == area {
			return
		}
	}
	a.ResearchAreas = append(a.ResearchAreas, area)
}

// RemoveResearchArea removes a research area
func (a *Author) RemoveResearchArea(area string) {
	for i, ra := range a.ResearchAreas {
		if ra == area {
			a.ResearchAreas = append(a.ResearchAreas[:i], a.ResearchAreas[i+1:]...)
			return
		}
	}
}

// UpdateMetrics updates author metrics based on papers
func (a *Author) UpdateMetrics(papers []Paper) {
	a.PaperCount = len(papers)
	
	// Calculate total citations
	totalCitations := 0
	citationsPerPaper := make([]int, len(papers))
	
	for i, paper := range papers {
		citationsPerPaper[i] = paper.CitationCount
		totalCitations += paper.CitationCount
	}
	
	a.CitationCount = totalCitations
	
	// Calculate H-index
	a.HIndex = calculateHIndex(citationsPerPaper)
}

// calculateHIndex calculates the H-index for an author
func calculateHIndex(citations []int) int {
	if len(citations) == 0 {
		return 0
	}
	
	// Sort citations in descending order
	for i := 0; i < len(citations)-1; i++ {
		for j := i + 1; j < len(citations); j++ {
			if citations[i] < citations[j] {
				citations[i], citations[j] = citations[j], citations[i]
			}
		}
	}
	
	hIndex := 0
	for i, citeCount := range citations {
		if citeCount >= i+1 {
			hIndex = i + 1
		} else {
			break
		}
	}
	
	return hIndex
}

// GetDisplayName returns the name to display for the author
func (a *Author) GetDisplayName() string {
	if a.Name != "" {
		return a.Name
	}
	if a.HasEmail() {
		return *a.Email
	}
	return a.ID
}

// IsProductiveAuthor returns true if author has significant research output
func (a *Author) IsProductiveAuthor() bool {
	return a.PaperCount >= 5 && a.HIndex >= 3
}

// generateAuthorID generates a unique author ID
func generateAuthorID(name string, email *string) string {
	base := "author_"
	if name != "" {
		// Use simplified name
		simplified := simplifyString(name)
		base += simplified
	} else if email != nil && *email != "" {
		// Use email prefix
		emailPrefix := extractEmailPrefix(*email)
		base += emailPrefix
	} else {
		base += generateRandomString(8)
	}
	
	// Add timestamp suffix to ensure uniqueness
	return base + "_" + generateTimestamp()
}

// simplifyString converts a string to a simple identifier
func simplifyString(s string) string {
	result := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else if r == ' ' {
			result += "_"
		}
	}
	if len(result) > 20 {
		result = result[:20]
	}
	return result
}

// extractEmailPrefix extracts the prefix from an email address
func extractEmailPrefix(email string) string {
	for i, r := range email {
		if r == '@' {
			return email[:i]
		}
	}
	return email
}

// generateTimestamp generates a timestamp-based suffix
func generateTimestamp() string {
	return time.Now().Format("20060102150405")
}

// AuthorFilter represents filters for author queries
type AuthorFilter struct {
	IDs           []string `json:"ids,omitempty"`
	Name          string   `json:"name,omitempty"`
	Email         string   `json:"email,omitempty"`
	Affiliation   string   `json:"affiliation,omitempty"`
	ORCID         string   `json:"orcid,omitempty"`
	ResearchAreas []string `json:"research_areas,omitempty"`
	MinPapers     *int     `json:"min_papers,omitempty"`
	MaxPapers     *int     `json:"max_papers,omitempty"`
	MinCitations  *int     `json:"min_citations,omitempty"`
	MaxCitations  *int     `json:"max_citations,omitempty"`
	MinHIndex     *int     `json:"min_h_index,omitempty"`
	MaxHIndex     *int     `json:"max_h_index,omitempty"`
	CreatedFrom   *time.Time `json:"created_from,omitempty"`
	CreatedTo     *time.Time `json:"created_to,omitempty"`
}

// AuthorSort represents sorting options for authors
type AuthorSort struct {
	Field string `json:"field" validate:"oneof=name paper_count citation_count h_index created_at updated_at"`
	Order string `json:"order" validate:"oneof=asc desc"`
}

// DefaultAuthorSort returns the default sort order
func DefaultAuthorSort() AuthorSort {
	return AuthorSort{
		Field: "name",
		Order: "asc",
	}
}