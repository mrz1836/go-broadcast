package sync

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ModuleCacheEntry represents a cached module version resolution
type ModuleCacheEntry struct {
	Value     string
	ExpiresAt time.Time
}

// ModuleCache provides thread-safe caching for module version resolutions
type ModuleCache struct {
	entries map[string]*ModuleCacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
	logger  *logrus.Logger
	done    chan struct{}
	once    sync.Once
}

// NewModuleCache creates a new module cache with the specified TTL
func NewModuleCache(ttl time.Duration, logger *logrus.Logger) *ModuleCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute // Default TTL
	}

	cache := &ModuleCache{
		entries: make(map[string]*ModuleCacheEntry),
		ttl:     ttl,
		logger:  logger,
		done:    make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Get retrieves a value from the cache
func (c *ModuleCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return "", false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}

	c.logger.WithFields(logrus.Fields{
		"key":   key,
		"value": entry.Value,
	}).Debug("Cache hit")

	return entry.Value, true
}

// Set stores a value in the cache
func (c *ModuleCache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &ModuleCacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}

	c.logger.WithFields(logrus.Fields{
		"key":   key,
		"value": value,
		"ttl":   c.ttl,
	}).Debug("Cache set")
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *ModuleCache) SetWithTTL(key, value string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &ModuleCacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	c.logger.WithFields(logrus.Fields{
		"key":   key,
		"value": value,
		"ttl":   ttl,
	}).Debug("Cache set with custom TTL")
}

// Delete removes a value from the cache
func (c *ModuleCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)

	c.logger.WithField("key", key).Debug("Cache delete")
}

// Clear removes all entries from the cache
func (c *ModuleCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldSize := len(c.entries)
	c.entries = make(map[string]*ModuleCacheEntry)

	c.logger.WithField("entries_cleared", oldSize).Info("Cache cleared")
}

// Size returns the number of entries in the cache
func (c *ModuleCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// cleanupExpired periodically removes expired entries
func (c *ModuleCache) cleanupExpired() {
	ticker := time.NewTicker(c.ttl / 2) // Cleanup every half TTL
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			expired := 0

			for key, entry := range c.entries {
				if now.After(entry.ExpiresAt) {
					delete(c.entries, key)
					expired++
				}
			}

			c.mu.Unlock()

			if expired > 0 {
				c.logger.WithField("expired_entries", expired).Debug("Cleaned up expired cache entries")
			}
		}
	}
}

// Close shuts down the cache and stops the cleanup goroutine
func (c *ModuleCache) Close() {
	c.once.Do(func() {
		close(c.done)
	})
}

// Stats returns cache statistics
func (c *ModuleCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalExpired int
	now := time.Now()

	for _, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			totalExpired++
		}
	}

	return map[string]interface{}{
		"total_entries": len(c.entries),
		"expired":       totalExpired,
		"active":        len(c.entries) - totalExpired,
		"ttl_seconds":   c.ttl.Seconds(),
	}
}

// GetOrCompute retrieves a value from cache or computes it if not present
func (c *ModuleCache) GetOrCompute(key string, compute func() (string, error)) (string, error) {
	// Try to get from cache first
	if value, found := c.Get(key); found {
		return value, nil
	}

	// Compute the value
	value, err := compute()
	if err != nil {
		return "", err
	}

	// Store in cache
	c.Set(key, value)

	return value, nil
}

// Invalidate removes all cache entries matching a pattern
func (c *ModuleCache) Invalidate(pattern string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	invalidated := 0
	for key := range c.entries {
		// Simple prefix matching for now
		if len(pattern) > 0 && len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			delete(c.entries, key)
			invalidated++
		}
	}

	if invalidated > 0 {
		c.logger.WithFields(logrus.Fields{
			"pattern":     pattern,
			"invalidated": invalidated,
		}).Debug("Cache entries invalidated")
	}

	return invalidated
}
