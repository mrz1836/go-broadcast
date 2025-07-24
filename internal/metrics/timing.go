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
// - Automatic warnings for slow operations (>30 seconds)
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
//	  AddField("branch", "main")
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

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/sirupsen/logrus"
)

// Timer tracks the duration of an operation with support for additional metadata.
//
// Timer provides comprehensive operation timing with automatic logging of
// duration, warnings for slow operations, and support for attaching
// contextual metadata through the AddField method.
type Timer struct {
	start     time.Time
	operation string
	logger    *logrus.Entry
	fields    logrus.Fields
	ctx       context.Context //nolint:containedctx // Context needed for cancellation checks during timer lifecycle
}

// StartTimer creates a new timer for an operation.
//
// This function creates a timer that begins tracking the duration of an
// operation immediately. The timer integrates with the structured logging
// system and supports metadata attachment for detailed operation context.
//
// Parameters:
// - ctx: Context for cancellation control and operation lifecycle
// - logger: Logger entry for structured output with correlation
// - operation: Name of the operation being timed for identification
//
// Returns:
// - Timer instance for tracking operation duration and metadata
//
// Side Effects:
// - Records the start time for duration calculation
// - Creates a logger entry with operation context
// - Initializes fields map for metadata storage
func StartTimer(ctx context.Context, logger *logrus.Entry, operation string) *Timer {
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

// Stop stops the timer and logs the duration with performance analysis.
//
// This method completes the timing measurement, calculates the total
// duration, and logs the results with appropriate warning levels for
// slow operations. It includes both machine-readable milliseconds and
// human-readable duration formatting.
//
// Returns:
// - Duration of the operation as time.Duration
//
// Side Effects:
// - Logs operation duration with context and metadata
// - Issues warnings for operations taking longer than 30 seconds
// - Includes all attached fields in the log entry
//
// Performance Analysis:
// - Operations > 30 seconds: WARNING level with slow operation message
// - Normal operations: DEBUG level with completion message
// - All operations include duration_ms and duration_human fields
func (t *Timer) Stop() time.Duration {
	// Calculate total duration since timer start
	duration := time.Since(t.start)

	// Add standard timing fields
	t.fields[logging.StandardFields.DurationMs] = duration.Milliseconds()
	t.fields["duration_human"] = duration.String()

	// Check for slow operations and log with appropriate level
	if duration > 30*time.Second {
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
// Parameters:
// - err: Error that occurred during the operation (can be nil)
//
// Returns:
// - Duration of the operation as time.Duration
//
// Side Effects:
// - Logs operation duration with error context
// - Uses ERROR level for failed operations, DEBUG for successful ones
// - Includes all attached fields and error information in log entry
func (t *Timer) StopWithError(err error) time.Duration {
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
		// Log successful operations based on duration
		t.fields[logging.StandardFields.Status] = "completed"
		if duration > 30*time.Second {
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
