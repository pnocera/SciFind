package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"scifind-backend/internal/config"
	"scifind-backend/internal/errors"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client wraps NATS connection and provides messaging functionality
type Client struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	config config.NATSConfig
	logger *slog.Logger
}

// NewClient creates a new NATS client
func NewClient(cfg config.NATSConfig, logger *slog.Logger) (*Client, error) {
	// Parse duration strings
	reconnectWait, err := time.ParseDuration(cfg.ReconnectWait)
	if err != nil {
		reconnectWait = 5 * time.Second
	}

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	// NATS connection options
	opts := []nats.Option{
		nats.Name(cfg.ClientID),
		nats.Timeout(timeout),
		nats.ReconnectWait(reconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", slog.String("error", err.Error()))
			} else {
				logger.Info("NATS disconnected gracefully")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", slog.String("url", nc.ConnectedUrl()))
		}),
	}

	// Connect to NATS
	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, errors.NewMessagingError(fmt.Sprintf("NATS connection failed: %v", err), map[string]interface{}{"url": cfg.URL})
	}

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, errors.NewMessagingError(fmt.Sprintf("JetStream creation failed: %v", err), map[string]interface{}{"error": err.Error()})
	}

	client := &Client{
		conn:   conn,
		js:     js,
		config: cfg,
		logger: logger,
	}

	logger.Info("NATS client created", slog.String("url", cfg.URL))
	return client, nil
}

// IsConnected returns true if connected to NATS
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// ConnectedURL returns the connected NATS URL
func (c *Client) ConnectedURL() string {
	if c.conn == nil {
		return ""
	}
	return c.conn.ConnectedUrl()
}

// Drain drains the NATS connection
func (c *Client) Drain() error {
	if c.conn != nil {
		return c.conn.Drain()
	}
	return nil
}

// Stats returns connection statistics
func (c *Client) Stats() nats.Statistics {
	if c.conn != nil {
		return c.conn.Stats()
	}
	return nats.Statistics{}
}

// Publish publishes a message to a subject
func (c *Client) Publish(ctx context.Context, subject string, data interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	
	// Serialize data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.NewSerializationError("Failed to serialize message data", data)
	}
	
	return c.conn.Publish(subject, jsonData)
}

// PublishAsync publishes a message asynchronously  
func (c *Client) PublishAsync(ctx context.Context, subject string, data interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	
	// Serialize data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.NewSerializationError("Failed to serialize message data", data)
	}
	
	// For async publishing, we just use the same Publish method as NATS handles it asynchronously
	return c.conn.Publish(subject, jsonData)
}

// Subscribe subscribes to a subject
func (c *Client) Subscribe(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}
	return c.conn.Subscribe(subject, handler)
}

// SubscribeQueue subscribes to a subject with queue group
func (c *Client) SubscribeQueue(subject, queue string, handler func(*nats.Msg)) (*nats.Subscription, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}
	return c.conn.QueueSubscribe(subject, queue, handler)
}

// GetStreamInfo retrieves information about a JetStream stream
func (c *Client) GetStreamInfo(streamName string) (*jetstream.StreamInfo, error) {
	if c.js == nil {
		return nil, fmt.Errorf("JetStream context is nil")
	}
	
	stream, err := c.js.Stream(context.Background(), streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}
	
	info, err := stream.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info for %s: %w", streamName, err)
	}
	
	return info, nil
}

// Close closes the NATS connection
func (c *Client) Close() error {
	if c.conn != nil {
		c.conn.Close()
		c.logger.Info("NATS connection closed")
	}
	return nil
}
