package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
	"github.com/stretchr/testify/mock"
)

func BenchmarkGitCommand_Simple(b *testing.B) {
	mockClient := &MockClient{}
	ctx := context.Background()

	// Setup mock expectations
	mockClient.On("GetCurrentBranch", mock.Anything, mock.Anything).Return("main", nil)

	benchmark.WithMemoryTracking(b, func() {
		_, _ = mockClient.GetCurrentBranch(ctx, "/tmp/repo")
	})
}

func BenchmarkGitCommand_WithOutput(b *testing.B) {
	sizes := benchmark.Sizes()

	for _, size := range sizes {
		b.Run(size.Name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			// Generate different sized diff outputs
			diffOutput := benchmark.GenerateGitDiff(getDiffFileCount(size.Size), 50)
			mockClient.On("Diff", mock.Anything, mock.Anything, mock.Anything).Return(diffOutput, nil)

			benchmark.WithMemoryTracking(b, func() {
				_, _ = mockClient.Diff(ctx, "/tmp/repo", false)
			})
		})
	}
}

func BenchmarkClone_Scenarios(b *testing.B) {
	scenarios := []struct {
		name string
		url  string
	}{
		{"Small_Repo", "https://github.com/octocat/Hello-World.git"},
		{"Medium_Repo", "https://github.com/user/medium-project.git"},
		{"Large_Repo", "https://github.com/user/large-project.git"},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			// Mock successful clone with varying delays to simulate repo sizes
			mockClient.On("Clone", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			benchmark.WithMemoryTracking(b, func() {
				tmpDir := benchmark.SetupBenchmarkRepo(b)
				_ = mockClient.Clone(ctx, scenario.url, tmpDir)
			})
		})
	}
}

func BenchmarkAdd_FileCount(b *testing.B) {
	fileCounts := []int{1, 10, 100, 1000}

	for _, count := range fileCounts {
		b.Run(fmt.Sprintf("Files_%d", count), func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()
			tmpDir := benchmark.SetupBenchmarkRepo(b)

			// Create test files using shared utility
			filePaths := benchmark.SetupBenchmarkFiles(b, tmpDir, count)
			files := make([]string, len(filePaths))
			for i, path := range filePaths {
				files[i] = filepath.Base(path)
			}

			// Mock add operation
			mockClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			benchmark.WithMemoryTracking(b, func() {
				_ = mockClient.Add(ctx, tmpDir, files...)
			})
		})
	}
}

func BenchmarkDiff_Sizes(b *testing.B) {
	sizes := benchmark.StandardSizes()

	for _, size := range sizes {
		b.Run(size.Name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			// Generate realistic diff output based on file count
			diffOutput := benchmark.GenerateGitDiff(size.FileCount/10, 50)
			mockClient.On("Diff", mock.Anything, mock.Anything, mock.Anything).Return(diffOutput, nil)

			benchmark.WithMemoryTracking(b, func() {
				_, _ = mockClient.Diff(ctx, "/tmp/repo", false)
			})
		})
	}
}

func BenchmarkBranch_Operations(b *testing.B) {
	operations := []struct {
		name string
		fn   func(client Client, ctx context.Context, repo string) error
	}{
		{"CreateBranch", func(client Client, ctx context.Context, repo string) error {
			return client.CreateBranch(ctx, repo, "feature-branch")
		}},
		{"Checkout", func(client Client, ctx context.Context, repo string) error {
			return client.Checkout(ctx, repo, "main")
		}},
		{"GetCurrentBranch", func(client Client, ctx context.Context, repo string) error {
			_, err := client.GetCurrentBranch(ctx, repo)
			return err
		}},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			// Setup appropriate mock expectations
			setupMockForOperation(mockClient, op.name)

			benchmark.WithMemoryTracking(b, func() {
				_ = op.fn(mockClient, ctx, "/tmp/repo")
			})
		})
	}
}

func BenchmarkCommit_MessageSizes(b *testing.B) {
	messageSizes := []struct {
		name    string
		message string
	}{
		{"Short", "Fix bug"},
		{"Medium", "Implement new feature with comprehensive error handling"},
		{"Long", strings.Repeat("This is a very detailed commit message that explains everything about the changes made. ", 10)},
	}

	for _, msgSize := range messageSizes {
		b.Run(msgSize.name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			mockClient.On("Commit", mock.Anything, mock.Anything, mock.Anything).Return(nil)

			benchmark.WithMemoryTracking(b, func() {
				_ = mockClient.Commit(ctx, "/tmp/repo", msgSize.message)
			})
		})
	}
}

func BenchmarkPush_Scenarios(b *testing.B) {
	scenarios := []struct {
		name  string
		force bool
	}{
		{"Normal", false},
		{"Force", true},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			mockClient.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			benchmark.WithMemoryTracking(b, func() {
				_ = mockClient.Push(ctx, "/tmp/repo", "origin", "main", scenario.force)
			})
		})
	}
}

func BenchmarkGetRemoteURL_Multiple(b *testing.B) {
	remotes := []string{"origin", "upstream", "fork"}

	for _, remote := range remotes {
		b.Run(remote, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			expectedURL := fmt.Sprintf("https://github.com/user/repo-%s.git", remote)
			mockClient.On("GetRemoteURL", mock.Anything, mock.Anything, mock.Anything).Return(expectedURL, nil)

			benchmark.WithMemoryTracking(b, func() {
				_, _ = mockClient.GetRemoteURL(ctx, "/tmp/repo", remote)
			})
		})
	}
}

func BenchmarkGitWorkflow_Complete(b *testing.B) {
	// Benchmark a complete git workflow
	b.Run("CompleteWorkflow", func(b *testing.B) {
		mockClient := &MockClient{}
		ctx := context.Background()

		// Setup all mock expectations for a complete workflow
		mockClient.On("CreateBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Commit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockClient.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		benchmark.WithMemoryTracking(b, func() {
			repo := "/tmp/repo"
			branch := "feature-branch"

			// Complete workflow
			_ = mockClient.CreateBranch(ctx, repo, branch)
			_ = mockClient.Add(ctx, repo, ".")
			_ = mockClient.Commit(ctx, repo, "Add new feature")
			_ = mockClient.Push(ctx, repo, "origin", branch, false)
		})
	})
}

func BenchmarkMemoryUsage_GitOperations(b *testing.B) {
	operations := []struct {
		name string
		fn   func()
	}{
		{"CreateMockClient", func() {
			_ = &MockClient{}
		}},
		{"LargeDiffParsing", func() {
			diffOutput := benchmark.GenerateGitDiff(100, 1000)
			_ = len(diffOutput) // Simulate processing
		}},
		{"MultipleFileAdd", func() {
			files := make([]string, 1000)
			for i := range files {
				files[i] = fmt.Sprintf("file%d.txt", i)
			}
			_ = files
		}},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, op.name, op.fn)
		})
	}
}

// Helper functions

func getDiffFileCount(sizeCategory string) int {
	switch sizeCategory {
	case "small":
		return 1
	case "medium":
		return 5
	case "large":
		return 20
	default:
		return 50
	}
}

func setupMockForOperation(mockClient *MockClient, operation string) {
	switch operation {
	case "CreateBranch":
		mockClient.On("CreateBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	case "Checkout":
		mockClient.On("Checkout", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	case "GetCurrentBranch":
		mockClient.On("GetCurrentBranch", mock.Anything, mock.Anything).Return("main", nil)
	}
}
