package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// realDiff mirrors the actual sync PR: two large action files plus small env files
// that carry the version bumps. The env changes are the source of truth.
const realDiff = `diff --git a/.github/env/10-mage-x.env b/.github/env/10-mage-x.env
--- a/.github/env/10-mage-x.env
+++ b/.github/env/10-mage-x.env
@@ -38,3 +38,3 @@
 # MAGE-X version
-MAGE_X_VERSION=v1.26.1
+MAGE_X_VERSION=v1.26.3
@@ -40,0 +41,2 @@
+MAGE_X_BENCHSTAT_VERSION_LATEST=true
+MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.26
diff --git a/.github/env/00-core.env b/.github/env/00-core.env
--- a/.github/env/00-core.env
+++ b/.github/env/00-core.env
@@ -32 +32 @@
-GOVULNCHECK_GO_VERSION=1.26.6
+GOVULNCHECK_GO_VERSION=1.27.0
diff --git a/.github/env/10-security.env b/.github/env/10-security.env
--- a/.github/env/10-security.env
+++ b/.github/env/10-security.env
@@ -5,2 +4,0 @@
-GO_SEC_VERSION=v2.21.0
-GO_SEC_MIN_VERSION=1.24
diff --git a/.github/env/10-pre-commit.env b/.github/env/10-pre-commit.env
--- a/.github/env/10-pre-commit.env
+++ b/.github/env/10-pre-commit.env
@@ -3 +3 @@
-GO_PRE_COMMIT_VERSION=v2.8.4
+GO_PRE_COMMIT_VERSION=v2.8.5
`

// hallucinatedBody is what a naive model produced (the reported failure): correct
// benchstat details, but fabricated MAGE_X_VERSION and GO_VERSION numbers.
const hallucinatedBody = `## What Changed

* Updated ` + "`MAGE_X_VERSION`" + ` from v1.13.1 to v1.14.0
* Updated ` + "`.github/env/00-core.env`" + ` changing ` + "`GO_VERSION`" + ` from 1.25.1 to 1.25.2
* Added ` + "`MAGE_X_BENCHSTAT_VERSION_LATEST`" + ` and ` + "`MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO`" + `

## Why It Was Necessary

* Keep the target repository aligned with its source

## Testing Performed

* Validated sync configuration and file mappings

## Impact / Risk

* Low risk: scoped to configuration files
`

func TestGenerateBody_RepairsHallucinatedVersions(t *testing.T) {
	provider := NewSuccessMock(hallucinatedBody)
	gen := NewPRBodyGenerator(provider, nil, nil, nil, nil, "", 0, nil)

	prCtx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		CommitSHA:    "abc1234",
		DiffSummary:  realDiff,
		FullDiff:     realDiff,
		ChangedFiles: []FileChange{{Path: ".github/env/10-mage-x.env", ChangeType: "modified", LinesAdded: 3, LinesRemoved: 1}},
	}

	body, err := gen.GenerateBody(context.Background(), prCtx)
	require.NoError(t, err)
	require.NotEmpty(t, body)

	// Hallucinated numbers must be gone.
	for _, bad := range []string{"v1.13.1", "v1.14.0", "1.25.1", "1.25.2"} {
		assert.NotContainsf(t, body, bad, "hallucinated %q must not survive", bad)
	}

	// Correct, machine-verified numbers must be present.
	assert.Contains(t, body, "v1.26.3") // MAGE_X_VERSION new
	assert.Contains(t, body, "1.27.0")  // GOVULNCHECK_GO_VERSION new
	assert.Contains(t, body, "v2.8.5")  // GO_PRE_COMMIT_VERSION new

	// Every real changed key must be represented (backfilled if the model omitted it).
	for _, key := range []string{"MAGE_X_VERSION", "GOVULNCHECK_GO_VERSION", "GO_SEC_VERSION", "GO_PRE_COMMIT_VERSION"} {
		assert.Containsf(t, body, key, "expected key %q to be represented", key)
	}

	// Structure intact.
	assert.Contains(t, body, "## What Changed")
	assert.Contains(t, body, "## Impact / Risk")
}

func TestGenerateBody_CleanResponseUnchangedFacts(t *testing.T) {
	// A model that already used the correct values should pass through with those
	// values intact and no duplication.
	clean := `## What Changed

* ` + "`MAGE_X_VERSION`" + `: v1.26.1 → v1.26.3

## Why It Was Necessary

* Alignment

## Testing Performed

* Validated config

## Impact / Risk

* Low
`
	provider := NewSuccessMock(clean)
	gen := NewPRBodyGenerator(provider, nil, nil, nil, nil, "", 0, nil)
	prCtx := &PRContext{
		SourceRepo: "owner/source", TargetRepo: "owner/target",
		FullDiff:    "--- a/.github/env/10-mage-x.env\n+++ b/.github/env/10-mage-x.env\n@@ -1 +1 @@\n-MAGE_X_VERSION=v1.26.1\n+MAGE_X_VERSION=v1.26.3\n",
		DiffSummary: "x",
	}

	body, err := gen.GenerateBody(context.Background(), prCtx)
	require.NoError(t, err)
	assert.Contains(t, body, "v1.26.3")
	assert.Equal(t, 1, countOccurrences(body, "MAGE_X_VERSION"))
}

func countOccurrences(s, sub string) int {
	count := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			count++
		}
	}
	return count
}
