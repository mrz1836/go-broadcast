package state

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Cached regex patterns for performance - compiled once at package init
var (
	// invalidCharsPattern validates branch prefix characters
	invalidCharsPattern = regexp.MustCompile(`[^a-zA-Z0-9/_-]`)

	// invalidScopeCharsPattern matches characters not allowed in the scope segment
	// of a sync branch name, which must stay within the branch parsing charset
	invalidScopeCharsPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

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

	// Compile new pattern - Format: prefix-{scope}-YYYYMMDD-HHMMSS-{commit}
	// The scope segment is intentionally generic so that branches created before
	// the scope changed from group ID to target repo still parse.
	escapedPrefix := regexp.QuoteMeta(prefix)
	pattern := fmt.Sprintf(`^(%s)-([a-zA-Z0-9_-]+)-(\d{8})-(\d{6})-([a-fA-F0-9]+)$`, escapedPrefix)
	compiled := regexp.MustCompile(pattern)

	// Store in cache (LoadOrStore handles race condition)
	actual, _ := branchPatternCache.LoadOrStore(prefix, compiled)
	return actual.(*regexp.Regexp)
}

// ParseSyncBranchName parses a branch name with the given prefix and returns its
// sync metadata, or ErrNotSyncBranch when the name only happens to share the
// prefix without matching the full generated format.
//
// Callers that act destructively on a branch should use this rather than a prefix
// check, so a hand-made branch under the same prefix is never mistaken for one
// go-broadcast created.
func ParseSyncBranchName(name, prefix string) (*BranchMetadata, error) {
	return parseSyncBranchNameWithPrefix(name, prefix)
}

// parseSyncBranchName parses a branch name to extract sync metadata
func parseSyncBranchName(name string) (*BranchMetadata, error) {
	// Legacy support for hardcoded chore/sync-files prefix
	return parseSyncBranchNameWithPrefix(name, "chore/sync-files")
}

// parseSyncBranchNameWithPrefix parses a branch name with a specific prefix to extract sync metadata
func parseSyncBranchNameWithPrefix(name, prefix string) (*BranchMetadata, error) {
	// Format: prefix-{scope}-YYYYMMDD-HHMMSS-{commit}
	branchPattern := getBranchPattern(prefix)

	matches := branchPattern.FindStringSubmatch(name)
	if matches == nil {
		// Not a sync branch
		return nil, ErrNotSyncBranch
	}

	// Extract components
	extractedPrefix := matches[1]
	scope := matches[2]
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
		Scope:     scope,
	}, nil
}

// FormatSyncBranchName creates a sync branch name scoped to a target repository.
//
// The scope segment identifies the target repo rather than the group that
// produced the sync, so group names stay internal to the configuration and two
// targets processed within the same second cannot generate the same name.
// Pass scope through FormatBranchScope first.
//
// The timestamp is normalized to UTC because parsing reads it back as UTC. Without
// this, a branch generated in any non-UTC zone parses to an instant hours away from
// when it was actually created, which silently corrupts any age comparison.
func FormatSyncBranchName(prefix, scope string, timestamp time.Time, commitSHA string) string {
	return fmt.Sprintf(
		"%s-%s-%s-%s",
		prefix,
		scope,
		timestamp.UTC().Format("20060102-150405"),
		commitSHA,
	)
}

// FormatBranchScope reduces a repository reference to the branch-name scope segment.
//
// It takes the repository name from an "owner/repo" pair and replaces any
// character outside the branch pattern's charset, so repos containing dots (a
// legal GitHub name) still produce parseable branch names.
func FormatBranchScope(repo string) string {
	name := repo
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}

	name = invalidScopeCharsPattern.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")

	if name == "" {
		return "unknown"
	}

	return name
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
