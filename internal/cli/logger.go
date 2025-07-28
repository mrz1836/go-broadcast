// Package cli provides logging configuration and trace level support for go-broadcast.
//
// This file implements the LoggerService which provides enhanced logging capabilities
// including trace level support beyond standard logrus levels and component-specific
// debug flag handling.
//
// Key features include:
// - Custom trace level implementation below debug level
// - Context-aware logging configuration based on verbose flags
// - Automatic caller information for deep debugging (-vvv)
// - Component-specific debug flag integration
//
// The LoggerService follows dependency injection pattern and requires context
// to be passed through all operations for proper cancellation support.
package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/sirupsen/logrus"
)

// TraceLevel uses the standard logrus TraceLevel for very detailed logging
const TraceLevel = logrus.TraceLevel

// LoggerService provides logging configuration and trace level support.
//
// This service manages the mapping of verbose flags to appropriate log levels
// and configures logrus with custom hooks for trace level support.
type LoggerService struct {
	config *LogConfig
}

// NewLoggerService creates a new logger service with the given configuration.
//
// Parameters:
// - config: Logging configuration containing verbose and debug settings
//
// Returns:
// - LoggerService instance configured with the provided settings
func NewLoggerService(config *LogConfig) *LoggerService {
	return &LoggerService{
		config: config,
	}
}

// TraceHook implements a custom logrus hook for trace level support.
//
// This hook allows logging at a level below debug by intercepting entries
// at the custom trace level and converting them to debug level entries
// with a [TRACE] prefix for easy identification.
type TraceHook struct {
	Enabled bool
}

// Levels returns the log levels this hook should fire for.
//
// Returns:
// - Slice of log levels, including the custom trace level when enabled
func (h *TraceHook) Levels() []logrus.Level {
	if h.Enabled {
		return []logrus.Level{TraceLevel}
	}
	return []logrus.Level{}
}

// Fire processes log entries at the trace level.
//
// This method converts trace level entries to debug level entries with
// a [TRACE] prefix to distinguish them from regular debug entries.
//
// Parameters:
// - entry: The log entry to process
//
// Returns:
// - Error if processing fails (always returns nil in current implementation)
func (h *TraceHook) Fire(entry *logrus.Entry) error {
	if entry.Level <= TraceLevel {
		entry.Level = logrus.DebugLevel
		entry.Message = "[TRACE] " + entry.Message
	}
	return nil
}

// ConfigureLogger sets up logrus with the provided configuration.
//
// This function performs the following steps:
// - Maps verbose count to appropriate log levels
// - Configures trace level support when needed
// - Sets up enhanced formatting for different verbose levels
// - Enables caller information for maximum verbosity (-vvv)
//
// Parameters:
// - ctx: Context for cancellation control
//
// Returns:
// - Error if configuration fails
//
// Side Effects:
// - Modifies global logrus configuration
// - Adds custom hooks for trace level support
// - Configures formatters based on verbose level
func (s *LoggerService) ConfigureLogger(ctx context.Context) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("logger configuration canceled: %w", ctx.Err())
	default:
	}

	// Validate configuration first
	level, err := s.mapVerboseLevelWithError()
	if err != nil {
		return fmt.Errorf("invalid log configuration: %w", err)
	}

	// Set the log level
	logrus.SetLevel(level)

	// Add custom trace level support if needed
	if s.config.Verbose >= 2 {
		logrus.AddHook(&TraceHook{
			Enabled: true,
		})
	}

	// Add redaction hook for automatic sensitive data protection
	redactionService := logging.NewRedactionService()
	redactionHook := redactionService.CreateHook()
	logrus.AddHook(redactionHook)

	// Configure formatter based on format preference and verbose level
	if s.config.LogFormat == "json" || s.config.JSONOutput {
		// Use structured JSON formatter
		logrus.SetFormatter(logging.NewStructuredFormatter())
	} else if s.config.Verbose >= 3 {
		// Maximum verbosity: include caller information with text format
		logrus.SetReportCaller(true)
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableColors:    false,
			FullTimestamp:    true,
			TimestampFormat:  "15:04:05.000",
			PadLevelText:     true,
			QuoteEmptyFields: true,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				return "", fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
			},
		})
	} else {
		// Standard text formatting for lower verbose levels
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableColors:    false,
			FullTimestamp:    true,
			TimestampFormat:  "15:04:05",
			PadLevelText:     true,
			QuoteEmptyFields: true,
		})
	}

	return nil
}

// IsTraceEnabled returns whether trace level logging is enabled.
//
// Returns:
// - True if trace level logging is active based on configuration
func (s *LoggerService) IsTraceEnabled() bool {
	return s.config.Verbose >= 2
}

// IsDebugEnabled returns whether debug level logging is enabled.
//
// Returns:
// - True if debug level logging is active based on configuration
func (s *LoggerService) IsDebugEnabled() bool {
	return s.config.Verbose >= 1 || s.mapVerboseToLevel() <= logrus.DebugLevel
}

// GetDebugFlags returns the component-specific debug flags.
//
// Returns:
// - DebugFlags struct containing all component debug settings
func (s *LoggerService) GetDebugFlags() DebugFlags {
	return s.config.Debug
}

// mapVerboseToLevel maps verbose count to appropriate logrus levels.
//
// This function implements the verbose flag mapping according to the plan:
// - No flag or -v 0: INFO level (existing behavior)
// - -v: DEBUG level
// - -vv: TRACE level (standard logrus)
// - -vvv+: TRACE level with caller info
//
// If LogLevel is explicitly set, verbose flags override it when present.
//
// Returns:
// - Appropriate logrus.Level based on configuration
func (s *LoggerService) mapVerboseToLevel() logrus.Level {
	level, _ := s.mapVerboseLevelWithError()
	return level
}

// mapVerboseLevelWithError maps verbose count to logrus levels with error handling.
//
// This function implements the same mapping as mapVerboseToLevel but returns
// errors for invalid configurations instead of silently defaulting.
//
// Returns:
// - Appropriate logrus.Level based on configuration
// - Error if configuration is invalid
func (s *LoggerService) mapVerboseLevelWithError() (logrus.Level, error) {
	// If verbose flag is used, it overrides explicit log level
	if s.config.Verbose > 0 {
		switch s.config.Verbose {
		case 1:
			return logrus.DebugLevel, nil
		case 2:
			return TraceLevel, nil
		default: // 3 or higher
			return TraceLevel, nil
		}
	}

	// Fall back to explicit log level if no verbose flag
	if s.config.LogLevel == "" {
		return logrus.InfoLevel, nil
	}

	level, err := logrus.ParseLevel(s.config.LogLevel)
	if err != nil {
		return logrus.InfoLevel, fmt.Errorf("invalid log level %q: %w", s.config.LogLevel, err)
	}

	return level, nil
}
