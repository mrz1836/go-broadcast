package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save current working directory
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWD) }()

	// Change to repository root for test
	err = os.Chdir("../../../..")
	require.NoError(t, err)

	// Test loading configuration
	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify some expected values
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, int64(10*1024*1024), cfg.MaxFileSize)
	assert.Equal(t, 100, cfg.MaxFilesOpen)
	assert.Equal(t, 300, cfg.Timeout)

	// Check that checks are enabled by default
	assert.True(t, cfg.Checks.Fumpt)
	assert.True(t, cfg.Checks.Lint)
	assert.True(t, cfg.Checks.ModTidy)
	assert.True(t, cfg.Checks.Whitespace)
	assert.True(t, cfg.Checks.EOF)
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"true value", "true", false, true},
		{"false value", "false", true, false},
		{"empty value", "", true, true},
		{"invalid value", "invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_BOOL", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_BOOL") }()

			result := getBoolEnv("TEST_BOOL", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"valid int", "42", 0, 42},
		{"empty value", "", 10, 10},
		{"invalid value", "abc", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_INT", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_INT") }()

			result := getIntEnv("TEST_INT", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{"value set", "test", "default", "test"},
		{"empty value", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_STRING", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_STRING") }()

			result := getStringEnv("TEST_STRING", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
