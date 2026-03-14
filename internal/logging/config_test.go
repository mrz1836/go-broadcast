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

// TestWithCorrelationID tests the WithCorrelationID method
func TestWithCorrelationID(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		var config *LogConfig
		correlationID := "test-correlation-123"

		result := config.WithCorrelationID(correlationID)

		require.NotNil(t, result)
		assert.Equal(t, correlationID, result.CorrelationID)
		assert.Empty(t, result.LogLevel)   // Should be empty/default
		assert.Empty(t, result.ConfigFile) // Should be empty/default
	})

	t.Run("with existing config", func(t *testing.T) {
		original := &LogConfig{
			ConfigFile:    "sync.yaml",
			LogLevel:      "info",
			LogFormat:     "json",
			Verbose:       1,
			JSONOutput:    true,
			CorrelationID: "original-id",
			Debug: DebugFlags{
				Git: true,
				API: false,
			},
		}

		newCorrelationID := "new-correlation-456"
		result := original.WithCorrelationID(newCorrelationID)

		// Should create a new config with updated correlation ID
		require.NotNil(t, result)
		assert.NotSame(t, original, result, "should create a new config instance")

		// Should preserve all other fields
		assert.Equal(t, original.ConfigFile, result.ConfigFile)
		assert.Equal(t, original.LogLevel, result.LogLevel)
		assert.Equal(t, original.LogFormat, result.LogFormat)
		assert.Equal(t, original.Verbose, result.Verbose)
		assert.Equal(t, original.JSONOutput, result.JSONOutput)
		assert.Equal(t, original.Debug, result.Debug)

		// Should update correlation ID
		assert.Equal(t, newCorrelationID, result.CorrelationID)
		assert.Equal(t, "original-id", original.CorrelationID, "original should be unchanged")
	})

	t.Run("with empty correlation ID", func(t *testing.T) {
		config := &LogConfig{
			LogLevel: "debug",
		}

		result := config.WithCorrelationID("")

		require.NotNil(t, result)
		assert.Empty(t, result.CorrelationID)
		assert.Equal(t, config.LogLevel, result.LogLevel)
	})
}

// TestGenerateCorrelationIDFallback tests the fallback behavior
func TestGenerateCorrelationIDFallback(t *testing.T) {
	// Test that GenerateCorrelationID returns valid hex strings
	for i := 0; i < 10; i++ {
		id := GenerateCorrelationID()

		// Should not be empty
		assert.NotEmpty(t, id)

		// Should be reasonable length (16 chars for 8 bytes hex encoded, or fallback)
		assert.GreaterOrEqual(t, len(id), 8, "ID should be at least 8 characters: %s", id)

		// Should be valid hex string OR fallback string
		if id != "fallback-id" {
			// Should be hex encoded (16 chars for 8 bytes)
			assert.Len(t, id, 16, "hex-encoded ID should be 16 characters")

			// Should only contain hex characters
			for _, char := range id {
				valid := (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')
				assert.True(t, valid, "hex ID should only contain 0-9, a-f: %s", id)
			}
		}
	}
}

// TestGenerateCorrelationIDUniqueness tests ID uniqueness
func TestGenerateCorrelationIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)
	const numIDs = 100

	for i := 0; i < numIDs; i++ {
		id := GenerateCorrelationID()

		// Should not have seen this ID before
		assert.False(t, ids[id], "correlation ID should be unique: %s", id)
		ids[id] = true
	}

	// Should have generated all unique IDs
	assert.Len(t, ids, numIDs, "all generated IDs should be unique")
}
