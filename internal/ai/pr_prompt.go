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

## Output Format Requirements (MUST follow)
1. Your response MUST contain exactly 4 markdown sections with ## headers
2. Each section MUST have 2-4 bullet points starting with *
3. DO NOT output a single-line commit message
4. DO NOT use conventional commit format (type(scope): message)
5. Be descriptive and detailed - this is a PR description, not a commit message
6. Start your response immediately with "## What Changed" - no preamble
7. DO NOT mention specific repository names - describe changes in terms of files/diff only, as this message may be used across multiple repositories

## Example of CORRECT PR Body Format
## What Changed
* Updated 4 GitHub workflow files to use mage-x v1.8.15
* Modified CI pipeline configuration for improved build performance
* Synchronized security scanning workflow with upstream changes

## Why It Was Necessary
* Incorporates latest workflow improvements and bug fixes
* Ensures consistent CI/CD behavior with upstream changes

## Testing Performed
* Validated YAML syntax of workflow files
* Verified no breaking changes in workflow configurations

## Impact / Risk
* **Low Risk**: Standard CI workflow updates
* No breaking changes to existing functionality

## WRONG Output Formats (DO NOT use)
- sync(ci): update workflows (THIS IS A COMMIT MESSAGE FORMAT - WRONG)
- sync: update files from source repository (TOO SHORT - WRONG)
- Any single line without ## headers (WRONG)

## Your Task
Generate a PR description following the CORRECT format above. You MUST include these exact 4 sections:

1. **## What Changed** - Technical summary based on the actual files and diff. Be specific about what files changed and why.

2. **## Why It Was Necessary** - Explain the purpose of these changes without mentioning specific repository names.

3. **## Testing Performed** - List validation steps appropriate for these specific file types.

4. **## Impact / Risk** - Assess risk based on what files actually changed.

IMPORTANT: Output ONLY the PR body with ## headers. Do NOT include code blocks. Start immediately with "## What Changed".
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
