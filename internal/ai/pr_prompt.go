package ai

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"
)

// prPromptTmpl is the cached parsed template for PR prompts.
//
//nolint:gochecknoglobals // Intentional caching for performance - parsed once per process
var (
	prPromptTmpl     *template.Template
	prPromptTmplOnce sync.Once
)

// PRContext contains all context needed to generate a PR body.
type PRContext struct {
	// SourceRepo is the source repository name (e.g., "owner/repo").
	SourceRepo string

	// TargetRepo is the target repository name (e.g., "owner/repo").
	TargetRepo string

	// CommitSHA is the source commit SHA being synced.
	CommitSHA string

	// ChangedFiles is the list of files changed in this sync.
	ChangedFiles []FileChange

	// DiffSummary is the truncated diff content for AI context.
	DiffSummary string

	// PRGuidelines is the loaded PR guidelines (optional, uses fallback if empty).
	PRGuidelines string
}

// prPromptTemplate is the template for PR body generation prompts.
const prPromptTemplate = `You are a technical writer generating a pull request description for go-broadcast repository synchronization.

## PR Guidelines to Follow
{{ if .PRGuidelines }}{{ .PRGuidelines }}{{ else }}
Every PR must include these four sections:
1. **What Changed** - Technical summary of changes
2. **Why It Was Necessary** - Context and motivation
3. **Testing Performed** - Validation steps taken
4. **Impact / Risk** - Risk assessment and breaking changes
{{ end }}

## Synchronization Context
- **Source Repository**: {{ .SourceRepo }}
- **Target Repository**: {{ .TargetRepo }}
- **Commit SHA**: {{ .CommitSHA }}

## Files Changed ({{ len .ChangedFiles }} files)
{{ range .ChangedFiles -}}
- {{ .Path }} ({{ .ChangeType }}{{ if or (gt .LinesAdded 0) (gt .LinesRemoved 0) }}, +{{ .LinesAdded }}/-{{ .LinesRemoved }} lines{{ end }})
{{ end }}

{{ if .DiffSummary }}## Diff Summary
` + "```diff" + `
{{ .DiffSummary }}
` + "```" + `
{{ end }}

## Your Task
Generate a PR description with these exact sections:

1. **## What Changed** - Technical summary based on the actual files and diff above. Be specific about what files changed and what the changes do.

2. **## Why It Was Necessary** - Explain the purpose of this synchronization from the source repository.

3. **## Testing Performed** - List validation steps appropriate for these specific file types and changes.

4. **## Impact / Risk** - Assess risk based on what files actually changed (config files, documentation, code, etc.).

Format with ## headers. Be specific and accurate based on the changes shown. Keep each section concise (2-4 bullet points).
Do NOT include code blocks in your response. Output the PR body directly.
`

// getPRPromptTmpl returns the cached parsed template for PR prompts.
// Uses sync.Once to ensure template is parsed only once.
func getPRPromptTmpl() *template.Template {
	prPromptTmplOnce.Do(func() {
		prPromptTmpl = template.Must(template.New("pr_prompt").Parse(prPromptTemplate))
	})
	return prPromptTmpl
}

// BuildPRPrompt constructs the full prompt for PR body generation.
// Uses text/template to render the prompt with PRContext data.
func BuildPRPrompt(ctx *PRContext) string {
	if ctx == nil {
		return ""
	}

	var buf bytes.Buffer
	if err := getPRPromptTmpl().Execute(&buf, ctx); err != nil {
		// Fallback to simple prompt on template error
		return fmt.Sprintf("Generate a PR description for syncing %d files from %s to %s.",
			len(ctx.ChangedFiles), ctx.SourceRepo, ctx.TargetRepo)
	}

	return buf.String()
}
