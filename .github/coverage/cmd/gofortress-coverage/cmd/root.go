// Package cmd provides the command-line interface for the GoFortress coverage tool
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "gofortress-coverage",
	Short: "Go-native coverage system for GoFortress CI/CD",
	Long: `GoFortress Coverage is a self-contained, Go-native coverage system that provides
professional coverage tracking, badge generation, and reporting while maintaining
the simplicity and performance that Go developers expect.

Built as a bolt-on solution completely encapsulated within the .github folder,
this tool replaces Codecov with zero external service dependencies.`,
	Version: "1.0.0",
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() { //nolint:gochecknoinits // CLI command initialization
	// Global flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "text", "Log format (text, json, pretty)")

	// Add subcommands
	rootCmd.AddCommand(completeCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(commentCmd)
	rootCmd.AddCommand(pagesCmd)
}
