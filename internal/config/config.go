package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Server struct {
		Port           int           `mapstructure:"port" validate:"min=1,max=65535"`
		Host           string        `mapstructure:"host"`
		Mode           string        `mapstructure:"mode" validate:"oneof=debug release test"`
		ReadTimeout    string `mapstructure:"read_timeout"`
		WriteTimeout   string `mapstructure:"write_timeout"`
		IdleTimeout    string `mapstructure:"idle_timeout"`
		MaxHeaderBytes int           `mapstructure:"max_header_bytes"`
		EnableGzip     bool          `mapstructure:"enable_gzip"`
		EnableCORS     bool          `mapstructure:"enable_cors"`
		EnableMetrics  bool          `mapstructure:"enable_metrics"`
	} `mapstructure:"server"`

	Database struct {
		Type       string `mapstructure:"type" validate:"oneof=postgres sqlite"`
		PostgreSQL struct {
			DSN          string `mapstructure:"dsn"`
			MaxConns     int    `mapstructure:"max_connections" validate:"min=1"`
			MaxIdle      int    `mapstructure:"max_idle" validate:"min=1"`
			MaxLifetime  string `mapstructure:"max_lifetime"`
			MaxIdleTime  string `mapstructure:"max_idle_time"`
			AutoMigrate  bool   `mapstructure:"auto_migrate"`
		} `mapstructure:"postgresql"`
		SQLite struct {
			Path        string `mapstructure:"path"`
			AutoMigrate bool   `mapstructure:"auto_migrate"`
		} `mapstructure:"sqlite"`
	} `mapstructure:"database"`

	NATS NATSConfig `mapstructure:"nats"`

	Providers struct {
		ArXiv struct {
			Enabled   bool   `mapstructure:"enabled"`
			BaseURL   string `mapstructure:"base_url"`
			RateLimit string `mapstructure:"rate_limit"`
			Timeout   string `mapstructure:"timeout"`
		} `mapstructure:"arxiv"`

		SemanticScholar struct {
			Enabled bool   `mapstructure:"enabled"`
			APIKey  string `mapstructure:"api_key"`
			BaseURL string `mapstructure:"base_url"`
			Timeout string `mapstructure:"timeout"`
		} `mapstructure:"semantic_scholar"`

		Exa struct {
			Enabled bool   `mapstructure:"enabled"`
			APIKey  string `mapstructure:"api_key"`
			BaseURL string `mapstructure:"base_url"`
			Timeout string `mapstructure:"timeout"`
		} `mapstructure:"exa"`

		Tavily struct {
			Enabled bool   `mapstructure:"enabled"`
			APIKey  string `mapstructure:"api_key"`
			BaseURL string `mapstructure:"base_url"`
			Timeout string `mapstructure:"timeout"`
		} `mapstructure:"tavily"`
	} `mapstructure:"providers"`

	Logging struct {
		Level     string `mapstructure:"level" validate:"oneof=debug info warn error"`
		Format    string `mapstructure:"format" validate:"oneof=json text"`
		AddSource bool   `mapstructure:"add_source"`
		Output    string `mapstructure:"output" validate:"oneof=stdout stderr file"`
		FilePath  string `mapstructure:"file_path"`
	} `mapstructure:"logging"`

	Security struct {
		APIKeys      []string `mapstructure:"api_keys"`
		RateLimit struct {
			Enabled    bool   `mapstructure:"enabled"`
			Requests   int    `mapstructure:"requests"`
			Window     string `mapstructure:"window"`
			BurstSize  int    `mapstructure:"burst_size"`
		} `mapstructure:"rate_limit"`
		CORS struct {
			Enabled        bool     `mapstructure:"enabled"`
			AllowedOrigins []string `mapstructure:"allowed_origins"`
			AllowedMethods []string `mapstructure:"allowed_methods"`
			AllowedHeaders []string `mapstructure:"allowed_headers"`
			MaxAge         string   `mapstructure:"max_age"`
		} `mapstructure:"cors"`
	} `mapstructure:"security"`

	Circuit struct {
		Enabled           bool   `mapstructure:"enabled"`
		FailureThreshold  int    `mapstructure:"failure_threshold"`
		SuccessThreshold  int    `mapstructure:"success_threshold"`
		Timeout           string `mapstructure:"timeout"`
		MaxRequests       int    `mapstructure:"max_requests"`
		SlidingWindow     string `mapstructure:"sliding_window"`
		MinRequestCount   int    `mapstructure:"min_request_count"`
	} `mapstructure:"circuit"`

	Retry struct {
		Enabled       bool   `mapstructure:"enabled"`
		MaxAttempts   int    `mapstructure:"max_attempts"`
		InitialDelay  string `mapstructure:"initial_delay"`
		MaxDelay      string `mapstructure:"max_delay"`
		BackoffFactor float64 `mapstructure:"backoff_factor"`
		Jitter        bool   `mapstructure:"jitter"`
	} `mapstructure:"retry"`

	Monitoring struct {
		Enabled    bool   `mapstructure:"enabled"`
		MetricsPort int   `mapstructure:"metrics_port"`
		HealthPath string `mapstructure:"health_path"`
		MetricsPath string `mapstructure:"metrics_path"`
	} `mapstructure:"monitoring"`
}

// TimeoutConfig contains parsed timeout durations
type TimeoutConfig struct {
	Default         time.Duration
	Database        time.Duration
	ExternalAPI     time.Duration
	Search          time.Duration
	FileProcessing  time.Duration
	HealthCheck     time.Duration
	Server          ServerTimeoutConfig
}

type ServerTimeoutConfig struct {
	Read  time.Duration
	Write time.Duration
	Idle  time.Duration
}

// LoadConfig loads configuration from environment variables and config files
func LoadConfig() (*Config, error) {
	return LoadConfigFromPath("configs/config.yaml")
}

// LoadConfigFromPath loads configuration from a specific path
func LoadConfigFromPath(configPath string) (*Config, error) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
	}
	
	// Set environment variable prefix
	viper.SetEnvPrefix("SCIFIND")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaults()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// GetTimeoutConfig returns parsed timeout configurations
func (c *Config) GetTimeoutConfig() (*TimeoutConfig, error) {
	serverRead, err := time.ParseDuration(c.Server.ReadTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid server read timeout: %w", err)
	}

	serverWrite, err := time.ParseDuration(c.Server.WriteTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid server write timeout: %w", err)
	}

	serverIdle, err := time.ParseDuration(c.Server.IdleTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid server idle timeout: %w", err)
	}

	return &TimeoutConfig{
		Default:         30 * time.Second,
		Database:        5 * time.Second,
		ExternalAPI:     15 * time.Second,
		Search:          30 * time.Second,
		FileProcessing:  60 * time.Second,
		HealthCheck:     5 * time.Second,
		Server: ServerTimeoutConfig{
			Read:  serverRead,
			Write: serverWrite,
			Idle:  serverIdle,
		},
	}, nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Mode == "debug"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Mode == "release"
}

// IsTest returns true if running in test mode
func (c *Config) IsTest() bool {
	return c.Server.Mode == "test"
}

// GetDatabaseConnectionString returns the appropriate database connection string
func (c *Config) GetDatabaseConnectionString() (string, error) {
	switch c.Database.Type {
	case "postgres":
		if c.Database.PostgreSQL.DSN == "" {
			return "", fmt.Errorf("PostgreSQL DSN is required when type is postgres")
		}
		return c.Database.PostgreSQL.DSN, nil
	case "sqlite":
		if c.Database.SQLite.Path == "" {
			return "", fmt.Errorf("SQLite path is required when type is sqlite")
		}
		return c.Database.SQLite.Path, nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", c.Database.Type)
	}
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")

	// Database defaults
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.postgresql.max_connections", 25)
	viper.SetDefault("database.postgresql.max_idle", 10)
	viper.SetDefault("database.postgresql.max_lifetime", "1h")
	viper.SetDefault("database.postgresql.max_idle_time", "30m")
	viper.SetDefault("database.postgresql.auto_migrate", true)
	viper.SetDefault("database.sqlite.path", "./scifind.db")
	viper.SetDefault("database.sqlite.auto_migrate", true)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.cluster_id", "scifind-cluster")
	viper.SetDefault("nats.client_id", "scifind-backend")
	viper.SetDefault("nats.max_reconnects", 10)
	viper.SetDefault("nats.reconnect_wait", "2s")
	viper.SetDefault("nats.timeout", "5s")
	
	// Embedded NATS server defaults
	viper.SetDefault("nats.embedded.enabled", false)
	viper.SetDefault("nats.embedded.host", "0.0.0.0")
	viper.SetDefault("nats.embedded.port", 4222)
	viper.SetDefault("nats.embedded.log_level", "INFO")
	viper.SetDefault("nats.embedded.log_file", "")
	viper.SetDefault("nats.embedded.cluster.name", "scifind-cluster")
	viper.SetDefault("nats.embedded.cluster.host", "0.0.0.0")
	viper.SetDefault("nats.embedded.cluster.port", 6222)
	viper.SetDefault("nats.embedded.cluster.routes", []string{})
	viper.SetDefault("nats.embedded.gateway.name", "scifind-gateway")
	viper.SetDefault("nats.embedded.gateway.host", "0.0.0.0")
	viper.SetDefault("nats.embedded.gateway.port", 7222)
	viper.SetDefault("nats.embedded.monitor.host", "0.0.0.0")
	viper.SetDefault("nats.embedded.monitor.port", 8222)
	viper.SetDefault("nats.embedded.accounts.system_account", "$SYS")
	viper.SetDefault("nats.embedded.limits.max_connections", 10000)
	viper.SetDefault("nats.embedded.limits.max_payload", "1MB")
	viper.SetDefault("nats.embedded.limits.max_pending", "64MB")
	
	// NATS TLS defaults
	viper.SetDefault("nats.tls.enabled", false)
	viper.SetDefault("nats.tls.cert_file", "")
	viper.SetDefault("nats.tls.key_file", "")
	viper.SetDefault("nats.tls.ca_file", "")
	viper.SetDefault("nats.tls.verify_and_map", false)
	viper.SetDefault("nats.tls.insecure_skip_verify", false)
	viper.SetDefault("nats.tls.client_auth.enabled", false)
	viper.SetDefault("nats.tls.client_auth.cert_file", "")
	viper.SetDefault("nats.tls.client_auth.key_file", "")
	
	// JetStream defaults
	viper.SetDefault("nats.jetstream.enabled", true)
	viper.SetDefault("nats.jetstream.domain", "")
	viper.SetDefault("nats.jetstream.store_dir", "./jetstream")
	viper.SetDefault("nats.jetstream.max_memory", "1GB")
	viper.SetDefault("nats.jetstream.max_storage", "10GB")
	viper.SetDefault("nats.jetstream.sync_interval", "2m")
	
	// Key-Value store defaults
	viper.SetDefault("nats.kv_store.enabled", true)
	viper.SetDefault("nats.kv_store.bucket", "scifind-cache")
	viper.SetDefault("nats.kv_store.ttl", "1h")
	
	// Object store defaults
	viper.SetDefault("nats.object_store.enabled", true)
	viper.SetDefault("nats.object_store.bucket", "scifind-objects")

	// Provider defaults
	viper.SetDefault("providers.arxiv.enabled", true)
	viper.SetDefault("providers.arxiv.base_url", "http://export.arxiv.org/api/query")
	viper.SetDefault("providers.arxiv.rate_limit", "3s")
	viper.SetDefault("providers.arxiv.timeout", "30s")
	
	viper.SetDefault("providers.semantic_scholar.enabled", true)
	viper.SetDefault("providers.semantic_scholar.base_url", "https://api.semanticscholar.org/graph/v1")
	viper.SetDefault("providers.semantic_scholar.timeout", "15s")
	
	viper.SetDefault("providers.exa.enabled", false)
	viper.SetDefault("providers.exa.base_url", "https://api.exa.ai")
	viper.SetDefault("providers.exa.timeout", "15s")
	
	viper.SetDefault("providers.tavily.enabled", false)
	viper.SetDefault("providers.tavily.base_url", "https://api.tavily.com")
	viper.SetDefault("providers.tavily.timeout", "15s")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.add_source", false)
	viper.SetDefault("logging.output", "stdout")

	// Security defaults
	viper.SetDefault("security.rate_limit.enabled", true)
	viper.SetDefault("security.rate_limit.requests", 100)
	viper.SetDefault("security.rate_limit.window", "1m")
	viper.SetDefault("security.rate_limit.burst_size", 10)
	viper.SetDefault("security.cors.enabled", true)
	viper.SetDefault("security.cors.allowed_origins", []string{"*"})
	viper.SetDefault("security.cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("security.cors.allowed_headers", []string{"*"})
	viper.SetDefault("security.cors.max_age", "12h")

	// Circuit breaker defaults
	viper.SetDefault("circuit.enabled", true)
	viper.SetDefault("circuit.failure_threshold", 5)
	viper.SetDefault("circuit.success_threshold", 3)
	viper.SetDefault("circuit.timeout", "60s")
	viper.SetDefault("circuit.max_requests", 10)
	viper.SetDefault("circuit.sliding_window", "60s")
	viper.SetDefault("circuit.min_request_count", 10)

	// Retry defaults
	viper.SetDefault("retry.enabled", true)
	viper.SetDefault("retry.max_attempts", 3)
	viper.SetDefault("retry.initial_delay", "1s")
	viper.SetDefault("retry.max_delay", "30s")
	viper.SetDefault("retry.backoff_factor", 2.0)
	viper.SetDefault("retry.jitter", true)

	// Monitoring defaults
	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.metrics_port", 9090)
	viper.SetDefault("monitoring.health_path", "/health")
	viper.SetDefault("monitoring.metrics_path", "/metrics")
}

// TLSConfig represents TLS configuration  
type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CertFile          string `mapstructure:"cert_file"`
	KeyFile           string `mapstructure:"key_file"`
	CAFile            string `mapstructure:"ca_file"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	ServerName         string `mapstructure:"server_name"`
}

// NATSConfig represents NATS configuration
type NATSConfig struct {
	URL             string   `mapstructure:"url" validate:"required,url"`
	ClusterID       string   `mapstructure:"cluster_id"`
	ClientID        string   `mapstructure:"client_id"`
	Subjects        []string `mapstructure:"subjects"`
	MaxReconnects   int      `mapstructure:"max_reconnects"`
	ReconnectWait   string   `mapstructure:"reconnect_wait"`
	Timeout         string   `mapstructure:"timeout"`
	Username        string   `mapstructure:"username"`
	Password        string   `mapstructure:"password"`
	Token           string   `mapstructure:"token"`
	PingInterval    int      `mapstructure:"ping_interval"` // seconds
	MaxPingsOut     int      `mapstructure:"max_pings_out"`
	
	// Embedded server configuration
	Embedded struct {
		Enabled    bool   `mapstructure:"enabled"`
		Host       string `mapstructure:"host"`
		Port       int    `mapstructure:"port"`
		LogLevel   string `mapstructure:"log_level"`
		LogFile    string `mapstructure:"log_file"`
		
		// Clustering configuration
		Cluster struct {
			Name   string   `mapstructure:"name"`
			Host   string   `mapstructure:"host"`
			Port   int      `mapstructure:"port"`
			Routes []string `mapstructure:"routes"`
		} `mapstructure:"cluster"`
		
		// Gateway configuration for super clusters
		Gateway struct {
			Name string   `mapstructure:"name"`
			Host string   `mapstructure:"host"`
			Port int      `mapstructure:"port"`
		} `mapstructure:"gateway"`
		
		// Monitoring configuration
		Monitor struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port"`
		} `mapstructure:"monitor"`
		
		// Accounts configuration
		Accounts struct {
			SystemAccount string `mapstructure:"system_account"`
		} `mapstructure:"accounts"`
		
		// Resource limits
		Limits struct {
			MaxConnections int    `mapstructure:"max_connections"`
			MaxPayload     string `mapstructure:"max_payload"`
			MaxPending     string `mapstructure:"max_pending"`
		} `mapstructure:"limits"`
	} `mapstructure:"embedded"`
	
	// TLS configuration
	TLS struct {
		Enabled            bool   `mapstructure:"enabled"`
		CertFile          string `mapstructure:"cert_file"`
		KeyFile           string `mapstructure:"key_file"`
		CAFile            string `mapstructure:"ca_file"`
		VerifyAndMap      bool   `mapstructure:"verify_and_map"`
		InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
		CertStore         string `mapstructure:"cert_store"`
		CertStoreType     string `mapstructure:"cert_store_type"`
		
		// mTLS (Mutual TLS) configuration
		ClientAuth struct {
			Enabled  bool   `mapstructure:"enabled"`
			CertFile string `mapstructure:"cert_file"`
			KeyFile  string `mapstructure:"key_file"`
		} `mapstructure:"client_auth"`
	} `mapstructure:"tls"`
	
	// JetStream configuration
	JetStream struct {
		Enabled       bool   `mapstructure:"enabled"`
		Domain        string `mapstructure:"domain"`
		StoreDir      string `mapstructure:"store_dir"`
		MaxMemory     string `mapstructure:"max_memory"`
		MaxStorage    string `mapstructure:"max_storage"`
		SyncInterval  string `mapstructure:"sync_interval"`
	} `mapstructure:"jetstream"`
	
	// Key-Value store configuration
	KVStore struct {
		Enabled  bool   `mapstructure:"enabled"`
		Bucket   string `mapstructure:"bucket"`
		TTL      string `mapstructure:"ttl"`
	} `mapstructure:"kv_store"`
	
	// Object store configuration
	ObjectStore struct {
		Enabled bool   `mapstructure:"enabled"`
		Bucket  string `mapstructure:"bucket"`
	} `mapstructure:"object_store"`
}