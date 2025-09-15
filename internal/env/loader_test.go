package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvFiles(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		chdirErr := os.Chdir(originalWd)
		require.NoError(t, chdirErr)
	}()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "go-broadcast-env-test")
	require.NoError(t, err)
	defer func() {
		removeErr := os.RemoveAll(tempDir)
		require.NoError(t, removeErr)
	}()

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create .github directory
	githubDir := filepath.Join(tempDir, ".github")
	err = os.MkdirAll(githubDir, 0o750)
	require.NoError(t, err)

	t.Run("loads base file only", func(t *testing.T) {
		// Create base file
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `TEST_VAR=base_value
GO_BROADCAST_AUTOMERGE_LABELS=automerge
ANOTHER_VAR=base_another`
		err := os.WriteFile(baseFile, []byte(baseContent), 0o600)
		require.NoError(t, err)

		// Clear any existing env vars
		origTestVar := os.Getenv("TEST_VAR")
		origAutomergeLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS")
		origAnotherVar := os.Getenv("ANOTHER_VAR")
		defer func() {
			if origTestVar != "" {
				_ = os.Setenv("TEST_VAR", origTestVar)
			} else {
				_ = os.Unsetenv("TEST_VAR")
			}
			if origAutomergeLabels != "" {
				_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", origAutomergeLabels)
			} else {
				_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")
			}
			if origAnotherVar != "" {
				_ = os.Setenv("ANOTHER_VAR", origAnotherVar)
			} else {
				_ = os.Unsetenv("ANOTHER_VAR")
			}
		}()
		_ = os.Unsetenv("TEST_VAR")
		_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")
		_ = os.Unsetenv("ANOTHER_VAR")

		// Load env files
		err = LoadEnvFiles()
		require.NoError(t, err)

		// Check that base values are loaded
		assert.Equal(t, "base_value", os.Getenv("TEST_VAR"))
		assert.Equal(t, "automerge", os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"))
		assert.Equal(t, "base_another", os.Getenv("ANOTHER_VAR"))

		// Clean up
		err = os.Remove(baseFile)
		require.NoError(t, err)
	})

	t.Run("loads base and custom with custom override", func(t *testing.T) {
		// Create base file
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `TEST_VAR=base_value
GO_BROADCAST_AUTOMERGE_LABELS=automerge
ANOTHER_VAR=base_another`
		err := os.WriteFile(baseFile, []byte(baseContent), 0o600)
		require.NoError(t, err)

		// Create custom file
		customFile := filepath.Join(githubDir, ".env.custom")
		customContent := `TEST_VAR=custom_value
GO_BROADCAST_AUTOMERGE_LABELS=ready-to-merge,auto-merge`
		err = os.WriteFile(customFile, []byte(customContent), 0o600)
		require.NoError(t, err)

		// Clear any existing env vars
		origTestVar := os.Getenv("TEST_VAR")
		origAutomergeLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS")
		origAnotherVar := os.Getenv("ANOTHER_VAR")
		defer func() {
			if origTestVar != "" {
				_ = os.Setenv("TEST_VAR", origTestVar)
			} else {
				_ = os.Unsetenv("TEST_VAR")
			}
			if origAutomergeLabels != "" {
				_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", origAutomergeLabels)
			} else {
				_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")
			}
			if origAnotherVar != "" {
				_ = os.Setenv("ANOTHER_VAR", origAnotherVar)
			} else {
				_ = os.Unsetenv("ANOTHER_VAR")
			}
		}()
		_ = os.Unsetenv("TEST_VAR")
		_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")
		_ = os.Unsetenv("ANOTHER_VAR")

		// Load env files
		err = LoadEnvFiles()
		require.NoError(t, err)

		// Check that custom values override base values
		assert.Equal(t, "custom_value", os.Getenv("TEST_VAR"))
		assert.Equal(t, "ready-to-merge,auto-merge", os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"))
		// ANOTHER_VAR should still be from base since it's not in custom
		assert.Equal(t, "base_another", os.Getenv("ANOTHER_VAR"))

		// Clean up
		err = os.Remove(baseFile)
		require.NoError(t, err)
		err = os.Remove(customFile)
		require.NoError(t, err)
	})

	t.Run("fails when base file missing", func(t *testing.T) {
		// Ensure no base file exists
		baseFile := filepath.Join(githubDir, ".env.base")
		_ = os.Remove(baseFile)

		// Attempt to load env files
		err := LoadEnvFiles()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required .env.base file not found")
	})

	t.Run("works when custom file missing", func(t *testing.T) {
		// Create base file
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `TEST_VAR=base_value`
		err := os.WriteFile(baseFile, []byte(baseContent), 0o600)
		require.NoError(t, err)

		// Ensure no custom file exists
		customFile := filepath.Join(githubDir, ".env.custom")
		_ = os.Remove(customFile)

		// Clear env var
		origTestVar := os.Getenv("TEST_VAR")
		defer func() {
			if origTestVar != "" {
				_ = os.Setenv("TEST_VAR", origTestVar)
			} else {
				_ = os.Unsetenv("TEST_VAR")
			}
		}()
		_ = os.Unsetenv("TEST_VAR")

		// Load env files
		err = LoadEnvFiles()
		require.NoError(t, err)

		// Check that base value is loaded
		assert.Equal(t, "base_value", os.Getenv("TEST_VAR"))

		// Clean up
		err = os.Remove(baseFile)
		require.NoError(t, err)
	})
}

func TestLoadEnvFilesFromDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "go-broadcast-env-dir-test")
	require.NoError(t, err)
	defer func() {
		removeErr := os.RemoveAll(tempDir)
		require.NoError(t, removeErr)
	}()

	// Create .github directory
	githubDir := filepath.Join(tempDir, ".github")
	err = os.MkdirAll(githubDir, 0o750)
	require.NoError(t, err)

	t.Run("loads from specified directory", func(t *testing.T) {
		// Create base file
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `TEST_DIR_VAR=dir_value`
		err := os.WriteFile(baseFile, []byte(baseContent), 0o600)
		require.NoError(t, err)

		// Clear env var
		origTestVar := os.Getenv("TEST_DIR_VAR")
		defer func() {
			if origTestVar != "" {
				_ = os.Setenv("TEST_DIR_VAR", origTestVar)
			} else {
				_ = os.Unsetenv("TEST_DIR_VAR")
			}
		}()
		_ = os.Unsetenv("TEST_DIR_VAR")

		// Load env files from directory
		err = LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// Check that value is loaded
		assert.Equal(t, "dir_value", os.Getenv("TEST_DIR_VAR"))
	})
}

func TestGetEnvWithFallback(t *testing.T) {
	testVar := "TEST_FALLBACK_VAR"

	// Clean up env var
	origValue := os.Getenv(testVar)
	defer func() {
		if origValue != "" {
			_ = os.Setenv(testVar, origValue)
		} else {
			_ = os.Unsetenv(testVar)
		}
	}()

	t.Run("returns env value when set", func(t *testing.T) {
		_ = os.Setenv(testVar, "actual_value")
		result := GetEnvWithFallback(testVar, "fallback_value")
		assert.Equal(t, "actual_value", result)
	})

	t.Run("returns fallback when env not set", func(t *testing.T) {
		_ = os.Unsetenv(testVar)
		result := GetEnvWithFallback(testVar, "fallback_value")
		assert.Equal(t, "fallback_value", result)
	})

	t.Run("returns fallback when env is empty", func(t *testing.T) {
		_ = os.Setenv(testVar, "")
		result := GetEnvWithFallback(testVar, "fallback_value")
		assert.Equal(t, "fallback_value", result)
	})
}

// TestAutomergeLabelsIntegration tests the specific use case that was failing
func TestAutomergeLabelsIntegration(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		chdirErr := os.Chdir(originalWd)
		require.NoError(t, chdirErr)
	}()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "go-broadcast-automerge-test")
	require.NoError(t, err)
	defer func() {
		removeErr := os.RemoveAll(tempDir)
		require.NoError(t, removeErr)
	}()

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create .github directory
	githubDir := filepath.Join(tempDir, ".github")
	err = os.MkdirAll(githubDir, 0o750)
	require.NoError(t, err)

	t.Run("automerge labels loaded from env.base", func(t *testing.T) {
		// Create base file with automerge configuration
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `GO_BROADCAST_AUTOMERGE_LABELS=automerge`
		err := os.WriteFile(baseFile, []byte(baseContent), 0o600)
		require.NoError(t, err)

		// Clear env var
		origAutomergeLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS")
		defer func() {
			if origAutomergeLabels != "" {
				_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", origAutomergeLabels)
			} else {
				_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")
			}
		}()
		_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")

		// Load env files
		err = LoadEnvFiles()
		require.NoError(t, err)

		// Check that automerge labels are loaded
		assert.Equal(t, "automerge", os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"))
	})

	t.Run("custom automerge labels override base", func(t *testing.T) {
		// Create base file
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `GO_BROADCAST_AUTOMERGE_LABELS=automerge`
		err := os.WriteFile(baseFile, []byte(baseContent), 0o600)
		require.NoError(t, err)

		// Create custom file with override
		customFile := filepath.Join(githubDir, ".env.custom")
		customContent := `GO_BROADCAST_AUTOMERGE_LABELS=ready-to-merge,approved`
		err = os.WriteFile(customFile, []byte(customContent), 0o600)
		require.NoError(t, err)

		// Clear env var
		origAutomergeLabels := os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS")
		defer func() {
			if origAutomergeLabels != "" {
				_ = os.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", origAutomergeLabels)
			} else {
				_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")
			}
		}()
		_ = os.Unsetenv("GO_BROADCAST_AUTOMERGE_LABELS")

		// Load env files
		err = LoadEnvFiles()
		require.NoError(t, err)

		// Check that custom labels override base
		assert.Equal(t, "ready-to-merge,approved", os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"))
	})
}
