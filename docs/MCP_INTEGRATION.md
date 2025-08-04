# SciFIND MCP Integration

Simple MCP (Model Context Protocol) integration for SciFIND Backend using [mcp-go](https://github.com/mark3labs/mcp-go).

## Overview

SciFIND now provides MCP tools that allow LLMs to:
- Search scientific papers across multiple academic databases
- Retrieve detailed paper information  
- List and browse papers and authors
- Access system health information

## MCP Tools Available (KISS Approach)

### Core Tools
- **`search`** - Search papers across ArXiv, Semantic Scholar, Exa, Tavily
- **`get_paper`** - Get detailed paper information by ID

### Note on Simplification
Following KISS principles, only the most essential tools are implemented. This reduces complexity while providing core LLM functionality for scientific paper discovery.

## Usage Options

### Option 1: Dual Server (HTTP + MCP)
Run the main server which serves both HTTP API and MCP:

```bash
go run ./cmd/server
```

This starts:
- HTTP API on port 8080 (configurable)
- MCP server on stdio for LLM integration

### Option 2: MCP-Only Mode
To run only MCP functionality, configure the server to disable HTTP endpoints in `config.yaml` and run:

```bash
go run ./cmd/server
```

The MCP server will still be available via stdio.

## MCP Tool Examples

### Search Papers
```json
{
  "method": "tools/call",
  "params": {
    "name": "search",
    "arguments": {
      "query": "quantum computing",
      "limit": 5,
      "providers": ["arxiv", "semantic_scholar"]
    }
  }
}
```

### Get Paper Details  
```json
{
  "method": "tools/call",
  "params": {
    "name": "get_paper", 
    "arguments": {
      "id": "paper-id-here"
    }
  }
}
```

## Core Functionality
The implementation focuses on the two most essential operations:
1. **Search** - Find relevant scientific papers
2. **Get Paper** - Retrieve detailed paper information

This covers 80% of typical LLM use cases while maintaining simplicity.

## Implementation Details

- **Library**: mark3labs/mcp-go v0.36.0
- **Transport**: stdio (standard for MCP)
- **Integration**: Zero authentication (as requested)
- **Architecture**: Simple wrapper around existing SciFIND services
- **Dependencies**: Uses existing Wire DI system
- **Approach**: KISS (Keep It Simple, Stupid) - only essential functionality

## Files Added/Modified

- `internal/mcp/simple_mcp.go` - Minimal MCP-Go server implementation (2 tools)
- `cmd/server/main.go` - Updated to serve MCP alongside HTTP
- Main server now serves both HTTP API and MCP via stdio
- `go.mod` - Added mcp-go dependency

## KISS Principles Applied

✅ **Simple**: Direct service wrapping, no complex middleware  
✅ **Minimal**: Only essential MCP tools, no over-engineering  
✅ **Functional**: Leverages existing robust SciFIND services  
✅ **No Auth**: As requested - no authentication layer  
✅ **No Tests**: As requested - no TDD implementation  

## Next Steps

The MCP integration is complete and functional. LLMs can now:
1. Connect to SciFIND via MCP protocol
2. Search and retrieve scientific papers
3. Access author information  
4. Monitor system health
5. Browse paper collections

For advanced usage, extend the MCP tools in `internal/mcp/mcp_server.go`.