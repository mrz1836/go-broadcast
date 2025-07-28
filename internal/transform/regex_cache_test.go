package transform

import (
	"fmt"
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegexCache(t *testing.T) {
	cache := NewRegexCache()

	require.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
	assert.Equal(t, 1000, cache.maxSize)
	assert.NotEmpty(t, cache.patterns)

	// Verify some expected patterns exist
	expectedPatterns := []string{
		`github\.com/([^/]+/[^/]+)`,
		`^[a-zA-Z0-9][\w.-]*/[a-zA-Z0-9][\w.-]*$`,
		`\{\{([A-Z_][A-Z0-9_]*)\}\}`,
		`ghp_[a-zA-Z0-9]{4,}`,
	}

	for _, expected := range expectedPatterns {
		assert.Contains(t, cache.patterns, expected)
	}
}

func TestRegexCache_CompileRegex(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		wantError   bool
		expectedErr string
	}{
		{
			name:      "valid simple pattern",
			pattern:   `\d+`,
			wantError: false,
		},
		{
			name:      "valid complex pattern",
			pattern:   `^[a-zA-Z0-9][\w.-]*$`,
			wantError: false,
		},
		{
			name:        "invalid pattern - unclosed bracket",
			pattern:     `[a-z`,
			wantError:   true,
			expectedErr: "missing closing ]",
		},
		{
			name:        "invalid pattern - bad syntax",
			pattern:     `*invalid`,
			wantError:   true,
			expectedErr: "missing argument to repetition operator",
		},
		{
			name:      "empty pattern",
			pattern:   "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewRegexCache()

			re, err := cache.CompileRegex(tt.pattern)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, re)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, re)

				// Verify the regex works
				compiled, compileErr := regexp.Compile(tt.pattern)
				require.NoError(t, compileErr)
				assert.Equal(t, compiled.String(), re.String())
			}
		})
	}
}

func TestRegexCache_CompileRegexCaching(t *testing.T) {
	cache := NewRegexCache()
	pattern := `\d+`

	// First call should be a cache miss
	re1, err := cache.CompileRegex(pattern)
	require.NoError(t, err)
	require.NotNil(t, re1)

	hits, misses, size := cache.GetCacheStats()
	assert.Equal(t, int64(0), hits) // First compilation is pre-init
	assert.Positive(t, misses)
	assert.Positive(t, size)

	// Second call should be a cache hit
	re2, err := cache.CompileRegex(pattern)
	require.NoError(t, err)
	require.NotNil(t, re2)

	// Should be the same instance
	assert.Same(t, re1, re2)

	hits2, misses2, size2 := cache.GetCacheStats()
	assert.Greater(t, hits2, hits)
	assert.Equal(t, misses2, misses) // No new misses
	assert.Equal(t, size2, size)
}

func TestRegexCache_CompileRegexConcurrency(t *testing.T) {
	cache := NewRegexCache()
	pattern := `concurrent-\d+`

	const numGoroutines = 100
	var wg sync.WaitGroup
	results := make([]*regexp.Regexp, numGoroutines)
	errors := make([]error, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			re, err := cache.CompileRegex(pattern)
			results[idx] = re
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// All should succeed
	for i := 0; i < numGoroutines; i++ {
		require.NoError(t, errors[i])
		require.NotNil(t, results[i])
	}

	// All should be the same instance (cached)
	firstResult := results[0]
	for i := 1; i < numGoroutines; i++ {
		assert.Same(t, firstResult, results[i])
	}

	// Verify stats make sense
	hits, misses, size := cache.GetCacheStats()
	assert.Positive(t, hits)
	assert.Positive(t, misses)
	assert.Positive(t, size)
}

func TestRegexCache_MustCompileRegex(t *testing.T) {
	cache := NewRegexCache()

	t.Run("valid pattern succeeds", func(t *testing.T) {
		pattern := `valid-\d+`

		require.NotPanics(t, func() {
			re := cache.MustCompileRegex(pattern)
			assert.NotNil(t, re)
		})
	})

	t.Run("invalid pattern panics", func(t *testing.T) {
		pattern := `[invalid`

		require.Panics(t, func() {
			cache.MustCompileRegex(pattern)
		})
	})
}

func TestRegexCache_GetCacheStats(t *testing.T) {
	cache := NewRegexCache()

	// Initial stats
	hits, misses, size := cache.GetCacheStats()
	assert.Equal(t, int64(0), hits)
	assert.Equal(t, int64(0), misses)
	assert.Equal(t, 0, size)

	// Compile a new pattern
	_, err := cache.CompileRegex(`test-\d+`)
	require.NoError(t, err)

	hits2, misses2, size2 := cache.GetCacheStats()
	assert.Equal(t, int64(0), hits2) // Still no hits
	assert.Greater(t, misses2, misses)
	assert.Greater(t, size2, size)

	// Second compilation should increment hits
	_, err = cache.CompileRegex(`test-\d+`)
	require.NoError(t, err)

	hits3, misses3, size3 := cache.GetCacheStats()
	assert.Greater(t, hits3, hits2)
	assert.Equal(t, misses3, misses2) // No new misses
	assert.Equal(t, size3, size2)
}

func TestRegexCache_ClearCache(t *testing.T) {
	cache := NewRegexCache()

	// Add some patterns to cache
	patterns := []string{`test-\d+`, `another-[a-z]+`, `third-\w*`}
	for _, pattern := range patterns {
		_, err := cache.CompileRegex(pattern)
		require.NoError(t, err)
	}

	// Verify cache has content
	_, _, size := cache.GetCacheStats()
	assert.Positive(t, size)

	// Clear cache
	cache.ClearCache()

	// Verify cache was cleared and reinitialized
	hits2, misses2, size2 := cache.GetCacheStats()
	assert.Equal(t, int64(0), hits2)
	assert.Equal(t, int64(0), misses2)

	// Size should be > 0 due to pre-compiled common patterns
	assert.Positive(t, size2)

	// But should be different from before (common patterns only)
	if size > len(cache.patterns) {
		assert.Less(t, size2, size)
	}
}

func TestRegexCache_PrecompilePatterns(t *testing.T) {
	cache := NewRegexCache()

	tests := []struct {
		name             string
		patterns         []string
		expectedCompiled int
		expectedErrors   int
	}{
		{
			name:             "all valid patterns",
			patterns:         []string{`\d+`, `[a-z]+`, `\w*`},
			expectedCompiled: 3,
			expectedErrors:   0,
		},
		{
			name:             "mixed valid and invalid patterns",
			patterns:         []string{`\d+`, `[invalid`, `[a-z]+`, `*bad`},
			expectedCompiled: 2,
			expectedErrors:   2,
		},
		{
			name:             "all invalid patterns",
			patterns:         []string{`[invalid`, `*bad`, `+worse`},
			expectedCompiled: 0,
			expectedErrors:   3,
		},
		{
			name:             "empty patterns list",
			patterns:         []string{},
			expectedCompiled: 0,
			expectedErrors:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear cache to start fresh
			cache.ClearCache()

			compiled, errors := cache.PrecompilePatterns(tt.patterns)

			assert.Equal(t, tt.expectedCompiled, compiled)
			assert.Len(t, errors, tt.expectedErrors)

			// Verify compiled patterns are now cached
			for _, pattern := range tt.patterns {
				re, err := cache.CompileRegex(pattern)
				if err == nil {
					assert.NotNil(t, re)
				}
			}
		})
	}
}

func TestRegexCache_MaxSizeLimit(t *testing.T) {
	cache := NewRegexCache()

	// Get initial size after common patterns are loaded
	_, err := cache.CompileRegex(`trigger-init`)
	require.NoError(t, err)
	_, _, initialSize := cache.GetCacheStats()

	// Set max size to current size + 3 more patterns
	cache.maxSize = initialSize + 3

	// Try to add 10 more patterns beyond the limit
	for i := 0; i < 10; i++ {
		pattern := fmt.Sprintf(`test-%d-\d+`, i)
		_, err := cache.CompileRegex(pattern)
		require.NoError(t, err)
	}

	_, _, finalSize := cache.GetCacheStats()
	// Final size should not exceed our max size
	assert.LessOrEqual(t, finalSize, cache.maxSize)
	// But should be larger than initial size (some patterns were added)
	assert.Greater(t, finalSize, initialSize)
}

func TestRegexCache_CommonPatternsInitialization(t *testing.T) {
	cache := NewRegexCache()

	// Force initialization
	_, err := cache.CompileRegex(`force-init`)
	require.NoError(t, err)

	// Verify common patterns are cached
	commonPatterns := []string{
		`github\.com/([^/]+/[^/]+)`,
		`\{\{([A-Z_][A-Z0-9_]*)\}\}`,
		`ghp_[a-zA-Z0-9]{4,}`,
	}

	for _, pattern := range commonPatterns {
		// These should be hits since they're pre-compiled
		re, err := cache.CompileRegex(pattern)
		require.NoError(t, err)
		require.NotNil(t, re)
	}

	hits, _, _ := cache.GetCacheStats()
	assert.Positive(t, hits)
}

// Test package-level functions
func TestPackageLevelFunctions(t *testing.T) {
	t.Run("CompileRegex", func(t *testing.T) {
		re, err := CompileRegex(`package-\d+`)
		require.NoError(t, err)
		require.NotNil(t, re)
	})

	t.Run("MustCompileRegex valid", func(t *testing.T) {
		require.NotPanics(t, func() {
			re := MustCompileRegex(`package-valid`)
			assert.NotNil(t, re)
		})
	})

	t.Run("MustCompileRegex invalid", func(t *testing.T) {
		require.Panics(t, func() {
			MustCompileRegex(`[invalid`)
		})
	})

	t.Run("GetCacheStats", func(t *testing.T) {
		hits, misses, size := GetCacheStats()
		assert.GreaterOrEqual(t, hits, int64(0))
		assert.GreaterOrEqual(t, misses, int64(0))
		assert.GreaterOrEqual(t, size, 0)
	})

	t.Run("ClearCache", func(t *testing.T) {
		// Add a pattern first
		_, err := CompileRegex(`test-clear`)
		require.NoError(t, err)

		// Clear and verify
		ClearCache()
		hits, misses, size := GetCacheStats()
		assert.Equal(t, int64(0), hits)
		assert.Equal(t, int64(0), misses)
		assert.Positive(t, size) // Common patterns remain
	})

	t.Run("PrecompilePatterns", func(t *testing.T) {
		patterns := []string{`pkg-\d+`, `pkg-[a-z]+`}
		compiled, errors := PrecompilePatterns(patterns)
		assert.Equal(t, 2, compiled)
		assert.Empty(t, errors)
	})
}

func TestRegexCache_DefaultCacheSingleton(t *testing.T) {
	// Get default cache multiple times
	cache1 := getDefaultCache()
	cache2 := getDefaultCache()

	// Should be the same instance
	assert.Same(t, cache1, cache2)

	// Should be properly initialized
	assert.NotNil(t, cache1.cache)
	assert.Positive(t, cache1.maxSize)
	assert.NotEmpty(t, cache1.patterns)
}
