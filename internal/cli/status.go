package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
)

//nolint:gochecknoglobals // Package-level variables for CLI flags
var (
	statusFlagsMu     sync.RWMutex
	jsonOutput        bool
	statusGroupFilter []string
	statusSkipGroups  []string
)

// setJSONOutput sets the JSON output flag (thread-safe, for testing)
func setJSONOutput(v bool) {
	statusFlagsMu.Lock()
	defer statusFlagsMu.Unlock()
	jsonOutput = v
}

// getJSONOutput returns the JSON output flag (thread-safe)
func getJSONOutput() bool {
	statusFlagsMu.RLock()
	defer statusFlagsMu.RUnlock()
	return jsonOutput
}

// getStatusGroupFilter returns a copy of the status group filter (thread-safe)
func getStatusGroupFilter() []string {
	statusFlagsMu.RLock()
	defer statusFlagsMu.RUnlock()
	return append([]string(nil), statusGroupFilter...)
}

// getStatusSkipGroups returns a copy of the status skip groups (thread-safe)
func getStatusSkipGroups() []string {
	statusFlagsMu.RLock()
	defer statusFlagsMu.RUnlock()
	return append([]string(nil), statusSkipGroups...)
}

// initStatus initializes status command flags
func initStatus() {
	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output status in JSON format")
	statusCmd.Flags().StringSliceVar(&statusGroupFilter, "groups", nil, "Only show status for these groups (by name or ID)")
	statusCmd.Flags().StringSliceVar(&statusSkipGroups, "skip-groups", nil, "Skip these groups (by name or ID)")
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
  • Out-of-date targets

For configurations with groups, you can filter which groups to display using
--groups or --skip-groups flags.`,
	Example: `  # Show status for all targets
  go-broadcast status --config sync.yaml

  # Output in JSON format
  go-broadcast status --json

  # Show status for specific groups only
  go-broadcast status --groups "core"
  go-broadcast status --groups "core,security"

  # Show all groups except specific ones
  go-broadcast status --skip-groups "experimental"`,
	Aliases: []string{"st"},
	RunE:    runStatus,
}

// SyncStatus represents the sync state for display
type SyncStatus struct {
	Source  SourceStatus   `json:"source,omitempty"`  // Non-group based status
	Targets []TargetStatus `json:"targets,omitempty"` // Non-group based status
	Groups  []GroupStatus  `json:"groups,omitempty"`  // Group-based status
}

// GroupStatus represents the status of a sync group
type GroupStatus struct {
	Name      string         `json:"name"`
	ID        string         `json:"id"`
	Priority  int            `json:"priority"`
	Enabled   bool           `json:"enabled"`
	DependsOn []string       `json:"depends_on,omitempty"`
	State     string         `json:"state"` // "ready", "synced", "pending", "error", "disabled"
	Source    SourceStatus   `json:"source"`
	Targets   []TargetStatus `json:"targets"`
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

	// Apply group filtering if specified
	cfg = FilterConfigByGroups(cfg, getStatusGroupFilter(), getStatusSkipGroups())

	// Initialize state discovery with real implementations
	status, err := getRealStatus(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to discover status: %w", err)
	}

	// Output status
	if getJSONOutput() {
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

	// Initialize GitHub client with comprehensive error handling
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

	// Discover current state with comprehensive error handling
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
	return convertStateToStatus(currentState, cfg), nil
}

// convertStateToStatus converts internal state to CLI status format
func convertStateToStatus(s *state.State, cfg *config.Config) *SyncStatus {
	// Check if we have groups in the configuration
	if cfg != nil && len(cfg.Groups) > 0 {
		return convertStateToGroupStatus(s, cfg)
	}

	// Convert to display format
	status := &SyncStatus{
		Source: SourceStatus{
			Repository:   s.Source.Repo,
			Branch:       s.Source.Branch,
			LatestCommit: s.Source.LatestCommit,
		},
		Targets: make([]TargetStatus, 0, len(s.Targets)),
	}

	// Get sorted list of target repositories for deterministic order
	repos := make([]string, 0, len(s.Targets))
	for repo := range s.Targets {
		repos = append(repos, repo)
	}
	sort.Strings(repos)

	// Convert each target state in sorted order
	for _, repo := range repos {
		targetState := s.Targets[repo]
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

// convertStateToGroupStatus converts state to group-based status format
func convertStateToGroupStatus(s *state.State, cfg *config.Config) *SyncStatus {
	status := &SyncStatus{
		Groups: make([]GroupStatus, 0),
	}

	// Get groups from configuration
	groups := cfg.Groups

	// Convert each group to status
	for _, group := range groups {
		groupStatus := GroupStatus{
			Name:      group.Name,
			ID:        group.ID,
			Priority:  group.Priority,
			Enabled:   group.Enabled == nil || *group.Enabled,
			DependsOn: group.DependsOn,
			Source: SourceStatus{
				Repository:   group.Source.Repo,
				Branch:       group.Source.Branch,
				LatestCommit: s.Source.LatestCommit,
			},
			Targets: make([]TargetStatus, 0),
		}

		// Determine group state based on targets
		allSynced := true
		hasError := false
		hasPending := false

		// Process targets for this group
		for _, target := range group.Targets {
			if targetState, exists := s.Targets[target.Repo]; exists {
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

				// Add sync branch if available
				if len(targetState.SyncBranches) > 0 {
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
					pr := targetState.OpenPRs[0]
					targetStatus.PullRequest = &PullRequestInfo{
						Number: pr.Number,
						State:  strings.ToLower(pr.State),
						URL:    fmt.Sprintf("https://github.com/%s/pull/%d", targetState.Repo, pr.Number),
						Title:  pr.Title,
					}
				}

				// Add error if status is conflict/error
				if targetState.Status == state.StatusConflict {
					errMsg := "Repository has conflicts that need manual resolution"
					targetStatus.Error = &errMsg
				}

				groupStatus.Targets = append(groupStatus.Targets, targetStatus)

				// Update group state flags
				if targetStatus.State != "synced" {
					allSynced = false
				}
				if targetStatus.State == "error" {
					hasError = true
				}
				if targetStatus.State == "pending" {
					hasPending = true
				}
			}
		}

		// Set group state
		if !groupStatus.Enabled {
			groupStatus.State = "disabled"
		} else if hasError {
			groupStatus.State = "error"
		} else if hasPending {
			groupStatus.State = "pending"
		} else if allSynced {
			groupStatus.State = "synced"
		} else {
			groupStatus.State = "ready"
		}

		status.Groups = append(status.Groups, groupStatus)
	}

	// Sort groups by priority
	sort.Slice(status.Groups, func(i, j int) bool {
		return status.Groups[i].Priority < status.Groups[j].Priority
	})

	return status
}

func outputJSON(status *SyncStatus) error {
	encoder := json.NewEncoder(output.Stdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(status)
}

func outputTextStatus(status *SyncStatus) error {
	// Check if we have groups or non-group format
	if len(status.Groups) > 0 {
		return outputGroupTextStatus(status)
	}

	// Non-group format display
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

// outputGroupTextStatus displays group-based status in text format
func outputGroupTextStatus(status *SyncStatus) error {
	output.Info("=== Sync Status (Group-Based Configuration) ===")
	output.Info("")

	for _, group := range status.Groups {
		// Group header with status icon
		var groupIcon string
		switch group.State {
		case "synced":
			groupIcon = "✓"
		case "pending":
			groupIcon = "⏳"
		case "error":
			groupIcon = "✗"
		case "disabled":
			groupIcon = "⊘"
		default:
			groupIcon = "•"
		}

		output.Info(fmt.Sprintf("%s Group: %s (%s) [Priority: %d]",
			groupIcon, group.Name, group.ID, group.Priority))

		// Group metadata
		output.Info(fmt.Sprintf("  Status: %s", group.State))
		if group.State == "disabled" {
			output.Info("  (Group is disabled)")
		}

		if len(group.DependsOn) > 0 {
			output.Info(fmt.Sprintf("  Dependencies: %s", strings.Join(group.DependsOn, ", ")))
		}

		output.Info(fmt.Sprintf("  Source: %s (branch: %s)",
			group.Source.Repository, group.Source.Branch))

		// Group targets
		if len(group.Targets) > 0 {
			output.Info(fmt.Sprintf("  Targets (%d):", len(group.Targets)))

			for _, target := range group.Targets {
				// Target status icon
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

				output.Info(fmt.Sprintf("    %s %s [%s]", icon, target.Repository, target.State))

				if target.PullRequest != nil {
					output.Info(fmt.Sprintf("      PR #%d: %s (%s)",
						target.PullRequest.Number,
						target.PullRequest.Title,
						target.PullRequest.State))
				}

				if target.LastSync != nil && len(target.LastSync.Commit) >= 7 {
					output.Info(fmt.Sprintf("      Last sync: %s (commit: %s)",
						target.LastSync.Timestamp,
						target.LastSync.Commit[:7]))
				}

				if target.Error != nil {
					output.Error(fmt.Sprintf("      Error: %s", *target.Error))
				}
			}
		} else {
			output.Info("  Targets: (none)")
		}

		output.Info("")
	}

	// Overall summary
	totalGroups := len(status.Groups)
	enabledGroups := 0
	syncedGroups := 0
	pendingGroups := 0
	errorGroups := 0

	for _, g := range status.Groups {
		if g.State != "disabled" {
			enabledGroups++
		}
		switch g.State {
		case "synced":
			syncedGroups++
		case "pending":
			pendingGroups++
		case "error":
			errorGroups++
		}
	}

	output.Info("=== Summary ===")
	output.Info(fmt.Sprintf("Total Groups: %d (%d enabled, %d disabled)",
		totalGroups, enabledGroups, totalGroups-enabledGroups))

	if enabledGroups > 0 {
		summary := []string{}
		if syncedGroups > 0 {
			summary = append(summary, fmt.Sprintf("%d synced", syncedGroups))
		}
		if pendingGroups > 0 {
			summary = append(summary, fmt.Sprintf("%d pending", pendingGroups))
		}
		if errorGroups > 0 {
			summary = append(summary, fmt.Sprintf("%d with errors", errorGroups))
		}
		if len(summary) > 0 {
			output.Info(fmt.Sprintf("Status: %s", strings.Join(summary, ", ")))
		}
	}

	return nil
}
