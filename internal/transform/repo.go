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
type repoTransformer struct{}

// NewRepoTransformer creates a new repository name transformer
func NewRepoTransformer() Transformer {
	return &repoTransformer{}
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
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Module declaration in go.mod
		{
			regex:       regexp.MustCompile(`(?m)^module\s+github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo)),
			replacement: fmt.Sprintf("module github.com/%s/%s", targetOrg, targetRepo),
		},
		// Import statements - match exact repo boundary
		{
			regex:       regexp.MustCompile(`"github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo) + `("|/[^"]*")`),
			replacement: fmt.Sprintf(`"github.com/%s/%s$1`, targetOrg, targetRepo),
		},
		// Import blocks - match when followed by slash, quote, or end
		{
			regex:       regexp.MustCompile(`github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo) + `(/|"|$)`),
			replacement: fmt.Sprintf(`github.com/%s/%s$1`, targetOrg, targetRepo),
		},
	}

	result := content
	for _, p := range patterns {
		result = p.regex.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// transformDocumentation handles documentation transformations
func (r *repoTransformer) transformDocumentation(content []byte, sourceOrg, sourceRepo, targetOrg, targetRepo string) []byte {
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// GitHub URLs
		{
			regex:       regexp.MustCompile(`https://github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo)),
			replacement: fmt.Sprintf("https://github.com/%s/%s", targetOrg, targetRepo),
		},
		// Go package references
		{
			regex:       regexp.MustCompile(`github\.com/` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo)),
			replacement: fmt.Sprintf("github.com/%s/%s", targetOrg, targetRepo),
		},
		// Plain org/repo references
		{
			regex:       regexp.MustCompile(`\b` + regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo) + `\b`),
			replacement: fmt.Sprintf("%s/%s", targetOrg, targetRepo),
		},
		// Repository name in titles or badges
		{
			regex:       regexp.MustCompile(`\b` + regexp.QuoteMeta(sourceRepo) + `\b`),
			replacement: targetRepo,
		},
	}

	result := content
	for _, p := range patterns {
		result = p.regex.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// transformConfig handles configuration file transformations
func (r *repoTransformer) transformConfig(content []byte, sourceOrg, sourceRepo, targetOrg, targetRepo string) []byte {
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Repository references
		{
			regex:       regexp.MustCompile(regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo)),
			replacement: fmt.Sprintf("%s/%s", targetOrg, targetRepo),
		},
		// Just the repository name when it appears alone
		{
			regex:       regexp.MustCompile(`"` + regexp.QuoteMeta(sourceRepo) + `"`),
			replacement: fmt.Sprintf(`"%s"`, targetRepo),
		},
		// Standalone repository name wherever it appears (with word boundaries)
		{
			regex:       regexp.MustCompile(`\b` + regexp.QuoteMeta(sourceRepo) + `\b`),
			replacement: targetRepo,
		},
	}

	result := content
	for _, p := range patterns {
		result = p.regex.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}

// transformGeneral applies general transformations for other file types
func (r *repoTransformer) transformGeneral(content []byte, sourceOrg, sourceRepo, targetOrg, targetRepo string) []byte {
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		// Repository references (org/repo format)
		{
			regex:       regexp.MustCompile(regexp.QuoteMeta(sourceOrg) + `/` + regexp.QuoteMeta(sourceRepo)),
			replacement: fmt.Sprintf("%s/%s", targetOrg, targetRepo),
		},
		// Standalone repository name wherever it appears (with word boundaries)
		{
			regex:       regexp.MustCompile(`\b` + regexp.QuoteMeta(sourceRepo) + `\b`),
			replacement: targetRepo,
		},
	}

	result := content
	for _, p := range patterns {
		result = p.regex.ReplaceAll(result, []byte(p.replacement))
	}

	return result
}
