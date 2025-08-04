# SciFIND

<div align="center">
  <img src="docs/scifind.png" alt="SciFIND Logo" width="300">
</div>

A Go-based REST API for scientific literature search across multiple academic databases.

## Overview

SciFIND Backend provides a unified API for searching scientific papers from ArXiv, Semantic Scholar, Exa, and Tavily. It aggregates results from multiple providers into a single response.

## Requirements

- Go 1.24+
- PostgreSQL 15+ (production) or SQLite (development)
- Docker (optional)

## Quick Start

### Using Docker

```bash
git clone https://github.com/scifind/backend.git
cd scifind-backend
docker-compose up -d
```

Test the API:
```bash
curl "http://localhost:8080/v1/search?query=quantum+computing&limit=5"
```

### Local Development

```bash
git clone https://github.com/scifind/backend.git
cd scifind-backend
go mod download
go generate ./cmd/server
go build -o scifind-server ./cmd/server
./scifind-server
```

## API Endpoints

- `GET /v1/search` - Search papers across providers
- `GET /v1/papers` - List papers
- `GET /v1/papers/{id}` - Get specific paper
- `GET /v1/authors` - List authors
- `GET /v1/authors/{id}` - Get author details
- `GET /health` - Health check
- `GET /swagger/index.html` - API documentation

## Configuration

Create a `config.yaml` file:

```yaml
server:
  port: 8080
  mode: debug

database:
  type: sqlite
  sqlite:
    path: ./scifind.db
    auto_migrate: true

providers:
  arxiv:
    enabled: true
  semantic_scholar:
    enabled: true
    api_key: ""  # Optional
  exa:
    enabled: false
    api_key: ""  # Required
  tavily:
    enabled: false
    api_key: ""  # Required
```

Environment variables override config file settings:
- `SCIFIND_SERVER_PORT` - Server port
- `SCIFIND_DATABASE_TYPE` - Database type
- `SCIFIND_PROVIDERS_EXA_API_KEY` - Exa API key
- `SCIFIND_PROVIDERS_TAVILY_API_KEY` - Tavily API key

## Development

### Build Commands

```bash
go build ./cmd/server          # Build server
go generate ./cmd/server       # Generate Wire dependency injection
go test ./...                  # Run tests
go run ./cmd/server            # Run server directly
```

### Project Structure

```
cmd/server/          # Main application
internal/
  api/               # HTTP handlers and routing
  config/            # Configuration management
  models/            # Data models
  providers/         # External API providers
  repository/        # Database layer
  services/          # Business logic
docs/                # Swagger documentation
```

## Providers

### ArXiv
- No API key required
- Physics, mathematics, computer science papers
- Free access

### Semantic Scholar
- API key optional for basic usage
- Enhanced rate limits with API key
- 200M+ academic papers

### Exa
- API key required
- Neural search capabilities
- Paid service

### Tavily
- API key required
- Real-time web search
- Paid service

## Deployment

### Docker Production

```bash
docker build -t scifind-backend .
docker run -p 8080:8080 -e SCIFIND_DATABASE_TYPE=postgres scifind-backend
```

### Database Setup

For PostgreSQL:
```bash
createdb scifind
export SCIFIND_DATABASE_TYPE=postgres
export SCIFIND_DATABASE_POSTGRES_DSN="host=localhost user=postgres dbname=scifind sslmode=disable"
```

## Architecture

The application follows clean architecture principles:

- **Handlers**: HTTP request/response handling
- **Services**: Business logic and provider coordination
- **Repository**: Data persistence layer
- **Providers**: External API integrations

Dependency injection is managed using Google Wire.

## MCP Server

SciFIND provides a Model Context Protocol (MCP) server for AI assistants to search scientific papers.

### MCP Tools

- `search` - Search scientific papers by query
- `get_paper` - Get paper details by ID

### Usage

Start the MCP server via stdio:
```bash
go run ./cmd/server --mcp
```

Search example:
```json
{
  "tool": "search",
  "arguments": {
    "query": "quantum computing"
  }
}
```

Get paper example:
```json
{
  "tool": "get_paper",
  "arguments": {
    "id": "paper-id-123"
  }
}
```

### MCP Configuration

The MCP server can be configured in `config.yaml`:

```yaml
mcp:
  enabled: true
  mode: stdio  # or http
  port: 8081   # for http mode
```

## License

MIT License