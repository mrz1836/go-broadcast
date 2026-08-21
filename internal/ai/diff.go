package ai

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// DiffTruncator truncates diffs to stay within token limits while preserving context.
// Thread-safe (stateless after construction).
type DiffTruncator struct {
	// MaxChars is the maximum total characters (config-driven; default: 16000).
	MaxChars int

	// MaxLinesPerFile is the maximum lines per file section (config-driven; default: 150).
	MaxLinesPerFile int
}

// NewDiffTruncator creates a truncator with the given configuration.
func NewDiffTruncator(cfg *Config) *DiffTruncator {
	return &DiffTruncator{
		MaxChars:        cfg.DiffMaxChars,
		MaxLinesPerFile: cfg.DiffMaxLinesPerFile,
	}
}

// Truncate intelligently truncates a diff while preserving file headers and context.
func (t *DiffTruncator) Truncate(fullDiff string) string {
	if len(fullDiff) <= t.MaxChars {
		return fullDiff
	}

	// Strategy:
	// 1. Parse diff into per-file sections (split on "diff --git" or "--- a/")
	// 2. For each file, keep header + first N lines of changes
	// 3. Add "[...truncated]" marker when content is cut
	// 4. Stop adding files when approaching MaxChars

	var result strings.Builder
	sections := splitDiffIntoSections(fullDiff)

	for _, section := range sections {
		truncatedSection := t.truncateSection(section)

		// Check if adding this section would exceed limit
		if result.Len()+len(truncatedSection) > t.MaxChars {
			result.WriteString("\n\n[additional files truncated for brevity]\n")
			break
		}

		result.WriteString(truncatedSection)
	}

	return result.String()
}

// truncateSection truncates a single file's diff section.
func (t *DiffTruncator) truncateSection(section string) string {
	lines := strings.Split(section, "\n")

	if len(lines) <= t.MaxLinesPerFile {
		return section
	}

	// Keep header lines (file paths) + first N content lines
	headerLines := findHeaderEndIndex(lines)

	keepLines := headerLines + t.MaxLinesPerFile
	if keepLines > len(lines) {
		return section
	}

	truncated := strings.Join(lines[:keepLines], "\n")
	return truncated + "\n[...truncated]\n"
}

// findHeaderEndIndex finds the index where the actual diff content starts.
// This preserves the file path headers and @@ markers.
func findHeaderEndIndex(lines []string) int {
	for i, line := range lines {
		if strings.HasPrefix(line, "@@") {
			return i + 1 // Include the @@ line
		}
	}
	// Fallback: keep first 4 lines as header
	if len(lines) > 4 {
		return 4
	}
	return len(lines)
}

// splitDiffIntoSections splits a unified diff into per-file sections.
func splitDiffIntoSections(diff string) []string {
	// Split on "diff --git" which marks file boundaries
	parts := strings.Split(diff, "diff --git")

	sections := make([]string, 0, len(parts))
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		if i > 0 {
			part = "diff --git" + part
		}
		sections = append(sections, part)
	}

	return sections
}

// TruncatePrioritized truncates a diff while guaranteeing that small, high-signal
// configuration files (env, go.mod, TOML, YAML) are included before large generated
// files. This prevents the historical failure mode where big action/workflow diffs
// consumed the entire character budget and pushed version-bearing .env files out of
// the diff entirely - while they still appeared in the file list - causing the model
// to hallucinate their version numbers.
//
// It returns the truncated diff plus the repository-relative paths of any files whose
// diff was omitted or trimmed, so the caller can tell the model to describe those
// files generically instead of inventing details.
func (t *DiffTruncator) TruncatePrioritized(fullDiff string) (truncatedDiff string, omittedFiles []string) {
	if len(fullDiff) <= t.MaxChars {
		return fullDiff, nil
	}

	sections := splitDiffIntoSections(fullDiff)
	order := sortSectionsByPriority(sections)

	// Reassemble in priority order, tracking which original sections were included
	// so the final diff can be emitted in the diff's natural order.
	included := make([]bool, len(sections))
	trimmed := make([]bool, len(sections))
	total := 0

	for _, i := range order {
		section := t.truncateSection(sections[i])
		if section != sections[i] {
			trimmed[i] = true
		}
		if total+len(section) > t.MaxChars && total > 0 {
			// Budget exhausted; remaining (lower-priority) sections are omitted.
			continue
		}
		included[i] = true
		total += len(section)
	}

	var result strings.Builder
	for i, section := range sections {
		if !included[i] {
			if p := sectionFilePath(section); p != "" {
				omittedFiles = append(omittedFiles, p)
			}
			continue
		}
		result.WriteString(t.truncateSection(section))
		if trimmed[i] {
			if p := sectionFilePath(section); p != "" {
				omittedFiles = append(omittedFiles, p)
			}
		}
	}

	if len(omittedFiles) > 0 {
		result.WriteString("\n\n[some files' diffs were omitted or trimmed to fit - see file list]\n")
	}

	return result.String(), omittedFiles
}

// TruncateWithSummary truncates the diff and returns a summary of what was truncated.
func (t *DiffTruncator) TruncateWithSummary(fullDiff string) (truncatedDiff string, truncated bool, fileCount int) {
	sections := splitDiffIntoSections(fullDiff)
	fileCount = len(sections)

	if len(fullDiff) <= t.MaxChars {
		return fullDiff, false, fileCount
	}

	var result strings.Builder
	includedFiles := 0

	for _, section := range sections {
		truncatedSection := t.truncateSection(section)

		// Check if adding this section would exceed limit
		if result.Len()+len(truncatedSection) > t.MaxChars {
			result.WriteString("\n\n[additional files truncated for brevity]\n")
			break
		}

		result.WriteString(truncatedSection)
		includedFiles++
	}

	return result.String(), includedFiles < fileCount || len(fullDiff) > t.MaxChars, fileCount
}

// CountDiffLines counts actual added and removed lines between old and new content.
// Returns (linesAdded, linesRemoved) - the actual number of changed lines, not total file lines.
func CountDiffLines(oldContent, newContent string) (added, removed int) {
	oldLines := difflib.SplitLines(oldContent)
	newLines := difflib.SplitLines(newContent)

	// Use difflib to get the actual operations
	matcher := difflib.NewMatcher(oldLines, newLines)
	opcodes := matcher.GetOpCodes()

	for _, op := range opcodes {
		switch op.Tag {
		case 'r': // Replace
			removed += op.I2 - op.I1
			added += op.J2 - op.J1
		case 'd': // Delete
			removed += op.I2 - op.I1
		case 'i': // Insert
			added += op.J2 - op.J1
		}
		// 'e' (equal) - no changes
	}

	return added, removed
}

// GenerateUnifiedDiff creates a unified diff from old/new content.
// Output format matches git diff for AI model compatibility.
// This is used to generate synthetic diffs in dry-run mode when no git repo is available.
func GenerateUnifiedDiff(filename, oldContent, newContent string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(oldContent),
		B:        difflib.SplitLines(newContent),
		FromFile: "a/" + filename,
		ToFile:   "b/" + filename,
		Context:  3,
	}
	result, _ := difflib.GetUnifiedDiffString(diff)
	return result
}

// GenerateNewFileDiff creates a unified diff for a new file (all lines added).
func GenerateNewFileDiff(filename, content string) string {
	diff := difflib.UnifiedDiff{
		A:        nil, // No original content
		B:        difflib.SplitLines(content),
		FromFile: "/dev/null",
		ToFile:   "b/" + filename,
		Context:  3,
	}
	result, _ := difflib.GetUnifiedDiffString(diff)
	return result
}

// GenerateDeletedFileDiff creates a unified diff for a deleted file (all lines removed).
func GenerateDeletedFileDiff(filename, originalContent string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(originalContent),
		B:        nil, // No new content
		FromFile: "a/" + filename,
		ToFile:   "/dev/null",
		Context:  3,
	}
	result, _ := difflib.GetUnifiedDiffString(diff)
	return result
}
