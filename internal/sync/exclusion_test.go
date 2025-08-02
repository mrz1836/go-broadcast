package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExclusionEngine tests the exclusion engine functionality
func TestExclusionEngine(t *testing.T) {
	patterns := []string{
		"*.log",
		"temp/**",
		"node_modules/",
		"!important.log",
		".git/**",
	}

	engine := NewExclusionEngine(patterns)
	require.NotNil(t, engine)

	// Test file exclusions
	testCases := []struct {
		path     string
		excluded bool
		desc     string
	}{
		{"test.log", true, "should exclude .log files"},
		{"important.log", false, "should not exclude negated patterns"},
		{"temp/file.txt", true, "should exclude files in temp directory"},
		{"node_modules/package.json", true, "should exclude node_modules"},
		{"src/main.go", false, "should not exclude normal source files"},
		{".git/config", true, "should exclude .git directory"},
		{"docs/readme.md", false, "should not exclude documentation"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := engine.IsExcluded(tc.path)
			assert.Equal(t, tc.excluded, result, "%s: %s", tc.desc, tc.path)
		})
	}
}

// TestExclusionEngineDirectories tests directory-specific exclusion
func TestExclusionEngineDirectories(t *testing.T) {
	patterns := []string{
		"vendor/",
		"node_modules/",
		".git/",
	}

	engine := NewExclusionEngine(patterns)
	require.NotNil(t, engine)

	// Test directory exclusions
	testCases := []struct {
		path     string
		excluded bool
		desc     string
	}{
		{"vendor", true, "should exclude vendor directory"},
		{"vendor/", true, "should exclude vendor directory with slash"},
		{"node_modules", true, "should exclude node_modules directory"},
		{"src", false, "should not exclude src directory"},
		{".git", true, "should exclude .git directory"},
		{"docs", false, "should not exclude docs directory"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := engine.IsDirectoryExcluded(tc.path)
			assert.Equal(t, tc.excluded, result, "%s: %s", tc.desc, tc.path)
		})
	}
}

// TestExclusionEngineCache tests cache functionality
func TestExclusionEngineCache(t *testing.T) {
	patterns := []string{"*.log", "temp/**"}
	engine := NewExclusionEngine(patterns)

	// Test the same path multiple times to verify caching
	path := "test.log"
	result1 := engine.IsExcluded(path)
	result2 := engine.IsExcluded(path)
	result3 := engine.IsExcluded(path)

	assert.True(t, result1, "First call should exclude .log file")
	assert.Equal(t, result1, result2, "Cached result should be the same")
	assert.Equal(t, result1, result3, "Cached result should be the same")

	// Clear cache and test again
	engine.ClearCache()
	result4 := engine.IsExcluded(path)
	assert.Equal(t, result1, result4, "Result after cache clear should be the same")
}

// TestExclusionEnginePatterns tests pattern management
func TestExclusionEnginePatterns(t *testing.T) {
	patterns := []string{"*.log", "temp/**"}
	engine := NewExclusionEngine(patterns)

	// Get patterns
	retrievedPatterns := engine.GetPatterns()

	// Should include default patterns plus user patterns
	assert.Greater(t, len(retrievedPatterns), len(patterns), "Should include default patterns")

	// Check that user patterns are included
	foundLog := false
	foundTemp := false
	for _, pattern := range retrievedPatterns {
		if pattern == "*.log" {
			foundLog = true
		}
		if pattern == "temp/**" {
			foundTemp = true
		}
	}
	assert.True(t, foundLog, "Should include user *.log pattern")
	assert.True(t, foundTemp, "Should include user temp/** pattern")
}

// TestExclusionEngineAddRemovePatterns tests dynamic pattern management
func TestExclusionEngineAddRemovePatterns(t *testing.T) {
	engine := NewExclusionEngine([]string{"*.log"})

	// Test that a file is excluded
	assert.True(t, engine.IsExcluded("test.log"), "Should exclude .log file")
	assert.False(t, engine.IsExcluded("test.txt"), "Should not exclude .txt file")

	// Add new pattern
	engine.AddPatterns([]string{"*.txt"})
	assert.True(t, engine.IsExcluded("test.txt"), "Should exclude .txt file after adding pattern")

	// Remove pattern
	engine.RemovePatterns([]string{"*.txt"})
	assert.False(t, engine.IsExcluded("test.txt"), "Should not exclude .txt file after removing pattern")
	assert.True(t, engine.IsExcluded("test.log"), "Should still exclude .log file")
}

// TestExclusionEngineDefaultPatterns tests that default exclusions work
func TestExclusionEngineDefaultPatterns(t *testing.T) {
	engine := NewExclusionEngine([]string{})

	// Test default exclusions
	testCases := []struct {
		path     string
		excluded bool
		desc     string
	}{
		{".git/config", true, "should exclude .git directory"},
		{"node_modules/package.json", true, "should exclude node_modules"},
		{".DS_Store", true, "should exclude .DS_Store"},
		{"file.tmp", true, "should exclude .tmp files"},
		{".env", true, "should exclude .env files"},
		{"src/main.go", false, "should not exclude source files"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := engine.IsExcluded(tc.path)
			assert.Equal(t, tc.excluded, result, "%s: %s", tc.desc, tc.path)
		})
	}
}
