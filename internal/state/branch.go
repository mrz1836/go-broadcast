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
	// Create pattern for the given prefix: prefix-YYYYMMDD-HHMMSS-{commit}
	escapedPrefix := regexp.QuoteMeta(prefix)
	pattern := fmt.Sprintf(`^(%s)-(\d{8})-(\d{6})-([a-fA-F0-9]+)$`, escapedPrefix)
	branchPattern := regexp.MustCompile(pattern)

	matches := branchPattern.FindStringSubmatch(name)
	if matches == nil {
		// Not a sync branch
		return nil, ErrNotSyncBranch
	}

	// Extract components
	extractedPrefix := matches[1]
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
		Prefix:    extractedPrefix,
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
