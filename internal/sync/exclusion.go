package sync

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// ExclusionEngine handles gitignore-style pattern matching with caching
type ExclusionEngine struct {
	patterns []exclusionPattern
	cache    sync.Map // map[string]bool for path -> excluded mapping
	mu       sync.RWMutex
}

// exclusionPattern represents a compiled exclusion pattern
type exclusionPattern struct {
	original string
	regex    *regexp.Regexp
	negate   bool
	isDir    bool
}

// NewExclusionEngine creates a new exclusion engine with default patterns
func NewExclusionEngine(patterns []string) *ExclusionEngine {
	engine := &ExclusionEngine{
		patterns: make([]exclusionPattern, 0, len(patterns)+len(defaultExclusions)),
	}

	// Add default exclusions first
	for _, pattern := range defaultExclusions {
		engine.addPattern(pattern)
	}

	// Add user-specified patterns
	for _, pattern := range patterns {
		engine.addPattern(pattern)
	}

	return engine
}

// defaultExclusions contains common patterns that should typically be excluded
var defaultExclusions = []string{
	".git/",
	".git/**",
	"**/.git/",
	"**/.git/**",
	"node_modules/",
	"node_modules/**",
	"**/node_modules/",
	"**/node_modules/**",
	".DS_Store",
	"**/.DS_Store",
	"Thumbs.db",
	"**/Thumbs.db",
	"*.tmp",
	"*.temp",
	"**/*.tmp",
	"**/*.temp",
	".env",
	".env.*",
	"**/.env",
	"**/.env.*",
}

// IsExcluded checks if a file path should be excluded based on the configured patterns
func (e *ExclusionEngine) IsExcluded(filePath string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(filePath)

	// Check cache first
	if cached, found := e.cache.Load(normalizedPath); found {
		return cached.(bool)
	}

	// Evaluate patterns
	excluded := e.evaluatePatterns(normalizedPath)

	// Cache the result
	e.cache.Store(normalizedPath, excluded)

	return excluded
}

// IsDirectoryExcluded checks if a directory should be excluded
// This is optimized for directory traversal to avoid walking excluded directories
func (e *ExclusionEngine) IsDirectoryExcluded(dirPath string) bool {
	// Normalize and ensure trailing slash for directory matching
	normalizedPath := filepath.ToSlash(dirPath)
	if !strings.HasSuffix(normalizedPath, "/") {
		normalizedPath += "/"
	}

	// Check cache first
	cacheKey := normalizedPath + "__DIR__"
	if cached, found := e.cache.Load(cacheKey); found {
		return cached.(bool)
	}

	// Evaluate patterns specifically for directories
	excluded := e.evaluateDirectoryPatterns(normalizedPath)

	// Cache the result
	e.cache.Store(cacheKey, excluded)

	return excluded
}

// ClearCache clears the pattern matching cache
func (e *ExclusionEngine) ClearCache() {
	e.cache.Range(func(key, value interface{}) bool {
		e.cache.Delete(key)
		return true
	})
}

// GetPatterns returns the original patterns for debugging
func (e *ExclusionEngine) GetPatterns() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	patterns := make([]string, len(e.patterns))
	for i, pattern := range e.patterns {
		patterns[i] = pattern.original
	}
	return patterns
}

// addPattern compiles and adds a pattern to the engine
func (e *ExclusionEngine) addPattern(pattern string) {
	if pattern == "" {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	compiled := e.compilePattern(pattern)
	if compiled.regex != nil {
		e.patterns = append(e.patterns, compiled)
	}
}

// compilePattern converts a gitignore-style pattern into a compiled regex pattern
func (e *ExclusionEngine) compilePattern(pattern string) exclusionPattern {
	original := pattern
	negate := false
	isDir := false

	// Handle negation
	if strings.HasPrefix(pattern, "!") {
		negate = true
		pattern = pattern[1:]
	}

	// Handle directory-only patterns
	if strings.HasSuffix(pattern, "/") {
		isDir = true
		pattern = strings.TrimSuffix(pattern, "/")
	}

	// Escape regex special characters except * and ?
	pattern = regexp.QuoteMeta(pattern)

	// Convert gitignore wildcards to regex
	pattern = strings.ReplaceAll(pattern, `\*\*`, `.*`)  // ** matches any number of directories
	pattern = strings.ReplaceAll(pattern, `\*`, `[^/]*`) // * matches any character except /
	pattern = strings.ReplaceAll(pattern, `\?`, `.`)     // ? matches any single character

	// Handle different pattern types
	var regexPattern string
	if strings.HasPrefix(original, "/") {
		// Absolute pattern from root
		regexPattern = "^" + strings.TrimPrefix(pattern, `/`) + "$"
	} else if strings.Contains(original, "/") {
		// Pattern contains slash - match from any directory level
		regexPattern = "(^|.*/)(" + pattern + ")($|/.*)"
	} else {
		// Simple filename pattern - match basename
		regexPattern = "(^|.*/)(" + pattern + ")($|/.*)"
	}

	// Compile regex
	regex, err := regexp.Compile(regexPattern)
	if err != nil {
		// If compilation fails, create a simple literal match
		literalPattern := "(^|.*/)(" + regexp.QuoteMeta(original) + ")($|/.*)"
		regex, _ = regexp.Compile(literalPattern)
	}

	return exclusionPattern{
		original: original,
		regex:    regex,
		negate:   negate,
		isDir:    isDir,
	}
}

// evaluatePatterns evaluates all patterns against a file path
func (e *ExclusionEngine) evaluatePatterns(path string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	excluded := false

	// Process patterns in order
	for _, pattern := range e.patterns {
		if pattern.regex == nil {
			continue
		}

		// For directory-only patterns, skip if this is not a directory path
		if pattern.isDir && !strings.HasSuffix(path, "/") {
			continue
		}

		if pattern.regex.MatchString(path) {
			if pattern.negate {
				excluded = false // Negation overrides previous exclusions
			} else {
				excluded = true
			}
		}
	}

	return excluded
}

// evaluateDirectoryPatterns evaluates patterns specifically for directories
func (e *ExclusionEngine) evaluateDirectoryPatterns(dirPath string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	excluded := false

	// Process patterns in order
	for _, pattern := range e.patterns {
		if pattern.regex == nil {
			continue
		}

		// Check both with and without trailing slash
		pathToCheck := strings.TrimSuffix(dirPath, "/")
		matches := pattern.regex.MatchString(pathToCheck) || pattern.regex.MatchString(dirPath)

		if matches {
			if pattern.negate {
				excluded = false // Negation overrides previous exclusions
			} else {
				excluded = true
			}
		}
	}

	return excluded
}

// AddPatterns adds additional exclusion patterns at runtime
func (e *ExclusionEngine) AddPatterns(patterns []string) {
	for _, pattern := range patterns {
		e.addPattern(pattern)
	}
	// Clear cache since patterns have changed
	e.ClearCache()
}

// RemovePatterns removes patterns by their original string (useful for dynamic management)
func (e *ExclusionEngine) RemovePatterns(patternsToRemove []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	removeSet := make(map[string]bool)
	for _, pattern := range patternsToRemove {
		removeSet[pattern] = true
	}

	filtered := make([]exclusionPattern, 0, len(e.patterns))
	for _, pattern := range e.patterns {
		if !removeSet[pattern.original] {
			filtered = append(filtered, pattern)
		}
	}

	e.patterns = filtered
	e.ClearCache()
}
