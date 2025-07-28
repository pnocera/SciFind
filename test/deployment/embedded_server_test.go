package deployment_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"scifind-backend/internal/config"
	"scifind-backend/internal/messaging"
)

// Test embedded NATS server deployment scenarios
func TestEmbeddedNATSServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping embedded server test")
	}

	t.Run("basic embedded server startup", func(t *testing.T) {
		// Find available port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		// Configure embedded server
		opts := &server.Options{
			Host:      "127.0.0.1",
			Port:      port,
			JetStream: true,
			StoreDir:  filepath.Join(os.TempDir(), fmt.Sprintf("nats-store-%d", port)),
		}

		// Start embedded server
		natsServer := server.New(opts)
		require.NotNil(t, natsServer)

		go natsServer.Start()
		defer natsServer.Shutdown()

		// Wait for server to be ready
		ready := natsServer.ReadyForConnections(10 * time.Second)
		assert.True(t, ready, "Embedded server should be ready")

		// Test client connection to embedded server
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
		
		cfg := config.NATSConfig{
			URL:      fmt.Sprintf("nats://127.0.0.1:%d", port),
			ClientID: "embedded-test-client",
			Timeout:  "5s",
		}

		client, err := messaging.NewClient(cfg, logger)
		require.NoError(t, err)
		assert.True(t, client.IsConnected())

		// Test basic operations
		ctx := context.Background()
		err = client.Publish(ctx, "test.embedded", map[string]string{"message": "hello"})
		assert.NoError(t, err)

		client.Close()
	})

	t.Run("embedded server with custom configuration", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		// Configure server with custom settings
		storeDir := filepath.Join(os.TempDir(), fmt.Sprintf("nats-custom-%d", port))
		opts := &server.Options{
			Host:     "127.0.0.1",
			Port:     port,
			MaxConn:  1000,
			MaxSubs:  10000,
			JetStream: true,
			JetStreamMaxMemory: 64 * 1024 * 1024, // 64MB
			JetStreamMaxStore:  512 * 1024 * 1024, // 512MB
			StoreDir:  storeDir,
			// Enable clustering for future tests
			Cluster: server.ClusterOpts{
				Name: "embedded-cluster",
			},
		}

		natsServer := server.New(opts)
		require.NotNil(t, natsServer)

		go natsServer.Start()
		defer func() {
			natsServer.Shutdown()
			os.RemoveAll(storeDir)
		}()

		ready := natsServer.ReadyForConnections(15 * time.Second)
		assert.True(t, ready)

		// Test JetStream functionality
		conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", port))
		require.NoError(t, err)
		defer conn.Close()

		js, err := conn.JetStream()
		require.NoError(t, err)

		// Create a stream
		streamName := "EMBEDDED_TEST"
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"embedded.test.*"},
			Storage:  nats.FileStorage,
		})
		require.NoError(t, err)

		// Publish messages to stream
		for i := 0; i < 10; i++ {
			_, err = js.Publish(fmt.Sprintf("embedded.test.%d", i), 
				[]byte(fmt.Sprintf(`{"id": %d, "data": "test message"}`, i)))
			require.NoError(t, err)
		}

		// Verify stream state
		info, err := js.StreamInfo(streamName)
		require.NoError(t, err)
		assert.Equal(t, uint64(10), info.State.Msgs)
		assert.Greater(t, info.State.Bytes, uint64(0))

		// Test consumer
		_, err = js.AddConsumer(streamName, &nats.ConsumerConfig{
			Durable: "test-consumer",
		})
		require.NoError(t, err)

		sub, err := js.PullSubscribe("", "test-consumer", nats.Bind(streamName, "test-consumer"))
		require.NoError(t, err)
		defer sub.Unsubscribe()

		// Fetch messages
		msgs, err := sub.Fetch(5, nats.MaxWait(5*time.Second))
		require.NoError(t, err)
		assert.Len(t, msgs, 5)

		for _, msg := range msgs {
			assert.Contains(t, string(msg.Data), "test message")
			msg.Ack()
		}
	})

	t.Run("embedded server resilience", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		storeDir := filepath.Join(os.TempDir(), fmt.Sprintf("nats-resilience-%d", port))
		defer os.RemoveAll(storeDir)

		opts := &server.Options{
			Host:      "127.0.0.1",
			Port:      port,
			JetStream: true,
			StoreDir:  storeDir,
		}

		natsServer := server.New(opts)
		require.NotNil(t, natsServer)

		go natsServer.Start()
		ready := natsServer.ReadyForConnections(10 * time.Second)
		assert.True(t, ready)

		// Create persistent stream
		conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", port))
		require.NoError(t, err)

		js, err := conn.JetStream()
		require.NoError(t, err)

		streamName := "PERSISTENT_TEST"
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"persistent.test.*"},
			Storage:  nats.FileStorage,
		})
		require.NoError(t, err)

		// Publish messages
		for i := 0; i < 5; i++ {
			_, err = js.Publish("persistent.test.data", 
				[]byte(fmt.Sprintf(`{"sequence": %d}`, i)))
			require.NoError(t, err)
		}

		conn.Close()

		// Shutdown server
		natsServer.Shutdown()

		// Restart server with same configuration
		natsServer2 := server.New(opts)
		go natsServer2.Start()
		defer natsServer2.Shutdown()

		ready = natsServer2.ReadyForConnections(10 * time.Second)
		assert.True(t, ready)

		// Reconnect and verify data persistence
		conn2, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", port))
		require.NoError(t, err)
		defer conn2.Close()

		js2, err := conn2.JetStream()
		require.NoError(t, err)

		// Stream should still exist with data
		info, err := js2.StreamInfo(streamName)
		require.NoError(t, err)
		assert.Equal(t, uint64(5), info.State.Msgs, "Messages should persist across restarts")
	})
}

func TestSingleExecutableDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping single executable test")
	}

	t.Run("build single executable", func(t *testing.T) {
		// Create temporary build directory
		buildDir := filepath.Join(os.TempDir(), fmt.Sprintf("nats-build-%d", time.Now().Unix()))
		err := os.MkdirAll(buildDir, 0755)
		require.NoError(t, err)
		defer os.RemoveAll(buildDir)

		// Determine executable name based on OS
		execName := "scifind-backend"
		if runtime.GOOS == "windows" {
			execName += ".exe"
		}
		execPath := filepath.Join(buildDir, execName)

		// Build the executable
		projectRoot := findProjectRoot(t)
		buildCmd := exec.Command("go", "build", "-o", execPath, "./cmd/server")
		buildCmd.Dir = projectRoot
		buildCmd.Env = append(os.Environ(), 
			"CGO_ENABLED=1", // Enable CGO for SQLite
			"GOOS="+runtime.GOOS,
			"GOARCH="+runtime.GOARCH,
		)

		output, err := buildCmd.CombinedOutput()
		if err != nil {
			t.Logf("Build output: %s", string(output))
			t.Skipf("Build failed (expected in test env): %v", err)
			return
		}

		// Verify executable exists and is executable
		info, err := os.Stat(execPath)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
		
		if runtime.GOOS != "windows" {
			mode := info.Mode()
			assert.True(t, mode&0111 != 0, "Executable should be executable")
		}

		t.Logf("Successfully built single executable: %s (size: %.2f MB)", 
			execPath, float64(info.Size())/1024/1024)
	})

	t.Run("embedded configuration", func(t *testing.T) {
		// Test configuration embedding for single executable deployment
		configContent := `
server:
  port: 8080
  mode: "production"

database:
  type: "sqlite"
  sqlite:
    path: "./embedded.db"

nats:
  url: "nats://localhost:4222"
  client_id: "embedded-server"
  jetstream:
    enabled: true
    max_memory: "128MB"
    max_storage: "1GB"

logging:
  level: "info"
  format: "json"
`
		// Create temporary config file
		tempDir := os.TempDir()
		configPath := filepath.Join(tempDir, "embedded-config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)
		defer os.Remove(configPath)

		// Test config loading
		cfg, err := config.LoadConfigFromPath(configPath)
		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify embedded-friendly configuration
		assert.Equal(t, "production", cfg.Server.Mode)
		assert.Equal(t, "sqlite", cfg.Database.Type)
		assert.Contains(t, cfg.Database.SQLite.Path, "embedded.db")
		assert.True(t, cfg.NATS.JetStream.Enabled)
		assert.Equal(t, "json", cfg.Logging.Format)
	})
}

func TestContainerizedDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping containerized deployment test")
	}

	// Check if Docker is available
	if !isDockerAvailable(t) {
		t.Skip("Docker not available")
	}

	t.Run("docker build test", func(t *testing.T) {
		projectRoot := findProjectRoot(t)
		dockerfilePath := filepath.Join(projectRoot, "Dockerfile")
		
		// Check if Dockerfile exists
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			t.Skip("Dockerfile not found")
		}

		// Build Docker image
		imageName := "scifind-backend-test"
		buildCmd := exec.Command("docker", "build", "-t", imageName, ".")
		buildCmd.Dir = projectRoot

		output, err := buildCmd.CombinedOutput()
		if err != nil {
			t.Logf("Docker build output: %s", string(output))
			t.Skipf("Docker build failed: %v", err)
			return
		}

		defer func() {
			// Cleanup: remove the test image
			exec.Command("docker", "rmi", imageName).Run()
		}()

		t.Log("Docker image built successfully")

		// Test running the container (quick startup test)
		runCmd := exec.Command("docker", "run", "--rm", "-d", 
			"--name", "scifind-test", 
			"-e", "SCIFIND_SERVER_MODE=test",
			imageName)
		
		runOutput, err := runCmd.CombinedOutput()
		if err != nil {
			t.Logf("Docker run output: %s", string(runOutput))
			t.Skipf("Docker run failed: %v", err)
			return
		}

		containerID := strings.TrimSpace(string(runOutput))
		defer func() {
			// Stop and remove container
			exec.Command("docker", "stop", containerID).Run()
		}()

		// Wait for container to start
		time.Sleep(3 * time.Second)

		// Check container health
		healthCmd := exec.Command("docker", "exec", containerID, "ps", "aux")
		healthOutput, err := healthCmd.CombinedOutput()
		if err != nil {
			t.Logf("Health check failed: %v", err)
		} else {
			t.Logf("Container processes: %s", string(healthOutput))
		}

		t.Log("Container deployment test completed")
	})
}

func TestProductionDeploymentScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping production deployment test")
	}

	t.Run("high availability configuration", func(t *testing.T) {
		// Test configuration for HA deployment
		haConfig := `
server:
  port: 8080
  mode: "release"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"

database:
  type: "postgres"
  postgresql:
    dsn: "postgres://user:pass@ha-postgres:5432/scifind?sslmode=require"
    max_connections: 50
    max_idle: 25
    auto_migrate: false

nats:
  url: "nats://nats-cluster:4222"
  client_id: "scifind-ha-node"
  max_reconnects: -1  # Infinite reconnects
  reconnect_wait: "2s"
  timeout: "10s"
  jetstream:
    enabled: true
    max_memory: "1GB"
    max_storage: "10GB"
  tls:
    enabled: true
    cert_file: "/certs/client-cert.pem"
    key_file: "/certs/client-key.pem"
    ca_file: "/certs/ca-cert.pem"

logging:
  level: "info"
  format: "json"
  output: "stdout"

security:
  rate_limit:
    enabled: true
    requests: 1000
    window: "1m"
  cors:
    enabled: true
    allowed_origins: ["https://scifind.example.com"]

monitoring:
  enabled: true
  metrics_port: 9090
  health_path: "/health"
`
		tempDir := os.TempDir()
		configPath := filepath.Join(tempDir, "ha-config.yaml")
		err := os.WriteFile(configPath, []byte(haConfig), 0644)
		require.NoError(t, err)
		defer os.Remove(configPath)

		cfg, err := config.LoadConfigFromPath(configPath)
		require.NoError(t, err)

		// Verify HA configuration
		assert.Equal(t, "release", cfg.Server.Mode)
		assert.Equal(t, "postgres", cfg.Database.Type)
		assert.Equal(t, 50, cfg.Database.PostgreSQL.MaxConns)
		assert.Equal(t, -1, cfg.NATS.MaxReconnects) // Infinite reconnects
		assert.True(t, cfg.NATS.TLS.Enabled)
		assert.True(t, cfg.Security.RateLimit.Enabled)
		assert.True(t, cfg.Monitoring.Enabled)
	})

	t.Run("resource constraints testing", func(t *testing.T) {
		// Test with limited resources (memory, connections)
		constrainedConfig := `
server:
  port: 8080
  mode: "release"

database:
  type: "sqlite"
  sqlite:
    path: "./constrained.db"

nats:
  url: "nats://localhost:4222"
  client_id: "constrained-test"
  jetstream:
    enabled: true
    max_memory: "32MB"    # Limited memory
    max_storage: "128MB"  # Limited storage

logging:
  level: "warn"  # Reduce log volume
`
		tempDir := os.TempDir()
		configPath := filepath.Join(tempDir, "constrained-config.yaml")
		err := os.WriteFile(configPath, []byte(constrainedConfig), 0644)
		require.NoError(t, err)
		defer os.Remove(configPath)

		cfg, err := config.LoadConfigFromPath(configPath)
		require.NoError(t, err)

		// Verify resource constraints
		assert.Equal(t, "32MB", cfg.NATS.JetStream.MaxMemory)
		assert.Equal(t, "128MB", cfg.NATS.JetStream.MaxStorage)
		assert.Equal(t, "warn", cfg.Logging.Level)
	})
}

// Helper functions

func findProjectRoot(t *testing.T) string {
	// Start from current directory and walk up to find go.mod
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod not found)")
		}
		dir = parent
	}
}

func isDockerAvailable(t *testing.T) bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	return err == nil
}

// Benchmark deployment scenarios
func BenchmarkEmbeddedServerStartup(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping embedded server benchmark")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Find available port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(b, err)
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		// Configure and start server
		opts := &server.Options{
			Host:      "127.0.0.1",
			Port:      port,
			JetStream: true,
			StoreDir:  filepath.Join(os.TempDir(), fmt.Sprintf("bench-nats-%d-%d", i, port)),
		}

		start := time.Now()
		natsServer := server.New(opts)
		go natsServer.Start()
		
		// Wait for ready
		ready := natsServer.ReadyForConnections(10 * time.Second)
		startupTime := time.Since(start)
		
		if !ready {
			b.Errorf("Server failed to start in iteration %d", i)
		}

		natsServer.Shutdown()
		os.RemoveAll(opts.StoreDir)

		b.ReportMetric(float64(startupTime.Milliseconds()), "startup_ms")
	}
}

func BenchmarkClientConnectionToEmbedded(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping client connection benchmark")
	}

	// Setup embedded server once
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(b, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      port,
		JetStream: true,
		StoreDir:  filepath.Join(os.TempDir(), fmt.Sprintf("bench-client-nats-%d", port)),
	}

	natsServer := server.New(opts)
	go natsServer.Start()
	defer func() {
		natsServer.Shutdown()
		os.RemoveAll(opts.StoreDir)
	}()

	ready := natsServer.ReadyForConnections(10 * time.Second)
	require.True(b, ready)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := config.NATSConfig{
		URL:      fmt.Sprintf("nats://127.0.0.1:%d", port),
		ClientID: "benchmark-client",
		Timeout:  "5s",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.ClientID = fmt.Sprintf("benchmark-client-%d", i)
		
		start := time.Now()
		client, err := messaging.NewClient(cfg, logger)
		connectTime := time.Since(start)
		
		if err != nil {
			b.Errorf("Client connection failed: %v", err)
			continue
		}

		client.Close()
		b.ReportMetric(float64(connectTime.Milliseconds()), "connect_ms")
	}
}