package transform

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrInvalidRepoFormat is returned when a repository format is invalid
var ErrInvalidRepoFormat = errors.New("invalid repository format")

// repoTransformer replaces repository names in specific contexts
type repoTransformer struct {
	cache *RegexCache
}

// NewRepoTransformer creates a new repository name transformer
func NewRepoTransformer() Transformer {
	return &repoTransformer{
		cache: getDefaultCache(),
	}
}

// Name returns the name of this transformer
func (r *repoTransformer) Name() string {
	return "repository-name-replacer"
}

// Transform applies repository name replacement to the content
func (r *repoTransformer) Transform(content []byte, ctx Context) ([]byte, error) {
	// Skip if source and target repos are the same
	if ctx.SourceRepo == ctx.TargetRepo {
		return content, nil
	}

	// Extract repository names
	sourceParts := strings.Split(ctx.SourceRepo, "/")

	targetParts := strings.Split(ctx.TargetRepo, "/")
	if len(sourceParts) != 2 || len(targetParts) != 2 {
		return content, fmt.Errorf("%w: source=%s, target=%s", ErrInvalidRepoFormat, ctx.SourceRepo, ctx.TargetRepo)
	}

	sourceOrg := sourceParts[0]
	sourceRepoName := sourceParts[1]
	targetOrg := targetParts[0]
	targetRepoName := targetParts[1]

	// Apply transformations based on file type
	result := content
	fileExt := strings.ToLower(filepath.Ext(ctx.FilePath))

	switch fileExt {
	case ".go", ".mod":
		result = r.transformGoFile(result, sourceOrg, sourceRepoName, targetOrg, targetRepoName)
	case ".md", ".txt", ".rst":
		result = r.transformDocumentation(result, sourceOrg, sourceRepoName, targetOrg, targetRepoName)
	case ".yaml", ".yml", ".json":
		result = r.transformConfig(result, sourceOrg, sourceRepoName, targetOrg, targetRepoName)
	default:
		// For other files, apply general transformations
		result = r.transformGeneral(result, sourceOrg, sourceRepoName, targetOrg, targetRepoName)
	}

	return result, nil
}

// transformGoFile handles Go-specific transformations
func (r *repoTransformer) transformGoFile(content []byte, sourceOrg, sourceRepo, targetOrg, targetRepo string) []byte {
	// Escape target values for safe use in replacement strings
	safeTargetOrg := escapeReplacement(targetOrg)
	safeTargetRepo := escapeReplacement(targetRepo)

	patterns := []struct {
		pattern     string
		replacement string
	}{
		// Module declaration in go.mod
		{
			pattern:     `(?m)^module\s+github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo),
			replacement: fmt.Sprintf("module github.com/%s/%s", safeTargetOrg, safeTargetRepo),
		},
		// Import statements - match exact repo boundary
		{
			pattern:     `"github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo) + `("|/[^"]*")`,
			replacement: fmt.Sprintf(`"github.com/%s/%s$1`, safeTargetOrg, safeTargetRepo),
		},
		// Import blocks - match when followed by slash, quote, or end
		{
			pattern:     `github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo) + `(/|"|$)`,
			replacement: fmt.Sprintf(`github.com/%s/%s$1`, safeTargetOrg, safeTargetRepo),
		},
	}

	result := content
	for _, p := range patterns {
		re, err := r.cache.CompileRegex(p.pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		result = re.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// transformDocumentation handles documentation transformations
func (r *repoTransformer) transformDocumentation(content []byte, sourceOrg, sourceRepo, targetOrg, targetRepo string) []byte {
	// Escape target values for safe use in replacement strings
	safeTargetOrg := escapeReplacement(targetOrg)
	safeTargetRepo := escapeReplacement(targetRepo)

	patterns := []struct {
		pattern     string
		replacement string
	}{
		// GitHub URLs
		{
			pattern:     `https://github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo),
			replacement: fmt.Sprintf("https://github.com/%s/%s", safeTargetOrg, safeTargetRepo),
		},
		// Go package references
		{
			pattern:     `github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo),
			replacement: fmt.Sprintf("github.com/%s/%s", safeTargetOrg, safeTargetRepo),
		},
		// Plain org/repo references
		{
			pattern:     `\b` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo) + `\b`,
			replacement: fmt.Sprintf("%s/%s", safeTargetOrg, safeTargetRepo),
		},
		// Repository name in titles or badges
		{
			pattern:     `\b` + regexp.QuoteMeta(sourceRepo) + `\b`,
			replacement: safeTargetRepo,
		},
	}

	result := content
	for _, p := range patterns {
		re, err := r.cache.CompileRegex(p.pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		result = re.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// transformConfig handles configuration file transformations
func (r *repoTransformer) transformConfig(content []byte, sourceOrg, sourceRepo, targetOrg, targetRepo string) []byte {
	// Escape target values for safe use in replacement strings
	safeTargetOrg := escapeReplacement(targetOrg)
	safeTargetRepo := escapeReplacement(targetRepo)

	patterns := []struct {
		pattern     string
		replacement string
	}{
		// Repository references
		{
			pattern:     regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo),
			replacement: fmt.Sprintf("%s/%s", safeTargetOrg, safeTargetRepo),
		},
		// Just the repository name when it appears alone
		{
			pattern:     `"` + regexp.QuoteMeta(sourceRepo) + `"`,
			replacement: fmt.Sprintf(`"%s"`, safeTargetRepo),
		},
		// Standalone repository name wherever it appears (with word boundaries)
		{
			pattern:     `\b` + regexp.QuoteMeta(sourceRepo) + `\b`,
			replacement: safeTargetRepo,
		},
	}

	result := content
	for _, p := range patterns {
		re, err := r.cache.CompileRegex(p.pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		result = re.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// transformGeneral applies general transformations for other file types
func (r *repoTransformer) transformGeneral(content []byte, sourceOrg, sourceRepo, targetOrg, targetRepo string) []byte {
	// Escape target values for safe use in replacement strings
	safeTargetOrg := escapeReplacement(targetOrg)
	safeTargetRepo := escapeReplacement(targetRepo)

	patterns := []struct {
		pattern     string
		replacement string
	}{
		// Repository references (org/repo format)
		{
			pattern:     regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo),
			replacement: fmt.Sprintf("%s/%s", safeTargetOrg, safeTargetRepo),
		},
		// Standalone repository name wherever it appears (with word boundaries)
		{
			pattern:     `\b` + regexp.QuoteMeta(sourceRepo) + `\b`,
			replacement: safeTargetRepo,
		},
	}

	result := content
	for _, p := range patterns {
		re, err := r.cache.CompileRegex(p.pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		result = re.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}
