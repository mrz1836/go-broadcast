package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/sirupsen/logrus"
)

// discoveryService implements the Discoverer interface
type discoveryService struct {
	gh     gh.Client
	logger *logrus.Logger
}

// NewDiscoverer creates a new state discoverer
func NewDiscoverer(ghClient gh.Client, logger *logrus.Logger) Discoverer {
	return &discoveryService{
		gh:     ghClient,
		logger: logger,
	}
}

// DiscoverState discovers the complete sync state by examining GitHub
func (d *discoveryService) DiscoverState(ctx context.Context, cfg *config.Config) (*State, error) {
	d.logger.Info("Discovering sync state from GitHub")

	state := &State{
		Source: SourceState{
			Repo:        cfg.Source.Repo,
			Branch:      cfg.Source.Branch,
			LastChecked: time.Now(),
		},
		Targets: make(map[string]*TargetState),
	}

	// Get source repository latest commit by getting branch details
	sourceBranch, err := d.gh.GetBranch(ctx, cfg.Source.Repo, cfg.Source.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get source branch: %w", err)
	}
	if sourceBranch != nil {
		state.Source.LatestCommit = sourceBranch.Commit.SHA
	}

	// Discover state for each target repository
	for _, target := range cfg.Targets {
		d.logger.WithField("repo", target.Repo).Debug("Discovering target state")

		targetState, err := d.DiscoverTargetState(ctx, target.Repo)
		if err != nil {
			return nil, fmt.Errorf("failed to discover state for %s: %w", target.Repo, err)
		}

		// Determine sync status based on source and target state
		targetState.Status = d.determineSyncStatus(state.Source, targetState)

		state.Targets[target.Repo] = targetState
	}

	return state, nil
}

// DiscoverTargetState discovers the state of a specific target repository
func (d *discoveryService) DiscoverTargetState(ctx context.Context, repo string) (*TargetState, error) {
	targetState := &TargetState{
		Repo:         repo,
		SyncBranches: []SyncBranch{},
		OpenPRs:      []gh.PR{},
		Status:       StatusUnknown,
	}

	// List all branches to find sync branches
	branches, err := d.gh.ListBranches(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// Find and parse sync branches
	for _, branch := range branches {
		if strings.HasPrefix(branch.Name, "sync/template-") {
			metadata, parseErr := d.ParseBranchName(branch.Name)
			if parseErr != nil {
				d.logger.WithError(parseErr).WithField("branch", branch.Name).Warn("Failed to parse sync branch")
				continue
			}
			if metadata != nil {
				targetState.SyncBranches = append(targetState.SyncBranches, SyncBranch{
					Name:     branch.Name,
					Metadata: metadata,
				})
			}
		}
	}

	// Find the latest sync commit from branches
	var latestSyncTime *time.Time

	for _, syncBranch := range targetState.SyncBranches {
		if syncBranch.Metadata != nil {
			if latestSyncTime == nil || syncBranch.Metadata.Timestamp.After(*latestSyncTime) {
				latestSyncTime = &syncBranch.Metadata.Timestamp
				targetState.LastSyncCommit = syncBranch.Metadata.CommitSHA
				targetState.LastSyncTime = latestSyncTime
			}
		}
	}

	// List open PRs
	prs, err := d.gh.ListPRs(ctx, repo, "open")
	if err != nil {
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	// Find sync-related PRs
	for _, pr := range prs {
		// Check if PR is from a sync branch
		if strings.HasPrefix(pr.Head.Ref, "sync/template-") {
			targetState.OpenPRs = append(targetState.OpenPRs, pr)
		}
	}

	return targetState, nil
}

// ParseBranchName parses a branch name to extract sync metadata
func (d *discoveryService) ParseBranchName(name string) (*BranchMetadata, error) {
	// This will be implemented in branch.go
	// For now, return a placeholder
	return parseSyncBranchName(name)
}

// determineSyncStatus determines the sync status based on source and target state
func (d *discoveryService) determineSyncStatus(source SourceState, target *TargetState) SyncStatus {
	// No sync history
	if target.LastSyncCommit == "" {
		return StatusBehind
	}

	// Has open PR - sync is pending
	if len(target.OpenPRs) > 0 {
		return StatusPending
	}

	// Check if target is up to date with source
	if target.LastSyncCommit == source.LatestCommit {
		return StatusUpToDate
	}

	// Target is behind source
	return StatusBehind
}
