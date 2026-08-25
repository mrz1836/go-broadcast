package gh

import (
	"strings"
	"unicode"
)

// summarizeChecks categorizes a slice of check runs into a CheckStatusSummary.
// total is reported separately because GitHub's check-runs endpoint returns its
// own total_count, which the normal fetch path preserves verbatim; the filtered
// path (WithoutChecks) recomputes it from the retained checks.
func summarizeChecks(checks []CheckRun, total int) *CheckStatusSummary {
	summary := &CheckStatusSummary{
		Total:  total,
		Checks: checks,
	}

	for _, check := range checks {
		switch check.Status {
		case "completed":
			summary.Completed++
			switch check.Conclusion {
			case "success", "neutral":
				summary.Passed++
			case "skipped":
				summary.Skipped++
			case "failure", "canceled", "timed_out", "action_required":
				summary.Failed++
			}
		case "queued", "in_progress":
			summary.Running++
		}
	}

	return summary
}

// WithoutChecks returns a new summary with any check whose name matches one of
// the provided ignore patterns removed, along with the names of the checks that
// were actually filtered out. Counts (Passed/Failed/Skipped/Running/Total) are
// recomputed from the retained checks so the CI gate treats ignored checks as if
// they were never configured. The receiver is not mutated.
//
// When ignore is empty (or nothing matches), the returned summary is equivalent
// to the receiver and the ignored-names slice is empty.
func (s *CheckStatusSummary) WithoutChecks(ignore []string) (*CheckStatusSummary, []string) {
	if len(ignore) == 0 || len(s.Checks) == 0 {
		return s, nil
	}

	kept := make([]CheckRun, 0, len(s.Checks))
	var ignored []string
	for _, check := range s.Checks {
		if matchesIgnoredCheck(check.Name, ignore) {
			ignored = append(ignored, check.Name)
			continue
		}
		kept = append(kept, check)
	}

	if len(ignored) == 0 {
		return s, nil
	}

	return summarizeChecks(kept, len(kept)), ignored
}

// matchesIgnoredCheck reports whether a check name matches any of the ignore
// patterns. Matching is case-insensitive and ignores a leading run of
// non-alphanumeric characters (e.g. an emoji + space prefix like "🤖 "), so a
// user can configure "Dependabot Auto-merge" without copying the emoji that
// GitHub prepends to the rendered check name.
func matchesIgnoredCheck(name string, patterns []string) bool {
	normalized := normalizeCheckName(name)
	if normalized == "" {
		return false
	}
	for _, pattern := range patterns {
		if normalizeCheckName(pattern) == normalized {
			return true
		}
	}
	return false
}

// normalizeCheckName lower-cases a check name and strips any leading run of
// non-alphanumeric runes (emoji, symbols, whitespace) so cosmetic prefixes do
// not affect matching.
func normalizeCheckName(s string) string {
	s = strings.TrimLeftFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.ToLower(strings.TrimSpace(s))
}
