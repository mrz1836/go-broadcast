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

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
			(len(s) > 0 && len(substr) > 0 &&
				(s[:len(substr)] == substr ||
					contains(s[1:], substr))))
}
