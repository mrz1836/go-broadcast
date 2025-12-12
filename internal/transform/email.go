package transform

import (
	"path/filepath"
	"regexp"
	"strings"
)

// emailTransformer replaces email addresses in specific contexts
type emailTransformer struct {
	cache *RegexCache
}

// NewEmailTransformer creates a new email address transformer
func NewEmailTransformer() Transformer {
	return &emailTransformer{
		cache: getDefaultCache(),
	}
}

// Name returns the name of this transformer
func (e *emailTransformer) Name() string {
	return "email-address-replacer"
}

// Transform applies email address replacement to the content
func (e *emailTransformer) Transform(content []byte, ctx Context) ([]byte, error) {
	result := content

	// Transform security email if both source and target are configured and different
	if ctx.SourceSecurityEmail != "" && ctx.TargetSecurityEmail != "" &&
		ctx.SourceSecurityEmail != ctx.TargetSecurityEmail {
		result = e.replaceEmail(result, ctx.SourceSecurityEmail, ctx.TargetSecurityEmail, ctx.FilePath)
	}

	// Transform support email if both source and target are configured and different
	if ctx.SourceSupportEmail != "" && ctx.TargetSupportEmail != "" &&
		ctx.SourceSupportEmail != ctx.TargetSupportEmail {
		result = e.replaceEmail(result, ctx.SourceSupportEmail, ctx.TargetSupportEmail, ctx.FilePath)
	}

	return result, nil
}

// escapeReplacement escapes $ characters in replacement strings to prevent
// regex backreference injection. In Go's regexp.ReplaceAll, $ is special:
// $1 means captured group 1. To use a literal $, it must be escaped as $$.
func escapeReplacement(s string) string {
	return strings.ReplaceAll(s, "$", "$$")
}

// replaceEmail replaces all occurrences of source email with target email in various contexts
func (e *emailTransformer) replaceEmail(content []byte, sourceEmail, targetEmail, filePath string) []byte {
	fileExt := strings.ToLower(filepath.Ext(filePath))

	switch fileExt {
	case ".md", ".txt", ".rst":
		return e.replaceEmailMarkdown(content, sourceEmail, targetEmail)
	case ".yaml", ".yml":
		return e.replaceEmailYAML(content, sourceEmail, targetEmail)
	case ".json":
		return e.replaceEmailJSON(content, sourceEmail, targetEmail)
	case ".html", ".htm":
		return e.replaceEmailHTML(content, sourceEmail, targetEmail)
	default:
		return e.replaceEmailGeneral(content, sourceEmail, targetEmail)
	}
}

// replaceEmailMarkdown handles email replacement in Markdown files
func (e *emailTransformer) replaceEmailMarkdown(content []byte, sourceEmail, targetEmail string) []byte {
	// Escape target email for safe use in replacement strings
	safeTarget := escapeReplacement(targetEmail)

	patterns := []struct {
		pattern     string
		replacement string
	}{
		// Markdown link: [text](mailto:email)
		{
			pattern:     `\[([^\]]+)\]\(mailto:` + regexp.QuoteMeta(sourceEmail) + `\)`,
			replacement: `[$1](mailto:` + safeTarget + `)`,
		},
		// Markdown link: [email](mailto:email)
		{
			pattern:     `\[` + regexp.QuoteMeta(sourceEmail) + `\]\(mailto:` + regexp.QuoteMeta(sourceEmail) + `\)`,
			replacement: `[` + safeTarget + `](mailto:` + safeTarget + `)`,
		},
		// mailto: link
		{
			pattern:     `mailto:` + regexp.QuoteMeta(sourceEmail),
			replacement: `mailto:` + safeTarget,
		},
		// Plain email address
		{
			pattern:     `\b` + regexp.QuoteMeta(sourceEmail) + `\b`,
			replacement: safeTarget,
		},
	}

	result := content
	for _, p := range patterns {
		re, err := e.cache.CompileRegex(p.pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		result = re.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// replaceEmailYAML handles email replacement in YAML files
func (e *emailTransformer) replaceEmailYAML(content []byte, sourceEmail, targetEmail string) []byte {
	// Escape target email for safe use in replacement strings
	safeTarget := escapeReplacement(targetEmail)

	patterns := []struct {
		pattern     string
		replacement string
	}{
		// Quoted email: "email@example.com"
		{
			pattern:     `"` + regexp.QuoteMeta(sourceEmail) + `"`,
			replacement: `"` + safeTarget + `"`,
		},
		// Single-quoted email: 'email@example.com'
		{
			pattern:     `'` + regexp.QuoteMeta(sourceEmail) + `'`,
			replacement: `'` + safeTarget + `'`,
		},
		// Unquoted email after key
		{
			pattern:     `(:\s*)` + regexp.QuoteMeta(sourceEmail) + `(\s|$)`,
			replacement: `${1}` + safeTarget + `${2}`,
		},
	}

	result := content
	for _, p := range patterns {
		re, err := e.cache.CompileRegex(p.pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		result = re.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// replaceEmailJSON handles email replacement in JSON files
func (e *emailTransformer) replaceEmailJSON(content []byte, sourceEmail, targetEmail string) []byte {
	// Escape target email for safe use in replacement strings
	safeTarget := escapeReplacement(targetEmail)

	// JSON always has quoted strings
	pattern := `"` + regexp.QuoteMeta(sourceEmail) + `"`
	re, err := e.cache.CompileRegex(pattern)
	if err != nil {
		return content // Return unchanged on invalid pattern
	}
	return re.ReplaceAll(content, []byte(`"`+safeTarget+`"`))
}

// replaceEmailHTML handles email replacement in HTML files
func (e *emailTransformer) replaceEmailHTML(content []byte, sourceEmail, targetEmail string) []byte {
	// Escape target email for safe use in replacement strings
	safeTarget := escapeReplacement(targetEmail)

	patterns := []struct {
		pattern     string
		replacement string
	}{
		// <a href="mailto:email">...</a>
		{
			pattern:     `<a\s+href="mailto:` + regexp.QuoteMeta(sourceEmail) + `"`,
			replacement: `<a href="mailto:` + safeTarget + `"`,
		},
		// mailto:email in href attributes
		{
			pattern:     `mailto:` + regexp.QuoteMeta(sourceEmail),
			replacement: `mailto:` + safeTarget,
		},
		// Plain email with word boundaries
		{
			pattern:     `\b` + regexp.QuoteMeta(sourceEmail) + `\b`,
			replacement: safeTarget,
		},
	}

	result := content
	for _, p := range patterns {
		re, err := e.cache.CompileRegex(p.pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		result = re.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// replaceEmailGeneral handles email replacement for general file types
func (e *emailTransformer) replaceEmailGeneral(content []byte, sourceEmail, targetEmail string) []byte {
	// Escape target email for safe use in replacement strings
	safeTarget := escapeReplacement(targetEmail)

	// Simple replacement with word boundaries
	pattern := `\b` + regexp.QuoteMeta(sourceEmail) + `\b`
	re, err := e.cache.CompileRegex(pattern)
	if err != nil {
		return content // Return unchanged on invalid pattern
	}
	return re.ReplaceAll(content, []byte(safeTarget))
}
