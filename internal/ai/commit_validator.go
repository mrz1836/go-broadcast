package ai

import (
	"strings"
)

// maxCommitMessageLength is the maximum allowed length for commit messages.
// Conventional commits recommend 50 chars, but we allow up to 72 for flexibility.
const maxCommitMessageLength = 72

// ValidateCommitMessage validates and cleans AI-generated commit messages.
// Ensures output follows go-broadcast conventional commit standards.
// This function is deterministic and fast - safe to call on every message.
func ValidateCommitMessage(msg string) string {
	// 1. Trim whitespace
	msg = strings.TrimSpace(msg)

	if msg == "" {
		return ""
	}

	// 2. Take only first line (discard multi-line body)
	if idx := strings.Index(msg, "\n"); idx != -1 {
		msg = strings.TrimSpace(msg[:idx])
	}

	// 3. Remove markdown formatting AI might add
	msg = removeMarkdownFormatting(msg)

	// 4. Ensure sync: or sync(scope): prefix
	msg = ensureSyncPrefix(msg)

	// 5. Remove trailing period (conventional commits don't use periods)
	msg = strings.TrimSuffix(msg, ".")

	// 6. Enforce character limit
	if len(msg) > maxCommitMessageLength {
		// Truncate at word boundary if possible, leaving room for "..."
		msg = truncateAtWordBoundary(msg, maxCommitMessageLength-3) + "..."
	}

	return msg
}

// removeMarkdownFormatting removes common markdown formatting that AI might add.
func removeMarkdownFormatting(msg string) string {
	// Remove code blocks
	msg = strings.TrimPrefix(msg, "```")
	msg = strings.TrimSuffix(msg, "```")

	// Remove inline code backticks
	msg = strings.TrimPrefix(msg, "`")
	msg = strings.TrimSuffix(msg, "`")

	// Trim any whitespace introduced
	return strings.TrimSpace(msg)
}

// ensureSyncPrefix ensures the message has a sync: or sync(scope): prefix.
// Converts common prefixes like chore:, feat:, fix: to sync:.
func ensureSyncPrefix(msg string) string {
	// Already has correct prefix
	if strings.HasPrefix(msg, "sync:") || strings.HasPrefix(msg, "sync(") {
		return msg
	}

	// Convert common prefixes to sync
	prefixesToConvert := []string{
		"chore(sync): ",
		"chore: ",
		"feat: ",
		"fix: ",
		"docs: ",
		"refactor: ",
		"test: ",
		"build: ",
		"ci: ",
	}

	for _, prefix := range prefixesToConvert {
		if strings.HasPrefix(msg, prefix) {
			msg = strings.TrimPrefix(msg, prefix)
			return "sync: " + msg
		}
	}

	// No recognized prefix, add sync:
	return "sync: " + msg
}

// truncateAtWordBoundary truncates string at word boundary for clean output.
func truncateAtWordBoundary(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Find last space before maxLen
	lastSpace := strings.LastIndex(s[:maxLen], " ")
	if lastSpace > maxLen/2 {
		return s[:lastSpace]
	}

	// No good word boundary, hard truncate
	return s[:maxLen]
}
