package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// ErrTargetNotFound is returned when a target repository is not found in the configuration
var ErrTargetNotFound = errors.New("target repository not found in configuration")

//nolint:gochecknoglobals // Package-level variables for CLI flags
var (
	cancelKeepBranches bool
	cancelComment      string
)

// initCancel initializes cancel command flags
func initCancel() {
	cancelCmd.Flags().BoolVar(&cancelKeepBranches, "keep-branches", false, "Close PRs but keep sync branches")
	cancelCmd.Flags().StringVar(&cancelComment, "comment", "", "Custom comment to add when closing PRs")
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var cancelCmd = &cobra.Command{
	Use:   "cancel [targets...]",
	Short: "Cancel active sync operations",
	Long: `Cancel active sync operations by closing open pull requests and optionally deleting sync branches.

This command finds all open sync pull requests for the specified targets (or all targets if none specified)
and closes them with a descriptive comment. By default, it also deletes the associated sync branches
to clean up the repositories.

Use this when you need to cancel a sync operation due to issues and want to re-sync later with
updated files or configuration.`,
	Example: `  # Cancel all active syncs
  go-broadcast cancel --config sync.yaml

  # Cancel syncs for specific repositories
  go-broadcast cancel org/repo1 org/repo2

  # Preview what would be canceled (dry run)
  go-broadcast cancel --dry-run --config sync.yaml

  # Close PRs but keep sync branches
  go-broadcast cancel --keep-branches --config sync.yaml

  # Add custom comment when closing PRs
  go-broadcast cancel --comment "Canceling due to configuration update" --config sync.yaml`,
	Aliases: []string{"c"},
	RunE:    runCancel,
}

// CancelResult represents the result of a cancel operation
type CancelResult struct {
	Repository    string `json:"repository"`
	PRNumber      *int   `json:"pr_number,omitempty"`
	PRClosed      bool   `json:"pr_closed"`
	BranchName    string `json:"branch_name,omitempty"`
	BranchDeleted bool   `json:"branch_deleted"`
	Error         string `json:"error,omitempty"`
}

// CancelSummary represents the overall cancel operation results
type CancelSummary struct {
	TotalTargets    int            `json:"total_targets"`
	PRsClosed       int            `json:"prs_closed"`
	BranchesDeleted int            `json:"branches_deleted"`
	Errors          int            `json:"errors"`
	Results         []CancelResult `json:"results"`
	DryRun          bool           `json:"dry_run"`
}

func runCancel(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	log := logrus.WithField("command", "cancel")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Filter targets if specified
	targetRepos := args
	if len(targetRepos) > 0 {
		log.WithField("targets", targetRepos).Info("Canceling syncs for specific targets")
	} else {
		log.Info("Canceling syncs for all configured targets")
	}

	// Perform cancel operation
	summary, err := performCancel(ctx, cfg, targetRepos)
	if err != nil {
		return fmt.Errorf("cancel operation failed: %w", err)
	}

	// Output results
	if globalFlags.DryRun {
		return outputCancelPreview(summary)
	}

	return outputCancelResults(summary)
}

func performCancel(ctx context.Context, cfg *config.Config, targetRepos []string) (*CancelSummary, error) {
	if cfg == nil {
		panic("config cannot be nil")
	}

	// Create logger for GitHub operations
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create logging config with minimal debug settings
	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			State: false,
			API:   false,
		},
		Verbose: 0,
	}

	// Initialize GitHub client
	ghClient, err := gh.NewClient(ctx, logger, logConfig)
	if err != nil {
		switch {
		case errors.Is(err, gh.ErrGHNotFound):
			return nil, fmt.Errorf("%w: Please install GitHub CLI: https://cli.github.com/", gh.ErrGHNotFound)
		case errors.Is(err, gh.ErrNotAuthenticated):
			return nil, fmt.Errorf("%w: Please run: gh auth login", gh.ErrNotAuthenticated)
		default:
			return nil, fmt.Errorf("failed to initialize GitHub client: %w", err)
		}
	}

	return performCancelWithClient(ctx, cfg, targetRepos, ghClient, logger, logConfig)
}

// performCancelWithClient performs cancel operation with injected dependencies for testing
func performCancelWithClient(ctx context.Context, cfg *config.Config, targetRepos []string, ghClient gh.Client, logger *logrus.Logger, logConfig *logging.LogConfig) (*CancelSummary, error) {
	if cfg == nil {
		panic("config cannot be nil")
	}

	// Initialize state discoverer
	discoverer := state.NewDiscoverer(ghClient, logger, logConfig)

	return performCancelWithDiscoverer(ctx, cfg, targetRepos, ghClient, discoverer)
}

// performCancelWithDiscoverer performs cancel operation with injected state discoverer for advanced testing
func performCancelWithDiscoverer(ctx context.Context, cfg *config.Config, targetRepos []string, ghClient gh.Client, discoverer state.Discoverer) (*CancelSummary, error) {
	if cfg == nil {
		panic("config cannot be nil")
	}

	// Discover current state
	currentState, err := discoverer.DiscoverState(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to discover sync state: %w", err)
	}

	// Filter targets if specified
	filteredTargets, err := filterTargets(currentState, targetRepos)
	if err != nil {
		return nil, fmt.Errorf("failed to filter targets: %w", err)
	}

	// Prepare summary
	summary := &CancelSummary{
		TotalTargets: len(filteredTargets),
		Results:      make([]CancelResult, 0, len(filteredTargets)),
		DryRun:       globalFlags.DryRun,
	}

	// Process each target
	for _, targetState := range filteredTargets {
		result := processCancelTarget(ctx, ghClient, targetState)
		summary.Results = append(summary.Results, result)

		// Update counters
		if result.PRClosed {
			summary.PRsClosed++
		}
		if result.BranchDeleted {
			summary.BranchesDeleted++
		}
		if result.Error != "" {
			summary.Errors++
		}
	}

	// Sort results by repository name for consistent output
	sort.Slice(summary.Results, func(i, j int) bool {
		return summary.Results[i].Repository < summary.Results[j].Repository
	})

	return summary, nil
}

func filterTargets(s *state.State, targetRepos []string) ([]*state.TargetState, error) {
	if len(targetRepos) == 0 {
		// Return all targets
		targets := make([]*state.TargetState, 0, len(s.Targets))
		for _, target := range s.Targets {
			// Only include targets with active syncs (open PRs or sync branches)
			if len(target.OpenPRs) > 0 || len(target.SyncBranches) > 0 {
				targets = append(targets, target)
			}
		}
		return targets, nil
	}

	// Filter to specified repositories
	targets := make([]*state.TargetState, 0, len(targetRepos))
	for _, repo := range targetRepos {
		target, exists := s.Targets[repo]
		if !exists {
			return nil, fmt.Errorf("%w: %q", ErrTargetNotFound, repo)
		}

		// Only include if there are active syncs
		if len(target.OpenPRs) > 0 || len(target.SyncBranches) > 0 {
			targets = append(targets, target)
		}
	}

	return targets, nil
}

func processCancelTarget(ctx context.Context, ghClient gh.Client, target *state.TargetState) CancelResult {
	result := CancelResult{
		Repository: target.Repo,
	}

	// Process open PRs - only handle the first one (most recent)
	if len(target.OpenPRs) > 0 {
		pr := target.OpenPRs[0]

		if globalFlags.DryRun {
			result.PRNumber = &pr.Number
			result.PRClosed = true // Would be closed
		} else {
			// Generate comment
			comment := generateCancelComment()
			if cancelComment != "" {
				comment = cancelComment
			}

			// Close the PR
			if err := ghClient.ClosePR(ctx, target.Repo, pr.Number, comment); err != nil {
				result.Error = fmt.Sprintf("failed to close PR #%d: %v", pr.Number, err)
				return result
			}

			result.PRNumber = &pr.Number
			result.PRClosed = true
		}
	}

	// Process sync branches
	if len(target.SyncBranches) > 0 {
		// Find the most recent sync branch
		var mostRecent *state.SyncBranch
		for i := range target.SyncBranches {
			branch := &target.SyncBranches[i]
			if branch.Metadata != nil {
				if mostRecent == nil || branch.Metadata.Timestamp.After(mostRecent.Metadata.Timestamp) {
					mostRecent = branch
				}
			}
		}

		if mostRecent != nil {
			result.BranchName = mostRecent.Name

			// Only delete if not keeping branches
			if !cancelKeepBranches {
				if globalFlags.DryRun {
					result.BranchDeleted = true // Would be deleted
				} else {
					// Delete the branch
					if err := ghClient.DeleteBranch(ctx, target.Repo, mostRecent.Name); err != nil {
						// Don't fail the entire operation if branch deletion fails
						if result.Error == "" {
							result.Error = fmt.Sprintf("failed to delete branch %s: %v", mostRecent.Name, err)
						} else {
							result.Error += fmt.Sprintf("; failed to delete branch %s: %v", mostRecent.Name, err)
						}
					} else {
						result.BranchDeleted = true
					}
				}
			}
		}
	}

	return result
}

func generateCancelComment() string {
	return fmt.Sprintf(`🚫 **Sync Operation Canceled**

This sync operation has been canceled using go-broadcast cancel command.

- **Canceled at**: %s
- **Reason**: Manual cancellation via CLI

You can safely ignore this PR. If you need to re-sync, run the sync command again with your updated configuration.

---
*This comment was automatically generated by go-broadcast.*`, time.Now().Format(time.RFC3339))
}

func outputCancelPreview(summary *CancelSummary) error {
	output.Warn("DRY-RUN MODE: No changes will be made")
	output.Info("")

	if summary.TotalTargets == 0 {
		output.Info("No active sync operations found to cancel")
		return nil
	}

	output.Info(fmt.Sprintf("Would cancel sync operations for %d target(s):", summary.TotalTargets))
	output.Info("")

	for _, result := range summary.Results {
		output.Info(fmt.Sprintf("📦 %s", result.Repository))

		if result.PRNumber != nil {
			output.Info(fmt.Sprintf("  ✓ Would close PR #%d", *result.PRNumber))
		}

		if result.BranchName != "" && !cancelKeepBranches {
			output.Info(fmt.Sprintf("  ✓ Would delete branch: %s", result.BranchName))
		} else if result.BranchName != "" {
			output.Info(fmt.Sprintf("  ⏸ Would keep branch: %s", result.BranchName))
		}

		if result.Error != "" {
			output.Error(fmt.Sprintf("  ✗ Error: %s", result.Error))
		}

		output.Info("")
	}

	output.Info("Summary (would):")
	output.Info(fmt.Sprintf("  PRs to close: %d", summary.PRsClosed))
	if !cancelKeepBranches {
		output.Info(fmt.Sprintf("  Branches to delete: %d", summary.BranchesDeleted))
	}

	return nil
}

func outputCancelResults(summary *CancelSummary) error {
	if summary.TotalTargets == 0 {
		output.Info("No active sync operations found to cancel")
		return nil
	}

	// Output JSON if requested
	if jsonOutput {
		encoder := json.NewEncoder(output.Stdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}

	// Text output
	output.Info(fmt.Sprintf("Canceled sync operations for %d target(s):", summary.TotalTargets))
	output.Info("")

	for _, result := range summary.Results {
		output.Info(fmt.Sprintf("📦 %s", result.Repository))

		if result.PRClosed && result.PRNumber != nil {
			output.Success(fmt.Sprintf("  ✓ Closed PR #%d", *result.PRNumber))
		} else if result.PRNumber != nil {
			output.Error(fmt.Sprintf("  ✗ Failed to close PR #%d", *result.PRNumber))
		}

		if result.BranchDeleted {
			output.Success(fmt.Sprintf("  ✓ Deleted branch: %s", result.BranchName))
		} else if result.BranchName != "" && !cancelKeepBranches {
			output.Error(fmt.Sprintf("  ✗ Failed to delete branch: %s", result.BranchName))
		} else if result.BranchName != "" {
			output.Info(fmt.Sprintf("  ⏸ Kept branch: %s", result.BranchName))
		}

		if result.Error != "" {
			output.Error(fmt.Sprintf("  Error: %s", result.Error))
		}

		output.Info("")
	}

	// Summary
	output.Info("Summary:")
	output.Success(fmt.Sprintf("  PRs closed: %d", summary.PRsClosed))

	if !cancelKeepBranches {
		output.Success(fmt.Sprintf("  Branches deleted: %d", summary.BranchesDeleted))
	}

	if summary.Errors > 0 {
		output.Error(fmt.Sprintf("  Errors: %d", summary.Errors))
	}

	if summary.Errors == 0 {
		output.Success("Cancel operation completed successfully!")
	} else {
		output.Warn("Cancel operation completed with some errors")
	}

	return nil
}
