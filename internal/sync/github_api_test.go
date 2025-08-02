package sync

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Test error variables
var (
	ErrNetworkTimeout    = errors.New("network timeout")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrCommitNotFound    = errors.New("commit not found")
	ErrTreeNotFound      = errors.New("tree not found")
)

// Test constants for common string literals
const (
	ownerRepoConst = "owner/repo"
	treeHashConst  = "abc123"
	mainBranchRef  = "main"
)

// GitHubAPISuite provides a test suite for GitHub Tree API integration
type GitHubAPISuite struct {
	suite.Suite

	mockClient *gh.MockClient
	api        *GitHubAPI
	logger     *logrus.Logger
}

// SetupTest sets up each test with fresh mocks and API instance
func (suite *GitHubAPISuite) SetupTest() {
	suite.mockClient = &gh.MockClient{}
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Create API with short TTL for testing
	opts := GitHubAPIOptions{
		CacheTTL:       100 * time.Millisecond,
		MaxCacheSize:   10,
		MaxRetries:     2,
		BaseRetryDelay: 10 * time.Millisecond,
	}
	suite.api = NewGitHubAPIWithOptions(suite.mockClient, suite.logger, opts)
}

// TearDownTest cleans up after each test
func (suite *GitHubAPISuite) TearDownTest() {
	suite.api.Close()
	suite.mockClient.AssertExpectations(suite.T())
}

// TestNewGitHubAPI tests constructor functions
func (suite *GitHubAPISuite) TestNewGitHubAPI() {
	// Test default constructor
	api := NewGitHubAPI(suite.mockClient, suite.logger)
	suite.NotNil(api)
	suite.Equal(5*time.Minute, api.cacheTTL)
	suite.Equal(3, api.maxRetries)
	api.Close()

	// Test with custom options
	opts := GitHubAPIOptions{
		CacheTTL:       10 * time.Minute,
		MaxCacheSize:   500,
		MaxRetries:     5,
		BaseRetryDelay: 2 * time.Second,
	}
	api2 := NewGitHubAPIWithOptions(suite.mockClient, suite.logger, opts)
	suite.NotNil(api2)
	suite.Equal(10*time.Minute, api2.cacheTTL)
	suite.Equal(5, api2.maxRetries)
	api2.Close()
}

// TestGetTree tests tree fetching functionality
func (suite *GitHubAPISuite) TestGetTree() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	// Mock commit response
	commit := &gh.Commit{
		SHA: treeSHA,
	}

	// Mock git tree response
	gitTree := &gh.GitTree{
		SHA: treeSHA,
		Tree: []gh.GitTreeNode{
			{Path: "README.md", Type: "blob", SHA: "file1", Mode: "100644"},
			{Path: "src/main.go", Type: "blob", SHA: "file2", Mode: "100644"},
			{Path: "src/util/helper.go", Type: "blob", SHA: "file3", Mode: "100644"},
			{Path: "src", Type: "tree", SHA: "dir1", Mode: "040000"},
			{Path: "src/util", Type: "tree", SHA: "dir2", Mode: "040000"},
		},
		Truncated: false,
	}

	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil)
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil)

	// Test successful tree fetch
	treeMap, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)
	suite.Require().NotNil(treeMap)

	// Verify tree structure
	suite.Equal(treeSHA, treeMap.sha)
	suite.Len(treeMap.files, 3)
	suite.Len(treeMap.directories, 2)

	// Verify files are correctly indexed
	suite.True(treeMap.HasFile("README.md"))
	suite.True(treeMap.HasFile("src/main.go"))
	suite.True(treeMap.HasFile("src/util/helper.go"))
	suite.False(treeMap.HasFile("nonexistent.txt"))

	// Verify directories are correctly indexed
	suite.True(treeMap.HasDirectory("src"))
	suite.True(treeMap.HasDirectory("src/util"))
	suite.False(treeMap.HasDirectory("nonexistent"))

	// Test file retrieval
	file, exists := treeMap.GetFile("src/main.go")
	suite.True(exists)
	suite.Equal("src/main.go", file.Path)
	suite.Equal("blob", file.Type)

	// Verify stats
	stats := treeMap.GetStats()
	suite.Equal(3, stats.TotalFiles)
	suite.Equal(2, stats.TotalDirectories)
	suite.Equal(2, stats.MaxDepth) // src/util/helper.go has depth 2
	suite.Equal(treeSHA, stats.TreeSHA)
}

// TestGetTreeWithCache tests caching behavior
func (suite *GitHubAPISuite) TestGetTreeWithCache() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{
		SHA:  treeSHA,
		Tree: []gh.GitTreeNode{{Path: "README.md", Type: "blob", SHA: "file1"}},
	}

	// Set up mocks for first call only
	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil).Once()
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil).Once()

	// First call should hit the API
	treeMap1, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)

	// Second call should hit the cache
	treeMap2, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)

	// Verify both return the same data
	suite.Equal(treeMap1.sha, treeMap2.sha)
	suite.Len(treeMap1.files, len(treeMap2.files))

	// Verify cache stats
	hits, misses, size, hitRate := suite.api.GetCacheStats()
	suite.Equal(int64(1), hits)
	suite.Equal(int64(1), misses)
	suite.Equal(1, size)
	suite.InEpsilon(0.5, hitRate, 0.01)
}

// TestGetTreeCacheExpiration tests cache TTL behavior
func (suite *GitHubAPISuite) TestGetTreeCacheExpiration() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{
		SHA:  treeSHA,
		Tree: []gh.GitTreeNode{{Path: "README.md", Type: "blob", SHA: "file1"}},
	}

	// Set up mocks for two API calls
	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil).Twice()
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil).Twice()

	// First call
	_, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Second call should hit API again
	_, err = suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)

	// Verify both calls were made
	suite.mockClient.AssertExpectations(suite.T())
}

// TestBatchCheckFiles tests batch file checking functionality
func (suite *GitHubAPISuite) TestBatchCheckFiles() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{
		SHA: treeSHA,
		Tree: []gh.GitTreeNode{
			{Path: "README.md", Type: "blob", SHA: "file1"},
			{Path: "src/main.go", Type: "blob", SHA: "file2"},
			{Path: "docs/guide.md", Type: "blob", SHA: "file3"},
		},
	}

	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil)
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil)

	filePaths := []string{
		"README.md",
		"src/main.go",
		"nonexistent.txt",
		"docs/guide.md",
		"missing/file.go",
	}

	results, err := suite.api.BatchCheckFiles(context.Background(), repo, ref, filePaths)
	suite.Require().NoError(err)
	suite.Require().Len(results, 5)

	// Verify results
	suite.True(results["README.md"])
	suite.True(results["src/main.go"])
	suite.False(results["nonexistent.txt"])
	suite.True(results["docs/guide.md"])
	suite.False(results["missing/file.go"])
}

// TestBatchCheckDirectories tests batch directory checking functionality
func (suite *GitHubAPISuite) TestBatchCheckDirectories() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{
		SHA: treeSHA,
		Tree: []gh.GitTreeNode{
			{Path: "src/main.go", Type: "blob", SHA: "file1"},
			{Path: "src/util/helper.go", Type: "blob", SHA: "file2"},
			{Path: "docs/README.md", Type: "blob", SHA: "file3"},
		},
	}

	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil)
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil)

	dirPaths := []string{
		"src",
		"src/util",
		"docs",
		"nonexistent",
		"",
	}

	results, err := suite.api.BatchCheckDirectories(context.Background(), repo, ref, dirPaths)
	suite.Require().NoError(err)
	suite.Require().Len(results, 5)

	// Verify results
	suite.True(results["src"])
	suite.True(results["src/util"])
	suite.True(results["docs"])
	suite.False(results["nonexistent"])
	suite.True(results[""]) // Root directory always exists
}

// TestGetFilesInDirectory tests directory file listing
func (suite *GitHubAPISuite) TestGetFilesInDirectory() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{
		SHA: treeSHA,
		Tree: []gh.GitTreeNode{
			{Path: "README.md", Type: "blob", SHA: "file1"},
			{Path: "src/main.go", Type: "blob", SHA: "file2"},
			{Path: "src/helper.go", Type: "blob", SHA: "file3"},
			{Path: "src/util/deep.go", Type: "blob", SHA: "file4"},
		},
	}

	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil)
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil)

	// Test root directory
	rootFiles, err := suite.api.GetFilesInDirectory(context.Background(), repo, ref, "")
	suite.Require().NoError(err)
	suite.Len(rootFiles, 1) // Only README.md
	suite.Equal("README.md", rootFiles[0].Path)

	// Test src directory
	srcFiles, err := suite.api.GetFilesInDirectory(context.Background(), repo, ref, "src")
	suite.Require().NoError(err)
	suite.Len(srcFiles, 2) // main.go and helper.go, not deep.go

	paths := make([]string, len(srcFiles))
	for i, file := range srcFiles {
		paths[i] = file.Path
	}
	suite.Contains(paths, "src/main.go")
	suite.Contains(paths, "src/helper.go")
	suite.NotContains(paths, "src/util/deep.go")
}

// TestRetryLogic tests retry behavior on failures
func (suite *GitHubAPISuite) TestRetryLogic() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{SHA: treeSHA, Tree: []gh.GitTreeNode{}}

	// First call succeeds for commit
	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil)

	// First tree call fails with retryable error
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).
		Return(nil, ErrNetworkTimeout).Once()

	// Second tree call succeeds
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).
		Return(gitTree, nil).Once()

	// Should succeed after retry
	treeMap, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)
	suite.NotNil(treeMap)

	// Verify retry stats
	treeFetches, cacheHits, cacheMisses, retries, rateLimits, avgTreeSize := suite.api.GetAPIStats()
	_ = treeFetches
	_ = cacheHits
	_ = cacheMisses
	_ = rateLimits
	_ = avgTreeSize
	suite.Equal(int64(1), retries)
}

// TestRateLimitHandling tests rate limit error handling
func (suite *GitHubAPISuite) TestRateLimitHandling() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}

	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil)

	// All calls fail with rate limit
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).
		Return(nil, ErrRateLimitExceeded).Times(3) // Initial + 2 retries

	// Should fail after all retries
	_, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "failed after 2 retries")

	// Verify rate limit stats
	treeFetches, cacheHits, cacheMisses, retries, rateLimits, avgTreeSize := suite.api.GetAPIStats()
	_ = treeFetches
	_ = cacheHits
	_ = cacheMisses
	_ = avgTreeSize
	suite.Equal(int64(2), retries)
	suite.Equal(int64(3), rateLimits) // Initial + 2 retries
}

// TestInvalidateCache tests cache invalidation
func (suite *GitHubAPISuite) TestInvalidateCache() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{SHA: treeSHA, Tree: []gh.GitTreeNode{}}

	// Set up for two API calls
	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil).Twice()
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil).Twice()

	// First call
	_, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)

	// Invalidate cache
	suite.api.InvalidateCache(repo, ref)

	// Second call should hit API again
	_, err = suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)

	// Verify both API calls were made
	suite.mockClient.AssertExpectations(suite.T())
}

// TestEmptyFilePaths tests batch operations with empty input
func (suite *GitHubAPISuite) TestEmptyFilePaths() {
	repo := ownerRepoConst
	ref := mainBranchRef

	// Should return empty results without API calls
	fileResults, err := suite.api.BatchCheckFiles(context.Background(), repo, ref, []string{})
	suite.Require().NoError(err)
	suite.Empty(fileResults)

	dirResults, err := suite.api.BatchCheckDirectories(context.Background(), repo, ref, []string{})
	suite.Require().NoError(err)
	suite.Empty(dirResults)

	// No API calls should have been made
	suite.mockClient.AssertExpectations(suite.T())
}

// TestTreeMapEdgeCases tests edge cases in TreeMap functionality
func (suite *GitHubAPISuite) TestTreeMapEdgeCases() {
	treeMap := &TreeMap{
		files: map[string]*GitTreeNode{
			"file.txt":            {Path: "file.txt", Type: "blob"},
			"dir/subfile.txt":     {Path: "dir/subfile.txt", Type: "blob"},
			"deep/nested/file.go": {Path: "deep/nested/file.go", Type: "blob"},
		},
		directories: map[string]bool{
			"dir":         true,
			"deep":        true,
			"deep/nested": true,
		},
		sha:       "test-sha",
		fetchedAt: time.Now(),
	}

	// Test file paths with leading/trailing slashes
	suite.True(treeMap.HasFile("/file.txt"))
	suite.True(treeMap.HasFile("file.txt"))
	suite.True(treeMap.HasFile("/dir/subfile.txt"))

	// Test directory paths with trailing slashes
	suite.True(treeMap.HasDirectory("dir/"))
	suite.True(treeMap.HasDirectory("dir"))
	suite.True(treeMap.HasDirectory("/deep/nested/"))
	suite.True(treeMap.HasDirectory("")) // Root always exists

	// Test GetFilesInDirectory with various path formats
	rootFiles := treeMap.GetFilesInDirectory("")
	suite.Len(rootFiles, 1) // Only file.txt

	dirFiles := treeMap.GetFilesInDirectory("dir")
	suite.Len(dirFiles, 1) // Only subfile.txt
	suite.Equal("dir/subfile.txt", dirFiles[0].Path)

	deepFiles := treeMap.GetFilesInDirectory("deep/nested/")
	suite.Len(deepFiles, 1) // Only file.go
	suite.Equal("deep/nested/file.go", deepFiles[0].Path)
}

// TestAPIStats tests API statistics tracking
func (suite *GitHubAPISuite) TestAPIStats() {
	// Initial stats should be zero
	treeFetches, cacheHits, cacheMisses, retries, rateLimits, avgTreeSize := suite.api.GetAPIStats()
	suite.Equal(int64(0), treeFetches)
	suite.Equal(int64(0), cacheHits)
	suite.Equal(int64(0), cacheMisses)
	suite.Equal(int64(0), retries)
	suite.Equal(int64(0), rateLimits)
	suite.Equal(int64(0), avgTreeSize)

	// After some operations, stats should be updated
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{
		SHA:  treeSHA,
		Tree: []gh.GitTreeNode{{Path: "file.txt", Type: "blob"}},
	}

	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil)
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil)

	_, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)

	// Check updated stats
	treeFetches, cacheHits, cacheMisses, _, _, avgTreeSize = suite.api.GetAPIStats()
	suite.Equal(int64(1), treeFetches)
	suite.Equal(int64(0), cacheHits)
	suite.Equal(int64(1), cacheMisses)
	suite.Positive(avgTreeSize)
}

// TestTruncatedTree tests handling of truncated trees
func (suite *GitHubAPISuite) TestTruncatedTree() {
	repo := ownerRepoConst
	ref := mainBranchRef
	treeSHA := treeHashConst

	commit := &gh.Commit{SHA: treeSHA}
	gitTree := &gh.GitTree{
		SHA:       treeSHA,
		Tree:      []gh.GitTreeNode{{Path: "file.txt", Type: "blob"}},
		Truncated: true, // Tree was truncated
	}

	suite.mockClient.On("GetCommit", context.Background(), repo, ref).Return(commit, nil)
	suite.mockClient.On("GetGitTree", context.Background(), repo, treeSHA, true).Return(gitTree, nil)

	// Should still succeed but log a warning
	treeMap, err := suite.api.GetTree(context.Background(), repo, ref)
	suite.Require().NoError(err)
	suite.NotNil(treeMap)
	suite.Equal(treeSHA, treeMap.sha)
}

// TestRunSuite runs the complete test suite
func TestGitHubAPISuite(t *testing.T) {
	suite.Run(t, new(GitHubAPISuite))
}

// TestTreeMapMethods tests TreeMap methods independently
func TestTreeMapMethods(t *testing.T) {
	// Create a sample tree map
	treeMap := &TreeMap{
		files: map[string]*GitTreeNode{
			"README.md":          {Path: "README.md", Type: "blob", SHA: "file1"},
			"src/main.go":        {Path: "src/main.go", Type: "blob", SHA: "file2"},
			"src/util/helper.go": {Path: "src/util/helper.go", Type: "blob", SHA: "file3"},
		},
		directories: map[string]bool{
			"src":      true,
			"src/util": true,
		},
		sha:       "tree-sha",
		fetchedAt: time.Now(),
	}

	t.Run("HasFile", func(t *testing.T) {
		assert.True(t, treeMap.HasFile("README.md"))
		assert.True(t, treeMap.HasFile("src/main.go"))
		assert.True(t, treeMap.HasFile("src/util/helper.go"))
		assert.False(t, treeMap.HasFile("nonexistent.txt"))

		// Test with leading slash
		assert.True(t, treeMap.HasFile("/README.md"))
		assert.True(t, treeMap.HasFile("/src/main.go"))
	})

	t.Run("HasDirectory", func(t *testing.T) {
		assert.True(t, treeMap.HasDirectory("src"))
		assert.True(t, treeMap.HasDirectory("src/util"))
		assert.False(t, treeMap.HasDirectory("nonexistent"))
		assert.True(t, treeMap.HasDirectory("")) // Root always exists

		// Test with trailing slash
		assert.True(t, treeMap.HasDirectory("src/"))
		assert.True(t, treeMap.HasDirectory("src/util/"))
	})

	t.Run("GetFile", func(t *testing.T) {
		file, exists := treeMap.GetFile("src/main.go")
		assert.True(t, exists)
		assert.NotNil(t, file)
		assert.Equal(t, "src/main.go", file.Path)
		assert.Equal(t, "blob", file.Type)
		assert.Equal(t, "file2", file.SHA)

		_, exists = treeMap.GetFile("nonexistent.txt")
		assert.False(t, exists)
	})

	t.Run("GetFilesInDirectory", func(t *testing.T) {
		// Root directory
		rootFiles := treeMap.GetFilesInDirectory("")
		assert.Len(t, rootFiles, 1)
		assert.Equal(t, "README.md", rootFiles[0].Path)

		// src directory
		srcFiles := treeMap.GetFilesInDirectory("src")
		assert.Len(t, srcFiles, 1) // Only main.go, not helper.go (which is in src/util)
		assert.Equal(t, "src/main.go", srcFiles[0].Path)

		// src/util directory
		utilFiles := treeMap.GetFilesInDirectory("src/util")
		assert.Len(t, utilFiles, 1)
		assert.Equal(t, "src/util/helper.go", utilFiles[0].Path)

		// Nonexistent directory
		emptyFiles := treeMap.GetFilesInDirectory("nonexistent")
		assert.Empty(t, emptyFiles)
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := treeMap.GetStats()
		assert.Equal(t, 3, stats.TotalFiles)
		assert.Equal(t, 2, stats.TotalDirectories)
		assert.Equal(t, 2, stats.MaxDepth) // src/util/helper.go has depth 2
		assert.Equal(t, "tree-sha", stats.TreeSHA)
		assert.False(t, stats.FetchedAt.IsZero())
	})
}

// TestErrorCases tests various error scenarios
func TestErrorCases(t *testing.T) {
	mockClient := &gh.MockClient{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ctx := context.Background()

	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	repo := ownerRepoConst
	ref := mainBranchRef

	t.Run("GetCommit error", func(t *testing.T) {
		mockClient.On("GetCommit", ctx, repo, ref).
			Return(nil, ErrCommitNotFound).Once()

		_, err := api.GetTree(ctx, repo, ref)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "get commit for ref")
	})

	t.Run("GetGitTree error", func(t *testing.T) {
		commit := &gh.Commit{SHA: treeHashConst}
		mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil).Once()
		mockClient.On("GetGitTree", ctx, repo, treeHashConst, true).
			Return(nil, ErrTreeNotFound).Once()

		_, err := api.GetTree(ctx, repo, ref)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fetch git tree")
	})

	mockClient.AssertExpectations(t)
}

// Benchmark tests for performance verification
func BenchmarkTreeMapOperations(b *testing.B) {
	// Create a large tree map for benchmarking
	treeMap := &TreeMap{
		files:       make(map[string]*GitTreeNode, 10000),
		directories: make(map[string]bool, 1000),
		sha:         "benchmark-sha",
		fetchedAt:   time.Now(),
	}

	// Populate with test data
	for i := 0; i < 10000; i++ {
		path := fmt.Sprintf("dir%d/subdir%d/file%d.go", i%100, i%10, i)
		treeMap.files[path] = &GitTreeNode{
			Path: path,
			Type: "blob",
			SHA:  fmt.Sprintf("sha%d", i),
		}
	}

	for i := 0; i < 1000; i++ {
		dirPath := fmt.Sprintf("dir%d", i%100)
		treeMap.directories[dirPath] = true
	}

	b.Run("HasFile", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			path := fmt.Sprintf("dir%d/subdir%d/file%d.go", i%100, i%10, i%10000)
			treeMap.HasFile(path)
		}
	})

	b.Run("HasDirectory", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dirPath := fmt.Sprintf("dir%d", i%100)
			treeMap.HasDirectory(dirPath)
		}
	})

	b.Run("GetFile", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			path := fmt.Sprintf("dir%d/subdir%d/file%d.go", i%100, i%10, i%10000)
			treeMap.GetFile(path)
		}
	})
}
