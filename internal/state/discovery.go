package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/sirupsen/logrus"
)

// discoveryService implements the Discoverer interface
type discoveryService struct {
	gh        gh.Client
	logger    *logrus.Logger
	logConfig *logging.LogConfig
}

// NewDiscoverer creates a new state discoverer.
//
// Parameters:
// - ghClient: GitHub client for API operations
// - logger: Logger instance for general logging
// - logConfig: Configuration for debug logging and verbose settings
//
// Returns:
// - Discoverer interface implementation for state discovery operations
func NewDiscoverer(ghClient gh.Client, logger *logrus.Logger, logConfig *logging.LogConfig) Discoverer {
	return &discoveryService{
		gh:        ghClient,
		logger:    logger,
		logConfig: logConfig,
	}
}

// DiscoverState discovers the complete sync state by examining GitHub with comprehensive debug logging support.
//
// This method provides detailed visibility into state discovery when debug logging is enabled,
// including source repository analysis, target repository scanning, timing metrics, and state correlation.
//
// Parameters:
// - ctx: Context for cancellation and timeout control
// - cfg: Configuration containing source and target repository information
//
// Returns:
// - Complete sync state information
// - Error if discovery fails
//
// Side Effects:
// - Logs detailed discovery progress when --debug-state flag is enabled
// - Records discovery timing and repository analysis metrics
func (d *discoveryService) DiscoverState(ctx context.Context, cfg *config.Config) (*State, error) {
	logger := logging.WithStandardFields(d.logger, d.logConfig, logging.ComponentNames.State)
	start := time.Now()

	// Debug logging when --debug-state flag is enabled
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.Operation:   logging.OperationTypes.StateDiscover,
			logging.StandardFields.SourceRepo:  cfg.Source.Repo,
			logging.StandardFields.BranchName:  cfg.Source.Branch,
			logging.StandardFields.TargetCount: len(cfg.Targets),
		}).Debug("Starting sync state discovery")
	} else {
		d.logger.Info("Discovering sync state from GitHub")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("state discovery canceled: %w", ctx.Err())
	default:
	}

	state := &State{
		Source: SourceState{
			Repo:        cfg.Source.Repo,
			Branch:      cfg.Source.Branch,
			LastChecked: time.Now(),
		},
		Targets: make(map[string]*TargetState),
	}

	// Get source repository latest commit
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.RepoName:   cfg.Source.Repo,
			logging.StandardFields.BranchName: cfg.Source.Branch,
		}).Debug("Discovering source repository state")
	}

	sourceStart := time.Now()
	sourceBranch, err := d.gh.GetBranch(ctx, cfg.Source.Repo, cfg.Source.Branch)
	sourceDuration := time.Since(sourceStart)

	if err != nil {
		if d.logConfig != nil && d.logConfig.Debug.State {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.RepoName:   cfg.Source.Repo,
				logging.StandardFields.BranchName: cfg.Source.Branch,
				logging.StandardFields.Error:      err.Error(),
				logging.StandardFields.DurationMs: sourceDuration.Milliseconds(),
				logging.StandardFields.Status:     "failed",
			}).Error("Failed to get source branch information")
		}
		return nil, fmt.Errorf("failed to get source branch: %w", err)
	}

	if sourceBranch != nil {
		state.Source.LatestCommit = sourceBranch.Commit.SHA

		if d.logConfig != nil && d.logConfig.Debug.State {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.RepoName:   cfg.Source.Repo,
				logging.StandardFields.BranchName: cfg.Source.Branch,
				logging.StandardFields.CommitSHA:  state.Source.LatestCommit,
				logging.StandardFields.DurationMs: sourceDuration.Milliseconds(),
				logging.StandardFields.Status:     "discovered",
			}).Debug("Source repository state discovered")
		}
	} else {
		if d.logConfig != nil && d.logConfig.Debug.State {
			logger.WithField("repo", cfg.Source.Repo).Warn("Source branch information not available")
		}
	}

	// Discover state for each target repository
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithField(logging.StandardFields.TargetCount, len(cfg.Targets)).Debug("Starting target repository discovery")
	}

	targetStates := make(map[string]*TargetState)
	for i, target := range cfg.Targets {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("target discovery canceled: %w", ctx.Err())
		default:
		}

		targetLogger := logger
		if d.logConfig != nil && d.logConfig.Debug.State {
			targetLogger = logger.WithFields(logrus.Fields{
				"target_index":                    i,
				logging.StandardFields.TargetRepo: target.Repo,
			})
			targetLogger.Trace("Discovering target repository state")
		} else {
			d.logger.WithField("repo", target.Repo).Debug("Discovering target state")
		}

		targetStart := time.Now()
		targetState, err := d.DiscoverTargetState(ctx, target.Repo, cfg.Defaults.BranchPrefix)
		targetDuration := time.Since(targetStart)

		if err != nil {
			if d.logConfig != nil && d.logConfig.Debug.State {
				targetLogger.WithFields(logrus.Fields{
					logging.StandardFields.Error:      err.Error(),
					logging.StandardFields.DurationMs: targetDuration.Milliseconds(),
					logging.StandardFields.Status:     "failed",
				}).Error("Failed to discover target repository state")
			}
			return nil, fmt.Errorf("failed to discover state for %s: %w", target.Repo, err)
		}

		// Determine sync status based on source and target state
		targetState.Status = d.determineSyncStatus(state.Source, targetState)

		if d.logConfig != nil && d.logConfig.Debug.State {
			targetLogger.WithFields(logrus.Fields{
				"sync_branches":                   len(targetState.SyncBranches),
				"open_prs":                        len(targetState.OpenPRs),
				"last_sync_commit":                targetState.LastSyncCommit,
				logging.StandardFields.SyncStatus: string(targetState.Status),
				logging.StandardFields.DurationMs: targetDuration.Milliseconds(),
				logging.StandardFields.Status:     "discovered",
			}).Debug("Target repository state discovered")
		}

		targetStates[target.Repo] = targetState
	}

	state.Targets = targetStates

	// Log successful discovery completion
	duration := time.Since(start)
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.DurationMs: duration.Milliseconds(),
			"targets_discovered":              len(state.Targets),
			logging.StandardFields.CommitSHA:  state.Source.LatestCommit,
			logging.StandardFields.Status:     "completed",
		}).Debug("Sync state discovery completed successfully")
	}

	return state, nil
}

// DiscoverTargetState discovers the state of a specific target repository with comprehensive debug logging support.
//
// This method provides detailed visibility into target repository analysis when debug logging is enabled,
// including branch discovery, sync branch parsing, PR detection, and timing metrics.
//
// Parameters:
// - ctx: Context for cancellation and timeout control
// - repo: Target repository name to analyze
// - branchPrefix: The branch prefix to use for sync branch detection
//
// Returns:
// - Target repository state information
// - Error if discovery fails
//
// Side Effects:
// - Logs detailed target analysis progress when --debug-state flag is enabled
// - Records API call timing and branch analysis metrics
func (d *discoveryService) DiscoverTargetState(ctx context.Context, repo, branchPrefix string) (*TargetState, error) {
	logger := logging.WithStandardFields(d.logger, d.logConfig, "target-discovery")
	logger = logger.WithField(logging.StandardFields.TargetRepo, repo)
	start := time.Now()

	// Debug logging when --debug-state flag is enabled
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.Debug("Starting target repository state discovery")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("target discovery canceled: %w", ctx.Err())
	default:
	}

	targetState := &TargetState{
		Repo:         repo,
		SyncBranches: []SyncBranch{},
		OpenPRs:      []gh.PR{},
		Status:       StatusUnknown,
	}

	// List all branches to find sync branches
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.Debug("Listing branches to find sync branches")
	}

	branchStart := time.Now()
	branches, err := d.gh.ListBranches(ctx, repo)
	branchDuration := time.Since(branchStart)

	if err != nil {
		if d.logConfig != nil && d.logConfig.Debug.State {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.Error:      err.Error(),
				logging.StandardFields.DurationMs: branchDuration.Milliseconds(),
				logging.StandardFields.Status:     "failed",
			}).Error("Failed to list branches")
		}
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithFields(logrus.Fields{
			"branch_count":                    len(branches),
			logging.StandardFields.DurationMs: branchDuration.Milliseconds(),
			logging.StandardFields.Status:     "success",
		}).Debug("Successfully listed branches")
	}

	// Find and parse sync branches
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.Debug("Analyzing branches for sync patterns")
	}

	syncBranchCount := 0
	syncBranchPrefix := branchPrefix + "-"
	for _, branch := range branches {
		if strings.HasPrefix(branch.Name, syncBranchPrefix) {
			syncBranchCount++

			if d.logConfig != nil && d.logConfig.Debug.State {
				logger.WithField(logging.StandardFields.BranchName, branch.Name).Trace("Found potential sync branch")
			}

			metadata, parseErr := d.ParseBranchNameWithPrefix(branch.Name, branchPrefix)
			if parseErr != nil {
				if d.logConfig != nil && d.logConfig.Debug.State {
					logger.WithFields(logrus.Fields{
						logging.StandardFields.BranchName: branch.Name,
						logging.StandardFields.Error:      parseErr.Error(),
						logging.StandardFields.Status:     "parse_failed",
					}).Warn("Failed to parse sync branch metadata")
				} else {
					d.logger.WithError(parseErr).WithField("branch", branch.Name).Warn("Failed to parse sync branch")
				}
				continue
			}

			if metadata != nil {
				targetState.SyncBranches = append(targetState.SyncBranches, SyncBranch{
					Name:     branch.Name,
					Metadata: metadata,
				})

				if d.logConfig != nil && d.logConfig.Debug.State {
					logger.WithFields(logrus.Fields{
						logging.StandardFields.BranchName: branch.Name,
						logging.StandardFields.CommitSHA:  metadata.CommitSHA,
						logging.StandardFields.Timestamp:  metadata.Timestamp,
						logging.StandardFields.Status:     "parsed",
					}).Trace("Successfully parsed sync branch")
				}
			}
		}
	}

	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithFields(logrus.Fields{
			"total_branches":          len(branches),
			"potential_sync_branches": syncBranchCount,
			"valid_sync_branches":     len(targetState.SyncBranches),
		}).Debug("Branch analysis completed")
	}

	// Find the latest sync commit from branches
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.Debug("Determining latest sync commit from branch metadata")
	}

	var latestSyncTime *time.Time
	for _, syncBranch := range targetState.SyncBranches {
		if syncBranch.Metadata != nil {
			if latestSyncTime == nil || syncBranch.Metadata.Timestamp.After(*latestSyncTime) {
				latestSyncTime = &syncBranch.Metadata.Timestamp
				targetState.LastSyncCommit = syncBranch.Metadata.CommitSHA
				targetState.LastSyncTime = latestSyncTime

				if d.logConfig != nil && d.logConfig.Debug.State {
					logger.WithFields(logrus.Fields{
						"latest_branch": syncBranch.Name,
						"commit_sha":    syncBranch.Metadata.CommitSHA,
						"sync_time":     syncBranch.Metadata.Timestamp,
					}).Trace("Updated latest sync commit")
				}
			}
		}
	}

	if d.logConfig != nil && d.logConfig.Debug.State {
		if latestSyncTime != nil {
			logger.WithFields(logrus.Fields{
				"last_sync_commit": targetState.LastSyncCommit,
				"last_sync_time":   latestSyncTime,
			}).Debug("Latest sync commit determined")
		} else {
			logger.Debug("No sync history found in branches")
		}
	}

	// List open PRs
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.Debug("Listing open PRs to find sync-related PRs")
	}

	prStart := time.Now()
	prs, err := d.gh.ListPRs(ctx, repo, "open")
	prDuration := time.Since(prStart)

	if err != nil {
		if d.logConfig != nil && d.logConfig.Debug.State {
			logger.WithFields(logrus.Fields{
				"error":       err.Error(),
				"duration_ms": prDuration.Milliseconds(),
			}).Error("Failed to list open PRs")
		}
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithFields(logrus.Fields{
			"total_prs":   len(prs),
			"duration_ms": prDuration.Milliseconds(),
		}).Debug("Successfully listed open PRs")
	}

	// Find sync-related PRs
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.Debug("Analyzing PRs for sync patterns")
	}

	syncPrCount := 0
	for _, pr := range prs {
		// Check if PR is from a sync branch
		if strings.HasPrefix(pr.Head.Ref, syncBranchPrefix) {
			syncPrCount++
			targetState.OpenPRs = append(targetState.OpenPRs, pr)

			if d.logConfig != nil && d.logConfig.Debug.State {
				logger.WithFields(logrus.Fields{
					"pr_number":   pr.Number,
					"pr_title":    pr.Title,
					"head_branch": pr.Head.Ref,
				}).Trace("Found sync-related PR")
			}
		}
	}

	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithFields(logrus.Fields{
			"total_prs": len(prs),
			"sync_prs":  len(targetState.OpenPRs),
		}).Debug("PR analysis completed")
	}

	// Log successful discovery completion
	duration := time.Since(start)
	if d.logConfig != nil && d.logConfig.Debug.State {
		logger.WithFields(logrus.Fields{
			"duration_ms":      duration.Milliseconds(),
			"sync_branches":    len(targetState.SyncBranches),
			"open_sync_prs":    len(targetState.OpenPRs),
			"last_sync_commit": targetState.LastSyncCommit,
		}).Debug("Target repository state discovery completed successfully")
	}

	return targetState, nil
}

// ParseBranchName parses a branch name to extract sync metadata
func (d *discoveryService) ParseBranchName(name string) (*BranchMetadata, error) {
	// This will be implemented in branch.go
	// For now, return a placeholder
	return parseSyncBranchName(name)
}

// ParseBranchNameWithPrefix parses a branch name with a specific prefix to extract sync metadata
func (d *discoveryService) ParseBranchNameWithPrefix(name, branchPrefix string) (*BranchMetadata, error) {
	return parseSyncBranchNameWithPrefix(name, branchPrefix)
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
