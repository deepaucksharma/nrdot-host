package supervisor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestHealthChecker_Check(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := DefaultHealthCheckerConfig()
	config.Endpoint = server.URL + "/health"
	config.Timeout = 2 * time.Second

	checker := NewHealthChecker(config, logger)

	ctx := context.Background()

	// Test successful health check
	err := checker.Check(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestHealthChecker_CheckFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := DefaultHealthCheckerConfig()
	config.Endpoint = server.URL + "/health"

	checker := NewHealthChecker(config, logger)

	ctx := context.Background()

	// Test failed health check
	err := checker.Check(ctx)
	if err == nil {
		t.Error("Expected health check to fail")
	}
}

func TestHealthChecker_CheckTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHealthCheckerConfig()
	config.Endpoint = server.URL + "/health"
	config.Timeout = 100 * time.Millisecond

	checker := NewHealthChecker(config, logger)

	ctx := context.Background()

	// Test timeout
	err := checker.Check(ctx)
	if err == nil {
		t.Error("Expected health check to timeout")
	}
}

func TestHealthChecker_Monitor(t *testing.T) {
	logger := zaptest.NewLogger(t)

	failCount := 0
	mu := &sync.Mutex{}
	// Create test server that fails initially then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if failCount < 3 {
			failCount++
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := DefaultHealthCheckerConfig()
	config.Endpoint = server.URL + "/health"
	config.Interval = 50 * time.Millisecond // Faster for testing
	config.FailureThreshold = 3

	checker := NewHealthChecker(config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start monitoring
	errCh := checker.Monitor(ctx)

	// Should receive error after threshold failures
	select {
	case err := <-errCh:
		if err == nil {
			t.Error("Expected error from monitor")
		}
	case <-ctx.Done():
		t.Error("Timeout waiting for health check failure")
	}
}

func TestHealthChecker_MonitorContextCancel(t *testing.T) {
	logger := zaptest.NewLogger(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultHealthCheckerConfig()
	config.Endpoint = server.URL + "/health"
	config.Interval = 100 * time.Millisecond

	checker := NewHealthChecker(config, logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Start monitoring
	errCh := checker.Monitor(ctx)

	// Cancel context
	cancel()

	// Should close error channel
	select {
	case _, ok := <-errCh:
		if ok {
			t.Error("Expected error channel to be closed")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for channel close")
	}
}

func TestHealthChecker_WaitForHealthy(t *testing.T) {
	logger := zaptest.NewLogger(t)

	healthy := false
	// Create test server that becomes healthy after delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	config := DefaultHealthCheckerConfig()
	config.Endpoint = server.URL + "/health"

	checker := NewHealthChecker(config, logger)

	ctx := context.Background()

	// Set healthy after delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		healthy = true
	}()

	// Wait for healthy
	err := checker.WaitForHealthy(ctx, 2*time.Second)
	if err != nil {
		t.Errorf("Failed to wait for healthy: %v", err)
	}
}

func TestHealthChecker_WaitForHealthyTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create test server that's always unhealthy
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := DefaultHealthCheckerConfig()
	config.Endpoint = server.URL + "/health"

	checker := NewHealthChecker(config, logger)

	ctx := context.Background()

	// Wait for healthy with short timeout
	err := checker.WaitForHealthy(ctx, 500*time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestHealthChecker_InvalidEndpoint(t *testing.T) {
	logger := zaptest.NewLogger(t)

	config := DefaultHealthCheckerConfig()
	config.Endpoint = "http://invalid-endpoint-that-does-not-exist:99999/health"
	config.Timeout = 100 * time.Millisecond

	checker := NewHealthChecker(config, logger)

	ctx := context.Background()

	// Check should fail
	err := checker.Check(ctx)
	if err == nil {
		t.Error("Expected error for invalid endpoint")
	}
}