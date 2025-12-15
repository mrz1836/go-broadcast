// Package worker provides a worker pool implementation for concurrent task execution.
package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Worker pool errors
var (
	ErrPoolShuttingDown = errors.New("pool is shutting down")
	ErrTaskQueueFull    = errors.New("task queue is full")
	ErrTaskPanicked     = errors.New("task panicked")
	ErrNilTask          = errors.New("task is nil")
	ErrInvalidWorkers   = errors.New("workers must be at least 1")
	ErrInvalidQueueSize = errors.New("queue size must be at least 1")
)

// Task represents a unit of work
type Task interface {
	Execute(ctx context.Context) error
	Name() string
}

// Result wraps task execution results
type Result struct {
	TaskName string
	Error    error
	Duration time.Duration
}

// Pool manages a pool of workers
type Pool struct {
	workers   int
	taskQueue chan Task
	results   chan Result
	wg        sync.WaitGroup

	// Lifecycle management
	// ctx is stored to control worker goroutine lifecycle after Start() is called.
	// The context is derived from the parent context passed to Start().
	ctx          context.Context //nolint:containedctx // Required for worker lifecycle management
	cancel       context.CancelFunc
	shutdownOnce sync.Once
	shuttingDown atomic.Bool
	started      atomic.Bool

	// Metrics
	tasksProcessed atomic.Int64
	tasksActive    atomic.Int32
}

// NewPool creates a new worker pool with the specified number of workers and queue size.
// Returns an error if workers < 1 or queueSize < 1.
func NewPool(workers, queueSize int) (*Pool, error) {
	if workers < 1 {
		return nil, ErrInvalidWorkers
	}
	if queueSize < 1 {
		return nil, ErrInvalidQueueSize
	}

	return &Pool{
		workers:   workers,
		taskQueue: make(chan Task, queueSize),
		results:   make(chan Result, queueSize),
	}, nil
}

// MustNewPool creates a new worker pool, panicking on invalid parameters.
// Use NewPool for error handling instead.
func MustNewPool(workers, queueSize int) *Pool {
	pool, err := NewPool(workers, queueSize)
	if err != nil {
		panic(err)
	}
	return pool
}

// Start begins processing tasks. It is idempotent - calling it multiple times has no effect.
// The provided context controls the lifetime of the workers.
func (p *Pool) Start(ctx context.Context) {
	// Ensure Start is only executed once
	if p.started.Swap(true) {
		return
	}

	// Create a derived context that we control for cancellation
	p.ctx, p.cancel = context.WithCancel(ctx)

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(p.ctx) //nolint:contextcheck // p.ctx is derived from parent ctx via WithCancel above
	}
}

// Submit adds a task to the queue.
// Returns ErrPoolShuttingDown if the pool is shutting down.
// Returns ErrNilTask if task is nil.
// Returns ErrTaskQueueFull if the queue is full (non-blocking).
func (p *Pool) Submit(task Task) error {
	if task == nil {
		return ErrNilTask
	}
	if p.shuttingDown.Load() {
		return ErrPoolShuttingDown
	}
	select {
	case p.taskQueue <- task:
		return nil
	default:
		return ErrTaskQueueFull
	}
}

// SubmitBatch submits multiple tasks
func (p *Pool) SubmitBatch(tasks []Task) error {
	for _, task := range tasks {
		if err := p.Submit(task); err != nil {
			return err
		}
	}
	return nil
}

// Results returns the results channel
func (p *Pool) Results() <-chan Result {
	return p.results
}

// Shutdown gracefully stops the pool. It is idempotent - calling it multiple times has no effect.
// It closes the task queue, waits for all workers to finish, then closes the results channel.
func (p *Pool) Shutdown() {
	p.shutdownOnce.Do(func() {
		p.shuttingDown.Store(true)
		if p.cancel != nil {
			p.cancel()
		}
		close(p.taskQueue)
		p.wg.Wait()
		close(p.results)
	})
}

// Stats returns current pool statistics
func (p *Pool) Stats() (processed int64, active int32, queued int) {
	return p.tasksProcessed.Load(), p.tasksActive.Load(), len(p.taskQueue)
}

// worker processes tasks from the queue
func (p *Pool) worker(ctx context.Context) {
	defer p.wg.Done()

	for task := range p.taskQueue {
		p.tasksActive.Add(1)
		start := time.Now()

		// Check if context is already canceled before processing
		var err error
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
			// Recover from panics to prevent worker crash
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("%w: %v", ErrTaskPanicked, r)
					}
				}()
				err = task.Execute(ctx)
			}()
		}

		result := Result{
			TaskName: task.Name(),
			Error:    err,
			Duration: time.Since(start),
		}

		// Update stats before sending result to ensure consistency:
		// consumers observing a result will always see updated stats
		p.tasksActive.Add(-1)
		p.tasksProcessed.Add(1)

		// Send result with context awareness to prevent deadlock
		select {
		case p.results <- result:
			// Result sent successfully
		case <-ctx.Done():
			// Context canceled while trying to send result - try one more time
			// with non-blocking send, then give up to avoid blocking shutdown
			select {
			case p.results <- result:
			default:
				// Results channel full and context canceled, drop result to unblock
			}
		}
	}
}
