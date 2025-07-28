// Package cli provides command-line interface functionality for go-broadcast.
//
// This package implements the CLI commands, flags, and logging configuration
// used throughout the application. It is designed to provide intuitive
// debugging capabilities and flexible logging output.
//
// Key features include:
// - Verbose flag support (-v, -vv, -vvv) for increasing log detail
// - Component-specific debug flags for targeted troubleshooting
// - Multiple output formats (text and JSON) for different use cases
// - Dependency injection pattern for configuration management
//
// The package follows Go conventions by avoiding global state and using
// dependency injection for all configuration and logging setup.
package cli

import "github.com/mrz1836/go-broadcast/internal/logging"

// LogConfig holds all logging and CLI configuration.
//
// This configuration is passed via dependency injection throughout the
// application to avoid global state and enable better testing isolation.
type LogConfig = logging.LogConfig

// DebugFlags contains component-specific debug flags for targeted troubleshooting.
//
// Each flag enables detailed debugging for a specific component or operation,
// allowing users to focus on particular areas without overwhelming log output.
type DebugFlags = logging.DebugFlags

// Flags contains all global flags for the CLI (legacy support)
type Flags struct {
	ConfigFile string
	DryRun     bool
	LogLevel   string
}

// globalFlags is the singleton instance of flags
//
//nolint:gochecknoglobals // CLI flags need to be accessible across command functions
var globalFlags = &Flags{
	ConfigFile: "sync.yaml",
	LogLevel:   "info",
}

// GetConfigFile returns the config file path
func GetConfigFile() string {
	if globalFlags == nil {
		return "sync.yaml" // Default value
	}
	return globalFlags.ConfigFile
}

// IsDryRun returns whether dry-run mode is enabled
func IsDryRun() bool {
	if globalFlags == nil {
		return false // Default value
	}
	return globalFlags.DryRun
}

// SetFlags updates the global flags
func SetFlags(f *Flags) {
	globalFlags = f
}

// ResetGlobalFlags resets the global flags to their default values
// This is primarily used for testing to ensure clean state between tests
func ResetGlobalFlags() {
	globalFlags.ConfigFile = "sync.yaml"
	globalFlags.DryRun = false
	globalFlags.LogLevel = "info"
}
