// Package logging provides logging configuration types and utilities.
//
// This package defines the logging configuration structures used throughout
// the application to enable component-specific debug logging and verbose
// output control. It avoids import cycles by being a leaf dependency.
package logging

import (
	"crypto/rand"
	"encoding/hex"
)

// LogConfig holds all logging and CLI configuration.
//
// This configuration is passed via dependency injection throughout the
// application to avoid global state and enable better testing isolation.
type LogConfig struct {
	ConfigFile    string
	DryRun        bool
	LogLevel      string
	Verbose       int // -v, -vv, -vvv support
	Debug         DebugFlags
	LogFormat     string   // "text" or "json"
	CorrelationID string   // Unique ID for request correlation
	JSONOutput    bool     // Enable JSON structured output
	GroupFilter   []string // Groups to sync (by name or ID)
	SkipGroups    []string // Groups to skip during sync
}

// DebugFlags contains component-specific debug flags for targeted troubleshooting.
//
// Each flag enables detailed logging for a specific component:
// - Git: Git command execution, timing, and output
// - API: GitHub API requests, responses, and timing
// - Transform: File transformation details and variable substitution
// - Config: Configuration loading and validation
// - State: State discovery and management operations
type DebugFlags struct {
	Git       bool // --debug-git flag
	API       bool // --debug-api flag
	Transform bool // --debug-transform flag
	Config    bool // --debug-config flag
	State     bool // --debug-state flag
}

// GenerateCorrelationID creates a unique correlation ID for request tracing.
//
// Returns a 16-byte hex-encoded string that can be used to correlate
// log entries across different components for the same operation.
func GenerateCorrelationID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple timestamp-based ID if crypto/rand fails
		return "fallback-id"
	}
	return hex.EncodeToString(bytes)
}

// WithCorrelationID creates a new LogConfig with the specified correlation ID.
//
// This is useful for creating child contexts that inherit the parent's
// correlation ID for cross-component operation tracing.
func (lc *LogConfig) WithCorrelationID(correlationID string) *LogConfig {
	if lc == nil {
		return &LogConfig{CorrelationID: correlationID}
	}

	newConfig := *lc
	newConfig.CorrelationID = correlationID
	return &newConfig
}
