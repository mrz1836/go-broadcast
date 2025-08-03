// Package config provides configuration loading for the GoFortress pre-commit system
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
)

// Config holds the configuration for the pre-commit system
type Config struct {
	// Core settings
	Enabled      bool   // ENABLE_PRE_COMMIT_SYSTEM
	Directory    string // Directory containing pre-commit tools (derived)
	LogLevel     string // PRE_COMMIT_SYSTEM_LOG_LEVEL
	MaxFileSize  int64  // PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB
	MaxFilesOpen int    // PRE_COMMIT_SYSTEM_MAX_FILES_OPEN
	Timeout      int    // PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS

	// Check configurations
	Checks struct {
		Fumpt      bool // PRE_COMMIT_SYSTEM_ENABLE_FUMPT
		Lint       bool // PRE_COMMIT_SYSTEM_ENABLE_LINT
		ModTidy    bool // PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY
		Whitespace bool // PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE
		EOF        bool // PRE_COMMIT_SYSTEM_ENABLE_EOF
	}

	// Tool versions
	ToolVersions struct {
		Fumpt        string // PRE_COMMIT_SYSTEM_FUMPT_VERSION
		GolangciLint string // PRE_COMMIT_SYSTEM_GOLANGCI_LINT_VERSION
	}

	// Performance settings
	Performance struct {
		ParallelWorkers int  // PRE_COMMIT_SYSTEM_PARALLEL_WORKERS
		FailFast        bool // PRE_COMMIT_SYSTEM_FAIL_FAST
	}

	// Check timeouts (in seconds)
	CheckTimeouts struct {
		Fumpt      int // PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT (default: 30)
		Lint       int // PRE_COMMIT_SYSTEM_LINT_TIMEOUT (default: 60)
		ModTidy    int // PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT (default: 30)
		Whitespace int // PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT (default: 30)
		EOF        int // PRE_COMMIT_SYSTEM_EOF_TIMEOUT (default: 30)
	}

	// Git settings
	Git struct {
		HooksPath       string   // PRE_COMMIT_SYSTEM_HOOKS_PATH (default: .git/hooks)
		ExcludePatterns []string // PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS
	}

	// UI settings
	UI struct {
		ColorOutput bool // PRE_COMMIT_SYSTEM_COLOR_OUTPUT (default: true)
	}
}

// Load reads configuration from .github/.env.shared
func Load() (*Config, error) {
	// Find .env.shared file
	envPath, err := findEnvFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find .env.shared: %w", err)
	}

	// Load environment file
	if err := godotenv.Load(envPath); err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", envPath, err)
	}

	cfg := &Config{
		Directory: filepath.Dir(envPath) + "/pre-commit",
	}

	// Core settings
	cfg.Enabled = getBoolEnv("ENABLE_PRE_COMMIT_SYSTEM", false)
	cfg.LogLevel = getStringEnv("PRE_COMMIT_SYSTEM_LOG_LEVEL", "info")
	cfg.MaxFileSize = int64(getIntEnv("PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB", 10)) * 1024 * 1024
	cfg.MaxFilesOpen = getIntEnv("PRE_COMMIT_SYSTEM_MAX_FILES_OPEN", 100)
	cfg.Timeout = getIntEnv("PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS", 300)

	// Check configurations
	cfg.Checks.Fumpt = getBoolEnv("PRE_COMMIT_SYSTEM_ENABLE_FUMPT", true)
	cfg.Checks.Lint = getBoolEnv("PRE_COMMIT_SYSTEM_ENABLE_LINT", true)
	cfg.Checks.ModTidy = getBoolEnv("PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY", true)
	cfg.Checks.Whitespace = getBoolEnv("PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE", true)
	cfg.Checks.EOF = getBoolEnv("PRE_COMMIT_SYSTEM_ENABLE_EOF", true)

	// Tool versions
	cfg.ToolVersions.Fumpt = getStringEnv("PRE_COMMIT_SYSTEM_FUMPT_VERSION", "latest")
	cfg.ToolVersions.GolangciLint = getStringEnv("PRE_COMMIT_SYSTEM_GOLANGCI_LINT_VERSION", "latest")

	// Performance settings
	cfg.Performance.ParallelWorkers = getIntEnv("PRE_COMMIT_SYSTEM_PARALLEL_WORKERS", 0) // 0 = auto
	cfg.Performance.FailFast = getBoolEnv("PRE_COMMIT_SYSTEM_FAIL_FAST", false)

	// Check timeouts
	cfg.CheckTimeouts.Fumpt = getIntEnv("PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT", 30)
	cfg.CheckTimeouts.Lint = getIntEnv("PRE_COMMIT_SYSTEM_LINT_TIMEOUT", 60)
	cfg.CheckTimeouts.ModTidy = getIntEnv("PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT", 30)
	cfg.CheckTimeouts.Whitespace = getIntEnv("PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT", 30)
	cfg.CheckTimeouts.EOF = getIntEnv("PRE_COMMIT_SYSTEM_EOF_TIMEOUT", 30)

	// Git settings
	cfg.Git.HooksPath = getStringEnv("PRE_COMMIT_SYSTEM_HOOKS_PATH", ".git/hooks")
	excludes := getStringEnv("PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS", "vendor/,node_modules/,.git/")
	if excludes != "" {
		cfg.Git.ExcludePatterns = strings.Split(excludes, ",")
		for i := range cfg.Git.ExcludePatterns {
			cfg.Git.ExcludePatterns[i] = strings.TrimSpace(cfg.Git.ExcludePatterns[i])
		}
	}

	// UI settings
	cfg.UI.ColorOutput = getBoolEnv("PRE_COMMIT_SYSTEM_COLOR_OUTPUT", true)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration and provides helpful error messages
func (c *Config) Validate() error {
	var errors []string

	// Validate timeouts
	if c.Timeout <= 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS must be greater than 0")
	}

	if c.CheckTimeouts.Fumpt <= 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.Lint <= 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_LINT_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.ModTidy <= 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.Whitespace <= 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.EOF <= 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_EOF_TIMEOUT must be greater than 0")
	}

	// Validate file size limits
	if c.MaxFileSize <= 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB must be greater than 0")
	}

	if c.MaxFilesOpen <= 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_MAX_FILES_OPEN must be greater than 0")
	}

	// Validate performance settings
	if c.Performance.ParallelWorkers < 0 {
		errors = append(errors, "PRE_COMMIT_SYSTEM_PARALLEL_WORKERS must be 0 (auto) or positive")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		errors = append(errors, "PRE_COMMIT_SYSTEM_LOG_LEVEL must be one of: debug, info, warn, error")
	}

	// Validate directory exists (skip in test environments)
	if c.Directory != "" && !isTestEnvironment() {
		if _, err := os.Stat(c.Directory); os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("pre-commit directory does not exist: %s", c.Directory))
		}
	}

	// Validate tool versions
	if c.ToolVersions.Fumpt != "" && c.ToolVersions.Fumpt != "latest" {
		if !isValidVersion(c.ToolVersions.Fumpt) {
			errors = append(errors, "PRE_COMMIT_SYSTEM_FUMPT_VERSION must be 'latest' or a valid version (e.g., v0.6.0)")
		}
	}

	if c.ToolVersions.GolangciLint != "" && c.ToolVersions.GolangciLint != "latest" {
		if !isValidVersion(c.ToolVersions.GolangciLint) {
			errors = append(errors, "PRE_COMMIT_SYSTEM_GOLANGCI_LINT_VERSION must be 'latest' or a valid version (e.g., v1.55.2)")
		}
	}

	// Validate exclude patterns
	for i, pattern := range c.Git.ExcludePatterns {
		if strings.TrimSpace(pattern) == "" {
			errors = append(errors, fmt.Sprintf("exclude pattern at index %d is empty", i))
		}
	}

	if len(errors) > 0 {
		return &ConfigValidationError{
			Errors: errors,
		}
	}

	return nil
}

// ConfigValidationError represents configuration validation errors
type ConfigValidationError struct {
	Errors []string
}

// Error implements the error interface
func (e *ConfigValidationError) Error() string {
	return fmt.Sprintf("configuration validation failed:\n  - %s", strings.Join(e.Errors, "\n  - "))
}

// isValidVersion checks if a version string is valid
func isValidVersion(version string) bool {
	// Simple validation for semantic versioning
	if strings.HasPrefix(version, "v") && len(version) > 1 {
		// Remove 'v' prefix and check basic format
		version = version[1:]
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return true
		}
	}
	return false
}

// isTestEnvironment checks if we're running in a test environment
func isTestEnvironment() bool {
	// Check if we're running under go test
	return strings.HasSuffix(os.Args[0], ".test") ||
		strings.Contains(os.Args[0], "/_test/") ||
		strings.Contains(os.Args[0], "\\test\\") ||
		os.Getenv("GO_TESTING") == "true"
}

// GetConfigHelp returns helpful information about configuration options
func GetConfigHelp() string {
	return `GoFortress Pre-commit System Configuration Help

Environment Variables:

Core Settings:
  ENABLE_PRE_COMMIT_SYSTEM=true/false          Enable/disable the pre-commit system
  PRE_COMMIT_SYSTEM_LOG_LEVEL=info             Log level (debug, info, warn, error)
  PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=10         Maximum file size to process (MB)
  PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=100          Maximum files to keep open
  PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=300         Global timeout in seconds

Check Configuration:
  PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true           Enable gofumpt formatting
  PRE_COMMIT_SYSTEM_ENABLE_LINT=true            Enable golangci-lint
  PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true        Enable go mod tidy
  PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true      Enable whitespace check
  PRE_COMMIT_SYSTEM_ENABLE_EOF=true             Enable EOF newline check

Tool Versions:
  PRE_COMMIT_SYSTEM_FUMPT_VERSION=latest        gofumpt version
  PRE_COMMIT_SYSTEM_GOLANGCI_LINT_VERSION=latest  golangci-lint version

Performance Settings:
  PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=0          Parallel workers (0=auto)
  PRE_COMMIT_SYSTEM_FAIL_FAST=false             Stop on first failure

Check Timeouts (seconds):
  PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT=30            gofumpt timeout
  PRE_COMMIT_SYSTEM_LINT_TIMEOUT=60             golangci-lint timeout
  PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT=30         go mod tidy timeout
  PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=30       whitespace check timeout
  PRE_COMMIT_SYSTEM_EOF_TIMEOUT=30              EOF check timeout

Git Settings:
  PRE_COMMIT_SYSTEM_HOOKS_PATH=.git/hooks       Git hooks directory
  PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/"  Exclude patterns

UI Settings:
  PRE_COMMIT_SYSTEM_COLOR_OUTPUT=true           Enable colored output

Example .github/.env.shared:
  # Enable the system
  ENABLE_PRE_COMMIT_SYSTEM=true

  # Configure checks
  PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
  PRE_COMMIT_SYSTEM_ENABLE_LINT=true
  PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true

  # Set timeouts
  PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT=30
  PRE_COMMIT_SYSTEM_LINT_TIMEOUT=120

  # Exclude patterns
  PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/,*.tmp,*.log"
`
}

// findEnvFile locates the .github/.env.shared file
func findEnvFile() (string, error) {
	// First, check if we're already in the right place
	if _, err := os.Stat(".github/.env.shared"); err == nil {
		return ".github/.env.shared", nil
	}

	// Walk up the directory tree looking for .github/.env.shared
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		envPath := filepath.Join(cwd, ".github", ".env.shared")
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// Reached root
			break
		}
		cwd = parent
	}

	return "", prerrors.ErrEnvFileNotFound
}

// Helper functions for environment variable parsing
func getBoolEnv(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return b
}

func getIntEnv(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

func getStringEnv(key string, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
