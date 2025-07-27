package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/.github/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/.github/coverage/internal/parser"
)

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse Go coverage data",
	Long:  `Parse Go coverage profile data and convert it to structured format for processing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Create parser with options
		p := parser.New()
		
		var options []parser.Option
		if cfg.Coverage.ExcludeTests {
			options = append(options, parser.WithExcludePatterns(cfg.Coverage.ExcludeFiles...))
		}
		if len(cfg.Coverage.ExcludePaths) > 0 {
			options = append(options, parser.WithExcludePaths(cfg.Coverage.ExcludePaths...))
		}

		// Parse coverage data
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		coverage, err := p.ParseFile(ctx, cfg.Coverage.InputFile, options...)
		if err != nil {
			return fmt.Errorf("failed to parse coverage file: %w", err)
		}

		// Print results
		fmt.Printf("Coverage Analysis Results:\n")
		fmt.Printf("==========================\n")
		fmt.Printf("Overall Coverage: %.2f%% (%d/%d lines)\n", 
			coverage.Percentage, coverage.CoveredLines, coverage.TotalLines)
		fmt.Printf("Mode: %s\n", coverage.Mode)
		fmt.Printf("Packages: %d\n", len(coverage.Packages))
		fmt.Printf("Timestamp: %s\n", coverage.Timestamp.Format(time.RFC3339))
		
		// Check threshold
		if coverage.Percentage < cfg.Coverage.Threshold {
			fmt.Printf("\n⚠️  Coverage %.2f%% is below threshold %.2f%%\n", 
				coverage.Percentage, cfg.Coverage.Threshold)
		} else {
			fmt.Printf("\n✅ Coverage %.2f%% meets threshold %.2f%%\n", 
				coverage.Percentage, cfg.Coverage.Threshold)
		}

		// Package breakdown
		if len(coverage.Packages) > 0 {
			fmt.Printf("\nPackage Breakdown:\n")
			fmt.Printf("------------------\n")
			for name, pkg := range coverage.Packages {
				fmt.Printf("  %s: %.2f%% (%d/%d lines)\n", 
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
				return fmt.Errorf("unsupported output format: %s", format)
			}
			
			if err != nil {
				return fmt.Errorf("failed to marshal coverage data: %w", err)
			}
			
			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			
			fmt.Printf("\nOutput saved to: %s\n", outputFile)
		}

		// Exit with non-zero if coverage is below threshold
		if coverage.Percentage < cfg.Coverage.Threshold {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	parseCmd.Flags().StringP("file", "f", "", "Coverage profile file (overrides config)")
	parseCmd.Flags().StringP("output", "o", "", "Output file for parsed data")
	parseCmd.Flags().String("format", "json", "Output format (json)")
	parseCmd.Flags().Bool("exclude-tests", false, "Exclude test files from coverage")
	parseCmd.Flags().Bool("exclude-generated", false, "Exclude generated files from coverage")
	parseCmd.Flags().Float64("threshold", 0, "Coverage threshold (0 to use config default)")
	parseCmd.MarkFlagRequired("file")
}