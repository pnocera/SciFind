# Contributing to SciFind Backend

Thank you for your interest in contributing to SciFind! This guide will help you get started with development, understand our coding standards, and learn about our contribution process.

## Table of Contents
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Project Structure](#project-structure)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Git Workflow](#git-workflow)
- [Pull Request Process](#pull-request-process)
- [Code Review Guidelines](#code-review-guidelines)
- [Adding New Features](#adding-new-features)
- [Documentation Standards](#documentation-standards)
- [Community Guidelines](#community-guidelines)

## Getting Started

### Prerequisites

Before contributing, ensure you have the following installed:

- **Go 1.24+** - [Installation Guide](https://golang.org/doc/install)
- **Docker & Docker Compose** - [Installation Guide](https://docs.docker.com/get-docker/)
- **Git** - [Installation Guide](https://git-scm.com/downloads)
- **Make** - Usually pre-installed on Unix systems
- **golangci-lint** - [Installation Guide](https://golangci-lint.run/usage/install/)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/scifind-backend.git
   cd scifind-backend
   ```

3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/scifind/scifind-backend.git
   ```

### Initial Setup

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Generate Wire dependencies:
   ```bash
   go generate ./cmd/server
   ```

3. Set up development configuration:
   ```bash
   cp configs/config.example.yaml configs/config.dev.yaml
   ```

4. Start development environment:
   ```bash
   make docker-up
   make dev
   ```

5. Verify setup:
   ```bash
   curl http://localhost:8080/health
   ```

## Development Environment

### Development Commands

```bash
# Development
make dev              # Run in development mode with hot reload
make build            # Build the application binary
make run              # Build and run the application

# Testing
make test             # Run all tests
make test-watch       # Run tests in watch mode
make test-coverage    # Run tests with coverage report

# Code Quality
make fmt              # Format Go code
make lint             # Run linter
make check            # Run both format and lint

# Database
make migrate-up       # Run database migrations
make migrate-down     # Rollback database migrations
make migrate-create NAME=migration_name  # Create new migration

# Docker
make docker-up        # Start development environment
make docker-down      # Stop development environment
make docker-logs      # View logs from all services
```

### IDE Configuration

#### VS Code
Recommended extensions:
- Go (official Go extension)
- Go Test Explorer
- Docker
- YAML
- GitLens

Settings (`.vscode/settings.json`):
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": "package",
    "go.formatTool": "goimports",
    "go.generateTestsFlags": ["-exported"]
}
```

#### GoLand/IntelliJ
- Enable Go modules support
- Configure golangci-lint as external tool
- Set up file watchers for auto-formatting

## Project Structure

Understanding the project structure helps with navigation and contribution:

```
scifind-backend/
├── cmd/
│   └── server/           # Application entry point
├── internal/
│   ├── api/             # HTTP API layer
│   │   ├── handlers/    # Request handlers
│   │   ├── middleware/  # HTTP middleware
│   │   └── router.go    # Route definitions
│   ├── config/          # Configuration management
│   ├── errors/          # Error handling utilities
│   ├── messaging/       # NATS messaging system
│   ├── models/          # Data models
│   ├── providers/       # External API providers
│   ├── repository/      # Data access layer
│   └── services/        # Business logic layer
├── test/
│   ├── unit/           # Unit tests
│   ├── integration/    # Integration tests
│   ├── e2e/           # End-to-end tests
│   ├── benchmarks/    # Performance benchmarks
│   └── testutil/      # Test utilities
├── docs/              # Documentation
├── configs/           # Configuration files
└── deployments/       # Deployment configurations
```

### Layer Responsibilities

| Layer | Purpose | Dependencies |
|-------|---------|-------------|
| **API** | HTTP request/response handling | Services |
| **Services** | Business logic and orchestration | Repository, Providers |
| **Repository** | Data access and persistence | Database |
| **Providers** | External API integration | HTTP clients |
| **Models** | Data structures and validation | None |

## Coding Standards

### Go Style Guide

We follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and [Effective Go](https://golang.org/doc/effective_go.html).

#### Naming Conventions

```go
// Package names: lowercase, no underscores
package providers

// Interface names: end with 'er' or descriptive noun
type SearchProvider interface{}
type ProviderManager interface{}

// Struct names: PascalCase
type SearchService struct{}

// Public methods: PascalCase
func (s *SearchService) Search(ctx context.Context, query string) error {}

// Private methods: camelCase
func (s *SearchService) validateQuery(query string) error {}

// Constants: descriptive names
const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)
```

#### Error Handling

```go
// Use descriptive error messages
func (s *SearchService) Search(ctx context.Context, query string) (*SearchResult, error) {
    if query == "" {
        return nil, fmt.Errorf("search query cannot be empty")
    }
    
    result, err := s.provider.Search(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to search provider %s: %w", s.provider.Name(), err)
    }
    
    return result, nil
}

// Custom error types for specific cases
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}
```

#### Context Usage

```go
// Always accept context as first parameter
func (s *SearchService) Search(ctx context.Context, query string) error {
    // Use context for cancellation and timeouts
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Pass context to all downstream calls
    return s.provider.Search(ctx, query)
}
```

#### Logging

```go
// Use structured logging with slog
func (s *SearchService) Search(ctx context.Context, query string) error {
    logger := s.logger.With(
        slog.String("operation", "search"),
        slog.String("query", query),
        slog.String("request_id", getRequestID(ctx)),
    )
    
    logger.Info("Starting search operation")
    
    result, err := s.provider.Search(ctx, query)
    if err != nil {
        logger.Error("Search operation failed", slog.String("error", err.Error()))
        return err
    }
    
    logger.Info("Search operation completed",
        slog.Int("result_count", len(result.Papers)),
        slog.Duration("duration", result.Duration))
    
    return nil
}
```

### Code Organization

#### File Structure
```go
// Each file should have a clear purpose
// service.go - main service implementation
// interfaces.go - service interfaces
// types.go - data types specific to service
// config.go - configuration structures
// errors.go - custom error types
```

#### Dependency Injection
```go
// Use interfaces for dependencies
type SearchService struct {
    logger   *slog.Logger
    repo     repository.SearchRepository
    provider providers.ProviderManager
    cache    cache.Manager
}

// Constructor with interface parameters
func NewSearchService(
    logger *slog.Logger,
    repo repository.SearchRepository,
    provider providers.ProviderManager,
    cache cache.Manager,
) services.SearchServiceInterface {
    return &SearchService{
        logger:   logger,
        repo:     repo,
        provider: provider,
        cache:    cache,
    }
}
```

### Performance Guidelines

#### Database Operations
```go
// Use efficient queries
func (r *PaperRepository) SearchByKeywords(ctx context.Context, keywords []string) ([]*models.Paper, error) {
    // Use proper indexing and LIMIT clauses
    query := r.db.WithContext(ctx).
        Where("title ILIKE ANY(?)", pq.Array(keywords)).
        Limit(100).
        Order("created_at DESC")
    
    var papers []*models.Paper
    if err := query.Find(&papers).Error; err != nil {
        return nil, fmt.Errorf("failed to search papers: %w", err)
    }
    
    return papers, nil
}
```

#### Concurrent Operations
```go
// Use goroutines and channels appropriately
func (m *ProviderManager) SearchAll(ctx context.Context, query *SearchQuery) (*AggregatedResult, error) {
    providers := m.GetEnabledProviders()
    results := make(chan *ProviderResult, len(providers))
    
    // Launch concurrent searches
    for _, provider := range providers {
        go func(p providers.SearchProvider) {
            defer func() {
                if r := recover(); r != nil {
                    m.logger.Error("Provider search panic", 
                        slog.String("provider", p.Name()),
                        slog.Any("panic", r))
                }
            }()
            
            result, err := p.Search(ctx, query)
            results <- &ProviderResult{
                Provider: p.Name(),
                Result:   result,
                Error:    err,
            }
        }(provider)
    }
    
    // Collect results with timeout
    return m.aggregateResults(ctx, results, len(providers))
}
```

## Testing Guidelines

### Test Structure

```go
// Unit test example
func TestSearchService_Search(t *testing.T) {
    // Arrange
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    mockRepo := &mocks.MockSearchRepository{}
    mockProvider := &mocks.MockProviderManager{}
    
    service := NewSearchService(logger, mockRepo, mockProvider, nil)
    
    // Mock expectations
    mockProvider.EXPECT().
        SearchAll(gomock.Any(), gomock.Any()).
        Return(&providers.AggregatedResult{
            Papers: []*models.Paper{{Title: "Test Paper"}},
        }, nil)
    
    // Act
    result, err := service.Search(context.Background(), &services.SearchRequest{
        Query: "test query",
        Limit: 10,
    })
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Len(t, result.Papers, 1)
    assert.Equal(t, "Test Paper", result.Papers[0].Title)
}
```

### Test Categories

#### Unit Tests (`test/unit/`)
- Test individual functions and methods
- Use mocks for dependencies
- Focus on business logic
- Should be fast (< 1ms per test)

#### Integration Tests (`test/integration/`)
- Test component interactions
- Use testcontainers for databases
- Test real HTTP endpoints
- Verify data persistence

#### End-to-End Tests (`test/e2e/`)
- Test complete user workflows
- Use real external services (in test environment)
- Verify system behavior under load
- Test deployment scenarios

### Test Utilities

```go
// Create test utilities in test/testutil/
func CreateTestDatabase(t *testing.T) *repository.Database {
    db, err := repository.NewDatabase(&config.Config{
        Database: config.DatabaseConfig{
            Type: "sqlite",
            SQLite: config.SQLiteConfig{
                Path: ":memory:",
            },
        },
    }, slog.Default())
    
    require.NoError(t, err)
    return db
}

func CreateTestLogger(t *testing.T) *slog.Logger {
    return slog.New(slog.NewTextHandler(io.Discard, nil))
}
```

## Git Workflow

### Branch Naming

```bash
# Feature branches
feature/add-semantic-scholar-provider
feature/improve-search-performance

# Bug fixes
fix/rate-limiting-bypass
fix/memory-leak-in-provider

# Documentation
docs/update-api-reference
docs/add-deployment-guide

# Refactoring
refactor/extract-provider-interface
refactor/simplify-error-handling
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# Format: type(scope): description

# Examples
feat(providers): add Semantic Scholar API integration
fix(api): resolve rate limiting bypass vulnerability
docs(readme): update installation instructions
refactor(services): extract search interface
test(integration): add provider health check tests
perf(database): optimize paper search query
chore(deps): update Go dependencies

# Breaking changes
feat(api)!: change search response format

# With body and footer
feat(providers): add circuit breaker support

Add circuit breaker pattern to prevent cascade failures when
external providers are unavailable. Configurable failure
thresholds and recovery strategies.

Closes #123
```

### Workflow Steps

1. **Sync with upstream:**
   ```bash
   git checkout main
   git fetch upstream
   git merge upstream/main
   git push origin main
   ```

2. **Create feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make changes and commit:**
   ```bash
   git add .
   git commit -m "feat(scope): add new feature"
   ```

4. **Push and create PR:**
   ```bash
   git push origin feature/your-feature-name
   # Create PR through GitHub UI
   ```

5. **Keep branch updated:**
   ```bash
   git checkout main
   git pull upstream main
   git checkout feature/your-feature-name
   git rebase main
   ```

## Pull Request Process

### Before Submitting

- [ ] Run tests: `make test`
- [ ] Run linter: `make lint`
- [ ] Update documentation if needed
- [ ] Add/update tests for new functionality
- [ ] Ensure commit messages follow conventional format
- [ ] Rebase on latest main branch

### PR Description Template

```markdown
## Description
Brief description of changes and motivation.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] No new warnings or errors
```

### PR Size Guidelines

- **Small PRs**: < 200 lines changed (preferred)
- **Medium PRs**: 200-500 lines changed
- **Large PRs**: > 500 lines changed (requires justification)

Break large changes into smaller, logical commits.

## Code Review Guidelines

### For Authors

- Write clear PR descriptions
- Respond to feedback promptly
- Make requested changes in separate commits
- Squash commits before merging (if requested)

### For Reviewers

#### What to Look For

1. **Correctness**: Does the code do what it's supposed to do?
2. **Security**: Are there any security vulnerabilities?
3. **Performance**: Will this code perform well at scale?
4. **Maintainability**: Is the code readable and well-structured?
5. **Tests**: Are there adequate tests for the changes?

#### Review Checklist

- [ ] Code follows established patterns
- [ ] Error handling is appropriate
- [ ] Security best practices followed
- [ ] Performance considerations addressed
- [ ] Tests cover edge cases
- [ ] Documentation is updated
- [ ] No sensitive data in code

#### Feedback Guidelines

```markdown
# Good feedback examples

## Suggestion
Consider using a more descriptive variable name here:
```go
// Instead of 'r'
result, err := provider.Search(ctx, query)
```

## Question
Why did you choose to use a goroutine here instead of processing sequentially?

## Nitpick
Minor: This line exceeds 100 characters. Consider breaking it up.

## Critical
This could lead to a SQL injection vulnerability. Please use parameterized queries.
```

## Adding New Features

### Feature Development Process

1. **Create an Issue**: Describe the feature and its motivation
2. **Design Discussion**: Discuss implementation approach
3. **Create Feature Branch**: Follow naming conventions
4. **Implement with Tests**: Write code and comprehensive tests
5. **Update Documentation**: Add/update relevant documentation
6. **Submit PR**: Follow PR guidelines

### Adding New Providers

When adding a new search provider:

1. **Create provider directory:**
   ```bash
   mkdir internal/providers/newprovider
   ```

2. **Implement provider interface:**
   ```go
   // internal/providers/newprovider/provider.go
   type Provider struct {
       config providers.ProviderConfig
       client *http.Client
       logger *slog.Logger
   }
   
   func NewProvider(config providers.ProviderConfig, logger *slog.Logger) providers.SearchProvider {
       return &Provider{
           config: config,
           client: &http.Client{Timeout: config.Timeout},
           logger: logger,
       }
   }
   
   func (p *Provider) Search(ctx context.Context, query *providers.SearchQuery) (*providers.SearchResult, error) {
       // Implementation
   }
   ```

3. **Add configuration support:**
   ```go
   // Update internal/config/config.go
   type ProvidersConfig struct {
       NewProvider struct {
           Enabled bool   `mapstructure:"enabled"`
           APIKey  string `mapstructure:"api_key"`
           BaseURL string `mapstructure:"base_url"`
           Timeout string `mapstructure:"timeout"`
       } `mapstructure:"newprovider"`
   }
   ```

4. **Register in Wire:**
   ```go
   // Update cmd/server/wire.go
   func initializeProviders(manager providers.ProviderManager, logger *slog.Logger) {
       // ... existing providers
       
       newProviderConfig := providers.ProviderConfig{
           Enabled: true,
           BaseURL: "https://api.newprovider.com/v1",
           // ... other config
       }
       newProvider := newprovider.NewProvider(newProviderConfig, logger)
       manager.RegisterProvider("newprovider", newProvider)
   }
   ```

5. **Add tests:**
   ```go
   // test/unit/providers/newprovider_test.go
   func TestNewProvider_Search(t *testing.T) {
       // Test implementation
   }
   ```

### Adding New API Endpoints

1. **Define request/response types:**
   ```go
   // internal/services/types.go
   type NewFeatureRequest struct {
       Parameter string `json:"parameter" validate:"required"`
   }
   
   type NewFeatureResponse struct {
       Result string `json:"result"`
   }
   ```

2. **Implement service method:**
   ```go
   // internal/services/service.go
   func (s *Service) NewFeature(ctx context.Context, req *NewFeatureRequest) (*NewFeatureResponse, error) {
       // Implementation
   }
   ```

3. **Add HTTP handler:**
   ```go
   // internal/api/handlers/handler.go
   func (h *Handler) NewFeature(c *gin.Context) {
       var req NewFeatureRequest
       if err := c.ShouldBindJSON(&req); err != nil {
           c.JSON(400, gin.H{"error": err.Error()})
           return
       }
       
       result, err := h.service.NewFeature(c.Request.Context(), &req)
       if err != nil {
           c.JSON(500, gin.H{"error": err.Error()})
           return
       }
       
       c.JSON(200, result)
   }
   ```

4. **Register route:**
   ```go
   // internal/api/router.go
   func setupRoutes(r *gin.Engine, handlers *handlers.Container) {
       v1 := r.Group("/v1")
       v1.POST("/new-feature", handlers.NewFeature)
   }
   ```

## Documentation Standards

### Code Documentation

```go
// Package documentation
// Package providers implements search provider interfaces and management.
//
// This package provides a unified interface for searching across multiple
// academic paper databases including ArXiv, Semantic Scholar, Exa, and Tavily.
package providers

// Interface documentation
// SearchProvider defines the interface that all search providers must implement.
// It provides methods for searching papers, retrieving individual papers,
// and checking provider health status.
type SearchProvider interface {
    // Search performs a search query and returns matching papers.
    // The context can be used for cancellation and timeout control.
    Search(ctx context.Context, query *SearchQuery) (*SearchResult, error)
    
    // GetPaper retrieves a specific paper by its ID from the provider.
    GetPaper(ctx context.Context, id string) (*models.Paper, error)
}
```

### API Documentation

Update `docs/API_REFERENCE.md` when adding new endpoints:

```markdown
### New Feature Endpoint

Endpoint for new feature functionality.

```http
POST /v1/new-feature
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `parameter` | string | ✅ | Description of parameter |

#### Example Request
```bash
curl -X POST "http://localhost:8080/v1/new-feature" \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"parameter": "value"}'
```
```

## Community Guidelines

### Code of Conduct

We are committed to providing a welcoming and inclusive environment for all contributors. Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

### Communication

- **GitHub Issues**: Bug reports, feature requests, questions
- **GitHub Discussions**: Design discussions, help requests
- **Pull Requests**: Code review and discussion

### Getting Help

If you need help:

1. Check existing documentation
2. Search GitHub issues
3. Create a new issue with:
   - Clear description of the problem
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details

### Recognition

Contributors will be recognized in:
- GitHub contributors list
- Release notes for significant contributions
- Annual contributor appreciation

Thank you for contributing to SciFind! Your contributions help make scientific literature more accessible to everyone.