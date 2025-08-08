package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test coverage data for integration tests
const testCoverageData = `mode: atomic
github.com/mrz1836/go-broadcast/coverage/internal/parser/parser.go:25.23,27.16 2 1
github.com/mrz1836/go-broadcast/coverage/internal/parser/parser.go:30.2,31.16 2 1
github.com/mrz1836/go-broadcast/coverage/internal/parser/parser.go:34.2,35.36 2 1
github.com/mrz1836/go-broadcast/coverage/internal/parser/parser.go:27.16,29.3 1 0
github.com/mrz1836/go-broadcast/coverage/internal/parser/parser.go:31.16,33.3 1 0
github.com/mrz1836/go-broadcast/coverage/internal/badge/generator.go:42.40,44.16 2 1
github.com/mrz1836/go-broadcast/coverage/internal/badge/generator.go:47.2,48.12 2 1
github.com/mrz1836/go-broadcast/coverage/internal/badge/generator.go:44.16,46.3 1 1
`

func TestParseCommand(t *testing.T) {
	// Disable GitHub integration for tests
	_ = os.Setenv("COVERAGE_POST_COMMENTS", "false")
	_ = os.Setenv("COVERAGE_CREATE_STATUSES", "false")
	defer func() { _ = os.Unsetenv("COVERAGE_POST_COMMENTS") }()
	defer func() { _ = os.Unsetenv("COVERAGE_CREATE_STATUSES") }()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "integration_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test coverage file
	coverageFile := filepath.Join(tempDir, "coverage.txt")
	err = os.WriteFile(coverageFile, []byte(testCoverageData), 0o600)
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		contains    []string
	}{
		{
			name: "successful parse with output",
			args: []string{
				"parse",
				"--file", coverageFile,
				"--output", filepath.Join(tempDir, "output.json"),
				"--format", "json",
			},
			expectError: false,
			contains: []string{
				"Coverage Analysis Results",
				"Overall Coverage:",
				"Mode: atomic",
				"Packages:",
				"Output saved to:",
			},
		},
		{
			name: "parse with threshold",
			args: []string{
				"parse",
				"--file", coverageFile,
				"--threshold", "50.0",
			},
			expectError: false,
			contains: []string{
				"Coverage Analysis Results",
				"meets threshold",
			},
		},
		{
			name: "parse with high threshold (should fail)",
			args: []string{
				"parse",
				"--file", coverageFile,
				"--threshold", "95.0",
			},
			expectError: true,
			contains: []string{
				"below threshold",
			},
		},
		{
			name: "parse missing file",
			args: []string{
				"parse",
				"--file", "/nonexistent/file.txt",
			},
			expectError: true,
			contains: []string{
				"failed to parse coverage file",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer

			// Create a new root command for each test
			testCmd := &cobra.Command{Use: "test"}
			testCmd.AddCommand(parseCmd)
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)
			testCmd.SetArgs(tt.args)

			// Execute command
			err := testCmd.Execute()

			// Check error expectation
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Check output contains expected strings
			output := buf.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}
		})
	}
}

func TestHistoryCommand(t *testing.T) {
	// Disable GitHub integration for tests
	_ = os.Setenv("COVERAGE_POST_COMMENTS", "false")
	_ = os.Setenv("COVERAGE_CREATE_STATUSES", "false")
	defer func() { _ = os.Unsetenv("COVERAGE_POST_COMMENTS") }()
	defer func() { _ = os.Unsetenv("COVERAGE_CREATE_STATUSES") }()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "integration_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test coverage file
	coverageFile := filepath.Join(tempDir, "coverage.txt")
	err = os.WriteFile(coverageFile, []byte(testCoverageData), 0o600)
	require.NoError(t, err)

	// Create history directory
	historyDir := filepath.Join(tempDir, "history")
	err = os.MkdirAll(historyDir, 0o750)
	require.NoError(t, err)

	// First, add some data to history for tests that need it
	_ = os.Setenv("COVERAGE_HISTORY_PATH", historyDir)
	defer func() { _ = os.Unsetenv("COVERAGE_HISTORY_PATH") }()

	// Add initial history entry
	addCmd := &cobra.Command{Use: "test"}
	// Create a fresh history command for setup
	setupHistoryCmd := &cobra.Command{
		Use:   "history",
		Short: "Manage coverage history",
		Long:  `Manage historical coverage data for trend analysis and tracking.`,
		RunE:  historyCmd.RunE,
	}
	setupHistoryCmd.Flags().StringP("add", "a", "", "Add coverage data file to history")
	setupHistoryCmd.Flags().StringP("branch", "b", "", "Branch name")
	setupHistoryCmd.Flags().StringP("commit", "c", "", "Commit SHA")
	setupHistoryCmd.Flags().String("commit-url", "", "Commit URL")
	setupHistoryCmd.Flags().BoolP("trend", "t", false, "Show trend analysis")
	setupHistoryCmd.Flags().BoolP("stats", "s", false, "Show statistics")
	setupHistoryCmd.Flags().Bool("cleanup", false, "Clean up old entries")
	setupHistoryCmd.Flags().IntP("days", "d", 30, "Number of days for trend analysis")
	setupHistoryCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")

	addCmd.AddCommand(setupHistoryCmd)
	addCmd.SetArgs([]string{"history", "--add", coverageFile, "--branch", "master", "--commit", "abc123"})
	var addBuf bytes.Buffer
	addCmd.SetOut(&addBuf)
	addCmd.SetErr(&addBuf)
	err = addCmd.Execute()
	require.NoError(t, err, "Failed to add initial history entry")

	tests := []struct {
		name        string
		args        []string
		expectError bool
		contains    []string
		envVars     map[string]string
	}{
		{
			name: "add coverage to history",
			args: []string{
				"history",
				"--add", coverageFile,
				"--branch", "master",
				"--commit", "abc123",
			},
			expectError: false,
			contains: []string{
				"Coverage recorded successfully!",
				"Branch: master",
				"Commit: abc123",
			},
			envVars: map[string]string{
				"COVERAGE_HISTORY_PATH": historyDir,
			},
		},
		{
			name: "show history statistics",
			args: []string{
				"history",
				"--stats",
			},
			expectError: false,
			contains: []string{
				"Coverage History Statistics",
				"Total Entries:",
			},
			envVars: map[string]string{
				"COVERAGE_HISTORY_PATH": historyDir,
				"COVERAGE_INPUT_FILE":   "/nonexistent/file.txt",
			},
		},
		{
			name: "show trend analysis",
			args: []string{
				"history",
				"--trend",
				"--branch", "master",
				"--days", "30",
			},
			expectError: false,
			contains: []string{
				"Coverage Trend Analysis",
				"Branch: master",
				"Period: 30 days",
			},
			envVars: map[string]string{
				"COVERAGE_HISTORY_PATH": historyDir,
				"COVERAGE_INPUT_FILE":   "/nonexistent/file.txt",
			},
		},
		{
			name: "show latest entry",
			args: []string{
				"history",
				"--branch", "master",
			},
			expectError: false,
			contains: []string{
				"Latest Coverage Entry",
				"Branch: master",
			},
			envVars: map[string]string{
				"COVERAGE_HISTORY_PATH": historyDir,
				"COVERAGE_INPUT_FILE":   "/nonexistent/file.txt",
			},
		},
		{
			name: "cleanup history",
			args: []string{
				"history",
				"--cleanup",
			},
			expectError: false,
			contains: []string{
				"History cleanup completed successfully!",
			},
			envVars: map[string]string{
				"COVERAGE_HISTORY_PATH": historyDir,
				"COVERAGE_INPUT_FILE":   "/nonexistent/file.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					_ = os.Unsetenv(key)
				}
			}()

			// Capture output
			var buf bytes.Buffer

			// Create a new root command for each test
			testCmd := &cobra.Command{Use: "test"}
			// Create a fresh history command for each test
			testHistoryCmd := &cobra.Command{
				Use:   "history",
				Short: "Manage coverage history",
				Long:  `Manage historical coverage data for trend analysis and tracking.`,
				RunE:  historyCmd.RunE,
			}
			testHistoryCmd.Flags().StringP("add", "a", "", "Add coverage data file to history")
			testHistoryCmd.Flags().StringP("branch", "b", "", "Branch name")
			testHistoryCmd.Flags().StringP("commit", "c", "", "Commit SHA")
			testHistoryCmd.Flags().String("commit-url", "", "Commit URL")
			testHistoryCmd.Flags().BoolP("trend", "t", false, "Show trend analysis")
			testHistoryCmd.Flags().BoolP("stats", "s", false, "Show statistics")
			testHistoryCmd.Flags().Bool("cleanup", false, "Clean up old entries")
			testHistoryCmd.Flags().IntP("days", "d", 30, "Number of days for trend analysis")
			testHistoryCmd.Flags().StringP("format", "f", "text", "Output format (text, json)")

			testCmd.AddCommand(testHistoryCmd)
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)
			testCmd.SetArgs(tt.args)

			// Execute command
			err := testCmd.Execute()

			// Check error expectation
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Check output contains expected strings
			output := buf.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}
		})
	}
}

func TestCommentCommand(t *testing.T) {
	// Disable GitHub integration for tests
	_ = os.Setenv("COVERAGE_POST_COMMENTS", "false")
	_ = os.Setenv("COVERAGE_CREATE_STATUSES", "false")
	_ = os.Unsetenv("GITHUB_PR_NUMBER") // Clear any leftover PR number
	defer func() { _ = os.Unsetenv("COVERAGE_POST_COMMENTS") }()
	defer func() { _ = os.Unsetenv("COVERAGE_CREATE_STATUSES") }()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "integration_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test coverage file
	coverageFile := filepath.Join(tempDir, "coverage.txt")
	err = os.WriteFile(coverageFile, []byte(testCoverageData), 0o600)
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		contains    []string
		envVars     map[string]string
	}{
		{
			name: "dry run comment generation",
			args: []string{
				"comment",
				"--pr", "123",
				"--coverage", coverageFile,
				"--dry-run",
				"--enable-analysis=false",
			},
			expectError: false,
			contains: []string{
				"PR Comment Preview (Dry Run)",
				"PR: 123",
				"Template: comprehensive",
				"Repository: test-owner/test-repo",
			},
			envVars: map[string]string{
				"GITHUB_TOKEN":            "fake-token",
				"GITHUB_REPOSITORY_OWNER": "test-owner",
				"GITHUB_REPOSITORY":       "test-owner/test-repo",
			},
		},
		{
			name: "missing GitHub token",
			args: []string{
				"comment",
				"--pr", "123",
				"--coverage", coverageFile,
			},
			expectError: true,
			contains: []string{
				"GitHub token is required",
			},
		},
		{
			name: "missing PR number",
			args: []string{
				"comment",
				"--coverage", coverageFile,
			},
			expectError: true,
			contains: []string{
				"pull request number is required",
			},
			envVars: map[string]string{
				"GITHUB_TOKEN":            "fake-token",
				"GITHUB_REPOSITORY_OWNER": "test-owner",
				"GITHUB_REPOSITORY":       "test-owner/test-repo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					_ = os.Unsetenv(key)
				}
			}()

			// Capture output
			var buf bytes.Buffer

			// Create a new root command for each test
			testCmd := &cobra.Command{Use: "test"}
			// Use the actual comment command with all its flags
			testCommentCmd := &cobra.Command{
				Use:   "comment",
				Short: "Create PR coverage comment with analysis and templates",
				Long:  `Create or update pull request comments with coverage information.`,
				RunE:  commentCmd.RunE,
			}
			// Add all the current flags from the actual comment command
			testCommentCmd.Flags().IntP("pr", "p", 0, "Pull request number (defaults to GITHUB_PR_NUMBER)")
			testCommentCmd.Flags().StringP("coverage", "c", "", "Coverage data file")
			testCommentCmd.Flags().String("base-coverage", "", "Base coverage data file for comparison")
			testCommentCmd.Flags().String("badge-url", "", "Badge URL (auto-generated if not provided)")
			testCommentCmd.Flags().String("report-url", "", "Report URL (auto-generated if not provided)")
			testCommentCmd.Flags().Bool("status", false, "Create status checks")
			testCommentCmd.Flags().Bool("block-merge", false, "Block PR merge on coverage failure")
			testCommentCmd.Flags().Bool("generate-badges", false, "Generate PR-specific badges")
			testCommentCmd.Flags().Bool("enable-analysis", true, "Enable detailed coverage analysis and comparison")
			testCommentCmd.Flags().Bool("anti-spam", true, "Enable anti-spam features")
			testCommentCmd.Flags().Bool("dry-run", false, "Show preview of comment without posting")

			testCmd.AddCommand(testCommentCmd)
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)
			testCmd.SetArgs(tt.args)

			// Execute command
			err := testCmd.Execute()

			// Check error expectation
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Check output contains expected strings
			output := buf.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}
		})
	}
}

func TestCompleteCommand(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "integration_test_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test coverage file
	coverageFile := filepath.Join(tempDir, "coverage.txt")
	err = os.WriteFile(coverageFile, []byte(testCoverageData), 0o600)
	require.NoError(t, err)

	// Create output directory
	outputDir := filepath.Join(tempDir, "output")

	tests := []struct {
		name        string
		args        []string
		expectError bool
		contains    []string
		checkFiles  []string
		envVars     map[string]string
	}{
		{
			name: "complete pipeline dry run",
			args: []string{
				"complete",
				"--input", coverageFile,
				"--output", outputDir,
				"--dry-run",
				"--skip-github",
			},
			expectError: false,
			contains: []string{
				"Starting GoFortress Coverage Pipeline",
				"Step 1: Parsing coverage data",
				"Step 2: Generating coverage badge",
				"Step 3: Generating HTML report",
				"Step 4: Generating coverage dashboard",
				"Step 5: Coverage history analysis",
				"Step 6: GitHub integration (skipped)",
				"Pipeline Complete!",
				"Mode: DRY RUN",
			},
			envVars: map[string]string{
				"COVERAGE_AUTO_CREATE_DIRS": "true",
				"COVERAGE_POST_COMMENTS":    "false",
				"COVERAGE_CREATE_STATUSES":  "false",
			},
		},
		{
			name: "complete pipeline with file generation",
			args: []string{
				"complete",
				"--input", coverageFile,
				"--output", outputDir,
				"--skip-github",
				"--skip-history",
			},
			expectError: false,
			contains: []string{
				"Starting GoFortress Coverage Pipeline",
				"Pipeline Complete!",
				"Badge:",
				"Report:",
			},
			checkFiles: []string{
				filepath.Join(outputDir, "reports", "branch", "master", "coverage.svg"),
				filepath.Join(outputDir, "reports", "branch", "master", "coverage.html"),
				filepath.Join(outputDir, "reports", "branch", "master", "index.html"),
				filepath.Join(outputDir, "reports", "branch", "master", "dashboard.html"),
				filepath.Join(outputDir, "coverage.svg"), // Also check root badge
			},
			envVars: map[string]string{
				"COVERAGE_AUTO_CREATE_DIRS": "true",
				"COVERAGE_POST_COMMENTS":    "false",
				"COVERAGE_CREATE_STATUSES":  "false",
			},
		},
		{
			name: "complete pipeline with GitHub context (dry run)",
			args: []string{
				"complete",
				"--input", coverageFile,
				"--output", outputDir,
				"--dry-run",
			},
			expectError: false,
			contains: []string{
				"Starting GoFortress Coverage Pipeline",
				"Step 6: GitHub integration",
				"PR comment creation is deprecated in complete command",
				"Would create commit status",
				"Pipeline Complete!",
			},
			envVars: map[string]string{
				"COVERAGE_AUTO_CREATE_DIRS": "true",
				"GITHUB_TOKEN":              "fake-token",
				"GITHUB_REPOSITORY_OWNER":   "test-owner",
				"GITHUB_REPOSITORY":         "test-owner/test-repo",
				"GITHUB_SHA":                "abc123def456",
				"GITHUB_PR_NUMBER":          "123",
				"COVERAGE_POST_COMMENTS":    "true",
				"COVERAGE_CREATE_STATUSES":  "true",
			},
		},
		{
			name: "missing input file",
			args: []string{
				"complete",
				"--input", "/nonexistent/file.txt",
				"--output", outputDir,
			},
			expectError: true,
			contains: []string{
				"failed to parse coverage file",
			},
			envVars: map[string]string{
				"COVERAGE_POST_COMMENTS":   "false",
				"COVERAGE_CREATE_STATUSES": "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					_ = os.Unsetenv(key)
				}
			}()

			// Capture output
			var buf bytes.Buffer

			// Create a new root command for each test
			testCmd := &cobra.Command{Use: "test"}
			// Create a fresh complete command for each test
			testCompleteCmd := &cobra.Command{
				Use:   "complete",
				Short: "Run complete coverage pipeline",
				Long: `Run the complete coverage pipeline: parse coverage, generate badge and report,
update history, and create GitHub PR comment if in PR context.`,
				RunE: completeCmd.RunE,
			}
			testCompleteCmd.Flags().StringP("input", "i", "", "Input coverage file")
			testCompleteCmd.Flags().StringP("output", "o", "", "Output directory")
			testCompleteCmd.Flags().Bool("skip-history", false, "Skip history tracking")
			testCompleteCmd.Flags().Bool("skip-github", false, "Skip GitHub integration")
			testCompleteCmd.Flags().Bool("dry-run", false, "Show what would be done without actually doing it")

			testCmd.AddCommand(testCompleteCmd)
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)
			testCmd.SetArgs(tt.args)

			// Execute command
			err := testCmd.Execute()

			// Check error expectation
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Check output contains expected strings
			output := buf.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}

			// Check files were created
			for _, filePath := range tt.checkFiles {
				assert.FileExists(t, filePath, "File should be created: %s", filePath)
			}
		})
	}
}

func TestRootCommandHelp(t *testing.T) {
	// Disable GitHub integration for tests
	_ = os.Setenv("COVERAGE_POST_COMMENTS", "false")
	_ = os.Setenv("COVERAGE_CREATE_STATUSES", "false")
	defer func() { _ = os.Unsetenv("COVERAGE_POST_COMMENTS") }()
	defer func() { _ = os.Unsetenv("COVERAGE_CREATE_STATUSES") }()

	// Capture output
	var buf bytes.Buffer

	// Create a new root command with all available subcommands
	testCmd := &cobra.Command{
		Use:   "gofortress-coverage",
		Short: "Go-native coverage system for GoFortress CI/CD",
		Long: `GoFortress Coverage is a self-contained, Go-native coverage system that provides
professional coverage tracking, badge generation, and reporting while maintaining
the simplicity and performance that Go developers expect.`,
	}
	testCmd.AddCommand(completeCmd, historyCmd, commentCmd, parseCmd)
	testCmd.SetOut(&buf)
	testCmd.SetErr(&buf)
	testCmd.SetArgs([]string{"--help"})

	// Execute command
	err := testCmd.Execute()
	require.NoError(t, err)

	// Check output contains expected commands (only the ones that currently exist)
	output := buf.String()
	expectedCommands := []string{"complete", "history", "comment", "parse"}
	for _, cmd := range expectedCommands {
		assert.Contains(t, output, cmd, "Help should contain command: %s", cmd)
	}

	// Verify removed commands are not present as actual commands (not just substring matches)
	assert.NotContains(t, output, "  report ", "Help should NOT contain report as a command")
	assert.NotContains(t, output, "  badge ", "Help should NOT contain badge as a command")
}

func TestCommandFlags(t *testing.T) {
	// Disable GitHub integration for tests
	_ = os.Setenv("COVERAGE_POST_COMMENTS", "false")
	_ = os.Setenv("COVERAGE_CREATE_STATUSES", "false")
	defer func() { _ = os.Unsetenv("COVERAGE_POST_COMMENTS") }()
	defer func() { _ = os.Unsetenv("COVERAGE_CREATE_STATUSES") }()

	tests := []struct {
		name     string
		cmd      *cobra.Command
		helpArgs []string
		contains []string
	}{
		// {
		// 	name:     "parse command flags",
		// 	cmd:      parseCmd,
		// 	helpArgs: []string{"parse", "--help"},
		// 	contains: []string{"--file", "--output", "--format", "--exclude-tests", "--threshold"},
		// },
		// {
		// 	name:     "badge command flags",
		// 	cmd:      badgeCmd,
		// 	helpArgs: []string{"badge", "--help"},
		// 	contains: []string{"--coverage", "--style", "--output", "--input", "--label", "--logo"},
		// },
		// {
		// 	name:     "report command flags",
		// 	cmd:      reportCmd,
		// 	helpArgs: []string{"report", "--help"},
		// 	contains: []string{"--input", "--output", "--theme", "--title", "--show-packages"},
		// },
		{
			name:     "history command flags",
			cmd:      historyCmd,
			helpArgs: []string{"history", "--help"},
			contains: []string{"--add", "--branch", "--commit", "--trend", "--stats", "--cleanup"},
		},
		{
			name:     "comment command flags",
			cmd:      commentCmd,
			helpArgs: []string{"comment", "--help"},
			contains: []string{"--pr", "--coverage", "--badge-url", "--status", "--dry-run"},
		},
		{
			name:     "complete command flags",
			cmd:      completeCmd,
			helpArgs: []string{"complete", "--help"},
			contains: []string{"--input", "--output", "--skip-history", "--skip-github", "--dry-run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer

			// Create a new root command
			testCmd := &cobra.Command{Use: "test"}
			testCmd.AddCommand(tt.cmd)
			testCmd.SetOut(&buf)
			testCmd.SetErr(&buf)
			testCmd.SetArgs(tt.helpArgs)

			// Execute command
			err := testCmd.Execute()
			require.NoError(t, err)

			// Check output contains expected flags
			output := buf.String()
			for _, flag := range tt.contains {
				assert.Contains(t, output, flag, "Help should contain flag: %s", flag)
			}
		})
	}
}

// Helper function to clear environment variables
func clearTestEnv() {
	envVars := []string{
		"COVERAGE_INPUT_FILE", "COVERAGE_OUTPUT_DIR", "COVERAGE_THRESHOLD",
		"COVERAGE_AUTO_CREATE_DIRS", "COVERAGE_HISTORY_PATH",
		"GITHUB_TOKEN", "GITHUB_REPOSITORY_OWNER", "GITHUB_REPOSITORY",
		"GITHUB_SHA", "GITHUB_PR_NUMBER",
	}

	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}
}

func TestMain(m *testing.M) {
	// Setup
	clearTestEnv()

	// Run tests
	code := m.Run()

	// Cleanup
	clearTestEnv()

	os.Exit(code)
}
