// Package logging provides logging configuration and utilities for go-broadcast.
package logging

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedactionService(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates new redaction service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewRedactionService()

			require.NotNil(t, service, "redaction service should not be nil")
			require.NotNil(t, service.sensitivePatterns, "sensitive patterns should be initialized")
			require.NotNil(t, service.sensitiveFields, "sensitive fields should be initialized")
			assert.NotEmpty(t, service.sensitivePatterns, "should have sensitive patterns")
			assert.NotEmpty(t, service.sensitiveFields, "should have sensitive fields")
		})
	}
}

func TestRedactionService_RedactSensitive(t *testing.T) {
	service := NewRedactionService()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "github token ghp_",
			input:    "Using token ghp_1234567890abcdefghijklmnopqrstuvwxyz123456",
			expected: "Using token ghp_***REDACTED***",
		},
		{
			name:     "github token ghs_",
			input:    "Server token ghs_abcdefghijklmnopqrstuvwxyz1234567890123456",
			expected: "Server token ghs_***REDACTED***",
		},
		{
			name:     "github pat token",
			input:    "PAT: github_pat_11ABCDEFGHIJKLMNOPQRSTUVWXYZ_1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "PAT: github_pat_***REDACTED***",
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWI",
			expected: "Authorization: Bearer ***REDACTED***",
		},
		{
			name:     "jwt token",
			input:    "JWT eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ",
			expected: "JWT ***REDACTED***",
		},
		{
			name:     "ssh private key",
			input:    "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAFwAAAAdzc2gtcn\n-----END OPENSSH PRIVATE KEY-----",
			expected: "***REDACTED_SSH_KEY***",
		},
		{
			name:     "base64 secret",
			input:    "secret=YWJjZGVmZ2hpams1bG1ub3BxcnN0dXZ3eHl6MTIzNDU2Nzg5MA==",
			expected: "secret=***REDACTED***",
		},
		{
			name:     "multiple tokens in same string",
			input:    "git clone https://ghp_token1@github.com/repo.git && curl -H 'Authorization: Bearer jwt_token2'",
			expected: "git clone https://ghp_***REDACTED***@github.com/repo.git && curl -H 'Authorization: Bearer ***REDACTED***'",
		},
		{
			name:     "no sensitive data",
			input:    "This is a normal log message with no secrets",
			expected: "This is a normal log message with no secrets",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "url with password",
			input:    "https://user:password123@github.com/repo.git",
			expected: "https://user:***REDACTED***@github.com/repo.git",
		},
		{
			name:     "api key in url",
			input:    "https://api.service.com/data?api_key=abc123def456&other=value",
			expected: "https://api.service.com/data?api_key=***REDACTED***&other=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.RedactSensitive(tt.input)
			assert.Equal(t, tt.expected, result, "redacted string should match expected")
		})
	}
}

func TestRedactionService_RedactLogEntry(t *testing.T) {
	service := NewRedactionService()

	tests := []struct {
		name     string
		entry    *logrus.Entry
		expected map[string]interface{}
	}{
		{
			name: "redact token in message",
			entry: &logrus.Entry{
				Message: "Using token ghp_1234567890abcdefghijklmnopqrstuvwxyz123456",
				Data:    logrus.Fields{},
			},
			expected: map[string]interface{}{
				"message": "Using token ghp_***REDACTED***",
			},
		},
		{
			name: "redact sensitive field values",
			entry: &logrus.Entry{
				Message: "Processing request",
				Data: logrus.Fields{
					"password":     "secret123",
					"token":        "ghp_abcdefghijklmnop",
					"api_key":      "key_123456789",
					"github_token": "ghs_987654321",
					"normal_field": "normal_value",
				},
			},
			expected: map[string]interface{}{
				"message":      "Processing request",
				"password":     "***REDACTED***",
				"token":        "ghp_***REDACTED***",
				"api_key":      "***REDACTED***",
				"github_token": "ghs_***REDACTED***",
				"normal_field": "normal_value",
			},
		},
		{
			name: "redact nested field names",
			entry: &logrus.Entry{
				Message: "Configuration loaded",
				Data: logrus.Fields{
					"config": map[string]interface{}{
						"database_password": "db_secret_123",
						"api_secret":        "api_secret_456",
						"normal_setting":    "value",
					},
					"user": "testuser",
				},
			},
			expected: map[string]interface{}{
				"message": "Configuration loaded",
				"config": map[string]interface{}{
					"database_password": "***REDACTED***",
					"api_secret":        "***REDACTED***",
					"normal_setting":    "value",
				},
				"user": "testuser",
			},
		},
		{
			name: "no sensitive data",
			entry: &logrus.Entry{
				Message: "Normal log message",
				Data: logrus.Fields{
					"operation": "sync",
					"duration":  "1.5s",
					"status":    "completed",
				},
			},
			expected: map[string]interface{}{
				"message":   "Normal log message",
				"operation": "sync",
				"duration":  "1.5s",
				"status":    "completed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the entry to avoid modifying the original test data
			entryCopy := &logrus.Entry{
				Message: tt.entry.Message,
				Data:    make(logrus.Fields),
			}
			for k, v := range tt.entry.Data {
				entryCopy.Data[k] = v
			}

			// Create and use hook for redaction
			hook := service.CreateHook()
			err := hook.Fire(entryCopy)
			require.NoError(t, err, "hook should not return error")

			// Check message redaction
			assert.Equal(t, tt.expected["message"], entryCopy.Message, "message should be redacted correctly")

			// Check field redaction
			for key, expectedValue := range tt.expected {
				if key == "message" {
					continue // Already checked above
				}

				actualValue, exists := entryCopy.Data[key]
				require.True(t, exists, "field %s should exist in redacted entry", key)
				assert.Equal(t, expectedValue, actualValue, "field %s should be redacted correctly", key)
			}
		})
	}
}

func TestRedactionService_IsSensitiveField(t *testing.T) {
	service := NewRedactionService()

	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{
			name:      "password field",
			fieldName: "password",
			expected:  true,
		},
		{
			name:      "token field",
			fieldName: "token",
			expected:  true,
		},
		{
			name:      "api_key field",
			fieldName: "api_key",
			expected:  true,
		},
		{
			name:      "github_token field",
			fieldName: "github_token",
			expected:  true,
		},
		{
			name:      "normal field",
			fieldName: "operation",
			expected:  false,
		},
		{
			name:      "case insensitive - PASSWORD",
			fieldName: "PASSWORD",
			expected:  true,
		},
		{
			name:      "substring match - user_password",
			fieldName: "user_password",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsSensitiveField(tt.fieldName)
			assert.Equal(t, tt.expected, result, "sensitive field detection should match expected")
		})
	}
}

func TestRedactionService_CreateHook(t *testing.T) {
	service := NewRedactionService()

	hook := service.CreateHook()

	require.NotNil(t, hook, "redaction hook should not be nil")
	assert.Equal(t, logrus.AllLevels, hook.Levels(), "hook should apply to all log levels")
}

func TestRedactionHook_Levels(t *testing.T) {
	service := NewRedactionService()
	hook := service.CreateHook()

	levels := hook.Levels()

	// Should apply to all log levels
	expectedLevels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}

	assert.Equal(t, expectedLevels, levels, "hook should apply to all log levels")
}

func TestRedactionHook_Fire(t *testing.T) {
	service := NewRedactionService()
	hook := service.CreateHook()

	tests := []struct {
		name        string
		entry       *logrus.Entry
		expectError bool
	}{
		{
			name: "successful redaction",
			entry: &logrus.Entry{
				Message: "Using token ghp_123456789",
				Data: logrus.Fields{
					"password": "secret",
					"normal":   "value",
				},
			},
			expectError: false,
		},
		{
			name: "entry with no sensitive data",
			entry: &logrus.Entry{
				Message: "Normal message",
				Data:    logrus.Fields{"status": "ok"},
			},
			expectError: false,
		},
		{
			name: "entry with nil data",
			entry: &logrus.Entry{
				Message: "Message with nil data",
				Data:    nil,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalMessage := tt.entry.Message

			err := hook.Fire(tt.entry)

			if tt.expectError {
				require.Error(t, err, "expected error during hook firing")
			} else {
				require.NoError(t, err, "hook should not return error")
			}

			// Verify that redaction occurred if there was sensitive data
			if originalMessage == "Using token ghp_123456789" {
				assert.Equal(t, "Using token ghp_***REDACTED***", tt.entry.Message, "message should be redacted")
			}
		})
	}
}

func TestNewAuditLogger(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates new audit logger",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewAuditLogger()

			require.NotNil(t, logger, "audit logger should not be nil")
		})
	}
}

func TestAuditLogger_LogAuthentication(t *testing.T) {
	logger := NewAuditLogger()

	tests := []struct {
		name    string
		success bool
		method  string
		user    string
	}{
		{
			name:    "successful authentication",
			success: true,
			method:  "github_token",
			user:    "testuser",
		},
		{
			name:    "failed authentication",
			success: false,
			method:  "github_cli",
			user:    "unknown",
		},
		{
			name:    "authentication with empty method",
			success: true,
			method:  "",
			user:    "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// This should not panic or error
			logger.LogAuthentication(tt.user, tt.method, tt.success)

			// Test passes if no panic occurs
		})
	}
}

func TestAuditLogger_LogConfigChange(t *testing.T) {
	logger := NewAuditLogger()

	tests := []struct {
		name   string
		action string
		path   string
		user   string
	}{
		{
			name:   "config load action",
			action: "load",
			path:   "/path/to/config.yaml",
			user:   "system",
		},
		{
			name:   "config update action",
			action: "update",
			path:   "/path/to/config.yaml",
			user:   "admin",
		},
		{
			name:   "config with empty path",
			action: "validate",
			path:   "",
			user:   "system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// This should not panic or error
			logger.LogConfigChange(tt.action, tt.path, tt.user)

			// Test passes if no panic occurs
		})
	}
}

func TestAuditLogger_LogRepositoryAccess(t *testing.T) {
	logger := NewAuditLogger()

	tests := []struct {
		name   string
		repo   string
		action string
		user   string
	}{
		{
			name:   "repository clone access",
			repo:   "owner/repo",
			action: "clone",
			user:   "github_cli",
		},
		{
			name:   "repository pr_create access",
			repo:   "owner/service",
			action: "pr_create",
			user:   "github_cli",
		},
		{
			name:   "access with empty repo",
			repo:   "",
			action: "access",
			user:   "system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// This should not panic or error
			logger.LogRepositoryAccess(tt.repo, tt.action, tt.user)

			// Test passes if no panic occurs
		})
	}
}
