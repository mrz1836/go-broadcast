package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// ErrUnsupportedOutputFormat indicates the specified output format is not supported
var ErrUnsupportedOutputFormat = errors.New("unsupported output format")

var parseCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "parse",
	Short: "Parse Go coverage data",
	Long:  `Parse Go coverage profile data and convert it to structured format for processing.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Get flags
		inputFile, _ := cmd.Flags().GetString("file")
		outputFile, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")
		excludeTests, _ := cmd.Flags().GetBool("exclude-tests")
		excludeGenerated, _ := cmd.Flags().GetBool("exclude-generated")
		threshold, _ := cmd.Flags().GetFloat64("threshold")

		// Load configuration
		cfg := config.Load()

		// Override with command line flags
		if inputFile != "" {
			cfg.Coverage.InputFile = inputFile
		}
		if excludeTests {
			cfg.Coverage.ExcludeTests = true
		}
		if excludeGenerated {
			cfg.Coverage.ExcludeGenerated = true
		}
		if threshold > 0 {
			cfg.Coverage.Threshold = threshold
		}

		// Validate configuration
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		// Create parser with config
		parserConfig := &parser.Config{
			ExcludePaths:     cfg.Coverage.ExcludePaths,
			ExcludeFiles:     cfg.Coverage.ExcludeFiles,
			ExcludeGenerated: cfg.Coverage.ExcludeGenerated,
		}
		p := parser.NewWithConfig(parserConfig)

		// Parse coverage data
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		coverage, err := p.ParseFile(ctx, cfg.Coverage.InputFile)
		if err != nil {
			return fmt.Errorf("failed to parse coverage file: %w", err)
		}

		// Print results
		cmd.Printf("Coverage Analysis Results:\n")
		cmd.Printf("==========================\n")
		cmd.Printf("Overall Coverage: %.2f%% (%d/%d lines)\n",
			coverage.Percentage, coverage.CoveredLines, coverage.TotalLines)
		cmd.Printf("Mode: %s\n", coverage.Mode)
		cmd.Printf("Packages: %d\n", len(coverage.Packages))
		cmd.Printf("Timestamp: %s\n", coverage.Timestamp.Format(time.RFC3339))

		// Check threshold
		if coverage.Percentage < cfg.Coverage.Threshold {
			cmd.Printf("\n⚠️  Coverage %.2f%% is below threshold %.2f%%\n",
				coverage.Percentage, cfg.Coverage.Threshold)
		} else {
			cmd.Printf("\n✅ Coverage %.2f%% meets threshold %.2f%%\n",
				coverage.Percentage, cfg.Coverage.Threshold)
		}

		// Package breakdown
		if len(coverage.Packages) > 0 {
			cmd.Printf("\nPackage Breakdown:\n")
			cmd.Printf("------------------\n")
			for name, pkg := range coverage.Packages {
				cmd.Printf("  %s: %.2f%% (%d/%d lines)\n",
					name, pkg.Percentage, pkg.CoveredLines, pkg.TotalLines)
			}
		}

		// Save output if requested
		if outputFile != "" {
			var data []byte
			switch format {
			case "json":
				data, err = json.MarshalIndent(coverage, "", "  ")
			default:
				return fmt.Errorf("%w: %s", ErrUnsupportedOutputFormat, format)
			}

			if err != nil {
				return fmt.Errorf("failed to marshal coverage data: %w", err)
			}

			if err := os.WriteFile(outputFile, data, 0600); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}

			cmd.Printf("\nOutput saved to: %s\n", outputFile)
		}

		// Return error if coverage is below threshold
		if coverage.Percentage < cfg.Coverage.Threshold {
			return fmt.Errorf("coverage %.2f%% is below threshold %.2f%%", coverage.Percentage, cfg.Coverage.Threshold) //nolint:err113 // Will refactor to shared errors package
		}

		return nil
	},
}

func init() { //nolint:gochecknoinits // CLI command initialization
	parseCmd.Flags().StringP("file", "f", "", "Coverage profile file (overrides config)")
	parseCmd.Flags().StringP("output", "o", "", "Output file for parsed data")
	parseCmd.Flags().String("format", "json", "Output format (json)")
	parseCmd.Flags().Bool("exclude-tests", false, "Exclude test files from coverage")
	parseCmd.Flags().Bool("exclude-generated", false, "Exclude generated files from coverage")
	parseCmd.Flags().Float64("threshold", 0, "Coverage threshold (0 to use config default)")
	_ = parseCmd.MarkFlagRequired("file")
}
