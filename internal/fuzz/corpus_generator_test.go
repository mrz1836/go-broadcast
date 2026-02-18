package fuzz

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateDeeplyNested tests the depth-bounded YAML generator
func TestGenerateDeeplyNested(t *testing.T) {
	t.Run("ZeroDepth", func(t *testing.T) {
		result := generateDeeplyNested(0)
		assert.Contains(t, result, "version: 1")
		assert.NotContains(t, result, "nested/repo")
	})

	t.Run("NegativeDepth", func(t *testing.T) {
		result := generateDeeplyNested(-5)
		assert.Contains(t, result, "version: 1")
		assert.NotContains(t, result, "nested/repo")
	})

	t.Run("NormalDepth", func(t *testing.T) {
		result := generateDeeplyNested(3)
		assert.Contains(t, result, "nested/repo0")
		assert.Contains(t, result, "nested/repo1")
		assert.Contains(t, result, "nested/repo2")
	})

	t.Run("ExcessiveDepthIsBounded", func(t *testing.T) {
		// Should not panic or hang with very large depth
		result := generateDeeplyNested(100000)
		assert.Contains(t, result, "version: 1")
		// Should be bounded to maxDepth (1000)
		assert.Contains(t, result, "nested/repo999")
	})
}

// TestGenerateDeeplyNestedJSON tests the depth-bounded JSON generator
func TestGenerateDeeplyNestedJSON(t *testing.T) {
	t.Run("ZeroDepth", func(t *testing.T) {
		result := generateDeeplyNestedJSON(0)
		assert.Equal(t, `"bottom"`, result)
	})

	t.Run("NegativeDepth", func(t *testing.T) {
		result := generateDeeplyNestedJSON(-5)
		assert.Equal(t, `"bottom"`, result)
	})

	t.Run("NormalDepth", func(t *testing.T) {
		result := generateDeeplyNestedJSON(2)
		assert.JSONEq(t, `{"level":{"level":"bottom"}}`, result)
	})

	t.Run("ExcessiveDepthIsBounded", func(t *testing.T) {
		// Should not panic or hang with very large depth
		result := generateDeeplyNestedJSON(100000)
		assert.NotEmpty(t, result)
		// Verify structure is valid (has matching braces)
		openBraces := 0
		for _, c := range result {
			switch c {
			case '{':
				openBraces++
			case '}':
				openBraces--
			}
		}
		assert.Equal(t, 0, openBraces, "JSON braces should be balanced")
	})
}

// TestCorpusGeneratorErrorCases tests error handling
func TestCorpusGeneratorErrorCases(t *testing.T) {
	t.Run("InvalidBaseDirectory", func(t *testing.T) {
		// Use a path that's a file, not a directory
		tmpFile, err := os.CreateTemp("", "fuzz-test-file")
		require.NoError(t, err)
		defer func() {
			_ = os.Remove(tmpFile.Name()) //nolint:gosec // G703: path from os.CreateTemp or trusted source, not user input
		}()
		_ = tmpFile.Close()

		gen := NewCorpusGenerator(tmpFile.Name())
		err = gen.GenerateConfigCorpus()
		assert.Error(t, err, "Should fail when base is a file")
	})

	t.Run("ReadOnlyDirectory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tmpDir, err := os.MkdirTemp("", "fuzz-readonly-test")
		require.NoError(t, err)
		defer func() {
			_ = os.Chmod(tmpDir, 0o700) //nolint:gosec // Restoring permissions for cleanup
			_ = os.RemoveAll(tmpDir)
		}()

		// Make directory read-only
		err = os.Chmod(tmpDir, 0o000)
		require.NoError(t, err)

		gen := NewCorpusGenerator(tmpDir)
		err = gen.GenerateConfigCorpus()
		assert.Error(t, err, "Should fail when directory is read-only")
	})
}

func TestCorpusGenerator(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "fuzz-corpus-test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	gen := NewCorpusGenerator(tmpDir)

	// Test individual generators
	t.Run("GenerateConfigCorpus", func(t *testing.T) {
		err2 := gen.GenerateConfigCorpus()
		require.NoError(t, err2)

		// Check files were created
		files, err2 := os.ReadDir(filepath.Join(tmpDir, "corpus", "config"))
		require.NoError(t, err2)
		assert.Greater(t, len(files), 20) // Should have many seed files
	})

	t.Run("GenerateGitCorpus", func(t *testing.T) {
		err2 := gen.GenerateGitCorpus()
		require.NoError(t, err2)

		files, err2 := os.ReadDir(filepath.Join(tmpDir, "corpus", "git"))
		require.NoError(t, err2)
		assert.Greater(t, len(files), 20)
	})

	t.Run("GenerateGHCorpus", func(t *testing.T) {
		err2 := gen.GenerateGHCorpus()
		require.NoError(t, err2)

		files, err2 := os.ReadDir(filepath.Join(tmpDir, "corpus", "gh"))
		require.NoError(t, err2)
		assert.Greater(t, len(files), 15)
	})

	t.Run("GenerateTransformCorpus", func(t *testing.T) {
		err2 := gen.GenerateTransformCorpus()
		require.NoError(t, err2)

		files, err2 := os.ReadDir(filepath.Join(tmpDir, "corpus", "transform"))
		require.NoError(t, err2)
		assert.Greater(t, len(files), 20)
	})

	t.Run("GenerateAll", func(t *testing.T) {
		// Clean up and start fresh
		_ = os.RemoveAll(tmpDir)
		err = os.MkdirAll(tmpDir, 0o750)
		require.NoError(t, err)

		err = gen.GenerateAll() // use = not := to avoid variable shadowing
		require.NoError(t, err)

		// Check all subdirectories have files
		categories := []string{"config", "git", "gh", "transform"}
		for _, cat := range categories {
			files, err := os.ReadDir(filepath.Join(tmpDir, "corpus", cat))
			require.NoError(t, err)
			assert.Greater(t, len(files), 10, "Category %s should have corpus files", cat)
		}
	})
}
