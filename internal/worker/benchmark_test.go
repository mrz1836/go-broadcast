// Package worker provides benchmarking tests for worker pool implementations and performance analysis.
package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
)

// testTask implements the Task interface for benchmarking
type testTask struct {
	name     string
	duration time.Duration
	workload func() error
}

func (t *testTask) Execute(ctx context.Context) error {
	select {
	case <-time.After(t.duration):
		if t.workload != nil {
			return t.workload()
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *testTask) Name() string {
	return t.name
}

// cpuIntensiveTask simulates CPU-intensive work
type cpuIntensiveTask struct {
	name       string
	iterations int
}

func (t *cpuIntensiveTask) Execute(ctx context.Context) error {
	result := 0
	for i := 0; i < t.iterations; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Simulate CPU work
			result += i * i
		}
	}
	_ = result // Use result to prevent optimization
	return nil
}

func (t *cpuIntensiveTask) Name() string {
	return t.name
}

// BenchmarkWorkerPool tests worker pool performance across different configurations
func BenchmarkWorkerPool(b *testing.B) {
	workerCounts := []int{1, 5, 10, 20}
	taskCounts := []int{10, 100, 1000}

	for _, workers := range workerCounts {
		for _, tasks := range taskCounts {
			b.Run(fmt.Sprintf("Workers_%d_Tasks_%d", workers, tasks), func(b *testing.B) {
				benchmark.WithMemoryTracking(b, func() {
					pool := NewPool(workers, tasks)
					pool.Start(context.Background())

					// Submit tasks
					for j := 0; j < tasks; j++ {
						task := &testTask{
							name:     fmt.Sprintf("task_%d", j),
							duration: time.Microsecond * 100, // Small but measurable work
						}
						if err := pool.Submit(task); err != nil {
							b.Fatalf("Failed to submit task: %v", err)
						}
					}

					// Collect results
					collected := 0
					for result := range pool.Results() {
						if result.Error != nil {
							b.Errorf("Task failed: %v", result.Error)
						}
						collected++
						if collected >= tasks {
							break
						}
					}

					pool.Shutdown()
				})
			})
		}
	}
}

// BenchmarkWorkerPoolThroughput measures tasks processed per second
func BenchmarkWorkerPoolThroughput(b *testing.B) {
	scenarios := []struct {
		name         string
		workers      int
		taskCount    int
		taskDuration time.Duration
	}{
		{"Fast_Tasks_Few_Workers", 2, 1000, time.Microsecond * 10},
		{"Fast_Tasks_Many_Workers", 10, 1000, time.Microsecond * 10},
		{"Slow_Tasks_Few_Workers", 2, 100, time.Millisecond},
		{"Slow_Tasks_Many_Workers", 10, 100, time.Millisecond},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				start := time.Now()

				pool := NewPool(scenario.workers, scenario.taskCount)
				pool.Start(context.Background())

				// Submit all tasks
				for j := 0; j < scenario.taskCount; j++ {
					task := &testTask{
						name:     fmt.Sprintf("task_%d", j),
						duration: scenario.taskDuration,
					}
					_ = pool.Submit(task) // Ignore error in benchmark
				}

				// Wait for completion
				collected := 0
				for range pool.Results() {
					collected++
					if collected >= scenario.taskCount {
						break
					}
				}

				pool.Shutdown()

				duration := time.Since(start)
				throughput := float64(scenario.taskCount) / duration.Seconds()
				b.ReportMetric(throughput, "tasks/sec")
			})
		})
	}
}

// BenchmarkWorkerPoolCPUIntensive tests CPU-bound workloads
func BenchmarkWorkerPoolCPUIntensive(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8}
	iterations := []int{1000, 10000, 100000}

	for _, workers := range workerCounts {
		for _, iter := range iterations {
			b.Run(fmt.Sprintf("Workers_%d_Iterations_%d", workers, iter), func(b *testing.B) {
				taskCount := 20 // Fixed number of tasks

				benchmark.WithMemoryTracking(b, func() {
					pool := NewPool(workers, taskCount)
					pool.Start(context.Background())

					// Submit CPU-intensive tasks
					for j := 0; j < taskCount; j++ {
						task := &cpuIntensiveTask{
							name:       fmt.Sprintf("cpu_task_%d", j),
							iterations: iter,
						}
						_ = pool.Submit(task) // Ignore error in benchmark
					}

					// Wait for completion
					collected := 0
					for range pool.Results() {
						collected++
						if collected >= taskCount {
							break
						}
					}

					pool.Shutdown()
				})
			})
		}
	}
}

// BenchmarkWorkerPoolMemoryUsage tests memory efficiency
func BenchmarkWorkerPoolMemoryUsage(b *testing.B) {
	scenarios := []struct {
		name      string
		workers   int
		queueSize int
		taskCount int
	}{
		{"Small_Queue", 5, 10, 50},
		{"Medium_Queue", 10, 100, 500},
		{"Large_Queue", 20, 1000, 5000},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			benchmark.WithMemoryTracking(b, func() {
				pool := NewPool(scenario.workers, scenario.queueSize)
				pool.Start(context.Background())

				// Submit all tasks first
				for j := 0; j < scenario.taskCount; j++ {
					task := &testTask{
						name:     fmt.Sprintf("task_%d", j),
						duration: time.Microsecond * 50,
					}
					_ = pool.Submit(task) // Ignore error in benchmark
				}

				// Collect results
				collected := 0
				for range pool.Results() {
					collected++
					if collected >= scenario.taskCount {
						break
					}
				}

				pool.Shutdown()
			})
		})
	}
}

// BenchmarkWorkerPoolScaling tests scaling characteristics
func BenchmarkWorkerPoolScaling(b *testing.B) {
	taskCount := 1000
	workerCounts := []int{1, 2, 4, 8, 16, 32}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				start := time.Now()

				pool := NewPool(workers, taskCount)
				pool.Start(context.Background())

				// Submit tasks
				for j := 0; j < taskCount; j++ {
					task := &testTask{
						name:     fmt.Sprintf("task_%d", j),
						duration: time.Microsecond * 100,
					}
					_ = pool.Submit(task) // Ignore error in benchmark
				}

				// Wait for completion
				collected := 0
				for range pool.Results() {
					collected++
					if collected >= taskCount {
						break
					}
				}

				pool.Shutdown()

				duration := time.Since(start)
				b.ReportMetric(duration.Seconds(), "total_time")
				b.ReportMetric(float64(taskCount)/duration.Seconds(), "tasks/sec")
			})
		})
	}
}

// BenchmarkWorkerPoolBatchSubmission tests batch submission performance
func BenchmarkWorkerPoolBatchSubmission(b *testing.B) {
	batchSizes := []int{1, 10, 100, 1000}
	workers := 10

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				pool := NewPool(workers, batchSize*2)
				pool.Start(context.Background())

				// Create batch of tasks
				tasks := make([]Task, batchSize)
				for j := 0; j < batchSize; j++ {
					tasks[j] = &testTask{
						name:     fmt.Sprintf("batch_task_%d", j),
						duration: time.Microsecond * 50,
					}
				}

				// Submit batch
				if err := pool.SubmitBatch(tasks); err != nil {
					b.Fatalf("Failed to submit batch: %v", err)
				}

				// Wait for completion
				collected := 0
				for range pool.Results() {
					collected++
					if collected >= batchSize {
						break
					}
				}

				pool.Shutdown()
			})
		})
	}
}

// BenchmarkWorkerPoolStats tests statistics collection overhead
func BenchmarkWorkerPoolStats(b *testing.B) {
	pool := NewPool(5, 100)
	pool.Start(context.Background())
	defer pool.Shutdown()

	// Submit some tasks to have meaningful stats
	for i := 0; i < 50; i++ {
		task := &testTask{
			name:     fmt.Sprintf("stats_task_%d", i),
			duration: time.Microsecond * 10,
		}
		_ = pool.Submit(task) // Ignore error in benchmark
	}

	benchmark.WithMemoryTracking(b, func() {
		_, _, _ = pool.Stats()
	})
}

// BenchmarkWorkerPoolContextCancellation tests cancellation performance
func BenchmarkWorkerPoolContextCancellation(b *testing.B) {
	scenarios := []struct {
		name      string
		workers   int
		taskCount int
	}{
		{"Few_Workers", 2, 100},
		{"Many_Workers", 10, 100},
		{"Many_Tasks", 5, 1000},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				pool := NewPool(scenario.workers, scenario.taskCount)
				pool.Start(context.Background())

				// Submit long-running tasks
				for j := 0; j < scenario.taskCount; j++ {
					task := &testTask{
						name:     fmt.Sprintf("cancel_task_%d", j),
						duration: time.Second, // Long duration that should be canceled
					}
					_ = pool.Submit(task) // Ignore error in benchmark
				}

				// Cancel after a short time
				time.Sleep(time.Millisecond * 10)
				pool.cancel() // Cancel the context

				// Wait for shutdown
				pool.Shutdown()
			})
		})
	}
}
