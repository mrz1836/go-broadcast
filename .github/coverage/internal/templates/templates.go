package templates

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"time"
)

//go:embed *.html
var embeddedFiles embed.FS

// TemplateManager handles template loading and rendering
type TemplateManager struct {
	templates *template.Template
	funcs     template.FuncMap
}

// DashboardData contains data for dashboard template rendering
type DashboardData struct {
	// Project information
	ProjectName string `json:"project_name"`

	// Overall metrics
	TotalCoverage     float64 `json:"total_coverage"`
	CoverageTrend     float64 `json:"coverage_trend"`
	CoveredFiles      int     `json:"covered_files"`
	TotalFiles        int     `json:"total_files"`
	FilesTrend        int     `json:"files_trend"`
	LinesToCover      int     `json:"lines_to_cover"`
	LinesToCoverTrend int     `json:"lines_to_cover_trend"`
	PackagesTracked   int     `json:"packages_tracked"`

	// Branch information
	Branches []BranchData `json:"branches"`

	// Metadata
	LastUpdated time.Time `json:"last_updated"`
	CommitSha   string    `json:"commit_sha"`

	// UI settings
	Theme       string `json:"theme"`
	ShowTrends  bool   `json:"show_trends"`
	ShowDetails bool   `json:"show_details"`
}

// BranchData contains branch-specific coverage information
type BranchData struct {
	Name         string    `json:"name"`
	Coverage     float64   `json:"coverage"`
	CoveredLines int       `json:"covered_lines"`
	TotalLines   int       `json:"total_lines"`
	Protected    bool      `json:"protected"`
	LastCommit   time.Time `json:"last_commit"`
	Trend        float64   `json:"trend"`
}

// ReportData contains data for coverage report template rendering
type ReportData struct {
	// Report metadata
	Title       string    `json:"title"`
	ProjectName string    `json:"project_name"`
	Generated   time.Time `json:"generated"`
	Branch      string    `json:"branch"`
	CommitSha   string    `json:"commit_sha"`

	// Coverage summary
	OverallCoverage float64        `json:"overall_coverage"`
	PackageStats    []PackageStats `json:"package_stats"`
	FileStats       []FileStats    `json:"file_stats"`

	// Configuration
	Theme       string `json:"theme"`
	ShowDetails bool   `json:"show_details"`
}

// PackageStats contains package-level coverage statistics
type PackageStats struct {
	Name         string  `json:"name"`
	Coverage     float64 `json:"coverage"`
	Files        int     `json:"files"`
	Lines        int     `json:"lines"`
	CoveredLines int     `json:"covered_lines"`
}

// FileStats contains file-level coverage statistics
type FileStats struct {
	Name         string  `json:"name"`
	Package      string  `json:"package"`
	Coverage     float64 `json:"coverage"`
	Lines        int     `json:"lines"`
	CoveredLines int     `json:"covered_lines"`
	Functions    int     `json:"functions"`
	CoveredFuncs int     `json:"covered_funcs"`
}

// NewTemplateManager creates a new template manager with embedded templates
func NewTemplateManager() (*TemplateManager, error) {
	tm := &TemplateManager{
		funcs: template.FuncMap{
			"formatFloat":      formatFloat,
			"formatPercentage": formatPercentage,
			"formatTime":       formatTime,
			"colorForCoverage": colorForCoverage,
			"badgeColor":       badgeColor,
			"add":              add,
			"sub":              sub,
			"mul":              mul,
			"div":              div,
		},
	}

	// Parse embedded templates
	tmpl, err := template.New("").Funcs(tm.funcs).ParseFS(embeddedFiles, "*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded templates: %w", err)
	}

	tm.templates = tmpl
	return tm, nil
}

// RenderDashboard renders the main dashboard HTML
func (tm *TemplateManager) RenderDashboard(_ context.Context, data DashboardData) (string, error) {
	var buf bytes.Buffer

	// Set default values if not provided
	if data.ProjectName == "" {
		data.ProjectName = "GoFortress Project"
	}
	if data.Theme == "" {
		data.Theme = "auto"
	}
	if data.LastUpdated.IsZero() {
		data.LastUpdated = time.Now()
	}

	// Render the dashboard template
	if err := tm.templates.ExecuteTemplate(&buf, "dashboard.html", data); err != nil {
		return "", fmt.Errorf("failed to render dashboard template: %w", err)
	}

	return buf.String(), nil
}

// RenderReport renders a coverage report HTML
func (tm *TemplateManager) RenderReport(_ context.Context, data ReportData) (string, error) {
	var buf bytes.Buffer

	// Set default values if not provided
	if data.Title == "" {
		data.Title = "Coverage Report"
	}
	if data.ProjectName == "" {
		data.ProjectName = "Go Project"
	}
	if data.Theme == "" {
		data.Theme = "auto"
	}
	if data.Generated.IsZero() {
		data.Generated = time.Now()
	}

	// Render the coverage report template
	if err := tm.templates.ExecuteTemplate(&buf, "coverage-report.html", data); err != nil {
		return "", fmt.Errorf("failed to render coverage report template: %w", err)
	}

	return buf.String(), nil
}

// WriteDashboard writes the dashboard HTML to a writer
func (tm *TemplateManager) WriteDashboard(ctx context.Context, w io.Writer, data DashboardData) error {
	content, err := tm.RenderDashboard(ctx, data)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(content))
	return err
}

// WriteReport writes a coverage report HTML to a writer
func (tm *TemplateManager) WriteReport(ctx context.Context, w io.Writer, data ReportData) error {
	content, err := tm.RenderReport(ctx, data)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(content))
	return err
}

// GetEmbeddedFile returns the content of an embedded file
func (tm *TemplateManager) GetEmbeddedFile(filename string) ([]byte, error) {
	return embeddedFiles.ReadFile(filename)
}

// ListEmbeddedFiles returns a list of embedded file names
func (tm *TemplateManager) ListEmbeddedFiles() ([]string, error) {
	entries, err := embeddedFiles.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// Template helper functions

func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}

func formatPercentage(f float64) string {
	return fmt.Sprintf("%.1f%%", f)
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05 UTC")
}

func colorForCoverage(coverage float64) string {
	switch {
	case coverage >= 90:
		return "#3fb950" // Excellent - bright green
	case coverage >= 80:
		return "#90c978" // Good - green
	case coverage >= 70:
		return "#d29922" // Acceptable - yellow
	case coverage >= 60:
		return "#f85149" // Low - orange
	default:
		return "#da3633" // Poor - red
	}
}

func badgeColor(coverage float64) string {
	return colorForCoverage(coverage)
}

// Math helper functions for templates
func add(a, b int) int { return a + b }

func sub(a, b int) int { return a - b }

func mul(a, b float64) float64 { return a * b }

func div(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}
