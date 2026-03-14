// Package logging provides logging configuration and utilities for go-broadcast.
package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStructuredFormatter(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates new structured formatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewStructuredFormatter()

			assert.NotNil(t, formatter, "structured formatter should not be nil")
			assert.Equal(t, time.RFC3339, formatter.TimestampFormat, "should use RFC3339 timestamp format")
			assert.False(t, formatter.DisableTimestamp, "timestamps should be enabled by default")
		})
	}
}

func TestStructuredFormatter_Format(t *testing.T) {
	formatter := NewStructuredFormatter()

	tests := []struct {
		name     string
		entry    *logrus.Entry
		validate func(t *testing.T, output []byte)
	}{
		{
			name: "basic log entry",
			entry: &logrus.Entry{
				Time:    time.Date(2024, 1, 15, 15, 4, 5, 123000000, time.UTC),
				Level:   logrus.InfoLevel,
				Message: "Test message",
				Data:    logrus.Fields{},
			},
			validate: func(t *testing.T, output []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(output, &result)
				require.NoError(t, err, "output should be valid JSON")

				assert.Equal(t, "2024-01-15T15:04:05Z", result["@timestamp"], "timestamp should be in RFC3339 format")
				assert.Equal(t, "info", result["level"], "level should be correct")
				assert.Equal(t, "Test message", result["message"], "message should be correct")
			},
		},
		{
			name: "log entry with fields",
			entry: &logrus.Entry{
				Time:    time.Date(2024, 1, 15, 15, 4, 5, 0, time.UTC),
				Level:   logrus.DebugLevel,
				Message: "Debug message",
				Data: logrus.Fields{
					"component":      "test",
					"operation":      "format_test",
					"duration_ms":    150,
					"correlation_id": "test-123",
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(output, &result)
				require.NoError(t, err, "output should be valid JSON")

				assert.Equal(t, "debug", result["level"], "level should be correct")
				assert.Equal(t, "Debug message", result["message"], "message should be correct")
				assert.Equal(t, "test", result["component"], "component field should be preserved")
				assert.Equal(t, "format_test", result["operation"], "operation field should be preserved")
				assert.InDelta(t, float64(150), result["duration_ms"], 0.001, "duration_ms should be preserved as number")
				assert.Equal(t, "test-123", result["correlation_id"], "correlation_id should be preserved")
			},
		},
		{
			name: "log entry with standardized fields",
			entry: &logrus.Entry{
				Time:    time.Date(2024, 1, 15, 15, 4, 5, 456000000, time.UTC),
				Level:   logrus.ErrorLevel,
				Message: "Error occurred",
				Data: logrus.Fields{
					StandardFields.Timestamp:     "2024-01-15T15:04:05.456Z",
					StandardFields.Component:     "sync-engine",
					StandardFields.CorrelationID: "error-456",
					StandardFields.Error:         "connection timeout",
					StandardFields.DurationMs:    5000,
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(output, &result)
				require.NoError(t, err, "output should be valid JSON")

				// Check that standardized fields are present
				assert.Equal(t, "2024-01-15T15:04:05Z", result["@timestamp"], "timestamp should be standardized")
				assert.Equal(t, "error", result["level"], "level should be correct")
				assert.Equal(t, "Error occurred", result["message"], "message should be correct")
				assert.Equal(t, "sync-engine", result["component"], "component should be preserved")
				assert.Equal(t, "error-456", result["correlation_id"], "correlation_id should be preserved")
				assert.Equal(t, "connection timeout", result["error"], "error should be preserved")
				assert.InDelta(t, float64(5000), result["duration_ms"], 0.001, "duration_ms should be preserved")
			},
		},
		{
			name: "log entry with nested fields",
			entry: &logrus.Entry{
				Time:    time.Date(2024, 1, 15, 15, 4, 5, 0, time.UTC),
				Level:   logrus.WarnLevel,
				Message: "Warning message",
				Data: logrus.Fields{
					"config": map[string]interface{}{
						"path":  "/path/to/config",
						"valid": true,
					},
					"metrics": map[string]interface{}{
						"count": 42,
						"rate":  1.5,
					},
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result map[string]interface{}
				err := json.Unmarshal(output, &result)
				require.NoError(t, err, "output should be valid JSON")

				assert.Equal(t, "warning", result["level"], "level should be correct")
				assert.Equal(t, "Warning message", result["message"], "message should be correct")

				// Check nested config object
				config, ok := result["config"].(map[string]interface{})
				assert.True(t, ok, "config should be a nested object")
				assert.Equal(t, "/path/to/config", config["path"], "nested config path should be preserved")
				assert.Equal(t, true, config["valid"], "nested config valid should be preserved")

				// Check nested metrics object
				metrics, ok := result["metrics"].(map[string]interface{})
				assert.True(t, ok, "metrics should be a nested object")
				assert.InDelta(t, float64(42), metrics["count"], 0.001, "nested metrics count should be preserved")
				assert.InEpsilon(t, 1.5, metrics["rate"], 0.001, "nested metrics rate should be preserved")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter.Format(tt.entry)
			require.NoError(t, err, "formatting should not return error")
			assert.NotEmpty(t, output, "output should not be empty")

			// Validate that output is valid JSON
			assert.True(t, json.Valid(output), "output should be valid JSON")

			// Run specific validation
			tt.validate(t, output)
		})
	}
}

func TestConfigureLogger(t *testing.T) {
	tests := []struct {
		name        string
		config      *LogConfig
		expectError bool
		validate    func(t *testing.T, logger *logrus.Logger)
	}{
		{
			name: "configure logger with text format",
			config: &LogConfig{
				LogFormat:     "text",
				LogLevel:      "info",
				Verbose:       0,
				CorrelationID: "test-123",
			},
			expectError: false,
			validate: func(_ *testing.T, logger *logrus.Logger) {
				assert.Equal(t, logrus.InfoLevel, logger.Level, "log level should be set correctly")
				_, ok := logger.Formatter.(*logrus.TextFormatter)
				assert.True(t, ok, "formatter should be TextFormatter for text format")
			},
		},
		{
			name: "configure logger with json format",
			config: &LogConfig{
				LogFormat:     "json",
				LogLevel:      "debug",
				Verbose:       0,
				CorrelationID: "test-456",
			},
			expectError: false,
			validate: func(_ *testing.T, logger *logrus.Logger) {
				assert.Equal(t, logrus.DebugLevel, logger.Level, "log level should be set correctly")
				_, ok := logger.Formatter.(*StructuredFormatter)
				assert.True(t, ok, "formatter should be StructuredFormatter for json format")
			},
		},
		{
			name: "configure logger with verbose level 1",
			config: &LogConfig{
				LogFormat:     "text",
				LogLevel:      "info",
				Verbose:       1,
				CorrelationID: "test-789",
			},
			expectError: false,
			validate: func(_ *testing.T, logger *logrus.Logger) {
				assert.Equal(t, logrus.DebugLevel, logger.Level, "verbose level 1 should set debug level")
			},
		},
		{
			name: "configure logger with verbose level 2",
			config: &LogConfig{
				LogFormat:     "json",
				LogLevel:      "info",
				Verbose:       2,
				CorrelationID: "test-abc",
			},
			expectError: false,
			validate: func(_ *testing.T, logger *logrus.Logger) {
				assert.Equal(t, logrus.TraceLevel, logger.Level, "verbose level 2 should set trace level")
			},
		},
		{
			name: "configure logger with verbose level 3",
			config: &LogConfig{
				LogFormat:     "text",
				LogLevel:      "warn",
				Verbose:       3,
				CorrelationID: "test-def",
			},
			expectError: false,
			validate: func(_ *testing.T, logger *logrus.Logger) {
				assert.Equal(t, logrus.TraceLevel, logger.Level, "verbose level 3+ should cap at trace level")
			},
		},
		{
			name: "configure logger with invalid log level",
			config: &LogConfig{
				LogFormat:     "text",
				LogLevel:      "invalid",
				Verbose:       0,
				CorrelationID: "test-ghi",
			},
			expectError: true,
			validate: func(_ *testing.T, _ *logrus.Logger) {
				// Should not be called when error is expected
			},
		},
		{
			name: "configure logger with empty correlation ID",
			config: &LogConfig{
				LogFormat:     "json",
				LogLevel:      "info",
				Verbose:       0,
				CorrelationID: "",
			},
			expectError: false,
			validate: func(_ *testing.T, logger *logrus.Logger) {
				assert.Equal(t, logrus.InfoLevel, logger.Level, "log level should be set correctly")
				_, ok := logger.Formatter.(*StructuredFormatter)
				assert.True(t, ok, "formatter should be StructuredFormatter for json format")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh logger for testing
			logger := logrus.New()
			logger.SetOutput(&bytes.Buffer{}) // Capture output

			err := ConfigureLogger(logger, tt.config)

			if tt.expectError {
				assert.Error(t, err, "expected error during logger configuration")
			} else {
				require.NoError(t, err, "logger configuration should not return error")
				tt.validate(t, logger)

				// Verify hooks are installed
				assert.NotEmpty(t, logger.Hooks, "logger should have hooks installed")

				// Verify redaction hook is present
				hasRedactionHook := false
				for _, hooks := range logger.Hooks {
					for _, hook := range hooks {
						if _, ok := hook.(*RedactionHook); ok {
							hasRedactionHook = true
							break
						}
					}
				}
				assert.True(t, hasRedactionHook, "redaction hook should be installed")
			}
		})
	}
}

func TestConfigureLogger_Integration(t *testing.T) {
	tests := []struct {
		name     string
		config   *LogConfig
		logFunc  func(logger *logrus.Logger)
		validate func(t *testing.T, output string)
	}{
		{
			name: "text format integration",
			config: &LogConfig{
				LogFormat:     "text",
				LogLevel:      "info",
				CorrelationID: "integration-test-1",
			},
			logFunc: func(logger *logrus.Logger) {
				logger.WithFields(logrus.Fields{
					"component": "test",
					"token":     "ghp_secret123",
				}).Info("Test message")
			},
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "Test message", "message should be in output")
				assert.Contains(t, output, "component=test", "fields should be in output")
				assert.Contains(t, output, "ghp_***REDACTED***", "token should be redacted")
				assert.NotContains(t, output, "ghp_secret123", "original token should not be in output")
			},
		},
		{
			name: "json format integration",
			config: &LogConfig{
				LogFormat:     "json",
				LogLevel:      "debug",
				CorrelationID: "integration-test-2",
			},
			logFunc: func(logger *logrus.Logger) {
				logger.WithFields(logrus.Fields{
					"component":   "test",
					"password":    "secret456",
					"duration_ms": 1500,
				}).Debug("Debug message")
			},
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result)
				require.NoError(t, err, "output should be valid JSON")

				assert.Equal(t, "debug", result["level"], "level should be correct")
				assert.Equal(t, "Debug message", result["message"], "message should be correct")
				assert.Equal(t, "test", result["component"], "component should be preserved")
				assert.Equal(t, "***REDACTED***", result["password"], "password should be redacted")
				assert.InDelta(t, float64(1500), result["duration_ms"], 0.001, "duration should be preserved")
				assert.Contains(t, result, "@timestamp", "timestamp should be present")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create logger with buffer to capture output
			var buffer bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&buffer)

			// Configure logger
			err := ConfigureLogger(logger, tt.config)
			require.NoError(t, err, "logger configuration should succeed")

			// Execute logging function
			tt.logFunc(logger)

			// Validate output
			output := buffer.String()
			assert.NotEmpty(t, output, "should have logged output")
			tt.validate(t, output)
		})
	}
}

func TestWithStandardFields(t *testing.T) {
	tests := []struct {
		name      string
		logger    *logrus.Entry
		component string
		config    *LogConfig
		validate  func(t *testing.T, result *logrus.Entry)
	}{
		{
			name:      "add standard fields to entry",
			logger:    logrus.NewEntry(logrus.New()),
			component: "test-component",
			config: &LogConfig{
				CorrelationID: "test-correlation-123",
			},
			validate: func(t *testing.T, result *logrus.Entry) {
				assert.Equal(t, "test-component", result.Data[StandardFields.Component], "component should be set")
				assert.Equal(t, "test-correlation-123", result.Data[StandardFields.CorrelationID], "correlation ID should be set")
			},
		},
		{
			name:      "handle nil config",
			logger:    logrus.NewEntry(logrus.New()),
			component: "nil-config-component",
			config:    nil,
			validate: func(t *testing.T, result *logrus.Entry) {
				assert.Equal(t, "nil-config-component", result.Data[StandardFields.Component], "component should be set even with nil config")
				// Should not crash or add correlation ID when config is nil
				_, hasCorrelationID := result.Data[StandardFields.CorrelationID]
				assert.False(t, hasCorrelationID, "correlation ID should not be set with nil config")
			},
		},
		{
			name:      "handle empty correlation ID",
			logger:    logrus.NewEntry(logrus.New()),
			component: "empty-correlation-component",
			config: &LogConfig{
				CorrelationID: "",
			},
			validate: func(t *testing.T, result *logrus.Entry) {
				assert.Equal(t, "empty-correlation-component", result.Data[StandardFields.Component], "component should be set")
				// Empty correlation ID should not be set
				_, hasCorrelationID := result.Data[StandardFields.CorrelationID]
				assert.False(t, hasCorrelationID, "empty correlation ID should not be set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithStandardFields(tt.logger.Logger, tt.config, tt.component)

			assert.NotNil(t, result, "result should not be nil")
			tt.validate(t, result)
		})
	}
}

func TestWithStandardFields_ContextPropagation(t *testing.T) {
	// Test that standard fields work correctly in a full logging context
	var buffer bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buffer)
	logger.SetLevel(logrus.DebugLevel)

	config := &LogConfig{
		LogFormat:     "json",
		CorrelationID: "context-test-789",
	}

	err := ConfigureLogger(logger, config)
	require.NoError(t, err, "logger configuration should succeed")

	// Create base entry and add standard fields
	entry := WithStandardFields(logger, config, "context-test-component")

	// Log a message
	entry.WithField("operation", "context_test").Info("Context test message")

	// Validate output
	output := buffer.String()
	assert.NotEmpty(t, output, "should have logged output")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &result)
	require.NoError(t, err, "output should be valid JSON")

	assert.Equal(t, "info", result["level"], "level should be correct")
	assert.Equal(t, "Context test message", result["message"], "message should be correct")
	assert.Equal(t, "context-test-component", result["component"], "component should be from standard fields")
	assert.Equal(t, "context-test-789", result["correlation_id"], "correlation ID should be from standard fields")
	assert.Equal(t, "context_test", result["operation"], "operation should be from additional field")
	assert.Contains(t, result, "@timestamp", "timestamp should be present")
}
