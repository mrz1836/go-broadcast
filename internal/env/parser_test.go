package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEnvFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("parses simple key-value pairs", func(t *testing.T) {
		content := `KEY1=value1
KEY2=value2`
		file := filepath.Join(tempDir, "simple.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Equal(t, "value1", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
	})

	t.Run("ignores comments and empty lines", func(t *testing.T) {
		content := `# This is a comment
KEY1=value1

# Another comment
KEY2=value2
`
		file := filepath.Join(tempDir, "comments.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Len(t, vars, 2)
		assert.Equal(t, "value1", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
	})

	t.Run("handles double quoted values", func(t *testing.T) {
		content := `KEY1="value with spaces"
KEY2="quoted value"`
		file := filepath.Join(tempDir, "double_quoted.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Equal(t, "value with spaces", vars["KEY1"])
		assert.Equal(t, "quoted value", vars["KEY2"])
	})

	t.Run("handles single quoted values", func(t *testing.T) {
		content := `KEY1='single quoted'
KEY2='another single'`
		file := filepath.Join(tempDir, "single_quoted.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Equal(t, "single quoted", vars["KEY1"])
		assert.Equal(t, "another single", vars["KEY2"])
	})

	t.Run("handles unquoted values", func(t *testing.T) {
		content := `KEY1=unquoted
KEY2=another_value`
		file := filepath.Join(tempDir, "unquoted.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Equal(t, "unquoted", vars["KEY1"])
		assert.Equal(t, "another_value", vars["KEY2"])
	})

	t.Run("handles empty values", func(t *testing.T) {
		content := `KEY1=
KEY2=""`
		file := filepath.Join(tempDir, "empty.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Empty(t, vars["KEY1"])
		assert.Empty(t, vars["KEY2"])
	})

	t.Run("handles values with equals sign", func(t *testing.T) {
		content := `KEY1=foo=bar
KEY2=a=b=c=d`
		file := filepath.Join(tempDir, "equals.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Equal(t, "foo=bar", vars["KEY1"])
		assert.Equal(t, "a=b=c=d", vars["KEY2"])
	})

	t.Run("strips inline comments for unquoted values", func(t *testing.T) {
		// Inline comments (space followed by #) are stripped for unquoted values
		content := `KEY1=value # this is a comment
KEY2=      # empty value with comment
KEY3=value#no-space-no-strip
KEY4="value # not stripped because quoted"
KEY5='value # also not stripped single quoted'
KEY6=https://example.com # url with comment`
		file := filepath.Join(tempDir, "inline_comment.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)

		// Inline comment after space is stripped
		assert.Equal(t, "value", vars["KEY1"])
		// Empty value with comment becomes empty string
		assert.Empty(t, vars["KEY2"])
		// No space before # means it's part of the value
		assert.Equal(t, "value#no-space-no-strip", vars["KEY3"])
		// Quoted values preserve the # even with space
		assert.Equal(t, "value # not stripped because quoted", vars["KEY4"])
		assert.Equal(t, "value # also not stripped single quoted", vars["KEY5"])
		// URL with comment - comment part stripped
		assert.Equal(t, "https://example.com", vars["KEY6"])
	})

	t.Run("handles complex real-world example", func(t *testing.T) {
		content := `# Database Configuration
DATABASE_URL=postgres://user:pass@localhost/db

# API Keys (DO NOT COMMIT REAL VALUES)
# API_KEY=

# Feature Flags
FEATURE_ENABLED=true
DEBUG_MODE=false

# Paths with special characters
LOG_PATH=/var/log/app.log
CONFIG_PATH="./config/settings.json"`
		file := filepath.Join(tempDir, "complex.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)

		assert.Equal(t, "postgres://user:pass@localhost/db", vars["DATABASE_URL"])
		assert.Equal(t, "true", vars["FEATURE_ENABLED"])
		assert.Equal(t, "false", vars["DEBUG_MODE"])
		assert.Equal(t, "/var/log/app.log", vars["LOG_PATH"])
		assert.Equal(t, "./config/settings.json", vars["CONFIG_PATH"])

		// Commented out key should not be present
		_, exists := vars["API_KEY"]
		assert.False(t, exists, "Commented out API_KEY should not be parsed")
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := parseEnvFile("/nonexistent/path/file.env")
		require.Error(t, err)
	})

	t.Run("handles whitespace around key and value", func(t *testing.T) {
		content := `  KEY1  =  value1
	KEY2	=	value2	`
		file := filepath.Join(tempDir, "whitespace.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Equal(t, "value1", vars["KEY1"])
		assert.Equal(t, "value2", vars["KEY2"])
	})
}

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantOk    bool
	}{
		{
			name:      "simple key-value",
			line:      "KEY=value",
			wantKey:   "KEY",
			wantValue: "value",
			wantOk:    true,
		},
		{
			name:      "empty value",
			line:      "KEY=",
			wantKey:   "KEY",
			wantValue: "",
			wantOk:    true,
		},
		{
			name:      "double quoted value",
			line:      `KEY="quoted"`,
			wantKey:   "KEY",
			wantValue: "quoted",
			wantOk:    true,
		},
		{
			name:      "single quoted value",
			line:      "KEY='single'",
			wantKey:   "KEY",
			wantValue: "single",
			wantOk:    true,
		},
		{
			name:      "whitespace around key and value",
			line:      "  KEY = value  ",
			wantKey:   "KEY",
			wantValue: "value",
			wantOk:    true,
		},
		{
			name:      "value with equals sign",
			line:      "KEY=foo=bar",
			wantKey:   "KEY",
			wantValue: "foo=bar",
			wantOk:    true,
		},
		{
			name:      "no equals sign",
			line:      "no_equals_sign",
			wantKey:   "",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "empty key",
			line:      "=no_key",
			wantKey:   "",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "underscore in key",
			line:      "MY_VAR=value",
			wantKey:   "MY_VAR",
			wantValue: "value",
			wantOk:    true,
		},
		{
			name:      "numbers in key",
			line:      "VAR123=value",
			wantKey:   "VAR123",
			wantValue: "value",
			wantOk:    true,
		},
		{
			name:      "url value",
			line:      "DATABASE_URL=postgres://user:pass@localhost:5432/db",
			wantKey:   "DATABASE_URL",
			wantValue: "postgres://user:pass@localhost:5432/db",
			wantOk:    true,
		},
		{
			name:      "path value",
			line:      "CONFIG_PATH=/etc/app/config.yaml",
			wantKey:   "CONFIG_PATH",
			wantValue: "/etc/app/config.yaml",
			wantOk:    true,
		},
		{
			name:      "boolean true",
			line:      "ENABLED=true",
			wantKey:   "ENABLED",
			wantValue: "true",
			wantOk:    true,
		},
		{
			name:      "boolean false",
			line:      "DISABLED=false",
			wantKey:   "DISABLED",
			wantValue: "false",
			wantOk:    true,
		},
		{
			name:      "numeric value",
			line:      "PORT=8080",
			wantKey:   "PORT",
			wantValue: "8080",
			wantOk:    true,
		},
		{
			name:      "comma separated value",
			line:      "LABELS=label1,label2,label3",
			wantKey:   "LABELS",
			wantValue: "label1,label2,label3",
			wantOk:    true,
		},
		{
			name:      "quoted value with spaces",
			line:      `MESSAGE="Hello, World!"`,
			wantKey:   "MESSAGE",
			wantValue: "Hello, World!",
			wantOk:    true,
		},
		{
			name:      "inline comment stripped",
			line:      "KEY=value # this is a comment",
			wantKey:   "KEY",
			wantValue: "value",
			wantOk:    true,
		},
		{
			name:      "empty value with inline comment",
			line:      "KEY=      # comment only",
			wantKey:   "KEY",
			wantValue: "",
			wantOk:    true,
		},
		{
			name:      "hash without space is not comment",
			line:      "KEY=value#hashtag",
			wantKey:   "KEY",
			wantValue: "value#hashtag",
			wantOk:    true,
		},
		{
			name:      "quoted value preserves hash",
			line:      `KEY="value # with hash"`,
			wantKey:   "KEY",
			wantValue: "value # with hash",
			wantOk:    true,
		},
		{
			name:      "single quoted value preserves hash",
			line:      `KEY='value # with hash'`,
			wantKey:   "KEY",
			wantValue: "value # with hash",
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, ok := parseEnvLine(tt.line)
			assert.Equal(t, tt.wantOk, ok, "ok mismatch")
			if ok {
				assert.Equal(t, tt.wantKey, key, "key mismatch")
				assert.Equal(t, tt.wantValue, value, "value mismatch")
			}
		})
	}
}

// TestParseEnvFileGoFortressCompatibility tests parsing of GoFortress .env.base format
func TestParseEnvFileGoFortressCompatibility(t *testing.T) {
	tempDir := t.TempDir()

	// Simulate a realistic GoFortress .env.base structure with inline comments
	content := `# ================================================================================================
#  GoFortress Configuration
# ================================================================================================

# Coverage Configuration
GO_COVERAGE_EXCLUDE_PATHS=test/,vendor/,testdata/

# Skip URL Checks
GO_COVERAGE_SKIP_URL_CHECKS=true

# Google Analytics
GOOGLE_ANALYTICS_ID=G-VKFVWG6GXM

# AI Generation Configuration
GO_BROADCAST_AI_ENABLED=false      # Default to disabled
GO_BROADCAST_AI_PROVIDER=anthropic # AI provider to use
GO_BROADCAST_AI_PR_ENABLED=        # Enable AI for PR body generation
GO_BROADCAST_AI_COMMIT_ENABLED=    # Enable AI for commit message generation
# GO_BROADCAST_AI_API_KEY= DO NOT SET IN FILES

# Automerge Labels
GO_BROADCAST_AUTOMERGE_LABELS=automerge`

	file := filepath.Join(tempDir, ".env.base")
	require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

	vars, err := parseEnvFile(file)
	require.NoError(t, err)

	// Verify expected values
	assert.Equal(t, "test/,vendor/,testdata/", vars["GO_COVERAGE_EXCLUDE_PATHS"])
	assert.Equal(t, "true", vars["GO_COVERAGE_SKIP_URL_CHECKS"])
	assert.Equal(t, "G-VKFVWG6GXM", vars["GOOGLE_ANALYTICS_ID"])
	assert.Equal(t, "automerge", vars["GO_BROADCAST_AUTOMERGE_LABELS"])

	// Verify inline comments are stripped
	assert.Equal(t, "false", vars["GO_BROADCAST_AI_ENABLED"], "Inline comment should be stripped")
	assert.Equal(t, "anthropic", vars["GO_BROADCAST_AI_PROVIDER"], "Inline comment should be stripped")

	// Verify empty values with inline comments become empty strings
	assert.Empty(t, vars["GO_BROADCAST_AI_PR_ENABLED"], "Empty value with comment should be empty")
	assert.Empty(t, vars["GO_BROADCAST_AI_COMMIT_ENABLED"], "Empty value with comment should be empty")

	// Verify commented API key is NOT parsed
	_, exists := vars["GO_BROADCAST_AI_API_KEY"]
	assert.False(t, exists, "Commented out API key should not be parsed")
}
