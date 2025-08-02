package sync

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// Static error variables
var (
	ErrContentSizeExceedsCache = errors.New("content size exceeds maximum cache size")
)

// CacheStats provides cache performance metrics
type CacheStats struct {
	Hits           int64     `json:"hits"`
	Misses         int64     `json:"misses"`
	Evictions      int64     `json:"evictions"`
	Size           int64     `json:"size"`
	MemoryUsage    int64     `json:"memory_usage_bytes"`
	HitRate        float64   `json:"hit_rate"`
	LastAccessed   time.Time `json:"last_accessed"`
	CreatedAt      time.Time `json:"created_at"`
	InvalidationID int64     `json:"invalidation_id"`
}

// ContentEntry represents a cached file content with metadata
type ContentEntry struct {
	Content    string
	Hash       string
	Size       int64
	ExpiresAt  time.Time
	AccessedAt time.Time
	CreatedAt  time.Time
}

// cacheKey represents the composite key for cache entries
type cacheKey struct {
	Repo   string
	Branch string
	Path   string
}

// String returns the string representation of the cache key
func (k cacheKey) String() string {
	return fmt.Sprintf("%s:%s:%s", k.Repo, k.Branch, k.Path)
}

// lruNode represents a node in the LRU doubly-linked list
type lruNode struct {
	key       cacheKey
	contentID string // SHA256 hash of content
	prev      *lruNode
	next      *lruNode
}

// ContentCache provides thread-safe caching of file contents with LRU eviction
type ContentCache struct {
	mu             sync.RWMutex
	contents       map[string]*ContentEntry // contentID -> entry
	keyToContentID map[cacheKey]string      // key -> contentID mapping
	lruHead        *lruNode                 // most recently used
	lruTail        *lruNode                 // least recently used

	// Configuration
	ttl            time.Duration
	maxMemoryBytes int64
	logger         *logrus.Entry

	// Metrics
	hits           atomic.Int64
	misses         atomic.Int64
	evictions      atomic.Int64
	currentMemory  atomic.Int64
	lastAccessed   atomic.Int64 // unix timestamp
	createdAt      time.Time
	invalidationID atomic.Int64

	// Cleanup
	stopCleanup chan struct{}
	cleanupOnce sync.Once
}

// NewContentCache creates a new content cache with the specified configuration
func NewContentCache(ttl time.Duration, maxMemoryBytes int64, logger *logrus.Entry) *ContentCache {
	if ttl <= 0 {
		ttl = 15 * time.Minute // Default 15-minute TTL
	}
	if maxMemoryBytes <= 0 {
		maxMemoryBytes = 100 * 1024 * 1024 // Default 100MB
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}

	cache := &ContentCache{
		contents:       make(map[string]*ContentEntry),
		keyToContentID: make(map[cacheKey]string),
		ttl:            ttl,
		maxMemoryBytes: maxMemoryBytes,
		logger:         logger.WithField("component", "content_cache"),
		stopCleanup:    make(chan struct{}),
		createdAt:      time.Now(),
	}

	// Initialize LRU list with sentinel nodes
	cache.lruHead = &lruNode{}
	cache.lruTail = &lruNode{}
	cache.lruHead.next = cache.lruTail
	cache.lruTail.prev = cache.lruHead

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves content from cache, returning content, hit/miss status, and error
func (c *ContentCache) Get(ctx context.Context, repo, branch, path string) (string, bool, error) {
	if ctx.Err() != nil {
		return "", false, ctx.Err()
	}

	key := cacheKey{Repo: repo, Branch: branch, Path: path}

	c.mu.RLock()
	contentID, exists := c.keyToContentID[key]
	if !exists {
		c.mu.RUnlock()
		c.misses.Add(1)
		return "", false, nil
	}

	entry, exists := c.contents[contentID]
	if !exists {
		c.mu.RUnlock()
		// Clean up stale key mapping
		c.mu.Lock()
		delete(c.keyToContentID, key)
		c.mu.Unlock()
		c.misses.Add(1)
		return "", false, nil
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		c.mu.RUnlock()
		c.evictExpired()
		c.misses.Add(1)
		return "", false, nil
	}
	c.mu.RUnlock()

	// Update access time and LRU position
	c.mu.Lock()
	entry.AccessedAt = time.Now()
	c.moveToHead(key, contentID)
	content := entry.Content
	c.mu.Unlock()

	c.hits.Add(1)
	c.lastAccessed.Store(time.Now().Unix())

	c.logger.WithFields(logrus.Fields{
		"repo":       repo,
		"branch":     branch,
		"path":       path,
		"content_id": contentID[:8],
		"size":       entry.Size,
	}).Debug("Cache hit")

	return content, true, nil
}

// Put stores content in cache with automatic deduplication and LRU management
func (c *ContentCache) Put(ctx context.Context, repo, branch, path, content string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	key := cacheKey{Repo: repo, Branch: branch, Path: path}
	contentID := c.hashContent(content)
	contentSize := int64(len(content))
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if content already exists (deduplication)
	if existing, exists := c.contents[contentID]; exists {
		// Update existing entry's expiration and access time
		existing.ExpiresAt = now.Add(c.ttl)
		existing.AccessedAt = now

		// Update key mapping and LRU position
		c.keyToContentID[key] = contentID
		c.moveToHead(key, contentID)

		c.logger.WithFields(logrus.Fields{
			"repo":       repo,
			"branch":     branch,
			"path":       path,
			"content_id": contentID[:8],
			"size":       contentSize,
		}).Debug("Cache put (deduplicated)")

		return nil
	}

	// Check memory limits and evict if necessary
	if err := c.ensureMemoryLimit(contentSize); err != nil {
		return fmt.Errorf("failed to ensure memory limit: %w", err)
	}

	// Create new entry
	entry := &ContentEntry{
		Content:    content,
		Hash:       contentID,
		Size:       contentSize,
		ExpiresAt:  now.Add(c.ttl),
		AccessedAt: now,
		CreatedAt:  now,
	}

	// Store content and update mappings
	c.contents[contentID] = entry
	c.keyToContentID[key] = contentID
	c.addToHead(key, contentID)
	c.currentMemory.Add(contentSize)

	c.logger.WithFields(logrus.Fields{
		"repo":       repo,
		"branch":     branch,
		"path":       path,
		"content_id": contentID[:8],
		"size":       contentSize,
		"memory":     c.currentMemory.Load(),
	}).Debug("Cache put (new)")

	return nil
}

// Invalidate removes all cached entries for a specific repository and branch
func (c *ContentCache) Invalidate(repo, branch string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.invalidationID.Add(1)
	keysToDelete := make([]cacheKey, 0)

	// Find all keys matching repo and branch
	for key := range c.keyToContentID {
		if key.Repo == repo && key.Branch == branch {
			keysToDelete = append(keysToDelete, key)
		}
	}

	// Remove entries
	for _, key := range keysToDelete {
		c.removeKey(key)
	}

	c.logger.WithFields(logrus.Fields{
		"repo":    repo,
		"branch":  branch,
		"removed": len(keysToDelete),
		"memory":  c.currentMemory.Load(),
	}).Info("Cache invalidated")
}

// InvalidateAll clears the entire cache
func (c *ContentCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.invalidationID.Add(1)
	removedCount := len(c.keyToContentID)

	// Clear all data structures
	c.contents = make(map[string]*ContentEntry)
	c.keyToContentID = make(map[cacheKey]string)
	c.lruHead.next = c.lruTail
	c.lruTail.prev = c.lruHead
	c.currentMemory.Store(0)

	c.logger.WithField("removed", removedCount).Info("Cache invalidated (all)")
}

// GetStats returns current cache statistics
func (c *ContentCache) GetStats() CacheStats {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses

	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	lastAccessedTime := time.Unix(c.lastAccessed.Load(), 0)
	if c.lastAccessed.Load() == 0 {
		lastAccessedTime = c.createdAt
	}

	c.mu.RLock()
	size := int64(len(c.keyToContentID))
	c.mu.RUnlock()

	return CacheStats{
		Hits:           hits,
		Misses:         misses,
		Evictions:      c.evictions.Load(),
		Size:           size,
		MemoryUsage:    c.currentMemory.Load(),
		HitRate:        hitRate,
		LastAccessed:   lastAccessedTime,
		CreatedAt:      c.createdAt,
		InvalidationID: c.invalidationID.Load(),
	}
}

// Warm preloads cache with known files that haven't changed
func (c *ContentCache) Warm(ctx context.Context, repo, branch string, files map[string]string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if len(files) == 0 {
		return nil
	}

	warmedCount := 0
	for path, content := range files {
		if err := c.Put(ctx, repo, branch, path, content); err != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"repo":   repo,
				"branch": branch,
				"path":   path,
			}).Warn("Failed to warm cache entry")
			continue
		}
		warmedCount++

		// Check for cancellation periodically
		if warmedCount%100 == 0 {
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}

	c.logger.WithFields(logrus.Fields{
		"repo":   repo,
		"branch": branch,
		"warmed": warmedCount,
		"total":  len(files),
		"memory": c.currentMemory.Load(),
	}).Info("Cache warmed")

	return nil
}

// Close stops the cleanup goroutine and releases resources
func (c *ContentCache) Close() error {
	c.cleanupOnce.Do(func() {
		close(c.stopCleanup)
	})
	return nil
}

// hashContent generates SHA256 hash of content for deduplication
func (c *ContentCache) hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// ensureMemoryLimit evicts entries if necessary to stay within memory limits
func (c *ContentCache) ensureMemoryLimit(newSize int64) error {
	currentMemory := c.currentMemory.Load()
	if currentMemory+newSize <= c.maxMemoryBytes {
		return nil
	}

	// Calculate how much memory we need to free
	targetMemory := c.maxMemoryBytes - newSize
	if targetMemory < 0 {
		return fmt.Errorf("%w: content size %d exceeds maximum cache size %d", ErrContentSizeExceedsCache, newSize, c.maxMemoryBytes)
	}

	// Evict LRU entries until we're under the limit
	evicted := 0
	current := c.lruTail.prev

	for current != c.lruHead && c.currentMemory.Load() > targetMemory {
		next := current.prev
		c.removeKey(current.key)
		current = next
		evicted++
	}

	if evicted > 0 {
		c.evictions.Add(int64(evicted))
		c.logger.WithFields(logrus.Fields{
			"evicted":       evicted,
			"memory_before": currentMemory,
			"memory_after":  c.currentMemory.Load(),
			"target_memory": targetMemory,
		}).Debug("LRU eviction completed")
	}

	return nil
}

// removeKey removes a key from all data structures
func (c *ContentCache) removeKey(key cacheKey) {
	contentID, exists := c.keyToContentID[key]
	if !exists {
		return
	}

	// Remove from key mapping
	delete(c.keyToContentID, key)

	// Check if this content is still referenced by other keys
	stillReferenced := false
	for _, otherContentID := range c.keyToContentID {
		if otherContentID == contentID {
			stillReferenced = true
			break
		}
	}

	// If no other keys reference this content, remove it
	if !stillReferenced {
		if entry, exists := c.contents[contentID]; exists {
			c.currentMemory.Add(-entry.Size)
			delete(c.contents, contentID)
		}
	}

	// Remove from LRU list
	c.removeFromLRU(key)
}

// LRU list management methods

// addToHead adds a new node to the head of the LRU list
func (c *ContentCache) addToHead(key cacheKey, contentID string) {
	node := &lruNode{
		key:       key,
		contentID: contentID,
	}

	node.prev = c.lruHead
	node.next = c.lruHead.next
	c.lruHead.next.prev = node
	c.lruHead.next = node
}

// moveToHead moves an existing node to the head of the LRU list
func (c *ContentCache) moveToHead(key cacheKey, contentID string) {
	c.removeFromLRU(key)
	c.addToHead(key, contentID)
}

// removeFromLRU removes a node from the LRU list
func (c *ContentCache) removeFromLRU(key cacheKey) {
	// Find and remove the node
	current := c.lruHead.next
	for current != c.lruTail {
		if current.key == key {
			current.prev.next = current.next
			current.next.prev = current.prev
			break
		}
		current = current.next
	}
}

// evictExpired removes expired entries from the cache
func (c *ContentCache) evictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expiredKeys := make([]cacheKey, 0)

	// Find expired keys
	for key, contentID := range c.keyToContentID {
		if entry, exists := c.contents[contentID]; exists {
			if now.After(entry.ExpiresAt) {
				expiredKeys = append(expiredKeys, key)
			}
		}
	}

	// Remove expired keys
	for _, key := range expiredKeys {
		c.removeKey(key)
	}

	if len(expiredKeys) > 0 {
		c.evictions.Add(int64(len(expiredKeys)))
		c.logger.WithFields(logrus.Fields{
			"expired": len(expiredKeys),
			"memory":  c.currentMemory.Load(),
		}).Debug("Expired entries evicted")
	}
}

// cleanup periodically removes expired entries
func (c *ContentCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.evictExpired()
		case <-c.stopCleanup:
			return
		}
	}
}
