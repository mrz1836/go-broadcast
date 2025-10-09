package transform

import (
	"path/filepath"
	"regexp"
	"strings"
)

// emailTransformer replaces email addresses in specific contexts
type emailTransformer struct{}

// NewEmailTransformer creates a new email address transformer
func NewEmailTransformer() Transformer {
	return &emailTransformer{}
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
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Markdown link: [text](mailto:email)
		{
			regex:       regexp.MustCompile(`\[([^\]]+)\]\(mailto:` + regexp.QuoteMeta(sourceEmail) + `\)`),
			replacement: `[$1](mailto:` + targetEmail + `)`,
		},
		// Markdown link: [email](mailto:email)
		{
			regex:       regexp.MustCompile(`\[` + regexp.QuoteMeta(sourceEmail) + `\]\(mailto:` + regexp.QuoteMeta(sourceEmail) + `\)`),
			replacement: `[` + targetEmail + `](mailto:` + targetEmail + `)`,
		},
		// mailto: link
		{
			regex:       regexp.MustCompile(`mailto:` + regexp.QuoteMeta(sourceEmail)),
			replacement: `mailto:` + targetEmail,
		},
		// Plain email address
		{
			regex:       regexp.MustCompile(`\b` + regexp.QuoteMeta(sourceEmail) + `\b`),
			replacement: targetEmail,
		},
	}

	result := content
	for _, p := range patterns {
		result = p.regex.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// replaceEmailYAML handles email replacement in YAML files
func (e *emailTransformer) replaceEmailYAML(content []byte, sourceEmail, targetEmail string) []byte {
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Quoted email: "email@example.com"
		{
			regex:       regexp.MustCompile(`"` + regexp.QuoteMeta(sourceEmail) + `"`),
			replacement: `"` + targetEmail + `"`,
		},
		// Single-quoted email: 'email@example.com'
		{
			regex:       regexp.MustCompile(`'` + regexp.QuoteMeta(sourceEmail) + `'`),
			replacement: `'` + targetEmail + `'`,
		},
		// Unquoted email after key
		{
			regex:       regexp.MustCompile(`(:\s*)` + regexp.QuoteMeta(sourceEmail) + `(\s|$)`),
			replacement: `${1}` + targetEmail + `${2}`,
		},
	}

	result := content
	for _, p := range patterns {
		result = p.regex.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// replaceEmailJSON handles email replacement in JSON files
func (e *emailTransformer) replaceEmailJSON(content []byte, sourceEmail, targetEmail string) []byte {
	// JSON always has quoted strings
	pattern := regexp.MustCompile(`"` + regexp.QuoteMeta(sourceEmail) + `"`)
	return pattern.ReplaceAll(content, []byte(`"`+targetEmail+`"`))
}

// replaceEmailHTML handles email replacement in HTML files
func (e *emailTransformer) replaceEmailHTML(content []byte, sourceEmail, targetEmail string) []byte {
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// <a href="mailto:email">...</a>
		{
			regex:       regexp.MustCompile(`<a\s+href="mailto:` + regexp.QuoteMeta(sourceEmail) + `"`),
			replacement: `<a href="mailto:` + targetEmail + `"`,
		},
		// mailto:email in href attributes
		{
			regex:       regexp.MustCompile(`mailto:` + regexp.QuoteMeta(sourceEmail)),
			replacement: `mailto:` + targetEmail,
		},
		// Plain email with word boundaries
		{
			regex:       regexp.MustCompile(`\b` + regexp.QuoteMeta(sourceEmail) + `\b`),
			replacement: targetEmail,
		},
	}

	result := content
	for _, p := range patterns {
		result = p.regex.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// replaceEmailGeneral handles email replacement for general file types
func (e *emailTransformer) replaceEmailGeneral(content []byte, sourceEmail, targetEmail string) []byte {
	// Simple replacement with word boundaries
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(sourceEmail) + `\b`)
	return pattern.ReplaceAll(content, []byte(targetEmail))
}
