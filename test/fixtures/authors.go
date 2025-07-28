package fixtures

import (
	"fmt"
	"time"

	"scifind-backend/internal/models"
)

// AuthorFixtures provides test author data
type AuthorFixtures struct{}

// NewAuthorFixtures creates a new author fixtures instance
func NewAuthorFixtures() *AuthorFixtures {
	return &AuthorFixtures{}
}

// BasicAuthor returns a basic test author
func (af *AuthorFixtures) BasicAuthor() *models.Author {
	return &models.Author{
		ID:           "auth_johndoe",
		Name:         "John Doe",
		Email:        stringPtr("john.doe@university.edu"),
		Affiliation:  stringPtr("University of Technology"),
		ORCID:        stringPtr("0000-0000-0000-0001"),
		Website:      stringPtr("https://johndoe.university.edu"),
		ResearchAreas: []string{"Machine Learning", "Artificial Intelligence", "Deep Learning"},
		PaperCount:   25,
		CitationCount: 1500,
		HIndex:       18,
	}
}

// AuthorWithoutEmail returns an author without email
func (af *AuthorFixtures) AuthorWithoutEmail() *models.Author {
	author := af.BasicAuthor()
	author.ID = "auth_janesmith"
	author.Name = "Jane Smith"
	author.Email = nil
	author.Affiliation = stringPtr("Tech Institute")
	author.ORCID = stringPtr("0000-0000-0000-0002")
	return author
}

// AuthorWithMinimalData returns an author with minimal data
func (af *AuthorFixtures) AuthorWithMinimalData() *models.Author {
	return &models.Author{
		ID:   "auth_minimal",
		Name: "Minimal Author",
	}
}

// ProductiveAuthor returns a highly productive author
func (af *AuthorFixtures) ProductiveAuthor() *models.Author {
	return &models.Author{
		ID:            "auth_productive",
		Name:          "Dr. Alice Researcher",
		Email:         stringPtr("alice@research.org"),
		Affiliation:   stringPtr("Research Institute"),
		ORCID:         stringPtr("0000-0000-0000-0003"),
		Website:       stringPtr("https://alice.research.org"),
		ResearchAreas: []string{"AI", "Machine Learning", "Computer Vision", "NLP"},
		PaperCount:    150,
		CitationCount: 10000,
		HIndex:        45,
	}
}

// NewAuthor returns a new author with few publications
func (af *AuthorFixtures) NewAuthor() *models.Author {
	return &models.Author{
		ID:           "auth_newbie",
		Name:         "Bob Newcomer",
		Email:        stringPtr("bob@newuni.edu"),
		Affiliation:  stringPtr("New University"),
		ORCID:        stringPtr("0000-0000-0000-0004"),
		ResearchAreas: []string{"Deep Learning", "Neural Networks"},
		PaperCount:   3,
		CitationCount: 15,
		HIndex:       2,
	}
}

// IndustrialAuthor returns an author from industry
func (af *AuthorFixtures) IndustrialAuthor() *models.Author {
	return &models.Author{
		ID:           "auth_industry",
		Name:         "Carol Tech",
		Email:        stringPtr("carol@techcorp.com"),
		Affiliation:  stringPtr("TechCorp AI Labs"),
		Website:      stringPtr("https://techcorp.com/researchers/carol"),
		ResearchAreas: []string{"Applied ML", "Computer Vision", "Robotics"},
		PaperCount:   35,
		CitationCount: 800,
		HIndex:       12,
	}
}

// MultiAffiliationAuthor returns an author with multiple affiliations
func (af *AuthorFixtures) MultiAffiliationAuthor() *models.Author {
	return &models.Author{
		ID:           "auth_multi",
		Name:         "Dr. David Multi",
		Email:        stringPtr("david@multi.edu"),
		Affiliation:  stringPtr("University A; University B; Research Lab C"),
		ORCID:        stringPtr("0000-0000-0000-0005"),
		ResearchAreas: []string{"Interdisciplinary AI", "Collaborative Research"},
		PaperCount:   60,
		CitationCount: 2500,
		HIndex:       25,
	}
}

// AuthorList returns a list of test authors
func (af *AuthorFixtures) AuthorList() []*models.Author {
	return []*models.Author{
		af.BasicAuthor(),
		af.AuthorWithoutEmail(),
		af.ProductiveAuthor(),
		af.NewAuthor(),
		af.IndustrialAuthor(),
		af.MultiAffiliationAuthor(),
	}
}

// AuthorsForPaper returns authors suitable for associating with a paper
func (af *AuthorFixtures) AuthorsForPaper() []*models.Author {
	return []*models.Author{
		af.BasicAuthor(),
		af.AuthorWithoutEmail(),
		af.NewAuthor(),
	}
}

// CollaboratingAuthors returns a set of authors who might collaborate
func (af *AuthorFixtures) CollaboratingAuthors() []*models.Author {
	authors := []*models.Author{}
	
	// Create a set of authors from the same field
	for i := 1; i <= 5; i++ {
		author := &models.Author{
			ID:           fmt.Sprintf("auth_collab_%d", i),
			Name:         fmt.Sprintf("Collaborator %d", i),
			Email:        stringPtr(fmt.Sprintf("collab%d@research.edu", i)),
			Affiliation:  stringPtr("Collaborative Research University"),
			ResearchAreas: []string{"Machine Learning", "Data Science"},
			PaperCount:   10 + i*5,
			CitationCount: 100 + i*50,
			HIndex:       5 + i,
		}
		authors = append(authors, author)
	}
	
	return authors
}

// AuthorsByProductivity returns authors sorted by productivity metrics
func (af *AuthorFixtures) AuthorsByProductivity() []*models.Author {
	return []*models.Author{
		af.ProductiveAuthor(),    // Highest productivity
		af.MultiAffiliationAuthor(),
		af.IndustrialAuthor(),
		af.BasicAuthor(),
		af.NewAuthor(),           // Lowest productivity
	}
}

// AuthorWithRecentActivity returns an author with recent publications
func (af *AuthorFixtures) AuthorWithRecentActivity() *models.Author {
	author := af.BasicAuthor()
	author.ID = "auth_recent"
	author.Name = "Dr. Recent Active"
	return author
}

// InactiveAuthor returns an author who hasn't been active recently
func (af *AuthorFixtures) InactiveAuthor() *models.Author {
	author := af.BasicAuthor()
	author.ID = "auth_inactive"
	author.Name = "Dr. Old Timer"
	return author
}

// AuthorFilter returns a test author filter
func (af *AuthorFixtures) AuthorFilter() *models.AuthorFilter {
	minPapers := 10
	maxPapers := 100
	minCitations := 100
	
	return &models.AuthorFilter{
		Name:         "John",
		Email:        "university.edu",
		Affiliation:  "University",
		ResearchAreas: []string{"Machine Learning", "AI"},
		MinPapers:    &minPapers,
		MaxPapers:    &maxPapers,
		MinCitations: &minCitations,
		MinHIndex:    intPtr(5),
	}
}

// AuthorSort returns a test author sort
func (af *AuthorFixtures) AuthorSort() *models.AuthorSort {
	return &models.AuthorSort{
		Field: "citation_count",
		Order: "desc",
	}
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(i int) *int {
	return &i
}