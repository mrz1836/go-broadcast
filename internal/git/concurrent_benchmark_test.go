package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/testutil"
	"github.com/mrz1836/go-broadcast/internal/worker"
	"github.com/sirupsen/logrus"
)

// gitTask implements worker.Task for git operations
type gitTask struct {
	name      string
	operation func(context.Context) error
}

func (t *gitTask) Execute(ctx context.Context) error {
	return t.operation(ctx)
}

func (t *gitTask) Name() string {
	return t.name
}

// writeBenchmarkFile creates a single test file for benchmark usage
func writeBenchmarkFile(b *testing.B, filePath, content string) {
	b.Helper()
	testutil.WriteBenchmarkFile(b, filePath, content)
}

// createBenchmarkTestFiles creates multiple test files with "test_file_" pattern for benchmark usage
func createBenchmarkTestFiles(b *testing.B, dir string, count int) []string {
	b.Helper()
	return benchmark.SetupBenchmarkFiles(b, dir, count)
}

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(ctx context.Context, b *testing.B) (string, Client, func()) {
	b.Helper()

	tmpDir := benchmark.SetupBenchmarkRepo(b)
	repoPath := filepath.Join(tmpDir, "test-repo")

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in benchmarks

	client, err := NewClient(logger, &logging.LogConfig{})
	if err != nil {
		b.Fatalf("Failed to create git client: %v", err)
	}

	// Initialize repository
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		b.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize git repo
	if err := client.(*gitClient).runCommand(client.(*gitClient).createCommand(ctx, repoPath, "init")); err != nil {
		b.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Set user config for commits
	if err := client.(*gitClient).runCommand(client.(*gitClient).createCommand(ctx, repoPath, "config", "user.name", "Test User")); err != nil {
		b.Fatalf("Failed to set git user name: %v", err)
	}
	if err := client.(*gitClient).runCommand(client.(*gitClient).createCommand(ctx, repoPath, "config", "user.email", "test@example.com")); err != nil {
		b.Fatalf("Failed to set git user email: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return repoPath, client, cleanup
}

// Helper to create git command for testing
func (g *gitClient) createCommand(ctx context.Context, repoPath string, args ...string) *exec.Cmd {
	fullArgs := []string{"-C", repoPath}
	fullArgs = append(fullArgs, args...)
	return exec.CommandContext(ctx, "git", fullArgs...) //nolint:gosec // Git command with controlled arguments
}

// BenchmarkConcurrentGitOperations tests concurrent git operations using worker pools
func BenchmarkConcurrentGitOperations(b *testing.B) {
	scenarios := []struct {
		name       string
		repos      int
		concurrent int
		operation  string
	}{
		{"Status_Sequential", 10, 1, "status"},
		{"Status_Concurrent_5", 10, 5, "status"},
		{"Status_Concurrent_10", 10, 10, "status"},
		{"Add_Sequential", 20, 1, "add"},
		{"Add_Concurrent_5", 20, 5, "add"},
		{"Add_Concurrent_10", 20, 10, "add"},
		{"Commit_Sequential", 15, 1, "commit"},
		{"Commit_Concurrent_5", 15, 5, "commit"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			ctx := context.Background()
			repoPath, client, cleanup := setupTestRepo(ctx, b)
			defer cleanup()

			// Create test files for each repo operation
			createBenchmarkTestFiles(b, repoPath, scenario.repos)

			benchmark.WithMemoryTracking(b, func() {
				pool := worker.NewPool(scenario.concurrent, scenario.repos)
				pool.Start(context.Background())

				// Create tasks based on operation type
				var tasks []worker.Task
				for j := 0; j < scenario.repos; j++ {
					var task worker.Task
					switch scenario.operation {
					case "status":
						task = &gitTask{
							name: fmt.Sprintf("status_%d", j),
							operation: func(ctx context.Context) error {
								_, err := client.GetCurrentBranch(ctx, repoPath)
								return err
							},
						}
					case "add":
						fileIndex := j
						task = &gitTask{
							name: fmt.Sprintf("add_%d", j),
							operation: func(ctx context.Context) error {
								fileName := fmt.Sprintf("test_file_%d.txt", fileIndex)
								return client.Add(ctx, repoPath, fileName)
							},
						}
					case "commit":
						task = &gitTask{
							name: fmt.Sprintf("commit_%d", j),
							operation: func(ctx context.Context) error {
								// Add a unique file first
								fileName := fmt.Sprintf("commit_file_%d_%d.txt", b.N, j)
								filePath := filepath.Join(repoPath, fileName)
								content := fmt.Sprintf("commit content %d", j)
								writeBenchmarkFile(b, filePath, content)

								if err := client.Add(ctx, repoPath, fileName); err != nil {
									return err
								}

								message := fmt.Sprintf("Commit %d", j)
								return client.Commit(ctx, repoPath, message)
							},
						}
					}
					tasks = append(tasks, task)
				}

				// Submit all tasks
				if err := pool.SubmitBatch(tasks); err != nil {
					b.Fatalf("Failed to submit batch: %v", err)
				}

				// Wait for completion
				completed := 0
				errorCount := 0
				for result := range pool.Results() {
					if result.Error != nil && !errors.Is(result.Error, ErrNoChanges) {
						errorCount++
					}
					completed++
					if completed >= scenario.repos {
						break
					}
				}

				pool.Shutdown()

				if errorCount > scenario.repos/2 { // Allow some failures but not too many
					b.Errorf("Too many errors: %d out of %d operations failed", errorCount, scenario.repos)
				}
			})
		})
	}
}

// BenchmarkBatchOperations tests the performance of batch git operations
func BenchmarkBatchOperations(b *testing.B) {
	fileCounts := []int{10, 50, 100, 500}

	for _, fileCount := range fileCounts {
		b.Run(fmt.Sprintf("BatchAdd_%d_Files", fileCount), func(b *testing.B) {
			ctx := context.Background()
			repoPath, client, cleanup := setupTestRepo(ctx, b)
			defer cleanup()

			// Create test files
			filePaths := testutil.CreateBenchmarkFiles(b, repoPath, fileCount)
			var files []string
			for _, filePath := range filePaths {
				files = append(files, filepath.Base(filePath))
			}

			batchClient := client.(BatchClient)

			benchmark.WithMemoryTracking(b, func() {
				if err := batchClient.BatchAddFiles(ctx, repoPath, files); err != nil {
					b.Errorf("Batch add failed: %v", err)
				}
			})
		})

		b.Run(fmt.Sprintf("BatchStatus_%d_Files", fileCount), func(b *testing.B) {
			ctx := context.Background()
			repoPath, client, cleanup := setupTestRepo(ctx, b)
			defer cleanup()

			// Create and add test files
			filePaths := testutil.CreateBenchmarkFiles(b, repoPath, fileCount)
			var files []string
			for _, filePath := range filePaths {
				files = append(files, filepath.Base(filePath))
			}

			batchClient := client.(BatchClient)

			benchmark.WithMemoryTracking(b, func() {
				_, err := batchClient.BatchStatus(ctx, repoPath, files)
				if err != nil {
					b.Errorf("Batch status failed: %v", err)
				}
			})
		})
	}
}

// BenchmarkConcurrentVsSequential compares concurrent vs sequential git operations
func BenchmarkConcurrentVsSequential(b *testing.B) {
	repoCount := 20
	fileCount := 10

	// Sequential benchmark
	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			start := time.Now()

			for repo := 0; repo < repoCount; repo++ {
				ctx := context.Background()
				repoPath, client, cleanup := setupTestRepo(ctx, b)

				// Create files
				testutil.CreateBenchmarkFiles(b, repoPath, fileCount)

				// Add files
				for file := 0; file < fileCount; file++ {
					fileName := fmt.Sprintf("file_%d.txt", file)
					if err := client.Add(ctx, repoPath, fileName); err != nil {
						b.Fatalf("Failed to add file: %v", err)
					}
				}

				cleanup()
			}

			duration := time.Since(start)
			b.ReportMetric(duration.Seconds(), "total_time")
		}
	})

	// Concurrent benchmark
	b.Run("Concurrent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			start := time.Now()

			pool := worker.NewPool(10, repoCount)
			pool.Start(context.Background())

			// Create tasks for each repo
			var tasks []worker.Task
			for repo := 0; repo < repoCount; repo++ {
				task := &gitTask{
					name: fmt.Sprintf("repo_operation_%d", repo),
					operation: func(ctx context.Context) error {
						repoPath, client, cleanup := setupTestRepo(ctx, b)
						defer cleanup()

						// Create and add files
						filePaths := testutil.CreateBenchmarkFiles(b, repoPath, fileCount)
						for _, filePath := range filePaths {
							fileName := filepath.Base(filePath)
							if err := client.Add(ctx, repoPath, fileName); err != nil {
								return err
							}
						}

						return nil
					},
				}
				tasks = append(tasks, task)
			}

			// Submit and wait
			if err := pool.SubmitBatch(tasks); err != nil {
				b.Fatalf("Failed to submit batch: %v", err)
			}

			completed := 0
			for range pool.Results() {
				completed++
				if completed >= repoCount {
					break
				}
			}

			pool.Shutdown()

			duration := time.Since(start)
			b.ReportMetric(duration.Seconds(), "total_time")
		}
	})
}

// BenchmarkGitOperationScaling tests how git operations scale with worker count
func BenchmarkGitOperationScaling(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16}
	operationCount := 50

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				start := time.Now()

				pool := worker.NewPool(workers, operationCount)
				pool.Start(context.Background())

				// Create git operation tasks
				var tasks []worker.Task
				for j := 0; j < operationCount; j++ {
					task := &gitTask{
						name: fmt.Sprintf("git_op_%d", j),
						operation: func(ctx context.Context) error {
							repoPath, client, cleanup := setupTestRepo(ctx, b)
							defer cleanup()

							// Create a file using testutil
							filePaths := testutil.CreateBenchmarkFiles(b, repoPath, 1)
							fileName := filepath.Base(filePaths[0])

							// Add and get status
							if err := client.Add(ctx, repoPath, fileName); err != nil {
								return err
							}

							_, err := client.GetCurrentBranch(ctx, repoPath)
							return err
						},
					}
					tasks = append(tasks, task)
				}

				if err := pool.SubmitBatch(tasks); err != nil {
					b.Fatalf("Failed to submit batch: %v", err)
				}

				completed := 0
				for range pool.Results() {
					completed++
					if completed >= operationCount {
						break
					}
				}

				pool.Shutdown()

				duration := time.Since(start)
				throughput := float64(operationCount) / duration.Seconds()
				b.ReportMetric(throughput, "ops/sec")
			})
		})
	}
}
