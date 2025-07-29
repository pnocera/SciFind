# SciFind Backend API Reference

Complete REST API documentation for the SciFind backend with examples, response schemas, and error handling.

## 📋 Table of Contents
- [Base URL & Authentication](#base-url--authentication)
- [Search Endpoints](#search-endpoints)
- [Paper Endpoints](#paper-endpoints)
- [Author Endpoints](#author-endpoints)
- [Provider Endpoints](#provider-endpoints)
- [Health & Monitoring](#health--monitoring)
- [Analytics Endpoints](#analytics-endpoints)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Examples](#examples)

## 🔗 Base URL & Authentication

### Base URL
```
http://localhost:8080/v1
```

### Authentication
Currently, the API does not require authentication for most endpoints. Provider configuration endpoints may require authentication in production deployments.

```http
Authorization: Bearer your-api-key-here
```

### Content Type
All requests and responses use JSON:
```http
Content-Type: application/json
```

## 🔍 Search Endpoints

### Search Papers
Search across multiple academic paper providers.

```http
GET /v1/search
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | ✅ | Search query string (1-1000 characters) |
| `limit` | integer | ❌ | Number of results (1-100, default: 20) |
| `offset` | integer | ❌ | Results offset (default: 0) |
| `providers` | string | ❌ | Comma-separated providers: `arxiv,semantic_scholar,exa,tavily` |
| `date_from` | string | ❌ | Start date (YYYY-MM-DD) |
| `date_to` | string | ❌ | End date (YYYY-MM-DD) |
| `author` | string | ❌ | Author name filter |
| `journal` | string | ❌ | Journal name filter |
| `category` | string | ❌ | Category filter |
| `subject` | string | ❌ | Subject area filter |

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/search?query=machine+learning&limit=10&providers=arxiv,semantic_scholar&date_from=2024-01-01"
```

#### Example Response
```json
{
  "request_id": "req_1234567890_abc123",
  "query": "machine learning",
  "papers": [
    {
      "id": 1,
      "title": "Advanced Machine Learning Techniques",
      "abstract": "This paper explores advanced techniques in machine learning...",
      "authors": ["John Doe", "Jane Smith"],
      "provider": "arxiv",
      "source_id": "2401.12345",
      "url": "https://arxiv.org/abs/2401.12345",
      "pdf_url": "https://arxiv.org/pdf/2401.12345.pdf",
      "published_date": "2024-01-15T00:00:00Z",
      "created_at": "2024-01-25T10:30:00Z",
      "updated_at": "2024-01-25T10:30:00Z",
      "categories": ["cs.LG", "cs.AI"],
      "keywords": ["machine learning", "neural networks"],
      "citation_count": 42,
      "is_open_access": true
    }
  ],
  "total_count": 150,
  "result_count": 1,
  "providers_used": ["arxiv", "semantic_scholar"],
  "providers_failed": [],
  "duration": 1234567890,
  "aggregation_strategy": "merge",
  "cache_hits": 0,
  "partial_failure": false,
  "errors": [],
  "timestamp": "2024-01-25T10:30:00Z"
}
```

### Get Paper by ID
Retrieve a specific paper from a provider.

```http
GET /v1/search/papers/{provider}/{id}
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `provider` | string | ✅ | Provider name: `arxiv`, `semantic_scholar`, `exa`, `tavily` |
| `id` | string | ✅ | Paper ID from the provider |

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/search/papers/arxiv/2401.12345"
```

#### Example Response
```json
{
  "paper": {
    "id": 1,
    "title": "Advanced Machine Learning Techniques",
    "abstract": "This paper explores advanced techniques in machine learning...",
    "authors": ["John Doe", "Jane Smith"],
    "provider": "arxiv",
    "source_id": "2401.12345",
    "url": "https://arxiv.org/abs/2401.12345",
    "pdf_url": "https://arxiv.org/pdf/2401.12345.pdf",
    "published_date": "2024-01-15T00:00:00Z",
    "created_at": "2024-01-25T10:30:00Z",
    "updated_at": "2024-01-25T10:30:00Z",
    "categories": ["cs.LG", "cs.AI"],
    "keywords": ["machine learning", "neural networks"],
    "citation_count": 42,
    "is_open_access": true
  },
  "source": "arxiv",
  "timestamp": "2024-01-25T10:30:00Z"
}
```

## 📄 Paper Endpoints

### List Papers
Retrieve a list of papers with optional filtering.

```http
GET /v1/papers
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `limit` | integer | ❌ | Number of results (1-100, default: 20) |
| `offset` | integer | ❌ | Results offset (default: 0) |
| `provider` | string | ❌ | Filter by provider |
| `author` | string | ❌ | Filter by author name |
| `category` | string | ❌ | Filter by category |

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/papers?limit=10&provider=arxiv"
```

### Create Paper
Create a new paper record.

```http
POST /v1/papers
```

#### Request Body
```json
{
  "title": "New Research Paper",
  "abstract": "This paper presents...",
  "authors": ["Author Name"],
  "provider": "manual",
  "source_id": "manual-001",
  "url": "https://example.com/paper",
  "categories": ["cs.AI"]
}
```

### Get Paper
Retrieve a specific paper by internal ID.

```http
GET /v1/papers/{id}
```

### Update Paper
Update an existing paper.

```http
PUT /v1/papers/{id}
```

### Delete Paper
Delete a paper record.

```http
DELETE /v1/papers/{id}
```

## 👥 Author Endpoints

### List Authors
Retrieve a list of authors.

```http
GET /v1/authors
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `limit` | integer | ❌ | Number of results (1-100, default: 20) |
| `offset` | integer | ❌ | Results offset (default: 0) |
| `query` | string | ❌ | Search query for author names |

### Get Author
Retrieve a specific author by ID.

```http
GET /v1/authors/{id}
```

### Get Author Papers
Retrieve papers by a specific author.

```http
GET /v1/authors/{id}/papers
```

## 🏗️ Provider Endpoints

### List Providers
Get available search providers and their status.

```http
GET /v1/search/providers
```

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/search/providers"
```

#### Example Response
```json
{
  "providers": {
    "arxiv": {
      "name": "arxiv",
      "enabled": true,
      "healthy": true,
      "last_check": "2024-01-25T10:30:00Z",
      "last_error": null,
      "circuit_state": "closed",
      "rate_limited": false,
      "reset_time": null,
      "avg_response_time": 234000000,
      "success_rate": 0.95,
      "api_version": "v1",
      "last_updated": "2024-01-25T10:30:00Z"
    },
    "semantic_scholar": {
      "name": "semantic_scholar",
      "enabled": true,
      "healthy": true,
      "last_check": "2024-01-25T10:30:00Z",
      "last_error": null,
      "circuit_state": "closed",
      "rate_limited": false,
      "reset_time": null,
      "avg_response_time": 156000000,
      "success_rate": 0.98,
      "api_version": "v1",
      "last_updated": "2024-01-25T10:30:00Z"
    }
  },
  "timestamp": "2024-01-25T10:30:00Z"
}
```

### Get Provider Metrics
Get performance metrics for search providers.

```http
GET /v1/search/providers/metrics
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `provider` | string | ❌ | Specific provider name |

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/search/providers/metrics?provider=arxiv"
```

#### Example Response
```json
{
  "providers": {
    "arxiv": {
      "total_requests": 1000,
      "successful_requests": 950,
      "failed_requests": 50,
      "cached_requests": 200,
      "avg_response_time": 234000000,
      "min_response_time": 100000000,
      "max_response_time": 2000000000,
      "p95_response_time": 500000000,
      "timeout_errors": 5,
      "rate_limit_errors": 2,
      "network_errors": 3,
      "parse_errors": 1,
      "rate_limit_hits": 10,
      "rate_limit_resets": 5,
      "circuit_open_count": 2,
      "circuit_close_count": 2,
      "total_results": 15000,
      "avg_results_per_query": 15.0,
      "window_start": "2024-01-25T09:30:00Z",
      "window_end": "2024-01-25T10:30:00Z"
    }
  },
  "timestamp": "2024-01-25T10:30:00Z"
}
```

### Configure Provider
Update configuration for a specific provider.

```http
PUT /v1/search/providers/{provider}/configure
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `provider` | string | ✅ | Provider name |

#### Request Body
```json
{
  "name": "arxiv",
  "enabled": true,
  "base_url": "https://export.arxiv.org/api/query",
  "api_key": "",
  "timeout": 30000000000,
  "retry_delay": 1000000000,
  "max_retries": 3,
  "rate_limit": {
    "requests_per_second": 1,
    "requests_per_minute": 20,
    "requests_per_hour": 1000,
    "burst_size": 5,
    "backoff_duration": 60000000000
  },
  "circuit_breaker": {
    "failure_threshold": 5,
    "success_threshold": 3,
    "timeout": 60000000000,
    "max_requests": 10,
    "interval": 60000000000
  },
  "cache": {
    "enabled": true,
    "ttl": 3600000000000,
    "max_size": 1000,
    "key_prefix": "arxiv:"
  }
}
```

#### Example Response
```json
{
  "provider_name": "arxiv",
  "status": {
    "name": "arxiv",
    "enabled": true,
    "healthy": true,
    "last_check": "2024-01-25T10:35:00Z",
    "last_error": null,
    "circuit_state": "closed",
    "rate_limited": false,
    "reset_time": null,
    "avg_response_time": 234000000,
    "success_rate": 0.95,
    "api_version": "v1",
    "last_updated": "2024-01-25T10:35:00Z"
  },
  "message": "Provider configuration updated successfully",
  "timestamp": "2024-01-25T10:35:00Z"
}
```

## 🏥 Health & Monitoring

### Liveness Check
Simple liveness check for Kubernetes and other orchestrators.

```http
GET /health/live
```

#### Example Request
```bash
curl -X GET "http://localhost:8080/health/live"
```

#### Example Response
```json
{
  "status": "alive",
  "timestamp": "2024-01-25T10:30:00Z",
  "uptime": "2h15m30s"
}
```

### Readiness Check
Comprehensive readiness check including dependencies.

```http
GET /health/ready
```

#### Example Request
```bash
curl -X GET "http://localhost:8080/health/ready"
```

#### Example Response
```json
{
  "status": "healthy",
  "timestamp": "2024-01-25T10:30:00Z",
  "version": "1.0.0",
  "build_time": "unknown",
  "git_commit": "unknown",
  "environment": "development",
  "uptime": "2h15m30s",
  "checks": {
    "database": {
      "status": "healthy",
      "duration": 5000000,
      "metadata": {
        "database": "connected"
      }
    },
    "nats": {
      "status": "healthy",
      "duration": 2000000,
      "metadata": {
        "nats": "status_unknown"
      }
    },
    "resources": {
      "status": "healthy",
      "duration": 1000000,
      "metadata": {
        "goroutines": "unknown",
        "memory": "unknown",
        "cpu": "unknown"
      }
    }
  }
}
```

### Comprehensive Health Check
Detailed health information including external APIs.

```http
GET /health
```

#### Example Request
```bash
curl -X GET "http://localhost:8080/health"
```

#### Example Response
```json
{
  "status": "healthy",
  "timestamp": "2024-01-25T10:30:00Z",
  "version": "1.0.0",
  "build_time": "unknown",
  "git_commit": "unknown",
  "environment": "development",
  "uptime": "2h15m30s",
  "checks": {
    "database": {
      "status": "healthy",
      "duration": 5000000,
      "metadata": {
        "database": "connected"
      }
    },
    "nats": {
      "status": "healthy",
      "duration": 2000000,
      "metadata": {
        "nats": "status_unknown"
      }
    },
    "resources": {
      "status": "healthy",
      "duration": 1000000,
      "metadata": {
        "goroutines": "unknown",
        "memory": "unknown",
        "cpu": "unknown"
      }
    },
    "external_apis": {
      "status": "healthy",
      "duration": 3000000,
      "metadata": {
        "arxiv": "unknown",
        "semantic_scholar": "unknown",
        "exa": "unknown",
        "tavily": "unknown"
      }
    }
  }
}
```

### Ping Endpoint
Simple ping endpoint for basic connectivity checks.

```http
GET /ping
```

#### Example Request
```bash
curl -X GET "http://localhost:8080/ping"
```

#### Example Response
```json
{
  "status": "alive",
  "timestamp": "2024-01-25T10:30:00Z",
  "uptime": "2h15m30s"
}
```

## 📊 Analytics Endpoints

### Get Analytics Metrics
Retrieve search analytics and performance metrics.

```http
GET /v1/analytics/metrics
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `time_range` | string | ✅ | Time range: `1h`, `6h`, `24h`, `7d`, `30d` |
| `start_time` | string | ❌ | Start time (ISO 8601) |
| `end_time` | string | ❌ | End time (ISO 8601) |
| `granularity` | string | ❌ | Granularity: `hour`, `day`, `week` |

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/analytics/metrics?time_range=24h&granularity=hour"
```

### Get Popular Queries
Retrieve most popular search queries.

```http
GET /v1/analytics/queries
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `limit` | integer | ❌ | Number of queries to return (default: 10) |
| `time_range` | string | ❌ | Time range: `1h`, `24h`, `7d`, `30d` |

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/analytics/queries?limit=20&time_range=7d"
```

## ❌ Error Handling

### Error Response Format
All errors follow a consistent format:

```json
{
  "error": "Error type",
  "message": "Detailed error message",
  "request_id": "uuid-12345",
  "timestamp": "2024-01-25T10:30:00Z"
}
```

### Common Error Codes

| HTTP Status | Error Type | Description |
|-------------|------------|-------------|
| 400 | `Invalid request` | Invalid parameters or request format |
| 404 | `Not Found` | Resource not found |
| 408 | `Request Timeout` | Request timeout occurred |
| 429 | `Rate limit exceeded` | Rate limit exceeded |
| 500 | `Internal Server Error` | Server error |
| 503 | `Service Unavailable` | Service temporarily unavailable |

### Error Examples

#### 400 Bad Request
```json
{
  "error": "Invalid request",
  "message": "query is required",
  "request_id": "req_1234567890_abc123",
  "timestamp": "2024-01-25T10:30:00Z"
}
```

#### 404 Not Found
```json
{
  "error": "Failed to get paper",
  "message": "paper not found",
  "timestamp": "2024-01-25T10:30:00Z"
}
```

#### 408 Request Timeout  
```json
{
  "error": "Search failed",
  "message": "request timeout: context deadline exceeded",
  "request_id": "req_1234567890_abc123",
  "timestamp": "2024-01-25T10:30:00Z"
}
```

#### 429 Rate Limit
```json
{
  "error": "Search failed",
  "message": "rate limit exceeded for provider",
  "request_id": "req_1234567890_abc123",
  "timestamp": "2024-01-25T10:30:00Z"
}
```

#### 500 Internal Server Error
```json
{
  "error": "Search failed",
  "message": "internal server error occurred",
  "request_id": "req_1234567890_abc123", 
  "timestamp": "2024-01-25T10:30:00Z"
}
```

## ⏱️ Rate Limiting

### Global Rate Limits
The application implements rate limiting to ensure fair usage and prevent abuse:

- **Default limit**: 100 requests per minute per IP
- **Burst allowance**: 10 additional requests
- **Window**: 1 minute rolling window

### Provider Rate Limits
Each external provider has its own rate limits that are respected:

| Provider | Default Rate Limit | Notes |
|----------|-------------------|--------|
| arXiv | 3 seconds between requests | Public API, no key required |
| Semantic Scholar | Variable | Depends on API key tier |
| Exa | Variable | Requires API key |
| Tavily | Variable | Requires API key |

### Rate Limit Configuration
Rate limits can be configured per environment:

```yaml
security:
  rate_limit:
    enabled: true
    requests: 100
    window: "1m"
    burst_size: 10
```

## 📚 Examples

### Basic Search
```bash
# Search for papers about neural networks
curl -X GET "http://localhost:8080/v1/search?query=neural+networks&limit=5"
```

### Advanced Search with Filters
```bash
# Search for recent AI papers with specific criteria
curl -X GET "http://localhost:8080/v1/search?query=artificial+intelligence&limit=20&providers=arxiv,semantic_scholar&date_from=2024-01-01&author=Hinton&category=cs.AI"
```

### Get Specific Paper
```bash
# Get paper by provider and ID
curl -X GET "http://localhost:8080/v1/search/papers/arxiv/2401.12345"
```

### Provider Management
```bash
# Check all providers
curl -X GET "http://localhost:8080/v1/search/providers"

# Get provider metrics
curl -X GET "http://localhost:8080/v1/search/providers/metrics?provider=arxiv"

# Configure provider
curl -X PUT "http://localhost:8080/v1/search/providers/arxiv/configure" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "arxiv",
    "enabled": true,
    "base_url": "https://export.arxiv.org/api/query",
    "timeout": 30000000000,
    "max_retries": 3
  }'
```

### Health Monitoring
```bash
# Basic health check
curl -X GET "http://localhost:8080/health"

# Liveness probe
curl -X GET "http://localhost:8080/health/live"

# Readiness probe  
curl -X GET "http://localhost:8080/health/ready"

# Simple ping
curl -X GET "http://localhost:8080/ping"
```

### Paper Management
```bash
# List papers
curl -X GET "http://localhost:8080/v1/papers?limit=10&provider=arxiv"

# Get specific paper
curl -X GET "http://localhost:8080/v1/papers/1"

# Create new paper
curl -X POST "http://localhost:8080/v1/papers" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "New Research Paper",
    "abstract": "This paper presents novel research...",
    "authors": ["Dr. Jane Smith"],
    "provider": "manual",
    "source_id": "manual-001",
    "url": "https://example.com/paper.pdf"
  }'
```

### Author Queries
```bash
# List authors
curl -X GET "http://localhost:8080/v1/authors?limit=10"

# Get author details
curl -X GET "http://localhost:8080/v1/authors/1"

# Get author's papers
curl -X GET "http://localhost:8080/v1/authors/1/papers"
```

## 🧪 Testing Your Integration

### Test Script
```bash
#!/bin/bash
# test_api.sh

BASE_URL="http://localhost:8080"

# Health check
echo "Testing health endpoint..."
curl -s "$BASE_URL/health" | jq .

# Search test
echo -e "\nTesting search..."
curl -s "$BASE_URL/v1/search?query=test&limit=3" | jq .

# Provider status
echo -e "\nTesting provider status..."
curl -s "$BASE_URL/v1/search/providers" | jq .

# Get API documentation
echo -e "\nTesting docs endpoint..."
curl -s "$BASE_URL/docs" | jq .
```

### Integration Testing
```bash
# Test complete workflow
echo "=== SciFind API Integration Test ==="

# 1. Check service is running
curl -f "$BASE_URL/ping" || { echo "Service not running"; exit 1; }

# 2. Perform search
SEARCH_RESULT=$(curl -s "$BASE_URL/v1/search?query=machine+learning&limit=1")
echo "Search completed: $(echo $SEARCH_RESULT | jq -r '.result_count') results"

# 3. Extract paper details
PROVIDER=$(echo $SEARCH_RESULT | jq -r '.papers[0].provider')
SOURCE_ID=$(echo $SEARCH_RESULT | jq -r '.papers[0].source_id')

# 4. Get specific paper
if [ "$PROVIDER" != "null" ] && [ "$SOURCE_ID" != "null" ]; then
    curl -s "$BASE_URL/v1/search/papers/$PROVIDER/$SOURCE_ID" | jq .
    echo "Paper retrieval test completed"
fi

# 5. Check provider health
curl -s "$BASE_URL/v1/search/providers" | jq '.providers | keys'
echo "Provider status check completed"
```

### Error Handling Testing
```bash
# Test error scenarios
echo "=== Error Handling Tests ==="

# Invalid query (empty)
curl -s "$BASE_URL/v1/search?query=" | jq .

# Invalid provider
curl -s "$BASE_URL/v1/search/papers/invalid/123" | jq .

# Non-existent paper
curl -s "$BASE_URL/v1/papers/999999" | jq .
```

## 📖 API Documentation

### Interactive Documentation
Access the interactive API documentation:
```
http://localhost:8080/docs
```

### Root Endpoint Information
Get basic service information:
```bash
curl -X GET "http://localhost:8080/"
```

Response:
```json
{
  "service": "SciFIND Backend",
  "version": "1.0.0", 
  "status": "running",
  "docs": "/docs",
  "health": "/health"
}
```

### MCP Protocol Support
The API also supports the Model Context Protocol (MCP) for AI assistant integration:

```json
{
  "mcp_server": {
    "description": "This server also supports Model Context Protocol",
    "methods": ["search", "get_paper", "list_capabilities", "get_schema", "ping"]
  }
}
```

## 📞 Support & Resources

### Documentation
- [Architecture Overview](ARCHITECTURE.md) - System design and components
- [Configuration Guide](CONFIGURATION.md) - Setup and configuration
- [Deployment Guide](DEPLOYMENT.md) - Production deployment
- [Performance Guide](PERFORMANCE.md) - Optimization and monitoring
- [Security Guide](SECURITY.md) - Security best practices
- [Testing Guide](TESTING.md) - Testing strategies and tools
- [Troubleshooting Guide](TROUBLESHOOTING.md) - Common issues and solutions
- [Contributing Guide](CONTRIBUTING.md) - Development and contribution guidelines

### API Support
For API-related questions:
1. Check the health endpoints to verify service status
2. Review error messages and request IDs for debugging
3. Consult the troubleshooting guide for common issues
4. Check provider status if search issues occur

### Best Practices
- Always include error handling for API calls
- Use appropriate timeout values for your use case
- Implement retry logic with exponential backoff
- Monitor provider status for availability
- Cache results when appropriate to reduce API calls
- Use pagination for large result sets