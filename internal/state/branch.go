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

// parseSyncBranchName parses a branch name to extract sync metadata
func parseSyncBranchName(name string) (*BranchMetadata, error) {
	// Legacy support for hardcoded chore/sync-files prefix
	return parseSyncBranchNameWithPrefix(name, "chore/sync-files")
}

// parseSyncBranchNameWithPrefix parses a branch name with a specific prefix to extract sync metadata
func parseSyncBranchNameWithPrefix(name, prefix string) (*BranchMetadata, error) {
	// Format: prefix-{groupID}-YYYYMMDD-HHMMSS-{commit}
	escapedPrefix := regexp.QuoteMeta(prefix)
	pattern := fmt.Sprintf(`^(%s)-([a-zA-Z0-9_-]+)-(\d{8})-(\d{6})-([a-fA-F0-9]+)$`, escapedPrefix)
	branchPattern := regexp.MustCompile(pattern)

	matches := branchPattern.FindStringSubmatch(name)
	if matches == nil {
		// Not a sync branch
		return nil, ErrNotSyncBranch
	}

	// Extract components
	extractedPrefix := matches[1]
	groupID := matches[2]
	dateStr := matches[3]
	timeStr := matches[4]
	commitSHA := matches[5]

	// Parse timestamp
	timestampStr := fmt.Sprintf("%s%s", dateStr, timeStr)
	timestamp, err := time.Parse("20060102150405", timestampStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp from branch name %s: %w", name, err)
	}

	return &BranchMetadata{
		Timestamp: timestamp,
		CommitSHA: commitSHA,
		Prefix:    extractedPrefix,
		GroupID:   groupID,
	}, nil
}

// FormatSyncBranchName creates a sync branch name with group ID
func FormatSyncBranchName(prefix, groupID string, timestamp time.Time, commitSHA string) string {
	return fmt.Sprintf("%s-%s-%s-%s",
		prefix,
		groupID,
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
