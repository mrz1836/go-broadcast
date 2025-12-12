package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiffTruncator(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}

	truncator := NewDiffTruncator(cfg)

	require.NotNil(t, truncator)
	assert.Equal(t, 4000, truncator.MaxChars)
	assert.Equal(t, 50, truncator.MaxLinesPerFile)
}

func TestDiffTruncator_Truncate_SmallDiff(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	smallDiff := `diff --git a/file.go b/file.go
index abc123..def456 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main

+import "fmt"
 func main() {}`

	result := truncator.Truncate(smallDiff)

	assert.Equal(t, smallDiff, result, "small diff should pass through unchanged")
}

func TestDiffTruncator_Truncate_LargeDiff(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        200,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Create a diff that exceeds MaxChars
	var largeDiff strings.Builder
	largeDiff.WriteString(`diff --git a/file.go b/file.go
index abc123..def456 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,100 @@
`)
	for i := 0; i < 50; i++ {
		largeDiff.WriteString("+line " + string(rune('A'+i%26)) + "\n")
	}

	result := truncator.Truncate(largeDiff.String())

	assert.Less(t, len(result), len(largeDiff.String()), "result should be shorter than input")
	assert.Contains(t, result, "[", "should contain truncation marker")
}

func TestDiffTruncator_Truncate_MultipleFiles(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        500,
		DiffMaxLinesPerFile: 10,
	}
	truncator := NewDiffTruncator(cfg)

	multiFileDiff := `diff --git a/file1.go b/file1.go
index abc123..def456 100644
--- a/file1.go
+++ b/file1.go
@@ -1,3 +1,4 @@
+line1
diff --git a/file2.go b/file2.go
index 111222..333444 100644
--- a/file2.go
+++ b/file2.go
@@ -1,3 +1,4 @@
+line2
diff --git a/file3.go b/file3.go
index 555666..777888 100644
--- a/file3.go
+++ b/file3.go
@@ -1,3 +1,4 @@
+line3`

	result := truncator.Truncate(multiFileDiff)

	// Should contain at least the first file
	assert.Contains(t, result, "file1.go")
}

func TestDiffTruncator_Truncate_PreservesHeaders(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        1000,
		DiffMaxLinesPerFile: 5,
	}
	truncator := NewDiffTruncator(cfg)

	diff := `diff --git a/file.go b/file.go
index abc123..def456 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,20 @@
+line1
+line2
+line3
+line4
+line5
+line6
+line7
+line8
+line9
+line10`

	result := truncator.Truncate(diff)

	// Headers should be preserved
	assert.Contains(t, result, "diff --git")
	assert.Contains(t, result, "--- a/file.go")
	assert.Contains(t, result, "+++ b/file.go")
	assert.Contains(t, result, "@@")
}

func TestDiffTruncator_Truncate_AddsTruncationMarker(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        100,
		DiffMaxLinesPerFile: 3,
	}
	truncator := NewDiffTruncator(cfg)

	diff := `diff --git a/file.go b/file.go
index abc123..def456 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,20 @@
+line1
+line2
+line3
+line4
+line5
+line6
+line7
+line8
+line9
+line10`

	result := truncator.Truncate(diff)

	// Should have some kind of truncation indicator
	hasTruncationMarker := strings.Contains(result, "[...truncated]") ||
		strings.Contains(result, "[additional files truncated")
	assert.True(t, hasTruncationMarker, "should contain truncation marker")
}

func TestDiffTruncator_TruncateWithSummary(t *testing.T) {
	tests := []struct {
		name              string
		maxChars          int
		maxLinesPerFile   int
		diff              string
		wantTruncated     bool
		wantFileCount     int
		wantResultSmaller bool
	}{
		{
			name:            "small diff not truncated",
			maxChars:        1000,
			maxLinesPerFile: 50,
			diff: `diff --git a/file.go b/file.go
@@ -1,1 +1,2 @@
+line1`,
			wantTruncated: false,
			wantFileCount: 1,
		},
		{
			name:            "large diff truncated",
			maxChars:        100,
			maxLinesPerFile: 50,
			diff: `diff --git a/file.go b/file.go
@@ -1,1 +1,50 @@
+line1
+line2
+line3
+line4
+line5
+line6
+line7
+line8
+line9
+line10`,
			wantTruncated:     true,
			wantFileCount:     1,
			wantResultSmaller: true,
		},
		{
			name:            "multiple files counted",
			maxChars:        2000,
			maxLinesPerFile: 50,
			diff: `diff --git a/file1.go b/file1.go
@@ -1,1 +1,2 @@
+line1
diff --git a/file2.go b/file2.go
@@ -1,1 +1,2 @@
+line2
diff --git a/file3.go b/file3.go
@@ -1,1 +1,2 @@
+line3`,
			wantTruncated: false,
			wantFileCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DiffMaxChars:        tt.maxChars,
				DiffMaxLinesPerFile: tt.maxLinesPerFile,
			}
			truncator := NewDiffTruncator(cfg)

			result, truncated, fileCount := truncator.TruncateWithSummary(tt.diff)

			assert.Equal(t, tt.wantTruncated, truncated)
			assert.Equal(t, tt.wantFileCount, fileCount)

			if tt.wantResultSmaller {
				assert.Less(t, len(result), len(tt.diff))
			}
		})
	}
}

func TestSplitDiffIntoSections(t *testing.T) {
	tests := []struct {
		name         string
		diff         string
		wantSections int
	}{
		{
			name:         "empty diff",
			diff:         "",
			wantSections: 0,
		},
		{
			name: "single file",
			diff: `diff --git a/file.go b/file.go
@@ -1,1 +1,2 @@
+line`,
			wantSections: 1,
		},
		{
			name: "two files",
			diff: `diff --git a/file1.go b/file1.go
@@ -1,1 +1,2 @@
+line1
diff --git a/file2.go b/file2.go
@@ -1,1 +1,2 @@
+line2`,
			wantSections: 2,
		},
		{
			name: "three files",
			diff: `diff --git a/file1.go b/file1.go
@@ -1,1 +1,2 @@
+line1
diff --git a/file2.go b/file2.go
@@ -1,1 +1,2 @@
+line2
diff --git a/file3.go b/file3.go
@@ -1,1 +1,2 @@
+line3`,
			wantSections: 3,
		},
		{
			name:         "whitespace only",
			diff:         "   \n\t  \n  ",
			wantSections: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections := splitDiffIntoSections(tt.diff)
			assert.Len(t, sections, tt.wantSections)

			// Each section should contain "diff --git" (except empty result)
			for _, section := range sections {
				assert.Contains(t, section, "diff --git")
			}
		})
	}
}

func TestFindHeaderEndIndex(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		wantIndex int
	}{
		{
			name: "finds @@ marker",
			lines: []string{
				"diff --git a/file.go b/file.go",
				"index abc..def 100644",
				"--- a/file.go",
				"+++ b/file.go",
				"@@ -1,3 +1,4 @@",
				"+new line",
			},
			wantIndex: 5, // Index after @@ line
		},
		{
			name: "no @@ marker uses fallback",
			lines: []string{
				"line1",
				"line2",
				"line3",
				"line4",
				"line5",
				"line6",
			},
			wantIndex: 4, // Fallback: first 4 lines
		},
		{
			name: "short file without @@ marker",
			lines: []string{
				"line1",
				"line2",
			},
			wantIndex: 2, // Returns length for short files
		},
		{
			name:      "empty lines",
			lines:     []string{},
			wantIndex: 0,
		},
		{
			name: "@@ at beginning",
			lines: []string{
				"@@ -1,3 +1,4 @@",
				"+new line",
			},
			wantIndex: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := findHeaderEndIndex(tt.lines)
			assert.Equal(t, tt.wantIndex, index)
		})
	}
}

func TestDiffTruncator_TruncateSection(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        10000,
		DiffMaxLinesPerFile: 5,
	}
	truncator := NewDiffTruncator(cfg)

	tests := []struct {
		name           string
		section        string
		wantTruncated  bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "short section unchanged",
			section: `diff --git a/file.go b/file.go
@@ -1,1 +1,2 @@
+line`,
			wantTruncated: false,
		},
		{
			name: "long section truncated",
			section: `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,15 @@
+line1
+line2
+line3
+line4
+line5
+line6
+line7
+line8
+line9
+line10`,
			wantTruncated:  true,
			wantContains:   []string{"diff --git", "@@", "[...truncated]"},
			wantNotContain: []string{"+line10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncator.truncateSection(tt.section)

			if tt.wantTruncated {
				assert.NotEqual(t, tt.section, result)
				assert.Contains(t, result, "[...truncated]")
			} else {
				assert.Equal(t, tt.section, result)
			}

			for _, s := range tt.wantContains {
				assert.Contains(t, result, s)
			}
			for _, s := range tt.wantNotContain {
				assert.NotContains(t, result, s)
			}
		})
	}
}

func TestDiffTruncator_EmptyDiff(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	result := truncator.Truncate("")
	assert.Empty(t, result)

	result, truncated, fileCount := truncator.TruncateWithSummary("")
	assert.Empty(t, result)
	assert.False(t, truncated)
	assert.Equal(t, 0, fileCount)
}

// Edge case tests for binary content, CRLF, and malformed diffs

func TestDiffTruncator_BinaryContent(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Simulated binary file diff (git marks these as "Binary files differ")
	binaryDiff := `diff --git a/image.png b/image.png
index abc123..def456 100644
Binary files a/image.png and b/image.png differ`

	result := truncator.Truncate(binaryDiff)
	assert.Equal(t, binaryDiff, result, "binary diff marker should pass through unchanged")
	assert.Contains(t, result, "Binary files")
}

func TestDiffTruncator_CRLFLineEndings(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Diff with CRLF line endings (Windows-style)
	crlfDiff := "diff --git a/file.txt b/file.txt\r\n" +
		"index abc..def 100644\r\n" +
		"--- a/file.txt\r\n" +
		"+++ b/file.txt\r\n" +
		"@@ -1,3 +1,4 @@\r\n" +
		"+new line\r\n"

	result := truncator.Truncate(crlfDiff)
	assert.Contains(t, result, "diff --git")
	assert.Contains(t, result, "+new line")
}

func TestDiffTruncator_MixedLineEndings(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Diff with mixed LF and CRLF
	mixedDiff := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\r\n" +
		"+++ b/file.txt\n" +
		"@@ -1,3 +1,4 @@\r\n" +
		"+line1\n" +
		"+line2\r\n"

	result := truncator.Truncate(mixedDiff)
	assert.Contains(t, result, "diff --git")
	assert.Contains(t, result, "+line1")
}

func TestDiffTruncator_MalformedDiff_NoHunkHeaders(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Malformed diff without @@ hunk headers
	malformedDiff := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
+line without hunk header`

	result := truncator.Truncate(malformedDiff)
	// Should not panic, should handle gracefully
	assert.Contains(t, result, "diff --git")
	assert.Contains(t, result, "+line without hunk header")
}

func TestDiffTruncator_MalformedDiff_NoFilePaths(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Malformed diff without proper file paths
	malformedDiff := `diff --git
@@ -1 +1 @@
-old
+new`

	result := truncator.Truncate(malformedDiff)
	// Should handle gracefully without panic
	assert.NotEmpty(t, result)
}

//nolint:gosmopolitan // intentional unicode test data
func TestDiffTruncator_UnicodeContent(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Diff with unicode content
	unicodeDiff := `diff --git a/文件.go b/文件.go
index abc..def 100644
--- a/文件.go
+++ b/文件.go
@@ -1,3 +1,4 @@
+日本語テキスト
+Ελληνικά
+🎉 Emoji content 🚀`

	result := truncator.Truncate(unicodeDiff)
	assert.Contains(t, result, "文件.go")
	assert.Contains(t, result, "日本語テキスト")
	assert.Contains(t, result, "🎉 Emoji content 🚀")
}

func TestDiffTruncator_VeryLongLines(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        500,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Diff with very long lines (minified JS, etc.)
	longLine := strings.Repeat("x", 1000)
	longLineDiff := `diff --git a/bundle.js b/bundle.js
@@ -1 +1 @@
-old
+` + longLine

	result := truncator.Truncate(longLineDiff)
	// Should truncate based on char limit
	assert.Less(t, len(result), len(longLineDiff))
}

func TestDiffTruncator_SingleCharDiff(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	singleCharDiff := "x"

	result := truncator.Truncate(singleCharDiff)
	assert.Equal(t, singleCharDiff, result)
}

func TestDiffTruncator_NullBytes(t *testing.T) {
	cfg := &Config{
		DiffMaxChars:        4000,
		DiffMaxLinesPerFile: 50,
	}
	truncator := NewDiffTruncator(cfg)

	// Diff with null bytes (binary content detection in git)
	diffWithNull := "diff --git a/file b/file\n" +
		"@@ -1 +1 @@\n" +
		"+content\x00with\x00nulls\n"

	result := truncator.Truncate(diffWithNull)
	// Should handle without panic
	assert.Contains(t, result, "diff --git")
}

func TestGenerateUnifiedDiff_Basic(t *testing.T) {
	oldContent := "line1\nline2\nline3\n"
	newContent := "line1\nmodified\nline3\n"

	result := GenerateUnifiedDiff("test.txt", oldContent, newContent)

	assert.Contains(t, result, "a/test.txt")
	assert.Contains(t, result, "b/test.txt")
	assert.Contains(t, result, "-line2")
	assert.Contains(t, result, "+modified")
}

func TestGenerateUnifiedDiff_EmptyOldContent(t *testing.T) {
	result := GenerateUnifiedDiff("new.txt", "", "new content\n")

	assert.Contains(t, result, "a/new.txt")
	assert.Contains(t, result, "b/new.txt")
	assert.Contains(t, result, "+new content")
}

func TestGenerateUnifiedDiff_EmptyNewContent(t *testing.T) {
	result := GenerateUnifiedDiff("deleted.txt", "old content\n", "")

	assert.Contains(t, result, "a/deleted.txt")
	assert.Contains(t, result, "b/deleted.txt")
	assert.Contains(t, result, "-old content")
}

func TestGenerateUnifiedDiff_BothEmpty(t *testing.T) {
	result := GenerateUnifiedDiff("empty.txt", "", "")

	// Should produce empty or minimal diff
	assert.NotContains(t, result, "+")
	assert.NotContains(t, result, "-")
}

//nolint:gosmopolitan // intentional unicode test data
func TestGenerateUnifiedDiff_UnicodeFilename(t *testing.T) {
	result := GenerateUnifiedDiff("文档.txt", "旧内容\n", "新内容\n")

	assert.Contains(t, result, "文档.txt")
	assert.Contains(t, result, "-旧内容")
	assert.Contains(t, result, "+新内容")
}

func TestGenerateNewFileDiff(t *testing.T) {
	result := GenerateNewFileDiff("new.go", "package main\n\nfunc main() {}\n")

	assert.Contains(t, result, "/dev/null")
	assert.Contains(t, result, "b/new.go")
	assert.Contains(t, result, "+package main")
	assert.Contains(t, result, "+func main() {}")
}

func TestGenerateNewFileDiff_EmptyContent(t *testing.T) {
	result := GenerateNewFileDiff("empty.txt", "")

	assert.Contains(t, result, "/dev/null")
	assert.Contains(t, result, "b/empty.txt")
}

func TestGenerateDeletedFileDiff(t *testing.T) {
	result := GenerateDeletedFileDiff("old.go", "package old\n\nfunc old() {}\n")

	assert.Contains(t, result, "a/old.go")
	assert.Contains(t, result, "/dev/null")
	assert.Contains(t, result, "-package old")
	assert.Contains(t, result, "-func old() {}")
}

func TestGenerateDeletedFileDiff_EmptyContent(t *testing.T) {
	result := GenerateDeletedFileDiff("empty.txt", "")

	assert.Contains(t, result, "a/empty.txt")
	assert.Contains(t, result, "/dev/null")
}
