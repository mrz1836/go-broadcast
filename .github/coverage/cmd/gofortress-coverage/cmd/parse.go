package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

var parseCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "parse",
	Short: "Parse Go coverage profile and display results",
	Long: `Parse a Go coverage profile file and display coverage analysis results.

This command analyzes Go coverage data and can output results in various formats,
check coverage thresholds, and save results to a file.`,
	RunE: runParse,
}

func init() { //nolint:gochecknoinits // CLI command initialization
	parseCmd.Flags().StringP("file", "f", "coverage.txt", "Path to coverage profile file")
	parseCmd.Flags().StringP("output", "o", "", "Output file path (optional)")
	parseCmd.Flags().String("format", "text", "Output format (text or json)")
	parseCmd.Flags().Float64("threshold", 0, "Coverage threshold percentage (0-100)")
}

func runParse(cmd *cobra.Command, _ []string) error {
	// Get flags
	coverageFile, _ := cmd.Flags().GetString("file")
	outputPath, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	threshold, _ := cmd.Flags().GetFloat64("threshold")

	// Parse coverage file
	p := parser.New()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	coverage, err := p.ParseFile(ctx, coverageFile)
	if err != nil {
		return fmt.Errorf("failed to parse coverage file: %w", err)
	}

	// Always display text summary first
	cmd.Println("Coverage Analysis Results")
	cmd.Println("========================")
	cmd.Printf("Overall Coverage: %.2f%%\n", coverage.Percentage)
	cmd.Printf("Mode: %s\n", coverage.Mode)
	cmd.Printf("Total Statements: %d\n", coverage.TotalLines)
	cmd.Printf("Covered Statements: %d\n", coverage.CoveredLines)
	cmd.Printf("Missed Statements: %d\n", coverage.TotalLines-coverage.CoveredLines)
	cmd.Println()

	// Display package information
	cmd.Printf("Packages: %d\n", len(coverage.Packages))
	for pkgName, pkg := range coverage.Packages {
		cmd.Printf("  - %s: %.2f%% (%d/%d statements)\n",
			pkgName,
			pkg.Percentage,
			pkg.CoveredLines,
			pkg.TotalLines)
	}

	// Handle output file based on format
	if outputPath != "" {
		var data []byte
		var err error

		if format == "json" {
			data, err = json.MarshalIndent(coverage, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal coverage data: %w", err)
			}
		} else {
			// For text format, save as JSON anyway since the file needs structured data
			data, err = json.MarshalIndent(coverage, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal coverage data: %w", err)
			}
		}

		if err := os.WriteFile(outputPath, data, 0o600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		cmd.Println()
		cmd.Printf("Output saved to: %s\n", outputPath)
	} else if format == "json" {
		// If no output file but JSON format requested, print JSON to stdout
		data, err := json.MarshalIndent(coverage, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal coverage data: %w", err)
		}
		cmd.Println()
		cmd.Println(string(data))
	}

	// Check threshold if specified
	if threshold > 0 {
		cmd.Println()
		if coverage.Percentage >= threshold {
			cmd.Printf("✅ Coverage %.2f%% meets threshold of %.2f%%\n", coverage.Percentage, threshold)
		} else {
			cmd.Printf("❌ Coverage %.2f%% is below threshold of %.2f%%\n", coverage.Percentage, threshold)
			return ErrCoverageBelowThreshold
		}
	}

	return nil
}
