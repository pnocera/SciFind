package messaging

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// MessageHandler represents a function that handles incoming messages
type MessageHandler func(ctx context.Context, msg *Message) error

// Message represents a NATS message
type Message struct {
	Subject string
	Data    []byte
	Headers nats.Header
	ReplySubject string
	msg     *nats.Msg      // Core NATS message
	jsMsg   jetstream.Msg  // JetStream message
}

// Subscription represents a NATS subscription
type Subscription struct {
	sub    *nats.Subscription
	logger *slog.Logger
}

// Ack acknowledges the message (for JetStream)
func (m *Message) Ack() error {
	if m.jsMsg != nil {
		return m.jsMsg.Ack()
	}
	return nil
}

// Nak negative acknowledges the message (for JetStream)
func (m *Message) Nak() error {
	if m.jsMsg != nil {
		return m.jsMsg.Nak()
	}
	return nil
}

// Reply sends a reply to the message
func (m *Message) Reply(data interface{}) error {
	if m.ReplySubject == "" {
		return fmt.Errorf("no reply subject")
	}
	
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal reply: %w", err)
	}
	
	if m.msg != nil {
		return m.msg.Respond(payload)
	}
	
	return fmt.Errorf("no underlying message to reply to")
}

// Unmarshal unmarshals the message data into a struct
func (m *Message) Unmarshal(v interface{}) error {
	return json.Unmarshal(m.Data, v)
}

// GetHeader returns a header value
func (m *Message) GetHeader(key string) string {
	return m.Headers.Get(key)
}

// Unsubscribe unsubscribes from the subscription
func (s *Subscription) Unsubscribe() error {
	if err := s.sub.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}
	
	s.logger.Info("Unsubscribed from subject",
		slog.String("subject", s.sub.Subject))
	
	return nil
}

// IsValid returns true if the subscription is still valid
func (s *Subscription) IsValid() bool {
	return s.sub.IsValid()
}

// PendingMessages returns the number of pending messages
func (s *Subscription) PendingMessages() (int, int, error) {
	return s.sub.Pending()
}

// Subject returns the subscription subject
func (s *Subscription) Subject() string {
	return s.sub.Subject
}

// Queue returns the subscription queue group (if any)
func (s *Subscription) Queue() string {
	return s.sub.Queue
}

// Paper Events

// PaperIndexedEvent represents a paper indexing event
type PaperIndexedEvent struct {
	PaperID       string    `json:"paper_id"`
	SourceProvider string   `json:"source_provider"`
	SourceID      string    `json:"source_id"`
	IndexedAt     int64     `json:"indexed_at"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PaperProcessingEvent represents a paper processing event
type PaperProcessingEvent struct {
	PaperID       string    `json:"paper_id"`
	Stage         string    `json:"stage"` // parsing, extraction, quality_scoring, etc.
	Status        string    `json:"status"` // started, completed, failed
	Progress      float64   `json:"progress"` // 0.0 to 1.0
	StartedAt     int64     `json:"started_at"`
	CompletedAt   *int64    `json:"completed_at,omitempty"`
	Error         string    `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PaperQualityUpdatedEvent represents a paper quality score update
type PaperQualityUpdatedEvent struct {
	PaperID      string  `json:"paper_id"`
	OldScore     float64 `json:"old_score"`
	NewScore     float64 `json:"new_score"`
	UpdatedAt    int64   `json:"updated_at"`
	UpdateReason string  `json:"update_reason"`
}

// Search Events

// SearchRequestEvent represents a search request
type SearchRequestEvent struct {
	RequestID    string                 `json:"request_id"`
	UserID       *string                `json:"user_id,omitempty"`
	Query        string                 `json:"query"`
	Providers    []string               `json:"providers"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	RequestedAt  int64                  `json:"requested_at"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
}

// SearchCompletedEvent represents a completed search
type SearchCompletedEvent struct {
	RequestID     string    `json:"request_id"`
	UserID        *string   `json:"user_id,omitempty"`
	Query         string    `json:"query"`
	ResultCount   int       `json:"result_count"`
	Duration      int64     `json:"duration_ms"`
	ProvidersUsed []string  `json:"providers_used"`
	CacheHit      bool      `json:"cache_hit"`
	CompletedAt   int64     `json:"completed_at"`
	Success       bool      `json:"success"`
	Error         string    `json:"error,omitempty"`
}

// SearchAnalyticsEvent represents search analytics data
type SearchAnalyticsEvent struct {
	EventType    string                 `json:"event_type"` // query, result_click, filter_applied, etc.
	RequestID    string                 `json:"request_id"`
	UserID       *string                `json:"user_id,omitempty"`
	Query        string                 `json:"query"`
	PaperID      *string                `json:"paper_id,omitempty"`
	Position     *int                   `json:"position,omitempty"` // Result position for clicks
	Timestamp    int64                  `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Notification Events

// SystemNotificationEvent represents a system notification
type SystemNotificationEvent struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"` // info, warning, error, alert
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Component    string                 `json:"component"` // papers, search, auth, etc.
	Severity     string                 `json:"severity"` // low, medium, high, critical
	Timestamp    int64                  `json:"timestamp"`
	UserID       *string                `json:"user_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt    *int64                 `json:"expires_at,omitempty"`
}

// HealthCheckEvent represents a health check event
type HealthCheckEvent struct {
	Component    string                 `json:"component"`
	Status       string                 `json:"status"` // healthy, unhealthy, degraded
	Timestamp    int64                  `json:"timestamp"`
	ResponseTime int64                  `json:"response_time_ms"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MetricsEvent represents a metrics collection event
type MetricsEvent struct {
	MetricName   string                 `json:"metric_name"`
	MetricType   string                 `json:"metric_type"` // counter, gauge, histogram
	Value        float64                `json:"value"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Timestamp    int64                  `json:"timestamp"`
	Component    string                 `json:"component"`
}

// Message Subjects (Constants for consistency)
const (
	// Paper subjects
	SubjectPaperIndexed         = "papers.indexed"
	SubjectPaperProcessing      = "papers.processing"
	SubjectPaperQualityUpdated  = "papers.quality_updated"
	SubjectPaperCitationsUpdated = "papers.citations_updated"
	
	// Indexing subjects
	SubjectIndexingStarted      = "indexing.started"
	SubjectIndexingCompleted    = "indexing.completed"
	SubjectIndexingFailed       = "indexing.failed"
	SubjectIndexingProgress     = "indexing.progress"
	
	// Search subjects
	SubjectSearchRequest        = "search.request"
	SubjectSearchCompleted      = "search.completed"
	SubjectSearchCached         = "search.cached"
	SubjectSearchAnalytics      = "search.analytics"
	
	// Analytics subjects
	SubjectAnalyticsQuery       = "analytics.query"
	SubjectAnalyticsClick       = "analytics.click"
	SubjectAnalyticsFilter      = "analytics.filter"
	SubjectAnalyticsExport      = "analytics.export"
	
	// Notification subjects
	SubjectNotificationSystem   = "notifications.system"
	SubjectNotificationUser     = "notifications.user"
	SubjectNotificationEmail    = "notifications.email"
	
	// Alert subjects
	SubjectAlertHealthCheck     = "alerts.health_check"
	SubjectAlertPerformance     = "alerts.performance"
	SubjectAlertSecurity        = "alerts.security"
	SubjectAlertError           = "alerts.error"
	
	// Metrics subjects
	SubjectMetricsSystem        = "metrics.system"
	SubjectMetricsApplication   = "metrics.application"
	SubjectMetricsUser          = "metrics.user"
)

// buildTLSConfig builds TLS configuration from NATS TLS config
func buildTLSConfig(cfg *struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
	CAFile   string `mapstructure:"ca_file"`
}) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	
	tlsConfig := &tls.Config{}
	
	// Load client certificate if configured
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	
	// Load CA certificate if configured
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}
	
	return tlsConfig, nil
}

// Publisher interface for publishing messages
type Publisher interface {
	Publish(ctx context.Context, subject string, data interface{}) error
	PublishAsync(ctx context.Context, subject string, data interface{}) error
}

// Subscriber interface for subscribing to messages
type Subscriber interface {
	Subscribe(ctx context.Context, subject string, handler MessageHandler) (*Subscription, error)
	SubscribeQueue(ctx context.Context, subject, queue string, handler MessageHandler) (*Subscription, error)
}

// Requester interface for request-reply messaging
type Requester interface {
	Request(ctx context.Context, subject string, data interface{}, timeout int64) (*Message, error)
}

// EventBus combines all messaging interfaces
type EventBus interface {
	Publisher
	Subscriber
	Requester
}

// Helper functions for creating events

// NewPaperIndexedEvent creates a new paper indexed event
func NewPaperIndexedEvent(paperID, sourceProvider, sourceID string, success bool, err error) *PaperIndexedEvent {
	event := &PaperIndexedEvent{
		PaperID:       paperID,
		SourceProvider: sourceProvider,
		SourceID:      sourceID,
		IndexedAt:     currentTimestamp(),
		Success:       success,
	}
	
	if err != nil {
		event.Error = err.Error()
	}
	
	return event
}

// NewSearchRequestEvent creates a new search request event
func NewSearchRequestEvent(requestID, query string, providers []string, userID *string) *SearchRequestEvent {
	return &SearchRequestEvent{
		RequestID:   requestID,
		UserID:      userID,
		Query:       query,
		Providers:   providers,
		RequestedAt: currentTimestamp(),
	}
}

// NewSystemNotificationEvent creates a new system notification event
func NewSystemNotificationEvent(notifType, title, message, component, severity string) *SystemNotificationEvent {
	return &SystemNotificationEvent{
		ID:        generateEventID(),
		Type:      notifType,
		Title:     title,
		Message:   message,
		Component: component,
		Severity:  severity,
		Timestamp: currentTimestamp(),
	}
}

// Helper functions

func currentTimestamp() int64 {
	return time.Now().UnixMilli()
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d_%s", currentTimestamp(), generateRandomString(6))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}