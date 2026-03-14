// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains race condition tests for cancel operations.
// These tests verify thread safety when accessing cancel-related global state.
package cli

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestCancelGlobalFlags_ConcurrentModification verifies that concurrent
// modifications to cancel-related global flags don't cause races.
//
// The cancel command uses package-level variables protected by cancelFlagsMu.
// This test verifies thread-safe access via getter/setter functions.
func TestCancelGlobalFlags_ConcurrentModification(_ *testing.T) {
	// Save original state using thread-safe getters
	defer resetCancelFlags()

	const goroutines = 10
	var wg sync.WaitGroup

	// Concurrent readers and writers using thread-safe accessors
	for i := 0; i < goroutines; i++ {
		wg.Add(2)

		// Writer
		go func(idx int) {
			defer wg.Done()
			// Thread-safe writes
			setCancelKeepBranches(idx%2 == 0)
			setCancelComment("comment-" + string(rune('0'+idx%10)))
			setCancelGroupFilter([]string{"group-" + string(rune('0'+idx%10))})
			setCancelSkipGroups([]string{"skip-" + string(rune('0'+idx%10))})
		}(i)

		// Reader
		go func() {
			defer wg.Done()
			// Thread-safe reads
			_ = getCancelKeepBranches()
			_ = getCancelComment()
			_ = getCancelGroupFilter()
			_ = getCancelSkipGroups()
		}()
	}

	wg.Wait()
}

// TestPerformCancel_ConcurrentDryRunToggle verifies behavior when
// globalFlags.DryRun is accessed during cancel operations.
//
// This is tested separately because DryRun affects output behavior.
func TestPerformCancel_ConcurrentDryRunToggle(_ *testing.T) {
	// Save and restore global state (thread-safe)
	oldFlags := GetGlobalFlags()
	defer func() { SetFlags(oldFlags) }()

	// This test verifies that concurrent access to globalFlags.DryRun
	// during processCancelTarget is handled safely by the mutex.

	const iterations = 50
	var wg sync.WaitGroup

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(dryRun bool) {
			defer wg.Done()

			// Use SetFlags which is synchronized
			SetFlags(&Flags{DryRun: dryRun})

			// Read back the flag to verify thread safety
			_ = IsDryRun()
		}(i%2 == 0)
	}

	wg.Wait()
}

// TestFilterTargets_ConcurrentAccess verifies that filterTargets
// can be called concurrently with the same state.
func TestFilterTargets_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	s := &state.State{
		Targets: map[string]*state.TargetState{
			"org/repo1": {Repo: "org/repo1", OpenPRs: []gh.PR{{Number: 1}}},
			"org/repo2": {Repo: "org/repo2", OpenPRs: []gh.PR{{Number: 2}}},
			"org/repo3": {Repo: "org/repo3", OpenPRs: []gh.PR{{Number: 3}}},
		},
	}

	const goroutines = 20
	var wg sync.WaitGroup
	results := make(chan []*state.TargetState, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			targets, err := filterTargets(s, nil)
			if err == nil {
				results <- targets
			}
		}()
	}

	wg.Wait()
	close(results)

	// All results should have same count
	for targets := range results {
		assert.Len(t, targets, 3, "all results should have same target count")
	}
}

// TestCancelResult_ConcurrentConstruction verifies that CancelResult
// structs can be constructed concurrently.
func TestCancelResult_ConcurrentConstruction(t *testing.T) {
	t.Parallel()

	const goroutines = 100
	var wg sync.WaitGroup
	results := make(chan CancelResult, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			prNum := idx
			result := CancelResult{
				Repository:    "org/repo-" + string(rune('0'+idx%10)),
				PRNumber:      &prNum,
				PRClosed:      idx%2 == 0,
				BranchName:    "sync/branch-" + string(rune('0'+idx%10)),
				BranchDeleted: idx%3 == 0,
			}
			results <- result
		}(i)
	}

	wg.Wait()
	close(results)

	// Verify all results were created
	count := 0
	for range results {
		count++
	}
	assert.Equal(t, goroutines, count, "should have all results")
}

// TestCancelSummary_ConcurrentUpdates verifies that CancelSummary
// fields can be updated from multiple goroutines.
//
// In real usage, summary is built sequentially, but this
// tests the struct's behavior under concurrent access.
func TestCancelSummary_ConcurrentUpdates(t *testing.T) {
	t.Parallel()

	summary := &CancelSummary{
		Results: make([]CancelResult, 0),
	}

	const goroutines = 10
	var wg sync.WaitGroup
	var mu sync.Mutex // External mutex since CancelSummary isn't thread-safe

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result := CancelResult{
				Repository: "org/repo-" + string(rune('0'+idx)), //nolint:gosec // G115: idx is bounded by goroutines count, safe conversion
				PRClosed:   true,
			}

			// Protected append since slice append isn't atomic
			mu.Lock()
			summary.Results = append(summary.Results, result)
			summary.PRsClosed++
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Without the mutex, this would race
	assert.Len(t, summary.Results, goroutines)
	assert.Equal(t, goroutines, summary.PRsClosed)
}
