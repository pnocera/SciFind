# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

**Build and Run:**
- `make build` - Build the application binary
- `make run` - Build and run the application
- `make dev` - Run in development mode with hot reload using ./configs/config.dev.yaml
- `go run ./cmd/server` - Direct Go run command

**Testing:**
- `make test` - Run all tests
- `make test-watch` - Run tests in watch mode
- `go test ./...` - Direct Go test command

**Code Quality:**
- `make fmt` - Format Go code with `go fmt`
- `make lint` - Run golangci-lint (requires golangci-lint installed)
- `make check` - Run both format and lint

**Docker Development:**
- `make docker-up` - Start full development environment with PostgreSQL, NATS, Prometheus, Grafana
- `make docker-down` - Stop development environment
- `make docker-logs` - View logs from all services

**Database:**
- `make migrate-up` - Run database migrations
- `make migrate-down` - Rollback database migrations
- `make migrate-create NAME=migration_name` - Create new migration

## Architecture Overview

**SciFIND Backend** is a high-performance scientific literature search engine that aggregates results from multiple academic databases (ArXiv, Semantic Scholar, Exa, Tavily) into a unified API.

### Core Components

**Dependency Injection:** Uses Google Wire for dependency injection. Main wiring is in `cmd/server/wire.go` with generated code in `wire_gen.go`. Run `go generate ./cmd/server` to regenerate wire dependencies.

**Configuration:** Viper-based configuration system in `internal/config/`. Uses YAML files (config.yaml, configs/config.dev.yaml) with environment variable overrides using `SCIFIND_` prefix.

**Database Layer:**
- GORM v2 with PostgreSQL (production) and SQLite (development)  
- Repository pattern in `internal/repository/`
- Models in `internal/models/`
- Database initialization via Wire injection

**HTTP API:**
- Gin router with middleware pipeline in `internal/api/router.go`
- RESTful endpoints: `/v1/search`, `/v1/papers`, `/v1/authors`
- Health checks at `/health`, `/health/live`, `/health/ready`
- Structured request/response handlers in `internal/api/handlers/`

**Search Providers:**
- Provider abstraction in `internal/providers/interfaces.go`
- Individual providers: `arxiv/`, `semantic_scholar/`, `exa/`, `tavily/`
- Provider manager coordinates multi-provider searches
- Circuit breaker and retry logic in `internal/errors/`

**Messaging:** 
- NATS.io integration with embedded server option
- Messaging abstraction in `internal/messaging/`
- Can run embedded NATS or connect to external NATS cluster

**Services Layer:**
- Business logic in `internal/services/`
- Service interfaces for testing and dependency inversion
- Wire-injected service containers

### Key Patterns

**Error Handling:** Custom error classification system with circuit breakers and retry mechanisms. Error types defined in `internal/errors/types.go`.

**Middleware Pipeline:** Request ID, CORS, security headers, structured logging, recovery middleware applied globally.

**Provider System:** Each search provider implements a common interface allowing transparent multi-provider searches with result aggregation.

**Health Checks:** Comprehensive health checking for database, messaging, and external providers with liveness/readiness probes.

## Configuration

**Development:** Uses SQLite by default. Configuration in `config.yaml` and `configs/config.dev.yaml`.

**Production:** Expects PostgreSQL and external NATS. Environment variables override YAML config using `SCIFIND_` prefix (e.g., `SCIFIND_SERVER_PORT=8080`).

**Provider API Keys:** Configure in YAML or via environment variables for enhanced search capabilities beyond free tiers.

## Testing

**Test Structure:**
- Unit tests: `test/unit/`
- Integration tests: `test/integration/` (uses testcontainers)
- E2E tests: `test/e2e/`
- Benchmarks: `test/benchmarks/`
- Test utilities and mocks: `test/testutil/`, `test/mocks/`

**Database Testing:** Uses testcontainers for PostgreSQL integration tests. SQLite for fast unit tests.

## Code Generation

- Wire dependency injection: `go generate ./cmd/server`
- Mock generation: Use mockgen for service interfaces when adding new ones