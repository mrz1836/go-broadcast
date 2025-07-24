package fuzz

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

		err := gen.GenerateAll()
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
