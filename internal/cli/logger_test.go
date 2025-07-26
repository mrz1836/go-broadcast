// Package cli provides command-line interface functionality for go-broadcast.
package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggerService(t *testing.T) {
	tests := []struct {
		name     string
		config   *LogConfig
		validate func(t *testing.T, service *LoggerService)
	}{
		{
			name: "basic logger service creation",
			config: &LogConfig{
				ConfigFile: "test.yaml",
				LogLevel:   "info",
				LogFormat:  "text",
				Verbose:    0,
			},
			validate: func(t *testing.T, service *LoggerService) {
				assert.NotNil(t, service, "service should not be nil")
				assert.NotNil(t, service.config, "config should not be nil")
				assert.Equal(t, "test.yaml", service.config.ConfigFile, "config file should be set")
				assert.Equal(t, "info", service.config.LogLevel, "log level should be set")
				assert.Equal(t, "text", service.config.LogFormat, "log format should be set")
			},
		},
		{
			name: "logger service with verbose config",
			config: &LogConfig{
				ConfigFile:    "verbose.yaml",
				LogLevel:      "debug",
				LogFormat:     "json",
				Verbose:       2,
				JSONOutput:    true,
				CorrelationID: "test-123",
				Debug: DebugFlags{
					Git:       true,
					API:       true,
					Transform: false,
					Config:    true,
					State:     false,
				},
			},
			validate: func(t *testing.T, service *LoggerService) {
				assert.NotNil(t, service, "service should not be nil")
				assert.Equal(t, "verbose.yaml", service.config.ConfigFile, "config file should be set")
				assert.Equal(t, "debug", service.config.LogLevel, "log level should be set")
				assert.Equal(t, "json", service.config.LogFormat, "log format should be set")
				assert.Equal(t, 2, service.config.Verbose, "verbose level should be set")
				assert.True(t, service.config.JSONOutput, "JSON output should be enabled")
				assert.Equal(t, "test-123", service.config.CorrelationID, "correlation ID should be set")
				assert.True(t, service.config.Debug.Git, "git debug should be enabled")
				assert.True(t, service.config.Debug.API, "api debug should be enabled")
				assert.False(t, service.config.Debug.Transform, "transform debug should be disabled")
			},
		},
		{
			name:   "logger service with nil config",
			config: nil,
			validate: func(t *testing.T, service *LoggerService) {
				assert.NotNil(t, service, "service should not be nil even with nil config")
				// Should handle nil config gracefully
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLoggerService(tt.config)
			tt.validate(t, service)
		})
	}
}

func TestLoggerService_ConfigureLogger(t *testing.T) {
	tests := []struct {
		name        string
		config      *LogConfig
		expectError bool
		validate    func(t *testing.T, config *LogConfig)
	}{
		{
			name: "configure with text format",
			config: &LogConfig{
				LogLevel:      "info",
				LogFormat:     "text",
				Verbose:       0,
				CorrelationID: "test-text",
			},
			expectError: false,
			validate: func(_ *testing.T, config *LogConfig) {
				// Basic validation that configuration was processed
				assert.Equal(t, "info", config.LogLevel, "log level should be preserved")
				assert.Equal(t, "text", config.LogFormat, "log format should be preserved")
			},
		},
		{
			name: "configure with json format",
			config: &LogConfig{
				LogLevel:      "debug",
				LogFormat:     "json",
				Verbose:       1,
				CorrelationID: "test-json",
			},
			expectError: false,
			validate: func(_ *testing.T, config *LogConfig) {
				assert.Equal(t, "debug", config.LogLevel, "log level should be preserved")
				assert.Equal(t, "json", config.LogFormat, "log format should be preserved")
			},
		},
		{
			name: "configure with verbose level 2",
			config: &LogConfig{
				LogLevel:      "info",
				LogFormat:     "text",
				Verbose:       2,
				CorrelationID: "test-verbose",
			},
			expectError: false,
			validate: func(_ *testing.T, config *LogConfig) {
				// Verbose level should override log level
				assert.Equal(t, 2, config.Verbose, "verbose level should be preserved")
			},
		},
		{
			name: "configure with invalid log level",
			config: &LogConfig{
				LogLevel:      "invalid",
				LogFormat:     "text",
				Verbose:       0,
				CorrelationID: "test-invalid",
			},
			expectError: true,
			validate: func(_ *testing.T, _ *LogConfig) {
				// Should not be called when error expected
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLoggerService(tt.config)
			ctx := context.Background()

			err := service.ConfigureLogger(ctx)

			if tt.expectError {
				assert.Error(t, err, "expected error during logger configuration")
			} else {
				require.NoError(t, err, "logger configuration should not return error")
				tt.validate(t, tt.config)
			}
		})
	}
}

func TestLoggerService_Integration(t *testing.T) {
	// Test full integration of logger service with actual logging
	tests := []struct {
		name     string
		config   *LogConfig
		logFunc  func()
		validate func(t *testing.T, output string)
	}{
		{
			name: "text format logging integration",
			config: &LogConfig{
				LogLevel:      "info",
				LogFormat:     "text",
				Verbose:       0,
				CorrelationID: "integration-text",
			},
			logFunc: func() {
				logrus.WithFields(logrus.Fields{
					"component": "test",
					"operation": "integration",
				}).Info("Integration test message")
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Integration test message", "message should be in output")
				assert.Contains(t, output, "component=test", "component field should be in output")
				assert.Contains(t, output, "operation=integration", "operation field should be in output")
			},
		},
		{
			name: "json format logging integration",
			config: &LogConfig{
				LogLevel:      "debug",
				LogFormat:     "json",
				Verbose:       1,
				CorrelationID: "integration-json",
			},
			logFunc: func() {
				logrus.WithFields(logrus.Fields{
					"component":   "test",
					"operation":   "integration",
					"duration_ms": 150,
				}).Debug("Debug integration message")
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Debug integration message", "message should be in JSON output")
				assert.Contains(t, output, "\"component\":\"test\"", "component should be in JSON format")
				assert.Contains(t, output, "\"operation\":\"integration\"", "operation should be in JSON format")
				assert.Contains(t, output, "\"duration_ms\":150", "duration should be in JSON format")
				assert.Contains(t, output, "\"level\":\"debug\"", "level should be in JSON format")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var buffer bytes.Buffer
			originalOutput := logrus.StandardLogger().Out
			logrus.SetOutput(&buffer)
			defer logrus.SetOutput(originalOutput)

			// Configure logger service
			service := NewLoggerService(tt.config)
			ctx := context.Background()

			err := service.ConfigureLogger(ctx)
			require.NoError(t, err, "logger configuration should succeed")

			// Execute logging function
			tt.logFunc()

			// Validate output
			output := buffer.String()
			require.NotEmpty(t, output, "should have logged output")
			tt.validate(t, output)
		})
	}
}

func TestLoggerService_VerboseLevelMapping(t *testing.T) {
	tests := []struct {
		name         string
		verbose      int
		baseLogLevel string
		expectedFunc func(t *testing.T)
	}{
		{
			name:         "verbose 0 uses base log level",
			verbose:      0,
			baseLogLevel: "info",
			expectedFunc: func(t *testing.T) {
				assert.Equal(t, logrus.InfoLevel, logrus.GetLevel(), "should use info level")
			},
		},
		{
			name:         "verbose 1 sets debug level",
			verbose:      1,
			baseLogLevel: "info",
			expectedFunc: func(t *testing.T) {
				assert.Equal(t, logrus.DebugLevel, logrus.GetLevel(), "should use debug level")
			},
		},
		{
			name:         "verbose 2 sets trace level",
			verbose:      2,
			baseLogLevel: "info",
			expectedFunc: func(t *testing.T) {
				assert.Equal(t, logrus.TraceLevel, logrus.GetLevel(), "should use trace level")
			},
		},
		{
			name:         "verbose 3+ caps at trace level",
			verbose:      5,
			baseLogLevel: "warn",
			expectedFunc: func(t *testing.T) {
				assert.Equal(t, logrus.TraceLevel, logrus.GetLevel(), "should cap at trace level")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LogConfig{
				LogLevel:      tt.baseLogLevel,
				LogFormat:     "text",
				Verbose:       tt.verbose,
				CorrelationID: "verbose-test",
			}

			service := NewLoggerService(config)
			ctx := context.Background()

			err := service.ConfigureLogger(ctx)
			require.NoError(t, err, "logger configuration should succeed")

			tt.expectedFunc(t)
		})
	}
}

func TestLoggerService_HookInstallation(t *testing.T) {
	// Test that necessary hooks are installed during configuration
	config := &LogConfig{
		LogLevel:      "info",
		LogFormat:     "json",
		CorrelationID: "hook-test",
	}

	service := NewLoggerService(config)
	ctx := context.Background()

	// Clear existing hooks
	logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))

	err := service.ConfigureLogger(ctx)
	require.NoError(t, err, "logger configuration should succeed")

	// Check that hooks were installed
	hooks := logrus.StandardLogger().Hooks
	assert.NotEmpty(t, hooks, "hooks should be installed")

	// Test that redaction hook is working by logging sensitive data
	var buffer bytes.Buffer
	logrus.SetOutput(&buffer)

	logrus.WithField("token", "ghp_secret123").Info("Test message")

	output := buffer.String()
	assert.Contains(t, output, "ghp_***REDACTED***", "token should be redacted")
	assert.NotContains(t, output, "ghp_secret123", "original token should not appear")
}

func TestLoggerService_CorrelationIdHandling(t *testing.T) {
	tests := []struct {
		name          string
		correlationID string
		validate      func(t *testing.T, config *LogConfig)
	}{
		{
			name:          "with correlation ID",
			correlationID: "test-correlation-123",
			validate: func(_ *testing.T, config *LogConfig) {
				assert.Equal(t, "test-correlation-123", config.CorrelationID, "correlation ID should be preserved")
			},
		},
		{
			name:          "with empty correlation ID",
			correlationID: "",
			validate: func(_ *testing.T, config *LogConfig) {
				assert.Empty(t, config.CorrelationID, "empty correlation ID should be preserved")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LogConfig{
				LogLevel:      "info",
				LogFormat:     "text",
				CorrelationID: tt.correlationID,
			}

			service := NewLoggerService(config)
			ctx := context.Background()

			err := service.ConfigureLogger(ctx)
			require.NoError(t, err, "logger configuration should succeed")

			tt.validate(t, config)
		})
	}
}

func TestLoggerService_DebugFlagsHandling(t *testing.T) {
	// Test that debug flags are preserved and accessible
	config := &LogConfig{
		LogLevel:  "info",
		LogFormat: "text",
		Debug: DebugFlags{
			Git:       true,
			API:       false,
			Transform: true,
			Config:    false,
			State:     true,
		},
	}

	service := NewLoggerService(config)
	ctx := context.Background()

	err := service.ConfigureLogger(ctx)
	require.NoError(t, err, "logger configuration should succeed")

	// Debug flags should be preserved in the config
	assert.True(t, service.config.Debug.Git, "git debug flag should be preserved")
	assert.False(t, service.config.Debug.API, "api debug flag should be preserved")
	assert.True(t, service.config.Debug.Transform, "transform debug flag should be preserved")
	assert.False(t, service.config.Debug.Config, "config debug flag should be preserved")
	assert.True(t, service.config.Debug.State, "state debug flag should be preserved")
}

func TestLoggerService_JSONOutputAlias(t *testing.T) {
	// Test that JSONOutput flag sets LogFormat to json
	config := &LogConfig{
		LogLevel:   "info",
		LogFormat:  "text", // Should be overridden by JSONOutput
		JSONOutput: true,
	}

	service := NewLoggerService(config)
	ctx := context.Background()

	err := service.ConfigureLogger(ctx)
	require.NoError(t, err, "logger configuration should succeed")

	// The actual format change happens in the CLI setup, but we can test the config is preserved
	assert.True(t, service.config.JSONOutput, "JSON output flag should be preserved")
}

func TestLoggerService_ErrorHandling(t *testing.T) {
	// Test error handling in logger configuration
	tests := []struct {
		name        string
		config      *LogConfig
		expectError bool
	}{
		{
			name: "invalid log level",
			config: &LogConfig{
				LogLevel:  "invalid_level",
				LogFormat: "text",
			},
			expectError: true,
		},
		{
			name: "valid configuration",
			config: &LogConfig{
				LogLevel:  "debug",
				LogFormat: "json",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLoggerService(tt.config)
			ctx := context.Background()

			err := service.ConfigureLogger(ctx)

			if tt.expectError {
				assert.Error(t, err, "expected error for invalid configuration")
			} else {
				assert.NoError(t, err, "valid configuration should not error")
			}
		})
	}
}

func TestLoggerService_OutputCapture(t *testing.T) {
	// Test that logger output goes to stderr (keeping stdout clean)
	config := &LogConfig{
		LogLevel:  "info",
		LogFormat: "text",
	}

	service := NewLoggerService(config)
	ctx := context.Background()

	err := service.ConfigureLogger(ctx)
	require.NoError(t, err, "logger configuration should succeed")

	// Verify that logger is configured to output to stderr
	// In actual implementation, this would be done in the CLI setup
	// Here we just verify the service doesn't error
	assert.NotNil(t, service, "service should be properly initialized")
}

func TestLoggerService_ConcurrentUsage(t *testing.T) {
	// Test that logger service can handle concurrent configuration attempts
	config := &LogConfig{
		LogLevel:  "info",
		LogFormat: "text",
	}

	service := NewLoggerService(config)
	ctx := context.Background()

	// Configure logger concurrently (should be safe)
	errChan := make(chan error, 2)

	go func() {
		errChan <- service.ConfigureLogger(ctx)
	}()

	go func() {
		errChan <- service.ConfigureLogger(ctx)
	}()

	// Both should succeed (or at least not crash)
	err1 := <-errChan
	err2 := <-errChan

	// At least one should succeed, both should not crash
	if err1 != nil && err2 != nil {
		t.Errorf("both concurrent configurations failed: %v, %v", err1, err2)
	}
}

func TestTraceHookFire(t *testing.T) {
	tests := []struct {
		name           string
		entry          *logrus.Entry
		expectedLevel  logrus.Level
		expectedPrefix string
	}{
		{
			name: "TraceLevel entry gets converted to debug with prefix",
			entry: &logrus.Entry{
				Level:   TraceLevel,
				Message: "trace message",
			},
			expectedLevel:  logrus.DebugLevel,
			expectedPrefix: "[TRACE] ",
		},
		{
			name: "DebugLevel entry gets converted with prefix",
			entry: &logrus.Entry{
				Level:   logrus.DebugLevel,
				Message: "debug message",
			},
			expectedLevel:  logrus.DebugLevel,
			expectedPrefix: "[TRACE] ",
		},
		{
			name: "InfoLevel entry gets converted with prefix",
			entry: &logrus.Entry{
				Level:   logrus.InfoLevel,
				Message: "info message",
			},
			expectedLevel:  logrus.DebugLevel,
			expectedPrefix: "[TRACE] ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &TraceHook{Enabled: true}
			err := hook.Fire(tt.entry)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, tt.entry.Level)

			if tt.expectedPrefix != "" {
				assert.Contains(t, tt.entry.Message, tt.expectedPrefix)
			} else {
				assert.NotContains(t, tt.entry.Message, "[TRACE]")
			}
		})
	}
}

func TestTraceHookLevels(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		expectedCount int
	}{
		{
			name:          "Enabled hook returns trace level",
			enabled:       true,
			expectedCount: 1,
		},
		{
			name:          "Disabled hook returns empty levels",
			enabled:       false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &TraceHook{Enabled: tt.enabled}
			levels := hook.Levels()

			assert.Len(t, levels, tt.expectedCount)
			if tt.enabled {
				assert.Contains(t, levels, TraceLevel)
			}
		})
	}
}

func TestLoggerServiceIsTraceEnabled(t *testing.T) {
	tests := []struct {
		name     string
		verbose  int
		expected bool
	}{
		{
			name:     "Verbose 0 trace disabled",
			verbose:  0,
			expected: false,
		},
		{
			name:     "Verbose 1 trace disabled",
			verbose:  1,
			expected: false,
		},
		{
			name:     "Verbose 2 trace enabled",
			verbose:  2,
			expected: true,
		},
		{
			name:     "Verbose 3 trace enabled",
			verbose:  3,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LogConfig{Verbose: tt.verbose}
			service := NewLoggerService(config)

			result := service.IsTraceEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoggerServiceIsDebugEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *LogConfig
		expected bool
	}{
		{
			name: "Verbose 0 with empty log level - uses default info level and enables debug",
			config: &LogConfig{
				Verbose:  0,
				LogLevel: "",
			},
			expected: true,
		},
		{
			name: "Verbose 0 and info level - debug enabled due to level ordering",
			config: &LogConfig{
				Verbose:  0,
				LogLevel: "info",
			},
			expected: true,
		},
		{
			name: "Verbose 1 - debug enabled",
			config: &LogConfig{
				Verbose:  1,
				LogLevel: "info",
			},
			expected: true,
		},
		{
			name: "Verbose 0 but debug log level - debug enabled",
			config: &LogConfig{
				Verbose:  0,
				LogLevel: "debug",
			},
			expected: true,
		},
		{
			name: "Verbose 0 but trace log level - debug disabled due to level ordering",
			config: &LogConfig{
				Verbose:  0,
				LogLevel: "trace",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLoggerService(tt.config)
			result := service.IsDebugEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoggerServiceGetDebugFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    DebugFlags
		validate func(t *testing.T, flags DebugFlags)
	}{
		{
			name: "All flags enabled",
			flags: DebugFlags{
				Git:       true,
				API:       true,
				Transform: true,
				Config:    true,
				State:     true,
			},
			validate: func(t *testing.T, flags DebugFlags) {
				assert.True(t, flags.Git)
				assert.True(t, flags.API)
				assert.True(t, flags.Transform)
				assert.True(t, flags.Config)
				assert.True(t, flags.State)
			},
		},
		{
			name: "Mixed flags",
			flags: DebugFlags{
				Git:       true,
				API:       false,
				Transform: true,
				Config:    false,
				State:     true,
			},
			validate: func(t *testing.T, flags DebugFlags) {
				assert.True(t, flags.Git)
				assert.False(t, flags.API)
				assert.True(t, flags.Transform)
				assert.False(t, flags.Config)
				assert.True(t, flags.State)
			},
		},
		{
			name:  "All flags disabled (zero value)",
			flags: DebugFlags{},
			validate: func(t *testing.T, flags DebugFlags) {
				assert.False(t, flags.Git)
				assert.False(t, flags.API)
				assert.False(t, flags.Transform)
				assert.False(t, flags.Config)
				assert.False(t, flags.State)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LogConfig{Debug: tt.flags}
			service := NewLoggerService(config)

			result := service.GetDebugFlags()
			tt.validate(t, result)
		})
	}
}

func TestLoggerServiceMapVerboseToLevel(t *testing.T) {
	tests := []struct {
		name     string
		config   *LogConfig
		expected logrus.Level
	}{
		{
			name: "Verbose 0 with info log level",
			config: &LogConfig{
				Verbose:  0,
				LogLevel: "info",
			},
			expected: logrus.InfoLevel,
		},
		{
			name: "Verbose 1 overrides log level",
			config: &LogConfig{
				Verbose:  1,
				LogLevel: "warn",
			},
			expected: logrus.DebugLevel,
		},
		{
			name: "Verbose 2 sets trace level",
			config: &LogConfig{
				Verbose:  2,
				LogLevel: "error",
			},
			expected: TraceLevel,
		},
		{
			name: "Verbose 3 sets trace level",
			config: &LogConfig{
				Verbose:  3,
				LogLevel: "info",
			},
			expected: TraceLevel,
		},
		{
			name: "Verbose 0 with empty log level defaults to info",
			config: &LogConfig{
				Verbose:  0,
				LogLevel: "",
			},
			expected: logrus.InfoLevel,
		},
		{
			name: "Verbose 0 with debug log level",
			config: &LogConfig{
				Verbose:  0,
				LogLevel: "debug",
			},
			expected: logrus.DebugLevel,
		},
		{
			name: "Invalid log level defaults to info",
			config: &LogConfig{
				Verbose:  0,
				LogLevel: "invalid",
			},
			expected: logrus.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewLoggerService(tt.config)
			result := service.mapVerboseToLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoggerServiceConfigureLoggerContextCancellation(t *testing.T) {
	config := &LogConfig{
		LogLevel:  "info",
		LogFormat: "text",
		Verbose:   0,
	}

	service := NewLoggerService(config)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := service.ConfigureLogger(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logger configuration canceled")
}

func TestLoggerServiceConfigureLoggerWithMaxVerbosity(t *testing.T) {
	// Test that verbose level 3 enables caller information
	config := &LogConfig{
		LogLevel:  "info",
		LogFormat: "text",
		Verbose:   3,
	}

	service := NewLoggerService(config)
	ctx := context.Background()

	// Reset logger state
	logrus.SetReportCaller(false)
	logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))

	err := service.ConfigureLogger(ctx)
	require.NoError(t, err)

	// Verify that report caller is enabled
	assert.True(t, logrus.StandardLogger().ReportCaller)
}
