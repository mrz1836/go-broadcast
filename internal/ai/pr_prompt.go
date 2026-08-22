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

	// FullDiff is the complete, untruncated diff. It is NOT rendered into the
	// prompt; it is used to extract authoritative facts and to guard the response
	// against hallucinated version numbers. Falls back to DiffSummary when empty.
	FullDiff string

	// VerifiedChanges is a machine-extracted, authoritative Markdown list of
	// key/value (version) changes. When present, the model is instructed to use
	// these exact values and invent nothing.
	VerifiedChanges string

	// OmittedFiles lists files whose diff was trimmed or omitted from DiffSummary
	// (e.g., large generated files). The model is told to describe these generically.
	OmittedFiles []string

	// PRGuidelines is the loaded PR guidelines (optional, uses fallback if empty).
	PRGuidelines string
}

// prPromptTemplate is the template for PR body generation prompts.
// IMPORTANT: The diff is placed FIRST to ensure the AI focuses on actual changes, not patterns.
const prPromptTemplate = `You are generating a PR description for an automated file synchronization.

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
{{ else }}
## NO DIFF AVAILABLE - DESCRIBE FROM THE FILE LIST BELOW
No line-level diff is available for this change, but the file list below IS accurate and reliable.
Write a genuinely useful description based on the file paths, their change types, and line counts:
- Infer each file's PURPOSE from its path (e.g. ".github/workflows/*.yml" = CI/CD workflow definition,
  "*.md" = documentation, "*_test.go" = tests, "*.env" = environment/config, "Dockerfile" = container build).
- Group related files and summarize at a high level (e.g. "Updated 12 GitHub Actions workflow and configuration files").
- Use the change type (added/modified/deleted) and +/- line counts to convey scale.

Because this is an automated file synchronization, describe the later sections in those terms:
- Why It Was Necessary: keeping the target repository aligned with its source repository.
- Testing Performed: sync-level validation (config and file mappings validated, transformations applied,
  no unintended changes) - do NOT claim you executed or ran the code.
- Impact / Risk: scope is limited to the synchronized files; assess risk from the file types and counts.

NEVER do the following - they produce a useless description:
- Do NOT say the diff is empty, missing, truncated, unavailable, or not visible.
- Do NOT say you cannot determine, verify, assess, or confirm the changes.
- Do NOT output "Unknown", "N/A", or any apology about missing information.
- Do NOT invent specific version numbers, variable names, or values you cannot see.
The reader wants a useful summary of WHAT these files are, not a report about what you could not see.
{{ end }}
{{ if .VerifiedChanges }}## VERIFIED VERSION & CONFIG CHANGES - AUTHORITATIVE (machine-extracted)
These are the exact configuration/version changes in this sync. Use these EXACT values.
Never state a version, key, or value that is not shown here or visible in the diff above.
{{ .VerifiedChanges }}

{{ end }}## Files Changed ({{ len .ChangedFiles }} files)
{{ range .ChangedFiles -}}
- {{ .Path }} ({{ .ChangeType }}, +{{ .LinesAdded }}/-{{ .LinesRemoved }})
{{ end }}{{ if .OmittedFiles }}
NOTE: The diff for these files was trimmed or omitted to save space. Describe them ONLY
generically by file type and change type - do NOT invent version numbers or values for them:
{{ range .OmittedFiles -}}
- {{ . }}
{{ end }}{{ end }}
{{ if .PRGuidelines }}## Additional Guidelines
{{ .PRGuidelines }}

{{ end }}## Output Format
Generate a PR description with these 4 sections. Start immediately with "## What Changed".

1. **## What Changed** - {{ if .VerifiedChanges }}Open with one short sentence, then present the VERIFIED VERSION & CONFIG CHANGES above as a single clean bulleted group using these EXACT values (group related bumps; keep each on its own bullet). Then cover any non-config changes (features, refactors, fixes, security) as their own bullets. Do NOT repeat a change in both the list and the prose, and never state a version that is not in that list or the diff.{{ else if .DiffSummary }}Describe ONLY what the diff shows. Quote version numbers exactly as they appear.{{ else }}Describe the change concretely from the file paths and change types above.{{ end }}
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
