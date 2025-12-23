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
// IMPORTANT: The diff is placed FIRST to ensure the AI focuses on actual changes, not patterns.
const prPromptTemplate = `You are generating a PR description. Your ONLY source of truth is the diff below.

{{ if .DiffSummary }}## ACTUAL DIFF - THIS IS YOUR ONLY SOURCE OF TRUTH
Read this diff carefully. You may ONLY describe changes that appear here.
` + "```diff" + `
{{ .DiffSummary }}
` + "```" + `

CRITICAL INSTRUCTIONS:
- The diff above shows EXACTLY what changed (lines starting with - were removed, + were added)
- You MUST describe ONLY what you see in this diff
- If you cannot see a specific version number change in the diff, DO NOT mention it
- If the diff shows "v1.12.1" changing to "v1.12.2", say exactly that - not "v1.11.0 to v1.12.2"
{{ else }}
## WARNING: No Diff Content Available
The diff is empty. Use ONLY generic descriptions like "Synchronized configuration files".
Do NOT invent specific changes - you have no information about what changed.
{{ end }}

## Files Changed ({{ len .ChangedFiles }} files)
{{ range .ChangedFiles -}}
- {{ .Path }}
{{ end }}

## HALLUCINATION PREVENTION
You are prone to hallucinating changes that are not in the diff. DO NOT:
- Mention GO_COVERAGE_VERSION, GO_PRE_COMMIT_VERSION, or other variables unless they appear in the diff
- Describe version changes that are not visible in the diff above
- Assume what a file contains based on its name - only describe what the diff shows
- Add details that sound plausible but are not in the diff

If the diff shows ONLY:
- MAGE_X_VERSION changing from v1.12.1 to v1.12.2
- A comment being modified
- permissions: contents: read being added

Then describe ONLY those changes. Nothing else.

{{ if .PRGuidelines }}## Additional Guidelines
{{ .PRGuidelines }}

{{ end }}## Output Format
Generate a PR description with these 4 sections. Start immediately with "## What Changed".

1. **## What Changed** - Describe ONLY what the diff shows. Quote version numbers exactly as they appear.
2. **## Why It Was Necessary** - Brief explanation (2-3 bullets)
3. **## Testing Performed** - Validation steps (2-3 bullets)
4. **## Impact / Risk** - Risk assessment (2-3 bullets)

Each section needs 2-4 bullet points starting with *.
Do NOT mention specific repository names.
Output ONLY the PR body - no preamble, no code blocks around your response.
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
		// Log template error (usually indicates a code bug) and use fallback
		logConfigWarning("PR prompt template execution failed: %v", err)
		return fmt.Sprintf("Generate a PR description for syncing %d files from %s to %s.",
			len(ctx.ChangedFiles), ctx.SourceRepo, ctx.TargetRepo)
	}

	return buf.String()
}
