package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"scifind-backend/internal/services"
)

// MCPServer implements the Model Context Protocol server
type MCPServer struct {
	searchService *services.SearchService
	paperService  *services.PaperService
	logger        *slog.Logger
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(
	searchService *services.SearchService,
	paperService *services.PaperService,
	logger *slog.Logger,
) *MCPServer {
	return &MCPServer{
		searchService: searchService,
		paperService:  paperService,
		logger:        logger,
	}
}

// MCPRequest represents an MCP protocol request
type MCPRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
	ID     interface{}     `json:"id,omitempty"`
}

// MCPResponse represents an MCP protocol response
type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
	ID     interface{} `json:"id,omitempty"`
}

// MCPError represents an MCP protocol error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Common MCP error codes
const (
	MCPErrorParseError     = -32700
	MCPErrorInvalidRequest = -32600
	MCPErrorMethodNotFound = -32601
	MCPErrorInvalidParams  = -32602
	MCPErrorInternalError  = -32603
)

// HandleRequest processes an MCP request and returns a response
func (s *MCPServer) HandleRequest(ctx context.Context, request *MCPRequest) *MCPResponse {
	response := &MCPResponse{ID: request.ID}

	switch request.Method {
	case "search":
		result, err := s.handleSearch(ctx, request.Params)
		if err != nil {
			response.Error = &MCPError{
				Code:    MCPErrorInternalError,
				Message: err.Error(),
			}
		} else {
			response.Result = result
		}

	case "get_paper":
		result, err := s.handleGetPaper(ctx, request.Params)
		if err != nil {
			response.Error = &MCPError{
				Code:    MCPErrorInternalError,
				Message: err.Error(),
			}
		} else {
			response.Result = result
		}

	case "list_capabilities":
		response.Result = s.getCapabilities()

	case "get_schema":
		result, err := s.handleGetSchema(ctx, request.Params)
		if err != nil {
			response.Error = &MCPError{
				Code:    MCPErrorInvalidParams,
				Message: err.Error(),
			}
		} else {
			response.Result = result
		}

	case "ping":
		response.Result = map[string]interface{}{
			"status":  "ok",
			"server":  "scifind-backend",
			"version": "1.0.0",
		}

	default:
		response.Error = &MCPError{
			Code:    MCPErrorMethodNotFound,
			Message: fmt.Sprintf("method '%s' not found", request.Method),
		}
	}

	return response
}

// SearchParams represents parameters for the search method
type SearchParams struct {
	Query     string            `json:"query"`
	Providers []string          `json:"providers,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Offset    int               `json:"offset,omitempty"`
	SortBy    string            `json:"sort_by,omitempty"`
	SortOrder string            `json:"sort_order,omitempty"`
}

// handleSearch processes a search request
func (s *MCPServer) handleSearch(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var searchParams SearchParams
	if err := json.Unmarshal(params, &searchParams); err != nil {
		return nil, fmt.Errorf("invalid search parameters: %w", err)
	}

	// Set defaults
	if searchParams.Limit == 0 {
		searchParams.Limit = 20
	}
	if searchParams.SortBy == "" {
		searchParams.SortBy = "relevance"
	}
	if searchParams.SortOrder == "" {
		searchParams.SortOrder = "desc"
	}

	// Create search request
	searchRequest := &services.SearchRequest{
		Query:     searchParams.Query,
		Providers: searchParams.Providers,
		Filters:   searchParams.Filters,
		Limit:     searchParams.Limit,
		Offset:    searchParams.Offset,
	}

	// Execute search
	result, err := s.searchService.Search(ctx, searchRequest)
	if err != nil {
		s.logger.Error("MCP search failed",
			slog.String("query", searchParams.Query),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	s.logger.Info("MCP search completed",
		slog.String("query", searchParams.Query),
		slog.Int("results", len(result.Papers)),
		slog.Duration("duration", result.Duration),
	)

	return result, nil
}

// GetPaperParams represents parameters for the get_paper method
type GetPaperParams struct {
	ID       string `json:"id"`
	Provider string `json:"provider,omitempty"`
}

// handleGetPaper processes a get paper request
func (s *MCPServer) handleGetPaper(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var paperParams GetPaperParams
	if err := json.Unmarshal(params, &paperParams); err != nil {
		return nil, fmt.Errorf("invalid paper parameters: %w", err)
	}

	// Get paper from database
	paper, err := s.paperService.GetByID(ctx, paperParams.ID)
	if err != nil {
		s.logger.Error("MCP get paper failed",
			slog.String("paper_id", paperParams.ID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to get paper: %w", err)
	}

	s.logger.Info("MCP get paper completed",
		slog.String("paper_id", paperParams.ID),
		slog.String("title", paper.Title),
	)

	return paper, nil
}

// getCapabilities returns the MCP server capabilities
func (s *MCPServer) getCapabilities() interface{} {
	return map[string]interface{}{
		"methods": []string{
			"search",
			"get_paper",
			"list_capabilities",
			"get_schema",
			"ping",
		},
		"schemas": []string{
			"search_request",
			"search_response",
			"paper",
			"author",
			"category",
		},
		"version": "1.0.0",
		"server_info": map[string]interface{}{
			"name":        "scifind-backend",
			"description": "Scientific paper search and discovery platform",
			"version":     "1.0.0",
		},
		"provider_support": []string{
			"arxiv",
			"semantic_scholar",
			"exa",
			"tavily",
		},
	}
}

// GetSchemaParams represents parameters for the get_schema method
type GetSchemaParams struct {
	Schema string `json:"schema"`
}

// handleGetSchema returns schema definitions
func (s *MCPServer) handleGetSchema(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var schemaParams GetSchemaParams
	if err := json.Unmarshal(params, &schemaParams); err != nil {
		return nil, fmt.Errorf("invalid schema parameters: %w", err)
	}

	switch schemaParams.Schema {
	case "search_request":
		return getSearchRequestSchema(), nil
	case "search_response":
		return getSearchResponseSchema(), nil
	case "paper":
		return getPaperSchema(), nil
	case "author":
		return getAuthorSchema(), nil
	case "category":
		return getCategorySchema(), nil
	default:
		return nil, fmt.Errorf("unknown schema: %s", schemaParams.Schema)
	}
}

// Schema definitions

func getSearchRequestSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query string",
				"minLength":   1,
				"maxLength":   1000,
			},
			"providers": map[string]interface{}{
				"type":        "array",
				"description": "List of search providers to use",
				"items": map[string]interface{}{
					"type": "string",
					"enum": []string{"arxiv", "semantic_scholar", "exa", "tavily"},
				},
			},
			"filters": map[string]interface{}{
				"type":        "object",
				"description": "Search filters",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results",
				"minimum":     1,
				"maximum":     100,
				"default":     20,
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Result offset for pagination",
				"minimum":     0,
				"default":     0,
			},
		},
		"required": []string{"query"},
	}
}

func getSearchResponseSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"papers": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"$ref": "#/schemas/paper"},
			},
			"total_count": map[string]interface{}{
				"type":        "integer",
				"description": "Total number of results available",
			},
			"duration": map[string]interface{}{
				"type":        "string",
				"description": "Search duration",
			},
			"providers": map[string]interface{}{
				"type":        "object",
				"description": "Provider-specific results",
			},
		},
	}
}

func getPaperSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Unique paper identifier",
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Paper title",
			},
			"abstract": map[string]interface{}{
				"type":        "string",
				"description": "Paper abstract",
			},
			"authors": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"$ref": "#/schemas/author"},
			},
			"categories": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"$ref": "#/schemas/category"},
			},
			"published_at": map[string]interface{}{
				"type":        "string",
				"format":      "date-time",
				"description": "Publication date",
			},
			"doi": map[string]interface{}{
				"type":        "string",
				"description": "Digital Object Identifier",
			},
			"arxiv_id": map[string]interface{}{
				"type":        "string",
				"description": "ArXiv identifier",
			},
		},
	}
}

func getAuthorSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Unique author identifier",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Author name",
			},
			"affiliation": map[string]interface{}{
				"type":        "string",
				"description": "Author affiliation",
			},
			"orcid": map[string]interface{}{
				"type":        "string",
				"description": "ORCID identifier",
			},
		},
	}
}

func getCategorySchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "Unique category identifier",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Category name",
			},
			"source": map[string]interface{}{
				"type":        "string",
				"description": "Category source (e.g., arxiv, msc)",
			},
			"source_code": map[string]interface{}{
				"type":        "string",
				"description": "Original category code",
			},
		},
	}
}

// ProcessMessage is a convenience method for processing raw JSON messages
func (s *MCPServer) ProcessMessage(ctx context.Context, message []byte) ([]byte, error) {
	var request MCPRequest
	if err := json.Unmarshal(message, &request); err != nil {
		response := &MCPResponse{
			Error: &MCPError{
				Code:    MCPErrorParseError,
				Message: "invalid JSON in request",
			},
		}
		return json.Marshal(response)
	}

	response := s.HandleRequest(ctx, &request)
	return json.Marshal(response)
}
