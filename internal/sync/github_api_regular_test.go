package sync

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// Static errors for testing
var (
	errAPIError                = errors.New("API error")
	errRateLimitExceeded       = errors.New("rate limit exceeded: 403")
	errRepositoryNotFound      = errors.New("repository not found: 404")
	errTimeoutError            = errors.New("timeout error")
	errRateLimitExceededSimple = errors.New("rate limit exceeded")
	errHTTP403Forbidden        = errors.New("HTTP 403 Forbidden")
	errXRateLimitRemaining     = errors.New("x-ratelimit-remaining: 0")
	errHTTP404NotFound         = errors.New("HTTP 404 Not Found")
	errConnectionRefused       = errors.New("connection refused")
	errRateLimitExceededUpper  = errors.New("RATE LIMIT EXCEEDED")
	errRequestTimeout          = errors.New("request timeout")
	errNetworkUnreachable      = errors.New("network unreachable")
	errHTTP502BadGateway       = errors.New("HTTP 502 Bad Gateway")
	errHTTP503ServiceUnavail   = errors.New("HTTP 503 Service Unavailable")
	errHTTP504GatewayTimeout   = errors.New("HTTP 504 Gateway Timeout")
	errTemporaryFailure        = errors.New("temporary failure")
	errHTTP401Unauthorized     = errors.New("HTTP 401 Unauthorized")
	errTimeoutErrorUpper       = errors.New("TIMEOUT ERROR")
)

// Test constants
const (
	testRepo = "owner/repo"
	testRef  = "main"
)

func TestTreeMap_HasFile(t *testing.T) {
	tm := &TreeMap{
		files: map[string]*gh.GitTreeNode{
			"README.md":            {Path: "README.md", Type: "blob"},
			"src/main.go":          {Path: "src/main.go", Type: "blob"},
			"docs/api.md":          {Path: "docs/api.md", Type: "blob"},
			"nested/deep/file.txt": {Path: "nested/deep/file.txt", Type: "blob"},
		},
		directories: make(map[string]bool),
	}

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"existing file in root", "README.md", true},
		{"existing file with leading slash", "/README.md", true},
		{"existing file in subdirectory", "src/main.go", true},
		{"nested file", "nested/deep/file.txt", true},
		{"non-existent file", "nonexistent.txt", false},
		{"directory path (not a file)", "src", false},
		{"partial path match", "README", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.HasFile(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTreeMap_HasDirectory(t *testing.T) {
	tm := &TreeMap{
		files: make(map[string]*gh.GitTreeNode),
		directories: map[string]bool{
			"src":         true,
			"docs":        true,
			"nested":      true,
			"nested/deep": true,
			"tests/unit":  true,
		},
	}

	tests := []struct {
		name     string
		dirPath  string
		expected bool
	}{
		{"existing directory", "src", true},
		{"nested directory", "nested/deep", true},
		{"directory with trailing slash", "src/", true},
		{"directory with leading slash", "/src", true},
		{"directory with both slashes", "/src/", true},
		{"root directory", "", true},
		{"root directory with slash", "/", true},
		{"non-existent directory", "nonexistent", false},
		{"partial match", "sr", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.HasDirectory(tt.dirPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTreeMap_GetFile(t *testing.T) {
	node1 := &gh.GitTreeNode{Path: "README.md", Type: "blob", SHA: "sha1"}
	node2 := &gh.GitTreeNode{Path: "src/main.go", Type: "blob", SHA: "sha2"}

	tm := &TreeMap{
		files: map[string]*gh.GitTreeNode{
			"README.md":   node1,
			"src/main.go": node2,
		},
		directories: make(map[string]bool),
	}

	tests := []struct {
		name           string
		filePath       string
		expectedNode   *gh.GitTreeNode
		expectedExists bool
	}{
		{"existing file", "README.md", node1, true},
		{"existing file with leading slash", "/src/main.go", node2, true},
		{"non-existent file", "nonexistent.txt", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, exists := tm.GetFile(tt.filePath)
			assert.Equal(t, tt.expectedExists, exists)
			assert.Equal(t, tt.expectedNode, node)
		})
	}
}

func TestTreeMap_GetFilesInDirectory(t *testing.T) {
	files := map[string]*gh.GitTreeNode{
		"README.md":               {Path: "README.md", Type: "blob"},
		"LICENSE":                 {Path: "LICENSE", Type: "blob"},
		"src/main.go":             {Path: "src/main.go", Type: "blob"},
		"src/config.go":           {Path: "src/config.go", Type: "blob"},
		"src/utils/helper.go":     {Path: "src/utils/helper.go", Type: "blob"},
		"docs/api.md":             {Path: "docs/api.md", Type: "blob"},
		"tests/unit/main_test.go": {Path: "tests/unit/main_test.go", Type: "blob"},
	}

	tm := &TreeMap{
		files:       files,
		directories: make(map[string]bool),
	}

	tests := []struct {
		name          string
		dirPath       string
		expectedPaths []string
	}{
		{
			name:          "root directory",
			dirPath:       "",
			expectedPaths: []string{"README.md", "LICENSE"},
		},
		{
			name:          "root directory with slash",
			dirPath:       "/",
			expectedPaths: []string{"README.md", "LICENSE"},
		},
		{
			name:          "src directory",
			dirPath:       "src",
			expectedPaths: []string{"src/main.go", "src/config.go"},
		},
		{
			name:          "src directory with trailing slash",
			dirPath:       "src/",
			expectedPaths: []string{"src/main.go", "src/config.go"},
		},
		{
			name:          "docs directory",
			dirPath:       "docs",
			expectedPaths: []string{"docs/api.md"},
		},
		{
			name:          "empty directory",
			dirPath:       "nonexistent",
			expectedPaths: []string{},
		},
		{
			name:          "nested directory",
			dirPath:       "tests/unit",
			expectedPaths: []string{"tests/unit/main_test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tm.GetFilesInDirectory(tt.dirPath)

			actualPaths := make([]string, len(result))
			for i, node := range result {
				actualPaths[i] = node.Path
			}

			assert.ElementsMatch(t, tt.expectedPaths, actualPaths)
		})
	}
}

func TestTreeMap_GetStats(t *testing.T) {
	now := time.Now()
	tm := &TreeMap{
		files: map[string]*gh.GitTreeNode{
			"file1.txt":                 {Path: "file1.txt"},                 // depth 0
			"dir1/file2.txt":            {Path: "dir1/file2.txt"},            // depth 1
			"dir1/dir2/file3.txt":       {Path: "dir1/dir2/file3.txt"},       // depth 2
			"very/deep/nested/file.txt": {Path: "very/deep/nested/file.txt"}, // depth 3
		},
		directories: map[string]bool{
			"dir1":             true,
			"dir1/dir2":        true,
			"very":             true,
			"very/deep":        true,
			"very/deep/nested": true,
		},
		sha:       "test-sha",
		fetchedAt: now,
	}

	stats := tm.GetStats()

	assert.Equal(t, 4, stats.TotalFiles)
	assert.Equal(t, 5, stats.TotalDirectories)
	assert.Equal(t, 3, stats.MaxDepth) // "very/deep/nested/file.txt" has 3 slashes
	assert.Equal(t, "test-sha", stats.TreeSHA)
	assert.Equal(t, now, stats.FetchedAt)
}

func TestAPIStats_GetStats(t *testing.T) {
	stats := &APIStats{}

	stats.TreeFetches.Store(10)
	stats.CacheHits.Store(20)
	stats.CacheMisses.Store(5)
	stats.TotalRetries.Store(3)
	stats.TotalRateLimit.Store(1)
	stats.AverageTreeSize.Store(100)

	treeFetches, cacheHits, cacheMisses, retries, rateLimits, avgTreeSize := stats.GetStats()

	assert.Equal(t, int64(10), treeFetches)
	assert.Equal(t, int64(20), cacheHits)
	assert.Equal(t, int64(5), cacheMisses)
	assert.Equal(t, int64(3), retries)
	assert.Equal(t, int64(1), rateLimits)
	assert.Equal(t, int64(100), avgTreeSize)
}

func TestNewGitHubAPI(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()

	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	assert.NotNil(t, api)
	assert.Equal(t, mockClient, api.client)
	assert.Equal(t, logger, api.logger)
	assert.NotNil(t, api.cache)
	assert.NotNil(t, api.stats)
}

func TestNewGitHubAPIWithOptions(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()

	opts := GitHubAPIOptions{
		CacheTTL:       10 * time.Minute,
		MaxCacheSize:   500,
		MaxRetries:     5,
		BaseRetryDelay: 2 * time.Second,
	}

	api := NewGitHubAPIWithOptions(mockClient, logger, opts)
	defer api.Close()

	assert.NotNil(t, api)
	assert.Equal(t, opts.CacheTTL, api.cacheTTL)
	assert.Equal(t, opts.MaxRetries, api.maxRetries)
	assert.Equal(t, opts.BaseRetryDelay, api.baseDelay)
}

func TestNewGitHubAPIWithOptions_Defaults(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()

	// Test with empty options (should use defaults)
	api := NewGitHubAPIWithOptions(mockClient, logger, GitHubAPIOptions{})
	defer api.Close()

	assert.Equal(t, 5*time.Minute, api.cacheTTL)
	assert.Equal(t, 3, api.maxRetries)
	assert.Equal(t, time.Second, api.baseDelay)
}

func TestGitHubAPI_GetTree_CacheHit(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Create a tree map to put in cache
	expectedTreeMap := &TreeMap{
		files: map[string]*gh.GitTreeNode{
			"README.md": {Path: "README.md", Type: "blob"},
		},
		directories: make(map[string]bool),
		sha:         "test-sha",
		fetchedAt:   time.Now(),
	}

	// Put in cache manually
	cacheKey := fmt.Sprintf("%s:%s", repo, ref)
	api.cache.Set(cacheKey, expectedTreeMap)

	// Call GetTree - should hit cache
	result, err := api.GetTree(ctx, repo, ref)

	require.NoError(t, err)
	assert.Equal(t, expectedTreeMap, result)

	// Verify cache statistics
	treeFetches, cacheHits, cacheMisses, _, _, lastError := api.GetAPIStats()
	_ = lastError                          // unused in this test
	assert.Equal(t, int64(0), treeFetches) // Should not fetch from API
	assert.Equal(t, int64(1), cacheHits)
	assert.Equal(t, int64(0), cacheMisses)
}

func TestGitHubAPI_GetTree_CacheMiss(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Mock the client responses
	commit := &gh.Commit{
		SHA: "commit-sha",
	}

	gitTree := &gh.GitTree{
		SHA: "tree-sha",
		Tree: []gh.GitTreeNode{
			{Path: "README.md", Type: "blob", SHA: "file-sha1"},
			{Path: "src/main.go", Type: "blob", SHA: "file-sha2"},
			{Path: "src", Type: "tree", SHA: "dir-sha1"},
		},
		Truncated: false,
	}

	mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil)
	mockClient.On("GetGitTree", ctx, repo, "commit-sha", true).Return(gitTree, nil)

	// Call GetTree
	result, err := api.GetTree(ctx, repo, ref)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "tree-sha", result.sha)
	assert.True(t, result.HasFile("README.md"))
	assert.True(t, result.HasFile("src/main.go"))
	assert.True(t, result.HasDirectory("src"))

	// Verify API statistics
	treeFetches, cacheHits, cacheMisses, _, _, lastError := api.GetAPIStats()
	_ = lastError // unused in this test
	assert.Equal(t, int64(1), treeFetches)
	assert.Equal(t, int64(0), cacheHits)
	assert.Equal(t, int64(1), cacheMisses)

	mockClient.AssertExpectations(t)
}

func TestGitHubAPI_GetTree_APIError(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Mock client to return an error
	expectedError := errAPIError
	mockClient.On("GetCommit", ctx, repo, ref).Return(nil, expectedError)

	// Call GetTree
	result, err := api.GetTree(ctx, repo, ref)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "API error")

	mockClient.AssertExpectations(t)
}

func TestGitHubAPI_BatchCheckFiles(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Test with empty file list
	result, err := api.BatchCheckFiles(ctx, repo, ref, []string{})
	require.NoError(t, err)
	assert.Empty(t, result)

	// Mock successful tree fetch
	commit := &gh.Commit{SHA: "commit-sha"}
	gitTree := &gh.GitTree{
		SHA: "tree-sha",
		Tree: []gh.GitTreeNode{
			{Path: "README.md", Type: "blob"},
			{Path: "src/main.go", Type: "blob"},
		},
	}

	mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil)
	mockClient.On("GetGitTree", ctx, repo, "commit-sha", true).Return(gitTree, nil)

	filePaths := []string{"README.md", "src/main.go", "nonexistent.txt"}
	result, err = api.BatchCheckFiles(ctx, repo, ref, filePaths)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.True(t, result["README.md"])
	assert.True(t, result["src/main.go"])
	assert.False(t, result["nonexistent.txt"])

	mockClient.AssertExpectations(t)
}

func TestGitHubAPI_BatchCheckDirectories(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Test with empty directory list
	result, err := api.BatchCheckDirectories(ctx, repo, ref, []string{})
	require.NoError(t, err)
	assert.Empty(t, result)

	// Mock successful tree fetch
	commit := &gh.Commit{SHA: "commit-sha"}
	gitTree := &gh.GitTree{
		SHA: "tree-sha",
		Tree: []gh.GitTreeNode{
			{Path: "src/main.go", Type: "blob"},
			{Path: "src", Type: "tree"},
			{Path: "docs", Type: "tree"},
		},
	}

	mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil)
	mockClient.On("GetGitTree", ctx, repo, "commit-sha", true).Return(gitTree, nil)

	dirPaths := []string{"src", "docs", "nonexistent"}
	result, err = api.BatchCheckDirectories(ctx, repo, ref, dirPaths)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.True(t, result["src"])
	assert.True(t, result["docs"])
	assert.False(t, result["nonexistent"])

	mockClient.AssertExpectations(t)
}

func TestGitHubAPI_GetFilesInDirectory(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef
	dirPath := "src"

	// Mock successful tree fetch
	commit := &gh.Commit{SHA: "commit-sha"}
	gitTree := &gh.GitTree{
		SHA: "tree-sha",
		Tree: []gh.GitTreeNode{
			{Path: "src/main.go", Type: "blob"},
			{Path: "src/config.go", Type: "blob"},
			{Path: "src/utils/helper.go", Type: "blob"}, // Should not be included (nested)
		},
	}

	mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil)
	mockClient.On("GetGitTree", ctx, repo, "commit-sha", true).Return(gitTree, nil)

	result, err := api.GetFilesInDirectory(ctx, repo, ref, dirPath)

	require.NoError(t, err)
	assert.Len(t, result, 2) // Only direct children, not nested

	paths := make([]string, len(result))
	for i, node := range result {
		paths[i] = node.Path
	}
	assert.ElementsMatch(t, []string{"src/main.go", "src/config.go"}, paths)

	mockClient.AssertExpectations(t)
}

func TestGitHubAPI_InvalidateCache(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	repo := testRepo
	ref := testRef
	cacheKey := fmt.Sprintf("%s:%s", repo, ref)

	// Put something in cache
	treeMap := &TreeMap{files: make(map[string]*gh.GitTreeNode)}
	api.cache.Set(cacheKey, treeMap)

	// Verify it's in cache
	_, exists := api.cache.Get(cacheKey)
	assert.True(t, exists)

	// Invalidate cache
	api.InvalidateCache(repo, ref)

	// Verify it's removed
	_, exists = api.cache.Get(cacheKey)
	assert.False(t, exists)
}

func TestGitHubAPI_GetCacheStats(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	hits, misses, size, hitRate := api.GetCacheStats()
	assert.GreaterOrEqual(t, hits, int64(0))
	assert.GreaterOrEqual(t, misses, int64(0))
	assert.GreaterOrEqual(t, size, 0)
	assert.GreaterOrEqual(t, hitRate, float64(0))
}

func TestGitHubAPI_Close(_ *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)

	// Should not panic
	api.Close()

	// Multiple calls should be safe
	api.Close()
}

func TestFetchTreeWithRetry_Success(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Mock successful response
	commit := &gh.Commit{SHA: "commit-sha"}
	gitTree := &gh.GitTree{
		SHA:  "tree-sha",
		Tree: []gh.GitTreeNode{},
	}

	mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil)
	mockClient.On("GetGitTree", ctx, repo, "commit-sha", true).Return(gitTree, nil)

	result, err := api.fetchTreeWithRetry(ctx, repo, ref)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "tree-sha", result.sha)

	mockClient.AssertExpectations(t)
}

func TestFetchTreeWithRetry_RateLimitRetry(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	// Set very short base delay for testing
	api := NewGitHubAPIWithOptions(mockClient, logger, GitHubAPIOptions{
		BaseRetryDelay: 1 * time.Millisecond,
		MaxRetries:     2,
	})
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Mock rate limit error first, then success
	rateLimitErr := errRateLimitExceeded
	commit := &gh.Commit{SHA: "commit-sha"}
	gitTree := &gh.GitTree{SHA: "tree-sha", Tree: []gh.GitTreeNode{}}

	mockClient.On("GetCommit", ctx, repo, ref).Return(nil, rateLimitErr).Once()
	mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil).Once()
	mockClient.On("GetGitTree", ctx, repo, "commit-sha", true).Return(gitTree, nil).Once()

	result, err := api.fetchTreeWithRetry(ctx, repo, ref)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify retry stats
	treeFetches, cacheHits, cacheMisses, retries, rateLimits, lastError := api.GetAPIStats()
	_ = treeFetches // unused in this test
	_ = cacheHits   // unused in this test
	_ = cacheMisses // unused in this test
	_ = lastError   // unused in this test
	assert.Equal(t, int64(1), retries)
	assert.Equal(t, int64(1), rateLimits)

	mockClient.AssertExpectations(t)
}

func TestFetchTreeWithRetry_NonRetryableError(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Mock non-retryable error (404)
	notFoundErr := errRepositoryNotFound
	mockClient.On("GetCommit", ctx, repo, ref).Return(nil, notFoundErr).Once()

	result, err := api.fetchTreeWithRetry(ctx, repo, ref)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "repository not found")

	mockClient.AssertExpectations(t)
}

func TestFetchTreeWithRetry_ContextCanceled(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPIWithOptions(mockClient, logger, GitHubAPIOptions{
		BaseRetryDelay: 100 * time.Millisecond, // Longer delay to test cancellation
		MaxRetries:     3,
	})
	defer api.Close()

	ctx, cancel := context.WithCancel(context.Background())
	repo := testRepo
	ref := testRef

	// Mock retryable error
	timeoutErr := errTimeoutError
	mockClient.On("GetCommit", ctx, repo, ref).Return(nil, timeoutErr).Once()

	// Cancel context during retry delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	result, err := api.fetchTreeWithRetry(ctx, repo, ref)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.Canceled, err)

	mockClient.AssertExpectations(t)
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"rate limit error", errRateLimitExceededSimple, true},
		{"403 error", errHTTP403Forbidden, true},
		{"x-ratelimit header", errXRateLimitRemaining, true},
		{"404 error", errHTTP404NotFound, false},
		{"network error", errConnectionRefused, false},
		{"uppercase rate limit", errRateLimitExceededUpper, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRateLimitError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitHubRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"timeout error", errRequestTimeout, true},
		{"connection error", errConnectionRefused, true},
		{"network error", errNetworkUnreachable, true},
		{"502 error", errHTTP502BadGateway, true},
		{"503 error", errHTTP503ServiceUnavail, true},
		{"504 error", errHTTP504GatewayTimeout, true},
		{"temporary error", errTemporaryFailure, true},
		{"404 error", errHTTP404NotFound, false},
		{"401 error", errHTTP401Unauthorized, false},
		{"uppercase errors", errTimeoutErrorUpper, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGitHubRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountTrue(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]bool
		expected int
	}{
		{"empty map", map[string]bool{}, 0},
		{"all true", map[string]bool{"a": true, "b": true, "c": true}, 3},
		{"all false", map[string]bool{"a": false, "b": false}, 0},
		{"mixed", map[string]bool{"a": true, "b": false, "c": true, "d": false}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countTrue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateAverageTreeSize(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	// First measurement
	api.updateAverageTreeSize(100)
	avg := api.stats.AverageTreeSize.Load()
	assert.Equal(t, int64(100), avg)

	// Second measurement (should be weighted average)
	api.updateAverageTreeSize(200)
	avg = api.stats.AverageTreeSize.Load()
	// Expected: (100*9 + 200) / 10 = 110
	assert.Equal(t, int64(110), avg)

	// Third measurement
	api.updateAverageTreeSize(50)
	avg = api.stats.AverageTreeSize.Load()
	// Expected: (110*9 + 50) / 10 = 104
	assert.Equal(t, int64(104), avg)
}

func TestFetchTree_TruncatedTree(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Mock response with truncated tree
	commit := &gh.Commit{SHA: "commit-sha"}
	gitTree := &gh.GitTree{
		SHA: "tree-sha",
		Tree: []gh.GitTreeNode{
			{Path: "file1.txt", Type: "blob"},
		},
		Truncated: true, // Important: tree was truncated
	}

	mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil)
	mockClient.On("GetGitTree", ctx, repo, "commit-sha", true).Return(gitTree, nil)

	result, err := api.fetchTree(ctx, repo, ref)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasFile("file1.txt"))

	mockClient.AssertExpectations(t)
}

func TestFetchTree_DirectoryStructure(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()
	api := NewGitHubAPI(mockClient, logger)
	defer api.Close()

	ctx := context.Background()
	repo := testRepo
	ref := testRef

	// Mock response with files that create directory structure
	commit := &gh.Commit{SHA: "commit-sha"}
	gitTree := &gh.GitTree{
		SHA: "tree-sha",
		Tree: []gh.GitTreeNode{
			{Path: "src/main.go", Type: "blob"},
			{Path: "src/utils/helper.go", Type: "blob"},
			{Path: "docs/api.md", Type: "blob"},
			{Path: "src", Type: "tree"},
			{Path: "docs", Type: "tree"},
			{Path: "src/utils", Type: "tree"},
		},
	}

	mockClient.On("GetCommit", ctx, repo, ref).Return(commit, nil)
	mockClient.On("GetGitTree", ctx, repo, "commit-sha", true).Return(gitTree, nil)

	result, err := api.fetchTree(ctx, repo, ref)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify files exist
	assert.True(t, result.HasFile("src/main.go"))
	assert.True(t, result.HasFile("src/utils/helper.go"))
	assert.True(t, result.HasFile("docs/api.md"))

	// Verify directories exist (both explicit and inferred from file paths)
	assert.True(t, result.HasDirectory("src"))
	assert.True(t, result.HasDirectory("docs"))
	assert.True(t, result.HasDirectory("src/utils"))

	mockClient.AssertExpectations(t)
}

func TestTreeAPIClient_Interface(_ *testing.T) {
	// This test ensures GitHubAPI implements the TreeAPIClient interface
	mockClient := gh.NewMockClient()
	logger := logrus.New()

	var _ TreeAPIClient = NewGitHubAPI(mockClient, logger)
}
