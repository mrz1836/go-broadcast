package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeEnvDir creates baseDir/.github/env/ and writes the given files into it.
func writeEnvDir(t *testing.T, baseDir string, files map[string]string) {
	t.Helper()
	envDir := filepath.Join(baseDir, ".github", "env")
	require.NoError(t, os.MkdirAll(envDir, 0o750))
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(envDir, name), []byte(content), 0o600))
	}
}

// writeLegacyEnvFiles creates the deprecated .github/.env.base and
// .github/.env.custom files, which the loader no longer reads.
func writeLegacyEnvFiles(t *testing.T, baseDir, base, custom string) {
	t.Helper()
	githubDir := filepath.Join(baseDir, ".github")
	require.NoError(t, os.MkdirAll(githubDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(githubDir, ".env.base"), []byte(base), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(githubDir, ".env.custom"), []byte(custom), 0o600))
}

// unsetEnv unsets the given variables for the duration of the test and
// restores their original values afterwards.
func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "") // registers restoration of the original value
		require.NoError(t, os.Unsetenv(key))
	}
}

func TestLoadEnvFiles(t *testing.T) {
	t.Run("loads .github/env files from working directory", func(t *testing.T) {
		tempDir := t.TempDir()
		writeEnvDir(t, tempDir, map[string]string{
			"00-core.env":    "TEST_VAR=core_value\nGO_BROADCAST_AUTOMERGE_LABELS=automerge\nANOTHER_VAR=core_another",
			"90-project.env": "TEST_VAR=project_value\nGO_BROADCAST_AUTOMERGE_LABELS=ready-to-merge,auto-merge",
		})
		t.Chdir(tempDir)
		unsetEnv(t, "TEST_VAR", "GO_BROADCAST_AUTOMERGE_LABELS", "ANOTHER_VAR")

		require.NoError(t, LoadEnvFiles())

		// Later files override earlier ones
		assert.Equal(t, "project_value", os.Getenv("TEST_VAR"))
		assert.Equal(t, "ready-to-merge,auto-merge", os.Getenv("GO_BROADCAST_AUTOMERGE_LABELS"))
		// ANOTHER_VAR is only in 00-core.env
		assert.Equal(t, "core_another", os.Getenv("ANOTHER_VAR"))
	})

	t.Run("no error when .github/env is missing", func(t *testing.T) {
		// Running outside a repository (e.g. "go-broadcast update" from $HOME)
		// must not produce an env file warning.
		t.Chdir(t.TempDir())

		require.NoError(t, LoadEnvFiles())
	})

	t.Run("no error when .github/env has no .env files", func(t *testing.T) {
		tempDir := t.TempDir()
		writeEnvDir(t, tempDir, map[string]string{"README.md": "# Not an env file"})
		t.Chdir(tempDir)

		require.NoError(t, LoadEnvFiles())
	})

	t.Run("ignores deprecated .env.base and .env.custom", func(t *testing.T) {
		tempDir := t.TempDir()
		writeLegacyEnvFiles(t, tempDir, "SOURCE=legacy_base", "SOURCE=legacy_custom")
		t.Chdir(tempDir)
		unsetEnv(t, "SOURCE")

		require.NoError(t, LoadEnvFiles())

		_, exists := os.LookupEnv("SOURCE")
		assert.False(t, exists, "legacy env files should not be loaded")
	})

	t.Run("uses .github/env when legacy files also exist", func(t *testing.T) {
		tempDir := t.TempDir()
		writeLegacyEnvFiles(t, tempDir, "SOURCE=legacy_base", "SOURCE=legacy_custom")
		writeEnvDir(t, tempDir, map[string]string{"00-core.env": "SOURCE=modular"})
		t.Chdir(tempDir)
		unsetEnv(t, "SOURCE")

		require.NoError(t, LoadEnvFiles())

		assert.Equal(t, "modular", os.Getenv("SOURCE"))
	})

	t.Run("returns error when an env file cannot be read", func(t *testing.T) {
		tempDir := t.TempDir()
		// A directory named like an env file is discovered but fails to read
		require.NoError(t, os.MkdirAll(filepath.Join(tempDir, ".github", "env", "00-core.env"), 0o750))
		t.Chdir(tempDir)

		require.Error(t, LoadEnvFiles())
	})
}

func TestLoadEnvFilesFromDir(t *testing.T) {
	t.Run("loads from specified directory", func(t *testing.T) {
		tempDir := t.TempDir()
		writeEnvDir(t, tempDir, map[string]string{"00-core.env": "TEST_DIR_VAR=dir_value"})
		unsetEnv(t, "TEST_DIR_VAR")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

		assert.Equal(t, "dir_value", os.Getenv("TEST_DIR_VAR"))
	})

	t.Run("no error when .github/env is missing", func(t *testing.T) {
		require.NoError(t, LoadEnvFilesFromDir(t.TempDir()))
	})

	t.Run("ignores deprecated .env.base and .env.custom", func(t *testing.T) {
		tempDir := t.TempDir()
		writeLegacyEnvFiles(t, tempDir, "TEST_DIR_VAR=legacy_base", "TEST_DIR_VAR=legacy_custom")
		unsetEnv(t, "TEST_DIR_VAR")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

		_, exists := os.LookupEnv("TEST_DIR_VAR")
		assert.False(t, exists, "legacy env files should not be loaded")
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

// TestOSEnvVarPrecedence tests that OS environment variables take precedence
// over values in .env files. This is the critical test that ensures shell exports
// and CI/CD secrets are respected.
func TestOSEnvVarPrecedence(t *testing.T) {
	t.Run("OS env var takes precedence over env file", func(t *testing.T) {
		tempDir := t.TempDir()
		writeEnvDir(t, tempDir, map[string]string{"00-core.env": "API_KEY=file_value"})

		// Set OS env var BEFORE loading files
		t.Setenv("API_KEY", "os_value")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

		// OS value should WIN (not be overwritten by file)
		assert.Equal(t, "os_value", os.Getenv("API_KEY"),
			"OS env var should take precedence over env file value")
	})

	t.Run("OS env var takes precedence over later override file", func(t *testing.T) {
		tempDir := t.TempDir()
		writeEnvDir(t, tempDir, map[string]string{
			"00-core.env":    "API_KEY=core_value",
			"90-project.env": "API_KEY=project_value",
		})

		t.Setenv("API_KEY", "os_value")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

		// OS value should WIN over both core AND project
		assert.Equal(t, "os_value", os.Getenv("API_KEY"),
			"OS env var should take precedence over 90-project.env value")
	})

	t.Run("file value is used when OS env var is not set", func(t *testing.T) {
		tempDir := t.TempDir()
		writeEnvDir(t, tempDir, map[string]string{"00-core.env": "NEW_VAR=file_value"})
		unsetEnv(t, "NEW_VAR")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

		assert.Equal(t, "file_value", os.Getenv("NEW_VAR"),
			"File value should be used when OS env var is not set")
	})
}

// TestAPIKeyPrecedence tests the specific use case of API keys from shell
// This simulates the user's actual scenario where they have ANTHROPIC_API_KEY
// set in their shell but it was being overwritten by env files.
func TestAPIKeyPrecedence(t *testing.T) {
	t.Run("API key from shell is preserved", func(t *testing.T) {
		tempDir := t.TempDir()
		// AI configuration (but NOT the API key)
		writeEnvDir(t, tempDir, map[string]string{"00-core.env": `GO_BROADCAST_AI_ENABLED=true
GO_BROADCAST_AI_PROVIDER=anthropic
# GO_BROADCAST_AI_API_KEY= intentionally commented out`})

		// Simulate user having API key set in shell (like from ~/.zshrc)
		t.Setenv("GO_BROADCAST_AI_API_KEY", "sk-ant-secret-key")
		unsetEnv(t, "GO_BROADCAST_AI_ENABLED", "GO_BROADCAST_AI_PROVIDER")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

		// API key from shell should be preserved
		assert.Equal(t, "sk-ant-secret-key", os.Getenv("GO_BROADCAST_AI_API_KEY"),
			"API key from shell should NOT be overwritten")

		// Other values should come from file
		assert.Equal(t, "true", os.Getenv("GO_BROADCAST_AI_ENABLED"))
		assert.Equal(t, "anthropic", os.Getenv("GO_BROADCAST_AI_PROVIDER"))
	})

	t.Run("CI/CD secret takes precedence over file", func(t *testing.T) {
		tempDir := t.TempDir()
		// Env file with placeholder and empty API key values
		writeEnvDir(t, tempDir, map[string]string{"00-core.env": "GITHUB_TOKEN=placeholder_token\nANTHROPIC_API_KEY="})

		// Simulate CI/CD environment where secrets are injected as env vars
		t.Setenv("GITHUB_TOKEN", "ghs_cicd_token")
		t.Setenv("ANTHROPIC_API_KEY", "sk-ant-cicd-key")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

		// CI/CD injected values should take precedence
		assert.Equal(t, "ghs_cicd_token", os.Getenv("GITHUB_TOKEN"),
			"CI/CD GITHUB_TOKEN should override file placeholder")
		assert.Equal(t, "sk-ant-cicd-key", os.Getenv("ANTHROPIC_API_KEY"),
			"CI/CD ANTHROPIC_API_KEY should override empty file value")
	})
}

// TestAIEnablementFromEnvFiles tests the full AI enablement scenario
// where 00-core.env has AI config with inline comments, 90-project.env enables AI,
// and the API key comes from the shell environment.
func TestAIEnablementFromEnvFiles(t *testing.T) {
	tempDir := t.TempDir()
	writeEnvDir(t, tempDir, map[string]string{
		// AI config using inline comments (like the real file)
		"00-core.env": `# AI Generation Configuration
GO_BROADCAST_AI_ENABLED=false      # Default to disabled
GO_BROADCAST_AI_PROVIDER=anthropic # AI provider to use
GO_BROADCAST_AI_PR_ENABLED=        # Enable AI for PR body generation
GO_BROADCAST_AI_COMMIT_ENABLED=    # Enable AI for commit message generation
# GO_BROADCAST_AI_API_KEY= DO NOT SET IN FILES`,
		// Project override that enables AI
		"90-project.env": `GO_BROADCAST_AI_ENABLED=true`,
	})

	// Simulate user having API key set in shell (like from ~/.zshrc)
	t.Setenv("GO_BROADCAST_AI_API_KEY", "sk-ant-test-key")

	// Clear other AI vars so they come from files
	unsetEnv(t, "GO_BROADCAST_AI_ENABLED", "GO_BROADCAST_AI_PROVIDER",
		"GO_BROADCAST_AI_PR_ENABLED", "GO_BROADCAST_AI_COMMIT_ENABLED")

	require.NoError(t, LoadEnvFilesFromDir(tempDir))

	// Verify AI configuration is correct
	assert.Equal(t, "true", os.Getenv("GO_BROADCAST_AI_ENABLED"),
		"AI should be enabled from 90-project.env override")
	assert.Equal(t, "sk-ant-test-key", os.Getenv("GO_BROADCAST_AI_API_KEY"),
		"API key should come from OS environment (shell)")
	assert.Equal(t, "anthropic", os.Getenv("GO_BROADCAST_AI_PROVIDER"),
		"Provider should come from 00-core.env (inline comment stripped)")

	// These should be empty because inline comments are stripped
	assert.Empty(t, os.Getenv("GO_BROADCAST_AI_PR_ENABLED"),
		"PR enabled should be empty (inline comment stripped)")
	assert.Empty(t, os.Getenv("GO_BROADCAST_AI_COMMIT_ENABLED"),
		"Commit enabled should be empty (inline comment stripped)")
}

// TestPrecedenceOrder validates the complete precedence chain:
// OS env > 90-project.env > 00-core.env > code defaults
func TestPrecedenceOrder(t *testing.T) {
	tempDir := t.TempDir()
	writeEnvDir(t, tempDir, map[string]string{
		"00-core.env":    "LEVEL1=core\nLEVEL2=core\nLEVEL3=core",
		"90-project.env": "LEVEL1=project\nLEVEL2=project",
	})

	// Set OS env var that overrides the highest level
	t.Setenv("LEVEL1", "os")
	unsetEnv(t, "LEVEL2", "LEVEL3")

	require.NoError(t, LoadEnvFilesFromDir(tempDir))

	// Verify the complete precedence chain
	assert.Equal(t, "os", os.Getenv("LEVEL1"),
		"LEVEL1: OS env should win (highest precedence)")
	assert.Equal(t, "project", os.Getenv("LEVEL2"),
		"LEVEL2: 90-project.env should win over 00-core.env")
	assert.Equal(t, "core", os.Getenv("LEVEL3"),
		"LEVEL3: 00-core.env should be used (not in project, not in OS)")

	// Test with GetEnvWithFallback for code default
	assert.Equal(t, "code_default", GetEnvWithFallback("NONEXISTENT_VAR", "code_default"),
		"NONEXISTENT_VAR: Code default should be used (not in any source)")
}

// TestEmptyEnvVarPreserved tests that explicitly set empty env vars are preserved
// and not overwritten by values from .env files. This is the fix for Issue #1.
func TestEmptyEnvVarPreserved(t *testing.T) {
	tempDir := t.TempDir()
	writeEnvDir(t, tempDir, map[string]string{"00-core.env": "EMPTY_TEST_VAR=file_value"})

	t.Run("explicitly empty env var is NOT overwritten", func(t *testing.T) {
		// Set env var to empty string BEFORE loading files
		t.Setenv("EMPTY_TEST_VAR", "")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

		// Empty value should be preserved (NOT overwritten by file_value)
		assert.Empty(t, os.Getenv("EMPTY_TEST_VAR"),
			"Explicitly set empty env var should NOT be overwritten by file value")
	})

	t.Run("unset env var IS set from file", func(t *testing.T) {
		unsetEnv(t, "EMPTY_TEST_VAR")

		require.NoError(t, LoadEnvFilesFromDir(tempDir))

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
