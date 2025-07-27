package pages

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
	if err := d.organizeArtifacts(ctx, tempDir, opts); err != nil {
		result.Message = fmt.Sprintf("Failed to organize artifacts: %v", err)
		return result, err
	}

	// Update dashboard
	if err := d.updateDashboard(ctx, tempDir, opts); err != nil {
		result.Message = fmt.Sprintf("Failed to update dashboard: %v", err)
		return result, err
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
	exists, err := d.branchExists(d.Config.PagesBranch)
	if err != nil {
		return fmt.Errorf("failed to check branch existence: %w", err)
	}

	if exists && !force {
		return fmt.Errorf("branch %s already exists (use --force to recreate)", d.Config.PagesBranch)
	}

	// Create orphan gh-pages branch
	if err := d.createOrphanBranch(ctx); err != nil {
		return fmt.Errorf("failed to create orphan branch: %w", err)
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
		return fmt.Errorf("either branch or PR number must be specified")
	}

	if opts.InputDir == "" {
		return fmt.Errorf("input directory is required")
	}

	if _, err := os.Stat(opts.InputDir); os.IsNotExist(err) {
		return fmt.Errorf("input directory does not exist: %s", opts.InputDir)
	}

	return nil
}

func (d *Deployer) prepareWorkspace(ctx context.Context, opts DeploymentOptions) (string, error) {
	// Create temporary directory for deployment workspace
	tempDir, err := os.MkdirTemp("", "gofortress-deploy-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clone gh-pages branch to temp directory
	if err := d.clonePagesBranch(ctx, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone pages branch: %w", err)
	}

	return tempDir, nil
}

func (d *Deployer) organizeArtifacts(ctx context.Context, workspaceDir string, opts DeploymentOptions) error {
	var targetPath string

	if opts.PRNumber != "" {
		// PR-specific deployment
		targetPath = filepath.Join(workspaceDir, "reports", "pr", sanitizePRNumber(opts.PRNumber))
		badgeTargetPath := filepath.Join(workspaceDir, "badges", "pr", sanitizePRNumber(opts.PRNumber)+".svg")

		if err := os.MkdirAll(filepath.Dir(badgeTargetPath), 0755); err != nil {
			return fmt.Errorf("failed to create PR badge directory: %w", err)
		}

		// TODO: Copy badge file to PR-specific location
		if opts.BadgeFile != "" {
			fmt.Printf("Would copy badge from %s to %s\n", opts.BadgeFile, badgeTargetPath) //nolint:forbidigo // TODO stub
		}
	} else {
		// Branch-specific deployment
		targetPath = filepath.Join(workspaceDir, "reports", sanitizeBranchName(opts.Branch))
		badgeTargetPath := filepath.Join(workspaceDir, "badges", sanitizeBranchName(opts.Branch)+".svg")

		if err := os.MkdirAll(filepath.Dir(badgeTargetPath), 0755); err != nil {
			return fmt.Errorf("failed to create branch badge directory: %w", err)
		}

		// TODO: Copy badge file to branch-specific location
		if opts.BadgeFile != "" {
			fmt.Printf("Would copy badge from %s to %s\n", opts.BadgeFile, badgeTargetPath) //nolint:forbidigo // TODO stub
		}
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// TODO: Copy report files to target location
	for _, reportFile := range opts.ReportFiles {
		fmt.Printf("Would copy report from %s to %s\n", reportFile, targetPath) //nolint:forbidigo // TODO stub
	}

	return nil
}

func (d *Deployer) updateDashboard(ctx context.Context, workspaceDir string, opts DeploymentOptions) error {
	dashboardPath := filepath.Join(workspaceDir, "index.html")

	// TODO: Generate updated dashboard content with new coverage data
	// For now, just ensure the dashboard file exists
	if _, err := os.Stat(dashboardPath); os.IsNotExist(err) {
		// Create basic dashboard if it doesn't exist
		dashboardContent := generateBasicDashboard(opts)
		if err := os.WriteFile(dashboardPath, []byte(dashboardContent), 0644); err != nil {
			return fmt.Errorf("failed to create dashboard: %w", err)
		}
	}

	return nil
}

func (d *Deployer) commitAndPush(ctx context.Context, workspaceDir string, opts DeploymentOptions) (string, error) {
	// Change to workspace directory
	originalDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(workspaceDir); err != nil {
		return "", fmt.Errorf("failed to change to workspace directory: %w", err)
	}

	// Configure git user
	if err := d.configureGitUser(); err != nil {
		return "", fmt.Errorf("failed to configure git user: %w", err)
	}

	// Add all changes
	if err := d.runGitCommand("add", "."); err != nil {
		return "", fmt.Errorf("failed to add changes: %w", err)
	}

	// Check if there are changes to commit
	hasChanges, err := d.hasChangesToCommit()
	if err != nil {
		return "", fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		return "", fmt.Errorf("no changes to commit")
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

	if err := d.runGitCommand("commit", "-m", commitMessage); err != nil {
		return "", fmt.Errorf("failed to commit changes: %w", err)
	}

	// Get commit SHA
	commitSha, err := d.getCommitSha()
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	// Push changes
	if err := d.runGitCommand("push", "origin", d.Config.PagesBranch); err != nil {
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
	filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func (d *Deployer) cleanupWorkspace(tempDir string) {
	os.RemoveAll(tempDir)
}

// Git helper methods

func (d *Deployer) branchExists(branchName string) (bool, error) {
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branchName)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func (d *Deployer) createOrphanBranch(ctx context.Context) error {
	// TODO: Implement orphan branch creation
	fmt.Printf("Would create orphan branch: %s\n", d.Config.PagesBranch)
	return nil
}

func (d *Deployer) clonePagesBranch(ctx context.Context, targetDir string) error {
	// TODO: Implement gh-pages branch cloning
	fmt.Printf("Would clone %s branch to %s\n", d.Config.PagesBranch, targetDir)
	return nil
}

func (d *Deployer) configureGitUser() error {
	if err := d.runGitCommand("config", "user.name", d.Config.CommitAuthor); err != nil {
		return err
	}
	return d.runGitCommand("config", "user.email", d.Config.CommitEmail)
}

func (d *Deployer) runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if d.Config.GitHubToken != "" {
		// TODO: Configure authentication with GitHub token
	}
	return cmd.Run()
}

func (d *Deployer) hasChangesToCommit() (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func (d *Deployer) getCommitSha() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (d *Deployer) createInitialDashboard(ctx context.Context, structure *StorageStructure) error {
	// TODO: Create initial dashboard using templates
	fmt.Printf("Would create initial dashboard at %s\n", structure.DashboardPath)
	return nil
}

func (d *Deployer) initialCommitAndPush(ctx context.Context) error {
	// TODO: Make initial commit and push
	fmt.Println("Would make initial commit and push") //nolint:forbidigo // TODO stub
	return nil
}

// generateBasicDashboard creates a basic dashboard HTML content
func generateBasicDashboard(opts DeploymentOptions) string {
	var target string
	if opts.PRNumber != "" {
		target = fmt.Sprintf("PR #%s", opts.PRNumber)
	} else {
		target = fmt.Sprintf("branch '%s'", opts.Branch)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Coverage Dashboard | GoFortress</title>
    <style>
        body { font-family: system-ui, sans-serif; margin: 0; padding: 2rem; background: #0d1117; color: #c9d1d9; }
        h1 { color: #58a6ff; margin-bottom: 0.5rem; }
        .metric { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 1.5rem; margin: 1rem 0; }
        .status { background: #1f2937; border-radius: 8px; padding: 1rem; text-align: center; margin-top: 2rem; }
    </style>
</head>
<body>
    <h1>🏰 Coverage Dashboard</h1>
    <div class="metric">
        <h3>📊 Latest Update</h3>
        <p>Coverage data updated for %s</p>
    </div>
    <div class="status">
        <p>🔄 Dashboard updated: %s</p>
    </div>
</body>
</html>`, target, time.Now().Format("2006-01-02 15:04:05 UTC"))
}
