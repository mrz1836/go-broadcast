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

	return body
}
