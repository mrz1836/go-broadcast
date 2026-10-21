// Package env provides utilities for loading environment variables from .env files.
// It follows the GoFortress pattern used by other tools in the ecosystem.
package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// Static errors for consistent error handling
var (
	ErrNotDirectory = errors.New("path is not a directory")
	ErrNoEnvFiles   = errors.New("no .env files found in directory")
)

// LoadEnvFiles loads environment variables from the modular .github/env/*.env
// files in the current working directory, with proper precedence:
//  1. OS environment variables (HIGHEST - never overwritten)
//  2. .github/env/*.env files (last-wins ordering)
//
// Variables already set in the OS environment are NEVER overwritten.
// This allows users to override via shell exports or CI/CD secrets.
//
// The env files are optional: when the working directory has no .github/env/
// directory with .env files (e.g. running "go-broadcast update" from $HOME),
// nothing is loaded and no error is returned.
//
// This ensures go-broadcast follows the same configuration pattern as
// go-coverage, go-pre-commit, mage-x, and other GoFortress tools.
func LoadEnvFiles() error {
	if envDir := findEnvDir(); envDir != "" {
		return LoadEnvDir(envDir, isCI())
	}
	return nil
}

// LoadEnvFilesFromDir loads environment files from .github/env/ under a specific directory.
// This is useful for testing or when running from a different working directory.
// Like LoadEnvFiles, it is a no-op when the directory has no .github/env/*.env files.
func LoadEnvFilesFromDir(dir string) error {
	if envDir := findEnvDirFromBase(dir); envDir != "" {
		return LoadEnvDir(envDir, isCI())
	}
	return nil
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

	slices.Sort(matches)

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
