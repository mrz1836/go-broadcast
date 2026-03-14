package sync

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/metrics"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// Static error variables
var (
	ErrSourceDirectoryNotExist      = errors.New("source directory does not exist")
	ErrAllDirectoryProcessingFailed = errors.New("all directory processing failed")
	ErrSourceDirectoryEmpty         = errors.New("source directory cannot be empty")
	ErrDestinationDirectoryEmpty    = errors.New("destination directory cannot be empty")
	ErrEmptyExclusionPattern        = errors.New("empty exclusion pattern not allowed")
	ErrUnsupportedModuleType        = errors.New("unsupported module type")
)

// DirectoryProcessor handles concurrent directory processing with worker pools
type DirectoryProcessor struct {
	exclusionEngine      *ExclusionEngine
	progressManager      *DirectoryProgressManager
	workerCount          int
	logger               *logrus.Entry
	moduleDetector       *ModuleDetector
	moduleResolver       *ModuleResolver
	moduleCache          *ModuleCache
	moduleSourceResolver *ModuleSourceResolver
	gitClient            git.Client
	sourceRepoURL        string
	tempDir              string
	moduleUpdates        []ModuleUpdateInfo // Tracks module updates for go.mod
	moduleUpdatesMu      sync.Mutex         // Protects moduleUpdates access
}

// ModuleSyncResult contains the result of module-aware sync preparation
type ModuleSyncResult struct {
	// SourcePath is the path to use for syncing (may differ from original if versioned)
	SourcePath string

	// ResolvedVersion is the version that was resolved from the constraint
	ResolvedVersion string

	// ModuleInfo contains the detected module information
	ModuleInfo *ModuleInfo

	// Cleanup should be called after sync completes to remove temporary directories
	Cleanup func()
}

// ModuleUpdateInfo tracks information needed to update go.mod references
type ModuleUpdateInfo struct {
	// DestPath is the destination path where go.mod should be updated
	DestPath string

	// ModuleName is the name of the module to update
	ModuleName string

	// Version is the version to set in go.mod
	Version string
}

// DirectoryProcessorOptions contains optional configuration for DirectoryProcessor
type DirectoryProcessorOptions struct {
	// GitClient is the git client for cloning repositories at specific tags
	GitClient git.Client

	// SourceRepoURL is the URL of the source repository (e.g., "https://github.com/org/repo")
	SourceRepoURL string

	// TempDir is the base directory for temporary clones
	TempDir string

	// ClearModuleCache clears the module version cache before processing
	ClearModuleCache bool
}

// NewDirectoryProcessor creates a new directory processor
func NewDirectoryProcessor(logger *logrus.Entry, workerCount int, opts *DirectoryProcessorOptions) *DirectoryProcessor {
	if workerCount <= 0 {
		workerCount = 10 // Default worker count
	}

	// Create module components with default settings
	moduleCache := NewModuleCache(5*time.Minute, logger.Logger)
	moduleDetector := NewModuleDetector(logger.Logger)
	moduleResolver := NewModuleResolver(logger.Logger, moduleCache)

	// Clear cache if requested
	if opts != nil && opts.ClearModuleCache {
		moduleCache.Clear()
		logger.Info("Cleared module version cache")
	}

	dp := &DirectoryProcessor{
		progressManager: NewDirectoryProgressManager(logger),
		workerCount:     workerCount,
		logger:          logger,
		moduleDetector:  moduleDetector,
		moduleResolver:  moduleResolver,
		moduleCache:     moduleCache,
	}

	// Set up module source resolver if git client is provided
	if opts != nil && opts.GitClient != nil {
		dp.gitClient = opts.GitClient
		dp.sourceRepoURL = opts.SourceRepoURL
		dp.tempDir = opts.TempDir
		dp.moduleSourceResolver = NewModuleSourceResolver(opts.GitClient, logger.Logger, moduleCache)
	}

	return dp
}

// Close shuts down the directory processor and cleans up resources
func (dp *DirectoryProcessor) Close() {
	if dp.moduleCache != nil {
		dp.moduleCache.Close()
	}
}

// processDirectories processes directory mappings for the RepositorySync
func (rs *RepositorySync) processDirectories(ctx context.Context) ([]FileChange, error) {
	if len(rs.target.Directories) == 0 {
		rs.logger.Debug("No directories configured for sync")
		return nil, nil
	}

	// Check for context cancellation early
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context canceled before directory processing: %w", err)
	}

	processTimer := metrics.StartTimer(ctx, rs.logger, "directory_processing").
		AddField("directory_count", len(rs.target.Directories))

	rs.logger.WithField("directory_count", len(rs.target.Directories)).Info("Processing directories")

	// Construct source repo URL for module-aware sync
	sourceRepoURL := ""
	if rs.sourceState != nil && rs.sourceState.Repo != "" {
		sourceRepoURL = fmt.Sprintf("https://github.com/%s", rs.sourceState.Repo)
	}

	// Build directory processor options
	var opts *DirectoryProcessorOptions
	if rs.engine != nil {
		clearCache := false
		if rs.engine.Options() != nil {
			clearCache = rs.engine.Options().ClearModuleCache
		}
		opts = &DirectoryProcessorOptions{
			GitClient:        rs.engine.GitClient(),
			SourceRepoURL:    sourceRepoURL,
			TempDir:          rs.tempDir,
			ClearModuleCache: clearCache,
		}
	}

	// Create directory processor with module-aware sync support
	processor := NewDirectoryProcessor(rs.logger, 10, opts)
	defer processor.Close()

	sourcePath := filepath.Join(rs.tempDir, "source")

	// Verify source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrSourceDirectoryNotExist, sourcePath)
	}

	var allChanges []FileChange
	var processingErrors []error

	// Process each directory mapping
	for _, dirMapping := range rs.target.Directories {
		// Check for context cancellation during processing
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context canceled during directory processing: %w", err)
		}

		changes, err := processor.ProcessDirectoryMapping(ctx, sourcePath, dirMapping, rs.target, rs.sourceState, rs.engine)
		if err != nil {
			// Log error and collect for potential failure decision
			rs.logger.WithError(err).WithField("directory", dirMapping.Src).Error("Failed to process directory")
			processingErrors = append(processingErrors, err)
			continue
		}
		allChanges = append(allChanges, changes...)
	}

	// If all directories failed, return an error
	if len(processingErrors) > 0 && len(allChanges) == 0 {
		return nil, fmt.Errorf("%w: %d errors occurred", ErrAllDirectoryProcessingFailed, len(processingErrors))
	}

	processTimer.AddField("total_changes", len(allChanges)).Stop()

	rs.logger.WithField("total_changes", len(allChanges)).Info("Directory processing completed")
	return allChanges, nil
}

// ProcessDirectoryMapping processes a single directory mapping
func (dp *DirectoryProcessor) ProcessDirectoryMapping(ctx context.Context, sourcePath string, dirMapping config.DirectoryMapping, target config.TargetConfig, sourceState *state.SourceState, engine *Engine) ([]FileChange, error) {
	logger := dp.logger.WithFields(logrus.Fields{
		"src_dir":  dirMapping.Src,
		"dest_dir": dirMapping.Dest,
		"delete":   dirMapping.Delete,
	})

	logger.Info("Processing directory mapping")

	// Handle directory deletion
	if dirMapping.Delete {
		return dp.processDirectoryDeletion(ctx, dirMapping, target, engine, logger)
	}

	// Create exclusion engine with directory-specific patterns
	dp.exclusionEngine = NewExclusionEngineWithIncludes(dirMapping.Exclude, dirMapping.IncludeOnly)

	// Build full source directory path
	fullSourceDir := filepath.Join(sourcePath, dirMapping.Src)

	// Check if source directory exists
	if _, err := os.Stat(fullSourceDir); os.IsNotExist(err) {
		logger.Warn("Source directory not found, skipping")
		return nil, internalerrors.ErrFileNotFound
	}

	// Track the effective source directory (may change if module versioning is used)
	effectiveSourceDir := fullSourceDir

	// Handle module-aware sync if configured
	if dirMapping.Module != nil {
		result, err := dp.handleModuleSync(ctx, fullSourceDir, dirMapping.Src, dirMapping.Module, logger)
		if err != nil {
			logger.WithError(err).Warn("Module sync handling failed, continuing with standard sync")
			// Continue with standard sync if module handling fails
		} else if result != nil {
			// Handle update_refs feature if enabled
			if dirMapping.Module != nil && dirMapping.Module.UpdateRefs && result.ModuleInfo != nil {
				// Build path to go.mod in destination directory
				goModPath := filepath.Join(dirMapping.Dest, "go.mod")

				// Store module update info to be applied during commit phase
				dp.moduleUpdatesMu.Lock()
				dp.moduleUpdates = append(dp.moduleUpdates, ModuleUpdateInfo{
					DestPath:   goModPath,
					ModuleName: result.ModuleInfo.Name,
					Version:    result.ResolvedVersion,
				})
				dp.moduleUpdatesMu.Unlock()

				logger.WithFields(logrus.Fields{
					"module":    result.ModuleInfo.Name,
					"version":   result.ResolvedVersion,
					"dest_path": goModPath,
				}).Info("Module reference update will be applied during commit")
			}

			if result.SourcePath != "" && result.SourcePath != fullSourceDir {
				effectiveSourceDir = result.SourcePath
				logger.WithFields(logrus.Fields{
					"module_type":      dirMapping.Module.Type,
					"resolved_version": result.ResolvedVersion,
					"versioned_source": effectiveSourceDir,
				}).Info("Using versioned module source for sync")
			} else {
				logger.WithField("module_type", dirMapping.Module.Type).Debug("Module sync configuration applied")
			}
			// Defer cleanup of versioned source
			defer result.Cleanup()
		}
	}

	// Discover files in the directory (use effectiveSourceDir which may be versioned)
	files, err := dp.discoverFiles(ctx, effectiveSourceDir, dirMapping)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files in directory %s: %w", dirMapping.Src, err)
	}

	if len(files) == 0 {
		logger.Info("No files found in directory")
		return nil, nil
	}

	// Create progress reporter
	progressReporter := dp.progressManager.GetReporter(dirMapping.Src, 50)
	progressReporter.Start(len(files))

	// Convert discovered files to processing jobs
	jobs := dp.createFileJobs(files, dirMapping)

	// Process files using batch processor
	batchProcessor := NewBatchProcessor(engine, target, sourceState, logger, dp.workerCount)

	// Use progress wrapper for batch processing
	progressWrapper := NewBatchProgressWrapper(progressReporter)
	changes, err := batchProcessor.ProcessFilesWithProgress(ctx, sourcePath, jobs, progressWrapper)
	if err != nil {
		progressReporter.Complete()
		return nil, fmt.Errorf("failed to process files in directory %s: %w", dirMapping.Src, err)
	}

	// Complete progress reporting
	directoryMetrics := progressReporter.Complete()

	logger.WithFields(logrus.Fields{
		"files_discovered":          directoryMetrics.FilesDiscovered,
		"files_processed":           directoryMetrics.FilesProcessed,
		"files_excluded":            directoryMetrics.FilesExcluded,
		"binary_files_skipped":      directoryMetrics.BinaryFilesSkipped,
		"binary_files_size_bytes":   directoryMetrics.BinaryFilesSize,
		"transform_errors":          directoryMetrics.TransformErrors,
		"transform_successes":       directoryMetrics.TransformSuccesses,
		"avg_transform_duration_ms": progressReporter.GetAverageTransformDuration().Milliseconds(),
		"changes":                   len(changes),
	}).Info("Directory mapping processed successfully")

	return changes, nil
}

// No need for sourceState interface since we're using state.SourceState directly

// DiscoveredFile represents a file discovered during directory traversal
type DiscoveredFile struct {
	RelativePath string // Path relative to the source directory
	FullPath     string // Full filesystem path
	Size         int64  // File size in bytes
	IsDir        bool   // Whether this is a directory
}

// discoverFiles walks the directory tree and discovers files to process
func (dp *DirectoryProcessor) discoverFiles(ctx context.Context, sourceDir string, dirMapping config.DirectoryMapping) ([]DiscoveredFile, error) {
	var files []DiscoveredFile
	var mu sync.Mutex

	// Determine if hidden files should be included
	includeHidden := true
	if dirMapping.IncludeHidden != nil {
		includeHidden = *dirMapping.IncludeHidden
	}

	// Walk the directory tree
	err := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		// Handle context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			dp.logger.WithError(err).WithField("path", path).Warn("Error walking directory path")
			return nil // Continue walking other paths
		}

		// Calculate relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			dp.logger.WithError(err).WithField("path", path).Warn("Failed to calculate relative path")
			return nil
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Check for hidden files/directories
		if !includeHidden && dp.isHidden(relPath) {
			if d.IsDir() {
				return filepath.SkipDir // Skip entire hidden directory
			}
			return nil // Skip hidden file
		}

		// Early exclusion check for directories to avoid walking excluded trees
		if d.IsDir() {
			if dp.exclusionEngine != nil && dp.exclusionEngine.IsDirectoryExcluded(relPath) {
				dp.logger.WithField("directory", relPath).Debug("Directory excluded by patterns")
				return filepath.SkipDir
			}
			// Record directory traversal for metrics
			dp.progressManager.GetReporter(dirMapping.Src, 50).RecordDirectoryWalked()
			return nil // Continue walking this directory
		}

		// Check if file should be excluded
		if dp.exclusionEngine != nil && dp.exclusionEngine.IsExcluded(relPath) {
			dp.logger.WithField("file", relPath).Debug("File excluded by patterns")
			dp.progressManager.GetReporter(dirMapping.Src, 50).RecordFileExcluded()
			return nil
		}

		// Get file info for size
		info, err := d.Info()
		if err != nil {
			dp.logger.WithError(err).WithField("file", relPath).Warn("Failed to get file info")
			return nil
		}

		// Add to discovered files
		mu.Lock()
		files = append(files, DiscoveredFile{
			RelativePath: relPath,
			FullPath:     path,
			Size:         info.Size(),
			IsDir:        d.IsDir(),
		})
		mu.Unlock()

		// Update progress metrics
		progressReporter := dp.progressManager.GetReporter(dirMapping.Src, 50)
		progressReporter.RecordFileDiscovered()
		progressReporter.AddTotalSize(info.Size())

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", sourceDir, err)
	}

	dp.logger.WithFields(logrus.Fields{
		"directory":        sourceDir,
		"files_discovered": len(files),
	}).Debug("Directory discovery completed")

	return files, nil
}

// createFileJobs converts discovered files into processing jobs with directory-specific metadata
func (dp *DirectoryProcessor) createFileJobs(files []DiscoveredFile, dirMapping config.DirectoryMapping) []FileJob {
	// First, count non-directory files to get total count
	totalFiles := 0
	for _, file := range files {
		if !file.IsDir {
			totalFiles++
		}
	}

	jobs := make([]FileJob, 0, totalFiles)
	fileIndex := 0

	for _, file := range files {
		// Skip directories
		if file.IsDir {
			continue
		}

		// Determine destination path
		destPath := dp.calculateDestinationPath(file.RelativePath, dirMapping)

		// Create directory-aware job using the new helper function
		job := NewDirectoryFileJob(
			filepath.Join(dirMapping.Src, file.RelativePath),
			destPath,
			dirMapping.Transform,
			&dirMapping, // Pass reference to directory mapping
			file.RelativePath,
			fileIndex,
			totalFiles,
		)

		jobs = append(jobs, job)
		fileIndex++
	}

	dp.logger.WithFields(logrus.Fields{
		"source_directory": dirMapping.Src,
		"jobs_created":     len(jobs),
		"total_files":      totalFiles,
	}).Debug("Directory file jobs created with enhanced metadata")

	return jobs
}

// calculateDestinationPath determines the destination path for a file
func (dp *DirectoryProcessor) calculateDestinationPath(relativePath string, dirMapping config.DirectoryMapping) string {
	// Check if structure should be preserved
	preserveStructure := true
	if dirMapping.PreserveStructure != nil {
		preserveStructure = *dirMapping.PreserveStructure
	}

	if preserveStructure {
		// Preserve directory structure
		return filepath.Join(dirMapping.Dest, relativePath)
	}

	// Flatten structure - put all files directly in destination directory
	filename := filepath.Base(relativePath)
	return filepath.Join(dirMapping.Dest, filename)
}

// isHidden checks if a path represents a hidden file or directory
func (dp *DirectoryProcessor) isHidden(path string) bool {
	// Split path into components and check each one
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}
	return false
}

// ValidateDirectoryMapping validates a directory mapping configuration
func ValidateDirectoryMapping(dirMapping config.DirectoryMapping) error {
	if dirMapping.Src == "" {
		return ErrSourceDirectoryEmpty
	}

	if dirMapping.Dest == "" {
		return ErrDestinationDirectoryEmpty
	}

	// Validate exclusion patterns
	for _, pattern := range dirMapping.Exclude {
		if pattern == "" {
			return ErrEmptyExclusionPattern
		}
	}

	return nil
}

// GetDirectoryStats returns statistics about directory processing
func (dp *DirectoryProcessor) GetDirectoryStats() map[string]DirectoryMetrics {
	return dp.progressManager.GetAllMetrics()
}

// CompleteAllDirectories completes all active directory processing and returns final metrics
func (dp *DirectoryProcessor) CompleteAllDirectories() map[string]DirectoryMetrics {
	return dp.progressManager.CompleteAll()
}

// SetWorkerCount updates the worker count for the processor
func (dp *DirectoryProcessor) SetWorkerCount(count int) {
	if count > 0 {
		dp.workerCount = count
	}
}

// GetWorkerCount returns the current worker count
func (dp *DirectoryProcessor) GetWorkerCount() int {
	return dp.workerCount
}

// handleModuleSync handles module-aware synchronization for a directory.
// It returns a ModuleSyncResult containing the source path to use (which may be
// a versioned clone if a version constraint was specified) and cleanup function.
func (dp *DirectoryProcessor) handleModuleSync(ctx context.Context, sourceDir, dirSrc string, moduleConfig *config.ModuleConfig, logger *logrus.Entry) (*ModuleSyncResult, error) {
	// Currently only support Go modules
	if moduleConfig.Type != "" && moduleConfig.Type != "go" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedModuleType, moduleConfig.Type)
	}

	// Default result uses the original source directory
	result := &ModuleSyncResult{
		SourcePath: sourceDir,
		Cleanup:    func() {}, // No-op cleanup by default
	}

	// Check if the directory contains a Go module
	moduleInfo, err := dp.moduleDetector.DetectModule(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to detect module: %w", err)
	}

	if moduleInfo == nil {
		logger.Debug("Directory does not contain a Go module, skipping module handling")
		return result, nil
	}

	result.ModuleInfo = moduleInfo

	logger.WithFields(logrus.Fields{
		"module_name":    moduleInfo.Name,
		"module_path":    moduleInfo.Path,
		"version_config": moduleConfig.Version,
		"check_tags":     moduleConfig.CheckTags,
	}).Info("Detected Go module for version-aware sync")

	// If version constraint is specified, resolve it
	if moduleConfig.Version != "" {
		// Determine if we should check git tags
		checkTags := true
		if moduleConfig.CheckTags != nil {
			checkTags = *moduleConfig.CheckTags
		}

		// Use the configured source repo URL for version resolution
		repoURL := dp.sourceRepoURL
		if repoURL == "" {
			logger.Warn("No source repo URL configured, cannot fetch versioned source")
			return result, nil
		}

		resolvedVersion, err := dp.moduleResolver.ResolveVersion(
			ctx,
			repoURL,
			moduleConfig.Version,
			checkTags,
		)
		if err != nil {
			logger.WithError(err).Warn("Failed to resolve module version constraint")
			// Continue with original source
			return result, nil
		}

		logger.WithFields(logrus.Fields{
			"constraint": moduleConfig.Version,
			"resolved":   resolvedVersion,
		}).Info("Resolved module version")

		result.ResolvedVersion = resolvedVersion
		moduleInfo.Version = resolvedVersion

		// If we have a module source resolver, fetch the versioned source
		if dp.moduleSourceResolver != nil && dp.tempDir != "" {
			versionedSource, err := dp.moduleSourceResolver.GetSourceAtVersion(
				ctx,
				repoURL,
				resolvedVersion,
				dirSrc, // Subdirectory within the repo
				dp.tempDir,
			)
			if err != nil {
				logger.WithError(err).Warn("Failed to fetch versioned source, using HEAD")
				return result, nil
			}

			logger.WithFields(logrus.Fields{
				"versioned_path": versionedSource.Path,
				"version":        resolvedVersion,
			}).Info("Using versioned module source")

			result.SourcePath = versionedSource.Path
			result.Cleanup = versionedSource.CleanupFunc
		}
	}

	return result, nil
}

// ProcessDirectoriesWithMetrics processes directories and returns detailed metrics
func (rs *RepositorySync) ProcessDirectoriesWithMetrics(ctx context.Context) ([]FileChange, map[string]DirectoryMetrics, error) {
	changes, err := rs.processDirectories(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Get metrics from the processor if available
	// Metrics collection requires tracking the processor instance
	m := make(map[string]DirectoryMetrics)

	return changes, m, nil
}

// DirectoryProcessingOptions configures directory processing behavior
type DirectoryProcessingOptions struct {
	WorkerCount            int                               // Number of concurrent workers
	ProgressThreshold      int                               // Minimum files for progress reporting
	ExclusionPatterns      []string                          // Additional exclusion patterns
	IncludeHidden          *bool                             // Override hidden file inclusion
	PreserveStructure      *bool                             // Override structure preservation
	CustomProgressReporter *DirectoryProgressReporterOptions // Custom progress reporting options
}

// ProcessDirectoriesWithOptions processes directories with custom options
func (rs *RepositorySync) ProcessDirectoriesWithOptions(ctx context.Context, opts DirectoryProcessingOptions) ([]FileChange, error) {
	if len(rs.target.Directories) == 0 {
		rs.logger.Debug("No directories configured for sync")
		return nil, nil
	}

	// Apply worker count
	workerCount := opts.WorkerCount
	if workerCount <= 0 {
		workerCount = 10
	}

	processTimer := metrics.StartTimer(ctx, rs.logger, "directory_processing_with_options").
		AddField("directory_count", len(rs.target.Directories)).
		AddField("worker_count", workerCount).
		AddField("progress_threshold", opts.ProgressThreshold)

	rs.logger.WithFields(logrus.Fields{
		"directory_count":    len(rs.target.Directories),
		"worker_count":       workerCount,
		"progress_threshold": opts.ProgressThreshold,
	}).Info("Processing directories with custom options")

	// Construct source repo URL for module-aware sync
	sourceRepoURL := ""
	if rs.sourceState != nil && rs.sourceState.Repo != "" {
		sourceRepoURL = fmt.Sprintf("https://github.com/%s", rs.sourceState.Repo)
	}

	// Build directory processor options
	var dpOpts *DirectoryProcessorOptions
	if rs.engine != nil {
		clearCache := false
		if rs.engine.Options() != nil {
			clearCache = rs.engine.Options().ClearModuleCache
		}
		dpOpts = &DirectoryProcessorOptions{
			GitClient:        rs.engine.GitClient(),
			SourceRepoURL:    sourceRepoURL,
			TempDir:          rs.tempDir,
			ClearModuleCache: clearCache,
		}
	}

	// Create directory processor with custom options
	processor := NewDirectoryProcessor(rs.logger, workerCount, dpOpts)
	defer processor.Close()

	sourcePath := filepath.Join(rs.tempDir, "source")
	var allChanges []FileChange

	// Process each directory mapping with options
	for _, dirMapping := range rs.target.Directories {
		// Apply option overrides to directory mapping
		modifiedMapping := dirMapping

		// Override hidden file inclusion if specified
		if opts.IncludeHidden != nil {
			modifiedMapping.IncludeHidden = opts.IncludeHidden
		}

		// Override structure preservation if specified
		if opts.PreserveStructure != nil {
			modifiedMapping.PreserveStructure = opts.PreserveStructure
		}

		// Add additional exclusion patterns
		if len(opts.ExclusionPatterns) > 0 {
			modifiedMapping.Exclude = append(modifiedMapping.Exclude, opts.ExclusionPatterns...)
		}

		changes, err := processor.ProcessDirectoryMapping(ctx, sourcePath, modifiedMapping, rs.target, rs.sourceState, rs.engine)
		if err != nil {
			// Log error but continue processing other directories
			rs.logger.WithError(err).WithField("directory", dirMapping.Src).Error("Failed to process directory with options")
			continue
		}
		allChanges = append(allChanges, changes...)
	}

	processTimer.AddField("total_changes", len(allChanges)).Stop()

	rs.logger.WithField("total_changes", len(allChanges)).Info("Directory processing with options completed")
	return allChanges, nil
}

// processDirectoryDeletion handles the deletion of an entire directory from the target repository
func (dp *DirectoryProcessor) processDirectoryDeletion(ctx context.Context, dirMapping config.DirectoryMapping, target config.TargetConfig, engine *Engine, logger *logrus.Entry) ([]FileChange, error) {
	logger.WithField("directory", dirMapping.Dest).Info("Processing directory deletion")

	// Get tree from GitHub API to find all files in the target directory
	treeAPI := NewGitHubAPI(engine.gh, dp.logger.Logger)
	treeMap, err := treeAPI.GetTree(ctx, target.Repo, "")
	if err != nil {
		logger.WithError(err).Error("Failed to fetch tree for directory deletion")
		return nil, fmt.Errorf("failed to fetch tree for directory deletion: %w", err)
	}

	// Get all files recursively in the directory to be deleted
	allFiles := treeMap.GetAllFilesInDirectoryRecursively(dirMapping.Dest)
	if len(allFiles) == 0 {
		logger.WithField("directory", dirMapping.Dest).Info("Directory does not exist in target repository, skipping deletion")
		return nil, internalerrors.ErrFileNotFound
	}

	logger.WithFields(logrus.Fields{
		"directory":  dirMapping.Dest,
		"file_count": len(allFiles),
	}).Info("Found files in directory to delete")

	// Create FileChange entries for each file to be deleted
	changes := make([]FileChange, 0, len(allFiles))
	for _, fileNode := range allFiles {
		// Get existing file content for proper tracking
		existingContent, err := dp.getExistingFileContent(ctx, engine, target.Repo, fileNode.Path, target.Branch)
		if err != nil {
			logger.WithError(err).WithField("file", fileNode.Path).Debug("Could not get existing content for deletion, continuing")
			existingContent = nil // Continue with deletion even if we can't get content
		}

		change := FileChange{
			Path:            fileNode.Path,
			Content:         nil, // No content for deletions
			OriginalContent: existingContent,
			IsNew:           false,
			IsDeleted:       true,
		}

		changes = append(changes, change)
	}

	logger.WithFields(logrus.Fields{
		"directory":     dirMapping.Dest,
		"total_changes": len(changes),
	}).Info("Directory deletion processed successfully")

	return changes, nil
}

// getExistingFileContent retrieves file content for deletion tracking
func (dp *DirectoryProcessor) getExistingFileContent(ctx context.Context, engine *Engine, repo, filePath, branch string) ([]byte, error) {
	fileContent, err := engine.gh.GetFile(ctx, repo, filePath, branch)
	if err != nil {
		return nil, err
	}
	return fileContent.Content, nil
}

// GetModuleUpdates returns the collected module update information
func (dp *DirectoryProcessor) GetModuleUpdates() []ModuleUpdateInfo {
	dp.moduleUpdatesMu.Lock()
	defer dp.moduleUpdatesMu.Unlock()
	// Return a copy to avoid race conditions
	result := make([]ModuleUpdateInfo, len(dp.moduleUpdates))
	copy(result, dp.moduleUpdates)
	return result
}
