package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"scifind-backend/internal/errors"
)

// EventPublisher provides high-level event publishing functionality
type EventPublisher struct {
	client *Client
	logger *slog.Logger
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(client *Client, logger *slog.Logger) *EventPublisher {
	return &EventPublisher{
		client: client,
		logger: logger,
	}
}

// Paper Events

// PublishPaperIndexed publishes a paper indexed event
func (p *EventPublisher) PublishPaperIndexed(ctx context.Context, paperID, sourceProvider, sourceID string, success bool, err error) error {
	event := NewPaperIndexedEvent(paperID, sourceProvider, sourceID, success, err)
	
	if err := p.client.PublishAsync(ctx, SubjectPaperIndexed, event); err != nil {
		return fmt.Errorf("failed to publish paper indexed event: %w", err)
	}
	
	p.logger.Debug("Published paper indexed event",
		slog.String("paper_id", paperID),
		slog.String("provider", sourceProvider),
		slog.Bool("success", success))
	
	return nil
}

// PublishPaperProcessing publishes a paper processing event
func (p *EventPublisher) PublishPaperProcessing(ctx context.Context, paperID, stage, status string, progress float64) error {
	event := &PaperProcessingEvent{
		PaperID:   paperID,
		Stage:     stage,
		Status:    status,
		Progress:  progress,
		StartedAt: currentTimestamp(),
	}
	
	if status == "completed" || status == "failed" {
		completedAt := currentTimestamp()
		event.CompletedAt = &completedAt
	}
	
	if err := p.client.PublishAsync(ctx, SubjectPaperProcessing, event); err != nil {
		return fmt.Errorf("failed to publish paper processing event: %w", err)
	}
	
	p.logger.Debug("Published paper processing event",
		slog.String("paper_id", paperID),
		slog.String("stage", stage),
		slog.String("status", status),
		slog.Float64("progress", progress))
	
	return nil
}

// PublishPaperQualityUpdated publishes a paper quality score update event
func (p *EventPublisher) PublishPaperQualityUpdated(ctx context.Context, paperID string, oldScore, newScore float64, reason string) error {
	event := &PaperQualityUpdatedEvent{
		PaperID:      paperID,
		OldScore:     oldScore,
		NewScore:     newScore,
		UpdatedAt:    currentTimestamp(),
		UpdateReason: reason,
	}
	
	if err := p.client.PublishAsync(ctx, SubjectPaperQualityUpdated, event); err != nil {
		return fmt.Errorf("failed to publish paper quality updated event: %w", err)
	}
	
	p.logger.Debug("Published paper quality updated event",
		slog.String("paper_id", paperID),
		slog.Float64("old_score", oldScore),
		slog.Float64("new_score", newScore))
	
	return nil
}

// Search Events

// PublishSearchRequest publishes a search request event
func (p *EventPublisher) PublishSearchRequest(ctx context.Context, requestID, query string, providers []string, userID *string, ipAddress, userAgent string) error {
	event := NewSearchRequestEvent(requestID, query, providers, userID)
	event.IPAddress = ipAddress
	event.UserAgent = userAgent
	
	if err := p.client.PublishAsync(ctx, SubjectSearchRequest, event); err != nil {
		return fmt.Errorf("failed to publish search request event: %w", err)
	}
	
	p.logger.Debug("Published search request event",
		slog.String("request_id", requestID),
		slog.String("query", query),
		slog.Any("providers", providers))
	
	return nil
}

// PublishSearchCompleted publishes a search completed event
func (p *EventPublisher) PublishSearchCompleted(ctx context.Context, requestID, query string, resultCount int, duration time.Duration, providersUsed []string, cacheHit bool, userID *string, err error) error {
	event := &SearchCompletedEvent{
		RequestID:     requestID,
		UserID:        userID,
		Query:         query,
		ResultCount:   resultCount,
		Duration:      duration.Milliseconds(),
		ProvidersUsed: providersUsed,
		CacheHit:      cacheHit,
		CompletedAt:   currentTimestamp(),
		Success:       err == nil,
	}
	
	if err != nil {
		event.Error = err.Error()
	}
	
	if err := p.client.PublishAsync(ctx, SubjectSearchCompleted, event); err != nil {
		return fmt.Errorf("failed to publish search completed event: %w", err)
	}
	
	p.logger.Debug("Published search completed event",
		slog.String("request_id", requestID),
		slog.String("query", query),
		slog.Int("result_count", resultCount),
		slog.Int64("duration_ms", duration.Milliseconds()),
		slog.Bool("cache_hit", cacheHit),
		slog.Bool("success", err == nil))
	
	return nil
}

// PublishSearchAnalytics publishes a search analytics event
func (p *EventPublisher) PublishSearchAnalytics(ctx context.Context, eventType, requestID, query string, userID *string, paperID *string, position *int, metadata map[string]interface{}) error {
	event := &SearchAnalyticsEvent{
		EventType: eventType,
		RequestID: requestID,
		UserID:    userID,
		Query:     query,
		PaperID:   paperID,
		Position:  position,
		Timestamp: currentTimestamp(),
		Metadata:  metadata,
	}
	
	if err := p.client.PublishAsync(ctx, SubjectSearchAnalytics, event); err != nil {
		return fmt.Errorf("failed to publish search analytics event: %w", err)
	}
	
	p.logger.Debug("Published search analytics event",
		slog.String("event_type", eventType),
		slog.String("request_id", requestID),
		slog.String("query", query))
	
	return nil
}

// Notification Events

// PublishSystemNotification publishes a system notification
func (p *EventPublisher) PublishSystemNotification(ctx context.Context, notifType, title, message, component, severity string, userID *string, metadata map[string]interface{}) error {
	event := NewSystemNotificationEvent(notifType, title, message, component, severity)
	event.UserID = userID
	event.Metadata = metadata
	
	if err := p.client.PublishAsync(ctx, SubjectNotificationSystem, event); err != nil {
		return fmt.Errorf("failed to publish system notification: %w", err)
	}
	
	p.logger.Info("Published system notification",
		slog.String("type", notifType),
		slog.String("title", title),
		slog.String("component", component),
		slog.String("severity", severity))
	
	return nil
}

// PublishHealthCheck publishes a health check event
func (p *EventPublisher) PublishHealthCheck(ctx context.Context, component, status string, responseTime time.Duration, err error, metadata map[string]interface{}) error {
	event := &HealthCheckEvent{
		Component:    component,
		Status:       status,
		Timestamp:    currentTimestamp(),
		ResponseTime: responseTime.Milliseconds(),
		Metadata:     metadata,
	}
	
	if err != nil {
		event.Error = err.Error()
	}
	
	if err := p.client.PublishAsync(ctx, SubjectAlertHealthCheck, event); err != nil {
		return fmt.Errorf("failed to publish health check event: %w", err)
	}
	
	p.logger.Debug("Published health check event",
		slog.String("component", component),
		slog.String("status", status),
		slog.Int64("response_time_ms", responseTime.Milliseconds()))
	
	return nil
}

// PublishMetrics publishes a metrics event
func (p *EventPublisher) PublishMetrics(ctx context.Context, metricName, metricType, component string, value float64, labels map[string]string) error {
	event := &MetricsEvent{
		MetricName: metricName,
		MetricType: metricType,
		Value:      value,
		Labels:     labels,
		Timestamp:  currentTimestamp(),
		Component:  component,
	}
	
	if err := p.client.PublishAsync(ctx, SubjectMetricsApplication, event); err != nil {
		return fmt.Errorf("failed to publish metrics event: %w", err)
	}
	
	p.logger.Debug("Published metrics event",
		slog.String("metric_name", metricName),
		slog.String("metric_type", metricType),
		slog.String("component", component),
		slog.Float64("value", value))
	
	return nil
}

// Convenience methods for common notifications

// PublishInfo publishes an info notification
func (p *EventPublisher) PublishInfo(ctx context.Context, component, title, message string, metadata map[string]interface{}) error {
	return p.PublishSystemNotification(ctx, "info", title, message, component, "low", nil, metadata)
}

// PublishWarning publishes a warning notification
func (p *EventPublisher) PublishWarning(ctx context.Context, component, title, message string, metadata map[string]interface{}) error {
	return p.PublishSystemNotification(ctx, "warning", title, message, component, "medium", nil, metadata)
}

// PublishError publishes an error notification
func (p *EventPublisher) PublishError(ctx context.Context, component, title, message string, err error, metadata map[string]interface{}) error {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	if err != nil {
		metadata["error"] = err.Error()
		if sciFindErr, ok := err.(*errors.SciFindError); ok {
			metadata["error_type"] = sciFindErr.Type
			metadata["error_code"] = sciFindErr.Code
		}
	}
	
	return p.PublishSystemNotification(ctx, "error", title, message, component, "high", nil, metadata)
}

// PublishAlert publishes a critical alert
func (p *EventPublisher) PublishAlert(ctx context.Context, component, title, message string, metadata map[string]interface{}) error {
	return p.PublishSystemNotification(ctx, "alert", title, message, component, "critical", nil, metadata)
}

// Indexing Events

// PublishIndexingStarted publishes an indexing started event
func (p *EventPublisher) PublishIndexingStarted(ctx context.Context, provider string, metadata map[string]interface{}) error {
	event := map[string]interface{}{
		"provider":  provider,
		"timestamp": currentTimestamp(),
		"metadata":  metadata,
	}
	
	if err := p.client.PublishAsync(ctx, SubjectIndexingStarted, event); err != nil {
		return fmt.Errorf("failed to publish indexing started event: %w", err)
	}
	
	p.logger.Info("Published indexing started event",
		slog.String("provider", provider))
	
	return nil
}

// PublishIndexingCompleted publishes an indexing completed event
func (p *EventPublisher) PublishIndexingCompleted(ctx context.Context, provider string, papersIndexed int, duration time.Duration, metadata map[string]interface{}) error {
	event := map[string]interface{}{
		"provider":        provider,
		"papers_indexed":  papersIndexed,
		"duration_ms":     duration.Milliseconds(),
		"timestamp":       currentTimestamp(),
		"metadata":        metadata,
	}
	
	if err := p.client.PublishAsync(ctx, SubjectIndexingCompleted, event); err != nil {
		return fmt.Errorf("failed to publish indexing completed event: %w", err)
	}
	
	p.logger.Info("Published indexing completed event",
		slog.String("provider", provider),
		slog.Int("papers_indexed", papersIndexed),
		slog.Int64("duration_ms", duration.Milliseconds()))
	
	return nil
}

// PublishIndexingFailed publishes an indexing failed event
func (p *EventPublisher) PublishIndexingFailed(ctx context.Context, provider string, err error, metadata map[string]interface{}) error {
	event := map[string]interface{}{
		"provider":  provider,
		"error":     err.Error(),
		"timestamp": currentTimestamp(),
		"metadata":  metadata,
	}
	
	if err := p.client.PublishAsync(ctx, SubjectIndexingFailed, event); err != nil {
		return fmt.Errorf("failed to publish indexing failed event: %w", err)
	}
	
	p.logger.Error("Published indexing failed event",
		slog.String("provider", provider),
		slog.String("error", err.Error()))
	
	return nil
}