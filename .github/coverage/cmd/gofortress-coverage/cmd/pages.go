package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mrz1836/go-broadcast/coverage/internal/pages"
	"github.com/spf13/cobra"
)

var pagesCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "pages",
	Short: "Manage GitHub Pages deployment",
	Long:  `Setup and deploy coverage reports to GitHub Pages with organized storage structure.`,
}

var pagesSetupCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "setup",
	Short: "Initialize GitHub Pages branch and structure",
	Long: `Create and configure the gh-pages branch with proper directory structure
for coverage badges, reports, and dashboard files.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		branch, _ := cmd.Flags().GetString("branch")
		force, _ := cmd.Flags().GetBool("force")
		verbose, _ := cmd.Flags().GetBool("verbose")

		ctx := context.Background()
		return setupGitHubPages(ctx, branch, force, verbose)
	},
}

var pagesDeployCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "deploy",
	Short: "Deploy coverage artifacts to GitHub Pages",
	Long: `Deploy generated coverage artifacts (badges, reports, dashboard) to the 
gh-pages branch with proper organization and cleanup.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		branch, _ := cmd.Flags().GetString("branch")
		commitSha, _ := cmd.Flags().GetString("commit")
		prNumber, _ := cmd.Flags().GetString("pr")
		inputDir, _ := cmd.Flags().GetString("input")
		message, _ := cmd.Flags().GetString("message")
		verbose, _ := cmd.Flags().GetBool("verbose")

		ctx := context.Background()
		return deployToGitHubPages(ctx, DeployOptions{
			Branch:    branch,
			CommitSha: commitSha,
			PRNumber:  prNumber,
			InputDir:  inputDir,
			Message:   message,
			Verbose:   verbose,
		})
	},
}

var pagesCleanCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "clean",
	Short: "Clean up old PR data and expired content",
	Long: `Remove old PR-specific coverage data and expired content to manage
storage usage and maintain organization.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		maxAge, _ := cmd.Flags().GetInt("max-age")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		verbose, _ := cmd.Flags().GetBool("verbose")

		ctx := context.Background()
		return cleanGitHubPages(ctx, CleanOptions{
			MaxAgeDays: maxAge,
			DryRun:     dryRun,
			Verbose:    verbose,
		})
	},
}

// DeployOptions contains options for GitHub Pages deployment
type DeployOptions struct {
	Branch    string
	CommitSha string
	PRNumber  string
	InputDir  string
	Message   string
	Verbose   bool
}

// CleanOptions contains options for GitHub Pages cleanup
type CleanOptions struct {
	MaxAgeDays int
	DryRun     bool
	Verbose    bool
}

// setupGitHubPages initializes the gh-pages branch with proper structure
func setupGitHubPages(ctx context.Context, branch string, force bool, verbose bool) error {
	if verbose {
		fmt.Printf("🚀 Setting up GitHub Pages on branch: %s\n", branch) //nolint:forbidigo // CLI output
	}

	// Get current working directory (repository root)
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting repository path: %w", err)
	}

	// Create deployer instance
	deployer := pages.NewGitHubPagesDeployer(repoPath, branch, verbose)

	// Run setup
	if err := deployer.Setup(ctx, force); err != nil {
		return fmt.Errorf("setting up GitHub Pages: %w", err)
	}

	fmt.Println("✅ GitHub Pages setup completed successfully") //nolint:forbidigo // CLI output

	if verbose {
		fmt.Printf("🌐 Your coverage dashboard will be available at: https://{owner}.github.io/{repo}/\n")  //nolint:forbidigo // CLI output
		fmt.Printf("📊 Coverage badges will be at: https://{owner}.github.io/{repo}/badges/{branch}.svg\n") //nolint:forbidigo // CLI output
	}

	return nil
}

// deployToGitHubPages deploys coverage artifacts to GitHub Pages
func deployToGitHubPages(ctx context.Context, opts DeployOptions) error {
	if opts.Verbose {
		fmt.Printf("🚀 Deploying coverage artifacts to GitHub Pages\n") //nolint:forbidigo // CLI output
		fmt.Printf("  📍 Branch: %s\n", opts.Branch)                    //nolint:forbidigo // CLI output
		fmt.Printf("  📂 Input Directory: %s\n", opts.InputDir)         //nolint:forbidigo // CLI output
		if opts.PRNumber != "" {
			fmt.Printf("  🔀 PR Number: %s\n", opts.PRNumber) //nolint:forbidigo // CLI output
		}
	}

	// Get current working directory (repository root)
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting repository path: %w", err)
	}

	// Create deployer instance
	deployer := pages.NewGitHubPagesDeployer(repoPath, "gh-pages", opts.Verbose)

	// Convert local options to pages.DeployOptions
	deployOpts := pages.DeployOptions{
		Branch:    opts.Branch,
		CommitSha: opts.CommitSha,
		PRNumber:  opts.PRNumber,
		InputDir:  opts.InputDir,
		Message:   opts.Message,
		Verbose:   opts.Verbose,
	}

	// Run deployment
	if err := deployer.Deploy(ctx, deployOpts); err != nil {
		return fmt.Errorf("deploying to GitHub Pages: %w", err)
	}

	fmt.Println("✅ Deployment completed successfully") //nolint:forbidigo // CLI output

	return nil
}

// cleanGitHubPages removes old PR data and expired content
func cleanGitHubPages(ctx context.Context, opts CleanOptions) error {
	if opts.Verbose {
		fmt.Printf("🧹 Cleaning up GitHub Pages content\n")    //nolint:forbidigo // CLI output
		fmt.Printf("  📅 Max age: %d days\n", opts.MaxAgeDays) //nolint:forbidigo // CLI output
		if opts.DryRun {
			fmt.Println("  🔍 Dry run mode - no changes will be made") //nolint:forbidigo // CLI output
		}
	}

	// Get current working directory (repository root)
	repoPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting repository path: %w", err)
	}

	// Create deployer instance
	deployer := pages.NewGitHubPagesDeployer(repoPath, "gh-pages", opts.Verbose)

	// Convert local options to pages.CleanOptions
	cleanOpts := pages.CleanOptions{
		MaxAgeDays: opts.MaxAgeDays,
		DryRun:     opts.DryRun,
		Verbose:    opts.Verbose,
	}

	// Run cleanup
	if err := deployer.Clean(ctx, cleanOpts); err != nil {
		return fmt.Errorf("cleaning GitHub Pages: %w", err)
	}

	fmt.Println("✅ Cleanup completed successfully") //nolint:forbidigo // CLI output

	return nil
}

func init() { //nolint:gochecknoinits // CLI command initialization
	// Setup command flags
	pagesSetupCmd.Flags().StringP("branch", "b", "gh-pages", "GitHub Pages branch name")
	pagesSetupCmd.Flags().Bool("force", false, "Force setup even if branch exists")
	pagesSetupCmd.Flags().BoolP("verbose", "v", false, "Verbose output")

	// Deploy command flags
	pagesDeployCmd.Flags().StringP("branch", "b", "", "Source branch name (required)")
	pagesDeployCmd.Flags().StringP("commit", "c", "", "Commit SHA (required)")
	pagesDeployCmd.Flags().StringP("pr", "p", "", "PR number (optional)")
	pagesDeployCmd.Flags().StringP("input", "i", ".", "Input directory with coverage artifacts")
	pagesDeployCmd.Flags().StringP("message", "m", "", "Custom commit message")
	pagesDeployCmd.Flags().BoolP("verbose", "v", false, "Verbose output")

	// Clean command flags
	pagesCleanCmd.Flags().IntP("max-age", "a", 30, "Maximum age in days for PR data")
	pagesCleanCmd.Flags().Bool("dry-run", false, "Show what would be cleaned without making changes")
	pagesCleanCmd.Flags().BoolP("verbose", "v", false, "Verbose output")

	// Add subcommands to pages command
	pagesCmd.AddCommand(pagesSetupCmd)
	pagesCmd.AddCommand(pagesDeployCmd)
	pagesCmd.AddCommand(pagesCleanCmd)
}
