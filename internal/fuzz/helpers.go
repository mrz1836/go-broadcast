// Package fuzz provides fuzzing utilities and security validation helpers
package fuzz

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ContainsShellMetachars checks for shell metacharacters that could lead to command injection
func ContainsShellMetachars(s string) bool {
	metachars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "<", ">", "\\", "'", "\"", "\n", "\r", "\t"}
	for _, char := range metachars {
		if strings.Contains(s, char) {
			return true
		}
	}
	// Check for null bytes
	if strings.Contains(s, "\x00") {
		return true
	}
	return false
}

// ContainsPathTraversal checks for path traversal attempts
func ContainsPathTraversal(path string) bool {
	dangerous := []string{
		"..", "../", "..\\",
		"/..", "\\..",
		"/etc/", "\\windows\\",
		"/dev/", "/proc/",
		"/sys/", "\\system32\\",
		"~", "$HOME", "%HOME%",
		"${", "%{",
	}
	pathLower := strings.ToLower(path)
	for _, pattern := range dangerous {
		if strings.Contains(pathLower, strings.ToLower(pattern)) {
			return true
		}
	}
	// Check for absolute paths
	if len(path) > 0 && (path[0] == '/' || path[0] == '\\') {
		return true
	}
	// Check for Windows drive letters
	if len(path) >= 2 && path[1] == ':' {
		return true
	}
	return false
}

// IsValidUTF8 validates UTF-8 encoding and checks for problematic characters
func IsValidUTF8(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		// Check for replacement character
		if r == unicode.ReplacementChar {
			return false
		}
		// Check for control characters (except common ones like \n, \r, \t)
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

// ContainsURLMetachars checks for characters that could lead to URL injection
func ContainsURLMetachars(url string) bool {
	// Check for common URL injection patterns
	dangerous := []string{
		"javascript:", "data:", "vbscript:",
		"file://", "dict://", "gopher://",
		"../", "..\\",
		"%00", "%0a", "%0d",
		"\r", "\n", "\t",
	}
	urlLower := strings.ToLower(url)
	for _, pattern := range dangerous {
		if strings.Contains(urlLower, pattern) {
			return true
		}
	}
	return false
}

// IsSafeBranchName checks if a branch name is safe for git operations
func IsSafeBranchName(branch string) bool {
	if branch == "" {
		return false
	}
	// Check for shell metacharacters
	if ContainsShellMetachars(branch) {
		return true // unsafe
	}
	// Check for git-specific dangerous patterns
	dangerous := []string{
		"..", "~", "^", ":", "\\",
		"@{", ".lock", " ", "\t",
	}
	for _, pattern := range dangerous {
		if strings.Contains(branch, pattern) {
			return false
		}
	}
	// Check if it starts with dash (could be interpreted as flag)
	if strings.HasPrefix(branch, "-") {
		return false
	}
	return true
}

// IsSafeRepoName checks if a repository name follows safe patterns
func IsSafeRepoName(repo string) bool {
	// Basic format check
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return false
	}
	// Both parts should be non-empty
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	// Check for dangerous patterns
	if ContainsShellMetachars(repo) {
		return false
	}
	if ContainsPathTraversal(repo) {
		return false
	}
	// Check for suspicious extensions that might indicate path injection
	suspicious := []string{".git", ".ssh", ".config", ".bash", ".sh"}
	repoLower := strings.ToLower(repo)
	for _, ext := range suspicious {
		if strings.HasSuffix(repoLower, ext) {
			return false
		}
	}
	return true
}

// HasExcessiveLength checks if input exceeds reasonable bounds
func HasExcessiveLength(s string, maxLen int) bool {
	return len(s) > maxLen
}

// ContainsNullByte checks for null byte injection
func ContainsNullByte(s string) bool {
	return strings.Contains(s, "\x00")
}
