package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
	"github.com/mrz1836/go-broadcast/coverage/internal/report"
)

var reportCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "report",
	Short: "Generate coverage report",
	Long:  `Generate interactive HTML coverage reports for GitHub Pages.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		inputFile, _ := cmd.Flags().GetString("input")
		outputFile, _ := cmd.Flags().GetString("output")
		theme, _ := cmd.Flags().GetString("theme")
		title, _ := cmd.Flags().GetString("title")
		showPackages, _ := cmd.Flags().GetBool("show-packages")
		showFiles, _ := cmd.Flags().GetBool("show-files")
		showMissing, _ := cmd.Flags().GetBool("show-missing")
		interactive, _ := cmd.Flags().GetBool("interactive")

		// Load configuration
		cfg := config.Load()

		// Set defaults from config
		if inputFile == "" {
			inputFile = cfg.Coverage.InputFile
		}
		if outputFile == "" {
			outputFile = cfg.Report.OutputFile
		}
		if theme == "" {
			theme = cfg.Report.Theme
		}
		if title == "" {
			title = cfg.Report.Title
		}

		// Parse coverage data
		p := parser.New()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		coverage, err := p.ParseFile(ctx, inputFile)
		if err != nil {
			return fmt.Errorf("failed to parse coverage file: %w", err)
		}

		// Create report generator with options
		reportConfig := &report.Config{
			Title:            title,
			Theme:            theme,
			ShowPackages:     showPackages,
			ShowFiles:        showFiles,
			ShowMissing:      showMissing,
			InteractiveTrees: interactive,
		}
		generator := report.NewWithConfig(reportConfig)

		// Generate report
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		htmlContent, err := generator.Generate(ctx, coverage)
		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}

		// Create output directory if needed
		if cfg.Storage.AutoCreate && outputFile != "" {
			dir := filepath.Dir(outputFile)
			if err := os.MkdirAll(dir, cfg.Storage.DirMode); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// Write report to file
		if err := os.WriteFile(outputFile, htmlContent, cfg.Storage.FileMode); err != nil {
			return fmt.Errorf("failed to write report file: %w", err)
		}

		// Print success message
		cmd.Printf("Coverage report generated successfully!\n")
		cmd.Printf("Title: %s\n", title)
		cmd.Printf("Theme: %s\n", theme)
		cmd.Printf("Output: %s\n", outputFile)
		cmd.Printf("Coverage: %.2f%% (%d/%d lines)\n",
			coverage.Percentage, coverage.CoveredLines, coverage.TotalLines)
		cmd.Printf("Packages: %d\n", len(coverage.Packages))

		if cfg.GitHub.Owner != "" && cfg.GitHub.Repository != "" {
			reportURL := cfg.GetReportURL()
			cmd.Printf("Public URL: %s\n", reportURL)
		}

		// Show coverage analysis
		var status string
		switch {
		case coverage.Percentage >= 90:
			status = "🟢 Excellent coverage!"
		case coverage.Percentage >= 80:
			status = "🟡 Good coverage"
		case coverage.Percentage >= 70:
			status = "🟠 Fair coverage"
		default:
			status = "🔴 Coverage needs improvement"
		}
		cmd.Printf("Status: %s\n", status)

		// Check threshold
		if coverage.Percentage < cfg.Coverage.Threshold {
			cmd.Printf("⚠️  Coverage %.2f%% is below threshold %.2f%%\n",
				coverage.Percentage, cfg.Coverage.Threshold)
		}

		return nil
	},
}

func init() { //nolint:gochecknoinits // CLI command initialization
	reportCmd.Flags().StringP("input", "i", "", "Input coverage file")
	reportCmd.Flags().StringP("output", "o", "", "Output HTML file")
	reportCmd.Flags().StringP("theme", "t", "", "Report theme (github-dark, light, github-light)")
	reportCmd.Flags().String("title", "", "Report title")
	reportCmd.Flags().Bool("show-packages", true, "Show package breakdown")
	reportCmd.Flags().Bool("show-files", true, "Show file breakdown")
	reportCmd.Flags().Bool("show-missing", true, "Show missing lines")
	reportCmd.Flags().Bool("interactive", true, "Enable interactive features")
}
