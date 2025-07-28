package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"scifind-backend/internal/config"
	"scifind-backend/internal/errors"
)

// Manager manages the messaging system lifecycle
type Manager struct {
	client     *Client
	publisher  *EventPublisher
	subscriber *EventSubscriber
	config     *config.NATSConfig
	logger     *slog.Logger

	// Lifecycle management
	started bool
	mu      sync.RWMutex
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewManager creates a new messaging manager
func NewManager(cfg *config.NATSConfig, logger *slog.Logger) (*Manager, error) {
	client, err := NewClient(*cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	publisher := NewEventPublisher(client, logger)
	subscriber := NewEventSubscriber(client, logger)

	return &Manager{
		client:     client,
		publisher:  publisher,
		subscriber: subscriber,
		config:     cfg,
		logger:     logger,
		stopCh:     make(chan struct{}),
	}, nil
}

// Start starts the messaging manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("messaging manager already started")
	}

	// Verify connection
	if !m.client.IsConnected() {
		return errors.NewMessagingError("NATS client is not connected", nil)
	}

	// Start health monitoring
	m.wg.Add(1)
	go m.healthMonitor(ctx)

	// Start metrics collection (always enabled for now)
	m.wg.Add(1)
	go m.metricsCollector(ctx)

	m.started = true

	m.logger.Info("Messaging manager started",
		slog.String("url", m.client.ConnectedURL()),
		slog.Bool("metrics_enabled", true))

	return nil
}

// Stop stops the messaging manager
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	m.logger.Info("Stopping messaging manager")

	// Signal goroutines to stop
	close(m.stopCh)

	// Wait for goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Debug("All messaging goroutines stopped")
	case <-time.After(30 * time.Second):
		m.logger.Warn("Timeout waiting for messaging goroutines to stop")
	}

	// Unsubscribe from all subjects
	if err := m.subscriber.UnsubscribeAll(); err != nil {
		m.logger.Error("Failed to unsubscribe from all subjects", slog.String("error", err.Error()))
	}

	// Drain and close connection
	if err := m.client.Drain(); err != nil {
		m.logger.Error("Failed to drain NATS connection", slog.String("error", err.Error()))
	}

	if err := m.client.Close(); err != nil {
		m.logger.Error("Failed to close NATS connection", slog.String("error", err.Error()))
	}

	m.started = false

	m.logger.Info("Messaging manager stopped")
	return nil
}

// Publisher returns the event publisher
func (m *Manager) Publisher() *EventPublisher {
	return m.publisher
}

// Subscriber returns the event subscriber
func (m *Manager) Subscriber() *EventSubscriber {
	return m.subscriber
}

// Client returns the underlying NATS client
func (m *Manager) Client() *Client {
	return m.client
}

// IsHealthy returns true if the messaging system is healthy
func (m *Manager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.started && m.client.IsConnected()
}

// GetStats returns messaging statistics
func (m *Manager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Connection stats
	natsStats := m.client.Stats()
	stats["connection"] = map[string]interface{}{
		"connected":     m.client.IsConnected(),
		"connected_url": m.client.ConnectedURL(),
		"in_msgs":       natsStats.InMsgs,
		"out_msgs":      natsStats.OutMsgs,
		"in_bytes":      natsStats.InBytes,
		"out_bytes":     natsStats.OutBytes,
		"reconnects":    natsStats.Reconnects,
	}

	// Subscription stats
	stats["subscriptions"] = m.subscriber.GetSubscriptionInfo()

	// Manager status
	stats["manager"] = map[string]interface{}{
		"started": m.started,
		"healthy": m.IsHealthy(),
	}

	return stats
}

// Ping performs a health check
func (m *Manager) Ping(ctx context.Context) error {
	if !m.IsHealthy() {
		return errors.NewHealthCheckError("messaging system is not healthy", "messaging")
	}

	// Try to publish a test message
	testSubject := "health.ping"
	testData := map[string]interface{}{
		"timestamp": time.Now().UnixMilli(),
		"source":    "messaging_manager",
	}

	if err := m.client.Publish(ctx, testSubject, testData); err != nil {
		return errors.NewHealthCheckError("messaging publish failed: " + err.Error(), "messaging")
	}

	return nil
}

// SetupDefaultHandlers sets up default event handlers for system monitoring
func (m *Manager) SetupDefaultHandlers(ctx context.Context) error {
	// Handle system notifications for logging
	if err := m.subscriber.OnSystemNotification(ctx, m.handleSystemNotification); err != nil {
		return fmt.Errorf("failed to setup system notification handler: %w", err)
	}

	// Handle health check events
	if err := m.subscriber.OnHealthCheck(ctx, m.handleHealthCheck); err != nil {
		return fmt.Errorf("failed to setup health check handler: %w", err)
	}

	// Handle search analytics for monitoring
	if err := m.subscriber.OnSearchCompleted(ctx, m.handleSearchCompleted); err != nil {
		return fmt.Errorf("failed to setup search completed handler: %w", err)
	}

	// Handle paper processing events for monitoring
	if err := m.subscriber.OnPaperProcessing(ctx, m.handlePaperProcessing); err != nil {
		return fmt.Errorf("failed to setup paper processing handler: %w", err)
	}

	m.logger.Info("Default event handlers setup completed")
	return nil
}

// healthMonitor periodically checks system health
func (m *Manager) healthMonitor(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			start := time.Now()
			err := m.Ping(ctx)
			duration := time.Since(start)

			status := "healthy"
			if err != nil {
				status = "unhealthy"
				m.logger.Error("Messaging health check failed",
					slog.String("error", err.Error()),
					slog.Duration("duration", duration))
			}

			// Publish health check event
			if publishErr := m.publisher.PublishHealthCheck(ctx, "messaging", status, duration, err, nil); publishErr != nil {
				m.logger.Error("Failed to publish health check event", slog.String("error", publishErr.Error()))
			}
		}
	}
}

// metricsCollector periodically collects and publishes metrics
func (m *Manager) metricsCollector(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.collectAndPublishMetrics(ctx)
		}
	}
}

// collectAndPublishMetrics collects and publishes messaging metrics
func (m *Manager) collectAndPublishMetrics(ctx context.Context) {
	stats := m.client.Stats()

	metrics := []struct {
		name   string
		value  float64
		labels map[string]string
	}{
		{"nats_messages_in_total", float64(stats.InMsgs), map[string]string{"direction": "in"}},
		{"nats_messages_out_total", float64(stats.OutMsgs), map[string]string{"direction": "out"}},
		{"nats_bytes_in_total", float64(stats.InBytes), map[string]string{"direction": "in"}},
		{"nats_bytes_out_total", float64(stats.OutBytes), map[string]string{"direction": "out"}},
		{"nats_reconnects_total", float64(stats.Reconnects), nil},
	}

	for _, metric := range metrics {
		if err := m.publisher.PublishMetrics(ctx, metric.name, "counter", "messaging", metric.value, metric.labels); err != nil {
			m.logger.Error("Failed to publish metric",
				slog.String("metric", metric.name),
				slog.String("error", err.Error()))
		}
	}
}

// Default event handlers

func (m *Manager) handleSystemNotification(event *SystemNotificationEvent) error {
	level := slog.LevelInfo
	switch event.Severity {
	case "low":
		level = slog.LevelDebug
	case "medium":
		level = slog.LevelWarn
	case "high", "critical":
		level = slog.LevelError
	}

	m.logger.Log(context.Background(), level, "System notification received",
		slog.String("id", event.ID),
		slog.String("type", event.Type),
		slog.String("title", event.Title),
		slog.String("message", event.Message),
		slog.String("component", event.Component),
		slog.String("severity", event.Severity))

	return nil
}

func (m *Manager) handleHealthCheck(event *HealthCheckEvent) error {
	if event.Status != "healthy" {
		m.logger.Warn("Component health check failed",
			slog.String("component", event.Component),
			slog.String("status", event.Status),
			slog.Int64("response_time_ms", event.ResponseTime),
			slog.String("error", event.Error))
	} else {
		m.logger.Debug("Component health check passed",
			slog.String("component", event.Component),
			slog.Int64("response_time_ms", event.ResponseTime))
	}

	return nil
}

func (m *Manager) handleSearchCompleted(event *SearchCompletedEvent) error {
	m.logger.Info("Search completed",
		slog.String("request_id", event.RequestID),
		slog.String("query", event.Query),
		slog.Int("result_count", event.ResultCount),
		slog.Int64("duration_ms", event.Duration),
		slog.Bool("cache_hit", event.CacheHit),
		slog.Bool("success", event.Success))

	return nil
}

func (m *Manager) handlePaperProcessing(event *PaperProcessingEvent) error {
	m.logger.Debug("Paper processing event",
		slog.String("paper_id", event.PaperID),
		slog.String("stage", event.Stage),
		slog.String("status", event.Status),
		slog.Float64("progress", event.Progress))

	return nil
}

// StreamManager provides JetStream management functionality
type StreamManager struct {
	client *Client
	logger *slog.Logger
}

// NewStreamManager creates a new stream manager
func NewStreamManager(client *Client, logger *slog.Logger) *StreamManager {
	return &StreamManager{
		client: client,
		logger: logger,
	}
}

// EnsureStreamsExist ensures all required streams exist
func (sm *StreamManager) EnsureStreamsExist(ctx context.Context) error {
	// This is already handled in client initialization
	// But we could add more sophisticated stream management here
	return nil
}

// GetStreamHealth returns health information for all streams
func (sm *StreamManager) GetStreamHealth(ctx context.Context) (map[string]interface{}, error) {
	streamNames := []string{"PAPERS", "SEARCH", "NOTIFICATIONS"}
	health := make(map[string]interface{})

	for _, streamName := range streamNames {
		info, err := sm.client.GetStreamInfo(streamName)
		if err != nil {
			health[streamName] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
			continue
		}

		health[streamName] = map[string]interface{}{
			"status":   "healthy",
			"messages": info.State.Msgs,
			"bytes":    info.State.Bytes,
			"subjects": info.Config.Subjects,
		}
	}

	return health, nil
}
