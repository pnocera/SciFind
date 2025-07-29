package embedded

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"scifind-backend/internal/config"
	"scifind-backend/internal/errors"

	"github.com/nats-io/nats-server/v2/server"
)

// EmbeddedServer wraps the NATS server and provides lifecycle management
type EmbeddedServer struct {
	server     *server.Server
	config     *config.NATSConfig
	logger     *slog.Logger
	
	// Lifecycle management
	started    bool
	mu         sync.RWMutex
	stopCh     chan struct{}
	
	// Health monitoring
	lastHealthCheck time.Time
	healthy         bool
}

// NewEmbeddedServer creates a new embedded NATS server instance
func NewEmbeddedServer(cfg *config.NATSConfig, logger *slog.Logger) (*EmbeddedServer, error) {
	if !cfg.Embedded.Enabled {
		return nil, fmt.Errorf("embedded NATS server is not enabled")
	}

	// Validate configuration
	if err := validateEmbeddedConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid embedded server configuration: %w", err)
	}

	logger.Info("Creating embedded NATS server",
		slog.String("host", cfg.Embedded.Host),
		slog.Int("port", cfg.Embedded.Port),
		slog.Bool("jetstream", cfg.JetStream.Enabled),
		slog.Bool("tls", cfg.TLS.Enabled))

	return &EmbeddedServer{
		config: cfg,
		logger: logger,
		stopCh: make(chan struct{}),
	}, nil
}

// Start starts the embedded NATS server
func (es *EmbeddedServer) Start(ctx context.Context) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.started {
		return fmt.Errorf("embedded NATS server is already started")
	}

	// Build server options
	opts, err := es.buildServerOptions()
	if err != nil {
		return fmt.Errorf("failed to build server options: %w", err)
	}

	// Create server instance
	es.logger.Info("Creating NATS server with options", 
		slog.String("host", opts.Host),
		slog.Int("port", opts.Port),
		slog.Bool("jetstream", opts.JetStream),
		slog.String("store_dir", opts.StoreDir))
	
	es.server, err = server.NewServer(opts)
	if err != nil {
		es.logger.Error("Failed to create NATS server", 
			slog.String("error", err.Error()),
			slog.String("host", es.config.Embedded.Host),
			slog.Int("port", es.config.Embedded.Port))
		return errors.NewMessagingError("failed to create NATS server", map[string]interface{}{
			"error": err.Error(),
			"host":  es.config.Embedded.Host,
			"port":  es.config.Embedded.Port,
		})
	}

	// Configure server logging
	es.configureServerLogging()

	// Start server in goroutine
	go func() {
		es.logger.Info("Starting embedded NATS server",
			slog.String("address", net.JoinHostPort(es.config.Embedded.Host, strconv.Itoa(es.config.Embedded.Port))))

		es.server.Start()
		es.logger.Info("NATS server.Start() called")
	}()

	// Wait for server to be ready for connections
	es.logger.Info("Waiting for server to be ready for connections...")
	
	// ReadyForConnections handles the waiting internally - use a reasonable timeout
	timeout := 10 * time.Second
	ready := es.server.ReadyForConnections(timeout)
	
	es.logger.Info("Server readiness check completed", 
		slog.Bool("ready", ready),
		slog.Bool("running", es.server.Running()),
		slog.Duration("timeout", timeout))
	
	if !ready {
		es.logger.Error("Server failed to become ready within timeout", 
			slog.Bool("running", es.server.Running()),
			slog.String("client_url", es.server.ClientURL()),
			slog.String("server_id", es.server.ID()),
			slog.Duration("timeout", timeout))
		
		// Return error - client should NOT connect if server is not ready
		return fmt.Errorf("embedded NATS server failed to become ready within %v", timeout)
	}

	es.started = true
	es.healthy = true
	es.lastHealthCheck = time.Now()

	// Start health monitoring
	go es.healthMonitor(ctx)

	es.logger.Info("Embedded NATS server started successfully",
		slog.String("client_url", es.server.ClientURL()),
		slog.String("server_id", es.server.ID()),
		slog.Bool("jetstream", es.config.JetStream.Enabled))

	return nil
}

// Stop stops the embedded NATS server gracefully
func (es *EmbeddedServer) Stop(ctx context.Context) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if !es.started {
		return nil
	}

	es.logger.Info("Stopping embedded NATS server")

	// Signal health monitor to stop
	close(es.stopCh)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		es.server.Shutdown()
	}()

	select {
	case <-done:
		es.logger.Info("Embedded NATS server stopped gracefully")
	case <-shutdownCtx.Done():
		es.logger.Warn("Embedded NATS server shutdown timeout, forcing stop")
		es.server.Shutdown()
	}

	es.started = false
	es.healthy = false

	es.logger.Info("Embedded NATS server shutdown complete")
	return nil
}

// IsHealthy returns true if the server is healthy
func (es *EmbeddedServer) IsHealthy() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.started && es.healthy && es.server != nil && es.server.Running()
}

// GetClientURL returns the client connection URL
func (es *EmbeddedServer) GetClientURL() string {
	if es.server == nil {
		return ""
	}
	return es.server.ClientURL()
}

// GetServerID returns the server ID
func (es *EmbeddedServer) GetServerID() string {
	if es.server == nil {
		return ""
	}
	return es.server.ID()
}

// GetStats returns server statistics
func (es *EmbeddedServer) GetStats() map[string]interface{} {
	if es.server == nil {
		return map[string]interface{}{
			"running": false,
		}
	}

	varz, _ := es.server.Varz(nil)
	
	stats := map[string]interface{}{
		"running":           es.server.Running(),
		"ready":            es.server.ReadyForConnections(0),
		"server_id":        es.server.ID(),
		"server_name":      es.server.Name(),
		"client_url":       es.server.ClientURL(),
		"connections":      varz.Connections,
		"total_connections": varz.TotalConnections,
		"routes":           varz.Routes,
		"remotes":          varz.Remotes,
		"in_msgs":          varz.InMsgs,
		"out_msgs":         varz.OutMsgs,
		"in_bytes":         varz.InBytes,
		"out_bytes":        varz.OutBytes,
		"slow_consumers":   varz.SlowConsumers,
		"subscriptions":    varz.Subscriptions,
		"http_req_stats":   varz.HTTPReqStats,
		"config_load_time": varz.ConfigLoadTime,
		"uptime":           varz.Uptime,
		"mem":              varz.Mem,
		"cores":            varz.Cores,
		"cpu":              varz.CPU,
	}

	// Add JetStream stats if enabled
	if es.config.JetStream.Enabled {
		if jsz, err := es.server.Jsz(nil); err == nil {
			stats["jetstream"] = map[string]interface{}{
				"config":   jsz.Config,
				"streams":  jsz.Streams,
				"consumers": jsz.Consumers,
				"messages": jsz.Messages,
				"bytes":    jsz.Bytes,
				"meta_cluster": jsz.Meta,
			}
		}
	}

	return stats
}

// buildServerOptions constructs server options from configuration
func (es *EmbeddedServer) buildServerOptions() (*server.Options, error) {
	es.logger.Info("Building server options", 
		slog.String("host", es.config.Embedded.Host),
		slog.Int("port", es.config.Embedded.Port))
	
	// Keep it simple - minimal configuration following NATS by Example pattern
	opts := &server.Options{
		Host:       es.config.Embedded.Host,
		Port:       es.config.Embedded.Port,
		ServerName: "scifind-embedded",
	}

	// Configure JetStream if enabled - this is the key part
	if es.config.JetStream.Enabled {
		opts.JetStream = true
		storeDir := es.config.JetStream.StoreDir
		if storeDir == "" {
			storeDir = "./jetstream"
		}
		
		// Ensure store directory exists
		if err := os.MkdirAll(storeDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create JetStream store directory: %w", err)
		}
		
		opts.StoreDir = storeDir
		es.logger.Info("JetStream enabled", slog.String("store_dir", storeDir))
	}

	return opts, nil
}

// buildTLSConfig creates TLS configuration for the server
func (es *EmbeddedServer) buildTLSConfig() (*tls.Config, error) {
	if es.config.TLS.CertFile == "" || es.config.TLS.KeyFile == "" {
		return nil, fmt.Errorf("TLS cert_file and key_file are required when TLS is enabled")
	}

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(es.config.TLS.CertFile, es.config.TLS.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Configure client authentication (mTLS) if enabled
	if es.config.TLS.ClientAuth.Enabled {
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		
		if es.config.TLS.CAFile != "" {
			// Load CA certificate for client verification
			caCert, err := os.ReadFile(es.config.TLS.CAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA file: %w", err)
			}
			
			caCertPool := tlsConfig.ClientCAs
			if caCertPool == nil {
				caCertPool = x509.NewCertPool()
			}
			
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
			
			tlsConfig.ClientCAs = caCertPool
		}
	}

	// Configure certificate verification
	if es.config.TLS.InsecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	return tlsConfig, nil
}

// configureJetStream sets up JetStream options
func (es *EmbeddedServer) configureJetStream(opts *server.Options) error {
	storeDir := es.config.JetStream.StoreDir
	if storeDir == "" {
		storeDir = "./jetstream"
	}

	// Ensure store directory exists
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		es.logger.Error("Failed to create JetStream store directory", 
			slog.String("dir", storeDir), 
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to create JetStream store directory: %w", err)
	}

	es.logger.Info("Configuring JetStream", slog.String("store_dir", storeDir))
	opts.JetStream = true
	opts.StoreDir = storeDir

	// Parse memory and storage limits
	if es.config.JetStream.MaxMemory != "" {
		if maxMem, err := parseByteSize(es.config.JetStream.MaxMemory); err == nil {
			opts.JetStreamMaxMemory = maxMem
		}
	}

	if es.config.JetStream.MaxStorage != "" {
		if maxStore, err := parseByteSize(es.config.JetStream.MaxStorage); err == nil {
			opts.JetStreamMaxStore = maxStore
		}
	}

	// Configure sync interval
	if es.config.JetStream.SyncInterval != "" {
		if syncInterval, err := time.ParseDuration(es.config.JetStream.SyncInterval); err == nil {
			opts.SyncInterval = syncInterval
		}
	}

	return nil
}

// configureCluster sets up cluster options
func (es *EmbeddedServer) configureCluster(opts *server.Options) {
	opts.Cluster.Host = es.config.Embedded.Cluster.Host
	opts.Cluster.Port = es.config.Embedded.Cluster.Port
	opts.Cluster.Name = es.config.Embedded.Cluster.Name

	// Add cluster routes
	if len(es.config.Embedded.Cluster.Routes) > 0 {
		for _, route := range es.config.Embedded.Cluster.Routes {
			if u, err := parseClusterURL(route); err == nil {
				opts.Routes = append(opts.Routes, u)
			}
		}
	}
}

// configureServerLogging sets up server logging integration
func (es *EmbeddedServer) configureServerLogging() {
	if es.server == nil {
		return
	}

	// Set log file if specified
	if es.config.Embedded.LogFile != "" {
		// Ensure log directory exists
		if dir := filepath.Dir(es.config.Embedded.LogFile); dir != "." {
			os.MkdirAll(dir, 0755)
		}
	}

	// The NATS server handles its own logging, but we can intercept it
	// For now, we'll let it use its default logging and rely on our application logs
}

// healthMonitor continuously monitors server health
func (es *EmbeddedServer) healthMonitor(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-es.stopCh:
			return
		case <-ticker.C:
			es.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check on the server
func (es *EmbeddedServer) performHealthCheck() {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.server == nil {
		es.healthy = false
		return
	}

	// Check if server is running and ready
	running := es.server.Running()
	ready := es.server.ReadyForConnections(1 * time.Second)

	es.healthy = running && ready
	es.lastHealthCheck = time.Now()

	if !es.healthy {
		es.logger.Error("Embedded NATS server health check failed",
			slog.Bool("running", running),
			slog.Bool("ready", ready))
	} else {
		es.logger.Debug("Embedded NATS server health check passed",
			slog.Bool("running", running),
			slog.Bool("ready", ready))
	}
}

// getClusterAddress returns the cluster address if clustering is enabled
func (es *EmbeddedServer) getClusterAddress() string {
	if es.config.Embedded.Cluster.Port <= 0 {
		return "not configured"
	}
	return net.JoinHostPort(es.config.Embedded.Cluster.Host, strconv.Itoa(es.config.Embedded.Cluster.Port))
}

// Utility functions

// validateEmbeddedConfig validates the embedded server configuration
func validateEmbeddedConfig(cfg *config.NATSConfig) error {
	if cfg.Embedded.Host == "" {
		return fmt.Errorf("embedded.host is required")
	}

	if cfg.Embedded.Port <= 0 || cfg.Embedded.Port > 65535 {
		return fmt.Errorf("embedded.port must be between 1 and 65535")
	}

	// Validate TLS configuration if enabled
	if cfg.TLS.Enabled {
		if cfg.TLS.CertFile == "" {
			return fmt.Errorf("tls.cert_file is required when TLS is enabled")
		}
		if cfg.TLS.KeyFile == "" {
			return fmt.Errorf("tls.key_file is required when TLS is enabled")
		}

		// Check if certificate files exist
		if _, err := os.Stat(cfg.TLS.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("tls.cert_file does not exist: %s", cfg.TLS.CertFile)
		}
		if _, err := os.Stat(cfg.TLS.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("tls.key_file does not exist: %s", cfg.TLS.KeyFile)
		}

		// Validate mTLS configuration
		if cfg.TLS.ClientAuth.Enabled {
			if cfg.TLS.CAFile == "" {
				return fmt.Errorf("tls.ca_file is required when client authentication is enabled")
			}
			if _, err := os.Stat(cfg.TLS.CAFile); os.IsNotExist(err) {
				return fmt.Errorf("tls.ca_file does not exist: %s", cfg.TLS.CAFile)
			}
		}
	}

	// Validate JetStream configuration if enabled
	if cfg.JetStream.Enabled {
		if cfg.JetStream.StoreDir != "" {
			// Check if we can create the store directory
			if err := os.MkdirAll(cfg.JetStream.StoreDir, 0755); err != nil {
				return fmt.Errorf("cannot create JetStream store directory: %w", err)
			}
		}
	}

	return nil
}

// parseByteSize parses size strings like "1MB", "1GB", etc.
func parseByteSize(size string) (int64, error) {
	// This is a simplified parser - in production you might want to use a library
	// or implement a more robust parser
	if size == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Simple conversion map
	units := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	// Find the unit suffix
	var number string
	var unit string
	
	for unitSuffix := range units {
		if len(size) > len(unitSuffix) && size[len(size)-len(unitSuffix):] == unitSuffix {
			number = size[:len(size)-len(unitSuffix)]
			unit = unitSuffix
			break
		}
	}

	if unit == "" {
		// No unit found, assume bytes
		number = size
		unit = "B"
	}

	// Parse the number
	if num, err := strconv.ParseInt(number, 10, 64); err == nil {
		return num * units[unit], nil
	}

	return 0, fmt.Errorf("invalid size format: %s", size)
}

// parseClusterURL parses a cluster route URL
func parseClusterURL(route string) (*url.URL, error) {
	// Add nats:// scheme if not present
	if !strings.Contains(route, "://") {
		route = "nats://" + route
	}
	return url.Parse(route)
}