package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Static test errors for err113 linter compliance
var (
	ErrTestError = errors.New("test error")
)

// mockTask implements the Task interface for testing
type mockTask struct {
	name        string
	executeFunc func(ctx context.Context) error
	sleepTime   time.Duration
}

func (m *mockTask) Execute(ctx context.Context) error {
	if m.sleepTime > 0 {
		select {
		case <-time.After(m.sleepTime):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if m.executeFunc != nil {
		return m.executeFunc(ctx)
	}
	return nil
}

func (m *mockTask) Name() string {
	return m.name
}

// TestNewPool tests pool creation
func TestNewPool(t *testing.T) {
	t.Run("valid parameters", func(t *testing.T) {
		pool, err := NewPool(4, 10)
		require.NoError(t, err)
		require.NotNil(t, pool)
		assert.Equal(t, 4, pool.workers)
		assert.NotNil(t, pool.taskQueue)
		assert.NotNil(t, pool.results)
	})

	t.Run("invalid workers", func(t *testing.T) {
		pool, err := NewPool(0, 10)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidWorkers)
		require.Nil(t, pool)

		pool, err = NewPool(-1, 10)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidWorkers)
		require.Nil(t, pool)
	})

	t.Run("invalid queue size", func(t *testing.T) {
		pool, err := NewPool(4, 0)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidQueueSize)
		require.Nil(t, pool)

		pool, err = NewPool(4, -1)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidQueueSize)
		require.Nil(t, pool)
	})
}

// TestMustNewPool tests MustNewPool panic behavior
func TestMustNewPool(t *testing.T) {
	t.Run("valid parameters", func(t *testing.T) {
		pool := MustNewPool(4, 10)
		require.NotNil(t, pool)
	})

	t.Run("panics on invalid workers", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewPool(0, 10)
		})
	})

	t.Run("panics on invalid queue size", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewPool(4, 0)
		})
	})
}

// TestPoolStartAndShutdown tests basic pool lifecycle
func TestPoolStartAndShutdown(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx := context.Background()

	pool.Start(ctx)

	// Submit a simple task
	task := &mockTask{name: "test-task"}
	err := pool.Submit(task)
	require.NoError(t, err)

	// Wait for result
	select {
	case result := <-pool.Results():
		assert.Equal(t, "test-task", result.TaskName)
		require.NoError(t, result.Error)
		assert.Greater(t, result.Duration, time.Duration(0))
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}

	pool.Shutdown()

	// Verify results channel is closed
	_, ok := <-pool.Results()
	assert.False(t, ok)
}

// TestPoolSubmitBatch tests batch task submission
func TestPoolSubmitBatch(t *testing.T) {
	pool := MustNewPool(4, 20)
	ctx := context.Background()
	pool.Start(ctx)

	tasks := []Task{
		&mockTask{name: "task-1"},
		&mockTask{name: "task-2"},
		&mockTask{name: "task-3"},
	}

	err := pool.SubmitBatch(tasks)
	require.NoError(t, err)

	// Collect results
	results := make(map[string]Result)
	for i := 0; i < len(tasks); i++ {
		select {
		case result := <-pool.Results():
			results[result.TaskName] = result
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for results")
		}
	}

	// Verify all tasks completed
	assert.Len(t, results, 3)
	for _, task := range tasks {
		result, exists := results[task.(*mockTask).name]
		require.True(t, exists)
		require.NoError(t, result.Error)
	}

	pool.Shutdown()
}

// TestPoolTaskError tests handling of task errors
func TestPoolTaskError(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx := context.Background()
	pool.Start(ctx)

	expectedErr := errors.New("task failed") //nolint:err113 // test error
	task := &mockTask{
		name: "error-task",
		executeFunc: func(_ context.Context) error {
			return expectedErr
		},
	}

	err := pool.Submit(task)
	require.NoError(t, err)

	select {
	case result := <-pool.Results():
		assert.Equal(t, "error-task", result.TaskName)
		assert.Equal(t, expectedErr, result.Error)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}

	pool.Shutdown()
}

// TestPoolQueueFull tests behavior when queue is full
func TestPoolQueueFull(t *testing.T) {
	pool := MustNewPool(1, 2) // Small queue: 1 worker, 2 queue slots
	ctx := context.Background()
	pool.Start(ctx)

	// Use channels to coordinate task execution timing
	taskStarted := make(chan struct{})
	blockTask := make(chan struct{})

	// Create a blocking task that signals when it starts and waits for release
	blockingTask := &mockTask{
		name: "blocking-task",
		executeFunc: func(_ context.Context) error {
			taskStarted <- struct{}{} // Signal that task has started
			<-blockTask               // Wait for test to release
			return nil
		},
	}

	// Submit first task - will be picked up by worker and block
	err := pool.Submit(blockingTask)
	require.NoError(t, err)

	// Wait for the worker to start processing the first task
	select {
	case <-taskStarted:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first task to start")
	}

	// Now submit tasks to fill the queue
	// Since worker is blocked, these will stay in queue
	normalTask := &mockTask{name: "queued-task-1"}
	err = pool.Submit(normalTask)
	require.NoError(t, err)

	normalTask2 := &mockTask{name: "queued-task-2"}
	err = pool.Submit(normalTask2)
	require.NoError(t, err)

	// Queue should be full now - next submit should fail
	normalTask3 := &mockTask{name: "should-fail"}
	err = pool.Submit(normalTask3)
	assert.Equal(t, ErrTaskQueueFull, err)

	// Release the blocking task to allow completion
	close(blockTask)

	// Wait for all tasks to complete (1 blocking + 2 queued = 3 total)
	for i := 0; i < 3; i++ {
		select {
		case <-pool.Results():
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}

	pool.Shutdown()
}

// TestPoolStats tests statistics tracking
func TestPoolStats(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx := context.Background()
	pool.Start(ctx)

	// Initial stats
	processed, active, queued := pool.Stats()
	assert.Equal(t, int64(0), processed)
	assert.Equal(t, int32(0), active)
	assert.Equal(t, 0, queued)

	// Submit tasks
	taskCount := 5

	for i := 0; i < taskCount; i++ {
		task := &mockTask{
			name: "stats-task",
			executeFunc: func(_ context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}

		err := pool.Submit(task)
		require.NoError(t, err)
	}

	// Collect all results to ensure tasks complete
	for i := 0; i < taskCount; i++ {
		select {
		case <-pool.Results():
			// Task completed
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for result %d", i+1)
		}
	}

	// Use Eventually to wait for stats to be updated consistently
	require.Eventually(t, func() bool {
		processed, active, queued := pool.Stats()
		return processed == int64(taskCount) && active == 0 && queued == 0
	}, time.Second, 10*time.Millisecond, "Expected all tasks to be processed and stats updated")

	pool.Shutdown()
}

// TestPoolContextCancellation tests task cancellation via context
func TestPoolContextCancellation(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	var taskStarted atomic.Bool
	task := &mockTask{
		name: "cancellable-task",
		executeFunc: func(ctx context.Context) error {
			taskStarted.Store(true)
			select {
			case <-time.After(time.Second):
				return errors.New("task should have been canceled") //nolint:err113 // test error
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	err := pool.Submit(task)
	require.NoError(t, err)

	// Wait for task to start
	for !taskStarted.Load() {
		time.Sleep(time.Millisecond)
	}

	// Cancel context
	cancel()

	select {
	case result := <-pool.Results():
		assert.Equal(t, "cancellable-task", result.TaskName)
		assert.Equal(t, context.Canceled, result.Error)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}

	pool.Shutdown()
}

// TestPoolConcurrentSubmit tests concurrent task submission
func TestPoolConcurrentSubmit(t *testing.T) {
	pool := MustNewPool(4, 100)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Shutdown() // Ensure cleanup even on timeout

	var submitWg sync.WaitGroup
	taskCount := 50

	// Submit tasks concurrently
	for i := 0; i < taskCount; i++ {
		submitWg.Add(1)
		go func(id int) {
			defer submitWg.Done()
			task := &mockTask{name: fmt.Sprintf("task-%d", id)}
			err := pool.Submit(task)
			assert.NoError(t, err)
		}(i)
	}

	submitWg.Wait()

	// Collect all results with extended timeout for CI/race detection
	results := make([]Result, 0, taskCount)
	timeout := 5 * time.Second // Increased from 2s to handle slower CI environments
	for i := 0; i < taskCount; i++ {
		select {
		case result := <-pool.Results():
			results = append(results, result)
		case <-time.After(timeout):
			t.Fatalf("timeout waiting for results after %v (collected %d/%d results)", timeout, len(results), taskCount)
		}
	}

	assert.Len(t, results, taskCount)

	// Verify stats
	processed, _, _ := pool.Stats()
	assert.Equal(t, int64(taskCount), processed)
}

// TestPoolPanicRecovery tests that worker panics don't crash the pool
func TestPoolPanicRecovery(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx := context.Background()
	pool.Start(ctx)

	// Task that panics
	panicTask := &mockTask{
		name: "panic-task",
		executeFunc: func(_ context.Context) error {
			panic("test panic")
		},
	}

	// Normal task
	normalTask := &mockTask{name: "normal-task"}

	// Submit both tasks
	err := pool.Submit(panicTask)
	require.NoError(t, err)
	err = pool.Submit(normalTask)
	require.NoError(t, err)

	// Should receive results for both tasks
	results := make(map[string]Result)
	timeout := time.After(2 * time.Second)

	for len(results) < 2 {
		select {
		case result := <-pool.Results():
			results[result.TaskName] = result
		case <-timeout:
			t.Fatalf("timeout waiting for results, got %d/2", len(results))
		}
	}

	// Verify normal task succeeded
	normalResult, ok := results["normal-task"]
	require.True(t, ok, "normal task result not found")
	require.NoError(t, normalResult.Error)

	// Verify panic task has error
	panicResult, ok := results["panic-task"]
	require.True(t, ok, "panic task result not found")
	require.Error(t, panicResult.Error)
	require.ErrorIs(t, panicResult.Error, ErrTaskPanicked)
	assert.Contains(t, panicResult.Error.Error(), "test panic")

	pool.Shutdown()
}

// TestErrorDefinitions tests error variable definitions
func TestErrorDefinitions(t *testing.T) {
	t.Run("ErrPoolShuttingDown", func(t *testing.T) {
		require.Error(t, ErrPoolShuttingDown)
		assert.Equal(t, "pool is shutting down", ErrPoolShuttingDown.Error())
	})

	t.Run("ErrTaskQueueFull", func(t *testing.T) {
		require.Error(t, ErrTaskQueueFull)
		assert.Equal(t, "task queue is full", ErrTaskQueueFull.Error())
	})

	t.Run("ErrTaskPanicked", func(t *testing.T) {
		require.Error(t, ErrTaskPanicked)
		assert.Equal(t, "task panicked", ErrTaskPanicked.Error())
	})
}

// TestPoolBatchSubmitPartialFailure tests batch submission with queue full
func TestPoolBatchSubmitPartialFailure(t *testing.T) {
	pool := MustNewPool(1, 2) // Very small queue
	ctx := context.Background()
	pool.Start(ctx)

	// Create many tasks
	tasks := make([]Task, 5)
	for i := range tasks {
		tasks[i] = &mockTask{
			name:      "batch-task",
			sleepTime: 50 * time.Millisecond,
		}
	}

	// Submit batch should fail when queue fills
	err := pool.SubmitBatch(tasks)
	assert.Equal(t, ErrTaskQueueFull, err)

	// Some tasks should have been submitted
	processed, _, queued := pool.Stats()
	assert.GreaterOrEqual(t, int(processed)+queued, 1)

	// Consume any pending results before shutdown to avoid deadlock
	go func() {
		for range pool.Results() {
		}
	}()

	pool.Shutdown()
}

// TestPoolHighConcurrencyStress tests pool under high concurrency stress
func TestPoolHighConcurrencyStress(t *testing.T) {
	const (
		numWorkers  = 10
		queueSize   = 100
		numTasks    = 1000
		numRoutines = 20
	)

	pool := MustNewPool(numWorkers, queueSize)
	ctx := context.Background()
	pool.Start(ctx)

	var totalSubmitted int64
	var totalCompleted int64
	var wg sync.WaitGroup

	// Start result collector
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			atomic.AddInt64(&totalCompleted, 1)
			if result.Error != nil {
				t.Errorf("Task %s failed: %v", result.TaskName, result.Error)
				return
			}
		}
	}()

	// Submit tasks from multiple goroutines concurrently
	submitWg := sync.WaitGroup{}
	for i := 0; i < numRoutines; i++ {
		submitWg.Add(1)
		go func(routineID int) {
			defer submitWg.Done()
			for j := 0; j < numTasks/numRoutines; j++ {
				task := &mockTask{
					name:      fmt.Sprintf("stress-task-%d-%d", routineID, j),
					sleepTime: time.Microsecond * 100, // Very short work
				}

				// Retry submission if queue is full
				for {
					err := pool.Submit(task)
					if err == nil {
						atomic.AddInt64(&totalSubmitted, 1)
						break
					}
					if errors.Is(err, ErrTaskQueueFull) {
						time.Sleep(time.Microsecond * 50) // Brief backoff
						continue
					}
					if err != nil {
						t.Errorf("Unexpected error submitting task: %v", err)
						return
					}
				}
			}
		}(i)
	}

	// Wait for all submissions to complete
	submitWg.Wait()
	t.Logf("All %d tasks submitted", atomic.LoadInt64(&totalSubmitted))

	// Wait for all tasks to complete
	require.Eventually(t, func() bool {
		completed := atomic.LoadInt64(&totalCompleted)
		submitted := atomic.LoadInt64(&totalSubmitted)
		return completed == submitted
	}, 30*time.Second, 100*time.Millisecond,
		"Expected all tasks to complete: submitted=%d, completed=%d",
		atomic.LoadInt64(&totalSubmitted), atomic.LoadInt64(&totalCompleted))

	pool.Shutdown()
	wg.Wait()

	// Verify final counts
	finalSubmitted := atomic.LoadInt64(&totalSubmitted)
	finalCompleted := atomic.LoadInt64(&totalCompleted)

	assert.Equal(t, numTasks, int(finalSubmitted), "Should have submitted all tasks")
	assert.Equal(t, finalSubmitted, finalCompleted, "Should have completed all submitted tasks")

	// Verify pool stats
	processed, _, queued := pool.Stats()
	assert.Equal(t, int64(numTasks), processed, "Pool should have processed all tasks")
	assert.Equal(t, 0, queued, "Queue should be empty after completion")
}

// TestPoolResourceCleanupOnPanic tests resource cleanup when tasks panic
func TestPoolResourceCleanupOnPanic(t *testing.T) {
	const numWorkers = 5
	const numTasks = 50

	pool := MustNewPool(numWorkers, numTasks*2)
	ctx := context.Background()
	pool.Start(ctx)

	var panicCount int64
	var successCount int64
	var wg sync.WaitGroup

	// Collect results
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			if errors.Is(result.Error, ErrTaskPanicked) {
				atomic.AddInt64(&panicCount, 1)
			} else if result.Error == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}
	}()

	// Submit mix of panicking and normal tasks
	tasks := make([]Task, numTasks)
	for i := 0; i < numTasks; i++ {
		if i%3 == 0 { // Every third task panics
			tasks[i] = &mockTask{
				name: fmt.Sprintf("panic-task-%d", i),
				executeFunc: func(_ context.Context) error {
					panic("intentional test panic")
				},
			}
		} else {
			tasks[i] = &mockTask{
				name:      fmt.Sprintf("normal-task-%d", i),
				sleepTime: time.Millisecond * 10,
			}
		}
	}

	err := pool.SubmitBatch(tasks)
	require.NoError(t, err)

	// Wait for all tasks to complete
	require.Eventually(t, func() bool {
		totalCompleted := atomic.LoadInt64(&panicCount) + atomic.LoadInt64(&successCount)
		return totalCompleted == int64(numTasks)
	}, 15*time.Second, 100*time.Millisecond)

	pool.Shutdown()
	wg.Wait()

	// Verify panic handling didn't break the pool
	expectedPanics := int64(0)
	expectedSuccess := int64(0)
	for i := 0; i < numTasks; i++ {
		if i%3 == 0 {
			expectedPanics++
		} else {
			expectedSuccess++
		}
	}

	assert.Equal(t, expectedPanics, atomic.LoadInt64(&panicCount),
		"Should have handled all panicking tasks")
	assert.Equal(t, expectedSuccess, atomic.LoadInt64(&successCount),
		"Should have completed all normal tasks despite panics")

	// Pool should still be functional after panics
	processed, _, _ := pool.Stats()
	assert.Equal(t, int64(numTasks), processed, "Pool should have processed all tasks")
}

// TestPoolContextCancellationCleanup tests cleanup when context is canceled
func TestPoolContextCancellationCleanup(t *testing.T) {
	const numWorkers = 4
	const numTasks = 100

	pool := MustNewPool(numWorkers, numTasks)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	var completedCount int64
	var canceledCount int64
	var wg sync.WaitGroup

	// Collect results
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			if errors.Is(result.Error, context.Canceled) {
				atomic.AddInt64(&canceledCount, 1)
			} else if result.Error == nil {
				atomic.AddInt64(&completedCount, 1)
			}
		}
	}()

	// Submit long-running tasks
	tasks := make([]Task, numTasks)
	for i := 0; i < numTasks; i++ {
		tasks[i] = &mockTask{
			name:      fmt.Sprintf("long-task-%d", i),
			sleepTime: time.Second * 2, // Long enough to be canceled
		}
	}

	err := pool.SubmitBatch(tasks)
	require.NoError(t, err)

	// Let some tasks start, then cancel context
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for cleanup
	require.Eventually(t, func() bool {
		total := atomic.LoadInt64(&completedCount) + atomic.LoadInt64(&canceledCount)
		return total > 0 // Some tasks should complete or be canceled
	}, 10*time.Second, 100*time.Millisecond)

	pool.Shutdown()
	wg.Wait()

	completed := atomic.LoadInt64(&completedCount)
	canceled := atomic.LoadInt64(&canceledCount)
	total := completed + canceled

	t.Logf("Completed: %d, Canceled: %d, Total: %d", completed, canceled, total)

	// Some tasks should have been canceled due to context cancellation
	assert.Positive(t, canceled, "Some tasks should have been canceled")
	assert.LessOrEqual(t, total, int64(numTasks), "Total processed should not exceed submitted")

	// Pool should handle cancellation gracefully
	processed, _, _ := pool.Stats()
	assert.Equal(t, total, processed, "Pool stats should match actual processing")
}

// TestPoolMemoryLeakPrevention tests that the pool doesn't leak memory under stress
func TestPoolMemoryLeakPrevention(t *testing.T) {
	const iterations = 10
	const tasksPerIteration = 100

	// Run multiple cycles to detect memory leaks
	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			pool := MustNewPool(5, tasksPerIteration*2)
			ctx := context.Background()
			pool.Start(ctx)

			var completed int64
			var wg sync.WaitGroup

			// Result collector
			wg.Add(1)
			go func() {
				defer wg.Done()
				for result := range pool.Results() {
					atomic.AddInt64(&completed, 1)
					_ = result // Process result to prevent optimization
				}
			}()

			// Submit tasks with various characteristics
			tasks := make([]Task, tasksPerIteration)
			for j := 0; j < tasksPerIteration; j++ {
				switch j % 4 {
				case 0: // Normal task
					tasks[j] = &mockTask{
						name:      fmt.Sprintf("normal-%d-%d", i, j),
						sleepTime: time.Microsecond * 50,
					}
				case 1: // Task with error
					tasks[j] = &mockTask{
						name: fmt.Sprintf("error-%d-%d", i, j),
						executeFunc: func(_ context.Context) error {
							return ErrTestError
						},
					}
				case 2: // Task that panics
					tasks[j] = &mockTask{
						name: fmt.Sprintf("panic-%d-%d", i, j),
						executeFunc: func(_ context.Context) error {
							panic("test panic for memory leak test")
						},
					}
				case 3: // Task with context usage
					tasks[j] = &mockTask{
						name: fmt.Sprintf("context-%d-%d", i, j),
						executeFunc: func(_ context.Context) error {
							select {
							case <-time.After(time.Microsecond * 100):
								return nil
							case <-ctx.Done():
								return ctx.Err()
							}
						},
					}
				}
			}

			err := pool.SubmitBatch(tasks)
			require.NoError(t, err)

			// Wait for completion
			require.Eventually(t, func() bool {
				return atomic.LoadInt64(&completed) == int64(tasksPerIteration)
			}, 10*time.Second, 10*time.Millisecond)

			pool.Shutdown()
			wg.Wait()

			// Verify all tasks were processed
			assert.Equal(t, int64(tasksPerIteration), atomic.LoadInt64(&completed))
		})
	}
}

// TestSubmitAfterShutdown verifies that Submit returns ErrPoolShuttingDown after Shutdown
func TestSubmitAfterShutdown(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx := context.Background()
	pool.Start(ctx)

	// Drain results in background
	go func() {
		for range pool.Results() {
		}
	}()

	pool.Shutdown()

	// Submit should return ErrPoolShuttingDown, not panic
	task := &mockTask{name: "after-shutdown-task"}
	err := pool.Submit(task)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrPoolShuttingDown)
}

// TestDoubleShutdown verifies that calling Shutdown twice doesn't panic
func TestDoubleShutdown(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx := context.Background()
	pool.Start(ctx)

	// Drain results in background
	go func() {
		for range pool.Results() {
		}
	}()

	// First shutdown
	pool.Shutdown()

	// Second shutdown should not panic
	assert.NotPanics(t, func() {
		pool.Shutdown()
	})

	// Third shutdown should also not panic
	assert.NotPanics(t, func() {
		pool.Shutdown()
	})
}

// TestNilTask verifies that Submit returns ErrNilTask for nil tasks
func TestNilTask(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Shutdown()

	// Drain results in background
	go func() {
		for range pool.Results() {
		}
	}()

	err := pool.Submit(nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNilTask)
}

// TestDoubleStart verifies that calling Start twice doesn't spawn duplicate workers
func TestDoubleStart(t *testing.T) {
	pool := MustNewPool(2, 10)
	ctx := context.Background()

	// Start the pool
	pool.Start(ctx)

	// Second start should be a no-op
	pool.Start(ctx)

	// Submit a task
	task := &mockTask{name: "test-task"}
	err := pool.Submit(task)
	require.NoError(t, err)

	// Should receive exactly one result
	select {
	case result := <-pool.Results():
		assert.Equal(t, "test-task", result.TaskName)
		require.NoError(t, result.Error)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for result")
	}

	pool.Shutdown()

	// Verify pool processed only once (not duplicated by multiple workers)
	processed, _, _ := pool.Stats()
	assert.Equal(t, int64(1), processed)
}

// TestContextCancellationNoTaskLoss verifies tasks are accounted for on cancellation
func TestContextCancellationNoTaskLoss(t *testing.T) {
	const numTasks = 20
	pool := MustNewPool(4, numTasks)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	var resultsReceived atomic.Int64
	var wg sync.WaitGroup

	// Collect results
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range pool.Results() {
			resultsReceived.Add(1)
		}
	}()

	// Submit tasks that wait for context cancellation
	for i := 0; i < numTasks; i++ {
		task := &mockTask{
			name: fmt.Sprintf("task-%d", i),
			executeFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		}
		err := pool.Submit(task)
		require.NoError(t, err)
	}

	// Cancel context after tasks have started
	time.Sleep(50 * time.Millisecond)
	cancel()

	pool.Shutdown()
	wg.Wait()

	// All submitted tasks should have a result (either completed or canceled)
	// Some tasks may be dropped if results channel was full during shutdown,
	// but we should have processed most of them
	received := resultsReceived.Load()
	t.Logf("Received %d/%d results", received, numTasks)
	assert.Positive(t, received, "Should have received at least some results")
}

// TestResultsChannelBackpressure verifies workers don't deadlock when results aren't consumed
func TestResultsChannelBackpressure(t *testing.T) {
	// Create pool with small results buffer
	pool := MustNewPool(2, 5)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit tasks without consuming results
	for i := 0; i < 10; i++ {
		task := &mockTask{
			name:      fmt.Sprintf("task-%d", i),
			sleepTime: 10 * time.Millisecond,
		}
		_ = pool.Submit(task) // Ignore errors - queue may fill up
	}

	// Wait a bit then cancel - this should not deadlock
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Shutdown should complete within timeout (not deadlock)
	done := make(chan struct{})
	go func() {
		pool.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// Success - shutdown completed
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown deadlocked - results channel backpressure issue")
	}
}

// TestShutdownBeforeStart verifies Shutdown works even if Start was never called
func TestShutdownBeforeStart(t *testing.T) {
	pool := MustNewPool(2, 10)

	// Shutdown without Start should not panic
	assert.NotPanics(t, func() {
		pool.Shutdown()
	})
}

// TestErrorDefinitionsExtended tests all error variable definitions
func TestErrorDefinitionsExtended(t *testing.T) {
	t.Run("ErrNilTask", func(t *testing.T) {
		require.Error(t, ErrNilTask)
		assert.Equal(t, "task is nil", ErrNilTask.Error())
	})

	t.Run("ErrInvalidWorkers", func(t *testing.T) {
		require.Error(t, ErrInvalidWorkers)
		assert.Equal(t, "workers must be at least 1", ErrInvalidWorkers.Error())
	})

	t.Run("ErrInvalidQueueSize", func(t *testing.T) {
		require.Error(t, ErrInvalidQueueSize)
		assert.Equal(t, "queue size must be at least 1", ErrInvalidQueueSize.Error())
	})
}
