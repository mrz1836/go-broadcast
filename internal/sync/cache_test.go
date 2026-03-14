//go:build bench_heavy

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
	"github.com/stretchr/testify/suite"
)

// ContentCacheTestSuite provides comprehensive cache testing
type ContentCacheTestSuite struct {
	suite.Suite

	cache  *ContentCache
	logger *logrus.Entry
}

func (suite *ContentCacheTestSuite) SetupTest() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	suite.logger = logrus.NewEntry(logger)

	// Use short TTL and small memory limit for testing
	suite.cache = NewContentCache(100*time.Millisecond, 1024, suite.logger)
}

func (suite *ContentCacheTestSuite) TearDownTest() {
	if suite.cache != nil {
		err := suite.cache.Close()
		suite.Require().NoError(err)
	}
}

func TestContentCacheTestSuite(t *testing.T) {
	suite.Run(t, new(ContentCacheTestSuite))
}

// TestBasicGetPutOperations tests fundamental cache operations
func (suite *ContentCacheTestSuite) TestBasicGetPutOperations() {
	ctx := context.Background()
	repo := "test/repo"
	branch := "master"
	path := "file.txt"
	content := "Hello, World!"

	t := suite.T()

	// Test cache miss
	result, hit, err := suite.cache.Get(ctx, repo, branch, path)
	require.NoError(t, err)
	assert.False(t, hit)
	assert.Empty(t, result)

	// Test put
	err = suite.cache.Put(ctx, repo, branch, path, content)
	require.NoError(t, err)

	// Test cache hit
	result, hit, err = suite.cache.Get(ctx, repo, branch, path)
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, content, result)

	// Verify stats
	stats := suite.cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(1), stats.Size)
	assert.Equal(t, int64(len(content)), stats.MemoryUsage)
	assert.InEpsilon(t, 0.5, stats.HitRate, 0.001)
}

// TestContentDeduplication verifies that identical content is deduplicated
func (suite *ContentCacheTestSuite) TestContentDeduplication() {
	ctx := context.Background()
	content := "shared content"

	t := suite.T()

	// Store same content with different keys
	err := suite.cache.Put(ctx, "repo1", "master", "file1.txt", content)
	require.NoError(t, err)

	err = suite.cache.Put(ctx, "repo2", "master", "file2.txt", content)
	require.NoError(t, err)

	err = suite.cache.Put(ctx, "repo1", "dev", "file3.txt", content)
	require.NoError(t, err)

	// Verify all can be retrieved
	result1, hit1, err := suite.cache.Get(ctx, "repo1", "master", "file1.txt")
	require.NoError(t, err)
	assert.True(t, hit1)
	assert.Equal(t, content, result1)

	result2, hit2, err := suite.cache.Get(ctx, "repo2", "master", "file2.txt")
	require.NoError(t, err)
	assert.True(t, hit2)
	assert.Equal(t, content, result2)

	result3, hit3, err := suite.cache.Get(ctx, "repo1", "dev", "file3.txt")
	require.NoError(t, err)
	assert.True(t, hit3)
	assert.Equal(t, content, result3)

	// Verify memory usage (should only store content once)
	stats := suite.cache.GetStats()
	assert.Equal(t, int64(3), stats.Size)                   // 3 keys
	assert.Equal(t, int64(len(content)), stats.MemoryUsage) // But only one copy of content
}

// TestTTLExpiration verifies that entries expire after TTL
func (suite *ContentCacheTestSuite) TestTTLExpiration() {
	ctx := context.Background()
	repo := "test/repo"
	branch := "master"
	path := "file.txt"
	content := "expiring content"

	t := suite.T()

	// Store content
	err := suite.cache.Put(ctx, repo, branch, path, content)
	require.NoError(t, err)

	// Verify it's cached
	result, hit, err := suite.cache.Get(ctx, repo, branch, path)
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, content, result)

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Verify it's expired
	result, hit, err = suite.cache.Get(ctx, repo, branch, path)
	require.NoError(t, err)
	assert.False(t, hit)
	assert.Empty(t, result)

	// Verify eviction was counted
	stats := suite.cache.GetStats()
	assert.Equal(t, int64(1), stats.Evictions)
}

// TestLRUEviction verifies LRU eviction behavior when memory limit is exceeded
func (suite *ContentCacheTestSuite) TestLRUEviction() {
	ctx := context.Background()
	t := suite.T()

	// Create cache with small memory limit that allows 2 entries but not 3
	cache := NewContentCache(time.Hour, 20, suite.logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	// Add entries that will exceed memory limit
	entries := []struct {
		repo, branch, path, content string
	}{
		{"repo1", "master", "file1.txt", "content1"}, // 8 bytes
		{"repo1", "master", "file2.txt", "content2"}, // 8 bytes
		{"repo1", "master", "file3.txt", "content3"}, // 8 bytes (should trigger eviction)
	}

	// Add first entry
	err := cache.Put(ctx, entries[0].repo, entries[0].branch, entries[0].path, entries[0].content)
	require.NoError(t, err)

	// Add second entry
	err = cache.Put(ctx, entries[1].repo, entries[1].branch, entries[1].path, entries[1].content)
	require.NoError(t, err)

	// Verify both are cached
	result1, hit1, err := cache.Get(ctx, entries[0].repo, entries[0].branch, entries[0].path)
	require.NoError(t, err)
	assert.True(t, hit1)
	assert.Equal(t, entries[0].content, result1)

	result2, hit2, err := cache.Get(ctx, entries[1].repo, entries[1].branch, entries[1].path)
	require.NoError(t, err)
	assert.True(t, hit2)
	assert.Equal(t, entries[1].content, result2)

	// Access first entry again to make it most recently used
	_, _, err = cache.Get(ctx, entries[0].repo, entries[0].branch, entries[0].path)
	require.NoError(t, err)

	// Add third entry (should evict second entry as it's LRU)
	err = cache.Put(ctx, entries[2].repo, entries[2].branch, entries[2].path, entries[2].content)
	require.NoError(t, err)

	// Verify first and third entries are still cached
	result1, hit1, err = cache.Get(ctx, entries[0].repo, entries[0].branch, entries[0].path)
	require.NoError(t, err)
	assert.True(t, hit1)
	assert.Equal(t, entries[0].content, result1)

	result3, hit3, err := cache.Get(ctx, entries[2].repo, entries[2].branch, entries[2].path)
	require.NoError(t, err)
	assert.True(t, hit3)
	assert.Equal(t, entries[2].content, result3)

	// Verify second entry was evicted
	result2, hit2, err = cache.Get(ctx, entries[1].repo, entries[1].branch, entries[1].path)
	require.NoError(t, err)
	assert.False(t, hit2)
	assert.Empty(t, result2)

	// Verify eviction stats
	stats := cache.GetStats()
	assert.Positive(t, stats.Evictions)
}

// TestCacheInvalidation verifies cache invalidation functionality
func (suite *ContentCacheTestSuite) TestCacheInvalidation() {
	ctx := context.Background()
	t := suite.T()

	// Add entries for different repos and branches
	entries := []struct {
		repo, branch, path, content string
	}{
		{"repo1", "master", "file1.txt", "content1"},
		{"repo1", "dev", "file2.txt", "content2"},
		{"repo2", "master", "file3.txt", "content3"},
		{"repo2", "dev", "file4.txt", "content4"},
	}

	// Store all entries
	for _, entry := range entries {
		err := suite.cache.Put(ctx, entry.repo, entry.branch, entry.path, entry.content)
		require.NoError(t, err)
	}

	// Verify all are cached
	for _, entry := range entries {
		result, hit, err := suite.cache.Get(ctx, entry.repo, entry.branch, entry.path)
		require.NoError(t, err)
		assert.True(t, hit)
		assert.Equal(t, entry.content, result)
	}

	// Invalidate repo1/main
	suite.cache.Invalidate("repo1", "master")

	// Verify repo1/main entries are invalidated
	result, hit, err := suite.cache.Get(ctx, "repo1", "master", "file1.txt")
	require.NoError(t, err)
	assert.False(t, hit)
	assert.Empty(t, result)

	// Verify other entries are still cached
	result, hit, err = suite.cache.Get(ctx, "repo1", "dev", "file2.txt")
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, "content2", result)

	result, hit, err = suite.cache.Get(ctx, "repo2", "master", "file3.txt")
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, "content3", result)

	// Test InvalidateAll
	suite.cache.InvalidateAll()

	// Verify all entries are invalidated
	for _, entry := range entries[1:] { // Skip first entry as it was already invalidated
		result, hit, err := suite.cache.Get(ctx, entry.repo, entry.branch, entry.path)
		require.NoError(t, err)
		assert.False(t, hit)
		assert.Empty(t, result)
	}

	// Verify stats show invalidation
	stats := suite.cache.GetStats()
	assert.Equal(t, int64(0), stats.Size)
	assert.Equal(t, int64(0), stats.MemoryUsage)
	assert.Positive(t, stats.InvalidationID)
}

// TestCacheWarming verifies cache warming functionality
func (suite *ContentCacheTestSuite) TestCacheWarming() {
	ctx := context.Background()
	t := suite.T()

	repo := "test/repo"
	branch := "master"
	files := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
		"file3.txt": "content3",
		"file4.txt": "content4",
		"file5.txt": "content5",
	}

	// Warm cache
	err := suite.cache.Warm(ctx, repo, branch, files)
	require.NoError(t, err)

	// Verify all files are cached
	for path, expectedContent := range files {
		result, hit, err := suite.cache.Get(ctx, repo, branch, path)
		require.NoError(t, err)
		assert.True(t, hit)
		assert.Equal(t, expectedContent, result)
	}

	// Verify stats
	stats := suite.cache.GetStats()
	assert.Equal(t, int64(len(files)), stats.Size)
	assert.Equal(t, int64(len(files)), stats.Hits) // All gets were hits
	assert.Equal(t, int64(0), stats.Misses)
}

// TestCacheWarmingWithCancellation verifies cache warming respects context cancellation
func (suite *ContentCacheTestSuite) TestCacheWarmingWithCancellation() {
	t := suite.T()

	repo := "test/repo"
	branch := "master"

	// Create many files to warm
	files := make(map[string]string)
	for i := 0; i < 1000; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = fmt.Sprintf("content%d", i)
	}

	// Create context that cancels very quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Add a small delay to ensure context is definitely canceled
	time.Sleep(1 * time.Millisecond)

	// Warm cache (should be canceled)
	err := suite.cache.Warm(ctx, repo, branch, files)
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

// TestThreadSafety verifies concurrent access is safe
func (suite *ContentCacheTestSuite) TestThreadSafety() {
	ctx := context.Background()
	t := suite.T()

	const numGoroutines = 100
	const numOperationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			repo := fmt.Sprintf("repo%d", id%10)
			branch := fmt.Sprintf("branch%d", id%5)

			for j := 0; j < numOperationsPerGoroutine; j++ {
				path := fmt.Sprintf("file%d_%d.txt", id, j)
				content := fmt.Sprintf("content_%d_%d", id, j)

				// Put content
				err := suite.cache.Put(ctx, repo, branch, path, content)
				assert.NoError(t, err)

				// Get content
				result, hit, err := suite.cache.Get(ctx, repo, branch, path)
				assert.NoError(t, err)
				if hit {
					assert.Equal(t, content, result)
				}

				// Occasionally invalidate
				if j%20 == 0 {
					suite.cache.Invalidate(repo, branch)
				}

				// Get stats (should not race)
				_ = suite.cache.GetStats()
			}
		}(i)
	}

	wg.Wait()

	// Verify cache is still functional
	stats := suite.cache.GetStats()
	suite.Positive(stats.Hits + stats.Misses)
	suite.Positive(stats.InvalidationID)
}

// TestMemoryLimits verifies memory limit enforcement
func (suite *ContentCacheTestSuite) TestMemoryLimits() {
	ctx := context.Background()
	t := suite.T()

	// Create cache with strict memory limit
	memoryLimit := int64(100)
	cache := NewContentCache(time.Hour, memoryLimit, suite.logger)
	defer func() {
		err := cache.Close()
		require.NoError(t, err)
	}()

	// Try to store content larger than limit
	largeContent := string(make([]byte, memoryLimit+10))
	err := cache.Put(ctx, "repo", "branch", "large.txt", largeContent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum cache size")

	// Store content that fits
	smallContent := "small"
	err = cache.Put(ctx, "repo", "branch", "small.txt", smallContent)
	require.NoError(t, err)

	// Verify memory usage is tracked
	stats := cache.GetStats()
	assert.Equal(t, int64(len(smallContent)), stats.MemoryUsage)
	assert.LessOrEqual(t, stats.MemoryUsage, memoryLimit)
}

// TestContextCancellation verifies context cancellation is respected
func (suite *ContentCacheTestSuite) TestContextCancellation() {
	t := suite.T()

	// Test Get with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, hit, err := suite.cache.Get(ctx, "repo", "branch", "file.txt")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.False(t, hit)
	assert.Empty(t, result)

	// Test Put with canceled context
	err = suite.cache.Put(ctx, "repo", "branch", "file.txt", "content")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestCacheStatsAccuracy verifies cache statistics accuracy
func (suite *ContentCacheTestSuite) TestCacheStatsAccuracy() {
	ctx := context.Background()
	t := suite.T()

	repo := "test/repo"
	branch := "master"

	// Initial stats
	stats := suite.cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Size)
	assert.Equal(t, int64(0), stats.MemoryUsage)
	assert.InDelta(t, float64(0), stats.HitRate, 0.001)

	// Generate some cache misses
	for i := 0; i < 5; i++ {
		_, hit, err := suite.cache.Get(ctx, repo, branch, fmt.Sprintf("file%d.txt", i))
		require.NoError(t, err)
		assert.False(t, hit)
	}

	stats = suite.cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(5), stats.Misses)
	assert.InDelta(t, float64(0), stats.HitRate, 0.001)

	// Add some content
	for i := 0; i < 3; i++ {
		err := suite.cache.Put(ctx, repo, branch, fmt.Sprintf("file%d.txt", i), fmt.Sprintf("content%d", i))
		require.NoError(t, err)
	}

	stats = suite.cache.GetStats()
	assert.Equal(t, int64(3), stats.Size)
	totalContentSize := int64(len("content0") + len("content1") + len("content2"))
	assert.Equal(t, totalContentSize, stats.MemoryUsage)

	// Generate some cache hits
	for i := 0; i < 3; i++ {
		_, hit, err := suite.cache.Get(ctx, repo, branch, fmt.Sprintf("file%d.txt", i))
		require.NoError(t, err)
		assert.True(t, hit)
	}

	stats = suite.cache.GetStats()
	assert.Equal(t, int64(3), stats.Hits)
	assert.Equal(t, int64(5), stats.Misses)
	assert.InDelta(t, float64(3)/float64(8), stats.HitRate, 0.001) // 3 hits out of 8 total

	// Test invalidation increments ID
	oldInvalidationID := stats.InvalidationID
	suite.cache.Invalidate(repo, branch)

	stats = suite.cache.GetStats()
	assert.Greater(t, stats.InvalidationID, oldInvalidationID)
	assert.Equal(t, int64(0), stats.Size)
	assert.Equal(t, int64(0), stats.MemoryUsage)
}

// TestCleanupGoroutine verifies the cleanup goroutine works correctly
func (suite *ContentCacheTestSuite) TestCleanupGoroutine() {
	t := suite.T()

	// Create cache with very short cleanup interval for testing
	cache := NewContentCache(50*time.Millisecond, 1024, suite.logger)

	ctx := context.Background()

	// Add content that will expire
	err := cache.Put(ctx, "repo", "master", "file.txt", "content")
	require.NoError(t, err)

	// Verify it's cached
	_, hit, err := cache.Get(ctx, "repo", "master", "file.txt")
	require.NoError(t, err)
	assert.True(t, hit)

	// Wait for expiration and cleanup
	time.Sleep(400 * time.Millisecond)

	// Verify it's cleaned up
	_, hit, err = cache.Get(ctx, "repo", "master", "file.txt")
	require.NoError(t, err)
	assert.False(t, hit)

	// Close cache and verify cleanup goroutine stops
	err = cache.Close()
	require.NoError(t, err)

	// Calling Close again should be safe
	err = cache.Close()
	require.NoError(t, err)
}

// TestHashContentConsistency verifies content hashing is consistent
func (suite *ContentCacheTestSuite) TestHashContentConsistency() {
	t := suite.T()

	content := "test content for hashing"

	hash1 := suite.cache.hashContent(content)
	hash2 := suite.cache.hashContent(content)

	assert.Equal(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA256 produces 64-character hex string

	// Different content should produce different hash
	differentContent := "different content"
	hash3 := suite.cache.hashContent(differentContent)
	assert.NotEqual(t, hash1, hash3)
}

// BenchmarkCacheOperations benchmarks cache performance
func BenchmarkCacheOperations(b *testing.B) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 100*1024*1024, logger)
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()
	content := "benchmark content"

	b.Run("Put", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cache.Put(ctx, "repo", "branch", fmt.Sprintf("file%d.txt", i), content)
		}
	})

	// Pre-populate cache for Get benchmark
	for i := 0; i < 1000; i++ {
		_ = cache.Put(ctx, "repo", "branch", fmt.Sprintf("file%d.txt", i), content)
	}

	b.Run("Get", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, _ = cache.Get(ctx, "repo", "branch", fmt.Sprintf("file%d.txt", i%1000))
		}
	})

	b.Run("GetStats", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cache.GetStats()
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent cache access
func BenchmarkConcurrentOperations(b *testing.B) {
	logger := logrus.NewEntry(logrus.New())
	cache := NewContentCache(time.Hour, 100*1024*1024, logger)
	defer func() {
		_ = cache.Close()
	}()

	ctx := context.Background()
	content := "benchmark content"

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			path := fmt.Sprintf("file%d.txt", i%1000)
			if i%2 == 0 {
				_ = cache.Put(ctx, "repo", "branch", path, content)
			} else {
				_, _, _ = cache.Get(ctx, "repo", "branch", path)
			}
			i++
		}
	})
}
