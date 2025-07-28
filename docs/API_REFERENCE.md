# SciFind Backend API Reference

Complete REST API documentation for the SciFind backend with examples, response schemas, and error handling.

## 📋 Table of Contents
- [Base URL & Authentication](#base-url--authentication)
- [Search Endpoints](#search-endpoints)
- [Paper Endpoints](#paper-endpoints)
- [Provider Endpoints](#provider-endpoints)
- [Health & Monitoring](#health--monitoring)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Examples](#examples)

## 🔗 Base URL & Authentication

### Base URL
```
http://localhost:8080/v1
```

### Authentication
API requests require an API key passed in the `Authorization` header:

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
GET /search
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | ✅ | Search query string |
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
curl -X GET "http://localhost:8080/v1/search?query=machine+learning&limit=10&providers=arxiv,semantic_scholar&date_from=2024-01-01" \
  -H "Authorization: Bearer your-api-key"
```

#### Example Response
```json
{
  "results": [
    {
      "id": "2401.12345",
      "title": "Advanced Machine Learning Techniques",
      "authors": ["John Doe", "Jane Smith"],
      "abstract": "This paper explores...",
      "published_date": "2024-01-15",
      "provider": "arxiv",
      "url": "https://arxiv.org/abs/2401.12345",
      "pdf_url": "https://arxiv.org/pdf/2401.12345.pdf",
      "categories": ["cs.LG", "cs.AI"],
      "citation_count": 42,
      "score": 0.95
    }
  ],
  "result_count": 150,
  "duration": "1.234s",
  "providers": ["arxiv", "semantic_scholar"],
  "query": "machine learning",
  "filters": {
    "date_from": "2024-01-01"
  }
}
```

### Get Paper by ID
Retrieve a specific paper from a provider.

```http
GET /search/papers/{provider}/{id}
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `provider` | string | ✅ | Provider name: `arxiv`, `semantic_scholar`, `exa`, `tavily` |
| `id` | string | ✅ | Paper ID |

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/search/papers/arxiv/2401.12345" \
  -H "Authorization: Bearer your-api-key"
```

#### Example Response
```json
{
  "paper": {
    "id": "2401.12345",
    "title": "Advanced Machine Learning Techniques",
    "authors": [
      {
        "name": "John Doe",
        "affiliation": "MIT"
      },
      {
        "name": "Jane Smith",
        "affiliation": "Stanford"
      }
    ],
    "abstract": "This paper explores advanced techniques...",
    "published_date": "2024-01-15",
    "updated_date": "2024-01-20",
    "provider": "arxiv",
    "url": "https://arxiv.org/abs/2401.12345",
    "pdf_url": "https://arxiv.org/pdf/2401.12345.pdf",
    "categories": ["cs.LG", "cs.AI"],
    "keywords": ["machine learning", "neural networks", "deep learning"],
    "citation_count": 42,
    "reference_count": 150,
    "doi": "10.1234/example.doi",
    "journal": "Journal of Machine Learning Research"
  },
  "provider": "arxiv",
  "retrieved_at": "2024-01-25T10:30:00Z"
}
```

## 🏗️ Provider Endpoints

### List Providers
Get available search providers and their status.

```http
GET /providers
```

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/providers" \
  -H "Authorization: Bearer your-api-key"
```

#### Example Response
```json
{
  "providers": {
    "arxiv": {
      "name": "arXiv",
      "enabled": true,
      "healthy": true,
      "last_check": "2024-01-25T10:30:00Z",
      "response_time": "234ms",
      "rate_limit": {
        "requests_per_minute": 20,
        "remaining": 18,
        "reset_time": "2024-01-25T10:31:00Z"
      }
    },
    "semantic_scholar": {
      "name": "Semantic Scholar",
      "enabled": true,
      "healthy": true,
      "last_check": "2024-01-25T10:30:00Z",
      "response_time": "156ms",
      "rate_limit": {
        "requests_per_minute": 100,
        "remaining": 95,
        "reset_time": "2024-01-25T10:31:00Z"
      }
    }
  }
}
```

### Get Provider Configuration
Get configuration for a specific provider.

```http
GET /providers/{provider}/config
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `provider` | string | ✅ | Provider name |

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/providers/arxiv/config" \
  -H "Authorization: Bearer your-api-key"
```

#### Example Response
```json
{
  "provider_name": "arxiv",
  "status": {
    "enabled": true,
    "healthy": true,
    "last_check": "2024-01-25T10:30:00Z"
  },
  "config": {
    "base_url": "http://export.arxiv.org/api/query",
    "rate_limit": "3s",
    "timeout": "30s",
    "max_retries": 3
  }
}
```

### Update Provider Configuration
Update configuration for a specific provider.

```http
PUT /providers/{provider}/config
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `provider` | string | ✅ | Provider name |
| `enabled` | boolean | ❌ | Enable/disable provider |
| `rate_limit` | string | ❌ | Rate limit duration |
| `timeout` | string | ❌ | Request timeout |
| `api_key` | string | ❌ | API key (for providers that require it) |

#### Example Request
```bash
curl -X PUT "http://localhost:8080/v1/providers/arxiv/config" \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "rate_limit": "5s",
    "timeout": "45s"
  }'
```

#### Example Response
```json
{
  "provider_name": "arxiv",
  "status": {
    "enabled": true,
    "healthy": true,
    "last_check": "2024-01-25T10:30:00Z"
  },
  "message": "Provider configuration updated successfully",
  "timestamp": "2024-01-25T10:35:00Z"
}
```

## 🏥 Health & Monitoring

### Health Check
Check the health status of the service.

```http
GET /health
```

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/health"
```

#### Example Response
```json
{
  "status": "healthy",
  "timestamp": "2024-01-25T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h15m30s",
  "services": {
    "database": "healthy",
    "nats": "healthy",
    "providers": {
      "arxiv": "healthy",
      "semantic_scholar": "healthy"
    }
  }
}
```

### Metrics
Get Prometheus metrics.

```http
GET /metrics
```

#### Example Request
```bash
curl -X GET "http://localhost:8080/v1/metrics"
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
| 400 | `Bad Request` | Invalid parameters or request format |
| 401 | `Unauthorized` | Missing or invalid API key |
| 403 | `Forbidden` | Insufficient permissions |
| 404 | `Not Found` | Resource not found |
| 429 | `Too Many Requests` | Rate limit exceeded |
| 500 | `Internal Server Error` | Server error |
| 503 | `Service Unavailable` | Service temporarily unavailable |

### Error Examples

#### 400 Bad Request
```json
{
  "error": "Invalid request",
  "message": "Invalid date format. Use YYYY-MM-DD format.",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-01-25T10:30:00Z"
}
```

#### 429 Rate Limit
```json
{
  "error": "Rate limit exceeded",
  "message": "You have exceeded the rate limit of 100 requests per minute. Please try again in 30 seconds.",
  "request_id": "550e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2024-01-25T10:30:00Z"
}
```

## ⏱️ Rate Limiting

### Global Rate Limits
- **Standard tier**: 100 requests per minute
- **Premium tier**: 1000 requests per minute
- **Enterprise tier**: Custom limits

### Provider Rate Limits
Each provider has its own rate limits:

| Provider | Rate Limit | Notes |
|----------|------------|--------|
| arXiv | 20 requests/minute | Public API |
| Semantic Scholar | 100 requests/minute | Requires API key |
| Exa | 1000 requests/minute | Requires API key |
| Tavily | 1000 requests/minute | Requires API key |

### Rate Limit Headers
Response headers include rate limit information:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1643107200
```

## 📚 Examples

### Basic Search
```bash
# Search for papers about neural networks
curl -X GET "http://localhost:8080/v1/search?query=neural+networks&limit=5" \
  -H "Authorization: Bearer your-api-key"
```

### Advanced Search with Filters
```bash
# Search for recent AI papers with specific criteria
curl -X GET "http://localhost:8080/v1/search?query=artificial+intelligence&limit=20&providers=arxiv,semantic_scholar&date_from=2024-01-01&author=Hinton&category=cs.AI" \
  -H "Authorization: Bearer your-api-key"
```

### Batch Paper Retrieval
```bash
# Get multiple papers by ID
curl -X GET "http://localhost:8080/v1/search/papers/arxiv/2401.12345" \
  -H "Authorization: Bearer your-api-key"

curl -X GET "http://localhost:8080/v1/search/papers/semantic_scholar/123456789" \
  -H "Authorization: Bearer your-api-key"
```

### Provider Status Monitoring
```bash
# Check all providers
curl -X GET "http://localhost:8080/v1/providers" \
  -H "Authorization: Bearer your-api-key"

# Check specific provider
curl -X GET "http://localhost:8080/v1/providers/arxiv/config" \
  -H "Authorization: Bearer your-api-key"
```

### Configuration Updates
```bash
# Update provider settings
curl -X PUT "http://localhost:8080/v1/providers/semantic_scholar/config" \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "timeout": "30s"
  }'
```

## 🧪 Testing Your Integration

### Test Script
```bash
#!/bin/bash
# test_api.sh

API_KEY="your-api-key"
BASE_URL="http://localhost:8080/v1"

# Health check
echo "Testing health endpoint..."
curl -s "$BASE_URL/health" | jq .

# Search test
echo -e "\nTesting search..."
curl -s "$BASE_URL/search?query=test&limit=3" \
  -H "Authorization: Bearer $API_KEY" | jq .

# Provider status
echo -e "\nTesting provider status..."
curl -s "$BASE_URL/providers" \
  -H "Authorization: Bearer $API_KEY" | jq .
```

### Postman Collection
Import the Postman collection from `docs/postman/scifind-api.postman_collection.json` for easy testing.

### OpenAPI Specification
Access the interactive API documentation at:
```
http://localhost:8080/docs
```

## 📞 Support

For API support:
- Check the [troubleshooting guide](TROUBLESHOOTING.md)
- Review [configuration options](CONFIGURATION.md)
- Open an issue with your `request_id` for debugging