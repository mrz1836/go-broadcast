package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
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
			workerLogger.Debug("Context cancelled, worker exiting")
			return ctx.Err()
		}
	}
}

// processFileJob processes a single file job with error handling
func (bp *BatchProcessor) processFileJob(ctx context.Context, sourcePath string, job FileJob, logger *logrus.Entry) fileProcessResult {
	logger = logger.WithFields(logrus.Fields{
		"source_path": job.SourcePath,
		"dest_path":   job.DestPath,
	})

	// Build full source path
	fullSourcePath := filepath.Join(sourcePath, job.SourcePath)

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

	// Apply transformations
	transformCtx := transform.Context{
		SourceRepo: bp.sourceState.Repo,
		TargetRepo: bp.target.Repo,
		FilePath:   job.DestPath,
		Variables:  job.Transform.Variables,
	}

	transformedContent := srcContent
	if job.Transform.RepoName || len(job.Transform.Variables) > 0 {
		transformedContent, err = bp.engine.transform.Transform(ctx, srcContent, transformCtx)
		if err != nil {
			logger.WithError(err).Error("Transformation failed")
			return fileProcessResult{
				Change: nil,
				Error:  fmt.Errorf("transformation failed for %s: %w", job.DestPath, err),
				Job:    job,
			}
		}
	}

	// Check if content actually changed (for existing files)
	existingContent, err := bp.getExistingFileContent(ctx, job.DestPath)
	if err == nil && string(existingContent) == string(transformedContent) {
		logger.Debug("File content unchanged, skipping")
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

	logger.Debug("File processed successfully")
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

	for result := range resultChan {
		if result.Error != nil {
			// Handle different error types gracefully
			if result.Error == internalerrors.ErrTransformNotFound {
				skipCount++
				bp.logger.WithField("file", result.Job.DestPath).Debug("File content unchanged, skipping")
				continue
			}
			if result.Error == internalerrors.ErrFileNotFound {
				skipCount++
				bp.logger.WithField("file", result.Job.SourcePath).Debug("Source file not found, skipping")
				continue
			}

			// For other errors, log but continue processing other files
			errorCount++
			bp.logger.WithError(result.Error).WithField("file", result.Job.SourcePath).Error("File processing failed")
			continue
		}

		if result.Change != nil {
			changes = append(changes, *result.Change)
		}
	}

	bp.logger.WithFields(logrus.Fields{
		"processed": len(changes),
		"skipped":   skipCount,
		"errors":    errorCount,
	}).Info("Batch processing completed")

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
	TotalJobs     int
	ProcessedJobs int
	SkippedJobs   int
	FailedJobs    int
	WorkerCount   int
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

			// Update progress
			if progressReporter != nil {
				mu.Lock()
				*processed++
				currentCount := int(*processed)
				mu.Unlock()

				progressReporter.UpdateProgress(currentCount, totalJobs, fmt.Sprintf("Processing files... (%d/%d)", currentCount, totalJobs))
			}

		case <-ctx.Done():
			workerLogger.Debug("Context cancelled, worker exiting")
			return ctx.Err()
		}
	}
}

// ProgressReporter defines the interface for progress reporting
type ProgressReporter interface {
	UpdateProgress(current, total int, message string)
}
