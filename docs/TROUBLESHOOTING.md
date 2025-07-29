# SciFind Backend Troubleshooting Guide

Common issues, debugging guides, and frequently asked questions for the SciFind backend.

## Table of Contents
- [Quick Diagnostics](#quick-diagnostics)
- [Common Issues](#common-issues)
- [Database Issues](#database-issues)
- [NATS Messaging Issues](#nats-messaging-issues)
- [Provider Issues](#provider-issues)
- [Performance Issues](#performance-issues)
- [Deployment Issues](#deployment-issues)
- [API Issues](#api-issues)
- [Security Issues](#security-issues)
- [Debugging Tools](#debugging-tools)
- [Frequently Asked Questions](#frequently-asked-questions)
- [Getting Help](#getting-help)

## Quick Diagnostics

### Health Check Commands

```bash
# Check application health
curl http://localhost:8080/health

# Check detailed health status
curl http://localhost:8080/health?detailed=true

# Check specific components
curl http://localhost:8080/health/database
curl http://localhost:8080/health/messaging
curl http://localhost:8080/health/providers
```

### System Status Commands

```bash
# Check if application is running
ps aux | grep scifind-backend

# Check port availability
lsof -i :8080

# Check logs
tail -f logs/scifind-backend.log

# Check resource usage
top -p $(pgrep scifind-backend)
```

### Environment Verification

```bash
# Check Go version
go version

# Check environment variables
env | grep SCIFIND

# Check configuration
./scifind-backend --config-check

# Check database connectivity
pg_isready -h localhost -p 5432
```

## Common Issues

### Application Won't Start

#### Symptom
```bash
$ make run
panic: failed to initialize application: database connection failed
```

#### Possible Causes & Solutions

1. **Database not running**
   ```bash
   # Check PostgreSQL status
   systemctl status postgresql
   
   # Start PostgreSQL
   systemctl start postgresql
   
   # For development, start Docker containers
   make docker-up
   ```

2. **Incorrect database configuration**
   ```bash
   # Verify database DSN
   echo $SCIFIND_DATABASE_POSTGRESQL_DSN
   
   # Test connection manually
   psql "postgres://user:pass@localhost:5432/scifind"
   ```

3. **Missing environment variables**
   ```bash
   # Check required variables
   echo $SCIFIND_SERVER_PORT
   echo $SCIFIND_DATABASE_TYPE
   
   # Load from .env file if needed
   source .env
   ```

4. **Port already in use**
   ```bash
   # Find what's using the port
   lsof -i :8080
   
   # Kill the process
   kill -9 $(lsof -t -i:8080)
   
   # Or use different port
   export SCIFIND_SERVER_PORT=8081
   ```

### High Memory Usage

#### Symptom
```bash
$ top
  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM
 1234 user      20   0  2.5g    1.2g     8m S   5.0 15.2 scifind-backend
```

#### Solutions

1. **Optimize database connection pool**
   ```yaml
   database:
     postgresql:
       max_connections: 10  # Reduce from default 25
       max_idle: 5
   ```

2. **Enable garbage collection monitoring**
   ```bash
   # Run with GC stats
   GODEBUG=gctrace=1 ./scifind-backend
   ```

3. **Check for memory leaks**
   ```bash
   # Profile memory usage
   curl http://localhost:8080/debug/pprof/heap > heap.prof
   go tool pprof heap.prof
   ```

### High CPU Usage

#### Symptom
CPU usage consistently above 80%

#### Solutions

1. **Check for infinite loops in providers**
   ```bash
   # Get CPU profile
   curl http://localhost:8080/debug/pprof/profile > cpu.prof
   go tool pprof cpu.prof
   ```

2. **Optimize search queries**
   ```bash
   # Enable query logging
   export SCIFIND_LOG_LEVEL=debug
   
   # Look for slow queries
   grep "slow query" logs/scifind-backend.log
   ```

3. **Rate limit aggressive clients**
   ```yaml
   security:
     rate_limit:
       requests: 50  # Reduce from 100
       window: "1m"
   ```

## Database Issues

### Connection Failures

#### PostgreSQL Connection Issues

**Symptom:**
```
failed to connect to PostgreSQL: connection refused
```

**Solutions:**

1. **Check PostgreSQL service**
   ```bash
   # Check if PostgreSQL is running
   sudo systemctl status postgresql
   
   # Start PostgreSQL
   sudo systemctl start postgresql
   
   # Enable auto-start
   sudo systemctl enable postgresql
   ```

2. **Verify connection string**
   ```bash
   # Test connection
   psql "postgres://username:password@localhost:5432/scifind"
   
   # Check DSN format
   export SCIFIND_DATABASE_POSTGRESQL_DSN="postgres://user:pass@host:port/db?sslmode=disable"
   ```

3. **Check firewall rules**
   ```bash
   # Allow PostgreSQL port
   sudo ufw allow 5432
   
   # Check listening ports
   netstat -ln | grep 5432
   ```

#### SQLite Issues

**Symptom:**
```
database is locked
```

**Solutions:**

1. **Check file permissions**
   ```bash
   # Ensure write permissions
   chmod 666 scifind.db
   chmod 777 $(dirname scifind.db)
   ```

2. **Close existing connections**
   ```bash
   # Find processes using the database
   lsof scifind.db
   
   # Kill if necessary
   kill -9 <PID>
   ```

### Migration Issues

#### Failed Migrations

**Symptom:**
```
migration failed: column already exists
```

**Solutions:**

1. **Check migration status**
   ```bash
   # Connect to database
   psql -d scifind
   
   # Check migration table
   SELECT * FROM schema_migrations;
   ```

2. **Rollback and retry**
   ```bash
   # Rollback migration
   make migrate-down
   
   # Re-run migration
   make migrate-up
   ```

3. **Manual migration repair**
   ```sql
   -- Fix migration state
   DELETE FROM schema_migrations WHERE version = 'problematic_version';
   
   -- Run specific migration
   \i migrations/001_create_papers.up.sql
   ```

### Performance Issues

#### Slow Queries

**Symptom:**
Database queries taking > 1 second

**Solutions:**

1. **Add database indexes**
   ```sql
   -- Index for search queries
   CREATE INDEX idx_papers_title_gin ON papers USING gin(to_tsvector('english', title));
   CREATE INDEX idx_papers_provider_source ON papers(provider, source_id);
   CREATE INDEX idx_papers_created_at ON papers(created_at DESC);
   ```

2. **Optimize query patterns**
   ```go
   // Bad: N+1 query problem
   for _, paper := range papers {
       authors := getAuthorsForPaper(paper.ID)
   }
   
   // Good: Use preloading
   db.Preload("Authors").Find(&papers)
   ```

3. **Enable query logging**
   ```yaml
   database:
     postgresql:
       log_level: info
       slow_query_threshold: "100ms"
   ```

## NATS Messaging Issues

### Connection Problems

#### NATS Server Not Running

**Symptom:**
```
failed to connect to NATS: connection refused
```

**Solutions:**

1. **Start NATS server**
   ```bash
   # Install NATS server
   go install github.com/nats-io/nats-server/v2@latest
   
   # Start with JetStream
   nats-server -js
   
   # Or use Docker
   docker run -p 4222:4222 nats:2.10-alpine -js
   ```

2. **Check NATS configuration**
   ```yaml
   nats:
     url: "nats://localhost:4222"
     cluster_id: "scifind-cluster"
     client_id: "scifind-backend"
   ```

3. **Verify network connectivity**
   ```bash
   # Test NATS connection
   nats-cli server ping nats://localhost:4222
   
   # Check NATS info
   curl http://localhost:8222/varz
   ```

#### JetStream Issues

**Symptom:**
```
JetStream not enabled on server
```

**Solutions:**

1. **Enable JetStream**
   ```bash
   # Start NATS with JetStream
   nats-server -js --store_dir=./jetstream
   ```

2. **Check JetStream status**
   ```bash
   # Verify JetStream is enabled
   nats-cli stream ls
   
   # Check JetStream info
   nats-cli server info
   ```

3. **Configure JetStream storage**
   ```yaml
   nats:
     jetstream:
       enabled: true
       store_dir: "./jetstream"
       max_memory: "1GB"
       max_storage: "10GB"
   ```

### Message Processing Issues

#### Messages Not Being Processed

**Symptom:**
Messages queued but not processed

**Solutions:**

1. **Check consumer status**
   ```bash
   # List consumers
   nats-cli consumer ls scifind-stream
   
   # Check consumer info
   nats-cli consumer info scifind-stream search-consumer
   ```

2. **Verify message handlers**
   ```go
   // Ensure message handlers are registered
   func (s *Service) Start() error {
       s.messaging.Subscribe("search.request", s.handleSearchRequest)
       return nil
   }
   ```

3. **Check for processing errors**
   ```bash
   # Look for error logs
   grep "message processing failed" logs/scifind-backend.log
   ```

## Provider Issues

### API Key Problems

#### Invalid API Key

**Symptom:**
```
provider request failed: 401 Unauthorized
```

**Solutions:**

1. **Verify API keys**
   ```bash
   # Check environment variables
   echo $SCIFIND_PROVIDERS_SEMANTIC_SCHOLAR_API_KEY
   
   # Test API key directly
   curl -H "x-api-key: $API_KEY" https://api.semanticscholar.org/graph/v1/paper/search?query=test
   ```

2. **Update configuration**
   ```yaml
   providers:
     semantic_scholar:
       api_key: "your-valid-api-key"
   ```

### Rate Limiting

#### Provider Rate Limits Exceeded

**Symptom:**
```
provider request failed: 429 Too Many Requests
```

**Solutions:**

1. **Adjust rate limits**
   ```yaml
   providers:
     arxiv:
       rate_limit: "5s"  # Increase delay between requests
   ```

2. **Implement exponential backoff**
   ```go
   // Retry with exponential backoff
   backoff := time.Second
   for i := 0; i < maxRetries; i++ {
       if err := provider.Search(ctx, query); err == nil {
           break
       }
       time.Sleep(backoff)
       backoff *= 2
   }
   ```

3. **Use multiple API keys**
   ```yaml
   providers:
     semantic_scholar:
       api_keys: ["key1", "key2", "key3"]  # Round-robin usage
   ```

### Provider Timeouts

#### Slow Provider Responses

**Symptom:**
```
provider request timeout after 30s
```

**Solutions:**

1. **Increase timeout**
   ```yaml
   providers:
     exa:
       timeout: "60s"  # Increase from 30s
   ```

2. **Implement circuit breaker**
   ```yaml
   circuit:
     enabled: true
     failure_threshold: 3
     timeout: "60s"
   ```

3. **Use provider health checks**
   ```bash
   # Check provider status
   curl http://localhost:8080/v1/providers
   ```

## Performance Issues

### Slow Search Responses

#### High Response Times

**Symptom:**
Search requests taking > 5 seconds

**Debugging Steps:**

1. **Profile the application**
   ```bash
   # Get performance profile
   curl http://localhost:8080/debug/pprof/profile?seconds=30 > profile.prof
   go tool pprof profile.prof
   ```

2. **Check database performance**
   ```sql
   -- Enable query logging
   ALTER SYSTEM SET log_statement = 'all';
   ALTER SYSTEM SET log_min_duration_statement = 100;
   
   -- Check slow queries
   SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;
   ```

3. **Monitor provider performance**
   ```bash
   # Check provider metrics
   curl http://localhost:8080/v1/providers/metrics
   ```

**Solutions:**

1. **Implement caching**
   ```yaml
   providers:
     cache:
       enabled: true
       ttl: "1h"
       max_size: 1000
   ```

2. **Optimize concurrent searches**
   ```go
   // Search providers concurrently
   func (m *ProviderManager) SearchAll(ctx context.Context, query *SearchQuery) {
       var wg sync.WaitGroup
       results := make(chan *ProviderResult, len(m.providers))
       
       for _, provider := range m.providers {
           wg.Add(1)
           go func(p SearchProvider) {
               defer wg.Done()
               result, err := p.Search(ctx, query)
               results <- &ProviderResult{Provider: p, Result: result, Error: err}
           }(provider)
       }
       
       wg.Wait()
       close(results)
   }
   ```

3. **Use database connection pooling**
   ```yaml
   database:
     postgresql:
       max_connections: 25
       max_idle: 10
       max_lifetime: "1h"
   ```

### Memory Leaks

#### Growing Memory Usage

**Symptom:**
Memory usage increases over time and doesn't decrease

**Debugging:**

1. **Memory profiling**
   ```bash
   # Take heap dump
   curl http://localhost:8080/debug/pprof/heap > heap.prof
   
   # Analyze with pprof
   go tool pprof heap.prof
   (pprof) top10
   (pprof) list main.functionName
   ```

2. **Check goroutine leaks**
   ```bash
   # Check goroutine count
   curl http://localhost:8080/debug/pprof/goroutine?debug=1
   ```

**Solutions:**

1. **Fix context cancellation**
   ```go
   // Ensure contexts are cancelled
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

2. **Close HTTP response bodies**
   ```go
   resp, err := http.Get(url)
   if err != nil {
       return err
   }
   defer resp.Body.Close() // Important!
   ```

3. **Limit concurrent operations**
   ```go
   // Use semaphore to limit concurrency
   sem := make(chan struct{}, 10)
   for _, item := range items {
       sem <- struct{}{}
       go func(item Item) {
           defer func() { <-sem }()
           processItem(item)
       }(item)
   }
   ```

## Deployment Issues

### Docker Issues

#### Container Won't Start

**Symptom:**
```
docker: Error response from daemon: container exited with code 1
```

**Solutions:**

1. **Check container logs**
   ```bash
   docker logs scifind-backend
   docker logs --tail 50 scifind-backend
   ```

2. **Verify image build**
   ```bash
   # Rebuild image
   docker build -t scifind-backend .
   
   # Check image layers
   docker history scifind-backend
   ```

3. **Test container interactively**
   ```bash
   # Run container with shell
   docker run -it --entrypoint /bin/sh scifind-backend
   
   # Check file permissions
   ls -la /app/
   ```

#### Environment Variable Issues

**Symptom:**
Configuration not loaded in container

**Solutions:**

1. **Pass environment variables**
   ```bash
   # Using docker run
   docker run -e SCIFIND_SERVER_PORT=8080 scifind-backend
   
   # Using .env file
   docker run --env-file .env scifind-backend
   ```

2. **Check Docker Compose**
   ```yaml
   services:
     scifind-backend:
       environment:
         - SCIFIND_SERVER_PORT=8080
         - SCIFIND_DATABASE_TYPE=postgres
       env_file:
         - .env
   ```

### Kubernetes Issues

#### Pod Crashes

**Symptom:**
```
$ kubectl get pods
NAME                    READY   STATUS             RESTARTS   AGE
scifind-backend-xxx     0/1     CrashLoopBackOff   5          5m
```

**Solutions:**

1. **Check pod logs**
   ```bash
   kubectl logs scifind-backend-xxx
   kubectl logs scifind-backend-xxx --previous
   ```

2. **Describe pod for events**
   ```bash
   kubectl describe pod scifind-backend-xxx
   ```

3. **Check resource limits**
   ```yaml
   resources:
     requests:
       memory: "256Mi"
       cpu: "250m"
     limits:
       memory: "512Mi"
       cpu: "500m"
   ```

#### Service Discovery Issues

**Symptom:**
Can't connect to database service

**Solutions:**

1. **Check service endpoints**
   ```bash
   kubectl get endpoints
   kubectl describe service postgres-service
   ```

2. **Verify DNS resolution**
   ```bash
   kubectl run debug --image=busybox --rm -it -- nslookup postgres-service
   ```

3. **Test connectivity**
   ```bash
   kubectl exec -it scifind-backend-xxx -- telnet postgres-service 5432
   ```

## API Issues

### Authentication Failures

#### Invalid API Key

**Symptom:**
```
HTTP 401 Unauthorized
{"error": "invalid API key"}
```

**Solutions:**

1. **Check API key format**
   ```bash
   # Correct format
   curl -H "Authorization: Bearer your-api-key" http://localhost:8080/v1/search
   
   # Not this
   curl -H "X-API-Key: your-api-key" http://localhost:8080/v1/search
   ```

2. **Verify API key in configuration**
   ```yaml
   security:
     api_keys:
       - "valid-api-key-1"
       - "valid-api-key-2"
   ```

### Request Validation Errors

#### Invalid Request Format

**Symptom:**
```
HTTP 400 Bad Request
{"error": "validation failed: query is required"}
```

**Solutions:**

1. **Check request body format**
   ```bash
   # Correct JSON format
   curl -X POST \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer api-key" \
     -d '{"query":"machine learning","limit":10}' \
     http://localhost:8080/v1/search
   ```

2. **Validate required fields**
   ```json
   {
     "query": "required field",
     "limit": 10,
     "offset": 0
   }
   ```

### CORS Issues

#### Cross-Origin Requests Blocked

**Symptom:**
```
Access to fetch at 'http://localhost:8080/v1/search' from origin 'http://localhost:3000' 
has been blocked by CORS policy
```

**Solutions:**

1. **Configure CORS**
   ```yaml
   security:
     cors:
       enabled: true
       allowed_origins: ["http://localhost:3000", "https://yourdomain.com"]
       allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
       allowed_headers: ["Authorization", "Content-Type"]
   ```

2. **Check preflight requests**
   ```bash
   # Test OPTIONS request
   curl -X OPTIONS \
     -H "Origin: http://localhost:3000" \
     -H "Access-Control-Request-Method: POST" \
     -H "Access-Control-Request-Headers: Authorization,Content-Type" \
     http://localhost:8080/v1/search
   ```

## Security Issues

### Rate Limiting Problems

#### Rate Limit False Positives

**Symptom:**
Legitimate users getting rate limited

**Solutions:**

1. **Adjust rate limit configuration**
   ```yaml
   security:
     rate_limit:
       requests: 200  # Increase limit
       window: "1m"
       burst_size: 20  # Allow bursts
   ```

2. **Implement user-based rate limiting**
   ```go
   // Rate limit by API key instead of IP
   limiter := rate.NewLimiter(rate.Every(time.Minute/100), 20)
   ```

3. **Whitelist trusted IPs**
   ```yaml
   security:
     rate_limit:
       whitelist_ips:
         - "192.168.1.0/24"
         - "10.0.0.0/8"
   ```

### SSL/TLS Issues

#### Certificate Problems

**Symptom:**
```
certificate verify failed: self signed certificate
```

**Solutions:**

1. **Use valid certificates**
   ```bash
   # Generate with Let's Encrypt
   certbot certonly --standalone -d api.yourdomain.com
   ```

2. **Configure TLS properly**
   ```yaml
   server:
     tls:
       cert_file: "/etc/ssl/certs/api.yourdomain.com.crt"
       key_file: "/etc/ssl/private/api.yourdomain.com.key"
   ```

3. **Test certificate**
   ```bash
   # Verify certificate
   openssl x509 -in cert.pem -text -noout
   
   # Test SSL connection
   openssl s_client -connect api.yourdomain.com:443
   ```

## Debugging Tools

### Application Debugging

#### Enable Debug Mode

```bash
# Set debug log level
export SCIFIND_LOG_LEVEL=debug

# Enable debug mode
export SCIFIND_SERVER_MODE=debug

# Enable pprof endpoints
export SCIFIND_DEBUG_ENABLED=true
```

#### Profiling Commands

```bash
# CPU profiling
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profiling
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Goroutine analysis
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof

# Trace analysis
curl http://localhost:8080/debug/pprof/trace?seconds=10 > trace.out
go tool trace trace.out
```

### Database Debugging

#### PostgreSQL Debugging

```sql
-- Check current connections
SELECT * FROM pg_stat_activity;

-- Check slow queries
SELECT query, mean_time, calls, total_time 
FROM pg_stat_statements 
ORDER BY total_time DESC 
LIMIT 10;

-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check index usage
SELECT 
    indexrelname,
    idx_tup_read,
    idx_tup_fetch,
    idx_scan
FROM pg_stat_user_indexes;
```

### Network Debugging

#### Connection Testing

```bash
# Test HTTP connectivity
curl -v http://localhost:8080/health

# Test database connectivity
pg_isready -h localhost -p 5432

# Test NATS connectivity
nats-cli server ping nats://localhost:4222

# Test provider APIs
curl -H "X-API-Key: your-key" https://api.semanticscholar.org/graph/v1/paper/search?query=test

# Network monitoring
netstat -tulpn | grep :8080
ss -tulpn | grep :8080
```

## Frequently Asked Questions

### General Questions

**Q: How do I check if the application is running correctly?**

A: Use the health check endpoint:
```bash
curl http://localhost:8080/health
```

**Q: Where are the log files located?**

A: By default, logs go to stdout. To save to file:
```bash
./scifind-backend > logs/app.log 2>&1
```

**Q: How do I update the application?**

A: 
```bash
# Pull latest code
git pull origin main

# Rebuild
make build

# Restart service
systemctl restart scifind-backend
```

### Configuration Questions

**Q: How do I change the database from SQLite to PostgreSQL?**

A: Update your configuration:
```yaml
database:
  type: "postgres"
  postgresql:
    dsn: "postgres://user:pass@localhost:5432/scifind"
```

**Q: How do I add a new API provider?**

A: See the [Adding New Providers](CONTRIBUTING.md#adding-new-providers) section in the contributing guide.

### Performance Questions

**Q: The application is using too much memory. How do I optimize it?**

A: 
1. Reduce database connection pool size
2. Enable garbage collection monitoring
3. Check for memory leaks using pprof
4. Optimize query patterns

**Q: Search responses are slow. How do I improve performance?**

A:
1. Enable caching
2. Add database indexes
3. Optimize provider timeout settings
4. Use concurrent provider searches

### Deployment Questions

**Q: How do I deploy to production?**

A: Follow the [deployment guide](DEPLOYMENT.md) for your platform.

**Q: Can I run multiple instances behind a load balancer?**

A: Yes, the application is stateless and supports horizontal scaling.

## Getting Help

### Before Asking for Help

1. **Check this troubleshooting guide**
2. **Search existing GitHub issues**
3. **Check application logs**
4. **Verify your configuration**
5. **Test with minimal setup**

### How to Report Issues

When reporting issues, please include:

1. **Environment information**:
   ```bash
   go version
   uname -a
   ./scifind-backend --version
   ```

2. **Configuration** (sanitized, remove API keys):
   ```yaml
   # Your config.yaml
   ```

3. **Error logs**:
   ```
   # Relevant log entries
   ```

4. **Steps to reproduce**:
   ```
   1. Do this
   2. Then this
   3. Error occurs
   ```

5. **Expected vs actual behavior**

### Contact Information

- **GitHub Issues**: [Create an issue](https://github.com/scifind/scifind-backend/issues)
- **Documentation**: Check other files in `/docs` directory
- **Community**: Join our discussions

### Emergency Support

For critical production issues:
1. Check monitoring dashboards
2. Verify health endpoints
3. Check resource usage
4. Review recent deployments
5. Rollback if necessary

Remember: Many issues can be resolved by restarting the application and checking the logs!