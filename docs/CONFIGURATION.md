# SciFind Backend Configuration Guide

Comprehensive configuration options for the SciFind backend with detailed explanations and examples.

## 📋 Table of Contents
- [Environment Variables](#environment-variables)
- [Configuration File](#configuration-file)
- [Database Configuration](#database-configuration)
- [NATS Configuration](#nats-configuration)
- [Provider Configuration](#provider-configuration)
- [Security Configuration](#security-configuration)
- [Performance Tuning](#performance-tuning)
- [Monitoring Configuration](#monitoring-configuration)

## 🔧 Environment Variables

All configuration can be set via environment variables using the `SCIFIND_` prefix:

```bash
# Server Configuration
export SCIFIND_SERVER_PORT=8080
export SCIFIND_SERVER_HOST="0.0.0.0"
export SCIFIND_SERVER_MODE="production"

# Database Configuration
export SCIFIND_DATABASE_TYPE="postgres"
export SCIFIND_DATABASE_POSTGRESQL_DSN="postgres://user:pass@localhost:5432/scifind"

# NATS Configuration
export SCIFIND_NATS_URL="nats://localhost:4222"
export SCIFIND_NATS_CLUSTER_ID="scifind-cluster"

# Provider API Keys
export SCIFIND_PROVIDERS_SEMANTIC_SCHOLAR_API_KEY="your_key_here"
export SCIFIND_PROVIDERS_EXA_API_KEY="your_key_here"
export SCIFIND_PROVIDERS_TAVILY_API_KEY="your_key_here"
```

## 📄 Configuration File

### Complete Configuration Example

```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"
  mode: "release"  # Options: debug, release, test
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  max_header_bytes: 1048576
  enable_gzip: true
  enable_cors: true
  enable_metrics: true

database:
  type: "postgres"  # Options: postgres, sqlite
  postgresql:
    dsn: "postgres://user:password@localhost:5432/scifind?sslmode=disable"
    max_connections: 25
    max_idle: 10
    max_lifetime: "1h"
    max_idle_time: "30m"
    auto_migrate: true
  sqlite:
    path: "./scifind.db"
    auto_migrate: true

nats:
  url: "nats://localhost:4222"
  cluster_id: "scifind-cluster"
  client_id: "scifind-backend"
  max_reconnects: 10
  reconnect_wait: "2s"
  timeout: "5s"
  ping_interval: 30
  max_pings_out: 2
  
  tls:
    enabled: false
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    ca_file: "/path/to/ca.pem"
  
  jetstream:
    enabled: true
    domain: ""
    max_memory: "1GB"
    max_storage: "10GB"
  
  kv_store:
    enabled: true
    bucket: "scifind-cache"
    ttl: "1h"

providers:
  arxiv:
    enabled: true
    base_url: "http://export.arxiv.org/api/query"
    rate_limit: "3s"
    timeout: "30s"
    retry_delay: "1s"
    max_retries: 3
  
  semantic_scholar:
    enabled: true
    api_key: "your_api_key_here"
    base_url: "https://api.semanticscholar.org/graph/v1"
    timeout: "15s"
  
  exa:
    enabled: false
    api_key: "your_api_key_here"
    base_url: "https://api.exa.ai"
    timeout: "15s"
  
  tavily:
    enabled: false
    api_key: "your_api_key_here"
    base_url: "https://api.tavily.com"
    timeout: "15s"

logging:
  level: "info"  # Options: debug, info, warn, error
  format: "json"  # Options: json, text
  add_source: false
  output: "stdout"

security:
  api_keys: ["your-api-key-1", "your-api-key-2"]
  rate_limit:
    enabled: true
    requests: 100
    window: "1m"
    burst_size: 10
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["*"]
    max_age: "12h"

circuit:
  enabled: true
  failure_threshold: 5
  success_threshold: 3
  timeout: "60s"
  max_requests: 10
  sliding_window: "60s"
  min_request_count: 10

retry:
  enabled: true
  max_attempts: 3
  initial_delay: "1s"
  max_delay: "30s"
  backoff_factor: 2.0
  jitter: true

monitoring:
  enabled: true
  metrics_port: 9090
  health_path: "/health"
  metrics_path: "/metrics"
```

## 🗄️ Database Configuration

### PostgreSQL Configuration

```yaml
database:
  type: "postgres"
  postgresql:
    dsn: "postgres://user:password@localhost:5432/scifind?sslmode=disable"
    max_connections: 25
    max_idle: 10
    max_lifetime: "1h"
    max_idle_time: "30m"
    auto_migrate: true
```

### SQLite Configuration (Development)

```yaml
database:
  type: "sqlite"
  sqlite:
    path: "./scifind.db"
    auto_migrate: true
```

### Connection Pool Settings

| Parameter | Description | Default |
|-----------|-------------|---------|
| `max_connections` | Maximum open connections | 25 |
| `max_idle` | Maximum idle connections | 10 |
| `max_lifetime` | Maximum connection lifetime | 1h |
| `max_idle_time` | Maximum idle time | 30m |
| `auto_migrate` | Auto-run migrations | true |

## 🚀 NATS Configuration

### Basic Configuration

```yaml
nats:
  url: "nats://localhost:4222"
  cluster_id: "scifind-cluster"
  client_id: "scifind-backend"
```

### TLS Configuration

```yaml
nats:
  tls:
    enabled: true
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    ca_file: "/path/to/ca.pem"
```

### JetStream Configuration

```yaml
nats:
  jetstream:
    enabled: true
    domain: ""
    max_memory: "1GB"
    max_storage: "10GB"
```

### Key-Value Store Configuration

```yaml
nats:
  kv_store:
    enabled: true
    bucket: "scifind-cache"
    ttl: "1h"
```

## 🔍 Provider Configuration

### ArXiv Provider

```yaml
providers:
  arxiv:
    enabled: true
    base_url: "http://export.arxiv.org/api/query"
    rate_limit: "3s"
    timeout: "30s"
    retry_delay: "1s"
    max_retries: 3
```

### Semantic Scholar Provider

```yaml
providers:
  semantic_scholar:
    enabled: true
    api_key: "your_api_key_here"
    base_url: "https://api.semanticscholar.org/graph/v1"
    timeout: "15s"
```

### Exa Provider

```yaml
providers:
  exa:
    enabled: true
    api_key: "your_api_key_here"
    base_url: "https://api.exa.ai"
    timeout: "15s"
```

### Tavily Provider

```yaml
providers:
  tavily:
    enabled: true
    api_key: "your_api_key_here"
    base_url: "https://api.tavily.com"
    timeout: "15s"
```

## 🔒 Security Configuration

### API Key Authentication

```yaml
security:
  api_keys:
    - "your-api-key-1"
    - "your-api-key-2"
```

### Rate Limiting

```yaml
security:
  rate_limit:
    enabled: true
    requests: 100
    window: "1m"
    burst_size: 10
```

### CORS Configuration

```yaml
security:
  cors:
    enabled: true
    allowed_origins: ["https://yourdomain.com"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Authorization", "Content-Type"]
    max_age: "12h"
```

## ⚡ Performance Tuning

### Circuit Breaker Configuration

```yaml
circuit:
  enabled: true
  failure_threshold: 5
  success_threshold: 3
  timeout: "60s"
  max_requests: 10
  sliding_window: "60s"
  min_request_count: 10
```

### Retry Configuration

```yaml
retry:
  enabled: true
  max_attempts: 3
  initial_delay: "1s"
  max_delay: "30s"
  backoff_factor: 2.0
  jitter: true
```

## 📊 Monitoring Configuration

### Metrics Configuration

```yaml
monitoring:
  enabled: true
  metrics_port: 9090
  health_path: "/health"
  metrics_path: "/metrics"
```

### Logging Configuration

```yaml
logging:
  level: "info"
  format: "json"
  add_source: false
  output: "stdout"
```

## 🔧 Environment-Specific Configurations

### Development Configuration

```yaml
server:
  mode: "debug"
  port: 8080

database:
  type: "sqlite"
  sqlite:
    path: "./dev.db"

logging:
  level: "debug"
```

### Production Configuration

```yaml
server:
  mode: "release"
  port: 8080

database:
  type: "postgres"
  postgresql:
    dsn: "postgres://user:pass@prod-db:5432/scifind?sslmode=require"
    max_connections: 50

logging:
  level: "info"
```

### Testing Configuration

```yaml
server:
  mode: "test"
  port: 8081

database:
  type: "sqlite"
  sqlite:
    path: ":memory:"

logging:
  level: "warn"
```

## 📝 Configuration Examples

### Production PostgreSQL with TLS

```yaml
database:
  type: "postgres"
  postgresql:
    dsn: "postgres://user:pass@prod-db:5432/scifind?sslmode=require"
    max_connections: 50
    max_idle: 20
    max_lifetime: "2h"
```

### High-Performance NATS Cluster

```yaml
nats:
  url: "nats://nats-cluster:4222"
  cluster_id: "scifind-cluster"
  client_id: "scifind-backend"
  jetstream:
    enabled: true
    max_memory: "2GB"
    max_storage: "50GB"
```

### Secure Production Setup

```yaml
security:
  api_keys: ["prod-key-1", "prod-key-2"]
  rate_limit:
    enabled: true
    requests: 1000
    window: "1m"
  cors:
    enabled: true
    allowed_origins: ["https://yourdomain.com"]
```

## 🔄 Configuration Reloading

The application supports hot-reloading of configuration:

1. Update the configuration file
2. Send a SIGHUP signal: `kill -HUP <pid>`
3. Configuration will be reloaded without restart

## 📋 Configuration Validation

Validate your configuration:

```bash
# Validate configuration
go run cmd/server/main.go --config-check

# Validate with specific file
go run cmd/server/main.go --config-file=config.yaml --config-check