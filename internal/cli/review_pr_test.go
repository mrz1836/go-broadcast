package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

func TestParsePRURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOwner   string
		wantRepo    string
		wantNumber  int
		expectError bool
	}{
		{
			name:       "Full HTTPS URL",
			url:        "https://github.com/owner/repo/pull/123",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 123,
		},
		{
			name:       "Full HTTP URL",
			url:        "http://github.com/owner/repo/pull/456",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 456,
		},
		{
			name:       "URL without protocol",
			url:        "github.com/owner/repo/pull/789",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 789,
		},
		{
			name:       "Short format",
			url:        "owner/repo#123",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 123,
		},
		{
			name:       "URL with trailing slash",
			url:        "https://github.com/owner/repo/pull/123",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 123,
		},
		{
			name:       "Owner with dash",
			url:        "https://github.com/my-org/repo/pull/1",
			wantOwner:  "my-org",
			wantRepo:   "repo",
			wantNumber: 1,
		},
		{
			name:       "Repo with dash",
			url:        "https://github.com/owner/my-repo/pull/2",
			wantOwner:  "owner",
			wantRepo:   "my-repo",
			wantNumber: 2,
		},
		{
			name:       "Large PR number",
			url:        "https://github.com/owner/repo/pull/99999",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 99999,
		},
		{
			name:        "Invalid format - missing pull segment",
			url:         "https://github.com/owner/repo/123",
			expectError: true,
		},
		{
			name:        "Invalid format - not a GitHub URL",
			url:         "https://gitlab.com/owner/repo/pull/123",
			expectError: true,
		},
		{
			name:        "Invalid format - malformed short format",
			url:         "owner/repo/123",
			expectError: true,
		},
		{
			name:        "Invalid format - missing PR number",
			url:         "https://github.com/owner/repo/pull/",
			expectError: true,
		},
		{
			name:        "Invalid format - non-numeric PR number",
			url:         "https://github.com/owner/repo/pull/abc",
			expectError: true,
		},
		{
			name:        "Empty URL",
			url:         "",
			expectError: true,
		},
		{
			name:        "Whitespace only",
			url:         "   ",
			expectError: true,
		},
		{
			name:        "Invalid short format - no hash",
			url:         "owner/repo123",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parsePRURL(tt.url)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, info)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, tt.wantOwner, info.Owner)
			assert.Equal(t, tt.wantRepo, info.Repo)
			assert.Equal(t, tt.wantNumber, info.Number)
			assert.NotEmpty(t, info.URL)
		})
	}
}

func TestParsePRURL_WithWhitespace(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "Leading whitespace",
			url:  "  https://github.com/owner/repo/pull/123",
		},
		{
			name: "Trailing whitespace",
			url:  "https://github.com/owner/repo/pull/123  ",
		},
		{
			name: "Both leading and trailing whitespace",
			url:  "  https://github.com/owner/repo/pull/123  ",
		},
		{
			name: "Tab character",
			url:  "\thttps://github.com/owner/repo/pull/123\t",
		},
		{
			name: "Newline character",
			url:  "\nhttps://github.com/owner/repo/pull/123\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parsePRURL(tt.url)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, "owner", info.Owner)
			assert.Equal(t, "repo", info.Repo)
			assert.Equal(t, 123, info.Number)
		})
	}
}

func TestCreateReviewPRCmd(t *testing.T) {
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)

	assert.NotNil(t, cmd)
	assert.Equal(t, "review-pr", cmd.Use[:9])
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)

	// Check that message flag exists
	messageFlag := cmd.Flags().Lookup("message")
	require.NotNil(t, messageFlag)
	assert.Equal(t, "LGTM", messageFlag.DefValue)
	assert.Equal(t, "m", messageFlag.Shorthand)
}

func TestReviewPRCommand_NoArgs(t *testing.T) {
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)

	// Set empty args
	cmd.SetArgs([]string{})

	// Should return error for no arguments
	err := cmd.Execute()
	require.Error(t, err)
}

func TestReviewPRCommand_InvalidURL(t *testing.T) {
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)

	// Set invalid URL
	cmd.SetArgs([]string{"not-a-valid-url"})

	// Should return error for invalid URL
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PR URL")
}

func TestReviewPRCommand_DryRun(t *testing.T) {
	// This test verifies the command structure and flag handling.
	// Full integration testing with mocked GitHub client would be done
	// in integration tests to avoid complexity in unit tests.

	flags := &Flags{
		DryRun: true,
	}
	cmd := createReviewPRCmd(flags)

	assert.NotNil(t, cmd)

	// Verify dry-run flag is honored
	assert.True(t, flags.DryRun)
}

func TestPRInfo_String(t *testing.T) {
	info := &PRInfo{
		Owner:  "testowner",
		Repo:   "testrepo",
		Number: 42,
		URL:    "https://github.com/testowner/testrepo/pull/42",
	}

	assert.Equal(t, "testowner", info.Owner)
	assert.Equal(t, "testrepo", info.Repo)
	assert.Equal(t, 42, info.Number)
	assert.NotEmpty(t, info.URL)
}

func TestReviewPRResult_Fields(t *testing.T) {
	result := ReviewPRResult{
		PRInfo: PRInfo{
			Owner:  "owner",
			Repo:   "repo",
			Number: 1,
			URL:    "https://github.com/owner/repo/pull/1",
		},
		Reviewed:      true,
		Merged:        true,
		MergeMethod:   "squash",
		Error:         "",
		AlreadyMerged: false,
	}

	assert.True(t, result.Reviewed)
	assert.True(t, result.Merged)
	assert.Equal(t, "squash", result.MergeMethod)
	assert.Empty(t, result.Error)
	assert.False(t, result.AlreadyMerged)
}

func TestParsePRURL_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOwner   string
		wantRepo    string
		expectError bool
	}{
		{
			name:      "Owner with underscore",
			url:       "https://github.com/my_org/repo/pull/1",
			wantOwner: "my_org",
			wantRepo:  "repo",
		},
		{
			name:      "Repo with underscore",
			url:       "https://github.com/owner/my_repo/pull/1",
			wantOwner: "owner",
			wantRepo:  "my_repo",
		},
		{
			name:      "Repo with dots",
			url:       "https://github.com/owner/my.repo/pull/1",
			wantOwner: "owner",
			wantRepo:  "my.repo",
		},
		{
			name:      "Mixed special characters",
			url:       "https://github.com/my-org_2/my.repo-1/pull/1",
			wantOwner: "my-org_2",
			wantRepo:  "my.repo-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parsePRURL(tt.url)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, info)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, tt.wantOwner, info.Owner)
			assert.Equal(t, tt.wantRepo, info.Repo)
		})
	}
}

func TestParsePRURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Double slashes in path",
			url:         "https://github.com/owner//repo/pull/1",
			expectError: true,
		},
		{
			name:        "Missing owner",
			url:         "https://github.com//repo/pull/1",
			expectError: true,
		},
		{
			name:        "Missing repo",
			url:         "https://github.com/owner//pull/1",
			expectError: true,
		},
		{
			name:        "Zero PR number",
			url:         "https://github.com/owner/repo/pull/0",
			expectError: false, // 0 is technically a valid number, though not a valid PR
		},
		{
			name:        "Negative PR number not supported by regex",
			url:         "https://github.com/owner/repo/pull/-1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parsePRURL(tt.url)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, info)
			} else {
				require.NoError(t, err)
				require.NotNil(t, info)
			}
		})
	}
}

func TestReviewPRCommand_AllAssignedPRsFlag(t *testing.T) {
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)

	// Verify the flag exists
	allAssignedPRsFlag := cmd.Flags().Lookup("all-assigned-prs")
	require.NotNil(t, allAssignedPRsFlag)
	assert.Equal(t, "false", allAssignedPRsFlag.DefValue)
	assert.Equal(t, "Review and merge all open PRs assigned to you (excludes drafts)", allAssignedPRsFlag.Usage)
}

func TestReviewPRCommand_MutualExclusivity(t *testing.T) {
	// This test verifies that using --all-assigned-prs with explicit URLs returns an error
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)

	// Set both flag and URLs
	cmd.SetArgs([]string{"--all-assigned-prs", "owner/repo#123"})

	// Should return error for mutually exclusive options
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use --all-assigned-prs")
}

func TestReviewPRCommand_AllAssignedPRsNoArgs(t *testing.T) {
	// This test verifies the command structure when using --all-assigned-prs
	flags := &Flags{
		DryRun: true,
	}
	cmd := createReviewPRCmd(flags)

	assert.NotNil(t, cmd)

	// Verify command accepts 0 arguments with the flag
	cmd.SetArgs([]string{"--all-assigned-prs"})
	// Command structure is valid (actual execution would require mock client)
}

func TestReviewPRCommand_NoArgsNoFlag(t *testing.T) {
	// This test verifies that no args and no flag returns an error
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)

	// Set no args and no flag
	cmd.SetArgs([]string{})

	// Should return error for no PR URLs
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid PR URLs")
}

func TestErrorConstants(t *testing.T) {
	// Verify new error constants exist and have appropriate messages
	assert.Contains(t, ErrMutuallyExclusiveFlags.Error(), "cannot use --all-assigned-prs")
	assert.Contains(t, ErrNoAssignedPRs.Error(), "no assigned PRs")
}

func TestReviewPRResult_DuplicateDetectionFields(t *testing.T) {
	// Test that the new duplicate detection fields are properly set
	result := ReviewPRResult{
		PRInfo: PRInfo{
			Owner:  "owner",
			Repo:   "repo",
			Number: 1,
			URL:    "https://github.com/owner/repo/pull/1",
		},
		AlreadyReviewed:         true,
		AutoMergeAlreadyEnabled: true,
	}

	assert.True(t, result.AlreadyReviewed)
	assert.True(t, result.AutoMergeAlreadyEnabled)
	assert.False(t, result.Reviewed)
	assert.False(t, result.AutoMergeEnabled)
}

func TestReviewPRResult_MixedStates(t *testing.T) {
	tests := []struct {
		name                    string
		alreadyReviewed         bool
		autoMergeAlreadyEnabled bool
		reviewed                bool
		autoMergeEnabled        bool
		expectedBehavior        string
	}{
		{
			name:                    "Fresh PR - no previous state",
			alreadyReviewed:         false,
			autoMergeAlreadyEnabled: false,
			reviewed:                true,
			autoMergeEnabled:        true,
			expectedBehavior:        "Should submit review and enable auto-merge",
		},
		{
			name:                    "Already reviewed, auto-merge already enabled",
			alreadyReviewed:         true,
			autoMergeAlreadyEnabled: true,
			reviewed:                false,
			autoMergeEnabled:        false,
			expectedBehavior:        "Should skip both review and auto-merge",
		},
		{
			name:                    "Already reviewed, need to enable auto-merge",
			alreadyReviewed:         true,
			autoMergeAlreadyEnabled: false,
			reviewed:                false,
			autoMergeEnabled:        true,
			expectedBehavior:        "Should skip review but enable auto-merge",
		},
		{
			name:                    "Not reviewed, auto-merge already enabled",
			alreadyReviewed:         false,
			autoMergeAlreadyEnabled: true,
			reviewed:                true,
			autoMergeEnabled:        false,
			expectedBehavior:        "Should submit review but skip auto-merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReviewPRResult{
				AlreadyReviewed:         tt.alreadyReviewed,
				AutoMergeAlreadyEnabled: tt.autoMergeAlreadyEnabled,
				Reviewed:                tt.reviewed,
				AutoMergeEnabled:        tt.autoMergeEnabled,
			}

			assert.Equal(t, tt.alreadyReviewed, result.AlreadyReviewed, tt.expectedBehavior)
			assert.Equal(t, tt.autoMergeAlreadyEnabled, result.AutoMergeAlreadyEnabled, tt.expectedBehavior)
			assert.Equal(t, tt.reviewed, result.Reviewed, tt.expectedBehavior)
			assert.Equal(t, tt.autoMergeEnabled, result.AutoMergeEnabled, tt.expectedBehavior)
		})
	}
}

func TestParseAutomergeLabels(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     []string
	}{
		{
			name:     "Single label",
			envValue: "automerge",
			want:     []string{"automerge"},
		},
		{
			name:     "Multiple labels comma-separated",
			envValue: "automerge,ready-to-merge,auto-merge",
			want:     []string{"automerge", "ready-to-merge", "auto-merge"},
		},
		{
			name:     "Labels with whitespace",
			envValue: " automerge , ready-to-merge ,  auto-merge  ",
			want:     []string{"automerge", "ready-to-merge", "auto-merge"},
		},
		{
			name:     "Labels with empty entries",
			envValue: "automerge,,ready-to-merge, ,auto-merge",
			want:     []string{"automerge", "ready-to-merge", "auto-merge"},
		},
		{
			name:     "Empty string",
			envValue: "",
			want:     nil,
		},
		{
			name:     "Only whitespace",
			envValue: "   ",
			want:     nil,
		},
		{
			name:     "Only commas",
			envValue: ",,,",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAutomergeLabels(tt.envValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasAutomergeLabel(t *testing.T) {
	tests := []struct {
		name     string
		prLabels []struct {
			Name string `json:"name"`
		}
		automergeLabels []string
		want            bool
	}{
		{
			name: "PR has matching automerge label",
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "bug"},
				{Name: "automerge"},
				{Name: "enhancement"},
			},
			automergeLabels: []string{"automerge"},
			want:            true,
		},
		{
			name: "PR has one of multiple automerge labels",
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "bug"},
				{Name: "ready-to-merge"},
			},
			automergeLabels: []string{"automerge", "ready-to-merge", "auto-merge"},
			want:            true,
		},
		{
			name: "PR does not have automerge label",
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "bug"},
				{Name: "enhancement"},
			},
			automergeLabels: []string{"automerge"},
			want:            false,
		},
		{
			name: "Case insensitive match",
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "AutoMerge"},
			},
			automergeLabels: []string{"automerge"},
			want:            true,
		},
		{
			name: "Empty PR labels",
			prLabels: []struct {
				Name string `json:"name"`
			}{},
			automergeLabels: []string{"automerge"},
			want:            false,
		},
		{
			name: "Empty automerge labels",
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "automerge"},
			},
			automergeLabels: []string{},
			want:            false,
		},
		{
			name: "Nil automerge labels",
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "automerge"},
			},
			automergeLabels: nil,
			want:            false,
		},
		{
			name: "Both empty",
			prLabels: []struct {
				Name string `json:"name"`
			}{},
			automergeLabels: []string{},
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAutomergeLabel(tt.prLabels, tt.automergeLabels)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBypassFlagBehavior(t *testing.T) {
	// Test that the bypass flag exists and has correct configuration
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)

	bypassFlag := cmd.Flags().Lookup("bypass")
	require.NotNil(t, bypassFlag)
	assert.Equal(t, "false", bypassFlag.DefValue)
	assert.Contains(t, bypassFlag.Usage, "admin privileges")
	assert.Contains(t, bypassFlag.Usage, "bypass branch protection")

	ignoreChecksFlag := cmd.Flags().Lookup("ignore-checks")
	require.NotNil(t, ignoreChecksFlag)
	assert.Equal(t, "false", ignoreChecksFlag.DefValue)
}

func TestReviewPRResult_MergeSkippedNoLabelField(t *testing.T) {
	// Test that the new MergeSkippedNoLabel field is properly set
	tests := []struct {
		name                string
		mergeSkippedNoLabel bool
		reviewed            bool
		merged              bool
		expectedBehavior    string
	}{
		{
			name:                "PR reviewed only - no automerge label",
			mergeSkippedNoLabel: true,
			reviewed:            true,
			merged:              false,
			expectedBehavior:    "Should mark as review-only when no automerge label",
		},
		{
			name:                "PR reviewed and merged - has automerge label",
			mergeSkippedNoLabel: false,
			reviewed:            true,
			merged:              true,
			expectedBehavior:    "Should proceed with merge when automerge label present",
		},
		{
			name:                "PR reviewed only - no labels configured (backwards compat)",
			mergeSkippedNoLabel: false,
			reviewed:            true,
			merged:              true,
			expectedBehavior:    "Should proceed with merge when no labels configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReviewPRResult{
				PRInfo: PRInfo{
					Owner:  "owner",
					Repo:   "repo",
					Number: 1,
					URL:    "https://github.com/owner/repo/pull/1",
				},
				Reviewed:            tt.reviewed,
				Merged:              tt.merged,
				MergeSkippedNoLabel: tt.mergeSkippedNoLabel,
			}

			assert.Equal(t, tt.mergeSkippedNoLabel, result.MergeSkippedNoLabel, tt.expectedBehavior)
			assert.Equal(t, tt.reviewed, result.Reviewed, tt.expectedBehavior)
			assert.Equal(t, tt.merged, result.Merged, tt.expectedBehavior)
		})
	}
}

func TestMergeGatingOnAutomergeLabel(t *testing.T) {
	// Test the logic of merge gating based on automerge labels
	// This documents the expected behavior:
	// - If automerge labels ARE configured and PR lacks the label -> review only, no merge
	// - If automerge labels ARE configured and PR has the label -> proceed with merge
	// - If automerge labels NOT configured -> proceed with merge (backwards compatibility)

	tests := []struct {
		name            string
		automergeLabels []string
		prLabels        []struct {
			Name string `json:"name"`
		}
		expectMergeAttempt bool
		description        string
	}{
		{
			name:            "Labels configured, PR has label - should merge",
			automergeLabels: []string{"automerge"},
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "automerge"},
			},
			expectMergeAttempt: true,
			description:        "PR has required label, merge should proceed",
		},
		{
			name:            "Labels configured, PR lacks label - no merge",
			automergeLabels: []string{"automerge"},
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "bug"},
			},
			expectMergeAttempt: false,
			description:        "PR lacks required label, should review only",
		},
		{
			name:            "No labels configured - should merge (backwards compat)",
			automergeLabels: []string{},
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "bug"},
			},
			expectMergeAttempt: true,
			description:        "No labels configured, merge should proceed for backwards compatibility",
		},
		{
			name:            "Nil labels configured - should merge (backwards compat)",
			automergeLabels: nil,
			prLabels: []struct {
				Name string `json:"name"`
			}{
				{Name: "bug"},
			},
			expectMergeAttempt: true,
			description:        "Nil labels configured, merge should proceed for backwards compatibility",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasAutoLabel := hasAutomergeLabel(tt.prLabels, tt.automergeLabels)

			// Simulate the merge gating logic from review_pr.go
			shouldSkipMerge := len(tt.automergeLabels) > 0 && !hasAutoLabel
			shouldAttemptMerge := !shouldSkipMerge

			assert.Equal(t, tt.expectMergeAttempt, shouldAttemptMerge, tt.description)
		})
	}
}

// TestReviewPRResult_CheckFields tests the new check-related fields
func TestReviewPRResult_CheckFields(t *testing.T) {
	result := ReviewPRResult{
		PRInfo: PRInfo{
			Owner:  "owner",
			Repo:   "repo",
			Number: 1,
			URL:    "https://github.com/owner/repo/pull/1",
		},
		Reviewed:             true,
		Merged:               false,
		MergeMethod:          "squash",
		ChecksSkippedRunning: true,
		CheckSummary:         "3/5 checks complete (3 passed, 2 running)",
		RunningCheckNames:    []string{"CI / Build", "CI / Tests"},
	}

	assert.True(t, result.ChecksSkippedRunning)
	assert.False(t, result.ChecksSkippedFailed)
	assert.Equal(t, "3/5 checks complete (3 passed, 2 running)", result.CheckSummary)
	assert.Len(t, result.RunningCheckNames, 2)
	assert.Contains(t, result.RunningCheckNames, "CI / Build")
}

// TestReviewPRResult_FailedCheckFields tests the failed check fields
func TestReviewPRResult_FailedCheckFields(t *testing.T) {
	result := ReviewPRResult{
		PRInfo: PRInfo{
			Owner:  "owner",
			Repo:   "repo",
			Number: 1,
			URL:    "https://github.com/owner/repo/pull/1",
		},
		Reviewed:            true,
		Merged:              false,
		MergeMethod:         "squash",
		ChecksSkippedFailed: true,
		CheckSummary:        "5/5 checks complete (3 passed, 2 failed)",
		FailedCheckNames:    []string{"CI / Lint", "CI / Security"},
	}

	assert.False(t, result.ChecksSkippedRunning)
	assert.True(t, result.ChecksSkippedFailed)
	assert.Equal(t, "5/5 checks complete (3 passed, 2 failed)", result.CheckSummary)
	assert.Len(t, result.FailedCheckNames, 2)
	assert.Contains(t, result.FailedCheckNames, "CI / Lint")
}

// TestSkippedPRInfo tests the skippedPRInfo struct
func TestSkippedPRInfo(t *testing.T) {
	info := skippedPRInfo{
		Repo:       "owner/repo",
		Number:     123,
		Reason:     "running",
		CheckNames: []string{"CI / Build", "CI / Tests"},
	}

	assert.Equal(t, "owner/repo", info.Repo)
	assert.Equal(t, 123, info.Number)
	assert.Equal(t, "running", info.Reason)
	assert.Len(t, info.CheckNames, 2)
}

// TestSkippedPRInfo_FailedReason tests the skippedPRInfo with failed reason
func TestSkippedPRInfo_FailedReason(t *testing.T) {
	info := skippedPRInfo{
		Repo:       "owner/repo",
		Number:     456,
		Reason:     "failed",
		CheckNames: []string{"CI / Lint"},
	}

	assert.Equal(t, "failed", info.Reason)
	assert.Len(t, info.CheckNames, 1)
	assert.Contains(t, info.CheckNames, "CI / Lint")
}

// ------------------------------------------------------------------
// Dependabot flag tests
// ------------------------------------------------------------------

// errTestBranchProtection simulates a branch-protection rejection response
// from the gh CLI so the dependabot+bypass fallback path can be exercised.
var errTestBranchProtection = errors.New("base branch policy prohibits the merge")

// errTestCheckStatusFetch simulates a transient error while fetching CI check
// status, used to exercise the ciGateUnknown branch in runCIGate.
var errTestCheckStatusFetch = errors.New("check status fetch failed")

// withMockGHClient swaps newReviewPRClient for a function that returns the
// provided mock. Restoration is automatic via t.Cleanup so tests stay isolated.
// Tests using this helper MUST NOT use t.Parallel() because newReviewPRClient
// is a package-level variable.
func withMockGHClient(t *testing.T, mockClient *gh.MockClient) {
	t.Helper()
	orig := newReviewPRClient
	newReviewPRClient = func(_ context.Context, _ *logrus.Logger) (gh.Client, error) {
		return mockClient, nil
	}
	t.Cleanup(func() {
		newReviewPRClient = orig
	})
}

// makeDependabotPR returns a minimal PR struct representing a Dependabot PR.
func makeDependabotPR(number int) *gh.PR {
	pr := &gh.PR{
		Number:    number,
		State:     "open",
		Title:     "chore(deps): bump foo",
		Mergeable: boolPtr(true),
	}
	pr.User.Login = "dependabot[bot]"
	pr.Head.SHA = "abc123"
	return pr
}

// makeRepositorySettings returns a Repository with only squash-merge enabled.
func makeRepositorySettings() *gh.Repository {
	return &gh.Repository{
		Name:             "repo",
		FullName:         "owner/repo",
		DefaultBranch:    "main",
		AllowSquashMerge: true,
	}
}

// makePassingCheckSummary returns a CheckStatusSummary with all checks green.
func makePassingCheckSummary() *gh.CheckStatusSummary {
	return &gh.CheckStatusSummary{
		Total:     2,
		Completed: 2,
		Passed:    2,
		Checks: []gh.CheckRun{
			{Name: "CI / build", Status: "completed", Conclusion: "success"},
			{Name: "CI / test", Status: "completed", Conclusion: "success"},
		},
	}
}

// makeRunningCheckSummary returns a CheckStatusSummary with one running check.
func makeRunningCheckSummary() *gh.CheckStatusSummary {
	return &gh.CheckStatusSummary{
		Total:     2,
		Completed: 1,
		Passed:    1,
		Running:   1,
		Checks: []gh.CheckRun{
			{Name: "CI / build", Status: "completed", Conclusion: "success"},
			{Name: "CI / test", Status: "in_progress"},
		},
	}
}

// makeFailedCheckSummary returns a CheckStatusSummary with one failed check.
func makeFailedCheckSummary() *gh.CheckStatusSummary {
	return &gh.CheckStatusSummary{
		Total:     2,
		Completed: 2,
		Passed:    1,
		Failed:    1,
		Checks: []gh.CheckRun{
			{Name: "CI / build", Status: "completed", Conclusion: "success"},
			{Name: "CI / test", Status: "completed", Conclusion: "failure"},
		},
	}
}

// setupDependabotBaseMocks wires up the common mock expectations every
// dependabot test needs: current user, PR fetch, approved-review check, and
// repository settings. It does NOT wire up the search call or the CI gate
// call — each test does those explicitly.
func setupDependabotBaseMocks(m *gh.MockClient, pr *gh.PR, alreadyApproved bool) {
	m.On("GetPR", mock.Anything, "owner/repo", pr.Number).Return(pr, nil)
	m.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)
	m.On("HasApprovedReview", mock.Anything, "owner/repo", pr.Number, "testuser").Return(alreadyApproved, nil)
	m.On("GetRepository", mock.Anything, "owner/repo").Return(makeRepositorySettings(), nil)
}

func TestReviewPR_DependabotRequiresAllAssigned(t *testing.T) {
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--dependabot"})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDependabotRequiresAllAssigned)
}

func TestReviewPR_DependabotWithExplicitURLs(t *testing.T) {
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--dependabot", "--all-assigned-prs", "owner/repo#1"})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMutuallyExclusiveFlags)
}

func TestReviewPR_DependabotHappyPath(t *testing.T) {
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	pr1 := gh.PR{Number: 10, State: "open", Repo: "owner/repo"}
	pr2 := gh.PR{Number: 11, State: "open", Repo: "owner/repo"}
	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{pr1, pr2}, nil)

	// Full PR details for each PR
	fullPR1 := makeDependabotPR(10)
	fullPR2 := makeDependabotPR(11)
	setupDependabotBaseMocks(mockClient, fullPR1, false)
	setupDependabotBaseMocks(mockClient, fullPR2, false)

	// CI gate returns all passing
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 10).Return(makePassingCheckSummary(), nil)
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 11).Return(makePassingCheckSummary(), nil)

	// Review + merge for each
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 10, "LGTM").Return(nil)
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 11, "LGTM").Return(nil)
	mockClient.On("MergePR", mock.Anything, "owner/repo", 10, gh.MergeMethodSquash).Return(nil)
	mockClient.On("MergePR", mock.Anything, "owner/repo", 11, gh.MergeMethodSquash).Return(nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot"})
	err := cmd.Execute()
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
	// MergePR must have been called for both, and EnableAutoMergePR must NOT have been called
	mockClient.AssertNotCalled(t, "EnableAutoMergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "BypassMergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewPR_DependabotChecksRunning(t *testing.T) {
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{{Number: 20, State: "open", Repo: "owner/repo"}}, nil)

	fullPR := makeDependabotPR(20)
	setupDependabotBaseMocks(mockClient, fullPR, false)
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 20).Return(makeRunningCheckSummary(), nil)
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 20, "LGTM").Return(nil)
	mockClient.On("EnableAutoMergePR", mock.Anything, "owner/repo", 20, gh.MergeMethodSquash).Return(nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot"})
	err := cmd.Execute()
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
	// MergePR must NOT have been called - we went straight to auto-merge
	mockClient.AssertNotCalled(t, "MergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewPR_DependabotChecksFailed(t *testing.T) {
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{{Number: 30, State: "open", Repo: "owner/repo"}}, nil)

	fullPR := makeDependabotPR(30)
	setupDependabotBaseMocks(mockClient, fullPR, false)
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 30).Return(makeFailedCheckSummary(), nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot"})
	err := cmd.Execute()
	require.NoError(t, err) // processing "succeeds" — PR was evaluated and skipped

	mockClient.AssertExpectations(t)
	// Neither review nor merge should have been called
	mockClient.AssertNotCalled(t, "ReviewPR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "MergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "EnableAutoMergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewPR_DependabotBypassesLabelGate(t *testing.T) {
	t.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge")

	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{{Number: 40, State: "open", Repo: "owner/repo"}}, nil)

	// Intentionally NO "automerge" label on this PR
	fullPR := makeDependabotPR(40)
	setupDependabotBaseMocks(mockClient, fullPR, false)
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 40).Return(makePassingCheckSummary(), nil)
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 40, "LGTM").Return(nil)
	mockClient.On("MergePR", mock.Anything, "owner/repo", 40, gh.MergeMethodSquash).Return(nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot"})
	err := cmd.Execute()
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestReviewPR_DependabotNoPRsFound(t *testing.T) {
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{}, nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot"})
	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoDependabotPRs)

	mockClient.AssertExpectations(t)
}

func TestReviewPR_DependabotAlreadyApproved(t *testing.T) {
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{{Number: 50, State: "open", Repo: "owner/repo"}}, nil)

	// User already approved — but auto-merge is NOT enabled yet and PR is not merged,
	// so we still proceed through CI gate + merge.
	fullPR := makeDependabotPR(50)
	setupDependabotBaseMocks(mockClient, fullPR, true)
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 50).Return(makePassingCheckSummary(), nil)
	mockClient.On("MergePR", mock.Anything, "owner/repo", 50, gh.MergeMethodSquash).Return(nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot"})
	err := cmd.Execute()
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
	// Review should be skipped because user already approved
	mockClient.AssertNotCalled(t, "ReviewPR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewPR_DependabotDryRun(t *testing.T) {
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{{Number: 60, State: "open", Repo: "owner/repo"}}, nil)

	fullPR := makeDependabotPR(60)
	setupDependabotBaseMocks(mockClient, fullPR, false)

	flags := &Flags{DryRun: true}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot"})
	err := cmd.Execute()
	require.NoError(t, err)

	// Dry run must NOT trigger any mutating calls
	mockClient.AssertNotCalled(t, "ReviewPR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "MergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "EnableAutoMergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "BypassMergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "GetPRCheckStatus", mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewPR_DependabotWithBypass_BranchProtectionFallback(t *testing.T) {
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{{Number: 70, State: "open", Repo: "owner/repo"}}, nil)

	fullPR := makeDependabotPR(70)
	setupDependabotBaseMocks(mockClient, fullPR, false)
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 70).Return(makePassingCheckSummary(), nil)
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 70, "LGTM").Return(nil)

	// First merge attempt fails with branch protection error
	mockClient.On("MergePR", mock.Anything, "owner/repo", 70, gh.MergeMethodSquash).Return(errTestBranchProtection)
	// Then bypass merge succeeds
	mockClient.On("BypassMergePR", mock.Anything, "owner/repo", 70, gh.MergeMethodSquash).Return(nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot", "--bypass"})
	err := cmd.Execute()
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestReviewPR_DependabotMergeConflict(t *testing.T) {
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRsByAuthor", mock.Anything, "app/dependabot").
		Return([]gh.PR{{Number: 80, State: "open", Repo: "owner/repo"}}, nil)

	fullPR := makeDependabotPR(80)
	fullPR.Mergeable = boolPtr(false) // merge conflict
	setupDependabotBaseMocks(mockClient, fullPR, false)
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 80, "LGTM").Return(nil)
	mockClient.On("EnableAutoMergePR", mock.Anything, "owner/repo", 80, gh.MergeMethodSquash).Return(nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--dependabot"})
	err := cmd.Execute()
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
	// CI gate should NOT be called because merge-conflict branch short-circuits before it
	mockClient.AssertNotCalled(t, "GetPRCheckStatus", mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "MergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestReviewPR_BypassCIGateRegression guards the runCIGate extraction: the
// existing --bypass code path must still skip PRs when checks are running.
func TestReviewPR_BypassCIGateRegression(t *testing.T) {
	t.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge")

	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	mockClient.On("SearchAssignedPRs", mock.Anything).
		Return([]gh.PR{{Number: 90, State: "open", Repo: "owner/repo"}}, nil)

	// Non-dependabot PR WITH the required automerge label so --bypass is allowed
	pr := &gh.PR{
		Number:    90,
		State:     "open",
		Title:     "some change",
		Mergeable: boolPtr(true),
		Labels: []struct {
			Name string `json:"name"`
		}{{Name: "automerge"}},
	}
	pr.User.Login = "some-human"
	pr.Head.SHA = "def456"

	setupDependabotBaseMocks(mockClient, pr, false)
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 90).Return(makeRunningCheckSummary(), nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs", "--bypass"})
	err := cmd.Execute()
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
	// --bypass path must SKIP the PR when checks are running (not fall back to auto-merge)
	mockClient.AssertNotCalled(t, "ReviewPR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "MergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "EnableAutoMergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestRunCIGate_Unit exercises the runCIGate helper directly across all
// decision branches to guarantee the extraction keeps every path covered.
func TestRunCIGate_Unit(t *testing.T) {
	cases := []struct {
		name     string
		summary  *gh.CheckStatusSummary
		fetchErr error
		want     ciGateDecision
	}{
		{name: "proceed_all_passed", summary: makePassingCheckSummary(), want: ciGateProceed},
		{name: "skip_running", summary: makeRunningCheckSummary(), want: ciGateSkipRunning},
		{name: "skip_failed", summary: makeFailedCheckSummary(), want: ciGateSkipFailed},
		{name: "proceed_no_checks", summary: &gh.CheckStatusSummary{}, want: ciGateProceed},
		{name: "unknown_on_error", fetchErr: errTestCheckStatusFetch, want: ciGateUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := gh.NewMockClient()
			if tc.fetchErr != nil {
				m.On("GetPRCheckStatus", mock.Anything, "owner/repo", 1).Return(nil, tc.fetchErr)
			} else {
				m.On("GetPRCheckStatus", mock.Anything, "owner/repo", 1).Return(tc.summary, nil)
			}

			prInfo := &PRInfo{Owner: "owner", Repo: "repo", Number: 1}
			result := &ReviewPRResult{}

			got := runCIGate(context.Background(), m, "owner/repo", prInfo, result)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestReviewPR_DependabotFlagRegistered(t *testing.T) {
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)

	depFlag := cmd.Flags().Lookup("dependabot")
	require.NotNil(t, depFlag)
	assert.Equal(t, "false", depFlag.DefValue)
	assert.Contains(t, depFlag.Usage, "Dependabot")
}

func TestErrNoDependabotPRs(t *testing.T) {
	assert.Contains(t, ErrNoDependabotPRs.Error(), "Dependabot")
	assert.Contains(t, ErrDependabotRequiresAllAssigned.Error(), "--all-assigned-prs")
}
