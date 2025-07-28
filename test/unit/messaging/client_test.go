package messaging_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scifind-backend/internal/config"
	"scifind-backend/internal/messaging"
	"scifind-backend/test/testutil"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestNewClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	natsUtil := testutil.SetupTestNATS(t)
	defer natsUtil.Cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name      string
		config    config.NATSConfig
		wantErr   bool
		errContains string
	}{
		{
			name: "valid config creates client",
			config: config.NATSConfig{
				URL:           natsUtil.URL(),
				ClientID:      "test-client",
				MaxReconnects: 5,
				ReconnectWait: "1s",
				Timeout:       "5s",
			},
			wantErr: false,
		},
		{
			name: "invalid URL returns error",
			config: config.NATSConfig{
				URL:           "invalid://url",
				ClientID:      "test-client",
				MaxReconnects: 5,
				ReconnectWait: "1s",
				Timeout:       "5s",
			},
			wantErr:     true,
			errContains: "NATS connection failed",
		},
		{
			name: "invalid timeout format uses default",
			config: config.NATSConfig{
				URL:           natsUtil.URL(),
				ClientID:      "test-client",
				MaxReconnects: 5,
				ReconnectWait: "invalid",
				Timeout:       "invalid",
			},
			wantErr: false, // Should use defaults and still work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := messaging.NewClient(tt.config, logger)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, client)
				return
			}
			
			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.True(t, client.IsConnected())
			assert.Equal(t, natsUtil.URL(), client.ConnectedURL())
			
			// Test stats
			stats := client.Stats()
			assert.NotNil(t, stats)
			
			// Cleanup
			assert.NoError(t, client.Close())
		})
	}
}

func TestClient_ConnectionLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	natsUtil := testutil.SetupTestNATS(t)
	defer natsUtil.Cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := config.NATSConfig{
		URL:           natsUtil.URL(),
		ClientID:      "lifecycle-test",
		MaxReconnects: 5,
		ReconnectWait: "100ms",
		Timeout:       "2s",
	}

	client, err := messaging.NewClient(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	t.Run("connection state checks", func(t *testing.T) {
		assert.True(t, client.IsConnected())
		assert.Equal(t, natsUtil.URL(), client.ConnectedURL())
	})

	t.Run("stats collection", func(t *testing.T) {
		stats := client.Stats()
		assert.GreaterOrEqual(t, stats.InMsgs, uint64(0))
		assert.GreaterOrEqual(t, stats.OutMsgs, uint64(0))
		assert.GreaterOrEqual(t, stats.InBytes, uint64(0))
		assert.GreaterOrEqual(t, stats.OutBytes, uint64(0))
	})

	t.Run("drain and close", func(t *testing.T) {
		// Test drain
		err := client.Drain()
		assert.NoError(t, err)

		// Test close
		err = client.Close()
		assert.NoError(t, err)
	})
}

func TestClient_PublishSubscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	natsUtil := testutil.SetupTestNATS(t)
	defer natsUtil.Cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := config.NATSConfig{
		URL:           natsUtil.URL(),
		ClientID:      "pubsub-test",
		MaxReconnects: 5,
		ReconnectWait: "100ms",
		Timeout:       "2s",
	}

	client, err := messaging.NewClient(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	t.Run("publish and subscribe to messages", func(t *testing.T) {
		subject := "test.messages"
		receivedCh := make(chan []byte, 1)

		// Subscribe
		sub, err := client.Subscribe(subject, func(msg *nats.Msg) {
			receivedCh <- msg.Data
		})
		require.NoError(t, err)
		defer sub.Unsubscribe()

		// Publish test data
		testData := map[string]interface{}{
			"id":      "test-123",
			"message": "Hello NATS!",
			"timestamp": time.Now().Unix(),
		}

		ctx := context.Background()
		err = client.Publish(ctx, subject, testData)
		require.NoError(t, err)

		// Wait for message
		select {
		case received := <-receivedCh:
			var receivedData map[string]interface{}
			err := json.Unmarshal(received, &receivedData)
			require.NoError(t, err)
			assert.Equal(t, testData["id"], receivedData["id"])
			assert.Equal(t, testData["message"], receivedData["message"])
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("publish async", func(t *testing.T) {
		subject := "test.async"
		receivedCh := make(chan []byte, 1)

		// Subscribe
		sub, err := client.Subscribe(subject, func(msg *nats.Msg) {
			receivedCh <- msg.Data
		})
		require.NoError(t, err)
		defer sub.Unsubscribe()

		// Publish async
		testData := map[string]string{"type": "async", "data": "test"}
		ctx := context.Background()
		err = client.PublishAsync(ctx, subject, testData)
		require.NoError(t, err)

		// Wait for message
		select {
		case received := <-receivedCh:
			var receivedData map[string]string
			err := json.Unmarshal(received, &receivedData)
			require.NoError(t, err)
			assert.Equal(t, testData["type"], receivedData["type"])
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for async message")
		}
	})

	t.Run("queue subscription", func(t *testing.T) {
		subject := "test.queue"
		queue := "workers"
		receivedCount := 0
		receivedCh := make(chan struct{}, 10)

		// Create multiple queue subscribers
		var subs []*nats.Subscription
		for i := 0; i < 3; i++ {
			sub, err := client.SubscribeQueue(subject, queue, func(msg *nats.Msg) {
				receivedCount++
				receivedCh <- struct{}{}
			})
			require.NoError(t, err)
			subs = append(subs, sub)
		}
		defer func() {
			for _, sub := range subs {
				sub.Unsubscribe()
			}
		}()

		// Send multiple messages
		ctx := context.Background()
		messageCount := 5
		for i := 0; i < messageCount; i++ {
			err := client.Publish(ctx, subject, map[string]int{"msg": i})
			require.NoError(t, err)
		}

		// Wait for all messages to be received
		for i := 0; i < messageCount; i++ {
			select {
			case <-receivedCh:
				// Message received
			case <-time.After(2 * time.Second):
				t.Fatalf("timeout waiting for message %d", i)
			}
		}

		assert.Equal(t, messageCount, receivedCount)
	})
}

func TestClient_ErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("publish with nil connection", func(t *testing.T) {
		// Create client with invalid URL to get connection failure
		cfg := config.NATSConfig{
			URL:      "nats://invalid:4222",
			ClientID: "error-test",
			Timeout:  "1s",
		}

		client, err := messaging.NewClient(cfg, logger)
		assert.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("publish with invalid data", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping integration test")
		}

		natsUtil := testutil.SetupTestNATS(t)
		defer natsUtil.Cleanup()

		cfg := config.NATSConfig{
			URL:      natsUtil.URL(),
			ClientID: "error-test",
			Timeout:  "2s",
		}

		client, err := messaging.NewClient(cfg, logger)
		require.NoError(t, err)
		defer client.Close()

		// Try to publish data that can't be JSON marshaled
		ctx := context.Background()
		invalidData := make(chan int) // channels can't be marshaled to JSON
		err = client.Publish(ctx, "test.invalid", invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Failed to serialize message data")
	})
}

func TestClient_JetStreamIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	natsUtil := testutil.SetupTestNATS(t)
	defer natsUtil.Cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := config.NATSConfig{
		URL:      natsUtil.URL(),
		ClientID: "jetstream-test",
		Timeout:  "5s",
	}

	client, err := messaging.NewClient(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	t.Run("get stream info", func(t *testing.T) {
		// Create a test stream first
		streamName := "TEST_STREAM"
		natsUtil.CreateTestStream(t, streamName, []string{"test.stream.*"})

		// Get stream info
		info, err := client.GetStreamInfo(streamName)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, streamName, info.Config.Name)
		assert.Contains(t, info.Config.Subjects, "test.stream.*")
	})

	t.Run("get non-existent stream info", func(t *testing.T) {
		info, err := client.GetStreamInfo("NON_EXISTENT")
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "failed to get stream")
	})
}

// Benchmark tests for performance analysis
func BenchmarkClient_Publish(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test")
	}

	natsUtil := testutil.SetupTestNATS(&testing.T{})
	defer natsUtil.Cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := config.NATSConfig{
		URL:      natsUtil.URL(),
		ClientID: "benchmark-test",
		Timeout:  "5s",
	}

	client, err := messaging.NewClient(cfg, logger)
	require.NoError(b, err)
	defer client.Close()

	testData := map[string]interface{}{
		"id":        "bench-test",
		"message":   "benchmark message",
		"timestamp": time.Now().Unix(),
		"metadata":  map[string]string{"source": "benchmark"},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := client.Publish(ctx, "benchmark.test", testData)
			if err != nil {
				b.Errorf("Publish failed: %v", err)
			}
		}
	})
}

func BenchmarkClient_Subscribe(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test")
	}

	natsUtil := testutil.SetupTestNATS(&testing.T{})
	defer natsUtil.Cleanup()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := config.NATSConfig{
		URL:      natsUtil.URL(),
		ClientID: "benchmark-sub-test",
		Timeout:  "5s",
	}

	client, err := messaging.NewClient(cfg, logger)
	require.NoError(b, err)
	defer client.Close()

	subject := "benchmark.subscribe"
	msgCount := 0

	// Subscribe
	sub, err := client.Subscribe(subject, func(msg *nats.Msg) {
		msgCount++
	})
	require.NoError(b, err)
	defer sub.Unsubscribe()

	testData := map[string]string{"bench": "data"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := client.Publish(ctx, subject, testData)
		if err != nil {
			b.Errorf("Publish failed: %v", err)
		}
	}

	// Wait a bit for all messages to be processed
	time.Sleep(100 * time.Millisecond)
	b.Logf("Processed %d messages", msgCount)
}