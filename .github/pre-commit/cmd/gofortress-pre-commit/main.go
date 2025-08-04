// Package main provides the entry point for the GoFortress pre-commit system
package main

import (
	"fmt"
	"os"

	"github.com/mrz1836/go-broadcast/pre-commit/cmd/gofortress-pre-commit/cmd"
)

// version information - set by ldflags during build
var (
	version   = "dev"
	commit    = "none"    //nolint:gochecknoglobals // Build var
	buildDate = "unknown" //nolint:gochecknoglobals // Build var
)

func main() {
	// Set version information for the root command
	cmd.SetVersionInfo(version, commit, buildDate)

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
