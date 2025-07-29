# SciFind Backend Testing Guide

Comprehensive testing strategy, tools, and best practices for the SciFind backend project.

## Table of Contents
- [Testing Philosophy](#testing-philosophy)
- [Test Structure](#test-structure)
- [Running Tests](#running-tests)
- [Unit Testing](#unit-testing)
- [Integration Testing](#integration-testing)
- [End-to-End Testing](#end-to-end-testing)
- [Performance Testing](#performance-testing)
- [Test Utilities](#test-utilities)
- [Mocking Strategy](#mocking-strategy)
- [Test Data Management](#test-data-management)
- [Continuous Integration](#continuous-integration)
- [Testing Best Practices](#testing-best-practices)

## Testing Philosophy

SciFind follows a comprehensive testing strategy based on the **Test Pyramid**:

```mermaid
graph TD
    subgraph "Test Pyramid"
        E2E["End-to-End Tests<br/>Few, Slow, Expensive<br/>Full System Integration"]
        INTEGRATION["Integration Tests<br/>Some, Medium Speed<br/>Component Integration"]
        UNIT["Unit Tests<br/>Many, Fast, Cheap<br/>Individual Functions"]
    end
    
    E2E --> INTEGRATION
    INTEGRATION --> UNIT
    
    style E2E fill:#ff9999
    style INTEGRATION fill:#ffcc99
    style UNIT fill:#99ff99
```

### Testing Principles

1. **Fast Feedback**: Unit tests run in milliseconds
2. **Test Isolation**: Each test is independent and can run alone
3. **Reliable**: Tests are deterministic and don't flake
4. **Maintainable**: Tests are easy to understand and modify
5. **Comprehensive**: Critical paths have multiple test levels

## Test Structure

### Directory Organization

```
test/
├── unit/                    # Unit tests (fast, isolated)
│   ├── services/           # Service layer tests
│   ├── providers/          # Provider tests
│   ├── repository/         # Repository tests
│   └── models/             # Model validation tests
├── integration/            # Integration tests (medium speed)
│   ├── api/               # API endpoint tests
│   ├── database/          # Database integration tests
│   └── messaging/         # NATS messaging tests
├── e2e/                   # End-to-end tests (slow, comprehensive)
│   ├── search/            # Search workflow tests
│   ├── health/            # Health check tests
│   └── scenarios/         # Complex user scenarios
├── benchmarks/            # Performance benchmarks
├── load/                  # Load testing scripts
├── fixtures/              # Test data fixtures
├── mocks/                 # Mock implementations
└── testutil/              # Testing utilities
    ├── database.go        # Database test helpers
    ├── http.go            # HTTP test helpers
    ├── messaging.go       # Messaging test helpers
    └── config.go          # Configuration helpers
```

### Test File Naming

```bash
# Unit tests
service_test.go           # Tests for service.go
provider_test.go          # Tests for provider.go

# Integration tests  
api_integration_test.go   # API integration tests
db_integration_test.go    # Database integration tests

# End-to-end tests
search_e2e_test.go        # Search workflow E2E tests
health_e2e_test.go        # Health check E2E tests

# Benchmarks
service_bench_test.go     # Performance benchmarks
```

## Running Tests

### Development Commands

```bash
# Run all tests
make test

# Run specific test categories
make test-unit           # Unit tests only
make test-integration    # Integration tests only 
make test-e2e           # End-to-end tests only

# Run tests with coverage
make test-coverage

# Run tests in watch mode (auto-rerun on changes)
make test-watch

# Run benchmarks
make test-bench

# Run load tests
make test-load
```

### Go Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/services/...

# Run specific test
go test -run TestSearchService_Search ./internal/services/

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./test/benchmarks/

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Test Environment Variables

```bash
# Test configuration
export SCIFIND_TEST_MODE=true
export SCIFIND_DATABASE_TYPE=sqlite
export SCIFIND_DATABASE_SQLITE_PATH=":memory:"
export SCIFIND_LOG_LEVEL=error

# Integration test settings
export SCIFIND_TEST_POSTGRES_URL="postgres://test:test@localhost:5432/scifind_test"
export SCIFIND_TEST_NATS_URL="nats://localhost:4222"

# Provider test keys (optional)
export SCIFIND_TEST_SEMANTIC_SCHOLAR_KEY="test-key"
export SCIFIND_TEST_EXA_KEY="test-key"
```

## Unit Testing

Unit tests focus on individual functions and methods in isolation.

### Service Layer Testing

```go
// test/unit/services/search_service_test.go
package services_test

import (
    "context"
    "testing"
    "time"

    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "scifind-backend/internal/services"
    "scifind-backend/test/mocks"
    "scifind-backend/test/testutil"
)

func TestSearchService_Search(t *testing.T) {
    tests := []struct {
        name           string
        request        *services.SearchRequest
        mockSetup      func(*mocks.MockSearchRepository, *mocks.MockProviderManager)
        expectedResult *services.SearchResponse
        expectedError  string
    }{
        {
            name: "successful search",
            request: &services.SearchRequest{
                Query: "machine learning",
                Limit: 10,
            },
            mockSetup: func(repo *mocks.MockSearchRepository, provider *mocks.MockProviderManager) {
                provider.EXPECT().
                    SearchAll(gomock.Any(), gomock.Any()).
                    Return(&providers.AggregatedResult{
                        Papers: []*models.Paper{
                            {Title: "ML Paper 1", Abstract: "Abstract 1"},
                            {Title: "ML Paper 2", Abstract: "Abstract 2"},
                        },
                        TotalCount: 2,
                    }, nil)
                
                repo.EXPECT().
                    SaveSearchHistory(gomock.Any(), gomock.Any()).
                    Return(nil)
            },
            expectedResult: &services.SearchResponse{
                Papers: []*models.Paper{
                    {Title: "ML Paper 1", Abstract: "Abstract 1"},
                    {Title: "ML Paper 2", Abstract: "Abstract 2"},
                },
                TotalCount: 2,
            },
        },
        {
            name: "empty query error",
            request: &services.SearchRequest{
                Query: "",
                Limit: 10,
            },
            mockSetup:     func(*mocks.MockSearchRepository, *mocks.MockProviderManager) {},
            expectedError: "search query cannot be empty",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            mockRepo := mocks.NewMockSearchRepository(ctrl)
            mockProvider := mocks.NewMockProviderManager(ctrl)
            logger := testutil.CreateTestLogger(t)

            tt.mockSetup(mockRepo, mockProvider)

            service := services.NewSearchService(
                mockRepo,
                nil, // paper repo not needed for this test
                nil, // messaging not needed
                mockProvider,
                logger,
            )

            // Act
            result, err := service.Search(context.Background(), tt.request)

            // Assert
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.Equal(t, tt.expectedResult.TotalCount, result.TotalCount)
                assert.Len(t, result.Papers, len(tt.expectedResult.Papers))
            }
        })
    }
}

func TestSearchService_ValidateQuery(t *testing.T) {
    service := createTestSearchService(t)

    tests := []struct {
        name          string
        query         string
        expectedError string
    }{
        {"valid query", "machine learning", ""},
        {"empty query", "", "search query cannot be empty"},
        {"too long query", strings.Repeat("a", 501), "query too long"},
        {"malicious query", "'; DROP TABLE papers; --", "invalid characters in query"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := service.ValidateQuery(tt.query)
            
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Provider Testing

```go
// test/unit/providers/arxiv_test.go
func TestArxivProvider_Search(t *testing.T) {
    // Mock HTTP client
    mockClient := &testutil.MockHTTPClient{}
    
    provider := &arxiv.Provider{
        client: mockClient,
        logger: testutil.CreateTestLogger(t),
        config: providers.ProviderConfig{
            BaseURL: "https://export.arxiv.org/api/query",
            Timeout: 30 * time.Second,
        },
    }

    tests := []struct {
        name           string
        query          *providers.SearchQuery
        mockResponse   string
        mockStatusCode int
        expectedCount  int
        expectedError  string
    }{
        {
            name: "successful search",
            query: &providers.SearchQuery{
                Query: "machine learning",
                Limit: 5,
            },
            mockResponse: `<?xml version="1.0" encoding="UTF-8"?>
            <feed xmlns="http://www.w3.org/2005/Atom">
                <entry>
                    <title>Test Paper</title>
                    <summary>Test abstract</summary>
                </entry>
            </feed>`,
            mockStatusCode: 200,
            expectedCount:  1,
        },
        {
            name: "API error",
            query: &providers.SearchQuery{
                Query: "test",
                Limit: 5,
            },
            mockStatusCode: 500,
            expectedError:  "API request failed",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mock response
            mockClient.SetResponse(tt.mockResponse, tt.mockStatusCode)

            // Act
            result, err := provider.Search(context.Background(), tt.query)

            // Assert
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Len(t, result.Papers, tt.expectedCount)
            }
        })
    }
}
```

### Model Validation Testing

```go
// test/unit/models/paper_test.go
func TestPaper_Validate(t *testing.T) {
    tests := []struct {
        name          string
        paper         *models.Paper
        expectedValid bool
        expectedError string
    }{
        {
            name: "valid paper",
            paper: &models.Paper{
                Title:     "Valid Paper Title",
                Abstract:  "Valid abstract content",
                Authors:   []string{"Author One", "Author Two"},
                Provider:  "arxiv",
                SourceID:  "2023.12345",
            },
            expectedValid: true,
        },
        {
            name: "missing title",
            paper: &models.Paper{
                Abstract: "Valid abstract",
                Provider: "arxiv",
            },
            expectedValid: false,
            expectedError: "title is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.paper.Validate()
            
            if tt.expectedValid {
                assert.NoError(t, err)
            } else {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            }
        })
    }
}
```

## Integration Testing

Integration tests verify that components work together correctly.

### API Integration Testing

```go
// test/integration/api/search_test.go
package api_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"

    "scifind-backend/test/testutil"
)

type SearchAPITestSuite struct {
    suite.Suite
    app    *testutil.TestApplication
    server *httptest.Server
}

func (s *SearchAPITestSuite) SetupSuite() {
    s.app = testutil.CreateTestApplication(s.T())
    s.server = httptest.NewServer(s.app.Router)
}

func (s *SearchAPITestSuite) TearDownSuite() {
    s.server.Close()
    s.app.Cleanup()
}

func (s *SearchAPITestSuite) TestSearchEndpoint() {
    tests := []struct {
        name           string
        request        map[string]interface{}
        expectedStatus int
        expectedCount  int
    }{
        {
            name: "valid search request",
            request: map[string]interface{}{
                "query": "machine learning",
                "limit": 10,
            },
            expectedStatus: 200,
            expectedCount:  10,
        },
        {
            name: "invalid request - empty query",
            request: map[string]interface{}{
                "query": "",
                "limit": 10,
            },
            expectedStatus: 400,
        },
    }

    for _, tt := range tests {
        s.Run(tt.name, func() {
            // Arrange
            requestBody, _ := json.Marshal(tt.request)
            req := httptest.NewRequest("POST", "/v1/search", bytes.NewReader(requestBody))
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("Authorization", "Bearer test-api-key")

            // Act
            resp := httptest.NewRecorder()
            s.app.Router.ServeHTTP(resp, req)

            // Assert
            assert.Equal(s.T(), tt.expectedStatus, resp.Code)

            if tt.expectedStatus == 200 {
                var response map[string]interface{}
                err := json.Unmarshal(resp.Body.Bytes(), &response)
                assert.NoError(s.T(), err)
                
                papers, ok := response["papers"].([]interface{})
                assert.True(s.T(), ok)
                assert.LessOrEqual(s.T(), len(papers), tt.expectedCount)
            }
        })
    }
}

func TestSearchAPITestSuite(t *testing.T) {
    suite.Run(t, new(SearchAPITestSuite))
}
```

### Database Integration Testing

```go
// test/integration/database/repository_test.go
package database_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"

    "scifind-backend/internal/repository"
    "scifind-backend/test/testutil"
)

type RepositoryTestSuite struct {
    suite.Suite
    pgContainer *postgres.PostgresContainer
    db          *repository.Database
    repos       *repository.Container
}

func (s *RepositoryTestSuite) SetupSuite() {
    ctx := context.Background()
    
    // Start PostgreSQL container
    var err error
    s.pgContainer, err = postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("scifind_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        postgres.WithSQLDriver("pgx"),
    )
    s.Require().NoError(err)

    // Connect to database
    dsn, err := s.pgContainer.ConnectionString(ctx, "sslmode=disable")
    s.Require().NoError(err)

    s.db, err = testutil.CreateTestDatabaseWithDSN(s.T(), dsn)
    s.Require().NoError(err)

    s.repos = repository.NewContainer(s.db.DB, testutil.CreateTestLogger(s.T()))
}

func (s *RepositoryTestSuite) TearDownSuite() {
    if s.pgContainer != nil {
        s.pgContainer.Terminate(context.Background())
    }
}

func (s *RepositoryTestSuite) SetupTest() {
    // Clean database before each test
    s.db.DB.Exec("TRUNCATE papers, authors, search_history CASCADE")
}

func (s *RepositoryTestSuite) TestPaperRepository_CreateAndGet() {
    // Arrange
    paper := &models.Paper{
        Title:     "Test Paper",
        Abstract:  "Test abstract",
        Provider:  "arxiv",
        SourceID:  "2023.12345",
        Authors:   []string{"Author One"},
        CreatedAt: time.Now(),
    }

    // Act - Create
    err := s.repos.Paper.Create(context.Background(), paper)
    s.Assert().NoError(err)
    s.Assert().NotEmpty(paper.ID)

    // Act - Get
    retrieved, err := s.repos.Paper.GetBySourceID(context.Background(), "arxiv", "2023.12345")
    s.Assert().NoError(err)
    s.Assert().NotNil(retrieved)

    // Assert
    s.Assert().Equal(paper.Title, retrieved.Title)
    s.Assert().Equal(paper.Abstract, retrieved.Abstract)
    s.Assert().Equal(paper.Provider, retrieved.Provider)
    s.Assert().Equal(paper.SourceID, retrieved.SourceID)
}

func (s *RepositoryTestSuite) TestSearchRepository_SaveAndGetHistory() {
    // Arrange
    searchQuery := &models.SearchHistory{
        Query:     "machine learning",
        Provider:  "arxiv",
        UserID:    "test-user",
        Results:   10,
        Duration:  100 * time.Millisecond,
        CreatedAt: time.Now(),
    }

    // Act
    err := s.repos.Search.SaveSearchHistory(context.Background(), searchQuery)
    s.Assert().NoError(err)

    // Get recent searches
    history, err := s.repos.Search.GetRecentSearches(context.Background(), "test-user", 10)
    s.Assert().NoError(err)
    s.Assert().Len(history, 1)
    s.Assert().Equal("machine learning", history[0].Query)
}

func TestRepositoryTestSuite(t *testing.T) {
    suite.Run(t, new(RepositoryTestSuite))
}
```

## End-to-End Testing

E2E tests verify complete user workflows and system behavior.

### Search Workflow E2E Testing

```go
// test/e2e/search_e2e_test.go
package e2e_test

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"

    "scifind-backend/test/testutil"
)

type SearchE2ETestSuite struct {
    suite.Suite
    app     *testutil.TestApplication
    baseURL string
    apiKey  string
}

func (s *SearchE2ETestSuite) SetupSuite() {
    s.app = testutil.CreateFullTestApplication(s.T())
    s.baseURL = fmt.Sprintf("http://localhost:%d", s.app.Port)
    s.apiKey = "test-api-key"
    
    // Wait for application to be ready
    s.waitForHealth()
}

func (s *SearchE2ETestSuite) TearDownSuite() {
    s.app.Cleanup()
}

func (s *SearchE2ETestSuite) waitForHealth() {
    client := &http.Client{Timeout: 5 * time.Second}
    
    for i := 0; i < 30; i++ {
        resp, err := client.Get(s.baseURL + "/health")
        if err == nil && resp.StatusCode == 200 {
            resp.Body.Close()
            return
        }
        if resp != nil {
            resp.Body.Close()
        }
        time.Sleep(1 * time.Second)
    }
    s.T().Fatal("Application failed to become healthy")
}

func (s *SearchE2ETestSuite) TestCompleteSearchWorkflow() {
    client := &http.Client{Timeout: 30 * time.Second}

    // Step 1: Perform search
    searchRequest := map[string]interface{}{
        "query": "machine learning",
        "limit": 5,
        "providers": []string{"arxiv"},
    }

    requestBody, _ := json.Marshal(searchRequest)
    req, _ := http.NewRequest("POST", s.baseURL+"/v1/search", bytes.NewReader(requestBody))
    req.Header.Set("Authorization", "Bearer "+s.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.Do(req)
    s.Assert().NoError(err)
    s.Assert().Equal(200, resp.StatusCode)

    var searchResponse map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&searchResponse)
    s.Assert().NoError(err)
    resp.Body.Close()

    // Verify search response
    papers, ok := searchResponse["papers"].([]interface{})
    s.Assert().True(ok)
    s.Assert().GreaterOrEqual(len(papers), 1)

    // Step 2: Get specific paper details
    if len(papers) > 0 {
        paper := papers[0].(map[string]interface{})
        paperID := paper["source_id"].(string)
        provider := paper["provider"].(string)

        req, _ = http.NewRequest("GET", 
            fmt.Sprintf("%s/v1/papers/%s/%s", s.baseURL, provider, paperID), 
            nil)
        req.Header.Set("Authorization", "Bearer "+s.apiKey)

        resp, err = client.Do(req)
        s.Assert().NoError(err)
        s.Assert().Equal(200, resp.StatusCode)

        var paperResponse map[string]interface{}
        err = json.NewDecoder(resp.Body).Decode(&paperResponse)
        s.Assert().NoError(err)
        resp.Body.Close()

        // Verify paper details
        paperData := paperResponse["paper"].(map[string]interface{})
        s.Assert().Equal(paperID, paperData["source_id"])
        s.Assert().Equal(provider, paperData["provider"])
    }

    // Step 3: Check provider status
    req, _ = http.NewRequest("GET", s.baseURL+"/v1/providers", nil)
    req.Header.Set("Authorization", "Bearer "+s.apiKey)

    resp, err = client.Do(req)
    s.Assert().NoError(err)
    s.Assert().Equal(200, resp.StatusCode)

    var providersResponse map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&providersResponse)
    s.Assert().NoError(err)
    resp.Body.Close()

    // Verify providers are available
    providers := providersResponse["providers"].(map[string]interface{})
    s.Assert().Contains(providers, "arxiv")
}

func (s *SearchE2ETestSuite) TestErrorHandling() {
    client := &http.Client{Timeout: 30 * time.Second}

    // Test invalid API key
    req, _ := http.NewRequest("POST", s.baseURL+"/v1/search", 
        bytes.NewReader([]byte(`{"query":"test","limit":5}`)))
    req.Header.Set("Authorization", "Bearer invalid-key")
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.Do(req)
    s.Assert().NoError(err)
    s.Assert().Equal(401, resp.StatusCode)
    resp.Body.Close()

    // Test malformed request
    req, _ = http.NewRequest("POST", s.baseURL+"/v1/search", 
        bytes.NewReader([]byte(`{invalid json}`)))
    req.Header.Set("Authorization", "Bearer "+s.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err = client.Do(req)
    s.Assert().NoError(err)
    s.Assert().Equal(400, resp.StatusCode)
    resp.Body.Close()
}

func TestSearchE2ETestSuite(t *testing.T) {
    suite.Run(t, new(SearchE2ETestSuite))
}
```

## Performance Testing

### Benchmarks

```go
// test/benchmarks/search_bench_test.go
func BenchmarkSearchService_Search(b *testing.B) {
    service := createBenchmarkSearchService(b)
    request := &services.SearchRequest{
        Query: "machine learning",
        Limit: 10,
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := service.Search(context.Background(), request)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

func BenchmarkProviderManager_SearchAll(b *testing.B) {
    manager := createBenchmarkProviderManager(b)
    query := &providers.SearchQuery{
        Query: "quantum computing",
        Limit: 20,
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := manager.SearchAll(context.Background(), query)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Load Testing

```go
// test/load/search_load_test.go
func TestSearchEndpointLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }

    app := testutil.CreateTestApplication(t)
    defer app.Cleanup()

    // Configuration
    concurrency := 50
    duration := 30 * time.Second
    requests := make(chan struct{}, concurrency)
    results := make(chan result, concurrency*100)

    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            client := &http.Client{Timeout: 10 * time.Second}
            
            ctx, cancel := context.WithTimeout(context.Background(), duration)
            defer cancel()

            for {
                select {
                case <-ctx.Done():
                    return
                case <-requests:
                    start := time.Now()
                    resp, err := client.Post(
                        app.BaseURL+"/v1/search",
                        "application/json",
                        strings.NewReader(`{"query":"test","limit":5}`),
                    )
                    
                    res := result{
                        duration: time.Since(start),
                        error:    err,
                    }
                    
                    if resp != nil {
                        res.statusCode = resp.StatusCode
                        resp.Body.Close()
                    }
                    
                    results <- res
                }
            }
        }()
    }

    // Send requests
    go func() {
        ticker := time.NewTicker(10 * time.Millisecond)
        defer ticker.Stop()
        
        ctx, cancel := context.WithTimeout(context.Background(), duration)
        defer cancel()
        
        for {
            select {
            case <-ctx.Done():
                close(requests)
                return
            case <-ticker.C:
                select {
                case requests <- struct{}{}:
                case <-ctx.Done():
                    close(requests)
                    return
                }
            }
        }
    }()

    // Wait for completion
    wg.Wait()
    close(results)

    // Analyze results
    var totalRequests int
    var successfulRequests int
    var totalDuration time.Duration
    var maxDuration time.Duration

    for res := range results {
        totalRequests++
        totalDuration += res.duration
        
        if res.duration > maxDuration {
            maxDuration = res.duration
        }
        
        if res.error == nil && res.statusCode == 200 {
            successfulRequests++
        }
    }

    avgDuration := totalDuration / time.Duration(totalRequests)
    successRate := float64(successfulRequests) / float64(totalRequests) * 100

    t.Logf("Load test results:")
    t.Logf("Total requests: %d", totalRequests)
    t.Logf("Successful requests: %d", successfulRequests)
    t.Logf("Success rate: %.2f%%", successRate)
    t.Logf("Average duration: %v", avgDuration)
    t.Logf("Max duration: %v", maxDuration)
    t.Logf("RPS: %.2f", float64(totalRequests)/duration.Seconds())

    // Assertions
    assert.GreaterOrEqual(t, successRate, 95.0, "Success rate should be at least 95%")
    assert.LessOrEqual(t, avgDuration, 500*time.Millisecond, "Average response time should be under 500ms")
}

type result struct {
    duration   time.Duration
    statusCode int
    error      error
}
```

## Test Utilities

### Database Utilities

```go
// test/testutil/database.go
package testutil

import (
    "testing"

    "scifind-backend/internal/config"
    "scifind-backend/internal/repository"
)

func CreateTestDatabase(t *testing.T) *repository.Database {
    cfg := &config.Config{
        Database: config.DatabaseConfig{
            Type: "sqlite",
            SQLite: config.SQLiteConfig{
                Path:        ":memory:",
                AutoMigrate: true,
            },
        },
    }

    db, err := repository.NewDatabase(cfg, CreateTestLogger(t))
    if err != nil {
        t.Fatalf("Failed to create test database: %v", err)
    }

    return db
}

func CreateTestDatabaseWithDSN(t *testing.T, dsn string) (*repository.Database, error) {
    cfg := &config.Config{
        Database: config.DatabaseConfig{
            Type: "postgres",
            PostgreSQL: config.PostgreSQLConfig{
                DSN:         dsn,
                AutoMigrate: true,
            },
        },
    }

    return repository.NewDatabase(cfg, CreateTestLogger(t))
}

func SeedTestData(t *testing.T, db *repository.Database) {
    // Insert test papers
    papers := []*models.Paper{
        {
            Title:    "Test Paper 1",
            Abstract: "Abstract for test paper 1",
            Provider: "arxiv",
            SourceID: "2023.00001",
            Authors:  []string{"Author One", "Author Two"},
        },
        {
            Title:    "Test Paper 2", 
            Abstract: "Abstract for test paper 2",
            Provider: "semantic_scholar",
            SourceID: "ss-12345",
            Authors:  []string{"Author Three"},
        },
    }

    for _, paper := range papers {
        if err := db.DB.Create(paper).Error; err != nil {
            t.Fatalf("Failed to seed test data: %v", err)
        }
    }
}
```

### HTTP Utilities

```go
// test/testutil/http.go
package testutil

import (
    "io"
    "net/http"
    "strings"
)

type MockHTTPClient struct {
    response   string
    statusCode int
    err        error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    if m.err != nil {
        return nil, m.err
    }

    return &http.Response{
        StatusCode: m.statusCode,
        Body:       io.NopCloser(strings.NewReader(m.response)),
        Header:     make(http.Header),
    }, nil
}

func (m *MockHTTPClient) SetResponse(response string, statusCode int) {
    m.response = response
    m.statusCode = statusCode
    m.err = nil
}

func (m *MockHTTPClient) SetError(err error) {
    m.err = err
}
```

## Mocking Strategy

### Mock Generation

```bash
# Generate mocks using mockgen
go install github.com/golang/mock/mockgen@latest

# Generate repository mocks
mockgen -source=internal/repository/interfaces.go -destination=test/mocks/repository_mocks.go

# Generate service mocks
mockgen -source=internal/services/interfaces.go -destination=test/mocks/service_mocks.go

# Generate provider mocks
mockgen -source=internal/providers/interfaces.go -destination=test/mocks/provider_mocks.go
```

### Mock Usage Examples

```go
// Using gomock in tests
func TestServiceWithMocks(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMockPaperRepository(ctrl)
    mockProvider := mocks.NewMockSearchProvider(ctrl)

    // Set expectations
    mockRepo.EXPECT().
        GetBySourceID(gomock.Any(), "arxiv", "2023.12345").
        Return(&models.Paper{Title: "Test Paper"}, nil).
        Times(1)

    // Use mocks in service
    service := NewService(mockRepo, mockProvider)
    result, err := service.GetPaper(context.Background(), "arxiv", "2023.12345")

    assert.NoError(t, err)
    assert.Equal(t, "Test Paper", result.Title)
}
```

## Test Data Management

### Fixtures

```go
// test/fixtures/papers.go
package fixtures

import "scifind-backend/internal/models"

func CreateTestPaper() *models.Paper {
    return &models.Paper{
        Title:     "Machine Learning in Scientific Research",
        Abstract:  "This paper explores the application of machine learning...",
        Authors:   []string{"Dr. Jane Smith", "Prof. John Doe"},
        Provider:  "arxiv",
        SourceID:  "2023.12345",
        URL:       "https://arxiv.org/abs/2023.12345",
        PDFURL:    "https://arxiv.org/pdf/2023.12345.pdf",
        Categories: []string{"cs.LG", "cs.AI"},
        Keywords:  []string{"machine learning", "artificial intelligence"},
    }
}

func CreateTestPapers(count int) []*models.Paper {
    papers := make([]*models.Paper, count)
    for i := 0; i < count; i++ {
        paper := CreateTestPaper()
        paper.SourceID = fmt.Sprintf("2023.%05d", i+1)
        paper.Title = fmt.Sprintf("Test Paper %d", i+1)
        papers[i] = paper
    }
    return papers
}
```

## Continuous Integration

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.24'
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run unit tests
      run: make test-unit
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: scifind_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      nats:
        image: nats:2.10-alpine
        ports:
          - 4222:4222

    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.24'
    
    - name: Run integration tests
      env:
        SCIFIND_TEST_POSTGRES_URL: postgres://postgres:postgres@localhost:5432/scifind_test?sslmode=disable
        SCIFIND_TEST_NATS_URL: nats://localhost:4222
      run: make test-integration

  e2e-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.24'
    
    - name: Start test environment
      run: make docker-up
    
    - name: Run E2E tests
      run: make test-e2e
    
    - name: Stop test environment
      run: make docker-down
```

## Testing Best Practices

### General Guidelines

1. **Test Independence**: Each test should be able to run in isolation
2. **Clear Test Names**: Use descriptive names that explain what is being tested
3. **AAA Pattern**: Arrange, Act, Assert - structure tests clearly
4. **Fast Tests**: Unit tests should run in milliseconds
5. **Deterministic**: Tests should produce consistent results

### Test Data

```go
// Good: Use builders or factories for test data
func createTestSearchRequest() *services.SearchRequest {
    return &services.SearchRequest{
        Query:  "machine learning",
        Limit:  10,
        Offset: 0,
    }
}

// Good: Use table-driven tests for multiple scenarios
func TestValidateQuery(t *testing.T) {
    tests := []struct {
        name          string
        query         string
        expectedError string
    }{
        {"valid query", "machine learning", ""},
        {"empty query", "", "query cannot be empty"},
        {"too long", strings.Repeat("a", 501), "query too long"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateQuery(tt.query)
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Error Testing

```go
// Good: Test both success and failure cases
func TestSearchService_Search(t *testing.T) {
    t.Run("successful search", func(t *testing.T) {
        // Test successful case
    })
    
    t.Run("provider error", func(t *testing.T) {
        // Test error handling
    })
    
    t.Run("invalid input", func(t *testing.T) {
        // Test input validation
    })
}
```

### Performance Testing

```go
// Good: Use benchmarks for performance-critical code
func BenchmarkSearchProvider_Search(b *testing.B) {
    provider := createTestProvider()
    query := createTestQuery()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := provider.Search(context.Background(), query)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Debugging Tests

### Common Issues

1. **Flaky Tests**: Tests that pass/fail inconsistently
   - Usually caused by timing issues or external dependencies
   - Use proper synchronization and mocking

2. **Slow Tests**: Tests that take too long to run
   - Often caused by database operations or network calls
   - Use in-memory databases and mocks

3. **Test Pollution**: Tests affecting each other
   - Ensure proper cleanup between tests
   - Use fresh test data for each test

### Debugging Commands

```bash
# Run specific test with verbose output
go test -v -run TestSpecificFunction ./internal/services/

# Run tests with race detection
go test -race ./...

# Run tests with coverage and open in browser
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Debug failing test
go test -v -run TestFailingTest ./... -count=1
```

This comprehensive testing guide ensures that SciFind backend maintains high quality, reliability, and performance through thorough testing at all levels.