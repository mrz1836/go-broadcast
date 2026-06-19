package sync

import (
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
)

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

	// RateLimitPreflightEnabled enables the pre-sync rate-limit gate
	RateLimitPreflightEnabled bool

	// IgnoreRateLimitPreflight forces the sync through even when the preflight
	// gate would halt (the CLI --ignore-rate-limit-preflight escape hatch)
	IgnoreRateLimitPreflight bool

	// RateLimitPrimaryMarginPercent is the percentage of the live primary budget
	// to keep as headroom before the preflight halts
	RateLimitPrimaryMarginPercent int

	// RateLimitSecondaryReserve is how many of the documented per-minute secondary
	// content-write slots to keep in reserve before the preflight halts
	RateLimitSecondaryReserve int

	// RateLimitFailClosed halts the sync when the rate-limit probe is unavailable
	// instead of failing open with a warning
	RateLimitFailClosed bool

	// ConfirmScope, when non-nil, is the operator-supplied resolved repository
	// count used to satisfy the blast-radius guard (the --confirm-scope=<N> flag).
	// nil means the flag was not provided. The value must equal the resolved repo
	// count; a boolean always-pass token is intentionally not accepted (Q7=A).
	ConfirmScope *int
}

// DefaultOptions returns the default sync options
func DefaultOptions() *Options {
	return &Options{
		DryRun:                        false,
		Force:                         false,
		MaxConcurrency:                5,
		UpdateExistingPRs:             true,
		Timeout:                       10 * time.Minute,
		CleanupTempFiles:              true,
		RateLimitPreflightEnabled:     true,
		RateLimitPrimaryMarginPercent: config.DefaultRateLimitPrimaryMarginPercent,
		RateLimitSecondaryReserve:     config.DefaultRateLimitSecondaryReserve,
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

// WithRateLimitPreflight enables or disables the pre-sync rate-limit gate
func (o *Options) WithRateLimitPreflight(enabled bool) *Options {
	o.RateLimitPreflightEnabled = enabled
	return o
}

// WithRateLimitMargins sets the preflight safety margins: the primary-budget
// headroom percentage and the per-minute secondary content-write reserve
func (o *Options) WithRateLimitMargins(primaryMarginPercent, secondaryReserve int) *Options {
	o.RateLimitPrimaryMarginPercent = primaryMarginPercent
	o.RateLimitSecondaryReserve = secondaryReserve
	return o
}

// WithRateLimitFailClosed sets whether to halt when the rate-limit probe fails
func (o *Options) WithRateLimitFailClosed(failClosed bool) *Options {
	o.RateLimitFailClosed = failClosed
	return o
}

// WithIgnoreRateLimitPreflight sets whether to force the sync through even when
// the preflight gate would halt
func (o *Options) WithIgnoreRateLimitPreflight(ignore bool) *Options {
	o.IgnoreRateLimitPreflight = ignore
	return o
}

// WithConfirmScope sets the operator-supplied resolved repository count used to
// satisfy the blast-radius guard. Pass nil to leave it unset (the flag was not
// provided), or a pointer to the confirmed repo count.
func (o *Options) WithConfirmScope(n *int) *Options {
	o.ConfirmScope = n
	return o
}
