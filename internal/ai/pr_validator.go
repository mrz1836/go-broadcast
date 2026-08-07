package ai

import (
	"strings"
)

// ValidatePRBody validates AI-generated PR body.
// Returns empty string if the response is invalid (looks like commit message).
// Valid PR bodies must have ## headers and multiple lines.
// This function is deterministic and fast - safe to call on every response.
func ValidatePRBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	// Reject single-line responses (commit message format)
	if !strings.Contains(body, "\n") {
		return ""
	}

	// Reject conventional commit format at start (case-insensitive)
	commitPrefixes := []string{"sync:", "sync(", "chore:", "chore(", "feat:", "feat(", "fix:", "fix(", "docs:", "docs("}
	lowerBody := strings.ToLower(body)
	for _, prefix := range commitPrefixes {
		if strings.HasPrefix(lowerBody, prefix) {
			return ""
		}
	}

	// Require at least one ## header (markdown section)
	if !strings.Contains(body, "## ") {
		return ""
	}

	// Reject "I can't see the diff" meta-commentary. When the diff is empty or
	// unavailable, the model sometimes writes an essay ABOUT the missing diff instead
	// of describing the files. That output has ## headers, so it passes the checks
	// above, but it is useless to reviewers. Rejecting it makes the caller fall back
	// to a description built from the file list instead.
	emptyDiffMarkers := []string{
		"diff provided is empty",
		"diff is empty",
		"empty diff",
		"diff content not visible",
		"diff content is not visible",
		"no actual code changes are visible",
		"without viewing the actual diff",
		"without viewing actual diff",
		"cannot be determined without",
		"unable to determine",
		"cannot verify testing",
		"not visible for assessment",
		"truncated content",
	}
	for _, marker := range emptyDiffMarkers {
		if strings.Contains(lowerBody, marker) {
			return ""
		}
	}

	return body
}
