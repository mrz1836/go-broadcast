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
