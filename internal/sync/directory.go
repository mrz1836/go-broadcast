package sync

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/metrics"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/sirupsen/logrus"
)

// DirectoryProcessor handles concurrent directory processing with worker pools
type DirectoryProcessor struct {
	exclusionEngine *ExclusionEngine
	progressManager *DirectoryProgressManager
	workerCount     int
	logger          *logrus.Entry
}

// NewDirectoryProcessor creates a new directory processor
func NewDirectoryProcessor(logger *logrus.Entry, workerCount int) *DirectoryProcessor {
	if workerCount <= 0 {
		workerCount = 10 // Default worker count
	}

	return &DirectoryProcessor{
		progressManager: NewDirectoryProgressManager(logger),
		workerCount:     workerCount,
		logger:          logger,
	}
}

// processDirectories processes directory mappings for the RepositorySync
func (rs *RepositorySync) processDirectories(ctx context.Context) ([]FileChange, error) {
	if len(rs.target.Directories) == 0 {
		rs.logger.Debug("No directories configured for sync")
		return nil, nil
	}

	processTimer := metrics.StartTimer(ctx, rs.logger, "directory_processing").
		AddField("directory_count", len(rs.target.Directories))

	rs.logger.WithField("directory_count", len(rs.target.Directories)).Info("Processing directories")

	// Create directory processor
	processor := NewDirectoryProcessor(rs.logger, 10) // Use default worker count

	sourcePath := filepath.Join(rs.tempDir, "source")
	var allChanges []FileChange

	// Process each directory mapping
	for _, dirMapping := range rs.target.Directories {
		changes, err := processor.ProcessDirectoryMapping(ctx, sourcePath, dirMapping, rs.target, rs.sourceState, rs.engine)
		if err != nil {
			// Log error but continue processing other directories
			rs.logger.WithError(err).WithField("directory", dirMapping.Src).Error("Failed to process directory")
			continue
		}
		allChanges = append(allChanges, changes...)
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
	dp.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)

	// Build full source directory path
	fullSourceDir := filepath.Join(sourcePath, dirMapping.Src)

	// Check if source directory exists
	if _, err := os.Stat(fullSourceDir); os.IsNotExist(err) {
		logger.Warn("Source directory not found, skipping")
		return nil, internalerrors.ErrFileNotFound
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
	metrics := progressReporter.Complete()

	logger.WithFields(logrus.Fields{
		"files_discovered": metrics.FilesDiscovered,
		"files_processed":  metrics.FilesProcessed,
		"files_excluded":   metrics.FilesExcluded,
		"changes":          len(changes),
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

// createFileJobs converts discovered files into processing jobs
func (dp *DirectoryProcessor) createFileJobs(files []DiscoveredFile, dirMapping config.DirectoryMapping) []FileJob {
	jobs := make([]FileJob, 0, len(files))

	for _, file := range files {
		// Skip directories
		if file.IsDir {
			continue
		}

		// Determine destination path
		destPath := dp.calculateDestinationPath(file.RelativePath, dirMapping)

		// Create job
		job := FileJob{
			SourcePath: filepath.Join(dirMapping.Src, file.RelativePath),
			DestPath:   destPath,
			Transform:  dirMapping.Transform,
		}

		jobs = append(jobs, job)
	}

	dp.logger.WithFields(logrus.Fields{
		"source_directory": dirMapping.Src,
		"jobs_created":     len(jobs),
	}).Debug("File jobs created")

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
		return fmt.Errorf("source directory cannot be empty")
	}

	if dirMapping.Dest == "" {
		return fmt.Errorf("destination directory cannot be empty")
	}

	// Validate exclusion patterns
	for _, pattern := range dirMapping.Exclude {
		if pattern == "" {
			return fmt.Errorf("empty exclusion pattern not allowed")
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

// ProcessDirectoriesWithMetrics processes directories and returns detailed metrics
func (rs *RepositorySync) ProcessDirectoriesWithMetrics(ctx context.Context) ([]FileChange, map[string]DirectoryMetrics, error) {
	changes, err := rs.processDirectories(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Get metrics from the processor if available
	// Note: This is a simplified version. In a full implementation,
	// we'd need to track the processor instance to get metrics.
	metrics := make(map[string]DirectoryMetrics)

	return changes, metrics, nil
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
