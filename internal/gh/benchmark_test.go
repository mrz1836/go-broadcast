package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
	"github.com/stretchr/testify/mock"
)

// isMainBranch checks if a branch name is one of the configured main branches
func isMainBranch(branchName string) bool {
	mainBranches := os.Getenv("MAIN_BRANCHES")
	if mainBranches == "" {
		mainBranches = "master,main"
	}

	branches := strings.Split(mainBranches, ",")
	for _, branch := range branches {
		if strings.TrimSpace(branch) == branchName {
			return true
		}
	}

	return false
}

func BenchmarkGHCommand_Simple(b *testing.B) {
	mockClient := &MockClient{}
	ctx := context.Background()

	// Mock simple branch retrieval
	branch := &Branch{
		Name:      "master",
		Protected: true,
		Commit: struct {
			SHA string `json:"sha"`
			URL string `json:"url"`
		}{
			SHA: "abc123def456",
			URL: "https://api.github.com/repos/user/repo/commits/abc123def456",
		},
	}
	mockClient.On("GetBranch", mock.Anything, mock.Anything, mock.Anything).Return(branch, nil)

	benchmark.WithMemoryTracking(b, func() {
		_, _ = mockClient.GetBranch(ctx, "user/repo", "master")
	})
}

func BenchmarkListBranches_Sizes(b *testing.B) {
	sizes := []struct {
		name  string
		count int
	}{
		{"Small", 5},
		{"Medium", 50},
		{"Large", 200},
		{"XLarge", 1000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			// Generate test branches
			branches := generateTestBranches(size.count)
			mockClient.On("ListBranches", mock.Anything, mock.Anything).Return(branches, nil)

			benchmark.WithMemoryTracking(b, func() {
				_, _ = mockClient.ListBranches(ctx, "user/repo")
			})
		})
	}
}

func BenchmarkParseJSON_Sizes(b *testing.B) {
	sizes := []struct {
		name      string
		itemCount int
	}{
		{"Small", 10},
		{"Medium", 100},
		{"Large", 1000},
		{"XLarge", 5000},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			// Generate JSON response data
			data := benchmark.GenerateJSONResponse(size.itemCount)

			benchmark.WithMemoryTracking(b, func() {
				var result []interface{}
				_ = json.Unmarshal(data, &result)
			})
		})
	}
}

func BenchmarkDecodeBase64_Sizes(b *testing.B) {
	sizes := benchmark.Sizes()

	for _, size := range sizes {
		b.Run(size.Name, func(b *testing.B) {
			// Generate base64 content of different sizes
			content := benchmark.GenerateBase64Content(getFileSizeBytes(size.Size))

			mockClient := &MockClient{}
			ctx := context.Background()

			fileContent := &FileContent{
				Path:    "test/file.txt",
				Content: []byte(content),
				SHA:     "abc123",
			}
			mockClient.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fileContent, nil)

			benchmark.WithMemoryTracking(b, func() {
				_, _ = mockClient.GetFile(ctx, "user/repo", "test/file.txt", "master")
			})
		})
	}
}

func BenchmarkPROperations_Scenarios(b *testing.B) {
	operations := []struct {
		name string
		fn   func(client Client, ctx context.Context) error
	}{
		{"CreatePR", func(client Client, ctx context.Context) error {
			req := PRRequest{
				Title: "Add new feature",
				Body:  "This PR adds a new feature with comprehensive tests",
				Head:  "feature-branch",
				Base:  "master",
			}
			_, err := client.CreatePR(ctx, "user/repo", req)
			return err
		}},
		{"GetPR", func(client Client, ctx context.Context) error {
			_, err := client.GetPR(ctx, "user/repo", 123)
			return err
		}},
		{"ListPRs_Open", func(client Client, ctx context.Context) error {
			_, err := client.ListPRs(ctx, "user/repo", "open")
			return err
		}},
		{"ListPRs_All", func(client Client, ctx context.Context) error {
			_, err := client.ListPRs(ctx, "user/repo", "all")
			return err
		}},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			setupMockForPROperation(mockClient, op.name)

			benchmark.WithMemoryTracking(b, func() {
				_ = op.fn(mockClient, ctx)
			})
		})
	}
}

func BenchmarkConcurrentAPICalls(b *testing.B) {
	concurrencyLevels := []int{1, 5, 10, 20}

	for _, level := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrent_%d", level), func(b *testing.B) {
			mockClient := &MockClient{}
			branch := generateTestBranch("master")
			mockClient.On("GetBranch", mock.Anything, mock.Anything, mock.Anything).Return(branch, nil)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				for pb.Next() {
					_, _ = mockClient.GetBranch(ctx, "user/repo", "master")
				}
			})
		})
	}
}

func BenchmarkFileOperations_Sizes(b *testing.B) {
	fileSizes := []struct {
		name string
		size string
	}{
		{"Small_1KB", "small"},
		{"Medium_100KB", "medium"},
		{"Large_1MB", "large"},
		{"XLarge_10MB", "xlarge"},
	}

	for _, fileSize := range fileSizes {
		b.Run(fileSize.name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			// Generate file content
			contentBytes := benchmark.GenerateTestData(fileSize.size)
			fileContent := &FileContent{
				Path:    "test/large-file.txt",
				Content: contentBytes,
				SHA:     "def456",
			}
			mockClient.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fileContent, nil)

			benchmark.WithMemoryTracking(b, func() {
				_, _ = mockClient.GetFile(ctx, "user/repo", "test/large-file.txt", "master")
			})
		})
	}
}

func BenchmarkCommitOperations(b *testing.B) {
	scenarios := []struct {
		name        string
		commitCount int
	}{
		{"Single", 1},
		{"Multiple_10", 10},
		{"Multiple_100", 100},
		{"Batch_1000", 1000},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			mockClient := &MockClient{}
			ctx := context.Background()

			// Generate test commits
			commits := generateTestCommits(scenario.commitCount)
			for _, commit := range commits {
				mockClient.On("GetCommit", mock.Anything, mock.Anything, commit.SHA).Return(commit, nil)
			}

			benchmark.WithMemoryTracking(b, func() {
				for _, commit := range commits {
					_, _ = mockClient.GetCommit(ctx, "user/repo", commit.SHA)
				}
			})
		})
	}
}

func BenchmarkJSONSerialization_ComplexStructures(b *testing.B) {
	structures := []struct {
		name string
		data interface{}
	}{
		{"Branch", generateTestBranch("master")},
		{"PR_Simple", generateTestPR(1, false)},
		{"PR_Complex", generateTestPR(1, true)},
		{"Commit", generateTestCommit("abc123")},
		{"FileContent", &FileContent{
			Path:    "test/file.txt",
			Content: benchmark.GenerateTestData("medium"),
			SHA:     "def456",
		}},
	}

	for _, structure := range structures {
		b.Run(structure.name+"_Marshal", func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				_, _ = json.Marshal(structure.data)
			})
		})

		b.Run(structure.name+"_Unmarshal", func(b *testing.B) {
			data, _ := json.Marshal(structure.data)
			benchmark.WithMemoryTracking(b, func() {
				switch structure.name {
				case "Branch":
					var branch Branch
					_ = json.Unmarshal(data, &branch)
				case "PR_Simple", "PR_Complex":
					var pr PR
					_ = json.Unmarshal(data, &pr)
				case "Commit":
					var commit Commit
					_ = json.Unmarshal(data, &commit)
				case "FileContent":
					var fileContent FileContent
					_ = json.Unmarshal(data, &fileContent)
				}
			})
		})
	}
}

func BenchmarkMemoryUsage_GitHubOperations(b *testing.B) {
	operations := []struct {
		name string
		fn   func()
	}{
		{"CreateMockClient", func() {
			_ = &MockClient{}
		}},
		{"LargeBranchList", func() {
			branches := generateTestBranches(1000)
			_ = len(branches)
		}},
		{"LargePRList", func() {
			prs := generateTestPRs(500)
			_ = len(prs)
		}},
		{"FileContentProcessing", func() {
			content := benchmark.GenerateTestData("large")
			fileContent := &FileContent{
				Path:    "large-file.txt",
				Content: content,
				SHA:     "abc123",
			}
			_ = len(fileContent.Content)
		}},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, op.name, op.fn)
		})
	}
}

// Helper functions for generating test data

func generateTestBranches(count int) []Branch {
	branches := make([]Branch, count)
	for i := 0; i < count; i++ {
		branches[i] = *generateTestBranch(fmt.Sprintf("branch-%d", i))
	}
	return branches
}

func generateTestBranch(name string) *Branch {
	return &Branch{
		Name:      name,
		Protected: isMainBranch(name),
		Commit: struct {
			SHA string `json:"sha"`
			URL string `json:"url"`
		}{
			SHA: fmt.Sprintf("abc123def456%s", name),
			URL: fmt.Sprintf("https://api.github.com/repos/user/repo/commits/abc123def456%s", name),
		},
	}
}

func generateTestPRs(count int) []PR {
	prs := make([]PR, count)
	for i := 0; i < count; i++ {
		prs[i] = *generateTestPR(i+1, i%3 == 0) // Every 3rd PR has labels
	}
	return prs
}

func generateTestPR(number int, withLabels bool) *PR {
	now := time.Now()
	pr := &PR{
		Number: number,
		State:  "open",
		Title:  fmt.Sprintf("PR #%d: Add feature", number),
		Body:   fmt.Sprintf("This is the body of PR #%d with detailed description", number),
		Head: struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		}{
			Ref: fmt.Sprintf("feature-branch-%d", number),
			SHA: fmt.Sprintf("head-sha-%d", number),
		},
		Base: struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		}{
			Ref: "master",
			SHA: "base-sha-main",
		},
		User: struct {
			Login string `json:"login"`
		}{
			Login: fmt.Sprintf("user-%d", number),
		},
		CreatedAt: now.Add(-time.Duration(number) * time.Hour),
		UpdatedAt: now.Add(-time.Duration(number/2) * time.Hour),
	}

	if withLabels {
		pr.Labels = []struct {
			Name string `json:"name"`
		}{
			{Name: "enhancement"},
			{Name: "bug"},
			{Name: "documentation"},
		}
	}

	return pr
}

func generateTestCommits(count int) []*Commit {
	commits := make([]*Commit, count)
	for i := 0; i < count; i++ {
		commits[i] = generateTestCommit(fmt.Sprintf("commit-sha-%d", i))
	}
	return commits
}

func generateTestCommit(sha string) *Commit {
	now := time.Now()
	return &Commit{
		SHA: sha,
		Commit: struct {
			Message string `json:"message"`
			Author  struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"author"`
			Committer struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"committer"`
		}{
			Message: fmt.Sprintf("Commit message for %s", sha),
			Author: struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			}{
				Name:  "Test Author",
				Email: "author@example.com",
				Date:  now,
			},
			Committer: struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			}{
				Name:  "Test Committer",
				Email: "committer@example.com",
				Date:  now,
			},
		},
		Parents: []struct {
			SHA string `json:"sha"`
		}{
			{SHA: "parent-sha-1"},
		},
	}
}

func getFileSizeBytes(sizeCategory string) int {
	switch sizeCategory {
	case "small":
		return 1024 // 1KB
	case "medium":
		return 1024 * 100 // 100KB
	case "large":
		return 1024 * 1024 // 1MB
	default:
		return 1024 * 1024 * 10 // 10MB
	}
}

func setupMockForPROperation(mockClient *MockClient, operation string) {
	switch operation {
	case "CreatePR":
		pr := generateTestPR(1, false)
		mockClient.On("CreatePR", mock.Anything, mock.Anything, mock.Anything).Return(pr, nil)
	case "GetPR":
		pr := generateTestPR(123, true)
		mockClient.On("GetPR", mock.Anything, mock.Anything, mock.Anything).Return(pr, nil)
	case "ListPRs_Open":
		prs := generateTestPRs(10)
		mockClient.On("ListPRs", mock.Anything, mock.Anything, "open").Return(prs, nil)
	case "ListPRs_All":
		prs := generateTestPRs(25)
		mockClient.On("ListPRs", mock.Anything, mock.Anything, "all").Return(prs, nil)
	}
}
