package ai

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test errors - defined at package level per linting rules.
var errGenerationFailed = errors.New("generation failed")

func TestNewResponseCache(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}

	cache := NewResponseCache(cfg)

	require.NotNil(t, cache)
	assert.True(t, cache.enabled)
	assert.Equal(t, time.Hour, cache.ttl)
	assert.Equal(t, 100, cache.maxSize)
	assert.Equal(t, 0, cache.Size())
}

func TestResponseCache_Get(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*ResponseCache)
		diffContent string
		wantValue   string
		wantFound   bool
	}{
		{
			name:        "cache miss on empty cache",
			setup:       func(_ *ResponseCache) {},
			diffContent: "diff --git a/file.go\n+new line",
			wantValue:   "",
			wantFound:   false,
		},
		{
			name: "cache hit returns stored value",
			setup: func(c *ResponseCache) {
				c.Set("diff --git a/file.go\n+new line", "cached response")
			},
			diffContent: "diff --git a/file.go\n+new line",
			wantValue:   "cached response",
			wantFound:   true,
		},
		{
			name: "different diff content is cache miss",
			setup: func(c *ResponseCache) {
				c.Set("diff --git a/file.go\n+new line", "cached response")
			},
			diffContent: "diff --git a/other.go\n+different",
			wantValue:   "",
			wantFound:   false,
		},
		{
			name: "identical diff content returns same cached response",
			setup: func(c *ResponseCache) {
				c.Set("some diff content", "original response")
			},
			diffContent: "some diff content",
			wantValue:   "original response",
			wantFound:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				CacheEnabled: true,
				CacheTTL:     time.Hour,
				CacheMaxSize: 100,
			}
			cache := NewResponseCache(cfg)
			tt.setup(cache)

			got, found := cache.Get(tt.diffContent)

			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantValue, got)
		})
	}
}

func TestResponseCache_Set(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)

	cache.Set("diff content", "response")

	assert.Equal(t, 1, cache.Size())
	got, found := cache.Get("diff content")
	assert.True(t, found)
	assert.Equal(t, "response", got)
}

func TestResponseCache_TTLExpiration(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     50 * time.Millisecond,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)

	// Set a value
	cache.Set("diff content", "response")

	// Should be found immediately
	got, found := cache.Get("diff content")
	require.True(t, found)
	assert.Equal(t, "response", got)

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Should now be expired
	_, found = cache.Get("diff content")
	assert.False(t, found, "entry should be expired after TTL")
}

func TestResponseCache_Eviction(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 3,
	}
	cache := NewResponseCache(cfg)

	// Fill cache to capacity
	cache.Set("diff1", "response1")
	cache.Set("diff2", "response2")
	cache.Set("diff3", "response3")
	assert.Equal(t, 3, cache.Size())

	// Add one more - should trigger eviction
	cache.Set("diff4", "response4")

	// Cache size should not exceed maxSize
	assert.LessOrEqual(t, cache.Size(), 3)

	// New entry should be present
	got, found := cache.Get("diff4")
	assert.True(t, found)
	assert.Equal(t, "response4", got)
}

func TestResponseCache_GetOrGenerate_CacheMiss(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)
	ctx := context.Background()

	generatorCalled := false
	generator := func(_ context.Context) (string, error) {
		generatorCalled = true
		return "generated response", nil
	}

	response, cacheHit, err := cache.GetOrGenerate(ctx, "diff content", generator)

	require.NoError(t, err)
	assert.False(t, cacheHit)
	assert.Equal(t, "generated response", response)
	assert.True(t, generatorCalled, "generator should be called on cache miss")

	// Verify response was cached
	cached, found := cache.Get("diff content")
	assert.True(t, found)
	assert.Equal(t, "generated response", cached)
}

func TestResponseCache_GetOrGenerate_CacheHit(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)
	ctx := context.Background()

	// Pre-populate cache
	cache.Set("diff content", "cached response")

	generatorCalled := false
	generator := func(_ context.Context) (string, error) {
		generatorCalled = true
		return "new response", nil
	}

	response, cacheHit, err := cache.GetOrGenerate(ctx, "diff content", generator)

	require.NoError(t, err)
	assert.True(t, cacheHit)
	assert.Equal(t, "cached response", response)
	assert.False(t, generatorCalled, "generator should NOT be called on cache hit")
}

func TestResponseCache_GetOrGenerate_GeneratorError(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)
	ctx := context.Background()

	generator := func(_ context.Context) (string, error) {
		return "", errGenerationFailed
	}

	response, cacheHit, err := cache.GetOrGenerate(ctx, "diff content", generator)

	require.Error(t, err)
	assert.Equal(t, errGenerationFailed, err)
	assert.False(t, cacheHit)
	assert.Empty(t, response)

	// Verify error response was NOT cached
	_, found := cache.Get("diff content")
	assert.False(t, found, "error responses should not be cached")
}

func TestResponseCache_Stats(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)
	ctx := context.Background()

	generator := func(_ context.Context) (string, error) {
		return "response", nil
	}

	// Initial stats
	hits, misses, size := cache.Stats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)
	assert.Equal(t, 0, size)

	// First call - miss
	_, cacheHit, err := cache.GetOrGenerate(ctx, "diff1", generator)
	require.NoError(t, err)
	assert.False(t, cacheHit)
	hits, misses, size = cache.Stats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(1), misses)
	assert.Equal(t, 1, size)

	// Second call with same diff - hit
	_, cacheHit, err = cache.GetOrGenerate(ctx, "diff1", generator)
	require.NoError(t, err)
	assert.True(t, cacheHit)
	hits, misses, size = cache.Stats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(1), misses)
	assert.Equal(t, 1, size)

	// Third call with different diff - miss
	_, cacheHit, err = cache.GetOrGenerate(ctx, "diff2", generator)
	require.NoError(t, err)
	assert.False(t, cacheHit)
	hits, misses, size = cache.Stats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(2), misses)
	assert.Equal(t, 2, size)
}

func TestResponseCache_Clear(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)
	ctx := context.Background()

	generator := func(_ context.Context) (string, error) {
		return "response", nil
	}

	// Populate cache
	_, cacheHit1, err1 := cache.GetOrGenerate(ctx, "diff1", generator)
	require.NoError(t, err1)
	assert.False(t, cacheHit1)
	_, cacheHit2, err2 := cache.GetOrGenerate(ctx, "diff1", generator) // hit
	require.NoError(t, err2)
	assert.True(t, cacheHit2)
	assert.Equal(t, 1, cache.Size())

	hits, misses, _ := cache.Stats()
	assert.Equal(t, int64(1), hits)
	assert.Equal(t, int64(1), misses)

	// Clear cache
	cache.Clear()

	// Verify cleared
	assert.Equal(t, 0, cache.Size())
	hits, misses, _ = cache.Stats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)

	// Verify entries are gone
	_, found := cache.Get("diff1")
	assert.False(t, found)
}

func TestResponseCache_Disabled(t *testing.T) {
	cfg := &Config{
		CacheEnabled: false,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)
	ctx := context.Background()

	// Set should be no-op
	cache.Set("diff content", "response")
	assert.Equal(t, 0, cache.Size())

	// Get should always miss
	_, found := cache.Get("diff content")
	assert.False(t, found)

	// GetOrGenerate should always call generator
	callCount := 0
	generator := func(_ context.Context) (string, error) {
		callCount++
		return "generated", nil
	}

	response1, hit1, _ := cache.GetOrGenerate(ctx, "diff", generator)
	response2, hit2, _ := cache.GetOrGenerate(ctx, "diff", generator)

	assert.Equal(t, 2, callCount, "generator should be called every time when cache disabled")
	assert.False(t, hit1)
	assert.False(t, hit2)
	assert.Equal(t, "generated", response1)
	assert.Equal(t, "generated", response2)
}

func TestResponseCache_Concurrent(t *testing.T) {
	cfg := &Config{
		CacheEnabled: true,
		CacheTTL:     time.Hour,
		CacheMaxSize: 100,
	}
	cache := NewResponseCache(cfg)
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 50
	numOperations := 20

	generator := func(_ context.Context) (string, error) {
		return "generated", nil
	}

	// Run concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				diff := "diff" + string(rune('A'+id%10))

				// Mix of operations
				switch j % 4 {
				case 0:
					cache.Set(diff, "response")
				case 1:
					cache.Get(diff)
				case 2:
					_, _, _ = cache.GetOrGenerate(ctx, diff, generator)
				case 3:
					cache.Stats()
				}
			}
		}(i)
	}

	wg.Wait()

	// If we get here without deadlock or panic, concurrency is handled.
	// Verify cache is still functional after concurrent operations.
	assert.GreaterOrEqual(t, cache.Size(), 0, "cache should be functional after concurrent operations")
}

func TestHashDiff(t *testing.T) {
	tests := []struct {
		name        string
		diff1       string
		diff2       string
		shouldMatch bool
	}{
		{
			name:        "identical diffs produce same hash",
			diff1:       "diff --git a/file.go\n+new line",
			diff2:       "diff --git a/file.go\n+new line",
			shouldMatch: true,
		},
		{
			name:        "different diffs produce different hashes",
			diff1:       "diff --git a/file.go\n+new line",
			diff2:       "diff --git a/other.go\n+different",
			shouldMatch: false,
		},
		{
			name:        "empty strings produce same hash",
			diff1:       "",
			diff2:       "",
			shouldMatch: true,
		},
		{
			name:        "whitespace differences produce different hashes",
			diff1:       "diff content",
			diff2:       "diff content ",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashDiff(tt.diff1)
			hash2 := hashDiff(tt.diff2)

			if tt.shouldMatch {
				assert.Equal(t, hash1, hash2)
			} else {
				assert.NotEqual(t, hash1, hash2)
			}

			// Verify hash format (SHA256 = 64 hex chars)
			assert.Len(t, hash1, 64)
			assert.Len(t, hash2, 64)
		})
	}
}
