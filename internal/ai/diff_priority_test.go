package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bigSection builds a large diff section for the given path so truncation is forced.
func bigSection(path string, lines int) string {
	var b strings.Builder
	b.WriteString("diff --git a/" + path + " b/" + path + "\n")
	b.WriteString("--- a/" + path + "\n")
	b.WriteString("+++ b/" + path + "\n")
	b.WriteString("@@ -1," + itoa(lines) + " +1," + itoa(lines) + " @@\n")
	for i := 0; i < lines; i++ {
		// Plain prose lines (no "key: value" / "key=value") so they are NOT parsed
		// as key changes - this section only exists to consume the char budget.
		b.WriteString("+        echo \"consume budget line " + itoa(i) + " padding padding\"\n")
	}
	return b.String()
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

func TestTruncatePrioritized_KeepsVersionFilesDropsBigFiles(t *testing.T) {
	// A large workflow file that would, under naive first-come truncation, consume
	// the whole budget and push the tiny env file out of the diff.
	big := bigSection(".github/workflows/fortress-test-matrix.yml", 400)
	envDiff := "diff --git a/.github/env/10-mage-x.env b/.github/env/10-mage-x.env\n" +
		"--- a/.github/env/10-mage-x.env\n+++ b/.github/env/10-mage-x.env\n" +
		"@@ -38,3 +38,3 @@\n-MAGE_X_VERSION=v1.26.1\n+MAGE_X_VERSION=v1.26.3\n"

	// Natural (git) order: big workflow first, then env.
	full := big + envDiff
	require.Greater(t, len(full), 2000)

	tr := &DiffTruncator{MaxChars: 1500, MaxLinesPerFile: 50}
	truncated, omitted := tr.TruncatePrioritized(full)

	// The version-bearing env file MUST survive.
	assert.Contains(t, truncated, "MAGE_X_VERSION=v1.26.3")
	// The big workflow file should be reported as omitted.
	assert.Contains(t, omitted, ".github/workflows/fortress-test-matrix.yml")
	// Extractor over the FULL diff still sees the change regardless of truncation.
	cs := ExtractChangeset(full)
	require.Len(t, cs.KeyChanges, 1)
	assert.Equal(t, "MAGE_X_VERSION", cs.KeyChanges[0].Key)
}

func TestTruncatePrioritized_UnderBudgetReturnsAll(t *testing.T) {
	full := "diff --git a/a.env b/a.env\n--- a/a.env\n+++ b/a.env\n@@ -1 +1 @@\n-A=1\n+A=2\n"
	tr := &DiffTruncator{MaxChars: 100000, MaxLinesPerFile: 150}
	truncated, omitted := tr.TruncatePrioritized(full)
	assert.Equal(t, full, truncated)
	assert.Empty(t, omitted)
}

func TestSortSectionsByPriority(t *testing.T) {
	sections := []string{
		"diff --git a/big.yml b/big.yml\n+++ b/big.yml\n" + strings.Repeat("x\n", 100),
		"diff --git a/small.env b/small.env\n+++ b/small.env\nA=1\n",
		"diff --git a/go.mod b/go.mod\n+++ b/go.mod\nx\n",
	}
	order := sortSectionsByPriority(sections)
	// env (rank 0) and go.mod (rank 0) must come before the yaml (rank 2).
	first := sectionFilePath(sections[order[0]])
	assert.True(t, first == "small.env" || first == "go.mod", "expected a rank-0 config file first, got %q", first)
	last := sectionFilePath(sections[order[len(order)-1]])
	assert.Equal(t, "big.yml", last)
}
