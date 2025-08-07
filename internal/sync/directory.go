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
	exclusionEngine *ExclusionEngine
	progressManager *DirectoryProgressManager
	workerCount     int
	logger          *logrus.Entry
	moduleDetector  *ModuleDetector
	moduleResolver  *ModuleResolver
	moduleCache     *ModuleCache
}

// NewDirectoryProcessor creates a new directory processor
func NewDirectoryProcessor(logger *logrus.Entry, workerCount int) *DirectoryProcessor {
	if workerCount <= 0 {
		workerCount = 10 // Default worker count
	}

	// Create module components with default settings
	moduleCache := NewModuleCache(5*time.Minute, logger.Logger)
	moduleDetector := NewModuleDetector(logger.Logger)
	moduleResolver := NewModuleResolver(logger.Logger, moduleCache)

	return &DirectoryProcessor{
		progressManager: NewDirectoryProgressManager(logger),
		workerCount:     workerCount,
		logger:          logger,
		moduleDetector:  moduleDetector,
		moduleResolver:  moduleResolver,
		moduleCache:     moduleCache,
	}
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

	// Create directory processor
	processor := NewDirectoryProcessor(rs.logger, 10) // Use default worker count
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
	})

	logger.Info("Processing directory mapping")

	// Create exclusion engine with directory-specific patterns
	dp.exclusionEngine = NewExclusionEngineWithIncludes(dirMapping.Exclude, dirMapping.IncludeOnly)

	// Build full source directory path
	fullSourceDir := filepath.Join(sourcePath, dirMapping.Src)

	// Check if source directory exists
	if _, err := os.Stat(fullSourceDir); os.IsNotExist(err) {
		logger.Warn("Source directory not found, skipping")
		return nil, internalerrors.ErrFileNotFound
	}

	// Handle module-aware sync if configured
	if dirMapping.Module != nil {
		if err := dp.handleModuleSync(ctx, fullSourceDir, dirMapping.Module, logger); err != nil {
			logger.WithError(err).Warn("Module sync handling failed, continuing with standard sync")
			// Continue with standard sync if module handling fails
		} else {
			logger.WithField("module_type", dirMapping.Module.Type).Debug("Module sync configuration applied")
		}
	}

	// Discover files in the directory
	files, err := dp.discoverFiles(ctx, fullSourceDir, dirMapping)
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
			if dp.exclusionEngine.IsDirectoryExcluded(relPath) {
				dp.logger.WithField("directory", relPath).Debug("Directory excluded by patterns")
				return filepath.SkipDir
			}
			// Record directory traversal for metrics
			dp.progressManager.GetReporter(dirMapping.Src, 50).RecordDirectoryWalked()
			return nil // Continue walking this directory
		}

		// Check if file should be excluded
		if dp.exclusionEngine.IsExcluded(relPath) {
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

// handleModuleSync handles module-aware synchronization for a directory
func (dp *DirectoryProcessor) handleModuleSync(ctx context.Context, sourceDir string, moduleConfig *config.ModuleConfig, logger *logrus.Entry) error {
	// Currently only support Go modules
	if moduleConfig.Type != "" && moduleConfig.Type != "go" {
		return fmt.Errorf("%w: %s", ErrUnsupportedModuleType, moduleConfig.Type)
	}

	// Check if the directory contains a Go module
	moduleInfo, err := dp.moduleDetector.DetectModule(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to detect module: %w", err)
	}

	if moduleInfo == nil {
		logger.Debug("Directory does not contain a Go module, skipping module handling")
		return nil
	}

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

		// Get the repository path from the module name (assuming GitHub for now)
		// This is a simplified implementation - in reality, we'd need to parse the module name
		// more carefully and handle various repository hosting services
		repoPath := moduleInfo.Name
		repoPath = strings.TrimPrefix(repoPath, "github.com/")

		resolvedVersion, err := dp.moduleResolver.ResolveVersion(
			ctx,
			fmt.Sprintf("https://github.com/%s", repoPath),
			moduleConfig.Version,
			checkTags,
		)
		if err != nil {
			logger.WithError(err).Warn("Failed to resolve module version constraint")
			// Continue without version resolution
			return nil
		}

		logger.WithFields(logrus.Fields{
			"constraint": moduleConfig.Version,
			"resolved":   resolvedVersion,
		}).Info("Resolved module version")

		// Store the resolved version for potential use in file processing
		// This could be used to update go.mod files if UpdateRefs is true
		moduleInfo.Version = resolvedVersion
	}

	// If UpdateRefs is true, we'll need to update go.mod references
	// This would be handled during file processing
	if moduleConfig.UpdateRefs {
		logger.Debug("Module reference updates will be applied during file processing")
	}

	return nil
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

	// Create directory processor with custom options
	processor := NewDirectoryProcessor(rs.logger, workerCount)
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
