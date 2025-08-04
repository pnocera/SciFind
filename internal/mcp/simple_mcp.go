package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"scifind-backend/internal/services"
)

// SimpleMCPServer is a minimal MCP implementation for SciFIND
type SimpleMCPServer struct {
	server        *server.MCPServer
	searchService *services.SearchService
	paperService  *services.PaperService
	authorService *services.AuthorService
	logger        *slog.Logger
}

// NewSimpleMCPServer creates a simple MCP server
func NewSimpleMCPServer(
	searchService *services.SearchService,
	paperService *services.PaperService,
	authorService *services.AuthorService,
	logger *slog.Logger,
) *SimpleMCPServer {
	// Create basic MCP server
	mcpServer := server.NewMCPServer(
		"SciFIND Backend",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s := &SimpleMCPServer{
		server:        mcpServer,
		searchService: searchService,
		paperService:  paperService,
		authorService: authorService,
		logger:        logger,
	}

	// Register simple tools
	s.registerSimpleTools()
	return s
}

// registerSimpleTools adds basic MCP tools
func (s *SimpleMCPServer) registerSimpleTools() {
	// Simple search tool
	searchTool := mcp.NewTool("search",
		mcp.WithDescription("Search scientific papers"),
		mcp.WithString("query", mcp.Required()),
	)
	s.server.AddTool(searchTool, s.handleSearch)

	// Simple get paper tool  
	getPaperTool := mcp.NewTool("get_paper",
		mcp.WithDescription("Get paper by ID"),
		mcp.WithString("id", mcp.Required()),
	)
	s.server.AddTool(getPaperTool, s.handleGetPaper)

	s.logger.Info("Registered 2 MCP tools: search, get_paper")
}

// handleSearch processes search requests
func (s *SimpleMCPServer) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments safely
	argsMap, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	// Get query parameter
	query, ok := argsMap["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query parameter required"), nil
	}

	// Create search request
	searchReq := &services.SearchRequest{
		Query:  query,
		Limit:  10, // Keep it simple
		Offset: 0,
	}

	// Execute search
	result, err := s.searchService.Search(ctx, searchReq)
	if err != nil {
		s.logger.Error("MCP search failed", slog.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	s.logger.Info("MCP search completed", 
		slog.String("query", query),
		slog.Int("results", len(result.Papers)))

	// Return JSON result
	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGetPaper processes get paper requests
func (s *SimpleMCPServer) handleGetPaper(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments safely
	argsMap, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments format"), nil
	}

	// Get ID parameter
	paperID, ok := argsMap["id"].(string)
	if !ok || paperID == "" {
		return mcp.NewToolResultError("id parameter required"), nil
	}

	// Get paper
	paper, err := s.paperService.GetByID(ctx, paperID)
	if err != nil {
		s.logger.Error("MCP get paper failed", 
			slog.String("paper_id", paperID),
			slog.String("error", err.Error()))
		return mcp.NewToolResultError(fmt.Sprintf("get paper failed: %v", err)), nil
	}

	s.logger.Info("MCP get paper completed", slog.String("paper_id", paperID))

	// Return JSON result
	resultJSON, _ := json.Marshal(paper)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// ServeStdio starts the MCP server via stdio
func (s *SimpleMCPServer) ServeStdio() error {
	s.logger.Info("Starting simple MCP server via stdio")
	return server.ServeStdio(s.server)
}

// GetServer returns the underlying server
func (s *SimpleMCPServer) GetServer() *server.MCPServer {
	return s.server
}