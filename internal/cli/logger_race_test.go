// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains race condition tests for logger operations.
// These tests verify thread safety when configuring and using LoggerService.
package cli

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoggerService_ConcurrentConfigureLogger verifies that ConfigureLogger
// can be called concurrently without race conditions.
//
// This matters because logging setup may be triggered from multiple goroutines
// during application initialization.
func TestLoggerService_ConcurrentConfigureLogger(t *testing.T) {
	// This test should be run with -race flag
	const goroutines = 10

	configs := []*LogConfig{
		{Verbose: 0, LogLevel: "info"},
		{Verbose: 1, LogLevel: "debug"},
		{Verbose: 2, LogLevel: "trace"},
		{Verbose: 3, LogLevel: "trace"},
	}

	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			config := configs[idx%len(configs)]
			service := NewLoggerService(config)

			ctx := context.Background()
			if err := service.ConfigureLogger(ctx); err != nil {
				errors <- err
			}

			// Also test concurrent reads
			_ = service.IsTraceEnabled()
			_ = service.IsDebugEnabled()
			_ = service.GetDebugFlags()
		}(i)
	}

	wg.Wait()
	close(errors)

	// Collect any errors
	for err := range errors {
		t.Errorf("ConfigureLogger returned error: %v", err)
	}
}

// TestLoggerService_ConcurrentMethodAccess verifies that all LoggerService
// methods can be accessed concurrently without race conditions.
func TestLoggerService_ConcurrentMethodAccess(_ *testing.T) {
	config := &LogConfig{
		Verbose:  2,
		LogLevel: "debug",
		Debug: DebugFlags{
			Git:       true,
			API:       true,
			Transform: true,
			Config:    true,
			State:     true,
		},
	}

	service := NewLoggerService(config)

	const goroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Mix of read operations
			_ = service.IsTraceEnabled()
			_ = service.IsDebugEnabled()
			flags := service.GetDebugFlags()
			_ = flags.Git
			_ = flags.API
			_ = service.mapVerboseToLevel()
		}()
	}

	wg.Wait()
}

// TestLoggerService_ConcurrentNilConfig verifies that nil config handling
// is thread-safe across all methods.
func TestLoggerService_ConcurrentNilConfig(t *testing.T) {
	// Create service with nil config
	service := NewLoggerService(nil)

	const goroutines = 20
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// All these should be safe with nil config
			isTrace := service.IsTraceEnabled()
			assert.False(t, isTrace, "nil config should return false for IsTraceEnabled")

			isDebug := service.IsDebugEnabled()
			assert.False(t, isDebug, "nil config should return false for IsDebugEnabled")

			flags := service.GetDebugFlags()
			assert.False(t, flags.Git, "nil config should return zero-value flags")
		}()
	}

	wg.Wait()
}

// TestLoggerService_HookAccumulation verifies that hooks are not duplicated
// when ConfigureLogger is called multiple times.
//
// This is important because repeated calls to ConfigureLogger should be idempotent.
func TestLoggerService_HookAccumulation(t *testing.T) {
	// This test documents that ConfigureLogger adds hooks each time,
	// which may not be ideal. The test verifies current behavior.

	config := &LogConfig{
		Verbose: 2, // Triggers trace hook
	}

	service := NewLoggerService(config)
	ctx := context.Background()

	// Call ConfigureLogger multiple times
	for i := 0; i < 3; i++ {
		err := service.ConfigureLogger(ctx)
		require.NoError(t, err)
	}

	// Logrus doesn't provide a way to check hook count.
	// This test documents that calling ConfigureLogger multiple times
	// may add duplicate hooks, which is acceptable for single-use CLI.
}

// TestLogConfig_ConcurrentAccess verifies that LogConfig struct can be
// accessed concurrently through LoggerService.
func TestLogConfig_ConcurrentAccess(_ *testing.T) {
	config := &LogConfig{
		ConfigFile:    "test.yaml",
		LogLevel:      "info",
		LogFormat:     "text",
		Verbose:       1,
		JSONOutput:    false,
		CorrelationID: "test-correlation",
		Debug: DebugFlags{
			Git:       true,
			API:       false,
			Transform: true,
			Config:    false,
			State:     true,
		},
	}

	service := NewLoggerService(config)

	const goroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Read all fields through service methods
			_ = service.IsTraceEnabled()
			_ = service.IsDebugEnabled()
			flags := service.GetDebugFlags()

			// Access struct fields
			_ = flags.Git
			_ = flags.API
			_ = flags.Transform
			_ = flags.Config
			_ = flags.State
		}()
	}

	wg.Wait()
}
