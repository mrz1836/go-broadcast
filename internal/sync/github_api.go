// Package sync provides GitHub Tree API integration for bulk file operations
package sync

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/cache"
	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

// GitTreeNode represents a single node in a Git tree
type GitTreeNode = gh.GitTreeNode

// GitTree represents a Git tree structure
type GitTree = gh.GitTree

// TreeMap provides O(1) file existence checks
type TreeMap struct {
	files       map[string]*GitTreeNode // Full file paths
	directories map[string]bool         // Directory paths
	sha         string                  // Tree SHA for cache key
	fetchedAt   time.Time               // When the tree was fetched
}

// HasFile checks if a file exists in the tree
func (tm *TreeMap) HasFile(filePath string) bool {
	_, exists := tm.files[strings.TrimPrefix(filePath, "/")]
	return exists
}

// HasDirectory checks if a directory exists in the tree
func (tm *TreeMap) HasDirectory(dirPath string) bool {
	cleanPath := strings.TrimPrefix(strings.TrimSuffix(dirPath, "/"), "/")
	if cleanPath == "" {
		return true // Root directory always exists
	}
	return tm.directories[cleanPath]
}

// GetFile returns the tree node for a specific file
func (tm *TreeMap) GetFile(filePath string) (*GitTreeNode, bool) {
	node, exists := tm.files[strings.TrimPrefix(filePath, "/")]
	return node, exists
}

// GetFilesInDirectory returns all files in a specific directory
func (tm *TreeMap) GetFilesInDirectory(dirPath string) []*GitTreeNode {
	cleanDir := strings.TrimPrefix(strings.TrimSuffix(dirPath, "/"), "/")
	if cleanDir != "" {
		cleanDir += "/"
	}

	var files []*GitTreeNode
	for filePath, node := range tm.files {
		if strings.HasPrefix(filePath, cleanDir) {
			// Only include direct children, not nested files
			relativePath := strings.TrimPrefix(filePath, cleanDir)
			if !strings.Contains(relativePath, "/") {
				files = append(files, node)
			}
		}
	}
	return files
}

// GetAllFilesInDirectoryRecursively returns all files in a directory and its subdirectories
func (tm *TreeMap) GetAllFilesInDirectoryRecursively(dirPath string) []*GitTreeNode {
	cleanDir := strings.TrimPrefix(strings.TrimSuffix(dirPath, "/"), "/")
	if cleanDir != "" {
		cleanDir += "/"
	}

	var files []*GitTreeNode
	for filePath, node := range tm.files {
		if strings.HasPrefix(filePath, cleanDir) {
			// Include all files under the directory, including nested files
			files = append(files, node)
		}
	}
	return files
}

// TreeStats provides statistics about the tree
type TreeStats struct {
	TotalFiles       int
	TotalDirectories int
	MaxDepth         int
	TreeSHA          string
	FetchedAt        time.Time
}

// GetStats returns statistics about the tree
func (tm *TreeMap) GetStats() TreeStats {
	maxDepth := 0
	for filePath := range tm.files {
		depth := strings.Count(filePath, "/")
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return TreeStats{
		TotalFiles:       len(tm.files),
		TotalDirectories: len(tm.directories),
		MaxDepth:         maxDepth,
		TreeSHA:          tm.sha,
		FetchedAt:        tm.fetchedAt,
	}
}

// APIStats tracks GitHub API call statistics
type APIStats struct {
	TreeFetches     atomic.Int64
	CacheHits       atomic.Int64
	CacheMisses     atomic.Int64
	TotalRetries    atomic.Int64
	TotalRateLimit  atomic.Int64
	AverageTreeSize atomic.Int64
}

// GetStats returns current API statistics
func (stats *APIStats) GetStats() (treeFetches, cacheHits, cacheMisses, retries, rateLimits, avgTreeSize int64) {
	return stats.TreeFetches.Load(),
		stats.CacheHits.Load(),
		stats.CacheMisses.Load(),
		stats.TotalRetries.Load(),
		stats.TotalRateLimit.Load(),
		stats.AverageTreeSize.Load()
}

// GitHubAPI provides GitHub Tree API integration with caching and bulk operations
type GitHubAPI struct {
	client     gh.Client
	cache      *cache.TTLCache
	cacheTTL   time.Duration
	maxRetries int
	baseDelay  time.Duration
	logger     *logrus.Logger
	stats      *APIStats
}

// NewGitHubAPI creates a new GitHub API client with tree caching support
func NewGitHubAPI(client gh.Client, logger *logrus.Logger) *GitHubAPI {
	return NewGitHubAPIWithOptions(client, logger, GitHubAPIOptions{})
}

// GitHubAPIOptions configures the GitHub API client
type GitHubAPIOptions struct {
	CacheTTL       time.Duration // Default: 5 minutes
	MaxCacheSize   int           // Default: 1000 repositories
	MaxRetries     int           // Default: 3
	BaseRetryDelay time.Duration // Default: 1 second
}

// NewGitHubAPIWithOptions creates a new GitHub API client with custom options
func NewGitHubAPIWithOptions(client gh.Client, logger *logrus.Logger, opts GitHubAPIOptions) *GitHubAPI {
	// Set defaults
	if opts.CacheTTL == 0 {
		opts.CacheTTL = 5 * time.Minute
	}
	if opts.MaxCacheSize == 0 {
		opts.MaxCacheSize = 1000
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	if opts.BaseRetryDelay == 0 {
		opts.BaseRetryDelay = time.Second
	}

	return &GitHubAPI{
		client:     client,
		cache:      cache.NewTTLCache(opts.CacheTTL, opts.MaxCacheSize),
		cacheTTL:   opts.CacheTTL,
		maxRetries: opts.MaxRetries,
		baseDelay:  opts.BaseRetryDelay,
		logger:     logger,
		stats:      &APIStats{},
	}
}

// GetTree fetches the Git tree for a repository using GitHub's tree API
func (api *GitHubAPI) GetTree(ctx context.Context, repo, ref string) (*TreeMap, error) {
	log := api.logger.WithFields(logrus.Fields{
		"component": "github_tree_api",
		"repo":      repo,
		"ref":       ref,
	})

	// Try cache first
	cacheKey := fmt.Sprintf("%s:%s", repo, ref)
	if cached, ok := api.cache.Get(cacheKey); ok {
		log.Debug("Tree found in cache")
		api.stats.CacheHits.Add(1)
		return cached.(*TreeMap), nil
	}

	api.stats.CacheMisses.Add(1)
	log.Debug("Tree not in cache, fetching from GitHub")

	// Fetch tree with retries
	treeMap, err := api.fetchTreeWithRetry(ctx, repo, ref)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, fmt.Sprintf("fetch tree for %s@%s", repo, ref))
	}

	// Cache the result
	api.cache.Set(cacheKey, treeMap)
	api.stats.TreeFetches.Add(1)

	log.WithFields(logrus.Fields{
		"files":       len(treeMap.files),
		"directories": len(treeMap.directories),
		"tree_sha":    treeMap.sha,
	}).Info("Successfully fetched and cached Git tree")

	return treeMap, nil
}

// BatchCheckFiles checks existence of multiple files in O(1) time using the tree
func (api *GitHubAPI) BatchCheckFiles(ctx context.Context, repo, ref string, filePaths []string) (map[string]bool, error) {
	if len(filePaths) == 0 {
		return make(map[string]bool), nil
	}

	log := api.logger.WithFields(logrus.Fields{
		"component":  "github_batch_check",
		"repo":       repo,
		"ref":        ref,
		"file_count": len(filePaths),
	})

	// Get tree map
	treeMap, err := api.GetTree(ctx, repo, ref)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get tree for batch file check")
	}

	// Check all files in O(1) per file
	results := make(map[string]bool, len(filePaths))
	for _, filePath := range filePaths {
		results[filePath] = treeMap.HasFile(filePath)
	}

	log.WithField("found_files", countTrue(results)).Debug("Batch file check completed")
	return results, nil
}

// BatchCheckDirectories checks existence of multiple directories
func (api *GitHubAPI) BatchCheckDirectories(ctx context.Context, repo, ref string, dirPaths []string) (map[string]bool, error) {
	if len(dirPaths) == 0 {
		return make(map[string]bool), nil
	}

	log := api.logger.WithFields(logrus.Fields{
		"component": "github_batch_dir_check",
		"repo":      repo,
		"ref":       ref,
		"dir_count": len(dirPaths),
	})

	// Get tree map
	treeMap, err := api.GetTree(ctx, repo, ref)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get tree for batch directory check")
	}

	// Check all directories
	results := make(map[string]bool, len(dirPaths))
	for _, dirPath := range dirPaths {
		results[dirPath] = treeMap.HasDirectory(dirPath)
	}

	log.WithField("found_dirs", countTrue(results)).Debug("Batch directory check completed")
	return results, nil
}

// GetFilesInDirectory returns all files in a specific directory
func (api *GitHubAPI) GetFilesInDirectory(ctx context.Context, repo, ref, dirPath string) ([]*GitTreeNode, error) {
	treeMap, err := api.GetTree(ctx, repo, ref)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get tree for directory listing")
	}

	return treeMap.GetFilesInDirectory(dirPath), nil
}

// InvalidateCache removes a repository's tree from cache
func (api *GitHubAPI) InvalidateCache(repo, ref string) {
	cacheKey := fmt.Sprintf("%s:%s", repo, ref)
	api.cache.Delete(cacheKey)
	api.logger.WithFields(logrus.Fields{
		"repo": repo,
		"ref":  ref,
	}).Debug("Invalidated tree cache")
}

// GetCacheStats returns cache statistics
func (api *GitHubAPI) GetCacheStats() (hits, misses int64, size int, hitRate float64) {
	return api.cache.Stats()
}

// GetAPIStats returns API call statistics
func (api *GitHubAPI) GetAPIStats() (treeFetches, cacheHits, cacheMisses, retries, rateLimits, avgTreeSize int64) {
	return api.stats.GetStats()
}

// Close closes the cache and cleanup resources
func (api *GitHubAPI) Close() {
	api.cache.Close()
}

// fetchTreeWithRetry fetches the tree with exponential backoff retry logic
func (api *GitHubAPI) fetchTreeWithRetry(ctx context.Context, repo, ref string) (*TreeMap, error) {
	var lastErr error

	for attempt := 0; attempt <= api.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			delay := time.Duration(float64(api.baseDelay) * math.Pow(2, float64(attempt-1)))
			// Use crypto/rand for secure jitter calculation
			jitterMax := big.NewInt(int64(delay / 4))
			jitterBig, err := rand.Int(rand.Reader, jitterMax)
			if err != nil {
				// Fallback to 10% of delay if crypto/rand fails
				jitterBig = big.NewInt(int64(delay / 10))
			}
			jitter := time.Duration(jitterBig.Int64())
			delay += jitter

			api.logger.WithFields(logrus.Fields{
				"repo":    repo,
				"ref":     ref,
				"attempt": attempt,
				"delay":   delay,
			}).Debug("Retrying tree fetch after delay")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			api.stats.TotalRetries.Add(1)
		}

		treeMap, err := api.fetchTree(ctx, repo, ref)
		if err == nil {
			// Update average tree size
			api.updateAverageTreeSize(len(treeMap.files))
			return treeMap, nil
		}

		lastErr = err

		// Check if it's a rate limit error
		if isRateLimitError(err) {
			api.stats.TotalRateLimit.Add(1)
			api.logger.WithField("repo", repo).Warn("GitHub API rate limit hit, will retry")
			continue
		}

		// For non-recoverable errors, don't retry
		if !isGitHubRetryableError(err) {
			break
		}
	}

	return nil, appErrors.WrapWithContext(lastErr, fmt.Sprintf("failed after %d retries", api.maxRetries))
}

// fetchTree performs the actual tree fetch from GitHub API
func (api *GitHubAPI) fetchTree(ctx context.Context, repo, ref string) (*TreeMap, error) {
	// First, get the commit to resolve the ref to a SHA
	commit, err := api.client.GetCommit(ctx, repo, ref)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get commit for ref")
	}

	treeSHA := commit.SHA // Use commit SHA as tree SHA initially

	// Fetch tree recursively using the GitHub Git Tree API
	gitTree, err := api.client.GetGitTree(ctx, repo, treeSHA, true)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "fetch git tree")
	}

	// Build tree map
	treeMap := &TreeMap{
		files:       make(map[string]*GitTreeNode),
		directories: make(map[string]bool),
		sha:         gitTree.SHA,
		fetchedAt:   time.Now(),
	}

	// Process all tree nodes
	for i := range gitTree.Tree {
		node := &gitTree.Tree[i]

		switch node.Type {
		case "blob":
			// It's a file
			treeMap.files[node.Path] = node

			// Also mark all parent directories as existing
			dir := filepath.Dir(node.Path)
			for dir != "." && dir != "/" {
				dir = strings.TrimSuffix(dir, "/")
				treeMap.directories[dir] = true
				dir = filepath.Dir(dir)
			}
		case "tree":
			// It's a directory
			treeMap.directories[node.Path] = true
		}
	}

	// Handle pagination if tree was truncated
	if gitTree.Truncated {
		api.logger.WithField("repo", repo).Warn("Git tree was truncated, some files may not be cached")
	}

	return treeMap, nil
}

// updateAverageTreeSize updates the rolling average of tree sizes
func (api *GitHubAPI) updateAverageTreeSize(newSize int) {
	// Simple rolling average implementation
	for {
		current := api.stats.AverageTreeSize.Load()
		var newAvg int64
		if current == 0 {
			// First measurement
			newAvg = int64(newSize)
		} else {
			// Weight new size as 10% of the average
			newAvg = (current*9 + int64(newSize)) / 10
		}
		if api.stats.AverageTreeSize.CompareAndSwap(current, newAvg) {
			break
		}
	}
}

// isRateLimitError checks if an error is due to GitHub API rate limiting
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "x-ratelimit")
}

// isGitHubRetryableError checks if an error is retryable
func isGitHubRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())

	// Network errors, timeouts, and 5xx errors are retryable
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "temporary")
}

// countTrue counts the number of true values in a boolean map
func countTrue(m map[string]bool) int {
	count := 0
	for _, v := range m {
		if v {
			count++
		}
	}
	return count
}

// TreeAPIClient provides tree-specific operations for GitHub repositories
type TreeAPIClient interface {
	// GetTree fetches the complete file tree for a repository
	GetTree(ctx context.Context, repo, ref string) (*TreeMap, error)

	// BatchCheckFiles efficiently checks multiple file paths
	BatchCheckFiles(ctx context.Context, repo, ref string, filePaths []string) (map[string]bool, error)

	// BatchCheckDirectories efficiently checks multiple directory paths
	BatchCheckDirectories(ctx context.Context, repo, ref string, dirPaths []string) (map[string]bool, error)

	// GetFilesInDirectory returns all files in a directory
	GetFilesInDirectory(ctx context.Context, repo, ref, dirPath string) ([]*GitTreeNode, error)

	// InvalidateCache removes cached tree data
	InvalidateCache(repo, ref string)

	// GetCacheStats returns cache performance metrics
	GetCacheStats() (hits, misses int64, size int, hitRate float64)

	// GetAPIStats returns API call statistics
	GetAPIStats() (treeFetches, cacheHits, cacheMisses, retries, rateLimits, avgTreeSize int64)

	// Close cleanup resources
	Close()
}

// Ensure GitHubAPI implements TreeAPIClient
var _ TreeAPIClient = (*GitHubAPI)(nil)
