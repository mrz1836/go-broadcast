package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// runReviewExplicitURL drives createRunReviewPR for a single explicit PR URL
// (no --all-assigned-prs), using the supplied flags pointers.
func runReviewExplicitURL(t *testing.T, url string) error {
	t.Helper()
	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{url})
	return cmd.Execute()
}

func TestReviewPR_GetPRError(t *testing.T) { //nolint:paralleltest // swaps package global newReviewPRClient
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)
	mockClient.On("GetPR", mock.Anything, "owner/repo", 5).Return(nil, errMockGH)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/5")
	// A single PR that fails to fetch results in failureCount>0 -> command errors.
	require.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestReviewPR_AlreadyMerged(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	merged := makeDependabotPR(6)
	now := time.Now()
	merged.MergedAt = &now
	mockClient.On("GetPR", mock.Anything, "owner/repo", 6).Return(merged, nil)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/6")
	require.NoError(t, err) // already merged counts as success
	mockClient.AssertExpectations(t)
}

func TestReviewPR_ClosedNotMerged(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	closed := makeDependabotPR(7)
	closed.State = "closed"
	mockClient.On("GetPR", mock.Anything, "owner/repo", 7).Return(closed, nil)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/7")
	require.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestReviewPR_GetCurrentUserError(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	pr := makeDependabotPR(8)
	mockClient.On("GetPR", mock.Anything, "owner/repo", 8).Return(pr, nil)
	mockClient.On("GetCurrentUser", mock.Anything).Return(nil, errMockGH)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/8")
	require.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestReviewPR_HasApprovedReviewError(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	pr := makeDependabotPR(9)
	mockClient.On("GetPR", mock.Anything, "owner/repo", 9).Return(pr, nil)
	mockClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)
	mockClient.On("HasApprovedReview", mock.Anything, "owner/repo", 9, "testuser").Return(false, errMockGH)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/9")
	require.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestReviewPR_DryRunDefault(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	pr := makeDependabotPR(30)
	setupDependabotBaseMocks(mockClient, pr, false)

	flags := &Flags{DryRun: true}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"https://github.com/owner/repo/pull/30"})
	require.NoError(t, cmd.Execute())
	// Dry-run must not call review/merge APIs.
	mockClient.AssertNotCalled(t, "ReviewPR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "MergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewPR_DryRunBypassIgnoreChecks(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	pr := makeDependabotPR(31)
	setupDependabotBaseMocks(mockClient, pr, false)

	flags := &Flags{DryRun: true}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--bypass", "--ignore-checks", "https://github.com/owner/repo/pull/31"})
	require.NoError(t, cmd.Execute())
	mockClient.AssertNotCalled(t, "MergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewPR_DryRunMergeConflict(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	pr := makeDependabotPR(32)
	notMergeable := false
	pr.Mergeable = &notMergeable
	setupDependabotBaseMocks(mockClient, pr, false)

	flags := &Flags{DryRun: true}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"https://github.com/owner/repo/pull/32"})
	require.NoError(t, cmd.Execute())
}

func TestReviewPR_MergeConflictEnablesAutoMerge(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)
	// No automerge-label gate: this non-Dependabot PR must be reviewed and have
	// auto-merge enabled. Pin the env so a leaked/ambient value can't flip the path.
	t.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "")

	pr := makeDependabotPR(33)
	notMergeable := false
	pr.Mergeable = &notMergeable
	setupDependabotBaseMocks(mockClient, pr, false)
	// Not self-authored, not already approved -> review then enable auto-merge.
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 33, "LGTM").Return(nil)
	mockClient.On("EnableAutoMergePR", mock.Anything, "owner/repo", 33, gh.MergeMethodSquash).Return(nil)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/33")
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestReviewPR_ImmediateMerge(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	// No GO_BROADCAST_AUTOMERGE_LABELS configured -> no label gate, no CI gate,
	// review then immediate merge.
	t.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "")

	pr := makeDependabotPR(40)
	setupDependabotBaseMocks(mockClient, pr, false)
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 40, "LGTM").Return(nil)
	mockClient.On("MergePR", mock.Anything, "owner/repo", 40, gh.MergeMethodSquash).Return(nil)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/40")
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestReviewPR_ReviewOnlyNoLabel(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	// Labels configured but PR lacks them -> review-only, merge skipped.
	t.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge")

	pr := makeDependabotPR(41) // no labels on the PR
	setupDependabotBaseMocks(mockClient, pr, false)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/41")
	require.NoError(t, err)
	mockClient.AssertNotCalled(t, "MergePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mockClient.AssertNotCalled(t, "ReviewPR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestReviewPR_SelfAuthoredImmediateMerge(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)
	t.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "")

	pr := makeDependabotPR(42)
	pr.User.Login = "testuser" // self-authored (matches GetCurrentUser)
	setupDependabotBaseMocks(mockClient, pr, false)
	mockClient.On("AddPRComment", mock.Anything, "owner/repo", 42, "LGTM").Return(nil)
	mockClient.On("MergePR", mock.Anything, "owner/repo", 42, gh.MergeMethodSquash).Return(nil)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/42")
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestReviewPR_BypassWithLabelPassingChecks(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)
	t.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "automerge")

	pr := makeDependabotPR(43)
	pr.Labels = []struct {
		Name string `json:"name"`
	}{{Name: "automerge"}}
	setupDependabotBaseMocks(mockClient, pr, false)
	// bypass allowed (label present) + checks pass -> review + immediate merge.
	mockClient.On("GetPRCheckStatus", mock.Anything, "owner/repo", 43).Return(makePassingCheckSummary(), nil)
	mockClient.On("ReviewPR", mock.Anything, "owner/repo", 43, "LGTM").Return(nil)
	mockClient.On("MergePR", mock.Anything, "owner/repo", 43, gh.MergeMethodSquash).Return(nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--bypass", "https://github.com/owner/repo/pull/43"})
	require.NoError(t, cmd.Execute())
	mockClient.AssertExpectations(t)
}

func TestReviewPR_BatchSummary(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)
	t.Setenv("GO_BROADCAST_AUTOMERGE_LABELS", "")

	for _, n := range []int{50, 51} {
		pr := makeDependabotPR(n)
		setupDependabotBaseMocks(mockClient, pr, false)
		mockClient.On("ReviewPR", mock.Anything, "owner/repo", n, "LGTM").Return(nil)
		mockClient.On("MergePR", mock.Anything, "owner/repo", n, gh.MergeMethodSquash).Return(nil)
	}

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{
		"https://github.com/owner/repo/pull/50",
		"https://github.com/owner/repo/pull/51",
	})
	require.NoError(t, cmd.Execute())
	mockClient.AssertExpectations(t)
}

func TestReviewPR_NoAssignedPRs(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)
	mockClient.On("SearchAssignedPRs", mock.Anything).Return([]gh.PR{}, nil)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs"})
	err := cmd.Execute()
	require.ErrorIs(t, err, ErrNoAssignedPRs)
}

func TestReviewPR_SearchError(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)
	mockClient.On("SearchAssignedPRs", mock.Anything).Return(nil, errMockGH)

	flags := &Flags{}
	cmd := createReviewPRCmd(flags)
	cmd.SetArgs([]string{"--all-assigned-prs"})
	require.Error(t, cmd.Execute())
}

func TestReviewPR_AlreadyApprovedAutoMergeEnabled(t *testing.T) { //nolint:paralleltest // swaps package global
	mockClient := gh.NewMockClient()
	withMockGHClient(t, mockClient)

	pr := makeDependabotPR(12)
	pr.AutoMerge = &gh.AutoMerge{}
	mockClient.On("GetPR", mock.Anything, "owner/repo", 12).Return(pr, nil)
	mockClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)
	// Already approved + auto-merge already enabled -> skipped as success, no review/merge.
	mockClient.On("HasApprovedReview", mock.Anything, "owner/repo", 12, "testuser").Return(true, nil)

	err := runReviewExplicitURL(t, "https://github.com/owner/repo/pull/12")
	require.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockClient.AssertNotCalled(t, "ReviewPR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
