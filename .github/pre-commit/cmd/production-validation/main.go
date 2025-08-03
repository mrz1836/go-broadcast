// Package main provides a CLI tool for generating production readiness validation reports
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/validation"
)

func main() {
	var (
		outputFormat = flag.String("format", "text", "Output format: text, json")
		outputFile   = flag.String("output", "", "Output file (default: stdout)")
		verbose      = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *verbose {
		log.Println("Starting GoFortress Pre-commit System production readiness validation...")
	}

	// Create validator
	validator, err := validation.NewProductionReadinessValidator()
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}
	defer validator.Cleanup()

	if *verbose {
		log.Println("Running comprehensive validation tests...")
	}

	// Generate report
	report, err := validator.GenerateReport()
	if err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	if *verbose {
		log.Printf("Validation completed. Overall score: %d/100", report.OverallScore)
	}

	// Format output
	var output string
	switch *outputFormat {
	case "json":
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		output = string(jsonData)
	case "text":
		output = report.FormatReport()
	default:
		log.Fatalf("Unsupported output format: %s", *outputFormat)
	}

	// Write output
	if *outputFile != "" {
		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(*outputFile), 0o755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}

		if err := os.WriteFile(*outputFile, []byte(output), 0o644); err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}

		if *verbose {
			log.Printf("Report written to: %s", *outputFile)
		}
	} else {
		fmt.Print(output)
	}

	// Exit with appropriate code
	if !report.ProductionReady {
		if *verbose {
			log.Println("System is NOT production ready")
		}
		os.Exit(1)
	}

	if *verbose {
		log.Println("System is production ready!")
	}
}
