package models

import (
	"time"

	"gorm.io/gorm"
)

// Category represents a classification category for papers
type Category struct {
	ID          string  `json:"id" gorm:"primaryKey;type:varchar(50)" validate:"required"`
	Name        string  `json:"name" gorm:"type:varchar(255);not null;uniqueIndex" validate:"required,min=1,max=255"`
	Description *string `json:"description,omitempty" gorm:"type:text" validate:"omitempty,max=1000"`
	ParentID    *string `json:"parent_id,omitempty" gorm:"type:varchar(50);index"`
	Level       int     `json:"level" gorm:"default:0;index" validate:"min=0,max=10"`
	
	// Hierarchy relationships
	Parent   *Category  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Children []Category `json:"children,omitempty" gorm:"foreignKey:ParentID"`
	
	// Classification metadata
	Source      string `json:"source" gorm:"type:varchar(100);not null" validate:"required,oneof=arxiv acm ieee manual"`
	SourceCode  string `json:"source_code" gorm:"type:varchar(100);not null" validate:"required"`
	IsActive    bool   `json:"is_active" gorm:"default:true;index"`
	
	// Usage statistics
	PaperCount int `json:"paper_count" gorm:"default:0;index" validate:"min=0"`
	
	// Relationships
	Papers []Paper `json:"papers,omitempty" gorm:"many2many:paper_categories;"`
	
	// Timestamps
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName returns the table name for GORM
func (Category) TableName() string {
	return "categories"
}

// BeforeCreate hook to set default values
func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = generateCategoryID(c.Source, c.SourceCode)
	}
	return nil
}

// IsTopLevel returns true if this is a top-level category
func (c *Category) IsTopLevel() bool {
	return c.ParentID == nil || *c.ParentID == ""
}

// HasChildren returns true if this category has sub-categories
func (c *Category) HasChildren() bool {
	return len(c.Children) > 0
}

// GetFullPath returns the full hierarchical path of the category
func (c *Category) GetFullPath() string {
	if c.IsTopLevel() {
		return c.Name
	}
	if c.Parent != nil {
		return c.Parent.GetFullPath() + " > " + c.Name
	}
	return c.Name
}

// GetAncestors returns all ancestor categories
func (c *Category) GetAncestors() []Category {
	var ancestors []Category
	current := c.Parent
	
	for current != nil {
		ancestors = append([]Category{*current}, ancestors...)
		current = current.Parent
	}
	
	return ancestors
}

// GetDescendants returns all descendant categories (recursive)
func (c *Category) GetDescendants() []Category {
	var descendants []Category
	
	for _, child := range c.Children {
		descendants = append(descendants, child)
		descendants = append(descendants, child.GetDescendants()...)
	}
	
	return descendants
}

// UpdatePaperCount updates the paper count for this category
func (c *Category) UpdatePaperCount(count int) {
	c.PaperCount = count
}

// IsArxivCategory returns true if this is an ArXiv category
func (c *Category) IsArxivCategory() bool {
	return c.Source == "arxiv"
}

// IsACMCategory returns true if this is an ACM category
func (c *Category) IsACMCategory() bool {
	return c.Source == "acm"
}

// IsIEEECategory returns true if this is an IEEE category
func (c *Category) IsIEEECategory() bool {
	return c.Source == "ieee"
}

// IsManualCategory returns true if this is a manually created category
func (c *Category) IsManualCategory() bool {
	return c.Source == "manual"
}

// generateCategoryID generates a unique category ID
func generateCategoryID(source, sourceCode string) string {
	return source + "_" + sourceCode
}

// PredefinedCategories contains common ArXiv categories
var PredefinedCategories = []Category{
	// Computer Science
	{
		ID:          "arxiv_cs",
		Name:        "Computer Science",
		Description: stringPtr("Computer Science"),
		Source:      "arxiv",
		SourceCode:  "cs",
		Level:       0,
		IsActive:    true,
	},
	{
		ID:          "arxiv_cs.AI",
		Name:        "Artificial Intelligence",
		Description: stringPtr("Artificial Intelligence"),
		ParentID:    stringPtr("arxiv_cs"),
		Source:      "arxiv",
		SourceCode:  "cs.AI",
		Level:       1,
		IsActive:    true,
	},
	{
		ID:          "arxiv_cs.CL",
		Name:        "Computation and Language",
		Description: stringPtr("Computation and Language"),
		ParentID:    stringPtr("arxiv_cs"),
		Source:      "arxiv",
		SourceCode:  "cs.CL",
		Level:       1,
		IsActive:    true,
	},
	{
		ID:          "arxiv_cs.CV",
		Name:        "Computer Vision and Pattern Recognition",
		Description: stringPtr("Computer Vision and Pattern Recognition"),
		ParentID:    stringPtr("arxiv_cs"),
		Source:      "arxiv",
		SourceCode:  "cs.CV",
		Level:       1,
		IsActive:    true,
	},
	{
		ID:          "arxiv_cs.LG",
		Name:        "Machine Learning",
		Description: stringPtr("Machine Learning"),
		ParentID:    stringPtr("arxiv_cs"),
		Source:      "arxiv",
		SourceCode:  "cs.LG",
		Level:       1,
		IsActive:    true,
	},
	
	// Physics
	{
		ID:          "arxiv_physics",
		Name:        "Physics",
		Description: stringPtr("Physics"),
		Source:      "arxiv",
		SourceCode:  "physics",
		Level:       0,
		IsActive:    true,
	},
	{
		ID:          "arxiv_quant-ph",
		Name:        "Quantum Physics",
		Description: stringPtr("Quantum Physics"),
		Source:      "arxiv",
		SourceCode:  "quant-ph",
		Level:       0,
		IsActive:    true,
	},
	
	// Mathematics
	{
		ID:          "arxiv_math",
		Name:        "Mathematics",
		Description: stringPtr("Mathematics"),
		Source:      "arxiv",
		SourceCode:  "math",
		Level:       0,
		IsActive:    true,
	},
	{
		ID:          "arxiv_stat",
		Name:        "Statistics",
		Description: stringPtr("Statistics"),
		Source:      "arxiv",
		SourceCode:  "stat",
		Level:       0,
		IsActive:    true,
	},
	{
		ID:          "arxiv_stat.ML",
		Name:        "Machine Learning (Statistics)",
		Description: stringPtr("Machine Learning from Statistics perspective"),
		ParentID:    stringPtr("arxiv_stat"),
		Source:      "arxiv",
		SourceCode:  "stat.ML",
		Level:       1,
		IsActive:    true,
	},
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// CategoryFilter represents filters for category queries
type CategoryFilter struct {
	IDs          []string `json:"ids,omitempty"`
	Names        []string `json:"names,omitempty"`
	Source       string   `json:"source,omitempty"`
	SourceCodes  []string `json:"source_codes,omitempty"`
	ParentID     *string  `json:"parent_id,omitempty"`
	Level        *int     `json:"level,omitempty"`
	IsActive     *bool    `json:"is_active,omitempty"`
	MinPapers    *int     `json:"min_papers,omitempty"`
	MaxPapers    *int     `json:"max_papers,omitempty"`
	CreatedFrom  *time.Time `json:"created_from,omitempty"`
	CreatedTo    *time.Time `json:"created_to,omitempty"`
}

// CategorySort represents sorting options for categories
type CategorySort struct {
	Field string `json:"field" validate:"oneof=name level paper_count created_at updated_at"`
	Order string `json:"order" validate:"oneof=asc desc"`
}

// DefaultCategorySort returns the default sort order
func DefaultCategorySort() CategorySort {
	return CategorySort{
		Field: "name",
		Order: "asc",
	}
}

// CategoryTree represents a hierarchical category structure
type CategoryTree struct {
	Category Category       `json:"category"`
	Children []CategoryTree `json:"children,omitempty"`
}

// BuildCategoryTree builds a hierarchical tree from flat categories
func BuildCategoryTree(categories []Category) []CategoryTree {
	categoryMap := make(map[string]*Category)
	var roots []CategoryTree
	
	// Create a map for quick lookup
	for i := range categories {
		categoryMap[categories[i].ID] = &categories[i]
	}
	
	// Build the tree
	for i := range categories {
		category := &categories[i]
		if category.IsTopLevel() {
			roots = append(roots, CategoryTree{Category: *category})
		}
	}
	
	// Recursively build children
	for i := range roots {
		buildChildren(&roots[i], categoryMap)
	}
	
	return roots
}

// buildChildren recursively builds children for a category tree node
func buildChildren(node *CategoryTree, categoryMap map[string]*Category) {
	for _, category := range categoryMap {
		if category.ParentID != nil && *category.ParentID == node.Category.ID {
			childNode := CategoryTree{Category: *category}
			buildChildren(&childNode, categoryMap)
			node.Children = append(node.Children, childNode)
		}
	}
}