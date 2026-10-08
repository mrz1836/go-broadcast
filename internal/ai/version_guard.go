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

// ApplyVerifiedChanges makes the machine-extracted config/version changes the single
// source of truth in "What Changed".
//
// The model frequently produces a DUPLICATE of the same bumps - once as a pasted
// verified list and again as prose ("Updated MAGE-X to v1.26.4 (from v1.26.1)").
// Prompt instructions alone do not reliably prevent this, so we fix it
// deterministically: remove any list item whose version tokens are all already
// covered by the extracted changeset (a pure restatement), then insert the single
// authoritative verified list at the top of the section. Bullets that add genuine
// context - no version tokens, or a version outside the changeset - are preserved.
func ApplyVerifiedChanges(body string, cs *Changeset) string {
	block := RenderVerifiedChanges(cs)
	if block == "" {
		return body
	}
	body = dropRestatedBullets(body, blockVersionTokens(cs), cs.SignificantChanges())
	body = insertUnderWhatChanged(body, block)
	return collapseBlankRuns(body)
}

// blockVersionTokens returns the normalized version tokens covered by the verified
// changes block (from the old and new values of each significant change).
func blockVersionTokens(cs *Changeset) map[string]struct{} {
	toks := make(map[string]struct{})
	for _, kc := range cs.SignificantChanges() {
		for _, t := range versionTokenRe.FindAllString(kc.Old+"\n"+kc.New, -1) {
			toks[normalizeVersionToken(t)] = struct{}{}
		}
	}
	return toks
}

// dropRestatedBullets removes list items in the "## What Changed" section that
// merely restate what the authoritative verified list will show: either all of the
// bullet's version tokens are covered by the block, or the bullet names a verified
// key together with its value (e.g. the model pasting "Added `COUNT` = `0`" from the
// prompt, which has no version token to catch). Narrative bullets and bullets with
// a version outside the block are kept, and every other section is left untouched.
func dropRestatedBullets(body string, covered map[string]struct{}, changes []KeyChange) string {
	if len(covered) == 0 && len(changes) == 0 {
		return body
	}
	const marker = "## What Changed"
	start := strings.Index(body, marker)
	if start < 0 {
		return body
	}
	end := len(body)
	if i := strings.Index(body[start+len(marker):], "\n## "); i >= 0 {
		end = start + len(marker) + i
	}

	lines := strings.Split(body[start:end], "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
			if restatesKeyChange(line, changes) {
				continue // pasted or reworded copy of a verified key/value change
			}
			if toks := versionTokenRe.FindAllString(line, -1); len(toks) > 0 && len(covered) > 0 {
				allCovered := true
				for _, t := range toks {
					if _, ok := covered[normalizeVersionToken(t)]; !ok {
						allCovered = false
						break
					}
				}
				if allCovered {
					continue // pure restatement of already-covered versions
				}
			}
		}
		kept = append(kept, line)
	}
	return body[:start] + strings.Join(kept, "\n") + body[end:]
}

// restatesKeyChange reports whether a bullet names one of the verified keys along
// with that change's value (the new value for added/modified keys, or a "remove"
// verb for removed keys), meaning it duplicates an authoritative bullet.
func restatesKeyChange(line string, changes []KeyChange) bool {
	lower := strings.ToLower(line)
	for _, kc := range changes {
		if !containsIdentifier(line, kc.Key) {
			continue
		}
		switch kc.Kind {
		case ChangeRemoved:
			if strings.Contains(lower, "remov") {
				return true
			}
		case ChangeAdded, ChangeModified:
			val := strings.TrimSuffix(truncateValue(kc.New), "…")
			if val == "" {
				val = kc.New
			}
			if val != "" && strings.Contains(line, val) {
				return true
			}
		}
	}
	return false
}

// containsIdentifier reports whether ident appears in s as a whole identifier, i.e.
// not embedded in a longer name ("COUNT" must not match "ACCOUNT_ID").
func containsIdentifier(s, ident string) bool {
	if ident == "" {
		return false
	}
	for from := 0; ; {
		i := strings.Index(s[from:], ident)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(ident)
		if (start == 0 || !isIdentByte(s[start-1])) && (end == len(s) || !isIdentByte(s[end])) {
			return true
		}
		from = start + 1
	}
}

// isIdentByte reports whether b can be part of a config key identifier.
func isIdentByte(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z'
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
