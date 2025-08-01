package sync

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DirectoryProgressReporter handles progress reporting for directory operations
type DirectoryProgressReporter struct {
	logger         *logrus.Entry
	threshold      int           // Minimum files to trigger progress reporting
	updateInterval time.Duration // Minimum time between progress updates
	lastUpdate     time.Time
	mu             sync.RWMutex
	enabled        bool
	directoryPath  string
	metrics        DirectoryMetrics
}

// DirectoryMetrics tracks meaningful metrics for directory processing
type DirectoryMetrics struct {
	FilesDiscovered   int
	FilesProcessed    int
	FilesExcluded     int
	FilesSkipped      int
	FilesErrored      int
	DirectoriesWalked int
	TotalSize         int64
	ProcessedSize     int64
	StartTime         time.Time
	EndTime           time.Time
}

// NewDirectoryProgressReporter creates a new directory progress reporter
func NewDirectoryProgressReporter(logger *logrus.Entry, directoryPath string, threshold int) *DirectoryProgressReporter {
	if threshold <= 0 {
		threshold = 50 // Default threshold for progress reporting
	}

	return &DirectoryProgressReporter{
		logger:         logger,
		threshold:      threshold,
		updateInterval: 2 * time.Second, // Update every 2 seconds at most
		directoryPath:  directoryPath,
		metrics: DirectoryMetrics{
			StartTime: time.Now(),
		},
	}
}

// Start initializes progress reporting if the file count exceeds threshold
func (dpr *DirectoryProgressReporter) Start(totalFiles int) {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()

	dpr.enabled = totalFiles >= dpr.threshold
	dpr.metrics.FilesDiscovered = totalFiles

	if dpr.enabled {
		dpr.logger.WithFields(logrus.Fields{
			"directory":   dpr.directoryPath,
			"total_files": totalFiles,
			"threshold":   dpr.threshold,
		}).Info("Directory processing started - progress reporting enabled")
	} else {
		dpr.logger.WithFields(logrus.Fields{
			"directory":   dpr.directoryPath,
			"total_files": totalFiles,
			"threshold":   dpr.threshold,
		}).Debug("Directory processing started - progress reporting disabled (below threshold)")
	}
}

// UpdateProgress updates the current progress with non-blocking behavior
func (dpr *DirectoryProgressReporter) UpdateProgress(current, total int, message string) {
	if !dpr.isEnabled() {
		return
	}

	dpr.mu.Lock()
	defer dpr.mu.Unlock()

	now := time.Now()
	// Rate limit updates to avoid spam
	if now.Sub(dpr.lastUpdate) < dpr.updateInterval {
		return
	}

	dpr.lastUpdate = now
	percentage := float64(current) / float64(total) * 100

	dpr.logger.WithFields(logrus.Fields{
		"directory":      dpr.directoryPath,
		"progress":       current,
		"total":          total,
		"percentage":     percentage,
		"message":        message,
		"files_excluded": dpr.metrics.FilesExcluded,
		"files_errored":  dpr.metrics.FilesErrored,
	}).Info("Directory processing progress")
}

// RecordFileDiscovered increments the files discovered counter
func (dpr *DirectoryProgressReporter) RecordFileDiscovered() {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.metrics.FilesDiscovered++
}

// RecordFileProcessed increments the files processed counter
func (dpr *DirectoryProgressReporter) RecordFileProcessed(size int64) {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.metrics.FilesProcessed++
	dpr.metrics.ProcessedSize += size
}

// RecordFileExcluded increments the files excluded counter
func (dpr *DirectoryProgressReporter) RecordFileExcluded() {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.metrics.FilesExcluded++
}

// RecordFileSkipped increments the files skipped counter
func (dpr *DirectoryProgressReporter) RecordFileSkipped() {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.metrics.FilesSkipped++
}

// RecordFileError increments the files errored counter
func (dpr *DirectoryProgressReporter) RecordFileError() {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.metrics.FilesErrored++
}

// RecordDirectoryWalked increments the directories walked counter
func (dpr *DirectoryProgressReporter) RecordDirectoryWalked() {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.metrics.DirectoriesWalked++
}

// AddTotalSize adds to the total size counter
func (dpr *DirectoryProgressReporter) AddTotalSize(size int64) {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.metrics.TotalSize += size
}

// Complete marks the directory processing as complete and reports final metrics
func (dpr *DirectoryProgressReporter) Complete() DirectoryMetrics {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()

	dpr.metrics.EndTime = time.Now()
	duration := dpr.metrics.EndTime.Sub(dpr.metrics.StartTime)

	fields := logrus.Fields{
		"directory":            dpr.directoryPath,
		"files_discovered":     dpr.metrics.FilesDiscovered,
		"files_processed":      dpr.metrics.FilesProcessed,
		"files_excluded":       dpr.metrics.FilesExcluded,
		"files_skipped":        dpr.metrics.FilesSkipped,
		"files_errored":        dpr.metrics.FilesErrored,
		"directories_walked":   dpr.metrics.DirectoriesWalked,
		"total_size_bytes":     dpr.metrics.TotalSize,
		"processed_size_bytes": dpr.metrics.ProcessedSize,
		"duration":             duration,
	}

	if dpr.enabled {
		dpr.logger.WithFields(fields).Info("Directory processing completed")
	} else {
		dpr.logger.WithFields(fields).Debug("Directory processing completed")
	}

	return dpr.metrics
}

// isEnabled checks if progress reporting is enabled
func (dpr *DirectoryProgressReporter) isEnabled() bool {
	dpr.mu.RLock()
	defer dpr.mu.RUnlock()
	return dpr.enabled
}

// GetMetrics returns the current metrics (thread-safe)
func (dpr *DirectoryProgressReporter) GetMetrics() DirectoryMetrics {
	dpr.mu.RLock()
	defer dpr.mu.RUnlock()
	return dpr.metrics
}

// SetThreshold updates the threshold for progress reporting
func (dpr *DirectoryProgressReporter) SetThreshold(threshold int) {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.threshold = threshold
}

// SetUpdateInterval updates the minimum interval between progress updates
func (dpr *DirectoryProgressReporter) SetUpdateInterval(interval time.Duration) {
	dpr.mu.Lock()
	defer dpr.mu.Unlock()
	dpr.updateInterval = interval
}

// IsProgressReportingNeeded determines if progress reporting should be enabled
func IsProgressReportingNeeded(fileCount, threshold int) bool {
	if threshold <= 0 {
		threshold = 50
	}
	return fileCount >= threshold
}

// DirectoryProgressReporterOptions configures directory progress reporting
type DirectoryProgressReporterOptions struct {
	Threshold      int           // Minimum files to trigger progress reporting
	UpdateInterval time.Duration // Minimum time between updates
	Enabled        bool          // Force enable/disable reporting
}

// NewDirectoryProgressReporterWithOptions creates a progress reporter with custom options
func NewDirectoryProgressReporterWithOptions(logger *logrus.Entry, directoryPath string, opts DirectoryProgressReporterOptions) *DirectoryProgressReporter {
	reporter := NewDirectoryProgressReporter(logger, directoryPath, opts.Threshold)

	if opts.UpdateInterval > 0 {
		reporter.SetUpdateInterval(opts.UpdateInterval)
	}

	if opts.Enabled {
		reporter.mu.Lock()
		reporter.enabled = true
		reporter.mu.Unlock()
	}

	return reporter
}

// BatchProgressWrapper wraps the DirectoryProgressReporter to implement ProgressReporter interface
type BatchProgressWrapper struct {
	reporter *DirectoryProgressReporter
}

// NewBatchProgressWrapper creates a wrapper for batch processing progress reporting
func NewBatchProgressWrapper(reporter *DirectoryProgressReporter) *BatchProgressWrapper {
	return &BatchProgressWrapper{
		reporter: reporter,
	}
}

// UpdateProgress implements ProgressReporter interface for batch processing
func (bpw *BatchProgressWrapper) UpdateProgress(current, total int, message string) {
	if bpw.reporter != nil {
		bpw.reporter.UpdateProgress(current, total, message)
	}
}

// DirectoryProgressManager manages multiple directory progress reporters
type DirectoryProgressManager struct {
	reporters map[string]*DirectoryProgressReporter
	mu        sync.RWMutex
	logger    *logrus.Entry
}

// NewDirectoryProgressManager creates a new progress manager
func NewDirectoryProgressManager(logger *logrus.Entry) *DirectoryProgressManager {
	return &DirectoryProgressManager{
		reporters: make(map[string]*DirectoryProgressReporter),
		logger:    logger,
	}
}

// GetReporter gets or creates a progress reporter for a directory
func (dpm *DirectoryProgressManager) GetReporter(directoryPath string, threshold int) *DirectoryProgressReporter {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	if reporter, exists := dpm.reporters[directoryPath]; exists {
		return reporter
	}

	reporter := NewDirectoryProgressReporter(dpm.logger, directoryPath, threshold)
	dpm.reporters[directoryPath] = reporter
	return reporter
}

// CompleteAll completes all active reporters and returns their metrics
func (dpm *DirectoryProgressManager) CompleteAll() map[string]DirectoryMetrics {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	results := make(map[string]DirectoryMetrics)
	for path, reporter := range dpm.reporters {
		results[path] = reporter.Complete()
	}

	// Clear reporters after completion
	dpm.reporters = make(map[string]*DirectoryProgressReporter)
	return results
}

// GetAllMetrics returns current metrics for all reporters
func (dpm *DirectoryProgressManager) GetAllMetrics() map[string]DirectoryMetrics {
	dpm.mu.RLock()
	defer dpm.mu.RUnlock()

	results := make(map[string]DirectoryMetrics)
	for path, reporter := range dpm.reporters {
		results[path] = reporter.GetMetrics()
	}
	return results
}
