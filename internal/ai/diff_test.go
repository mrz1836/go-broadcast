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
