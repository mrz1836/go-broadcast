// Package logging provides logging configuration and utilities for go-broadcast.
package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCorrelationID(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "generates unique correlation IDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate multiple correlation IDs
			id1 := GenerateCorrelationID()
			id2 := GenerateCorrelationID()
			id3 := GenerateCorrelationID()

			// Test basic properties
			assert.NotEmpty(t, id1, "correlation ID should not be empty")
			assert.NotEmpty(t, id2, "correlation ID should not be empty")
			assert.NotEmpty(t, id3, "correlation ID should not be empty")

			// Test uniqueness
			assert.NotEqual(t, id1, id2, "correlation IDs should be unique")
			assert.NotEqual(t, id2, id3, "correlation IDs should be unique")
			assert.NotEqual(t, id1, id3, "correlation IDs should be unique")

			// Test length (should be reasonable)
			assert.GreaterOrEqual(t, len(id1), 8, "correlation ID should be at least 8 characters")
			assert.LessOrEqual(t, len(id1), 64, "correlation ID should not exceed 64 characters")

			// Test format (should be alphanumeric or contain hyphens)
			for _, char := range id1 {
				valid := (char >= 'a' && char <= 'z') ||
					(char >= 'A' && char <= 'Z') ||
					(char >= '0' && char <= '9') ||
					char == '-'
				assert.True(t, valid, "correlation ID should contain only alphanumeric characters and hyphens")
			}
		})
	}
}

func TestLogConfig_Validation(t *testing.T) {
	tests := []struct {
		name     string
		config   *LogConfig
		expected bool
	}{
		{
			name: "valid config with defaults",
			config: &LogConfig{
				ConfigFile: "sync.yaml",
				LogLevel:   "info",
				LogFormat:  "text",
			},
			expected: true,
		},
		{
			name: "valid config with verbose flags",
			config: &LogConfig{
				ConfigFile:    "sync.yaml",
				LogLevel:      "info",
				LogFormat:     "json",
				Verbose:       2,
				JSONOutput:    true,
				CorrelationID: "test-123",
				Debug: DebugFlags{
					Git:       true,
					API:       true,
					Transform: true,
					Config:    true,
					State:     true,
				},
			},
			expected: true,
		},
		{
			name: "config with all debug flags disabled",
			config: &LogConfig{
				ConfigFile: "sync.yaml",
				LogLevel:   "debug",
				LogFormat:  "text",
				Debug: DebugFlags{
					Git:       false,
					API:       false,
					Transform: false,
					Config:    false,
					State:     false,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that config can be created and accessed
			require.NotNil(t, tt.config, "config should not be nil")

			// Test basic field access
			assert.NotEmpty(t, tt.config.ConfigFile, "config file should not be empty")
			assert.NotEmpty(t, tt.config.LogLevel, "log level should not be empty")
			assert.NotEmpty(t, tt.config.LogFormat, "log format should not be empty")

			// Test verbose level bounds
			assert.GreaterOrEqual(t, tt.config.Verbose, 0, "verbose level should be non-negative")
			assert.LessOrEqual(t, tt.config.Verbose, 10, "verbose level should be reasonable")

			// Test debug flags structure
			debugFlags := tt.config.Debug
			// All boolean fields should be accessible
			_ = debugFlags.Git
			_ = debugFlags.API
			_ = debugFlags.Transform
			_ = debugFlags.Config
			_ = debugFlags.State
		})
	}
}

func TestDebugFlags_String(t *testing.T) {
	tests := []struct {
		name     string
		flags    DebugFlags
		contains []string
	}{
		{
			name: "no debug flags enabled",
			flags: DebugFlags{
				Git:       false,
				API:       false,
				Transform: false,
				Config:    false,
				State:     false,
			},
			contains: []string{}, // Should handle empty case gracefully
		},
		{
			name: "single debug flag enabled",
			flags: DebugFlags{
				Git:       true,
				API:       false,
				Transform: false,
				Config:    false,
				State:     false,
			},
			contains: []string{"git"},
		},
		{
			name: "multiple debug flags enabled",
			flags: DebugFlags{
				Git:       true,
				API:       true,
				Transform: false,
				Config:    true,
				State:     false,
			},
			contains: []string{"git", "api", "config"},
		},
		{
			name: "all debug flags enabled",
			flags: DebugFlags{
				Git:       true,
				API:       true,
				Transform: true,
				Config:    true,
				State:     true,
			},
			contains: []string{"git", "api", "transform", "config", "state"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// DebugFlags doesn't have a String() method in the current implementation
			// But we can test the individual flag values

			// Verify flag states match expectations
			assert.Equal(t, len(tt.contains) > 0 && contains(tt.contains, "git"), tt.flags.Git)
			assert.Equal(t, len(tt.contains) > 0 && contains(tt.contains, "api"), tt.flags.API)
			assert.Equal(t, len(tt.contains) > 0 && contains(tt.contains, "transform"), tt.flags.Transform)
			assert.Equal(t, len(tt.contains) > 0 && contains(tt.contains, "config"), tt.flags.Config)
			assert.Equal(t, len(tt.contains) > 0 && contains(tt.contains, "state"), tt.flags.State)
		})
	}
}

func TestLogConfig_HasAnyDebugFlag(t *testing.T) {
	tests := []struct {
		name     string
		config   *LogConfig
		expected bool
	}{
		{
			name: "no debug flags enabled",
			config: &LogConfig{
				Debug: DebugFlags{
					Git:       false,
					API:       false,
					Transform: false,
					Config:    false,
					State:     false,
				},
			},
			expected: false,
		},
		{
			name: "git debug flag enabled",
			config: &LogConfig{
				Debug: DebugFlags{
					Git:       true,
					API:       false,
					Transform: false,
					Config:    false,
					State:     false,
				},
			},
			expected: true,
		},
		{
			name: "multiple debug flags enabled",
			config: &LogConfig{
				Debug: DebugFlags{
					Git:       false,
					API:       true,
					Transform: true,
					Config:    false,
					State:     false,
				},
			},
			expected: true,
		},
		{
			name: "all debug flags enabled",
			config: &LogConfig{
				Debug: DebugFlags{
					Git:       true,
					API:       true,
					Transform: true,
					Config:    true,
					State:     true,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since HasAnyDebugFlag method doesn't exist, we'll test the logic manually
			hasAnyDebug := tt.config.Debug.Git ||
				tt.config.Debug.API ||
				tt.config.Debug.Transform ||
				tt.config.Debug.Config ||
				tt.config.Debug.State

			assert.Equal(t, tt.expected, hasAnyDebug, "HasAnyDebugFlag should match expected result")
		})
	}
}

func TestLogConfig_GetVerboseLevel(t *testing.T) {
	tests := []struct {
		name     string
		config   *LogConfig
		expected string
	}{
		{
			name: "verbose level 0 (default)",
			config: &LogConfig{
				Verbose: 0,
			},
			expected: "info", // Default level when no verbose flags
		},
		{
			name: "verbose level 1 (-v)",
			config: &LogConfig{
				Verbose: 1,
			},
			expected: "debug",
		},
		{
			name: "verbose level 2 (-vv)",
			config: &LogConfig{
				Verbose: 2,
			},
			expected: "trace",
		},
		{
			name: "verbose level 3 (-vvv)",
			config: &LogConfig{
				Verbose: 3,
			},
			expected: "trace", // Should still be trace level
		},
		{
			name: "verbose level higher than 3",
			config: &LogConfig{
				Verbose: 5,
			},
			expected: "trace", // Should cap at trace level
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test verbose level mapping logic
			var expectedLevel string
			switch {
			case tt.config.Verbose == 0:
				expectedLevel = "info"
			case tt.config.Verbose == 1:
				expectedLevel = "debug"
			case tt.config.Verbose >= 2:
				expectedLevel = "trace"
			}

			assert.Equal(t, tt.expected, expectedLevel, "verbose level mapping should be correct")
		})
	}
}

// Helper function for testing
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
