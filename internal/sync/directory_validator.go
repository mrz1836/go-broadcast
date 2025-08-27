package sync

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// Static error variables
var (
	ErrValidationFailed          = errors.New("validation failed")
	ErrTransformValidationFailed = errors.New("transform validation failed")
	ErrEncodingValidationFailed  = errors.New("encoding validation failed")
)

// ValidationResult represents the outcome of a validation operation
type ValidationResult struct {
	Valid   bool              `json:"valid"`
	Errors  []string          `json:"errors,omitempty"`
	Summary ValidationSummary `json:"summary"`
}

// ValidationSummary provides statistics about the validation run
type ValidationSummary struct {
	TotalFiles        int           `json:"total_files"`
	ValidFiles        int           `json:"valid_files"`
	InvalidFiles      int           `json:"invalid_files"`
	MissingFiles      int           `json:"missing_files"`
	ExtraFiles        int           `json:"extra_files"`
	ContentMismatches int           `json:"content_mismatches"`
	TransformErrors   int           `json:"transform_errors"`
	StructureErrors   int           `json:"structure_errors"`
	ExclusionErrors   int           `json:"exclusion_errors"`
	Duration          time.Duration `json:"duration"`
	BytesValidated    int64         `json:"bytes_validated"`
}

// FileValidationError contains detailed information about a file validation failure
type FileValidationError struct {
	FilePath    string   `json:"file_path"`
	ErrorType   string   `json:"error_type"`
	Expected    string   `json:"expected,omitempty"`
	Actual      string   `json:"actual,omitempty"`
	Details     string   `json:"details,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// PerformanceValidationResult tracks performance metrics validation
type PerformanceValidationResult struct {
	Valid                 bool                         `json:"valid"`
	APICallsOptimized     bool                         `json:"api_calls_optimized"`
	CacheHitRateGood      bool                         `json:"cache_hit_rate_good"`
	MemoryUsageAcceptable bool                         `json:"memory_usage_acceptable"`
	ProcessingTimeGood    bool                         `json:"processing_time_good"`
	Metrics               ValidationPerformanceMetrics `json:"metrics"`
	Thresholds            PerformanceThresholds        `json:"thresholds"`
	Recommendations       []string                     `json:"recommendations,omitempty"`
}

// ValidationPerformanceMetrics contains actual performance measurements
type ValidationPerformanceMetrics struct {
	APICalls       int           `json:"api_calls"`
	CacheHits      int           `json:"cache_hits"`
	CacheMisses    int           `json:"cache_misses"`
	CacheHitRate   float64       `json:"cache_hit_rate"`
	MemoryUsage    int64         `json:"memory_usage_bytes"`
	ProcessingTime time.Duration `json:"processing_time"`
	FilesProcessed int           `json:"files_processed"`
	ThroughputMBps float64       `json:"throughput_mbps"`
}

// PerformanceThresholds defines acceptable performance limits
type PerformanceThresholds struct {
	MaxAPICalls       int           `json:"max_api_calls"`
	MinCacheHitRate   float64       `json:"min_cache_hit_rate"`
	MaxMemoryMB       int64         `json:"max_memory_mb"`
	MaxProcessingTime time.Duration `json:"max_processing_time"`
	MinThroughputMBps float64       `json:"min_throughput_mbps"`
}

// DirectoryValidator provides utilities for validating directory sync results
type DirectoryValidator struct {
	logger                *logrus.Entry
	exclusionEngine       *ExclusionEngine
	performanceThresholds PerformanceThresholds
	mu                    sync.RWMutex
}

// integrityJob represents a file integrity validation job
type integrityJob struct {
	sourcePath    string
	sourceContent string
	destContent   string
	exists        bool
}

// ValidationOptions configures validation behavior
type ValidationOptions struct {
	CheckContent           bool                   `json:"check_content"`
	CheckStructure         bool                   `json:"check_structure"`
	CheckExclusions        bool                   `json:"check_exclusions"`
	CheckTransforms        bool                   `json:"check_transforms"`
	CheckPerformance       bool                   `json:"check_performance"`
	CheckIntegrity         bool                   `json:"check_integrity"`
	PerformanceThresholds  *PerformanceThresholds `json:"performance_thresholds,omitempty"`
	IgnoreHiddenFiles      bool                   `json:"ignore_hidden_files"`
	MaxConcurrency         int                    `json:"max_concurrency"`
	DetailedErrorReporting bool                   `json:"detailed_error_reporting"`
}

// NewDirectoryValidator creates a new directory validator
func NewDirectoryValidator(logger *logrus.Entry) *DirectoryValidator {
	return &DirectoryValidator{
		logger: logger,
		performanceThresholds: PerformanceThresholds{
			MaxAPICalls:       1000,
			MinCacheHitRate:   0.8,
			MaxMemoryMB:       500,
			MaxProcessingTime: 10 * time.Minute,
			MinThroughputMBps: 1.0,
		},
	}
}

// SetPerformanceThresholds updates the performance validation thresholds
func (dv *DirectoryValidator) SetPerformanceThresholds(thresholds PerformanceThresholds) {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	dv.performanceThresholds = thresholds
}

// GetPerformanceThresholds returns the current performance validation thresholds
func (dv *DirectoryValidator) GetPerformanceThresholds() PerformanceThresholds {
	dv.mu.RLock()
	defer dv.mu.RUnlock()
	return dv.performanceThresholds
}

// ValidateSyncResults compares source and destination directories to ensure sync was successful
func (dv *DirectoryValidator) ValidateSyncResults(ctx context.Context, sourceDir, destDir string, dirMapping config.DirectoryMapping, opts ValidationOptions) (*ValidationResult, error) {
	startTime := time.Now()

	logger := dv.logger.WithFields(logrus.Fields{
		"source_dir": sourceDir,
		"dest_dir":   destDir,
		"operation":  "validate_sync_results",
	})

	logger.Info("Starting sync results validation")

	result := &ValidationResult{
		Valid:   true,
		Errors:  []string{},
		Summary: ValidationSummary{},
	}

	// Set up exclusion engine for this directory mapping
	dv.exclusionEngine = NewExclusionEngineWithIncludes(dirMapping.Exclude, dirMapping.IncludeOnly)

	// Validate that source directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("source directory validation failed: %w", err)
	}

	// Validate that destination directory exists
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("destination directory validation failed: %w", err)
	}

	// Discover files in both directories
	sourceFiles, err := dv.discoverFiles(ctx, sourceDir, dirMapping, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to discover source files: %w", err)
	}

	destFiles, err := dv.discoverFiles(ctx, destDir, dirMapping, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to discover destination files: %w", err)
	}

	result.Summary.TotalFiles = len(sourceFiles)

	// Validate file mappings
	if opts.CheckStructure {
		dv.validateDirectoryStructure(sourceFiles, destFiles, dirMapping, result, logger)
	}

	// Validate file content
	if opts.CheckContent {
		dv.validateFileContent(ctx, sourceDir, destDir, sourceFiles, destFiles, dirMapping, result, opts, logger)
	}

	// Validate exclusions
	if opts.CheckExclusions {
		dv.validateExclusions(sourceFiles, destFiles, dirMapping, result, logger)
	}

	// Complete validation summary
	result.Summary.Duration = time.Since(startTime)
	result.Summary.ValidFiles = result.Summary.TotalFiles - result.Summary.InvalidFiles

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	logger.WithFields(logrus.Fields{
		"valid":         result.Valid,
		"total_files":   result.Summary.TotalFiles,
		"invalid_files": result.Summary.InvalidFiles,
		"duration_ms":   result.Summary.Duration.Milliseconds(),
	}).Info("Sync results validation completed")

	return result, nil
}

// ValidateTransformApplication verifies that transforms were applied correctly to directory files
func (dv *DirectoryValidator) ValidateTransformApplication(_ context.Context, originalFiles, transformedFiles map[string]string, transform config.Transform, _ ValidationOptions) (*ValidationResult, error) {
	startTime := time.Now()

	logger := dv.logger.WithFields(logrus.Fields{
		"operation":         "validate_transform_application",
		"original_count":    len(originalFiles),
		"transformed_count": len(transformedFiles),
	})

	logger.Info("Starting transform application validation")

	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
		Summary: ValidationSummary{
			TotalFiles: len(originalFiles),
		},
	}

	// Check that all original files have corresponding transformed files
	for originalPath, originalContent := range originalFiles {
		transformedContent, exists := transformedFiles[originalPath]
		if !exists {
			result.Errors = append(result.Errors, fmt.Sprintf("missing transformed file: %s", originalPath))
			result.Summary.MissingFiles++
			continue
		}

		// Validate transform was applied correctly
		if err := dv.validateTransformResult(originalContent, transformedContent, transform, originalPath); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("transform validation failed for %s: %v", originalPath, err))
			result.Summary.TransformErrors++
		}
	}

	// Check for extra files that shouldn't exist
	for transformedPath := range transformedFiles {
		if _, exists := originalFiles[transformedPath]; !exists {
			result.Errors = append(result.Errors, fmt.Sprintf("unexpected transformed file: %s", transformedPath))
			result.Summary.ExtraFiles++
		}
	}

	result.Summary.Duration = time.Since(startTime)
	result.Summary.InvalidFiles = result.Summary.MissingFiles + result.Summary.ExtraFiles + result.Summary.TransformErrors
	result.Summary.ValidFiles = result.Summary.TotalFiles - result.Summary.InvalidFiles

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	logger.WithFields(logrus.Fields{
		"valid":            result.Valid,
		"transform_errors": result.Summary.TransformErrors,
		"missing_files":    result.Summary.MissingFiles,
		"extra_files":      result.Summary.ExtraFiles,
		"duration_ms":      result.Summary.Duration.Milliseconds(),
	}).Info("Transform application validation completed")

	return result, nil
}

// ValidateExclusionCompliance ensures excluded files were properly filtered out
func (dv *DirectoryValidator) ValidateExclusionCompliance(ctx context.Context, sourceDir, destDir string, dirMapping config.DirectoryMapping, _ ValidationOptions) (*ValidationResult, error) {
	startTime := time.Now()

	logger := dv.logger.WithFields(logrus.Fields{
		"source_dir":      sourceDir,
		"dest_dir":        destDir,
		"exclusion_count": len(dirMapping.Exclude),
		"operation":       "validate_exclusion_compliance",
	})

	logger.Info("Starting exclusion compliance validation")

	result := &ValidationResult{
		Valid:   true,
		Errors:  []string{},
		Summary: ValidationSummary{},
	}

	// Set up exclusion engine
	dv.exclusionEngine = NewExclusionEngineWithIncludes(dirMapping.Exclude, dirMapping.IncludeOnly)

	// Walk destination directory and check for excluded files
	err := filepath.WalkDir(destDir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Handle context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate relative path
		relPath, err := filepath.Rel(destDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		result.Summary.TotalFiles++

		// Check if this file should have been excluded
		if dv.exclusionEngine.IsExcluded(relPath) {
			result.Errors = append(result.Errors, fmt.Sprintf("excluded file found in destination: %s", relPath))
			result.Summary.ExclusionErrors++
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk destination directory: %w", err)
	}

	result.Summary.Duration = time.Since(startTime)
	result.Summary.InvalidFiles = result.Summary.ExclusionErrors
	result.Summary.ValidFiles = result.Summary.TotalFiles - result.Summary.InvalidFiles

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	logger.WithFields(logrus.Fields{
		"valid":            result.Valid,
		"exclusion_errors": result.Summary.ExclusionErrors,
		"total_files":      result.Summary.TotalFiles,
		"duration_ms":      result.Summary.Duration.Milliseconds(),
	}).Info("Exclusion compliance validation completed")

	return result, nil
}

// ValidateDirectoryStructure verifies that directory structure was preserved correctly
func (dv *DirectoryValidator) ValidateDirectoryStructure(ctx context.Context, sourceDir, destDir string, dirMapping config.DirectoryMapping, opts ValidationOptions) (*ValidationResult, error) {
	startTime := time.Now()

	logger := dv.logger.WithFields(logrus.Fields{
		"source_dir":         sourceDir,
		"dest_dir":           destDir,
		"preserve_structure": dirMapping.PreserveStructure,
		"operation":          "validate_directory_structure",
	})

	logger.Info("Starting directory structure validation")

	result := &ValidationResult{
		Valid:   true,
		Errors:  []string{},
		Summary: ValidationSummary{},
	}

	// Set up exclusion engine for this directory mapping
	dv.exclusionEngine = NewExclusionEngineWithIncludes(dirMapping.Exclude, dirMapping.IncludeOnly)

	// Determine if structure should be preserved
	preserveStructure := true
	if dirMapping.PreserveStructure != nil {
		preserveStructure = *dirMapping.PreserveStructure
	}

	// Discover source files
	sourceFiles, err := dv.discoverFiles(ctx, sourceDir, dirMapping, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to discover source files: %w", err)
	}

	// Discover destination files
	destFiles, err := dv.discoverFiles(ctx, destDir, dirMapping, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to discover destination files: %w", err)
	}

	result.Summary.TotalFiles = len(sourceFiles)

	// Validate structure preservation
	dv.validateDirectoryStructure(sourceFiles, destFiles, dirMapping, result, logger)

	// If structure should be flattened, validate that
	if !preserveStructure {
		dv.validateFlattenedStructure(sourceFiles, destFiles, result, logger)
	}

	result.Summary.Duration = time.Since(startTime)
	result.Summary.InvalidFiles = result.Summary.StructureErrors
	result.Summary.ValidFiles = result.Summary.TotalFiles - result.Summary.InvalidFiles

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	logger.WithFields(logrus.Fields{
		"valid":            result.Valid,
		"structure_errors": result.Summary.StructureErrors,
		"total_files":      result.Summary.TotalFiles,
		"duration_ms":      result.Summary.Duration.Milliseconds(),
	}).Info("Directory structure validation completed")

	return result, nil
}

// ValidateFileIntegrity checks file content matches and wasn't corrupted during sync
func (dv *DirectoryValidator) ValidateFileIntegrity(_ context.Context, sourceFiles, destFiles map[string]string, opts ValidationOptions) (*ValidationResult, error) {
	startTime := time.Now()

	logger := dv.logger.WithFields(logrus.Fields{
		"source_count": len(sourceFiles),
		"dest_count":   len(destFiles),
		"operation":    "validate_file_integrity",
	})

	logger.Info("Starting file integrity validation")

	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
		Summary: ValidationSummary{
			TotalFiles: len(sourceFiles),
		},
	}

	// Set up worker pool for concurrent validation
	maxConcurrency := opts.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = runtime.NumCPU()
	}

	// Create job channel and results channel
	jobs := make(chan integrityJob, len(sourceFiles))
	results := make(chan FileValidationError, len(sourceFiles))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				dv.validateFileIntegrityJob(job, results)
			}
		}()
	}

	// Submit jobs
	go func() {
		defer close(jobs)
		for sourcePath, sourceContent := range sourceFiles {
			destContent, exists := destFiles[sourcePath]
			jobs <- integrityJob{
				sourcePath:    sourcePath,
				sourceContent: sourceContent,
				destContent:   destContent,
				exists:        exists,
			}
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	for validationError := range results {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", validationError.ErrorType, validationError.Details))
		switch validationError.ErrorType {
		case "missing_file":
			result.Summary.MissingFiles++
		case "content_mismatch":
			result.Summary.ContentMismatches++
		}
	}

	result.Summary.Duration = time.Since(startTime)
	result.Summary.InvalidFiles = result.Summary.MissingFiles + result.Summary.ContentMismatches
	result.Summary.ValidFiles = result.Summary.TotalFiles - result.Summary.InvalidFiles

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	logger.WithFields(logrus.Fields{
		"valid":              result.Valid,
		"missing_files":      result.Summary.MissingFiles,
		"content_mismatches": result.Summary.ContentMismatches,
		"duration_ms":        result.Summary.Duration.Milliseconds(),
	}).Info("File integrity validation completed")

	return result, nil
}

// ValidateValidationPerformanceMetrics verifies sync performance meets expected targets
func (dv *DirectoryValidator) ValidateValidationPerformanceMetrics(_ context.Context, metrics ValidationPerformanceMetrics, opts ValidationOptions) (*PerformanceValidationResult, error) {
	logger := dv.logger.WithField("operation", "validate_performance_metrics")

	logger.Info("Starting performance metrics validation")

	// Use provided thresholds or defaults
	thresholds := dv.GetPerformanceThresholds()
	if opts.PerformanceThresholds != nil {
		thresholds = *opts.PerformanceThresholds
	}

	result := &PerformanceValidationResult{
		Valid:           true,
		Metrics:         metrics,
		Thresholds:      thresholds,
		Recommendations: []string{},
	}

	// Validate API call efficiency
	result.APICallsOptimized = metrics.APICalls <= thresholds.MaxAPICalls
	if !result.APICallsOptimized {
		result.Valid = false
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Consider optimizing API calls. Used %d, limit is %d", metrics.APICalls, thresholds.MaxAPICalls))
	}

	// Validate cache hit rate
	result.CacheHitRateGood = metrics.CacheHitRate >= thresholds.MinCacheHitRate
	if !result.CacheHitRateGood {
		result.Valid = false
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Cache hit rate %.2f%% is below threshold %.2f%%", metrics.CacheHitRate*100, thresholds.MinCacheHitRate*100))
	}

	// Validate memory usage
	memoryMB := metrics.MemoryUsage / (1024 * 1024)
	result.MemoryUsageAcceptable = memoryMB <= thresholds.MaxMemoryMB
	if !result.MemoryUsageAcceptable {
		result.Valid = false
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Memory usage %dMB exceeds limit %dMB", memoryMB, thresholds.MaxMemoryMB))
	}

	// Validate processing time
	result.ProcessingTimeGood = metrics.ProcessingTime <= thresholds.MaxProcessingTime
	if !result.ProcessingTimeGood {
		result.Valid = false
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Processing time %v exceeds limit %v", metrics.ProcessingTime, thresholds.MaxProcessingTime))
	}

	// Validate throughput
	throughputGood := metrics.ThroughputMBps >= thresholds.MinThroughputMBps
	if !throughputGood {
		result.Valid = false
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Throughput %.2f MB/s is below minimum %.2f MB/s", metrics.ThroughputMBps, thresholds.MinThroughputMBps))
	}

	logger.WithFields(logrus.Fields{
		"valid":           result.Valid,
		"api_calls":       metrics.APICalls,
		"cache_hit_rate":  metrics.CacheHitRate,
		"memory_mb":       memoryMB,
		"processing_time": metrics.ProcessingTime,
		"throughput_mbps": metrics.ThroughputMBps,
		"recommendations": len(result.Recommendations),
	}).Info("Performance metrics validation completed")

	return result, nil
}

// ValidateAPIEfficiency checks that API call optimization targets were met
func (dv *DirectoryValidator) ValidateAPIEfficiency(_ context.Context, apiCalls, expectedMaxCalls int) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:   apiCalls <= expectedMaxCalls,
		Errors:  []string{},
		Summary: ValidationSummary{},
	}

	if !result.Valid {
		result.Errors = append(result.Errors,
			fmt.Sprintf("API call limit exceeded: %d calls made, maximum allowed: %d", apiCalls, expectedMaxCalls))
	}

	dv.logger.WithFields(logrus.Fields{
		"api_calls": apiCalls,
		"max_calls": expectedMaxCalls,
		"efficient": result.Valid,
	}).Info("API efficiency validation completed")

	return result, nil
}

// ValidateCacheUtilization verifies cache hit rates meet expectations
func (dv *DirectoryValidator) ValidateCacheUtilization(_ context.Context, cacheHits, cacheMisses int, expectedHitRate float64) (*ValidationResult, error) {
	totalRequests := cacheHits + cacheMisses
	actualHitRate := 0.0
	if totalRequests > 0 {
		actualHitRate = float64(cacheHits) / float64(totalRequests)
	}

	result := &ValidationResult{
		Valid:   actualHitRate >= expectedHitRate,
		Errors:  []string{},
		Summary: ValidationSummary{},
	}

	if !result.Valid {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Cache hit rate %.2f%% is below expected %.2f%%", actualHitRate*100, expectedHitRate*100))
	}

	dv.logger.WithFields(logrus.Fields{
		"cache_hits":        cacheHits,
		"cache_misses":      cacheMisses,
		"hit_rate":          actualHitRate,
		"expected_hit_rate": expectedHitRate,
		"efficient":         result.Valid,
	}).Info("Cache utilization validation completed")

	return result, nil
}

// ValidateMemoryUsage ensures memory usage stays within expected bounds
func (dv *DirectoryValidator) ValidateMemoryUsage(_ context.Context, memoryUsage, maxMemoryBytes int64) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  memoryUsage <= maxMemoryBytes,
		Errors: []string{},
		Summary: ValidationSummary{
			BytesValidated: memoryUsage,
		},
	}

	if !result.Valid {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Memory usage %d bytes exceeds limit %d bytes", memoryUsage, maxMemoryBytes))
	}

	dv.logger.WithFields(logrus.Fields{
		"memory_usage":  memoryUsage,
		"memory_limit":  maxMemoryBytes,
		"within_bounds": result.Valid,
	}).Info("Memory usage validation completed")

	return result, nil
}

// ValidateProgressReporting verifies progress reporting worked correctly
func (dv *DirectoryValidator) ValidateProgressReporting(_ context.Context, expectedFiles, reportedFiles int, progressUpdates []string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  expectedFiles == reportedFiles,
		Errors: []string{},
		Summary: ValidationSummary{
			TotalFiles: expectedFiles,
		},
	}

	if expectedFiles != reportedFiles {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Progress reporting mismatch: expected %d files, reported %d files", expectedFiles, reportedFiles))
	}

	// Validate that progress updates were generated
	if len(progressUpdates) == 0 && expectedFiles > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "No progress updates were generated")
	}

	dv.logger.WithFields(logrus.Fields{
		"expected_files":   expectedFiles,
		"reported_files":   reportedFiles,
		"progress_updates": len(progressUpdates),
		"accurate":         result.Valid,
	}).Info("Progress reporting validation completed")

	return result, nil
}

// Helper methods

// discoverFiles walks a directory and discovers all files, respecting exclusions
func (dv *DirectoryValidator) discoverFiles(ctx context.Context, dir string, dirMapping config.DirectoryMapping, opts ValidationOptions) (map[string]DiscoveredFile, error) {
	files := make(map[string]DiscoveredFile)
	var mu sync.Mutex

	// Determine if hidden files should be included
	includeHidden := true
	if dirMapping.IncludeHidden != nil {
		includeHidden = *dirMapping.IncludeHidden
	}
	if opts.IgnoreHiddenFiles {
		includeHidden = false
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Handle context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate relative path
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath == "." || d.IsDir() {
			return nil
		}

		// Check for hidden files
		if !includeHidden && dv.isHidden(relPath) {
			return nil
		}

		// Check exclusions
		if dv.exclusionEngine != nil && dv.exclusionEngine.IsExcluded(relPath) {
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			return err
		}

		mu.Lock()
		files[relPath] = DiscoveredFile{
			RelativePath: relPath,
			FullPath:     path,
			Size:         info.Size(),
			IsDir:        d.IsDir(),
		}
		mu.Unlock()

		return nil
	})

	return files, err
}

// validateDirectoryStructure validates that the directory structure is correct
func (dv *DirectoryValidator) validateDirectoryStructure(sourceFiles, destFiles map[string]DiscoveredFile, dirMapping config.DirectoryMapping, result *ValidationResult, _ *logrus.Entry) {
	preserveStructure := true
	if dirMapping.PreserveStructure != nil {
		preserveStructure = *dirMapping.PreserveStructure
	}

	for relPath := range sourceFiles {
		expectedDestPath := relPath
		if !preserveStructure {
			expectedDestPath = filepath.Base(relPath)
		}

		if _, exists := destFiles[expectedDestPath]; !exists {
			result.Errors = append(result.Errors, fmt.Sprintf("missing file in destination: %s", expectedDestPath))
			result.Summary.MissingFiles++
		}
	}

	// Check for extra files in destination
	for destPath := range destFiles {
		expectedSourcePath := destPath
		if !preserveStructure {
			// For flattened structure, need to find source file with same base name
			found := false
			for sourcePath := range sourceFiles {
				if filepath.Base(sourcePath) == destPath {
					found = true
					break
				}
			}
			if !found {
				result.Errors = append(result.Errors, fmt.Sprintf("unexpected file in destination: %s", destPath))
				result.Summary.ExtraFiles++
			}
		} else {
			if _, exists := sourceFiles[expectedSourcePath]; !exists {
				result.Errors = append(result.Errors, fmt.Sprintf("unexpected file in destination: %s", destPath))
				result.Summary.ExtraFiles++
			}
		}
	}
}

// validateFlattenedStructure validates that directory structure was correctly flattened
func (dv *DirectoryValidator) validateFlattenedStructure(_, destFiles map[string]DiscoveredFile, result *ValidationResult, _ *logrus.Entry) {
	// In flattened structure, all files should be at root level
	for destPath := range destFiles {
		if strings.Contains(destPath, string(filepath.Separator)) {
			result.Errors = append(result.Errors, fmt.Sprintf("nested file found in flattened destination: %s", destPath))
			result.Summary.StructureErrors++
		}
	}
}

// validateFileContent validates that file contents match between source and destination
func (dv *DirectoryValidator) validateFileContent(_ context.Context, _, _ string, sourceFiles, destFiles map[string]DiscoveredFile, _ config.DirectoryMapping, result *ValidationResult, _ ValidationOptions, logger *logrus.Entry) {
	// Implementation depends on whether we're checking raw content or transformed content
	// For now, we'll do a basic size comparison and checksum validation

	for relPath, sourceFile := range sourceFiles {
		destFile, exists := destFiles[relPath]
		if !exists {
			continue // Already handled in structure validation
		}

		// Compare file sizes first (quick check)
		if sourceFile.Size != destFile.Size {
			result.Errors = append(result.Errors, fmt.Sprintf("file size mismatch for %s: source=%d, dest=%d", relPath, sourceFile.Size, destFile.Size))
			result.Summary.ContentMismatches++
			continue
		}

		// Compare file checksums
		sourceChecksum, err := dv.calculateFileChecksum(sourceFile.FullPath)
		if err != nil {
			logger.WithError(err).WithField("file", relPath).Warn("Failed to calculate source checksum")
			continue
		}

		destChecksum, err := dv.calculateFileChecksum(destFile.FullPath)
		if err != nil {
			logger.WithError(err).WithField("file", relPath).Warn("Failed to calculate destination checksum")
			continue
		}

		if sourceChecksum != destChecksum {
			result.Errors = append(result.Errors, fmt.Sprintf("content checksum mismatch for %s", relPath))
			result.Summary.ContentMismatches++
		}

		result.Summary.BytesValidated += sourceFile.Size
	}
}

// validateExclusions validates that exclusion patterns were applied correctly
func (dv *DirectoryValidator) validateExclusions(_, destFiles map[string]DiscoveredFile, _ config.DirectoryMapping, result *ValidationResult, _ *logrus.Entry) {
	// Check that no excluded files made it to destination
	for destPath := range destFiles {
		if dv.exclusionEngine.IsExcluded(destPath) {
			result.Errors = append(result.Errors, fmt.Sprintf("excluded file found in destination: %s", destPath))
			result.Summary.ExclusionErrors++
		}
	}
}

// validateTransformResult validates that a transform was applied correctly
func (dv *DirectoryValidator) validateTransformResult(original, transformed string, transform config.Transform, _ string) error {
	// Basic validation - check that if repo_name transform is enabled, repository names were replaced
	if transform.RepoName {
		// This is a simplified check - in practice, you'd need access to the actual repo names
		if original == transformed {
			return fmt.Errorf("repo_name transform enabled but content unchanged: %w", ErrValidationFailed)
		}
	}

	// Check variable substitutions
	for variable := range transform.Variables {
		placeholder := fmt.Sprintf("{{%s}}", variable)
		if strings.Contains(original, placeholder) && strings.Contains(transformed, placeholder) {
			return fmt.Errorf("variable %s not substituted: %w", variable, ErrTransformValidationFailed)
		}
	}

	// Validate UTF-8 encoding
	if !utf8.ValidString(transformed) {
		return fmt.Errorf("transformed content contains invalid UTF-8: %w", ErrEncodingValidationFailed)
	}

	return nil
}

// validateFileIntegrityJob validates the integrity of a single file
func (dv *DirectoryValidator) validateFileIntegrityJob(job integrityJob, results chan<- FileValidationError) {
	if !job.exists {
		results <- FileValidationError{
			FilePath:  job.sourcePath,
			ErrorType: "missing_file",
			Details:   "File exists in source but not in destination",
		}
		return
	}

	// Compare content checksums
	sourceHash := fmt.Sprintf("%x", sha256.Sum256([]byte(job.sourceContent)))
	destHash := fmt.Sprintf("%x", sha256.Sum256([]byte(job.destContent)))

	if sourceHash != destHash {
		results <- FileValidationError{
			FilePath:  job.sourcePath,
			ErrorType: "content_mismatch",
			Expected:  sourceHash,
			Actual:    destHash,
			Details:   "File content checksums do not match",
		}
	}
}

// calculateFileChecksum calculates MD5 checksum of a file
func (dv *DirectoryValidator) calculateFileChecksum(filePath string) (string, error) {
	// Clean the file path to prevent directory traversal
	cleanPath := filepath.Clean(filePath)
	file, err := os.Open(cleanPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			dv.logger.WithError(closeErr).Warn("Failed to close file")
		}
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// isHidden checks if a file path represents a hidden file
func (dv *DirectoryValidator) isHidden(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && part != "." && part != ".." {
			return true
		}
	}
	return false
}

// DefaultValidationOptions returns default validation options
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		CheckContent:           true,
		CheckStructure:         true,
		CheckExclusions:        true,
		CheckTransforms:        true,
		CheckPerformance:       false,
		CheckIntegrity:         true,
		IgnoreHiddenFiles:      false,
		MaxConcurrency:         runtime.NumCPU(),
		DetailedErrorReporting: true,
	}
}

// ValidateAllAspects performs comprehensive validation of directory sync results
func (dv *DirectoryValidator) ValidateAllAspects(ctx context.Context, sourceDir, destDir string, dirMapping config.DirectoryMapping, opts ValidationOptions) (*ValidationResult, error) {
	logger := dv.logger.WithField("operation", "validate_all_aspects")
	logger.Info("Starting comprehensive directory sync validation")

	// Run all validations and combine results
	syncResult, err := dv.ValidateSyncResults(ctx, sourceDir, destDir, dirMapping, opts)
	if err != nil {
		return nil, fmt.Errorf("sync results validation failed: %w", err)
	}

	structureResult, err := dv.ValidateDirectoryStructure(ctx, sourceDir, destDir, dirMapping, opts)
	if err != nil {
		return nil, fmt.Errorf("directory structure validation failed: %w", err)
	}

	exclusionResult, err := dv.ValidateExclusionCompliance(ctx, sourceDir, destDir, dirMapping, opts)
	if err != nil {
		return nil, fmt.Errorf("exclusion compliance validation failed: %w", err)
	}

	// Combine all results
	combinedResult := &ValidationResult{
		Valid:  syncResult.Valid && structureResult.Valid && exclusionResult.Valid,
		Errors: append(append(syncResult.Errors, structureResult.Errors...), exclusionResult.Errors...),
		Summary: ValidationSummary{
			TotalFiles:        syncResult.Summary.TotalFiles,
			ValidFiles:        syncResult.Summary.ValidFiles,
			InvalidFiles:      syncResult.Summary.InvalidFiles + structureResult.Summary.InvalidFiles + exclusionResult.Summary.InvalidFiles,
			MissingFiles:      syncResult.Summary.MissingFiles,
			ExtraFiles:        syncResult.Summary.ExtraFiles,
			ContentMismatches: syncResult.Summary.ContentMismatches,
			TransformErrors:   syncResult.Summary.TransformErrors,
			StructureErrors:   structureResult.Summary.StructureErrors,
			ExclusionErrors:   exclusionResult.Summary.ExclusionErrors,
			Duration:          syncResult.Summary.Duration + structureResult.Summary.Duration + exclusionResult.Summary.Duration,
			BytesValidated:    syncResult.Summary.BytesValidated,
		},
	}

	logger.WithFields(logrus.Fields{
		"overall_valid": combinedResult.Valid,
		"total_errors":  len(combinedResult.Errors),
		"total_files":   combinedResult.Summary.TotalFiles,
		"invalid_files": combinedResult.Summary.InvalidFiles,
		"duration_ms":   combinedResult.Summary.Duration.Milliseconds(),
	}).Info("Comprehensive directory sync validation completed")

	return combinedResult, nil
}
