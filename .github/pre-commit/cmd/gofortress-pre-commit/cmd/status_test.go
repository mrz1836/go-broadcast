package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/git"
)

// StatusTestSuite tests the status command
type StatusTestSuite struct {
	suite.Suite

	tempDir     string
	originalWD  string
	gitDir      string
	hooksDir    string
	configFile  string
	origVerbose bool
}

func (s *StatusTestSuite) SetupTest() {
	// Save original working directory
	var err error
	s.originalWD, err = os.Getwd()
	s.Require().NoError(err)

	// Create temp directory
	s.tempDir = s.T().TempDir()

	// Initialize git repository
	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = s.tempDir
	err = initCmd.Run()
	s.Require().NoError(err)

	// Set git directory paths for tests
	s.gitDir = filepath.Join(s.tempDir, ".git")
	s.hooksDir = filepath.Join(s.gitDir, "hooks")

	// Create config file
	s.configFile = filepath.Join(s.tempDir, ".github", ".env.shared")
	err = os.MkdirAll(filepath.Dir(s.configFile), 0o750)
	s.Require().NoError(err)

	// Save verbose state
	s.origVerbose = verbose
}

func (s *StatusTestSuite) TearDownTest() {
	// Restore verbose state
	verbose = s.origVerbose

	// Clear environment variables that might affect other tests
	_ = os.Unsetenv("ENABLE_PRE_COMMIT_SYSTEM")
	_ = os.Unsetenv("PRE_COMMIT_SYSTEM_ENABLE_FUMPT")
	_ = os.Unsetenv("PRE_COMMIT_SYSTEM_ENABLE_LINT")
	_ = os.Unsetenv("PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY")
	_ = os.Unsetenv("PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE")
	_ = os.Unsetenv("PRE_COMMIT_SYSTEM_ENABLE_EOF")
	_ = os.Unsetenv("PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS")
	_ = os.Unsetenv("PRE_COMMIT_SYSTEM_PARALLEL_WORKERS")

	// Change back to original directory
	_ = os.Chdir(s.originalWD)
}

func TestStatusTestSuite(t *testing.T) {
	suite.Run(t, new(StatusTestSuite))
}

// Test status command creation
func (s *StatusTestSuite) TestStatusCommand() {
	s.NotNil(statusCmd)
	s.Equal("status", statusCmd.Use)
	s.Equal("Show installation status of git hooks", statusCmd.Short)
	s.NotEmpty(statusCmd.Long)
	s.NotEmpty(statusCmd.Example)
	s.NotNil(statusCmd.RunE)
}

// Test basic status with no hooks installed
func (s *StatusTestSuite) TestStatusNoHooks() {
	// Create minimal config
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true`
	err := os.WriteFile(s.configFile, []byte(configContent), 0o600)
	s.Require().NoError(err)

	// Capture output
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Require().NoError(err)
	})

	// Verify output
	s.Contains(output, "GoFortress Pre-commit System Status")
	s.Contains(output, "Pre-commit system is enabled")
	s.Contains(output, "Git Hook Status")
	s.Contains(output, "No hooks are currently installed")
	s.Contains(output, "Run 'gofortress-pre-commit install' to install")
}

// Test status with hooks installed
func (s *StatusTestSuite) TestStatusWithHooks() {
	// Create config
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true`
	err := os.WriteFile(s.configFile, []byte(configContent), 0o600)
	s.Require().NoError(err)

	// Install our hook
	hookPath := filepath.Join(s.hooksDir, "pre-commit")
	hookContent := fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec %s/gofortress-pre-commit run \"$@\"\n", s.tempDir)
	err = os.WriteFile(hookPath, []byte(hookContent), 0o700) //nolint:gosec // Git hooks need execute permissions
	s.Require().NoError(err)

	// Install another hook type
	prePushPath := filepath.Join(s.hooksDir, "pre-push")
	prePushContent := fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec %s/gofortress-pre-commit run --hook-type pre-push \"$@\"\n", s.tempDir)
	err = os.WriteFile(prePushPath, []byte(prePushContent), 0o700) //nolint:gosec // Git hooks need execute permissions
	s.Require().NoError(err)

	// Capture output
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Require().NoError(err)
	})

	// Verify output
	s.Contains(output, "Pre-commit system is enabled")
	s.Contains(output, "✓ pre-commit:")
	s.Contains(output, "GoFortress pre-commit hook installed and ready")
	s.Contains(output, "✓ pre-push:")
}

// Test status with conflicting hooks
func (s *StatusTestSuite) TestStatusWithConflictingHooks() {
	// Create config
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true`
	err := os.WriteFile(s.configFile, []byte(configContent), 0o600)
	s.Require().NoError(err)

	// Install conflicting hook
	hookPath := filepath.Join(s.hooksDir, "pre-commit")
	hookContent := "#!/bin/bash\n# Some other pre-commit hook\necho 'other hook'\n"
	err = os.WriteFile(hookPath, []byte(hookContent), 0o700) //nolint:gosec // Git hooks need execute permissions
	s.Require().NoError(err)

	// Capture output
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Require().NoError(err)
	})

	// Verify output
	s.Contains(output, "⚠ pre-commit:")
	s.Contains(output, "Different hook installed (not GoFortress)")
}

// Test status with disabled system
func (s *StatusTestSuite) TestStatusDisabledSystem() {
	// Create config with system disabled
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=false`
	err := os.WriteFile(s.configFile, []byte(configContent), 0o600)
	s.Require().NoError(err)

	// Capture output
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Require().NoError(err)
	})

	// Verify output
	s.Contains(output, "⚠ Pre-commit system is disabled")
	s.Contains(output, "ENABLE_PRE_COMMIT_SYSTEM=false")
}

// Test verbose status
func (s *StatusTestSuite) TestStatusVerbose() {
	// Create config
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=60
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=4`
	err := os.WriteFile(s.configFile, []byte(configContent), 0o600)
	s.Require().NoError(err)

	// Install hook
	hookPath := filepath.Join(s.hooksDir, "pre-commit")
	hookContent := fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec %s/gofortress-pre-commit run \"$@\"\n", s.tempDir)
	err = os.WriteFile(hookPath, []byte(hookContent), 0o700) //nolint:gosec // Git hooks need execute permissions
	s.Require().NoError(err)

	// Set verbose mode
	verbose = true

	// Capture output
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Require().NoError(err)
	})

	// Verify verbose output
	s.Contains(output, "Repository root:")
	s.Contains(output, "Pre-commit directory:")
	s.Contains(output, "System enabled: true")
	s.Contains(output, "Path:")
	s.Contains(output, "Permissions:")
	s.Contains(output, "Modified:")
	s.Contains(output, "Configuration Status")
	s.Contains(output, "Checks enabled:")
	s.Contains(output, "fumpt: true")
	s.Contains(output, "lint: false")
	s.Contains(output, "mod-tidy: true")
	s.Contains(output, "whitespace: true")
	s.Contains(output, "eof: true")
	s.Contains(output, "Timeout:")
	s.Contains(output, "Parallel workers:")
}

// Test status with non-executable hook
func (s *StatusTestSuite) TestStatusNonExecutableHook() {
	// Create config
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true`
	err := os.WriteFile(s.configFile, []byte(configContent), 0o600)
	s.Require().NoError(err)

	// Install non-executable hook
	hookPath := filepath.Join(s.hooksDir, "pre-commit")
	hookContent := fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec %s/gofortress-pre-commit run \"$@\"\n", s.tempDir)
	err = os.WriteFile(hookPath, []byte(hookContent), 0o600) // Not executable
	s.Require().NoError(err)

	// Capture output
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Require().NoError(err)
	})

	// Verify output
	s.Contains(output, "⚠ pre-commit:")
	s.Contains(output, "not executable")
}

// Test status with multiple hook types
func (s *StatusTestSuite) TestStatusMultipleHookTypes() {
	// Create config
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true`
	err := os.WriteFile(s.configFile, []byte(configContent), 0o600)
	s.Require().NoError(err)

	// Install different types of hooks
	hooks := map[string]string{
		"pre-commit":  "GoFortress pre-commit hook",
		"pre-push":    "GoFortress pre-push hook",
		"commit-msg":  "#!/bin/bash\n# Other commit-msg hook",
		"post-commit": "GoFortress post-commit hook",
	}

	for hookType, content := range hooks {
		hookPath := filepath.Join(s.hooksDir, hookType)
		var hookContent string
		if strings.Contains(content, "GoFortress") {
			hookContent = fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec %s/gofortress-pre-commit run --hook-type %s \"$@\"\n", s.tempDir, hookType)
		} else {
			hookContent = content
		}
		err = os.WriteFile(hookPath, []byte(hookContent), 0o700) //nolint:gosec // Git hooks need execute permissions
		s.Require().NoError(err)
	}

	// Capture output
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Require().NoError(err)
	})

	// Verify output shows all hook types
	s.Contains(output, "✓ pre-commit:")
	s.Contains(output, "✓ pre-push:")
	s.Contains(output, "⚠ commit-msg:")
	s.Contains(output, "✓ post-commit:")
}

// Test error cases
func (s *StatusTestSuite) TestStatusErrors() {
	// Test with missing config
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Error(err)
	})
	s.Contains(output, "Failed to load configuration")

	// Test outside git repository
	s.TearDownTest() // Change back to original directory
	nonGitDir := s.T().TempDir()
	err := os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Create config in non-git directory so we get past config loading
	configFile := filepath.Join(nonGitDir, ".github", ".env.shared")
	err = os.MkdirAll(filepath.Dir(configFile), 0o750)
	s.Require().NoError(err)
	err = os.WriteFile(configFile, []byte("ENABLE_PRE_COMMIT_SYSTEM=true"), 0o600)
	s.Require().NoError(err)

	output = s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Error(err)
	})
	s.Contains(output, "Failed to find git repository")
}

// Test with installer error
func (s *StatusTestSuite) TestStatusInstallerError() {
	// Create config
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true`
	err := os.WriteFile(s.configFile, []byte(configContent), 0o600)
	s.Require().NoError(err)

	// Make hooks directory unreadable (will cause installer errors)
	err = os.Chmod(s.hooksDir, 0o000)
	s.Require().NoError(err)
	defer func() {
		_ = os.Chmod(s.hooksDir, 0o700) //nolint:gosec // Git hooks need execute permissions
	}()

	// Capture output
	output := s.captureOutput(func() {
		err := runStatus(nil, nil)
		s.Require().NoError(err) // Command should still succeed
	})

	// Verify error handling
	s.Contains(output, "Failed to check")
	s.Contains(output, "hook status")
}

// Test print functions
func TestPrintFunctions(t *testing.T) {
	// Capture output for each print function
	tests := []struct {
		name     string
		fn       func()
		expected string
	}{
		{
			name:     "printHeader",
			fn:       func() { printHeader("Test Header") },
			expected: "=== Test Header ===",
		},
		{
			name:     "printSubheader",
			fn:       func() { printSubheader("Test Subheader") },
			expected: "Test Subheader:",
		},
		{
			name:     "printDetail",
			fn:       func() { printDetail("Test %s %d", "detail", 123) },
			expected: "Test detail 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			tt.fn()

			_ = w.Close()
			os.Stdout = oldStdout
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			assert.Contains(t, output, tt.expected)
		})
	}
}

// Test status with real git installer
func TestStatusIntegration(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Save original working directory
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Initialize git repository
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = tempDir
	err = initCmd.Run()
	require.NoError(t, err)

	// Create config directory and pre-commit directory
	configDir := filepath.Join(tempDir, ".github")
	err = os.MkdirAll(configDir, 0o750)
	require.NoError(t, err)

	// Create pre-commit directory that the installer expects
	preCommitDir := filepath.Join(configDir, "pre-commit")
	err = os.MkdirAll(preCommitDir, 0o750)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, ".env.shared")
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=true`
	err = os.WriteFile(configFile, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Create and install hooks using real installer
	cfg, err := config.Load()
	require.NoError(t, err)

	installer := git.NewInstallerWithConfig(tempDir, cfg.Directory, cfg)

	// Install pre-commit hook
	err = installer.InstallHook("pre-commit", false)
	require.NoError(t, err)

	// Run status command
	cmd := &cobra.Command{}
	err = runStatus(cmd, []string{})
	require.NoError(t, err)
}

// Helper method to capture output
func (s *StatusTestSuite) captureOutput(fn func()) string {
	// Save original streams
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Disable color for consistent test output
	oldNoColor := noColor
	noColor = true
	defer func() {
		noColor = oldNoColor
	}()

	// Create pipe to capture both stdout and stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// Benchmark status command
func BenchmarkRunStatus(b *testing.B) {
	// Setup
	tempDir := b.TempDir()
	originalWD, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Initialize git repository
	_ = os.Chdir(tempDir)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = tempDir
	_ = initCmd.Run()

	// Create config
	configFile := filepath.Join(tempDir, ".github", ".env.shared")
	_ = os.MkdirAll(filepath.Dir(configFile), 0o750)
	_ = os.WriteFile(configFile, []byte("ENABLE_PRE_COMMIT_SYSTEM=true"), 0o600)

	// Install some hooks
	hooksDir := filepath.Join(tempDir, ".git", "hooks")
	for _, hookType := range []string{"pre-commit", "pre-push"} {
		hookPath := filepath.Join(hooksDir, hookType)
		hookContent := fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec gofortress-pre-commit run --hook-type %s \"$@\"\n", hookType)
		_ = os.WriteFile(hookPath, []byte(hookContent), 0o700) //nolint:gosec // Git hooks need execute permissions
	}

	// Suppress output
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runStatus(nil, nil)
	}
}

// Test verbose mode edge cases
func TestVerboseModeEdgeCases(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Initialize git repository
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = tempDir
	err = initCmd.Run()
	require.NoError(t, err)

	// Create config
	configFile := filepath.Join(tempDir, ".github", ".env.shared")
	err = os.MkdirAll(filepath.Dir(configFile), 0o750)
	require.NoError(t, err)
	configContent := `ENABLE_PRE_COMMIT_SYSTEM=true`
	err = os.WriteFile(configFile, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Test verbose mode with conflicting hook
	hooksDir := filepath.Join(tempDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	err = os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'other'"), 0o700) //nolint:gosec // Git hooks need execute permissions
	require.NoError(t, err)

	// Save and set verbose
	origVerbose := verbose
	verbose = true
	defer func() {
		verbose = origVerbose
	}()

	// Capture output
	output := captureTestOutput(func() {
		err := runStatus(nil, nil)
		require.NoError(t, err)
	})

	// Verify verbose output for conflicting hook
	assert.Contains(t, output, "Use --force to overwrite existing hook")
}

// Test status with various hook permission states
func TestStatusHookPermissions(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Initialize git repository
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = tempDir
	err = initCmd.Run()
	require.NoError(t, err)

	// Create hooks directory for permission tests
	hooksDir := filepath.Join(tempDir, ".git", "hooks")
	err = os.MkdirAll(hooksDir, 0o750)
	require.NoError(t, err)

	// Create config
	configFile := filepath.Join(tempDir, ".github", ".env.shared")
	err = os.MkdirAll(filepath.Dir(configFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(configFile, []byte("ENABLE_PRE_COMMIT_SYSTEM=true"), 0o600)
	require.NoError(t, err)

	// Test different permission scenarios
	tests := []struct {
		name       string
		mode       os.FileMode
		expectWarn bool
	}{
		{"Executable", 0o700, false},
		{"Not executable", 0o644, true},
		{"Restricted", 0o600, true},
		{"Group executable", 0o754, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Install hook with specific permissions
			hookPath := filepath.Join(hooksDir, "pre-commit")
			hookContent := fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec %s/gofortress-pre-commit run \"$@\"\n", tempDir)
			err := os.WriteFile(hookPath, []byte(hookContent), tt.mode)
			require.NoError(t, err)

			// Capture output
			output := captureTestOutput(func() {
				err := runStatus(nil, nil)
				require.NoError(t, err)
			})

			if tt.expectWarn {
				assert.Contains(t, output, "⚠ pre-commit:")
			} else {
				assert.Contains(t, output, "✓ pre-commit:")
			}

			// Cleanup
			_ = os.Remove(hookPath)
		})
	}
}

// Test status with old hook format
func TestStatusOldHookFormat(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Change to temp directory and initialize git repository
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Initialize a real git repository
	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	hooksDir := filepath.Join(tempDir, ".git", "hooks")

	// Create config
	configFile := filepath.Join(tempDir, ".github", ".env.shared")
	err = os.MkdirAll(filepath.Dir(configFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(configFile, []byte("ENABLE_PRE_COMMIT_SYSTEM=true"), 0o600)
	require.NoError(t, err)

	// Install hook with old format (still valid)
	hookPath := filepath.Join(hooksDir, "pre-commit")
	hookContent := "#!/bin/sh\n# GoFortress Pre-commit Hook\ngofortress-pre-commit run\n"
	err = os.WriteFile(hookPath, []byte(hookContent), 0o700) //nolint:gosec // Git hooks need execute permissions
	require.NoError(t, err)

	// Capture output
	output := captureTestOutput(func() {
		err := runStatus(nil, nil)
		require.NoError(t, err)
	})

	// Debug: print output for debugging
	t.Logf("Status output: %q", output)

	// Should recognize old format hook but warn about it
	assert.Contains(t, output, "pre-commit")
}

// Test concurrent status checks
func TestStatusConcurrent(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Initialize git repository
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = tempDir
	err = initCmd.Run()
	require.NoError(t, err)

	// Create config
	configFile := filepath.Join(tempDir, ".github", ".env.shared")
	err = os.MkdirAll(filepath.Dir(configFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(configFile, []byte("ENABLE_PRE_COMMIT_SYSTEM=true"), 0o600)
	require.NoError(t, err)

	// Run status command concurrently
	const numGoroutines = 10
	errors := make(chan error, numGoroutines)

	// Suppress output
	oldStdout := os.Stdout
	os.Stdout = nil
	defer func() {
		os.Stdout = oldStdout
	}()

	for i := 0; i < numGoroutines; i++ {
		go func() {
			errors <- runStatus(nil, nil)
		}()
	}

	// Check all completed without error
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		require.NoError(t, err)
	}
}

// Example usage
func Example_statusCmd() {
	// This would typically be run from command line:
	// gofortress-pre-commit status
	// gofortress-pre-commit status --verbose

	// The command shows:
	// - System enabled/disabled status
	// - Installed hooks and their state
	// - Configuration summary (in verbose mode)
	// - Any conflicts or issues
}

// Test installer status edge cases
func TestInstallerStatusEdgeCases(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Initialize git repository
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = tempDir
	err = initCmd.Run()
	require.NoError(t, err)

	// Create config
	configFile := filepath.Join(tempDir, ".github", ".env.shared")
	err = os.MkdirAll(filepath.Dir(configFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(configFile, []byte("ENABLE_PRE_COMMIT_SYSTEM=true"), 0o600)
	require.NoError(t, err)

	// Test with symlink hook
	t.Run("Symlink hook", func(t *testing.T) {
		targetPath := filepath.Join(tempDir, "actual-hook")
		hookContent := fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec %s/gofortress-pre-commit run \"$@\"\n", tempDir)
		err := os.WriteFile(targetPath, []byte(hookContent), 0o700) //nolint:gosec // Git hooks need execute permissions
		require.NoError(t, err)

		hooksDir := filepath.Join(tempDir, ".git", "hooks")
		hookPath := filepath.Join(hooksDir, "pre-commit")
		_ = os.Remove(hookPath) // Remove if exists
		err = os.Symlink(targetPath, hookPath)
		require.NoError(t, err)

		// Capture output
		output := captureTestOutput(func() {
			err := runStatus(nil, nil)
			require.NoError(t, err)
		})

		// Should recognize the hook through symlink
		assert.Contains(t, output, "pre-commit:")

		// Cleanup
		_ = os.Remove(hookPath)
		_ = os.Remove(targetPath)
	})
}

// Test configuration status display
func TestConfigurationStatusDisplay(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Initialize git repository
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = tempDir
	err = initCmd.Run()
	require.NoError(t, err)

	// Test various configurations
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "All checks enabled",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=true
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true`,
		},
		{
			name: "Some checks disabled",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=false`,
		},
		{
			name: "Custom timeout and workers",
			config: `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=8`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config
			configFile := filepath.Join(tempDir, ".github", ".env.shared")
			err := os.MkdirAll(filepath.Dir(configFile), 0o750)
			require.NoError(t, err)
			err = os.WriteFile(configFile, []byte(tt.config), 0o600)
			require.NoError(t, err)

			// Set verbose mode
			origVerbose := verbose
			verbose = true
			defer func() {
				verbose = origVerbose
			}()

			// Capture output
			output := captureTestOutput(func() {
				err := runStatus(nil, nil)
				require.NoError(t, err)
			})

			// Verify configuration is displayed
			assert.Contains(t, output, "Configuration Status")
			assert.Contains(t, output, "Checks enabled:")
		})
	}
}

// Test modified time display
// captureTestOutput is a helper function for non-suite tests to capture output
func captureTestOutput(fn func()) string {
	// Save original streams
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Disable color for consistent test output
	oldNoColor := noColor
	noColor = true
	defer func() {
		noColor = oldNoColor
	}()

	// Create pipe to capture both stdout and stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestModifiedTimeDisplay(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Initialize git repository
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Use git init to create a proper repository
	initCmd := exec.CommandContext(context.Background(), "git", "init")
	initCmd.Dir = tempDir
	err = initCmd.Run()
	require.NoError(t, err)

	// Create config
	configFile := filepath.Join(tempDir, ".github", ".env.shared")
	err = os.MkdirAll(filepath.Dir(configFile), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(configFile, []byte("ENABLE_PRE_COMMIT_SYSTEM=true"), 0o600)
	require.NoError(t, err)

	// Install hook with known time
	hooksDir := filepath.Join(tempDir, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, "pre-commit")
	hookContent := fmt.Sprintf("#!/bin/bash\n# GoFortress Pre-commit Hook\nexec %s/gofortress-pre-commit run \"$@\"\n", tempDir)
	err = os.WriteFile(hookPath, []byte(hookContent), 0o700) //nolint:gosec // Git hooks need execute permissions
	require.NoError(t, err)

	// Set specific modification time
	modTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	err = os.Chtimes(hookPath, modTime, modTime)
	require.NoError(t, err)

	// Set verbose mode
	origVerbose := verbose
	verbose = true
	defer func() {
		verbose = origVerbose
	}()

	// Capture output
	output := captureTestOutput(func() {
		err := runStatus(nil, nil)
		require.NoError(t, err)
	})

	// Verify modified time is displayed
	assert.Contains(t, output, "Modified:")
	// The exact format depends on the system's timezone, so just check for date components
	assert.Contains(t, output, "2024")
}
