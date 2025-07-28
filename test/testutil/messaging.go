package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NATSTestUtil provides NATS testing utilities
type NATSTestUtil struct {
	container testcontainers.Container
	conn      *nats.Conn
	url       string
	cleanup   func()
}

// SetupTestNATS creates a NATS container for testing
func SetupTestNATS(t *testing.T) *NATSTestUtil {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"--jetstream", "--store_dir=/data"},
		WaitingFor: wait.ForListeningPort("4222/tcp").
			WithStartupTimeout(30 * time.Second),
	}

	natsContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Get connection URL
	host, err := natsContainer.Host(ctx)
	require.NoError(t, err)

	port, err := natsContainer.MappedPort(ctx, "4222/tcp")
	require.NoError(t, err)

	url := "nats://" + host + ":" + port.Port()

	// Connect to NATS
	conn, err := nats.Connect(url, nats.Timeout(10*time.Second))
	require.NoError(t, err)

	return &NATSTestUtil{
		container: natsContainer,
		conn:      conn,
		url:       url,
		cleanup: func() {
			if conn != nil && !conn.IsClosed() {
				conn.Close()
			}
			if err := natsContainer.Terminate(ctx); err != nil {
				t.Logf("failed to terminate NATS container: %s", err)
			}
		},
	}
}

// Connection returns the NATS connection
func (n *NATSTestUtil) Connection() *nats.Conn {
	return n.conn
}

// URL returns the NATS connection URL
func (n *NATSTestUtil) URL() string {
	return n.url
}

// Cleanup cleans up the NATS container
func (n *NATSTestUtil) Cleanup() {
	if n.cleanup != nil {
		n.cleanup()
	}
}

// PublishTestMessage publishes a test message to the given subject
func (n *NATSTestUtil) PublishTestMessage(t *testing.T, subject string, data []byte) {
	err := n.conn.Publish(subject, data)
	require.NoError(t, err)
}

// SubscribeAndWait subscribes to a subject and waits for a message
func (n *NATSTestUtil) SubscribeAndWait(t *testing.T, subject string, timeout time.Duration) *nats.Msg {
	ch := make(chan *nats.Msg, 1)
	
	sub, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		select {
		case ch <- msg:
		default:
		}
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		t.Fatalf("Timeout waiting for message on subject %s", subject)
		return nil
	}
}

// DrainSubject drains all messages from a subject
func (n *NATSTestUtil) DrainSubject(t *testing.T, subject string) []*nats.Msg {
	var messages []*nats.Msg
	
	sub, err := n.conn.SubscribeSync(subject)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Collect all available messages
	for {
		msg, err := sub.NextMsg(100 * time.Millisecond)
		if err != nil {
			break // No more messages
		}
		messages = append(messages, msg)
	}

	return messages
}

// CreateJetStreamContext creates a JetStream context for testing
func (n *NATSTestUtil) CreateJetStreamContext(t *testing.T) nats.JetStreamContext {
	js, err := n.conn.JetStream()
	require.NoError(t, err)
	return js
}

// CreateTestStream creates a test stream
func (n *NATSTestUtil) CreateTestStream(t *testing.T, name string, subjects []string) {
	js := n.CreateJetStreamContext(t)
	
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     name,
		Subjects: subjects,
	})
	require.NoError(t, err)
}

// CreateTestConsumer creates a test consumer
func (n *NATSTestUtil) CreateTestConsumer(t *testing.T, stream, consumer string) {
	js := n.CreateJetStreamContext(t)
	
	_, err := js.AddConsumer(stream, &nats.ConsumerConfig{
		Durable: consumer,
	})
	require.NoError(t, err)
}

// WaitForStreamMessage waits for a message on a JetStream
func (n *NATSTestUtil) WaitForStreamMessage(t *testing.T, stream, consumer string, timeout time.Duration) *nats.Msg {
	js := n.CreateJetStreamContext(t)
	
	sub, err := js.PullSubscribe("", consumer, nats.Bind(stream, consumer))
	require.NoError(t, err)
	defer sub.Unsubscribe()

	msgs, err := sub.Fetch(1, nats.MaxWait(timeout))
	require.NoError(t, err)
	require.Len(t, msgs, 1)

	return msgs[0]
}