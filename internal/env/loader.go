// Package env provides utilities for loading environment variables from .env files.
// It follows the GoFortress pattern used by other tools in the ecosystem.
package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Static errors for consistent error handling
var (
	ErrEnvBaseFileNotFound = errors.New("required .env.base file not found")
)

// LoadEnvFiles loads environment variables with proper precedence:
//  1. OS environment variables (HIGHEST - never overwritten)
//  2. .env.custom (project overrides)
//  3. .env.base (defaults)
//
// Variables already set in the OS environment are NEVER overwritten.
// This allows users to override via shell exports or CI/CD secrets.
//
// This ensures go-broadcast follows the same configuration pattern as
// go-coverage, go-pre-commit, mage-x, and other GoFortress tools.
func LoadEnvFiles() error {
	// Define file paths following GoFortress convention
	baseFile := ".github/.env.base"
	customFile := ".github/.env.custom"

	return loadEnvFilesInternal(baseFile, customFile)
}

// LoadEnvFilesFromDir loads environment files from a specific directory.
// This is useful for testing or when running from a different working directory.
func LoadEnvFilesFromDir(dir string) error {
	baseFile := filepath.Join(dir, ".github", ".env.base")
	customFile := filepath.Join(dir, ".github", ".env.custom")

	return loadEnvFilesInternal(baseFile, customFile)
}

// loadEnvFilesInternal is the shared implementation for loading env files.
func loadEnvFilesInternal(baseFile, customFile string) error {
	// Check base file exists (required)
	if _, err := os.Stat(baseFile); os.IsNotExist(err) {
		return fmt.Errorf("%w at %s", ErrEnvBaseFileNotFound, baseFile)
	}

	// Parse base configuration
	baseVars, err := parseEnvFile(baseFile)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", baseFile, err)
	}

	// Parse custom overrides (optional)
	customVars := make(map[string]string)
	if _, err := os.Stat(customFile); err == nil {
		customVars, err = parseEnvFile(customFile)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", customFile, err)
		}
	}
	// If custom file doesn't exist, that's fine - it's optional

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
	for key, value := range merged {
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("failed to set %s: %w", key, err)
			}
		}
		// If OS env is already set, we respect it (don't overwrite)
	}

	return nil
}

// GetEnvWithFallback gets an environment variable with a fallback value.
// This is useful for configuration values that should have sensible defaults.
func GetEnvWithFallback(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
