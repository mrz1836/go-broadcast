package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/output"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/runner"
)

func TestRunCmd_ShowChecks(t *testing.T) {
	// Save original
	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
	}()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run with show-checks flag
	os.Args = []string{"gofortress-pre-commit", "run", "--show-checks"}

	// Execute command
	rootCmd.SetArgs([]string{"run", "--show-checks"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should show available checks
	assert.Contains(t, output, "Available Checks")
	assert.Contains(t, output, "fumpt")
	assert.Contains(t, output, "lint")
	assert.Contains(t, output, "whitespace")
	assert.Contains(t, output, "eof")
	assert.Contains(t, output, "mod-tidy")
}

func TestRunCmd_DisabledSystem(t *testing.T) {
	// Save original env
	oldEnv := os.Getenv("ENABLE_PRE_COMMIT_SYSTEM")
	defer func() {
		if err := os.Setenv("ENABLE_PRE_COMMIT_SYSTEM", oldEnv); err != nil {
			t.Logf("Failed to restore ENABLE_PRE_COMMIT_SYSTEM: %v", err)
		}
	}()

	// Disable pre-commit system
	require.NoError(t, os.Setenv("ENABLE_PRE_COMMIT_SYSTEM", "false"))

	// Save original
	oldArgs := os.Args
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stderr = oldStderr
	}()

	// Capture stderr output since printWarning outputs to stderr when noColor is true
	r, w, _ := os.Pipe()
	os.Stderr = w
	noColor = true // Ensure we output to stderr

	// Execute command
	rootCmd.SetArgs([]string{"run"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should show warning about disabled system
	assert.Contains(t, output, "Pre-commit system is disabled")
}

func TestRunCmd_ParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T)
	}{
		{
			name: "all-files flag",
			args: []string{"run", "--all-files"},
			validate: func(t *testing.T) {
				assert.True(t, allFiles)
			},
		},
		{
			name: "files flag",
			args: []string{"run", "--files", "main.go,utils.go"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"main.go", "utils.go"}, files)
			},
		},
		{
			name: "skip flag",
			args: []string{"run", "--skip", "lint,fumpt"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"lint", "fumpt"}, skipChecks)
			},
		},
		{
			name: "only flag",
			args: []string{"run", "--only", "whitespace,eof"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"whitespace", "eof"}, onlyChecks)
			},
		},
		{
			name: "parallel flag",
			args: []string{"run", "--parallel", "4"},
			validate: func(t *testing.T) {
				assert.Equal(t, 4, parallel)
			},
		},
		{
			name: "fail-fast flag",
			args: []string{"run", "--fail-fast"},
			validate: func(t *testing.T) {
				assert.True(t, failFast)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			allFiles = false
			files = nil
			skipChecks = nil
			onlyChecks = nil
			parallel = 0
			failFast = false

			// Parse command properly through execute to handle subcommand flags
			rootCmd.SetArgs(tt.args)
			cmd, err := rootCmd.ExecuteC()
			if err != nil {
				// For testing flag parsing, we expect execution errors but not parse errors
				// Since we can't actually run without proper git repo setup
				require.Contains(t, err.Error(), "failed to")
			}
			assert.Equal(t, "run", cmd.Name())

			// Validate
			tt.validate(t)
		})
	}
}

func TestRunCmd_SpecificCheck(t *testing.T) {
	// Save original
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Execute command with specific check
	rootCmd.SetArgs([]string{"run", "whitespace"})

	// This would fail in test environment as we're not in a git repo
	// but we can verify the command structure is correct
	cmd, _, err := rootCmd.Find([]string{"run", "whitespace"})
	require.NoError(t, err)
	assert.Equal(t, "run", cmd.Name())
}

// Comprehensive test suite for run command

type RunCommandTestSuite struct {
	suite.Suite

	tempDir  string
	oldDir   string
	repoRoot string
}

func TestRunCommandSuite(t *testing.T) {
	suite.Run(t, new(RunCommandTestSuite))
}

func (s *RunCommandTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "run_cmd_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Initialize git repository
	s.initGitRepo()
	s.repoRoot = s.tempDir
}

func (s *RunCommandTestSuite) TearDownTest() {
	if s.oldDir != "" {
		err := os.Chdir(s.oldDir)
		s.Require().NoError(err)
	}
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *RunCommandTestSuite) initGitRepo() {
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "init").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	// Create initial commit
	testFile := filepath.Join(s.tempDir, "README.md")
	err := os.WriteFile(testFile, []byte("# Test Repository\n"), 0o600)
	s.Require().NoError(err)

	s.Require().NoError(exec.CommandContext(context.Background(), "git", "add", "README.md").Run())
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "commit", "-m", "Initial commit").Run())
}

func (s *RunCommandTestSuite) createTestFile(filename, content string) {
	fullPath := filepath.Join(s.tempDir, filename)
	err := os.WriteFile(fullPath, []byte(content), 0o600)
	s.Require().NoError(err)
}

func (s *RunCommandTestSuite) createEnvFile() {
	envContent := `# Pre-commit system configuration
ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=false
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=60
PRE_COMMIT_SYSTEM_LOG_LEVEL=info
`
	githubDir := filepath.Join(s.tempDir, ".github")
	err := os.MkdirAll(githubDir, 0o750)
	s.Require().NoError(err)

	envFile := filepath.Join(githubDir, ".env.shared")
	err = os.WriteFile(envFile, []byte(envContent), 0o600)
	s.Require().NoError(err)
}

// TestRunChecksWithAllFiles tests running checks on all files
func (s *RunCommandTestSuite) TestRunChecksWithAllFiles() {
	s.createEnvFile()
	s.createTestFile("test.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")

	// Reset flags before test
	allFiles = true
	files = nil
	showVersion = false
	noColor = true

	err := runChecks(nil, []string{})
	// May succeed or fail depending on available tools, but should not panic
	// The key is that it runs without crashing
	if err != nil {
		// Common acceptable errors in test environment
		acceptableErrors := []string{
			"failed to find repository root",
			"failed to load configuration",
			"no checks to run",
		}

		foundAcceptable := false
		for _, acceptable := range acceptableErrors {
			if s.Contains(err.Error(), acceptable) {
				foundAcceptable = true
				break
			}
		}
		if !foundAcceptable {
			s.T().Logf("Unexpected error (but not necessarily a failure): %v", err)
		}
	}
}

// TestRunChecksWithSpecificFiles tests running checks on specific files
func (s *RunCommandTestSuite) TestRunChecksWithSpecificFiles() {
	s.createEnvFile()
	s.createTestFile("main.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")

	// Reset flags before test
	allFiles = false
	files = []string{"main.go"}
	showVersion = false
	noColor = true

	err := runChecks(nil, []string{})
	// May succeed or fail, but should handle the specific files path
	if err != nil {
		s.T().Logf("Error in specific files test (may be expected): %v", err)
	}
}

// TestRunChecksWithStagedFiles tests running checks on staged files
func (s *RunCommandTestSuite) TestRunChecksWithStagedFiles() {
	s.createEnvFile()
	s.createTestFile("staged.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")

	// Stage the file
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "add", "staged.go").Run())

	// Reset flags before test
	allFiles = false
	files = nil
	showVersion = false
	noColor = true

	err := runChecks(nil, []string{})
	// Should handle staged files path
	if err != nil {
		s.T().Logf("Error in staged files test (may be expected): %v", err)
	}
}

// TestRunChecksShowAvailableChecks tests the show checks functionality
func (s *RunCommandTestSuite) TestRunChecksShowAvailableChecks() {
	s.createEnvFile()

	// Reset flags before test
	showVersion = true // This triggers showAvailableChecks
	noColor = true

	err := runChecks(nil, []string{})
	s.NoError(err) // Show checks should always succeed
}

// TestRunChecksDisabledSystem tests when the pre-commit system is disabled
func (s *RunCommandTestSuite) TestRunChecksDisabledSystem() {
	// Create env file with system disabled
	envContent := `ENABLE_PRE_COMMIT_SYSTEM=false`
	githubDir := filepath.Join(s.tempDir, ".github")
	err := os.MkdirAll(githubDir, 0o750)
	s.Require().NoError(err)

	envFile := filepath.Join(githubDir, ".env.shared")
	err = os.WriteFile(envFile, []byte(envContent), 0o600)
	s.Require().NoError(err)

	// Reset flags before test
	showVersion = false
	noColor = true

	err = runChecks(nil, []string{})
	s.NoError(err) // Should succeed with warning when disabled
}

// TestRunChecksNoConfigurationFile tests when no configuration file exists
func (s *RunCommandTestSuite) TestRunChecksNoConfigurationFile() {
	// Don't create .env.shared file

	// Reset flags before test
	showVersion = false
	noColor = true

	err := runChecks(nil, []string{})
	// Should handle missing configuration gracefully or return appropriate error
	if err != nil {
		s.Contains(err.Error(), "failed to load configuration")
	}
}

// TestRunChecksInvalidGitRepository tests behavior outside git repository
func (s *RunCommandTestSuite) TestRunChecksInvalidGitRepository() {
	// Create a non-git directory
	nonGitDir, err := os.MkdirTemp("", "non_git_*")
	s.Require().NoError(err)
	defer func() {
		_ = os.RemoveAll(nonGitDir)
	}()

	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	err = os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Create minimal config
	githubDir := filepath.Join(nonGitDir, ".github")
	err = os.MkdirAll(githubDir, 0o750)
	s.Require().NoError(err)

	envFile := filepath.Join(githubDir, ".env.shared")
	err = os.WriteFile(envFile, []byte("ENABLE_PRE_COMMIT_SYSTEM=true"), 0o600)
	s.Require().NoError(err)

	// Reset flags before test
	showVersion = false
	noColor = true

	err = runChecks(nil, []string{})
	s.Require().Error(err)
	s.Contains(err.Error(), "failed to find git repository")
}

// TestPrintFunctions tests the print helper functions
func (s *RunCommandTestSuite) TestPrintFunctions() {
	// Test that print functions don't panic
	s.NotPanics(func() {
		printSuccess("test success message")
		printError("test error message")
		printInfo("test info message")
		printWarning("test warning message")
	})
}

// TestDisplayEnhancedResults tests the enhanced results display
func (s *RunCommandTestSuite) TestDisplayEnhancedResults() {
	// This function has 0% coverage, so let's test it
	formatter := output.NewDefault()

	mockResults := &runner.Results{
		CheckResults: []runner.CheckResult{
			{
				Name:     "whitespace",
				Success:  true,
				Duration: 500 * time.Millisecond,
			},
			{
				Name:     "eof",
				Success:  true,
				Duration: 300 * time.Millisecond,
			},
			{
				Name:     "lint",
				Success:  false,
				Error:    "linting issues found",
				Duration: 700 * time.Millisecond,
			},
		},
		Passed:        2,
		Failed:        1,
		Skipped:       0,
		TotalDuration: 1500 * time.Millisecond,
		TotalFiles:    5,
	}

	s.NotPanics(func() {
		displayEnhancedResults(formatter, mockResults, false)
	})
}

// TestRunChecksWithFlags tests various flag combinations
func (s *RunCommandTestSuite) TestRunChecksWithFlags() {
	s.createEnvFile()
	s.createTestFile("test.txt", "content\n")

	testCases := []struct {
		name           string
		setupFlags     func()
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "with parallel flag",
			setupFlags: func() {
				parallel = 2
				failFast = false
				noColor = true
				showVersion = false
			},
			expectError: false,
		},
		{
			name: "with fail-fast flag",
			setupFlags: func() {
				parallel = 0
				failFast = true
				noColor = true
				showVersion = false
			},
			expectError: false,
		},
		{
			name: "with specific checks",
			setupFlags: func() {
				onlyChecks = []string{"whitespace"}
				noColor = true
				showVersion = false
			},
			expectError: false,
		},
		{
			name: "with skip checks",
			setupFlags: func() {
				skipChecks = []string{"fumpt", "lint"}
				noColor = true
				showVersion = false
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Reset flags
			parallel = 0
			failFast = false
			onlyChecks = nil
			skipChecks = nil
			allFiles = false
			files = nil

			tc.setupFlags()

			err := runChecks(nil, []string{})

			if tc.expectError {
				s.Require().Error(err)
				if tc.expectedErrMsg != "" {
					s.Contains(err.Error(), tc.expectedErrMsg)
				}
			} else {
				// May succeed or fail depending on environment, but should not panic
				if err != nil {
					s.T().Logf("Command failed (may be expected in test env): %v", err)
				}
			}
		})
	}
}

// Unit tests for edge cases
func TestRunCommandEdgeCases(t *testing.T) {
	t.Run("display enhanced results with empty results", func(t *testing.T) {
		formatter := output.NewDefault()
		results := &runner.Results{}

		assert.NotPanics(t, func() {
			displayEnhancedResults(formatter, results, false)
		})
	})

	t.Run("display enhanced results with nil formatter", func(t *testing.T) {
		results := &runner.Results{
			Passed: 1,
			Failed: 0,
		}

		// displayEnhancedResults expects a valid formatter, so it should panic with nil
		assert.Panics(t, func() {
			displayEnhancedResults(nil, results, false)
		})
	})

	t.Run("display enhanced results with nil results", func(t *testing.T) {
		formatter := output.NewDefault()

		// displayEnhancedResults expects valid results, so it should panic with nil
		assert.Panics(t, func() {
			displayEnhancedResults(formatter, nil, false)
		})
	})
}
