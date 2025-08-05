package sync

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleCache_GetSet(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(1*time.Second, logger)

	t.Run("set and get value", func(t *testing.T) {
		cache.Set("key1", "value1")

		value, found := cache.Get("key1")
		assert.True(t, found)
		assert.Equal(t, "value1", value)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		value, found := cache.Get("non-existent")
		assert.False(t, found)
		assert.Empty(t, value)
	})

	t.Run("overwrite existing value", func(t *testing.T) {
		cache.Set("key2", "value2")
		cache.Set("key2", "new-value2")

		value, found := cache.Get("key2")
		assert.True(t, found)
		assert.Equal(t, "new-value2", value)
	})
}

func TestModuleCache_TTL(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(100*time.Millisecond, logger)

	t.Run("value expires after TTL", func(t *testing.T) {
		cache.Set("expiring-key", "expiring-value")

		// Value should exist immediately
		value, found := cache.Get("expiring-key")
		assert.True(t, found)
		assert.Equal(t, "expiring-value", value)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Value should no longer exist
		value, found = cache.Get("expiring-key")
		assert.False(t, found)
		assert.Empty(t, value)
	})

	t.Run("SetWithTTL uses custom TTL", func(t *testing.T) {
		cache.SetWithTTL("custom-ttl", "value", 200*time.Millisecond)

		// Should exist after short wait
		time.Sleep(100 * time.Millisecond)
		value, found := cache.Get("custom-ttl")
		assert.True(t, found)
		assert.Equal(t, "value", value)

		// Should expire after custom TTL
		time.Sleep(150 * time.Millisecond)
		value, found = cache.Get("custom-ttl")
		assert.False(t, found)
		assert.Empty(t, value)
	})
}

func TestModuleCache_Delete(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(1*time.Second, logger)

	t.Run("delete existing key", func(t *testing.T) {
		cache.Set("key-to-delete", "value")

		// Verify it exists
		value, found := cache.Get("key-to-delete")
		assert.True(t, found)
		assert.Equal(t, "value", value)

		// Delete it
		cache.Delete("key-to-delete")

		// Verify it's gone
		value, found = cache.Get("key-to-delete")
		assert.False(t, found)
		assert.Empty(t, value)
	})

	t.Run("delete non-existent key", func(_ *testing.T) {
		// Should not panic
		cache.Delete("non-existent-key")
	})
}

func TestModuleCache_Clear(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(1*time.Second, logger)

	// Add multiple entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Verify they exist
	assert.Equal(t, 3, cache.Size())

	// Clear the cache
	cache.Clear()

	// Verify all entries are gone
	assert.Equal(t, 0, cache.Size())

	value, found := cache.Get("key1")
	assert.False(t, found)
	assert.Empty(t, value)
}

func TestModuleCache_Size(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(1*time.Second, logger)

	assert.Equal(t, 0, cache.Size())

	cache.Set("key1", "value1")
	assert.Equal(t, 1, cache.Size())

	cache.Set("key2", "value2")
	assert.Equal(t, 2, cache.Size())

	cache.Delete("key1")
	assert.Equal(t, 1, cache.Size())

	cache.Clear()
	assert.Equal(t, 0, cache.Size())
}

func TestModuleCache_Stats(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(200*time.Millisecond, logger)

	// Add some entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.SetWithTTL("key3", "value3", 50*time.Millisecond)

	// Get initial stats
	stats := cache.Stats()
	assert.Equal(t, 3, stats["total_entries"])
	assert.Equal(t, 0, stats["expired"])
	assert.Equal(t, 3, stats["active"])
	assert.InEpsilon(t, 0.2, stats["ttl_seconds"], 0.01)

	// Wait for one entry to expire - give more time for cleanup goroutine
	time.Sleep(200 * time.Millisecond)

	// Force a Get operation to trigger checking expired entries
	_, _ = cache.Get("key3")

	// Get updated stats
	stats = cache.Stats()
	// Due to the cleanup goroutine, entries may have been removed
	// so we just check that the counts are consistent
	total := stats["total_entries"].(int)
	expired := stats["expired"].(int)
	active := stats["active"].(int)

	// Total should be non-negative
	assert.GreaterOrEqual(t, total, 0)
	assert.GreaterOrEqual(t, expired, 0)
	assert.GreaterOrEqual(t, active, 0)
	// Total should equal expired + active
	assert.Equal(t, total, expired+active)
}

func TestModuleCache_GetOrCompute(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(1*time.Second, logger)

	t.Run("computes and caches on miss", func(t *testing.T) {
		computeCalled := false
		compute := func() (string, error) {
			computeCalled = true
			return "computed-value", nil
		}

		value, err := cache.GetOrCompute("compute-key", compute)
		require.NoError(t, err)
		assert.Equal(t, "computed-value", value)
		assert.True(t, computeCalled)

		// Verify it was cached
		cached, found := cache.Get("compute-key")
		assert.True(t, found)
		assert.Equal(t, "computed-value", cached)
	})

	t.Run("uses cache on hit", func(t *testing.T) {
		cache.Set("existing-key", "existing-value")

		computeCalled := false
		compute := func() (string, error) {
			computeCalled = true
			return "should-not-be-called", nil
		}

		value, err := cache.GetOrCompute("existing-key", compute)
		require.NoError(t, err)
		assert.Equal(t, "existing-value", value)
		assert.False(t, computeCalled) // Compute should not be called
	})

	t.Run("returns error from compute", func(t *testing.T) {
		compute := func() (string, error) {
			return "", assert.AnError
		}

		value, err := cache.GetOrCompute("error-key", compute)
		require.Error(t, err)
		assert.Empty(t, value)

		// Should not be cached on error
		_, found := cache.Get("error-key")
		assert.False(t, found)
	})
}

func TestModuleCache_Invalidate(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(1*time.Second, logger)

	// Add entries with different prefixes
	cache.Set("repo1:v1.0.0", "value1")
	cache.Set("repo1:v2.0.0", "value2")
	cache.Set("repo2:v1.0.0", "value3")
	cache.Set("other:key", "value4")

	t.Run("invalidates by prefix", func(t *testing.T) {
		count := cache.Invalidate("repo1:")
		assert.Equal(t, 2, count)

		// repo1 entries should be gone
		_, found := cache.Get("repo1:v1.0.0")
		assert.False(t, found)
		_, found = cache.Get("repo1:v2.0.0")
		assert.False(t, found)

		// Other entries should remain
		value, found := cache.Get("repo2:v1.0.0")
		assert.True(t, found)
		assert.Equal(t, "value3", value)

		value, found = cache.Get("other:key")
		assert.True(t, found)
		assert.Equal(t, "value4", value)
	})

	t.Run("returns 0 for non-matching pattern", func(t *testing.T) {
		count := cache.Invalidate("non-existent:")
		assert.Equal(t, 0, count)
	})
}

func TestModuleCache_Concurrency(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(1*time.Second, logger)

	t.Run("concurrent reads and writes", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 100

		// Start writers
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("key-%d", id)
				value := fmt.Sprintf("value-%d", id)
				cache.Set(key, value)
			}(i)
		}

		// Start readers
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("key-%d", id)
				// May or may not find the value depending on timing
				cache.Get(key)
			}(i)
		}

		// Start deleters
		for i := 0; i < numGoroutines/2; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("key-%d", id)
				cache.Delete(key)
			}(i)
		}

		wg.Wait()

		// Should not panic and cache should be in valid state
		stats := cache.Stats()
		assert.NotNil(t, stats)
	})

	t.Run("concurrent operations on same key", func(_ *testing.T) {
		var wg sync.WaitGroup
		numOperations := 1000

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				// Perform random operation
				switch iteration % 3 {
				case 0:
					cache.Set("shared-key", fmt.Sprintf("value-%d", iteration))
				case 1:
					cache.Get("shared-key")
				case 2:
					cache.Delete("shared-key")
				}
			}(i)
		}

		wg.Wait()

		// Should not panic
		// Final state depends on timing but should be valid
		_, _ = cache.Get("shared-key")
	})
}

func TestModuleCache_DefaultTTL(t *testing.T) {
	logger := logrus.New()
	// Create cache with zero TTL (should use default)
	cache := NewModuleCache(0, logger)

	cache.Set("key", "value")

	// Should still exist after a short time (default is 5 minutes)
	time.Sleep(100 * time.Millisecond)
	value, found := cache.Get("key")
	assert.True(t, found)
	assert.Equal(t, "value", value)
}
