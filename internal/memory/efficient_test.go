package memory

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errLoadFailed = errors.New("load failed")

func TestNewStringIntern(t *testing.T) {
	si := NewStringIntern()

	require.NotNil(t, si)
	require.Equal(t, DefaultStringInternSize, si.maxSize)
	require.NotNil(t, si.values)
	require.Empty(t, si.values)
}

func TestNewStringInternWithSize(t *testing.T) {
	tests := []struct {
		name    string
		maxSize int
		want    int
	}{
		{
			name:    "CustomSize",
			maxSize: 5000,
			want:    5000,
		},
		{
			name:    "ZeroSize",
			maxSize: 0,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			si := NewStringInternWithSize(tt.maxSize)

			require.NotNil(t, si)
			require.Equal(t, tt.want, si.maxSize)
			require.NotNil(t, si.values)
		})
	}
}

func TestStringInternBasicFunctionality(t *testing.T) {
	si := NewStringInternWithSize(10)

	tests := []struct {
		name   string
		input  string
		verify func(t *testing.T, result string, si *StringIntern)
	}{
		{
			name:  "FirstIntern",
			input: "test-string",
			verify: func(t *testing.T, result string, si *StringIntern) {
				require.Equal(t, "test-string", result)
				stats := si.GetStats()
				require.Equal(t, int64(1), stats.Misses)
				require.Equal(t, int64(0), stats.Hits)
				require.Equal(t, int64(1), stats.Size)
			},
		},
		{
			name:  "DuplicateIntern",
			input: "test-string",
			verify: func(t *testing.T, result string, si *StringIntern) {
				require.Equal(t, "test-string", result)
				stats := si.GetStats()
				require.Equal(t, int64(1), stats.Misses)
				require.Equal(t, int64(1), stats.Hits)
				require.Equal(t, int64(1), stats.Size)
			},
		},
		{
			name:  "DifferentString",
			input: "another-string",
			verify: func(t *testing.T, result string, si *StringIntern) {
				require.Equal(t, "another-string", result)
				stats := si.GetStats()
				require.Equal(t, int64(2), stats.Misses)
				require.Equal(t, int64(1), stats.Hits)
				require.Equal(t, int64(2), stats.Size)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := si.Intern(tt.input)
			tt.verify(t, result, si)
		})
	}
}

func TestStringInternEviction(t *testing.T) {
	si := NewStringInternWithSize(3) // Small size to trigger eviction

	// Fill the cache
	si.Intern("string1")
	si.Intern("string2")
	si.Intern("string3")

	stats := si.GetStats()
	require.Equal(t, int64(3), stats.Size)
	require.Equal(t, int64(0), stats.Evicted)

	// This should trigger eviction
	si.Intern("string4")

	stats = si.GetStats()
	require.Equal(t, int64(3), stats.Size) // Size should remain at max
	require.Positive(t, stats.Evicted)     // Some strings should be evicted
}

func TestStringInternConcurrency(t *testing.T) {
	si := NewStringIntern()
	numGoroutines := 10
	stringsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent access
	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer wg.Done()

			for j := 0; j < stringsPerGoroutine; j++ {
				str := si.Intern("test-string")
				assert.Equal(t, "test-string", str)
			}
		}(i)
	}

	wg.Wait()

	stats := si.GetStats()
	require.Equal(t, int64(1), stats.Size)                                   // Only one unique string
	require.Equal(t, int64(1), stats.Misses)                                 // Only one miss (first intern)
	require.Equal(t, int64(numGoroutines*stringsPerGoroutine-1), stats.Hits) // All others are hits
}

func TestStringInternStatsHitRate(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(si *StringIntern)
		expected float64
	}{
		{
			name:     "NoOperations",
			setup:    func(_ *StringIntern) {},
			expected: 0.0,
		},
		{
			name: "AllMisses",
			setup: func(si *StringIntern) {
				si.Intern("str1")
				si.Intern("str2")
				si.Intern("str3")
			},
			expected: 0.0,
		},
		{
			name: "MixedHitsAndMisses",
			setup: func(si *StringIntern) {
				si.Intern("str1") // Miss
				si.Intern("str1") // Hit
				si.Intern("str1") // Hit
				si.Intern("str2") // Miss
			},
			expected: 50.0, // 2 hits out of 4 operations
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			si := NewStringIntern()
			tt.setup(si)

			stats := si.GetStats()
			require.InDelta(t, tt.expected, stats.HitRate(), 0.1)
		})
	}
}

func TestPreallocateSlice(t *testing.T) {
	tests := []struct {
		name          string
		estimatedSize int
		want          struct {
			length   int
			capacity int
			isNil    bool
		}
	}{
		{
			name:          "PositiveSize",
			estimatedSize: 100,
			want: struct {
				length   int
				capacity int
				isNil    bool
			}{
				length:   0,
				capacity: 100,
				isNil:    false,
			},
		},
		{
			name:          "ZeroSize",
			estimatedSize: 0,
			want: struct {
				length   int
				capacity int
				isNil    bool
			}{
				isNil: true,
			},
		},
		{
			name:          "NegativeSize",
			estimatedSize: -10,
			want: struct {
				length   int
				capacity int
				isNil    bool
			}{
				isNil: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PreallocateSlice[string](tt.estimatedSize)

			if tt.want.isNil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Len(t, result, tt.want.length)
				require.Equal(t, tt.want.capacity, cap(result))
			}
		})
	}
}

func TestPreallocateMap(t *testing.T) {
	tests := []struct {
		name          string
		estimatedSize int
		want          struct {
			notNil bool
		}
	}{
		{
			name:          "PositiveSize",
			estimatedSize: 100,
			want: struct {
				notNil bool
			}{
				notNil: true,
			},
		},
		{
			name:          "ZeroSize",
			estimatedSize: 0,
			want: struct {
				notNil bool
			}{
				notNil: true,
			},
		},
		{
			name:          "NegativeSize",
			estimatedSize: -10,
			want: struct {
				notNil bool
			}{
				notNil: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PreallocateMap[string, string](tt.estimatedSize)

			if tt.want.notNil {
				require.NotNil(t, result)
				require.Empty(t, result)
			}
		})
	}
}

func TestReuseSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  struct {
			length   int
			capacity int
		}
	}{
		{
			name:  "FilledSlice",
			input: []string{"a", "b", "c", "d", "e"},
			want: struct {
				length   int
				capacity int
			}{
				length:   0,
				capacity: 5,
			},
		},
		{
			name:  "EmptySlice",
			input: []string{},
			want: struct {
				length   int
				capacity int
			}{
				length:   0,
				capacity: 0,
			},
		},
		{
			name:  "NilSlice",
			input: nil,
			want: struct {
				length   int
				capacity int
			}{
				length:   0,
				capacity: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReuseSlice(tt.input)

			require.Len(t, result, tt.want.length)
			require.Equal(t, tt.want.capacity, cap(result))
		})
	}
}

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()

	require.Equal(t, uint64(100), thresholds.HeapAllocMB)
	require.Equal(t, uint64(200), thresholds.HeapSysMB)
	require.Equal(t, uint64(10), thresholds.GCPercent)
	require.Equal(t, 1000, thresholds.NumGoroutines)
}

func TestAlertSeverityString(t *testing.T) {
	tests := []struct {
		severity AlertSeverity
		expected string
	}{
		{AlertInfo, "INFO"},
		{AlertWarning, "WARNING"},
		{AlertCritical, "CRITICAL"},
		{AlertSeverity(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.severity.String())
		})
	}
}

func TestNewPressureMonitor(t *testing.T) {
	thresholds := DefaultThresholds()
	alertCallback := func(_ Alert) {}

	monitor := NewPressureMonitor(thresholds, alertCallback)

	require.NotNil(t, monitor)
	require.Equal(t, thresholds, monitor.thresholds)
	require.NotNil(t, monitor.alertCallback)
	require.Equal(t, 30*time.Second, monitor.monitoringInterval)
	require.NotNil(t, monitor.stopChan)
	require.False(t, monitor.GetMonitorStats().MonitoringEnabled)
}

func TestPressureMonitorStartStop(t *testing.T) {
	monitor := NewPressureMonitor(DefaultThresholds(), nil)

	// Test starting monitoring
	monitor.StartMonitoring()
	require.True(t, monitor.GetMonitorStats().MonitoringEnabled)

	// Test starting again (should not panic)
	monitor.StartMonitoring()
	require.True(t, monitor.GetMonitorStats().MonitoringEnabled)

	// Test stopping monitoring
	monitor.StopMonitoring()
	require.False(t, monitor.GetMonitorStats().MonitoringEnabled)

	// Test stopping again (should not panic)
	monitor.StopMonitoring()
	require.False(t, monitor.GetMonitorStats().MonitoringEnabled)
}

func TestPressureMonitorForceGC(t *testing.T) {
	monitor := NewPressureMonitor(DefaultThresholds(), nil)

	initialStats := monitor.GetMonitorStats()
	require.Equal(t, int64(0), initialStats.GCForcedCount)

	monitor.ForceGC()

	updatedStats := monitor.GetMonitorStats()
	require.Equal(t, int64(1), updatedStats.GCForcedCount)
}

func TestPressureMonitorGetCurrentMemStats(t *testing.T) {
	monitor := NewPressureMonitor(DefaultThresholds(), nil)

	stats := monitor.GetCurrentMemStats()
	// Initially, stats should be zero values
	require.Equal(t, uint64(0), stats.Alloc)
}

func TestNewLazyLoader(t *testing.T) {
	loader := func() (string, error) {
		return "test-value", nil
	}

	ll := NewLazyLoader(loader)

	require.NotNil(t, ll)
	require.NotNil(t, ll.loader)
	require.False(t, ll.loaded)
	require.Equal(t, int64(0), ll.GetLoadCount())
}

func TestLazyLoaderGet(t *testing.T) {
	tests := []struct {
		name   string
		loader func() (string, error)
		want   struct {
			value   string
			hasErr  bool
			loadCnt int64
		}
	}{
		{
			name: "SuccessfulLoad",
			loader: func() (string, error) {
				return "loaded-value", nil
			},
			want: struct {
				value   string
				hasErr  bool
				loadCnt int64
			}{
				value:   "loaded-value",
				hasErr:  false,
				loadCnt: 1,
			},
		},
		{
			name: "LoadWithError",
			loader: func() (string, error) {
				return "", errLoadFailed
			},
			want: struct {
				value   string
				hasErr  bool
				loadCnt int64
			}{
				value:   "",
				hasErr:  true,
				loadCnt: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ll := NewLazyLoader(tt.loader)

			// First call should load
			value, err := ll.Get()
			if tt.want.hasErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.want.value, value)
			require.True(t, ll.IsLoaded())
			require.Equal(t, tt.want.loadCnt, ll.GetLoadCount())

			// Second call should not load again
			value2, err2 := ll.Get()
			if tt.want.hasErr {
				require.Error(t, err2)
			} else {
				require.NoError(t, err2)
			}
			require.Equal(t, tt.want.value, value2)
			require.Equal(t, tt.want.loadCnt, ll.GetLoadCount()) // Should not increment
		})
	}
}

func TestLazyLoaderReset(t *testing.T) {
	loader := func() (string, error) {
		return "test-value", nil
	}

	ll := NewLazyLoader(loader)

	// Load initially
	value, err := ll.Get()
	require.NoError(t, err)
	require.Equal(t, "test-value", value)
	require.True(t, ll.IsLoaded())

	// Reset
	ll.Reset()
	require.False(t, ll.IsLoaded())

	// Load again should work
	value2, err2 := ll.Get()
	require.NoError(t, err2)
	require.Equal(t, "test-value", value2)
	require.True(t, ll.IsLoaded())
	require.Equal(t, int64(2), ll.GetLoadCount()) // Should increment
}

func TestLazyLoaderConcurrency(t *testing.T) {
	loadCount := 0
	mu := sync.Mutex{}

	loader := func() (string, error) {
		mu.Lock()
		defer mu.Unlock()
		loadCount++
		time.Sleep(10 * time.Millisecond) // Simulate slow load
		return "concurrent-value", nil
	}

	ll := NewLazyLoader(loader)

	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	// Concurrent access
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			value, err := ll.Get()
			results[index] = value
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all got the same value
	for i := 0; i < numGoroutines; i++ {
		require.NoError(t, errors[i])
		require.Equal(t, "concurrent-value", results[i])
	}

	// Verify loader was called only once
	mu.Lock()
	require.Equal(t, 1, loadCount)
	mu.Unlock()

	require.Equal(t, int64(1), ll.GetLoadCount())
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"AGreater", 10, 5, 10},
		{"BGreater", 5, 10, 10},
		{"Equal", 7, 7, 7},
		{"Negative", -5, -10, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maxInt(tt.a, tt.b)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestPressureMonitorCheckMemoryPressure tests the memory pressure checking functionality
func TestPressureMonitorCheckMemoryPressure(t *testing.T) {
	alerts := []Alert{}
	mu := sync.Mutex{}

	alertCallback := func(alert Alert) {
		mu.Lock()
		defer mu.Unlock()
		alerts = append(alerts, alert)
	}

	// Create monitor with very low thresholds to trigger alerts
	thresholds := Thresholds{
		HeapAllocMB:   1,     // 1MB - very low to trigger
		HeapSysMB:     1,     // 1MB - very low to trigger
		GCPercent:     100,   // High enough to not trigger
		NumGoroutines: 10000, // High enough to not trigger
	}

	monitor := NewPressureMonitor(thresholds, alertCallback)

	// Manually check memory pressure
	monitor.checkMemoryPressure()

	// Give time for alerts to be sent (async)
	time.Sleep(50 * time.Millisecond)

	// Check that alerts were triggered
	mu.Lock()
	defer mu.Unlock()

	require.GreaterOrEqual(t, len(alerts), 1) // At least one memory alert

	// Verify alert types
	alertTypes := make(map[string]bool)
	for _, alert := range alerts {
		alertTypes[alert.Type] = true
		require.NotEmpty(t, alert.Message)
		require.NotZero(t, alert.Timestamp)
		require.NotNil(t, alert.Stats)
	}

	// At least one of the memory alerts should be triggered
	require.True(t, alertTypes["heap_alloc"] || alertTypes["heap_sys"], "Expected at least one memory alert")
}

// TestPressureMonitorGoroutineAlert tests goroutine count alerting
func TestPressureMonitorGoroutineAlert(t *testing.T) {
	alerts := []Alert{}
	mu := sync.Mutex{}

	alertCallback := func(alert Alert) {
		mu.Lock()
		defer mu.Unlock()
		alerts = append(alerts, alert)
	}

	// Create monitor with very low goroutine threshold
	thresholds := Thresholds{
		HeapAllocMB:   10000, // Very high to not trigger
		HeapSysMB:     10000, // Very high to not trigger
		GCPercent:     100,   // High enough to not trigger
		NumGoroutines: 1,     // Very low to trigger
	}

	monitor := NewPressureMonitor(thresholds, alertCallback)

	// Check memory pressure - should trigger goroutine alert
	monitor.checkMemoryPressure()

	// Give time for alerts to be sent (async)
	time.Sleep(50 * time.Millisecond)

	// Check that goroutine alert was triggered
	mu.Lock()
	defer mu.Unlock()

	require.GreaterOrEqual(t, len(alerts), 1)

	// Find goroutine alert
	found := false
	for _, alert := range alerts {
		if alert.Type == "goroutines" {
			found = true
			require.Contains(t, alert.Message, "Goroutine count")
			require.Contains(t, alert.Message, "exceeds threshold")
		}
	}
	require.True(t, found, "Expected goroutine alert not found")
}

// TestPressureMonitorCalculateSeverity tests severity calculation
func TestPressureMonitorCalculateSeverity(t *testing.T) {
	monitor := NewPressureMonitor(DefaultThresholds(), nil)

	tests := []struct {
		name      string
		actual    float64
		threshold float64
		expected  AlertSeverity
	}{
		{
			name:      "Below threshold",
			actual:    90,
			threshold: 100,
			expected:  AlertInfo,
		},
		{
			name:      "Just above threshold",
			actual:    110,
			threshold: 100,
			expected:  AlertInfo,
		},
		{
			name:      "150% of threshold",
			actual:    150,
			threshold: 100,
			expected:  AlertWarning,
		},
		{
			name:      "200% of threshold",
			actual:    200,
			threshold: 100,
			expected:  AlertCritical,
		},
		{
			name:      "300% of threshold",
			actual:    300,
			threshold: 100,
			expected:  AlertCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.calculateSeverity("test", tt.actual, tt.threshold)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestPressureMonitorAlertCallback tests alert callback functionality
func TestPressureMonitorAlertCallback(t *testing.T) {
	var mu sync.Mutex
	alertReceived := false
	var receivedAlert Alert

	alertCallback := func(alert Alert) {
		mu.Lock()
		defer mu.Unlock()
		alertReceived = true
		receivedAlert = alert
	}

	monitor := NewPressureMonitor(DefaultThresholds(), alertCallback)

	// Send a test alert directly
	testAlert := Alert{
		Type:      "test",
		Message:   "Test alert",
		Timestamp: time.Now(),
		Severity:  AlertWarning,
	}

	monitor.sendAlert(testAlert)

	// Give time for async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.True(t, alertReceived)
	require.Equal(t, testAlert.Type, receivedAlert.Type)
	require.Equal(t, testAlert.Message, receivedAlert.Message)

	// Check stats
	stats := monitor.GetMonitorStats()
	require.Equal(t, int64(1), stats.AlertCount)
	require.Equal(t, int64(1), stats.HighPressureEvents)
}

// TestPressureMonitorMonitoringLoop tests the continuous monitoring loop
func TestPressureMonitorMonitoringLoop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running monitoring test")
	}

	alertCount := int64(0)
	mu := sync.Mutex{}

	alertCallback := func(_ Alert) {
		mu.Lock()
		defer mu.Unlock()
		alertCount++
	}

	// Create monitor with very short interval for testing
	monitor := NewPressureMonitor(DefaultThresholds(), alertCallback)
	monitor.monitoringInterval = 100 * time.Millisecond

	// Start monitoring
	monitor.StartMonitoring()

	// Let it run for a bit
	time.Sleep(350 * time.Millisecond)

	// Stop monitoring
	monitor.StopMonitoring()

	// Should have checked at least 3 times
	stats := monitor.GetMonitorStats()
	require.False(t, stats.MonitoringEnabled)
}

// TestPressureMonitorNilCallback tests monitor without alert callback
func TestPressureMonitorNilCallback(t *testing.T) {
	// Create monitor with nil callback
	monitor := NewPressureMonitor(DefaultThresholds(), nil)

	// Send alert - should not panic
	testAlert := Alert{
		Type:      "test",
		Message:   "Test alert",
		Timestamp: time.Now(),
		Severity:  AlertInfo,
	}

	monitor.sendAlert(testAlert)

	// Check stats
	stats := monitor.GetMonitorStats()
	require.Equal(t, int64(1), stats.AlertCount)
}

// TestPressureMonitorConcurrentStartStop tests concurrent start/stop operations
func TestPressureMonitorConcurrentStartStop(t *testing.T) {
	monitor := NewPressureMonitor(DefaultThresholds(), nil)

	// Test concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines * 2)

	// Half goroutines start monitoring
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			monitor.StartMonitoring()
		}()
	}

	// Half goroutines stop monitoring
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			monitor.StopMonitoring()
		}()
	}

	wg.Wait()

	// Should not panic and should be in a consistent state
	stats := monitor.GetMonitorStats()
	require.NotNil(t, stats)
}

// TestPressureMonitorStopChannelRaceCondition tests the stop channel race condition handling
func TestPressureMonitorStopChannelRaceCondition(t *testing.T) {
	monitor := NewPressureMonitor(DefaultThresholds(), nil)

	// Start monitoring
	monitor.StartMonitoring()

	// Close the stop channel manually to simulate race condition
	monitor.mu.Lock()
	close(monitor.stopChan)
	monitor.mu.Unlock()

	// Try to stop monitoring - should not panic
	monitor.StopMonitoring()

	// Verify state
	stats := monitor.GetMonitorStats()
	require.False(t, stats.MonitoringEnabled)
}

// BenchmarkStringIntern tests the performance of string interning
func BenchmarkStringIntern(b *testing.B) {
	si := NewStringIntern()
	testStrings := []string{
		"repository-name-1", "repository-name-2", "branch-main", "branch-develop",
		"file-path-1", "file-path-2", "commit-hash-1", "commit-hash-2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str := testStrings[i%len(testStrings)]
		si.Intern(str)
	}
}

// BenchmarkStringInternConcurrent tests concurrent string interning performance
func BenchmarkStringInternConcurrent(b *testing.B) {
	si := NewStringIntern()
	testString := "concurrent-test-string"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			si.Intern(testString)
		}
	})
}
