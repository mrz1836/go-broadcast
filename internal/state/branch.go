package state

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"
)

// Cached regex patterns for performance - compiled once at package init
var (
	// invalidCharsPattern validates branch prefix characters
	invalidCharsPattern = regexp.MustCompile(`[^a-zA-Z0-9/_-]`)

	// branchPatternCache caches compiled regex patterns keyed by prefix
	// to avoid recompilation on every parseSyncBranchNameWithPrefix call
	branchPatternCache sync.Map //nolint:gochecknoglobals // intentional cache for performance
)

// Branch validation errors
var (
	ErrBranchPrefixEmpty   = errors.New("branch prefix cannot be empty")
	ErrBranchPrefixInvalid = errors.New("branch prefix contains invalid characters")
	ErrNotSyncBranch       = errors.New("not a sync branch")
)

// getBranchPattern returns a cached compiled regex for the given branch prefix.
// This avoids recompiling the same regex pattern on every call.
func getBranchPattern(prefix string) *regexp.Regexp {
	// Check cache first
	if cached, ok := branchPatternCache.Load(prefix); ok {
		return cached.(*regexp.Regexp)
	}

	// Compile new pattern - Format: prefix-{groupID}-YYYYMMDD-HHMMSS-{commit}
	escapedPrefix := regexp.QuoteMeta(prefix)
	pattern := fmt.Sprintf(`^(%s)-([a-zA-Z0-9_-]+)-(\d{8})-(\d{6})-([a-fA-F0-9]+)$`, escapedPrefix)
	compiled := regexp.MustCompile(pattern)

	// Store in cache (LoadOrStore handles race condition)
	actual, _ := branchPatternCache.LoadOrStore(prefix, compiled)
	return actual.(*regexp.Regexp)
}

// parseSyncBranchName parses a branch name to extract sync metadata
func parseSyncBranchName(name string) (*BranchMetadata, error) {
	// Legacy support for hardcoded chore/sync-files prefix
	return parseSyncBranchNameWithPrefix(name, "chore/sync-files")
}

// parseSyncBranchNameWithPrefix parses a branch name with a specific prefix to extract sync metadata
func parseSyncBranchNameWithPrefix(name, prefix string) (*BranchMetadata, error) {
	// Format: prefix-{groupID}-YYYYMMDD-HHMMSS-{commit}
	branchPattern := getBranchPattern(prefix)

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

	// Check for invalid characters using cached pattern
	if invalidCharsPattern.MatchString(prefix) {
		return ErrBranchPrefixInvalid
	}

	return nil
}
