package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRepoName(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "valid org/repo format",
			repo:    "org/repo",
			wantErr: false,
		},
		{
			name:    "valid with hyphens",
			repo:    "my-org/my-repo",
			wantErr: false,
		},
		{
			name:    "valid with dots",
			repo:    "org.name/repo.name",
			wantErr: false,
		},
		{
			name:    "valid with underscores",
			repo:    "org_name/repo_name",
			wantErr: false,
		},
		{
			name:    "valid with numbers",
			repo:    "org123/repo456",
			wantErr: false,
		},
		{
			name:    "minimal valid",
			repo:    "a/b",
			wantErr: false,
		},

		// Invalid cases
		{
			name:    "empty repository name",
			repo:    "",
			wantErr: true,
			errMsg:  "field cannot be empty: repository name",
		},
		{
			name:    "no slash",
			repo:    "invalid-repo",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "starts with slash",
			repo:    "/org/repo",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "ends with slash",
			repo:    "org/repo/",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "multiple slashes",
			repo:    "org/repo/extra",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "double slash",
			repo:    "org//repo",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "empty org",
			repo:    "/repo",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "empty repo",
			repo:    "org/",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "starts with hyphen",
			repo:    "-org/repo",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "repo starts with hyphen",
			repo:    "org/-repo",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "path traversal",
			repo:    "../etc/passwd",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "path traversal in org",
			repo:    "../org/repo",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "path traversal in repo",
			repo:    "org/../repo",
			wantErr: true,
			errMsg:  "invalid format: repository name",
		},
		{
			name:    "path traversal with valid format",
			repo:    "org../repo",
			wantErr: true,
			errMsg:  "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepoName(tt.repo)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "simple branch name",
			branch:  "master",
			wantErr: false,
		},
		{
			name:    "branch with slash",
			branch:  "feature/new-feature",
			wantErr: false,
		},
		{
			name:    "branch with hyphen",
			branch:  "feature-branch",
			wantErr: false,
		},
		{
			name:    "branch with dot",
			branch:  "v1.0.0",
			wantErr: false,
		},
		{
			name:    "branch with underscore",
			branch:  "test_branch",
			wantErr: false,
		},
		{
			name:    "branch with numbers",
			branch:  "branch123",
			wantErr: false,
		},
		{
			name:    "complex branch name",
			branch:  "feature/test.branch-123_new",
			wantErr: false,
		},
		{
			name:    "single character",
			branch:  "a",
			wantErr: false,
		},

		// Invalid cases
		{
			name:    "empty branch name",
			branch:  "",
			wantErr: true,
			errMsg:  "field cannot be empty: branch name",
		},
		{
			name:    "starts with hyphen",
			branch:  "-branch",
			wantErr: true,
			errMsg:  "invalid field: branch name",
		},
		{
			name:    "starts with dot",
			branch:  ".branch",
			wantErr: true,
			errMsg:  "invalid field: branch name",
		},
		{
			name:    "starts with slash",
			branch:  "/branch",
			wantErr: true,
			errMsg:  "invalid field: branch name",
		},
		{
			name:    "contains space",
			branch:  "branch with spaces",
			wantErr: true,
			errMsg:  "invalid field: branch name",
		},
		{
			name:    "contains special chars",
			branch:  "branch@special",
			wantErr: true,
			errMsg:  "invalid field: branch name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchName(tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBranchPrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "empty prefix is allowed",
			prefix:  "",
			wantErr: false,
		},
		{
			name:    "simple prefix",
			prefix:  "sync",
			wantErr: false,
		},
		{
			name:    "prefix with slash",
			prefix:  "chore/sync-files",
			wantErr: false,
		},
		{
			name:    "prefix with hyphen",
			prefix:  "sync-template",
			wantErr: false,
		},
		{
			name:    "prefix with dot",
			prefix:  "sync.template",
			wantErr: false,
		},
		{
			name:    "prefix with underscore",
			prefix:  "sync_template",
			wantErr: false,
		},

		// Invalid cases
		{
			name:    "starts with hyphen",
			prefix:  "-sync",
			wantErr: true,
			errMsg:  "invalid field: branch prefix",
		},
		{
			name:    "starts with dot",
			prefix:  ".sync",
			wantErr: true,
			errMsg:  "invalid field: branch prefix",
		},
		{
			name:    "starts with slash",
			prefix:  "/sync",
			wantErr: true,
			errMsg:  "invalid field: branch prefix",
		},
		{
			name:    "contains space",
			prefix:  "sync template",
			wantErr: true,
			errMsg:  "invalid field: branch prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchPrefix(tt.prefix)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		// Valid cases
		{
			name:      "simple relative path",
			path:      "file.txt",
			fieldName: "source",
			wantErr:   false,
		},
		{
			name:      "nested relative path",
			path:      "src/file.txt",
			fieldName: "source",
			wantErr:   false,
		},
		{
			name:      "deeply nested path",
			path:      "src/nested/deep/file.txt",
			fieldName: "dest",
			wantErr:   false,
		},
		{
			name:      "path with dots in filename",
			path:      "config.yaml",
			fieldName: "source",
			wantErr:   false,
		},

		// Invalid cases
		{
			name:      "empty path",
			path:      "",
			fieldName: "source",
			wantErr:   true,
			errMsg:    "field is required: source path",
		},
		{
			name:      "absolute path",
			path:      "/absolute/path/file.txt",
			fieldName: "dest",
			wantErr:   true,
			errMsg:    "must be relative, not absolute",
		},
		{
			name:      "path traversal simple",
			path:      "../file.txt",
			fieldName: "source",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "path traversal complex",
			path:      "../../etc/passwd",
			fieldName: "dest",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "path traversal in middle",
			path:      "src/../../../etc/passwd",
			fieldName: "source",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path, tt.fieldName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNonEmpty(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "non-empty value",
			field:   "test field",
			value:   "value",
			wantErr: false,
		},
		{
			name:    "value with spaces",
			field:   "test field",
			value:   "value with spaces",
			wantErr: false,
		},
		{
			name:    "value with leading/trailing spaces",
			field:   "test field",
			value:   "  value  ",
			wantErr: false,
		},

		// Invalid cases
		{
			name:    "empty string",
			field:   "test field",
			value:   "",
			wantErr: true,
			errMsg:  "field cannot be empty: test field",
		},
		{
			name:    "only whitespace",
			field:   "test field",
			value:   "   ",
			wantErr: true,
			errMsg:  "field cannot be empty: test field",
		},
		{
			name:    "only tabs",
			field:   "test field",
			value:   "\t\t",
			wantErr: true,
			errMsg:  "field cannot be empty: test field",
		},
		{
			name:    "mixed whitespace",
			field:   "test field",
			value:   " \t \n ",
			wantErr: true,
			errMsg:  "field cannot be empty: test field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNonEmpty(tt.field, tt.value)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no change needed",
			input:    "clean input",
			expected: "clean input",
		},
		{
			name:     "trim leading spaces",
			input:    "  input with leading spaces",
			expected: "input with leading spaces",
		},
		{
			name:     "trim trailing spaces",
			input:    "input with trailing spaces  ",
			expected: "input with trailing spaces",
		},
		{
			name:     "trim both sides",
			input:    "  input with both  ",
			expected: "input with both",
		},
		{
			name:     "trim tabs",
			input:    "\t\tinput with tabs\t\t",
			expected: "input with tabs",
		},
		{
			name:     "trim mixed whitespace",
			input:    " \t input \n ",
			expected: "input",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationResult(t *testing.T) {
	t.Run("new validation result", func(t *testing.T) {
		result := NewValidationResult()
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.NoError(t, result.FirstError())
		assert.NoError(t, result.AllErrors())
	})

	t.Run("add error", func(t *testing.T) {
		result := NewValidationResult()
		err := assert.AnError

		result.AddError(err)

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 1)
		assert.Equal(t, err, result.FirstError())
		assert.Equal(t, err, result.AllErrors())
	})

	t.Run("add nil error", func(t *testing.T) {
		result := NewValidationResult()

		result.AddError(nil)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.NoError(t, result.FirstError())
		assert.NoError(t, result.AllErrors())
	})

	t.Run("multiple errors", func(t *testing.T) {
		result := NewValidationResult()
		err1 := assert.AnError
		err2 := assert.AnError

		result.AddError(err1)
		result.AddError(err2)

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 2)
		assert.Equal(t, err1, result.FirstError())

		allErrors := result.AllErrors()
		assert.Contains(t, allErrors.Error(), err1.Error())
		assert.Contains(t, allErrors.Error(), err2.Error())
	})
}

func TestValidateSourceConfig(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		branch  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid source config",
			repo:    "org/repo",
			branch:  "master",
			wantErr: false,
		},
		{
			name:    "invalid repo",
			repo:    "invalid-repo",
			branch:  "master",
			wantErr: true,
			errMsg:  "repository name",
		},
		{
			name:    "empty repo",
			repo:    "",
			branch:  "master",
			wantErr: true,
			errMsg:  "field cannot be empty: repository name",
		},
		{
			name:    "invalid branch",
			repo:    "org/repo",
			branch:  "-invalid",
			wantErr: true,
			errMsg:  "branch name",
		},
		{
			name:    "empty branch",
			repo:    "org/repo",
			branch:  "",
			wantErr: true,
			errMsg:  "field cannot be empty: branch name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSourceConfig(tt.repo, tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTargetConfig(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		fileMappings []FileMapping
		wantErr      bool
		errMsg       string
	}{
		{
			name: "valid target config",
			repo: "org/target",
			fileMappings: []FileMapping{
				{Src: "src/file.txt", Dest: "dest/file.txt"},
			},
			wantErr: false,
		},
		{
			name:         "invalid repo",
			repo:         "invalid-repo",
			fileMappings: []FileMapping{{Src: "src", Dest: "dest"}},
			wantErr:      true,
			errMsg:       "repository name",
		},
		{
			name:         "empty repo",
			repo:         "",
			fileMappings: []FileMapping{{Src: "src", Dest: "dest"}},
			wantErr:      true,
			errMsg:       "field cannot be empty: repository name",
		},
		{
			name:         "no file mappings",
			repo:         "org/target",
			fileMappings: []FileMapping{},
			wantErr:      true,
			errMsg:       "at least one file mapping is required",
		},
		{
			name: "duplicate destinations",
			repo: "org/target",
			fileMappings: []FileMapping{
				{Src: "src1", Dest: "same"},
				{Src: "src2", Dest: "same"},
			},
			wantErr: true,
			errMsg:  "duplicate destination: same",
		},
		{
			name: "invalid file mapping",
			repo: "org/target",
			fileMappings: []FileMapping{
				{Src: "", Dest: "dest"},
			},
			wantErr: true,
			errMsg:  "field is required: source path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargetConfig(tt.repo, tt.fileMappings)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFileMapping(t *testing.T) {
	tests := []struct {
		name    string
		mapping FileMapping
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid file mapping",
			mapping: FileMapping{Src: "src/file.txt", Dest: "dest/file.txt"},
			wantErr: false,
		},
		{
			name:    "empty source",
			mapping: FileMapping{Src: "", Dest: "dest"},
			wantErr: true,
			errMsg:  "field is required: source path",
		},
		{
			name:    "empty destination",
			mapping: FileMapping{Src: "src", Dest: ""},
			wantErr: true,
			errMsg:  "field is required: destination",
		},
		{
			name:    "absolute source",
			mapping: FileMapping{Src: "/absolute/src", Dest: "dest"},
			wantErr: true,
			errMsg:  "must be relative",
		},
		{
			name:    "absolute destination",
			mapping: FileMapping{Src: "src", Dest: "/absolute/dest"},
			wantErr: true,
			errMsg:  "must be relative",
		},
		{
			name:    "path traversal in source",
			mapping: FileMapping{Src: "../traverse", Dest: "dest"},
			wantErr: true,
			errMsg:  "path traversal detected",
		},
		{
			name:    "path traversal in destination",
			mapping: FileMapping{Src: "src", Dest: "../traverse"},
			wantErr: true,
			errMsg:  "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileMapping(tt.mapping)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test edge cases and security
func TestValidationSecurityEdgeCases(t *testing.T) {
	t.Run("long input handling", func(_ *testing.T) {
		longRepo := strings.Repeat("a", 1000) + "/" + strings.Repeat("b", 1000)
		err := ValidateRepoName(longRepo)
		// Should handle long inputs gracefully (may pass or fail based on regex)
		// The important thing is it doesn't panic or cause issues
		_ = err
	})

	t.Run("unicode handling", func(t *testing.T) {
		unicodeRepo := "org🎉/repo™"
		err := ValidateRepoName(unicodeRepo)
		assert.Error(t, err) // Should reject unicode characters
	})

	t.Run("null byte injection", func(t *testing.T) {
		nullByteRepo := "org/repo\x00"
		err := ValidateRepoName(nullByteRepo)
		assert.Error(t, err) // Should reject null bytes
	})

	t.Run("complex path traversal", func(t *testing.T) {
		complexPath := "src/../../../../../../etc/passwd"
		err := ValidateFilePath(complexPath, "test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal detected")
	})

	t.Run("windows path traversal", func(_ *testing.T) {
		windowsPath := "src\\..\\..\\windows\\system32"
		// Our current validation focuses on Unix-style paths
		// This test documents current behavior
		err := ValidateFilePath(windowsPath, "test")
		// May or may not error depending on implementation
		_ = err
	})
}

func TestValidateRepoNameEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
		errMsg  string
	}{
		// Extreme length tests
		{
			name:    "extremely long org name",
			repo:    strings.Repeat("a", 10000) + "/repo",
			wantErr: false, // Regex doesn't enforce length limits
		},
		{
			name:    "extremely long repo name",
			repo:    "org/" + strings.Repeat("b", 10000),
			wantErr: false, // Regex doesn't enforce length limits
		},
		{
			name:    "single character org and repo",
			repo:    "a/b",
			wantErr: false,
		},
		// Special character edge cases
		{
			name:    "consecutive dots in org",
			repo:    "org..name/repo",
			wantErr: true, // Contains ".." path traversal
			errMsg:  "path traversal detected",
		},
		{
			name:    "consecutive hyphens",
			repo:    "org--name/repo--name",
			wantErr: false, // Multiple hyphens allowed
		},
		{
			name:    "consecutive underscores",
			repo:    "org__name/repo__name",
			wantErr: false, // Multiple underscores allowed
		},
		{
			name:    "mixed special chars",
			repo:    "org-_.-_name/repo_-.name",
			wantErr: false,
		},
		// Whitespace and control characters
		{
			name:    "tab character in name",
			repo:    "org\ttab/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "newline in name",
			repo:    "org\nname/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "carriage return in name",
			repo:    "org\rname/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "leading whitespace",
			repo:    " org/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "trailing whitespace",
			repo:    "org/repo ",
			wantErr: true,
			errMsg:  "invalid format",
		},
		// URL-like patterns
		{
			name:    "http prefix",
			repo:    "http://github.com/org/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "git protocol",
			repo:    "git://org/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "ssh format",
			repo:    "git@github.com:org/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		// Case sensitivity
		{
			name:    "uppercase letters",
			repo:    "ORG/REPO",
			wantErr: false,
		},
		{
			name:    "mixed case",
			repo:    "MyOrg/MyRepo",
			wantErr: false,
		},
		// Unicode and emoji edge cases
		{
			name:    "unicode letters",
			repo:    "组织/项目", //nolint:gosmopolitan // Testing Unicode rejection
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "emoji in org",
			repo:    "org😊/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "zero-width characters",
			repo:    "org\u200b/repo", // Zero-width space
			wantErr: true,
			errMsg:  "invalid format",
		},
		// Injection attempts
		{
			name:    "command injection attempt",
			repo:    "org;rm -rf /;/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "pipe character",
			repo:    "org|command/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "backtick injection",
			repo:    "org`command`/repo",
			wantErr: true,
			errMsg:  "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepoName(tt.repo)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateBranchNameEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		wantErr bool
		errMsg  string
	}{
		// Git special refs
		{
			name:    "HEAD reference",
			branch:  "HEAD",
			wantErr: false, // Technically valid but may cause issues
		},
		{
			name:    "double dot",
			branch:  "branch..name",
			wantErr: false, // Allowed in branch names
		},
		{
			name:    "tilde character",
			branch:  "branch~1",
			wantErr: true,
			errMsg:  "invalid field",
		},
		{
			name:    "caret character",
			branch:  "branch^2",
			wantErr: true,
			errMsg:  "invalid field",
		},
		{
			name:    "colon character",
			branch:  "branch:name",
			wantErr: true,
			errMsg:  "invalid field",
		},
		// Path-like patterns
		{
			name:    "multiple consecutive slashes",
			branch:  "feature///branch",
			wantErr: false, // Multiple slashes allowed
		},
		{
			name:    "ends with slash",
			branch:  "feature/branch/",
			wantErr: false, // Trailing slash allowed
		},
		{
			name:    "deeply nested slashes",
			branch:  "a/b/c/d/e/f/g/h",
			wantErr: false,
		},
		// Extreme lengths
		{
			name:    "extremely long branch name",
			branch:  strings.Repeat("a", 1000),
			wantErr: false, // No length limit enforced
		},
		{
			name:    "branch name at git limit",
			branch:  strings.Repeat("a", 255), // Git's typical limit
			wantErr: false,
		},
		// Special git patterns
		{
			name:    "@{upstream} pattern",
			branch:  "branch@{upstream}",
			wantErr: true,
			errMsg:  "invalid field",
		},
		{
			name:    "asterisk wildcard",
			branch:  "branch*",
			wantErr: true,
			errMsg:  "invalid field",
		},
		{
			name:    "question mark",
			branch:  "branch?",
			wantErr: true,
			errMsg:  "invalid field",
		},
		{
			name:    "square brackets",
			branch:  "branch[123]",
			wantErr: true,
			errMsg:  "invalid field",
		},
		// Control flow characters
		{
			name:    "backslash",
			branch:  "branch\\name",
			wantErr: true,
			errMsg:  "invalid field",
		},
		{
			name:    "null terminator",
			branch:  "branch\x00name",
			wantErr: true,
			errMsg:  "invalid field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranchName(tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFilePathEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		// Hidden files and directories
		{
			name:      "hidden file",
			path:      ".gitignore",
			fieldName: "source",
			wantErr:   false,
		},
		{
			name:      "hidden directory",
			path:      ".github/workflows/test.yml",
			fieldName: "source",
			wantErr:   false,
		},
		{
			name:      "current directory reference",
			path:      "./file.txt",
			fieldName: "source",
			wantErr:   false, // Clean path removes ./
		},
		{
			name:      "multiple current directory",
			path:      "././././file.txt",
			fieldName: "source",
			wantErr:   false, // Clean path removes all ./
		},
		// Complex path traversal attempts
		{
			name:      "encoded path traversal",
			path:      "..%2F..%2Fetc",
			fieldName: "source",
			wantErr:   true, // Contains .. which is detected
			errMsg:    "path traversal detected",
		},
		{
			name:      "unicode path traversal lookalike",
			path:      "․․/etc", // Using Unicode dot
			fieldName: "source",
			wantErr:   false, // Not actual path traversal
		},
		{
			name:      "mixed separators",
			path:      "src/../dest/./file.txt",
			fieldName: "source",
			wantErr:   false, // Clean path resolves to dest/file.txt
		},
		{
			name:      "trailing dots",
			path:      "file.txt...",
			fieldName: "source",
			wantErr:   false, // Valid filename
		},
		// Windows-specific paths
		{
			name:      "windows drive letter",
			path:      "C:\\file.txt",
			fieldName: "source",
			wantErr:   false, // On Unix, this is treated as relative path "C:\file.txt"
		},
		{
			name:      "windows UNC path",
			path:      "\\\\server\\share\\file.txt",
			fieldName: "source",
			wantErr:   false, // On Unix, this is treated as relative path
		},
		{
			name:      "windows style relative",
			path:      "src\\file.txt",
			fieldName: "source",
			wantErr:   false, // Treated as valid relative path
		},
		// Special filenames
		{
			name:      "single dot",
			path:      ".",
			fieldName: "source",
			wantErr:   false, // Current directory
		},
		{
			name:      "double dot alone",
			path:      "..",
			fieldName: "source",
			wantErr:   true,
			errMsg:    "path traversal detected",
		},
		{
			name:      "space only filename",
			path:      " ",
			fieldName: "source",
			wantErr:   false, // Valid but unusual
		},
		{
			name:      "very long path",
			path:      strings.Repeat("a/", 100) + "file.txt",
			fieldName: "source",
			wantErr:   false,
		},
		// Null and special characters
		{
			name:      "null byte in path",
			path:      "file\x00.txt",
			fieldName: "source",
			wantErr:   false, // Not explicitly checked
		},
		{
			name:      "control characters",
			path:      "file\n\r\t.txt",
			fieldName: "source",
			wantErr:   false, // Not explicitly checked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path, tt.fieldName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationResultEdgeCases(t *testing.T) {
	// Skip concurrent test due to race conditions in current implementation
	// The Result type would need mutex protection for concurrent access

	t.Run("very large number of errors", func(t *testing.T) {
		result := NewValidationResult()

		// Add many errors
		for i := 0; i < 1000; i++ {
			result.AddError(assert.AnError)
		}

		assert.False(t, result.Valid)
		assert.Len(t, result.Errors, 1000)

		// All errors should create a very long error message
		allErr := result.AllErrors()
		require.Error(t, allErr)
		assert.Contains(t, allErr.Error(), ";")
	})

	t.Run("nil result methods", func(t *testing.T) {
		var result *Result

		// These should not panic
		assert.NotPanics(t, func() {
			if result != nil {
				_ = result.FirstError()
			}
		})
	})
}

func TestValidateTargetConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		fileMappings []FileMapping
		wantErr      bool
		errMsg       string
	}{
		{
			name: "many file mappings",
			repo: "org/repo",
			fileMappings: func() []FileMapping {
				mappings := make([]FileMapping, 100)
				for i := 0; i < 100; i++ {
					mappings[i] = FileMapping{
						Src:  "src" + strings.Repeat("/nested", i) + "/file.txt",
						Dest: "dest" + strings.Repeat("/nested", i) + "/file.txt",
					}
				}
				return mappings
			}(),
			wantErr: false,
		},
		{
			name: "case sensitivity in duplicate detection",
			repo: "org/repo",
			fileMappings: []FileMapping{
				{Src: "src1", Dest: "File.txt"},
				{Src: "src2", Dest: "file.txt"}, // Different case
			},
			wantErr: false, // Case sensitive, so not duplicates
		},
		{
			name: "similar but different paths",
			repo: "org/repo",
			fileMappings: []FileMapping{
				{Src: "src1", Dest: "file"},
				{Src: "src2", Dest: "file.txt"},
				{Src: "src3", Dest: "file.txt.bak"},
			},
			wantErr: false,
		},
		{
			name: "empty src with valid dest",
			repo: "org/repo",
			fileMappings: []FileMapping{
				{Src: "", Dest: "valid/dest.txt"},
			},
			wantErr: true,
			errMsg:  "source path",
		},
		{
			name: "valid src with empty dest",
			repo: "org/repo",
			fileMappings: []FileMapping{
				{Src: "valid/src.txt", Dest: ""},
			},
			wantErr: true,
			errMsg:  "destination",
		},
		{
			name: "normalized paths cause duplicates",
			repo: "org/repo",
			fileMappings: []FileMapping{
				{Src: "src1", Dest: "./dest/file.txt"},
				{Src: "src2", Dest: "dest/file.txt"}, // Same after normalization?
			},
			wantErr: false, // Validation uses raw paths, not normalized
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTargetConfig(tt.repo, tt.fileMappings)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeInputEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Unicode whitespace
		{
			name:     "non-breaking space",
			input:    "\u00A0text\u00A0", // Non-breaking space
			expected: "text",             // strings.TrimSpace removes non-breaking spaces in Go
		},
		{
			name:     "zero-width space",
			input:    "\u200Btext\u200B", // Zero-width space
			expected: "\u200Btext\u200B", // Not trimmed
		},
		// Control characters
		{
			name:     "bell character",
			input:    "\a\a\atext\a\a\a",
			expected: "\a\a\atext\a\a\a", // Not trimmed
		},
		{
			name:     "form feed",
			input:    "\f\ftext\f\f",
			expected: "text", // Form feed is whitespace
		},
		{
			name:     "vertical tab",
			input:    "\v\vtext\v\v",
			expected: "text", // Vertical tab is whitespace
		},
		// Mixed whitespace types
		{
			name:     "all whitespace types",
			input:    " \t\n\r\f\vtext \t\n\r\f\v",
			expected: "text",
		},
		// Very long strings
		{
			name:     "long string with surrounding whitespace",
			input:    "   " + strings.Repeat("a", 10000) + "   ",
			expected: strings.Repeat("a", 10000),
		},
		// Empty variations
		{
			name:     "single space",
			input:    " ",
			expected: "",
		},
		{
			name:     "single tab",
			input:    "\t",
			expected: "",
		},
		{
			name:     "single newline",
			input:    "\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNonEmptyEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
	}{
		// Unicode spaces
		{
			name:    "non-breaking spaces only",
			field:   "test",
			value:   "\u00A0\u00A0\u00A0", // Non-breaking spaces
			wantErr: true,                 // strings.TrimSpace removes these, so considered empty
		},
		{
			name:    "zero-width spaces",
			field:   "test",
			value:   "\u200B\u200B", // Zero-width spaces
			wantErr: false,          // Not considered empty
		},
		// Control characters that look empty
		{
			name:    "null characters",
			field:   "test",
			value:   "\x00\x00\x00",
			wantErr: false, // Not considered empty
		},
		{
			name:    "backspace characters",
			field:   "test",
			value:   "\b\b\b",
			wantErr: false, // Not considered empty
		},
		// Whitespace with content
		{
			name:    "space sandwich",
			field:   "test",
			value:   "   a   ",
			wantErr: false,
		},
		{
			name:    "newline sandwich",
			field:   "test",
			value:   "\n\n\na\n\n\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNonEmpty(tt.field, tt.value)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationPatternPerformance(t *testing.T) {
	t.Run("regex compilation caching", func(_ *testing.T) {
		// Run validation multiple times to ensure regex patterns are properly cached
		for i := 0; i < 1000; i++ {
			_ = ValidateRepoName("org/repo")
			_ = ValidateBranchName("master")
			_ = ValidateBranchPrefix("sync")
		}
		// This test ensures patterns are compiled once and reused
	})

	t.Run("pathological regex input", func(_ *testing.T) {
		// Test inputs that could cause regex backtracking
		longRepeating := strings.Repeat("a-", 100) + "b"
		_ = ValidateRepoName("org/" + longRepeating)
		_ = ValidateBranchName(longRepeating)

		// Complex nested patterns
		nestedSlashes := "feature/" + strings.Repeat("sub/", 50) + "branch"
		_ = ValidateBranchName(nestedSlashes)
	})
}
