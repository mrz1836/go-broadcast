package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allowedSet(tokens ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(tokens))
	for _, t := range tokens {
		m[normalizeVersionToken(t)] = struct{}{}
	}
	return m
}

func TestGuardVersions_StripsHallucinations(t *testing.T) {
	body := "## What Changed\n" +
		"\n" +
		"* Updated `MAGE_X_VERSION` from v1.13.1 to v1.14.0\n" + // both hallucinated
		"* Updated `GOVULNCHECK_GO_VERSION` from 1.26.6 to 1.27.0\n" + // both real
		"* Added a new benchstat input\n" // no version token
	allowed := allowedSet("1.26.6", "1.27.0")

	got, violations := GuardVersions(body, allowed)

	assert.NotContains(t, got, "v1.13.1")
	assert.NotContains(t, got, "v1.14.0")
	assert.Contains(t, got, "1.26.6")
	assert.Contains(t, got, "1.27.0")
	assert.Contains(t, got, "Added a new benchstat input")
	assert.ElementsMatch(t, []string{"v1.13.1", "v1.14.0"}, violations)
}

func TestGuardVersions_ExactReportedFailure(t *testing.T) {
	// Reproduces the two Copilot-flagged claims from the real PR.
	body := "## What Changed\n\n" +
		"* updated MAGE_X_VERSION from v1.13.1 to v1.14.0\n" +
		"* changing GO_VERSION from 1.25.1 to 1.25.2\n"
	// The real diff only ever contained these tokens.
	allowed := allowedSet("v1.26.1", "v1.26.3", "1.26.6", "1.27.0")

	got, violations := GuardVersions(body, allowed)

	for _, bad := range []string{"v1.13.1", "v1.14.0", "1.25.1", "1.25.2"} {
		assert.NotContainsf(t, got, bad, "hallucinated token %q must be removed", bad)
	}
	assert.Len(t, violations, 4)
}

func TestGuardVersions_KeepsCleanBody(t *testing.T) {
	body := "## What Changed\n\n* Something without versions\n\n## Impact\n\n* Low risk"
	got, violations := GuardVersions(body, allowedSet())
	assert.Empty(t, violations)
	assert.Contains(t, got, "Something without versions")
	assert.Contains(t, got, "Low risk")
}

func TestGuardVersions_EmptyBody(t *testing.T) {
	got, violations := GuardVersions("", allowedSet("1.0.0"))
	assert.Empty(t, got)
	assert.Empty(t, violations)
}

func TestApplyVerifiedChanges_RemovesRestatementAndInsertsBlock(t *testing.T) {
	// Model duplicated: a version restatement bullet plus a genuine narrative bullet.
	body := "## What Changed\n\n" +
		"* Updated MAGE-X to v1.26.3 (from v1.26.1)\n" +
		"* Added documentation explaining the new setup\n\n" +
		"## Impact\n\n* Low"
	cs := &Changeset{KeyChanges: []KeyChange{
		{File: "a.env", Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified},
	}}

	got := ApplyVerifiedChanges(body, cs)

	// Authoritative block present once; the prose restatement is removed.
	assert.Contains(t, got, "`MAGE_X_VERSION`: `v1.26.1` → `v1.26.3`")
	assert.NotContains(t, got, "Updated MAGE-X to v1.26.3 (from v1.26.1)")
	assert.Equal(t, 1, strings.Count(got, "v1.26.3"), "version must appear exactly once")
	// Genuine narrative bullet (no version token) is preserved.
	assert.Contains(t, got, "Added documentation explaining the new setup")
	// Placed under What Changed, before Impact.
	assert.Less(t, strings.Index(got, "MAGE_X_VERSION"), strings.Index(got, "## Impact"))
}

func TestApplyVerifiedChanges_KeepsBulletWithUncoveredVersion(t *testing.T) {
	// A narrative bullet citing a version NOT in the changeset must survive.
	body := "## What Changed\n\n* Still supports Go 1.25.x runners\n\n## Impact\n\n* Low"
	cs := &Changeset{KeyChanges: []KeyChange{
		{File: "a.env", Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified},
	}}
	got := ApplyVerifiedChanges(body, cs)
	assert.Contains(t, got, "Still supports Go 1.25.x runners")
}

func TestApplyVerifiedChanges_NoChangesNoop(t *testing.T) {
	body := "## What Changed\n\n* nothing"
	assert.Equal(t, body, ApplyVerifiedChanges(body, &Changeset{}))
	assert.Equal(t, body, ApplyVerifiedChanges(body, nil))
}

func TestInsertUnderWhatChanged_NoHeader(t *testing.T) {
	got := insertUnderWhatChanged("## Summary\n\n* item", "* `K`: `1` → `2`")
	assert.Contains(t, got, "## What Changed")
	assert.Contains(t, got, "* `K`: `1` → `2`")
}

func TestCollapseBlankRuns(t *testing.T) {
	got := collapseBlankRuns("a\n\n\n\nb\n\n\nc")
	assert.Equal(t, "a\n\nb\n\nc", got)
}

func TestGuardThenApply_Integration(t *testing.T) {
	// Full deterministic pipeline: hallucinated body + real changeset -> correct body.
	body := "## What Changed\n\n" +
		"* Updated `MAGE_X_VERSION` from v1.13.1 to v1.14.0\n" +
		"* Bumped `GOVULNCHECK_GO_VERSION` from 1.25.1 to 1.25.2\n" +
		"\n## Why It Was Necessary\n\n* Keep repos aligned\n" +
		"\n## Testing Performed\n\n* Validated config\n" +
		"\n## Impact / Risk\n\n* Low risk\n"
	cs := &Changeset{
		KeyChanges: []KeyChange{
			{File: ".github/env/10-mage-x.env", Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified},
			{File: ".github/env/00-core.env", Key: "GOVULNCHECK_GO_VERSION", Old: "1.26.6", New: "1.27.0", Kind: ChangeModified},
		},
		VersionTokens: allowedSet("v1.26.1", "v1.26.3", "1.26.6", "1.27.0"),
	}

	guarded, violations := GuardVersions(body, cs.VersionTokens)
	final := ApplyVerifiedChanges(guarded, cs)

	require.NotEmpty(t, violations)
	// Correct values present:
	assert.Contains(t, final, "v1.26.3")
	assert.Contains(t, final, "1.27.0")
	assert.Contains(t, final, "MAGE_X_VERSION")
	assert.Contains(t, final, "GOVULNCHECK_GO_VERSION")
	// Hallucinations gone:
	for _, bad := range []string{"v1.13.1", "v1.14.0", "1.25.1", "1.25.2"} {
		assert.NotContains(t, final, bad)
	}
	// Narrative sections preserved:
	assert.Contains(t, final, "Keep repos aligned")
	assert.Contains(t, final, "Low risk")
}
