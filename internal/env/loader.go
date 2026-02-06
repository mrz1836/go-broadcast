// Package env provides utilities for loading environment variables from .env files.
// It follows the GoFortress pattern used by other tools in the ecosystem.
package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Static errors for consistent error handling
var (
	ErrEnvBaseFileNotFound = errors.New("required .env.base file not found")
	ErrNotDirectory        = errors.New("path is not a directory")
	ErrNoEnvFiles          = errors.New("no .env files found in directory")
)

// LoadEnvFiles loads environment variables with proper precedence:
//  1. OS environment variables (HIGHEST - never overwritten)
//  2. Modular .github/env/*.env files (preferred, last-wins ordering)
//  3. Legacy .env.custom (project overrides) + .env.base (defaults)
//
// Variables already set in the OS environment are NEVER overwritten.
// This allows users to override via shell exports or CI/CD secrets.
//
// If a modular .github/env/ directory with .env files exists, it is used.
// Otherwise, falls back to the legacy .env.base + .env.custom pattern.
//
// This ensures go-broadcast follows the same configuration pattern as
// go-coverage, go-pre-commit, mage-x, and other GoFortress tools.
func LoadEnvFiles() error {
	// Try modular mode first (preferred)
	if envDir := findEnvDir(); envDir != "" {
		return LoadEnvDir(envDir, isCI())
	}
	// Fall back to legacy mode
	return loadEnvFilesInternal(".github/.env.base", ".github/.env.custom")
}

// LoadEnvFilesFromDir loads environment files from a specific directory.
// This is useful for testing or when running from a different working directory.
// Prefers modular .github/env/ if available, falls back to legacy files.
func LoadEnvFilesFromDir(dir string) error {
	// Try modular mode first (preferred)
	if envDir := findEnvDirFromBase(dir); envDir != "" {
		return LoadEnvDir(envDir, isCI())
	}
	// Fall back to legacy mode
	baseFile := filepath.Join(dir, ".github", ".env.base")
	customFile := filepath.Join(dir, ".github", ".env.custom")
	return loadEnvFilesInternal(baseFile, customFile)
}

// LoadEnvDir loads all .env files from a directory in lexicographic order.
// Files are merged with overload semantics (later files override earlier ones).
// After merging, variables are only set if not already present in the OS environment.
//
// When skipLocal is true, 99-local.env is skipped (intended for CI environments
// where local developer overrides should not apply).
func LoadEnvDir(dirPath string, skipLocal bool) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("failed to access env directory %s: %w", dirPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotDirectory, dirPath)
	}

	matches, err := filepath.Glob(filepath.Join(dirPath, "*.env"))
	if err != nil {
		return fmt.Errorf("failed to glob env files in %s: %w", dirPath, err)
	}

	sort.Strings(matches)

	merged := make(map[string]string)
	loaded := 0
	for _, file := range matches {
		if skipLocal && filepath.Base(file) == "99-local.env" {
			continue
		}
		if err := loadAndApplyEnvFile(file, merged); err != nil {
			return fmt.Errorf("failed to load %s: %w", file, err)
		}
		loaded++
	}

	if loaded == 0 {
		return fmt.Errorf("%w: %s", ErrNoEnvFiles, dirPath)
	}

	// Apply merged vars (OS env wins)
	for key, value := range merged {
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set %s: %w", key, err)
			}
		}
	}

	return nil
}

// isCI returns true when running in a CI environment.
func isCI() bool {
	return os.Getenv("CI") == "true"
}

// findEnvDir looks for a .github/env/ directory relative to the current working
// directory that contains at least one .env file.
func findEnvDir() string {
	dir := filepath.Join(".github", "env")
	if hasEnvFiles(dir) {
		return dir
	}
	return ""
}

// findEnvDirFromBase looks for a .github/env/ directory relative to the given
// base directory that contains at least one .env file.
func findEnvDirFromBase(baseDir string) string {
	dir := filepath.Join(baseDir, ".github", "env")
	if hasEnvFiles(dir) {
		return dir
	}
	return ""
}

// hasEnvFiles returns true if dirPath is a directory containing at least one .env file.
func hasEnvFiles(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if err != nil || !info.IsDir() {
		return false
	}
	matches, err := filepath.Glob(filepath.Join(dirPath, "*.env"))
	if err != nil {
		return false
	}
	return len(matches) > 0
}

// loadEnvFilesInternal is the shared implementation for loading env files.
func loadEnvFilesInternal(baseFile, customFile string) error {
	// Parse base configuration (required file)
	// We open the file directly instead of checking with Stat first to avoid
	// a TOCTOU (time-of-check-to-time-of-use) race condition.
	baseVars, err := parseEnvFile(baseFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w at %s", ErrEnvBaseFileNotFound, baseFile)
		}
		return fmt.Errorf("failed to parse %s: %w", baseFile, err)
	}

	// Parse custom overrides (optional file)
	// We try to parse directly and ignore NotExist errors to avoid TOCTOU races.
	customVars, err := parseEnvFile(customFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to parse %s: %w", customFile, err)
		}
		// Custom file doesn't exist - that's fine, it's optional
		customVars = make(map[string]string)
	}

	// Merge: custom overrides base
	merged := make(map[string]string)
	for k, v := range baseVars {
		merged[k] = v
	}
	for k, v := range customVars {
		merged[k] = v // custom wins over base
	}

	// Apply: ONLY set if not already in OS environment
	// This ensures OS env vars (from shell, CI/CD secrets, etc.) take precedence
	// We use LookupEnv to distinguish between "not set" and "set to empty string".
	// An explicitly set empty string (e.g., FOO="") is preserved and NOT overwritten.
	for key, value := range merged {
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set %s: %w", key, err)
			}
		}
		// If OS env is already set (even to empty), we respect it (don't overwrite)
	}

	return nil
}

// GetEnvWithFallback gets an environment variable with a fallback value.
// This is useful for configuration values that should have sensible defaults.
//
// This function returns the fallback if the env var is not set OR if it
// is set to an empty string. If you need to distinguish between "not set" and
// "set to empty", use GetEnvOrDefault instead.
func GetEnvWithFallback(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// GetEnvOrDefault gets an environment variable with a default value.
// Unlike GetEnvWithFallback, this function preserves explicitly set empty values.
// The default is only used when the variable is truly not set in the environment.
func GetEnvOrDefault(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
