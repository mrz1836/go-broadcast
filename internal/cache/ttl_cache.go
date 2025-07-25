package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

// Entry represents a cached value
type Entry struct {
	Value     interface{}
	ExpiresAt time.Time
}

// TTLCache provides time-based caching
type TTLCache struct {
	mu      sync.RWMutex
	items   map[string]Entry
	ttl     time.Duration
	maxSize int

	// Metrics
	hits   atomic.Int64
	misses atomic.Int64

	// Cleanup
	stopCleanup chan struct{}
	once        sync.Once
}

// NewTTLCache creates a new TTL cache
func NewTTLCache(ttl time.Duration, maxSize int) *TTLCache {
	cache := &TTLCache{
		items:       make(map[string]Entry),
		ttl:         ttl,
		maxSize:     maxSize,
		stopCleanup: make(chan struct{}),
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

// GetOrLoad retrieves from cache or loads using the provided function
func (c *TTLCache) GetOrLoad(key string, loader func() (interface{}, error)) (interface{}, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	val, err := loader()
	if err != nil {
		return nil, err
	}

	c.Set(key, val)
	return val, nil
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

// Stats returns cache statistics
func (c *TTLCache) Stats() (hits, misses int64, size int, hitRate float64) {
	c.mu.RLock()
	size = len(c.items)
	c.mu.RUnlock()

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
	ticker := time.NewTicker(c.ttl / 2)
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

// evictOldest removes the oldest entry
func (c *TTLCache) evictOldest() {
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
