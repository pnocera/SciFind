package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
	"scifind-backend/internal/errors"
)

// EventSubscriber provides high-level event subscription functionality
type EventSubscriber struct {
	client        *Client
	logger        *slog.Logger
	subscriptions map[string]*Subscription
	handlers      map[string][]MessageHandler
	mu            sync.RWMutex
}

// NewEventSubscriber creates a new event subscriber
func NewEventSubscriber(client *Client, logger *slog.Logger) *EventSubscriber {
	return &EventSubscriber{
		client:        client,
		logger:        logger,
		subscriptions: make(map[string]*Subscription),
		handlers:      make(map[string][]MessageHandler),
	}
}

// Subscribe to a specific subject
func (s *EventSubscriber) Subscribe(ctx context.Context, subject string, handler MessageHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Add handler to the list
	s.handlers[subject] = append(s.handlers[subject], handler)
	
	// If this is the first handler for this subject, create a subscription
	if len(s.handlers[subject]) == 1 {
		subscription, err := s.client.Subscribe(subject, func(m *nats.Msg) {
			// Convert to internal Message type and call handler
			msg := &Message{
				Subject: m.Subject,
				Data:    m.Data,
				ReplySubject: m.Reply,
			}
			for _, handler := range s.handlers[subject] {
				handler(context.Background(), msg)
			}
		})
		if err != nil {
			delete(s.handlers, subject)
			return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
		}
		s.subscriptions[subject] = &Subscription{
			sub:    subscription,
			logger: s.logger,
		}
	}
	
	s.logger.Info("Added handler for subject",
		slog.String("subject", subject),
		slog.Int("total_handlers", len(s.handlers[subject])))
	
	return nil
}

// SubscribeQueue subscribes to a subject with a queue group
func (s *EventSubscriber) SubscribeQueue(ctx context.Context, subject, queue string, handler MessageHandler) error {
	key := fmt.Sprintf("%s:%s", subject, queue)
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// For queue subscriptions, we create individual subscriptions
	subscription, err := s.client.SubscribeQueue(subject, queue, func(m *nats.Msg) {
		// Convert to internal Message type and call handler
		msg := &Message{
			Subject: m.Subject,
			Data:    m.Data,
			ReplySubject: m.Reply,
		}
		handler(context.Background(), msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to queue %s for subject %s: %w", queue, subject, err)
	}
	
	s.subscriptions[key] = &Subscription{
		sub:    subscription,
		logger: s.logger,
	}
	
	s.logger.Info("Subscribed to queue",
		slog.String("subject", subject),
		slog.String("queue", queue))
	
	return nil
}

// Unsubscribe from a subject
func (s *EventSubscriber) Unsubscribe(subject string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	subscription, exists := s.subscriptions[subject]
	if !exists {
		return fmt.Errorf("no subscription found for subject: %s", subject)
	}
	
	if err := subscription.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe from %s: %w", subject, err)
	}
	
	delete(s.subscriptions, subject)
	delete(s.handlers, subject)
	
	s.logger.Info("Unsubscribed from subject", slog.String("subject", subject))
	return nil
}

// UnsubscribeAll unsubscribes from all subjects
func (s *EventSubscriber) UnsubscribeAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var errs []error
	
	for subject, subscription := range s.subscriptions {
		if err := subscription.Unsubscribe(); err != nil {
			errs = append(errs, fmt.Errorf("failed to unsubscribe from %s: %w", subject, err))
		}
	}
	
	s.subscriptions = make(map[string]*Subscription)
	s.handlers = make(map[string][]MessageHandler)
	
	if len(errs) > 0 {
		return fmt.Errorf("errors during unsubscribe: %v", errs)
	}
	
	s.logger.Info("Unsubscribed from all subjects")
	return nil
}

// createMultiHandler creates a handler that calls all registered handlers for a subject
func (s *EventSubscriber) createMultiHandler(subject string) MessageHandler {
	return func(ctx context.Context, msg *Message) error {
		s.mu.RLock()
		handlers := s.handlers[subject]
		s.mu.RUnlock()
		
		var errs []error
		
		// Call all handlers for this subject
		for i, handler := range handlers {
			if err := handler(ctx, msg); err != nil {
				s.logger.Error("Handler failed",
					slog.String("subject", subject),
					slog.Int("handler_index", i),
					slog.String("error", err.Error()))
				errs = append(errs, err)
			}
		}
		
		// Return error if any handler failed
		if len(errs) > 0 {
			return fmt.Errorf("handler errors: %v", errs)
		}
		
		return nil
	}
}

// Paper Event Handlers

// OnPaperIndexed registers a handler for paper indexed events
func (s *EventSubscriber) OnPaperIndexed(ctx context.Context, handler func(event *PaperIndexedEvent) error) error {
	return s.Subscribe(ctx, SubjectPaperIndexed, func(ctx context.Context, msg *Message) error {
		var event PaperIndexedEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_paper_indexed", err)
		}
		return handler(&event)
	})
}

// OnPaperProcessing registers a handler for paper processing events
func (s *EventSubscriber) OnPaperProcessing(ctx context.Context, handler func(event *PaperProcessingEvent) error) error {
	return s.Subscribe(ctx, SubjectPaperProcessing, func(ctx context.Context, msg *Message) error {
		var event PaperProcessingEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_paper_processing", err)
		}
		return handler(&event)
	})
}

// OnPaperQualityUpdated registers a handler for paper quality updated events
func (s *EventSubscriber) OnPaperQualityUpdated(ctx context.Context, handler func(event *PaperQualityUpdatedEvent) error) error {
	return s.Subscribe(ctx, SubjectPaperQualityUpdated, func(ctx context.Context, msg *Message) error {
		var event PaperQualityUpdatedEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_paper_quality_updated", err)
		}
		return handler(&event)
	})
}

// Search Event Handlers

// OnSearchRequest registers a handler for search request events
func (s *EventSubscriber) OnSearchRequest(ctx context.Context, handler func(event *SearchRequestEvent) error) error {
	return s.Subscribe(ctx, SubjectSearchRequest, func(ctx context.Context, msg *Message) error {
		var event SearchRequestEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_search_request", err)
		}
		return handler(&event)
	})
}

// OnSearchCompleted registers a handler for search completed events
func (s *EventSubscriber) OnSearchCompleted(ctx context.Context, handler func(event *SearchCompletedEvent) error) error {
	return s.Subscribe(ctx, SubjectSearchCompleted, func(ctx context.Context, msg *Message) error {
		var event SearchCompletedEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_search_completed", err)
		}
		return handler(&event)
	})
}

// OnSearchAnalytics registers a handler for search analytics events
func (s *EventSubscriber) OnSearchAnalytics(ctx context.Context, handler func(event *SearchAnalyticsEvent) error) error {
	return s.Subscribe(ctx, SubjectSearchAnalytics, func(ctx context.Context, msg *Message) error {
		var event SearchAnalyticsEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_search_analytics", err)
		}
		return handler(&event)
	})
}

// Notification Event Handlers

// OnSystemNotification registers a handler for system notifications
func (s *EventSubscriber) OnSystemNotification(ctx context.Context, handler func(event *SystemNotificationEvent) error) error {
	return s.Subscribe(ctx, SubjectNotificationSystem, func(ctx context.Context, msg *Message) error {
		var event SystemNotificationEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_system_notification", err)
		}
		return handler(&event)
	})
}

// OnHealthCheck registers a handler for health check events
func (s *EventSubscriber) OnHealthCheck(ctx context.Context, handler func(event *HealthCheckEvent) error) error {
	return s.Subscribe(ctx, SubjectAlertHealthCheck, func(ctx context.Context, msg *Message) error {
		var event HealthCheckEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_health_check", err)
		}
		return handler(&event)
	})
}

// OnMetrics registers a handler for metrics events
func (s *EventSubscriber) OnMetrics(ctx context.Context, handler func(event *MetricsEvent) error) error {
	return s.Subscribe(ctx, SubjectMetricsApplication, func(ctx context.Context, msg *Message) error {
		var event MetricsEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_metrics", err)
		}
		return handler(&event)
	})
}

// Indexing Event Handlers

// OnIndexingStarted registers a handler for indexing started events
func (s *EventSubscriber) OnIndexingStarted(ctx context.Context, handler func(provider string, metadata map[string]interface{}) error) error {
	return s.Subscribe(ctx, SubjectIndexingStarted, func(ctx context.Context, msg *Message) error {
		var event map[string]interface{}
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_indexing_started", err)
		}
		
		provider, _ := event["provider"].(string)
		metadata, _ := event["metadata"].(map[string]interface{})
		
		return handler(provider, metadata)
	})
}

// OnIndexingCompleted registers a handler for indexing completed events
func (s *EventSubscriber) OnIndexingCompleted(ctx context.Context, handler func(provider string, papersIndexed int, durationMs int64, metadata map[string]interface{}) error) error {
	return s.Subscribe(ctx, SubjectIndexingCompleted, func(ctx context.Context, msg *Message) error {
		var event map[string]interface{}
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_indexing_completed", err)
		}
		
		provider, _ := event["provider"].(string)
		papersIndexed, _ := event["papers_indexed"].(float64) // JSON numbers are float64
		durationMs, _ := event["duration_ms"].(float64)
		metadata, _ := event["metadata"].(map[string]interface{})
		
		return handler(provider, int(papersIndexed), int64(durationMs), metadata)
	})
}

// OnIndexingFailed registers a handler for indexing failed events
func (s *EventSubscriber) OnIndexingFailed(ctx context.Context, handler func(provider string, err error, metadata map[string]interface{}) error) error {
	return s.Subscribe(ctx, SubjectIndexingFailed, func(ctx context.Context, msg *Message) error {
		var event map[string]interface{}
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_indexing_failed", err)
		}
		
		provider, _ := event["provider"].(string)
		errorStr, _ := event["error"].(string)
		metadata, _ := event["metadata"].(map[string]interface{})
		
		return handler(provider, fmt.Errorf(errorStr), metadata)
	})
}

// Queue-based Event Handlers (for load balancing)

// OnPaperProcessingQueue registers a queue-based handler for paper processing events
func (s *EventSubscriber) OnPaperProcessingQueue(ctx context.Context, queueGroup string, handler func(event *PaperProcessingEvent) error) error {
	return s.SubscribeQueue(ctx, SubjectPaperProcessing, queueGroup, func(ctx context.Context, msg *Message) error {
		var event PaperProcessingEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_paper_processing_queue", err)
		}
		return handler(&event)
	})
}

// OnSearchRequestQueue registers a queue-based handler for search request events
func (s *EventSubscriber) OnSearchRequestQueue(ctx context.Context, queueGroup string, handler func(event *SearchRequestEvent) error) error {
	return s.SubscribeQueue(ctx, SubjectSearchRequest, queueGroup, func(ctx context.Context, msg *Message) error {
		var event SearchRequestEvent
		if err := msg.Unmarshal(&event); err != nil {
			return errors.NewSerializationError("unmarshal_search_request_queue", err)
		}
		return handler(&event)
	})
}

// GetSubscriptionInfo returns information about all active subscriptions
func (s *EventSubscriber) GetSubscriptionInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	info := make(map[string]interface{})
	
	for subject, subscription := range s.subscriptions {
		pending, _, _ := subscription.PendingMessages()
		info[subject] = map[string]interface{}{
			"valid":           subscription.IsValid(),
			"pending_messages": pending,
			"queue":           subscription.Queue(),
			"handlers":        len(s.handlers[subject]),
		}
	}
	
	return info
}