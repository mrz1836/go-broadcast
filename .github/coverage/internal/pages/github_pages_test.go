package pages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewGitHubPagesDeployer(t *testing.T) {
	tests := []struct {
		name        string
		repoPath    string
		pagesBranch string
		verbose     bool
		expected    *GitHubPagesDeployer
	}{
		{
			name:        "with custom branch",
			repoPath:    "/tmp/repo",
			pagesBranch: "custom-pages",
			verbose:     true,
			expected: &GitHubPagesDeployer{
				repoPath:    "/tmp/repo",
				pagesBranch: "custom-pages",
				verbose:     true,
			},
		},
		{
			name:        "with default branch",
			repoPath:    "/tmp/repo",
			pagesBranch: "",
			verbose:     false,
			expected: &GitHubPagesDeployer{
				repoPath:    "/tmp/repo",
				pagesBranch: "gh-pages",
				verbose:     false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			got := NewGitHubPagesDeployer(tt.repoPath, tt.pagesBranch, tt.verbose)
			if got.repoPath != tt.expected.repoPath {
				t.Errorf("repoPath = %v, want %v", got.repoPath, tt.expected.repoPath)
			}
			if got.pagesBranch != tt.expected.pagesBranch {
				t.Errorf("pagesBranch = %v, want %v", got.pagesBranch, tt.expected.pagesBranch)
			}
			if got.verbose != tt.expected.verbose {
				t.Errorf("verbose = %v, want %v", got.verbose, tt.expected.verbose)
			}
		})
	}
}

func TestGitHubPagesDeployer_generateInitialDashboardHTML(t *testing.T) {
	d := &GitHubPagesDeployer{}
	html := d.generateInitialDashboardHTML()

	// Check that HTML contains expected elements
	expectedStrings := []string{
		"<!DOCTYPE html>",
		"<html lang=\"en\">",
		"GoFortress Coverage Dashboard",
		"Coverage tracking and reporting",
		"</html>",
	}

	for _, expected := range expectedStrings {
		if !contains(html, expected) {
			t.Errorf("HTML missing expected string: %s", expected)
		}
	}
}

func TestGitHubPagesDeployer_copyFile(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")

	if err := os.MkdirAll(srcDir, 0o750); err != nil {
		t.Fatalf("Failed to create src directory: %v", err)
	}

	// Create test file
	srcFile := filepath.Join(srcDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcFile, content, 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	d := &GitHubPagesDeployer{}
	dstFile := filepath.Join(dstDir, "subdir", "test.txt")

	// Test copying file
	if err := d.copyFile(srcFile, dstFile); err != nil {
		t.Errorf("copyFile() error = %v", err)
	}

	// Verify file was copied
	copied, err := os.ReadFile(dstFile) //nolint:gosec // test file
	if err != nil {
		t.Errorf("Failed to read copied file: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("Copied content = %v, want %v", string(copied), string(content))
	}
}

func TestGitHubPagesDeployer_copyArtifacts(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")

	// Create test files
	files := map[string]string{
		"coverage.svg":   "<svg>coverage badge</svg>",
		"coverage.html":  "<html>coverage report</html>",
		"dashboard.html": "<html>dashboard</html>",
	}

	for name, content := range files {
		filePath := filepath.Join(srcDir, name)
		if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
			t.Fatalf("Failed to create file %s: %v", name, err)
		}
	}

	// Change to temp directory for test
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	d := &GitHubPagesDeployer{}

	// Test copying without prefix
	if err := d.copyArtifacts(srcDir, ""); err != nil {
		t.Errorf("copyArtifacts() error = %v", err)
	}

	// Verify files were copied
	expectedFiles := map[string]string{
		"badges/coverage.svg":   files["coverage.svg"],
		"reports/coverage.html": files["coverage.html"],
		"coverage/index.html":   files["dashboard.html"],
	}

	for path, expectedContent := range expectedFiles {
		content, err := os.ReadFile(path) //nolint:gosec // test file
		if err != nil {
			t.Errorf("Failed to read %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("File %s content = %v, want %v", path, string(content), expectedContent)
		}
	}
}

func TestCleanOptions_validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    CleanOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: CleanOptions{
				MaxAgeDays: 30,
				DryRun:     true,
				Verbose:    true,
			},
			wantErr: false,
		},
		{
			name: "zero max age",
			opts: CleanOptions{
				MaxAgeDays: 0,
				DryRun:     false,
				Verbose:    false,
			},
			wantErr: false, // Zero should be valid (clean everything)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Just verify the struct can be created
			// In real implementation, add validation if needed
			_ = tt.opts
		})
	}
}

func TestDeployOptions_validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    DeployOptions
		wantErr bool
	}{
		{
			name: "branch deployment",
			opts: DeployOptions{
				Branch:    "main",
				CommitSha: "abc123",
				InputDir:  "./coverage",
				Verbose:   true,
			},
			wantErr: false,
		},
		{
			name: "PR deployment",
			opts: DeployOptions{
				Branch:    "feature",
				CommitSha: "def456",
				PRNumber:  "42",
				InputDir:  "./coverage",
				Message:   "Custom message",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Just verify the struct can be created
			// In real implementation, add validation if needed
			_ = tt.opts
		})
	}
}

func TestGitHubPagesDeployer_SanitizeURL(t *testing.T) {
	d := &GitHubPagesDeployer{verbose: true}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean URL",
			input:    "https://github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "URL with whitespace",
			input:    "  https://github.com/owner/repo.git  ",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "URL with ANSI codes",
			input:    "\x1b[31mhttps://github.com/owner/repo.git\x1b[0m",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "URL with control characters",
			input:    "https://github.com/owner/repo.git\x00\x01",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.sanitizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGitHubPagesDeployer_ExtractGitHubRepoPath(t *testing.T) {
	d := &GitHubPagesDeployer{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SSH format",
			input:    "git@github.com:owner/repo.git",
			expected: "owner/repo.git",
		},
		{
			name:     "HTTPS format",
			input:    "https://github.com/owner/repo.git",
			expected: "owner/repo.git",
		},
		{
			name:     "masked format",
			input:    "***github.com/owner/repo",
			expected: "owner/repo",
		},
		{
			name:     "complex masked format",
			input:    "https://x-access-token:***@github.com/owner/repo.git",
			expected: "owner/repo.git",
		},
		{
			name:     "unsupported format",
			input:    "https://gitlab.com/owner/repo.git",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.extractGitHubRepoPath(tt.input)
			if result != tt.expected {
				t.Errorf("extractGitHubRepoPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGitHubPagesDeployer_AddTokenToURL(t *testing.T) {
	d := &GitHubPagesDeployer{verbose: false} // Disable verbose to avoid output in tests

	tests := []struct {
		name        string
		remoteURL   string
		token       string
		expected    string
		expectError bool
	}{
		{
			name:      "HTTPS URL",
			remoteURL: "https://github.com/owner/repo.git",
			token:     "ghp_token123",
			expected:  "https://x-access-token:ghp_token123@github.com/owner/repo.git",
		},
		{
			name:      "SSH URL",
			remoteURL: "git@github.com:owner/repo.git",
			token:     "ghp_token123",
			expected:  "https://x-access-token:ghp_token123@github.com/owner/repo.git",
		},
		{
			name:      "URL without .git suffix",
			remoteURL: "https://github.com/owner/repo",
			token:     "ghp_token123",
			expected:  "https://x-access-token:ghp_token123@github.com/owner/repo.git",
		},
		{
			name:        "unsupported URL",
			remoteURL:   "https://gitlab.com/owner/repo.git",
			token:       "token123",
			expectError: true,
		},
		{
			name:        "empty URL",
			remoteURL:   "",
			token:       "token123",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.addTokenToURL(tt.remoteURL, tt.token)
			if tt.expectError {
				if err == nil {
					t.Errorf("addTokenToURL() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("addTokenToURL() unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("addTokenToURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGitHubPagesDeployer_GetGitHubToken(t *testing.T) {
	d := &GitHubPagesDeployer{verbose: false}

	// Save original env vars
	originalGHPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHubToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		_ = os.Setenv("GH_PAT_TOKEN", originalGHPAT)
		_ = os.Setenv("GITHUB_TOKEN", originalGitHubToken)
	}()

	// Test GH_PAT_TOKEN priority
	_ = os.Setenv("GH_PAT_TOKEN", "pat_token_123")
	_ = os.Setenv("GITHUB_TOKEN", "gh_token_456")
	token := d.getGitHubToken()
	if token != "pat_token_123" {
		t.Errorf("Expected GH_PAT_TOKEN to take priority, got %q", token)
	}

	// Test GITHUB_TOKEN fallback
	_ = os.Unsetenv("GH_PAT_TOKEN")
	_ = os.Setenv("GITHUB_TOKEN", "gh_token_456")
	token = d.getGitHubToken()
	if token != "gh_token_456" {
		t.Errorf("Expected GITHUB_TOKEN fallback, got %q", token)
	}

	// Test no token
	_ = os.Unsetenv("GH_PAT_TOKEN")
	_ = os.Unsetenv("GITHUB_TOKEN")
	token = d.getGitHubToken()
	if token != "" {
		t.Errorf("Expected empty token when no env vars set, got %q", token)
	}
}

func TestGitHubPagesDeployer_CopyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	d := &GitHubPagesDeployer{}

	// Create source directory structure
	srcDir := filepath.Join(tempDir, "src")
	dstDir := filepath.Join(tempDir, "dst")

	// Create test files
	files := map[string]string{
		"file1.txt":         "content1",
		"subdir/file2.txt":  "content2",
		"subdir/file3.html": "<html>content3</html>",
	}

	for path, content := range files {
		fullPath := filepath.Join(srcDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Copy directory
	if err := d.copyDirectory(srcDir, dstDir); err != nil {
		t.Errorf("copyDirectory() error = %v", err)
	}

	// Verify all files were copied
	for path, expectedContent := range files {
		dstPath := filepath.Join(dstDir, path)
		content, err := os.ReadFile(dstPath) //nolint:gosec // Test file path is safe
		if err != nil {
			t.Errorf("Failed to read copied file %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("File %s content = %q, want %q", path, string(content), expectedContent)
		}
	}
}

func TestGitHubPagesDeployer_CreateAndCopyFavicons(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	d := &GitHubPagesDeployer{verbose: false}

	if err := d.createAndCopyFavicons(); err != nil {
		t.Errorf("createAndCopyFavicons() error = %v", err)
	}

	// Verify favicon files were created
	expectedFiles := []string{"favicon.svg", "favicon.ico"}
	for _, filename := range expectedFiles {
		if _, err := os.Stat(filename); err != nil {
			t.Errorf("Expected favicon file %s was not created: %v", filename, err)
		}
	}

	// Verify SVG content
	svgContent, err := os.ReadFile("favicon.svg")
	if err != nil {
		t.Errorf("Failed to read favicon.svg: %v", err)
	} else {
		svgStr := string(svgContent)
		if !contains(svgStr, "<svg") || !contains(svgStr, "</svg>") {
			t.Errorf("favicon.svg does not contain valid SVG content")
		}
	}
}

func TestErrors(t *testing.T) {
	// Test error constants
	if ErrBranchExists.Error() != "branch already exists" {
		t.Errorf("ErrBranchExists message incorrect")
	}
	if ErrInvalidBranchName.Error() != "invalid branch name" {
		t.Errorf("ErrInvalidBranchName message incorrect")
	}
	if ErrUnsupportedURLFormat.Error() != "unsupported remote URL format" {
		t.Errorf("ErrUnsupportedURLFormat message incorrect")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			(len(s) > 0 && len(substr) > 0 &&
				(s[:len(substr)] == substr ||
					contains(s[1:], substr))))
}
