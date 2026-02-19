package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExclusionEngine_AddIncludePattern(t *testing.T) {
	t.Parallel()

	t.Run("adds valid include pattern", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes(nil, nil)
		engine.addIncludePattern("*.go")
		assert.Len(t, engine.includePatterns, 1)
	})

	t.Run("ignores empty pattern", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes(nil, nil)
		engine.addIncludePattern("")
		assert.Empty(t, engine.includePatterns)
	})

	t.Run("adds multiple patterns", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes(nil, nil)
		engine.addIncludePattern("*.go")
		engine.addIncludePattern("*.md")
		assert.Len(t, engine.includePatterns, 2)
	})
}

func TestExclusionEngine_EvaluateIncludePatterns(t *testing.T) {
	t.Parallel()

	t.Run("included file is not excluded", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes(nil, []string{"*.go"})
		excluded := engine.evaluateIncludePatterns("main.go")
		assert.False(t, excluded)
	})

	t.Run("non-included file is excluded", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes(nil, []string{"*.go"})
		excluded := engine.evaluateIncludePatterns("readme.md")
		assert.True(t, excluded)
	})

	t.Run("included but also explicitly excluded", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes([]string{"vendor/**"}, []string{"*.go"})
		excluded := engine.evaluateIncludePatterns("vendor/lib.go")
		assert.True(t, excluded)
	})

	t.Run("directory pattern skips non-directory path", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes(nil, []string{"src/"})
		// "main.go" doesn't end with "/" so dir-only include pattern doesn't match
		excluded := engine.evaluateIncludePatterns("main.go")
		assert.True(t, excluded)
	})
}

func TestExclusionEngine_EvaluateDirectoryIncludePatterns(t *testing.T) {
	t.Parallel()

	t.Run("included directory is not excluded", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes(nil, []string{"src/**"})
		excluded := engine.evaluateDirectoryIncludePatterns("src/")
		assert.False(t, excluded)
	})

	t.Run("non-included directory is excluded", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes(nil, []string{"src/**"})
		excluded := engine.evaluateDirectoryIncludePatterns("vendor/")
		assert.True(t, excluded)
	})

	t.Run("included but explicitly excluded directory", func(t *testing.T) {
		t.Parallel()

		engine := NewExclusionEngineWithIncludes([]string{"src/test/"}, []string{"src/**"})
		excluded := engine.evaluateDirectoryIncludePatterns("src/test/")
		assert.True(t, excluded)
	})
}
