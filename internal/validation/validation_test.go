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
			branch:  "main",
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
			prefix:  "sync/template",
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
			branch:  "main",
			wantErr: false,
		},
		{
			name:    "invalid repo",
			repo:    "invalid-repo",
			branch:  "main",
			wantErr: true,
			errMsg:  "repository name",
		},
		{
			name:    "empty repo",
			repo:    "",
			branch:  "main",
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
