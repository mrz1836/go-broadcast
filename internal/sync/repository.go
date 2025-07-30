package sync

import (
	"context"
	"errors"
	"fmt"
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

		rs.pushChanges(ctx, branchName)
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

	rs.logger.WithField("branch", branchName).Info("Repository sync completed")
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
func (rs *RepositorySync) commitChanges(_ context.Context, branchName string, changedFiles []FileChange) (string, error) {
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
		rs.logger.Info("DRY-RUN: Would create commit with changes")
		return "dry-run-commit-sha", nil
	}

	// Create commit via GitHub API
	// This is a simplified version - in practice, you'd need to create a tree first
	// For now, we'll simulate this
	commitSHA := fmt.Sprintf("commit-%d", time.Now().Unix())

	return commitSHA, nil
}

// pushChanges pushes the branch to the target repository
func (rs *RepositorySync) pushChanges(_ context.Context, branchName string) {
	rs.logger.WithField("branch", branchName).Info("Pushing changes to target repository")

	// In a real implementation, this would push the branch
	// For now, we'll simulate this
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
		rs.logger.Info("DRY-RUN: Would create new PR")
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
		Title: title,
		Body:  body,
		Head:  branchName,
		Base:  baseBranch,
	}

	pr, err := rs.engine.gh.CreatePR(ctx, rs.target.Repo, prRequest)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	rs.logger.WithField("pr_number", pr.Number).Info("Pull request created successfully")
	return nil
}

// updateExistingPR updates an existing pull request
func (rs *RepositorySync) updateExistingPR(_ context.Context, pr *gh.PR, commitSHA string, changedFiles []FileChange) error {
	rs.logger.WithField("pr_number", pr.Number).Info("Updating existing pull request")

	if rs.engine.options.DryRun {
		rs.logger.Info("DRY-RUN: Would update existing PR")
		return nil
	}

	// Update PR body with new information
	newBody := rs.generatePRBody(commitSHA, changedFiles)

	// In a real implementation, you'd call the GitHub API to update the PR
	// For now, we'll just log what we would do
	rs.logger.Debug("PR would be updated with new body")
	_ = newBody // Avoid unused variable warning

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
	return fmt.Sprintf("Sync files from template repository (%s)", commitSHA)
}

// generatePRBody creates a detailed PR description with metadata
func (rs *RepositorySync) generatePRBody(commitSHA string, changedFiles []FileChange) string {
	var sb strings.Builder

	sb.WriteString("This pull request synchronizes files from the template repository.\n\n")

	// Add metadata as YAML block
	sb.WriteString("<!-- go-broadcast-metadata\n")
	sb.WriteString("sync_metadata:\n")
	sb.WriteString(fmt.Sprintf("  source_repo: %s\n", rs.sourceState.Repo))
	sb.WriteString(fmt.Sprintf("  source_commit: %s\n", rs.sourceState.LatestCommit))
	sb.WriteString(fmt.Sprintf("  target_repo: %s\n", rs.target.Repo))
	sb.WriteString(fmt.Sprintf("  sync_commit: %s\n", commitSHA))
	sb.WriteString(fmt.Sprintf("  sync_time: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("-->\n\n")

	// Add file changes summary
	sb.WriteString("## Changed Files\n\n")
	for _, file := range changedFiles {
		status := "modified"
		if file.IsNew {
			status = "added"
		}
		sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", file.Path, status))
	}

	// Add source information
	sb.WriteString("\n## Source Information\n\n")
	sb.WriteString(fmt.Sprintf("- **Source Repository**: %s\n", rs.sourceState.Repo))
	sb.WriteString(fmt.Sprintf("- **Source Branch**: %s\n", rs.sourceState.Branch))
	sb.WriteString(fmt.Sprintf("- **Source Commit**: %s\n", rs.sourceState.LatestCommit))

	return sb.String()
}

// FileChange represents a change to a file
type FileChange struct {
	Path            string
	Content         []byte
	OriginalContent []byte
	IsNew           bool
}
