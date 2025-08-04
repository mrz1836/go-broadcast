package builtin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWhitespaceCheckWithConfig(t *testing.T) {
	tests := []struct {
		name              string
		config            *config.Config
		expectedTimeout   time.Duration
		expectedAutoStage bool
	}{
		{
			name:              "nil config",
			config:            nil,
			expectedTimeout:   30 * time.Second,
			expectedAutoStage: false,
		},
		{
			name: "config with auto-stage disabled",
			config: &config.Config{
				CheckTimeouts: struct {
					Fumpt      int
					Lint       int
					ModTidy    int
					Whitespace int
					EOF        int
				}{
					Whitespace: 60,
				},
				CheckBehaviors: struct {
					WhitespaceAutoStage bool
					EOFAutoStage        bool
				}{
					WhitespaceAutoStage: false,
				},
			},
			expectedTimeout:   60 * time.Second,
			expectedAutoStage: false,
		},
		{
			name: "config with auto-stage enabled",
			config: &config.Config{
				CheckTimeouts: struct {
					Fumpt      int
					Lint       int
					ModTidy    int
					Whitespace int
					EOF        int
				}{
					Whitespace: 90,
				},
				CheckBehaviors: struct {
					WhitespaceAutoStage bool
					EOFAutoStage        bool
				}{
					WhitespaceAutoStage: true,
				},
			},
			expectedTimeout:   90 * time.Second,
			expectedAutoStage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewWhitespaceCheckWithConfig(tt.config)

			assert.Equal(t, "whitespace", check.Name())
			assert.Equal(t, tt.expectedTimeout, check.timeout)
			assert.Equal(t, tt.config, check.config)
			assert.Equal(t, tt.expectedAutoStage, check.autoStage)
		})
	}
}

func TestWhitespaceCheck_Run_WithAutoStage(t *testing.T) {
	// Skip if not in a git repository or git is not available
	if !isGitAvailable() || !isInGitRepo() {
		t.Skip("Skipping auto-stage tests: git not available or not in git repository")
	}

	// Create temporary directory within the repository
	tmpDir, err := os.MkdirTemp(".", "whitespace_test")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			t.Logf("Failed to remove temp dir %s: %v", tmpDir, removeErr)
		}
	}()

	// Create test file with whitespace issues
	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main   \n\nfunc main() {  \n}"), 0o600)
	require.NoError(t, err)

	// Create config with auto-staging enabled
	cfg := &config.Config{
		Directory: filepath.Join(".", "pre-commit"), // Current directory structure
		CheckTimeouts: struct {
			Fumpt      int
			Lint       int
			ModTidy    int
			Whitespace int
			EOF        int
		}{
			Whitespace: 30,
		},
		CheckBehaviors: struct {
			WhitespaceAutoStage bool
			EOFAutoStage        bool
		}{
			WhitespaceAutoStage: true,
		},
	}

	check := NewWhitespaceCheckWithConfig(cfg)

	// Ensure file is not staged initially
	if resetErr := exec.CommandContext(context.Background(), "git", "reset", "HEAD", testFile).Run(); resetErr != nil { //nolint:gosec // test code with controlled input
		t.Logf("Failed to reset git HEAD for %s: %v", testFile, resetErr)
	}

	// Run the check
	err = check.Run(context.Background(), []string{testFile})

	// Should still return error indicating issues were found and fixed
	require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)

	// Check that the file was modified (whitespace removed)
	content, err := os.ReadFile(filepath.Clean(testFile))
	require.NoError(t, err)
	expected := "package main\n\nfunc main() {\n}"
	assert.Equal(t, expected, string(content))

	// Check that the file was staged
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	require.NoError(t, err)

	staged := strings.TrimSpace(string(output))
	assert.Contains(t, staged, filepath.Base(testFile))
}

func TestWhitespaceCheck_Run_AutoStageError(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "whitespace_test")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			t.Logf("Failed to remove temp dir %s: %v", tmpDir, removeErr)
		}
	}()

	// Create test file with whitespace issues
	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main   \n"), 0o600)
	require.NoError(t, err)

	// Create config with auto-staging enabled but invalid directory
	cfg := &config.Config{
		Directory: "/invalid/directory/pre-commit",
		CheckTimeouts: struct {
			Fumpt      int
			Lint       int
			ModTidy    int
			Whitespace int
			EOF        int
		}{
			Whitespace: 30,
		},
		CheckBehaviors: struct {
			WhitespaceAutoStage bool
			EOFAutoStage        bool
		}{
			WhitespaceAutoStage: true,
		},
	}

	check := NewWhitespaceCheckWithConfig(cfg)

	// Run the check
	err = check.Run(context.Background(), []string{testFile})

	// Should return error that includes auto-staging failure
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auto-staging failed")
}

func TestWhitespaceCheck_StageFiles(t *testing.T) {
	if !isGitAvailable() || !isInGitRepo() {
		t.Skip("Skipping git staging tests: git not available or not in git repository")
	}

	// Create temporary directory within the repository
	tmpDir, err := os.MkdirTemp(".", "stage_test")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			t.Logf("Failed to remove temp dir %s: %v", tmpDir, removeErr)
		}
	}()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o600)
	require.NoError(t, err)

	cfg := &config.Config{
		Directory: filepath.Join(".", "pre-commit"),
	}

	check := NewWhitespaceCheckWithConfig(cfg)

	// Ensure file is not staged initially
	if resetErr := exec.CommandContext(context.Background(), "git", "reset", "HEAD", testFile).Run(); resetErr != nil { //nolint:gosec // test code with controlled input
		t.Logf("Failed to reset git HEAD for %s: %v", testFile, resetErr)
	}

	// Stage the file
	err = check.stageFiles(context.Background(), []string{testFile})
	require.NoError(t, err)

	// Verify file was staged
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	require.NoError(t, err)

	staged := strings.TrimSpace(string(output))
	assert.Contains(t, staged, filepath.Base(testFile))
}

func TestWhitespaceCheck_AutoStageFields(t *testing.T) {
	// Test that existing constructors have autoStage set to false
	check1 := NewWhitespaceCheck()
	assert.False(t, check1.autoStage)
	assert.Nil(t, check1.config)

	check2 := NewWhitespaceCheckWithTimeout(60 * time.Second)
	assert.False(t, check2.autoStage)
	assert.Nil(t, check2.config)

	// Test that new constructor can enable auto-staging
	cfg := &config.Config{
		CheckBehaviors: struct {
			WhitespaceAutoStage bool
			EOFAutoStage        bool
		}{
			WhitespaceAutoStage: true,
		},
	}

	check3 := NewWhitespaceCheckWithConfig(cfg)
	assert.True(t, check3.autoStage)
	assert.Equal(t, cfg, check3.config)
}

// Helper functions

func isGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func isInGitRepo() bool {
	cmd := exec.CommandContext(context.Background(), "git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}
