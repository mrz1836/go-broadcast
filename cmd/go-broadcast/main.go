// Package main is the entry point for the go-broadcast CLI tool.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/mrz1836/go-broadcast/internal/cli"
	"github.com/mrz1836/go-broadcast/internal/env"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// errPanicRecovered is returned when a panic is recovered during application execution.
var errPanicRecovered = errors.New("panic recovered")

func main() {
	app := NewApp()
	if err := app.Run(os.Args[1:]); err != nil {
		// Error already displayed by outputHandler in Run() or by cli.Execute()
		// Just exit with error code
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

// NewAppWithDependencies creates a new App instance with injectable dependencies.
// Panics if either dependency is nil to fail fast during initialization.
func NewAppWithDependencies(outputHandler OutputHandler, cliExecutor CLIExecutor) *App {
	if outputHandler == nil {
		panic("outputHandler must not be nil")
	}
	if cliExecutor == nil {
		panic("cliExecutor must not be nil")
	}
	return &App{
		outputHandler: outputHandler,
		cliExecutor:   cliExecutor,
	}
}

// Run executes the application with the given arguments.
// The args parameter is accepted for API consistency but is currently unused
// because cobra reads directly from os.Args. This allows future flexibility
// to pass custom args if needed.
func (a *App) Run(_ []string) (err error) {
	// Handle panics gracefully - must be first to catch all panics including Init/LoadEnvFiles
	defer func() {
		if r := recover(); r != nil {
			// Log error with stack trace for debugging
			a.outputHandler.Error(fmt.Sprintf("Fatal error: %v\n%s", r, debug.Stack()))
			// Return error so main() exits with non-zero code
			err = fmt.Errorf("%w: %v", errPanicRecovered, r)
		}
	}()

	// Initialize colored output
	a.outputHandler.Init()

	// Load environment configuration files (.env.base and .env.custom)
	// This follows the GoFortress pattern used by other tools in the ecosystem
	if envErr := env.LoadEnvFiles(); envErr != nil {
		// Don't fail hard on env file loading errors, but warn the user
		// This allows go-broadcast to work without env files if needed
		a.outputHandler.Error(fmt.Sprintf("Warning: Failed to load environment files: %v", envErr))
	}

	// Execute CLI
	err = a.cliExecutor.Execute()
	if err != nil {
		// Display error to user
		a.outputHandler.Error(err.Error())
	}
	return err
}
