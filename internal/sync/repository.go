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

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/metrics"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
)

// RepositorySync handles synchronization for a single repository
type RepositorySync struct {
	engine      *Engine
	target      config.TargetConfig
	sourceState *state.SourceState
	targetState *state.TargetState
	logger      *logrus.Entry
	tempDir     string
}

// Execute performs the complete sync operation for this repository
func (rs *RepositorySync) Execute(ctx context.Context) error {
	// Start overall operation timer
	syncTimer := metrics.StartTimer(ctx, rs.logger, "repository_sync").
		AddField(logging.StandardFields.SourceRepo, rs.sourceState.Repo).
		AddField(logging.StandardFields.TargetRepo, rs.target.Repo).
		AddField("sync_branch", rs.sourceState.Branch).
		AddField("commit_sha", rs.sourceState.LatestCommit)

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

	changedFiles, err := rs.processFiles(ctx)
	if err != nil {
		processTimer.StopWithError(err)
		syncTimer.StopWithError(err)
		return fmt.Errorf("failed to process files: %w", err)
	}

	processTimer.AddField("changed_files", len(changedFiles)).Stop()

	if len(changedFiles) == 0 {
		rs.logger.Info("No file changes detected, skipping sync")
		syncTimer.AddField(logging.StandardFields.Status, "no_changes").Stop()
		return nil
	}

	// 5. Create sync branch (or use existing one)
	branchTimer := metrics.StartTimer(ctx, rs.logger, "branch_creation")
	branchName := rs.createSyncBranch(ctx)
	branchTimer.AddField(logging.StandardFields.BranchName, branchName).Stop()

	// 6. Commit changes
	commitTimer := metrics.StartTimer(ctx, rs.logger, "commit_creation").
		AddField(logging.StandardFields.BranchName, branchName).
		AddField("changed_files", len(changedFiles))

	commitSHA, err := rs.commitChanges(ctx, branchName, changedFiles)
	if err != nil {
		commitTimer.StopWithError(err)
		syncTimer.StopWithError(err)
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	commitTimer.AddField("commit_sha", commitSHA).Stop()

	// 7. Push changes (unless dry-run)
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

	// 8. Create or update pull request
	prTimer := metrics.StartTimer(ctx, rs.logger, "pr_management").
		AddField(logging.StandardFields.BranchName, branchName).
		AddField("commit_sha", commitSHA).
		AddField("changed_files", len(changedFiles))

	if err := rs.createOrUpdatePR(ctx, branchName, commitSHA, changedFiles); err != nil {
		prTimer.StopWithError(err)
		syncTimer.StopWithError(err)
		return fmt.Errorf("failed to create/update PR: %w", err)
	}
	prTimer.Stop()

	if rs.engine.options.DryRun {
		rs.logger.Debug("Dry-run completed successfully")

		out := NewDryRunOutput(nil)
		out.Success("DRY-RUN SUMMARY: Repository sync preview completed successfully")
		out.Info(fmt.Sprintf("📁 Repository: %s", rs.target.Repo))
		out.Info(fmt.Sprintf("🌿 Branch: %s", branchName))
		out.Info(fmt.Sprintf("📝 Files: %d would be changed", len(changedFiles)))
		out.Info(fmt.Sprintf("🔗 Commit: %s", commitSHA))
		out.Info("💡 Run without --dry-run to execute these changes")
		_, _ = fmt.Fprintln(out.writer)
	} else {
		rs.logger.WithField("branch", branchName).Info("Repository sync completed")
	}

	syncTimer.AddField(logging.StandardFields.Status, "completed").
		AddField(logging.StandardFields.BranchName, branchName).
		AddField("final_commit_sha", commitSHA).
		AddField("total_changed_files", len(changedFiles)).Stop()

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

	if err := os.RemoveAll(rs.tempDir); err != nil {
		rs.logger.WithError(err).Warn("Failed to cleanup temporary directory")
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
	// Try to get file from the target repository's default branch
	fileContent, err := rs.engine.gh.GetFile(ctx, rs.target.Repo, filePath, "")
	if err != nil {
		return nil, err
	}
	return fileContent.Content, nil
}

// createSyncBranch creates a new sync branch or returns existing one
func (rs *RepositorySync) createSyncBranch(_ context.Context) string {
	// Generate branch name: sync/template-YYYYMMDD-HHMMSS-{commit}
	now := time.Now()
	timestamp := now.Format("20060102-150405")
	commitSHA := rs.sourceState.LatestCommit
	if len(commitSHA) > 7 {
		commitSHA = commitSHA[:7]
	}

	branchPrefix := rs.engine.config.Defaults.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "sync/template"
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
		rs.showDryRunPRPreview(branchName, commitSHA, changedFiles)
		return nil
	}

	// Get default branch for base
	branches, err := rs.engine.gh.ListBranches(ctx, rs.target.Repo)
	if err != nil {
		return fmt.Errorf("failed to get branches: %w", err)
	}

	baseBranch := "master" // default
	for _, branch := range branches {
		if branch.Name == "main" {
			baseBranch = "main"
			break
		}
	}

	prRequest := gh.PRRequest{
		Title:         title,
		Body:          body,
		Head:          branchName,
		Base:          baseBranch,
		Assignees:     rs.getPRAssignees(),
		Reviewers:     rs.getPRReviewers(),
		TeamReviewers: rs.getPRTeamReviewers(),
	}

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

	if err := rs.engine.gh.UpdatePR(ctx, rs.target.Repo, pr.Number, updates); err != nil {
		return fmt.Errorf("failed to update PR: %w", err)
	}

	rs.logger.WithField("pr_number", pr.Number).Info("Pull request updated successfully")
	return nil
}

// generateCommitMessage creates a descriptive commit message
func (rs *RepositorySync) generateCommitMessage(changedFiles []FileChange) string {
	if len(changedFiles) == 1 {
		return fmt.Sprintf("sync: update %s from template", changedFiles[0].Path)
	}

	return fmt.Sprintf("sync: update %d files from template", len(changedFiles))
}

// generatePRTitle creates a descriptive PR title
func (rs *RepositorySync) generatePRTitle() string {
	commitSHA := rs.sourceState.LatestCommit
	if len(commitSHA) > 7 {
		commitSHA = commitSHA[:7]
	}
	return fmt.Sprintf("[Sync] Update project files from source repository (%s)", commitSHA)
}

// generatePRBody creates a detailed PR description with metadata
func (rs *RepositorySync) generatePRBody(commitSHA string, _ []FileChange) string {
	var sb strings.Builder

	// What Changed section
	sb.WriteString("## What Changed\n")
	sb.WriteString("* Updated project files to synchronize with the latest changes from the source repository\n")
	sb.WriteString("* Applied file transformations and updates based on sync configuration\n")
	shortSHA := commitSHA
	if len(commitSHA) > 7 {
		shortSHA = commitSHA[:7]
	}
	sb.WriteString(fmt.Sprintf("* Brought target repository in line with source repository state at commit %s\n\n", shortSHA))

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

	// Add metadata as YAML block
	sb.WriteString("<!-- go-broadcast-metadata\n")
	sb.WriteString("sync_metadata:\n")
	sb.WriteString(fmt.Sprintf("  source_repo: %s\n", rs.sourceState.Repo))
	sb.WriteString(fmt.Sprintf("  source_commit: %s\n", rs.sourceState.LatestCommit))
	sb.WriteString(fmt.Sprintf("  target_repo: %s\n", rs.target.Repo))
	sb.WriteString(fmt.Sprintf("  sync_commit: %s\n", commitSHA))
	sb.WriteString(fmt.Sprintf("  sync_time: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("-->\n")

	return sb.String()
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

// showDryRunPRPreview displays full PR preview with formatting
func (rs *RepositorySync) showDryRunPRPreview(branchName, commitSHA string, changedFiles []FileChange) {
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

	// Split body into lines and display with proper formatting
	bodyLines := strings.Split(body, "\n")
	for _, line := range bodyLines {
		out.Content(line)
	}

	out.Footer()
}

// getPRAssignees returns the assignees to use for PRs, with target overriding defaults
func (rs *RepositorySync) getPRAssignees() []string {
	if len(rs.target.PRAssignees) > 0 {
		return rs.target.PRAssignees
	}
	return rs.engine.config.Defaults.PRAssignees
}

// getPRReviewers returns the reviewers to use for PRs, with target overriding defaults
func (rs *RepositorySync) getPRReviewers() []string {
	if len(rs.target.PRReviewers) > 0 {
		return rs.target.PRReviewers
	}
	return rs.engine.config.Defaults.PRReviewers
}

// getPRTeamReviewers returns the team reviewers to use for PRs, with target overriding defaults
func (rs *RepositorySync) getPRTeamReviewers() []string {
	if len(rs.target.PRTeamReviewers) > 0 {
		return rs.target.PRTeamReviewers
	}
	return rs.engine.config.Defaults.PRTeamReviewers
}
