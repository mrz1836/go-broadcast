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

func TestParseCommand(t *testing.T) { //nolint:revive // function naming
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

func TestBadgeCommand(t *testing.T) { //nolint:revive // function naming
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
		checkFiles  []string
		envVars     map[string]string
	}{
		{
			name: "generate badge with coverage percentage",
			args: []string{
				"badge",
				"--coverage", "85.5",
				"--output", filepath.Join(tempDir, "badge.svg"),
				"--style", "flat",
			},
			expectError: false,
			contains: []string{
				"Coverage badge generated successfully!",
				"Coverage: 85.50%",
				"Style: flat",
				"🟡 Good",
			},
			checkFiles: []string{filepath.Join(tempDir, "badge.svg")},
		},
		{
			name: "generate badge from input file",
			args: []string{
				"badge",
				"--input", coverageFile,
				"--output", filepath.Join(tempDir, "badge2.svg"),
				"--style", "flat-square",
			},
			expectError: false,
			contains: []string{
				"Coverage badge generated successfully!",
				"Style: flat-square",
			},
			checkFiles: []string{filepath.Join(tempDir, "badge2.svg")},
		},
		{
			name: "missing coverage percentage and input",
			args: []string{
				"badge",
				"--output", filepath.Join(tempDir, "badge3.svg"),
			},
			expectError: true,
			contains: []string{
				"coverage percentage is required",
			},
			envVars: map[string]string{
				"COVERAGE_INPUT_FILE": "/nonexistent/coverage.txt",
			},
		},
		{
			name: "invalid coverage percentage",
			args: []string{
				"badge",
				"--coverage", "150.0",
				"--output", filepath.Join(tempDir, "badge4.svg"),
			},
			expectError: true,
			contains: []string{
				"coverage percentage must be between 0 and 100",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for config
			_ = os.Setenv("COVERAGE_AUTO_CREATE_DIRS", "true")
			defer func() { _ = os.Unsetenv("COVERAGE_AUTO_CREATE_DIRS") }()

			// Set test-specific environment variables
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
			// Create a fresh badge command for each test
			testBadgeCmd := &cobra.Command{
				Use:   "badge",
				Short: "Generate coverage badge",
				Long:  `Generate SVG coverage badges for README files and GitHub Pages.`,
				RunE:  badgeCmd.RunE,
			}
			testBadgeCmd.Flags().Float64P("coverage", "c", 0, "Coverage percentage")
			testBadgeCmd.Flags().StringP("style", "s", "", "Badge style")
			testBadgeCmd.Flags().StringP("output", "o", "", "Output file path")
			testBadgeCmd.Flags().StringP("input", "i", "", "Input coverage file")
			testBadgeCmd.Flags().StringP("label", "l", "", "Badge label")
			testBadgeCmd.Flags().String("logo", "", "Logo")
			testBadgeCmd.Flags().String("logo-color", "", "Logo color")

			testCmd.AddCommand(testBadgeCmd)
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

				// Verify SVG content
				content, err := os.ReadFile(filePath) //nolint:gosec // Test file path is controlled
				require.NoError(t, err)
				assert.Contains(t, string(content), "<svg", "File should contain SVG content")
			}
		})
	}
}

func TestReportCommand(t *testing.T) { //nolint:revive // function naming
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
		checkFiles  []string
		envVars     map[string]string
	}{
		{
			name: "generate HTML report",
			args: []string{
				"report",
				"--input", coverageFile,
				"--output", filepath.Join(tempDir, "report.html"),
				"--theme", "github-dark",
				"--title", "Test Coverage Report",
			},
			expectError: false,
			contains: []string{
				"Coverage report generated successfully!",
				"Title: Test Coverage Report",
				"Theme: github-dark",
			},
			checkFiles: []string{filepath.Join(tempDir, "report.html")},
			envVars: map[string]string{
				"COVERAGE_AUTO_CREATE_DIRS": "true",
			},
		},
		{
			name: "generate report with options",
			args: []string{
				"report",
				"--input", coverageFile,
				"--output", filepath.Join(tempDir, "report2.html"),
				"--theme", "light",
				"--show-packages=false",
				"--interactive=false",
			},
			expectError: false,
			contains: []string{
				"Coverage report generated successfully!",
				"Theme: light",
			},
			checkFiles: []string{filepath.Join(tempDir, "report2.html")},
			envVars: map[string]string{
				"COVERAGE_AUTO_CREATE_DIRS": "true",
			},
		},
		{
			name: "missing input file",
			args: []string{
				"report",
				"--input", "/nonexistent/file.txt",
				"--output", filepath.Join(tempDir, "report3.html"),
			},
			expectError: true,
			contains: []string{
				"failed to parse coverage file",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for config
			_ = os.Setenv("COVERAGE_AUTO_CREATE_DIRS", "true")
			defer func() { _ = os.Unsetenv("COVERAGE_AUTO_CREATE_DIRS") }()

			// Set test-specific environment variables
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
			// Create a fresh report command for each test
			testReportCmd := &cobra.Command{
				Use:   "report",
				Short: "Generate HTML coverage report",
				Long:  `Generate comprehensive HTML coverage reports with detailed analysis.`,
				RunE:  reportCmd.RunE,
			}
			testReportCmd.Flags().StringP("input", "i", "", "Input coverage file")
			testReportCmd.Flags().StringP("output", "o", "", "Output HTML file")
			testReportCmd.Flags().StringP("theme", "t", "", "Report theme (github-dark, light, github-light)")
			testReportCmd.Flags().String("title", "", "Report title")
			testReportCmd.Flags().Bool("show-packages", true, "Show package breakdown")
			testReportCmd.Flags().Bool("show-files", true, "Show file breakdown")
			testReportCmd.Flags().Bool("show-missing", true, "Show missing lines")
			testReportCmd.Flags().Bool("interactive", true, "Enable interactive features")

			testCmd.AddCommand(testReportCmd)
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

				// Verify HTML content
				content, err := os.ReadFile(filePath) //nolint:gosec // Test file path is controlled
				require.NoError(t, err)
				assert.Contains(t, string(content), "<html", "File should contain HTML content")
				assert.Contains(t, string(content), "Coverage Report", "File should contain report title")
			}
		})
	}
}

func TestHistoryCommand(t *testing.T) { //nolint:revive // function naming
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
	addCmd.SetArgs([]string{"history", "--add", coverageFile, "--branch", "main", "--commit", "abc123"})
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
				"--branch", "main",
				"--commit", "abc123",
			},
			expectError: false,
			contains: []string{
				"Coverage recorded successfully!",
				"Branch: main",
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
				"--branch", "main",
				"--days", "30",
			},
			expectError: false,
			contains: []string{
				"Coverage Trend Analysis",
				"Branch: main",
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
				"--branch", "main",
			},
			expectError: false,
			contains: []string{
				"Latest Coverage Entry",
				"Branch: main",
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

func TestCommentCommand(t *testing.T) { //nolint:revive // function naming
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
			},
			expectError: false,
			contains: []string{
				"Dry run mode",
				"would post the following comment",
				"Coverage Report",
				"PR: 123",
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
			// Create a fresh comment command for each test
			testCommentCmd := &cobra.Command{
				Use:   "comment",
				Short: "Create PR coverage comment",
				Long:  `Create or update pull request comments with coverage information.`,
				RunE:  commentCmd.RunE,
			}
			testCommentCmd.Flags().IntP("pr", "p", 0, "Pull request number")
			testCommentCmd.Flags().StringP("coverage", "c", "", "Coverage file path")
			testCommentCmd.Flags().String("badge-url", "", "Badge URL")
			testCommentCmd.Flags().String("report-url", "", "Report URL")
			testCommentCmd.Flags().Bool("status", false, "Create commit status")
			testCommentCmd.Flags().Bool("dry-run", false, "Show what would be posted without posting")

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

func TestCompleteCommand(t *testing.T) { //nolint:revive // function naming
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
				"Step 4: Updating coverage history",
				"Step 5: GitHub integration (skipped)",
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
				filepath.Join(outputDir, "coverage.svg"),
				filepath.Join(outputDir, "coverage.html"),
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
				"Step 5: GitHub integration",
				"Would post PR comment",
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

func TestRootCommandHelp(t *testing.T) { //nolint:revive // function naming
	// Disable GitHub integration for tests
	_ = os.Setenv("COVERAGE_POST_COMMENTS", "false")
	_ = os.Setenv("COVERAGE_CREATE_STATUSES", "false")
	defer func() { _ = os.Unsetenv("COVERAGE_POST_COMMENTS") }()
	defer func() { _ = os.Unsetenv("COVERAGE_CREATE_STATUSES") }()

	// Capture output
	var buf bytes.Buffer

	// Create a new root command
	testCmd := &cobra.Command{Use: "test"}
	testCmd.AddCommand(parseCmd, badgeCmd, reportCmd, historyCmd, commentCmd, completeCmd)
	testCmd.SetOut(&buf)
	testCmd.SetErr(&buf)
	testCmd.SetArgs([]string{"--help"})

	// Execute command
	err := testCmd.Execute()
	require.NoError(t, err)

	// Check output contains expected commands
	output := buf.String()
	expectedCommands := []string{"parse", "badge", "report", "history", "comment", "complete"}
	for _, cmd := range expectedCommands {
		assert.Contains(t, output, cmd, "Help should contain command: %s", cmd)
	}
}

func TestCommandFlags(t *testing.T) { //nolint:revive // function naming
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
		{
			name:     "parse command flags",
			cmd:      parseCmd,
			helpArgs: []string{"parse", "--help"},
			contains: []string{"--file", "--output", "--format", "--exclude-tests", "--threshold"},
		},
		{
			name:     "badge command flags",
			cmd:      badgeCmd,
			helpArgs: []string{"badge", "--help"},
			contains: []string{"--coverage", "--style", "--output", "--input", "--label", "--logo"},
		},
		{
			name:     "report command flags",
			cmd:      reportCmd,
			helpArgs: []string{"report", "--help"},
			contains: []string{"--input", "--output", "--theme", "--title", "--show-packages"},
		},
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
func clearTestEnv() { //nolint:revive // function naming
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

func TestMain(m *testing.M) { //nolint:revive // function naming
	// Setup
	clearTestEnv()

	// Run tests
	code := m.Run()

	// Cleanup
	clearTestEnv()

	os.Exit(code)
}
