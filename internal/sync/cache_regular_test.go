package sync

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testBranch = "main"
)

func TestNewContentCache(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	// Test with valid parameters
	cache := NewContentCache(time.Minute, 1024*1024, logger)
	assert.NotNil(t, cache)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Size)
	assert.Equal(t, int64(0), stats.MemoryUsage)

	// Test with zero TTL (should use default)
	cache2 := NewContentCache(0, 1024, logger)
	assert.NotNil(t, cache2)
	defer func() {
		err := cache2.Close()
		require.NoError(t, err)
	}()

	// Test with zero memory limit (should use default)
	cache3 := NewContentCache(time.Minute, 0, logger)
	assert.NotNil(t, cache3)
	defer func() {
		err := cache3.Close()
		require.NoError(t, err)
	}()

	// Test with nil logger (should use default)
	cache4 := NewContentCache(time.Minute, 1024, nil)
	assert.NotNil(t, cache4)
	defer func() {
		err := cache4.Close()
		require.NoError(t, err)
	}()
}

func TestCacheKey_String(t *testing.T) {
	key := cacheKey{
		Repo:   "test/repo",
		Branch: "main",
		Path:   "path/to/file.txt",
	}

	expected := "test/repo:main:path/to/file.txt"
	assert.Equal(t, expected, key.String())
}

func TestContentCache_GetPutBasic(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()
	repo := "test/repo"
	branch := testBranch
	path := "file.txt"
	content := "Hello, World!"

	// Test cache miss
	result, hit, err := cache.Get(ctx, repo, branch, path)
	require.NoError(t, err)
	assert.False(t, hit)
	assert.Empty(t, result)

	// Test put
	err = cache.Put(ctx, repo, branch, path, content)
	require.NoError(t, err)

	// Test cache hit
	result, hit, err = cache.Get(ctx, repo, branch, path)
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, content, result)

	// Test stats
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(1), stats.Size)
	assert.Equal(t, int64(len(content)), stats.MemoryUsage)
	assert.InEpsilon(t, 0.5, stats.HitRate, 0.001)
}

func TestContentCache_ContextCancellation(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	// Test Get with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, hit, err := cache.Get(ctx, "repo", "branch", "file.txt")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.False(t, hit)
	assert.Empty(t, result)

	// Test Put with canceled context
	err = cache.Put(ctx, "repo", "branch", "file.txt", "content")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestContentCache_Deduplication(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()
	content := "shared content"

	// Store same content with different keys
	err := cache.Put(ctx, "repo1", "main", "file1.txt", content)
	require.NoError(t, err)

	err = cache.Put(ctx, "repo2", "main", "file2.txt", content)
	require.NoError(t, err)

	// Memory usage should only count content once
	stats := cache.GetStats()
	assert.Equal(t, int64(2), stats.Size)                   // 2 keys
	assert.Equal(t, int64(len(content)), stats.MemoryUsage) // But only one copy of content
}

func TestContentCache_TTLExpiration(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(10*time.Millisecond, 1024, logger) // Very short TTL
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()
	repo := "test/repo"
	branch := testBranch
	path := "file.txt"
	content := "expiring content"

	// Store content
	err := cache.Put(ctx, repo, branch, path, content)
	require.NoError(t, err)

	// Verify it's cached
	result, hit, err := cache.Get(ctx, repo, branch, path)
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, content, result)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Verify it's expired
	result, hit, err = cache.Get(ctx, repo, branch, path)
	require.NoError(t, err)
	assert.False(t, hit)
	assert.Empty(t, result)
}

func TestContentCache_MemoryLimits(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	memoryLimit := int64(100)
	cache := NewContentCache(time.Hour, memoryLimit, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	// Try to store content larger than limit
	largeContent := string(make([]byte, memoryLimit+10))
	err := cache.Put(ctx, "repo", "branch", "large.txt", largeContent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum cache size")

	// Store content that fits
	smallContent := "small"
	err = cache.Put(ctx, "repo", "branch", "small.txt", smallContent)
	require.NoError(t, err)

	// Verify memory usage
	stats := cache.GetStats()
	assert.Equal(t, int64(len(smallContent)), stats.MemoryUsage)
	assert.LessOrEqual(t, stats.MemoryUsage, memoryLimit)
}

func TestContentCache_LRUEviction(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 20, logger) // Small memory limit
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	// Add entries that will exceed memory limit
	entries := []struct {
		repo, branch, path, content string
	}{
		{"repo1", "main", "file1.txt", "content1"}, // 8 bytes
		{"repo1", "main", "file2.txt", "content2"}, // 8 bytes
		{"repo1", "main", "file3.txt", "content3"}, // 8 bytes (should trigger eviction)
	}

	// Add first two entries
	err := cache.Put(ctx, entries[0].repo, entries[0].branch, entries[0].path, entries[0].content)
	require.NoError(t, err)

	err = cache.Put(ctx, entries[1].repo, entries[1].branch, entries[1].path, entries[1].content)
	require.NoError(t, err)

	// Access first entry to make it most recently used
	_, _, err = cache.Get(ctx, entries[0].repo, entries[0].branch, entries[0].path)
	require.NoError(t, err)

	// Add third entry (should evict second entry as it's LRU)
	err = cache.Put(ctx, entries[2].repo, entries[2].branch, entries[2].path, entries[2].content)
	require.NoError(t, err)

	// Verify first and third entries are still cached
	_, hit1, err := cache.Get(ctx, entries[0].repo, entries[0].branch, entries[0].path)
	require.NoError(t, err)
	assert.True(t, hit1)

	_, hit3, err := cache.Get(ctx, entries[2].repo, entries[2].branch, entries[2].path)
	require.NoError(t, err)
	assert.True(t, hit3)

	// Verify second entry was evicted
	_, hit2, err := cache.Get(ctx, entries[1].repo, entries[1].branch, entries[1].path)
	require.NoError(t, err)
	assert.False(t, hit2)

	// Verify eviction stats
	stats := cache.GetStats()
	assert.Positive(t, stats.Evictions)
}

func TestContentCache_Invalidate(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	// Add entries for different repos and branches
	entries := []struct {
		repo, branch, path, content string
	}{
		{"repo1", "main", "file1.txt", "content1"},
		{"repo1", "dev", "file2.txt", "content2"},
		{"repo2", "main", "file3.txt", "content3"},
	}

	// Store all entries
	for _, entry := range entries {
		err := cache.Put(ctx, entry.repo, entry.branch, entry.path, entry.content)
		require.NoError(t, err)
	}

	// Verify all are cached
	for _, entry := range entries {
		_, hit, err := cache.Get(ctx, entry.repo, entry.branch, entry.path)
		require.NoError(t, err)
		assert.True(t, hit)
	}

	// Invalidate repo1/main
	cache.Invalidate("repo1", "main")

	// Verify repo1/main entry is invalidated
	_, hit, err := cache.Get(ctx, "repo1", "main", "file1.txt")
	require.NoError(t, err)
	assert.False(t, hit)

	// Verify other entries are still cached
	_, hit, err = cache.Get(ctx, "repo1", "dev", "file2.txt")
	require.NoError(t, err)
	assert.True(t, hit)

	_, hit, err = cache.Get(ctx, "repo2", "main", "file3.txt")
	require.NoError(t, err)
	assert.True(t, hit)
}

func TestContentCache_InvalidateAll(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	// Add some entries
	err := cache.Put(ctx, "repo1", "main", "file1.txt", "content1")
	require.NoError(t, err)
	err = cache.Put(ctx, "repo2", "dev", "file2.txt", "content2")
	require.NoError(t, err)

	// Verify entries exist
	stats := cache.GetStats()
	assert.Equal(t, int64(2), stats.Size)

	// Invalidate all
	cache.InvalidateAll()

	// Verify all entries are gone
	stats = cache.GetStats()
	assert.Equal(t, int64(0), stats.Size)
	assert.Equal(t, int64(0), stats.MemoryUsage)
	assert.Positive(t, stats.InvalidationID)

	// Verify gets return misses
	_, hit1, err := cache.Get(ctx, "repo1", "main", "file1.txt")
	require.NoError(t, err)
	assert.False(t, hit1)

	_, hit2, err := cache.Get(ctx, "repo2", "dev", "file2.txt")
	require.NoError(t, err)
	assert.False(t, hit2)
}

func TestContentCache_Warm(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()
	repo := "test/repo"
	branch := testBranch

	// Test warming with empty map
	err := cache.Warm(ctx, repo, branch, map[string]string{})
	require.NoError(t, err)

	// Test warming with files
	files := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
		"file3.txt": "content3",
	}

	err = cache.Warm(ctx, repo, branch, files)
	require.NoError(t, err)

	// Verify all files are cached
	for path, expectedContent := range files {
		result, hit, err := cache.Get(ctx, repo, branch, path)
		require.NoError(t, err)
		assert.True(t, hit)
		assert.Equal(t, expectedContent, result)
	}

	// Verify stats
	stats := cache.GetStats()
	assert.Equal(t, int64(len(files)), stats.Size)
	assert.Equal(t, int64(len(files)), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
}

func TestContentCache_WarmWithCancellation(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	files := map[string]string{
		"file1.txt": "content1",
	}

	err := cache.Warm(ctx, "repo", "branch", files)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestContentCache_HashContent(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	content := "test content"

	hash1 := cache.hashContent(content)
	hash2 := cache.hashContent(content)

	// Same content should produce same hash
	assert.Equal(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA256 produces 64-character hex string

	// Different content should produce different hash
	differentContent := "different content"
	hash3 := cache.hashContent(differentContent)
	assert.NotEqual(t, hash1, hash3)
}

func TestContentCache_GetStats(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	// Initial stats
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Evictions)
	assert.Equal(t, int64(0), stats.Size)
	assert.Equal(t, int64(0), stats.MemoryUsage)
	assert.InDelta(t, float64(0), stats.HitRate, 0.001)
	assert.Equal(t, int64(0), stats.InvalidationID)
	assert.False(t, stats.CreatedAt.IsZero())

	// Generate a miss
	_, _, err := cache.Get(ctx, "repo", "branch", "file.txt")
	require.NoError(t, err)

	stats = cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.InDelta(t, float64(0), stats.HitRate, 0.001)

	// Add content and generate a hit
	err = cache.Put(ctx, "repo", "branch", "file.txt", "content")
	require.NoError(t, err)

	_, _, err = cache.Get(ctx, "repo", "branch", "file.txt")
	require.NoError(t, err)

	stats = cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(1), stats.Size)
	assert.Equal(t, int64(len("content")), stats.MemoryUsage)
	assert.InDelta(t, float64(0.5), stats.HitRate, 0.001)
	assert.False(t, stats.LastAccessed.IsZero())
}

func TestContentCache_ThreadSafety(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()
	const numGoroutines = 10
	const numOps = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOps; j++ {
				path := fmt.Sprintf("file_%d_%d.txt", id, j)
				content := fmt.Sprintf("content_%d_%d", id, j)

				// Put content
				err := cache.Put(ctx, "repo", "branch", path, content)
				assert.NoError(t, err)

				// Get content
				result, hit, err := cache.Get(ctx, "repo", "branch", path)
				assert.NoError(t, err)
				if hit {
					assert.Equal(t, content, result)
				}

				// Get stats (should not race)
				_ = cache.GetStats()
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still functional
	stats := cache.GetStats()
	assert.Positive(t, stats.Hits+stats.Misses)
}

func TestContentCache_Close(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024, logger)

	// Close should succeed
	err := cache.Close()
	require.NoError(t, err)

	// Calling Close again should be safe
	err = cache.Close()
	require.NoError(t, err)
}

func TestContentCache_StaleKeyCleanup(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	// Put content
	err := cache.Put(ctx, "repo", "branch", "file.txt", "content")
	require.NoError(t, err)

	// Manually corrupt the cache state to simulate stale key mapping
	cache.mu.Lock()
	// Find the content ID and delete just the content entry (not the key mapping)
	key := cacheKey{Repo: "repo", Branch: "branch", Path: "file.txt"}
	contentID := cache.keyToContentID[key]
	delete(cache.contents, contentID) // Remove content but leave key mapping
	cache.mu.Unlock()

	// Get should detect stale mapping and clean it up
	_, hit, err := cache.Get(ctx, "repo", "branch", "file.txt")
	require.NoError(t, err)
	assert.False(t, hit)

	// Verify key mapping was cleaned up
	cache.mu.RLock()
	_, exists := cache.keyToContentID[key]
	cache.mu.RUnlock()
	assert.False(t, exists)
}

func TestContentCache_EvictExpired(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	// Very short TTL so entries expire quickly
	cache := NewContentCache(50*time.Millisecond, 1024*1024, logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	ctx := context.Background()

	// Insert 3 entries
	entries := []struct {
		repo, branch, path, content string
	}{
		{"repo1", "main", "file1.txt", "content1"},
		{"repo1", "main", "file2.txt", "content2"},
		{"repo1", "main", "file3.txt", "content3"},
	}
	for _, e := range entries {
		err := cache.Put(ctx, e.repo, e.branch, e.path, e.content)
		require.NoError(t, err)
	}

	// Verify all 3 are present
	stats := cache.GetStats()
	assert.Equal(t, int64(3), stats.Size)

	// Wait for TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Manually call evictExpired (internal method, same package)
	cache.evictExpired()

	// All entries should be gone
	statsAfter := cache.GetStats()
	assert.Equal(t, int64(0), statsAfter.Size)
	assert.Positive(t, statsAfter.Evictions)
}
