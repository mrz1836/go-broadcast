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
	config    *config.Config
	gh        gh.Client
	git       git.Client
	state     state.Discoverer
	transform transform.Chain
	options   *Options
	logger    *logrus.Logger
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

	// Config is already in multi-source format

	// 1. Discover current state from GitHub for all sources
	log.Info("Discovering current state from GitHub")
	currentState, err := e.state.DiscoverState(ctx, e.config)
	if err != nil {
		return appErrors.WrapWithContext(err, "discover current state")
	}

	log.WithFields(logrus.Fields{
		"source_count": len(e.config.Mappings),
		"target_count": len(currentState.Targets),
	}).Info("State discovery completed")

	// 2. Build sync tasks for all source-target pairs
	syncTasks, err := e.buildSyncTasks(targetFilter, currentState)
	if err != nil {
		return appErrors.WrapWithContext(err, "build sync tasks")
	}

	if len(syncTasks) == 0 {
		log.Info("No targets require synchronization")
		return nil
	}

	log.WithField("sync_tasks", len(syncTasks)).Info("Sync tasks created")

	// 3. Create progress tracker
	progress := NewProgressTracker(len(syncTasks), e.options.DryRun)

	// 4. Process sync tasks concurrently
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(e.options.MaxConcurrency)

	for _, task := range syncTasks {
		g.Go(func() error {
			return e.syncTask(ctx, task, currentState, progress)
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

// Task represents a single source-to-target synchronization task
type Task struct {
	Source     config.SourceConfig
	Target     config.TargetConfig
	Defaults   *config.DefaultConfig
	MappingIdx int // Index of the mapping this task belongs to
}

// buildSyncTasks creates sync tasks for all source-target pairs that need synchronization
func (e *Engine) buildSyncTasks(targetFilter []string, currentState *state.State) ([]Task, error) {
	var tasks []Task

	// Build filter map for efficiency
	filterMap := make(map[string]bool)
	for _, f := range targetFilter {
		filterMap[f] = true
	}

	// Process each mapping
	for mappingIdx, mapping := range e.config.Mappings {
		// Process each target in this mapping
		for _, target := range mapping.Targets {
			// Apply filter if specified
			if len(filterMap) > 0 && !filterMap[target.Repo] {
				continue
			}

			// Check if sync is needed (unless forced)
			if !e.options.Force && !e.needsSyncForPair(mapping.Source, target, currentState) {
				e.logger.WithFields(logrus.Fields{
					"source": mapping.Source.Repo,
					"target": target.Repo,
				}).Info("Target is up-to-date for this source, skipping")
				continue
			}

			// Create sync task
			task := Task{
				Source:     mapping.Source,
				Target:     target,
				Defaults:   mapping.Defaults,
				MappingIdx: mappingIdx,
			}
			tasks = append(tasks, task)
		}
	}

	if len(filterMap) > 0 && len(tasks) == 0 {
		return nil, fmt.Errorf("%w: %v", appErrors.ErrNoMatchingTargets, targetFilter)
	}

	return tasks, nil
}

// needsSyncForPair determines if a specific source-target pair needs synchronization
func (e *Engine) needsSyncForPair(source config.SourceConfig, target config.TargetConfig, currentState *state.State) bool {
	targetState, exists := currentState.Targets[target.Repo]
	if !exists {
		// No state found, sync needed
		return true
	}

	// Check if this specific source has been synced to this target
	_, sourceExists := currentState.Sources[source.Repo]
	if !sourceExists {
		// Source state not found, sync needed
		return true
	}

	// Check if target is behind this specific source
	// This would be enhanced in the state management update
	switch targetState.Status {
	case state.StatusUpToDate:
		// Need to check if up-to-date with THIS source specifically
		// For now, assume needs sync if multiple sources exist
		return len(e.config.Mappings) > 1
	case state.StatusBehind:
		return true
	case state.StatusPending:
		// PR is open, check if we should update it
		return e.options.UpdateExistingPRs
	case state.StatusConflict:
		// Conflicts require manual intervention
		e.logger.WithFields(logrus.Fields{
			"source": source.Repo,
			"target": target.Repo,
		}).Warn("Repository has conflicts, skipping automatic sync")
		return false
	default:
		// Unknown status, err on the side of caution and sync
		return true
	}
}

// syncTask handles synchronization for a single source-target pair
func (e *Engine) syncTask(ctx context.Context, task Task, currentState *state.State, progress *ProgressTracker) error {
	log := e.logger.WithFields(logrus.Fields{
		"source_repo": task.Source.Repo,
		"target_repo": task.Target.Repo,
		"component":   "sync_task",
	})

	taskID := fmt.Sprintf("%s->%s", task.Source.Repo, task.Target.Repo)
	progress.StartRepository(taskID)
	defer progress.FinishRepository(taskID)

	log.Info("Starting sync task")

	// Get source and target states
	sourceState, sourceExists := currentState.Sources[task.Source.Repo]
	if !sourceExists {
		return appErrors.SourceStateNotFoundError(task.Source.Repo)
	}

	targetState := currentState.Targets[task.Target.Repo]

	// Create repository sync with task-specific configuration
	repoSync := &RepositorySync{
		engine:      e,
		source:      task.Source,
		target:      task.Target,
		sourceState: &sourceState,
		targetState: targetState,
		logger:      log,
		defaults:    task.Defaults,
	}

	// Execute the sync
	if err := repoSync.Execute(ctx); err != nil {
		log.WithError(err).Error("Repository sync failed")
		progress.RecordError(taskID, err)
		return err
	}

	progress.RecordSuccess(taskID)
	return nil
}
