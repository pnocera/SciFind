# Testing Suite Documentation

This directory contains a comprehensive testing suite for the scifind-backend Go application, implementing modern testing best practices with testcontainers, mocks, and comprehensive coverage.

## 📁 Directory Structure

```
test/
├── README.md                    # This documentation
├── benchmarks/                  # Performance benchmark tests
│   └── repository_bench_test.go # Repository performance benchmarks
├── e2e/                        # End-to-end tests
│   └── health_test.go          # Health endpoint E2E tests
├── fixtures/                   # Test data fixtures
│   ├── authors.go              # Author test data
│   └── papers.go               # Paper test data
├── integration/                # Integration tests
│   └── repository_test.go      # Database integration tests
├── mocks/                      # Mock implementations
│   └── repository_mocks.go     # Repository mocks using testify
├── testutil/                   # Test utilities and helpers
│   ├── config.go               # Test configuration utilities
│   ├── database.go             # Database test setup with testcontainers
│   ├── http.go                 # HTTP testing utilities
│   └── messaging.go            # NATS messaging test setup
└── unit/                       # Unit tests
    └── models/                 # Model unit tests
        └── paper_test.go       # Paper model tests
```

## 🚀 Getting Started

### Prerequisites

- Go 1.24+
- Docker (for testcontainers)
- Make

### Installation

1. Install dependencies:
```bash
make deps
```

2. Install testing tools:
```bash
make install-tools
```

### Running Tests

#### All Tests
```bash
make test
```

#### Unit Tests Only (Fast)
```bash
make test-unit
# or
make test-short
```

#### Integration Tests
```bash
make test-integration
```

#### End-to-End Tests
```bash
make test-e2e
```

#### Benchmark Tests
```bash
make test-bench
```

#### With Coverage
```bash
make test-coverage
```

#### With Race Detection
```bash
make test-race
```

## 🏗️ Test Architecture

### Test Pyramid

Our testing follows the test pyramid pattern:

```
       /\
      /E2E\      <- Few, high-value integration tests
     /------\
    /Integr. \   <- Moderate integration tests  
   /----------\
  /   Unit     \ <- Many, fast, isolated unit tests
 /--------------\
```

### Test Categories

#### 1. Unit Tests (`test/unit/`)
- **Purpose**: Test individual components in isolation
- **Speed**: Very fast (< 100ms per test)
- **Scope**: Single functions, methods, and structs
- **Dependencies**: Mocked or stubbed
- **Examples**: Model validation, business logic, utility functions

#### 2. Integration Tests (`test/integration/`)
- **Purpose**: Test component interactions with real dependencies
- **Speed**: Medium (< 5s per test)
- **Scope**: Database operations, message queues, external APIs
- **Dependencies**: Real databases via testcontainers
- **Examples**: Repository operations, database transactions

#### 3. End-to-End Tests (`test/e2e/`)
- **Purpose**: Test complete user workflows
- **Speed**: Slow (5-30s per test)
- **Scope**: Full application stack
- **Dependencies**: Real services and databases
- **Examples**: API endpoints, health checks, full request cycles

#### 4. Benchmark Tests (`test/benchmarks/`)
- **Purpose**: Measure and track performance
- **Focus**: Database operations, critical paths, concurrency
- **Metrics**: Throughput, latency, memory usage

## 🛠️ Test Utilities

### Database Testing (`testutil/database.go`)

Provides utilities for database testing with both SQLite and PostgreSQL:

```go
// Setup in-memory SQLite for fast unit tests
dbUtil := testutil.SetupTestDatabase(t, false)
defer dbUtil.Cleanup()

// Setup PostgreSQL container for integration tests
dbUtil := testutil.SetupTestDatabase(t, true)
defer dbUtil.Cleanup()

// Use in tests
db := dbUtil.DB()
```

Features:
- Automatic migration
- Test data seeding
- Transaction testing
- Table truncation
- PostgreSQL and SQLite support

### NATS Testing (`testutil/messaging.go`)

Utilities for testing NATS messaging:

```go
natsUtil := testutil.SetupTestNATS(t)
defer natsUtil.Cleanup()

// Publish test messages
natsUtil.PublishTestMessage(t, "test.subject", []byte("test data"))

// Subscribe and wait for messages
msg := natsUtil.SubscribeAndWait(t, "test.subject", 5*time.Second)
```

### HTTP Testing (`testutil/http.go`)

HTTP testing utilities with Gin:

```go
httpUtil := testutil.SetupTestHTTPServer(t)
httpUtil.StartServer()
defer httpUtil.StopServer()

// Make requests
resp := httpUtil.MakeJSONRequest(t, "POST", "/api/papers", paperData)
httpUtil.AssertJSONResponse(t, resp, http.StatusCreated, &result)
```

### Test Configuration (`testutil/config.go`)

Configuration management for tests:

```go
// Basic test config
cfg := testutil.TestConfig(t)

// PostgreSQL test config
cfg := testutil.TestConfigWithPostgreSQL(t, connectionString)

// Config with mock providers
cfg := testutil.TestConfigWithProviders(t, mockArxivURL, mockSSURL)
```

## 📊 Fixtures and Test Data

### Paper Fixtures (`fixtures/papers.go`)

Provides comprehensive paper test data:

```go
paperFixtures := fixtures.NewPaperFixtures()

// Get different types of papers
basicPaper := paperFixtures.BasicPaper()
highQualityPaper := paperFixtures.HighQualityPaper()
unpublishedPaper := paperFixtures.UnpublishedPaper()
paperList := paperFixtures.PaperList()
```

### Author Fixtures (`fixtures/authors.go`)

Provides author test data:

```go
authorFixtures := fixtures.NewAuthorFixtures()

// Get different types of authors
basicAuthor := authorFixtures.BasicAuthor()
productiveAuthor := authorFixtures.ProductiveAuthor()
authorList := authorFixtures.AuthorList()
```

## 🎭 Mocking Strategy

We use [testify/mock](https://github.com/stretchr/testify) for mocking:

### Repository Mocks (`mocks/repository_mocks.go`)

```go
mockRepo := mocks.NewMockRepository()
mockPaperRepo := mockRepo.GetMockPaperRepo()

// Set expectations
mockPaperRepo.On("GetByID", ctx, "test-id").Return(expectedPaper, nil)

// Use in test
paper, err := mockPaperRepo.GetByID(ctx, "test-id")

// Verify expectations
mockPaperRepo.AssertExpectations(t)
```

### Mock HTTP Servers

For external API testing:

```go
// Create mock ArXiv server
mockServer := testutil.CreateMockArxivServer(t)
defer mockServer.Close()

// Use mock server URL in configuration
cfg.Providers.ArXiv.BaseURL = mockServer.URL
```

## 🎯 Testing Best Practices

### 1. Test Naming
```go
func TestPaper_IsPublished(t *testing.T) {
    t.Run("published paper", func(t *testing.T) {
        // Test published paper
    })
    
    t.Run("unpublished paper", func(t *testing.T) {
        // Test unpublished paper
    })
}
```

### 2. Table-Driven Tests
```go
func TestPaper_Validation(t *testing.T) {
    tests := []struct {
        name    string
        paper   *models.Paper
        wantErr bool
    }{
        {"valid paper", validPaper, false},
        {"missing title", paperWithoutTitle, true},
        {"invalid DOI", paperWithInvalidDOI, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePaper(tt.paper)
            if (err != nil) != tt.wantErr {
                t.Errorf("validatePaper() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 3. Test Cleanup
```go
func TestRepository(t *testing.T) {
    dbUtil := testutil.SetupTestDatabase(t, false)
    defer dbUtil.Cleanup() // Always cleanup
    
    t.Run("test case", func(t *testing.T) {
        dbUtil.TruncateAllTables(t) // Clean state for each test
        // Test logic
    })
}
```

### 4. Context and Timeouts
```go
func TestSlowOperation(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    result, err := slowOperation(ctx)
    require.NoError(t, err)
}
```

### 5. Parallel Tests
```go
func TestConcurrentOperations(t *testing.T) {
    t.Parallel() // Run in parallel with other tests
    
    // Test logic
}
```

## ⚡ Performance Testing

### Benchmark Tests

Located in `test/benchmarks/`, these measure:
- Database operation performance
- Memory usage
- Concurrent access patterns
- Batch operation efficiency

Run benchmarks:
```bash
# Run all benchmarks
make test-bench

# Run specific benchmark
go test -bench=BenchmarkPaperRepository_Create ./test/benchmarks/

# With memory profiling
go test -bench=. -memprofile=mem.prof ./test/benchmarks/
```

### Performance Criteria
- Database operations: < 10ms for simple queries
- Batch operations: > 1000 ops/second
- Memory usage: Stable, no leaks
- Concurrent access: Linear scaling up to 100 goroutines

## 🔧 Configuration

### Test Configuration

Tests use configuration from `testutil/config.go`:
- SQLite in-memory database for unit tests
- PostgreSQL containers for integration tests
- Disabled external providers
- Reduced logging for performance
- Permissive security settings

### Environment Variables

Override test configuration with environment variables:
```bash
export SCIFIND_DATABASE_TYPE=postgres
export SCIFIND_DATABASE_POSTGRESQL_DSN="postgres://user:pass@localhost/testdb"
export SCIFIND_NATS_URL="nats://localhost:4222"
```

## 🚀 CI/CD Integration

### GitHub Actions

The `.github/workflows/test.yml` workflow runs:
- Unit tests with race detection
- Integration tests with PostgreSQL and NATS
- E2E tests with Docker
- Benchmark tests
- Coverage reporting
- Security scanning

### Test Commands in CI
```bash
# Unit tests (fast)
make test-short

# Integration tests
make test-integration

# Full test suite
make ci

# With coverage
make test-coverage
```

## 📈 Metrics and Reporting

### Coverage Reports
```bash
# Generate coverage report
make test-coverage

# View in browser
open coverage.html
```

### Benchmark Results
```bash
# Run benchmarks with output
make test-bench > benchmark_results.txt
```

### Test Metrics
- **Coverage Target**: >80% statement coverage
- **Test Speed**: Unit tests <1s, Integration <30s
- **Reliability**: Tests should be deterministic and stable

## 🐛 Debugging Tests

### Verbose Output
```bash
make test-verbose
```

### Run Single Test
```bash
go test -v -run TestPaper_IsPublished ./test/unit/models/
```

### Debug with Delve
```bash
dlv test ./test/unit/models/ -- -test.run TestPaper_IsPublished
```

### Test with Containers
```bash
# Force container usage
go test -tags=containers ./test/integration/
```

## 📚 Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Testcontainers Go](https://golang.testcontainers.org/)
- [GORM Testing](https://gorm.io/docs/testing.html)
- [Gin Testing](https://gin-gonic.com/docs/testing/)

## 🤝 Contributing

When adding new tests:

1. Follow the existing directory structure
2. Use appropriate test categories (unit/integration/e2e)
3. Include both positive and negative test cases
4. Use fixtures for test data
5. Mock external dependencies
6. Add benchmark tests for performance-critical code
7. Update this documentation if needed

## 📋 Checklist for New Features

- [ ] Unit tests for business logic
- [ ] Integration tests for database operations
- [ ] Mock external dependencies
- [ ] Add benchmark tests if performance-critical
- [ ] Update fixtures if new models added
- [ ] Test error handling and edge cases
- [ ] Verify tests pass in CI/CD
- [ ] Check coverage meets requirements