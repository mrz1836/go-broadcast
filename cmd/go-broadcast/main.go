// Package main is the entry point for the go-broadcast CLI tool.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mrz1836/go-broadcast/internal/cli"
	"github.com/mrz1836/go-broadcast/internal/env"
	"github.com/mrz1836/go-broadcast/internal/output"
)

func main() {
	app := NewApp()
	if err := app.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}

// App represents the main application with testable components
type App struct {
	outputHandler OutputHandler
	cliExecutor   CLIExecutor
}

// OutputHandler defines interface for output operations
type OutputHandler interface {
	Init()
	Error(msg string)
}

// CLIExecutor defines interface for CLI execution
type CLIExecutor interface {
	Execute() error
}

// DefaultOutputHandler implements OutputHandler using the output package
type DefaultOutputHandler struct{}

func (d *DefaultOutputHandler) Init() {
	output.Init()
}

func (d *DefaultOutputHandler) Error(msg string) {
	output.Error(msg)
}

// DefaultCLIExecutor implements CLIExecutor using the cli package
type DefaultCLIExecutor struct{}

func (d *DefaultCLIExecutor) Execute() error {
	return cli.ExecuteWithContext(context.Background())
}

// NewApp creates a new App instance with default implementations
func NewApp() *App {
	return &App{
		outputHandler: &DefaultOutputHandler{},
		cliExecutor:   &DefaultCLIExecutor{},
	}
}

// NewAppWithDependencies creates a new App instance with injectable dependencies
func NewAppWithDependencies(outputHandler OutputHandler, cliExecutor CLIExecutor) *App {
	return &App{
		outputHandler: outputHandler,
		cliExecutor:   cliExecutor,
	}
}

// Run executes the application with the given arguments
func (a *App) Run(_ []string) error {
	// Initialize colored output
	a.outputHandler.Init()

	// Load environment configuration files (.env.base and .env.custom)
	// This follows the GoFortress pattern used by other tools in the ecosystem
	if err := env.LoadEnvFiles(); err != nil {
		// Don't fail hard on env file loading errors, but warn the user
		// This allows go-broadcast to work without env files if needed
		a.outputHandler.Error(fmt.Sprintf("Warning: Failed to load environment files: %v", err))
	}

	// Handle panics gracefully
	defer func() {
		if r := recover(); r != nil {
			a.outputHandler.Error(fmt.Sprintf("Fatal error: %v", r))
		}
	}()

	// Execute CLI
	return a.cliExecutor.Execute()
}
