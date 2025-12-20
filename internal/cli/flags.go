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

import (
	"sync"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

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
	ConfigFile       string
	DryRun           bool
	LogLevel         string
	GroupFilter      []string // Groups to sync (by name or ID)
	SkipGroups       []string // Groups to skip during sync
	Automerge        bool     // Enable automerge labels on created PRs
	ClearModuleCache bool     // Clear module version cache before sync
}

// globalFlags is the singleton instance of flags
// globalFlagsMu protects concurrent access to globalFlags
//
//nolint:gochecknoglobals // CLI flags need to be accessible across command functions
var (
	globalFlags = &Flags{
		ConfigFile: "sync.yaml",
		LogLevel:   "info",
	}
	globalFlagsMu sync.RWMutex
)

// GetConfigFile returns the config file path (thread-safe)
func GetConfigFile() string {
	globalFlagsMu.RLock()
	defer globalFlagsMu.RUnlock()
	if globalFlags == nil {
		return "sync.yaml" // Default value
	}
	return globalFlags.ConfigFile
}

// IsDryRun returns whether dry-run mode is enabled (thread-safe)
func IsDryRun() bool {
	globalFlagsMu.RLock()
	defer globalFlagsMu.RUnlock()
	if globalFlags == nil {
		return false // Default value
	}
	return globalFlags.DryRun
}

// SetFlags updates the global flags (thread-safe)
func SetFlags(f *Flags) {
	globalFlagsMu.Lock()
	defer globalFlagsMu.Unlock()
	globalFlags = f
}

// ResetGlobalFlags resets the global flags to their default values (thread-safe)
// This is primarily used for testing to ensure clean state between tests
func ResetGlobalFlags() {
	globalFlagsMu.Lock()
	defer globalFlagsMu.Unlock()
	globalFlags.ConfigFile = "sync.yaml"
	globalFlags.DryRun = false
	globalFlags.LogLevel = "info"
}

// GetGlobalFlags returns a copy of the current global flags (thread-safe)
// This is useful for tests that need to save and restore flag state
func GetGlobalFlags() *Flags {
	globalFlagsMu.RLock()
	defer globalFlagsMu.RUnlock()
	if globalFlags == nil {
		return &Flags{ConfigFile: "sync.yaml", LogLevel: "info"}
	}
	// Return a copy to prevent race conditions
	return &Flags{
		ConfigFile:       globalFlags.ConfigFile,
		DryRun:           globalFlags.DryRun,
		LogLevel:         globalFlags.LogLevel,
		GroupFilter:      append([]string(nil), globalFlags.GroupFilter...),
		SkipGroups:       append([]string(nil), globalFlags.SkipGroups...),
		Automerge:        globalFlags.Automerge,
		ClearModuleCache: globalFlags.ClearModuleCache,
	}
}
