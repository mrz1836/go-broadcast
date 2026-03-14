package sync

import "time"

// Options configures the behavior of the sync engine
type Options struct {
	// DryRun indicates whether to simulate changes without making them
	DryRun bool

	// Force indicates whether to sync even if targets appear up-to-date
	Force bool

	// MaxConcurrency controls how many repositories can be synced simultaneously
	MaxConcurrency int

	// UpdateExistingPRs indicates whether to update existing sync PRs
	UpdateExistingPRs bool

	// Timeout is the maximum time to wait for each repository sync
	Timeout time.Duration

	// CleanupTempFiles indicates whether to clean up temporary files after sync
	CleanupTempFiles bool

	// GroupFilter specifies which groups to sync (by name or ID)
	// Empty means sync all groups
	GroupFilter []string

	// SkipGroups specifies which groups to skip (by name or ID)
	SkipGroups []string

	// Automerge indicates whether to add automerge labels to created PRs
	Automerge bool

	// AutomergeLabels specifies the labels to add when automerge is enabled
	AutomergeLabels []string

	// AIEnabled indicates whether AI text generation is enabled (master switch)
	AIEnabled bool

	// AIPREnabled indicates whether AI PR body generation is enabled
	AIPREnabled bool

	// AICommitEnabled indicates whether AI commit message generation is enabled
	AICommitEnabled bool

	// ClearModuleCache indicates whether to clear the module version cache before sync
	ClearModuleCache bool
}

// DefaultOptions returns the default sync options
func DefaultOptions() *Options {
	return &Options{
		DryRun:            false,
		Force:             false,
		MaxConcurrency:    5,
		UpdateExistingPRs: true,
		Timeout:           10 * time.Minute,
		CleanupTempFiles:  true,
	}
}

// WithDryRun sets the dry-run option
func (o *Options) WithDryRun(dryRun bool) *Options {
	o.DryRun = dryRun
	return o
}

// WithForce sets the force option
func (o *Options) WithForce(force bool) *Options {
	o.Force = force
	return o
}

// WithMaxConcurrency sets the maximum concurrency
func (o *Options) WithMaxConcurrency(maxConcurrency int) *Options {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}
	o.MaxConcurrency = maxConcurrency
	return o
}

// WithTimeout sets the sync timeout
func (o *Options) WithTimeout(timeout time.Duration) *Options {
	o.Timeout = timeout
	return o
}

// WithGroupFilter sets the groups to sync
func (o *Options) WithGroupFilter(groups []string) *Options {
	o.GroupFilter = groups
	return o
}

// WithSkipGroups sets the groups to skip
func (o *Options) WithSkipGroups(skipGroups []string) *Options {
	o.SkipGroups = skipGroups
	return o
}

// WithAutomerge sets the automerge option
func (o *Options) WithAutomerge(automerge bool) *Options {
	o.Automerge = automerge
	return o
}

// WithAutomergeLabels sets the automerge labels
func (o *Options) WithAutomergeLabels(labels []string) *Options {
	o.AutomergeLabels = labels
	return o
}

// WithAIEnabled sets the AI generation master switch
func (o *Options) WithAIEnabled(enabled bool) *Options {
	o.AIEnabled = enabled
	return o
}

// WithAIPREnabled sets AI PR body generation
func (o *Options) WithAIPREnabled(enabled bool) *Options {
	o.AIPREnabled = enabled
	return o
}

// WithAICommitEnabled sets AI commit message generation
func (o *Options) WithAICommitEnabled(enabled bool) *Options {
	o.AICommitEnabled = enabled
	return o
}

// WithClearModuleCache sets whether to clear the module version cache before sync
func (o *Options) WithClearModuleCache(enabled bool) *Options {
	o.ClearModuleCache = enabled
	return o
}
