package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Required by cobra
var (
	// Version information
	version   string
	commit    string
	buildDate string

	// Global flags
	verbose bool
	noColor bool
)

// rootCmd represents the base command
//
//nolint:gochecknoglobals // Required by cobra
var rootCmd = &cobra.Command{
	Use:   "gofortress-pre-commit",
	Short: "GoFortress Pre-commit System - Fast, Go-native git pre-commit checks",
	Long: `GoFortress Pre-commit System is a high-performance, Go-native replacement
for traditional pre-commit frameworks. It provides fast, parallel execution
of code quality checks with zero Python dependencies.

Key features:
  - Lightning fast parallel execution
  - Zero runtime dependencies (single binary)
  - Environment-based configuration via .github/.env.shared
  - Seamless CI/CD integration
  - Native make command compatibility`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// SetVersionInfo sets the version information for the command
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	buildDate = d
}

//nolint:gochecknoinits // Required by cobra
func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	// Version flag
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildDate)
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
`)

	// Add subcommands
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(statusCmd)
}

func initConfig() {
	// Disable color if requested or if not in a terminal
	if noColor || os.Getenv("NO_COLOR") != "" {
		color.NoColor = true
	}

	// Set up paths relative to repository root
	// The binary should be run from the repository root or have access to .github/
	if _, err := os.Stat(".github/.env.shared"); os.IsNotExist(err) {
		// Try to find the repository root
		cwd, _ := os.Getwd()
		for cwd != "/" && cwd != "" {
			if _, err := os.Stat(filepath.Join(cwd, ".github/.env.shared")); err == nil {
				_ = os.Chdir(cwd)
				break
			}
			cwd = filepath.Dir(cwd)
		}
	}
}

// Helper functions for consistent output
func printSuccess(format string, args ...interface{}) {
	if !noColor {
		color.Green("✓ " + fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "✓ %s\n", fmt.Sprintf(format, args...))
	}
}

func printError(format string, args ...interface{}) {
	if !noColor {
		color.Red("✗ " + fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "✗ %s\n", fmt.Sprintf(format, args...))
	}
}

func printInfo(format string, args ...interface{}) {
	if !noColor {
		color.Blue("ℹ " + fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "ℹ %s\n", fmt.Sprintf(format, args...))
	}
}

func printWarning(format string, args ...interface{}) {
	if !noColor {
		color.Yellow("⚠ " + fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "⚠ %s\n", fmt.Sprintf(format, args...))
	}
}
