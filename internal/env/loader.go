// Package env provides utilities for loading environment variables from .env files.
// It follows the GoFortress pattern used by other tools in the ecosystem.
package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Static errors for consistent error handling
var (
	ErrEnvBaseFileNotFound = errors.New("required .env.base file not found")
)

// LoadEnvFiles loads environment variables from .env.base and .env.custom files
// following the GoFortress pattern used by other tools in the ecosystem.
//
// Loading Strategy:
// 1. Load .github/.env.base (required) - contains default configuration
// 2. Load .github/.env.custom (optional) - project-specific overrides
// 3. Merge with custom values taking precedence over base values
//
// This ensures go-broadcast follows the same configuration pattern as
// go-coverage, go-pre-commit, mage-x, and other GoFortress tools.
func LoadEnvFiles() error {
	// Define file paths following GoFortress convention
	baseFile := ".github/.env.base"
	customFile := ".github/.env.custom"

	// Load base configuration (required)
	if _, err := os.Stat(baseFile); os.IsNotExist(err) {
		return fmt.Errorf("%w at %s", ErrEnvBaseFileNotFound, baseFile)
	}

	// Load base environment variables
	if err := godotenv.Overload(baseFile); err != nil {
		return fmt.Errorf("failed to load %s: %w", baseFile, err)
	}

	// Load custom overrides (optional)
	if _, err := os.Stat(customFile); err == nil {
		// Custom file exists, load it to override base values
		if err := godotenv.Overload(customFile); err != nil {
			return fmt.Errorf("failed to load %s: %w", customFile, err)
		}
	}
	// If custom file doesn't exist, that's fine - it's optional

	return nil
}

// LoadEnvFilesFromDir loads environment files from a specific directory.
// This is useful for testing or when running from a different working directory.
func LoadEnvFilesFromDir(dir string) error {
	baseFile := filepath.Join(dir, ".github", ".env.base")
	customFile := filepath.Join(dir, ".github", ".env.custom")

	// Load base configuration (required)
	if _, err := os.Stat(baseFile); os.IsNotExist(err) {
		return fmt.Errorf("%w at %s", ErrEnvBaseFileNotFound, baseFile)
	}

	// Load base environment variables
	if err := godotenv.Overload(baseFile); err != nil {
		return fmt.Errorf("failed to load %s: %w", baseFile, err)
	}

	// Load custom overrides (optional)
	if _, err := os.Stat(customFile); err == nil {
		// Custom file exists, load it to override base values
		if err := godotenv.Overload(customFile); err != nil {
			return fmt.Errorf("failed to load %s: %w", customFile, err)
		}
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
