package sync

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger returns a logger that discards output, for guard unit tests.
func testLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// scopeWith builds a ResolvedScope with the given group and repo counts (the
// only fields the guard reads). The Repos slice is populated so Summary/Prompt
// have something to list.
func scopeWith(groupCount, repoCount int) ResolvedScope {
	repos := make([]string, repoCount)
	for i := range repos {
		repos[i] = "org/repo"
	}
	groups := make([]string, groupCount)
	for i := range groups {
		groups[i] = "group"
	}
	return ResolvedScope{
		Groups:     groups,
		Repos:      repos,
		GroupCount: groupCount,
		RepoCount:  repoCount,
	}
}

func TestResolvedScope_GuardTriggered(t *testing.T) {
	tests := []struct {
		name       string
		groupCount int
		repoCount  int
		want       bool
	}{
		{name: "single group single repo passes", groupCount: 1, repoCount: 1, want: false},
		{name: "single group five repos passes", groupCount: 1, repoCount: 5, want: false},
		{name: "single group six repos trips", groupCount: 1, repoCount: 6, want: true},
		{name: "two groups trips", groupCount: 2, repoCount: 2, want: true},
		{name: "two groups within repo limit still trips on group", groupCount: 2, repoCount: 4, want: true},
		{name: "zero scope passes", groupCount: 0, repoCount: 0, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, scopeWith(tt.groupCount, tt.repoCount).GuardTriggered())
		})
	}
}

func TestDecideScopeGuard(t *testing.T) {
	tests := []struct {
		name         string
		scope        ResolvedScope
		confirmScope *int
		interactive  bool
		wantProceed  bool
		wantPrompt   bool
		wantExpected int
	}{
		{
			name:        "not triggered proceeds without confirmation",
			scope:       scopeWith(1, 3),
			wantProceed: true,
		},
		{
			name:         "multi-group non-interactive no flag refuses",
			scope:        scopeWith(3, 3),
			interactive:  false,
			wantProceed:  false,
			wantExpected: 3,
		},
		{
			name:         "single-group over five repos non-interactive no flag refuses",
			scope:        scopeWith(1, 6),
			interactive:  false,
			wantProceed:  false,
			wantExpected: 6,
		},
		{
			name:         "matching confirm-scope proceeds",
			scope:        scopeWith(3, 3),
			confirmScope: intPtr(3),
			wantProceed:  true,
			wantExpected: 3,
		},
		{
			name:         "mismatched confirm-scope refuses",
			scope:        scopeWith(3, 3),
			confirmScope: intPtr(2),
			wantProceed:  false,
			wantExpected: 3,
		},
		{
			name:         "interactive no flag needs prompt",
			scope:        scopeWith(3, 3),
			interactive:  true,
			wantProceed:  false,
			wantPrompt:   true,
			wantExpected: 3,
		},
		{
			name:         "confirm-scope wins over interactive",
			scope:        scopeWith(3, 3),
			confirmScope: intPtr(3),
			interactive:  true,
			wantProceed:  true,
			wantExpected: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := decideScopeGuard(tt.scope, tt.confirmScope, tt.interactive)
			assert.Equal(t, tt.wantProceed, d.Proceed, "Proceed")
			assert.Equal(t, tt.wantPrompt, d.NeedPrompt, "NeedPrompt")
			if tt.wantExpected != 0 {
				assert.Equal(t, tt.wantExpected, d.ExpectedN, "ExpectedN")
			}
			if !tt.wantProceed && !tt.wantPrompt {
				assert.NotEmpty(t, d.Reason, "refusal should carry a reason")
			}
		})
	}
}

// TestDecideScopeGuard_SingleGroupOverFiveReposTokenIsRepoCount proves AC-6: when
// the guard trips on the repo threshold inside a single group, the token is the
// repo count — --confirm-scope=1 (the group count) fails closed and
// --confirm-scope=<repoCount> proceeds.
func TestDecideScopeGuard_SingleGroupOverFiveReposTokenIsRepoCount(t *testing.T) {
	scope := scopeWith(1, 6)

	// --confirm-scope=1 (the group count) must fail closed.
	d := decideScopeGuard(scope, intPtr(1), false)
	assert.False(t, d.Proceed, "single-group trip must not accept --confirm-scope=1")
	assert.NotEmpty(t, d.Reason)

	// --confirm-scope=6 (the repo count) proceeds.
	d = decideScopeGuard(scope, intPtr(6), false)
	assert.True(t, d.Proceed, "single-group trip must accept --confirm-scope=<repoCount>")
}

// fakeConfirmer is an injectable ScopeConfirmer for exercising the interactive
// branch without a real TTY.
type fakeConfirmer struct {
	interactive bool
	reply       int
	err         error
	calls       int
}

func (f *fakeConfirmer) Interactive() bool { return f.interactive }

func (f *fakeConfirmer) Prompt(_ ResolvedScope) (int, error) {
	f.calls++
	return f.reply, f.err
}

func TestResolvedScope_GuardTriggered_BlastRadiusConstants(t *testing.T) {
	// The thresholds are the named constants (guards against accidental drift).
	assert.Equal(t, 1, BlastRadiusMaxGroups)
	assert.Equal(t, 5, BlastRadiusMaxRepos)
}

func TestRunBlastRadiusGuard_InteractiveConfirmer(t *testing.T) {
	scope := scopeWith(3, 3)

	t.Run("typed count matches proceeds", func(t *testing.T) {
		e := &Engine{options: &Options{}, logger: testLogger(), scopeConfirmer: &fakeConfirmer{interactive: true, reply: 3}}
		require.NoError(t, e.runBlastRadiusGuard(scope))
	})

	t.Run("typed count mismatch refuses", func(t *testing.T) {
		fc := &fakeConfirmer{interactive: true, reply: 2}
		e := &Engine{options: &Options{}, logger: testLogger(), scopeConfirmer: fc}
		err := e.runBlastRadiusGuard(scope)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrScopeConfirmationRequired)
		assert.Equal(t, 1, fc.calls)
	})
}

func TestRunBlastRadiusGuard_NonInteractiveDefaultDeny(t *testing.T) {
	scope := scopeWith(3, 3)
	e := &Engine{options: &Options{}, logger: testLogger(), scopeConfirmer: &fakeConfirmer{interactive: false}}

	err := e.runBlastRadiusGuard(scope)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrScopeConfirmationRequired)
}

func TestRunBlastRadiusGuard_ConfirmScopeFlag(t *testing.T) {
	scope := scopeWith(3, 3)

	t.Run("matching flag proceeds even non-interactive", func(t *testing.T) {
		e := &Engine{options: &Options{ConfirmScope: intPtr(3)}, logger: testLogger(), scopeConfirmer: &fakeConfirmer{interactive: false}}
		require.NoError(t, e.runBlastRadiusGuard(scope))
	})

	t.Run("mismatched flag refuses", func(t *testing.T) {
		e := &Engine{options: &Options{ConfirmScope: intPtr(2)}, logger: testLogger(), scopeConfirmer: &fakeConfirmer{interactive: false}}
		err := e.runBlastRadiusGuard(scope)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrScopeConfirmationRequired)
	})
}

// TestRunBlastRadiusGuard_IgnoreRateLimitDoesNotBypass proves AC-7: the guard is
// independent of the rate-limit preflight. Setting IgnoreRateLimitPreflight on a
// guarded non-interactive scope still refuses without a correct --confirm-scope.
func TestRunBlastRadiusGuard_IgnoreRateLimitDoesNotBypass(t *testing.T) {
	scope := scopeWith(3, 3)
	e := &Engine{
		options:        &Options{IgnoreRateLimitPreflight: true},
		logger:         testLogger(),
		scopeConfirmer: &fakeConfirmer{interactive: false},
	}

	err := e.runBlastRadiusGuard(scope)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrScopeConfirmationRequired)
}
