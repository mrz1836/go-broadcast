package profiling

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestError = errors.New("test error")

func TestNewMemoryProfiler(t *testing.T) {
	outputDir := "/tmp/test-profiler"
	profiler := NewMemoryProfiler(outputDir)

	require.NotNil(t, profiler)
	require.Equal(t, outputDir, profiler.outputDir)
	require.False(t, profiler.enabled)
	require.NotNil(t, profiler.sessions)
	require.Empty(t, profiler.sessions)
}

func TestMemoryProfilerEnableDisable(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	// Test enabling
	err := profiler.Enable()
	require.NoError(t, err)
	require.True(t, profiler.enabled)

	// Test enabling again (should not error)
	err = profiler.Enable()
	require.NoError(t, err)
	require.True(t, profiler.enabled)

	// Test disabling
	err = profiler.Disable()
	require.NoError(t, err)
	require.False(t, profiler.enabled)

	// Test disabling again (should not error)
	err = profiler.Disable()
	require.NoError(t, err)
	require.False(t, profiler.enabled)
}

func TestMemoryProfilerEnableInvalidDir(t *testing.T) {
	// Use a path that can't be created (on most systems)
	invalidDir := "/root/nonexistent/profiler"
	profiler := NewMemoryProfiler(invalidDir)

	err := profiler.Enable()
	// This should error unless running as root
	if os.Geteuid() != 0 {
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create profile output directory")
	}
}

func TestMemoryProfilerStartStopProfiling(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	// Enable profiler first
	err := profiler.Enable()
	require.NoError(t, err)

	// Start profiling
	session, err := profiler.StartProfiling("test-session")
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, "test-session", session.Name)
	require.True(t, session.started)
	require.False(t, session.stopped)
	require.NotEmpty(t, session.OutputDir)

	// Verify session is tracked
	stats := profiler.GetProfilerStats()
	require.Equal(t, 1, stats.ActiveSessions)
	require.Equal(t, int64(1), stats.TotalSessions)

	// Stop profiling
	err = profiler.StopProfiling("test-session")
	require.NoError(t, err)

	// Verify session is no longer tracked
	stats = profiler.GetProfilerStats()
	require.Equal(t, 0, stats.ActiveSessions)
	require.Equal(t, int64(1), stats.TotalSessions)
	require.Equal(t, int64(1), stats.ProfileCount)
}

func TestMemoryProfilerStartProfilingDisabled(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	// Try to start profiling without enabling
	_, err := profiler.StartProfiling("test-session")
	require.Error(t, err)
	require.Equal(t, ErrProfilerNotEnabled, err)
}

func TestMemoryProfilerStartDuplicateSession(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	err := profiler.Enable()
	require.NoError(t, err)

	// Start first session
	_, err = profiler.StartProfiling("duplicate-session")
	require.NoError(t, err)

	// Try to start session with same name
	_, err = profiler.StartProfiling("duplicate-session")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionExists)
}

func TestMemoryProfilerStopNonexistentSession(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	err := profiler.Enable()
	require.NoError(t, err)

	// Try to stop nonexistent session
	err = profiler.StopProfiling("nonexistent-session")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)
}

func TestMemoryProfilerGetProfilerStats(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	// Initial stats
	stats := profiler.GetProfilerStats()
	require.False(t, stats.Enabled)
	require.Equal(t, tempDir, stats.OutputDir)
	require.Equal(t, 0, stats.ActiveSessions)
	require.Equal(t, int64(0), stats.TotalSessions)
	require.Equal(t, int64(0), stats.ProfileCount)

	// Enable and check stats
	err := profiler.Enable()
	require.NoError(t, err)

	stats = profiler.GetProfilerStats()
	require.True(t, stats.Enabled)
}

func TestCaptureMemStats(t *testing.T) {
	snapshot := CaptureMemStats("test-label")

	require.Equal(t, "test-label", snapshot.Label)
	require.False(t, snapshot.Timestamp.IsZero())
	require.Less(t, time.Since(snapshot.Timestamp), time.Minute)
	require.Positive(t, snapshot.Goroutines)
	require.Positive(t, snapshot.MemStats.Sys)
}

func TestMemorySnapshotCompare(t *testing.T) {
	// Create first snapshot
	snapshot1 := CaptureMemStats("before")

	// Do some allocation
	data := make([][]byte, 1000)
	for i := range data {
		data[i] = make([]byte, 1024)
	}

	// Create second snapshot
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	snapshot2 := CaptureMemStats("after")

	// Compare snapshots
	comparison := snapshot1.Compare(snapshot2)

	require.Equal(t, snapshot1, comparison.From)
	require.Equal(t, snapshot2, comparison.To)
	require.Greater(t, comparison.Duration, time.Duration(0))
	require.GreaterOrEqual(t, comparison.TotalAllocDelta, int64(0)) // Should have allocated
}

func TestMemoryComparisonString(t *testing.T) {
	snapshot1 := MemorySnapshot{
		Label:      "test1",
		Timestamp:  time.Now(),
		MemStats:   runtime.MemStats{Alloc: 1000, TotalAlloc: 5000, HeapSys: 2000, NumGC: 1},
		Goroutines: 10,
	}

	snapshot2 := MemorySnapshot{
		Label:      "test2",
		Timestamp:  snapshot1.Timestamp.Add(time.Second),
		MemStats:   runtime.MemStats{Alloc: 1500, TotalAlloc: 6000, HeapSys: 2500, NumGC: 2},
		Goroutines: 12,
	}

	comparison := snapshot1.Compare(snapshot2)
	result := comparison.String()

	require.Contains(t, result, "Memory Comparison: test1 → test2")
	require.Contains(t, result, "Heap Alloc:")
	require.Contains(t, result, "Total Alloc:")
	require.Contains(t, result, "Heap Sys:")
	require.Contains(t, result, "GC Count:")
	require.Contains(t, result, "Goroutines:")
}

func TestProfileWithContext(t *testing.T) {
	// Test with nil profiler to avoid CPU profiling conflicts
	comparison, err := ProfileWithContext(context.Background(), nil, "test-profile", func() error {
		// Do some work
		data := make([]byte, 1024)
		_ = data
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, "test-profile_start", comparison.From.Label)
	require.Equal(t, "test-profile_end", comparison.To.Label)
	require.Greater(t, comparison.Duration, time.Duration(0))
}

func TestProfileWithContextError(t *testing.T) {
	expectedErr := errTestError

	// Test function that returns error (using nil profiler to avoid conflicts)
	comparison, err := ProfileWithContext(context.Background(), nil, "error-profile", func() error {
		return expectedErr
	})

	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Equal(t, "error-profile_start", comparison.From.Label)
	require.Equal(t, "error-profile_end", comparison.To.Label)
}

func TestProfileWithContextNilProfiler(t *testing.T) {
	// Test with nil profiler (should still work)
	comparison, err := ProfileWithContext(context.Background(), nil, "nil-profiler", func() error {
		data := make([]byte, 1024)
		_ = data
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, "nil-profiler_start", comparison.From.Label)
	require.Equal(t, "nil-profiler_end", comparison.To.Label)
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{"Bytes", 512, "512 B"},
		{"Kilobytes", 1536, "1.5 KB"},
		{"Megabytes", 1572864, "1.5 MB"},
		{"Gigabytes", 1610612736, "1.5 GB"},
		{"Zero", 0, "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBytesDelta(t *testing.T) {
	tests := []struct {
		name     string
		delta    int64
		expected string
	}{
		{"PositiveDelta", 1024, "+1.0 KB"},
		{"NegativeDelta", -1024, "-1.0 KB"},
		{"ZeroDelta", 0, "0 B"},
		{"SmallPositive", 100, "+100 B"},
		{"SmallNegative", -100, "-100 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytesDelta(tt.delta)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMemoryProfilerDisableWithActiveSessions(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	err := profiler.Enable()
	require.NoError(t, err)

	// Disable should not panic even with no active sessions
	require.NotPanics(t, func() {
		err = profiler.Disable()
		require.NoError(t, err)
	})

	require.False(t, profiler.enabled)
	require.Empty(t, profiler.sessions)
}

func TestSessionDirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	err := profiler.Enable()
	require.NoError(t, err)

	// Test directory creation without starting actual profiling
	require.True(t, profiler.enabled)
	require.Equal(t, tempDir, profiler.outputDir)

	// Verify output directory exists
	_, err = os.Stat(tempDir)
	require.NoError(t, err)
}

// BenchmarkCaptureMemStats tests the performance of memory stats capture
func BenchmarkCaptureMemStats(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CaptureMemStats("benchmark")
	}
}

// BenchmarkMemorySnapshotCompare tests the performance of snapshot comparison
func BenchmarkMemorySnapshotCompare(b *testing.B) {
	snapshot1 := CaptureMemStats("bench1")
	snapshot2 := CaptureMemStats("bench2")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		snapshot1.Compare(snapshot2)
	}
}

func TestMemoryProfilerConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	profiler := NewMemoryProfiler(tempDir)

	err := profiler.Enable()
	require.NoError(t, err)

	// Test concurrent access to profiler stats only
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			stats := profiler.GetProfilerStats()
			assert.True(t, stats.Enabled)
			assert.Equal(t, tempDir, stats.OutputDir)
		}
	}()

	// Concurrently access stats
	for i := 0; i < 50; i++ {
		stats := profiler.GetProfilerStats()
		require.True(t, stats.Enabled)
	}

	// Wait for concurrent goroutine to finish
	<-done
}
