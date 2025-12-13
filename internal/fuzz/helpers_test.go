package fuzz

import (
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestContainsShellMetachars tests shell metacharacter detection
func TestContainsShellMetachars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Safe string",
			input:    "hello-world_123",
			expected: false,
		},
		{
			name:     "Semicolon injection",
			input:    "test; rm -rf /",
			expected: true,
		},
		{
			name:     "Ampersand injection",
			input:    "test & whoami",
			expected: true,
		},
		{
			name:     "Pipe injection",
			input:    "test | cat /etc/passwd",
			expected: true,
		},
		{
			name:     "Backtick injection",
			input:    "test`whoami`",
			expected: true,
		},
		{
			name:     "Dollar sign injection",
			input:    "test$(id)",
			expected: true,
		},
		{
			name:     "Parentheses injection",
			input:    "test(whoami)",
			expected: true,
		},
		{
			name:     "Braces injection",
			input:    "test{whoami}",
			expected: true,
		},
		{
			name:     "Redirect injection",
			input:    "test > /tmp/evil",
			expected: true,
		},
		{
			name:     "Backslash injection",
			input:    "test\\whoami",
			expected: true,
		},
		{
			name:     "Single quote injection",
			input:    "test'whoami'",
			expected: true,
		},
		{
			name:     "Double quote injection",
			input:    "test\"whoami\"",
			expected: true,
		},
		{
			name:     "Newline injection",
			input:    "test\nwhoami",
			expected: true,
		},
		{
			name:     "Carriage return injection",
			input:    "test\rwhoami",
			expected: true,
		},
		{
			name:     "Tab injection",
			input:    "test\twhoami",
			expected: true,
		},
		{
			name:     "Null byte injection",
			input:    "test\x00whoami",
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsShellMetachars(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %v for input: %q", tt.expected, tt.input)
		})
	}
}

// TestContainsPathTraversal tests path traversal detection
func TestContainsPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Safe relative path",
			input:    "docs/readme.txt",
			expected: false,
		},
		{
			name:     "Safe filename",
			input:    "config.yaml",
			expected: false,
		},
		{
			name:     "Dot-dot traversal",
			input:    "../../../etc/passwd",
			expected: true,
		},
		{
			name:     "Windows dot-dot traversal",
			input:    "..\\..\\windows\\system32",
			expected: true,
		},
		{
			name:     "Unix dot-dot in path",
			input:    "path/.../file",
			expected: true,
		},
		{
			name:     "Windows dot-dot in path",
			input:    "path\\..\\file",
			expected: true,
		},
		{
			name:     "Etc directory access",
			input:    "/etc/passwd",
			expected: true,
		},
		{
			name:     "Windows system access",
			input:    "\\windows\\system32\\cmd.exe",
			expected: true,
		},
		{
			name:     "Dev directory access",
			input:    "/dev/null",
			expected: true,
		},
		{
			name:     "Proc directory access",
			input:    "/proc/version",
			expected: true,
		},
		{
			name:     "Sys directory access",
			input:    "/sys/class/net",
			expected: true,
		},
		{
			name:     "System32 directory access",
			input:    "\\system32\\drivers",
			expected: true,
		},
		{
			name:     "Home directory tilde",
			input:    "~/secret.txt",
			expected: true,
		},
		{
			name:     "Home environment variable",
			input:    "$HOME/.ssh/id_rsa",
			expected: true,
		},
		{
			name:     "Windows home environment",
			input:    "%HOME%\\Documents\\secret.doc",
			expected: true,
		},
		{
			name:     "Variable expansion",
			input:    "${PWD}/secret",
			expected: true,
		},
		{
			name:     "Windows variable expansion",
			input:    "%{USERPROFILE}\\secret",
			expected: true,
		},
		{
			name:     "Absolute Unix path",
			input:    "/usr/bin/ls",
			expected: true,
		},
		{
			name:     "Absolute Windows path",
			input:    "\\Program Files\\app.exe",
			expected: true,
		},
		{
			name:     "Windows drive letter",
			input:    "C:\\temp\\file.txt",
			expected: true,
		},
		{
			name:     "Case insensitive etc",
			input:    "PATH/ETC/passwd",
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsPathTraversal(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %v for input: %q", tt.expected, tt.input)
		})
	}
}

// TestIsValidUTF8 tests UTF-8 validation
func TestIsValidUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid ASCII",
			input:    "Hello World",
			expected: true,
		},
		{
			name:     "Valid UTF-8 emoji",
			input:    "Hello 👋 World",
			expected: true,
		},
		{
			name:     "Valid UTF-8 accents",
			input:    "Héllo Wörld",
			expected: true,
		},
		{
			name:     "Valid UTF-8 Chinese",
			input:    "你好世界", //nolint:gosmopolitan // Testing Unicode handling
			expected: true,
		},
		{
			name:     "Valid with newline",
			input:    "Line 1\nLine 2",
			expected: true,
		},
		{
			name:     "Valid with carriage return",
			input:    "Line 1\rLine 2",
			expected: true,
		},
		{
			name:     "Valid with tab",
			input:    "Column 1\tColumn 2",
			expected: true,
		},
		{
			name:     "Invalid with null byte",
			input:    "Hello\x00World",
			expected: false,
		},
		{
			name:     "Invalid with bell character",
			input:    "Hello\x07World",
			expected: false,
		},
		{
			name:     "Invalid with escape character",
			input:    "Hello\x1bWorld",
			expected: false,
		},
		{
			name:     "Invalid with backspace",
			input:    "Hello\x08World",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidUTF8(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %v for input: %q", tt.expected, tt.input)
		})
	}
}

// TestContainsURLMetachars tests URL metacharacter detection
func TestContainsURLMetachars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Safe HTTP URL",
			input:    "https://example.com/path",
			expected: false,
		},
		{
			name:     "Safe HTTPS URL",
			input:    "https://api.github.com/repos/user/repo",
			expected: false,
		},
		{
			name:     "JavaScript injection",
			input:    "javascript:alert('xss')",
			expected: true,
		},
		{
			name:     "Data URL injection",
			input:    "data:text/html,<script>alert('xss')</script>",
			expected: true,
		},
		{
			name:     "VBScript injection",
			input:    "vbscript:msgbox('xss')",
			expected: true,
		},
		{
			name:     "File protocol",
			input:    "file:///etc/passwd",
			expected: true,
		},
		{
			name:     "Dict protocol",
			input:    "dict://attacker.com:11111/",
			expected: true,
		},
		{
			name:     "Gopher protocol",
			input:    "gopher://evil.com/",
			expected: true,
		},
		{
			name:     "Path traversal in URL",
			input:    "https://example.com/../../../etc/passwd",
			expected: true,
		},
		{
			name:     "Windows path traversal in URL",
			input:    "https://example.com/..\\..\\windows\\system32",
			expected: true,
		},
		{
			name:     "Null byte URL encoding",
			input:    "https://example.com/path%00.txt",
			expected: true,
		},
		{
			name:     "Newline URL encoding",
			input:    "https://example.com/path%0a",
			expected: true,
		},
		{
			name:     "Carriage return URL encoding",
			input:    "https://example.com/path%0d",
			expected: true,
		},
		{
			name:     "Raw carriage return",
			input:    "https://example.com/path\r",
			expected: true,
		},
		{
			name:     "Raw newline",
			input:    "https://example.com/path\n",
			expected: true,
		},
		{
			name:     "Tab character",
			input:    "https://example.com/path\t",
			expected: true,
		},
		{
			name:     "Case insensitive javascript",
			input:    "JAVASCRIPT:alert(1)",
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsURLMetachars(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %v for input: %q", tt.expected, tt.input)
		})
	}
}

// TestIsSafeBranchName tests git branch name safety
func TestIsSafeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Safe branch name",
			input:    "feature/user-auth",
			expected: true,
		},
		{
			name:     "Safe branch with numbers",
			input:    "feature/issue-123",
			expected: true,
		},
		{
			name:     "Safe branch with dots",
			input:    "release/v1.2.3",
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Branch with shell metachar semicolon",
			input:    "branch;rm -rf /",
			expected: false, // Contains shell metacharacter, unsafe
		},
		{
			name:     "Branch with dot-dot",
			input:    "feature/../master",
			expected: false,
		},
		{
			name:     "Branch with tilde",
			input:    "branch~1",
			expected: false,
		},
		{
			name:     "Branch with caret",
			input:    "branch^1",
			expected: false,
		},
		{
			name:     "Branch with colon",
			input:    "origin:master",
			expected: false,
		},
		{
			name:     "Branch with backslash",
			input:    "feature\\branch",
			expected: false, // Contains shell metacharacter (backslash), unsafe
		},
		{
			name:     "Branch with at-brace",
			input:    "branch@{1}",
			expected: false, // Contains shell metacharacter (braces), unsafe
		},
		{
			name:     "Branch with lock suffix",
			input:    "branch.lock",
			expected: false,
		},
		{
			name:     "Branch with space",
			input:    "branch name",
			expected: false,
		},
		{
			name:     "Branch with tab",
			input:    "branch\tname",
			expected: false, // Contains shell metacharacter (tab), unsafe
		},
		{
			name:     "Branch starting with dash",
			input:    "-delete-everything",
			expected: false,
		},
		{
			name:     "Valid branch with underscore",
			input:    "feature_branch",
			expected: true,
		},
		{
			name:     "Valid branch with hyphen",
			input:    "feature-branch",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeBranchName(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %v for input: %q", tt.expected, tt.input)
		})
	}
}

// TestIsSafeRepoName tests repository name safety
func TestIsSafeRepoName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid repo format",
			input:    "user/repository",
			expected: true,
		},
		{
			name:     "Valid org repo",
			input:    "organization/project-name",
			expected: true,
		},
		{
			name:     "Valid with numbers",
			input:    "user123/repo456",
			expected: true,
		},
		{
			name:     "Valid with underscores",
			input:    "user_name/repo_name",
			expected: true,
		},
		{
			name:     "Invalid format no slash",
			input:    "repository",
			expected: false,
		},
		{
			name:     "Invalid format too many slashes",
			input:    "user/group/repository",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Empty owner",
			input:    "/repository",
			expected: false,
		},
		{
			name:     "Empty repo",
			input:    "user/",
			expected: false,
		},
		{
			name:     "Shell metachar injection",
			input:    "user/repo;rm -rf /",
			expected: false,
		},
		{
			name:     "Path traversal",
			input:    "user/../../../etc/passwd",
			expected: false,
		},
		{
			name:     "Git suffix",
			input:    "user/repo.git",
			expected: false,
		},
		{
			name:     "SSH suffix",
			input:    "user/repo.ssh",
			expected: false,
		},
		{
			name:     "Config suffix",
			input:    "user/repo.config",
			expected: false,
		},
		{
			name:     "Bash suffix",
			input:    "user/repo.bash",
			expected: false,
		},
		{
			name:     "Shell suffix",
			input:    "user/repo.sh",
			expected: false,
		},
		{
			name:     "Case insensitive git suffix",
			input:    "user/REPO.GIT",
			expected: false,
		},
		{
			name:     "Valid with dots in name",
			input:    "user/my.project",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeRepoName(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %v for input: %q", tt.expected, tt.input)
		})
	}
}

// TestHasExcessiveLength tests length validation
func TestHasExcessiveLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected bool
	}{
		{
			name:     "Short string within limit",
			input:    "hello",
			maxLen:   10,
			expected: false,
		},
		{
			name:     "String at exact limit",
			input:    "hello",
			maxLen:   5,
			expected: false,
		},
		{
			name:     "String exceeding limit",
			input:    "hello world",
			maxLen:   5,
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			maxLen:   0,
			expected: false,
		},
		{
			name:     "Empty string with positive limit",
			input:    "",
			maxLen:   10,
			expected: false,
		},
		{
			name:     "Long string with high limit",
			input:    strings.Repeat("a", 1000),
			maxLen:   1001,
			expected: false,
		},
		{
			name:     "Long string exceeding high limit",
			input:    strings.Repeat("a", 1000),
			maxLen:   999,
			expected: true,
		},
		{
			name:     "UTF-8 string within limit",
			input:    "hello 世界", //nolint:gosmopolitan // Testing Unicode handling
			maxLen:   20,
			expected: false,
		},
		{
			name:     "UTF-8 string exceeding limit",
			input:    "hello 世界", //nolint:gosmopolitan // Testing Unicode handling
			maxLen:   5,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasExcessiveLength(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result, "Expected %v for input length %d vs max %d", tt.expected, len(tt.input), tt.maxLen)
		})
	}
}

// TestContainsNullByte tests null byte detection
func TestContainsNullByte(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Safe string",
			input:    "hello world",
			expected: false,
		},
		{
			name:     "String with null byte at end",
			input:    "hello\x00",
			expected: true,
		},
		{
			name:     "String with null byte at start",
			input:    "\x00hello",
			expected: true,
		},
		{
			name:     "String with null byte in middle",
			input:    "hel\x00lo",
			expected: true,
		},
		{
			name:     "String with multiple null bytes",
			input:    "\x00hello\x00world\x00",
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "String with other control chars",
			input:    "hello\t\n\r",
			expected: false,
		},
		{
			name:     "Binary data with null",
			input:    string([]byte{0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00, 0x57, 0x6f, 0x72, 0x6c, 0x64}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsNullByte(tt.input)
			assert.Equal(t, tt.expected, result, "Expected %v for input: %q", tt.expected, tt.input)
		})
	}
}

// TestIsSafeBranchNameShellMetachars verifies shell metacharacter rejection
func TestIsSafeBranchNameShellMetachars(t *testing.T) {
	// Branches with shell metacharacters should be rejected as unsafe
	testCases := []struct {
		name  string
		input string
	}{
		{"semicolon", "branch;rm -rf /"},
		{"ampersand", "branch && echo pwned"},
		{"pipe", "branch | cat /etc/passwd"},
		{"backtick", "branch`whoami`"},
		{"dollar", "branch$(id)"},
		{"newline", "branch\nmalicious"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsSafeBranchName(tc.input)
			assert.False(t, result, "Branch with %s should be unsafe", tc.name)
		})
	}
}

// TestFuzzHelpersEdgeCases tests edge cases and boundary conditions
func TestFuzzHelpersEdgeCases(t *testing.T) {
	t.Run("MaxUnicodeCharacter", func(t *testing.T) {
		maxUnicode := string(rune(0x10FFFF))
		result := IsValidUTF8(maxUnicode)
		assert.True(t, result, "Max valid Unicode character should be valid")
	})

	t.Run("ReplacementCharacter", func(t *testing.T) {
		replacementChar := string(rune(0xFFFD))
		result := IsValidUTF8(replacementChar)
		assert.False(t, result, "Unicode replacement character should be invalid")
	})

	t.Run("ZeroLengthLimit", func(t *testing.T) {
		result := HasExcessiveLength("a", 0)
		assert.True(t, result, "Any non-empty string should exceed zero length limit")
	})

	t.Run("NegativeLengthLimit", func(t *testing.T) {
		result := HasExcessiveLength("hello", -1)
		assert.True(t, result, "Any string should exceed negative length limit")
	})

	t.Run("PathTraversalCaseInsensitive", func(t *testing.T) {
		result := ContainsPathTraversal("PATH/TO/ETC/PASSWD")
		assert.True(t, result, "Path traversal detection should be case insensitive")
	})

	t.Run("URLMetacharsCaseInsensitive", func(t *testing.T) {
		result := ContainsURLMetachars("JAVASCRIPT:ALERT(1)")
		assert.True(t, result, "URL metachar detection should be case insensitive")
	})
}

// TestFuzzHelpersConcurrency tests thread safety of functions
func TestFuzzHelpersConcurrency(t *testing.T) {
	t.Run("ConcurrentAccess", func(_ *testing.T) {
		// Test that the functions are safe for concurrent access
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(_ int) {
				defer wg.Done()

				testStr := "test-string"

				// Call all functions concurrently
				ContainsShellMetachars(testStr)
				ContainsPathTraversal(testStr)
				IsValidUTF8(testStr)
				ContainsURLMetachars(testStr)
				IsSafeBranchName(testStr)
				IsSafeRepoName("user/repo")
				_ = HasExcessiveLength(testStr, 100)
				ContainsNullByte(testStr)
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
	})
}

// TestInputLengthValidation verifies that excessively long inputs are treated as suspicious
func TestInputLengthValidation(t *testing.T) {
	// Create a string longer than maxInputLength (10000)
	longInput := strings.Repeat("a", 10001)
	normalInput := strings.Repeat("a", 100)

	t.Run("ContainsShellMetachars_LongInput", func(t *testing.T) {
		result := ContainsShellMetachars(longInput)
		assert.True(t, result, "Excessively long input should be treated as suspicious")
	})

	t.Run("ContainsShellMetachars_NormalInput", func(t *testing.T) {
		result := ContainsShellMetachars(normalInput)
		assert.False(t, result, "Normal input without metacharacters should be safe")
	})

	t.Run("ContainsPathTraversal_LongInput", func(t *testing.T) {
		result := ContainsPathTraversal(longInput)
		assert.True(t, result, "Excessively long path should be treated as suspicious")
	})

	t.Run("ContainsPathTraversal_NormalInput", func(t *testing.T) {
		result := ContainsPathTraversal(normalInput)
		assert.False(t, result, "Normal path without traversal should be safe")
	})

	t.Run("ContainsURLMetachars_LongInput", func(t *testing.T) {
		result := ContainsURLMetachars(longInput)
		assert.True(t, result, "Excessively long URL should be treated as suspicious")
	})

	t.Run("ContainsURLMetachars_NormalInput", func(t *testing.T) {
		result := ContainsURLMetachars("https://" + normalInput + ".com")
		assert.False(t, result, "Normal URL should be safe")
	})

	t.Run("IsSafeBranchName_LongInput", func(t *testing.T) {
		result := IsSafeBranchName(longInput)
		assert.False(t, result, "Excessively long branch name should be unsafe")
	})

	t.Run("IsSafeBranchName_NormalInput", func(t *testing.T) {
		result := IsSafeBranchName(normalInput)
		assert.True(t, result, "Normal branch name should be safe")
	})

	t.Run("IsSafeRepoName_LongInput", func(t *testing.T) {
		result := IsSafeRepoName(longInput + "/" + normalInput)
		assert.False(t, result, "Excessively long repo name should be unsafe")
	})

	t.Run("IsSafeRepoName_NormalInput", func(t *testing.T) {
		result := IsSafeRepoName("org/repo")
		assert.True(t, result, "Normal repo name should be safe")
	})
}
