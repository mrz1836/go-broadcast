package analytics

import (
	"time"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// HasChanged compares current snapshot data against the latest stored snapshot
// to determine if any meaningful changes have occurred.
//
// Fields compared:
//   - Stars, Forks, OpenIssues, OpenPRs, BranchCount
//   - LatestRelease, LatestTag
//   - PushedAt
//   - DependabotAlertCount, CodeScanningAlertCount, SecretScanningAlertCount
//
// At typical activity levels, ~80-90% of repos remain unchanged on any given day,
// so this provides massive DB savings by skipping unnecessary snapshot writes.
//
// Returns:
//   - true if data has changed (should write new snapshot)
//   - false if data is identical (skip snapshot write)
//   - true if previous is nil (first snapshot)
func HasChanged(current, previous *db.RepositorySnapshot) bool {
	// First snapshot - always write
	if previous == nil {
		return true
	}

	// Compare core metrics
	if current.Stars != previous.Stars {
		return true
	}
	if current.Forks != previous.Forks {
		return true
	}
	if current.OpenIssues != previous.OpenIssues {
		return true
	}
	if current.OpenPRs != previous.OpenPRs {
		return true
	}
	if current.BranchCount != previous.BranchCount {
		return true
	}

	// Compare release information
	if current.LatestRelease != previous.LatestRelease {
		return true
	}
	if current.LatestTag != previous.LatestTag {
		return true
	}

	// Compare activity timestamps
	if !timeEqual(current.PushedAt, previous.PushedAt) {
		return true
	}

	// Compare security alert counts
	if current.DependabotAlertCount != previous.DependabotAlertCount {
		return true
	}
	if current.CodeScanningAlertCount != previous.CodeScanningAlertCount {
		return true
	}
	if current.SecretScanningAlertCount != previous.SecretScanningAlertCount {
		return true
	}

	// No changes detected
	return false
}

// timeEqual safely compares two *time.Time values.
// Returns true if both are nil, or both are non-nil and equal.
func timeEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}
