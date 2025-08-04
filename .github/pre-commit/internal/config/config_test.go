package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestLoad(t *testing.T) {
	// Save current working directory
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWD) }()

	// Change to repository root for test
	err = os.Chdir("../../../..")
	require.NoError(t, err)

	// Test loading configuration
	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify some expected values
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, int64(10*1024*1024), cfg.MaxFileSize)
	assert.Equal(t, 100, cfg.MaxFilesOpen)
	assert.Equal(t, 120, cfg.Timeout)

	// Check that checks are enabled by default
	assert.True(t, cfg.Checks.Fumpt)
	assert.True(t, cfg.Checks.Lint)
	assert.True(t, cfg.Checks.ModTidy)
	assert.True(t, cfg.Checks.Whitespace)
	assert.True(t, cfg.Checks.EOF)
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"true value", "true", false, true},
		{"false value", "false", true, false},
		{"empty value", "", true, true},
		{"invalid value", "invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_BOOL", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_BOOL") }()

			result := getBoolEnv("TEST_BOOL", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"valid int", "42", 0, 42},
		{"empty value", "", 10, 10},
		{"invalid value", "abc", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_INT", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_INT") }()

			result := getIntEnv("TEST_INT", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{"value set", "test", "default", "test"},
		{"empty value", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_STRING", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_STRING") }()

			result := getStringEnv("TEST_STRING", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Comprehensive test suite for config functionality

type ConfigTestSuite struct {
	suite.Suite

	tempDir string
	oldDir  string
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "config_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Clear environment variables to ensure clean test state
	s.clearEnvVars()
}

func (s *ConfigTestSuite) TearDownTest() {
	// Clear environment variables after test
	s.clearEnvVars()

	if s.oldDir != "" {
		err := os.Chdir(s.oldDir)
		s.Require().NoError(err)
	}
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *ConfigTestSuite) clearEnvVars() {
	envVars := []string{
		"ENABLE_PRE_COMMIT_SYSTEM",
		"PRE_COMMIT_SYSTEM_LOG_LEVEL",
		"PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB",
		"PRE_COMMIT_SYSTEM_MAX_FILES_OPEN",
		"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS",
		"PRE_COMMIT_SYSTEM_ENABLE_FUMPT",
		"PRE_COMMIT_SYSTEM_ENABLE_LINT",
		"PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY",
		"PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE",
		"PRE_COMMIT_SYSTEM_ENABLE_EOF",
		"PRE_COMMIT_SYSTEM_FUMPT_VERSION",
		"PRE_COMMIT_SYSTEM_GOLANGCI_LINT_VERSION",
		"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS",
		"PRE_COMMIT_SYSTEM_FAIL_FAST",
		"PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT",
		"PRE_COMMIT_SYSTEM_LINT_TIMEOUT",
		"PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT",
		"PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT",
		"PRE_COMMIT_SYSTEM_EOF_TIMEOUT",
		"PRE_COMMIT_SYSTEM_HOOKS_PATH",
		"PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS",
		"PRE_COMMIT_SYSTEM_COLOR_OUTPUT",
	}

	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}
}

func (s *ConfigTestSuite) createEnvFile(content string) {
	githubDir := filepath.Join(s.tempDir, ".github")
	err := os.MkdirAll(githubDir, 0o750)
	s.Require().NoError(err)

	envFile := filepath.Join(githubDir, ".env.shared")
	err = os.WriteFile(envFile, []byte(content), 0o600)
	s.Require().NoError(err)
}

// TestLoadWithCustomConfiguration tests loading with custom environment variables
func (s *ConfigTestSuite) TestLoadWithCustomConfiguration() {
	envContent := `# Custom configuration
ENABLE_PRE_COMMIT_SYSTEM=false
PRE_COMMIT_SYSTEM_LOG_LEVEL=debug
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=5
PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=50
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=false
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_FUMPT_VERSION=v0.5.0
PRE_COMMIT_SYSTEM_GOLANGCI_LINT_VERSION=v1.54.0
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=4
PRE_COMMIT_SYSTEM_FAIL_FAST=true
PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT=60
PRE_COMMIT_SYSTEM_LINT_TIMEOUT=90
PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT=45
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=15
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=10
PRE_COMMIT_SYSTEM_HOOKS_PATH=.git/custom-hooks
PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS=vendor/,dist/,build/
PRE_COMMIT_SYSTEM_COLOR_OUTPUT=false
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)

	// Core settings
	s.False(cfg.Enabled)
	s.Equal("debug", cfg.LogLevel)
	s.Equal(int64(5*1024*1024), cfg.MaxFileSize)
	s.Equal(50, cfg.MaxFilesOpen)
	s.Equal(120, cfg.Timeout)

	// Check configurations
	s.False(cfg.Checks.Fumpt)
	s.False(cfg.Checks.Lint)
	s.True(cfg.Checks.ModTidy)
	s.False(cfg.Checks.Whitespace)
	s.True(cfg.Checks.EOF)

	// Tool versions
	s.Equal("v0.5.0", cfg.ToolVersions.Fumpt)
	s.Equal("v1.54.0", cfg.ToolVersions.GolangciLint)

	// Performance settings
	s.Equal(4, cfg.Performance.ParallelWorkers)
	s.True(cfg.Performance.FailFast)

	// Check timeouts
	s.Equal(60, cfg.CheckTimeouts.Fumpt)
	s.Equal(90, cfg.CheckTimeouts.Lint)
	s.Equal(45, cfg.CheckTimeouts.ModTidy)
	s.Equal(15, cfg.CheckTimeouts.Whitespace)
	s.Equal(10, cfg.CheckTimeouts.EOF)

	// Git settings
	s.Equal(".git/custom-hooks", cfg.Git.HooksPath)
	s.Equal([]string{"vendor/", "dist/", "build/"}, cfg.Git.ExcludePatterns)

	// UI settings
	s.False(cfg.UI.ColorOutput)

	// Directory should be derived from env file location
	// The config code uses filepath.Dir(envPath) + "/pre-commit"
	// When envPath is ".github/.env.shared", Directory becomes ".github/pre-commit"
	// When envPath is absolute, Directory becomes absolute
	expectedDir := ".github/pre-commit"
	s.Equal(expectedDir, cfg.Directory)
}

// TestLoadWithMinimalConfiguration tests loading with minimal configuration
func (s *ConfigTestSuite) TestLoadWithMinimalConfiguration() {
	envContent := `ENABLE_PRE_COMMIT_SYSTEM=true
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)

	// Should use defaults for unspecified values
	s.True(cfg.Enabled)
	s.Equal("info", cfg.LogLevel)
	s.Equal(int64(10*1024*1024), cfg.MaxFileSize)
	s.Equal(100, cfg.MaxFilesOpen)
	s.Equal(300, cfg.Timeout)
	s.True(cfg.Checks.Fumpt)
	s.True(cfg.Checks.Lint)
	s.True(cfg.Checks.ModTidy)
	s.True(cfg.Checks.Whitespace)
	s.True(cfg.Checks.EOF)
}

// TestLoadWithEmptyExcludePatterns tests exclude patterns handling
func (s *ConfigTestSuite) TestLoadWithEmptyExcludePatterns() {
	envContent := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS=
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)
	// When empty string is provided via environment variable,
	// getStringEnv returns the default value "vendor/,node_modules/,.git/"
	// So we expect the default patterns to be present
	s.Equal([]string{"vendor/", "node_modules/", ".git/"}, cfg.Git.ExcludePatterns)
}

// TestLoadWithSpacedExcludePatterns tests exclude patterns with spaces
func (s *ConfigTestSuite) TestLoadWithSpacedExcludePatterns() {
	envContent := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS=vendor/ , node_modules/ , .git/
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)
	s.Equal([]string{"vendor/", "node_modules/", ".git/"}, cfg.Git.ExcludePatterns)
}

// TestLoadMissingEnvFile tests behavior when .env.shared file is not found
func (s *ConfigTestSuite) TestLoadMissingEnvFile() {
	// Don't create .env.shared file
	cfg, err := Load()
	s.Require().Error(err)
	s.Nil(cfg)
	s.Contains(err.Error(), "failed to find .env.shared")
}

// TestLoadCorruptedEnvFile tests behavior with corrupted env file
func (s *ConfigTestSuite) TestLoadCorruptedEnvFile() {
	// Create a directory instead of a file to simulate corruption
	githubDir := filepath.Join(s.tempDir, ".github")
	err := os.MkdirAll(githubDir, 0o750)
	s.Require().NoError(err)

	envPath := filepath.Join(githubDir, ".env.shared")
	err = os.Mkdir(envPath, 0o750) // Create directory instead of file
	s.Require().NoError(err)

	cfg, err := Load()
	s.Require().Error(err)
	s.Nil(cfg)
	s.Contains(err.Error(), "failed to load")
}

// TestFindEnvFileInParentDirectories tests finding env file in parent directories
func (s *ConfigTestSuite) TestFindEnvFileInParentDirectories() {
	// Create env file in parent directory
	envContent := `ENABLE_PRE_COMMIT_SYSTEM=true
`
	s.createEnvFile(envContent)

	// Create subdirectory and change to it
	subDir := filepath.Join(s.tempDir, "subdir", "deep")
	err := os.MkdirAll(subDir, 0o750)
	s.Require().NoError(err)

	err = os.Chdir(subDir)
	s.Require().NoError(err)

	// Should find env file in parent
	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)
	s.True(cfg.Enabled)
}

// TestFindEnvFileInCurrentDirectory tests finding env file in current directory
func (s *ConfigTestSuite) TestFindEnvFileInCurrentDirectory() {
	envContent := `ENABLE_PRE_COMMIT_SYSTEM=true
`
	s.createEnvFile(envContent)

	// Should find env file in current directory
	envPath, err := findEnvFile()
	s.Require().NoError(err)
	s.Equal(".github/.env.shared", envPath)
}

// TestConfigStructInitialization tests that all config fields are properly initialized
func (s *ConfigTestSuite) TestConfigStructInitialization() {
	envContent := `ENABLE_PRE_COMMIT_SYSTEM=true
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)

	// Verify all major struct fields are initialized
	s.NotEmpty(cfg.Directory)
	s.NotEmpty(cfg.LogLevel)
	s.Positive(cfg.MaxFileSize)
	s.Positive(cfg.MaxFilesOpen)
	s.Positive(cfg.Timeout)
	s.NotEmpty(cfg.ToolVersions.Fumpt)
	s.NotEmpty(cfg.ToolVersions.GolangciLint)
	s.GreaterOrEqual(cfg.Performance.ParallelWorkers, 0)
	s.Positive(cfg.CheckTimeouts.Fumpt)
	s.Positive(cfg.CheckTimeouts.Lint)
	s.Positive(cfg.CheckTimeouts.ModTidy)
	s.Positive(cfg.CheckTimeouts.Whitespace)
	s.Positive(cfg.CheckTimeouts.EOF)
	s.NotEmpty(cfg.Git.HooksPath)
}

// Unit tests for edge cases and error conditions

func TestGetBoolEnvEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"1 as true", "1", false, true},
		{"0 as false", "0", true, false},
		{"TRUE uppercase", "TRUE", false, true},
		{"FALSE uppercase", "FALSE", true, false},
		{"yes value", "yes", false, false}, // Should use default for invalid
		{"no value", "no", true, true},     // Should use default for invalid
		{"random string", "random", false, false},
		{"whitespace value", " true ", false, false}, // Whitespace should fail parsing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Setenv("TEST_BOOL_EDGE", tt.envValue))
			defer func() {
				if err := os.Unsetenv("TEST_BOOL_EDGE"); err != nil {
					t.Logf("Failed to unset TEST_BOOL_EDGE: %v", err)
				}
			}()

			result := getBoolEnv("TEST_BOOL_EDGE", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntEnvEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"negative int", "-42", 0, -42},
		{"zero", "0", 10, 0},
		{"large number", "999999", 0, 999999},
		{"float value", "42.5", 5, 5}, // Should use default for invalid
		{"whitespace", " 42 ", 5, 5},  // Should use default for invalid
		{"hex value", "0x42", 5, 5},   // Should use default for invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Setenv("TEST_INT_EDGE", tt.envValue))
			defer func() {
				if err := os.Unsetenv("TEST_INT_EDGE"); err != nil {
					t.Logf("Failed to unset TEST_INT_EDGE: %v", err)
				}
			}()

			result := getIntEnv("TEST_INT_EDGE", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringEnvEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{"whitespace value", "  spaces  ", "default", "  spaces  "},
		{"special characters", "!@#$%^&*()", "default", "!@#$%^&*()"},
		{"unicode", "テスト", "default", "テスト"},
		{"newlines", "line1\nline2", "default", "line1\nline2"},
		{"empty string", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Setenv("TEST_STRING_EDGE", tt.envValue))
			defer func() {
				if err := os.Unsetenv("TEST_STRING_EDGE"); err != nil {
					t.Logf("Failed to unset TEST_STRING_EDGE: %v", err)
				}
			}()

			result := getStringEnv("TEST_STRING_EDGE", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFindEnvFileErrors tests error conditions in findEnvFile
func TestFindEnvFileErrors(t *testing.T) {
	// Test when we can't get current working directory
	// This is hard to test directly, but we can test the search logic

	// Create temp directory structure without .github/.env.shared
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Should return error when no .env.shared found
	_, err = findEnvFile()
	assert.Error(t, err)
}

// TestLoadIntegrationWithRealProject tests loading in a real project structure
func TestLoadIntegrationWithRealProject(t *testing.T) {
	// Create a realistic project structure
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create project structure
	projectDirs := []string{
		"cmd/myapp",
		"internal/pkg",
		"pkg/api",
		".github",
	}

	for _, dir := range projectDirs {
		err = os.MkdirAll(dir, 0o750)
		require.NoError(t, err)
	}

	// Create .env.shared file
	envContent := `# Project configuration
ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=debug
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=true
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
`
	envFile := filepath.Join(tmpDir, ".github", ".env.shared")
	err = os.WriteFile(envFile, []byte(envContent), 0o600)
	require.NoError(t, err)

	// Test loading from various subdirectories
	subDirs := []string{
		".",
		"cmd/myapp",
		"internal/pkg",
		"pkg/api",
	}

	for _, subDir := range subDirs {
		t.Run("from "+subDir, func(t *testing.T) {
			err = os.Chdir(filepath.Join(tmpDir, subDir))
			require.NoError(t, err)

			cfg, err := Load()
			require.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.True(t, cfg.Enabled)
			assert.Equal(t, "debug", cfg.LogLevel)

			// Directory should always point to .github/pre-commit
			// The behavior depends on whether we're in the current directory or subdirectory
			if subDir == "." {
				// When in root directory, env file is found as ".github/.env.shared"
				// so Directory becomes ".github/pre-commit"
				assert.Equal(t, ".github/pre-commit", cfg.Directory)
			} else {
				// When in subdirectory, env file is found with absolute path
				// so Directory becomes absolute path
				expectedDir := filepath.Join(tmpDir, ".github", "pre-commit")
				// Use EvalSymlinks to handle macOS /var vs /private/var differences
				// Only resolve if the path exists, otherwise compare parent directories
				if _, err := os.Stat(expectedDir); err == nil {
					expectedDirResolved, err := filepath.EvalSymlinks(expectedDir)
					require.NoError(t, err)
					actualDirResolved, err := filepath.EvalSymlinks(cfg.Directory)
					require.NoError(t, err)
					assert.Equal(t, expectedDirResolved, actualDirResolved)
				} else {
					// Compare parent directories since pre-commit dir doesn't exist
					expectedParent := filepath.Dir(expectedDir)
					actualParent := filepath.Dir(cfg.Directory)
					expectedParentResolved, err := filepath.EvalSymlinks(expectedParent)
					require.NoError(t, err)
					actualParentResolved, err := filepath.EvalSymlinks(actualParent)
					require.NoError(t, err)
					assert.Equal(t, expectedParentResolved, actualParentResolved)
					assert.Equal(t, "pre-commit", filepath.Base(cfg.Directory))
				}
			}
		})
	}
}
