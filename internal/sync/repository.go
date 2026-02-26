package sync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/ai"
	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/metrics"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// Static error variables
var (
	ErrSourceDirectoryNotExistForMetrics       = errors.New("source directory does not exist for metrics processing")
	ErrAllDirectoryProcessingWithMetricsFailed = errors.New("all directory processing with metrics failed")
)

// Constants
const (
	mainBranch = "master"
)

// RepositorySync handles synchronization for a single repository
type RepositorySync struct {
	engine      *Engine
	target      config.TargetConfig
	sourceState *state.SourceState
	targetState *state.TargetState
	logger      *logrus.Entry
	tempDir     string
	// stagedRepoPath is the path to the cloned repo with staged changes, used for AI diff generation
	stagedRepoPath string
	// Performance metrics tracking
	syncMetrics *PerformanceMetrics
	// commitAIGenerated tracks if commit message was AI-generated (for PR metadata)
	commitAIGenerated bool
	// moduleUpdates tracks module version updates for go.mod references
	moduleUpdates []ModuleUpdateInfo
	// lastPRNumber stores the PR number after creation/update for metrics recording
	lastPRNumber *int
	// lastPRURL stores the PR URL after creation/update for metrics recording
	lastPRURL string
}

// PerformanceMetrics tracks performance metrics for the entire sync operation
type PerformanceMetrics struct {
	StartTime          time.Time
	EndTime            time.Time
	DirectoryMetrics   map[string]DirectoryMetrics // keyed by source directory path
	directoryMetricsMu sync.RWMutex                // Protects DirectoryMetrics map access
	FileMetrics        FileProcessingMetrics
	APICallsSaved      int // Total API calls saved by using tree API or caching
	CacheHits          int // Number of cache hits
	CacheMisses        int // Number of cache misses
	TotalAPIRequests   int // Total API requests made
}

// GetDirectoryMetric returns a copy of the directory metrics for the given path (thread-safe).
func (pm *PerformanceMetrics) GetDirectoryMetric(dirPath string) (DirectoryMetrics, bool) {
	pm.directoryMetricsMu.RLock()
	defer pm.directoryMetricsMu.RUnlock()
	metrics, exists := pm.DirectoryMetrics[dirPath]
	return metrics, exists
}

// SetDirectoryMetric sets the directory metrics for the given path (thread-safe).
func (pm *PerformanceMetrics) SetDirectoryMetric(dirPath string, metrics DirectoryMetrics) {
	pm.directoryMetricsMu.Lock()
	defer pm.directoryMetricsMu.Unlock()
	pm.DirectoryMetrics[dirPath] = metrics
}

// IterateDirectoryMetrics safely iterates over all directory metrics (thread-safe).
// The callback receives a copy of each metric, so modifications won't affect the original.
func (pm *PerformanceMetrics) IterateDirectoryMetrics(fn func(dirPath string, metrics DirectoryMetrics)) {
	pm.directoryMetricsMu.RLock()
	defer pm.directoryMetricsMu.RUnlock()
	for dirPath, metrics := range pm.DirectoryMetrics {
		fn(dirPath, metrics)
	}
}

// FileProcessingMetrics tracks metrics for individual file processing
type FileProcessingMetrics struct {
	FilesProcessed       int   // Total files discovered/examined (for compatibility)
	FilesChanged         int   // Files that actually changed in git (what appears in PR)
	FilesAttempted       int   // Files go-broadcast attempted to change (before git filtering)
	FilesSkipped         int   // Files skipped during processing
	FilesDeleted         int   // Files that were deleted from target repositories
	ProcessingTimeMs     int64 // Time spent processing files
	FilesActuallyChanged int   // Alias for FilesChanged for clarity
}

// Execute performs the complete sync operation for this repository
func (rs *RepositorySync) Execute(ctx context.Context) error {
	// Initialize performance metrics tracking
	rs.syncMetrics = &PerformanceMetrics{
		StartTime:        time.Now(),
		DirectoryMetrics: make(map[string]DirectoryMetrics),
	}

	// Track variables for deferred metrics recording
	var (
		finalBranchName    string
		finalCommitSHA     string
		finalAllChanges    []FileChange
		finalActualChanges []string
		finalErr           error
		finalStatus        string // explicit override for early returns (skipped, no_changes)
	)

	// Defer metrics recording (captures success or failure)
	defer func() {
		if rs.engine.syncRepo != nil {
			metricsCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if err := rs.recordTargetResult(metricsCtx, finalBranchName, finalCommitSHA,
				finalAllChanges, finalActualChanges, finalErr, finalStatus); err != nil {
				rs.logger.WithError(err).Warn("Failed to record target result metrics")
			}
		}
	}()

	// Start overall operation timer
	syncTimer := metrics.StartTimer(ctx, rs.logger, "repository_sync").
		AddField(logging.StandardFields.SourceRepo, rs.sourceState.Repo).
		AddField(logging.StandardFields.TargetRepo, rs.target.Repo).
		AddField("sync_branch", rs.sourceState.Branch).
		AddField("commit_sha", rs.sourceState.LatestCommit)
	// Add group context if available
	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		syncTimer = syncTimer.
			AddField("group_name", currentGroup.Name).
			AddField("group_id", currentGroup.ID)
	}

	// 1. Check if sync is actually needed
	syncCheckTimer := metrics.StartTimer(ctx, rs.logger, "sync_check")
	needsSync := rs.engine.options.Force || rs.needsSync()
	syncCheckTimer.AddField("force_sync", rs.engine.options.Force).
		AddField("needs_sync", needsSync).Stop()

	if !needsSync {
		rs.logger.Info("Repository is up-to-date, skipping sync")
		syncTimer.AddField(logging.StandardFields.Status, "skipped").Stop()
		finalStatus = TargetStatusSkipped
		return nil
	}

	// 2. Pre-sync validation and cleanup
	validationTimer := metrics.StartTimer(ctx, rs.logger, "pre_sync_validation")
	if err := rs.validateAndCleanupOrphanedBranches(ctx); err != nil {
		validationTimer.StopWithError(err)
		rs.logger.WithError(err).Warn("Pre-sync validation completed with warnings")
		// Don't fail sync for cleanup issues, just log them
	} else {
		validationTimer.Stop()
	}

	// 3. Create temporary directory
	tempDirTimer := metrics.StartTimer(ctx, rs.logger, "temp_dir_creation")
	if err := rs.createTempDir(); err != nil {
		tempDirTimer.StopWithError(err)
		syncTimer.StopWithError(err)
		finalErr = err
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	tempDirTimer.AddField("temp_dir", rs.tempDir).Stop()
	defer rs.cleanup()

	// 3. Clone source repository
	cloneTimer := metrics.StartTimer(ctx, rs.logger, "source_clone").
		AddField(logging.StandardFields.SourceRepo, rs.sourceState.Repo).
		AddField("source_branch", rs.sourceState.Branch).
		AddField("commit_sha", rs.sourceState.LatestCommit)

	if err := rs.cloneSource(ctx); err != nil {
		cloneTimer.StopWithError(err)
		syncTimer.StopWithError(err)
		finalErr = err
		return fmt.Errorf("failed to clone source: %w", err)
	}
	cloneTimer.Stop()

	// 4. Process and transform files
	processTimer := metrics.StartTimer(ctx, rs.logger, "file_processing").
		AddField("file_count", len(rs.target.Files))

	fileProcessingStart := time.Now()
	changedFiles, err := rs.processFiles(ctx)
	if err != nil {
		processTimer.StopWithError(err)
		syncTimer.StopWithError(err)
		finalErr = err
		return fmt.Errorf("failed to process files: %w", err)
	}
	fileProcessingDuration := time.Since(fileProcessingStart)

	// Count deleted files
	deletedFileCount := 0
	for _, fileChange := range changedFiles {
		if fileChange.IsDeleted {
			deletedFileCount++
		}
	}

	// Store initial file processing metrics (will be updated after directory processing)
	rs.syncMetrics.FileMetrics = FileProcessingMetrics{
		FilesProcessed:   len(rs.target.Files),
		FilesChanged:     len(changedFiles),
		FilesSkipped:     len(rs.target.Files) - len(changedFiles),
		FilesDeleted:     deletedFileCount,
		ProcessingTimeMs: fileProcessingDuration.Milliseconds(),
	}

	processTimer.AddField("changed_files", len(changedFiles)).Stop()

	// 5. Process directories with metrics collection
	directoryChanges, directoryMetrics, err := rs.processDirectoriesWithMetrics(ctx)
	if err != nil {
		syncTimer.StopWithError(err)
		finalErr = err
		return fmt.Errorf("failed to process directories: %w", err)
	}

	// Store directory metrics for PR metadata
	if rs.syncMetrics != nil {
		rs.syncMetrics.DirectoryMetrics = directoryMetrics
	}

	// Combine file and directory changes
	allChanges := append(changedFiles, directoryChanges...)

	// Update file processing metrics to reflect the complete picture
	// This ensures metrics accurately represent what was actually processed
	totalDirectoryFiles := 0
	if rs.syncMetrics.DirectoryMetrics != nil {
		for _, metrics := range rs.syncMetrics.DirectoryMetrics {
			totalDirectoryFiles += metrics.FilesProcessed
		}
	}

	// Store attempted changes for metrics calculation after commit
	totalFilesProcessed := rs.syncMetrics.FileMetrics.FilesProcessed + totalDirectoryFiles
	filesAttempted := len(allChanges)

	rs.logger.WithFields(logrus.Fields{
		"file_changes":      len(changedFiles),
		"directory_changes": len(directoryChanges),
		"total_changes":     len(allChanges),
	}).Info("File and directory processing completed")

	if len(allChanges) == 0 {
		rs.logger.Info("No file or directory changes detected, skipping sync")
		syncTimer.AddField(logging.StandardFields.Status, "no_changes").Stop()
		finalStatus = TargetStatusNoChanges
		return nil
	}

	// 6. Create sync branch (or use existing one)
	branchTimer := metrics.StartTimer(ctx, rs.logger, "branch_creation")
	branchName := rs.createSyncBranch(ctx)
	branchTimer.AddField(logging.StandardFields.BranchName, branchName).Stop()

	// 7. Commit changes
	commitTimer := metrics.StartTimer(ctx, rs.logger, "commit_creation").
		AddField(logging.StandardFields.BranchName, branchName).
		AddField("changed_files", len(allChanges))

	commitSHA, actualChangedFiles, err := rs.commitChanges(ctx, branchName, allChanges)
	// Capture for metrics recording
	finalBranchName = branchName
	finalCommitSHA = commitSHA
	finalAllChanges = allChanges
	finalActualChanges = actualChangedFiles
	if err != nil {
		commitTimer.StopWithError(err)
		// Check if it's because there are no changes to sync
		if errors.Is(err, internalerrors.ErrNoChangesToSync) {
			rs.logger.WithFields(logrus.Fields{
				"branch":        branchName,
				"files_checked": len(allChanges),
			}).Info("Repository is already synchronized - no PR needed")
			// Successfully complete sync without creating PR
			syncTimer.AddField("status", "up_to_date").Stop()
			finalStatus = TargetStatusNoChanges
			return nil
		}
		syncTimer.StopWithError(err)
		finalErr = err
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	commitTimer.AddField("commit_sha", commitSHA).Stop()

	// Update FileMetrics with actual git changes (not just attempted changes)
	rs.syncMetrics.FileMetrics = FileProcessingMetrics{
		FilesProcessed:       totalFilesProcessed,
		FilesChanged:         len(actualChangedFiles), // Files that actually changed in git commit
		FilesAttempted:       filesAttempted,          // Files go-broadcast attempted to change
		FilesSkipped:         totalFilesProcessed - filesAttempted,
		FilesDeleted:         deletedFileCount, // Keep the original count from first metrics
		ProcessingTimeMs:     fileProcessingDuration.Milliseconds(),
		FilesActuallyChanged: len(actualChangedFiles), // Alias for clarity
	}

	rs.logger.WithFields(logrus.Fields{
		"files_processed":        totalFilesProcessed,
		"files_attempted":        filesAttempted,
		"files_actually_changed": len(actualChangedFiles),
		"files_skipped":          totalFilesProcessed - filesAttempted,
	}).Info("Updated metrics with actual git changes")

	// Update directory metrics with actual git changes
	rs.updateDirectoryMetricsWithActualChanges(actualChangedFiles)

	// 8. Push changes (unless dry-run)
	if !rs.engine.options.DryRun {
		pushTimer := metrics.StartTimer(ctx, rs.logger, "branch_push").
			AddField(logging.StandardFields.BranchName, branchName).
			AddField("commit_sha", commitSHA)

		if rs.logger != nil {
			rs.logger.Info("Pushing changes to remote...")
		}
		if err := rs.pushChanges(ctx, branchName); err != nil {
			pushTimer.StopWithError(err)
			syncTimer.StopWithError(err)
			finalErr = err
			return fmt.Errorf("failed to push changes: %w", err)
		}
		pushTimer.Stop()
	} else {
		rs.logger.Debug("DRY-RUN: Skipping branch push")
	}

	// 9. Create or update pull request
	prTimer := metrics.StartTimer(ctx, rs.logger, "pr_management").
		AddField(logging.StandardFields.BranchName, branchName).
		AddField("commit_sha", commitSHA).
		AddField("changed_files", len(allChanges))

	if err := rs.createOrUpdatePR(ctx, branchName, commitSHA, allChanges, actualChangedFiles); err != nil {
		prTimer.StopWithError(err)
		syncTimer.StopWithError(err)
		finalErr = err
		return fmt.Errorf("failed to create/update PR: %w", err)
	}
	prTimer.Stop()

	// Finalize performance metrics
	rs.syncMetrics.EndTime = time.Now()

	if rs.engine.options.DryRun {
		rs.logger.Debug("Dry-run completed successfully")

		out := NewDryRunOutput(nil)
		out.Success("DRY-RUN SUMMARY: Repository sync preview completed successfully")
		// Add group context if available
		if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
			out.Info(fmt.Sprintf("📋 Group: %s (%s)", currentGroup.Name, currentGroup.ID))
		}
		out.Info(fmt.Sprintf("📁 Repository: %s", rs.target.Repo))
		out.Info(fmt.Sprintf("🌿 Branch: %s", branchName))
		out.Info(fmt.Sprintf("📝 Files: %d would be changed", len(allChanges)))
		out.Info(fmt.Sprintf("🔗 Commit: %s", commitSHA))
		out.Info("💡 Run without --dry-run to execute these changes")
		_, _ = fmt.Fprintln(out.writer)
	} else {
		rs.logger.WithField("branch", branchName).Info("Repository sync completed")
	}

	syncTimer.AddField(logging.StandardFields.Status, "completed").
		AddField(logging.StandardFields.BranchName, branchName).
		AddField("final_commit_sha", commitSHA).
		AddField("total_changed_files", len(allChanges)).Stop()

	return nil
}

// needsSync determines if this repository actually needs synchronization
func (rs *RepositorySync) needsSync() bool {
	if rs.targetState == nil {
		return true // No state means never synced
	}

	// Check if source commit is different from last synced commit
	return rs.targetState.LastSyncCommit != rs.sourceState.LatestCommit
}

// validateAndCleanupOrphanedBranches checks for and cleans up orphaned sync branches
func (rs *RepositorySync) validateAndCleanupOrphanedBranches(ctx context.Context) error {
	rs.logger.Debug("Running pre-sync validation for orphaned branches")

	// List all branches in the target repository
	branches, err := rs.engine.gh.ListBranches(ctx, rs.target.Repo)
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	// Look for orphaned sync branches (branches that match our pattern but have no PR)
	orphanedBranches := make([]string, 0)
	syncBranchPrefix := rs.getBranchPrefix()

	for _, branch := range branches {
		// Check if this is a sync branch (matches our prefix pattern)
		if strings.HasPrefix(branch.Name, syncBranchPrefix) {
			// Check if there's an existing PR for this branch
			if existingPR := rs.findExistingPRForBranch(branch.Name); existingPR == nil {
				orphanedBranches = append(orphanedBranches, branch.Name)
			}
		}
	}

	// Clean up orphaned branches
	if len(orphanedBranches) > 0 {
		rs.logger.WithField("orphaned_branches", len(orphanedBranches)).Info("Found orphaned sync branches, cleaning up")

		for _, branchName := range orphanedBranches {
			rs.logger.WithField("branch_name", branchName).Debug("Deleting orphaned sync branch")
			if err := rs.engine.gh.DeleteBranch(ctx, rs.target.Repo, branchName); err != nil {
				if !errors.Is(err, gh.ErrBranchNotFound) {
					rs.logger.WithError(err).WithField("branch_name", branchName).Warn("Failed to delete orphaned branch")
				}
			} else {
				rs.logger.WithField("branch_name", branchName).Info("Deleted orphaned sync branch")
			}
		}
	}

	return nil
}

// findExistingPRForBranch finds an existing PR for the specified branch name
func (rs *RepositorySync) findExistingPRForBranch(branchName string) *gh.PR {
	if rs.targetState == nil {
		return nil
	}

	for _, pr := range rs.targetState.OpenPRs {
		if pr.Head.Ref == branchName {
			return &pr
		}
	}
	return nil
}

// getBranchPrefix returns the branch prefix used for sync branches
func (rs *RepositorySync) getBranchPrefix() string {
	// Check if we have a current group with branch prefix
	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil && currentGroup.Defaults.BranchPrefix != "" {
		return currentGroup.Defaults.BranchPrefix
	}

	// Check if we have any groups in config with branch prefix
	if len(rs.engine.config.Groups) > 0 && rs.engine.config.Groups[0].Defaults.BranchPrefix != "" {
		return rs.engine.config.Groups[0].Defaults.BranchPrefix
	}

	// Fall back to default prefix
	return "chore/sync-files"
}

// createTempDir creates a temporary directory for the sync operation
func (rs *RepositorySync) createTempDir() error {
	tempDir, err := os.MkdirTemp("", "go-broadcast-sync-*")
	if err != nil {
		return err
	}

	rs.tempDir = tempDir
	rs.logger.WithField("temp_dir", tempDir).Debug("Created temporary directory")
	return nil
}

// cleanup removes temporary files unless configured otherwise
func (rs *RepositorySync) cleanup() {
	if !rs.engine.options.CleanupTempFiles || rs.tempDir == "" {
		return
	}

	// First attempt: try recursive permission fix for macOS compatibility
	if err := rs.cleanupWithPermissionFix(); err != nil {
		rs.logger.WithError(err).Warn("Failed to cleanup temporary directory")

		// Fallback: try forced removal
		rs.logger.Debug("Attempting fallback cleanup strategy")
		if err := rs.forceCleanup(); err != nil {
			rs.logger.WithError(err).Error("Failed to cleanup temporary directory with fallback strategy")
		}
	} else {
		rs.logger.Debug("Cleaned up temporary directory")
	}
}

// cleanupWithPermissionFix attempts to fix permissions recursively before cleanup
func (rs *RepositorySync) cleanupWithPermissionFix() error {
	// Walk the directory tree and fix permissions for all files and directories
	err := filepath.Walk(rs.tempDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			// Log the error but continue walking
			rs.logger.WithError(walkErr).WithField("path", path).Debug("Error accessing path during permission fix")
			return walkErr
		}

		if info.IsDir() {
			// Make directories readable and writable (executable for directories is needed for traversal)
			// Using 0700 is acceptable for directories as it's needed for cleanup
			// #nosec G302
			if chmodErr := os.Chmod(path, 0o700); chmodErr != nil {
				rs.logger.WithError(chmodErr).WithField("path", path).Debug("Failed to chmod directory")
			}
		} else {
			// Make files readable and writable
			if chmodErr := os.Chmod(path, 0o600); chmodErr != nil {
				rs.logger.WithError(chmodErr).WithField("path", path).Debug("Failed to chmod file")
			}
		}
		return nil
	})
	if err != nil {
		rs.logger.WithError(err).Debug("Error during permission fix walk")
		return fmt.Errorf("permission fix walk failed: %w", err)
	}

	// Now try to remove the directory
	return os.RemoveAll(rs.tempDir)
}

// forceCleanup attempts various fallback strategies for stubborn directories
func (rs *RepositorySync) forceCleanup() error {
	var lastErr error

	// Strategy 1: Try changing owner permissions first
	// #nosec G302 - 0700 is needed for temporary directory cleanup
	if err := os.Chmod(rs.tempDir, 0o700); err != nil {
		rs.logger.WithError(err).Debug("Failed to change temp directory permissions to 700")
	}

	// Strategy 2: Multiple removal attempts with delay
	for i := 0; i < 3; i++ {
		if err := os.RemoveAll(rs.tempDir); err != nil {
			lastErr = err
			rs.logger.WithError(err).WithField("attempt", i+1).Debug("Cleanup attempt failed")
			time.Sleep(time.Millisecond * 100) // Brief delay before retry
		} else {
			return nil // Success
		}
	}

	return lastErr
}

// cloneSource clones the source repository at the specific commit
func (rs *RepositorySync) cloneSource(ctx context.Context) error {
	rs.logger.WithFields(logrus.Fields{
		"source_repo":   rs.sourceState.Repo,
		"source_branch": rs.sourceState.Branch,
		"commit_sha":    rs.sourceState.LatestCommit,
	}).Info("Cloning source repository")

	// Clone the repository
	sourceURL := fmt.Sprintf("https://github.com/%s.git", rs.sourceState.Repo)
	sourcePath := filepath.Join(rs.tempDir, "source")

	// Get blob size limit from current group config
	var opts *git.CloneOptions
	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		opts = &git.CloneOptions{BlobSizeLimit: currentGroup.Source.BlobSizeLimit}
	}

	if err := rs.engine.git.Clone(ctx, sourceURL, sourcePath, opts); err != nil {
		return err
	}

	// Checkout specific commit
	if err := rs.engine.git.Checkout(ctx, sourcePath, rs.sourceState.LatestCommit); err != nil {
		return err
	}

	rs.logger.Debug("Source repository cloned successfully")
	return nil
}

// processFiles processes all configured files and applies transformations
func (rs *RepositorySync) processFiles(ctx context.Context) ([]FileChange, error) {
	rs.logger.WithField("file_count", len(rs.target.Files)).Info("Processing files")

	var changedFiles []FileChange
	sourcePath := filepath.Join(rs.tempDir, "source")

	for _, fileMapping := range rs.target.Files {
		change, err := rs.processFile(ctx, sourcePath, fileMapping)
		if err != nil {
			// Handle recoverable errors gracefully
			if errors.Is(err, internalerrors.ErrTransformNotFound) {
				rs.logger.WithField("file", fileMapping.Dest).Debug("File content unchanged, skipping")
				continue
			}
			if errors.Is(err, internalerrors.ErrFileNotFound) {
				rs.logger.WithField("file", fileMapping.Src).Debug("Source file not found, skipping")
				continue
			}
			// For any other error, fail the operation
			return nil, fmt.Errorf("failed to process file %s: %w", fileMapping.Src, err)
		}

		if change != nil {
			changedFiles = append(changedFiles, *change)
		}
	}

	rs.logger.WithField("changed_files", len(changedFiles)).Info("File processing completed")
	return changedFiles, nil
}

// processFile processes a single file mapping
func (rs *RepositorySync) processFile(ctx context.Context, sourcePath string, fileMapping config.FileMapping) (*FileChange, error) {
	// Handle file deletion
	if fileMapping.Delete {
		return rs.processFileDeletion(ctx, fileMapping)
	}

	srcPath := filepath.Join(sourcePath, fileMapping.Src)

	// Check if source file exists
	srcContent, err := os.ReadFile(srcPath) //nolint:gosec // Path is constructed from trusted configuration
	if err != nil {
		if os.IsNotExist(err) {
			rs.logger.WithField("file", fileMapping.Src).Warn("Source file not found, skipping")
			return nil, internalerrors.ErrFileNotFound
		}
		return nil, err
	}

	// Apply transformations
	transformCtx := transform.Context{
		SourceRepo: rs.sourceState.Repo,
		TargetRepo: rs.target.Repo,
		FilePath:   fileMapping.Dest,
		Variables:  rs.target.Transform.Variables,
	}

	// Add email configuration if available
	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		transformCtx.SourceSecurityEmail = currentGroup.Source.SecurityEmail
		transformCtx.SourceSupportEmail = currentGroup.Source.SupportEmail
		// Use target-specific emails if set, otherwise use source emails
		if rs.target.SecurityEmail != "" {
			transformCtx.TargetSecurityEmail = rs.target.SecurityEmail
		} else {
			transformCtx.TargetSecurityEmail = currentGroup.Source.SecurityEmail
		}
		if rs.target.SupportEmail != "" {
			transformCtx.TargetSupportEmail = rs.target.SupportEmail
		} else {
			transformCtx.TargetSupportEmail = currentGroup.Source.SupportEmail
		}
	}

	transformedContent := srcContent
	if rs.target.Transform.RepoName || len(rs.target.Transform.Variables) > 0 {
		transformedContent, err = rs.engine.transform.Transform(ctx, srcContent, transformCtx)
		if err != nil {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}
	}

	// Check if content actually changed (for existing files)
	existingContent, err := rs.getExistingFileContent(ctx, fileMapping.Dest)
	if err == nil {
		// Enhanced logging for content comparison
		existingStr := string(existingContent)
		transformedStr := string(transformedContent)
		contentMatches := existingStr == transformedStr

		rs.logger.WithFields(logrus.Fields{
			"file":                     fileMapping.Dest,
			"existing_content_size":    len(existingContent),
			"transformed_content_size": len(transformedContent),
			"content_matches":          contentMatches,
		}).Debug("Comparing existing vs transformed content")

		if contentMatches {
			rs.logger.WithField("file", fileMapping.Dest).Debug("File content unchanged, skipping")
			return nil, internalerrors.ErrTransformNotFound
		}
	} else {
		rs.logger.WithError(err).WithField("file", fileMapping.Dest).Debug("Could not get existing file content, treating as new file")
	}

	// Use existing target content for OriginalContent (shows actual PR changes)
	// Fall back to source content for new files where no existing content exists
	originalContent := existingContent
	if originalContent == nil {
		originalContent = srcContent
	}

	return &FileChange{
		Path:            fileMapping.Dest,
		Content:         transformedContent,
		OriginalContent: originalContent,
		IsNew:           err != nil, // err means file doesn't exist
	}, nil
}

// getExistingFileContent retrieves the current content of a file from the target repo
func (rs *RepositorySync) getExistingFileContent(ctx context.Context, filePath string) ([]byte, error) {
	// Track API request
	rs.TrackAPIRequest()

	// Try to get file from the target repository's configured branch
	fileContent, err := rs.engine.gh.GetFile(ctx, rs.target.Repo, filePath, rs.target.Branch)
	if err != nil {
		return nil, err
	}
	return fileContent.Content, nil
}

// processFileDeletion handles the deletion of a file from the target repository
func (rs *RepositorySync) processFileDeletion(ctx context.Context, fileMapping config.FileMapping) (*FileChange, error) {
	rs.logger.WithField("file", fileMapping.Dest).Info("Processing file deletion")

	// Check if file exists in target repository
	existingContent, err := rs.getExistingFileContent(ctx, fileMapping.Dest)
	if err != nil {
		rs.logger.WithError(err).WithField("file", fileMapping.Dest).Debug("File does not exist in target repository, skipping deletion")
		return nil, internalerrors.ErrFileNotFound
	}

	rs.logger.WithFields(logrus.Fields{
		"file":         fileMapping.Dest,
		"content_size": len(existingContent),
	}).Debug("File exists in target, marking for deletion")

	// Return a FileChange with deletion flag
	return &FileChange{
		Path:            fileMapping.Dest,
		Content:         nil, // No content for deletions
		OriginalContent: existingContent,
		IsNew:           false,
		IsDeleted:       true,
	}, nil
}

// createSyncBranch creates a new sync branch or returns existing one
func (rs *RepositorySync) createSyncBranch(_ context.Context) string {
	// Generate branch name: chore/sync-files-{groupID}-YYYYMMDD-HHMMSS-{commit}
	now := time.Now()
	timestamp := now.Format("20060102-150405")
	commitSHA := rs.sourceState.LatestCommit
	if len(commitSHA) > 7 {
		commitSHA = commitSHA[:7]
	}

	var branchPrefix string
	var groupID string
	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		branchPrefix = currentGroup.Defaults.BranchPrefix
		groupID = currentGroup.ID
	} else {
		// Get defaults from the first group (since we have a single group in temporary config)
		if len(rs.engine.config.Groups) > 0 {
			branchPrefix = rs.engine.config.Groups[0].Defaults.BranchPrefix
			groupID = rs.engine.config.Groups[0].ID
		}
	}
	if branchPrefix == "" {
		branchPrefix = "chore/sync-files"
	}
	// Group ID is always required
	if groupID == "" {
		rs.logger.Warn("No group ID found, using 'default'")
		groupID = "default"
	}

	branchName := fmt.Sprintf("%s-%s-%s-%s", branchPrefix, groupID, timestamp, commitSHA)

	rs.logger.WithField("branch_name", branchName).Info("Creating sync branch")

	if rs.engine.options.DryRun {
		rs.logger.Info("DRY-RUN: Would create sync branch")
		return branchName
	}

	// Create branch in target repository
	// We'll create the branch when we push, so just return the name for now
	return branchName
}

// commitChanges creates a commit with the changed files and returns commit SHA and actual changed files.
// Even in dry-run mode, this clones the repo and stages files to generate accurate AI content.
func (rs *RepositorySync) commitChanges(ctx context.Context, branchName string, changedFiles []FileChange) (string, []string, error) {
	if len(changedFiles) == 0 {
		return "", nil, internalerrors.ErrNoFilesToCommit
	}

	// Clone the target repository for making changes
	// We do this even in dry-run mode to get accurate diffs for AI generation
	targetPath := filepath.Join(rs.tempDir, "target")
	targetURL := fmt.Sprintf("https://github.com/%s.git", rs.target.Repo)

	// Disable partial clone for target repo - we need full blob content for accurate diffs.
	// Partial clone with lazy blob fetching can cause git diff to show wrong base content
	// because blobs may be fetched from origin/HEAD (master) instead of origin/development.
	// See: https://github.com/mrz1836/go-broadcast/issues/XXX for details.
	opts := &git.CloneOptions{BlobSizeLimit: "0"} // "0" disables blob filtering

	// Clone from the configured target branch (rs.target.Branch) to ensure the local diff
	// matches what GitHub shows in the PR diff. This is critical for AI-generated descriptions.
	// We intentionally do NOT clone from existing sync branches because:
	// 1. Old sync branches may have been created from a different base branch
	// 2. The PR diff is always relative to target branch, not old sync branches
	// 3. Cloning from wrong base causes AI to describe incorrect changes
	targetBranch := rs.target.Branch
	if targetBranch != "" {
		rs.logger.WithField("target_branch", targetBranch).Info("Cloning repository with target branch")
		if err := rs.engine.git.CloneWithBranch(ctx, targetURL, targetPath, targetBranch, opts); err != nil {
			return "", nil, fmt.Errorf("failed to clone target repository with branch %s: %w", targetBranch, err)
		}
	} else {
		if err := rs.engine.git.Clone(ctx, targetURL, targetPath, opts); err != nil {
			return "", nil, fmt.Errorf("failed to clone target repository: %w", err)
		}
	}

	// Create and checkout the new sync branch from the target branch
	if err := rs.engine.git.CreateBranch(ctx, targetPath, branchName); err != nil {
		// Check if it's a branch already exists error (local branch)
		if errors.Is(err, git.ErrBranchAlreadyExists) {
			rs.logger.WithField("branch", branchName).Warn("Branch already exists locally, checking out existing branch")
			// Try to checkout the existing branch instead
			if checkoutErr := rs.engine.git.Checkout(ctx, targetPath, branchName); checkoutErr != nil {
				return "", nil, fmt.Errorf("failed to checkout existing branch %s: %w", branchName, checkoutErr)
			}
		} else {
			return "", nil, fmt.Errorf("failed to create branch %s: %w", branchName, err)
		}
	} else {
		// Branch was created successfully, checkout
		if err := rs.engine.git.Checkout(ctx, targetPath, branchName); err != nil {
			return "", nil, fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
		}
	}

	// Apply go.mod module updates if any
	if len(rs.moduleUpdates) > 0 {
		rs.logger.WithField("module_updates", len(rs.moduleUpdates)).Info("Applying module updates to go.mod")
		goModUpdater := NewGoModUpdater(rs.logger.Logger)

		for _, update := range rs.moduleUpdates {
			goModPath := filepath.Join(targetPath, update.DestPath)

			// Check if go.mod exists in target repository
			if _, err := os.Stat(goModPath); os.IsNotExist(err) {
				rs.logger.WithFields(logrus.Fields{
					"path":   update.DestPath,
					"module": update.ModuleName,
				}).Warn("go.mod not found in target repository, skipping update")
				continue
			}

			// Read the current go.mod
			// #nosec G304 -- goModPath is constructed from controlled values (temp dir + config dest path)
			currentContent, err := os.ReadFile(goModPath)
			if err != nil {
				rs.logger.WithError(err).WithFields(logrus.Fields{
					"path":   update.DestPath,
					"module": update.ModuleName,
				}).Warn("Failed to read go.mod, skipping update")
				continue
			}

			// Try to update existing dependency, or add if it doesn't exist
			updatedContent, modified, err := goModUpdater.UpdateDependency(
				currentContent,
				update.ModuleName,
				update.Version,
			)
			if err != nil {
				rs.logger.WithError(err).WithFields(logrus.Fields{
					"path":   update.DestPath,
					"module": update.ModuleName,
				}).Warn("Failed to update go.mod dependency, skipping")
				continue
			}

			// If not modified, try adding as new dependency
			if !modified {
				updatedContent, modified, err = goModUpdater.AddDependency(
					currentContent,
					update.ModuleName,
					update.Version,
				)
				if err != nil {
					rs.logger.WithError(err).WithFields(logrus.Fields{
						"path":   update.DestPath,
						"module": update.ModuleName,
					}).Warn("Failed to add go.mod dependency, skipping")
					continue
				}
			}

			// Write the updated go.mod if modified
			if modified {
				if err := os.WriteFile(goModPath, updatedContent, 0o600); err != nil {
					return "", nil, fmt.Errorf("failed to write updated go.mod %s: %w", update.DestPath, err)
				}

				rs.logger.WithFields(logrus.Fields{
					"path":    update.DestPath,
					"module":  update.ModuleName,
					"version": update.Version,
				}).Info("Updated module reference in go.mod")
			}
		}
	}

	// Apply file changes to the target repository
	var filesToDelete []string
	for _, fileChange := range changedFiles {
		destPath := filepath.Join(targetPath, fileChange.Path)

		if fileChange.IsDeleted {
			// Handle file deletion
			rs.logger.WithField("file", fileChange.Path).Debug("Marking file for deletion")
			filesToDelete = append(filesToDelete, fileChange.Path)

			// Remove the file from filesystem if it exists
			if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
				return "", nil, fmt.Errorf("failed to remove file %s: %w", fileChange.Path, err)
			}
		} else {
			// Handle file creation/modification
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
				return "", nil, fmt.Errorf("failed to create directory for %s: %w", fileChange.Path, err)
			}

			// Write the file content
			if err := os.WriteFile(destPath, fileChange.Content, 0o600); err != nil {
				return "", nil, fmt.Errorf("failed to write file %s: %w", fileChange.Path, err)
			}
		}
	}

	// Remove deleted files from git tracking
	if len(filesToDelete) > 0 {
		rs.logger.WithField("files_to_delete", len(filesToDelete)).Debug("Removing deleted files from git")
		if err := rs.engine.git.BatchRemoveFiles(ctx, targetPath, filesToDelete, false); err != nil {
			// Log warning but don't fail - files might not be tracked
			rs.logger.WithError(err).WithField("files", filesToDelete).Warn("Failed to remove files from git, continuing")
		}
	}

	// Stage all changes - this prepares for both AI diff generation and the actual commit
	if err := rs.engine.git.Add(ctx, targetPath, "."); err != nil {
		return "", nil, fmt.Errorf("failed to stage changes: %w", err)
	}

	// Store the target path for AI diff generation
	rs.stagedRepoPath = targetPath

	// Generate commit message AFTER staging so we have the real git diff
	commitMsg, aiGenerated := rs.generateCommitMessage(ctx, changedFiles)
	rs.commitAIGenerated = aiGenerated // Store for PR metadata block

	// Log AI usage for commit message
	if aiGenerated {
		rs.logger.WithField("component", "ai_commit").Info("AI generated commit message")
	} else if rs.engine != nil && rs.engine.commitGenerator != nil {
		rs.logger.WithField("component", "ai_commit").Debug("AI commit generation failed, using static template")
	}

	rs.logger.WithFields(logrus.Fields{
		"branch":       branchName,
		"files":        len(changedFiles),
		"commit_msg":   commitMsg,
		"ai_generated": aiGenerated,
	}).Info("Creating commit")

	// For dry-run: show preview and return without committing
	if rs.engine.options.DryRun {
		rs.showDryRunCommitInfo(commitMsg, changedFiles, aiGenerated)
		rs.showDryRunFileChanges(changedFiles)
		// For dry run, return the expected files as if they all changed
		dryRunFiles := make([]string, len(changedFiles))
		for i, file := range changedFiles {
			dryRunFiles[i] = file.Path
		}
		return "dry-run-commit-sha", dryRunFiles, nil
	}

	// Create the commit
	if err := rs.engine.git.Commit(ctx, targetPath, commitMsg); err != nil {
		// Check if it's because there are no changes to commit
		if errors.Is(err, git.ErrNoChanges) {
			rs.logger.WithFields(logrus.Fields{
				"branch":     branchName,
				"files":      len(changedFiles),
				"commit_msg": commitMsg,
			}).Info("No changes to commit - files are already synchronized")

			// Return special error to indicate no actual changes
			return "", nil, internalerrors.ErrNoChangesToSync
		}
		return "", nil, fmt.Errorf("failed to create commit: %w", err)
	}

	// Get the commit SHA
	commitSHA, err := rs.engine.git.GetCurrentCommitSHA(ctx, targetPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get commit SHA: %w", err)
	}

	// Get the actual files that were changed in the commit
	actualChangedFiles, err := rs.engine.git.GetChangedFiles(ctx, targetPath)
	if err != nil {
		rs.logger.WithError(err).Warn("Failed to get actual changed files, using all attempted files")
		// Fallback to using all the files we attempted to change
		actualChangedFiles = make([]string, len(changedFiles))
		for i, file := range changedFiles {
			actualChangedFiles[i] = file.Path
		}
	}

	return commitSHA, actualChangedFiles, nil
}

// pushChanges pushes the branch to the target repository
func (rs *RepositorySync) pushChanges(ctx context.Context, branchName string) error {
	rs.logger.WithField("branch", branchName).Info("Pushing changes to target repository")

	targetPath := filepath.Join(rs.tempDir, "target")

	// Push the branch to the origin remote (which is the target repository)
	if err := rs.engine.git.Push(ctx, targetPath, "origin", branchName, false); err != nil {
		// Check if it's a branch already exists error
		if errors.Is(err, git.ErrBranchAlreadyExists) {
			rs.logger.WithFields(logrus.Fields{
				"branch_name": branchName,
				"target_repo": rs.target.Repo,
			}).Warn("Branch already exists on remote, attempting force push to recover from partial sync")

			// Try force push to overwrite the existing branch
			if forceErr := rs.engine.git.Push(ctx, targetPath, "origin", branchName, true); forceErr != nil {
				return fmt.Errorf("failed to force push branch %s after detecting existing branch: %w", branchName, forceErr)
			}

			rs.logger.WithField("branch", branchName).Info("Successfully force pushed branch to recover from existing branch conflict")
			return nil
		}
		return fmt.Errorf("failed to push branch %s to target repository: %w", branchName, err)
	}

	return nil
}

// createOrUpdatePR creates a new PR or updates an existing one
func (rs *RepositorySync) createOrUpdatePR(ctx context.Context, branchName, commitSHA string, changedFiles []FileChange, actualChangedFiles []string) error {
	// Check if PR already exists for this branch
	existingPR := rs.findExistingPR(branchName)

	if existingPR != nil {
		return rs.updateExistingPR(ctx, existingPR, commitSHA, changedFiles, actualChangedFiles)
	}

	return rs.createNewPR(ctx, branchName, commitSHA, changedFiles, actualChangedFiles)
}

// findExistingPR finds an existing PR for the sync branch
func (rs *RepositorySync) findExistingPR(branchName string) *gh.PR {
	if rs.targetState == nil {
		return nil
	}

	for _, pr := range rs.targetState.OpenPRs {
		if pr.Head.Ref == branchName {
			return &pr
		}
	}

	return nil
}

// checkForExistingPRViAPI checks for existing PRs via direct GitHub API call
// This is a fallback when state discovery might have missed PRs
func (rs *RepositorySync) checkForExistingPRViAPI(ctx context.Context, branchName string) (*gh.PR, error) {
	// Get all open PRs for this repository
	prs, err := rs.engine.gh.ListPRs(ctx, rs.target.Repo, "open")
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	// Look for PR with matching head branch
	for _, pr := range prs {
		if pr.Head.Ref == branchName {
			rs.logger.WithFields(logrus.Fields{
				"pr_number":   pr.Number,
				"branch_name": branchName,
				"pr_state":    pr.State,
			}).Debug("Found existing PR via API")
			return &pr, nil
		}
	}

	return nil, internalerrors.ErrPRNotFound
}

// createNewPR creates a new pull request
func (rs *RepositorySync) createNewPR(ctx context.Context, branchName, commitSHA string, changedFiles []FileChange, actualChangedFiles []string) error {
	title := rs.generatePRTitle()
	body, aiGenerated := rs.generatePRBody(ctx, commitSHA, changedFiles, actualChangedFiles)

	// Log AI usage for PR body
	if aiGenerated {
		rs.logger.WithField("component", "ai_pr").Info("AI generated PR body")
	} else if rs.engine != nil && rs.engine.prGenerator != nil {
		rs.logger.WithField("component", "ai_pr").Debug("AI PR body generation failed, using static template")
	}

	rs.logger.WithFields(logrus.Fields{
		"branch":       branchName,
		"title":        title,
		"ai_generated": aiGenerated,
	}).Info("Creating new pull request")

	if rs.engine.options.DryRun {
		rs.showDryRunPRPreview(ctx, branchName, title, body, aiGenerated)
		return nil
	}

	// Get base branch for PR - use target branch if specified, otherwise auto-detect
	var baseBranch string
	if rs.targetState != nil && rs.targetState.Branch != "" {
		// Use configured target branch but validate it exists
		baseBranch = rs.targetState.Branch

		// Validate that the target branch actually exists
		rs.TrackAPIRequest()
		_, err := rs.engine.gh.GetBranch(ctx, rs.target.Repo, baseBranch)
		if err != nil {
			return fmt.Errorf("configured target branch %q does not exist in repository %s: %w", baseBranch, rs.target.Repo, err)
		}

		rs.logger.WithFields(logrus.Fields{
			"configured_branch": baseBranch,
			"target_repo":       rs.target.Repo,
		}).Debug("Using configured target branch for PR base")
	} else {
		// Auto-detect default branch
		rs.TrackAPIRequest()
		branches, err := rs.engine.gh.ListBranches(ctx, rs.target.Repo)
		if err != nil {
			return fmt.Errorf("failed to get branches: %w", err)
		}

		baseBranch = "master" // default fallback
		for _, branch := range branches {
			if branch.Name == mainBranch {
				baseBranch = mainBranch
				break
			}
		}
		rs.logger.WithFields(logrus.Fields{
			"detected_branch": baseBranch,
			"target_repo":     rs.target.Repo,
		}).Debug("Auto-detected default branch for PR base")
	}

	// Get current user to filter out from reviewers
	rs.TrackAPIRequest()
	currentUser, err := rs.engine.gh.GetCurrentUser(ctx)
	if err != nil {
		rs.logger.WithError(err).Warn("Failed to get current user for reviewer filtering")
	}

	// Filter author from reviewers
	reviewers := rs.getPRReviewers()
	if currentUser != nil && len(reviewers) > 0 {
		filteredReviewers := make([]string, 0, len(reviewers))
		for _, reviewer := range reviewers {
			if reviewer != currentUser.Login {
				filteredReviewers = append(filteredReviewers, reviewer)
			} else {
				rs.logger.WithField("reviewer", reviewer).Info("Filtering PR author from reviewers list")
			}
		}
		reviewers = filteredReviewers
	}

	prRequest := gh.PRRequest{
		Title:         title,
		Body:          body,
		Head:          branchName,
		Base:          baseBranch,
		Labels:        rs.getPRLabels(),
		Assignees:     rs.getPRAssignees(),
		Reviewers:     reviewers,
		TeamReviewers: rs.getPRTeamReviewers(),
	}

	if rs.logger != nil {
		rs.logger.Info("Creating pull request on GitHub...")
	}
	rs.TrackAPIRequest()
	pr, err := rs.engine.gh.CreatePR(ctx, rs.target.Repo, prRequest)
	if err != nil {
		// Check if it's a validation failure (HTTP 422)
		if errors.Is(err, gh.ErrPRValidationFailed) {
			rs.logger.WithFields(logrus.Fields{
				"branch_name": branchName,
				"target_repo": rs.target.Repo,
				"error":       err,
			}).Warn("PR validation failed - checking for existing PR")

			// First, check if a PR already exists for this branch via direct API call
			existingPR, prErr := rs.checkForExistingPRViAPI(ctx, branchName)
			if prErr != nil && !errors.Is(prErr, internalerrors.ErrPRNotFound) {
				rs.logger.WithError(prErr).Debug("Failed to check for existing PR via API")
			} else if existingPR != nil {
				rs.logger.WithFields(logrus.Fields{
					"pr_number":   existingPR.Number,
					"branch_name": branchName,
				}).Info("Found existing PR for branch - updating instead of creating new one")
				// Update the existing PR instead
				return rs.updateExistingPR(ctx, existingPR, commitSHA, changedFiles, actualChangedFiles)
			}

			// If no existing PR found, try branch cleanup and retry
			rs.logger.Debug("No existing PR found, attempting branch cleanup and retry")
			if deleteErr := rs.engine.gh.DeleteBranch(ctx, rs.target.Repo, branchName); deleteErr != nil {
				rs.logger.WithError(deleteErr).Debug("Failed to delete orphaned branch (may not exist)")
			}

			// Retry PR creation after branch cleanup
			rs.TrackAPIRequest()
			pr, err = rs.engine.gh.CreatePR(ctx, rs.target.Repo, prRequest)
			if err != nil {
				// If still failing after cleanup, log warning but don't fail entire sync
				rs.logger.WithError(err).WithFields(logrus.Fields{
					"branch_name": branchName,
					"target_repo": rs.target.Repo,
				}).Warn("PR creation failed after cleanup - branch may have been pushed but PR not created")
				return fmt.Errorf("failed to create PR after cleanup: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create PR: %w", err)
		}
	}

	rs.logger.WithField("pr_number", pr.Number).Info("Pull request created successfully")

	// Capture PR info for metrics recording
	rs.lastPRNumber = &pr.Number
	rs.lastPRURL = fmt.Sprintf("https://github.com/%s/pull/%d", rs.target.Repo, pr.Number)

	return nil
}

// updateExistingPR updates an existing pull request
func (rs *RepositorySync) updateExistingPR(ctx context.Context, pr *gh.PR, commitSHA string, changedFiles []FileChange, actualChangedFiles []string) error {
	rs.logger.WithField("pr_number", pr.Number).Info("Updating existing pull request")

	if rs.engine.options.DryRun {
		out := NewDryRunOutput(nil)

		out.Header("🔄 DRY-RUN: Existing Pull Request Update Preview")
		out.Field("Repository", rs.target.Repo)
		out.Field("PR Number", fmt.Sprintf("#%d", pr.Number))
		out.Field("Current Title", pr.Title)
		out.Separator()
		out.Success("PR would be updated with new file changes")
		out.Field("Files to sync", fmt.Sprintf("%d", len(changedFiles)))
		out.Field("New commit", commitSHA)
		out.Footer()

		// Show the files that would be updated
		rs.showDryRunFileChanges(changedFiles)
		return nil
	}

	// Update PR body with new information
	newBody, _ := rs.generatePRBody(ctx, commitSHA, changedFiles, actualChangedFiles)

	// Update the PR via GitHub API
	updates := gh.PRUpdate{
		Body: &newBody,
	}

	rs.TrackAPIRequest()
	if err := rs.engine.gh.UpdatePR(ctx, rs.target.Repo, pr.Number, updates); err != nil {
		return fmt.Errorf("failed to update PR: %w", err)
	}

	rs.logger.WithField("pr_number", pr.Number).Info("Pull request updated successfully")

	// Capture PR info for metrics recording
	rs.lastPRNumber = &pr.Number
	rs.lastPRURL = fmt.Sprintf("https://github.com/%s/pull/%d", rs.target.Repo, pr.Number)

	return nil
}

// getDiffForAI retrieves and truncates the diff for AI context.
// Uses the real git diff from stagedRepoPath if available (after repo is cloned and files staged).
// Falls back to synthetic diff from changedFiles if stagedRepoPath is not set.
func (rs *RepositorySync) getDiffForAI(ctx context.Context, changedFiles []FileChange) string {
	if rs.engine.diffTruncator == nil {
		rs.logger.Debug("AI diff generation skipped: diffTruncator is nil")
		return ""
	}

	var diff string
	var diffSource string

	// Prefer real git diff from staged repo if available
	// Use DiffIgnoreWhitespace to avoid line ending normalization masking real changes
	if rs.stagedRepoPath != "" {
		var err error
		diff, err = rs.engine.git.DiffIgnoreWhitespace(ctx, rs.stagedRepoPath, true) // staged changes, ignore whitespace
		if err != nil {
			rs.logger.WithError(err).Debug("Failed to get git diff for AI context, falling back to synthetic diff")
		} else if diff != "" {
			diffSource = "git"
		}
	}

	// Fall back to synthetic diff if git diff is empty or failed
	if diff == "" && len(changedFiles) > 0 {
		diff = rs.generateSyntheticDiff(changedFiles)
		diffSource = "synthetic"
	}

	// Log the diff being passed to AI for debugging
	originalLen := len(diff)
	truncatedDiff := rs.engine.diffTruncator.Truncate(diff)
	truncated := len(truncatedDiff) < originalLen

	rs.logger.WithFields(logrus.Fields{
		"diff_source":      diffSource,
		"original_length":  originalLen,
		"truncated_length": len(truncatedDiff),
		"was_truncated":    truncated,
		"file_count":       len(changedFiles),
	}).Debug("Diff prepared for AI generation")

	// Log diff content preview at trace level for debugging
	if rs.logger.Logger.IsLevelEnabled(logrus.TraceLevel) {
		preview := truncatedDiff
		if len(preview) > 500 {
			preview = preview[:500] + "...[truncated in log]"
		}
		rs.logger.WithField("diff_preview", preview).Trace("Diff content preview for AI")
	}

	// Optionally write diff to file for debugging (set GO_BROADCAST_DEBUG_DIFF_PATH env var)
	// Each repo gets its own debug file to avoid overwrites during multi-repo syncs
	if debugPath := os.Getenv("GO_BROADCAST_DEBUG_DIFF_PATH"); debugPath != "" {
		// Create per-repo debug file path (e.g., /tmp/debug.txt.owner_repo)
		repoSuffix := strings.ReplaceAll(rs.target.Repo, "/", "_")
		repoDebugPath := fmt.Sprintf("%s.%s", debugPath, repoSuffix)
		if err := os.WriteFile(repoDebugPath, []byte(diff), 0o600); err != nil { //nolint:gosec // G703: repoDebugPath is constructed from env var + safe suffix, not user input
			rs.logger.WithError(err).Warn("Failed to write debug diff file")
		} else {
			rs.logger.WithField("path", repoDebugPath).Debug("Wrote full diff to debug file")
		}
	}

	return truncatedDiff
}

// generateSyntheticDiff creates a unified diff from FileChange data.
// This enables AI generation in dry-run mode without needing a git repository.
// It uses the Content and OriginalContent fields from FileChange to generate diffs.
func (rs *RepositorySync) generateSyntheticDiff(changedFiles []FileChange) string {
	var sb strings.Builder

	for _, file := range changedFiles {
		// Log content availability for debugging synthetic diff issues
		rs.logger.WithFields(logrus.Fields{
			"file":                 file.Path,
			"is_new":               file.IsNew,
			"is_deleted":           file.IsDeleted,
			"has_content":          file.Content != nil,
			"has_original_content": file.OriginalContent != nil,
			"content_len":          len(file.Content),
			"original_content_len": len(file.OriginalContent),
		}).Trace("Generating synthetic diff for file")

		var fileDiff string
		if file.IsNew {
			// New file: all lines added
			fileDiff = ai.GenerateNewFileDiff(file.Path, string(file.Content))
		} else if file.IsDeleted {
			// Deleted file: all lines removed
			fileDiff = ai.GenerateDeletedFileDiff(file.Path, string(file.OriginalContent))
		} else {
			// Modified file: diff between original and new content
			oldContent := ""
			newContent := ""
			if file.OriginalContent != nil {
				oldContent = string(file.OriginalContent)
			}
			if file.Content != nil {
				newContent = string(file.Content)
			}

			// Warn if both contents are identical (no actual diff)
			if oldContent == newContent {
				rs.logger.WithField("file", file.Path).Debug("Synthetic diff: original and new content are identical, no diff generated")
			}

			fileDiff = ai.GenerateUnifiedDiff(file.Path, oldContent, newContent)
		}

		if fileDiff != "" {
			sb.WriteString(fileDiff)
		} else {
			rs.logger.WithField("file", file.Path).Debug("Synthetic diff: no diff generated for file")
		}
	}

	return sb.String()
}

// convertToAIFileChanges converts sync FileChange to AI FileChange type.
func (rs *RepositorySync) convertToAIFileChanges(files []FileChange) []ai.FileChange {
	result := make([]ai.FileChange, 0, len(files))
	for _, f := range files {
		changeType := "modified"
		if f.IsNew {
			changeType = "added"
		} else if f.IsDeleted {
			changeType = "deleted"
		}

		// Calculate actual line changes (not total file lines)
		linesAdded := 0
		linesRemoved := 0
		if f.Content != nil && f.OriginalContent != nil {
			// Modified file: count actual diff lines
			linesAdded, linesRemoved = ai.CountDiffLines(string(f.OriginalContent), string(f.Content))
		} else if f.Content != nil {
			// New file: all lines added
			linesAdded = strings.Count(string(f.Content), "\n")
		} else if f.OriginalContent != nil {
			// Deleted file: all lines removed
			linesRemoved = strings.Count(string(f.OriginalContent), "\n")
		}

		result = append(result, ai.FileChange{
			Path:         f.Path,
			ChangeType:   changeType,
			LinesAdded:   linesAdded,
			LinesRemoved: linesRemoved,
		})
	}
	return result
}

// generateCommitMessage creates a descriptive commit message.
// Tries AI generation first if enabled, falls back to static template.
func (rs *RepositorySync) generateCommitMessage(ctx context.Context, changedFiles []FileChange) (string, bool) {
	// Try AI generation if enabled (check engine is not nil for tests)
	if rs.engine != nil && rs.engine.commitGenerator != nil {
		commitCtx := &ai.CommitContext{
			SourceRepo:   rs.sourceState.Repo,
			TargetRepo:   rs.target.Repo,
			ChangedFiles: rs.convertToAIFileChanges(changedFiles),
			DiffSummary:  rs.getDiffForAI(ctx, changedFiles), // Use synthetic diff for consistency
		}

		msg, err := rs.engine.commitGenerator.GenerateMessage(ctx, commitCtx)
		if msg != "" {
			// Check if AI actually generated or if fallback was used
			aiGenerated := !errors.Is(err, ai.ErrFallbackUsed)
			if !aiGenerated {
				rs.logger.Debug("AI commit message generation used fallback")
			}
			return msg, aiGenerated
		}
		rs.logger.Warn("AI commit message generation failed, using static fallback")
	}

	// Existing static generation (fallback)
	if len(changedFiles) == 1 {
		return fmt.Sprintf("sync: update %s from source repository", changedFiles[0].Path), false
	}

	return fmt.Sprintf("sync: update %d files from source repository", len(changedFiles)), false
}

// generatePRTitle creates a descriptive PR title
func (rs *RepositorySync) generatePRTitle() string {
	commitSHA := rs.sourceState.LatestCommit
	if len(commitSHA) > 7 {
		commitSHA = commitSHA[:7]
	}
	return fmt.Sprintf("[Sync] Update project files from source repository (%s)", commitSHA)
}

// generatePRBody creates a detailed PR description with metadata including directory sync info.
// Tries AI generation first if enabled, falls back to static template.
// Returns (body, aiGenerated) where aiGenerated indicates if AI successfully generated the body.
func (rs *RepositorySync) generatePRBody(ctx context.Context, commitSHA string, changedFiles []FileChange, actualChangedFiles []string) (string, bool) {
	var sb strings.Builder

	// Try AI generation if enabled (check engine is not nil for tests)
	if rs.logger != nil {
		rs.logger.Info("Generating PR body...")
	}
	if rs.engine != nil && rs.engine.prGenerator != nil {
		// Filter changedFiles to only include files that actually changed in git
		// This prevents AI from hallucinating changes for files that were examined but not modified
		var filteredChanges []FileChange
		if len(actualChangedFiles) > 0 {
			actualChangedSet := make(map[string]bool, len(actualChangedFiles))
			for _, path := range actualChangedFiles {
				actualChangedSet[path] = true
			}
			for _, fc := range changedFiles {
				if actualChangedSet[fc.Path] {
					filteredChanges = append(filteredChanges, fc)
				}
			}
			rs.logger.WithFields(logrus.Fields{
				"examined_files": len(changedFiles),
				"actual_changes": len(filteredChanges),
			}).Debug("Filtered file list for AI generation")
		} else {
			// Fallback: if actualChangedFiles is empty/nil, use all changedFiles
			filteredChanges = changedFiles
		}

		diffSummary := rs.getDiffForAI(ctx, filteredChanges)

		// Warn when diff is empty but files changed - AI may produce inaccurate description
		if diffSummary == "" && len(filteredChanges) > 0 {
			rs.logger.WithField("file_count", len(filteredChanges)).Warn(
				"Empty diff generated despite having changed files - AI may produce inaccurate PR description")
		}

		prCtx := &ai.PRContext{
			SourceRepo:   rs.sourceState.Repo,
			TargetRepo:   rs.target.Repo,
			CommitSHA:    commitSHA,
			ChangedFiles: rs.convertToAIFileChanges(filteredChanges),
			DiffSummary:  diffSummary,
		}

		aiBody, err := rs.engine.prGenerator.GenerateBody(ctx, prCtx)
		if aiBody != "" {
			// Check if AI actually generated or if fallback was used
			aiGenerated := !errors.Is(err, ai.ErrFallbackUsed)
			if aiGenerated {
				sb.WriteString(aiBody)
				sb.WriteString("\n\n")
				// CRITICAL: Metadata is NEVER AI-generated - always append static metadata
				// Use filteredChanges so metadata reflects what AI actually saw
				rs.writeMetadataBlock(&sb, commitSHA, filteredChanges, true) // PR body was AI-generated
				return sb.String(), true
			}
			rs.logger.Debug("AI PR body generation used fallback")
		} else if err != nil && !errors.Is(err, ai.ErrFallbackUsed) {
			rs.logger.Warn("AI PR body generation failed, using static fallback")
		}
	}

	// Existing static generation (fallback)

	// What Changed section with enhanced details
	sb.WriteString("## What Changed\n")
	rs.writeChangeSummary(&sb, changedFiles, actualChangedFiles)
	shortSHA := commitSHA
	if len(commitSHA) > 7 {
		shortSHA = commitSHA[:7]
	}
	fmt.Fprintf(&sb, "* Brought target repository in line with source repository state at commit %s\n\n", shortSHA)

	// Directory synchronization details (if directories are configured)
	if len(rs.target.Directories) > 0 {
		rs.writeDirectorySyncDetails(&sb)
	}

	// Performance metrics section
	rs.writePerformanceMetrics(&sb)

	// Why It Was Necessary section
	sb.WriteString("## Why It Was Necessary\n")
	sb.WriteString("This synchronization ensures the target repository stays up-to-date with the latest changes from the configured source repository. ")
	sb.WriteString("The sync operation identifies and applies only the necessary file changes while maintaining consistency across repositories.\n\n")

	// Testing Performed section
	sb.WriteString("## Testing Performed\n")
	sb.WriteString("* Validated sync configuration and file mappings\n")
	sb.WriteString("* Verified file transformations applied correctly\n")
	sb.WriteString("* Confirmed no unintended changes were introduced\n")
	sb.WriteString("* All automated checks and linters passed\n\n")

	// Impact / Risk section
	sb.WriteString("## Impact / Risk\n")
	sb.WriteString("* **Low Risk**: Standard sync operation with established patterns\n")
	sb.WriteString("* **No Breaking Changes**: File updates maintain backward compatibility\n")
	sb.WriteString("* **Performance**: No impact on application performance\n")
	sb.WriteString("* **Dependencies**: No dependency changes included in this sync\n\n")

	// Add enhanced metadata as YAML block
	rs.writeMetadataBlock(&sb, commitSHA, changedFiles, false) // PR body was NOT AI-generated

	return sb.String(), false // Static fallback was used, not AI
}

// writeChangeSummary writes a detailed summary of what changed in the sync
func (rs *RepositorySync) writeChangeSummary(sb *strings.Builder, changedFiles []FileChange, actualChangedFiles []string) {
	// Use actualChangedFiles (files that actually changed in git) for accurate counts
	fileChanges := 0
	directoryChanges := 0

	if actualChangedFiles != nil {
		// Count actual changes by type - files vs directories
		for _, filePath := range actualChangedFiles {
			if rs.isDirectoryFile(filePath) {
				directoryChanges++
			} else {
				fileChanges++
			}
		}
	} else {
		// Fallback to attempted changes if actualChangedFiles not available
		for _, change := range changedFiles {
			if rs.isDirectoryFile(change.Path) {
				directoryChanges++
			} else {
				fileChanges++
			}
		}
	}

	if fileChanges > 0 {
		fmt.Fprintf(sb, "* Updated %d individual file(s) to synchronize with the source repository\n", fileChanges)
	}

	if directoryChanges > 0 {
		fmt.Fprintf(sb, "* Synchronized %d file(s) from directory mappings\n", directoryChanges)
	}

	if len(rs.target.Files) > 0 || len(rs.target.Directories) > 0 {
		sb.WriteString("* Applied file transformations and updates based on sync configuration\n")
	}
}

// writeDirectorySyncDetails writes detailed information about directory synchronization
func (rs *RepositorySync) writeDirectorySyncDetails(sb *strings.Builder) {
	sb.WriteString("## Directory Synchronization Details\n")
	sb.WriteString("The following directories were synchronized:\n\n")

	for _, dirMapping := range rs.target.Directories {
		fmt.Fprintf(sb, "### `%s` → `%s`\n", dirMapping.Src, dirMapping.Dest)

		// Get metrics for this directory if available
		if rs.syncMetrics != nil && rs.syncMetrics.DirectoryMetrics != nil {
			if metrics, exists := rs.syncMetrics.GetDirectoryMetric(dirMapping.Src); exists {
				fmt.Fprintf(sb, "* **Files synced**: %d\n", metrics.FilesChanged)
				fmt.Fprintf(sb, "* **Files examined**: %d\n", metrics.FilesProcessed)
				fmt.Fprintf(sb, "* **Files excluded**: %d\n", metrics.FilesExcluded)

				if metrics.EndTime.After(metrics.StartTime) {
					duration := metrics.EndTime.Sub(metrics.StartTime)
					fmt.Fprintf(sb, "* **Processing time**: %dms\n", duration.Milliseconds())
				}

				if metrics.BinaryFilesSkipped > 0 {
					fmt.Fprintf(sb, "* **Binary files skipped**: %d (%.2f KB)\n",
						metrics.BinaryFilesSkipped, float64(metrics.BinaryFilesSize)/1024)
				}
			}
		}

		// Show exclusion patterns if any
		if len(dirMapping.Exclude) > 0 {
			sb.WriteString("* **Exclusion patterns**: ")
			for i, pattern := range dirMapping.Exclude {
				if i > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(sb, "`%s`", pattern)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
}

// writePerformanceMetrics writes performance metrics for the sync operation
func (rs *RepositorySync) writePerformanceMetrics(sb *strings.Builder) {
	if rs.syncMetrics == nil {
		return
	}

	sb.WriteString("## Performance Metrics\n")

	// Overall timing
	if rs.syncMetrics.EndTime.After(rs.syncMetrics.StartTime) {
		totalDuration := rs.syncMetrics.EndTime.Sub(rs.syncMetrics.StartTime)
		fmt.Fprintf(sb, "* **Total sync time**: %s\n", totalDuration.Round(time.Millisecond))
	}

	// File processing metrics (now shows actual git changes vs examined files)
	if rs.syncMetrics.FileMetrics.FilesProcessed > 0 {
		fmt.Fprintf(sb, "* **Files processed**: %d (%d changed, %d deleted, %d skipped)\n",
			rs.syncMetrics.FileMetrics.FilesProcessed,
			rs.syncMetrics.FileMetrics.FilesChanged-rs.syncMetrics.FileMetrics.FilesDeleted,
			rs.syncMetrics.FileMetrics.FilesDeleted,
			rs.syncMetrics.FileMetrics.FilesSkipped)

		// Show breakdown of what happened to examined files
		if rs.syncMetrics.FileMetrics.FilesAttempted > 0 {
			fmt.Fprintf(sb, "* **Files attempted to change**: %d (go-broadcast processing)\n",
				rs.syncMetrics.FileMetrics.FilesAttempted)
		}

		if rs.syncMetrics.FileMetrics.ProcessingTimeMs > 0 {
			fmt.Fprintf(sb, "* **File processing time**: %dms\n", rs.syncMetrics.FileMetrics.ProcessingTimeMs)
		}
	}

	// API efficiency metrics
	if rs.syncMetrics.APICallsSaved > 0 {
		fmt.Fprintf(sb, "* **API calls saved**: %d (through optimization)\n", rs.syncMetrics.APICallsSaved)
	}

	// Cache performance
	if rs.syncMetrics.CacheHits > 0 || rs.syncMetrics.CacheMisses > 0 {
		total := rs.syncMetrics.CacheHits + rs.syncMetrics.CacheMisses
		hitRate := float64(rs.syncMetrics.CacheHits) / float64(total) * 100
		fmt.Fprintf(sb, "* **Cache hit rate**: %.1f%% (%d hits, %d misses)\n",
			hitRate, rs.syncMetrics.CacheHits, rs.syncMetrics.CacheMisses)
	}

	sb.WriteString("\n")
}

// writeMetadataBlock writes the machine-parseable metadata block
func (rs *RepositorySync) writeMetadataBlock(sb *strings.Builder, commitSHA string, changedFiles []FileChange, prBodyAIGenerated bool) {
	sb.WriteString("<!-- go-broadcast-metadata\n")

	// Add group information if available
	if rs.engine != nil {
		if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
			sb.WriteString("group:\n")
			fmt.Fprintf(sb, "  id: %s\n", currentGroup.ID)
			fmt.Fprintf(sb, "  name: %s\n", currentGroup.Name)
		}
	}

	// Add diff debugging info - helps diagnose AI description mismatches
	sb.WriteString("diff_info:\n")
	fmt.Fprintf(sb, "  staged_repo_available: %t\n", rs.stagedRepoPath != "")
	fmt.Fprintf(sb, "  changed_files_count: %d\n", len(changedFiles))
	// Count files with/without original content to diagnose synthetic diff issues
	withOriginal := 0
	withoutOriginal := 0
	for _, f := range changedFiles {
		if f.OriginalContent != nil {
			withOriginal++
		} else {
			withoutOriginal++
		}
	}
	fmt.Fprintf(sb, "  files_with_original_content: %d\n", withOriginal)
	fmt.Fprintf(sb, "  files_without_original_content: %d\n", withoutOriginal)

	sb.WriteString("sync_metadata:\n")
	fmt.Fprintf(sb, "  source_repo: %s\n", rs.sourceState.Repo)
	fmt.Fprintf(sb, "  source_commit: %s\n", rs.sourceState.LatestCommit)
	fmt.Fprintf(sb, "  target_repo: %s\n", rs.target.Repo)
	fmt.Fprintf(sb, "  sync_commit: %s\n", commitSHA)
	fmt.Fprintf(sb, "  sync_time: %s\n", time.Now().Format(time.RFC3339))

	// Add AI generation status
	sb.WriteString("ai_generated:\n")
	fmt.Fprintf(sb, "  commit_message: %t\n", rs.commitAIGenerated)
	fmt.Fprintf(sb, "  pr_body: %t\n", prBodyAIGenerated)

	// Add directory information if directories are configured
	if len(rs.target.Directories) > 0 {
		sb.WriteString("directories:\n")
		for _, dirMapping := range rs.target.Directories {
			fmt.Fprintf(sb, "  - src: %s\n", dirMapping.Src)
			fmt.Fprintf(sb, "    dest: %s\n", dirMapping.Dest)

			// Add exclusion patterns
			if len(dirMapping.Exclude) > 0 {
				sb.WriteString("    excluded: [")
				for i, pattern := range dirMapping.Exclude {
					if i > 0 {
						sb.WriteString(", ")
					}
					fmt.Fprintf(sb, "\"%s\"", pattern)
				}
				sb.WriteString("]\n")
			}

			// Add metrics if available
			if rs.syncMetrics != nil && rs.syncMetrics.DirectoryMetrics != nil {
				if metrics, exists := rs.syncMetrics.GetDirectoryMetric(dirMapping.Src); exists {
					fmt.Fprintf(sb, "    files_synced: %d\n", metrics.FilesChanged)
					fmt.Fprintf(sb, "    files_examined: %d\n", metrics.FilesProcessed)
					fmt.Fprintf(sb, "    files_excluded: %d\n", metrics.FilesExcluded)
					if metrics.EndTime.After(metrics.StartTime) {
						duration := metrics.EndTime.Sub(metrics.StartTime)
						fmt.Fprintf(sb, "    processing_time_ms: %d\n", duration.Milliseconds())
					}
				}
			}
		}
	}

	// Add performance metrics
	if rs.syncMetrics != nil {
		sb.WriteString("performance:\n")

		// Total file counts (FileMetrics now includes directory files)
		if rs.syncMetrics.FileMetrics.FilesProcessed > 0 {
			fmt.Fprintf(sb, "  total_files: %d\n", rs.syncMetrics.FileMetrics.FilesProcessed)
		}
		if rs.syncMetrics.FileMetrics.FilesDeleted > 0 {
			fmt.Fprintf(sb, "  files_deleted: %d\n", rs.syncMetrics.FileMetrics.FilesDeleted)
		}

		if rs.syncMetrics.APICallsSaved > 0 {
			fmt.Fprintf(sb, "  api_calls_saved: %d\n", rs.syncMetrics.APICallsSaved)
		}

		if rs.syncMetrics.CacheHits > 0 {
			fmt.Fprintf(sb, "  cache_hits: %d\n", rs.syncMetrics.CacheHits)
		}
	}

	sb.WriteString("-->\n")
}

// isDirectoryFile determines if a file change is from directory processing
func (rs *RepositorySync) isDirectoryFile(filePath string) bool {
	// Check if the file path matches any of the configured directory destinations
	for _, dirMapping := range rs.target.Directories {
		if strings.HasPrefix(filePath, dirMapping.Dest+"/") || filePath == dirMapping.Dest {
			return true
		}
	}
	return false
}

// processDirectoriesWithMetrics processes directories and collects detailed metrics
func (rs *RepositorySync) processDirectoriesWithMetrics(ctx context.Context) ([]FileChange, map[string]DirectoryMetrics, error) {
	if len(rs.target.Directories) == 0 {
		rs.logger.Debug("No directories configured for sync")
		return nil, make(map[string]DirectoryMetrics), nil
	}

	// Check for context cancellation early
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("context canceled before directory processing with metrics: %w", err)
	}

	processTimer := metrics.StartTimer(ctx, rs.logger, "directory_processing_with_metrics").
		AddField("directory_count", len(rs.target.Directories))

	rs.logger.WithField("directory_count", len(rs.target.Directories)).Info("Processing directories with metrics collection")

	// Construct source repo URL for module-aware sync
	sourceRepoURL := ""
	if rs.sourceState != nil && rs.sourceState.Repo != "" {
		sourceRepoURL = fmt.Sprintf("https://github.com/%s", rs.sourceState.Repo)
	}

	// Build directory processor options
	var opts *DirectoryProcessorOptions
	if rs.engine != nil {
		opts = &DirectoryProcessorOptions{
			GitClient:     rs.engine.GitClient(),
			SourceRepoURL: sourceRepoURL,
			TempDir:       rs.tempDir,
		}
	}

	// Create directory processor with module-aware sync support
	processor := NewDirectoryProcessor(rs.logger, 10, opts)
	defer processor.Close()

	var allChanges []FileChange
	collectedMetrics := make(map[string]DirectoryMetrics)
	var processingErrors []error

	// Process each directory mapping and collect metrics
	for _, dirMapping := range rs.target.Directories {
		// Check for context cancellation during processing
		if err := ctx.Err(); err != nil {
			return nil, nil, fmt.Errorf("context canceled during directory processing with metrics: %w", err)
		}

		dirProcessingStart := time.Now()

		// Build the source path using the same logic as processDirectories
		// This should match the pattern used in regular directory processing
		sourcePath := filepath.Join(rs.tempDir, "source")
		fullSourceDir := filepath.Join(sourcePath, dirMapping.Src)

		// Verify source directory exists
		if _, err := os.Stat(fullSourceDir); os.IsNotExist(err) {
			rs.logger.WithError(err).WithField("directory", dirMapping.Src).Error("Source directory does not exist")
			processingErrors = append(processingErrors, fmt.Errorf("%w: %s", ErrSourceDirectoryNotExistForMetrics, fullSourceDir))
			continue
		}

		// Pass the sourcePath to ProcessDirectoryMapping (same as regular processDirectories)
		// ProcessDirectoryMapping will join it with dirMapping.Src internally
		changes, err := processor.ProcessDirectoryMapping(ctx, sourcePath, dirMapping, rs.target, rs.sourceState, rs.engine)
		if err != nil {
			// Log error and collect for potential failure decision
			rs.logger.WithError(err).WithField("directory", dirMapping.Src).Error("Failed to process directory")
			processingErrors = append(processingErrors, err)
			continue
		}

		// Collect metrics for this directory
		// Use source path as the key to match how metrics are retrieved when generating PR body
		metricKey := dirMapping.Src

		dirStats := processor.GetDirectoryStats()
		if dirMetrics, exists := dirStats[dirMapping.Src]; exists {
			// Ensure timing is set if not already
			if dirMetrics.EndTime.IsZero() {
				dirMetrics.EndTime = time.Now()
			}
			if dirMetrics.StartTime.IsZero() {
				dirMetrics.StartTime = dirProcessingStart
			}
			collectedMetrics[metricKey] = dirMetrics
		} else {
			// Create basic metrics if not available from processor
			collectedMetrics[metricKey] = DirectoryMetrics{
				StartTime:      dirProcessingStart,
				EndTime:        time.Now(),
				FilesProcessed: len(changes),
			}
		}

		allChanges = append(allChanges, changes...)
	}

	// If all directories failed, return an error with details from the first error
	if len(processingErrors) > 0 && len(allChanges) == 0 {
		// Include the first error's details to preserve context cancellation information
		return nil, nil, fmt.Errorf("%w: %d errors occurred, first error: %w", ErrAllDirectoryProcessingWithMetricsFailed, len(processingErrors), processingErrors[0])
	}

	processTimer.AddField("total_changes", len(allChanges)).
		AddField("directories_processed", len(collectedMetrics)).Stop()

	// Collect module updates from processor
	rs.moduleUpdates = processor.GetModuleUpdates()
	if len(rs.moduleUpdates) > 0 {
		rs.logger.WithField("module_updates", len(rs.moduleUpdates)).Info("Collected module updates for go.mod")
	}

	rs.logger.WithFields(logrus.Fields{
		"total_changes":         len(allChanges),
		"directories_processed": len(collectedMetrics),
	}).Info("Directory processing with metrics completed")

	return allChanges, collectedMetrics, nil
}

// TrackAPICallSaved increments the API calls saved counter
func (rs *RepositorySync) TrackAPICallSaved(count int) {
	if rs.syncMetrics != nil {
		rs.syncMetrics.APICallsSaved += count
	}
}

// TrackCacheHit increments the cache hit counter
func (rs *RepositorySync) TrackCacheHit() {
	if rs.syncMetrics != nil {
		rs.syncMetrics.CacheHits++
	}
}

// TrackCacheMiss increments the cache miss counter
func (rs *RepositorySync) TrackCacheMiss() {
	if rs.syncMetrics != nil {
		rs.syncMetrics.CacheMisses++
	}
}

// TrackAPIRequest increments the total API requests counter
func (rs *RepositorySync) TrackAPIRequest() {
	if rs.syncMetrics != nil {
		rs.syncMetrics.TotalAPIRequests++
	}
}

// DryRunOutput handles clean console output for dry-run mode
type DryRunOutput struct {
	writer io.Writer
}

// NewDryRunOutput creates a new DryRunOutput instance
func NewDryRunOutput(writer io.Writer) *DryRunOutput {
	if writer == nil {
		writer = os.Stdout
	}
	return &DryRunOutput{writer: writer}
}

// Header prints a formatted header
func (d *DryRunOutput) Header(title string) {
	_, _ = fmt.Fprintln(d.writer)
	_, _ = fmt.Fprintf(d.writer, "🔍 %s\n", title)
	_, _ = fmt.Fprintln(d.writer, "┌─────────────────────────────────────────────────────────────────")
}

// Field prints a formatted field with label and value
func (d *DryRunOutput) Field(label, value string) {
	_, _ = fmt.Fprintf(d.writer, "│ %s: %s\n", label, value)
}

// Separator prints a horizontal separator line
func (d *DryRunOutput) Separator() {
	_, _ = fmt.Fprintln(d.writer, "├─────────────────────────────────────────────────────────────────")
}

// Content prints content with proper formatting
func (d *DryRunOutput) Content(line string) {
	if strings.TrimSpace(line) == "" {
		_, _ = fmt.Fprintln(d.writer, "│")
	} else {
		_, _ = fmt.Fprintf(d.writer, "│ %s\n", line)
	}
}

// Footer prints the closing border
func (d *DryRunOutput) Footer() {
	_, _ = fmt.Fprintln(d.writer, "└─────────────────────────────────────────────────────────────────")
}

// Info prints an informational message
func (d *DryRunOutput) Info(message string) {
	_, _ = fmt.Fprintf(d.writer, "   %s\n", message)
}

// Warning prints a warning message
func (d *DryRunOutput) Warning(message string) {
	_, _ = fmt.Fprintf(d.writer, "⚠️  %s\n", message)
}

// Success prints a success message
func (d *DryRunOutput) Success(message string) {
	_, _ = fmt.Fprintf(d.writer, "✅ %s\n", message)
}

// recordTargetResult records per-target metrics after a sync attempt (success or failure).
func (rs *RepositorySync) recordTargetResult(
	ctx context.Context,
	branchName, commitSHA string,
	allChanges []FileChange,
	actualChangedFiles []string,
	syncErr error,
	statusOverride string,
) error {
	run := rs.engine.GetCurrentRun()
	if run == nil {
		return nil
	}

	log := rs.logger.WithField("component", "metrics_recording")

	// Determine status
	status := statusOverride
	if status == "" {
		if syncErr != nil {
			status = TargetStatusFailed
		} else {
			status = TargetStatusSuccess
		}
	}

	// Resolve repo DB ID
	repoID, err := rs.engine.syncRepo.LookupRepoID(ctx, rs.target.Repo)
	if err != nil {
		log.WithError(err).Debug("Could not resolve target repo DB ID, skipping target result recording")
		return nil
	}

	// Resolve group DB ID
	var groupExternalID string
	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		groupExternalID = currentGroup.ID
	} else if len(rs.engine.config.Groups) > 0 {
		groupExternalID = rs.engine.config.Groups[0].ID
	}

	var targetID uint
	if groupExternalID != "" {
		groupDBID, gErr := rs.engine.syncRepo.LookupGroupID(ctx, groupExternalID)
		if gErr != nil {
			log.WithError(gErr).Debug("Could not resolve group DB ID, skipping target result recording")
			return nil
		}
		tID, tErr := rs.engine.syncRepo.LookupTargetID(ctx, groupDBID, rs.target.Repo)
		if tErr != nil {
			log.WithError(tErr).Debug("Could not resolve target DB ID, skipping target result recording")
			return nil
		}
		targetID = tID
	}

	// Calculate per-file line counts
	var totalLinesAdded, totalLinesRemoved int
	var totalBytesChanged int64
	fileChanges := make([]BroadcastSyncFileChange, 0, len(allChanges))

	for i, fc := range allChanges {
		linesAdded, linesRemoved := ai.CountDiffLines(string(fc.OriginalContent), string(fc.Content))
		totalLinesAdded += linesAdded
		totalLinesRemoved += linesRemoved
		totalBytesChanged += int64(len(fc.Content))

		// Determine change type
		changeType := FileChangeTypeModified
		if fc.IsNew {
			changeType = FileChangeTypeAdded
		} else if fc.IsDeleted {
			changeType = FileChangeTypeDeleted
		}

		fileChanges = append(fileChanges, BroadcastSyncFileChange{
			FilePath:     fc.Path,
			ChangeType:   changeType,
			LinesAdded:   linesAdded,
			LinesRemoved: linesRemoved,
			SizeBytes:    int64(len(fc.Content)),
			Position:     i,
		})
	}

	// Build timing info
	endTime := time.Now()
	var startedAt time.Time
	if rs.syncMetrics != nil {
		startedAt = rs.syncMetrics.StartTime
	} else {
		startedAt = endTime
	}
	durationMs := endTime.Sub(startedAt).Milliseconds()

	// Determine files changed count (use actual git changes if available)
	filesChanged := len(actualChangedFiles)
	if filesChanged == 0 {
		filesChanged = len(allChanges)
	}

	// Build error message
	var errorMsg string
	if syncErr != nil {
		errorMsg = syncErr.Error()
	}

	// Determine PR state
	var prState string
	if rs.lastPRNumber != nil {
		prState = "open"
	}

	result := BroadcastSyncTargetResult{
		BroadcastSyncRunID: run.ID,
		TargetID:           targetID,
		RepoID:             repoID,
		StartedAt:          startedAt,
		EndedAt:            &endTime,
		DurationMs:         durationMs,
		Status:             status,
		BranchName:         branchName,
		SourceCommitSHA:    commitSHA,
		FilesProcessed:     len(allChanges),
		FilesChanged:       filesChanged,
		LinesAdded:         totalLinesAdded,
		LinesRemoved:       totalLinesRemoved,
		BytesChanged:       totalBytesChanged,
		PRNumber:           rs.lastPRNumber,
		PRURL:              rs.lastPRURL,
		PRState:            prState,
		ErrorMessage:       errorMsg,
	}

	if err := rs.engine.syncRepo.CreateTargetResult(ctx, &result); err != nil {
		return fmt.Errorf("failed to create target result: %w", err)
	}

	// Set the result ID on file changes and create them
	if len(fileChanges) > 0 {
		for i := range fileChanges {
			fileChanges[i].BroadcastSyncTargetResultID = result.ID
		}
		if err := rs.engine.syncRepo.CreateFileChanges(ctx, fileChanges); err != nil {
			log.WithError(err).Warn("Failed to create file change records")
		}
	}

	// Aggregate stats to the run totals
	rs.engine.RecordTargetStats(filesChanged, totalLinesAdded, totalLinesRemoved)

	log.WithFields(logrus.Fields{
		"target_repo":   rs.target.Repo,
		"status":        status,
		"files_changed": filesChanged,
		"lines_added":   totalLinesAdded,
		"lines_removed": totalLinesRemoved,
	}).Debug("Target result metrics recorded")

	return nil
}

// FileChange represents a change to a file
type FileChange struct {
	Path            string
	Content         []byte
	OriginalContent []byte
	IsNew           bool
	IsDeleted       bool
}

// showDryRunCommitInfo displays commit information preview for dry-run.
// Accepts pre-generated commit message to avoid redundant AI calls.
// aiGenerated indicates whether the message was actually generated by AI (not fallback).
func (rs *RepositorySync) showDryRunCommitInfo(commitMsg string, changedFiles []FileChange, aiGenerated bool) {
	rs.logger.Debug("Showing dry-run commit preview")

	out := NewDryRunOutput(nil)

	out.Header("📋 COMMIT PREVIEW")
	out.Field("Message", commitMsg)
	// Show AI status indicator based on actual generation result
	if aiGenerated {
		out.Field("Source", "🤖 AI-generated")
	} else if rs.engine != nil && rs.engine.commitGenerator != nil {
		out.Field("Source", "📝 Static template (AI generation failed)")
	} else {
		out.Field("Source", "📝 Static template")
	}
	out.Field("Files", fmt.Sprintf("%d changed", len(changedFiles)))

	// Show file summary
	fileNames := make([]string, 0, len(changedFiles))
	for _, file := range changedFiles {
		fileNames = append(fileNames, file.Path)
	}
	if len(fileNames) <= 3 {
		out.Field("", strings.Join(fileNames, ", "))
	} else {
		out.Field("", fmt.Sprintf("%s, ... and %d more",
			strings.Join(fileNames[:3], ", "), len(fileNames)-3))
	}
	out.Footer()
}

// showDryRunFileChanges displays file changes in a readable format
func (rs *RepositorySync) showDryRunFileChanges(changedFiles []FileChange) {
	rs.logger.WithField("changed_files", len(changedFiles)).Debug("Showing file changes preview")

	out := NewDryRunOutput(nil)
	_, _ = fmt.Fprintln(out.writer, "📄 FILE CHANGES:")

	for _, file := range changedFiles {
		status := "modified"
		icon := "📝"
		if file.IsNew {
			status = "added"
			icon = "✨"
		}

		// Calculate size info if content is available
		sizeInfo := ""
		if len(file.Content) > 0 {
			if file.IsNew {
				sizeInfo = fmt.Sprintf(" (+%d bytes)", len(file.Content))
			} else if len(file.OriginalContent) > 0 {
				sizeDiff := len(file.Content) - len(file.OriginalContent)
				if sizeDiff > 0 {
					sizeInfo = fmt.Sprintf(" (+%d bytes)", sizeDiff)
				} else if sizeDiff < 0 {
					sizeInfo = fmt.Sprintf(" (%d bytes)", sizeDiff)
				}
			}
		}

		out.Info(fmt.Sprintf("%s %s (%s)%s", icon, file.Path, status, sizeInfo))
	}
}

// formatAssignmentList formats a slice of strings into a comma-separated list or returns "none"
func (rs *RepositorySync) formatAssignmentList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

// formatReviewersWithFiltering formats reviewers list showing which ones will be filtered
func (rs *RepositorySync) formatReviewersWithFiltering(reviewers []string, currentUserLogin string) string {
	if len(reviewers) == 0 {
		return "none"
	}

	formatted := make([]string, 0, len(reviewers))
	for _, reviewer := range reviewers {
		if currentUserLogin != "" && reviewer == currentUserLogin {
			formatted = append(formatted, fmt.Sprintf("%s (author - will be filtered)", reviewer))
		} else {
			formatted = append(formatted, reviewer)
		}
	}
	return strings.Join(formatted, ", ")
}

// showDryRunPRPreview displays full PR preview with formatting.
// Accepts pre-generated title and body to avoid redundant AI calls.
// aiGenerated indicates whether the body was actually generated by AI (not fallback).
func (rs *RepositorySync) showDryRunPRPreview(ctx context.Context, branchName, title, body string, aiGenerated bool) {
	rs.logger.WithFields(logrus.Fields{
		"branch": branchName,
	}).Debug("Showing PR preview")

	out := NewDryRunOutput(nil)

	out.Header("DRY-RUN: Pull Request Preview")
	out.Field("Repository", rs.target.Repo)
	out.Field("Branch", branchName)
	out.Separator()
	out.Field("Title", title)
	// Show AI status indicator based on actual generation result
	if aiGenerated {
		out.Field("Body Source", "🤖 AI-generated")
	} else if rs.engine != nil && rs.engine.prGenerator != nil {
		out.Field("Body Source", "📝 Static template (AI generation failed)")
	} else {
		out.Field("Body Source", "📝 Static template (AI disabled)")
	}
	out.Separator()

	// Get current user for reviewer filtering display
	var currentUserLogin string
	rs.TrackAPIRequest()
	currentUser, err := rs.engine.gh.GetCurrentUser(ctx)
	if err != nil {
		rs.logger.WithError(err).Debug("Failed to get current user for dry-run display")
	} else if currentUser != nil {
		currentUserLogin = currentUser.Login
	}

	// Show PR assignment details
	out.Content("Assignment Details:")
	out.Content(fmt.Sprintf("• Assignees: %s", rs.formatAssignmentList(rs.getPRAssignees())))
	out.Content(fmt.Sprintf("• Labels: %s", rs.formatAssignmentList(rs.getPRLabels())))
	out.Content(fmt.Sprintf("• Reviewers: %s", rs.formatReviewersWithFiltering(rs.getPRReviewers(), currentUserLogin)))
	out.Content(fmt.Sprintf("• Team Reviewers: %s", rs.formatAssignmentList(rs.getPRTeamReviewers())))
	out.Separator()

	// Split body into lines and display with proper formatting
	bodyLines := strings.Split(body, "\n")
	for _, line := range bodyLines {
		out.Content(line)
	}

	out.Footer()
}

// mergeUniqueStrings merges two string slices, removing duplicates while preserving order
// Items from the first slice take precedence in ordering
func (rs *RepositorySync) mergeUniqueStrings(slice1, slice2 []string) []string {
	if len(slice1) == 0 && len(slice2) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(slice1)+len(slice2))

	// Add items from first slice
	for _, item := range slice1 {
		if item != "" && !seen[item] {
			result = append(result, item)
			seen[item] = true
		}
	}

	// Add items from second slice that haven't been seen
	for _, item := range slice2 {
		if item != "" && !seen[item] {
			result = append(result, item)
			seen[item] = true
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// getPRAssignees returns the assignees to use for PRs, merging global + target assignments
func (rs *RepositorySync) getPRAssignees() []string {
	var global []string
	var defaults []string

	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		global = currentGroup.Global.PRAssignees
		defaults = currentGroup.Defaults.PRAssignees
	} else {
		// Get from the first group (since we have a single group in temporary config)
		if len(rs.engine.config.Groups) > 0 {
			global = rs.engine.config.Groups[0].Global.PRAssignees
			defaults = rs.engine.config.Groups[0].Defaults.PRAssignees
		}
	}

	target := rs.target.PRAssignees

	// Merge global + target (unique)
	combined := rs.mergeUniqueStrings(global, target)

	// Fall back to defaults if no assignments
	if len(combined) == 0 {
		return defaults
	}
	return combined
}

// getPRReviewers returns the reviewers to use for PRs, merging global + target assignments
func (rs *RepositorySync) getPRReviewers() []string {
	var global []string
	var defaults []string

	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		global = currentGroup.Global.PRReviewers
		defaults = currentGroup.Defaults.PRReviewers
	} else {
		// Get from the first group (since we have a single group in temporary config)
		if len(rs.engine.config.Groups) > 0 {
			global = rs.engine.config.Groups[0].Global.PRReviewers
			defaults = rs.engine.config.Groups[0].Defaults.PRReviewers
		}
	}

	target := rs.target.PRReviewers

	// Merge global + target (unique)
	combined := rs.mergeUniqueStrings(global, target)

	// Fall back to defaults if no assignments
	if len(combined) == 0 {
		return defaults
	}
	return combined
}

// getPRLabels returns the labels to use for PRs, merging global + target assignments
func (rs *RepositorySync) getPRLabels() []string {
	var global []string
	var defaults []string

	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		global = currentGroup.Global.PRLabels
		defaults = currentGroup.Defaults.PRLabels
	} else {
		// Get from the first group (since we have a single group in temporary config)
		if len(rs.engine.config.Groups) > 0 {
			global = rs.engine.config.Groups[0].Global.PRLabels
			defaults = rs.engine.config.Groups[0].Defaults.PRLabels
		}
	}

	target := rs.target.PRLabels

	// Merge global + target (unique)
	combined := rs.mergeUniqueStrings(global, target)

	// Fall back to defaults if no assignments
	if len(combined) == 0 {
		combined = defaults
	}

	// Add automerge labels if automerge is enabled
	if rs.engine.options != nil && rs.engine.options.Automerge && len(rs.engine.options.AutomergeLabels) > 0 {
		combined = rs.mergeUniqueStrings(combined, rs.engine.options.AutomergeLabels)
	}

	return combined
}

// getPRTeamReviewers returns the team reviewers to use for PRs, merging global + target assignments
func (rs *RepositorySync) getPRTeamReviewers() []string {
	var global []string
	var defaults []string

	if currentGroup := rs.engine.GetCurrentGroup(); currentGroup != nil {
		global = currentGroup.Global.PRTeamReviewers
		defaults = currentGroup.Defaults.PRTeamReviewers
	} else {
		// Get from the first group (since we have a single group in temporary config)
		if len(rs.engine.config.Groups) > 0 {
			global = rs.engine.config.Groups[0].Global.PRTeamReviewers
			defaults = rs.engine.config.Groups[0].Defaults.PRTeamReviewers
		}
	}

	target := rs.target.PRTeamReviewers

	// Merge global + target (unique)
	combined := rs.mergeUniqueStrings(global, target)

	// Fall back to defaults if no assignments
	if len(combined) == 0 {
		return defaults
	}
	return combined
}

// updateDirectoryMetricsWithActualChanges updates directory metrics with the files that actually changed in git
func (rs *RepositorySync) updateDirectoryMetricsWithActualChanges(actualChangedFiles []string) {
	if rs.syncMetrics == nil || rs.syncMetrics.DirectoryMetrics == nil {
		return
	}

	// Reset FilesChanged counts for all directories.
	// We collect keys first, then update outside the iteration to avoid deadlock
	// (IterateDirectoryMetrics holds RLock, SetDirectoryMetric needs Lock).
	var dirPaths []string
	rs.syncMetrics.IterateDirectoryMetrics(func(dirPath string, _ DirectoryMetrics) {
		dirPaths = append(dirPaths, dirPath)
	})
	for _, dirPath := range dirPaths {
		if metrics, exists := rs.syncMetrics.GetDirectoryMetric(dirPath); exists {
			metrics.FilesChanged = 0
			rs.syncMetrics.SetDirectoryMetric(dirPath, metrics)
		}
	}

	// Count actual changes per directory
	for _, filePath := range actualChangedFiles {
		// Find which directory this file belongs to
		for _, dirMapping := range rs.target.Directories {
			if rs.isFileInDirectory(filePath, dirMapping.Dest) {
				if metrics, exists := rs.syncMetrics.GetDirectoryMetric(dirMapping.Src); exists {
					metrics.FilesChanged++
					rs.syncMetrics.SetDirectoryMetric(dirMapping.Src, metrics)
					rs.logger.WithFields(logrus.Fields{
						"file":      filePath,
						"directory": dirMapping.Src,
						"new_count": metrics.FilesChanged,
					}).Debug("Updated directory FilesChanged count with actual git change")
					break // File can only belong to one directory mapping
				}
			}
		}
	}

	// Log final counts for verification
	rs.syncMetrics.IterateDirectoryMetrics(func(dirPath string, metrics DirectoryMetrics) {
		rs.logger.WithFields(logrus.Fields{
			"directory":       dirPath,
			"files_processed": metrics.FilesProcessed,
			"files_changed":   metrics.FilesChanged,
		}).Debug("Final directory metrics after git change tracking")
	})
}

// isFileInDirectory checks if a file path belongs to a specific directory
func (rs *RepositorySync) isFileInDirectory(filePath, directoryPath string) bool {
	// Normalize paths to use forward slashes
	filePath = filepath.ToSlash(filePath)
	directoryPath = filepath.ToSlash(directoryPath)

	// Check if file is directly in the directory or a subdirectory
	return strings.HasPrefix(filePath, directoryPath+"/") || filePath == directoryPath
}
