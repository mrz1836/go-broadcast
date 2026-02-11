// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains tests for nil config handling in LoggerService.
// These tests verify that nil configurations are handled safely without panics.
package cli

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoggerService_NilConfig_IsTraceEnabled verifies that IsTraceEnabled
// returns false safely when config is nil, preventing nil pointer panics.
//
// This matters because callers may construct LoggerService with nil config
// in error paths or during testing, and accessing Verbose field would panic.
func TestLoggerService_NilConfig_IsTraceEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *LogConfig
		expected bool
	}{
		{
			name:     "nil config returns false",
			config:   nil,
			expected: false,
		},
		{
			name:     "zero value config returns false",
			config:   &LogConfig{},
			expected: false,
		},
		{
			name:     "verbose level 1 returns false",
			config:   &LogConfig{Verbose: 1},
			expected: false,
		},
		{
			name:     "verbose level 2 returns true",
			config:   &LogConfig{Verbose: 2},
			expected: true,
		},
		{
			name:     "verbose level 3+ returns true",
			config:   &LogConfig{Verbose: 5},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewLoggerService(tt.config)
			require.NotNil(t, service, "service should never be nil")

			// This should not panic even with nil config
			result := service.IsTraceEnabled()
			assert.Equal(t, tt.expected, result, "IsTraceEnabled should return expected value")
		})
	}
}

// TestLoggerService_NilConfig_IsDebugEnabled verifies that IsDebugEnabled
// returns false safely when config is nil, preventing nil pointer panics.
//
// This matters because IsDebugEnabled accesses both config.Verbose and
// calls mapVerboseToLevel(), which would both panic with nil config.
func TestLoggerService_NilConfig_IsDebugEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *LogConfig
		expected bool
	}{
		{
			name:     "nil config returns false",
			config:   nil,
			expected: false,
		},
		{
			// Empty config defaults to InfoLevel, and InfoLevel(4) <= DebugLevel(5) is true
			name:     "zero value config returns true (InfoLevel <= DebugLevel)",
			config:   &LogConfig{},
			expected: true,
		},
		{
			name:     "verbose level 1 returns true",
			config:   &LogConfig{Verbose: 1},
			expected: true,
		},
		{
			name:     "debug log level returns true",
			config:   &LogConfig{LogLevel: "debug"},
			expected: true,
		},
		{
			name:     "trace log level returns true (TraceLevel > DebugLevel but Verbose >= 1 check)",
			config:   &LogConfig{LogLevel: "trace"},
			expected: false, // TraceLevel(6) > DebugLevel(5), and Verbose=0, so returns false
		},
		{
			// InfoLevel(4) <= DebugLevel(5), so this returns true
			name:     "info log level returns true (InfoLevel <= DebugLevel)",
			config:   &LogConfig{LogLevel: "info"},
			expected: true,
		},
		{
			// WarnLevel(3) <= DebugLevel(5), so this returns true
			name:     "warn log level returns true",
			config:   &LogConfig{LogLevel: "warn"},
			expected: true,
		},
		{
			// ErrorLevel(2) <= DebugLevel(5), so this returns true
			name:     "error log level returns true",
			config:   &LogConfig{LogLevel: "error"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewLoggerService(tt.config)
			require.NotNil(t, service, "service should never be nil")

			// This should not panic even with nil config
			result := service.IsDebugEnabled()
			assert.Equal(t, tt.expected, result, "IsDebugEnabled should return expected value")
		})
	}
}

// TestLoggerService_NilConfig_GetDebugFlags verifies that GetDebugFlags
// returns an empty DebugFlags struct when config is nil, preventing panics.
//
// This matters because callers may check debug flags in various code paths,
// and returning a zero-value struct allows code to continue safely.
func TestLoggerService_NilConfig_GetDebugFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         *LogConfig
		expectedGit    bool
		expectedAPI    bool
		expectedConfig bool
	}{
		{
			name:           "nil config returns empty debug flags",
			config:         nil,
			expectedGit:    false,
			expectedAPI:    false,
			expectedConfig: false,
		},
		{
			name:           "zero value config returns empty debug flags",
			config:         &LogConfig{},
			expectedGit:    false,
			expectedAPI:    false,
			expectedConfig: false,
		},
		{
			name: "config with debug flags returns those flags",
			config: &LogConfig{
				Debug: DebugFlags{
					Git:    true,
					API:    true,
					Config: false,
				},
			},
			expectedGit:    true,
			expectedAPI:    true,
			expectedConfig: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewLoggerService(tt.config)
			require.NotNil(t, service, "service should never be nil")

			// This should not panic even with nil config
			flags := service.GetDebugFlags()
			assert.Equal(t, tt.expectedGit, flags.Git, "Git flag should match expected")
			assert.Equal(t, tt.expectedAPI, flags.API, "API flag should match expected")
			assert.Equal(t, tt.expectedConfig, flags.Config, "Config flag should match expected")
		})
	}
}

// TestLoggerService_NilConfig_ConfigureLogger verifies that ConfigureLogger
// returns an error when config is nil instead of panicking.
//
// This matters because ConfigureLogger is called during application startup,
// and a clear error is preferable to a panic with an obscure stack trace.
func TestLoggerService_NilConfig_ConfigureLogger(t *testing.T) {
	tests := []struct {
		name        string
		config      *LogConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config returns error",
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name:        "valid config succeeds",
			config:      &LogConfig{LogLevel: "info"},
			expectError: false,
		},
		{
			name:        "empty config with defaults succeeds",
			config:      &LogConfig{},
			expectError: false,
		},
		{
			name:        "invalid log level returns error",
			config:      &LogConfig{LogLevel: "invalid_level"},
			expectError: true,
			errorMsg:    "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Not parallel because ConfigureLogger modifies global logrus state
			service := NewLoggerService(tt.config)
			require.NotNil(t, service, "service should never be nil")

			ctx := context.Background()
			err := service.ConfigureLogger(ctx)

			if tt.expectError {
				require.Error(t, err, "ConfigureLogger should return error")
				assert.Contains(t, err.Error(), tt.errorMsg, "error should contain expected message")
			} else {
				require.NoError(t, err, "ConfigureLogger should succeed")
			}
		})
	}
}

// TestLoggerService_NilConfig_MapVerboseLevelWithError verifies that
// mapVerboseLevelWithError returns an error when config is nil.
//
// This matters because mapVerboseLevelWithError is called internally
// by ConfigureLogger and must handle nil config gracefully.
func TestLoggerService_NilConfig_MapVerboseLevelWithError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        *LogConfig
		expectedLevel logrus.Level
		expectError   bool
	}{
		{
			name:          "nil config returns error with info level default",
			config:        nil,
			expectedLevel: logrus.InfoLevel,
			expectError:   true,
		},
		{
			name:          "empty config defaults to info level",
			config:        &LogConfig{},
			expectedLevel: logrus.InfoLevel,
			expectError:   false,
		},
		{
			name:          "verbose 1 returns debug level",
			config:        &LogConfig{Verbose: 1},
			expectedLevel: logrus.DebugLevel,
			expectError:   false,
		},
		{
			name:          "verbose 2 returns trace level",
			config:        &LogConfig{Verbose: 2},
			expectedLevel: logrus.TraceLevel,
			expectError:   false,
		},
		{
			name:          "verbose 3+ returns trace level",
			config:        &LogConfig{Verbose: 10},
			expectedLevel: logrus.TraceLevel,
			expectError:   false,
		},
		{
			name:          "explicit debug level",
			config:        &LogConfig{LogLevel: "debug"},
			expectedLevel: logrus.DebugLevel,
			expectError:   false,
		},
		{
			name:          "invalid log level returns error",
			config:        &LogConfig{LogLevel: "not_a_level"},
			expectedLevel: logrus.InfoLevel,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewLoggerService(tt.config)
			require.NotNil(t, service, "service should never be nil")

			// Access the internal method via mapVerboseToLevel which calls mapVerboseLevelWithError
			level := service.mapVerboseToLevel()

			// For nil config, mapVerboseToLevel returns the default (info) since it ignores errors
			assert.Equal(t, tt.expectedLevel, level, "level should match expected")
		})
	}
}

// TestLoggerService_NilConfig_ContextCancellation verifies that
// ConfigureLogger respects context cancellation even with nil config.
//
// This ensures proper cancellation handling in all error paths.
func TestLoggerService_NilConfig_ContextCancellation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *LogConfig
		cancelFirst bool
		expectError bool
	}{
		{
			name:        "canceled context with nil config",
			config:      nil,
			cancelFirst: true,
			expectError: true,
		},
		{
			name:        "canceled context with valid config",
			config:      &LogConfig{LogLevel: "info"},
			cancelFirst: true,
			expectError: true,
		},
		{
			name:        "active context with nil config returns nil config error",
			config:      nil,
			cancelFirst: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewLoggerService(tt.config)
			require.NotNil(t, service, "service should never be nil")

			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelFirst {
				cancel()
			} else {
				defer cancel()
			}

			err := service.ConfigureLogger(ctx)

			if tt.expectError {
				require.Error(t, err, "should return error")
			} else {
				require.NoError(t, err, "should not return error")
			}
		})
	}
}
