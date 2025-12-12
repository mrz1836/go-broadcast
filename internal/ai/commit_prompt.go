package ai

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"
)

// commitPromptTmpl is the cached parsed template for commit prompts.
//
//nolint:gochecknoglobals // Intentional caching for performance - parsed once per process
var (
	commitPromptTmpl     *template.Template
	commitPromptTmplOnce sync.Once
)

// CommitContext contains context for commit message generation.
type CommitContext struct {
	// SourceRepo is the source repository name (e.g., "owner/repo").
	SourceRepo string

	// TargetRepo is the target repository name (e.g., "owner/repo").
	TargetRepo string

	// ChangedFiles is the list of files changed in this sync.
	ChangedFiles []FileChange

	// DiffSummary is the truncated diff content for AI context.
	DiffSummary string

	// GroupName is optional: for multi-group syncs, identifies the sync group.
	GroupName string
}

// commitPromptTemplate is the template for commit message generation prompts.
const commitPromptTemplate = `Generate a git commit message for a repository synchronization.

## Context
- Source: {{ .SourceRepo }}
- Target: {{ .TargetRepo }}
{{ if .GroupName }}- Group: {{ .GroupName }}{{ end }}

## Changed Files ({{ len .ChangedFiles }} files)
{{ range .ChangedFiles -}}
- {{ .Path }} ({{ .ChangeType }})
{{ end }}

{{ if .DiffSummary }}## Diff Summary
{{ .DiffSummary }}
{{ end }}

## Requirements (MUST follow)
1. Use conventional commits format: type(scope): subject
2. Use "sync" as the type (per go-broadcast conventions)
3. Subject line MUST be under 50 characters
4. Be specific about what changed, not generic
5. Use imperative mood ("update", "add", "remove")
6. NO trailing period
7. NO body - single line only

## Good Examples
- sync: update README.md from source repository
- sync: update 3 workflow files for CI improvements
- sync(docs): synchronize API documentation
- sync: add new linter configuration files

## Bad Examples (DO NOT use these patterns)
- sync: update files from source repository (too generic)
- sync: synchronize repository (meaningless)
- Updated some files (not conventional commit format)
- sync: Update README. (wrong case, has period)

Generate ONLY the commit message, nothing else. Single line only. No quotes or formatting.
`

// getCommitPromptTmpl returns the cached parsed template for commit prompts.
// Uses sync.Once to ensure template is parsed only once.
func getCommitPromptTmpl() *template.Template {
	commitPromptTmplOnce.Do(func() {
		commitPromptTmpl = template.Must(template.New("commit_prompt").Parse(commitPromptTemplate))
	})
	return commitPromptTmpl
}

// BuildCommitPrompt constructs the prompt for commit message generation.
// Uses text/template to render the prompt with CommitContext data.
func BuildCommitPrompt(ctx *CommitContext) string {
	if ctx == nil {
		return ""
	}

	var buf bytes.Buffer
	if err := getCommitPromptTmpl().Execute(&buf, ctx); err != nil {
		// Fallback to simple prompt on template error
		return fmt.Sprintf("Generate a sync commit message for %d files from %s to %s. Use conventional commits format with sync: prefix.",
			len(ctx.ChangedFiles), ctx.SourceRepo, ctx.TargetRepo)
	}

	return buf.String()
}
