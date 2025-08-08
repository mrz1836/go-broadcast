// Package metrics provides performance monitoring and timing utilities.
package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestStartTimer(t *testing.T) {
	tests := []struct {
		name      string
		ctxFunc   func() context.Context
		logger    *logrus.Entry
		operation string
		validate  func(t *testing.T, timer *Timer)
	}{
		{
			name:      "basic timer creation",
			ctxFunc:   func() context.Context { return context.Background() },
			logger:    logrus.NewEntry(logrus.New()),
			operation: "test_operation",
			validate: func(t *testing.T, timer *Timer) {
				assert.NotNil(t, timer, "timer should not be nil")
				assert.Equal(t, "test_operation", timer.operation, "operation should be set")
				assert.NotNil(t, timer.logger, "logger should be set")
				assert.NotNil(t, timer.fields, "fields should be initialized")
				assert.NotNil(t, timer.ctx, "context should be set")
				assert.False(t, timer.start.IsZero(), "start time should be set")
			},
		},
		{
			name:      "timer with context cancellation",
			ctxFunc:   func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx },
			logger:    logrus.NewEntry(logrus.New()),
			operation: "canceled_operation",
			validate: func(t *testing.T, timer *Timer) {
				assert.NotNil(t, timer, "timer should not be nil")
				assert.Equal(t, "canceled_operation", timer.operation, "operation should be set")
				assert.True(t, timer.CheckCancellation(), "should detect cancellation")
			},
		},
		{
			name:      "timer with logger that has existing fields",
			ctxFunc:   func() context.Context { return context.Background() },
			logger:    logrus.NewEntry(logrus.New()).WithField("existing", "value"),
			operation: "operation_with_logger_fields",
			validate: func(t *testing.T, timer *Timer) {
				assert.NotNil(t, timer, "timer should not be nil")
				assert.Equal(t, "operation_with_logger_fields", timer.operation, "operation should be set")

				// Check that operation field was added to logger
				operationValue, exists := timer.logger.Data[logging.StandardFields.Operation]
				assert.True(t, exists, "operation field should be added to logger")
				assert.Equal(t, "operation_with_logger_fields", operationValue, "operation value should be correct")

				// Check that existing fields are preserved
				existingValue, exists := timer.logger.Data["existing"]
				assert.True(t, exists, "existing field should be preserved")
				assert.Equal(t, "value", existingValue, "existing value should be preserved")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timer := StartTimer(tt.ctxFunc(), tt.logger, tt.operation)
			tt.validate(t, timer)
		})
	}
}

func TestTimer_AddField(t *testing.T) {
	timer := StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "test")

	tests := []struct {
		name     string
		key      string
		value    interface{}
		validate func(t *testing.T, timer *Timer)
	}{
		{
			name:  "add string field",
			key:   "repo",
			value: "owner/repo",
			validate: func(t *testing.T, timer *Timer) {
				assert.Equal(t, "owner/repo", timer.fields["repo"], "string field should be added")
			},
		},
		{
			name:  "add integer field",
			key:   "count",
			value: 42,
			validate: func(t *testing.T, timer *Timer) {
				assert.Equal(t, 42, timer.fields["count"], "integer field should be added")
			},
		},
		{
			name:  "add boolean field",
			key:   "success",
			value: true,
			validate: func(t *testing.T, timer *Timer) {
				assert.Equal(t, true, timer.fields["success"], "boolean field should be added")
			},
		},
		{
			name:  "add nil field",
			key:   "optional",
			value: nil,
			validate: func(t *testing.T, timer *Timer) {
				assert.Nil(t, timer.fields["optional"], "nil field should be added")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timer.AddField(tt.key, tt.value)

			// Test method chaining
			assert.Equal(t, timer, result, "AddField should return the same timer for chaining")

			tt.validate(t, timer)
		})
	}
}

func TestTimer_AddField_Chaining(t *testing.T) {
	timer := StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "chain_test")

	// Test method chaining
	result := timer.AddField("field1", "value1").
		AddField("field2", 123).
		AddField("field3", true)

	assert.Equal(t, timer, result, "method chaining should return the same timer")
	assert.Equal(t, "value1", timer.fields["field1"], "first field should be set")
	assert.Equal(t, 123, timer.fields["field2"], "second field should be set")
	assert.Equal(t, true, timer.fields["field3"], "third field should be set")
}

func TestTimer_Stop(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Timer
		validate func(t *testing.T, timer *Timer, duration time.Duration)
	}{
		{
			name: "normal operation timing",
			setup: func() *Timer {
				timer := StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "normal_op")
				timer.AddField("test", "value")
				return timer
			},
			validate: func(t *testing.T, timer *Timer, duration time.Duration) {
				assert.Positive(t, duration, "duration should be positive")
				assert.Less(t, duration, time.Second, "duration should be reasonable for test")

				// Check that timing fields were added
				durationMs, exists := timer.fields[logging.StandardFields.DurationMs]
				assert.True(t, exists, "duration_ms field should be added")
				assert.Equal(t, duration.Milliseconds(), durationMs, "duration_ms should match")

				durationHuman, exists := timer.fields["duration_human"]
				assert.True(t, exists, "duration_human field should be added")
				assert.Equal(t, duration.String(), durationHuman, "duration_human should match")
			},
		},
		{
			name: "operation with no additional fields",
			setup: func() *Timer {
				return StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "no_fields_op")
			},
			validate: func(t *testing.T, timer *Timer, duration time.Duration) {
				assert.GreaterOrEqual(t, duration, time.Duration(0), "duration should be non-negative")
				assert.Contains(t, timer.fields, logging.StandardFields.DurationMs, "duration_ms should be present")
				assert.Contains(t, timer.fields, "duration_human", "duration_human should be present")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timer := tt.setup()

			// Add a small delay to ensure measurable duration
			time.Sleep(1 * time.Millisecond)

			duration := timer.Stop()
			tt.validate(t, timer, duration)
		})
	}
}

func TestTimer_StopWithError(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*Timer, error)
		validate func(t *testing.T, timer *Timer, duration time.Duration, err error)
	}{
		{
			name: "successful operation",
			setup: func() (*Timer, error) {
				timer := StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "success_op")
				timer.AddField("test", "value")
				return timer, nil
			},
			validate: func(t *testing.T, timer *Timer, duration time.Duration, _ error) {
				assert.Positive(t, duration, "duration should be positive")

				// Check timing fields
				assert.Contains(t, timer.fields, logging.StandardFields.DurationMs, "duration_ms should be present")
				assert.Contains(t, timer.fields, "duration_human", "duration_human should be present")

				// Check status field for success
				status, exists := timer.fields[logging.StandardFields.Status]
				assert.True(t, exists, "status field should be present")
				assert.Equal(t, "completed", status, "status should be 'completed' for success")

				// Should not have error field
				_, hasError := timer.fields[logging.StandardFields.Error]
				assert.False(t, hasError, "error field should not be present for successful operation")
			},
		},
		{
			name: "failed operation",
			setup: func() (*Timer, error) {
				timer := StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "failed_op")
				timer.AddField("test", "value")
				return timer, assert.AnError
			},
			validate: func(t *testing.T, timer *Timer, duration time.Duration, err error) {
				assert.Positive(t, duration, "duration should be positive")

				// Check timing fields
				assert.Contains(t, timer.fields, logging.StandardFields.DurationMs, "duration_ms should be present")
				assert.Contains(t, timer.fields, "duration_human", "duration_human should be present")

				// Check status field for failure
				status, exists := timer.fields[logging.StandardFields.Status]
				assert.True(t, exists, "status field should be present")
				assert.Equal(t, "failed", status, "status should be 'failed' for error")

				// Check error field
				errorField, exists := timer.fields[logging.StandardFields.Error]
				assert.True(t, exists, "error field should be present")
				assert.Equal(t, err.Error(), errorField, "error field should contain error message")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timer, err := tt.setup()

			// Add a small delay to ensure measurable duration
			time.Sleep(1 * time.Millisecond)

			duration := timer.StopWithError(err)
			tt.validate(t, timer, duration, err)
		})
	}
}

func TestTimer_CheckCancellation(t *testing.T) {
	tests := []struct {
		name     string
		ctxFunc  func() context.Context
		expected bool
	}{
		{
			name:     "context not canceled",
			ctxFunc:  func() context.Context { return context.Background() },
			expected: false,
		},
		{
			name:     "context canceled",
			ctxFunc:  func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx },
			expected: true,
		},
		{
			name: "context with timeout not expired",
			ctxFunc: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
				// Store cancel to avoid lint warning, but don't call it
				// We want to test a non-expired timeout
				_ = cancel
				return ctx
			},
			expected: false,
		},
		{
			name: "context with timeout expired",
			ctxFunc: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
				defer cancel()
				time.Sleep(time.Millisecond)
				return ctx
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timer := StartTimer(tt.ctxFunc(), logrus.NewEntry(logrus.New()), "cancellation_test")

			result := timer.CheckCancellation()
			assert.Equal(t, tt.expected, result, "cancellation check should match expected result")
		})
	}
}

func TestTimer_GetElapsed(t *testing.T) {
	timer := StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "elapsed_test")

	// Get initial elapsed time (should be very small)
	elapsed1 := timer.GetElapsed()
	assert.GreaterOrEqual(t, elapsed1, time.Duration(0), "initial elapsed time should be non-negative")
	assert.Less(t, elapsed1, time.Millisecond, "initial elapsed time should be very small")

	// Wait a bit and check again
	time.Sleep(5 * time.Millisecond)
	elapsed2 := timer.GetElapsed()

	assert.Greater(t, elapsed2, elapsed1, "second elapsed time should be greater than first")
	assert.GreaterOrEqual(t, elapsed2, 5*time.Millisecond, "elapsed time should include sleep time")

	// Timer should still be running
	elapsed3 := timer.GetElapsed()
	assert.GreaterOrEqual(t, elapsed3, elapsed2, "timer should continue running")
}

func TestTimer_Integration(t *testing.T) {
	// Test full timer lifecycle with realistic usage
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	entry := logrus.NewEntry(logger)
	timer := StartTimer(context.Background(), entry, "integration_test")

	// Add fields throughout operation
	timer.AddField("repo", "owner/test-repo")
	timer.AddField("files_processed", 0)

	// Simulate some work
	time.Sleep(2 * time.Millisecond)

	// Check elapsed time during operation
	elapsed := timer.GetElapsed()
	assert.GreaterOrEqual(t, elapsed, 2*time.Millisecond, "elapsed time should include work time")

	// Update fields
	timer.AddField("files_processed", 5)
	timer.AddField("changes_detected", true)

	// Check cancellation (should be false)
	assert.False(t, timer.CheckCancellation(), "operation should not be canceled")

	// Complete operation
	duration := timer.Stop()

	// Validate final state
	assert.GreaterOrEqual(t, duration, 2*time.Millisecond, "total duration should include all work time")

	// Check all fields are present
	assert.Equal(t, "owner/test-repo", timer.fields["repo"], "repo field should be preserved")
	assert.Equal(t, 5, timer.fields["files_processed"], "files_processed should be updated value")
	assert.Equal(t, true, timer.fields["changes_detected"], "changes_detected should be preserved")
	assert.Contains(t, timer.fields, logging.StandardFields.DurationMs, "duration_ms should be added")
	assert.Contains(t, timer.fields, "duration_human", "duration_human should be added")
}

func TestTimer_ConcurrentUsage(t *testing.T) {
	// Test that multiple timers can be used concurrently without interference
	const numTimers = 10
	timers := make([]*Timer, numTimers)

	// Start multiple timers
	for i := 0; i < numTimers; i++ {
		logger := logrus.NewEntry(logrus.New())
		timer := StartTimer(context.Background(), logger, "concurrent_test")
		timer.AddField("timer_id", i)
		timers[i] = timer
	}

	// Let them run for different amounts of time
	for i := 0; i < numTimers; i++ {
		time.Sleep(time.Duration(i+1) * time.Millisecond)
		timers[i].AddField("iterations", i*10)
	}

	// Stop all timers and verify they're independent
	for i, timer := range timers {
		duration := timer.Stop()

		assert.Positive(t, duration, "timer %d should have positive duration", i)
		assert.Equal(t, i, timer.fields["timer_id"], "timer %d should have correct ID", i)
		assert.Equal(t, i*10, timer.fields["iterations"], "timer %d should have correct iterations", i)
		assert.Contains(t, timer.fields, logging.StandardFields.DurationMs, "timer %d should have duration_ms", i)
	}
}

func TestTimer_SlowOperation(t *testing.T) {
	// Test behavior with simulated slow operation (but not actually slow in test)
	timer := StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "slow_test")

	// Manually set start time to simulate slow operation for warning logic
	// This is a bit of a hack, but allows us to test the warning behavior without actually waiting
	timer.start = time.Now().Add(-35 * time.Second)

	duration := timer.Stop()

	// Duration should reflect the artificial start time
	assert.Greater(t, duration, 30*time.Second, "simulated slow operation should show long duration")

	// Fields should still be populated correctly
	assert.Contains(t, timer.fields, logging.StandardFields.DurationMs, "duration_ms should be present")
	assert.Contains(t, timer.fields, "duration_human", "duration_human should be present")

	durationMs := timer.fields[logging.StandardFields.DurationMs]
	assert.Greater(t, durationMs.(int64), int64(30000), "duration_ms should show > 30 seconds")
}

func TestTimer_FieldOverwrite(t *testing.T) {
	// Test that fields can be overwritten
	timer := StartTimer(context.Background(), logrus.NewEntry(logrus.New()), "overwrite_test")

	// Set initial field
	timer.AddField("status", "starting")
	assert.Equal(t, "starting", timer.fields["status"], "initial status should be set")

	// Overwrite field
	timer.AddField("status", "processing")
	assert.Equal(t, "processing", timer.fields["status"], "status should be overwritten")

	// Add more fields
	timer.AddField("progress", 0.5)

	// Overwrite again
	timer.AddField("status", "completed")
	assert.Equal(t, "completed", timer.fields["status"], "status should be overwritten again")
	assert.InEpsilon(t, 0.5, timer.fields["progress"], 0.001, "progress should be preserved")

	// Stop and verify final fields
	timer.Stop()
	assert.Equal(t, "completed", timer.fields["status"], "final status should be preserved")
	assert.InEpsilon(t, 0.5, timer.fields["progress"], 0.001, "progress should still be preserved")
	assert.Contains(t, timer.fields, logging.StandardFields.DurationMs, "timing fields should be added")
}
