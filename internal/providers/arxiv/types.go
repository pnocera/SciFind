package arxiv

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ArxivFeed represents the root element of ArXiv API response
type ArxivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []ArxivEntry `xml:"entry"`
}

// ArxivEntry represents a single ArXiv paper entry
type ArxivEntry struct {
	ID         string          `xml:"id"`
	Title      string          `xml:"title"`
	Summary    string          `xml:"summary"`
	Published  string          `xml:"published"`
	Updated    string          `xml:"updated"`
	Authors    []ArxivAuthor   `xml:"author"`
	Categories []ArxivCategory `xml:"category"`
	Links      []ArxivLink     `xml:"link"`
	Comment    string          `xml:"comment"`
	Journal    string          `xml:"journal_ref"`
	DOI        string          `xml:"doi"`
}

// ArxivAuthor represents an author in ArXiv response
type ArxivAuthor struct {
	Name        string `xml:"name"`
	Affiliation string `xml:"affiliation"`
}

// ArxivCategory represents a category in ArXiv response
type ArxivCategory struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr"`
	Label  string `xml:"label,attr"`
}

// ArxivLink represents a link in ArXiv response
type ArxivLink struct {
	Href  string `xml:"href,attr"`
	Rel   string `xml:"rel,attr"`
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
}

// ArxivError represents an error response from ArXiv API
type ArxivError struct {
	Code    string `xml:"code"`
	Message string `xml:"message"`
}

// ArxivQueryBuilder helps build complex ArXiv queries
type ArxivQueryBuilder struct {
	terms []string
}

// NewQueryBuilder creates a new ArXiv query builder
func NewQueryBuilder() *ArxivQueryBuilder {
	return &ArxivQueryBuilder{
		terms: make([]string, 0),
	}
}

// Title adds a title search term
func (qb *ArxivQueryBuilder) Title(title string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("ti:\"%s\"", title))
	return qb
}

// Abstract adds an abstract search term
func (qb *ArxivQueryBuilder) Abstract(abstract string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("abs:\"%s\"", abstract))
	return qb
}

// Author adds an author search term
func (qb *ArxivQueryBuilder) Author(author string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("au:\"%s\"", author))
	return qb
}

// Category adds a category search term
func (qb *ArxivQueryBuilder) Category(category string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("cat:%s", category))
	return qb
}

// All adds an all-fields search term
func (qb *ArxivQueryBuilder) All(term string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("all:\"%s\"", term))
	return qb
}

// Comment adds a comment search term
func (qb *ArxivQueryBuilder) Comment(comment string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("co:\"%s\"", comment))
	return qb
}

// Journal adds a journal reference search term
func (qb *ArxivQueryBuilder) Journal(journal string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("jr:\"%s\"", journal))
	return qb
}

// SubjectClass adds a subject class search term
func (qb *ArxivQueryBuilder) SubjectClass(class string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("subj-class:%s", class))
	return qb
}

// ReportNumber adds a report number search term
func (qb *ArxivQueryBuilder) ReportNumber(number string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("rn:\"%s\"", number))
	return qb
}

// ID adds an ID search term
func (qb *ArxivQueryBuilder) ID(id string) *ArxivQueryBuilder {
	qb.terms = append(qb.terms, fmt.Sprintf("id:%s", id))
	return qb
}

// SubmittedDateRange adds a submitted date range filter
func (qb *ArxivQueryBuilder) SubmittedDateRange(from, to string) *ArxivQueryBuilder {
	if from != "" && to != "" {
		qb.terms = append(qb.terms, fmt.Sprintf("submittedDate:[%s TO %s]", from, to))
	} else if from != "" {
		qb.terms = append(qb.terms, fmt.Sprintf("submittedDate:[%s TO *]", from))
	} else if to != "" {
		qb.terms = append(qb.terms, fmt.Sprintf("submittedDate:[* TO %s]", to))
	}
	return qb
}

// LastUpdatedDateRange adds a last updated date range filter
func (qb *ArxivQueryBuilder) LastUpdatedDateRange(from, to string) *ArxivQueryBuilder {
	if from != "" && to != "" {
		qb.terms = append(qb.terms, fmt.Sprintf("lastUpdatedDate:[%s TO %s]", from, to))
	} else if from != "" {
		qb.terms = append(qb.terms, fmt.Sprintf("lastUpdatedDate:[%s TO *]", from))
	} else if to != "" {
		qb.terms = append(qb.terms, fmt.Sprintf("lastUpdatedDate:[* TO %s]", to))
	}
	return qb
}

// AND combines terms with AND operator
func (qb *ArxivQueryBuilder) AND() *ArxivQueryBuilder {
	if len(qb.terms) > 0 {
		last := len(qb.terms) - 1
		qb.terms[last] += " AND"
	}
	return qb
}

// OR combines terms with OR operator
func (qb *ArxivQueryBuilder) OR() *ArxivQueryBuilder {
	if len(qb.terms) > 0 {
		last := len(qb.terms) - 1
		qb.terms[last] += " OR"
	}
	return qb
}

// NOT negates the next term
func (qb *ArxivQueryBuilder) NOT() *ArxivQueryBuilder {
	qb.terms = append(qb.terms, "NOT")
	return qb
}

// Build constructs the final query string
func (qb *ArxivQueryBuilder) Build() string {
	return strings.Join(qb.terms, " ")
}

// ArxivSortBy represents sort options for ArXiv API
type ArxivSortBy string

const (
	SortByRelevance      ArxivSortBy = "relevance"
	SortByLastUpdated    ArxivSortBy = "lastUpdatedDate"
	SortBySubmittedDate  ArxivSortBy = "submittedDate"
)

// ArxivSortOrder represents sort order for ArXiv API
type ArxivSortOrder string

const (
	SortOrderAscending  ArxivSortOrder = "ascending"
	SortOrderDescending ArxivSortOrder = "descending"
)

// ArxivQueryParams represents parameters for ArXiv API query
type ArxivQueryParams struct {
	SearchQuery string         `json:"search_query"`
	IDList      []string       `json:"id_list,omitempty"`
	Start       int            `json:"start"`
	MaxResults  int            `json:"max_results"`
	SortBy      ArxivSortBy    `json:"sort_by"`
	SortOrder   ArxivSortOrder `json:"sort_order"`
}

// ToURLParams converts query parameters to URL parameters
func (aqp *ArxivQueryParams) ToURLParams() url.Values {
	params := url.Values{}
	
	if aqp.SearchQuery != "" {
		params.Set("search_query", aqp.SearchQuery)
	}
	
	if len(aqp.IDList) > 0 {
		params.Set("id_list", strings.Join(aqp.IDList, ","))
	}
	
	params.Set("start", strconv.Itoa(aqp.Start))
	params.Set("max_results", strconv.Itoa(aqp.MaxResults))
	
	if aqp.SortBy != "" {
		params.Set("sortBy", string(aqp.SortBy))
	}
	
	if aqp.SortOrder != "" {
		params.Set("sortOrder", string(aqp.SortOrder))
	}
	
	return params
}

// ArxivFieldMap maps search fields to ArXiv query prefixes
var ArxivFieldMap = map[string]string{
	"title":        "ti",
	"abstract":     "abs",
	"author":       "au",
	"category":     "cat",
	"comment":      "co",
	"journal":      "jr",
	"subject":      "subj-class",
	"report":       "rn",
	"id":           "id",
	"all":          "all",
}

// CategoryHierarchy represents ArXiv category hierarchy
var CategoryHierarchy = map[string][]string{
	"cs": {
		"cs.AI", "cs.AR", "cs.CC", "cs.CE", "cs.CG", "cs.CL", "cs.CR", "cs.CV",
		"cs.CY", "cs.DB", "cs.DC", "cs.DL", "cs.DM", "cs.DS", "cs.ET", "cs.FL",
		"cs.GL", "cs.GR", "cs.GT", "cs.HC", "cs.IR", "cs.IT", "cs.LG", "cs.LO",
		"cs.MA", "cs.MM", "cs.MS", "cs.NA", "cs.NE", "cs.NI", "cs.OH", "cs.OS",
		"cs.PF", "cs.PL", "cs.RO", "cs.SC", "cs.SD", "cs.SE", "cs.SI", "cs.SY",
	},
	"math": {
		"math.AC", "math.AG", "math.AP", "math.AT", "math.CA", "math.CO", "math.CT",
		"math.CV", "math.DG", "math.DS", "math.FA", "math.GM", "math.GN", "math.GR",
		"math.GT", "math.HO", "math.IT", "math.KT", "math.LO", "math.MG", "math.MP",
		"math.NA", "math.NT", "math.OA", "math.OC", "math.PR", "math.QA", "math.RA",
		"math.RT", "math.SG", "math.SP", "math.ST",
	},
	"physics": {
		"physics.acc-ph", "physics.ao-ph", "physics.app-ph", "physics.atm-clus",
		"physics.atom-ph", "physics.bio-ph", "physics.chem-ph", "physics.class-ph",
		"physics.comp-ph", "physics.data-an", "physics.ed-ph", "physics.flu-dyn",
		"physics.gen-ph", "physics.geo-ph", "physics.hist-ph", "physics.ins-det",
		"physics.med-ph", "physics.optics", "physics.plasm-ph", "physics.pop-ph",
		"physics.soc-ph", "physics.space-ph",
	},
	"astro-ph": {
		"astro-ph.CO", "astro-ph.EP", "astro-ph.GA", "astro-ph.HE", "astro-ph.IM", "astro-ph.SR",
	},
	"cond-mat": {
		"cond-mat.dis-nn", "cond-mat.mes-hall", "cond-mat.mtrl-sci", "cond-mat.other",
		"cond-mat.quant-gas", "cond-mat.soft", "cond-mat.stat-mech", "cond-mat.str-el",
		"cond-mat.supr-con",
	},
	"hep": {
		"hep-ex", "hep-lat", "hep-ph", "hep-th",
	},
	"nucl": {
		"nucl-ex", "nucl-th",
	},
	"q-bio": {
		"q-bio.BM", "q-bio.CB", "q-bio.GN", "q-bio.MN", "q-bio.NC", "q-bio.OT",
		"q-bio.PE", "q-bio.QM", "q-bio.SC", "q-bio.TO",
	},
	"q-fin": {
		"q-fin.CP", "q-fin.EC", "q-fin.GN", "q-fin.MF", "q-fin.PM", "q-fin.PR",
		"q-fin.RM", "q-fin.ST", "q-fin.TR",
	},
	"stat": {
		"stat.AP", "stat.CO", "stat.ME", "stat.ML", "stat.OT", "stat.TH",
	},
	"econ": {
		"econ.EM", "econ.GN", "econ.TH",
	},
	"eess": {
		"eess.AS", "eess.IV", "eess.SP", "eess.SY",
	},
}

// IsValidCategory checks if a category is valid in ArXiv
func IsValidCategory(category string) bool {
	for _, categories := range CategoryHierarchy {
		for _, validCat := range categories {
			if validCat == category {
				return true
			}
		}
	}
	
	// Check top-level categories
	topLevel := []string{"cs", "math", "physics", "astro-ph", "cond-mat", "gr-qc", 
		"hep-ex", "hep-lat", "hep-ph", "hep-th", "math-ph", "nlin", "nucl-ex", 
		"nucl-th", "q-bio", "q-fin", "quant-ph", "stat", "econ", "eess"}
	
	for _, cat := range topLevel {
		if cat == category {
			return true
		}
	}
	
	return false
}

// GetSubcategories returns subcategories for a given top-level category
func GetSubcategories(topLevel string) []string {
	if subcats, exists := CategoryHierarchy[topLevel]; exists {
		return subcats
	}
	return []string{}
}

// GetTopLevelCategory extracts the top-level category from a full category
func GetTopLevelCategory(category string) string {
	parts := strings.Split(category, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return category
}