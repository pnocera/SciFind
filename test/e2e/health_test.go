package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"

	"scifind-backend/test/testutil"
)

func TestHealthEndpoint_E2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	// Setup test environment
	dbUtil := testutil.SetupTestDatabase(t, false)
	defer dbUtil.Cleanup()

	natsUtil := testutil.SetupTestNATS(t)
	defer natsUtil.Cleanup()

	httpUtil := testutil.SetupTestHTTPServer(t)

	// Setup health endpoints
	setupHealthRoutes(httpUtil, dbUtil, natsUtil)
	
	httpUtil.StartServer()
	defer httpUtil.StopServer()

	t.Run("health check returns 200", func(t *testing.T) {
		resp := httpUtil.MakeRequest(t, "GET", "/health", nil, nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var healthResp map[string]interface{}
		httpUtil.AssertJSONResponse(t, resp, http.StatusOK, &healthResp)

		assert.Equal(t, "ok", healthResp["status"])
		assert.Contains(t, healthResp, "timestamp")
		assert.Contains(t, healthResp, "checks")
	})

	t.Run("health check includes all components", func(t *testing.T) {
		resp := httpUtil.MakeRequest(t, "GET", "/health", nil, nil)
		
		var healthResp map[string]interface{}
		httpUtil.AssertJSONResponse(t, resp, http.StatusOK, &healthResp)

		checks := healthResp["checks"].(map[string]interface{})
		
		// Should include database check
		assert.Contains(t, checks, "database")
		dbCheck := checks["database"].(map[string]interface{})
		assert.Equal(t, "ok", dbCheck["status"])

		// Should include NATS check
		assert.Contains(t, checks, "messaging")
		natsCheck := checks["messaging"].(map[string]interface{})
		assert.Equal(t, "ok", natsCheck["status"])
	})

	t.Run("readiness check", func(t *testing.T) {
		resp := httpUtil.MakeRequest(t, "GET", "/ready", nil, nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var readyResp map[string]interface{}
		httpUtil.AssertJSONResponse(t, resp, http.StatusOK, &readyResp)

		assert.Equal(t, "ready", readyResp["status"])
	})

	t.Run("liveness check", func(t *testing.T) {
		resp := httpUtil.MakeRequest(t, "GET", "/live", nil, nil)
		assert.Equal(t, http.StatusOK, resp.Code)

		var liveResp map[string]interface{}
		httpUtil.AssertJSONResponse(t, resp, http.StatusOK, &liveResp)

		assert.Equal(t, "alive", liveResp["status"])
	})
}

func TestHealthEndpoint_WithFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	httpUtil := testutil.SetupTestHTTPServer(t)

	// Setup health endpoints with simulated failures
	setupHealthRoutesWithFailures(httpUtil)
	
	httpUtil.StartServer()
	defer httpUtil.StopServer()

	t.Run("health check returns 503 when components fail", func(t *testing.T) {
		resp := httpUtil.MakeRequest(t, "GET", "/health", nil, nil)
		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

		var healthResp map[string]interface{}
		httpUtil.AssertJSONResponse(t, resp, http.StatusServiceUnavailable, &healthResp)

		assert.Equal(t, "error", healthResp["status"])
		
		checks := healthResp["checks"].(map[string]interface{})
		dbCheck := checks["database"].(map[string]interface{})
		assert.Equal(t, "error", dbCheck["status"])
		assert.Contains(t, dbCheck, "error")
	})
}

func TestHealthEndpoint_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	httpUtil := testutil.SetupTestHTTPServer(t)

	// Setup health endpoints with slow responses
	setupHealthRoutesWithTimeout(httpUtil)
	
	httpUtil.StartServer()
	defer httpUtil.StopServer()

	t.Run("health check times out", func(t *testing.T) {
		start := time.Now()
		resp := httpUtil.MakeRequest(t, "GET", "/health", nil, nil)
		duration := time.Since(start)

		// Should timeout and return within reasonable time
		assert.Less(t, duration, 6*time.Second)
		assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

		var healthResp map[string]interface{}
		httpUtil.AssertJSONResponse(t, resp, http.StatusServiceUnavailable, &healthResp)

		assert.Equal(t, "error", healthResp["status"])
	})
}

// Helper functions to setup health routes

func setupHealthRoutes(httpUtil *testutil.HTTPTestUtil, dbUtil *testutil.DatabaseTestUtil, natsUtil *testutil.NATSTestUtil) {
	router := httpUtil.Router()

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		checks := make(map[string]interface{})

		// Database health check
		if err := dbUtil.DB().WithContext(ctx).Exec("SELECT 1").Error; err != nil {
			checks["database"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			checks["database"] = map[string]interface{}{
				"status": "ok",
			}
		}

		// NATS health check
		if natsUtil.Connection().Status() != nats.CONNECTED {
			checks["messaging"] = map[string]interface{}{
				"status": "error",
				"error":  "NATS not connected",
			}
		} else {
			checks["messaging"] = map[string]interface{}{
				"status": "ok",
			}
		}

		// Overall status
		status := "ok"
		statusCode := http.StatusOK
		for _, check := range checks {
			if checkMap := check.(map[string]interface{}); checkMap["status"] == "error" {
				status = "error"
				statusCode = http.StatusServiceUnavailable
				break
			}
		}

		c.JSON(statusCode, map[string]interface{}{
			"status":    status,
			"timestamp": time.Now().UTC(),
			"checks":    checks,
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "ready",
			"timestamp": time.Now().UTC(),
		})
	})

	router.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "alive",
			"timestamp": time.Now().UTC(),
		})
	})
}

func setupHealthRoutesWithFailures(httpUtil *testutil.HTTPTestUtil) {
	router := httpUtil.Router()

	router.GET("/health", func(c *gin.Context) {
		checks := map[string]interface{}{
			"database": map[string]interface{}{
				"status": "error",
				"error":  "database connection failed",
			},
			"messaging": map[string]interface{}{
				"status": "error", 
				"error":  "NATS connection failed",
			},
		}

		c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "error",
			"timestamp": time.Now().UTC(),
			"checks":    checks,
		})
	})
}

func setupHealthRoutesWithTimeout(httpUtil *testutil.HTTPTestUtil) {
	router := httpUtil.Router()

	router.GET("/health", func(c *gin.Context) {
		// Simulate slow health check
		time.Sleep(10 * time.Second)

		c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status": "error",
			"error":  "timeout",
		})
	})
}