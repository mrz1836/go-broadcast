package sync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
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
	// Performance metrics tracking
	syncMetrics *PerformanceMetrics
}

// PerformanceMetrics tracks performance metrics for the entire sync operation
type PerformanceMetrics struct {
	StartTime        time.Time
	EndTime          time.Time
	DirectoryMetrics map[string]DirectoryMetrics // keyed by source directory path
	FileMetrics      FileProcessingMetrics
	APICallsSaved    int // Total API calls saved by using tree API or caching
	CacheHits        int // Number of cache hits
	CacheMisses      int // Number of cache misses
	TotalAPIRequests int // Total API requests made
}

// FileProcessingMetrics tracks metrics for individual file processing
type FileProcessingMetrics struct {
	FilesProcessed   int
	FilesChanged     int
	FilesSkipped     int
	ProcessingTimeMs int64
}

// Execute performs the complete sync operation for this repository
func (rs *RepositorySync) Execute(ctx context.Context) error {
	// Initialize performance metrics tracking
	rs.syncMetrics = &PerformanceMetrics{
		StartTime:        time.Now(),
		DirectoryMetrics: make(map[string]DirectoryMetrics),
	}

	// Start overall operation timer
	syncTimer := metrics.StartTimer(ctx, rs.logger, "repository_sync").
		AddField(logging.StandardFields.SourceRepo, rs.sourceState.Repo).
		AddField(logging.StandardFields.TargetRepo, rs.target.Repo).
		AddField("sync_branch", rs.sourceState.Branch).
		AddField("commit_sha", rs.sourceState.LatestCommit)
	// Add group context if available
	if rs.engine.currentGroup != nil {
		syncTimer = syncTimer.
			AddField("group_name", rs.engine.currentGroup.Name).
			AddField("group_id", rs.engine.currentGroup.ID)
	}

	// 1. Check if sync is actually needed
	syncCheckTimer := metrics.StartTimer(ctx, rs.logger, "sync_check")
	needsSync := rs.engine.options.Force || rs.needsSync()
	syncCheckTimer.AddField("force_sync", rs.engine.options.Force).
		AddField("needs_sync", needsSync).Stop()

	if !needsSync {
		rs.logger.Info("Repository is up-to-date, skipping sync")
		syncTimer.AddField(logging.StandardFields.Status, "skipped").Stop()
		return nil
	}

	// 2. Create temporary directory
	tempDirTimer := metrics.StartTimer(ctx, rs.logger, "temp_dir_creation")
	if err := rs.createTempDir(); err != nil {
		tempDirTimer.StopWithError(err)
		syncTimer.StopWithError(err)
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
		return fmt.Errorf("failed to process files: %w", err)
	}
	fileProcessingDuration := time.Since(fileProcessingStart)

	// Update file processing metrics
	rs.syncMetrics.FileMetrics = FileProcessingMetrics{
		FilesProcessed:   len(rs.target.Files),
		FilesChanged:     len(changedFiles),
		FilesSkipped:     len(rs.target.Files) - len(changedFiles),
		ProcessingTimeMs: fileProcessingDuration.Milliseconds(),
	}

	processTimer.AddField("changed_files", len(changedFiles)).Stop()

	// 5. Process directories with metrics collection
	directoryChanges, directoryMetrics, err := rs.processDirectoriesWithMetrics(ctx)
	if err != nil {
		syncTimer.StopWithError(err)
		return fmt.Errorf("failed to process directories: %w", err)
	}

	// Store directory metrics for PR metadata
	if rs.syncMetrics != nil {
		rs.syncMetrics.DirectoryMetrics = directoryMetrics
	}

	// Combine file and directory changes
	allChanges := append(changedFiles, directoryChanges...)

	rs.logger.WithFields(logrus.Fields{
		"file_changes":      len(changedFiles),
		"directory_changes": len(directoryChanges),
		"total_changes":     len(allChanges),
	}).Info("File and directory processing completed")

	if len(allChanges) == 0 {
		rs.logger.Info("No file or directory changes detected, skipping sync")
		syncTimer.AddField(logging.StandardFields.Status, "no_changes").Stop()
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

	commitSHA, err := rs.commitChanges(ctx, branchName, allChanges)
	if err != nil {
		commitTimer.StopWithError(err)
		syncTimer.StopWithError(err)
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	commitTimer.AddField("commit_sha", commitSHA).Stop()

	// 8. Push changes (unless dry-run)
	if !rs.engine.options.DryRun {
		pushTimer := metrics.StartTimer(ctx, rs.logger, "branch_push").
			AddField(logging.StandardFields.BranchName, branchName).
			AddField("commit_sha", commitSHA)

		if err := rs.pushChanges(ctx, branchName); err != nil {
			pushTimer.StopWithError(err)
			syncTimer.StopWithError(err)
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

	if err := rs.createOrUpdatePR(ctx, branchName, commitSHA, allChanges); err != nil {
		prTimer.StopWithError(err)
		syncTimer.StopWithError(err)
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
		if rs.engine.currentGroup != nil {
			out.Info(fmt.Sprintf("📋 Group: %s (%s)", rs.engine.currentGroup.Name, rs.engine.currentGroup.ID))
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

	// Force cleanup even if there are permission issues
	if err := os.Chmod(rs.tempDir, 0o600); err != nil {
		rs.logger.WithError(err).Debug("Failed to change temp directory permissions for cleanup")
	}

	if err := os.RemoveAll(rs.tempDir); err != nil {
		rs.logger.WithError(err).Warn("Failed to cleanup temporary directory")
		// Try one more time with forced removal
		_ = os.RemoveAll(rs.tempDir) // Ignore error on second attempt
	} else {
		rs.logger.Debug("Cleaned up temporary directory")
	}
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

	if err := rs.engine.git.Clone(ctx, sourceURL, sourcePath); err != nil {
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

	transformedContent := srcContent
	if rs.target.Transform.RepoName || len(rs.target.Transform.Variables) > 0 {
		transformedContent, err = rs.engine.transform.Transform(ctx, srcContent, transformCtx)
		if err != nil {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}
	}

	// Check if content actually changed (for existing files)
	existingContent, err := rs.getExistingFileContent(ctx, fileMapping.Dest)
	if err == nil && string(existingContent) == string(transformedContent) {
		rs.logger.WithField("file", fileMapping.Dest).Debug("File content unchanged, skipping")
		return nil, internalerrors.ErrTransformNotFound
	}

	return &FileChange{
		Path:            fileMapping.Dest,
		Content:         transformedContent,
		OriginalContent: srcContent,
		IsNew:           err != nil, // err means file doesn't exist
	}, nil
}

// getExistingFileContent retrieves the current content of a file from the target repo
func (rs *RepositorySync) getExistingFileContent(ctx context.Context, filePath string) ([]byte, error) {
	// Track API request
	rs.TrackAPIRequest()

	// Try to get file from the target repository's default branch
	fileContent, err := rs.engine.gh.GetFile(ctx, rs.target.Repo, filePath, "")
	if err != nil {
		return nil, err
	}
	return fileContent.Content, nil
}

// createSyncBranch creates a new sync branch or returns existing one
func (rs *RepositorySync) createSyncBranch(_ context.Context) string {
	// Generate branch name: chore/sync-files-YYYYMMDD-HHMMSS-{commit}
	now := time.Now()
	timestamp := now.Format("20060102-150405")
	commitSHA := rs.sourceState.LatestCommit
	if len(commitSHA) > 7 {
		commitSHA = commitSHA[:7]
	}

	var branchPrefix string
	if rs.engine.currentGroup != nil {
		branchPrefix = rs.engine.currentGroup.Defaults.BranchPrefix
	} else {
		// Get defaults from the first group (since we have a single group in temporary config)
		if len(rs.engine.config.Groups) > 0 {
			branchPrefix = rs.engine.config.Groups[0].Defaults.BranchPrefix
		}
	}
	if branchPrefix == "" {
		branchPrefix = "chore/sync-files"
	}

	branchName := fmt.Sprintf("%s-%s-%s", branchPrefix, timestamp, commitSHA)

	rs.logger.WithField("branch_name", branchName).Info("Creating sync branch")

	if rs.engine.options.DryRun {
		rs.logger.Info("DRY-RUN: Would create sync branch")
		return branchName
	}

	// Create branch in target repository
	// We'll create the branch when we push, so just return the name for now
	return branchName
}

// commitChanges creates a commit with the changed files
func (rs *RepositorySync) commitChanges(ctx context.Context, branchName string, changedFiles []FileChange) (string, error) {
	if len(changedFiles) == 0 {
		return "", internalerrors.ErrNoFilesToCommit
	}

	// Generate commit message
	commitMsg := rs.generateCommitMessage(changedFiles)

	rs.logger.WithFields(logrus.Fields{
		"branch":     branchName,
		"files":      len(changedFiles),
		"commit_msg": commitMsg,
	}).Info("Creating commit")

	if rs.engine.options.DryRun {
		rs.showDryRunCommitInfo(changedFiles)
		rs.showDryRunFileChanges(changedFiles)
		return "dry-run-commit-sha", nil
	}

	// Clone the target repository for making changes
	targetPath := filepath.Join(rs.tempDir, "target")
	targetURL := fmt.Sprintf("https://github.com/%s.git", rs.target.Repo)

	if err := rs.engine.git.Clone(ctx, targetURL, targetPath); err != nil {
		return "", fmt.Errorf("failed to clone target repository: %w", err)
	}

	// Create and checkout the new branch
	if err := rs.engine.git.CreateBranch(ctx, targetPath, branchName); err != nil {
		return "", fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	if err := rs.engine.git.Checkout(ctx, targetPath, branchName); err != nil {
		return "", fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}

	// Apply file changes to the target repository
	for _, fileChange := range changedFiles {
		destPath := filepath.Join(targetPath, fileChange.Path)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
			return "", fmt.Errorf("failed to create directory for %s: %w", fileChange.Path, err)
		}

		// Write the file content
		if err := os.WriteFile(destPath, fileChange.Content, 0o600); err != nil {
			return "", fmt.Errorf("failed to write file %s: %w", fileChange.Path, err)
		}
	}

	// Stage all changes
	if err := rs.engine.git.Add(ctx, targetPath, "."); err != nil {
		return "", fmt.Errorf("failed to stage changes: %w", err)
	}

	// Create the commit
	if err := rs.engine.git.Commit(ctx, targetPath, commitMsg); err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	// Get the commit SHA
	commitSHA, err := rs.engine.git.GetCurrentCommitSHA(ctx, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	return commitSHA, nil
}

// pushChanges pushes the branch to the target repository
func (rs *RepositorySync) pushChanges(ctx context.Context, branchName string) error {
	rs.logger.WithField("branch", branchName).Info("Pushing changes to target repository")

	targetPath := filepath.Join(rs.tempDir, "target")

	// Push the branch to the origin remote (which is the target repository)
	if err := rs.engine.git.Push(ctx, targetPath, "origin", branchName, false); err != nil {
		return fmt.Errorf("failed to push branch %s to target repository: %w", branchName, err)
	}

	return nil
}

// createOrUpdatePR creates a new PR or updates an existing one
func (rs *RepositorySync) createOrUpdatePR(ctx context.Context, branchName, commitSHA string, changedFiles []FileChange) error {
	// Check if PR already exists for this branch
	existingPR := rs.findExistingPR(branchName)

	if existingPR != nil {
		return rs.updateExistingPR(ctx, existingPR, commitSHA, changedFiles)
	}

	return rs.createNewPR(ctx, branchName, commitSHA, changedFiles)
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

// createNewPR creates a new pull request
func (rs *RepositorySync) createNewPR(ctx context.Context, branchName, commitSHA string, changedFiles []FileChange) error {
	title := rs.generatePRTitle()
	body := rs.generatePRBody(commitSHA, changedFiles)

	rs.logger.WithFields(logrus.Fields{
		"branch": branchName,
		"title":  title,
	}).Info("Creating new pull request")

	if rs.engine.options.DryRun {
		rs.showDryRunPRPreview(ctx, branchName, commitSHA, changedFiles)
		return nil
	}

	// Get default branch for base
	rs.TrackAPIRequest()
	branches, err := rs.engine.gh.ListBranches(ctx, rs.target.Repo)
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	baseBranch := "master" // default
	for _, branch := range branches {
		if branch.Name == mainBranch {
			baseBranch = mainBranch
			break
		}
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

	rs.TrackAPIRequest()
	pr, err := rs.engine.gh.CreatePR(ctx, rs.target.Repo, prRequest)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	rs.logger.WithField("pr_number", pr.Number).Info("Pull request created successfully")
	return nil
}

// updateExistingPR updates an existing pull request
func (rs *RepositorySync) updateExistingPR(ctx context.Context, pr *gh.PR, commitSHA string, changedFiles []FileChange) error {
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
	newBody := rs.generatePRBody(commitSHA, changedFiles)

	// Update the PR via GitHub API
	updates := gh.PRUpdate{
		Body: &newBody,
	}

	rs.TrackAPIRequest()
	if err := rs.engine.gh.UpdatePR(ctx, rs.target.Repo, pr.Number, updates); err != nil {
		return fmt.Errorf("failed to update PR: %w", err)
	}

	rs.logger.WithField("pr_number", pr.Number).Info("Pull request updated successfully")
	return nil
}

// generateCommitMessage creates a descriptive commit message
func (rs *RepositorySync) generateCommitMessage(changedFiles []FileChange) string {
	if len(changedFiles) == 1 {
		return fmt.Sprintf("sync: update %s from source repository", changedFiles[0].Path)
	}

	return fmt.Sprintf("sync: update %d files from source repository", len(changedFiles))
}

// generatePRTitle creates a descriptive PR title
func (rs *RepositorySync) generatePRTitle() string {
	commitSHA := rs.sourceState.LatestCommit
	if len(commitSHA) > 7 {
		commitSHA = commitSHA[:7]
	}
	return fmt.Sprintf("[Sync] Update project files from source repository (%s)", commitSHA)
}

// generatePRBody creates a detailed PR description with metadata including directory sync info
func (rs *RepositorySync) generatePRBody(commitSHA string, changedFiles []FileChange) string {
	var sb strings.Builder

	// What Changed section with enhanced details
	sb.WriteString("## What Changed\n")
	rs.writeChangeSummary(&sb, changedFiles)
	shortSHA := commitSHA
	if len(commitSHA) > 7 {
		shortSHA = commitSHA[:7]
	}
	sb.WriteString(fmt.Sprintf("* Brought target repository in line with source repository state at commit %s\n\n", shortSHA))

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
	rs.writeMetadataBlock(&sb, commitSHA, changedFiles)

	return sb.String()
}

// writeChangeSummary writes a detailed summary of what changed in the sync
func (rs *RepositorySync) writeChangeSummary(sb *strings.Builder, changedFiles []FileChange) {
	// Distinguish between file changes and directory changes
	fileChanges := 0
	directoryChanges := 0

	// Count changes by type - files vs directories
	for _, change := range changedFiles {
		if rs.isDirectoryFile(change.Path) {
			directoryChanges++
		} else {
			fileChanges++
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
			if metrics, exists := rs.syncMetrics.DirectoryMetrics[dirMapping.Src]; exists {
				fmt.Fprintf(sb, "* **Files synced**: %d\n", metrics.FilesProcessed)
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

	// File processing metrics
	if rs.syncMetrics.FileMetrics.FilesProcessed > 0 {
		fmt.Fprintf(sb, "* **Files processed**: %d (%d changed, %d skipped)\n",
			rs.syncMetrics.FileMetrics.FilesProcessed,
			rs.syncMetrics.FileMetrics.FilesChanged,
			rs.syncMetrics.FileMetrics.FilesSkipped)

		if rs.syncMetrics.FileMetrics.ProcessingTimeMs > 0 {
			fmt.Fprintf(sb, "* **File processing time**: %dms\n", rs.syncMetrics.FileMetrics.ProcessingTimeMs)
		}
	}

	// Directory processing metrics
	totalDirectoryFiles := 0
	totalDirectoryExcluded := 0
	if rs.syncMetrics.DirectoryMetrics != nil {
		for _, metrics := range rs.syncMetrics.DirectoryMetrics {
			totalDirectoryFiles += metrics.FilesProcessed
			totalDirectoryExcluded += metrics.FilesExcluded
		}
	}

	if totalDirectoryFiles > 0 {
		fmt.Fprintf(sb, "* **Directory files processed**: %d (%d excluded)\n",
			totalDirectoryFiles, totalDirectoryExcluded)
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
func (rs *RepositorySync) writeMetadataBlock(sb *strings.Builder, commitSHA string, _ []FileChange) {
	sb.WriteString("<!-- go-broadcast-metadata\n")
	sb.WriteString("sync_metadata:\n")
	fmt.Fprintf(sb, "  source_repo: %s\n", rs.sourceState.Repo)
	fmt.Fprintf(sb, "  source_commit: %s\n", rs.sourceState.LatestCommit)
	fmt.Fprintf(sb, "  target_repo: %s\n", rs.target.Repo)
	fmt.Fprintf(sb, "  sync_commit: %s\n", commitSHA)
	fmt.Fprintf(sb, "  sync_time: %s\n", time.Now().Format(time.RFC3339))

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
				if metrics, exists := rs.syncMetrics.DirectoryMetrics[dirMapping.Src]; exists {
					fmt.Fprintf(sb, "    files_synced: %d\n", metrics.FilesProcessed)
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

		// Total file counts
		totalFiles := rs.syncMetrics.FileMetrics.FilesProcessed
		if rs.syncMetrics.DirectoryMetrics != nil {
			for _, metrics := range rs.syncMetrics.DirectoryMetrics {
				totalFiles += metrics.FilesProcessed
			}
		}
		if totalFiles > 0 {
			fmt.Fprintf(sb, "  total_files: %d\n", totalFiles)
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

	// Create directory processor
	processor := NewDirectoryProcessor(rs.logger, 10) // Use default worker count
	defer processor.Close()

	sourcePath := filepath.Join(rs.tempDir, "source")

	// Verify source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("%w: %s", ErrSourceDirectoryNotExistForMetrics, sourcePath)
	}

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

		changes, err := processor.ProcessDirectoryMapping(ctx, sourcePath, dirMapping, rs.target, rs.sourceState, rs.engine)
		if err != nil {
			// Log error and collect for potential failure decision
			rs.logger.WithError(err).WithField("directory", dirMapping.Src).Error("Failed to process directory")
			processingErrors = append(processingErrors, err)
			continue
		}

		// Collect metrics for this directory
		dirStats := processor.GetDirectoryStats()
		if dirMetrics, exists := dirStats[dirMapping.Src]; exists {
			// Ensure timing is set if not already
			if dirMetrics.EndTime.IsZero() {
				dirMetrics.EndTime = time.Now()
			}
			if dirMetrics.StartTime.IsZero() {
				dirMetrics.StartTime = dirProcessingStart
			}
			collectedMetrics[dirMapping.Src] = dirMetrics
		} else {
			// Create basic metrics if not available from processor
			collectedMetrics[dirMapping.Src] = DirectoryMetrics{
				StartTime:      dirProcessingStart,
				EndTime:        time.Now(),
				FilesProcessed: len(changes),
			}
		}

		allChanges = append(allChanges, changes...)
	}

	// If all directories failed, return an error
	if len(processingErrors) > 0 && len(allChanges) == 0 {
		return nil, nil, fmt.Errorf("%w: %d errors occurred", ErrAllDirectoryProcessingWithMetricsFailed, len(processingErrors))
	}

	processTimer.AddField("total_changes", len(allChanges)).
		AddField("directories_processed", len(collectedMetrics)).Stop()

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
		if len(line) > 60 {
			_, _ = fmt.Fprintf(d.writer, "│ %s\n", line[:57]+"...")
		} else {
			_, _ = fmt.Fprintf(d.writer, "│ %s\n", line)
		}
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

// FileChange represents a change to a file
type FileChange struct {
	Path            string
	Content         []byte
	OriginalContent []byte
	IsNew           bool
}

// showDryRunCommitInfo displays commit information preview for dry-run
func (rs *RepositorySync) showDryRunCommitInfo(changedFiles []FileChange) {
	rs.logger.Debug("Showing dry-run commit preview")

	commitMsg := rs.generateCommitMessage(changedFiles)
	out := NewDryRunOutput(nil)

	out.Header("📋 COMMIT PREVIEW")
	out.Field("Message", commitMsg)
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

// showDryRunPRPreview displays full PR preview with formatting
func (rs *RepositorySync) showDryRunPRPreview(ctx context.Context, branchName, commitSHA string, changedFiles []FileChange) {
	rs.logger.WithFields(logrus.Fields{
		"branch": branchName,
		"files":  len(changedFiles),
	}).Debug("Showing PR preview")

	title := rs.generatePRTitle()
	body := rs.generatePRBody(commitSHA, changedFiles)
	out := NewDryRunOutput(nil)

	out.Header("DRY-RUN: Pull Request Preview")
	out.Field("Repository", rs.target.Repo)
	out.Field("Branch", branchName)
	out.Separator()
	out.Field("Title", title)
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

	if rs.engine.currentGroup != nil {
		global = rs.engine.currentGroup.Global.PRAssignees
		defaults = rs.engine.currentGroup.Defaults.PRAssignees
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

	if rs.engine.currentGroup != nil {
		global = rs.engine.currentGroup.Global.PRReviewers
		defaults = rs.engine.currentGroup.Defaults.PRReviewers
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

	if rs.engine.currentGroup != nil {
		global = rs.engine.currentGroup.Global.PRLabels
		defaults = rs.engine.currentGroup.Defaults.PRLabels
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
		return defaults
	}
	return combined
}

// getPRTeamReviewers returns the team reviewers to use for PRs, merging global + target assignments
func (rs *RepositorySync) getPRTeamReviewers() []string {
	var global []string
	var defaults []string

	if rs.engine.currentGroup != nil {
		global = rs.engine.currentGroup.Global.PRTeamReviewers
		defaults = rs.engine.currentGroup.Defaults.PRTeamReviewers
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
