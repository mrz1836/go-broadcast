package gh

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSummarizeChecks verifies each GitHub status/conclusion is categorized into
// the correct summary bucket, including the aggregate Completed count.
func TestSummarizeChecks(t *testing.T) {
	checks := []CheckRun{
		{Name: "success", Status: "completed", Conclusion: "success"},
		{Name: "neutral", Status: "completed", Conclusion: "neutral"},
		{Name: "skipped", Status: "completed", Conclusion: "skipped"},
		{Name: "failure", Status: "completed", Conclusion: "failure"},
		{Name: "canceled", Status: "completed", Conclusion: "canceled"},
		{Name: "timed_out", Status: "completed", Conclusion: "timed_out"},
		{Name: "action_required", Status: "completed", Conclusion: "action_required"},
		{Name: "queued", Status: "queued"},
		{Name: "in_progress", Status: "in_progress"},
	}

	summary := summarizeChecks(checks, len(checks))

	assert.Equal(t, len(checks), summary.Total)
	assert.Equal(t, 2, summary.Passed)    // success + neutral
	assert.Equal(t, 1, summary.Skipped)   // skipped
	assert.Equal(t, 4, summary.Failed)    // failure + canceled + timed_out + action_required
	assert.Equal(t, 2, summary.Running)   // queued + in_progress
	assert.Equal(t, 7, summary.Completed) // every completed status regardless of conclusion
	assert.Len(t, summary.Checks, len(checks))
}

// TestSummarizeChecks_Empty ensures an empty check list yields a zeroed summary
// that NoChecks() recognizes.
func TestSummarizeChecks_Empty(t *testing.T) {
	summary := summarizeChecks(nil, 0)
	assert.Equal(t, 0, summary.Total)
	assert.True(t, summary.NoChecks())
	assert.False(t, summary.AllPassed())
}

// TestNormalizeCheckName covers casing, whitespace, and leading emoji/symbol
// stripping used for ignore-list matching.
func TestNormalizeCheckName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "Dependabot Auto-merge", want: "dependabot auto-merge"},
		{name: "emoji_prefix", in: "🤖 Dependabot Auto-merge", want: "dependabot auto-merge"},
		{name: "surrounding_whitespace", in: "  CI / build  ", want: "ci / build"},
		{name: "leading_symbols", in: "✅✅ done", want: "done"},
		{name: "empty", in: "", want: ""},
		{name: "only_symbols", in: "🤖 ✅", want: ""},
		{name: "digits_lead", in: "1st check", want: "1st check"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeCheckName(tc.in))
		})
	}
}

// TestMatchesIgnoredCheck verifies matching semantics: case-insensitive, emoji
// prefix tolerant, and non-matching / empty-name behavior.
func TestMatchesIgnoredCheck(t *testing.T) {
	patterns := []string{"Dependabot Auto-merge", "CodeQL"}

	cases := []struct {
		name  string
		check string
		want  bool
	}{
		{name: "exact", check: "Dependabot Auto-merge", want: true},
		{name: "emoji_prefixed_check", check: "🤖 Dependabot Auto-merge", want: true},
		{name: "different_case", check: "dependabot auto-merge", want: true},
		{name: "second_pattern", check: "CodeQL", want: true},
		{name: "no_match", check: "CI / build", want: false},
		{name: "empty_name", check: "", want: false},
		{name: "symbols_only_name", check: "🤖", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, matchesIgnoredCheck(tc.check, patterns))
		})
	}

	// An empty pattern set never matches.
	assert.False(t, matchesIgnoredCheck("Dependabot Auto-merge", nil))
}

// TestWithoutChecks_NoOpCases verifies the receiver is returned unchanged when
// there is nothing to filter.
func TestWithoutChecks_NoOpCases(t *testing.T) {
	summary := summarizeChecks([]CheckRun{
		{Name: "CI / build", Status: "completed", Conclusion: "success"},
	}, 1)

	t.Run("empty_ignore_list", func(t *testing.T) {
		got, ignored := summary.WithoutChecks(nil)
		assert.Same(t, summary, got)
		assert.Nil(t, ignored)
	})

	t.Run("no_match", func(t *testing.T) {
		got, ignored := summary.WithoutChecks([]string{"nonexistent"})
		assert.Same(t, summary, got)
		assert.Nil(t, ignored)
	})

	t.Run("no_checks", func(t *testing.T) {
		empty := summarizeChecks(nil, 0)
		got, ignored := empty.WithoutChecks([]string{"anything"})
		assert.Same(t, empty, got)
		assert.Nil(t, ignored)
	})
}

// TestWithoutChecks_FiltersFailedCheck models the real-world case: an advisory
// "🤖 Dependabot Auto-merge" check has failed but is not a required check, so
// ignoring it flips the summary from failing to all-passed.
func TestWithoutChecks_FiltersFailedCheck(t *testing.T) {
	summary := summarizeChecks([]CheckRun{
		{Name: "CI / build", Status: "completed", Conclusion: "success"},
		{Name: "CI / test", Status: "completed", Conclusion: "success"},
		{Name: "🤖 Dependabot Auto-merge", Status: "completed", Conclusion: "failure"},
	}, 3)

	require.True(t, summary.HasFailedChecks())

	filtered, ignored := summary.WithoutChecks([]string{"Dependabot Auto-merge"})

	assert.Equal(t, []string{"🤖 Dependabot Auto-merge"}, ignored)
	assert.Equal(t, 2, filtered.Total)
	assert.Equal(t, 2, filtered.Passed)
	assert.Equal(t, 0, filtered.Failed)
	assert.False(t, filtered.HasFailedChecks())
	assert.True(t, filtered.AllPassed())

	// Receiver must be untouched (WithoutChecks returns a new summary).
	assert.Equal(t, 1, summary.Failed)
	assert.Len(t, summary.Checks, 3)
}

// TestWithoutChecks_IgnoreAll ensures filtering every check yields an empty
// summary (Total 0), which the gate treats as "no checks configured".
func TestWithoutChecks_IgnoreAll(t *testing.T) {
	summary := summarizeChecks([]CheckRun{
		{Name: "only-check", Status: "completed", Conclusion: "failure"},
	}, 1)

	filtered, ignored := summary.WithoutChecks([]string{"only-check"})

	assert.Equal(t, []string{"only-check"}, ignored)
	assert.Equal(t, 0, filtered.Total)
	assert.True(t, filtered.NoChecks())
	assert.Empty(t, filtered.Checks)
}

// TestWithoutChecks_MultiplePatterns verifies multiple ignore patterns filter
// several checks and report every ignored name.
func TestWithoutChecks_MultiplePatterns(t *testing.T) {
	summary := summarizeChecks([]CheckRun{
		{Name: "CI / build", Status: "completed", Conclusion: "success"},
		{Name: "🤖 Dependabot Auto-merge", Status: "completed", Conclusion: "failure"},
		{Name: "flaky-scan", Status: "completed", Conclusion: "timed_out"},
	}, 3)

	filtered, ignored := summary.WithoutChecks([]string{"dependabot auto-merge", "flaky-scan"})

	assert.ElementsMatch(t, []string{"🤖 Dependabot Auto-merge", "flaky-scan"}, ignored)
	assert.Equal(t, 1, filtered.Total)
	assert.Equal(t, 1, filtered.Passed)
	assert.Equal(t, 0, filtered.Failed)
	assert.True(t, filtered.AllPassed())
}
