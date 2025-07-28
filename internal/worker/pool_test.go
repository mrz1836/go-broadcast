package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	pool := NewPool(4, 10)
	require.NotNil(t, pool)
	assert.Equal(t, 4, pool.workers)
	assert.NotNil(t, pool.taskQueue)
	assert.NotNil(t, pool.results)
	assert.NotNil(t, pool.cancel)
}

// TestPoolStartAndShutdown tests basic pool lifecycle
func TestPoolStartAndShutdown(t *testing.T) {
	pool := NewPool(2, 10)
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
	pool := NewPool(4, 20)
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
	pool := NewPool(2, 10)
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
	pool := NewPool(1, 2) // Small queue
	ctx := context.Background()
	pool.Start(ctx)

	// Submit tasks that take time
	slowTask := &mockTask{
		name:      "slow-task",
		sleepTime: 500 * time.Millisecond,
	}

	// Fill the queue
	err := pool.Submit(slowTask)
	require.NoError(t, err)
	err = pool.Submit(slowTask)
	require.NoError(t, err)

	// Queue should be full now
	err = pool.Submit(slowTask)
	assert.Equal(t, ErrTaskQueueFull, err)

	// Wait for tasks to complete
	for i := 0; i < 2; i++ {
		select {
		case <-pool.Results():
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for results")
		}
	}

	pool.Shutdown()
}

// TestPoolStats tests statistics tracking
func TestPoolStats(t *testing.T) {
	pool := NewPool(2, 10)
	ctx := context.Background()
	pool.Start(ctx)

	// Initial stats
	processed, active, queued := pool.Stats()
	assert.Equal(t, int64(0), processed)
	assert.Equal(t, int32(0), active)
	assert.Equal(t, 0, queued)

	// Submit tasks
	var wg sync.WaitGroup
	taskCount := 5

	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		task := &mockTask{
			name: "stats-task",
			executeFunc: func(_ context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}

		err := pool.Submit(task)
		require.NoError(t, err)

		go func() {
			defer wg.Done()
			<-pool.Results()
		}()
	}

	// Wait for all tasks to complete
	wg.Wait()

	// Final stats
	processed, active, queued = pool.Stats()
	assert.Equal(t, int64(taskCount), processed)
	assert.Equal(t, int32(0), active)
	assert.Equal(t, 0, queued)

	pool.Shutdown()
}

// TestPoolContextCancellation tests task cancellation via context
func TestPoolContextCancellation(t *testing.T) {
	pool := NewPool(2, 10)
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
	pool := NewPool(4, 100)
	ctx := context.Background()
	pool.Start(ctx)

	var submitWg sync.WaitGroup
	taskCount := 50

	// Submit tasks concurrently
	for i := 0; i < taskCount; i++ {
		submitWg.Add(1)
		go func(id int) {
			defer submitWg.Done()
			task := &mockTask{name: string(rune(id))}
			err := pool.Submit(task)
			assert.NoError(t, err)
		}(i)
	}

	submitWg.Wait()

	// Collect all results
	results := make([]Result, 0, taskCount)
	for i := 0; i < taskCount; i++ {
		select {
		case result := <-pool.Results():
			results = append(results, result)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}

	assert.Len(t, results, taskCount)

	// Verify stats
	processed, _, _ := pool.Stats()
	assert.Equal(t, int64(taskCount), processed)

	pool.Shutdown()
}

// TestPoolPanicRecovery tests that worker panics don't crash the pool
func TestPoolPanicRecovery(t *testing.T) {
	pool := NewPool(2, 10)
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
	pool := NewPool(1, 2) // Very small queue
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
		for range pool.Results() { //nolint:revive // intentionally draining channel
		}
	}()

	pool.Shutdown()
}
