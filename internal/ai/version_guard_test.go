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

func TestApplyVerifiedChanges_RemovesPastedNonVersionBlock(t *testing.T) {
	// Reproduces the duplicated "What Changed" list: the model pasted the verified
	// bullets from the prompt verbatim. They carry no version token, so the
	// version-based filter alone kept them and the block appeared twice.
	cs := &Changeset{KeyChanges: []KeyChange{
		{File: ".github/env/20-workflows.env", Key: "DEPENDABOT_AUTO_MERGE_PATCH_SAME_OWNER", New: "true", Kind: ChangeAdded},
		{File: ".github/env/20-workflows.env", Key: "OLD_FLAG", Old: "1", Kind: ChangeRemoved},
	}}
	body := "## What Changed\n\n" +
		"* Added `DEPENDABOT_AUTO_MERGE_PATCH_SAME_OWNER` = `true` (`.github/env/20-workflows.env`)\n" +
		"* Removed `OLD_FLAG` (`.github/env/20-workflows.env`)\n" +
		"* Introduced same-owner dependency detection logic\n\n" +
		"## Impact\n\n* Low"

	got := ApplyVerifiedChanges(body, cs)

	assert.Equal(t, 1, strings.Count(got, "DEPENDABOT_AUTO_MERGE_PATCH_SAME_OWNER"), "verified bullet must appear exactly once")
	assert.Equal(t, 1, strings.Count(got, "OLD_FLAG"), "removed bullet must appear exactly once")
	assert.Contains(t, got, "Introduced same-owner dependency detection logic")
}

func TestApplyVerifiedChanges_KeepsBulletWithSubstringKey(t *testing.T) {
	// "COUNT" must not match inside "ACCOUNT_ID".
	cs := &Changeset{KeyChanges: []KeyChange{
		{File: "a.env", Key: "COUNT", New: "0", Kind: ChangeAdded},
	}}
	body := "## What Changed\n\n* Default ACCOUNT_ID now set to 0 for tests\n\n## Impact\n\n* Low"
	got := ApplyVerifiedChanges(body, cs)
	assert.Contains(t, got, "Default ACCOUNT_ID now set to 0 for tests")
}

func TestApplyVerifiedChanges_ThenTighten_SingleTightList(t *testing.T) {
	// Reproduces the loose What Changed list: the verified block was inserted above
	// the model's bullets with a blank line between them, so GitHub rendered every
	// bullet in the section as a spaced-out paragraph.
	cs := &Changeset{KeyChanges: []KeyChange{
		{File: "a.env", Key: "MAGE_X_VERSION", Old: "v1.28.0", New: "v1.29.0", Kind: ChangeModified},
		{File: "a.env", Key: "OSV_SCANNER_VERSION", Old: "v2.6.0", New: "v2.7.0", Kind: ChangeModified},
	}}
	body := "## What Changed\n\n" +
		"* Replaced `interface{}` with `any` in documentation examples\n" +
		"* Updated benchmark example to use `for b.Loop()`\n\n" +
		"## Why It Was Necessary\n\n* Keep repos aligned"

	got := tightenLists(ApplyVerifiedChanges(body, cs))

	want := "## What Changed\n\n" +
		"* `MAGE_X_VERSION`: `v1.28.0` → `v1.29.0` (`a.env`)\n" +
		"* `OSV_SCANNER_VERSION`: `v2.6.0` → `v2.7.0` (`a.env`)\n" +
		"* Replaced `interface{}` with `any` in documentation examples\n" +
		"* Updated benchmark example to use `for b.Loop()`\n\n" +
		"## Why It Was Necessary\n\n* Keep repos aligned"
	assert.Equal(t, want, got)
}

func TestTightenLists(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "removes blank lines between bullets",
			in:   "## A\n\n* one\n\n* two\n\n\n- three\n\n## B\n\n* x",
			want: "## A\n\n* one\n* two\n- three\n\n## B\n\n* x",
		},
		{
			name: "removes blank lines between ordered and nested items",
			in:   "1. one\n\n2. two\n\n   * nested\n\n3. three",
			want: "1. one\n2. two\n   * nested\n3. three",
		},
		{
			name: "keeps blank line before a paragraph",
			in:   "* one\n\nA closing paragraph.\n\n* two",
			want: "* one\n\nA closing paragraph.\n\n* two",
		},
		{
			name: "does not treat bold text as a bullet",
			in:   "* one\n\n**Note**: something",
			want: "* one\n\n**Note**: something",
		},
		{
			name: "leaves fenced code blocks untouched",
			in:   "* one\n\n```\n- a\n\n- b\n```\n\n* two",
			want: "* one\n\n```\n- a\n\n- b\n```\n\n* two",
		},
		{
			name: "empty body",
			in:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tightenLists(tt.in))
		})
	}
}

func TestContainsIdentifier(t *testing.T) {
	assert.True(t, containsIdentifier("* Added `COUNT` = `0`", "COUNT"))
	assert.True(t, containsIdentifier("COUNT", "COUNT"))
	assert.False(t, containsIdentifier("ACCOUNT_ID", "COUNT"))
	assert.False(t, containsIdentifier("COUNT_MAX", "COUNT"))
	assert.True(t, containsIdentifier("ACCOUNT then COUNT", "COUNT"))
	assert.False(t, containsIdentifier("anything", ""))
}
