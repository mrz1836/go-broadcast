package ai

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ChangeKind classifies how a key/value pair changed in a diff.
type ChangeKind int

const (
	// ChangeModified indicates a key's value changed from Old to New.
	ChangeModified ChangeKind = iota
	// ChangeAdded indicates a new key was introduced.
	ChangeAdded
	// ChangeRemoved indicates an existing key was deleted.
	ChangeRemoved
)

// maxVerifiedBullets caps how many verified-change bullets are rendered into the
// prompt/body so a very large sync does not produce an unbounded list.
const maxVerifiedBullets = 40

// maxValueDisplayLen truncates long values (e.g. multi-line JSON) when displayed
// in a verified-change bullet.
const maxValueDisplayLen = 80

// versionTokenRe matches semantic-version-like tokens such as "v1.26.3", "1.27.0",
// "1.26", and "v2.8.5-rc.1". It intentionally accepts an optional leading "v" and a
// 1-3 segment numeric core with an optional pre-release/build suffix.
var versionTokenRe = regexp.MustCompile(`(?i)\bv?\d+\.\d+(?:\.\d+){0,2}(?:-[0-9A-Za-z.]+)?`)

// kvEqualsRe matches "KEY=value" / "KEY = value" style assignments (env, Make,
// Dockerfile ARG/ENV, TOML, INI).
var kvEqualsRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_.-]*)\s*=\s*(.*)$`)

// kvColonRe matches "key: value" style assignments (YAML, some config formats).
// A space after the colon is required to avoid matching URLs like "https://...".
var kvColonRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_.-]*):\s+(.*)$`)

// KeyChange is a single deterministic key/value change extracted from a diff.
// These are machine-extracted facts - never AI-generated - and are treated as
// the authoritative source of truth for version numbers in PR descriptions.
type KeyChange struct {
	// File is the repository-relative path where the change occurred.
	File string
	// Key is the configuration key that changed (e.g., "MAGE_X_VERSION").
	Key string
	// Old is the previous value (empty for ChangeAdded).
	Old string
	// New is the new value (empty for ChangeRemoved).
	New string
	// Kind classifies the change.
	Kind ChangeKind
}

// Bullet renders a KeyChange as a single Markdown list item with exact values.
func (kc KeyChange) Bullet() string {
	loc := ""
	if kc.File != "" {
		loc = fmt.Sprintf(" (`%s`)", kc.File)
	}
	switch kc.Kind {
	case ChangeAdded:
		return fmt.Sprintf("* Added `%s` = `%s`%s", kc.Key, truncateValue(kc.New), loc)
	case ChangeRemoved:
		return fmt.Sprintf("* Removed `%s`%s", kc.Key, loc)
	case ChangeModified:
		fallthrough
	default:
		return fmt.Sprintf("* `%s`: `%s` → `%s`%s", kc.Key, truncateValue(kc.Old), truncateValue(kc.New), loc)
	}
}

// IsSignificant reports whether a change is meaningful enough to assert as an
// authoritative "verified change" in a PR description. This filters out incidental
// key/value-shaped lines from structured files (e.g., YAML "using: composite" or a
// shell assignment inside a composite action) while keeping every real version or
// configuration-constant change.
//
// A change qualifies when either its old or new value contains a version token, or
// its key is a configuration-constant style identifier (UPPER_SNAKE_CASE).
func (kc KeyChange) IsSignificant() bool {
	if versionTokenRe.MatchString(kc.Old) || versionTokenRe.MatchString(kc.New) {
		return true
	}
	return isConfigConstantKey(kc.Key)
}

// isConfigConstantKey reports whether k looks like a configuration constant, i.e.
// it contains at least one uppercase letter and consists solely of uppercase
// letters, digits, and underscores (e.g., "MAGE_X_VERSION", "GO_SEC_MIN_VERSION").
func isConfigConstantKey(k string) bool {
	if k == "" {
		return false
	}
	hasUpper := false
	for _, r := range k {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= '0' && r <= '9', r == '_':
			// allowed, but not sufficient on their own
		default:
			return false
		}
	}
	return hasUpper
}

// SignificantChanges returns only the changes worth asserting authoritatively.
func (cs *Changeset) SignificantChanges() []KeyChange {
	if cs == nil {
		return nil
	}
	out := make([]KeyChange, 0, len(cs.KeyChanges))
	for _, kc := range cs.KeyChanges {
		if kc.IsSignificant() {
			out = append(out, kc)
		}
	}
	return out
}

// Changeset is the complete deterministic view of a diff used to keep AI-generated
// PR descriptions factually correct.
type Changeset struct {
	// KeyChanges are the ordered key/value changes discovered across all files.
	KeyChanges []KeyChange
	// VersionTokens is the set of every version-like token that legitimately
	// appears anywhere in the diff (context and changed lines). Used by the guard
	// to detect hallucinated version numbers with high confidence.
	VersionTokens map[string]struct{}
}

// ExtractChangeset parses a unified diff into structured, authoritative facts.
// It is a pure function: deterministic, allocation-bounded, and safe to call on
// every sync. An empty or unparseable diff yields an empty (non-nil) Changeset.
func ExtractChangeset(diff string) *Changeset {
	cs := &Changeset{VersionTokens: map[string]struct{}{}}
	if strings.TrimSpace(diff) == "" {
		return cs
	}

	// Collect every version token from the entire diff (context + changed lines).
	// The guard only flags tokens with zero support here, minimizing false positives.
	for _, tok := range versionTokenRe.FindAllString(diff, -1) {
		cs.VersionTokens[normalizeVersionToken(tok)] = struct{}{}
	}

	seen := make(map[string]struct{})
	for _, section := range splitDiffIntoSections(diff) {
		file := sectionFilePath(section)
		for _, kc := range extractSectionChanges(file, section) {
			dedupeKey := kc.File + "\x00" + kc.Key
			if _, ok := seen[dedupeKey]; ok {
				continue
			}
			seen[dedupeKey] = struct{}{}
			cs.KeyChanges = append(cs.KeyChanges, kc)
		}
	}

	return cs
}

// HasKeyChanges reports whether any structured key/value changes were found.
func (cs *Changeset) HasKeyChanges() bool {
	return cs != nil && len(cs.KeyChanges) > 0
}

// extractSectionChanges derives KeyChanges from a single file's diff section by
// pairing removed and added key/value lines.
func extractSectionChanges(file, section string) []KeyChange {
	type kv struct{ key, val string }
	var removed, added []kv
	removedIdx := map[string]string{}
	addedIdx := map[string]string{}

	for _, line := range strings.Split(section, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			// File header lines - ignore.
			continue
		case strings.HasPrefix(line, "+"):
			if k, v, ok := parseKeyValue(line[1:]); ok {
				added = append(added, kv{k, v})
				addedIdx[k] = v
			}
		case strings.HasPrefix(line, "-"):
			if k, v, ok := parseKeyValue(line[1:]); ok {
				removed = append(removed, kv{k, v})
				removedIdx[k] = v
			}
		}
	}

	var changes []KeyChange
	handled := make(map[string]struct{})

	// Removed lines first: a matching added line means "modified", otherwise "removed".
	for _, r := range removed {
		if _, done := handled[r.key]; done {
			continue
		}
		handled[r.key] = struct{}{}
		if newVal, ok := addedIdx[r.key]; ok {
			if newVal != r.val {
				changes = append(changes, KeyChange{File: file, Key: r.key, Old: r.val, New: newVal, Kind: ChangeModified})
			}
			continue
		}
		changes = append(changes, KeyChange{File: file, Key: r.key, Old: r.val, Kind: ChangeRemoved})
	}

	// Added lines with no matching removed line are brand-new keys.
	for _, a := range added {
		if _, done := handled[a.key]; done {
			continue
		}
		if _, wasRemoved := removedIdx[a.key]; wasRemoved {
			continue
		}
		handled[a.key] = struct{}{}
		changes = append(changes, KeyChange{File: file, Key: a.key, New: a.val, Kind: ChangeAdded})
	}

	return changes
}

// RenderVerifiedChanges renders the authoritative key/value changes as a Markdown
// block for injection into the AI prompt. Returns an empty string when there are
// no structured changes.
func RenderVerifiedChanges(cs *Changeset) string {
	changes := cs.SignificantChanges()
	if len(changes) == 0 {
		return ""
	}
	var b strings.Builder
	for n, kc := range changes {
		if n >= maxVerifiedBullets {
			fmt.Fprintf(&b, "* ...and %d more change(s)\n", len(changes)-n)
			break
		}
		b.WriteString(kc.Bullet())
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// parseKeyValue extracts a key and value from a single (diff-marker-stripped) line.
// It supports "KEY=value" and "key: value" forms and skips blank/comment lines.
func parseKeyValue(s string) (key, val string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "//") {
		return "", "", false
	}
	if m := kvEqualsRe.FindStringSubmatch(s); m != nil {
		return m[1], strings.TrimSpace(m[2]), true
	}
	if m := kvColonRe.FindStringSubmatch(s); m != nil {
		return m[1], strings.TrimSpace(m[2]), true
	}
	return "", "", false
}

// sectionFilePath extracts the repository-relative path for a diff section,
// preferring the "+++ b/" header and falling back to the "diff --git" or
// "--- a/" headers (the latter covers deleted files).
func sectionFilePath(section string) string {
	var fromPath string
	for _, line := range strings.Split(section, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ b/"):
			return strings.TrimSpace(strings.TrimPrefix(line, "+++ b/"))
		case strings.HasPrefix(line, "--- a/"):
			fromPath = strings.TrimSpace(strings.TrimPrefix(line, "--- a/"))
		case strings.HasPrefix(line, "diff --git a/"):
			rest := strings.TrimPrefix(line, "diff --git a/")
			if i := strings.Index(rest, " b/"); i >= 0 {
				return strings.TrimSpace(rest[:i])
			}
		}
	}
	return fromPath
}

// normalizeVersionToken canonicalizes a version token for set membership so that
// "v1.26.3", "V1.26.3", and "1.26.3" all compare equal.
func normalizeVersionToken(tok string) string {
	tok = strings.ToLower(strings.TrimSpace(tok))
	tok = strings.TrimPrefix(tok, "v")
	return tok
}

// truncateValue shortens long values for display while preserving the leading,
// most-informative portion.
func truncateValue(v string) string {
	v = strings.TrimSpace(v)
	// Strip a single layer of surrounding quotes for readability.
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	if len(v) > maxValueDisplayLen {
		return v[:maxValueDisplayLen] + "…"
	}
	return v
}

// versionFilePriority ranks diff sections so that small, high-signal configuration
// files (which hold the version numbers reviewers care about) are never dropped in
// favor of large generated files. Lower rank == higher priority.
func versionFilePriority(path string) int {
	p := strings.ToLower(path)
	switch {
	case strings.HasSuffix(p, ".env") || strings.Contains(p, "/env/"):
		return 0
	case strings.HasSuffix(p, ".mod") || strings.HasSuffix(p, ".sum"):
		return 0
	case strings.HasSuffix(p, ".toml") || strings.HasSuffix(p, ".ini"):
		return 1
	case strings.HasSuffix(p, ".yml") || strings.HasSuffix(p, ".yaml"):
		return 2
	default:
		return 3
	}
}

// sortSectionsByPriority returns section indices ordered so higher-priority
// (smaller, config-like) files come first, then by ascending size. The original
// slice is not modified.
func sortSectionsByPriority(sections []string) []int {
	idx := make([]int, len(sections))
	for i := range sections {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ia, ib := idx[a], idx[b]
		pa := versionFilePriority(sectionFilePath(sections[ia]))
		pb := versionFilePriority(sectionFilePath(sections[ib]))
		if pa != pb {
			return pa < pb
		}
		return len(sections[ia]) < len(sections[ib])
	})
	return idx
}
