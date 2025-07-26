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
	cancel    context.CancelFunc

	// Metrics
	tasksProcessed atomic.Int64
	tasksActive    atomic.Int32
}

// NewPool creates a new worker pool
func NewPool(workers int, queueSize int) *Pool {
	_, cancel := context.WithCancel(context.Background())

	return &Pool{
		workers:   workers,
		taskQueue: make(chan Task, queueSize),
		results:   make(chan Result, queueSize),
		cancel:    cancel,
	}
}

// Start begins processing tasks
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Submit adds a task to the queue
func (p *Pool) Submit(task Task) error {
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

// Shutdown gracefully stops the pool
func (p *Pool) Shutdown() {
	close(p.taskQueue)
	p.wg.Wait()
	close(p.results)
}

// Stats returns current pool statistics
func (p *Pool) Stats() (processed int64, active int32, queued int) {
	return p.tasksProcessed.Load(), p.tasksActive.Load(), len(p.taskQueue)
}

// worker processes tasks from the queue
func (p *Pool) worker(ctx context.Context, _ int) {
	defer p.wg.Done()

	for task := range p.taskQueue {
		select {
		case <-ctx.Done():
			return
		default:
			p.tasksActive.Add(1)
			start := time.Now()

			// Recover from panics to prevent worker crash
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("%w: %v", ErrTaskPanicked, r)
					}
				}()
				err = task.Execute(ctx)
			}()

			p.results <- Result{
				TaskName: task.Name(),
				Error:    err,
				Duration: time.Since(start),
			}

			p.tasksActive.Add(-1)
			p.tasksProcessed.Add(1)
		}
	}
}
