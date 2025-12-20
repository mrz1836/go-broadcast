// Package sync provides the core synchronization engine
package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/mrz1836/go-broadcast/internal/ai"
	"github.com/mrz1836/go-broadcast/internal/config"
	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// Engine orchestrates the complete synchronization process
type Engine struct {
	config         *config.Config
	currentGroup   *config.Group // Current group being processed
	currentGroupMu sync.RWMutex  // Protects currentGroup access
	gh             gh.Client
	git            git.Client
	state          state.Discoverer
	transform      transform.Chain
	options        *Options
	logger         *logrus.Logger

	// AI text generation (optional, nil when disabled)
	prGenerator     *ai.PRBodyGenerator
	commitGenerator *ai.CommitMessageGenerator
	responseCache   *ai.ResponseCache
	diffTruncator   *ai.DiffTruncator
}

// NewEngine creates a new sync engine with the provided dependencies
func NewEngine(
	ctx context.Context,
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

	e := &Engine{
		config:    cfg,
		gh:        ghClient,
		git:       gitClient,
		state:     stateDiscoverer,
		transform: transformChain,
		options:   opts,
		logger:    logrus.StandardLogger(),
	}

	// Initialize AI components (non-fatal if it fails)
	e.initializeAI(ctx)

	return e
}

// SetLogger sets a custom logger for the engine
func (e *Engine) SetLogger(logger *logrus.Logger) {
	e.logger = logger
}

// SetPRGenerator sets a custom PR body generator for testing.
// Pass nil to disable AI PR body generation.
func (e *Engine) SetPRGenerator(gen *ai.PRBodyGenerator) {
	e.prGenerator = gen
}

// SetCommitGenerator sets a custom commit message generator for testing.
// Pass nil to disable AI commit message generation.
func (e *Engine) SetCommitGenerator(gen *ai.CommitMessageGenerator) {
	e.commitGenerator = gen
}

// SetResponseCache sets a custom response cache for testing.
func (e *Engine) SetResponseCache(cache *ai.ResponseCache) {
	e.responseCache = cache
}

// SetDiffTruncator sets a custom diff truncator for testing.
func (e *Engine) SetDiffTruncator(truncator *ai.DiffTruncator) {
	e.diffTruncator = truncator
}

// GetCurrentGroup returns the current group being processed (thread-safe).
func (e *Engine) GetCurrentGroup() *config.Group {
	e.currentGroupMu.RLock()
	defer e.currentGroupMu.RUnlock()
	return e.currentGroup
}

// SetCurrentGroup sets the current group being processed (thread-safe).
func (e *Engine) SetCurrentGroup(group *config.Group) {
	e.currentGroupMu.Lock()
	defer e.currentGroupMu.Unlock()
	e.currentGroup = group
}

// GitClient returns the git client for repository operations.
func (e *Engine) GitClient() git.Client {
	return e.git
}

// Options returns the sync options.
func (e *Engine) Options() *Options {
	return e.options
}

// initializeAI sets up AI text generation components.
// This is non-fatal - if AI initialization fails, we continue without AI.
func (e *Engine) initializeAI(ctx context.Context) {
	log := e.logger.WithField("component", "ai_init")

	// Load AI configuration from environment
	cfg := ai.LoadConfig()

	// Check if AI is enabled and provide detailed reason if not
	if !cfg.IsEnabled() {
		if !cfg.Enabled {
			log.Warn("AI text generation disabled by configuration (GO_BROADCAST_AI_ENABLED=false)")
		} else if cfg.APIKey == "" {
			log.Warn("AI text generation disabled: no API key configured (set GO_BROADCAST_AI_API_KEY or ANTHROPIC_API_KEY)")
		} else {
			log.Debug("AI text generation disabled")
		}
		return
	}

	// Create shared components
	e.responseCache = ai.NewResponseCache(cfg)
	e.diffTruncator = ai.NewDiffTruncator(cfg)
	retryConfig := ai.RetryConfigFromConfig(cfg)

	// Create provider
	provider, err := ai.NewProviderFromEnv(ctx, log)
	if err != nil {
		log.WithError(err).Warn("AI provider initialization failed, AI features disabled")
		return
	}

	// Initialize PR generator if enabled
	if cfg.IsPREnabled() {
		guidelines := ai.LoadPRGuidelines(".")
		e.prGenerator = ai.NewPRBodyGenerator(
			provider,
			e.responseCache,
			e.diffTruncator,
			retryConfig,
			guidelines,
			cfg.Timeout,
			log.WithField("generator", "pr_body"),
		)
		log.Info("AI PR body generation enabled")
	}

	// Initialize commit generator if enabled
	if cfg.IsCommitEnabled() {
		e.commitGenerator = ai.NewCommitMessageGenerator(
			provider,
			e.responseCache,
			e.diffTruncator,
			retryConfig,
			cfg.Timeout,
			log.WithField("generator", "commit_message"),
		)
		log.Info("AI commit message generation enabled")
	}
}

// Sync orchestrates the complete synchronization process
func (e *Engine) Sync(ctx context.Context, targetFilter []string) error {
	log := e.logger.WithField("component", "sync_engine")

	log.Info("Starting sync operation")
	if e.options.DryRun {
		log.Warn("DRY-RUN MODE: No changes will be made")
	}

	// Get groups using compatibility layer
	groups := e.config.Groups
	if len(groups) == 0 {
		log.Info("No groups found in configuration")
		return nil
	}

	log.WithField("group_count", len(groups)).Info("Processing sync groups")

	// Check if we have multiple groups - use orchestrator if so
	if len(groups) > 1 {
		// Use the orchestrator for multi-group execution
		orchestrator := NewGroupOrchestrator(e.config, e, e.logger)
		return orchestrator.ExecuteGroups(ctx, groups)
	}

	// Single group - execute directly
	group := groups[0]
	e.SetCurrentGroup(&group) // Set current group for RepositorySync to use
	log.WithField("group_name", group.Name).Info("Processing single group")

	// Execute the single group sync
	return e.executeSingleGroup(ctx, group, targetFilter)
}

// executeSingleGroup handles synchronization for a single group
func (e *Engine) executeSingleGroup(ctx context.Context, group config.Group, targetFilter []string) error {
	log := e.logger.WithField("component", "sync_engine")

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
	progress := NewProgressTrackerWithGroup(len(syncTargets), e.options.DryRun, group.Name, group.ID)

	// 4. Process repositories concurrently with error collection
	var g errgroup.Group
	g.SetLimit(e.options.MaxConcurrency)

	// Collect all errors instead of failing fast
	errorCollector := make(chan error, len(syncTargets))
	var hasContextError atomic.Bool

	for _, target := range syncTargets {
		g.Go(func() error {
			if err := e.syncRepository(ctx, target, currentState, progress); err != nil {
				// Check if this is a context cancellation error
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					hasContextError.Store(true)
				}

				// Send error to collector but don't return it (prevents context cancellation)
				select {
				case errorCollector <- err:
				default:
					// Channel full, log the error
					e.logger.WithError(err).WithField("repo", target.Repo).Error("Failed to collect sync error")
				}
			}
			return nil // Always return nil to prevent errgroup from canceling context
		})
	}

	// 5. Wait for all syncs to complete
	_ = g.Wait() // Always returns nil since we handle errors via errorCollector
	close(errorCollector)

	// Collect and log all errors
	collectedErrors := make([]error, 0, len(syncTargets))
	for err := range errorCollector {
		collectedErrors = append(collectedErrors, err)
		progress.SetError(err)

		// Check if this is a context error (by type or string content)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			hasContextError.Store(true)
		} else {
			// Also check error message for context-related terms
			errMsg := err.Error()
			if strings.Contains(errMsg, "context canceled") ||
				strings.Contains(errMsg, "context deadline exceeded") ||
				strings.Contains(errMsg, "deadline exceeded") {
				hasContextError.Store(true)
			}
		}
	}

	// 6. Report final results with detailed error information
	results := progress.GetResults()
	log.WithFields(logrus.Fields{
		"successful": results.Successful,
		"failed":     results.Failed,
		"skipped":    results.Skipped,
		"duration":   results.Duration,
		"errors":     len(collectedErrors),
	}).Info("Sync operation completed")

	// Log individual errors for debugging
	for i, err := range collectedErrors {
		log.WithError(err).WithField("error_index", i+1).Error("Individual sync failure")
	}

	if results.Failed > 0 {
		// If context was canceled/timeout, include context information in the error
		if hasContextError.Load() {
			if ctx.Err() != nil {
				return fmt.Errorf("%w: %w", appErrors.ErrSyncFailed, ctx.Err())
			}
			return fmt.Errorf("%w: context canceled", appErrors.ErrSyncFailed)
		}

		// Include details from the first few errors to provide better context
		var errorDetails []string
		maxDetailsToInclude := 3 // Limit to first 3 errors to keep message readable
		for i, err := range collectedErrors {
			if i >= maxDetailsToInclude {
				break
			}
			errorDetails = append(errorDetails, err.Error())
		}

		if len(errorDetails) > 0 {
			detailsStr := strings.Join(errorDetails, "; ")
			return fmt.Errorf("%w: completed with %d failures out of %d targets (%s)", appErrors.ErrSyncFailed, results.Failed, len(syncTargets), detailsStr)
		}

		return fmt.Errorf("%w: completed with %d failures out of %d targets", appErrors.ErrSyncFailed, results.Failed, len(syncTargets))
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
	groups := e.config.Groups
	if len(groups) > 0 {
		return e.filterGroupTargets(targetFilter, groups[0], currentState)
	}

	// Fallback to direct field access for incomplete configs (like in tests)
	var targets []config.TargetConfig

	// Get all targets from all groups
	allTargets := []config.TargetConfig{}
	for _, group := range e.config.Groups {
		allTargets = append(allTargets, group.Targets...)
	}

	// If no filter specified, use all configured targets
	if len(targetFilter) == 0 {
		targets = allTargets
	} else {
		// Filter targets based on command line arguments
		for _, target := range allTargets {
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
	fields := logrus.Fields{
		"target_repo": target.Repo,
		"component":   "repository_sync",
	}
	// Add group context if available
	if currentGroup := e.GetCurrentGroup(); currentGroup != nil {
		fields["group_name"] = currentGroup.Name
		fields["group_id"] = currentGroup.ID
	}
	log := e.logger.WithFields(fields)

	progress.StartRepository(target.Repo)
	defer progress.FinishRepository(target.Repo)

	log.Info("Starting repository sync")

	// Get target state
	targetState, exists := currentState.Targets[target.Repo]
	if !exists {
		log.Warn("Target state not found in current state, proceeding with nil state")
		targetState = nil
	}

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
