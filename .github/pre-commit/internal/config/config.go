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

	// Git settings
	Git struct {
		HooksPath       string   // PRE_COMMIT_SYSTEM_HOOKS_PATH (default: .git/hooks)
		ExcludePatterns []string // PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS
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

	// Git settings
	cfg.Git.HooksPath = getStringEnv("PRE_COMMIT_SYSTEM_HOOKS_PATH", ".git/hooks")
	excludes := getStringEnv("PRE_COMMIT_SYSTEM_EXCLUDE_PATTERNS", "vendor/,node_modules/,.git/")
	if excludes != "" {
		cfg.Git.ExcludePatterns = strings.Split(excludes, ",")
		for i := range cfg.Git.ExcludePatterns {
			cfg.Git.ExcludePatterns[i] = strings.TrimSpace(cfg.Git.ExcludePatterns[i])
		}
	}

	return cfg, nil
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
