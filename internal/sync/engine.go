// Package sync provides the core synchronization engine
package sync

import (
	"context"
	"fmt"

	"github.com/mrz1836/go-broadcast/internal/config"
	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Engine orchestrates the complete synchronization process
type Engine struct {
	config       *config.Config
	currentGroup *config.Group // Current group being processed
	gh           gh.Client
	git          git.Client
	state        state.Discoverer
	transform    transform.Chain
	options      *Options
	logger       *logrus.Logger
}

// NewEngine creates a new sync engine with the provided dependencies
func NewEngine(
	cfg *config.Config,
	ghClient gh.Client,
	gitClient git.Client,
	stateDiscoverer state.Discoverer,
	transformChain transform.Chain,
	opts *Options,
) *Engine {
	if opts == nil {
		opts = DefaultOptions()
	}

	return &Engine{
		config:    cfg,
		gh:        ghClient,
		git:       gitClient,
		state:     stateDiscoverer,
		transform: transformChain,
		options:   opts,
		logger:    logrus.StandardLogger(),
	}
}

// SetLogger sets a custom logger for the engine
func (e *Engine) SetLogger(logger *logrus.Logger) {
	e.logger = logger
}

// Sync orchestrates the complete synchronization process
func (e *Engine) Sync(ctx context.Context, targetFilter []string) error {
	log := e.logger.WithField("component", "sync_engine")

	log.Info("Starting sync operation")
	if e.options.DryRun {
		log.Warn("DRY-RUN MODE: No changes will be made")
	}

	// Get groups using compatibility layer
	groups := e.config.GetGroups()
	if len(groups) == 0 {
		log.Info("No groups found in configuration")
		return nil
	}

	log.WithField("group_count", len(groups)).Info("Processing sync groups")

	// For Phase 2a, we handle single group (compatibility mode)
	// Future phases will implement full group orchestration
	group := groups[0]
	e.currentGroup = &group // Set current group for RepositorySync to use
	log.WithField("group_name", group.Name).Info("Processing group")

	// 1. Discover current state from GitHub
	log.Info("Discovering current state from GitHub")
	currentState, err := e.state.DiscoverState(ctx, e.config)
	if err != nil {
		return appErrors.WrapWithContext(err, "discover current state")
	}

	log.WithFields(logrus.Fields{
		"source_repo":   currentState.Source.Repo,
		"source_commit": currentState.Source.LatestCommit,
		"target_count":  len(currentState.Targets),
	}).Info("State discovery completed")

	// 2. Determine which targets to sync using group's targets
	syncTargets, err := e.filterGroupTargets(targetFilter, group, currentState)
	if err != nil {
		return appErrors.WrapWithContext(err, "filter targets")
	}

	if len(syncTargets) == 0 {
		log.Info("No targets require synchronization")
		return nil
	}

	log.WithField("sync_targets", len(syncTargets)).Info("Targets selected for sync")

	// 3. Create progress tracker
	progress := NewProgressTracker(len(syncTargets), e.options.DryRun)

	// 4. Process repositories concurrently
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(e.options.MaxConcurrency)

	for _, target := range syncTargets {
		g.Go(func() error {
			return e.syncRepository(ctx, target, currentState, progress)
		})
	}

	// 5. Wait for all syncs to complete
	if err := g.Wait(); err != nil {
		progress.SetError(err)
		return appErrors.WrapWithContext(err, "complete sync operation")
	}

	// 6. Report final results
	results := progress.GetResults()
	log.WithFields(logrus.Fields{
		"successful": results.Successful,
		"failed":     results.Failed,
		"skipped":    results.Skipped,
		"duration":   results.Duration,
	}).Info("Sync operation completed")

	if results.Failed > 0 {
		return fmt.Errorf("%w: completed with %d failures", appErrors.ErrSyncFailed, results.Failed)
	}

	return nil
}

// filterGroupTargets determines which targets need to be synced based on filters, group, and current state
func (e *Engine) filterGroupTargets(targetFilter []string, group config.Group, currentState *state.State) ([]config.TargetConfig, error) {
	var targets []config.TargetConfig

	// If no filter specified, use all targets from the group
	if len(targetFilter) == 0 {
		targets = group.Targets
	} else {
		// Filter targets based on command line arguments
		for _, target := range group.Targets {
			for _, filter := range targetFilter {
				if target.Repo == filter {
					targets = append(targets, target)
					break
				}
			}
		}

		if len(targets) == 0 {
			return nil, fmt.Errorf("%w: %v", appErrors.ErrNoMatchingTargets, targetFilter)
		}
	}

	// Use common filtering logic
	return e.filterTargetsFromList(targets, currentState)
}

// filterTargets determines which targets need to be synced based on filters and current state (legacy method)
func (e *Engine) filterTargets(targetFilter []string, currentState *state.State) ([]config.TargetConfig, error) {
	// Try to use compatibility layer first
	groups := e.config.GetGroups()
	if len(groups) > 0 {
		return e.filterGroupTargets(targetFilter, groups[0], currentState)
	}

	// Fallback to direct field access for incomplete configs (like in tests)
	var targets []config.TargetConfig

	// If no filter specified, use all configured targets
	if len(targetFilter) == 0 {
		targets = e.config.Targets
	} else {
		// Filter targets based on command line arguments
		for _, target := range e.config.Targets {
			for _, filter := range targetFilter {
				if target.Repo == filter {
					targets = append(targets, target)
					break
				}
			}
		}

		if len(targets) == 0 {
			return nil, fmt.Errorf("%w: %v", appErrors.ErrNoMatchingTargets, targetFilter)
		}
	}

	return e.filterTargetsFromList(targets, currentState)
}

// filterTargetsFromList filters targets from a provided list based on sync necessity
func (e *Engine) filterTargetsFromList(targets []config.TargetConfig, currentState *state.State) ([]config.TargetConfig, error) {
	// Further filter based on sync necessity (unless forced)
	if !e.options.Force {
		var syncNeeded []config.TargetConfig

		for _, target := range targets {
			if e.needsSync(target, currentState) {
				syncNeeded = append(syncNeeded, target)
			} else {
				e.logger.WithField("repo", target.Repo).Info("Target is up-to-date, skipping")
			}
		}

		targets = syncNeeded
	}

	return targets, nil
}

// needsSync determines if a target repository needs synchronization
func (e *Engine) needsSync(target config.TargetConfig, currentState *state.State) bool {
	targetState, exists := currentState.Targets[target.Repo]
	if !exists {
		// No state found, sync needed
		return true
	}

	switch targetState.Status {
	case state.StatusUpToDate:
		return false
	case state.StatusBehind:
		return true
	case state.StatusPending:
		// PR is open, check if we should update it
		return e.options.UpdateExistingPRs
	case state.StatusConflict:
		// Conflicts require manual intervention
		e.logger.WithField("repo", target.Repo).Warn("Repository has conflicts, skipping automatic sync")
		return false
	default:
		// Unknown status, err on the side of caution and sync
		return true
	}
}

// syncRepository handles synchronization for a single repository
func (e *Engine) syncRepository(ctx context.Context, target config.TargetConfig, currentState *state.State, progress *ProgressTracker) error {
	log := e.logger.WithFields(logrus.Fields{
		"target_repo": target.Repo,
		"component":   "repository_sync",
	})

	progress.StartRepository(target.Repo)
	defer progress.FinishRepository(target.Repo)

	log.Info("Starting repository sync")

	// Get target state
	targetState := currentState.Targets[target.Repo]

	// Create repository syncer
	repoSync := &RepositorySync{
		engine:      e,
		target:      target,
		sourceState: &currentState.Source,
		targetState: targetState,
		logger:      log,
	}

	// Execute sync
	err := repoSync.Execute(ctx)
	if err != nil {
		log.WithError(err).Error("Repository sync failed")
		progress.RecordError(target.Repo, err)
		return appErrors.WrapWithContext(err, fmt.Sprintf("sync %s", target.Repo))
	}

	log.Info("Repository sync completed successfully")
	progress.RecordSuccess(target.Repo)
	return nil
}
