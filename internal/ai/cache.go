package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"
)

// ResponseCache caches AI responses keyed by SHA256 hash of diff content.
// This dramatically reduces API calls when syncing identical files to multiple repos.
// Thread-safe for concurrent use.
type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
	enabled bool
	hits    int64
	misses  int64
}

// CacheEntry holds a cached AI response.
type CacheEntry struct {
	Response  string
	CreatedAt time.Time
}

// NewResponseCache creates a new cache with the given configuration.
func NewResponseCache(cfg *Config) *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]*CacheEntry),
		maxSize: cfg.CacheMaxSize,
		ttl:     cfg.CacheTTL,
		enabled: cfg.CacheEnabled,
	}
}

// hashDiff creates a SHA256 hash of the diff content.
func hashDiff(diff string) string {
	h := sha256.New()
	h.Write([]byte(diff))
	return hex.EncodeToString(h.Sum(nil))
}

// Get retrieves a cached response if it exists and hasn't expired.
func (c *ResponseCache) Get(diffContent string) (string, bool) {
	if !c.enabled {
		return "", false
	}
	hash := hashDiff(diffContent)
	return c.getByHash(hash)
}

// getByHash retrieves a cached response by pre-computed hash.
// Avoids redundant hashing when called from GetOrGenerate.
func (c *ResponseCache) getByHash(hash string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[hash]
	if !ok {
		return "", false
	}

	// Check TTL
	if time.Since(entry.CreatedAt) > c.ttl {
		return "", false // Expired
	}

	return entry.Response, true
}

// Set stores a response in the cache.
func (c *ResponseCache) Set(diffContent, response string) {
	if !c.enabled {
		return
	}
	hash := hashDiff(diffContent)
	c.setByHash(hash, response)
}

// setByHash stores a response by pre-computed hash.
// Avoids redundant hashing when called from GetOrGenerate.
func (c *ResponseCache) setByHash(hash, response string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entries if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[hash] = &CacheEntry{
		Response:  response,
		CreatedAt: time.Now(),
	}
}

// GetOrGenerate checks cache first, calls generator only on cache miss.
// The keyPrefix differentiates different types of generation (e.g., "commit:", "pr:").
// Returns the response, whether it was a cache hit, and any error from generation.
func (c *ResponseCache) GetOrGenerate(
	ctx context.Context,
	keyPrefix string,
	diffContent string,
	generator func(context.Context) (string, error),
) (response string, cacheHit bool, err error) {
	if !c.enabled {
		// Cache disabled - generate directly
		response, err = generator(ctx)
		return response, false, err
	}

	// Compute hash once for both get and set operations
	cacheKey := keyPrefix + diffContent
	hash := hashDiff(cacheKey)

	// Check cache first using pre-computed hash
	if cached, ok := c.getByHash(hash); ok {
		c.mu.Lock()
		c.hits++
		c.mu.Unlock()
		return cached, true, nil
	}

	c.mu.Lock()
	c.misses++
	c.mu.Unlock()

	// Cache miss - generate new response
	response, err = generator(ctx)
	if err != nil {
		return "", false, err
	}

	// Store in cache using pre-computed hash
	c.setByHash(hash, response)

	return response, false, nil
}

// cacheEvictionPercentage defines what percentage of entries to evict when cache is full.
const cacheEvictionPercentage = 10

// evictOldest removes entries older than TTL/2 or the oldest 10% if needed.
// Must be called with mu held.
func (c *ResponseCache) evictOldest() {
	now := time.Now()
	toDelete := make([]string, 0)

	// First pass: remove entries older than TTL/2
	for hash, entry := range c.entries {
		if now.Sub(entry.CreatedAt) > c.ttl/2 {
			toDelete = append(toDelete, hash)
		}
	}

	for _, hash := range toDelete {
		delete(c.entries, hash)
	}

	// If still at capacity, remove oldest entries
	if len(c.entries) >= c.maxSize {
		// Find and remove the oldest entries (configured percentage)
		targetRemoval := c.maxSize / cacheEvictionPercentage
		if targetRemoval < 1 {
			targetRemoval = 1
		}

		type entryAge struct {
			hash string
			age  time.Duration
		}
		ages := make([]entryAge, 0, len(c.entries))
		for hash, entry := range c.entries {
			ages = append(ages, entryAge{hash: hash, age: now.Sub(entry.CreatedAt)})
		}

		// Sort by age descending (oldest first) - O(n log n) instead of O(n*k) selection
		sort.Slice(ages, func(i, j int) bool {
			return ages[i].age > ages[j].age
		})

		// Delete the oldest entries
		for i := 0; i < targetRemoval && i < len(ages); i++ {
			delete(c.entries, ages[i].hash)
		}
	}
}

// Stats returns cache statistics.
func (c *ResponseCache) Stats() (hits, misses int64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, len(c.entries)
}

// Clear removes all entries from the cache.
func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
	c.hits = 0
	c.misses = 0
}

// Size returns the current number of entries in the cache.
func (c *ResponseCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
