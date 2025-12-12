package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCommitPrompt_NilContext(t *testing.T) {
	result := BuildCommitPrompt(nil)
	assert.Empty(t, result)
}

func TestBuildCommitPrompt_EmptyContext(t *testing.T) {
	ctx := &CommitContext{}
	result := BuildCommitPrompt(ctx)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Source:")
	assert.Contains(t, result, "Target:")
	assert.Contains(t, result, "Changed Files (0 files)")
}

func TestBuildCommitPrompt_BasicContext(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/source-repo",
		TargetRepo: "owner/target-repo",
		ChangedFiles: []FileChange{
			{Path: "README.md", ChangeType: "modified"},
			{Path: "main.go", ChangeType: "added"},
		},
		DiffSummary: "diff content here",
	}

	result := BuildCommitPrompt(ctx)

	assert.Contains(t, result, "Source: owner/source-repo")
	assert.Contains(t, result, "Target: owner/target-repo")
	assert.Contains(t, result, "Changed Files (2 files)")
	assert.Contains(t, result, "README.md (modified)")
	assert.Contains(t, result, "main.go (added)")
	assert.Contains(t, result, "diff content here")
	assert.Contains(t, result, "conventional commits format")
}

func TestBuildCommitPrompt_WithGroupName(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		GroupName:  "workflows",
	}

	result := BuildCommitPrompt(ctx)

	assert.Contains(t, result, "Group: workflows")
}

func TestBuildCommitPrompt_WithoutGroupName(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	result := BuildCommitPrompt(ctx)

	// Should not contain the Group line at all
	assert.NotContains(t, result, "Group:")
}

func TestBuildCommitPrompt_WithoutDiffSummary(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	result := BuildCommitPrompt(ctx)

	// Should not contain Diff Summary section
	assert.NotContains(t, result, "## Diff Summary")
}

func TestBuildCommitPrompt_SpecialCharactersInRepoNames(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/repo-with-dashes_and_underscores",
		TargetRepo: "org-name/target.repo",
		ChangedFiles: []FileChange{
			{Path: "path/to/file with spaces.txt", ChangeType: "modified"},
			{Path: "special[chars].go", ChangeType: "deleted"},
		},
	}

	result := BuildCommitPrompt(ctx)

	// Should contain all special characters properly
	assert.Contains(t, result, "repo-with-dashes_and_underscores")
	assert.Contains(t, result, "org-name/target.repo")
	assert.Contains(t, result, "file with spaces.txt")
	assert.Contains(t, result, "special[chars].go")
}

func TestBuildCommitPrompt_LargeFileCount(t *testing.T) {
	files := make([]FileChange, 100)
	for i := 0; i < 100; i++ {
		files[i] = FileChange{
			Path:       "file" + string(rune('0'+i%10)) + ".go",
			ChangeType: "modified",
		}
	}

	ctx := &CommitContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: files,
	}

	result := BuildCommitPrompt(ctx)

	assert.Contains(t, result, "Changed Files (100 files)")
}

func TestBuildCommitPrompt_AllChangeTypes(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: "added.go", ChangeType: "added"},
			{Path: "modified.go", ChangeType: "modified"},
			{Path: "deleted.go", ChangeType: "deleted"},
			{Path: "renamed.go", ChangeType: "renamed"},
		},
	}

	result := BuildCommitPrompt(ctx)

	assert.Contains(t, result, "added.go (added)")
	assert.Contains(t, result, "modified.go (modified)")
	assert.Contains(t, result, "deleted.go (deleted)")
	assert.Contains(t, result, "renamed.go (renamed)")
}

func TestBuildCommitPrompt_ContainsRequirements(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	result := BuildCommitPrompt(ctx)

	// Should contain key requirements
	assert.Contains(t, result, "conventional commits format")
	assert.Contains(t, result, "sync")
	assert.Contains(t, result, "50 characters")
	assert.Contains(t, result, "imperative mood")
}

func TestBuildCommitPrompt_ContainsExamples(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	result := BuildCommitPrompt(ctx)

	// Should contain good and bad examples
	assert.Contains(t, result, "Good Examples")
	assert.Contains(t, result, "Bad Examples")
	assert.Contains(t, result, "sync: update README.md")
}

//nolint:gosmopolitan // intentional unicode test data
func TestBuildCommitPrompt_UnicodeContent(t *testing.T) {
	ctx := &CommitContext{
		SourceRepo: "owner/レポジトリ",
		TargetRepo: "owner/目标仓库",
		ChangedFiles: []FileChange{
			{Path: "文件.go", ChangeType: "modified"},
			{Path: "файл.txt", ChangeType: "added"},
			{Path: "αβγ.md", ChangeType: "deleted"},
		},
		DiffSummary: "这是一些差异 🎉",
	}

	result := BuildCommitPrompt(ctx)

	assert.Contains(t, result, "レポジトリ")
	assert.Contains(t, result, "目标仓库")
	assert.Contains(t, result, "文件.go")
	assert.Contains(t, result, "файл.txt")
	assert.Contains(t, result, "αβγ.md")
	assert.Contains(t, result, "🎉")
}

func TestBuildCommitPrompt_LongDiffSummary(t *testing.T) {
	// Create a very long diff summary
	longDiff := strings.Repeat("diff --git a/file.go b/file.go\n", 100)

	ctx := &CommitContext{
		SourceRepo:  "owner/source",
		TargetRepo:  "owner/target",
		DiffSummary: longDiff,
	}

	result := BuildCommitPrompt(ctx)

	// Should contain the full diff
	assert.Contains(t, result, longDiff)
}

func TestGetCommitPromptTmpl_Caching(t *testing.T) {
	// Get template twice - should return same instance
	tmpl1 := getCommitPromptTmpl()
	tmpl2 := getCommitPromptTmpl()

	assert.Same(t, tmpl1, tmpl2, "Template should be cached and return same instance")
}
