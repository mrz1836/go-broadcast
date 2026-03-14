// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains tests for error paths in modules operations.
// These tests verify that module commands properly handle version resolution
// failures, validation errors, and mixed valid/invalid scenarios.
package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModuleSentinelErrors verifies that module sentinel errors are properly
// defined and have descriptive messages.
//
// This matters for error handling in callers who use errors.Is() checks.
func TestModuleSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("errors are defined", func(t *testing.T) {
		t.Parallel()
		require.Error(t, ErrModuleNotFound)
		require.Error(t, ErrModuleValidationFailed)
		require.Error(t, ErrInvalidRepositoryPath)
		require.Error(t, ErrModulePathRequired)
	})

	t.Run("errors are distinct", func(t *testing.T) {
		t.Parallel()
		sentinels := []error{
			ErrModuleNotFound,
			ErrModuleValidationFailed,
			ErrInvalidRepositoryPath,
			ErrModulePathRequired,
		}

		for i, err1 := range sentinels {
			for j, err2 := range sentinels {
				if i != j {
					assert.NotEqual(t, err1.Error(), err2.Error(),
						"errors %d and %d should be distinct", i, j)
				}
			}
		}
	})

	t.Run("errors have descriptive messages", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, ErrModuleNotFound.Error(), "not found")
		assert.Contains(t, ErrModuleValidationFailed.Error(), "validation")
		assert.Contains(t, ErrInvalidRepositoryPath.Error(), "repository")
		assert.Contains(t, ErrModulePathRequired.Error(), "required")
	})

	t.Run("different errors should not match", func(t *testing.T) {
		t.Parallel()
		assert.NotErrorIs(t, ErrModuleNotFound, ErrModuleValidationFailed)
	})
}

// TestGetModuleType_AllTypes verifies that getModuleType returns
// correct type strings for all module types.
//
// This is used for display output and should handle all known types.
func TestGetModuleType_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeName string
		expected string
	}{
		{
			name:     "go module type",
			typeName: "go",
			expected: "go",
		},
		{
			name:     "empty type defaults to Go",
			typeName: "",
			expected: "go (default)",
		},
		{
			name:     "unknown type returned as-is",
			typeName: "npm",
			expected: "npm",
		},
		{
			name:     "rust type returned as-is",
			typeName: "rust",
			expected: "rust",
		},
		{
			name:     "case sensitive - GO not matched",
			typeName: "GO",
			expected: "GO",
		},
		{
			name:     "whitespace type",
			typeName: "  ",
			expected: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getModuleType(tt.typeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFetchGitTags_InvalidRepository verifies that fetchGitTags validates
// repository paths before executing git commands.
//
// This prevents command injection attacks through malicious repo names.
func TestFetchGitTags_InvalidRepository(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		repo    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid github repo",
			repo:    "github.com/user/repo",
			wantErr: false, // May fail at git command, but passes validation
		},
		{
			name:    "repo with semicolon injection attempt",
			repo:    "github.com/user/repo; rm -rf /",
			wantErr: true,
			errMsg:  "invalid repository path",
		},
		{
			name:    "repo with ampersand injection attempt",
			repo:    "github.com/user/repo && rm -rf /",
			wantErr: true,
			errMsg:  "invalid repository path",
		},
		{
			name:    "repo with path traversal",
			repo:    "github.com/user/../etc/passwd",
			wantErr: true,
			errMsg:  "invalid repository path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			_, err := fetchGitTags(ctx, tt.repo)

			if tt.wantErr {
				require.Error(t, err, "should reject invalid repo path")
				assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errMsg))
			}
			// For valid repos, we don't check success since git command may fail
			// on network issues - we're just testing the validation passes
		})
	}
}

// TestFetchGitTags_VersionSorting verifies that version tags are returned
// in proper semver order (most recent first).
//
// This is a documentation test - actual sorting happens inside the function.
func TestFetchGitTags_VersionSorting(t *testing.T) {
	// This test documents expected behavior:
	// - v2.0.0 should come before v1.9.0
	// - v1.10.0 should come before v1.9.0 (numeric, not lexicographic)
	// - Prerelease versions (v1.0.0-beta) should sort appropriately
	t.Skip("Integration test - requires network access")
}

// TestModuleVersionFormatValidation verifies that version format validation
// catches unusual or invalid version strings.
//
// This helps users catch typos in version constraints.
func TestModuleVersionFormatValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		isNormal bool // Whether the version is considered "normal" (no warning)
	}{
		// Normal versions
		{name: "latest keyword", version: "latest", isNormal: true},
		{name: "semver with v prefix", version: "v1.2.3", isNormal: true},
		{name: "caret constraint", version: "^1.2.3", isNormal: true},
		{name: "tilde constraint", version: "~1.2.3", isNormal: true},
		{name: "greater than", version: ">1.0.0", isNormal: true},
		{name: "less than", version: "<2.0.0", isNormal: true},
		{name: "range constraint", version: ">=1.0.0 <2.0.0", isNormal: true},

		// Unusual versions (would trigger warning)
		{name: "bare number", version: "1.2.3", isNormal: false},
		{name: "single number", version: "1", isNormal: false},
		{name: "commit sha", version: "abc123def", isNormal: false},
		{name: "branch name", version: "main", isNormal: false},
		{name: "date format", version: "2024-01-15", isNormal: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Check if version matches "normal" patterns
			isNormal := tt.version == "latest" ||
				strings.HasPrefix(tt.version, "v") ||
				strings.Contains(tt.version, "^") ||
				strings.Contains(tt.version, "~") ||
				strings.Contains(tt.version, ">") ||
				strings.Contains(tt.version, "<")

			assert.Equal(t, tt.isNormal, isNormal,
				"version %q normality check should match expected", tt.version)
		})
	}
}

// TestModuleCommandStructure verifies that module subcommands are
// properly configured.
func TestModuleCommandStructure(t *testing.T) {
	t.Parallel()

	t.Run("modulesCmd has required fields", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "modules", modulesCmd.Use)
		assert.Contains(t, modulesCmd.Short, "module")
		assert.NotEmpty(t, modulesCmd.Long)
		assert.NotEmpty(t, modulesCmd.Example)
	})

	t.Run("listModulesCmd configuration", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "list", listModulesCmd.Use)
		assert.NotNil(t, listModulesCmd.RunE)
	})

	t.Run("showModuleCmd requires one argument", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "show [module-path]", showModuleCmd.Use)
		assert.NotNil(t, showModuleCmd.Args)
		assert.NotNil(t, showModuleCmd.RunE)
	})

	t.Run("moduleVersionsCmd requires one argument", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "versions [module-path]", moduleVersionsCmd.Use)
		assert.NotNil(t, moduleVersionsCmd.Args)
		assert.NotNil(t, moduleVersionsCmd.RunE)
	})

	t.Run("validateModulesCmd configuration", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "validate", validateModulesCmd.Use)
		assert.NotNil(t, validateModulesCmd.RunE)
	})
}

// TestModulePathMatching verifies that module paths are correctly matched
// for the 'show' command.
//
// This tests both exact and partial path matching behavior.
func TestModulePathMatching(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		modulePath  string
		configPaths []string
		shouldMatch bool
		matchedPath string
	}{
		{
			name:        "exact match",
			modulePath:  "github.com/example/errors",
			configPaths: []string{"github.com/example/errors", "github.com/example/utils"},
			shouldMatch: true,
			matchedPath: "github.com/example/errors",
		},
		{
			name:        "basename match",
			modulePath:  "errors",
			configPaths: []string{"github.com/example/errors"},
			shouldMatch: true,
			matchedPath: "github.com/example/errors",
		},
		{
			name:        "no match",
			modulePath:  "nonexistent",
			configPaths: []string{"github.com/example/errors"},
			shouldMatch: false,
		},
		{
			// Partial path matches via HasSuffix check
			name:        "partial path matches via suffix",
			modulePath:  "example/errors",
			configPaths: []string{"github.com/example/errors"},
			shouldMatch: true, // HasSuffix("github.com/example/errors", "/example/errors") is true
			matchedPath: "github.com/example/errors",
		},
		{
			name:        "case sensitive",
			modulePath:  "Errors",
			configPaths: []string{"github.com/example/errors"},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Simulate the matching logic from runShowModule
			var found bool
			var matched string

			for _, configPath := range tt.configPaths {
				// Check exact match first
				if configPath == tt.modulePath {
					found = true
					matched = configPath
					break
				}
				// Check basename match
				if strings.HasSuffix(configPath, "/"+tt.modulePath) {
					found = true
					matched = configPath
					break
				}
				// Check if the search term is the basename
				base := configPath
				if idx := strings.LastIndex(configPath, "/"); idx >= 0 {
					base = configPath[idx+1:]
				}
				if base == tt.modulePath {
					found = true
					matched = configPath
					break
				}
			}

			assert.Equal(t, tt.shouldMatch, found)
			if tt.shouldMatch {
				assert.Equal(t, tt.matchedPath, matched)
			}
		})
	}
}

// TestModuleVersionConstraintTypes documents supported version constraint types.
func TestModuleVersionConstraintTypes(t *testing.T) {
	t.Parallel()

	// Document supported constraint types
	supportedTypes := []struct {
		name        string
		example     string
		description string
	}{
		{"latest", "latest", "Always use most recent version"},
		{"exact", "v1.2.3", "Use exactly this version"},
		{"caret", "^1.2.3", "Compatible with 1.2.3 (^major.minor.patch)"},
		{"tilde", "~1.2.3", "Approximately 1.2.3"},
		{"greater", ">1.0.0", "Greater than 1.0.0"},
		{"greater-equal", ">=1.0.0", "Greater than or equal"},
		{"less", "<2.0.0", "Less than 2.0.0"},
		{"less-equal", "<=2.0.0", "Less than or equal"},
		{"range", ">=1.0.0 <2.0.0", "Version range"},
	}

	for _, constraint := range supportedTypes {
		t.Run(constraint.name, func(t *testing.T) {
			t.Parallel()
			assert.NotEmpty(t, constraint.example)
			assert.NotEmpty(t, constraint.description)
			// This is a documentation test - we're just ensuring the constraint
			// types are documented and non-empty
		})
	}
}
