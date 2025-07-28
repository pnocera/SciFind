package embedded

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"scifind-backend/internal/config"
	"scifind-backend/internal/messaging"
)

// Manager manages both the embedded NATS server and client coordination
type Manager struct {
	server     *EmbeddedServer
	client     *messaging.Client
	msgManager *messaging.Manager
	config     *config.NATSConfig
	logger     *slog.Logger

	// Lifecycle management
	started bool
	mu      sync.RWMutex
}

// NewManager creates a new embedded NATS manager
func NewManager(cfg *config.NATSConfig, logger *slog.Logger) (*Manager, error) {
	manager := &Manager{
		config: cfg,
		logger: logger,
	}

	// Create embedded server if enabled
	if cfg.Embedded.Enabled {
		server, err := NewEmbeddedServer(cfg, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedded server: %w", err)
		}
		manager.server = server
	}

	return manager, nil
}

// Start starts the embedded server (if enabled) and then the client
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("embedded manager is already started")
	}

	// Start embedded server first if enabled
	if m.server != nil {
		m.logger.Info("Starting embedded NATS server")
		if err := m.server.Start(ctx); err != nil {
			return fmt.Errorf("failed to start embedded server: %w", err)
		}

		// Update client URL to use the embedded server
		m.config.URL = m.server.GetClientURL()
		m.logger.Info("Updated client URL to use embedded server",
			slog.String("url", m.config.URL))
	}

	// Give the server a moment to fully initialize
	if m.server != nil {
		time.Sleep(500 * time.Millisecond)
		
		// Verify server is ready
		if !m.server.IsHealthy() {
			return fmt.Errorf("embedded server is not healthy after startup")
		}
	}

	// Create messaging client
	client, err := messaging.NewClient(*m.config, m.logger)
	if err != nil {
		// If we started the embedded server, clean it up
		if m.server != nil {
			m.server.Stop(ctx)
		}
		return fmt.Errorf("failed to create messaging client: %w", err)
	}
	m.client = client

	// Create messaging manager
	msgManager, err := messaging.NewManager(m.config, m.logger)
	if err != nil {
		// Cleanup on failure
		m.client.Close()
		if m.server != nil {
			m.server.Stop(ctx)
		}
		return fmt.Errorf("failed to create messaging manager: %w", err)
	}
	m.msgManager = msgManager

	// Start messaging manager
	if err := m.msgManager.Start(ctx); err != nil {
		// Cleanup on failure
		m.client.Close()
		if m.server != nil {
			m.server.Stop(ctx)
		}
		return fmt.Errorf("failed to start messaging manager: %w", err)
	}

	m.started = true

	m.logger.Info("Embedded NATS manager started successfully",
		slog.Bool("embedded_server", m.server != nil),
		slog.String("client_url", m.config.URL),
		slog.Bool("jetstream", m.config.JetStream.Enabled))

	return nil
}

// Stop stops the messaging manager, client, and embedded server (if running)
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	m.logger.Info("Stopping embedded NATS manager")

	var errors []error

	// Stop messaging manager first
	if m.msgManager != nil {
		if err := m.msgManager.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop messaging manager: %w", err))
		}
	}

	// Close client connection
	if m.client != nil {
		if err := m.client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close client: %w", err))
		}
	}

	// Stop embedded server last
	if m.server != nil {
		if err := m.server.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop embedded server: %w", err))
		}
	}

	m.started = false

	if len(errors) > 0 {
		// Log all errors
		for _, err := range errors {
			m.logger.Error("Error during shutdown", slog.String("error", err.Error()))
		}
		return fmt.Errorf("multiple errors during shutdown: %d errors occurred", len(errors))
	}

	m.logger.Info("Embedded NATS manager stopped successfully")
	return nil
}

// IsHealthy returns true if the system is healthy
func (m *Manager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return false
	}

	// Check embedded server health if running
	if m.server != nil && !m.server.IsHealthy() {
		return false
	}

	// Check messaging manager health
	if m.msgManager != nil && !m.msgManager.IsHealthy() {
		return false
	}

	return true
}

// GetClient returns the messaging client
func (m *Manager) GetClient() *messaging.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client
}

// GetManager returns the messaging manager
func (m *Manager) GetManager() *messaging.Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.msgManager
}

// GetEmbeddedServer returns the embedded server (may be nil)
func (m *Manager) GetEmbeddedServer() *EmbeddedServer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.server
}

// IsEmbeddedServerEnabled returns true if embedded server is enabled
func (m *Manager) IsEmbeddedServerEnabled() bool {
	return m.server != nil
}

// GetStats returns comprehensive statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"started":         m.started,
		"healthy":         m.IsHealthy(),
		"embedded_server": m.server != nil,
	}

	// Add embedded server stats
	if m.server != nil {
		stats["server"] = m.server.GetStats()
	}

	// Add messaging manager stats
	if m.msgManager != nil {
		stats["messaging"] = m.msgManager.GetStats()
	}

	return stats
}

// Ping performs a comprehensive health check
func (m *Manager) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.started {
		return fmt.Errorf("embedded manager is not started")
	}

	// Check embedded server health
	if m.server != nil && !m.server.IsHealthy() {
		return fmt.Errorf("embedded server is not healthy")
	}

	// Check messaging manager health
	if m.msgManager != nil {
		if err := m.msgManager.Ping(ctx); err != nil {
			return fmt.Errorf("messaging manager ping failed: %w", err)
		}
	}

	return nil
}

// SetupDefaultHandlers sets up default event handlers
func (m *Manager) SetupDefaultHandlers(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.msgManager != nil {
		return m.msgManager.SetupDefaultHandlers(ctx)
	}

	return nil
}

// Publisher returns the event publisher from the messaging manager
func (m *Manager) Publisher() *messaging.EventPublisher {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.msgManager != nil {
		return m.msgManager.Publisher()
	}
	return nil
}

// Subscriber returns the event subscriber from the messaging manager
func (m *Manager) Subscriber() *messaging.EventSubscriber {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.msgManager != nil {
		return m.msgManager.Subscriber()
	}
	return nil
}

// IsConnected returns true if the client is connected
func (m *Manager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.client != nil {
		return m.client.IsConnected()
	}
	return false
}

// Close is an alias for Stop to maintain interface compatibility
func (m *Manager) Close() error {
	return m.Stop(context.Background())
}