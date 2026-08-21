//go:build ailive

// Package ai live tests exercise the PR body generator against a REAL AI model
// through the production Genkit provider.
//
// These are intentionally excluded from the default build and CI via the "ailive"
// build tag. They require a configured API key in the environment and make real
// (billable) network calls. Run them locally to validate real-model behavior:
//
//	go test -tags ailive -run TestLive ./internal/ai -v
//
// The code path under test is exactly the production path: NewGenkitProvider ->
// BuildPRPrompt (with verified changes injected) -> real model -> ValidatePRBody
// -> GuardVersions -> EnsureVerifiedChanges.
package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// hallucinatedTokens are the fabricated version numbers from the real-world failure.
// After the deterministic guard runs, NONE of these may appear because they exist
// nowhere in the diff.
var hallucinatedTokens = []string{"v1.13.1", "v1.14.0", "1.25.1", "1.25.2", "v1.13", "1.13.1"}

// requiredTokens are the correct, machine-verified values that MUST appear.
var requiredTokens = []string{"1.26.3", "1.27.0", "2.8.5"}

// syncVersionBumpDiff mirrors a real sync PR: two large action files (which force
// truncation) plus small env files that carry the version bumps. The env changes
// are the source of truth for version accuracy.
const syncVersionBumpDiff = `diff --git a/.github/actions/setup-benchstat/action.yml b/.github/actions/setup-benchstat/action.yml
--- a/.github/actions/setup-benchstat/action.yml
+++ b/.github/actions/setup-benchstat/action.yml
@@ -10,6 +10,24 @@ inputs:
   benchstat-version:
     description: "Pinned benchstat version for Go 1.25.x runners"
     required: false
     default: "v0.0.0-20231201000000-abcdef123456"
+  benchstat-version-latest:
+    description: "Newer benchstat version for Go >= 1.26 runners"
+    required: false
+    default: "v0.0.0-20240601000000-fedcba654321"
+  benchstat-latest-min-go:
+    description: "Minimum Go minor version required to use benchstat-version-latest"
+    required: false
+    default: "1.26"
+runs:
+  using: composite
+  steps:
+    - name: Select benchstat version based on Go toolchain
+      shell: bash
+      run: |
+        go_minor="$(go env GOVERSION | sed -E 's/go1\.([0-9]+).*/\1/')"
+        if [ "${go_minor}" -ge "26" ]; then
+          echo "benchstat=${{ inputs.benchstat-version-latest }}" >> "$GITHUB_OUTPUT"
+        else
+          echo "benchstat=${{ inputs.benchstat-version }}" >> "$GITHUB_OUTPUT"
+        fi
diff --git a/.github/actions/validate-test-results/action.yml b/.github/actions/validate-test-results/action.yml
--- a/.github/actions/validate-test-results/action.yml
+++ b/.github/actions/validate-test-results/action.yml
@@ -20,6 +20,40 @@ outputs:
   total-packages:
     description: "Total number of packages tested"
     value: ${{ steps.parse.outputs.total-packages }}
+  passed-packages:
+    description: "Number of packages that passed"
+    value: ${{ steps.parse.outputs.passed-packages }}
+  failed-packages:
+    description: "Number of packages that failed"
+    value: ${{ steps.parse.outputs.failed-packages }}
+  skipped-packages:
+    description: "Number of packages skipped"
+    value: ${{ steps.parse.outputs.skipped-packages }}
+  total-tests:
+    description: "Total number of tests executed"
+    value: ${{ steps.parse.outputs.total-tests }}
+  passed-tests:
+    description: "Number of tests that passed"
+    value: ${{ steps.parse.outputs.passed-tests }}
+  failed-tests:
+    description: "Number of tests that failed"
+    value: ${{ steps.parse.outputs.failed-tests }}
+  skipped-tests:
+    description: "Number of tests skipped"
+    value: ${{ steps.parse.outputs.skipped-tests }}
diff --git a/.github/env/00-core.env b/.github/env/00-core.env
--- a/.github/env/00-core.env
+++ b/.github/env/00-core.env
@@ -30,3 +30,3 @@
 # Govulncheck-specific Go version for vulnerability scanning
-GOVULNCHECK_GO_VERSION=1.26.6
+GOVULNCHECK_GO_VERSION=1.27.0
diff --git a/.github/env/10-mage-x.env b/.github/env/10-mage-x.env
--- a/.github/env/10-mage-x.env
+++ b/.github/env/10-mage-x.env
@@ -35,7 +35,10 @@
 # Benchstat configuration
+MAGE_X_BENCHSTAT_VERSION_LATEST=true
+MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.26
 # MAGE-X version
-MAGE_X_VERSION=v1.26.1
+MAGE_X_VERSION=v1.26.3
diff --git a/.github/env/10-pre-commit.env b/.github/env/10-pre-commit.env
--- a/.github/env/10-pre-commit.env
+++ b/.github/env/10-pre-commit.env
@@ -3 +3 @@
-GO_PRE_COMMIT_VERSION=v2.8.4
+GO_PRE_COMMIT_VERSION=v2.8.5
diff --git a/.github/env/10-security.env b/.github/env/10-security.env
--- a/.github/env/10-security.env
+++ b/.github/env/10-security.env
@@ -4,3 +4 @@
 # Security scanning tool versions
-GO_SEC_VERSION=v2.21.0
-GO_SEC_MIN_VERSION=1.24
`

func liveConfig(t *testing.T) *Config {
	t.Helper()
	cfg := LoadConfig()
	if cfg.APIKey == "" {
		t.Skip("no AI API key in environment; set GO_BROADCAST_AI_API_KEY to run live tests")
	}
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel(cfg.Provider)
	}
	return cfg
}

func newLivePRGenerator(t *testing.T, cfg *Config) *PRBodyGenerator {
	t.Helper()
	logger := logrus.NewEntry(logrus.New())
	provider, err := NewGenkitProvider(context.Background(), cfg, logger)
	if err != nil {
		t.Skipf("could not initialize %s provider: %v", cfg.Provider, err)
	}
	if !provider.IsAvailable() {
		t.Skipf("%s provider not available", cfg.Provider)
	}
	return NewPRBodyGenerator(provider, nil, NewDiffTruncator(cfg), DefaultRetryConfig(), cfg, "", 90*time.Second, logger)
}

// TestLivePRBody_VersionAccuracy is the headline test: feed the real sync diff to a
// real model and assert the final PR body is factually correct about versions,
// repeatedly (model output is non-deterministic; "flawless" must hold every time).
func TestLivePRBody_VersionAccuracy(t *testing.T) {
	cfg := liveConfig(t)
	gen := newLivePRGenerator(t, cfg)

	tr := NewDiffTruncator(cfg)
	truncated, omitted := tr.TruncatePrioritized(syncVersionBumpDiff)

	prCtx := &PRContext{
		SourceRepo:   "owner/source-repo",
		TargetRepo:   "owner/target-repo",
		CommitSHA:    "deadbeef",
		DiffSummary:  truncated,
		FullDiff:     syncVersionBumpDiff,
		OmittedFiles: omitted,
		ChangedFiles: []FileChange{
			{Path: ".github/actions/setup-benchstat/action.yml", ChangeType: "modified", LinesAdded: 18, LinesRemoved: 0},
			{Path: ".github/actions/validate-test-results/action.yml", ChangeType: "modified", LinesAdded: 34, LinesRemoved: 0},
			{Path: ".github/env/00-core.env", ChangeType: "modified", LinesAdded: 1, LinesRemoved: 1},
			{Path: ".github/env/10-mage-x.env", ChangeType: "modified", LinesAdded: 3, LinesRemoved: 1},
			{Path: ".github/env/10-pre-commit.env", ChangeType: "modified", LinesAdded: 1, LinesRemoved: 1},
			{Path: ".github/env/10-security.env", ChangeType: "modified", LinesAdded: 0, LinesRemoved: 2},
		},
	}

	const attempts = 2
	for i := 0; i < attempts; i++ {
		body, genErr := gen.GenerateBody(context.Background(), prCtx)
		if genErr != nil {
			t.Fatalf("attempt %d: GenerateBody failed: %v", i+1, genErr)
		}
		t.Logf("=== attempt %d body ===\n%s\n", i+1, body)

		for _, bad := range hallucinatedTokens {
			if strings.Contains(body, bad) {
				t.Errorf("attempt %d: body contains hallucinated version %q", i+1, bad)
			}
		}
		for _, want := range requiredTokens {
			if !strings.Contains(body, want) {
				t.Errorf("attempt %d: body missing required version %q", i+1, want)
			}
		}
		for _, key := range []string{"MAGE_X_VERSION", "GOVULNCHECK_GO_VERSION", "GO_PRE_COMMIT_VERSION"} {
			if !strings.Contains(body, key) {
				t.Errorf("attempt %d: body missing key %q", i+1, key)
			}
		}
		if !strings.Contains(body, "## What Changed") {
			t.Errorf("attempt %d: body missing '## What Changed' section", i+1)
		}
	}
}

// TestLivePRBody_NoInventedKeys guards the other direction: the model must not claim
// a change to GO_VERSION, which is NOT in the diff (only GOVULNCHECK_GO_VERSION is).
func TestLivePRBody_NoInventedKeys(t *testing.T) {
	cfg := liveConfig(t)
	gen := newLivePRGenerator(t, cfg)

	diff := "diff --git a/.github/env/00-core.env b/.github/env/00-core.env\n" +
		"--- a/.github/env/00-core.env\n+++ b/.github/env/00-core.env\n" +
		"@@ -30,3 +30,3 @@\n # Primary Go version\n GO_VERSION=1.25.1\n" +
		"-GOVULNCHECK_GO_VERSION=1.26.6\n+GOVULNCHECK_GO_VERSION=1.27.0\n"

	prCtx := &PRContext{
		SourceRepo:  "owner/source-repo",
		TargetRepo:  "owner/target-repo",
		DiffSummary: diff,
		FullDiff:    diff,
		ChangedFiles: []FileChange{
			{Path: ".github/env/00-core.env", ChangeType: "modified", LinesAdded: 1, LinesRemoved: 1},
		},
	}

	body, genErr := gen.GenerateBody(context.Background(), prCtx)
	if genErr != nil {
		t.Fatalf("GenerateBody failed: %v", genErr)
	}
	t.Logf("=== body ===\n%s\n", body)

	// GO_VERSION=1.25.1 is unchanged context; the body must not invent a bump to 1.25.2.
	if strings.Contains(body, "1.25.2") {
		t.Errorf("body invented a GO_VERSION bump to 1.25.2")
	}
	if !strings.Contains(body, "1.27.0") {
		t.Errorf("body missing the real GOVULNCHECK_GO_VERSION bump to 1.27.0")
	}
}

func newLiveCommitGenerator(t *testing.T, cfg *Config) *CommitMessageGenerator {
	t.Helper()
	logger := logrus.NewEntry(logrus.New())
	provider, err := NewGenkitProvider(context.Background(), cfg, logger)
	if err != nil {
		t.Skipf("could not initialize %s provider: %v", cfg.Provider, err)
	}
	if !provider.IsAvailable() {
		t.Skipf("%s provider not available", cfg.Provider)
	}
	return NewCommitMessageGenerator(provider, nil, NewDiffTruncator(cfg), DefaultRetryConfig(), cfg, 90*time.Second, logger)
}

// TestLiveCommitAccuracy verifies a real-model commit subject is a valid conventional
// commit and never cites a hallucinated version.
func TestLiveCommitAccuracy(t *testing.T) {
	cfg := liveConfig(t)
	gen := newLiveCommitGenerator(t, cfg)

	tr := NewDiffTruncator(cfg)
	truncated, _ := tr.TruncatePrioritized(syncVersionBumpDiff)

	commitCtx := &CommitContext{
		SourceRepo:   "owner/source-repo",
		TargetRepo:   "owner/target-repo",
		DiffSummary:  truncated,
		FullDiff:     syncVersionBumpDiff,
		ChangedFiles: []FileChange{{Path: ".github/env/10-mage-x.env", ChangeType: "modified"}},
	}

	const attempts = 2
	for i := 0; i < attempts; i++ {
		msg, genErr := gen.GenerateMessage(context.Background(), commitCtx)
		if genErr != nil {
			t.Fatalf("attempt %d: GenerateMessage failed: %v", i+1, genErr)
		}
		t.Logf("attempt %d commit subject: %q", i+1, msg)

		if !strings.HasPrefix(msg, "sync:") && !strings.HasPrefix(msg, "sync(") {
			t.Errorf("attempt %d: not a sync-prefixed conventional commit: %q", i+1, msg)
		}
		if len(msg) > 72 {
			t.Errorf("attempt %d: subject too long (%d): %q", i+1, len(msg), msg)
		}
		for _, bad := range hallucinatedTokens {
			if strings.Contains(msg, bad) {
				t.Errorf("attempt %d: subject contains hallucinated version %q: %q", i+1, bad, msg)
			}
		}
	}
}
