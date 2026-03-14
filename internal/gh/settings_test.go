package gh

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("nil include and exclude", func(t *testing.T) {
		t.Parallel()

		rs := BuildBranchRuleset("nil-conditions", nil, nil, []string{"deletion"})

		require.NotNil(t, rs.Conditions)
		assert.Nil(t, rs.Conditions.RefName.Include)
		assert.Nil(t, rs.Conditions.RefName.Exclude)
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

	t.Run("nil include and exclude", func(t *testing.T) {
		t.Parallel()

		rs := BuildTagRuleset("nil-conditions", nil, nil, []string{"update"})

		require.NotNil(t, rs.Conditions)
		assert.Nil(t, rs.Conditions.RefName.Include)
		assert.Nil(t, rs.Conditions.RefName.Exclude)
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
