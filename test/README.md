# NATS Testing Suite - Comprehensive Documentation

This directory contains comprehensive tests for the NATS messaging system implementation in scifind-backend, covering all aspects from unit tests to deployment validation.

## Test Structure

```
test/
├── unit/messaging/          # Unit tests for NATS components
├── integration/            # Integration tests with full system
├── security/              # mTLS and security tests
├── load/                  # Performance and load tests
├── deployment/            # Deployment scenario tests
├── testutil/              # Test utilities and helpers
└── README.md             # This documentation
```

## Test Categories

### 1. Unit Tests (`test/unit/messaging/`)

#### `client_test.go`
- **Purpose**: Tests NATS client connection lifecycle and basic operations
- **Coverage**: 
  - Client creation with various configurations
  - Connection state management
  - Publish/Subscribe operations
  - Error handling scenarios
  - JetStream integration
- **Key Features**:
  - Connection timeout validation
  - Invalid configuration handling
  - Message serialization/deserialization
  - Queue subscription testing
  - Performance benchmarks

#### `manager_test.go`
- **Purpose**: Tests messaging manager lifecycle and coordination
- **Coverage**:
  - Manager startup/shutdown procedures
  - Health monitoring functionality
  - Event handling systems
  - Concurrent access safety
  - Metrics collection
- **Key Features**:
  - Background goroutine management
  - Default event handler setup
  - Stats collection and reporting
  - Error scenario handling

### 2. Integration Tests (`test/integration/`)

#### `health_check_test.go`
- **Purpose**: Full system health check validation
- **Coverage**:
  - Health service integration with messaging
  - System info collection
  - External service health monitoring
  - Event-driven health reporting
  - Performance monitoring
- **Key Features**:
  - Multi-component health validation
  - Real-time health event processing
  - Metrics integration testing
  - Resilience under load

### 3. Security Tests (`test/security/`)

#### `mtls_test.go`
- **Purpose**: Mutual TLS security validation
- **Coverage**:
  - Certificate generation and validation
  - mTLS connection establishment
  - Certificate authentication
  - Security failure scenarios
  - JetStream over mTLS
- **Key Features**:
  - Full certificate chain testing
  - Client certificate validation
  - CA certificate verification
  - Performance impact measurement
  - Security breach prevention

### 4. Load Tests (`test/load/`)

#### `messaging_load_test.go`
- **Purpose**: Performance and throughput validation
- **Coverage**:
  - High-throughput message publishing
  - Concurrent subscriber handling
  - Queue group load distribution
  - Large message processing
  - Resource utilization
- **Key Features**:
  - Configurable load test scenarios
  - Real-time metrics collection
  - Performance baseline validation
  - Concurrency testing
  - Throughput measurement

### 5. Deployment Tests (`test/deployment/`)

#### `embedded_server_test.go`
- **Purpose**: Deployment scenario validation
- **Coverage**:
  - Embedded NATS server startup
  - Single executable deployment
  - Container deployment validation
  - Production configuration testing
  - Resource constraint handling
- **Key Features**:
  - Server configuration validation
  - Persistence across restarts
  - Docker integration testing
  - Production readiness checks

## Running Tests

### Prerequisites

1. **Go 1.24+** with modules enabled
2. **Docker** (for container tests)
3. **NATS Server** (embedded in tests)
4. **Testcontainers** for integration tests

### Basic Test Execution

```bash
# Run all tests
go test ./test/...

# Run specific test categories
go test ./test/unit/messaging/...
go test ./test/integration/...
go test ./test/security/...
go test ./test/load/...
go test ./test/deployment/...

# Run with verbose output
go test -v ./test/...

# Run only short tests (excludes integration/load tests)
go test -short ./test/...
```

### Advanced Test Options

```bash
# Run tests with race detection
go test -race ./test/...

# Run tests with coverage
go test -cover ./test/...
go test -coverprofile=coverage.out ./test/...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./test/...
go test -bench=BenchmarkNATS ./test/load/

# Run specific test functions
go test -run TestNATS_BasicLoadTest ./test/load/
go test -run TestMTLS ./test/security/
```

### Load Testing Configuration

The load tests accept environment variables for configuration:

```bash
# High-throughput test
LOAD_TEST_PUBLISHERS=10 \
LOAD_TEST_SUBSCRIBERS=10 \
LOAD_TEST_MESSAGES=10000 \
go test -run TestNATS_HighThroughputLoad ./test/load/

# Large message test
LOAD_TEST_MESSAGE_SIZE=102400 \
go test -run TestNATS_LargeMessageLoad ./test/load/
```

## Test Utilities

### `testutil/messaging.go`
Provides comprehensive NATS testing utilities:

- **SetupTestNATS()**: Creates containerized NATS server
- **NATSTestUtil**: Helper methods for test operations
- **JetStream helpers**: Stream and consumer management
- **Message utilities**: Publishing and subscription helpers

### Key Utility Features

- Automatic container lifecycle management
- JetStream context creation
- Stream and consumer setup
- Message publishing/receiving helpers
- Connection URL management
- Cleanup automation

## Performance Benchmarks

### Expected Performance Baselines

Based on test results, the following performance baselines are expected:

#### Message Throughput
- **Basic Publishing**: >10,000 msg/sec (1KB messages)
- **High Throughput**: >50,000 msg/sec (512B messages)
- **Large Messages**: >1,000 msg/sec (10KB messages)

#### Latency Metrics
- **Average Latency**: <5ms (local network)
- **95th Percentile**: <20ms
- **99th Percentile**: <50ms

#### Connection Performance
- **Client Connection**: <100ms
- **Embedded Server Startup**: <2 seconds
- **Health Check**: <10ms

### Load Test Scenarios

1. **Basic Load Test**
   - 5 publishers, 5 subscribers
   - 1,000 messages per publisher
   - 1KB message size
   - Target: >1,000 msg/sec

2. **High Throughput Test**
   - CPU-scaled publishers/subscribers
   - 5,000 messages per publisher
   - 512B message size
   - Target: >5,000 msg/sec

3. **Large Message Test**
   - 2 publishers, 2 subscribers
   - 500 messages per publisher
   - 10KB message size
   - Target: >5 MB/sec throughput

4. **Concurrent Subscribers Test**
   - 1 publisher, 20 subscribers
   - 2,000 messages total
   - Fan-out verification
   - Target: All messages to all subscribers

5. **Queue Group Test**
   - 1 publisher, 5 queue workers
   - 10,000 messages total
   - Load balancing verification
   - Target: Equal distribution

## Security Testing

### mTLS Test Scenarios

1. **Certificate Generation**
   - CA certificate creation
   - Server certificate generation
   - Client certificate generation
   - Certificate chain validation

2. **Connection Security**
   - Successful mTLS handshake
   - Client certificate validation
   - Server certificate verification
   - Certificate mismatch detection

3. **Security Failures**
   - Connection without client certificate
   - Invalid CA certificate
   - Expired certificate handling
   - Certificate file not found

4. **JetStream Security**
   - Stream operations over mTLS
   - Consumer security validation
   - Secure message persistence
   - Performance impact measurement

## Integration Testing

### Health Check Integration

1. **Component Health Monitoring**
   - Database health validation
   - Messaging health verification
   - External service monitoring
   - System resource tracking

2. **Event-Driven Health**
   - Health event publishing
   - Real-time health monitoring
   - Alert generation
   - Metrics collection

3. **Resilience Testing**
   - Concurrent health checks
   - Health monitoring under load
   - Recovery from failures
   - Performance degradation detection

## Deployment Testing

### Embedded Server Scenarios

1. **Basic Embedded Server**
   - Server startup and configuration
   - Client connection validation
   - Basic operations verification
   - Graceful shutdown

2. **Custom Configuration**
   - JetStream configuration
   - Resource limits
   - Clustering preparation
   - Storage configuration

3. **Persistence Validation**
   - Data persistence across restarts
   - Stream state recovery
   - Consumer state restoration
   - Configuration preservation

### Production Deployment

1. **High Availability Configuration**
   - Multi-node preparation
   - TLS configuration
   - Database clustering
   - Load balancer readiness

2. **Resource Constraints**
   - Memory-limited deployment
   - Storage-limited deployment
   - Connection-limited scenarios
   - Performance under constraints

3. **Container Deployment**
   - Docker build validation
   - Container startup testing
   - Health check endpoints
   - Resource utilization

## Continuous Integration

### CI/CD Pipeline Integration

```yaml
# Example GitHub Actions configuration
name: NATS Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      # Unit and Integration Tests
      - name: Run Unit Tests
        run: go test -v -race ./test/unit/...
      
      - name: Run Integration Tests
        run: go test -v ./test/integration/...
      
      # Security Tests
      - name: Run Security Tests
        run: go test -v ./test/security/...
      
      # Load Tests (reduced for CI)
      - name: Run Load Tests
        run: go test -v -short ./test/load/...
        env:
          LOAD_TEST_DURATION: "10s"
      
      # Coverage Report
      - name: Coverage Report
        run: |
          go test -coverprofile=coverage.out ./test/...
          go tool cover -func=coverage.out
```

## Troubleshooting

### Common Issues

1. **Container Startup Failures**
   - Ensure Docker is running
   - Check available ports
   - Verify container resources

2. **Connection Timeouts**
   - Increase timeout values
   - Check network connectivity
   - Verify server startup

3. **Certificate Issues**
   - Verify certificate paths
   - Check certificate validity
   - Ensure CA certificate availability

4. **Performance Issues**
   - Monitor system resources
   - Check for resource contention
   - Adjust test parameters

### Debug Configuration

```go
// Enable debug logging for tests
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

// Enable NATS debug tracing
opts := []nats.Option{
    nats.Timeout(30 * time.Second),
    nats.ReconnectWait(1 * time.Second),
    nats.MaxReconnects(5),
}
```

## Contributing

### Adding New Tests

1. **Follow naming conventions**: `*_test.go` files
2. **Use testify assertions**: Consistent error reporting
3. **Include benchmarks**: Performance regression detection
4. **Add documentation**: Clear test purpose and coverage
5. **Handle cleanup**: Proper resource management

### Test Quality Guidelines

1. **Isolation**: Tests should not depend on each other
2. **Reproducibility**: Tests should produce consistent results
3. **Performance**: Tests should complete in reasonable time
4. **Cleanup**: All resources should be properly cleaned up
5. **Documentation**: Clear test descriptions and expectations

## Metrics and Monitoring

### Test Metrics Collection

The test suite collects various metrics:

- **Performance metrics**: Latency, throughput, resource usage
- **Reliability metrics**: Error rates, connection success rates
- **Resource metrics**: Memory usage, connection counts
- **Security metrics**: Certificate validation, encryption overhead

### Monitoring Integration

Tests can integrate with monitoring systems:

```go
// Example Prometheus metrics integration
func publishMetrics(duration time.Duration, messageCount int64) {
    testDurationMetric.Observe(duration.Seconds())
    testMessageCountMetric.Add(float64(messageCount))
}
```

## Best Practices

### Test Development

1. **Test Pyramid**: More unit tests, fewer integration tests
2. **Fast Feedback**: Keep unit tests fast, isolate slow tests
3. **Clear Naming**: Test names should describe behavior
4. **Good Coverage**: Aim for >80% code coverage
5. **Edge Cases**: Test boundary conditions and error paths

### Performance Testing

1. **Baseline Establishment**: Set performance expectations
2. **Regular Monitoring**: Track performance trends
3. **Resource Awareness**: Monitor system resource usage
4. **Realistic Scenarios**: Test with production-like data
5. **Scalability Testing**: Validate performance under load

### Security Testing

1. **Comprehensive Coverage**: Test all security features
2. **Failure Scenarios**: Verify security failure handling
3. **Certificate Management**: Test certificate lifecycle
4. **Performance Impact**: Measure security overhead
5. **Regular Updates**: Keep security tests current

This comprehensive test suite ensures the NATS messaging system is robust, performant, and production-ready across all deployment scenarios.