// Package main provides a CLI tool for generating production readiness validation reports
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/validation"
)

// Dependencies that can be mocked for testing
type dependencies struct {
	newProductionReadinessValidator func() (*validation.ProductionReadinessValidator, error)
	generateReport                  func(v *validation.ProductionReadinessValidator) (*validation.ProductionReadinessReport, error)
	logFatalf                       func(format string, v ...interface{})
	osExit                          func(code int)
}

// getDependencies returns the default dependencies
func getDependencies() dependencies {
	return dependencies{
		newProductionReadinessValidator: validation.NewProductionReadinessValidator,
		generateReport: func(v *validation.ProductionReadinessValidator) (*validation.ProductionReadinessReport, error) {
			return v.GenerateReport()
		},
		logFatalf: log.Fatalf,
		osExit:    os.Exit,
	}
}

func main() {
	mainWithDeps(getDependencies())
}

func mainWithDeps(deps dependencies) {
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
	validator, err := deps.newProductionReadinessValidator()
	if err != nil {
		deps.logFatalf("Failed to create validator: %v", err)
	}
	defer validator.Cleanup()

	if *verbose {
		log.Println("Running comprehensive validation tests...")
	}

	// Generate report
	report, err := deps.generateReport(validator)
	if err != nil {
		deps.logFatalf("Failed to generate report: %v", err)
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
			deps.logFatalf("Failed to marshal JSON: %v", err)
		}
		output = string(jsonData)
	case "text":
		output = report.FormatReport()
	default:
		deps.logFatalf("Unsupported output format: %s", *outputFormat)
	}

	// Write output
	if *outputFile != "" {
		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(*outputFile), 0o750); err != nil {
			deps.logFatalf("Failed to create output directory: %v", err)
		}

		if err := os.WriteFile(*outputFile, []byte(output), 0o600); err != nil {
			deps.logFatalf("Failed to write output file: %v", err)
		}

		if *verbose {
			log.Printf("Report written to: %s", *outputFile)
		}
	} else {
		if _, err := os.Stdout.WriteString(output); err != nil {
			deps.logFatalf("Failed to write output: %v", err)
		}
	}

	// Exit with appropriate code
	if !report.ProductionReady {
		if *verbose {
			log.Println("System is NOT production ready")
		}
		deps.osExit(1)
	}

	if *verbose {
		log.Println("System is production ready!")
	}
}
