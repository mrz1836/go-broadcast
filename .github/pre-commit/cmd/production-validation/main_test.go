package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/validation"
)

var (
	errValidatorTest  = errors.New("validator error")
	errReportTest     = errors.New("report error")
	errNotImplemented = errors.New("not implemented") //nolint:unused // Placeholder for future use
)

// Test main function with various command line arguments
func TestMain_CommandLineArgs(t *testing.T) {
	// Build the binary for testing
	ctx := context.Background()
	binaryPath := filepath.Join(t.TempDir(), "production-validation-test")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".") // #nosec G204 - test code with trusted input
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	tests := []struct {
		name           string
		args           []string
		expectSuccess  bool
		expectInOutput []string
	}{
		{
			name:           "default text format",
			args:           []string{},
			expectSuccess:  false, // System won't be production ready in test env
			expectInOutput: []string{"Production Readiness Report"},
		},
		{
			name:           "json format",
			args:           []string{"-format", "json"},
			expectSuccess:  false,
			expectInOutput: []string{`"overall_score"`, `"production_ready"`},
		},
		{
			name:           "verbose mode",
			args:           []string{"-verbose"},
			expectSuccess:  false,
			expectInOutput: []string{"Starting GoFortress", "Validation completed"},
		},
		{
			name:           "output to file",
			args:           []string{"-output", filepath.Join(t.TempDir(), "report.txt")},
			expectSuccess:  false,
			expectInOutput: []string{}, // Output goes to file
		},
		{
			name:           "json format with verbose",
			args:           []string{"-format", "json", "-verbose"},
			expectSuccess:  false,
			expectInOutput: []string{`"overall_score"`, "Starting GoFortress"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.CommandContext(ctx, binaryPath, tt.args...) // #nosec G204 - test code with controlled input
			output, err := cmd.CombinedOutput()

			if tt.expectSuccess {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			outputStr := string(output)
			for _, expected := range tt.expectInOutput {
				assert.Contains(t, outputStr, expected)
			}
		})
	}
}

// Test invalid format handling
func TestMain_InvalidFormat(t *testing.T) {
	// Save original values
	oldArgs := os.Args
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stderr = oldStderr
		log.SetOutput(os.Stderr)
	}()

	// Capture stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	log.SetOutput(w)

	// Set args with invalid format
	os.Args = []string{"production-validation", "-format", "invalid"}

	// Capture log.Fatal
	fatalCalled := false
	oldLogFatalf := logFatalf
	logFatalf = func(format string, v ...interface{}) {
		fatalCalled = true
		assert.Contains(t, fmt.Sprintf(format, v...), "Unsupported output format")
		panic("log.Fatal called")
	}
	defer func() {
		logFatalf = oldLogFatalf
	}()

	// Run main and expect panic from log.Fatal
	assert.Panics(t, func() {
		main()
	})

	assert.True(t, fatalCalled)

	if err := w.Close(); err != nil {
		t.Logf("Failed to close pipe writer: %v", err)
	}
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Logf("Failed to copy output: %v", err)
	}
}

// Test output file creation
func TestMain_OutputFileCreation(t *testing.T) {
	// Create a temp directory for output
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "subdir", "report.json")

	// Save original values
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	}()

	// Create new flag set to avoid conflicts
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Set args to output JSON to file
	os.Args = []string{"production-validation", "-format", "json", "-output", outputPath}

	// Mock the validator to return a simple report
	oldNewValidator := newProductionReadinessValidator
	newProductionReadinessValidator = func() (*validation.ProductionReadinessValidator, error) {
		return &validation.ProductionReadinessValidator{}, nil
	}
	defer func() {
		newProductionReadinessValidator = oldNewValidator
	}()

	// Mock GenerateReport
	oldGenerateReport := generateReport
	generateReport = func(_ *validation.ProductionReadinessValidator) (*validation.ProductionReadinessReport, error) {
		return &validation.ProductionReadinessReport{
			OverallScore:    75,
			ProductionReady: true,
		}, nil
	}
	defer func() {
		generateReport = oldGenerateReport
	}()

	// Run main
	exitCode := runMainWithExitCode()

	// Check file was created
	assert.FileExists(t, outputPath)

	// Verify JSON content
	content, err := os.ReadFile(outputPath) // #nosec G304 - test file path is controlled
	require.NoError(t, err)

	var report validation.ProductionReadinessReport
	err = json.Unmarshal(content, &report)
	require.NoError(t, err)
	assert.Equal(t, 75, report.OverallScore)

	// Should exit with 0 since ProductionReady is true
	assert.Equal(t, 0, exitCode)
}

// Test error handling
func TestMain_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func()
		expectedError string
	}{
		{
			name: "validator creation error",
			setupMock: func() {
				newProductionReadinessValidator = func() (*validation.ProductionReadinessValidator, error) {
					return nil, errValidatorTest
				}
			},
			expectedError: "Failed to create validator",
		},
		{
			name: "report generation error",
			setupMock: func() {
				newProductionReadinessValidator = func() (*validation.ProductionReadinessValidator, error) {
					return &validation.ProductionReadinessValidator{}, nil
				}
				generateReport = func(_ *validation.ProductionReadinessValidator) (*validation.ProductionReadinessReport, error) {
					return nil, errReportTest
				}
			},
			expectedError: "Failed to generate report",
		},
		{
			name: "output write error",
			setupMock: func() {
				newProductionReadinessValidator = func() (*validation.ProductionReadinessValidator, error) {
					return &validation.ProductionReadinessValidator{}, nil
				}
				generateReport = func(_ *validation.ProductionReadinessValidator) (*validation.ProductionReadinessReport, error) {
					return &validation.ProductionReadinessReport{}, nil
				}
				// Set output to invalid path
				os.Args = []string{"production-validation", "-output", "/nonexistent/path/file.txt"}
			},
			expectedError: "Failed to create output directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			oldArgs := os.Args
			oldCommandLine := flag.CommandLine
			oldNewValidator := newProductionReadinessValidator
			oldGenerateReport := generateReport
			defer func() {
				os.Args = oldArgs
				flag.CommandLine = oldCommandLine
				newProductionReadinessValidator = oldNewValidator
				generateReport = oldGenerateReport
			}()

			// Create new flag set to avoid conflicts
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Reset args
			os.Args = []string{"production-validation"}

			// Setup mock
			tt.setupMock()

			// Capture log.Fatal
			fatalCalled := false
			var fatalMessage string
			oldLogFatalf := logFatalf
			logFatalf = func(format string, v ...interface{}) {
				fatalCalled = true
				fatalMessage = fmt.Sprintf(format, v...)
				panic("log.Fatal called")
			}
			defer func() {
				logFatalf = oldLogFatalf
			}()

			// Run main and expect panic
			assert.Panics(t, func() {
				main()
			})

			assert.True(t, fatalCalled)
			assert.Contains(t, fatalMessage, tt.expectedError)
		})
	}
}

// Test exit codes
func TestMain_ExitCodes(t *testing.T) {
	tests := []struct {
		name             string
		productionReady  bool
		expectedExitCode int
	}{
		{
			name:             "production ready",
			productionReady:  true,
			expectedExitCode: 0,
		},
		{
			name:             "not production ready",
			productionReady:  false,
			expectedExitCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			oldArgs := os.Args
			oldCommandLine := flag.CommandLine
			oldNewValidator := newProductionReadinessValidator
			oldGenerateReport := generateReport
			oldOsExit := osExit
			defer func() {
				os.Args = oldArgs
				flag.CommandLine = oldCommandLine
				newProductionReadinessValidator = oldNewValidator
				generateReport = oldGenerateReport
				osExit = oldOsExit
			}()

			// Create new flag set to avoid conflicts
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Set args
			os.Args = []string{"production-validation"}

			// Mock validator and report
			newProductionReadinessValidator = func() (*validation.ProductionReadinessValidator, error) {
				return &validation.ProductionReadinessValidator{}, nil
			}
			generateReport = func(_ *validation.ProductionReadinessValidator) (*validation.ProductionReadinessReport, error) {
				return &validation.ProductionReadinessReport{
					ProductionReady: tt.productionReady,
					OverallScore:    80,
				}, nil
			}

			// Mock osExit to capture exit codes
			exitCode := 0
			osExit = func(code int) {
				exitCode = code
				panic(code)
			}

			// Run main and expect panic if exit code is non-zero
			if tt.expectedExitCode != 0 {
				assert.Panics(t, main)
			} else {
				assert.NotPanics(t, main)
			}

			assert.Equal(t, tt.expectedExitCode, exitCode)
		})
	}
}

// Test flag parsing
func TestMain_FlagParsing(t *testing.T) {
	// Save original command line flags
	oldCommandLine := flag.CommandLine
	defer func() {
		flag.CommandLine = oldCommandLine
	}()

	// Create new flag set for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	tests := []struct {
		name          string
		args          []string
		expectFormat  string
		expectOutput  string
		expectVerbose bool
	}{
		{
			name:          "default values",
			args:          []string{"production-validation"},
			expectFormat:  "text",
			expectOutput:  "",
			expectVerbose: false,
		},
		{
			name:          "all flags set",
			args:          []string{"production-validation", "-format", "json", "-output", "report.json", "-verbose"},
			expectFormat:  "json",
			expectOutput:  "report.json",
			expectVerbose: true,
		},
		{
			name:          "short form verbose",
			args:          []string{"production-validation", "-verbose=true"},
			expectFormat:  "text",
			expectOutput:  "",
			expectVerbose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Set args
			os.Args = tt.args

			// Parse flags as main would
			var (
				outputFormat = flag.String("format", "text", "Output format: text, json")
				outputFile   = flag.String("output", "", "Output file (default: stdout)")
				verbose      = flag.Bool("verbose", false, "Enable verbose output")
			)
			flag.Parse()

			assert.Equal(t, tt.expectFormat, *outputFormat)
			assert.Equal(t, tt.expectOutput, *outputFile)
			assert.Equal(t, tt.expectVerbose, *verbose)
		})
	}
}

// Helper functions to make main testable - these need to be global for testing
// These variables are now defined in main.go for testability

// runMainWithExitCode runs main and returns the exit code
func runMainWithExitCode() int {
	exitCode := 0
	oldExit := osExit
	osExit = func(code int) {
		exitCode = code
		panic(code)
	}
	defer func() {
		osExit = oldExit
		_ = recover() // Expected panic from mocked exit
	}()

	main()
	return exitCode
}

// osExit is mocked for testing - now defined in main.go

// Benchmark main execution
func BenchmarkMain(b *testing.B) {
	// Save original values
	oldArgs := os.Args
	oldStdout := os.Stdout
	oldNewValidator := newProductionReadinessValidator
	oldGenerateReport := generateReport
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		newProductionReadinessValidator = oldNewValidator
		generateReport = oldGenerateReport
	}()

	// Discard output
	os.Stdout, _ = os.Open(os.DevNull)

	// Mock fast validator
	newProductionReadinessValidator = func() (*validation.ProductionReadinessValidator, error) {
		return &validation.ProductionReadinessValidator{}, nil
	}
	generateReport = func(_ *validation.ProductionReadinessValidator) (*validation.ProductionReadinessReport, error) {
		return &validation.ProductionReadinessReport{
			OverallScore:    80,
			ProductionReady: true,
		}, nil
	}

	// Set args
	os.Args = []string{"production-validation"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runMainWithExitCode()
	}
}

// Test verbose output
func TestMain_VerboseOutput(t *testing.T) {
	// Build test binary
	ctx := context.Background()
	binaryPath := filepath.Join(t.TempDir(), "production-validation-verbose")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".") // #nosec G204 - test code with trusted input
	err := buildCmd.Run()
	require.NoError(t, err)

	// Run with verbose flag
	cmd := exec.CommandContext(ctx, binaryPath, "-verbose") // #nosec G204 - test code with controlled input
	output, _ := cmd.CombinedOutput()
	outputStr := string(output)

	// Check verbose messages
	assert.Contains(t, outputStr, "Starting GoFortress Pre-commit System")
	assert.Contains(t, outputStr, "Running comprehensive validation tests")
	assert.Contains(t, outputStr, "Validation completed")
	assert.Contains(t, outputStr, "Overall score:")
}

// Example showing production validation usage
func Example_main() {
	// The production-validation tool generates readiness reports
	// for the GoFortress pre-commit system

	// Usage:
	// production-validation [flags]
	//
	// Flags:
	//   -format string   Output format: text, json (default "text")
	//   -output string   Output file (default: stdout)
	//   -verbose         Enable verbose output

	// Examples:
	// production-validation                          # Generate text report to stdout
	// production-validation -format json             # Generate JSON report
	// production-validation -output report.txt       # Save report to file
	// production-validation -verbose                 # Show detailed progress

	// Exit codes:
	// 0 - System is production ready
	// 1 - System is not production ready or error occurred

	fmt.Println("Production readiness validation tool")
}

// Test main function with real validator (integration test)
func TestMain_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Build and run the actual binary
	ctx := context.Background()
	binaryPath := filepath.Join(t.TempDir(), "production-validation-integration")
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".") // #nosec G204 - test code with trusted input
	err := buildCmd.Run()
	require.NoError(t, err)

	// Run with JSON output to temp file
	outputPath := filepath.Join(t.TempDir(), "integration-report.json")
	cmd := exec.CommandContext(ctx, binaryPath, "-format", "json", "-output", outputPath) // #nosec G204 - test code with controlled input
	_ = cmd.Run()                                                                         // Expected to fail in test environment

	// The command will likely fail (exit 1) because the system won't be production ready
	// in test environment, but the report should still be generated
	assert.FileExists(t, outputPath)

	// Verify JSON is valid
	content, err := os.ReadFile(outputPath) // #nosec G304 - test file path is controlled
	require.NoError(t, err)

	var report validation.ProductionReadinessReport
	err = json.Unmarshal(content, &report)
	require.NoError(t, err)

	// Verify report has expected fields
	assert.True(t, report.OverallScore >= 0 && report.OverallScore <= 100)
	assert.NotNil(t, report.SystemInfo)
}
