package state

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Branch validation errors
var (
	ErrBranchPrefixEmpty   = errors.New("branch prefix cannot be empty")
	ErrBranchPrefixInvalid = errors.New("branch prefix contains invalid characters")
	ErrNotSyncBranch       = errors.New("not a sync branch")
)

// branchPattern matches sync branch names: sync/template-YYYYMMDD-HHMMSS-{commit}
var branchPattern = regexp.MustCompile(`^(sync/template)-(\d{8})-(\d{6})-([a-fA-F0-9]+)$`)

// parseSyncBranchName parses a branch name to extract sync metadata
func parseSyncBranchName(name string) (*BranchMetadata, error) {
	matches := branchPattern.FindStringSubmatch(name)
	if matches == nil {
		// Not a sync branch
		return nil, ErrNotSyncBranch
	}

	// Extract components
	prefix := matches[1]
	dateStr := matches[2]
	timeStr := matches[3]
	commitSHA := matches[4]

	// Parse timestamp
	timestampStr := fmt.Sprintf("%s%s", dateStr, timeStr)

	timestamp, err := time.Parse("20060102150405", timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp from branch name %s: %w", name, err)
	}

	return &BranchMetadata{
		Timestamp: timestamp,
		CommitSHA: commitSHA,
		Prefix:    prefix,
	}, nil
}

// FormatSyncBranchName creates a sync branch name from metadata
func FormatSyncBranchName(prefix string, timestamp time.Time, commitSHA string) string {
	return fmt.Sprintf("%s-%s-%s",
		prefix,
		timestamp.Format("20060102-150405"),
		commitSHA,
	)
}

// ValidateBranchPrefix checks if a branch prefix is valid
func ValidateBranchPrefix(prefix string) error {
	if prefix == "" {
		return ErrBranchPrefixEmpty
	}

	// Check for invalid characters
	invalidChars := regexp.MustCompile(`[^a-zA-Z0-9/_-]`)
	if invalidChars.MatchString(prefix) {
		return ErrBranchPrefixInvalid
	}

	return nil
}
