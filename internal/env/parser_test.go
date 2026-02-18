package env

import (
	"os"
	"path/filepath"
	"strings"
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
		testDBURL := "postgres://user:pass@localhost/db" //nolint:gosec // G101: fake test credential, not a real password
		content := "# Database Configuration\nDATABASE_URL=" + testDBURL + "\n\n" +
			"# API Keys (DO NOT COMMIT REAL VALUES)\n# API_KEY=\n\n" +
			"# Feature Flags\nFEATURE_ENABLED=true\nDEBUG_MODE=false\n\n" +
			"# Paths with special characters\nLOG_PATH=/var/log/app.log\n" +
			"CONFIG_PATH=\"./config/settings.json\""
		file := filepath.Join(tempDir, "complex.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)

		assert.Equal(t, testDBURL, vars["DATABASE_URL"])
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
		{ //nolint:gosec // G101: fake test credential in URL, not a real password
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

// TestExportPrefix tests that shell-style "export KEY=value" format is handled correctly.
// This is the fix for Issue #4.
func TestExportPrefix(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("parses export prefix in file", func(t *testing.T) {
		content := `export VAR1=value1
export VAR2="quoted value"
export VAR3='single quoted'
VAR4=no_export
export VAR5=`
		file := filepath.Join(tempDir, "export.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)

		assert.Equal(t, "value1", vars["VAR1"], "export prefix should be stripped")
		assert.Equal(t, "quoted value", vars["VAR2"], "export with quoted value")
		assert.Equal(t, "single quoted", vars["VAR3"], "export with single quoted value")
		assert.Equal(t, "no_export", vars["VAR4"], "line without export")
		assert.Empty(t, vars["VAR5"], "export with empty value")

		// Verify "export" is NOT part of the key
		_, exists := vars["export VAR1"]
		assert.False(t, exists, "key should not include 'export ' prefix")
	})

	t.Run("parseEnvLine handles export prefix", func(t *testing.T) {
		tests := []struct {
			line      string
			wantKey   string
			wantValue string
			wantOk    bool
		}{
			{"export FOO=bar", "FOO", "bar", true},
			{"export FOO=", "FOO", "", true},
			{"export FOO=\"quoted\"", "FOO", "quoted", true},
			{"export FOO='single'", "FOO", "single", true},
			{"exportFOO=bar", "exportFOO", "bar", true}, // no space = not export prefix
			{"export=value", "export", "value", true},   // "export" as key name
		}

		for _, tt := range tests {
			t.Run(tt.line, func(t *testing.T) {
				key, value, ok := parseEnvLine(tt.line)
				assert.Equal(t, tt.wantOk, ok)
				if ok {
					assert.Equal(t, tt.wantKey, key)
					assert.Equal(t, tt.wantValue, value)
				}
			})
		}
	})
}

// TestUnmatchedQuotes tests behavior when quotes are not properly closed.
// This documents Issue #5 behavior: unmatched quotes are preserved as-is.
func TestUnmatchedQuotes(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantOk    bool
	}{
		{
			name:      "unmatched double quote at start",
			line:      `KEY="unmatched`,
			wantKey:   "KEY",
			wantValue: `"unmatched`, // preserved with leading quote
			wantOk:    true,
		},
		{
			name:      "unmatched single quote at start",
			line:      `KEY='unmatched`,
			wantKey:   "KEY",
			wantValue: `'unmatched`, // preserved with leading quote
			wantOk:    true,
		},
		{
			name:      "unmatched double quote at end",
			line:      `KEY=unmatched"`,
			wantKey:   "KEY",
			wantValue: `unmatched"`, // preserved with trailing quote
			wantOk:    true,
		},
		{
			name:      "mismatched quotes",
			line:      `KEY="value'`,
			wantKey:   "KEY",
			wantValue: `"value'`, // preserved as-is
			wantOk:    true,
		},
		{
			name:      "single char double quote",
			line:      `KEY="`,
			wantKey:   "KEY",
			wantValue: `"`, // single quote preserved
			wantOk:    true,
		},
		{
			name:      "properly matched quotes",
			line:      `KEY="matched"`,
			wantKey:   "KEY",
			wantValue: "matched", // quotes stripped
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

// TestLongLine tests that lines within the buffer limit are handled correctly
// and lines exceeding the limit cause an error. This tests Issue #6.
func TestLongLine(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("handles line at buffer limit", func(t *testing.T) {
		// Create a line just under 100KB (well within limit)
		longValue := strings.Repeat("x", 100*1024)
		content := "LONG_KEY=" + longValue
		file := filepath.Join(tempDir, "long_ok.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Equal(t, longValue, vars["LONG_KEY"])
	})

	t.Run("handles multiple normal lines after long line", func(t *testing.T) {
		// Ensure parsing continues correctly after a long line
		longValue := strings.Repeat("y", 50*1024)
		content := "FIRST=first\nLONG_KEY=" + longValue + "\nLAST=last"
		file := filepath.Join(tempDir, "long_middle.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Equal(t, "first", vars["FIRST"])
		assert.Equal(t, longValue, vars["LONG_KEY"])
		assert.Equal(t, "last", vars["LAST"])
	})

	t.Run("errors on line exceeding max length", func(t *testing.T) {
		// Create a line exceeding MaxLineLength (1MB)
		tooLongValue := strings.Repeat("z", MaxLineLength+100)
		content := "TOO_LONG=" + tooLongValue
		file := filepath.Join(tempDir, "too_long.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		_, err := parseEnvFile(file)
		require.Error(t, err, "should error on line exceeding max length")
		assert.Contains(t, err.Error(), "too long", "error should mention line too long")
	})
}

// TestScannerError tests that scanner errors are properly returned.
// This tests Issue #7 - the scanner.Err() return path.
func TestScannerError(t *testing.T) {
	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := parseEnvFile("/nonexistent/path/to/file.env")
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err), "should return not-exist error")
	})

	t.Run("returns error for directory instead of file", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := parseEnvFile(tempDir) // try to read a directory
		require.Error(t, err)
	})

	t.Run("handles empty file without error", func(t *testing.T) {
		tempDir := t.TempDir()
		file := filepath.Join(tempDir, "empty.env")
		require.NoError(t, os.WriteFile(file, []byte(""), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Empty(t, vars, "empty file should produce empty map")
	})

	t.Run("handles file with only comments", func(t *testing.T) {
		tempDir := t.TempDir()
		content := "# comment 1\n# comment 2\n# comment 3"
		file := filepath.Join(tempDir, "comments_only.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Empty(t, vars, "file with only comments should produce empty map")
	})

	t.Run("handles file with only empty lines", func(t *testing.T) {
		tempDir := t.TempDir()
		content := "\n\n\n\n"
		file := filepath.Join(tempDir, "empty_lines.env")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		vars, err := parseEnvFile(file)
		require.NoError(t, err)
		assert.Empty(t, vars, "file with only empty lines should produce empty map")
	})
}
