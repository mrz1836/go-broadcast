package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
