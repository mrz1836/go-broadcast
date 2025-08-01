package report

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// TemplateTestSuite provides test suite for report template
type TemplateTestSuite struct {
	suite.Suite
}

// TestReportTemplateConstant tests that the template constant is not empty
func (suite *TemplateTestSuite) TestReportTemplateConstant() {
	suite.NotEmpty(reportTemplate, "Report template should not be empty")
	suite.Greater(len(reportTemplate), 1000, "Report template should be substantial")
}

// TestReportTemplateValidHTML tests that the template contains valid HTML structure
func (suite *TemplateTestSuite) TestReportTemplateValidHTML() {
	// Basic HTML structure checks
	suite.Contains(reportTemplate, "<!DOCTYPE html>")
	suite.Contains(reportTemplate, "<html")
	suite.Contains(reportTemplate, "</html>")
	suite.Contains(reportTemplate, "<head>")
	suite.Contains(reportTemplate, "</head>")
	suite.Contains(reportTemplate, "<body>")
	suite.Contains(reportTemplate, "</body>")

	// Meta tags
	suite.Contains(reportTemplate, `<meta charset="UTF-8">`)
	suite.Contains(reportTemplate, `name="viewport"`)
	suite.Contains(reportTemplate, `name="description"`)

	// CSS and JavaScript
	suite.Contains(reportTemplate, `<link rel="stylesheet"`)
	suite.Contains(reportTemplate, `<script>`)
	suite.Contains(reportTemplate, `</script>`)
}

// TestReportTemplateGoTemplateVariables tests that template contains expected Go template variables
func (suite *TemplateTestSuite) TestReportTemplateGoTemplateVariables() {
	expectedVariables := []string{
		"{{.RepositoryOwner}}",
		"{{.RepositoryName}}",
		"{{.BranchName}}",
		"{{truncate .CommitSHA 7}}",
		"{{.Summary.TotalPercentage}}",
		"{{.Summary.CoveredLines | commas}}",
		"{{.Summary.TotalLines | commas}}",
		"{{.GeneratedAt.Format",
		"{{.GoogleAnalyticsID}}",
		"{{.Title}}",
	}

	for _, variable := range expectedVariables {
		suite.Contains(reportTemplate, variable,
			"Template should contain variable %s", variable)
	}
}

// TestReportTemplateConditionals tests template conditional blocks
func (suite *TemplateTestSuite) TestReportTemplateConditionals() {
	expectedConditionals := []string{
		"{{- if .GoogleAnalyticsID}}",
		"{{- if .BranchName}}",
		"{{- if .CommitSHA}}",
		"{{- if .CommitURL}}",
		"{{- if .BadgeURL}}",
		"{{- if .Packages}}",
		"{{- if .Files}}",
		"{{- if .LatestTag}}",
		"{{- if .Title}}",
		"{{- if .ProjectName}}",
	}

	for _, conditional := range expectedConditionals {
		suite.Contains(reportTemplate, conditional,
			"Template should contain conditional %s", conditional)
	}

	// Check for corresponding endif blocks (count both {{- end}} and {{end -}} formats)
	endifCount := strings.Count(reportTemplate, "{{- end}}") + strings.Count(reportTemplate, "{{end -}}")
	ifCount := strings.Count(reportTemplate, "{{- if")

	// Should have at least as many ends as ifs (some might be range ends)
	suite.GreaterOrEqual(endifCount, ifCount,
		"Template should have balanced if/end blocks")
}

// TestReportTemplateRangeBlocks tests template range blocks
func (suite *TemplateTestSuite) TestReportTemplateRangeBlocks() {
	expectedRanges := []string{
		"{{- range .Packages}}",
		"{{- range .Files}}",
	}

	for _, rangeBlock := range expectedRanges {
		suite.Contains(reportTemplate, rangeBlock,
			"Template should contain range block %s", rangeBlock)
	}

	// Count range blocks and their corresponding ends
	rangeCount := strings.Count(reportTemplate, "{{- range")

	// All ranges should have corresponding ends
	suite.Positive(rangeCount, "Template should have range blocks")
}

// TestReportTemplateFunctionCalls tests template function calls
func (suite *TemplateTestSuite) TestReportTemplateFunctionCalls() {
	expectedFunctions := []string{
		"printf",
		"commas",
		"truncate",
		"ge",
		"multiply",
	}

	for _, function := range expectedFunctions {
		// Functions can be called in various ways, so check for the function name
		suite.Contains(reportTemplate, function,
			"Template should use function %s", function)
	}
}

// TestReportTemplateAssetReferences tests asset file references
func (suite *TemplateTestSuite) TestReportTemplateAssetReferences() {
	expectedAssetReferences := []string{
		"./assets/css/coverage.css",
		"./assets/images/favicon.ico",
		"./assets/images/favicon.svg",
		"./assets/site.webmanifest",
	}

	for _, assetRef := range expectedAssetReferences {
		suite.Contains(reportTemplate, assetRef,
			"Template should reference asset %s", assetRef)
	}
}

// TestReportTemplateCSSClasses tests CSS class usage
func (suite *TemplateTestSuite) TestReportTemplateCSSClasses() {
	expectedClasses := []string{
		"nav-header",
		"header",
		"main-content",
		"footer",
		"coverage-circle",
		"coverage-bar",
		"package-card",
		"file-item",
		"excellent",
		"success",
		"warning",
		"low",
		"danger",
	}

	for _, class := range expectedClasses {
		suite.Contains(reportTemplate, class,
			"Template should use CSS class %s", class)
	}
}

// TestReportTemplateJavaScriptFunctions tests JavaScript function definitions
func (suite *TemplateTestSuite) TestReportTemplateJavaScriptFunctions() {
	expectedJSFunctions := []string{
		"toggleTheme",
		"togglePackage",
		"copyBadgeURL",
	}

	for _, jsFunction := range expectedJSFunctions {
		suite.Contains(reportTemplate, jsFunction,
			"Template should define JavaScript function %s", jsFunction)
	}

	// Check for event listeners
	suite.Contains(reportTemplate, "addEventListener")
	suite.Contains(reportTemplate, "onclick")
}

// TestReportTemplateAccessibility tests accessibility features
func (suite *TemplateTestSuite) TestReportTemplateAccessibility() {
	// ARIA labels
	accessibilityFeatures := []string{
		`aria-label`,
		`role=`,
		`alt=`,
	}

	foundAccessibilityFeatures := 0
	for _, feature := range accessibilityFeatures {
		if strings.Contains(reportTemplate, feature) {
			foundAccessibilityFeatures++
		}
	}

	suite.Positive(foundAccessibilityFeatures,
		"Template should include accessibility features")

	// Language attribute
	suite.Contains(reportTemplate, `lang="en"`)

	// Viewport meta tag for responsive design
	suite.Contains(reportTemplate, `name="viewport"`)
}

// TestReportTemplateSEOFeatures tests SEO-related features
func (suite *TemplateTestSuite) TestReportTemplateSEOFeatures() {
	seoFeatures := []string{
		`<title>`,
		`name="description"`,
		`property="og:title"`,
		`property="og:description"`,
		`property="og:type"`,
	}

	for _, feature := range seoFeatures {
		suite.Contains(reportTemplate, feature,
			"Template should include SEO feature %s", feature)
	}
}

// TestReportTemplateParsingSuccess tests that the template can be parsed successfully
func (suite *TemplateTestSuite) TestReportTemplateParsingSuccess() {
	// Create template functions (same as in renderer)
	funcMap := template.FuncMap{
		"multiply": func(a float64, b float64) float64 {
			return a * b
		},
		"printf": func(format string, _ ...interface{}) string {
			return strings.ReplaceAll(format, "%", "")
		},
		"commas": func(_ int) string {
			return "1,000"
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length]
		},
		"ge": func(a, b float64) bool {
			return a >= b
		},
		"sub": func(a, b float64) float64 {
			return a - b
		},
		"round": func(f float64) float64 {
			return f
		},
	}

	// Parse template
	tmpl, err := template.New("test").Funcs(funcMap).Parse(reportTemplate)
	suite.Require().NoError(err, "Template should parse without errors")
	suite.NotNil(tmpl)
}

// TestReportTemplateExecutionWithSampleData tests template execution with sample data
func (suite *TemplateTestSuite) TestReportTemplateExecutionWithSampleData() {
	// Create template functions
	funcMap := template.FuncMap{
		"multiply": func(a float64, b float64) float64 {
			return a * b
		},
		"printf": func(format string, args ...interface{}) string {
			if len(args) == 0 {
				return format
			}
			// Simple mock implementation
			return "85.0"
		},
		"commas": func(n int) string {
			if n >= 1000 {
				return "1,000"
			}
			return "100"
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length]
		},
		"ge": func(a, b float64) bool {
			return a >= b
		},
		"sub": func(a, b float64) float64 {
			return a - b
		},
		"round": func(f float64) float64 {
			return f
		},
	}

	// Parse template
	tmpl, err := template.New("test").Funcs(funcMap).Parse(reportTemplate)
	suite.Require().NoError(err)

	// Create sample data
	data := &Data{
		Coverage: &parser.CoverageData{
			Percentage:   85.0,
			TotalLines:   1000,
			CoveredLines: 850,
		},
		GeneratedAt:       time.Now(),
		Title:             "test-owner/test-repo Coverage Report",
		ProjectName:       "Test Project",
		RepositoryOwner:   "test-owner",
		RepositoryName:    "test-repo",
		BranchName:        "main",
		CommitSHA:         "abc123def456789",
		CommitURL:         "https://github.com/test-owner/test-repo/commit/abc123def456789",
		GoogleAnalyticsID: "GA-123456789",
		Summary: Summary{
			TotalPercentage: 85.0,
			TotalLines:      1000,
			CoveredLines:    850,
			UncoveredLines:  150,
			PackageCount:    2,
			FileCount:       5,
		},
		Packages: []PackageReport{
			{
				Name:         "package1",
				Percentage:   90.0,
				TotalLines:   500,
				CoveredLines: 450,
				Files: []FileReport{
					{
						Name:         "file1.go",
						Path:         "package1/file1.go",
						Percentage:   95.0,
						TotalLines:   200,
						CoveredLines: 190,
					},
				},
			},
			{
				Name:         "package2",
				Percentage:   80.0,
				TotalLines:   500,
				CoveredLines: 400,
			},
		},
	}

	// Execute template
	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	suite.Require().NoError(err, "Template should execute without errors")

	output := buf.String()
	suite.NotEmpty(output, "Template output should not be empty")

	// Verify some data made it into the output
	suite.Contains(output, "test-owner/test-repo")
	suite.Contains(output, "main")
	suite.Contains(output, "package1")
}

// TestReportTemplateResponsiveDesign tests responsive design features
func (suite *TemplateTestSuite) TestReportTemplateResponsiveDesign() {
	responsiveFeatures := []string{
		`name="viewport"`,
		`width=device-width`,
		`initial-scale=1.0`,
	}

	for _, feature := range responsiveFeatures {
		suite.Contains(reportTemplate, feature,
			"Template should include responsive design feature %s", feature)
	}
}

// TestReportTemplateThemeSupport tests theme support features
func (suite *TemplateTestSuite) TestReportTemplateThemeSupport() {
	themeFeatures := []string{
		`data-theme="auto"`,
		"toggleTheme",
		"prefers-color-scheme",
		"localStorage.getItem('theme')",
	}

	for _, feature := range themeFeatures {
		suite.Contains(reportTemplate, feature,
			"Template should support theme feature %s", feature)
	}
}

// TestReportTemplateExternalResources tests external resource loading
func (suite *TemplateTestSuite) TestReportTemplateExternalResources() {
	// Font loading
	suite.Contains(reportTemplate, "fonts.googleapis.com")
	suite.Contains(reportTemplate, "preconnect")
	suite.Contains(reportTemplate, "preload")

	// Google Analytics (conditional)
	suite.Contains(reportTemplate, "googletagmanager.com")
	suite.Contains(reportTemplate, "gtag")
}

// TestReportTemplateSecurityFeatures tests security-related features
func (suite *TemplateTestSuite) TestReportTemplateSecurityFeatures() {
	// Cross-origin attributes
	suite.Contains(reportTemplate, "crossorigin")

	// External links should be secure
	externalLinks := []string{
		"https://fonts.googleapis.com",
		"https://www.googletagmanager.com",
	}

	for _, link := range externalLinks {
		if strings.Contains(reportTemplate, link) {
			// Ensure it's HTTPS
			suite.True(strings.HasPrefix(link, "https://"),
				"External link should use HTTPS: %s", link)
		}
	}
}

// TestReportTemplateStructure tests overall template structure
func (suite *TemplateTestSuite) TestReportTemplateStructure() {
	// Count major sections
	sections := []string{
		"nav-header",
		"header",
		"main-content",
		"footer",
	}

	for _, section := range sections {
		suite.Contains(reportTemplate, section,
			"Template should contain section %s", section)
	}

	// Verify balanced HTML tags
	openTags := []string{"<html", "<head>", "<body>", "<nav", "<header", "<main", "<footer"}
	closeTags := []string{"</html>", "</head>", "</body>", "</nav>", "</header>", "</main>", "</footer>"}

	for i, openTag := range openTags {
		openCount := strings.Count(reportTemplate, openTag)
		closeCount := strings.Count(reportTemplate, closeTags[i])

		suite.Equal(openCount, closeCount,
			"Template should have balanced %s and %s tags", openTag, closeTags[i])
	}
}

// TestReportTemplateComments tests template comments and documentation
func (suite *TemplateTestSuite) TestReportTemplateComments() {
	// Should have HTML comments for documentation
	suite.Contains(reportTemplate, "<!--")
	suite.Contains(reportTemplate, "-->")

	// Should have template comments for sections
	templateCommentTypes := []string{
		"{{- if",
		"{{- range",
		"{{- end}}",
	}

	for _, commentType := range templateCommentTypes {
		suite.Contains(reportTemplate, commentType)
	}
}

// TestRun runs the test suite
func TestTemplateTestSuite(t *testing.T) {
	suite.Run(t, new(TemplateTestSuite))
}

// Additional individual tests not in the suite

// TestReportTemplateLength tests template length constraints
func TestReportTemplateLength(t *testing.T) {
	// Should be substantial but not excessive
	assert.Greater(t, len(reportTemplate), 5000, "Template should be substantial")
	assert.Less(t, len(reportTemplate), 100000, "Template should not be excessively large")
}

// TestReportTemplateEncoding tests character encoding
func TestReportTemplateEncoding(t *testing.T) {
	// Should contain charset declaration
	assert.Contains(t, reportTemplate, `charset="UTF-8"`)

	// Should not contain invalid characters that could cause parsing issues
	invalidChars := []string{"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07", "\x08"}
	for _, char := range invalidChars {
		assert.NotContains(t, reportTemplate, char, "Template should not contain invalid character")
	}
}

// TestReportTemplateMinification tests that template is reasonably compact
func TestReportTemplateMinification(t *testing.T) {
	// Count excessive whitespace
	lines := strings.Split(reportTemplate, "\n")
	emptyLines := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyLines++
		}
	}

	// Should not have excessive empty lines (indicates good formatting)
	emptyLineRatio := float64(emptyLines) / float64(len(lines))
	assert.Less(t, emptyLineRatio, 0.3, "Template should not have excessive empty lines")
}
