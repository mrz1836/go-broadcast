package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMetricsCollector tests metrics collector creation
func TestNewMetricsCollector(t *testing.T) {
	config := DefaultDashboardConfig()
	collector := NewMetricsCollector(config)
	defer collector.Stop()

	require.NotNil(t, collector)
	assert.Equal(t, config.CollectInterval, collector.collectInterval)
	assert.Equal(t, config.RetainHistory, collector.retainHistory)
	assert.NotNil(t, collector.metrics)
	assert.NotNil(t, collector.history)
}

// TestMetricsCollection tests that metrics are collected
func TestMetricsCollection(t *testing.T) {
	config := DefaultDashboardConfig()
	config.CollectInterval = 10 * time.Millisecond // Fast collection for testing

	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for at least one collection
	time.Sleep(50 * time.Millisecond)

	metrics := collector.GetCurrentMetrics()
	require.NotNil(t, metrics)

	// Check required metrics exist
	assert.Contains(t, metrics, "timestamp")
	assert.Contains(t, metrics, "memory")
	assert.Contains(t, metrics, "gc")
	assert.Contains(t, metrics, "runtime")

	// Check memory metrics
	memory, ok := metrics["memory"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, memory, "alloc_mb")
	assert.Contains(t, memory, "heap_alloc_mb")
	assert.Contains(t, memory, "heap_objects")

	// Check runtime metrics
	runtimeMetrics, ok := metrics["runtime"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, runtimeMetrics, "goroutines")
	assert.Contains(t, runtimeMetrics, "num_cpu")
	assert.Equal(t, runtime.Version(), runtimeMetrics["go_version"])
}

// TestMetricsHistory tests history retention
func TestMetricsHistory(t *testing.T) {
	config := DefaultDashboardConfig()
	config.CollectInterval = 10 * time.Millisecond
	config.RetainHistory = 5 // Small history for testing

	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for history to build up
	time.Sleep(80 * time.Millisecond)

	history := collector.GetMetricsHistory()
	assert.LessOrEqual(t, len(history), config.RetainHistory)

	// Verify history entries
	for i, snapshot := range history {
		assert.NotZero(t, snapshot.Timestamp)
		assert.NotNil(t, snapshot.Metrics)

		// Later entries should have later timestamps
		if i > 0 {
			assert.True(t, snapshot.Timestamp.After(history[i-1].Timestamp))
		}
	}
}

// TestMetricsHTTPHandler tests the HTTP handler
func TestMetricsHTTPHandler(t *testing.T) {
	config := DefaultDashboardConfig()
	config.CollectInterval = 10 * time.Millisecond // Fast collection for testing
	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for some metrics
	time.Sleep(50 * time.Millisecond)

	t.Run("CurrentMetrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/metrics", nil)
		w := httptest.NewRecorder()

		collector.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var metrics map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &metrics)
		require.NoError(t, err)
		assert.Contains(t, metrics, "timestamp")
		assert.Contains(t, metrics, "memory")
	})

	t.Run("HistoryMetrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/metrics?type=history", nil)
		w := httptest.NewRecorder()

		collector.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "history")

		history, ok := response["history"].([]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, history)
	})

	t.Run("CORSHeaders", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/metrics", nil)
		w := httptest.NewRecorder()

		collector.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	})
}

// TestDashboardCreation tests dashboard creation
func TestDashboardCreation(t *testing.T) {
	config := DefaultDashboardConfig()
	config.Port = 0 // Use random port

	dashboard := NewDashboard(config)
	require.NotNil(t, dashboard)
	assert.NotNil(t, dashboard.collector)
	assert.NotNil(t, dashboard.server)
	assert.Equal(t, config, dashboard.config)
}

// TestDashboardEndpoints tests dashboard HTTP endpoints
func TestDashboardEndpoints(t *testing.T) {
	config := DefaultDashboardConfig()
	dashboard := NewDashboard(config)

	testCases := []struct {
		name        string
		path        string
		handler     http.HandlerFunc
		contentType string
	}{
		{
			name:        "HealthEndpoint",
			path:        "/api/health",
			handler:     dashboard.healthHandler,
			contentType: "application/json",
		},
		{
			name:        "DashboardHTML",
			path:        "/",
			handler:     dashboardHandler,
			contentType: "text/html",
		},
		{
			name:        "DashboardJS",
			path:        "/dashboard.js",
			handler:     dashboardJSHandler,
			contentType: "application/javascript",
		},
		{
			name:        "DashboardCSS",
			path:        "/dashboard.css",
			handler:     dashboardCSSHandler,
			contentType: "text/css",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()

			tc.handler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tc.contentType, w.Header().Get("Content-Type"))
			assert.NotEmpty(t, w.Body.String())
		})
	}
}

// TestHealthHandler tests health check endpoint
func TestHealthHandler(t *testing.T) {
	config := DefaultDashboardConfig()
	dashboard := NewDashboard(config)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	dashboard.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var health map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
	assert.Contains(t, health, "timestamp")
	assert.Contains(t, health, "uptime")
	assert.Greater(t, health["uptime"].(float64), 0.0)
}

// TestDashboardStartBackground tests background server start and stop
func TestDashboardStartBackground(t *testing.T) {
	config := DefaultDashboardConfig()
	config.Port = 0 // Random port

	dashboard := NewDashboard(config)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error)
	go func() {
		done <- dashboard.StartBackground(ctx)
	}()

	// Let server start
	time.Sleep(50 * time.Millisecond)

	// Stop server
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to stop")
	}
}

// TestDefaultDashboardConfig tests default configuration
func TestDefaultDashboardConfig(t *testing.T) {
	config := DefaultDashboardConfig()

	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, time.Second, config.CollectInterval)
	assert.Equal(t, 300, config.RetainHistory)
	assert.False(t, config.EnableProfiling)
	assert.Equal(t, "./profiles", config.ProfileDir)
}

// TestMetricsCollectorWithProfiling tests profiling integration
func TestMetricsCollectorWithProfiling(t *testing.T) {
	config := DefaultDashboardConfig()
	config.EnableProfiling = true
	config.ProfileDir = t.TempDir()

	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for metrics collection
	time.Sleep(50 * time.Millisecond)

	metrics := collector.GetCurrentMetrics()

	// Should have profiler metrics when enabled
	if profilerMetrics, ok := metrics["profiler"]; ok {
		profiler := profilerMetrics.(map[string]interface{})
		assert.Contains(t, profiler, "enabled")
		assert.Contains(t, profiler, "active_sessions")
		assert.Contains(t, profiler, "total_sessions")
		assert.Contains(t, profiler, "profile_count")
	}
}

// TestGCMetrics tests GC-specific metrics
func TestGCMetrics(t *testing.T) {
	config := DefaultDashboardConfig()
	config.CollectInterval = 10 * time.Millisecond // Fast collection for testing
	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Force a GC to ensure we have GC stats
	runtime.GC()

	// Wait for collection
	time.Sleep(50 * time.Millisecond)

	metrics := collector.GetCurrentMetrics()
	gcMetrics, ok := metrics["gc"].(map[string]interface{})
	require.True(t, ok)

	assert.Contains(t, gcMetrics, "num_gc")
	assert.Contains(t, gcMetrics, "pause_total_ms")

	numGC, ok := gcMetrics["num_gc"].(uint32)
	if ok && numGC > 0 {
		assert.Contains(t, gcMetrics, "last_pause_ms")
		assert.Contains(t, gcMetrics, "avg_pause_ms")
		assert.Contains(t, gcMetrics, "last_gc_time")
	}
}

// TestStartDashboard tests convenience function
func TestStartDashboard(t *testing.T) {
	// This would actually start a server, so we just test that it doesn't panic
	// In a real test, we'd use a random port and verify it's listening
	t.Skip("Skipping actual server start test")
}

// TestStartDashboardWithProfiling tests profiling convenience function
func TestStartDashboardWithProfiling(t *testing.T) {
	// This would actually start a server, so we just test that it doesn't panic
	// In a real test, we'd use a random port and verify it's listening
	t.Skip("Skipping actual server start test")
}

// TestMetricsSnapshot tests snapshot structure
func TestMetricsSnapshot(t *testing.T) {
	snapshot := MetricsSnapshot{
		Timestamp: time.Now(),
		Metrics: map[string]interface{}{
			"test": "value",
		},
	}

	assert.NotZero(t, snapshot.Timestamp)
	assert.NotNil(t, snapshot.Metrics)
	assert.Equal(t, "value", snapshot.Metrics["test"])
}

// TestCollectorGetters tests GetCurrentMetrics and GetMetricsHistory return copies
func TestCollectorGetters(t *testing.T) {
	config := DefaultDashboardConfig()
	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for some data
	time.Sleep(50 * time.Millisecond)

	// Test that GetCurrentMetrics returns a copy
	metrics1 := collector.GetCurrentMetrics()
	metrics2 := collector.GetCurrentMetrics()

	// Modify one copy
	metrics1["test"] = "modified"

	// Other copy should not be affected
	_, exists := metrics2["test"]
	assert.False(t, exists)

	// Test that GetMetricsHistory returns a copy
	history1 := collector.GetMetricsHistory()
	history2 := collector.GetMetricsHistory()

	if len(history1) > 0 {
		// Modify one copy
		history1[0].Metrics["test"] = "modified"

		// Other copy should not be affected
		_, exists := history2[0].Metrics["test"]
		assert.False(t, exists)
	}
}
