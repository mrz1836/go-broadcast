package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// BatchProcessor handles concurrent file processing with worker pools
type BatchProcessor struct {
	engine      *Engine
	target      config.TargetConfig
	sourceState *state.SourceState
	logger      *logrus.Entry
	workerCount int
}

// FileJob represents a file processing job
type FileJob struct {
	SourcePath string
	DestPath   string
	Transform  config.Transform

	// Directory-specific fields
	IsFromDirectory  bool
	DirectoryMapping *config.DirectoryMapping
	RelativePath     string
	FileIndex        int
	TotalFiles       int
}

// NewBatchProcessor creates a new batch processor with the specified worker count
func NewBatchProcessor(engine *Engine, target config.TargetConfig, sourceState *state.SourceState, logger *logrus.Entry, workerCount int) *BatchProcessor {
	if workerCount <= 0 {
		workerCount = 10 // Default worker count
	}

	return &BatchProcessor{
		engine:      engine,
		target:      target,
		sourceState: sourceState,
		logger:      logger,
		workerCount: workerCount,
	}
}

// ProcessFiles processes multiple files concurrently with error resilience
func (bp *BatchProcessor) ProcessFiles(ctx context.Context, sourcePath string, jobs []FileJob) ([]FileChange, error) {
	if len(jobs) == 0 {
		return nil, nil
	}

	bp.logger.WithField("job_count", len(jobs)).Info("Starting batch file processing")

	// Create channels for job distribution
	jobChan := make(chan FileJob, len(jobs))
	resultChan := make(chan fileProcessResult, len(jobs))

	// Start worker goroutines
	g, ctx := errgroup.WithContext(ctx)

	// Limit concurrency to prevent resource exhaustion
	g.SetLimit(bp.workerCount)

	// Start workers
	for i := 0; i < bp.workerCount; i++ {
		workerID := i
		g.Go(func() error {
			return bp.worker(ctx, workerID, sourcePath, jobChan, resultChan)
		})
	}

	// Send jobs to workers
	go func() {
		defer close(jobChan)
		for _, job := range jobs {
			select {
			case jobChan <- job:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for all workers to complete
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("batch processing failed: %w", err)
	}

	// Collect results
	close(resultChan)
	return bp.collectResults(resultChan), nil
}

// fileProcessResult represents the result of processing a single file
type fileProcessResult struct {
	Change *FileChange
	Error  error
	Job    FileJob
}

// worker processes files from the job channel
func (bp *BatchProcessor) worker(ctx context.Context, workerID int, sourcePath string, jobChan <-chan FileJob, resultChan chan<- fileProcessResult) error {
	workerLogger := bp.logger.WithField("worker_id", workerID)
	workerLogger.Debug("Starting batch processor worker")

	for {
		select {
		case job, ok := <-jobChan:
			if !ok {
				workerLogger.Debug("Job channel closed, worker exiting")
				return nil
			}

			// Process the file job
			result := bp.processFileJob(ctx, sourcePath, job, workerLogger)

			// Send result (non-blocking)
			select {
			case resultChan <- result:
			case <-ctx.Done():
				return ctx.Err()
			}

		case <-ctx.Done():
			workerLogger.Debug("Context canceled, worker exiting")
			return ctx.Err()
		}
	}
}

// TransformMetrics tracks transformation performance and outcomes
type TransformMetrics struct {
	BinaryFilesSkipped int
	TransformDuration  time.Duration
	TransformErrors    int
	TransformSuccess   int
	TotalFilesChecked  int
}

// processFileJob processes a single file job with enhanced error handling, binary detection, and metrics
func (bp *BatchProcessor) processFileJob(ctx context.Context, sourcePath string, job FileJob, logger *logrus.Entry) fileProcessResult {
	return bp.processFileJobWithReporter(ctx, sourcePath, job, logger, nil)
}

// processFileJobWithReporter processes a single file job with enhanced progress reporting
func (bp *BatchProcessor) processFileJobWithReporter(ctx context.Context, sourcePath string, job FileJob, logger *logrus.Entry, progressReporter EnhancedProgressReporter) fileProcessResult {
	processStart := time.Now()
	metrics := &TransformMetrics{}
	defer func() {
		metrics.TransformDuration = time.Since(processStart)
		if bp.logger.Level >= logrus.DebugLevel {
			bp.logger.WithFields(logrus.Fields{
				"file":                  job.DestPath,
				"binary_files_skipped":  metrics.BinaryFilesSkipped,
				"transform_duration_ms": metrics.TransformDuration.Milliseconds(),
				"transform_errors":      metrics.TransformErrors,
				"transform_success":     metrics.TransformSuccess,
				"total_files_checked":   metrics.TotalFilesChecked,
			}).Debug("File job processing metrics")
		}
	}()

	logger = logger.WithFields(logrus.Fields{
		"source_path":       job.SourcePath,
		"dest_path":         job.DestPath,
		"is_from_directory": job.IsFromDirectory,
	})

	if job.IsFromDirectory {
		logger = logger.WithFields(logrus.Fields{
			"relative_path": job.RelativePath,
			"file_index":    job.FileIndex,
			"total_files":   job.TotalFiles,
		})
		logger.Debug("Processing directory file job")
	} else {
		logger.Debug("Processing regular file job")
	}

	// Build full source path
	fullSourcePath := filepath.Join(sourcePath, job.SourcePath)
	logger.WithField("full_source_path", fullSourcePath).Debug("Reading source file")

	// Check if source file exists
	srcContent, err := os.ReadFile(fullSourcePath) //nolint:gosec // Path is constructed from trusted configuration
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("Source file not found, skipping")
			return fileProcessResult{
				Change: nil,
				Error:  internalerrors.ErrFileNotFound,
				Job:    job,
			}
		}
		logger.WithError(err).Error("Failed to read source file")
		return fileProcessResult{
			Change: nil,
			Error:  fmt.Errorf("failed to read source file %s: %w", job.SourcePath, err),
			Job:    job,
		}
	}

	metrics.TotalFilesChecked++
	logger.WithField("content_size", len(srcContent)).Debug("Source file content loaded")

	// Check for binary content before applying transformations
	if transform.IsBinary(job.SourcePath, srcContent) {
		metrics.BinaryFilesSkipped++

		// Report binary file metrics to progress reporter
		if progressReporter != nil {
			progressReporter.RecordBinaryFileSkipped(int64(len(srcContent)))
		}

		logger.WithFields(logrus.Fields{
			"file_path":    job.SourcePath,
			"content_size": len(srcContent),
		}).Info("Binary file detected, skipping transformations")

		// Check if content actually changed (for existing files)
		existingContent, existingErr := bp.getExistingFileContent(ctx, job.DestPath)
		if existingErr == nil && string(existingContent) == string(srcContent) {
			logger.Debug("Binary file content unchanged, skipping")
			return fileProcessResult{
				Change: nil,
				Error:  internalerrors.ErrTransformNotFound,
				Job:    job,
			}
		}

		// Create file change for binary file (no transformation)
		change := &FileChange{
			Path:            job.DestPath,
			Content:         srcContent, // Use original content for binary files
			OriginalContent: srcContent,
			IsNew:           existingErr != nil, // existingErr means file doesn't exist
		}

		logger.Debug("Binary file processed successfully (no transformation)")
		metrics.TransformSuccess++

		// Report successful processing
		if progressReporter != nil {
			progressReporter.RecordTransformSuccess(time.Since(processStart))
		}

		return fileProcessResult{
			Change: change,
			Error:  nil,
			Job:    job,
		}
	}

	logger.Debug("Text file detected, applying transformations")

	// Apply transformations with enhanced context and error isolation
	transformedContent := srcContent
	if job.Transform.RepoName || len(job.Transform.Variables) > 0 {
		transformStart := time.Now()
		logger.WithFields(logrus.Fields{
			"repo_name_transform": job.Transform.RepoName,
			"variables_count":     len(job.Transform.Variables),
			"variables":           job.Transform.Variables,
		}).Debug("Starting content transformation")

		// Create appropriate transform context based on job type
		var transformContext transform.Context
		if job.IsFromDirectory && job.DirectoryMapping != nil {
			// Use DirectoryTransformContext for directory-aware transformations
			baseCtx := transform.Context{
				SourceRepo: bp.sourceState.Repo,
				TargetRepo: bp.target.Repo,
				FilePath:   job.DestPath,
				Variables:  job.Transform.Variables,
				LogConfig: &logging.LogConfig{
					Debug: logging.DebugFlags{
						Transform: bp.logger.Level >= logrus.DebugLevel,
					},
					Verbose: func() int {
						if bp.logger.Level >= logrus.DebugLevel {
							return 2
						}
						return 0
					}(),
				},
			}

			dirTransformCtx := transform.NewDirectoryTransformContext(
				baseCtx,
				job.DirectoryMapping,
				job.RelativePath,
				job.FileIndex,
				job.TotalFiles,
			)

			logger.WithFields(logrus.Fields{
				"directory_context": dirTransformCtx.String(),
				"source_dir":        job.DirectoryMapping.Src,
				"dest_dir":          job.DirectoryMapping.Dest,
			}).Debug("Using DirectoryTransformContext")

			// DirectoryTransformContext embeds Context, so we can use it directly
			transformContext = dirTransformCtx.Context
		} else {
			// Use regular Context for single file transformations
			transformContext = transform.Context{
				SourceRepo: bp.sourceState.Repo,
				TargetRepo: bp.target.Repo,
				FilePath:   job.DestPath,
				Variables:  job.Transform.Variables,
				LogConfig: &logging.LogConfig{
					Debug: logging.DebugFlags{
						Transform: bp.logger.Level >= logrus.DebugLevel,
					},
					Verbose: func() int {
						if bp.logger.Level >= logrus.DebugLevel {
							return 2
						}
						return 0
					}(),
				},
			}

			logger.WithField("transform_context", fmt.Sprintf("%+v", transformContext)).Debug("Using regular TransformContext")
		}

		// Apply transformation with error isolation - don't fail entire batch on transform errors
		transformedContent, err = bp.engine.transform.Transform(ctx, srcContent, transformContext)
		transformDuration := time.Since(transformStart)

		logger.WithFields(logrus.Fields{
			"transform_duration_ms": transformDuration.Milliseconds(),
			"content_size_before":   len(srcContent),
			"content_size_after":    len(transformedContent),
			"content_changed":       len(srcContent) != len(transformedContent) || string(srcContent) != string(transformedContent),
		}).Debug("Transformation completed")

		if err != nil {
			metrics.TransformErrors++

			// Report transformation error to progress reporter
			if progressReporter != nil {
				progressReporter.RecordTransformError()
			}

			// Log error but don't fail the entire batch - use original content as fallback
			logger.WithError(err).WithFields(logrus.Fields{
				"fallback_strategy":     "use_original_content",
				"transform_duration_ms": transformDuration.Milliseconds(),
			}).Warn("Transformation failed, using original content as fallback")

			// Use original content if transformation fails
			transformedContent = srcContent
		} else {
			metrics.TransformSuccess++

			// Report transformation success to progress reporter
			if progressReporter != nil {
				progressReporter.RecordTransformSuccess(transformDuration)
			}

			logger.Debug("Transformation successful")
		}
	} else {
		logger.Debug("No transformations configured, using original content")
		metrics.TransformSuccess++

		// Report success for no-transform case (instantaneous)
		if progressReporter != nil {
			progressReporter.RecordTransformSuccess(0)
		}
	}

	// Check if content actually changed (for existing files)
	existingContent, err := bp.getExistingFileContent(ctx, job.DestPath)
	if err == nil && string(existingContent) == string(transformedContent) {
		logger.Debug("File content unchanged after transformation, skipping")
		return fileProcessResult{
			Change: nil,
			Error:  internalerrors.ErrTransformNotFound,
			Job:    job,
		}
	}

	// Create file change
	change := &FileChange{
		Path:            job.DestPath,
		Content:         transformedContent,
		OriginalContent: srcContent,
		IsNew:           err != nil, // err means file doesn't exist
	}

	logger.WithFields(logrus.Fields{
		"is_new_file":           change.IsNew,
		"final_content_size":    len(change.Content),
		"original_content_size": len(change.OriginalContent),
	}).Debug("File processed successfully")

	return fileProcessResult{
		Change: change,
		Error:  nil,
		Job:    job,
	}
}

// collectResults collects and filters results from the result channel
func (bp *BatchProcessor) collectResults(resultChan <-chan fileProcessResult) []FileChange {
	var changes []FileChange
	var errorCount int
	var skipCount int
	var directoryFilesCount int

	for result := range resultChan {
		// Track directory vs regular files
		if result.Job.IsFromDirectory {
			directoryFilesCount++
		}

		// Track binary files processed (successful results that might have been binary)
		if result.Error == nil && result.Change != nil {
			// We can't easily detect if it was binary here, but we can log it was processed
			if result.Job.IsFromDirectory {
				bp.logger.WithFields(logrus.Fields{
					"file":          result.Job.DestPath,
					"relative_path": result.Job.RelativePath,
					"directory":     result.Job.DirectoryMapping.Src,
				}).Debug("Directory file processed successfully")
			}
		}

		if result.Error != nil {
			// Handle different error types gracefully
			if errors.Is(result.Error, internalerrors.ErrTransformNotFound) {
				skipCount++
				bp.logger.WithField("file", result.Job.DestPath).Debug("File content unchanged, skipping")
				continue
			}
			if errors.Is(result.Error, internalerrors.ErrFileNotFound) {
				skipCount++
				bp.logger.WithField("file", result.Job.SourcePath).Debug("Source file not found, skipping")
				continue
			}

			// For other errors, log but continue processing other files
			errorCount++
			bp.logger.WithError(result.Error).WithFields(logrus.Fields{
				"file":              result.Job.SourcePath,
				"is_from_directory": result.Job.IsFromDirectory,
			}).Error("File processing failed")
			continue
		}

		if result.Change != nil {
			changes = append(changes, *result.Change)
		}
	}

	bp.logger.WithFields(logrus.Fields{
		"processed":       len(changes),
		"skipped":         skipCount,
		"errors":          errorCount,
		"directory_files": directoryFilesCount,
		"regular_files":   len(changes) + skipCount + errorCount - directoryFilesCount,
	}).Info("Batch processing completed with enhanced metrics")

	return changes
}

// getExistingFileContent retrieves the current content of a file from the target repo
func (bp *BatchProcessor) getExistingFileContent(ctx context.Context, filePath string) ([]byte, error) {
	// Try to get file from the target repository's default branch
	fileContent, err := bp.engine.gh.GetFile(ctx, bp.target.Repo, filePath, "")
	if err != nil {
		return nil, err
	}
	return fileContent.Content, nil
}

// ProcessorStats provides statistics about batch processing performance
type ProcessorStats struct {
	TotalJobs          int
	ProcessedJobs      int
	SkippedJobs        int
	FailedJobs         int
	WorkerCount        int
	BinaryFilesSkipped int
	DirectoryFiles     int
	RegularFiles       int
	TransformErrors    int
	TransformSuccess   int
}

// GetStats returns processing statistics
func (bp *BatchProcessor) GetStats() ProcessorStats {
	return ProcessorStats{
		WorkerCount: bp.workerCount,
	}
}

// ConfiguredWorkerCount returns the configured worker count
func (bp *BatchProcessor) ConfiguredWorkerCount() int {
	return bp.workerCount
}

// SetWorkerCount updates the worker count (affects future processing only)
func (bp *BatchProcessor) SetWorkerCount(count int) {
	if count > 0 {
		bp.workerCount = count
	}
}

// NewFileJob creates a new FileJob for regular file processing
func NewFileJob(sourcePath, destPath string, transform config.Transform) FileJob {
	return FileJob{
		SourcePath:       sourcePath,
		DestPath:         destPath,
		Transform:        transform,
		IsFromDirectory:  false,
		DirectoryMapping: nil,
		RelativePath:     "",
		FileIndex:        0,
		TotalFiles:       1,
	}
}

// NewDirectoryFileJob creates a new FileJob for directory file processing
func NewDirectoryFileJob(
	sourcePath, destPath string,
	transform config.Transform,
	directoryMapping *config.DirectoryMapping,
	relativePath string,
	fileIndex, totalFiles int,
) FileJob {
	return FileJob{
		SourcePath:       sourcePath,
		DestPath:         destPath,
		Transform:        transform,
		IsFromDirectory:  true,
		DirectoryMapping: directoryMapping,
		RelativePath:     relativePath,
		FileIndex:        fileIndex,
		TotalFiles:       totalFiles,
	}
}

// ProcessFilesWithProgress processes files with progress reporting
func (bp *BatchProcessor) ProcessFilesWithProgress(ctx context.Context, sourcePath string, jobs []FileJob, progressReporter ProgressReporter) ([]FileChange, error) {
	if len(jobs) == 0 {
		return nil, nil
	}

	bp.logger.WithField("job_count", len(jobs)).Info("Starting batch file processing with progress reporting")

	// Create channels for job distribution
	jobChan := make(chan FileJob, len(jobs))
	resultChan := make(chan fileProcessResult, len(jobs))

	// Initialize progress
	if progressReporter != nil {
		progressReporter.UpdateProgress(0, len(jobs), "Starting file processing...")
	}

	// Start worker goroutines
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(bp.workerCount)

	// Progress tracking
	var processed int32
	var mu sync.Mutex

	// Start workers with progress tracking
	for i := 0; i < bp.workerCount; i++ {
		workerID := i
		g.Go(func() error {
			return bp.workerWithProgress(ctx, workerID, sourcePath, jobChan, resultChan, &processed, &mu, len(jobs), progressReporter)
		})
	}

	// Send jobs to workers
	go func() {
		defer close(jobChan)
		for _, job := range jobs {
			select {
			case jobChan <- job:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for all workers to complete
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("batch processing failed: %w", err)
	}

	// Final progress update
	if progressReporter != nil {
		progressReporter.UpdateProgress(len(jobs), len(jobs), "File processing completed")
	}

	// Collect results
	close(resultChan)
	return bp.collectResults(resultChan), nil
}

// workerWithProgress processes files with progress updates
func (bp *BatchProcessor) workerWithProgress(ctx context.Context, workerID int, sourcePath string, jobChan <-chan FileJob, resultChan chan<- fileProcessResult, processed *int32, mu *sync.Mutex, totalJobs int, progressReporter ProgressReporter) error {
	workerLogger := bp.logger.WithField("worker_id", workerID)
	workerLogger.Debug("Starting batch processor worker with progress tracking")

	// Try to cast to enhanced progress reporter
	enhancedReporter, _ := progressReporter.(EnhancedProgressReporter)

	for {
		select {
		case job, ok := <-jobChan:
			if !ok {
				workerLogger.Debug("Job channel closed, worker exiting")
				return nil
			}

			// Process the file job with enhanced reporting if available
			var result fileProcessResult
			if enhancedReporter != nil {
				result = bp.processFileJobWithReporter(ctx, sourcePath, job, workerLogger, enhancedReporter)
			} else {
				result = bp.processFileJob(ctx, sourcePath, job, workerLogger)
			}

			// Send result (non-blocking)
			select {
			case resultChan <- result:
			case <-ctx.Done():
				return ctx.Err()
			}

			// Update progress
			if progressReporter != nil {
				mu.Lock()
				*processed++
				currentCount := int(*processed)
				mu.Unlock()

				progressReporter.UpdateProgress(currentCount, totalJobs, fmt.Sprintf("Processing files... (%d/%d)", currentCount, totalJobs))
			}

		case <-ctx.Done():
			workerLogger.Debug("Context canceled, worker exiting")
			return ctx.Err()
		}
	}
}

// ProgressReporter defines the interface for progress reporting
type ProgressReporter interface {
	UpdateProgress(current, total int, message string)
}

// EnhancedProgressReporter extends ProgressReporter with binary file metrics support
type EnhancedProgressReporter interface {
	ProgressReporter
	RecordBinaryFileSkipped(size int64)
	RecordTransformError()
	RecordTransformSuccess(duration time.Duration)
}
