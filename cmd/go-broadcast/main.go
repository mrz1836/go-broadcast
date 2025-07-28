// Package main is the entry point for the go-broadcast CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/mrz1836/go-broadcast/internal/cli"
	"github.com/mrz1836/go-broadcast/internal/output"
)

func main() {
	// Initialize colored output
	output.Init()

	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			output.Error(fmt.Sprintf("Fatal error: %v", r))
			os.Exit(1)
		}
	}()

	// Execute CLI
	cli.Execute()
}
