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
	assert.False(t, opts.Automerge)
	assert.Empty(t, opts.AutomergeLabels)
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

func TestOptionsWithAutomerge(t *testing.T) {
	opts := DefaultOptions().WithAutomerge(true)

	assert.True(t, opts.Automerge)

	// Test toggling back
	opts = opts.WithAutomerge(false)
	assert.False(t, opts.Automerge)
}

func TestOptionsWithAutomergeLabels(t *testing.T) {
	labels := []string{"automerge", "ready-to-merge"}
	opts := DefaultOptions().WithAutomergeLabels(labels)

	assert.Equal(t, labels, opts.AutomergeLabels)

	// Test setting empty labels
	opts = opts.WithAutomergeLabels([]string{})
	assert.Empty(t, opts.AutomergeLabels)

	// Test setting nil labels
	opts = opts.WithAutomergeLabels(nil)
	assert.Nil(t, opts.AutomergeLabels)
}

func TestOptionsWithGroupFilter(t *testing.T) {
	groups := []string{"core", "security"}
	opts := DefaultOptions().WithGroupFilter(groups)

	assert.Equal(t, groups, opts.GroupFilter)
}

func TestOptionsWithSkipGroups(t *testing.T) {
	skipGroups := []string{"experimental", "dev"}
	opts := DefaultOptions().WithSkipGroups(skipGroups)

	assert.Equal(t, skipGroups, opts.SkipGroups)
}

func TestOptionsChaining(t *testing.T) {
	automergeLabels := []string{"automerge", "ready"}
	groups := []string{"core"}

	opts := DefaultOptions().
		WithDryRun(true).
		WithForce(true).
		WithMaxConcurrency(3).
		WithTimeout(15 * time.Minute).
		WithAutomerge(true).
		WithAutomergeLabels(automergeLabels).
		WithGroupFilter(groups).
		WithSkipGroups([]string{"experimental"})

	assert.True(t, opts.DryRun)
	assert.True(t, opts.Force)
	assert.Equal(t, 3, opts.MaxConcurrency)
	assert.Equal(t, 15*time.Minute, opts.Timeout)
	assert.True(t, opts.Automerge)
	assert.Equal(t, automergeLabels, opts.AutomergeLabels)
	assert.Equal(t, groups, opts.GroupFilter)
	assert.Equal(t, []string{"experimental"}, opts.SkipGroups)
}
