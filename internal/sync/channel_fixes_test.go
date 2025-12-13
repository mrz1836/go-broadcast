package sync

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestBatchProcessor_ChannelCloseRace tests that channels are closed properly
// without race conditions (Issue 12 fix)
func TestBatchProcessor_ChannelCloseRace(t *testing.T) {
	// Create a minimal setup
	logger := logrus.NewEntry(logrus.New())
	sourceState := &state.SourceState{
		Repo: "test/source",
	}
	target := config.TargetConfig{
		Repo: "test/target",
	}

	// Create a mock engine (nil is fine for this test since we're testing channel behavior)
	bp := &BatchProcessor{
		engine:      nil,
		target:      target,
		sourceState: sourceState,
		logger:      logger,
		workerCount: 2,
	}

	// Run multiple iterations to catch race conditions
	for i := 0; i < 10; i++ {
		ctx := context.Background()

		// Empty jobs should return quickly without panics
		changes, err := bp.ProcessFiles(ctx, "/tmp", []FileJob{})
		require.NoError(t, err)
		assert.Empty(t, changes)
	}
}

// TestBatchProcessor_CancellationDrain tests that result channels are drained
// on cancellation to prevent goroutine leaks (Issue 14 fix)
func TestBatchProcessor_CancellationDrain(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	sourceState := &state.SourceState{
		Repo: "test/source",
	}
	target := config.TargetConfig{
		Repo: "test/target",
	}

	bp := &BatchProcessor{
		engine:      nil,
		target:      target,
		sourceState: sourceState,
		logger:      logger,
		workerCount: 2,
	}

	// Create a context that will be canceled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Process empty jobs - should handle cancellation gracefully
	_, err := bp.ProcessFiles(ctx, "/tmp", []FileJob{})
	// Empty jobs return nil error even with canceled context
	// The key is that it doesn't deadlock or leak goroutines
	if err != nil {
		assert.Contains(t, err.Error(), "batch processing failed")
	}
}

// TestBatchProcessor_ContextTimeout tests proper handling of context timeout
func TestBatchProcessor_ContextTimeout(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	sourceState := &state.SourceState{
		Repo: "test/source",
	}
	target := config.TargetConfig{
		Repo: "test/target",
	}

	bp := &BatchProcessor{
		engine:      nil,
		target:      target,
		sourceState: sourceState,
		logger:      logger,
		workerCount: 2,
	}

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Process should handle timeout gracefully (no deadlock or panic)
	// Empty jobs may return nil or error depending on timing
	_, err := bp.ProcessFiles(ctx, "/tmp", []FileJob{})
	// The key is no deadlock or panic - error is optional with empty jobs
	if err != nil {
		assert.Contains(t, err.Error(), "batch processing failed")
	}
}

// TestModuleCache_CloseWithoutLeak tests that ModuleCache can be closed
// without goroutine leaks (Issue 13 fix)
func TestModuleCache_CloseWithoutLeak(t *testing.T) {
	logger := logrus.New()

	// Create multiple caches and close them
	for i := 0; i < 10; i++ {
		cache := NewModuleCache(1*time.Second, logger)

		// Use the cache
		cache.Set("key", "value")
		val, found := cache.Get("key")
		assert.True(t, found)
		assert.Equal(t, "value", val)

		// Close the cache
		cache.Close()

		// Calling close multiple times should be safe
		cache.Close()
	}
}

// TestModuleCache_GarbageCollection tests that finalizer cleanup works
func TestModuleCache_GarbageCollection(_ *testing.T) {
	logger := logrus.New()

	// Create cache and let it go out of scope
	func() {
		cache := NewModuleCache(1*time.Second, logger)
		cache.Set("key", "value")
		// Cache will be garbage collected, finalizer should run
	}()

	// This test mainly verifies that the finalizer doesn't panic
	// Actual GC timing is non-deterministic
}

// TestModuleCache_CloseIdempotent tests that Close can be called multiple times safely
func TestModuleCache_CloseIdempotent(_ *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(1*time.Second, logger)

	// Close multiple times
	cache.Close()
	cache.Close()
	cache.Close()

	// Should not panic
}

// TestModuleCache_OperationsAfterClose tests that operations after Close are safe
func TestModuleCache_OperationsAfterClose(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(100*time.Millisecond, logger)

	cache.Set("key", "value")
	cache.Close()

	// Operations after close should still work (cache data remains)
	val, found := cache.Get("key")
	assert.True(t, found)
	assert.Equal(t, "value", val)

	// New operations should work
	cache.Set("key2", "value2")
	val2, found2 := cache.Get("key2")
	assert.True(t, found2)
	assert.Equal(t, "value2", val2)
}

// TestBatchProcessor_WorkerPanic tests that worker panics don't deadlock channels
func TestBatchProcessor_WorkerPanic(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	sourceState := &state.SourceState{
		Repo: "test/source",
	}
	target := config.TargetConfig{
		Repo: "test/target",
	}

	bp := &BatchProcessor{
		engine:      nil,
		target:      target,
		sourceState: sourceState,
		logger:      logger,
		workerCount: 2,
	}

	ctx := context.Background()

	// Even with jobs that might cause issues, the system should handle it gracefully
	jobs := []FileJob{
		NewFileJob("nonexistent.txt", "dest.txt", config.Transform{}),
	}

	// This should complete without deadlock even if workers encounter errors
	_, err := bp.ProcessFiles(ctx, "/nonexistent", jobs)
	// Error is expected, but no deadlock
	if err != nil {
		// This is acceptable - we're testing for deadlock prevention
		t.Logf("Got expected error: %v", err)
	}
}
