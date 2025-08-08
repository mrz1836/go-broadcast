package sync

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/errors"
)

func TestNewProgressTracker(t *testing.T) {
	tracker := NewProgressTracker(5, true)

	assert.NotNil(t, tracker)
	assert.Equal(t, 5, tracker.totalRepos)
	assert.True(t, tracker.dryRun)
	assert.Equal(t, 0, tracker.completed)
	assert.Equal(t, 0, tracker.successful)
	assert.Equal(t, 0, tracker.failed)
	assert.NotNil(t, tracker.errors)
	assert.NotNil(t, tracker.repoStatus)
}

func TestProgressTrackerRecordSuccess(t *testing.T) {
	tracker := NewProgressTracker(3, false)

	tracker.RecordSuccess("org/repo1")

	assert.Equal(t, 1, tracker.successful)
	assert.Equal(t, RepoStatusSuccess, tracker.repoStatus["org/repo1"])
}

func TestProgressTrackerRecordError(t *testing.T) {
	tracker := NewProgressTracker(3, false)
	testErr := errors.ErrTest

	tracker.RecordError("org/repo1", testErr)

	assert.Equal(t, 1, tracker.failed)
	assert.Equal(t, RepoStatusFailed, tracker.repoStatus["org/repo1"])
	assert.Equal(t, testErr, tracker.errors["org/repo1"])
	assert.Equal(t, testErr, tracker.lastError)
}

func TestProgressTrackerRecordSkipped(t *testing.T) {
	tracker := NewProgressTracker(3, false)

	tracker.RecordSkipped("org/repo1", "up-to-date")

	assert.Equal(t, 1, tracker.skipped)
	assert.Equal(t, RepoStatusSkipped, tracker.repoStatus["org/repo1"])
}

func TestProgressTrackerStartFinishRepository(t *testing.T) {
	tracker := NewProgressTracker(2, false)

	// Start repository
	tracker.StartRepository("org/repo1")
	assert.Equal(t, RepoStatusInProgress, tracker.repoStatus["org/repo1"])

	// Finish repository without explicit success/error
	tracker.FinishRepository("org/repo1")
	assert.Equal(t, 1, tracker.completed)
	assert.Equal(t, 1, tracker.successful)
	assert.Equal(t, RepoStatusSuccess, tracker.repoStatus["org/repo1"])

	// Start and finish with explicit success
	tracker.StartRepository("org/repo2")
	tracker.RecordSuccess("org/repo2")
	tracker.FinishRepository("org/repo2")

	assert.Equal(t, 2, tracker.completed)
	assert.Equal(t, 2, tracker.successful)
	assert.Equal(t, RepoStatusSuccess, tracker.repoStatus["org/repo2"])
}

func TestProgressTrackerGetProgress(t *testing.T) {
	tracker := NewProgressTracker(5, false)

	// Initial progress
	completed, total, percentage := tracker.GetProgress()
	assert.Equal(t, 0, completed)
	assert.Equal(t, 5, total)
	assert.InDelta(t, 0.0, percentage, 0.001)

	// After some completions
	tracker.completed = 2
	completed, total, percentage = tracker.GetProgress()
	assert.Equal(t, 2, completed)
	assert.Equal(t, 5, total)
	assert.InEpsilon(t, 40.0, percentage, 0.001)

	// All completed
	tracker.completed = 5
	completed, total, percentage = tracker.GetProgress()
	assert.Equal(t, 5, completed)
	assert.Equal(t, 5, total)
	assert.InEpsilon(t, 100.0, percentage, 0.001)
}

func TestProgressTrackerGetResults(t *testing.T) {
	tracker := NewProgressTracker(3, true)

	// Record some results
	tracker.RecordSuccess("org/repo1")
	tracker.RecordError("org/repo2", errors.ErrTest)
	tracker.RecordSkipped("org/repo3", "up-to-date")

	// Small delay to ensure duration > 0
	time.Sleep(1 * time.Millisecond)

	results := tracker.GetResults()

	assert.Equal(t, 3, results.TotalRepos)
	assert.Equal(t, 1, results.Successful)
	assert.Equal(t, 1, results.Failed)
	assert.Equal(t, 1, results.Skipped)
	assert.Positive(t, results.Duration)
	assert.NotNil(t, results.Errors)
	assert.Len(t, results.Errors, 1)
	assert.True(t, results.DryRun)
	assert.GreaterOrEqual(t, results.Duration, time.Duration(0))
}

func TestProgressTrackerHasErrors(t *testing.T) {
	tracker := NewProgressTracker(3, false)

	// Initially no errors
	assert.False(t, tracker.HasErrors())

	// After recording an error
	tracker.RecordError("org/repo1", errors.ErrTest)
	assert.True(t, tracker.HasErrors())

	// After setting global error
	tracker2 := NewProgressTracker(3, false)
	assert.False(t, tracker2.HasErrors())
	tracker2.SetError(errors.ErrTest)
	assert.True(t, tracker2.HasErrors())
}

func TestProgressTrackerGetLastError(t *testing.T) {
	tracker := NewProgressTracker(3, false)

	// Initially no error
	require.NoError(t, tracker.GetLastError())

	// After recording an error
	tracker.RecordError("org/repo1", errors.ErrTest)
	assert.Equal(t, errors.ErrTest, tracker.GetLastError())

	// After setting global error
	tracker.SetError(errors.ErrTest)
	assert.Equal(t, errors.ErrTest, tracker.GetLastError())
}

func TestProgressTrackerConcurrency(t *testing.T) {
	// Test that the progress tracker is thread-safe
	tracker := NewProgressTracker(100, false)

	// Simulate concurrent operations
	done := make(chan bool, 3)

	// Goroutine 1: Record successes
	go func() {
		for i := 0; i < 30; i++ {
			tracker.RecordSuccess(fmt.Sprintf("success-%d", i))
		}
		done <- true
	}()

	// Goroutine 2: Record errors
	go func() {
		for i := 0; i < 30; i++ {
			tracker.RecordError(fmt.Sprintf("error-%d", i), errors.ErrTest)
		}
		done <- true
	}()

	// Goroutine 3: Record skipped
	go func() {
		for i := 0; i < 30; i++ {
			tracker.RecordSkipped(fmt.Sprintf("skipped-%d", i), "reason")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify final state
	results := tracker.GetResults()
	assert.Equal(t, 30, results.Successful)
	assert.Equal(t, 30, results.Failed)
	assert.Equal(t, 30, results.Skipped)
	assert.Len(t, results.Errors, 30)
}
