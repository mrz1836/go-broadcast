package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeterministicCommitSubject(t *testing.T) {
	tests := []struct {
		name string
		cs   *Changeset
		want string
	}{
		{
			name: "single modified version",
			cs:   &Changeset{KeyChanges: []KeyChange{{Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified}}},
			want: "sync: bump MAGE_X_VERSION to v1.26.3",
		},
		{
			name: "single added",
			cs:   &Changeset{KeyChanges: []KeyChange{{Key: "GO_NEW_FLAG", New: "true", Kind: ChangeAdded}}},
			want: "sync: add GO_NEW_FLAG",
		},
		{
			name: "single removed",
			cs:   &Changeset{KeyChanges: []KeyChange{{Key: "GO_SEC_VERSION", Old: "v2.0.0", Kind: ChangeRemoved}}},
			want: "sync: remove GO_SEC_VERSION",
		},
		{
			name: "multiple significant",
			cs: &Changeset{KeyChanges: []KeyChange{
				{Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified},
				{Key: "GO_PRE_COMMIT_VERSION", Old: "v2.8.4", New: "v2.8.5", Kind: ChangeModified},
			}},
			want: "sync: update 2 config versions",
		},
		{
			name: "no significant changes",
			cs:   &Changeset{KeyChanges: []KeyChange{{Key: "using", New: "composite", Kind: ChangeAdded}}},
			want: "",
		},
		{
			name: "modified with very long value keeps key only",
			cs: &Changeset{KeyChanges: []KeyChange{{
				Key:  "BENCHSTAT_PIN_VERSION",
				Old:  "v0.0.0-20231201000000-abcdef123456",
				New:  "v0.0.0-20240601000000-fedcba654321000000000000000000000000",
				Kind: ChangeModified,
			}}},
			want: "sync: bump BENCHSTAT_PIN_VERSION",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeterministicCommitSubject(tt.cs)
			assert.Equal(t, tt.want, got)
			if got != "" {
				assert.LessOrEqual(t, len(got), maxCommitMessageLength)
			}
		})
	}
}

func TestGuardCommitSubject(t *testing.T) {
	cs := &Changeset{
		KeyChanges:    []KeyChange{{Key: "MAGE_X_VERSION", Old: "v1.26.1", New: "v1.26.3", Kind: ChangeModified}},
		VersionTokens: allowedSet("v1.26.1", "v1.26.3"),
	}

	t.Run("hallucinated version replaced with verified subject", func(t *testing.T) {
		got, hallucinated := GuardCommitSubject("sync: bump MAGE_X_VERSION to v9.9.9", cs)
		assert.True(t, hallucinated)
		assert.Equal(t, "sync: bump MAGE_X_VERSION to v1.26.3", got)
	})

	t.Run("correct version passes through", func(t *testing.T) {
		in := "sync: bump MAGE_X_VERSION to v1.26.3"
		got, hallucinated := GuardCommitSubject(in, cs)
		assert.False(t, hallucinated)
		assert.Equal(t, in, got)
	})

	t.Run("no version tokens passes through", func(t *testing.T) {
		in := "sync: update workflow configuration"
		got, hallucinated := GuardCommitSubject(in, cs)
		assert.False(t, hallucinated)
		assert.Equal(t, in, got)
	})

	t.Run("nil changeset is a no-op", func(t *testing.T) {
		got, hallucinated := GuardCommitSubject("sync: bump X to v9.9.9", nil)
		assert.False(t, hallucinated)
		assert.Equal(t, "sync: bump X to v9.9.9", got)
	})
}

func TestGenerateMessage_RepairsHallucinatedCommitSubject(t *testing.T) {
	diff := "diff --git a/.github/env/10-mage-x.env b/.github/env/10-mage-x.env\n" +
		"--- a/.github/env/10-mage-x.env\n+++ b/.github/env/10-mage-x.env\n" +
		"@@ -38 +38 @@\n-MAGE_X_VERSION=v1.26.1\n+MAGE_X_VERSION=v1.26.3\n"

	// Model invents a wrong version in the subject.
	provider := NewSuccessMock("sync: bump MAGE_X_VERSION to v1.14.0")
	gen := NewCommitMessageGenerator(provider, nil, nil, nil, nil, 0, nil)

	commitCtx := &CommitContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		DiffSummary:  diff,
		FullDiff:     diff,
		ChangedFiles: []FileChange{{Path: ".github/env/10-mage-x.env", ChangeType: "modified"}},
	}

	msg, err := gen.GenerateMessage(context.Background(), commitCtx)
	require.NoError(t, err)
	assert.Equal(t, "sync: bump MAGE_X_VERSION to v1.26.3", msg)
	assert.NotContains(t, msg, "1.14.0")
}

func TestGenerateMessage_FallbackUsesDeterministicSubject(t *testing.T) {
	diff := "--- a/.github/env/00-core.env\n+++ b/.github/env/00-core.env\n" +
		"@@ -32 +32 @@\n-GOVULNCHECK_GO_VERSION=1.26.6\n+GOVULNCHECK_GO_VERSION=1.27.0\n"

	// No provider -> fallback path. It should still be specific, not generic.
	gen := NewCommitMessageGenerator(NewUnavailableMock(), nil, nil, nil, nil, 0, nil)
	commitCtx := &CommitContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		FullDiff:     diff,
		ChangedFiles: []FileChange{{Path: ".github/env/00-core.env", ChangeType: "modified"}},
	}

	msg, err := gen.GenerateMessage(context.Background(), commitCtx)
	require.ErrorIs(t, err, ErrFallbackUsed)
	assert.Equal(t, "sync: bump GOVULNCHECK_GO_VERSION to 1.27.0", msg)
}
