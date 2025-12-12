package ai

import (
	"strings"
)

// DiffTruncator truncates diffs to stay within token limits while preserving context.
// Thread-safe (stateless after construction).
type DiffTruncator struct {
	// MaxChars is the maximum total characters (default: 4000).
	MaxChars int

	// MaxLinesPerFile is the maximum lines per file section (default: 50).
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
