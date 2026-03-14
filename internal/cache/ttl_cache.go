package cache

import (
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// Entry represents a cached value
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// TTLCache provides time-based caching with automatic expiration and cleanup.
//
// The cache uses a background goroutine for periodic cleanup of expired entries.
// IMPORTANT: Always call Close() when done to stop the cleanup goroutine and
// prevent resource leaks.
type TTLCache struct {
	mu      sync.RWMutex
	items   map[string]Entry
	ttl     time.Duration
	maxSize int

	// Metrics
	hits   atomic.Int64
	misses atomic.Int64

	// Cleanup
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	once            sync.Once

	// Singleflight for GetOrLoad to prevent thundering herd
	group singleflight.Group
}

// DefaultTTL is the default cache TTL when an invalid value is provided.
const DefaultTTL = time.Minute

// DefaultMaxSize is the default maximum cache size when an invalid value is provided.
const DefaultMaxSize = 1000

// MinCleanupInterval is the minimum interval between cleanup runs.
const MinCleanupInterval = time.Millisecond

// NewTTLCache creates a new TTL cache.
//
// Parameters:
//   - ttl: Time-to-live for cache entries. If <= 0, defaults to 1 minute.
//   - maxSize: Maximum number of entries. If <= 0, defaults to 1000.
//
// IMPORTANT: Call Close() when done to stop the background cleanup goroutine.
func NewTTLCache(ttl time.Duration, maxSize int) *TTLCache {
	// Validate and apply defaults for TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	// Validate and apply defaults for maxSize
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}

	// Calculate cleanup interval (ttl/2, but at least MinCleanupInterval)
	cleanupInterval := ttl / 2
	if cleanupInterval < MinCleanupInterval {
		cleanupInterval = MinCleanupInterval
	}

	cache := &TTLCache{
		items:           make(map[string]Entry),
		ttl:             ttl,
		maxSize:         maxSize,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from cache
func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		c.misses.Add(1)
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		c.misses.Add(1)
		return nil, false
	}

	c.hits.Add(1)
	return entry.Value, true
}

// Set stores a value in cache
func (c *TTLCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entry if at capacity
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// GetOrLoad retrieves from cache or loads using the provided function.
//
// This method uses singleflight to prevent the "thundering herd" problem:
// when multiple goroutines request the same missing key simultaneously,
// only one will call the loader function, and the result is shared.
func (c *TTLCache) GetOrLoad(key string, loader func() (interface{}, error)) (interface{}, error) {
	// Fast path: check if value exists in cache
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	// Use singleflight to ensure only one loader runs per key
	val, err, _ := c.group.Do(key, func() (interface{}, error) {
		// Double-check after acquiring singleflight lock
		// (another goroutine may have populated the cache)
		if val, ok := c.Get(key); ok {
			return val, nil
		}

		// Call the loader
		val, err := loader()
		if err != nil {
			return nil, err
		}

		// Cache the result
		c.Set(key, val)
		return val, nil
	})

	return val, err
}

// Delete removes a key from the cache
func (c *TTLCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all entries from the cache
func (c *TTLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]Entry)
	c.hits.Store(0)
	c.misses.Store(0)
}

// Stats returns cache statistics.
//
// The returned values represent a consistent snapshot - all values
// are captured under the same lock to ensure consistency.
func (c *TTLCache) Stats() (hits, misses int64, size int, hitRate float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	size = len(c.items)
	hits = c.hits.Load()
	misses = c.misses.Load()

	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return hits, misses, size, hitRate
}

// Size returns current cache size
func (c *TTLCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Close stops the cleanup goroutine
func (c *TTLCache) Close() {
	c.once.Do(func() {
		close(c.stopCleanup)
	})
}

// cleanup periodically removes expired entries
func (c *TTLCache) cleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, entry := range c.items {
				if now.After(entry.ExpiresAt) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		case <-c.stopCleanup:
			return
		}
	}
}

// evictOldest removes an entry to make room for a new one.
// It first tries to remove an expired entry, falling back to the oldest valid entry.
// This is O(n) - for high-performance use cases with large caches,
// consider using a heap-based or LRU eviction strategy.
func (c *TTLCache) evictOldest() {
	now := time.Now()

	// First pass: try to find and remove any expired entry
	for key, entry := range c.items {
		if now.After(entry.ExpiresAt) {
			delete(c.items, key)
			return
		}
	}

	// Second pass: no expired entries, remove the oldest valid entry
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.items {
		if oldestTime.IsZero() || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}
