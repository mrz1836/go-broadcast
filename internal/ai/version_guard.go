package ai

import (
	"strings"
)

// GuardVersions removes any line from an AI-generated PR body that contains a
// version-like token which does not appear anywhere in the diff. This is the
// deterministic backstop against hallucinated version numbers: if the model
// writes "v1.13.1" but that token exists nowhere in the actual diff, the claim
// is fabricated and the offending line is dropped.
//
// allowed is the set of normalized version tokens known to be legitimate
// (typically Changeset.VersionTokens, derived from the full untruncated diff).
// The returned violations slice lists the raw hallucinated tokens that were
// found, enabling telemetry across large multi-repo syncs.
//
// GuardVersions is a pure function and never returns an error.
func GuardVersions(body string, allowed map[string]struct{}) (string, []string) {
	if body == "" {
		return "", nil
	}

	var violations []string
	lines := strings.Split(body, "\n")
	kept := make([]string, 0, len(lines))

	for _, line := range lines {
		tokens := versionTokenRe.FindAllString(line, -1)
		lineHasHallucination := false
		for _, tok := range tokens {
			if _, ok := allowed[normalizeVersionToken(tok)]; !ok {
				violations = append(violations, tok)
				lineHasHallucination = true
			}
		}
		if lineHasHallucination {
			// Drop the entire line: a bullet or sentence asserting a version
			// number that is not in the diff is unsalvageable and worse than
			// silence. The correct facts are re-added by EnsureVerifiedChanges.
			continue
		}
		kept = append(kept, line)
	}

	return collapseBlankRuns(strings.Join(kept, "\n")), violations
}

// EnsureVerifiedChanges guarantees that every deterministically-extracted key
// change is represented in the body with correct values. Any change whose key is
// entirely absent (e.g., because GuardVersions dropped a hallucinated line, or the
// model simply omitted it) is appended as an authoritative bullet under the
// "## What Changed" section.
//
// A change is considered "represented" if its key name already appears anywhere in
// the body - the guard has already corrected the numbers on those lines, so we only
// backfill genuinely missing facts and avoid duplicating what the model wrote well.
func EnsureVerifiedChanges(body string, cs *Changeset) string {
	if !cs.HasKeyChanges() {
		return body
	}

	var missing []KeyChange
	for _, kc := range cs.SignificantChanges() {
		if !strings.Contains(body, kc.Key) {
			missing = append(missing, kc)
		}
	}
	if len(missing) == 0 {
		return body
	}

	var block strings.Builder
	for i, kc := range missing {
		if i >= maxVerifiedBullets {
			break
		}
		block.WriteString(kc.Bullet())
		block.WriteString("\n")
	}

	return insertUnderWhatChanged(body, strings.TrimRight(block.String(), "\n"))
}

// insertUnderWhatChanged places block immediately after the "## What Changed"
// heading. If that heading is absent, a new section is appended so the facts are
// never lost.
func insertUnderWhatChanged(body, block string) string {
	if block == "" {
		return body
	}

	const marker = "## What Changed"
	idx := strings.Index(body, marker)
	if idx < 0 {
		trimmed := strings.TrimRight(body, "\n")
		if trimmed == "" {
			return marker + "\n\n" + block
		}
		return trimmed + "\n\n" + marker + "\n\n" + block
	}

	// Find the end of the heading line.
	nl := strings.IndexByte(body[idx:], '\n')
	if nl < 0 {
		return body + "\n\n" + block
	}
	insertAt := idx + nl + 1
	return body[:insertAt] + "\n" + block + "\n" + body[insertAt:]
}

// collapseBlankRuns compresses runs of 3+ blank lines (which can appear after the
// guard drops lines) down to a single blank line, keeping the Markdown tidy.
func collapseBlankRuns(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blankRun := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankRun++
			if blankRun > 1 {
				continue
			}
		} else {
			blankRun = 0
		}
		out = append(out, line)
	}
	return strings.Trim(strings.Join(out, "\n"), "\n")
}
