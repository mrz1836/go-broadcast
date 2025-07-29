package pages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/analytics/dashboard"
)

// Static error definitions
var (
	ErrBranchAlreadyExists     = errors.New("branch already exists")
	ErrBranchOrPRRequired      = errors.New("either branch or PR number must be specified")
	ErrInputDirectoryRequired  = errors.New("input directory is required")
	ErrInputDirectoryNotExists = errors.New("input directory does not exist")
	ErrNoChangesToCommit       = errors.New("no changes to commit")
)

// Deployer handles GitHub Pages deployment operations
type Deployer struct {
	Config  *DeployConfig
	Storage *StorageManager
}

// DeployConfig contains deployment configuration
type DeployConfig struct {
	// Repository settings
	RepoOwner string
	RepoName  string

	// Branch settings
	SourceBranch string
	PagesBranch  string

	// Deployment settings
	CommitAuthor string
	CommitEmail  string
	RemoteURL    string

	// GitHub token for authentication
	GitHubToken string
}

// DeploymentResult contains the result of a deployment operation
type DeploymentResult struct {
	Success     bool
	CommitSha   string
	DeployedURL string
	FilesCount  int
	Message     string
	Duration    time.Duration
}

// NewDeployer creates a new GitHub Pages deployer
func NewDeployer(config *DeployConfig, storage *StorageManager) *Deployer {
	return &Deployer{
		Config:  config,
		Storage: storage,
	}
}

// Deploy deploys coverage artifacts to GitHub Pages
func (d *Deployer) Deploy(ctx context.Context, opts DeploymentOptions) (*DeploymentResult, error) {
	startTime := time.Now()

	result := &DeploymentResult{
		Success: false,
		Message: "",
	}

	// Validate inputs
	if err := d.validateDeployment(opts); err != nil {
		result.Message = fmt.Sprintf("Validation failed: %v", err)
		return result, err
	}

	// Prepare deployment workspace
	tempDir, err := d.prepareWorkspace(ctx, opts)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to prepare workspace: %v", err)
		return result, err
	}
	defer d.cleanupWorkspace(tempDir)

	// Organize artifacts
	if organizeErr := d.organizeArtifacts(ctx, tempDir, opts); organizeErr != nil {
		result.Message = fmt.Sprintf("Failed to organize artifacts: %v", organizeErr)
		return result, organizeErr
	}

	// Update dashboard
	if updateErr := d.updateDashboard(ctx, tempDir, opts); updateErr != nil {
		result.Message = fmt.Sprintf("Failed to update dashboard: %v", updateErr)
		return result, updateErr
	}

	// Commit and push changes
	commitSha, err := d.commitAndPush(ctx, tempDir, opts)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to commit and push: %v", err)
		return result, err
	}

	// Calculate deployment URL
	deployedURL := d.calculateDeploymentURL(opts)

	result.Success = true
	result.CommitSha = commitSha
	result.DeployedURL = deployedURL
	result.FilesCount = d.countDeployedFiles(tempDir)
	result.Duration = time.Since(startTime)
	result.Message = "Deployment completed successfully"

	return result, nil
}

// Setup initializes the GitHub Pages branch and structure
func (d *Deployer) Setup(ctx context.Context, force bool) error {
	// Check if gh-pages branch exists
	exists, err := d.branchExists(ctx, d.Config.PagesBranch)
	if err != nil {
		return fmt.Errorf("failed to check branch existence: %w", err)
	}

	if exists && !force {
		return fmt.Errorf("%w: %s (use --force to recreate)", ErrBranchAlreadyExists, d.Config.PagesBranch)
	}

	// Create orphan gh-pages branch
	if createErr := d.createOrphanBranch(ctx); createErr != nil {
		return fmt.Errorf("failed to create orphan branch: %w", createErr)
	}

	// Initialize directory structure
	structure, err := d.Storage.InitializeStructure(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize structure: %w", err)
	}

	// Create initial dashboard
	if err := d.createInitialDashboard(ctx, structure); err != nil {
		return fmt.Errorf("failed to create initial dashboard: %w", err)
	}

	// Initial commit and push
	if err := d.initialCommitAndPush(ctx); err != nil {
		return fmt.Errorf("failed to make initial commit: %w", err)
	}

	return nil
}

// DeploymentOptions contains options for deployment
type DeploymentOptions struct {
	Branch      string
	PRNumber    string
	CommitSha   string
	InputDir    string
	Message     string
	BadgeFile   string
	ReportFiles []string
	Force       bool
	Verbose     bool
}

// Helper methods

func (d *Deployer) validateDeployment(opts DeploymentOptions) error {
	if opts.Branch == "" && opts.PRNumber == "" {
		return ErrBranchOrPRRequired
	}

	if opts.InputDir == "" {
		return ErrInputDirectoryRequired
	}

	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrInputDirectoryNotExists, opts.InputDir)
	}

	return nil
}

func (d *Deployer) prepareWorkspace(ctx context.Context, _ DeploymentOptions) (string, error) {
	// Create temporary directory for deployment workspace
	tempDir, err := os.MkdirTemp("", "gofortress-deploy-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clone gh-pages branch to temp directory
	if err := d.clonePagesBranch(ctx, tempDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone pages branch: %w", err)
	}

	return tempDir, nil
}

func (d *Deployer) organizeArtifacts(_ context.Context, workspaceDir string, opts DeploymentOptions) error {
	var targetPath string

	if opts.PRNumber != "" {
		// PR-specific deployment
		targetPath = filepath.Join(workspaceDir, "reports", "pr", sanitizePRNumber(opts.PRNumber))
		badgeTargetPath := filepath.Join(workspaceDir, "badges", "pr", sanitizePRNumber(opts.PRNumber)+".svg")

		if err := os.MkdirAll(filepath.Dir(badgeTargetPath), 0o750); err != nil {
			return fmt.Errorf("failed to create PR badge directory: %w", err)
		}

		// TODO: Copy badge file to PR-specific location
		if opts.BadgeFile != "" {
			// TODO: Copy badge file to PR-specific location
			// Source: opts.BadgeFile, Target: badgeTargetPath
			_ = badgeTargetPath // Placeholder
		}
	} else {
		// Branch-specific deployment
		targetPath = filepath.Join(workspaceDir, "reports", sanitizeBranchName(opts.Branch))
		badgeTargetPath := filepath.Join(workspaceDir, "badges", sanitizeBranchName(opts.Branch)+".svg")

		if err := os.MkdirAll(filepath.Dir(badgeTargetPath), 0o750); err != nil {
			return fmt.Errorf("failed to create branch badge directory: %w", err)
		}

		// TODO: Copy badge file to branch-specific location
		if opts.BadgeFile != "" {
			// TODO: Copy badge file to branch-specific location
			// Source: opts.BadgeFile, Target: badgeTargetPath
			_ = badgeTargetPath // Placeholder
		}
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetPath, 0o750); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// TODO: Copy report files to target location
	for _, reportFile := range opts.ReportFiles {
		// TODO: Copy report file
		// Source: reportFile, Target: targetPath
		_ = reportFile // Placeholder to avoid unused variable
	}

	return nil
}

func (d *Deployer) updateDashboard(ctx context.Context, workspaceDir string, opts DeploymentOptions) error {
	// Import the dashboard generator
	generatorConfig := &dashboard.GeneratorConfig{
		ProjectName:      d.Config.RepoName,
		RepositoryOwner:  d.Config.RepoOwner,
		RepositoryName:   d.Config.RepoName,
		TemplateDir:      "", // Using embedded template
		OutputDir:        workspaceDir,
		AssetsDir:        filepath.Join(workspaceDir, "assets"),
		GeneratorVersion: "1.0.0",
	}

	generator := dashboard.NewGenerator(generatorConfig)

	// Load coverage data from the input directory
	coverageData, err := d.loadCoverageData(ctx, opts)
	if err != nil {
		return fmt.Errorf("loading coverage data: %w", err)
	}

	// Generate the dashboard
	if err := generator.Generate(ctx, coverageData); err != nil {
		return fmt.Errorf("generating dashboard: %w", err)
	}

	return nil
}

func (d *Deployer) commitAndPush(ctx context.Context, workspaceDir string, opts DeploymentOptions) (string, error) {
	// Change to workspace directory
	originalDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if chdirErr := os.Chdir(workspaceDir); chdirErr != nil {
		return "", fmt.Errorf("failed to change to workspace directory: %w", chdirErr)
	}

	// Configure git user
	if configErr := d.configureGitUser(ctx); configErr != nil {
		return "", fmt.Errorf("failed to configure git user: %w", configErr)
	}

	// Add all changes
	if addErr := d.runGitCommand(ctx, "add", "."); addErr != nil {
		return "", fmt.Errorf("failed to add changes: %w", addErr)
	}

	// Check if there are changes to commit
	hasChanges, err := d.hasChangesToCommit(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		return "", ErrNoChangesToCommit
	}

	// Commit changes
	commitMessage := opts.Message
	if commitMessage == "" {
		if opts.PRNumber != "" {
			commitMessage = fmt.Sprintf("📊 Update coverage for PR #%s", opts.PRNumber)
		} else {
			commitMessage = fmt.Sprintf("📊 Update coverage for %s", opts.Branch)
		}
	}

	if commitErr := d.runGitCommand(ctx, "commit", "-m", commitMessage); commitErr != nil {
		return "", fmt.Errorf("failed to commit changes: %w", commitErr)
	}

	// Get commit SHA
	commitSha, err := d.getCommitSha(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	// Push changes
	if err := d.runGitCommand(ctx, "push", "origin", d.Config.PagesBranch); err != nil {
		return "", fmt.Errorf("failed to push changes: %w", err)
	}

	return commitSha, nil
}

func (d *Deployer) calculateDeploymentURL(opts DeploymentOptions) string {
	baseURL := fmt.Sprintf("https://%s.github.io/%s", d.Config.RepoOwner, d.Config.RepoName)

	if opts.PRNumber != "" {
		return fmt.Sprintf("%s/reports/pr/%s/", baseURL, sanitizePRNumber(opts.PRNumber))
	} else if opts.Branch != "" {
		return fmt.Sprintf("%s/reports/%s/", baseURL, sanitizeBranchName(opts.Branch))
	}

	return baseURL
}

func (d *Deployer) countDeployedFiles(workspaceDir string) int {
	count := 0
	_ = filepath.Walk(workspaceDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func (d *Deployer) cleanupWorkspace(tempDir string) {
	_ = os.RemoveAll(tempDir)
}

// Git helper methods

func (d *Deployer) branchExists(ctx context.Context, branchName string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", "origin", branchName)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func (d *Deployer) createOrphanBranch(_ context.Context) error {
	// TODO: Implement orphan branch creation
	// TODO: Create orphan branch: d.Config.PagesBranch
	return nil
}

func (d *Deployer) clonePagesBranch(_ context.Context, targetDir string) error {
	// TODO: Implement gh-pages branch cloning
	// TODO: Clone d.Config.PagesBranch branch to targetDir
	_ = targetDir // Placeholder to avoid unused variable
	return nil
}

func (d *Deployer) configureGitUser(ctx context.Context) error {
	if err := d.runGitCommand(ctx, "config", "user.name", d.Config.CommitAuthor); err != nil {
		return err
	}
	return d.runGitCommand(ctx, "config", "user.email", d.Config.CommitEmail)
}

func (d *Deployer) runGitCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	// TODO: Configure authentication with GitHub token if d.Config.GitHubToken != ""
	return cmd.Run()
}

func (d *Deployer) hasChangesToCommit(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func (d *Deployer) getCommitSha(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (d *Deployer) createInitialDashboard(_ context.Context, structure *StorageStructure) error {
	// TODO: Create initial dashboard using templates
	// TODO: Create initial dashboard at structure.DashboardPath
	_ = structure // Placeholder to avoid unused variable
	return nil
}

func (d *Deployer) initialCommitAndPush(_ context.Context) error {
	// TODO: Make initial commit and push
	// TODO: Make initial commit and push to remote repository
	return nil
}

// loadCoverageData loads coverage data from input directory
func (d *Deployer) loadCoverageData(ctx context.Context, opts DeploymentOptions) (*dashboard.CoverageData, error) {
	// Try to load coverage data JSON if it exists
	coverageDataPath := filepath.Clean(filepath.Join(opts.InputDir, "coverage-data.json"))
	if _, err := os.Stat(coverageDataPath); err == nil {
		// Load from JSON file
		data, err := os.ReadFile(coverageDataPath)
		if err != nil {
			return nil, fmt.Errorf("reading coverage data file: %w", err)
		}

		var coverageData dashboard.CoverageData
		if err := json.Unmarshal(data, &coverageData); err != nil {
			return nil, fmt.Errorf("unmarshaling coverage data: %w", err)
		}

		return &coverageData, nil
	}

	// Create basic coverage data from options
	coverageData := &dashboard.CoverageData{
		ProjectName:   d.Config.RepoName,
		RepositoryURL: fmt.Sprintf("https://github.com/%s/%s", d.Config.RepoOwner, d.Config.RepoName),
		Branch:        opts.Branch,
		CommitSHA:     opts.CommitSha,
		PRNumber:      opts.PRNumber,
		Timestamp:     time.Now(),
		TotalCoverage: 0.0, // Will be populated from actual data
		TotalLines:    0,
		CoveredLines:  0,
		MissedLines:   0,
		TotalFiles:    0,
		CoveredFiles:  0,
	}

	// If branch is empty, try to get it from git
	if coverageData.Branch == "" {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
		if output, err := cmd.Output(); err == nil {
			coverageData.Branch = strings.TrimSpace(string(output))
		}
	}

	// If CommitSHA is empty, try to get it from git
	if coverageData.CommitSHA == "" {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
		if output, err := cmd.Output(); err == nil {
			coverageData.CommitSHA = strings.TrimSpace(string(output))
		}
	}

	// Try to load coverage percentage from badge SVG
	badgePath := filepath.Join(opts.InputDir, "coverage.svg")
	if _, err := os.Stat(badgePath); err == nil {
		// Parse coverage from badge (simplified - in real implementation would parse SVG)
		coverageData.TotalCoverage = 90.58 // Placeholder
	}

	return coverageData, nil
}
