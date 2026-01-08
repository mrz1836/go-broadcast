package monitoring

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForHistory polls until history has at least minEntries, with timeout.
// Returns true if the condition was met, false on timeout.
// This is more reliable than time.Sleep() under race detection with t.Parallel().
func waitForHistory(collector *MetricsCollector, minEntries int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(collector.GetMetricsHistory()) >= minEntries {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return false
}

// TestMetricsConcurrentReadWrite tests concurrent reads and writes to metrics.
// This test specifically targets the deep copy fix for race conditions.
func TestMetricsConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	config := DefaultDashboardConfig()
	config.CollectInterval = 5 * time.Millisecond
	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for initial metrics collection using polling (more reliable than fixed sleep)
	require.True(t, waitForHistory(collector, 1, 500*time.Millisecond),
		"Timed out waiting for initial metrics collection")

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				// Get metrics - should be a deep copy
				metrics := collector.GetCurrentMetrics()

				// Modify returned data - should not affect other copies or internal state
				if mem, ok := metrics["memory"].(map[string]interface{}); ok {
					mem["test_value"] = 42
				}

				// Get history - should also be a deep copy
				history := collector.GetMetricsHistory()
				if len(history) > 0 {
					history[0].Metrics["test"] = "value"
				}
			}
		}()
	}

	wg.Wait()
}

// TestMetricsDeepCopyIsolation verifies that returned metrics are truly isolated.
func TestMetricsDeepCopyIsolation(t *testing.T) {
	t.Parallel()

	config := DefaultDashboardConfig()
	config.CollectInterval = 5 * time.Millisecond
	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for metrics collection
	time.Sleep(20 * time.Millisecond)

	// Get two copies of metrics
	metrics1 := collector.GetCurrentMetrics()
	metrics2 := collector.GetCurrentMetrics()

	// Modify nested map in first copy
	if mem, ok := metrics1["memory"].(map[string]interface{}); ok {
		mem["injected_value"] = "should_not_appear"
	}

	// Verify second copy was not affected
	if mem2, ok := metrics2["memory"].(map[string]interface{}); ok {
		_, exists := mem2["injected_value"]
		assert.False(t, exists, "Second copy should not contain injected value")
	}

	// Verify internal state was not affected
	metrics3 := collector.GetCurrentMetrics()
	if mem3, ok := metrics3["memory"].(map[string]interface{}); ok {
		_, exists := mem3["injected_value"]
		assert.False(t, exists, "Internal state should not contain injected value")
	}
}

// TestHistoryDeepCopyIsolation verifies that returned history is truly isolated.
func TestHistoryDeepCopyIsolation(t *testing.T) {
	t.Parallel()

	config := DefaultDashboardConfig()
	config.CollectInterval = 5 * time.Millisecond
	config.RetainHistory = 10
	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for history to build up using polling (more reliable than fixed sleep under race detection)
	require.True(t, waitForHistory(collector, 1, 500*time.Millisecond),
		"Timed out waiting for history to be populated")

	history1 := collector.GetMetricsHistory()
	history2 := collector.GetMetricsHistory()

	require.NotEmpty(t, history1, "History should not be empty")

	// Modify nested map in first history copy
	if mem, ok := history1[0].Metrics["memory"].(map[string]interface{}); ok {
		mem["injected_value"] = "should_not_appear"
	}

	// Verify second copy was not affected
	if mem2, ok := history2[0].Metrics["memory"].(map[string]interface{}); ok {
		_, exists := mem2["injected_value"]
		assert.False(t, exists, "Second history copy should not contain injected value")
	}
}

// TestProfilerConcurrentAccess tests concurrent access when profiler is enabled.
func TestProfilerConcurrentAccess(t *testing.T) {
	t.Parallel()

	config := DefaultDashboardConfig()
	config.CollectInterval = 1 * time.Millisecond
	config.EnableProfiling = true
	config.ProfileDir = t.TempDir()

	collector := NewMetricsCollector(config)
	defer collector.Stop()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				metrics := collector.GetCurrentMetrics()
				// Access profiler metrics if present
				if profiler, ok := metrics["profiler"].(map[string]interface{}); ok {
					_ = profiler["enabled"]
					_ = profiler["active_sessions"]
				}
			}
		}()
	}

	wg.Wait()
}

// TestConcurrentMetricsAndHistory tests concurrent access to both current metrics and history.
func TestConcurrentMetricsAndHistory(t *testing.T) {
	t.Parallel()

	config := DefaultDashboardConfig()
	config.CollectInterval = 2 * time.Millisecond
	config.RetainHistory = 50
	collector := NewMetricsCollector(config)
	defer collector.Stop()

	// Wait for some history to build using polling (more reliable than fixed sleep)
	require.True(t, waitForHistory(collector, 1, 500*time.Millisecond),
		"Timed out waiting for history to build")

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Half the goroutines read current metrics
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				metrics := collector.GetCurrentMetrics()
				// Access and modify - should be safe
				if gc, ok := metrics["gc"].(map[string]interface{}); ok {
					gc["test"] = j
				}
			}
		}()
	}

	// Half the goroutines read history
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				history := collector.GetMetricsHistory()
				// Access and modify - should be safe
				if len(history) > 0 {
					history[0].Metrics["test"] = j
				}
			}
		}()
	}

	wg.Wait()
}

// TestCollectorStopRace tests stopping the collector while metrics are being read.
func TestCollectorStopRace(t *testing.T) {
	t.Parallel()

	config := DefaultDashboardConfig()
	config.CollectInterval = 1 * time.Millisecond
	collector := NewMetricsCollector(config)

	// Wait for collection to start
	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(2)

	// One goroutine reads metrics continuously
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = collector.GetCurrentMetrics()
			_ = collector.GetMetricsHistory()
		}
	}()

	// Another goroutine stops the collector
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		collector.Stop()
	}()

	wg.Wait()
}

// TestMultipleStopCalls tests that Stop() is idempotent.
func TestMultipleStopCalls(t *testing.T) {
	t.Parallel()

	config := DefaultDashboardConfig()
	collector := NewMetricsCollector(config)

	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(10)

	// Multiple goroutines call Stop concurrently
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			assert.NotPanics(t, func() {
				collector.Stop()
			})
		}()
	}

	wg.Wait()
}

// TestDeepCopyNilMap tests that deepCopyMetrics handles nil gracefully.
func TestDeepCopyNilMap(t *testing.T) {
	t.Parallel()

	result := deepCopyMetrics(nil)
	assert.Nil(t, result, "deepCopyMetrics(nil) should return nil")
}

// TestDeepCopyNestedMaps tests that deeply nested maps are properly copied.
func TestDeepCopyNestedMaps(t *testing.T) {
	t.Parallel()

	original := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "value",
			},
		},
	}

	copied := deepCopyMetrics(original)

	// Modify nested value in copy
	level1 := copied["level1"].(map[string]interface{})
	level2 := level1["level2"].(map[string]interface{})
	level2["level3"] = "modified"

	// Original should not be affected
	origLevel1 := original["level1"].(map[string]interface{})
	origLevel2 := origLevel1["level2"].(map[string]interface{})
	assert.Equal(t, "value", origLevel2["level3"], "Original should not be modified")
}
