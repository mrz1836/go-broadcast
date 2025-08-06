package sync

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ProgressTracker tracks the progress of sync operations across multiple repositories
type ProgressTracker struct {
	mu         sync.RWMutex
	totalRepos int
	completed  int
	successful int
	failed     int
	skipped    int
	errors     map[string]error
	repoStatus map[string]RepoStatus
	startTime  time.Time
	dryRun     bool
	lastError  error
	// Group context for better logging
	groupName string
	groupID   string
}

// RepoStatus represents the status of a repository sync
type RepoStatus string

const (
	// RepoStatusPending indicates the repo sync hasn't started
	RepoStatusPending RepoStatus = "pending"

	// RepoStatusInProgress indicates the repo sync is running
	RepoStatusInProgress RepoStatus = "in_progress"

	// RepoStatusSuccess indicates the repo sync completed successfully
	RepoStatusSuccess RepoStatus = "success"

	// RepoStatusFailed indicates the repo sync failed
	RepoStatusFailed RepoStatus = "failed"

	// RepoStatusSkipped indicates the repo sync was skipped
	RepoStatusSkipped RepoStatus = "skipped"
)

// Results contains the final results of a sync operation
type Results struct {
	TotalRepos int
	Successful int
	Failed     int
	Skipped    int
	Duration   time.Duration
	Errors     map[string]error
	DryRun     bool
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(totalRepos int, dryRun bool) *ProgressTracker {
	return &ProgressTracker{
		totalRepos: totalRepos,
		errors:     make(map[string]error),
		repoStatus: make(map[string]RepoStatus),
		startTime:  time.Now(),
		dryRun:     dryRun,
	}
}

// NewProgressTrackerWithGroup creates a new progress tracker with group context
func NewProgressTrackerWithGroup(totalRepos int, dryRun bool, groupName, groupID string) *ProgressTracker {
	return &ProgressTracker{
		totalRepos: totalRepos,
		errors:     make(map[string]error),
		repoStatus: make(map[string]RepoStatus),
		startTime:  time.Now(),
		dryRun:     dryRun,
		groupName:  groupName,
		groupID:    groupID,
	}
}

// StartRepository marks a repository as started
func (p *ProgressTracker) StartRepository(repo string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.repoStatus[repo] = RepoStatusInProgress

	fields := logrus.Fields{
		"repo":     repo,
		"progress": p.getProgressString(),
		"dry_run":  p.dryRun,
	}
	if p.groupName != "" {
		fields["group_name"] = p.groupName
	}
	if p.groupID != "" {
		fields["group_id"] = p.groupID
	}
	logrus.WithFields(fields).Info("Starting repository sync")
}

// FinishRepository marks a repository as finished (used with defer)
func (p *ProgressTracker) FinishRepository(repo string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Only update if not already set by RecordSuccess/RecordError
	if p.repoStatus[repo] == RepoStatusInProgress {
		p.repoStatus[repo] = RepoStatusSuccess
		p.successful++
	}

	p.completed++
}

// RecordSuccess records a successful repository sync
func (p *ProgressTracker) RecordSuccess(repo string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.repoStatus[repo] = RepoStatusSuccess
	p.successful++

	fields := logrus.Fields{
		"repo":     repo,
		"progress": p.getProgressString(),
		"dry_run":  p.dryRun,
	}
	if p.groupName != "" {
		fields["group_name"] = p.groupName
	}
	if p.groupID != "" {
		fields["group_id"] = p.groupID
	}
	logrus.WithFields(fields).Info("Repository sync completed successfully")
}

// RecordError records a failed repository sync
func (p *ProgressTracker) RecordError(repo string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.repoStatus[repo] = RepoStatusFailed
	p.errors[repo] = err
	p.failed++
	p.lastError = err

	fields := logrus.Fields{
		"repo":     repo,
		"error":    err.Error(),
		"progress": p.getProgressString(),
		"dry_run":  p.dryRun,
	}
	if p.groupName != "" {
		fields["group_name"] = p.groupName
	}
	if p.groupID != "" {
		fields["group_id"] = p.groupID
	}
	logrus.WithFields(fields).Error("Repository sync failed")
}

// RecordSkipped records a skipped repository sync
func (p *ProgressTracker) RecordSkipped(repo, reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.repoStatus[repo] = RepoStatusSkipped
	p.skipped++

	fields := logrus.Fields{
		"repo":     repo,
		"reason":   reason,
		"progress": p.getProgressString(),
		"dry_run":  p.dryRun,
	}
	if p.groupName != "" {
		fields["group_name"] = p.groupName
	}
	if p.groupID != "" {
		fields["group_id"] = p.groupID
	}
	logrus.WithFields(fields).Info("Repository sync skipped")
}

// SetError sets a global error for the sync operation
func (p *ProgressTracker) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastError = err
}

// GetResults returns the final results of the sync operation
func (p *ProgressTracker) GetResults() *Results {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &Results{
		TotalRepos: p.totalRepos,
		Successful: p.successful,
		Failed:     p.failed,
		Skipped:    p.skipped,
		Duration:   time.Since(p.startTime),
		Errors:     p.copyErrors(),
		DryRun:     p.dryRun,
	}
}

// GetProgress returns current progress information
func (p *ProgressTracker) GetProgress() (completed, total int, percentage float64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	completed = p.completed
	total = p.totalRepos

	if total > 0 {
		percentage = float64(completed) / float64(total) * 100
	}

	return completed, total, percentage
}

// HasErrors returns true if any errors were recorded
func (p *ProgressTracker) HasErrors() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.errors) > 0 || p.lastError != nil
}

// GetLastError returns the most recent error
func (p *ProgressTracker) GetLastError() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.lastError
}

// getProgressString returns a progress string (must be called with lock held)
func (p *ProgressTracker) getProgressString() string {
	return fmt.Sprintf("%d/%d", p.completed, p.totalRepos)
}

// copyErrors creates a copy of the errors map (must be called with lock held)
func (p *ProgressTracker) copyErrors() map[string]error {
	errors := make(map[string]error, len(p.errors))
	for repo, err := range p.errors {
		errors[repo] = err
	}
	return errors
}
