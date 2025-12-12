package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildPRPrompt_NilContext(t *testing.T) {
	result := BuildPRPrompt(nil)
	assert.Empty(t, result)
}

func TestBuildPRPrompt_EmptyContext(t *testing.T) {
	ctx := &PRContext{}
	result := BuildPRPrompt(ctx)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Source Repository")
	assert.Contains(t, result, "Target Repository")
	assert.Contains(t, result, "Files Changed (0 files)")
}

func TestBuildPRPrompt_BasicContext(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/source-repo",
		TargetRepo: "owner/target-repo",
		CommitSHA:  "abc123def",
		ChangedFiles: []FileChange{
			{Path: "README.md", ChangeType: "modified", LinesAdded: 10, LinesRemoved: 5},
			{Path: "main.go", ChangeType: "added", LinesAdded: 100, LinesRemoved: 0},
		},
		DiffSummary: "diff content here",
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, "owner/source-repo")
	assert.Contains(t, result, "owner/target-repo")
	assert.Contains(t, result, "abc123def")
	assert.Contains(t, result, "Files Changed (2 files)")
	assert.Contains(t, result, "README.md (modified, +10/-5 lines)")
	assert.Contains(t, result, "main.go (added, +100/-0 lines)")
	assert.Contains(t, result, "diff content here")
}

func TestBuildPRPrompt_WithCustomGuidelines(t *testing.T) {
	ctx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		PRGuidelines: "Custom guideline 1\nCustom guideline 2",
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, "Custom guideline 1")
	assert.Contains(t, result, "Custom guideline 2")
}

func TestBuildPRPrompt_DefaultGuidelines(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	result := BuildPRPrompt(ctx)

	// Should contain default guidelines sections
	assert.Contains(t, result, "What Changed")
	assert.Contains(t, result, "Why It Was Necessary")
	assert.Contains(t, result, "Testing Performed")
	assert.Contains(t, result, "Impact / Risk")
}

func TestBuildPRPrompt_WithoutDiffSummary(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	result := BuildPRPrompt(ctx)

	// Should not contain Diff Summary section markers
	assert.NotContains(t, result, "```diff")
}

func TestBuildPRPrompt_WithDiffSummary(t *testing.T) {
	ctx := &PRContext{
		SourceRepo:  "owner/source",
		TargetRepo:  "owner/target",
		DiffSummary: "some diff content",
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, "## Diff Summary")
	assert.Contains(t, result, "```diff")
	assert.Contains(t, result, "some diff content")
}

func TestBuildPRPrompt_SpecialCharactersInRepoNames(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/repo-with-dashes_and_underscores",
		TargetRepo: "org-name/target.repo",
		CommitSHA:  "a1b2c3d4e5f6",
		ChangedFiles: []FileChange{
			{Path: "path/to/file with spaces.txt", ChangeType: "modified"},
			{Path: "special[chars].go", ChangeType: "deleted"},
		},
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, "repo-with-dashes_and_underscores")
	assert.Contains(t, result, "org-name/target.repo")
	assert.Contains(t, result, "file with spaces.txt")
	assert.Contains(t, result, "special[chars].go")
}

func TestBuildPRPrompt_LargeFileCount(t *testing.T) {
	files := make([]FileChange, 100)
	for i := 0; i < 100; i++ {
		files[i] = FileChange{
			Path:         "file" + string(rune('0'+i%10)) + ".go",
			ChangeType:   "modified",
			LinesAdded:   i,
			LinesRemoved: i / 2,
		}
	}

	ctx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		ChangedFiles: files,
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, "Files Changed (100 files)")
}

func TestBuildPRPrompt_FilesWithLineStats(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: "added.go", ChangeType: "added", LinesAdded: 50, LinesRemoved: 0},
			{Path: "modified.go", ChangeType: "modified", LinesAdded: 20, LinesRemoved: 10},
			{Path: "deleted.go", ChangeType: "deleted", LinesAdded: 0, LinesRemoved: 100},
			{Path: "nochange.go", ChangeType: "modified", LinesAdded: 0, LinesRemoved: 0},
		},
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, "added.go (added, +50/-0 lines)")
	assert.Contains(t, result, "modified.go (modified, +20/-10 lines)")
	assert.Contains(t, result, "deleted.go (deleted, +0/-100 lines)")
	// File with no line changes should not show line stats
	assert.Contains(t, result, "nochange.go (modified)")
	assert.NotContains(t, result, "nochange.go (modified, +0/-0 lines)")
}

func TestBuildPRPrompt_ContainsFormatRequirements(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	result := BuildPRPrompt(ctx)

	// Should contain format requirements
	assert.Contains(t, result, "4 markdown sections")
	assert.Contains(t, result, "DO NOT output a single-line commit message")
	assert.Contains(t, result, "DO NOT use conventional commit format")
}

func TestBuildPRPrompt_ContainsExamples(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
	}

	result := BuildPRPrompt(ctx)

	// Should contain example format
	assert.Contains(t, result, "CORRECT PR Body Format")
	assert.Contains(t, result, "WRONG Output Formats")
}

//nolint:gosmopolitan // intentional unicode test data
func TestBuildPRPrompt_UnicodeContent(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/レポジトリ",
		TargetRepo: "owner/目标仓库",
		CommitSHA:  "unicode123",
		ChangedFiles: []FileChange{
			{Path: "文件.go", ChangeType: "modified", LinesAdded: 5, LinesRemoved: 3},
			{Path: "файл.txt", ChangeType: "added", LinesAdded: 10, LinesRemoved: 0},
			{Path: "αβγ.md", ChangeType: "deleted", LinesAdded: 0, LinesRemoved: 20},
		},
		DiffSummary:  "这是一些差异 🎉",
		PRGuidelines: "指南说明 📋",
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, "レポジトリ")
	assert.Contains(t, result, "目标仓库")
	assert.Contains(t, result, "文件.go")
	assert.Contains(t, result, "файл.txt")
	assert.Contains(t, result, "αβγ.md")
	assert.Contains(t, result, "🎉")
	assert.Contains(t, result, "📋")
}

func TestBuildPRPrompt_LongDiffSummary(t *testing.T) {
	longDiff := strings.Repeat("diff --git a/file.go b/file.go\n+added line\n-removed line\n", 100)

	ctx := &PRContext{
		SourceRepo:  "owner/source",
		TargetRepo:  "owner/target",
		DiffSummary: longDiff,
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, longDiff)
}

func TestBuildPRPrompt_LongGuidelines(t *testing.T) {
	longGuidelines := strings.Repeat("Guideline rule number X: Do something specific.\n", 50)

	ctx := &PRContext{
		SourceRepo:   "owner/source",
		TargetRepo:   "owner/target",
		PRGuidelines: longGuidelines,
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, longGuidelines)
}

func TestGetPRPromptTmpl_Caching(t *testing.T) {
	// Get template twice - should return same instance
	tmpl1 := getPRPromptTmpl()
	tmpl2 := getPRPromptTmpl()

	assert.Same(t, tmpl1, tmpl2, "Template should be cached and return same instance")
}

func TestBuildPRPrompt_EmptyCommitSHA(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		CommitSHA:  "",
	}

	result := BuildPRPrompt(ctx)

	// Should still contain the Commit SHA label, just with empty value
	assert.Contains(t, result, "Commit SHA")
}

func TestBuildPRPrompt_AllChangeTypes(t *testing.T) {
	ctx := &PRContext{
		SourceRepo: "owner/source",
		TargetRepo: "owner/target",
		ChangedFiles: []FileChange{
			{Path: "added.go", ChangeType: "added"},
			{Path: "modified.go", ChangeType: "modified"},
			{Path: "deleted.go", ChangeType: "deleted"},
			{Path: "renamed.go", ChangeType: "renamed"},
			{Path: "copied.go", ChangeType: "copied"},
		},
	}

	result := BuildPRPrompt(ctx)

	assert.Contains(t, result, "added.go (added)")
	assert.Contains(t, result, "modified.go (modified)")
	assert.Contains(t, result, "deleted.go (deleted)")
	assert.Contains(t, result, "renamed.go (renamed)")
	assert.Contains(t, result, "copied.go (copied)")
}
