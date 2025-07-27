package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/.github/coverage/internal/badge"
	"github.com/mrz1836/go-broadcast/.github/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/.github/coverage/internal/parser"
)

var badgeCmd = &cobra.Command{
	Use:   "badge",
	Short: "Generate coverage badge",
	Long:  `Generate SVG coverage badges for README files and GitHub Pages.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		coverage, _ := cmd.Flags().GetFloat64("coverage")
		style, _ := cmd.Flags().GetString("style")
		outputFile, _ := cmd.Flags().GetString("output")
		inputFile, _ := cmd.Flags().GetString("input")
		label, _ := cmd.Flags().GetString("label")
		logo, _ := cmd.Flags().GetString("logo")
		logoColor, _ := cmd.Flags().GetString("logo-color")

		// Load configuration
		cfg := config.Load()

		// If no coverage percentage provided, try to parse from input file
		if coverage == 0 && inputFile != "" {
			p := parser.New()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			coverageData, err := p.ParseFile(ctx, inputFile)
			if err != nil {
				return fmt.Errorf("failed to parse coverage file: %w", err)
			}
			coverage = coverageData.Percentage
		}

		if coverage == 0 {
			return fmt.Errorf("coverage percentage is required (use --coverage or --input)")
		}

		// Validate coverage percentage
		if coverage < 0 || coverage > 100 {
			return fmt.Errorf("coverage percentage must be between 0 and 100, got %.2f", coverage)
		}

		// Set defaults from config
		if style == "" {
			style = cfg.Badge.Style
		}
		if label == "" {
			label = cfg.Badge.Label
		}
		if logo == "" {
			logo = cfg.Badge.Logo
		}
		if logoColor == "" {
			logoColor = cfg.Badge.LogoColor
		}
		if outputFile == "" {
			outputFile = cfg.Badge.OutputFile
		}

		// Create badge generator with options
		var options []badge.Option
		if label != "coverage" {
			options = append(options, badge.WithLabel(label))
		}
		if style != "flat" {
			options = append(options, badge.WithStyle(style))
		}
		if logo != "" {
			options = append(options, badge.WithLogo(logo))
		}
		if logoColor != "white" {
			options = append(options, badge.WithLogoColor(logoColor))
		}

		generator := badge.New(options...)

		// Generate badge
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		svgContent, err := generator.Generate(ctx, coverage)
		if err != nil {
			return fmt.Errorf("failed to generate badge: %w", err)
		}

		// Create output directory if needed
		if cfg.Storage.AutoCreate && outputFile != "" {
			dir := filepath.Dir(outputFile)
			if err := os.MkdirAll(dir, cfg.Storage.DirMode); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		// Write badge to file
		if err := os.WriteFile(outputFile, []byte(svgContent), cfg.Storage.FileMode); err != nil {
			return fmt.Errorf("failed to write badge file: %w", err)
		}

		// Print success message
		fmt.Printf("Coverage badge generated successfully!\n")
		fmt.Printf("Coverage: %.2f%%\n", coverage)
		fmt.Printf("Style: %s\n", style)
		fmt.Printf("Output: %s\n", outputFile)
		
		// Show color based on coverage
		var status string
		switch {
		case coverage >= 90:
			status = "🟢 Excellent"
		case coverage >= 80:
			status = "🟡 Good"
		case coverage >= 70:
			status = "🟠 Fair"
		default:
			status = "🔴 Needs Improvement"
		}
		fmt.Printf("Status: %s\n", status)

		return nil
	},
}

func init() {
	badgeCmd.Flags().Float64P("coverage", "c", 0, "Coverage percentage (0-100)")
	badgeCmd.Flags().StringP("style", "s", "", "Badge style (flat, flat-square, for-the-badge)")
	badgeCmd.Flags().StringP("output", "o", "", "Output SVG file")
	badgeCmd.Flags().StringP("input", "i", "", "Input coverage file to parse percentage from")
	badgeCmd.Flags().String("label", "", "Badge label text")
	badgeCmd.Flags().String("logo", "", "Logo URL or name")
	badgeCmd.Flags().String("logo-color", "", "Logo color")
}