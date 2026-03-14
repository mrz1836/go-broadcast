package transform

import (
	"regexp"
	"sync"
)

// RegexCache provides thread-safe regex compilation and caching
type RegexCache struct {
	cache    map[string]*regexp.Regexp
	mu       sync.RWMutex
	initOnce sync.Once
	stats    CacheStats
	patterns []string
	maxSize  int
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	hits   int64
	misses int64
	size   int
	mu     sync.RWMutex
}

// NewRegexCache creates a new regex cache with default patterns
func NewRegexCache() *RegexCache {
	return &RegexCache{
		cache:   make(map[string]*regexp.Regexp),
		maxSize: 1000, // Reasonable cache size limit
		patterns: []string{
			// GitHub repository patterns
			`github\.com/([^/]+/[^/]+)`,
			`^[a-zA-Z0-9][\w.-]*/[a-zA-Z0-9][\w.-]*$`, // Repository validation

			// Branch patterns
			`^[a-zA-Z0-9][\w./\-]*$`,                              // Branch validation
			`^(chore/sync-files)-(\d{8})-(\d{6})-([a-fA-F0-9]+)$`, // Sync branch pattern

			// Template variable patterns
			`\{\{([A-Z_][A-Z0-9_]*)\}\}`, // {{VARIABLE}} format
			`\$\{([A-Z_][A-Z0-9_]*)\}`,   // ${VARIABLE} format

			// GitHub token patterns (for redaction)
			`ghp_[a-zA-Z0-9]{4,}`,         // GitHub personal tokens
			`ghs_[a-zA-Z0-9]{4,}`,         // GitHub app tokens
			`github_pat_[a-zA-Z0-9_]{4,}`, // New GitHub PAT format
			`ghr_[a-zA-Z0-9]{4,}`,         // GitHub refresh tokens

			// Authentication patterns
			`(Bearer|Token)\s+([^\s'"]+)`,                   // Bearer/Token headers
			`JWT\s+([a-zA-Z0-9_.-]{20,})`,                   // JWT tokens
			`(password|token|secret|key|api_key)=([^\s&]+)`, // URL parameters
			`://([^:]+):([^@]+)@`,                           // URL passwords

			// Security patterns
			`-----BEGIN[A-Z\s]+PRIVATE KEY-----[\s\S]*?-----END[A-Z\s]+PRIVATE KEY-----`, // SSH keys
			`\b([a-zA-Z0-9+/]{40,}={0,2})\b`,                                             // Base64 secrets
			`([A-Z_]*(?:TOKEN|SECRET|KEY|PASSWORD|PASS)[A-Z_]*=)([^\s]+)`,                // Environment variables
			`\b[a-zA-Z_]*token[a-zA-Z0-9_]*\b`,                                           // Generic tokens

			// File and content patterns
			`[^a-zA-Z0-9/_-]`, // Invalid characters for branch names
		},
	}
}

var (
	defaultCache *RegexCache //nolint:gochecknoglobals // Package-level singleton pattern
	cacheOnce    sync.Once   //nolint:gochecknoglobals // Package-level singleton pattern
)

// getDefaultCache returns the default regex cache, creating it if necessary
func getDefaultCache() *RegexCache {
	cacheOnce.Do(func() {
		defaultCache = NewRegexCache()
	})
	return defaultCache
}

// initCommonPatterns pre-compiles common patterns into the cache
func (rc *RegexCache) initCommonPatterns() {
	// Pre-compile common patterns
	for _, pattern := range rc.patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			rc.cache[pattern] = re
		}
		// Silently ignore compilation errors for invalid patterns
	}
}

// CompileRegex returns a compiled regex, using cache when possible.
//
// This function provides thread-safe access to a regex cache that eliminates
// repeated compilation overhead. It uses a double-checked locking pattern
// for optimal performance with concurrent access.
//
// Parameters:
// - pattern: Regular expression pattern string to compile
//
// Returns:
// - Compiled *regexp.Regexp instance
// - Error if pattern compilation fails
//
// Performance:
// - Fast path: Read-only cache lookup for cached patterns
// - Slow path: Compilation and caching for new patterns
// - Thread-safe: Uses RWMutex for concurrent access
func (rc *RegexCache) CompileRegex(pattern string) (*regexp.Regexp, error) {
	// Ensure common patterns are initialized
	rc.initOnce.Do(rc.initCommonPatterns)

	// Fast path: read from cache with read lock
	rc.mu.RLock()
	re, ok := rc.cache[pattern]
	rc.mu.RUnlock()

	if ok {
		// Update cache statistics atomically
		rc.stats.mu.Lock()
		rc.stats.hits++
		rc.stats.mu.Unlock()
		return re, nil
	}

	// Slow path: compile and cache with write lock
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have cached it)
	if cached, ok := rc.cache[pattern]; ok {
		// Update cache statistics
		rc.stats.mu.Lock()
		rc.stats.hits++
		rc.stats.mu.Unlock()
		return cached, nil
	}

	// Compile the pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Update cache statistics for compilation failure
		rc.stats.mu.Lock()
		rc.stats.misses++
		rc.stats.mu.Unlock()
		return nil, err
	}

	// Cache the compiled regex if within size limit
	if len(rc.cache) < rc.maxSize {
		rc.cache[pattern] = re
	}
	// When cache is full, pattern is compiled but not cached.
	// This is intentional to prevent unbounded memory growth.
	// Consider implementing LRU eviction for high-throughput scenarios.

	// Update cache statistics
	rc.stats.mu.Lock()
	rc.stats.misses++
	rc.stats.size = len(rc.cache)
	rc.stats.mu.Unlock()

	return re, nil
}

// CompileRegex returns a compiled regex using the default cache.
func CompileRegex(pattern string) (*regexp.Regexp, error) {
	return getDefaultCache().CompileRegex(pattern)
}

// MustCompileRegex compiles a regex pattern and panics if compilation fails.
//
// This is a convenience function for patterns that are known to be valid
// and should never fail compilation. It uses the same caching mechanism
// as CompileRegex for optimal performance.
//
// Parameters:
// - pattern: Regular expression pattern string to compile
//
// Returns:
// - Compiled *regexp.Regexp instance
//
// Panics:
// - If pattern compilation fails
func (rc *RegexCache) MustCompileRegex(pattern string) *regexp.Regexp {
	re, err := rc.CompileRegex(pattern)
	if err != nil {
		panic(err)
	}
	return re
}

// MustCompileRegex compiles a regex pattern using the default cache and panics if compilation fails.
func MustCompileRegex(pattern string) *regexp.Regexp {
	return getDefaultCache().MustCompileRegex(pattern)
}

// GetCacheStats returns current cache performance statistics.
//
// Returns:
// - hits: Number of successful cache lookups
// - misses: Number of cache misses requiring compilation
// - size: Current number of cached patterns
//
// Usage:
// This function is useful for monitoring cache effectiveness and
// tuning cache size limits or pre-compilation strategies.
func (rc *RegexCache) GetCacheStats() (hits, misses int64, size int) {
	rc.stats.mu.RLock()
	defer rc.stats.mu.RUnlock()
	return rc.stats.hits, rc.stats.misses, rc.stats.size
}

// GetCacheStats returns cache statistics from the default cache.
func GetCacheStats() (hits, misses int64, size int) {
	return getDefaultCache().GetCacheStats()
}

// ClearCache removes all cached regex patterns.
//
// This function is primarily useful for testing or when patterns
// need to be refreshed. It preserves the pre-compiled common patterns
// by re-initializing them after clearing the cache.
func (rc *RegexCache) ClearCache() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Clear the cache
	rc.cache = make(map[string]*regexp.Regexp)

	// Re-initialize common patterns
	for _, pattern := range rc.patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			rc.cache[pattern] = re
		}
	}

	// Reset statistics in a single lock acquisition
	rc.stats.mu.Lock()
	rc.stats.hits = 0
	rc.stats.misses = 0
	rc.stats.size = len(rc.cache)
	rc.stats.mu.Unlock()
}

// ClearCache clears the default regex cache.
func ClearCache() {
	getDefaultCache().ClearCache()
}

// PrecompilePatterns compiles and caches a list of patterns.
//
// This function is useful for warming the cache with application-specific
// patterns that are known to be used frequently.
//
// Parameters:
// - patterns: Slice of regex pattern strings to pre-compile
//
// Returns:
// - Number of patterns successfully compiled and cached
// - Slice of errors for patterns that failed to compile
func (rc *RegexCache) PrecompilePatterns(patterns []string) (int, []error) {
	var errors []error
	compiled := 0

	for _, pattern := range patterns {
		if _, err := rc.CompileRegex(pattern); err != nil {
			errors = append(errors, err)
		} else {
			compiled++
		}
	}

	return compiled, errors
}

// PrecompilePatterns pre-compiles patterns using the default cache.
func PrecompilePatterns(patterns []string) (int, []error) {
	return getDefaultCache().PrecompilePatterns(patterns)
}
