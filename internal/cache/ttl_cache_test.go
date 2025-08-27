package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTTLCacheBasicOperations tests basic cache operations
func TestTTLCacheBasicOperations(t *testing.T) {
	cache := NewTTLCache(time.Second, 10)
	defer cache.Close()

	// Test Set and Get
	t.Run("SetAndGet", func(t *testing.T) {
		cache.Set("key1", "value1")

		value, exists := cache.Get("key1")
		require.True(t, exists)
		require.Equal(t, "value1", value)
	})

	// Test Get non-existent key
	t.Run("GetNonExistentKey", func(t *testing.T) {
		value, exists := cache.Get("nonexistent")
		require.False(t, exists)
		require.Nil(t, value)
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		cache.Set("key2", "value2")

		_, exists := cache.Get("key2")
		require.True(t, exists)

		cache.Delete("key2")

		value, exists := cache.Get("key2")
		require.False(t, exists)
		require.Nil(t, value)
	})

	// Test Clear
	t.Run("Clear", func(t *testing.T) {
		cache.Set("key3", "value3")
		cache.Set("key4", "value4")

		cache.Clear()

		value, exists := cache.Get("key3")
		require.False(t, exists)
		require.Nil(t, value)

		value, exists = cache.Get("key4")
		require.False(t, exists)
		require.Nil(t, value)

		require.Equal(t, 0, cache.Size())
	})
}

// TestTTLCacheExpiration tests TTL expiration functionality
func TestTTLCacheExpiration(t *testing.T) {
	cache := NewTTLCache(100*time.Millisecond, 10)
	defer cache.Close()

	cache.Set("expiring", "value")

	// Should exist immediately
	value, exists := cache.Get("expiring")
	require.True(t, exists)
	require.Equal(t, "value", value)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	value, exists = cache.Get("expiring")
	require.False(t, exists)
	require.Nil(t, value)
}

// TestTTLCacheGetOrLoad tests the GetOrLoad functionality
func TestTTLCacheGetOrLoad(t *testing.T) {
	cache := NewTTLCache(time.Second, 10)
	defer cache.Close()

	loadCount := 0
	loader := func() (interface{}, error) {
		loadCount++
		return "loaded value", nil
	}

	// First call should load
	value, err := cache.GetOrLoad("key", loader)
	require.NoError(t, err)
	require.Equal(t, "loaded value", value)
	require.Equal(t, 1, loadCount)

	// Second call should use cached value
	value, err = cache.GetOrLoad("key", loader)
	require.NoError(t, err)
	require.Equal(t, "loaded value", value)
	require.Equal(t, 1, loadCount)
}

// TestTTLCacheGetOrLoadError tests GetOrLoad with loader error
func TestTTLCacheGetOrLoadError(t *testing.T) {
	cache := NewTTLCache(time.Second, 10)
	defer cache.Close()

	expectedErr := context.DeadlineExceeded
	loader := func() (interface{}, error) {
		return nil, expectedErr
	}

	value, err := cache.GetOrLoad("key", loader)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.Nil(t, value)

	// Verify nothing was cached
	value, exists := cache.Get("key")
	require.False(t, exists)
	require.Nil(t, value)
}

// TestTTLCacheMaxSize tests cache eviction when max size is reached
func TestTTLCacheMaxSize(t *testing.T) {
	cache := NewTTLCache(time.Hour, 3) // Small cache size
	defer cache.Close()

	// Fill cache to capacity
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	require.Equal(t, 3, cache.Size())

	// Adding another should evict the oldest
	cache.Set("key4", "value4")

	require.Equal(t, 3, cache.Size())

	// key4 should exist
	value, exists := cache.Get("key4")
	require.True(t, exists)
	require.Equal(t, "value4", value)
}

// TestTTLCacheStats tests cache statistics
func TestTTLCacheStats(t *testing.T) {
	cache := NewTTLCache(100*time.Millisecond, 10)
	defer cache.Close()

	// Initial stats
	hits, misses, size, hitRate := cache.Stats()
	require.Equal(t, int64(0), hits)
	require.Equal(t, int64(0), misses)
	require.Equal(t, 0, size)
	require.InDelta(t, 0.0, hitRate, 0.001)

	// Add some entries
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Get existing key (hit)
	_, _ = cache.Get("key1")

	// Get non-existent key (miss)
	_, _ = cache.Get("nonexistent")

	// Get expired key (miss)
	cache.Set("expired", "value")
	time.Sleep(200 * time.Millisecond)
	_, _ = cache.Get("expired")

	hits, misses, size, hitRate = cache.Stats()
	require.Equal(t, int64(1), hits)
	require.Equal(t, int64(2), misses)
	require.Equal(t, 0, size) // all keys have expired after 200ms
	require.InDelta(t, 0.333, hitRate, 0.001)
}

// TestTTLCacheConcurrency tests concurrent access
func TestTTLCacheConcurrency(t *testing.T) {
	cache := NewTTLCache(time.Second, 100)
	defer cache.Close()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := "key" + string(rune(id))
				value := "value" + string(rune(id))

				// Mix of operations
				switch j % 4 {
				case 0:
					cache.Set(key, value)
				case 1:
					_, _ = cache.Get(key)
				case 2:
					cache.Delete(key)
				case 3:
					_, _ = cache.GetOrLoad(key, func() (interface{}, error) {
						return value, nil
					})
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still functional
	cache.Set("final", "test")
	value, exists := cache.Get("final")
	require.True(t, exists)
	require.Equal(t, "test", value)
}

// TestTTLCacheCleanup tests the cleanup goroutine
func TestTTLCacheCleanup(t *testing.T) {
	cache := NewTTLCache(100*time.Millisecond, 10)
	defer cache.Close()

	// Add entries that will expire
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	require.Equal(t, 2, cache.Size())

	// Wait for entries to expire and cleanup to run
	// Cleanup runs at ttl/2 intervals
	time.Sleep(200 * time.Millisecond)

	// Force a read to check if entries are still there
	_, exists1 := cache.Get("key1")
	_, exists2 := cache.Get("key2")

	require.False(t, exists1)
	require.False(t, exists2)
}

// TestTTLCacheClose tests proper shutdown
func TestTTLCacheClose(t *testing.T) {
	cache := NewTTLCache(time.Second, 10)

	cache.Set("key", "value")

	// Close the cache
	cache.Close()

	// Operations should still work (no panic)
	value, exists := cache.Get("key")
	require.True(t, exists)
	require.Equal(t, "value", value)

	// Closing again should not panic
	cache.Close()
}

// TestTTLCacheMetrics tests cache metrics functionality
func TestTTLCacheMetrics(t *testing.T) {
	cache := NewTTLCache(time.Second, 10)
	defer cache.Close()

	// Perform some operations
	cache.Set("key1", "value1")
	_, _ = cache.Get("key1")        // Hit
	_, _ = cache.Get("nonexistent") // Miss

	// Check stats
	hits, misses, size, hitRate := cache.Stats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(1), misses)
	assert.GreaterOrEqual(t, size, 1)
	assert.InDelta(t, 0.5, hitRate, 0.001) // 50% hit rate
}
