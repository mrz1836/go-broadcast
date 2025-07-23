package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.NotNil(t, opts)
	assert.False(t, opts.DryRun)
	assert.False(t, opts.Force)
	assert.Equal(t, 5, opts.MaxConcurrency)
	assert.True(t, opts.UpdateExistingPRs)
	assert.Equal(t, 10*time.Minute, opts.Timeout)
	assert.True(t, opts.CleanupTempFiles)
}

func TestOptionsWithDryRun(t *testing.T) {
	opts := DefaultOptions().WithDryRun(true)

	assert.True(t, opts.DryRun)

	// Test toggling back
	opts = opts.WithDryRun(false)
	assert.False(t, opts.DryRun)
}

func TestOptionsWithForce(t *testing.T) {
	opts := DefaultOptions().WithForce(true)

	assert.True(t, opts.Force)

	// Test toggling back
	opts = opts.WithForce(false)
	assert.False(t, opts.Force)
}

func TestOptionsWithMaxConcurrency(t *testing.T) {
	opts := DefaultOptions()

	// Test setting valid concurrency
	opts = opts.WithMaxConcurrency(10)
	assert.Equal(t, 10, opts.MaxConcurrency)

	// Test setting zero (should become 1)
	opts = opts.WithMaxConcurrency(0)
	assert.Equal(t, 1, opts.MaxConcurrency)

	// Test setting negative (should become 1)
	opts = opts.WithMaxConcurrency(-5)
	assert.Equal(t, 1, opts.MaxConcurrency)
}

func TestOptionsWithTimeout(t *testing.T) {
	timeout := 5 * time.Minute
	opts := DefaultOptions().WithTimeout(timeout)

	assert.Equal(t, timeout, opts.Timeout)
}

func TestOptionsChaining(t *testing.T) {
	opts := DefaultOptions().
		WithDryRun(true).
		WithForce(true).
		WithMaxConcurrency(3).
		WithTimeout(15 * time.Minute)

	assert.True(t, opts.DryRun)
	assert.True(t, opts.Force)
	assert.Equal(t, 3, opts.MaxConcurrency)
	assert.Equal(t, 15*time.Minute, opts.Timeout)
}
