package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
)

// initStatus initializes status command flags
func initStatus() {
	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output status in JSON format")
}

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

	// TODO: Initialize state discovery with real implementations
	// For now, we'll show mock status
	status, err := getMockStatus(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	// Output status
	if jsonOutput {
		return outputJSON(status)
	}

	return outputTextStatus(status)
}

func getMockStatus(_ context.Context, cfg *config.Config) (*SyncStatus, error) {
	// Mock implementation - will be replaced with real state discovery
	status := &SyncStatus{
		Source: SourceStatus{
			Repository:   cfg.Source.Repo,
			Branch:       cfg.Source.Branch,
			LatestCommit: "abc123def456",
		},
		Targets: []TargetStatus{},
	}

	// Add mock status for each target
	for _, target := range cfg.Targets {
		targetStatus := TargetStatus{
			Repository: target.Repo,
			State:      "synced",
			LastSync: &SyncInfo{
				Timestamp: "2024-01-15T12:00:00Z",
				Commit:    "abc123def456",
			},
		}

		// Mock different states for demo
		switch target.Repo {
		case cfg.Targets[0].Repo:
			targetStatus.State = "synced"
		default:
			targetStatus.State = "outdated"
			branch := "sync/template-20240115-120000-abc123"
			targetStatus.SyncBranch = &branch
			targetStatus.PullRequest = &PullRequestInfo{
				Number: 42,
				State:  "open",
				URL:    fmt.Sprintf("https://github.com/%s/pull/42", target.Repo),
				Title:  "Sync template updates",
			}
		}

		status.Targets = append(status.Targets, targetStatus)
	}

	return status, nil
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
