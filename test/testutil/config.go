package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"scifind-backend/internal/config"
)

// TestConfig creates a test configuration
func TestConfig(t *testing.T) *config.Config {
	cfg := &config.Config{}

	// Server configuration
	cfg.Server.Port = 0 // Let the system assign a port
	cfg.Server.Host = "localhost"
	cfg.Server.Mode = "test"
	cfg.Server.ReadTimeout = "5s"
	cfg.Server.WriteTimeout = "5s"
	cfg.Server.IdleTimeout = "30s"

	// Database configuration (SQLite in-memory for tests)
	cfg.Database.Type = "sqlite"
	cfg.Database.SQLite.Path = ":memory:"
	cfg.Database.SQLite.AutoMigrate = true

	// NATS configuration (will be overridden by test container)
	cfg.NATS.URL = "nats://localhost:4222"
	cfg.NATS.ClusterID = "test-cluster"
	cfg.NATS.ClientID = "test-client"
	cfg.NATS.MaxReconnects = 3
	cfg.NATS.ReconnectWait = "1s"
	cfg.NATS.Timeout = "5s"
	cfg.NATS.JetStream.Enabled = true
	cfg.NATS.KVStore.Enabled = true
	cfg.NATS.KVStore.Bucket = "test-cache"
	cfg.NATS.KVStore.TTL = "5m"

	// Provider configurations (disabled for tests)
	cfg.Providers.ArXiv.Enabled = false
	cfg.Providers.SemanticScholar.Enabled = false
	cfg.Providers.Exa.Enabled = false
	cfg.Providers.Tavily.Enabled = false

	// Logging configuration
	cfg.Logging.Level = "error" // Reduce noise in tests
	cfg.Logging.Format = "json"
	cfg.Logging.AddSource = false
	cfg.Logging.Output = "stdout"

	// Security configuration (permissive for tests)
	cfg.Security.RateLimit.Enabled = false
	cfg.Security.CORS.Enabled = true
	cfg.Security.CORS.AllowedOrigins = []string{"*"}
	cfg.Security.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	cfg.Security.CORS.AllowedHeaders = []string{"*"}

	// Circuit breaker (disabled for tests)
	cfg.Circuit.Enabled = false

	// Retry (reduced for faster tests)
	cfg.Retry.Enabled = true
	cfg.Retry.MaxAttempts = 2
	cfg.Retry.InitialDelay = "100ms"
	cfg.Retry.MaxDelay = "1s"
	cfg.Retry.BackoffFactor = 1.5
	cfg.Retry.Jitter = false

	// Monitoring
	cfg.Monitoring.Enabled = false

	return cfg
}

// TestConfigWithPostgreSQL creates a test configuration with PostgreSQL
func TestConfigWithPostgreSQL(t *testing.T, connectionString string) *config.Config {
	cfg := TestConfig(t)
	
	// Override database configuration for PostgreSQL
	cfg.Database.Type = "postgres"
	cfg.Database.PostgreSQL.DSN = connectionString
	cfg.Database.PostgreSQL.MaxConns = 5
	cfg.Database.PostgreSQL.MaxIdle = 2
	cfg.Database.PostgreSQL.MaxLifetime = "5m"
	cfg.Database.PostgreSQL.MaxIdleTime = "1m"
	cfg.Database.PostgreSQL.AutoMigrate = true

	return cfg
}

// TestConfigWithNATS creates a test configuration with NATS URL
func TestConfigWithNATS(t *testing.T, natsURL string) *config.Config {
	cfg := TestConfig(t)
	cfg.NATS.URL = natsURL
	return cfg
}

// TestConfigFromEnv creates a test configuration from environment variables
func TestConfigFromEnv(t *testing.T) *config.Config {
	// Set test environment variables
	os.Setenv("SCIFIND_SERVER_MODE", "test")
	os.Setenv("SCIFIND_DATABASE_TYPE", "sqlite")
	os.Setenv("SCIFIND_DATABASE_SQLITE_PATH", ":memory:")
	os.Setenv("SCIFIND_LOGGING_LEVEL", "error")
	
	defer func() {
		// Clean up environment variables
		os.Unsetenv("SCIFIND_SERVER_MODE")
		os.Unsetenv("SCIFIND_DATABASE_TYPE")
		os.Unsetenv("SCIFIND_DATABASE_SQLITE_PATH")
		os.Unsetenv("SCIFIND_LOGGING_LEVEL")
	}()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load test config from env: %v", err)
	}

	return cfg
}

// CreateTempConfigFile creates a temporary config file for testing
func CreateTempConfigFile(t *testing.T, content string) string {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	
	return configPath
}

// TestConfigYAML returns a test configuration in YAML format
func TestConfigYAML() string {
	return `
server:
  port: 0
  host: "localhost"
  mode: "test"
  read_timeout: "5s"
  write_timeout: "5s"
  idle_timeout: "30s"

database:
  type: "sqlite"
  sqlite:
    path: ":memory:"
    auto_migrate: true

nats:
  url: "nats://localhost:4222"
  cluster_id: "test-cluster"
  client_id: "test-client"
  max_reconnects: 3
  reconnect_wait: "1s"
  timeout: "5s"
  jetstream:
    enabled: true
  kv_store:
    enabled: true
    bucket: "test-cache"
    ttl: "5m"

providers:
  arxiv:
    enabled: false
  semantic_scholar:
    enabled: false
  exa:
    enabled: false
  tavily:
    enabled: false

logging:
  level: "error"
  format: "json"
  add_source: false
  output: "stdout"

security:
  rate_limit:
    enabled: false
  cors:
    enabled: true
    allowed_origins: ["*"]
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["*"]

circuit:
  enabled: false

retry:
  enabled: true
  max_attempts: 2
  initial_delay: "100ms"
  max_delay: "1s"
  backoff_factor: 1.5
  jitter: false

monitoring:
  enabled: false
`
}

// TestConfigWithProviders creates a test configuration with mock providers
func TestConfigWithProviders(t *testing.T, arxivURL, semanticScholarURL string) *config.Config {
	cfg := TestConfig(t)

	// Enable providers with test URLs
	cfg.Providers.ArXiv.Enabled = true
	cfg.Providers.ArXiv.BaseURL = arxivURL
	cfg.Providers.ArXiv.Timeout = "5s"
	cfg.Providers.ArXiv.RateLimit = "100ms"

	cfg.Providers.SemanticScholar.Enabled = true
	cfg.Providers.SemanticScholar.BaseURL = semanticScholarURL
	cfg.Providers.SemanticScholar.Timeout = "5s"
	cfg.Providers.SemanticScholar.APIKey = "test-api-key"

	return cfg
}

// ValidateTestConfig validates that a configuration is suitable for testing
func ValidateTestConfig(t *testing.T, cfg *config.Config) {
	// Ensure test mode
	if !cfg.IsTest() {
		t.Error("Configuration should be in test mode")
	}

	// Ensure database is SQLite or has test prefix
	if cfg.Database.Type == "postgres" {
		connStr, _ := cfg.GetDatabaseConnectionString()
		if !contains(connStr, "test") {
			t.Error("PostgreSQL connection string should contain 'test' for safety")
		}
	}

	// Ensure logging is not too verbose
	if cfg.Logging.Level == "debug" {
		t.Log("Debug logging enabled in tests may produce excessive output")
	}

	// Ensure monitoring is disabled (for test performance)
	if cfg.Monitoring.Enabled {
		t.Log("Monitoring enabled in tests may affect performance")
	}
}

// GetTestTimeoutConfig returns timeout configuration suitable for tests
func GetTestTimeoutConfig() *config.TimeoutConfig {
	return &config.TimeoutConfig{
		Default:         5000,  // 5s
		Database:        2000,  // 2s
		ExternalAPI:     3000,  // 3s
		Search:          10000, // 10s
		FileProcessing:  15000, // 15s
		HealthCheck:     1000,  // 1s
		Server: config.ServerTimeoutConfig{
			Read:  5000,  // 5s
			Write: 5000,  // 5s
			Idle:  30000, // 30s
		},
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

// Extension interface for testing framework
type TestingT interface {
	Error(args ...interface{})
	Fatalf(format string, args ...interface{})
	TempDir() string
}

// Ensure *testing.T implements TestingT
var _ TestingT = (*testing.T)(nil)