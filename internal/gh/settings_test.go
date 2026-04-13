package gh

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	errTestCLIError            = errors.New("cli error")
	errTestGetFailed           = errors.New("get failed")
	errTestUpdateFailed        = errors.New("update failed")
	errTestServerError500      = errors.New("server error 500")
	errTestServerError         = errors.New("server error")
	errTestInternalServerError = errors.New("internal server error")
	errTestLabelsError         = errors.New("labels error")
	errTestLabelsFetchError    = errors.New("labels fetch error")
	errTestTopicsError         = errors.New("topics error")
)

func TestBuildBranchRuleset(t *testing.T) {
	t.Parallel()

	t.Run("basic branch ruleset", func(t *testing.T) {
		t.Parallel()

		include := []string{"refs/heads/main", "refs/heads/release/*"}
		exclude := []string{"refs/heads/dev"}
		rules := []string{"deletion", "pull_request"}

		rs := BuildBranchRuleset("my-branch-rules", include, exclude, rules)

		assert.Equal(t, "my-branch-rules", rs.Name)
		assert.Equal(t, "branch", rs.Target)
		assert.Equal(t, "active", rs.Enforcement)

		// Verify bypass actors
		require.Len(t, rs.BypassActors, 1)
		assert.Equal(t, 5, rs.BypassActors[0].ActorID)
		assert.Equal(t, "RepositoryRole", rs.BypassActors[0].ActorType)
		assert.Equal(t, "always", rs.BypassActors[0].BypassMode)

		// Verify conditions
		require.NotNil(t, rs.Conditions)
		assert.Equal(t, include, rs.Conditions.RefName.Include)
		assert.Equal(t, exclude, rs.Conditions.RefName.Exclude)

		// Verify rules
		require.Len(t, rs.Rules, 2)
		assert.Equal(t, "deletion", rs.Rules[0].Type)
		assert.Equal(t, "pull_request", rs.Rules[1].Type)
	})

	t.Run("default name when empty", func(t *testing.T) {
		t.Parallel()

		rs := BuildBranchRuleset("", []string{"refs/heads/main"}, nil, []string{"deletion"})

		assert.Equal(t, "branch-protection", rs.Name)
		assert.Equal(t, "branch", rs.Target)
	})

	t.Run("custom name is preserved", func(t *testing.T) {
		t.Parallel()

		rs := BuildBranchRuleset("custom-name", []string{"refs/heads/*"}, nil, nil)

		assert.Equal(t, "custom-name", rs.Name)
	})

	t.Run("empty rules produce empty slice", func(t *testing.T) {
		t.Parallel()

		rs := BuildBranchRuleset("no-rules", []string{"refs/heads/main"}, []string{}, []string{})

		assert.Empty(t, rs.Rules)
		assert.NotNil(t, rs.Rules, "rules should be an empty slice, not nil")
	})

	t.Run("nil rules produce empty slice", func(t *testing.T) {
		t.Parallel()

		rs := BuildBranchRuleset("nil-rules", []string{"refs/heads/main"}, nil, nil)

		assert.Empty(t, rs.Rules)
		assert.NotNil(t, rs.Rules, "rules should be an empty slice, not nil")
	})

	t.Run("nil include and exclude become empty slices", func(t *testing.T) {
		t.Parallel()

		rs := BuildBranchRuleset("nil-conditions", nil, nil, []string{"deletion"})

		require.NotNil(t, rs.Conditions)
		assert.Equal(t, []string{}, rs.Conditions.RefName.Include)
		assert.Equal(t, []string{}, rs.Conditions.RefName.Exclude)
	})

	t.Run("many rules", func(t *testing.T) {
		t.Parallel()

		rules := []string{"deletion", "update", "pull_request", "required_signatures", "non_fast_forward"}
		rs := BuildBranchRuleset("many-rules", []string{"refs/heads/*"}, nil, rules)

		require.Len(t, rs.Rules, 5)
		for i, r := range rules {
			assert.Equal(t, r, rs.Rules[i].Type)
		}
	})
}

func TestBuildTagRuleset(t *testing.T) {
	t.Parallel()

	t.Run("basic tag ruleset", func(t *testing.T) {
		t.Parallel()

		include := []string{"refs/tags/v*"}
		exclude := []string{"refs/tags/test-*"}
		rules := []string{"deletion", "update"}

		rs := BuildTagRuleset("my-tag-rules", include, exclude, rules)

		assert.Equal(t, "my-tag-rules", rs.Name)
		assert.Equal(t, "tag", rs.Target)
		assert.Equal(t, "active", rs.Enforcement)

		// Verify bypass actors
		require.Len(t, rs.BypassActors, 1)
		assert.Equal(t, 5, rs.BypassActors[0].ActorID)
		assert.Equal(t, "RepositoryRole", rs.BypassActors[0].ActorType)
		assert.Equal(t, "always", rs.BypassActors[0].BypassMode)

		// Verify conditions
		require.NotNil(t, rs.Conditions)
		assert.Equal(t, include, rs.Conditions.RefName.Include)
		assert.Equal(t, exclude, rs.Conditions.RefName.Exclude)

		// Verify rules
		require.Len(t, rs.Rules, 2)
		assert.Equal(t, "deletion", rs.Rules[0].Type)
		assert.Equal(t, "update", rs.Rules[1].Type)
	})

	t.Run("default name when empty", func(t *testing.T) {
		t.Parallel()

		rs := BuildTagRuleset("", []string{"refs/tags/v*"}, nil, []string{"deletion"})

		assert.Equal(t, "tag-protection", rs.Name)
		assert.Equal(t, "tag", rs.Target)
	})

	t.Run("custom name is preserved", func(t *testing.T) {
		t.Parallel()

		rs := BuildTagRuleset("release-tags", []string{"refs/tags/*"}, nil, nil)

		assert.Equal(t, "release-tags", rs.Name)
	})

	t.Run("empty rules produce empty slice", func(t *testing.T) {
		t.Parallel()

		rs := BuildTagRuleset("no-rules", []string{"refs/tags/v*"}, []string{}, []string{})

		assert.Empty(t, rs.Rules)
		assert.NotNil(t, rs.Rules, "rules should be an empty slice, not nil")
	})

	t.Run("nil rules produce empty slice", func(t *testing.T) {
		t.Parallel()

		rs := BuildTagRuleset("nil-rules", []string{"refs/tags/v*"}, nil, nil)

		assert.Empty(t, rs.Rules)
		assert.NotNil(t, rs.Rules, "rules should be an empty slice, not nil")
	})

	t.Run("nil include and exclude become empty slices", func(t *testing.T) {
		t.Parallel()

		rs := BuildTagRuleset("nil-conditions", nil, nil, []string{"update"})

		require.NotNil(t, rs.Conditions)
		assert.Equal(t, []string{}, rs.Conditions.RefName.Include)
		assert.Equal(t, []string{}, rs.Conditions.RefName.Exclude)
	})

	t.Run("many rules", func(t *testing.T) {
		t.Parallel()

		rules := []string{"deletion", "update", "non_fast_forward"}
		rs := BuildTagRuleset("many-rules", []string{"refs/tags/*"}, nil, rules)

		require.Len(t, rs.Rules, 3)
		for i, r := range rules {
			assert.Equal(t, r, rs.Rules[i].Type)
		}
	})
}

func TestBuildBranchRuleset_VsTagRuleset_TargetDifference(t *testing.T) {
	t.Parallel()

	include := []string{"refs/heads/main"}
	rules := []string{"deletion"}

	branch := BuildBranchRuleset("test", include, nil, rules)
	tag := BuildTagRuleset("test", include, nil, rules)

	assert.Equal(t, "branch", branch.Target)
	assert.Equal(t, "tag", tag.Target)

	// Both should share the same enforcement, bypass actors, conditions structure, and rules
	assert.Equal(t, branch.Enforcement, tag.Enforcement)
	assert.Equal(t, branch.BypassActors, tag.BypassActors)
	assert.Equal(t, branch.Rules, tag.Rules)
	assert.Equal(t, branch.Conditions.RefName, tag.Conditions.RefName)
}

func TestBuildBranchRuleset_DefaultName_VsTagRuleset_DefaultName(t *testing.T) {
	t.Parallel()

	branch := BuildBranchRuleset("", nil, nil, nil)
	tag := BuildTagRuleset("", nil, nil, nil)

	assert.Equal(t, "branch-protection", branch.Name)
	assert.Equal(t, "tag-protection", tag.Name)
}

func TestBuildRuleset_NilIncludeExclude_SerializeToEmptyArrays(t *testing.T) {
	t.Parallel()

	t.Run("branch ruleset nil slices produce [] not null in JSON", func(t *testing.T) {
		t.Parallel()

		rs := BuildBranchRuleset("test", nil, nil, nil)
		data, err := json.Marshal(rs.Conditions)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"include":[]`, "include should serialize as empty array")
		assert.Contains(t, jsonStr, `"exclude":[]`, "exclude should serialize as empty array")
		assert.NotContains(t, jsonStr, `"include":null`)
		assert.NotContains(t, jsonStr, `"exclude":null`)
	})

	t.Run("tag ruleset nil slices produce [] not null in JSON", func(t *testing.T) {
		t.Parallel()

		rs := BuildTagRuleset("test", nil, nil, nil)
		data, err := json.Marshal(rs.Conditions)
		require.NoError(t, err)

		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"include":[]`, "include should serialize as empty array")
		assert.Contains(t, jsonStr, `"exclude":[]`, "exclude should serialize as empty array")
		assert.NotContains(t, jsonStr, `"include":null`)
		assert.NotContains(t, jsonStr, `"exclude":null`)
	})
}

// Tests for githubClient settings methods

func TestCreateRepository_Success(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	repo := Repository{Name: "my-repo", FullName: "owner/my-repo"}
	repoJSON, err := json.Marshal(repo)
	require.NoError(t, err)

	opts := CreateRepoOptions{Name: "owner/my-repo", Description: "test repo", Private: true}
	mockRunner.On("Run", ctx, "gh", []string{"repo", "create", "owner/my-repo", "--description", "test repo", "--private", "--clone=false"}).
		Return([]byte{}, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/my-repo"}).
		Return(repoJSON, nil)

	result, err := client.CreateRepository(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "my-repo", result.Name)
	mockRunner.AssertExpectations(t)
}

func TestCreateRepository_RunnerError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	opts := CreateRepoOptions{Name: "owner/my-repo", Description: "", Private: false}
	mockRunner.On("Run", ctx, "gh", []string{"repo", "create", "owner/my-repo", "--description", "", "--public", "--clone=false"}).
		Return(nil, errTestCLIError)

	result, err := client.CreateRepository(ctx, opts)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "create repository")
	mockRunner.AssertExpectations(t)
}

func TestCreateRepository_GetRepositoryError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	opts := CreateRepoOptions{Name: "owner/my-repo", Description: "desc", Private: true}
	mockRunner.On("Run", ctx, "gh", []string{"repo", "create", "owner/my-repo", "--description", "desc", "--private", "--clone=false"}).
		Return([]byte{}, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/my-repo"}).
		Return(nil, errTestGetFailed)

	result, err := client.CreateRepository(ctx, opts)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get created repository")
	mockRunner.AssertExpectations(t)
}

func TestUpdateRepoSettings_Success(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	settings := RepoSettings{HasIssues: true, AllowSquashMerge: true}
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo", "--method", "PATCH", "--input", "-"}).
		Return([]byte{}, nil)

	err := client.UpdateRepoSettings(ctx, "owner/repo", settings)
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestUpdateRepoSettings_RunnerError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	settings := RepoSettings{HasIssues: true}
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo", "--method", "PATCH", "--input", "-"}).
		Return(nil, errTestUpdateFailed)

	err := client.UpdateRepoSettings(ctx, "owner/repo", settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
	mockRunner.AssertExpectations(t)
}

func TestGetRepoSettings_Success(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	settings := RepoSettings{HasIssues: true, AllowSquashMerge: true, SquashMergeCommitTitle: "PR_TITLE"}
	settingsJSON, err := json.Marshal(settings)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo"}).
		Return(settingsJSON, nil)

	result, err := client.GetRepoSettings(ctx, "owner/repo")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.HasIssues)
	assert.True(t, result.AllowSquashMerge)
	assert.Equal(t, "PR_TITLE", result.SquashMergeCommitTitle)
	mockRunner.AssertExpectations(t)
}

func TestGetRepoSettings_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetRepoSettings(ctx, "owner/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	require.ErrorIs(t, err, ErrRepositoryNotFound)
	mockRunner.AssertExpectations(t)
}

func TestGetRepoSettings_RunnerError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo"}).
		Return(nil, errTestServerError500)

	result, err := client.GetRepoSettings(ctx, "owner/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get repo settings")
	mockRunner.AssertExpectations(t)
}

func TestGetRepoSettings_ParseError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo"}).
		Return([]byte("not valid json{{"), nil)

	result, err := client.GetRepoSettings(ctx, "owner/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parse repo settings")
	mockRunner.AssertExpectations(t)
}

func TestListRulesets_Success(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	rulesets := []Ruleset{
		{ID: 1, Name: "branch-protection", Target: "branch", Enforcement: "active"},
		{ID: 2, Name: "tag-protection", Target: "tag", Enforcement: "active"},
	}
	rulesetsJSON, err := json.Marshal(rulesets)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/rulesets"}).
		Return(rulesetsJSON, nil)

	result, err := client.ListRulesets(ctx, "owner/repo")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "branch-protection", result[0].Name)
	mockRunner.AssertExpectations(t)
}

func TestListRulesets_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/rulesets"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.ListRulesets(ctx, "owner/repo")
	require.NoError(t, err)
	assert.Nil(t, result)
	mockRunner.AssertExpectations(t)
}

func TestListRulesets_RunnerError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/rulesets"}).
		Return(nil, errTestServerError)

	result, err := client.ListRulesets(ctx, "owner/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "list rulesets")
	mockRunner.AssertExpectations(t)
}

func TestListRulesets_ParseError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/rulesets"}).
		Return([]byte("invalid json{{"), nil)

	result, err := client.ListRulesets(ctx, "owner/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parse rulesets")
	mockRunner.AssertExpectations(t)
}

func TestCreateOrUpdateRuleset_CreateNew(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/rulesets"}).
		Return([]byte("[]"), nil)
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/rulesets", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	ruleset := Ruleset{Name: "branch-protection", Target: "branch", Enforcement: "active"}
	err := client.CreateOrUpdateRuleset(ctx, "owner/repo", ruleset)
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestCreateOrUpdateRuleset_UpdateExisting(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	existing := []Ruleset{{ID: 42, Name: "branch-protection", Target: "branch", Enforcement: "active"}}
	existingJSON, err := json.Marshal(existing)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/rulesets"}).
		Return(existingJSON, nil)
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/rulesets/42", "--method", "PUT", "--input", "-"}).
		Return([]byte("{}"), nil)

	ruleset := Ruleset{Name: "branch-protection", Target: "branch", Enforcement: "active"}
	err = client.CreateOrUpdateRuleset(ctx, "owner/repo", ruleset)
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestCreateOrUpdateRuleset_List404_ThenCreate(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/rulesets"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/rulesets", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	ruleset := Ruleset{Name: "new-ruleset", Target: "branch", Enforcement: "active"}
	err := client.CreateOrUpdateRuleset(ctx, "owner/repo", ruleset)
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestCreateOrUpdateRuleset_ListError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/rulesets"}).
		Return(nil, errTestInternalServerError)

	ruleset := Ruleset{Name: "branch-protection", Target: "branch", Enforcement: "active"}
	err := client.CreateOrUpdateRuleset(ctx, "owner/repo", ruleset)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list rulesets for upsert")
	mockRunner.AssertExpectations(t)
}

func TestListLabels_Success(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	labels := []Label{
		{Name: "bug", Color: "d73a4a", Description: "Bug"},
		{Name: "feature", Color: "a2eeef"},
	}
	labelsJSON, err := json.Marshal(labels)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/labels", "--paginate"}).
		Return(labelsJSON, nil)

	result, err := client.ListLabels(ctx, "owner/repo")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "bug", result[0].Name)
	mockRunner.AssertExpectations(t)
}

func TestListLabels_RunnerError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/labels", "--paginate"}).
		Return(nil, errTestLabelsError)

	result, err := client.ListLabels(ctx, "owner/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "list labels")
	mockRunner.AssertExpectations(t)
}

func TestListLabels_ParseError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/owner/repo/labels", "--paginate"}).
		Return([]byte("not json{{"), nil)

	result, err := client.ListLabels(ctx, "owner/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parse labels")
	mockRunner.AssertExpectations(t)
}

func TestSyncLabels_NewLabel(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// PATCH returns 404 (label doesn't exist yet) → fall back to POST
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/labels/bug", "--method", "PATCH", "--input", "-"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/labels", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	labels := []Label{{Name: "bug", Color: "d73a4a"}}
	err := client.SyncLabels(ctx, "owner/repo", labels)
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestSyncLabels_ExistingLabel(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// PATCH succeeds directly — no ListLabels needed with upsert pattern
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/labels/bug", "--method", "PATCH", "--input", "-"}).
		Return([]byte("{}"), nil)

	labels := []Label{{Name: "bug", Color: "d73a4a"}}
	err := client.SyncLabels(ctx, "owner/repo", labels)
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestSyncLabels_PatchNonFoundError_LogsAndContinues(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// PATCH fails with a non-404 error; should log warning and return nil (not propagate error)
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/labels/bug", "--method", "PATCH", "--input", "-"}).
		Return(nil, errTestLabelsFetchError)

	labels := []Label{{Name: "bug", Color: "d73a4a"}}
	err := client.SyncLabels(ctx, "owner/repo", labels)
	require.NoError(t, err) // errors are logged, not returned
	mockRunner.AssertExpectations(t)
}

func TestSyncLabels_MixedNew_And_Existing(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// "bug" exists: PATCH succeeds; "feature" is new: PATCH returns 404, POST creates it
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/labels/bug", "--method", "PATCH", "--input", "-"}).
		Return([]byte("{}"), nil)
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/labels/feature", "--method", "PATCH", "--input", "-"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/labels", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	labels := []Label{{Name: "bug", Color: "d73a4a"}, {Name: "feature", Color: "a2eeef"}}
	err := client.SyncLabels(ctx, "owner/repo", labels)
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestSetTopics_Success(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/topics", "--method", "PUT", "--input", "-"}).
		Return([]byte("{}"), nil)

	err := client.SetTopics(ctx, "owner/repo", []string{"go", "library"})
	require.NoError(t, err)
	mockRunner.AssertExpectations(t)
}

func TestSetTopics_RunnerError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/owner/repo/topics", "--method", "PUT", "--input", "-"}).
		Return(nil, errTestTopicsError)

	err := client.SetTopics(ctx, "owner/repo", []string{"go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "topics error")
	mockRunner.AssertExpectations(t)
}
