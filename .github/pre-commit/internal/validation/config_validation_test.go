package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
)

// ConfigValidationTestSuite validates configuration loading under various scenarios
type ConfigValidationTestSuite struct {
	suite.Suite
	tempDir     string
	originalWD  string
	originalEnv map[string]string
}

// SetupSuite initializes the test environment
func (s *ConfigValidationTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	require.NoError(s.T(), err)

	// Create temporary directory structure
	s.tempDir = s.T().TempDir()

	// Store original environment variables that might affect tests
	s.originalEnv = make(map[string]string)
	envVarsToSave := []string{
		"ENABLE_PRE_COMMIT_SYSTEM", "PRE_COMMIT_SYSTEM_LOG_LEVEL",
		"PRE_COMMIT_SYSTEM_ENABLE_FUMPT", "PRE_COMMIT_SYSTEM_ENABLE_LINT",
		"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS", "NO_COLOR", "CI",
	}

	for _, envVar := range envVarsToSave {
		s.originalEnv[envVar] = os.Getenv(envVar)
	}

	// Change to temp directory for tests
	require.NoError(s.T(), os.Chdir(s.tempDir))
}

// TearDownSuite cleans up the test environment
func (s *ConfigValidationTestSuite) TearDownSuite() {
	// Restore original working directory
	_ = os.Chdir(s.originalWD)

	// Restore original environment variables
	for envVar, originalValue := range s.originalEnv {
		if originalValue == "" {
			_ = os.Unsetenv(envVar)
		} else {
			_ = os.Setenv(envVar, originalValue)
		}
	}
}

// TearDownTest cleans up after each test
func (s *ConfigValidationTestSuite) TearDownTest() {
	// Clean up any environment variables set during the test
	envVarsToClean := []string{
		"ENABLE_PRE_COMMIT_SYSTEM", "PRE_COMMIT_SYSTEM_LOG_LEVEL",
		"PRE_COMMIT_SYSTEM_ENABLE_FUMPT", "PRE_COMMIT_SYSTEM_ENABLE_LINT",
		"PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS", "NO_COLOR", "CI",
		"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS", "PRE_COMMIT_SYSTEM_FAIL_FAST",
	}

	for _, envVar := range envVarsToClean {
		if originalValue, exists := s.originalEnv[envVar]; exists {
			if originalValue == "" {
				_ = os.Unsetenv(envVar)
			} else {
				_ = os.Setenv(envVar, originalValue)
			}
		} else {
			_ = os.Unsetenv(envVar)
		}
	}
}

// TestMissingConfigFile validates behavior when .env.shared is missing
func (s *ConfigValidationTestSuite) TestMissingConfigFile() {
	// Ensure no .env.shared file exists
	_, err := config.Load()

	// Should return a specific error about missing env file
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.Is(err, errors.ErrEnvFileNotFound) ||
		strings.Contains(err.Error(), ".env.shared"))
}

// TestInvalidConfigFile validates behavior with malformed config files
func (s *ConfigValidationTestSuite) TestInvalidConfigFile() {
	testCases := []struct {
		name        string
		content     string
		shouldError bool
		description string
	}{
		{
			name:        "Empty File",
			content:     "",
			shouldError: false, // Should use defaults
			description: "Empty configuration file should use defaults",
		},
		{
			name:        "Invalid Boolean",
			content:     "ENABLE_PRE_COMMIT_SYSTEM=maybe\n",
			shouldError: false, // Should use default for invalid boolean
			description: "Invalid boolean should fall back to default",
		},
		{
			name:        "Invalid Integer",
			content:     "PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=not-a-number\n",
			shouldError: false, // Should use default for invalid integer
			description: "Invalid integer should fall back to default",
		},
		{
			name: "Invalid Timeout Values",
			content: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=0
PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT=-1
`,
			shouldError: true, // Validation should catch invalid timeouts
			description: "Zero or negative timeouts should cause validation error",
		},
		{
			name: "Invalid Log Level",
			content: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=invalid
`,
			shouldError: true, // Validation should catch invalid log level
			description: "Invalid log level should cause validation error",
		},
		{
			name: "Malformed Environment Variables",
			content: `ENABLE_PRE_COMMIT_SYSTEM=true
=invalid_line_no_key
KEY_WITHOUT_VALUE
VALID_KEY=valid_value
`,
			shouldError: false, // godotenv should handle malformed lines gracefully
			description: "Malformed lines should be ignored, valid ones processed",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create .github directory
			githubDir := filepath.Join(s.tempDir, ".github")
			require.NoError(s.T(), os.MkdirAll(githubDir, 0o755))

			// Create .env.shared file with test content
			envFile := filepath.Join(githubDir, ".env.shared")
			require.NoError(s.T(), os.WriteFile(envFile, []byte(tc.content), 0o644))

			// Try to load configuration
			cfg, err := config.Load()

			if tc.shouldError {
				assert.Error(s.T(), err, tc.description)
			} else {
				assert.NoError(s.T(), err, tc.description)
				assert.NotNil(s.T(), cfg, "Configuration should be loaded successfully")
			}

			// Clean up for next test
			require.NoError(s.T(), os.RemoveAll(githubDir))
		})
	}
}

// TestEnvironmentVariablePrecedence validates environment variable precedence
func (s *ConfigValidationTestSuite) TestEnvironmentVariablePrecedence() {
	// Create base .env.shared file
	githubDir := filepath.Join(s.tempDir, ".github")
	require.NoError(s.T(), os.MkdirAll(githubDir, 0o755))

	envFile := filepath.Join(githubDir, ".env.shared")
	baseConfig := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=info
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=2
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
`
	require.NoError(s.T(), os.WriteFile(envFile, []byte(baseConfig), 0o644))

	testCases := []struct {
		name            string
		envOverrides    map[string]string
		expectedLog     string
		expectedWorkers int
		expectedFumpt   bool
		description     string
	}{
		{
			name: "Environment Variables Override File",
			envOverrides: map[string]string{
				"PRE_COMMIT_SYSTEM_LOG_LEVEL":        "debug",
				"PRE_COMMIT_SYSTEM_PARALLEL_WORKERS": "4",
				"PRE_COMMIT_SYSTEM_ENABLE_FUMPT":     "false",
			},
			expectedLog:     "debug",
			expectedWorkers: 4,
			expectedFumpt:   false,
			description:     "Environment variables should take precedence over file values",
		},
		{
			name: "Partial Override",
			envOverrides: map[string]string{
				"PRE_COMMIT_SYSTEM_LOG_LEVEL": "warn",
			},
			expectedLog:     "warn",
			expectedWorkers: 2,    // From file
			expectedFumpt:   true, // From file
			description:     "Only overridden values should change, others use file values",
		},
		{
			name:            "No Override",
			envOverrides:    map[string]string{},
			expectedLog:     "info",
			expectedWorkers: 2,
			expectedFumpt:   true,
			description:     "Without overrides, should use file values",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Set environment variable overrides
			for key, value := range tc.envOverrides {
				require.NoError(s.T(), os.Setenv(key, value))
			}

			// Load configuration
			cfg, err := config.Load()
			require.NoError(s.T(), err, tc.description)

			// Validate expected values
			assert.Equal(s.T(), tc.expectedLog, cfg.LogLevel, "Log level should match expected")
			assert.Equal(s.T(), tc.expectedWorkers, cfg.Performance.ParallelWorkers, "Workers should match expected")
			assert.Equal(s.T(), tc.expectedFumpt, cfg.Checks.Fumpt, "Fumpt setting should match expected")

			// Clean up environment variables
			for key := range tc.envOverrides {
				require.NoError(s.T(), os.Unsetenv(key))
			}
		})
	}
}

// TestConfigurationDefaults validates default values are properly set
func (s *ConfigValidationTestSuite) TestConfigurationDefaults() {
	// Create minimal .env.shared file with only required setting
	githubDir := filepath.Join(s.tempDir, ".github")
	require.NoError(s.T(), os.MkdirAll(githubDir, 0o755))

	envFile := filepath.Join(githubDir, ".env.shared")
	minimalConfig := `ENABLE_PRE_COMMIT_SYSTEM=true
`
	require.NoError(s.T(), os.WriteFile(envFile, []byte(minimalConfig), 0o644))

	cfg, err := config.Load()
	require.NoError(s.T(), err, "Minimal configuration should load successfully")

	// Validate all defaults
	expectedDefaults := map[string]interface{}{
		"LogLevel":                    "info",
		"MaxFileSize":                 int64(10 * 1024 * 1024), // 10MB
		"MaxFilesOpen":                100,
		"Timeout":                     300,
		"Checks.Fumpt":                true,
		"Checks.Lint":                 true,
		"Checks.ModTidy":              true,
		"Checks.Whitespace":           true,
		"Checks.EOF":                  true,
		"ToolVersions.Fumpt":          "latest",
		"ToolVersions.GolangciLint":   "latest",
		"Performance.ParallelWorkers": 0, // Auto
		"Performance.FailFast":        false,
		"CheckTimeouts.Fumpt":         30,
		"CheckTimeouts.Lint":          60,
		"CheckTimeouts.ModTidy":       30,
		"CheckTimeouts.Whitespace":    30,
		"CheckTimeouts.EOF":           30,
		"Git.HooksPath":               ".git/hooks",
		"UI.ColorOutput":              true,
	}

	// Use reflection-like validation for nested structures
	assert.Equal(s.T(), expectedDefaults["LogLevel"], cfg.LogLevel)
	assert.Equal(s.T(), expectedDefaults["MaxFileSize"], cfg.MaxFileSize)
	assert.Equal(s.T(), expectedDefaults["MaxFilesOpen"], cfg.MaxFilesOpen)
	assert.Equal(s.T(), expectedDefaults["Timeout"], cfg.Timeout)

	assert.Equal(s.T(), expectedDefaults["Checks.Fumpt"], cfg.Checks.Fumpt)
	assert.Equal(s.T(), expectedDefaults["Checks.Lint"], cfg.Checks.Lint)
	assert.Equal(s.T(), expectedDefaults["Checks.ModTidy"], cfg.Checks.ModTidy)
	assert.Equal(s.T(), expectedDefaults["Checks.Whitespace"], cfg.Checks.Whitespace)
	assert.Equal(s.T(), expectedDefaults["Checks.EOF"], cfg.Checks.EOF)

	assert.Equal(s.T(), expectedDefaults["ToolVersions.Fumpt"], cfg.ToolVersions.Fumpt)
	assert.Equal(s.T(), expectedDefaults["ToolVersions.GolangciLint"], cfg.ToolVersions.GolangciLint)

	assert.Equal(s.T(), expectedDefaults["Performance.ParallelWorkers"], cfg.Performance.ParallelWorkers)
	assert.Equal(s.T(), expectedDefaults["Performance.FailFast"], cfg.Performance.FailFast)

	assert.Equal(s.T(), expectedDefaults["CheckTimeouts.Fumpt"], cfg.CheckTimeouts.Fumpt)
	assert.Equal(s.T(), expectedDefaults["CheckTimeouts.Lint"], cfg.CheckTimeouts.Lint)
	assert.Equal(s.T(), expectedDefaults["CheckTimeouts.ModTidy"], cfg.CheckTimeouts.ModTidy)
	assert.Equal(s.T(), expectedDefaults["CheckTimeouts.Whitespace"], cfg.CheckTimeouts.Whitespace)
	assert.Equal(s.T(), expectedDefaults["CheckTimeouts.EOF"], cfg.CheckTimeouts.EOF)

	assert.Equal(s.T(), expectedDefaults["Git.HooksPath"], cfg.Git.HooksPath)
	assert.Equal(s.T(), expectedDefaults["UI.ColorOutput"], cfg.UI.ColorOutput)

	// Validate exclude patterns default
	expectedExcludes := []string{"vendor/", "node_modules/", ".git/"}
	assert.Equal(s.T(), expectedExcludes, cfg.Git.ExcludePatterns)
}

// TestConfigurationValidation validates the validation logic
func (s *ConfigValidationTestSuite) TestConfigurationValidation() {
	testCases := []struct {
		name          string
		config        string
		shouldError   bool
		expectedError string
		description   string
	}{
		{
			name: "Valid Configuration",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=debug
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=60
PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT=30
PRE_COMMIT_SYSTEM_LINT_TIMEOUT=120
PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT=30
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=15
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=15
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=50
PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=200
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=4
`,
			shouldError: false,
			description: "Valid configuration should pass validation",
		},
		{
			name: "Zero Timeout",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=0
`,
			shouldError:   true,
			expectedError: "TIMEOUT_SECONDS must be greater than 0",
			description:   "Zero timeout should fail validation",
		},
		{
			name: "Negative File Size",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=-1
`,
			shouldError:   true,
			expectedError: "MAX_FILE_SIZE_MB must be greater than 0",
			description:   "Negative file size should fail validation",
		},
		{
			name: "Invalid Log Level",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=trace
`,
			shouldError:   true,
			expectedError: "LOG_LEVEL must be one of",
			description:   "Invalid log level should fail validation",
		},
		{
			name: "Invalid Tool Version",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_FUMPT_VERSION=invalid-version
`,
			shouldError:   true,
			expectedError: "FUMPT_VERSION must be 'latest' or a valid version",
			description:   "Invalid tool version should fail validation",
		},
		{
			name: "Negative Parallel Workers",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=-1
`,
			shouldError:   true,
			expectedError: "PARALLEL_WORKERS must be 0 (auto) or positive",
			description:   "Negative parallel workers should fail validation",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create .github directory
			githubDir := filepath.Join(s.tempDir, ".github")
			require.NoError(s.T(), os.MkdirAll(githubDir, 0o755))

			// Create .env.shared file with test content
			envFile := filepath.Join(githubDir, ".env.shared")
			require.NoError(s.T(), os.WriteFile(envFile, []byte(tc.config), 0o644))

			// Try to load configuration
			cfg, err := config.Load()

			if tc.shouldError {
				assert.Error(s.T(), err, tc.description)
				if tc.expectedError != "" {
					assert.Contains(s.T(), err.Error(), tc.expectedError,
						"Error should contain expected message")
				}
				assert.Nil(s.T(), cfg, "Configuration should be nil on validation error")
			} else {
				assert.NoError(s.T(), err, tc.description)
				assert.NotNil(s.T(), cfg, "Configuration should be loaded successfully")
			}

			// Clean up for next test
			require.NoError(s.T(), os.RemoveAll(githubDir))
		})
	}
}

// TestPartialConfiguration validates behavior with partial configurations
func (s *ConfigValidationTestSuite) TestPartialConfiguration() {
	testCases := []struct {
		name        string
		config      string
		description string
	}{
		{
			name: "Only Checks Enabled",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=false
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
`,
			description: "Configuration with only check settings should work",
		},
		{
			name: "Only Performance Settings",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=1
PRE_COMMIT_SYSTEM_FAIL_FAST=true
`,
			description: "Configuration with only performance settings should work",
		},
		{
			name: "Only Timeout Settings",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT=45
PRE_COMMIT_SYSTEM_LINT_TIMEOUT=90
`,
			description: "Configuration with only timeout settings should work",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create .github directory
			githubDir := filepath.Join(s.tempDir, ".github")
			require.NoError(s.T(), os.MkdirAll(githubDir, 0o755))

			// Create .env.shared file with test content
			envFile := filepath.Join(githubDir, ".env.shared")
			require.NoError(s.T(), os.WriteFile(envFile, []byte(tc.config), 0o644))

			// Load configuration
			cfg, err := config.Load()
			assert.NoError(s.T(), err, tc.description)
			assert.NotNil(s.T(), cfg, "Configuration should be loaded successfully")
			assert.True(s.T(), cfg.Enabled, "System should be enabled")

			// Clean up for next test
			require.NoError(s.T(), os.RemoveAll(githubDir))
		})
	}
}

// TestConfigurationDirectoryDetection validates directory detection logic
func (s *ConfigValidationTestSuite) TestConfigurationDirectoryDetection() {
	// Create nested directory structure
	nestedDir := filepath.Join(s.tempDir, "deep", "nested", "directory")
	require.NoError(s.T(), os.MkdirAll(nestedDir, 0o755))

	// Create .github/.env.shared at the root
	githubDir := filepath.Join(s.tempDir, ".github")
	require.NoError(s.T(), os.MkdirAll(githubDir, 0o755))

	envFile := filepath.Join(githubDir, ".env.shared")
	testConfig := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=debug
`
	require.NoError(s.T(), os.WriteFile(envFile, []byte(testConfig), 0o644))

	// Change to nested directory
	require.NoError(s.T(), os.Chdir(nestedDir))

	// Configuration loading should find the .env.shared file by walking up
	cfg, err := config.Load()
	assert.NoError(s.T(), err, "Should find .env.shared file by walking up directories")
	assert.NotNil(s.T(), cfg, "Configuration should be loaded")
	assert.Equal(s.T(), "debug", cfg.LogLevel, "Should load correct configuration")

	// Change back to temp directory
	require.NoError(s.T(), os.Chdir(s.tempDir))
}

// TestConfigurationHelp validates the configuration help functionality
func (s *ConfigValidationTestSuite) TestConfigurationHelp() {
	help := config.GetConfigHelp()

	// Validate that help contains key information
	assert.Contains(s.T(), help, "ENABLE_PRE_COMMIT_SYSTEM", "Help should document main enable flag")
	assert.Contains(s.T(), help, "PRE_COMMIT_SYSTEM_LOG_LEVEL", "Help should document log level")
	assert.Contains(s.T(), help, "Example .github/.env.shared", "Help should include example")
	assert.Contains(s.T(), help, "Core Settings", "Help should have sections")
	assert.Contains(s.T(), help, "Check Configuration", "Help should document checks")
	assert.Contains(s.T(), help, "Performance Settings", "Help should document performance")

	// Should be non-empty and reasonably long
	assert.Greater(s.T(), len(help), 1000, "Help should be comprehensive")
}

// TestSuite runs the configuration validation test suite
func TestConfigValidationTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigValidationTestSuite))
}
