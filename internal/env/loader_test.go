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

func TestLoadEnvFilesFromDir_ErrorPaths(t *testing.T) {
	t.Run("fails when base file missing", func(t *testing.T) {
		// Create a temporary directory without .env.base
		tempDir, err := os.MkdirTemp("", "go-broadcast-env-missing-test")
		require.NoError(t, err)
		defer func() {
			removeErr := os.RemoveAll(tempDir)
			require.NoError(t, removeErr)
		}()

		// Create .github directory but no .env.base
		githubDir := filepath.Join(tempDir, ".github")
		err = os.MkdirAll(githubDir, 0o750)
		require.NoError(t, err)

		// Attempt to load env files
		err = LoadEnvFilesFromDir(tempDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required .env.base file not found")
	})

	t.Run("loads with custom file present", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "go-broadcast-env-custom-test")
		require.NoError(t, err)
		defer func() {
			removeErr := os.RemoveAll(tempDir)
			require.NoError(t, removeErr)
		}()

		// Create .github directory
		githubDir := filepath.Join(tempDir, ".github")
		err = os.MkdirAll(githubDir, 0o750)
		require.NoError(t, err)

		// Create base file
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `TEST_DIR_BASE_VAR=base_value`
		err = os.WriteFile(baseFile, []byte(baseContent), 0o600)
		require.NoError(t, err)

		// Create custom file with override
		customFile := filepath.Join(githubDir, ".env.custom")
		customContent := `TEST_DIR_BASE_VAR=custom_value`
		err = os.WriteFile(customFile, []byte(customContent), 0o600)
		require.NoError(t, err)

		// Clear env var
		origTestVar := os.Getenv("TEST_DIR_BASE_VAR")
		defer func() {
			if origTestVar != "" {
				_ = os.Setenv("TEST_DIR_BASE_VAR", origTestVar)
			} else {
				_ = os.Unsetenv("TEST_DIR_BASE_VAR")
			}
		}()
		_ = os.Unsetenv("TEST_DIR_BASE_VAR")

		// Load env files from directory
		err = LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// Check that custom value overrides base
		assert.Equal(t, "custom_value", os.Getenv("TEST_DIR_BASE_VAR"))
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

// TestOSEnvVarPrecedence tests that OS environment variables take precedence
// over values in .env files. This is the critical test that ensures shell exports
// and CI/CD secrets are respected.
func TestOSEnvVarPrecedence(t *testing.T) {
	t.Run("OS env var takes precedence over base file", func(t *testing.T) {
		tempDir := t.TempDir()
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))

		// Create base file with a value
		baseFile := filepath.Join(githubDir, ".env.base")
		require.NoError(t, os.WriteFile(baseFile, []byte("API_KEY=file_value"), 0o600))

		// Set OS env var BEFORE loading files
		require.NoError(t, os.Setenv("API_KEY", "os_value"))
		defer func() { _ = os.Unsetenv("API_KEY") }()

		// Load env files
		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// OS value should WIN (not be overwritten by file)
		assert.Equal(t, "os_value", os.Getenv("API_KEY"),
			"OS env var should take precedence over .env.base file value")
	})

	t.Run("OS env var takes precedence over custom file", func(t *testing.T) {
		tempDir := t.TempDir()
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))

		// Create base and custom files with different values
		baseFile := filepath.Join(githubDir, ".env.base")
		require.NoError(t, os.WriteFile(baseFile, []byte("API_KEY=base_value"), 0o600))

		customFile := filepath.Join(githubDir, ".env.custom")
		require.NoError(t, os.WriteFile(customFile, []byte("API_KEY=custom_value"), 0o600))

		// Set OS env var BEFORE loading files
		require.NoError(t, os.Setenv("API_KEY", "os_value"))
		defer func() { _ = os.Unsetenv("API_KEY") }()

		// Load env files
		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// OS value should WIN over both base AND custom
		assert.Equal(t, "os_value", os.Getenv("API_KEY"),
			"OS env var should take precedence over .env.custom file value")
	})

	t.Run("file value is used when OS env var is not set", func(t *testing.T) {
		tempDir := t.TempDir()
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))

		// Create base file with a value
		baseFile := filepath.Join(githubDir, ".env.base")
		require.NoError(t, os.WriteFile(baseFile, []byte("NEW_VAR=file_value"), 0o600))

		// Ensure OS env var is NOT set
		_ = os.Unsetenv("NEW_VAR")
		defer func() { _ = os.Unsetenv("NEW_VAR") }()

		// Load env files
		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// File value should be used since OS var is not set
		assert.Equal(t, "file_value", os.Getenv("NEW_VAR"),
			"File value should be used when OS env var is not set")
	})

	t.Run("mixed precedence - some from OS, some from files", func(t *testing.T) {
		tempDir := t.TempDir()
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))

		// Create base file with multiple vars
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `VAR_FROM_OS=base_value
VAR_FROM_BASE=base_only
VAR_FROM_CUSTOM=base_value`
		require.NoError(t, os.WriteFile(baseFile, []byte(baseContent), 0o600))

		// Create custom file that overrides one var
		customFile := filepath.Join(githubDir, ".env.custom")
		require.NoError(t, os.WriteFile(customFile, []byte("VAR_FROM_CUSTOM=custom_value"), 0o600))

		// Set one OS env var
		require.NoError(t, os.Setenv("VAR_FROM_OS", "os_value"))
		_ = os.Unsetenv("VAR_FROM_BASE")
		_ = os.Unsetenv("VAR_FROM_CUSTOM")
		defer func() {
			_ = os.Unsetenv("VAR_FROM_OS")
			_ = os.Unsetenv("VAR_FROM_BASE")
			_ = os.Unsetenv("VAR_FROM_CUSTOM")
		}()

		// Load env files
		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// Check each variable uses correct precedence
		assert.Equal(t, "os_value", os.Getenv("VAR_FROM_OS"),
			"VAR_FROM_OS should be from OS (highest precedence)")
		assert.Equal(t, "base_only", os.Getenv("VAR_FROM_BASE"),
			"VAR_FROM_BASE should be from base file (not in custom)")
		assert.Equal(t, "custom_value", os.Getenv("VAR_FROM_CUSTOM"),
			"VAR_FROM_CUSTOM should be from custom file (overrides base)")
	})
}

// TestAPIKeyPrecedence tests the specific use case of API keys from shell
// This simulates the user's actual scenario where they have ANTHROPIC_API_KEY
// set in their shell but it was being overwritten by env files.
func TestAPIKeyPrecedence(t *testing.T) {
	t.Run("API key from shell is preserved", func(t *testing.T) {
		tempDir := t.TempDir()
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))

		// Create base file with AI configuration (but NOT the API key)
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `GO_BROADCAST_AI_ENABLED=true
GO_BROADCAST_AI_PROVIDER=anthropic
# GO_BROADCAST_AI_API_KEY= intentionally commented out`
		require.NoError(t, os.WriteFile(baseFile, []byte(baseContent), 0o600))

		// Simulate user having API key set in shell (like from ~/.zshrc)
		require.NoError(t, os.Setenv("GO_BROADCAST_AI_API_KEY", "sk-ant-secret-key"))
		defer func() { _ = os.Unsetenv("GO_BROADCAST_AI_API_KEY") }()

		// Clear other vars that will be set from files
		_ = os.Unsetenv("GO_BROADCAST_AI_ENABLED")
		_ = os.Unsetenv("GO_BROADCAST_AI_PROVIDER")
		defer func() {
			_ = os.Unsetenv("GO_BROADCAST_AI_ENABLED")
			_ = os.Unsetenv("GO_BROADCAST_AI_PROVIDER")
		}()

		// Load env files
		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// API key from shell should be preserved
		assert.Equal(t, "sk-ant-secret-key", os.Getenv("GO_BROADCAST_AI_API_KEY"),
			"API key from shell should NOT be overwritten")

		// Other values should come from file
		assert.Equal(t, "true", os.Getenv("GO_BROADCAST_AI_ENABLED"))
		assert.Equal(t, "anthropic", os.Getenv("GO_BROADCAST_AI_PROVIDER"))
	})

	t.Run("CI/CD secret takes precedence over file", func(t *testing.T) {
		tempDir := t.TempDir()
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))

		// Create base file that has an empty API key placeholder
		baseFile := filepath.Join(githubDir, ".env.base")
		baseContent := `GITHUB_TOKEN=placeholder_token
ANTHROPIC_API_KEY=`
		require.NoError(t, os.WriteFile(baseFile, []byte(baseContent), 0o600))

		// Simulate CI/CD environment where secrets are injected as env vars
		require.NoError(t, os.Setenv("GITHUB_TOKEN", "ghs_cicd_token"))
		require.NoError(t, os.Setenv("ANTHROPIC_API_KEY", "sk-ant-cicd-key"))
		defer func() {
			_ = os.Unsetenv("GITHUB_TOKEN")
			_ = os.Unsetenv("ANTHROPIC_API_KEY")
		}()

		// Load env files
		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// CI/CD injected values should take precedence
		assert.Equal(t, "ghs_cicd_token", os.Getenv("GITHUB_TOKEN"),
			"CI/CD GITHUB_TOKEN should override file placeholder")
		assert.Equal(t, "sk-ant-cicd-key", os.Getenv("ANTHROPIC_API_KEY"),
			"CI/CD ANTHROPIC_API_KEY should override empty file value")
	})
}

// TestAIEnablementFromEnvFiles tests the full AI enablement scenario
// where .env.base has AI config with inline comments, .env.custom enables AI,
// and the API key comes from the shell environment.
func TestAIEnablementFromEnvFiles(t *testing.T) {
	tempDir := t.TempDir()
	githubDir := filepath.Join(tempDir, ".github")
	require.NoError(t, os.MkdirAll(githubDir, 0o750))

	// Create .env.base with AI config using inline comments (like the real file)
	baseContent := `# AI Generation Configuration
GO_BROADCAST_AI_ENABLED=false      # Default to disabled
GO_BROADCAST_AI_PROVIDER=anthropic # AI provider to use
GO_BROADCAST_AI_PR_ENABLED=        # Enable AI for PR body generation
GO_BROADCAST_AI_COMMIT_ENABLED=    # Enable AI for commit message generation
# GO_BROADCAST_AI_API_KEY= DO NOT SET IN FILES`
	baseFile := filepath.Join(githubDir, ".env.base")
	require.NoError(t, os.WriteFile(baseFile, []byte(baseContent), 0o600))

	// Create .env.custom that enables AI (user override)
	customContent := `GO_BROADCAST_AI_ENABLED=true`
	customFile := filepath.Join(githubDir, ".env.custom")
	require.NoError(t, os.WriteFile(customFile, []byte(customContent), 0o600))

	// Simulate user having API key set in shell (like from ~/.zshrc)
	require.NoError(t, os.Setenv("GO_BROADCAST_AI_API_KEY", "sk-ant-test-key"))

	// Clear other AI vars so they come from files
	_ = os.Unsetenv("GO_BROADCAST_AI_ENABLED")
	_ = os.Unsetenv("GO_BROADCAST_AI_PROVIDER")
	_ = os.Unsetenv("GO_BROADCAST_AI_PR_ENABLED")
	_ = os.Unsetenv("GO_BROADCAST_AI_COMMIT_ENABLED")

	defer func() {
		_ = os.Unsetenv("GO_BROADCAST_AI_API_KEY")
		_ = os.Unsetenv("GO_BROADCAST_AI_ENABLED")
		_ = os.Unsetenv("GO_BROADCAST_AI_PROVIDER")
		_ = os.Unsetenv("GO_BROADCAST_AI_PR_ENABLED")
		_ = os.Unsetenv("GO_BROADCAST_AI_COMMIT_ENABLED")
	}()

	// Load env files
	err := LoadEnvFilesFromDir(tempDir)
	require.NoError(t, err)

	// Verify AI configuration is correct
	assert.Equal(t, "true", os.Getenv("GO_BROADCAST_AI_ENABLED"),
		"AI should be enabled from .env.custom override")
	assert.Equal(t, "sk-ant-test-key", os.Getenv("GO_BROADCAST_AI_API_KEY"),
		"API key should come from OS environment (shell)")
	assert.Equal(t, "anthropic", os.Getenv("GO_BROADCAST_AI_PROVIDER"),
		"Provider should come from .env.base (inline comment stripped)")

	// These should be empty because inline comments are stripped
	assert.Empty(t, os.Getenv("GO_BROADCAST_AI_PR_ENABLED"),
		"PR enabled should be empty (inline comment stripped)")
	assert.Empty(t, os.Getenv("GO_BROADCAST_AI_COMMIT_ENABLED"),
		"Commit enabled should be empty (inline comment stripped)")
}

// TestPrecedenceOrder validates the complete precedence chain:
// OS env > .env.custom > .env.base > code defaults
func TestPrecedenceOrder(t *testing.T) {
	tempDir := t.TempDir()
	githubDir := filepath.Join(tempDir, ".github")
	require.NoError(t, os.MkdirAll(githubDir, 0o750))

	// Create base file
	baseFile := filepath.Join(githubDir, ".env.base")
	baseContent := `LEVEL1=base
LEVEL2=base
LEVEL3=base`
	require.NoError(t, os.WriteFile(baseFile, []byte(baseContent), 0o600))

	// Create custom file that overrides some values
	customFile := filepath.Join(githubDir, ".env.custom")
	customContent := `LEVEL1=custom
LEVEL2=custom`
	require.NoError(t, os.WriteFile(customFile, []byte(customContent), 0o600))

	// Set OS env var that overrides the highest level
	require.NoError(t, os.Setenv("LEVEL1", "os"))
	_ = os.Unsetenv("LEVEL2")
	_ = os.Unsetenv("LEVEL3")
	defer func() {
		_ = os.Unsetenv("LEVEL1")
		_ = os.Unsetenv("LEVEL2")
		_ = os.Unsetenv("LEVEL3")
	}()

	// Load env files
	err := LoadEnvFilesFromDir(tempDir)
	require.NoError(t, err)

	// Verify the complete precedence chain
	assert.Equal(t, "os", os.Getenv("LEVEL1"),
		"LEVEL1: OS env should win (highest precedence)")
	assert.Equal(t, "custom", os.Getenv("LEVEL2"),
		"LEVEL2: .env.custom should win over .env.base")
	assert.Equal(t, "base", os.Getenv("LEVEL3"),
		"LEVEL3: .env.base should be used (not in custom, not in OS)")

	// Test with GetEnvWithFallback for code default
	assert.Equal(t, "code_default", GetEnvWithFallback("NONEXISTENT_VAR", "code_default"),
		"NONEXISTENT_VAR: Code default should be used (not in any source)")
}

// TestEmptyEnvVarPreserved tests that explicitly set empty env vars are preserved
// and not overwritten by values from .env files. This is the fix for Issue #1.
func TestEmptyEnvVarPreserved(t *testing.T) {
	tempDir := t.TempDir()
	githubDir := filepath.Join(tempDir, ".github")
	require.NoError(t, os.MkdirAll(githubDir, 0o750))

	// Create base file with a value
	baseFile := filepath.Join(githubDir, ".env.base")
	require.NoError(t, os.WriteFile(baseFile, []byte("EMPTY_TEST_VAR=file_value"), 0o600))

	t.Run("explicitly empty env var is NOT overwritten", func(t *testing.T) {
		// Set env var to empty string BEFORE loading files
		t.Setenv("EMPTY_TEST_VAR", "")

		// Load env files
		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// Empty value should be preserved (NOT overwritten by file_value)
		assert.Empty(t, os.Getenv("EMPTY_TEST_VAR"),
			"Explicitly set empty env var should NOT be overwritten by file value")
	})

	t.Run("unset env var IS set from file", func(t *testing.T) {
		// Ensure env var is NOT set
		_ = os.Unsetenv("EMPTY_TEST_VAR")
		t.Cleanup(func() { _ = os.Unsetenv("EMPTY_TEST_VAR") })

		// Load env files
		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		// Unset var should now have file value
		assert.Equal(t, "file_value", os.Getenv("EMPTY_TEST_VAR"),
			"Unset env var should be set from file")
	})
}

// TestGetEnvOrDefault tests the new GetEnvOrDefault function that preserves
// explicitly set empty values (unlike GetEnvWithFallback).
func TestGetEnvOrDefault(t *testing.T) {
	testVar := "TEST_GET_ENV_OR_DEFAULT"

	t.Run("returns value when set to non-empty", func(t *testing.T) {
		t.Setenv(testVar, "actual_value")
		result := GetEnvOrDefault(testVar, "default_value")
		assert.Equal(t, "actual_value", result)
	})

	t.Run("returns empty string when explicitly set to empty", func(t *testing.T) {
		t.Setenv(testVar, "")
		result := GetEnvOrDefault(testVar, "default_value")
		assert.Empty(t, result, "GetEnvOrDefault should preserve explicitly empty values")
	})

	t.Run("returns default when env not set", func(t *testing.T) {
		_ = os.Unsetenv(testVar)
		t.Cleanup(func() { _ = os.Unsetenv(testVar) })
		result := GetEnvOrDefault(testVar, "default_value")
		assert.Equal(t, "default_value", result)
	})

	t.Run("contrast with GetEnvWithFallback on empty", func(t *testing.T) {
		t.Setenv(testVar, "")

		// GetEnvOrDefault preserves empty
		assert.Empty(t, GetEnvOrDefault(testVar, "default"),
			"GetEnvOrDefault should return empty string")

		// GetEnvWithFallback returns fallback for empty
		assert.Equal(t, "default", GetEnvWithFallback(testVar, "default"),
			"GetEnvWithFallback should return fallback for empty string")
	})
}

// TestLoadEnvDir tests the core directory loading functionality.
func TestLoadEnvDir(t *testing.T) {
	t.Run("loads multiple numbered files in correct order", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create files with different numbered prefixes
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "00-core.env"),
			[]byte("CORE_VAR=core\nSHARED=from_core"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "10-tools.env"),
			[]byte("TOOL_VAR=tool\nSHARED=from_tools"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "90-project.env"),
			[]byte("PROJECT_VAR=project\nSHARED=from_project"), 0o600))

		_ = os.Unsetenv("CORE_VAR")
		_ = os.Unsetenv("TOOL_VAR")
		_ = os.Unsetenv("PROJECT_VAR")
		_ = os.Unsetenv("SHARED")
		defer func() {
			_ = os.Unsetenv("CORE_VAR")
			_ = os.Unsetenv("TOOL_VAR")
			_ = os.Unsetenv("PROJECT_VAR")
			_ = os.Unsetenv("SHARED")
		}()

		err := LoadEnvDir(tempDir, false)
		require.NoError(t, err)

		assert.Equal(t, "core", os.Getenv("CORE_VAR"))
		assert.Equal(t, "tool", os.Getenv("TOOL_VAR"))
		assert.Equal(t, "project", os.Getenv("PROJECT_VAR"))
		// Last file (90-project.env) should win for shared key
		assert.Equal(t, "from_project", os.Getenv("SHARED"))
	})

	t.Run("OS env vars are never overwritten", func(t *testing.T) {
		tempDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "00-core.env"),
			[]byte("PROTECTED=from_file\nNEW_VAR=from_file"), 0o600))

		t.Setenv("PROTECTED", "from_os")
		_ = os.Unsetenv("NEW_VAR")
		defer func() { _ = os.Unsetenv("NEW_VAR") }()

		err := LoadEnvDir(tempDir, false)
		require.NoError(t, err)

		assert.Equal(t, "from_os", os.Getenv("PROTECTED"),
			"OS env var should not be overwritten")
		assert.Equal(t, "from_file", os.Getenv("NEW_VAR"),
			"Unset var should be set from file")
	})

	t.Run("returns error for non-directory path", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "not-a-dir.env")
		require.NoError(t, os.WriteFile(tempFile, []byte("KEY=val"), 0o600))

		err := LoadEnvDir(tempFile, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotDirectory)
	})

	t.Run("returns error for directory with no .env files", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create a non-.env file
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "README.md"),
			[]byte("# Not an env file"), 0o600))

		err := LoadEnvDir(tempDir, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoEnvFiles)
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		err := LoadEnvDir("/nonexistent/path/to/env", false)
		require.Error(t, err)
	})
}

// TestLoadEnvDirSkipLocal tests CI awareness with 99-local.env.
func TestLoadEnvDirSkipLocal(t *testing.T) {
	t.Run("skipLocal true skips 99-local.env", func(t *testing.T) {
		tempDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "00-core.env"),
			[]byte("CORE_VAR=core\nOVERRIDE=core"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "99-local.env"),
			[]byte("LOCAL_VAR=local\nOVERRIDE=local"), 0o600))

		_ = os.Unsetenv("CORE_VAR")
		_ = os.Unsetenv("LOCAL_VAR")
		_ = os.Unsetenv("OVERRIDE")
		defer func() {
			_ = os.Unsetenv("CORE_VAR")
			_ = os.Unsetenv("LOCAL_VAR")
			_ = os.Unsetenv("OVERRIDE")
		}()

		err := LoadEnvDir(tempDir, true)
		require.NoError(t, err)

		assert.Equal(t, "core", os.Getenv("CORE_VAR"))
		// LOCAL_VAR should NOT be set because 99-local.env was skipped
		assert.Empty(t, os.Getenv("LOCAL_VAR"),
			"99-local.env should be skipped when skipLocal is true")
		// OVERRIDE should be from core, not local
		assert.Equal(t, "core", os.Getenv("OVERRIDE"),
			"99-local.env overrides should not apply when skipped")
	})

	t.Run("skipLocal false loads 99-local.env", func(t *testing.T) {
		tempDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "00-core.env"),
			[]byte("CORE_VAR=core\nOVERRIDE=core"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "99-local.env"),
			[]byte("LOCAL_VAR=local\nOVERRIDE=local"), 0o600))

		_ = os.Unsetenv("CORE_VAR")
		_ = os.Unsetenv("LOCAL_VAR")
		_ = os.Unsetenv("OVERRIDE")
		defer func() {
			_ = os.Unsetenv("CORE_VAR")
			_ = os.Unsetenv("LOCAL_VAR")
			_ = os.Unsetenv("OVERRIDE")
		}()

		err := LoadEnvDir(tempDir, false)
		require.NoError(t, err)

		assert.Equal(t, "core", os.Getenv("CORE_VAR"))
		assert.Equal(t, "local", os.Getenv("LOCAL_VAR"),
			"99-local.env should be loaded when skipLocal is false")
		assert.Equal(t, "local", os.Getenv("OVERRIDE"),
			"99-local.env should override earlier files")
	})

	t.Run("other files always loaded regardless of skipLocal", func(t *testing.T) {
		tempDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "00-core.env"),
			[]byte("A=1"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "50-middle.env"),
			[]byte("B=2"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "99-local.env"),
			[]byte("C=3"), 0o600))

		_ = os.Unsetenv("A")
		_ = os.Unsetenv("B")
		_ = os.Unsetenv("C")
		defer func() {
			_ = os.Unsetenv("A")
			_ = os.Unsetenv("B")
			_ = os.Unsetenv("C")
		}()

		err := LoadEnvDir(tempDir, true)
		require.NoError(t, err)

		assert.Equal(t, "1", os.Getenv("A"), "00-core.env should always load")
		assert.Equal(t, "2", os.Getenv("B"), "50-middle.env should always load")
		assert.Empty(t, os.Getenv("C"), "99-local.env should be skipped")
	})
}

// TestLoadEnvDirSortOrder tests deterministic lexicographic ordering.
func TestLoadEnvDirSortOrder(t *testing.T) {
	tempDir := t.TempDir()

	// Create files deliberately out of filesystem order
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "20-second.env"),
		[]byte("ORDER=20"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "00-first.env"),
		[]byte("ORDER=00"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "10-middle.env"),
		[]byte("ORDER=10"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "90-last.env"),
		[]byte("ORDER=90"), 0o600))

	_ = os.Unsetenv("ORDER")
	defer func() { _ = os.Unsetenv("ORDER") }()

	err := LoadEnvDir(tempDir, false)
	require.NoError(t, err)

	// 90-last.env should be loaded last (highest number), so its value wins
	assert.Equal(t, "90", os.Getenv("ORDER"),
		"Last file in sorted order should win")
}

// TestLoadEnvFilesModularPreferred tests that modular .github/env/ is preferred over legacy.
func TestLoadEnvFilesModularPreferred(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		chdirErr := os.Chdir(originalWd)
		require.NoError(t, chdirErr)
	}()

	t.Run("uses modular when both exist", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "env-modular-preferred-test")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		require.NoError(t, os.Chdir(tempDir))

		// Create legacy files
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(githubDir, ".env.base"),
			[]byte("SOURCE=legacy"), 0o600))

		// Create modular directory
		envDir := filepath.Join(githubDir, "env")
		require.NoError(t, os.MkdirAll(envDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(envDir, "00-core.env"),
			[]byte("SOURCE=modular"), 0o600))

		_ = os.Unsetenv("SOURCE")
		defer func() { _ = os.Unsetenv("SOURCE") }()

		err = LoadEnvFiles()
		require.NoError(t, err)

		assert.Equal(t, "modular", os.Getenv("SOURCE"),
			"Modular env dir should be preferred over legacy files")
	})

	t.Run("falls back to legacy when modular dir absent", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "env-legacy-fallback-test")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		require.NoError(t, os.Chdir(tempDir))

		// Create only legacy files (no modular dir)
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(githubDir, ".env.base"),
			[]byte("SOURCE=legacy"), 0o600))

		_ = os.Unsetenv("SOURCE")
		defer func() { _ = os.Unsetenv("SOURCE") }()

		err = LoadEnvFiles()
		require.NoError(t, err)

		assert.Equal(t, "legacy", os.Getenv("SOURCE"),
			"Should fall back to legacy when modular dir is absent")
	})

	t.Run("falls back to legacy when modular dir has no env files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "env-empty-modular-test")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		require.NoError(t, os.Chdir(tempDir))

		// Create legacy files
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(githubDir, ".env.base"),
			[]byte("SOURCE=legacy"), 0o600))

		// Create modular dir with only non-.env files
		envDir := filepath.Join(githubDir, "env")
		require.NoError(t, os.MkdirAll(envDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(envDir, "README.md"),
			[]byte("# Not an env file"), 0o600))

		_ = os.Unsetenv("SOURCE")
		defer func() { _ = os.Unsetenv("SOURCE") }()

		err = LoadEnvFiles()
		require.NoError(t, err)

		assert.Equal(t, "legacy", os.Getenv("SOURCE"),
			"Should fall back to legacy when modular dir has no .env files")
	})
}

// TestLoadEnvFilesFromDirModularPreferred tests modular-first behavior for LoadEnvFilesFromDir.
func TestLoadEnvFilesFromDirModularPreferred(t *testing.T) {
	t.Run("uses modular when both exist", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create legacy files
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(githubDir, ".env.base"),
			[]byte("SOURCE=legacy"), 0o600))

		// Create modular directory
		envDir := filepath.Join(githubDir, "env")
		require.NoError(t, os.MkdirAll(envDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(envDir, "00-core.env"),
			[]byte("SOURCE=modular"), 0o600))

		_ = os.Unsetenv("SOURCE")
		defer func() { _ = os.Unsetenv("SOURCE") }()

		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		assert.Equal(t, "modular", os.Getenv("SOURCE"),
			"Modular env dir should be preferred over legacy files")
	})

	t.Run("falls back to legacy when modular dir absent", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create only legacy files
		githubDir := filepath.Join(tempDir, ".github")
		require.NoError(t, os.MkdirAll(githubDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(githubDir, ".env.base"),
			[]byte("SOURCE=legacy"), 0o600))

		_ = os.Unsetenv("SOURCE")
		defer func() { _ = os.Unsetenv("SOURCE") }()

		err := LoadEnvFilesFromDir(tempDir)
		require.NoError(t, err)

		assert.Equal(t, "legacy", os.Getenv("SOURCE"),
			"Should fall back to legacy when modular dir is absent")
	})
}

// TestLoadEnvDirOSPrecedence tests that OS env vars take precedence with modular loading.
func TestLoadEnvDirOSPrecedence(t *testing.T) {
	t.Run("mixed scenario with some from OS, some from files", func(t *testing.T) {
		tempDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "00-core.env"),
			[]byte("FROM_OS=file_value\nFROM_FILE=file_value\nFROM_OVERRIDE=core"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "90-project.env"),
			[]byte("FROM_OVERRIDE=project"), 0o600))

		t.Setenv("FROM_OS", "os_value")
		_ = os.Unsetenv("FROM_FILE")
		_ = os.Unsetenv("FROM_OVERRIDE")
		defer func() {
			_ = os.Unsetenv("FROM_FILE")
			_ = os.Unsetenv("FROM_OVERRIDE")
		}()

		err := LoadEnvDir(tempDir, false)
		require.NoError(t, err)

		assert.Equal(t, "os_value", os.Getenv("FROM_OS"),
			"OS env var should be preserved")
		assert.Equal(t, "file_value", os.Getenv("FROM_FILE"),
			"Unset var should come from file")
		assert.Equal(t, "project", os.Getenv("FROM_OVERRIDE"),
			"Later file should override earlier file")
	})
}

// TestIsCI tests the CI detection helper.
func TestIsCI(t *testing.T) {
	t.Run("returns true when CI=true", func(t *testing.T) {
		t.Setenv("CI", "true")
		assert.True(t, isCI())
	})

	t.Run("returns false when CI is not set", func(t *testing.T) {
		_ = os.Unsetenv("CI")
		t.Cleanup(func() { _ = os.Unsetenv("CI") })
		assert.False(t, isCI())
	})

	t.Run("returns false when CI is other value", func(t *testing.T) {
		t.Setenv("CI", "false")
		assert.False(t, isCI())
	})

	t.Run("returns false when CI is empty", func(t *testing.T) {
		t.Setenv("CI", "")
		assert.False(t, isCI())
	})
}

// TestHasEnvFiles tests the directory validation helper.
func TestHasEnvFiles(t *testing.T) {
	t.Run("returns true for directory with .env files", func(t *testing.T) {
		tempDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "00-core.env"),
			[]byte("KEY=val"), 0o600))
		assert.True(t, hasEnvFiles(tempDir))
	})

	t.Run("returns false for directory without .env files", func(t *testing.T) {
		tempDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "README.md"),
			[]byte("# readme"), 0o600))
		assert.False(t, hasEnvFiles(tempDir))
	})

	t.Run("returns false for empty directory", func(t *testing.T) {
		tempDir := t.TempDir()
		assert.False(t, hasEnvFiles(tempDir))
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		assert.False(t, hasEnvFiles("/nonexistent/path"))
	})

	t.Run("returns false for file path", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "file.env")
		require.NoError(t, os.WriteFile(tempFile, []byte("KEY=val"), 0o600))
		assert.False(t, hasEnvFiles(tempFile))
	})
}

// TestFindEnvDir tests the directory discovery helper.
func TestFindEnvDir(t *testing.T) {
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		chdirErr := os.Chdir(originalWd)
		require.NoError(t, chdirErr)
	}()

	t.Run("finds .github/env with env files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "env-find-test")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		envDir := filepath.Join(tempDir, ".github", "env")
		require.NoError(t, os.MkdirAll(envDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(envDir, "00-core.env"),
			[]byte("KEY=val"), 0o600))

		require.NoError(t, os.Chdir(tempDir))

		result := findEnvDir()
		assert.NotEmpty(t, result)
		assert.Equal(t, filepath.Join(".github", "env"), result)
	})

	t.Run("returns empty when .github/env missing", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "env-find-missing-test")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		require.NoError(t, os.Chdir(tempDir))

		result := findEnvDir()
		assert.Empty(t, result)
	})

	t.Run("returns empty when .github/env has no .env files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "env-find-empty-test")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		envDir := filepath.Join(tempDir, ".github", "env")
		require.NoError(t, os.MkdirAll(envDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(envDir, "README.md"),
			[]byte("# readme"), 0o600))

		require.NoError(t, os.Chdir(tempDir))

		result := findEnvDir()
		assert.Empty(t, result)
	})
}

// TestEnvLoadingRaceConditionNote documents that these tests manipulate global
// environment variables and therefore cannot be run with t.Parallel().
// The actual loader functions are thread-safe (no internal shared state),
// but the tests themselves modify shared global state (os env vars).
func TestEnvLoadingRaceConditionNote(t *testing.T) {
	// This test exists to document the race condition concern.
	// Tests in this file intentionally do NOT call t.Parallel() because they
	// modify global environment variables via os.Setenv/os.Unsetenv.
	//
	// If you need to run tests in parallel, use t.Setenv() which automatically
	// marks the test as incompatible with parallel execution.
	t.Log("Tests in env package modify global env vars and cannot run in parallel")
}
