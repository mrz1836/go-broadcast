// Package metrics provides performance monitoring and timing utilities.
//
// This package implements comprehensive operation timing with support for
// nested operations, context metadata, and automatic performance warnings.
// It is designed to provide visibility into operation durations across
// the entire go-broadcast application.
//
// Key features include:
// - Timer struct for tracking operation duration
// - Context-aware timing with cancellation support
// - Metadata attachment through AddField method
// - Configurable warnings for slow operations (default: 60 seconds)
// - Human-readable duration formatting
// - Integration with structured logging
//
// Usage examples:
//
//	// Basic operation timing
//	timer := metrics.StartTimer(ctx, logger, "repository_sync")
//	defer timer.Stop()
//
//	// Timer with additional context
//	timer := metrics.StartTimer(ctx, logger, "git_clone").
//	  AddField("repo", "owner/repo").
//	  AddField("branch", "master")
//	defer timer.Stop()
//
//	// Timer with custom threshold for long-running operations
//	timer := metrics.StartTimer(ctx, logger, "large_sync").
//	  WithThreshold(5 * time.Minute)
//	defer timer.Stop()
//
// Important notes:
// - All timing operations accept context.Context for cancellation
// - Timer.Stop() should always be called to complete timing measurement
// - Nested timers are supported for detailed operation breakdown
package metrics

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

// DefaultSlowOperationThreshold is the default duration threshold for slow operation warnings.
// Operations exceeding this threshold will be logged at WARN level.
const DefaultSlowOperationThreshold = 60 * time.Second

// Timer tracks the duration of an operation with support for additional metadata.
//
// Timer provides comprehensive operation timing with automatic logging of
// duration, warnings for slow operations, and support for attaching
// contextual metadata through the AddField method.
//
// Thread Safety: Timer is NOT safe for concurrent use from multiple goroutines.
// Each Timer instance should be used by a single goroutine. If you need to
// add fields from multiple goroutines, use external synchronization.
type Timer struct {
	start     time.Time
	operation string
	logger    *logrus.Entry
	fields    logrus.Fields
	ctx       context.Context //nolint:containedctx // Context needed for cancellation checks during timer lifecycle
	stopped   bool            // Tracks whether Stop() has been called to prevent double-logging
	threshold time.Duration   // Custom threshold for slow operation warnings, 0 means use default
}

// StartTimer creates a new timer for an operation.
//
// This function creates a timer that begins tracking the duration of an
// operation immediately. The timer integrates with the structured logging
// system and supports metadata attachment for detailed operation context.
//
// Parameters:
// - ctx: Context for cancellation control and operation lifecycle (nil defaults to context.Background())
// - logger: Logger entry for structured output with correlation (must not be nil)
// - operation: Name of the operation being timed for identification
//
// Returns:
// - Timer instance for tracking operation duration and metadata
//
// Panics:
// - If logger is nil
//
// Side Effects:
// - Records the start time for duration calculation
// - Creates a logger entry with operation context
// - Initializes fields map for metadata storage
func StartTimer(ctx context.Context, logger *logrus.Entry, operation string) *Timer { //nolint:contextcheck // Intentional: nil ctx defaults to Background() per documented API contract
	if logger == nil {
		panic("metrics: StartTimer logger cannot be nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Create timer with current timestamp and operation context
	timer := &Timer{
		start:     time.Now(),
		operation: operation,
		logger:    logger.WithField(logging.StandardFields.Operation, operation),
		fields:    make(logrus.Fields),
		ctx:       ctx,
	}

	return timer
}

// AddField adds a field to be logged when the timer stops.
//
// This method allows attaching additional context and metadata to the
// timer that will be included in the final timing log entry. This is
// useful for providing operation-specific details like repository names,
// file counts, or other relevant metrics.
//
// Parameters:
// - key: Field name for the metadata entry
// - value: Field value (any type supported by logrus)
//
// Returns:
// - Timer instance for method chaining
//
// Side Effects:
// - Stores the field in the timer's metadata for later logging
//
// Notes:
// - Method chaining is supported for multiple field assignments
// - Fields are logged when Stop() is called
func (t *Timer) AddField(key string, value interface{}) *Timer {
	t.fields[key] = value
	return t
}

// WithThreshold sets a custom duration threshold for slow operation warnings.
//
// Operations exceeding this threshold will be logged at WARN level instead
// of DEBUG level. This allows different operations to have appropriate
// thresholds based on their expected duration.
//
// Parameters:
// - d: Duration threshold for slow operation warnings
//   - Positive value: Use as custom threshold
//   - Zero: Use DefaultSlowOperationThreshold (60 seconds)
//   - Negative value: Disable slow operation warnings entirely
//
// Returns:
// - Timer instance for method chaining
//
// Notes:
// - Method chaining is supported with AddField
// - Default threshold is 60 seconds if not set
func (t *Timer) WithThreshold(d time.Duration) *Timer {
	t.threshold = d
	return t
}

// Stop stops the timer and logs the duration with performance analysis.
//
// This method completes the timing measurement, calculates the total
// duration, and logs the results with appropriate warning levels for
// slow operations. It includes both machine-readable milliseconds and
// human-readable duration formatting.
//
// Stop is idempotent - calling it multiple times returns the original duration
// without logging again. This prevents duplicate log entries from deferred calls.
//
// Returns:
// - Duration of the operation as time.Duration (0 if already stopped)
//
// Side Effects:
// - Logs operation duration with context and metadata (first call only)
// - Issues warnings for operations exceeding the configured threshold
// - Includes all attached fields in the log entry
//
// Performance Analysis:
// - Operations exceeding threshold: WARNING level with slow operation message
// - Normal operations: DEBUG level with completion message
// - All operations include duration_ms and duration_human fields
func (t *Timer) Stop() time.Duration {
	if t.stopped {
		return 0
	}
	t.stopped = true

	// Calculate total duration since timer start
	duration := time.Since(t.start)

	// Add standard timing fields
	t.fields[logging.StandardFields.DurationMs] = duration.Milliseconds()
	t.fields["duration_human"] = duration.String()

	// Determine threshold for slow operation warning
	threshold := t.threshold
	if threshold == 0 {
		threshold = DefaultSlowOperationThreshold
	}

	// Check for slow operations and log with appropriate level
	// Negative threshold disables slow operation warnings
	if threshold > 0 && duration > threshold {
		t.logger.WithFields(t.fields).Warn("Operation took longer than expected")
	} else {
		t.logger.WithFields(t.fields).Debug("Operation completed")
	}

	return duration
}

// StopWithError stops the timer and logs the duration with error context.
//
// This method is similar to Stop() but includes error information in the
// log entry. It's useful for timing operations that may fail, providing
// both duration and error context for debugging.
//
// StopWithError is idempotent - calling it multiple times (or after Stop())
// returns 0 without logging again. This prevents duplicate log entries.
//
// Parameters:
// - err: Error that occurred during the operation (can be nil)
//
// Returns:
// - Duration of the operation as time.Duration (0 if already stopped)
//
// Side Effects:
// - Logs operation duration with error context (first call only)
// - Uses ERROR level for failed operations, DEBUG for successful ones
// - Includes all attached fields and error information in log entry
func (t *Timer) StopWithError(err error) time.Duration {
	if t.stopped {
		return 0
	}
	t.stopped = true

	duration := time.Since(t.start)

	// Add standard timing fields
	t.fields[logging.StandardFields.DurationMs] = duration.Milliseconds()
	t.fields["duration_human"] = duration.String()

	if err != nil {
		// Log failed operations at ERROR level with error context
		t.fields[logging.StandardFields.Error] = err.Error()
		t.fields[logging.StandardFields.Status] = "failed"
		t.logger.WithFields(t.fields).Error("Operation failed")
	} else {
		// Determine threshold for slow operation warning
		threshold := t.threshold
		if threshold == 0 {
			threshold = DefaultSlowOperationThreshold
		}

		// Log successful operations based on duration
		// Negative threshold disables slow operation warnings
		t.fields[logging.StandardFields.Status] = "completed"
		if threshold > 0 && duration > threshold {
			t.logger.WithFields(t.fields).Warn("Operation completed but took longer than expected")
		} else {
			t.logger.WithFields(t.fields).Debug("Operation completed successfully")
		}
	}

	return duration
}

// CheckCancellation checks if the operation context has been canceled.
//
// This method provides a convenient way to check for context cancellation
// during long-running operations. It can be called periodically to ensure
// responsive cancellation behavior.
//
// Returns:
// - true if the context has been canceled, false otherwise
//
// Notes:
// - Should be called periodically during long operations
// - Does not affect timer state or logging
// - Provides early cancellation detection
func (t *Timer) CheckCancellation() bool {
	select {
	case <-t.ctx.Done():
		return true
	default:
		return false
	}
}

// GetElapsed returns the current elapsed time without stopping the timer.
//
// This method provides access to the current elapsed duration without
// completing the timing operation. It's useful for progress monitoring
// or intermediate timing checks.
//
// Returns:
// - Current elapsed duration since timer start
//
// Side Effects:
// - None - timer continues running normally
//
// Notes:
// - Timer continues running after this call
// - Can be called multiple times for progress monitoring
func (t *Timer) GetElapsed() time.Duration {
	return time.Since(t.start)
}
