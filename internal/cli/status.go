package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Package-level variable for CLI flag
var (
	jsonOutput bool
)

// initStatus initializes status command flags
func initStatus() {
	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output status in JSON format")
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync state for all targets",
	Long: `Display the current synchronization state for all target repositories.

Shows information about:
  • Open pull requests
  • Sync branches
  • Last sync timestamp and commit
  • Out-of-date targets`,
	Example: `  # Show status for all targets
  go-broadcast status --config sync.yaml

  # Output in JSON format
  go-broadcast status --json`,
	Aliases: []string{"st"},
	RunE:    runStatus,
}

// SyncStatus represents the sync state for display
type SyncStatus struct {
	Source  SourceStatus   `json:"source"`
	Targets []TargetStatus `json:"targets"`
}

// SourceStatus represents source repository status
type SourceStatus struct {
	Repository   string `json:"repository"`
	Branch       string `json:"branch"`
	LatestCommit string `json:"latest_commit"`
}

// TargetStatus represents a target repository status
type TargetStatus struct {
	Repository  string           `json:"repository"`
	State       string           `json:"state"` // "synced", "outdated", "pending", "error"
	SyncBranch  *string          `json:"sync_branch,omitempty"`
	PullRequest *PullRequestInfo `json:"pull_request,omitempty"`
	LastSync    *SyncInfo        `json:"last_sync,omitempty"`
	Error       *string          `json:"error,omitempty"`
}

// PullRequestInfo contains PR details
type PullRequestInfo struct {
	Number int    `json:"number"`
	State  string `json:"state"`
	URL    string `json:"url"`
	Title  string `json:"title"`
}

// SyncInfo contains last sync details
type SyncInfo struct {
	Timestamp string `json:"timestamp"`
	Commit    string `json:"commit"`
}

func runStatus(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	_ = logrus.WithField("command", "status")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize state discovery with real implementations
	status, err := getRealStatus(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to discover status: %w", err)
	}

	// Output status
	if jsonOutput {
		return outputJSON(status)
	}

	return outputTextStatus(status)
}

func getRealStatus(ctx context.Context, cfg *config.Config) (*SyncStatus, error) {
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

	// Initialize GitHub client with enhanced error handling
	ghClient, err := gh.NewClient(ctx, logger, logConfig)
	if err != nil {
		// Provide specific error messages for common issues
		switch {
		case errors.Is(err, gh.ErrGHNotFound):
			return nil, fmt.Errorf("%w: Please install GitHub CLI: https://cli.github.com/", gh.ErrGHNotFound)
		case errors.Is(err, gh.ErrNotAuthenticated):
			return nil, fmt.Errorf("%w: Please run: gh auth login", gh.ErrNotAuthenticated)
		default:
			return nil, fmt.Errorf("failed to initialize GitHub client: %w", err)
		}
	}

	// Initialize state discoverer
	discoverer := state.NewDiscoverer(ghClient, logger, logConfig)

	// Discover current state with enhanced error handling
	currentState, err := discoverer.DiscoverState(ctx, cfg)
	if err != nil {
		// Provide specific error messages for common GitHub API issues
		switch {
		case errors.Is(err, gh.ErrRateLimited):
			return nil, fmt.Errorf("%w: Please try again later", gh.ErrRateLimited)
		case errors.Is(err, gh.ErrBranchNotFound):
			return nil, fmt.Errorf("%w: Please check your configuration", gh.ErrBranchNotFound)
		default:
			return nil, fmt.Errorf("failed to discover sync state: %w", err)
		}
	}

	// Convert to CLI status format
	return convertStateToStatus(currentState), nil
}

// convertStateToStatus converts internal state to CLI status format
func convertStateToStatus(s *state.State) *SyncStatus {
	status := &SyncStatus{
		Source: SourceStatus{
			Repository:   s.Source.Repo,
			Branch:       s.Source.Branch,
			LatestCommit: s.Source.LatestCommit,
		},
		Targets: make([]TargetStatus, 0, len(s.Targets)),
	}

	// Convert each target state
	for _, targetState := range s.Targets {
		targetStatus := TargetStatus{
			Repository: targetState.Repo,
			State:      convertSyncStatus(targetState.Status),
		}

		// Add last sync information if available
		if targetState.LastSyncCommit != "" && targetState.LastSyncTime != nil {
			targetStatus.LastSync = &SyncInfo{
				Timestamp: targetState.LastSyncTime.Format(time.RFC3339),
				Commit:    targetState.LastSyncCommit,
			}
		}

		// Add sync branch if available (use the most recent one)
		if len(targetState.SyncBranches) > 0 {
			// Find the most recent sync branch
			var mostRecent *state.SyncBranch
			for i := range targetState.SyncBranches {
				branch := &targetState.SyncBranches[i]
				if branch.Metadata != nil {
					if mostRecent == nil || branch.Metadata.Timestamp.After(mostRecent.Metadata.Timestamp) {
						mostRecent = branch
					}
				}
			}
			if mostRecent != nil {
				targetStatus.SyncBranch = &mostRecent.Name
			}
		}

		// Add pull request information if available
		if len(targetState.OpenPRs) > 0 {
			// Use the first open PR (most recent)
			pr := targetState.OpenPRs[0]
			targetStatus.PullRequest = &PullRequestInfo{
				Number: pr.Number,
				State:  strings.ToLower(pr.State),
				URL:    fmt.Sprintf("https://github.com/%s/pull/%d", targetState.Repo, pr.Number),
				Title:  pr.Title,
			}
		}

		status.Targets = append(status.Targets, targetStatus)
	}

	return status
}

// convertSyncStatus converts internal sync status to CLI display format
func convertSyncStatus(s state.SyncStatus) string {
	switch s {
	case state.StatusUpToDate:
		return "synced"
	case state.StatusBehind:
		return "outdated"
	case state.StatusPending:
		return "pending"
	case state.StatusConflict:
		return "error"
	case state.StatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

func outputJSON(status *SyncStatus) error {
	encoder := json.NewEncoder(output.Stdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}

func outputTextStatus(status *SyncStatus) error {
	output.Info(fmt.Sprintf("Source: %s (branch: %s)", status.Source.Repository, status.Source.Branch))
	output.Info(fmt.Sprintf("Latest commit: %s", status.Source.LatestCommit))
	output.Info("")

	output.Info(fmt.Sprintf("Targets (%d):", len(status.Targets)))

	for _, target := range status.Targets {
		// Status icon
		var icon string

		switch target.State {
		case "synced":
			icon = "✓"
		case "outdated":
			icon = "⚠"
		case "pending":
			icon = "⏳"
		case "error":
			icon = "✗"
		default:
			icon = "?"
		}

		output.Info(fmt.Sprintf("  %s %s [%s]", icon, target.Repository, target.State))

		if target.PullRequest != nil {
			output.Info(fmt.Sprintf("    PR #%d: %s (%s)",
				target.PullRequest.Number,
				target.PullRequest.Title,
				target.PullRequest.State))
			output.Info(fmt.Sprintf("    URL: %s", target.PullRequest.URL))
		}

		if target.LastSync != nil {
			output.Info(fmt.Sprintf("    Last sync: %s (commit: %s)",
				target.LastSync.Timestamp,
				target.LastSync.Commit[:7]))
		}

		if target.SyncBranch != nil {
			output.Info(fmt.Sprintf("    Branch: %s", *target.SyncBranch))
		}

		if target.Error != nil {
			output.Error(fmt.Sprintf("    Error: %s", *target.Error))
		}

		output.Info("")
	}

	// Summary
	synced := 0
	outdated := 0
	pending := 0
	errors := 0

	for _, t := range status.Targets {
		switch t.State {
		case "synced":
			synced++
		case "outdated":
			outdated++
		case "pending":
			pending++
		case "error":
			errors++
		}
	}

	output.Info("Summary:")
	output.Success(fmt.Sprintf("  Synced: %d", synced))

	if outdated > 0 {
		output.Warn(fmt.Sprintf("  Outdated: %d", outdated))
	}

	if pending > 0 {
		output.Info(fmt.Sprintf("  Pending: %d", pending))
	}

	if errors > 0 {
		output.Error(fmt.Sprintf("  Errors: %d", errors))
	}

	return nil
}
