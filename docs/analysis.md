# SciFind Backend Repository Analysis

## Project Overview

**SciFind** is a Go-based REST API for scientific literature search across multiple academic databases including ArXiv, Semantic Scholar, Exa, and Tavily. It provides a unified interface for searching and aggregating results from multiple providers into a single response.

## Architecture Analysis

### Core Architecture Pattern
The application follows **Clean Architecture** principles with a layered structure:
- **API Layer**: HTTP handlers and routing
- **Service Layer**: Business logic and provider coordination
- **Repository Layer**: Data persistence
- **Provider Layer**: External API integrations
- **Model Layer**: Data structures and validation

### Key Design Patterns
- **Dependency Injection**: Managed using Google Wire
- **Repository Pattern**: For data access abstraction
- **Provider Pattern**: For external service integrations
- **Circuit Breaker**: For fault tolerance
- **Strategy Pattern**: For search aggregation strategies

## Technology Stack

### Core Dependencies
- **Language**: Go 1.24+
- **Web Framework**: Gin-gonic/gin v1.10.1
- **ORM**: GORM v1.30.1 with support for PostgreSQL and SQLite
- **Configuration**: Viper v1.20.1 for configuration management
- **Testing**: Testcontainers-go v0.38.0 for integration testing
- **Dependency Injection**: Google Wire v0.6.0

### External Integrations
- **Database**: PostgreSQL 15+ (production), SQLite (development)
- **Message Queue**: NATS with optional embedded server
- **Academic Providers**:
  - ArXiv (no API key required)
  - Semantic Scholar (optional API key)
  - Exa (API key required)
  - Tavily (API key required)

## Project Structure

```
scifind-backend/
├── cmd/server/              # Main application entry point
├── internal/
│   ├── api/                 # HTTP layer
│   │   ├── handlers/       # Request handlers
│   │   └── middleware/       # HTTP middleware
│   ├── config/              # Configuration management
│   ├── models/              # Data models
│   ├── providers/           # External API integrations
│   ├── repository/          # Data persistence
│   ├── services/            # Business logic
│   └── messaging/           # NATS integration
├── docs/                    # API documentation
├── test/                    # Test suites
├── memory/                  # Memory management
└── configs/                 # Configuration files
```

## Key Features

### Search Capabilities
- **Multi-provider search**: Search across ArXiv, Semantic Scholar, Exa, and Tavily
- **Aggregation strategies**: 
  - Merge results from all providers
  - First successful result
  - Fastest response
  - Best quality
  - Round-robin selection
- **Deduplication**: Automatic removal of duplicate papers
- **Caching**: Result caching for improved performance

### Data Models
- **Paper**: Comprehensive scientific paper metadata
  - DOI, ArXiv ID support
  - Author relationships
  - Citation tracking
  - Category classification
  - Full-text and PDF availability
  - Quality scoring
- **Author**: Author information and relationships
- **Category**: Paper classification system

### API Endpoints
- `GET /v1/search` - Search papers across providers
- `GET /v1/papers` - List papers
- `GET /v1/papers/{id}` - Get specific paper
- `GET /v1/authors` - List authors
- `GET /v1/authors/{id}` - Get author details
- `GET /health` - Health check
- `GET /swagger/index.html` - API documentation

### Configuration Management
- **Flexible configuration**: YAML configuration files
- **Environment variables**: Override config file settings
- **Provider configuration**: Per-provider settings and API keys
- **Database switching**: Support for PostgreSQL and SQLite
- **Feature flags**: Enable/disable specific providers

## External Provider Integrations

### ArXiv Provider
- **Base URL**: https://export.arxiv.org/api/query
- **Features**: Physics, mathematics, computer science papers
- **Rate Limiting**: 3-second intervals
- **Authentication**: None required

### Semantic Scholar Provider
- **Base URL**: https://api.semanticscholar.org/graph/v1
- **Features**: 200M+ academic papers
- **Rate Limiting**: Configurable timeout
- **Authentication**: Optional API key for enhanced rate limits

### Exa Provider
- **Base URL**: https://api.exa.ai
- **Features**: Neural search capabilities
- **Authentication**: Required API key
- **Status**: Optional (disabled by default)

### Tavily Provider
- **Base URL**: https://api.tavily.com
- **Features**: Real-time web search
- **Authentication**: Required API key
- **Status**: Optional (disabled by default)

## Development & Testing

### Build Commands
- `make build` - Build the application binary
- `make dev` - Run in development mode with hot reload
- `make run` - Build and run the application
- `make test` - Run all tests
- `make test-watch` - Run tests in watch mode
- `make fmt` - Format Go code
- `make lint` - Run linter (golangci-lint)

### Testing Strategy
- **Unit Tests**: Individual component testing
- **Integration Tests**: Database and provider integration
- **E2E Tests**: Full application testing
- **Benchmarks**: Performance testing
- **Testcontainers**: Container-based testing

### Docker Support
- **Development**: `docker-compose up -d`
- **Production**: `docker build -t scifind-backend .`
- **Multi-stage builds**: Optimized production images

## Configuration Reference

### Server Configuration
```yaml
server:
  port: 8080
  host: "0.0.0.0"
  mode: "debug"  # debug, release, test
```

### Database Configuration
```yaml
database:
  type: "sqlite"  # sqlite or postgres
  sqlite:
    path: "./scifind.db"
    auto_migrate: true
  postgresql:
    dsn: "host=localhost user=postgres dbname=scifind sslmode=disable"
    max_connections: 25
    auto_migrate: true
```

### Provider Configuration
```yaml
providers:
  arxiv:
    enabled: true
    base_url: "https://export.arxiv.org/api/query"
    timeout: "30s"
  semantic_scholar:
    enabled: true
    api_key: ""  # Optional
    timeout: "15s"
  exa:
    enabled: false
    api_key: ""  # Required
  tavily:
    enabled: false
    api_key: ""  # Required
```

### NATS Configuration
```yaml
nats:
  url: "nats://localhost:4222"
  embedded:
    enabled: false
    host: "0.0.0.0"
    port: 4222
```

## Library Documentation Summary

### Gin Web Framework
- **Purpose**: HTTP web framework for Go
- **Features**: High performance, middleware support, JSON validation
- **Usage**: REST API development with clean routing
- **Key Methods**: GET, POST, PUT, DELETE, middleware chaining

### GORM ORM
- **Purpose**: Object-Relational Mapping for Go
- **Features**: Database abstraction, migrations, associations
- **Usage**: Database operations with SQLite/PostgreSQL
- **Key Methods**: AutoMigrate, Create, First, Update, Delete

### Testcontainers-Go
- **Purpose**: Container-based testing
- **Features**: Docker container management, database testing
- **Usage**: Integration testing with real databases
- **Key Methods**: RunContainer, Terminate, ConnectionString

### Viper Configuration
- **Purpose**: Configuration management
- **Features**: YAML/JSON support, environment variables
- **Usage**: Application configuration and settings
- **Key Methods**: SetConfigFile, GetString, GetBool, Unmarshal

## Security Features

### API Security
- **Rate limiting**: Configurable request limits
- **CORS**: Cross-origin resource sharing configuration
- **Security headers**: HTTP security headers middleware
- **Input validation**: Request validation and sanitization

### Authentication
- **API keys**: Optional API key authentication
- **Provider keys**: Secure storage of external API keys
- **Environment variables**: Secure configuration management

## Performance Considerations

### Optimization Features
- **Caching**: Result caching for frequently requested data
- **Connection pooling**: Database connection management
- **Concurrent processing**: Parallel provider searches
- **Request timeouts**: Configurable timeout handling

### Monitoring
- **Health checks**: Database and provider health monitoring
- **Metrics**: Performance and usage metrics
- **Logging**: Structured logging with configurable levels

## Deployment Strategies

### Local Development
```bash
git clone https://github.com/scifind/backend.git
cd scifind-backend
make dev
```

### Docker Production
```bash
docker build -t scifind-backend .
docker run -p 8080:8080 -e SCIFIND_DATABASE_TYPE=postgres scifind-backend
```

### Cloud Deployment
- **Containerized**: Ready for Kubernetes deployment
- **Database**: Supports cloud databases (PostgreSQL)
- **Configuration**: Environment-based configuration
- **Scalability**: Horizontal scaling support

## Future Enhancements

### MCP Integration
- **Model Context Protocol**: Future support for MCP
- **Method support**: search, get_paper, list_capabilities
- **Schema validation**: Structured API responses

### Additional Features
- **Enhanced caching**: Redis integration
- **Background processing**: Job queue support
- **API versioning**: Versioned API endpoints
- **Documentation**: Enhanced OpenAPI/Swagger specs

## Code Quality Standards

### Go Conventions
- **Naming**: Following Go naming conventions
- **Structure**: Clean architecture patterns
- **Testing**: Comprehensive test coverage
- **Documentation**: Godoc comments

### Best Practices
- **Error handling**: Proper error wrapping and handling
- **Logging**: Structured logging with context
- **Configuration**: Environment-based configuration
- **Security**: Secure credential management