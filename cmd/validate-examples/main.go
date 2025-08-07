// Package main implements an example configuration validator for go-broadcast.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
)

type validationResult struct {
	totalExamples   int
	validExamples   int
	invalidExamples int
}

type exampleConfig struct {
	file        string
	description string
}

func main() {
	redColor := color.New(color.FgRed)
	greenColor := color.New(color.FgGreen)
	yellowColor := color.New(color.FgYellow, color.Bold)
	blueColor := color.New(color.FgBlue)
	boldRed := color.New(color.FgRed, color.Bold)
	boldGreen := color.New(color.FgGreen, color.Bold)
	boldBlue := color.New(color.FgBlue, color.Bold)

	var (
		helpFlag    bool
		verboseFlag bool
	)

	flag.BoolVar(&helpFlag, "h", false, "Show help message")
	flag.BoolVar(&helpFlag, "help", false, "Show help message")
	flag.BoolVar(&verboseFlag, "v", false, "Enable verbose output")
	flag.BoolVar(&verboseFlag, "verbose", false, "Enable verbose output")
	flag.Parse()

	if helpFlag {
		showUsage()
		os.Exit(0)
	}

	if verboseFlag {
		// Enable verbose output by setting debug environment variable
		_ = os.Setenv("DEBUG", "1")
	}

	printHeader(boldBlue)

	// Check if go-broadcast binary exists
	if _, err := os.Stat("./go-broadcast"); os.IsNotExist(err) {
		_, _ = boldRed.Println("Error: go-broadcast binary not found. Please build it first:")
		_, _ = fmt.Fprintln(os.Stdout, "  make build-go")
		os.Exit(1)
	}

	result := &validationResult{}

	// Validate existing file-only examples
	printSection(yellowColor, "File Sync Examples")
	validateExamples(result, []exampleConfig{
		{"examples/minimal.yaml", "Minimal configuration for simple file sync"},
		{"examples/sync.yaml", "Complete example with all features"},
		{"examples/microservices.yaml", "Microservices architecture sync"},
		{"examples/multi-language.yaml", "Multi-language project sync"},
		{"examples/ci-cd-only.yaml", "CI/CD pipeline synchronization"},
		{"examples/documentation.yaml", "Documentation template sync"},
	}, blueColor, boldRed, boldGreen)

	// Validate directory sync examples
	printSection(yellowColor, "Directory Sync Examples")
	validateExamples(result, []exampleConfig{
		{"examples/directory-sync.yaml", "Comprehensive directory sync examples"},
		{"examples/github-workflows.yaml", "GitHub infrastructure sync"},
		{"examples/large-directories.yaml", "Large directory management"},
		{"examples/exclusion-patterns.yaml", "Exclusion pattern showcase"},
		{"examples/github-complete.yaml", "Complete GitHub directory sync"},
	}, blueColor, boldRed, boldGreen)

	// Test documented commands
	printSection(yellowColor, "Command Testing")
	testCommands(blueColor, boldRed, boldGreen)

	// Test dry-run mode
	printSection(yellowColor, "Dry-Run Testing")
	testDryRun(blueColor, yellowColor, boldGreen)

	// Print summary
	printSummary(result, boldBlue, redColor, greenColor, boldGreen, boldRed)
}

func showUsage() {
	_, _ = fmt.Fprintln(os.Stdout, "Usage:", os.Args[0], "[options]")
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "Options:")
	_, _ = fmt.Fprintln(os.Stdout, "  -h, --help     Show this help message")
	_, _ = fmt.Fprintln(os.Stdout, "  -v, --verbose  Enable verbose output")
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "This script validates all example configurations in the examples/ directory")
	_, _ = fmt.Fprintln(os.Stdout, "and tests documented commands to ensure they work correctly.")
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "Prerequisites:")
	_, _ = fmt.Fprintln(os.Stdout, "  - go-broadcast binary must be built (run: make build-go)")
	_, _ = fmt.Fprintln(os.Stdout, "  - All example files must exist in examples/ directory")
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "Examples:")
	_, _ = fmt.Fprintln(os.Stdout, "  "+os.Args[0]+"                    # Validate all examples")
	_, _ = fmt.Fprintln(os.Stdout, "  "+os.Args[0]+" --verbose          # Validate with verbose output")
}

func printHeader(boldBlue *color.Color) {
	_, _ = boldBlue.Println("===============================================")
	_, _ = boldBlue.Println("  go-broadcast Example Configuration Validation")
	_, _ = boldBlue.Println("===============================================")
	_, _ = fmt.Fprintln(os.Stdout)
}

func printSection(yellowColor *color.Color, title string) {
	_, _ = yellowColor.Printf("--- %s ---\n", title)
}

func validateConfig(result *validationResult, configFile, description string, blueColor, boldRed, boldGreen *color.Color) {
	_, _ = blueColor.Printf("Validating: %s\n", configFile)
	_, _ = fmt.Fprintf(os.Stdout, "Description: %s\n", description)

	result.totalExamples++

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "./go-broadcast", "validate", "--config", configFile)
	if err := cmd.Run(); err != nil {
		_, _ = boldRed.Printf("❌ INVALID: %s\n", configFile)
		result.invalidExamples++
	} else {
		_, _ = boldGreen.Printf("✅ VALID: %s\n", configFile)
		result.validExamples++
	}
	_, _ = fmt.Fprintln(os.Stdout)
}

func validateExamples(result *validationResult, configs []exampleConfig, blueColor, boldRed, boldGreen *color.Color) {
	for _, config := range configs {
		validateConfig(result, config.file, config.description, blueColor, boldRed, boldGreen)
	}
}

func testCommand(command, description string, blueColor, boldRed, boldGreen *color.Color) {
	_, _ = blueColor.Printf("Testing: %s\n", command)
	_, _ = fmt.Fprintf(os.Stdout, "Description: %s\n", description)

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		_, _ = boldRed.Printf("❌ COMMAND FAILED: %s\n", command)
	} else {
		_, _ = boldGreen.Printf("✅ COMMAND WORKS: %s\n", command)
	}
	_, _ = fmt.Fprintln(os.Stdout)
}

func testCommands(blueColor, boldRed, boldGreen *color.Color) {
	commands := []struct {
		cmd  string
		desc string
	}{
		{"./go-broadcast --version", "Version command"},
		{"./go-broadcast --help", "Help command"},
		{"./go-broadcast validate --help", "Validate help command"},
		{"./go-broadcast sync --help", "Sync help command"},
		{"./go-broadcast status --help", "Status help command"},
		{"./go-broadcast diagnose --help", "Diagnose help command"},
		{"./go-broadcast cancel --help", "Cancel help command"},
	}

	for _, c := range commands {
		testCommand(c.cmd, c.desc, blueColor, boldRed, boldGreen)
	}
}

func testDryRun(blueColor, yellowColor, boldGreen *color.Color) {
	_, _ = blueColor.Println("Testing dry-run mode with minimal configuration...")

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "./go-broadcast", "sync", "--dry-run", "--config", "examples/minimal.yaml")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		_, _ = yellowColor.Println("⚠️  Dry-run requires valid repository access (expected)")
	} else {
		_, _ = boldGreen.Println("✅ Dry-run mode works correctly")
	}
	_, _ = fmt.Fprintln(os.Stdout)
}

func printSummary(result *validationResult, boldBlue, redColor, greenColor, boldGreen, boldRed *color.Color) {
	_, _ = boldBlue.Println("===============================================")
	_, _ = boldBlue.Println("  VALIDATION SUMMARY")
	_, _ = boldBlue.Println("===============================================")
	_, _ = fmt.Fprintf(os.Stdout, "Total examples tested: %d\n", result.totalExamples)

	if result.invalidExamples > 0 {
		_, _ = redColor.Printf("Invalid configurations: %d\n", result.invalidExamples)
	} else {
		_, _ = greenColor.Printf("Invalid configurations: %d\n", result.invalidExamples)
	}

	_, _ = greenColor.Printf("Valid configurations: %d\n", result.validExamples)
	_, _ = fmt.Fprintln(os.Stdout)

	if result.invalidExamples == 0 {
		_, _ = boldGreen.Println("🎉 ALL EXAMPLES VALID!")
		os.Exit(0)
	}
	_, _ = boldRed.Println("❌ Some examples failed validation")
	os.Exit(1)
}
