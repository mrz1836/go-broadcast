// Package report generates HTML coverage reports
package report

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
	"github.com/mrz1836/go-broadcast/coverage/internal/templates"
)

// Generator creates beautiful, interactive HTML coverage reports with cutting-edge UX
type Generator struct {
	config          *Config
	templateManager *templates.TemplateManager
}

// Config holds report generation configuration
type Config struct {
	Theme            string
	Title            string
	ShowPackages     bool
	ShowFiles        bool
	ShowMissing      bool
	DarkMode         bool
	Responsive       bool
	InteractiveTrees bool
}

// Data represents the complete data needed for report generation
type Data struct {
	Coverage    *parser.CoverageData
	Config      *Config
	GeneratedAt time.Time
	Version     string
	ProjectName string
	BranchName  string
	CommitSHA   string
	CommitURL   string
	BadgeURL    string
	Summary     Summary
	Packages    []PackageReport
}

// Summary provides high-level coverage statistics
type Summary struct {
	TotalPercentage  float64
	TotalLines       int
	CoveredLines     int
	UncoveredLines   int
	PackageCount     int
	FileCount        int
	ChangeStatus     string // "improved", "declined", "stable"
	PreviousCoverage float64
}

// PackageReport represents coverage data for a package in the report
type PackageReport struct {
	Name         string
	Percentage   float64
	TotalLines   int
	CoveredLines int
	Files        []FileReport
	Status       string // coverage status indicator
}

// FileReport represents coverage data for a file in the report
type FileReport struct {
	Name         string
	Path         string
	Percentage   float64
	TotalLines   int
	CoveredLines int
	Status       string
	Lines        []LineReport
}

// LineReport represents coverage data for a single line
type LineReport struct {
	Number  int
	Content string
	Covered bool
	Count   int
	Class   string // CSS class for styling
}

// New creates a new report generator with default configuration
func New() *Generator {
	tm, err := templates.NewTemplateManager()
	if err != nil {
		// For now, panic - in production we'd handle this better
		panic(fmt.Sprintf("failed to create template manager: %v", err))
	}

	return &Generator{
		config: &Config{
			Theme:            "github-dark",
			Title:            "Coverage Report",
			ShowPackages:     true,
			ShowFiles:        true,
			ShowMissing:      true,
			DarkMode:         true,
			Responsive:       true,
			InteractiveTrees: true,
		},
		templateManager: tm,
	}
}

// NewWithConfig creates a new report generator with custom configuration
func NewWithConfig(config *Config) *Generator {
	tm, err := templates.NewTemplateManager()
	if err != nil {
		// For now, panic - in production we'd handle this better
		panic(fmt.Sprintf("failed to create template manager: %v", err))
	}

	return &Generator{
		config:          config,
		templateManager: tm,
	}
}

// Generate creates an interactive HTML coverage report
func (g *Generator) Generate(ctx context.Context, coverage *parser.CoverageData, options ...Option) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Apply options
	config := *g.config // copy
	for _, opt := range options {
		opt(&config)
	}

	// Build report data
	reportData := g.buildReportData(coverage, &config)

	// Generate HTML
	return g.renderHTML(ctx, reportData)
}

// buildReportData constructs the report data structure
func (g *Generator) buildReportData(coverage *parser.CoverageData, config *Config) *Data {
	packages := make([]PackageReport, 0, len(coverage.Packages))
	totalFiles := 0

	// Sort packages by name for consistent ordering
	packageNames := make([]string, 0, len(coverage.Packages))
	for name := range coverage.Packages {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)

	for _, name := range packageNames {
		pkg := coverage.Packages[name]
		files := make([]FileReport, 0, len(pkg.Files))

		// Sort files by name
		fileNames := make([]string, 0, len(pkg.Files))
		for fileName := range pkg.Files {
			fileNames = append(fileNames, fileName)
		}
		sort.Strings(fileNames)

		for _, fileName := range fileNames {
			file := pkg.Files[fileName]
			files = append(files, FileReport{
				Name:         g.extractFileName(fileName),
				Path:         fileName,
				Percentage:   file.Percentage,
				TotalLines:   file.TotalLines,
				CoveredLines: file.CoveredLines,
				Status:       g.getStatusClass(file.Percentage),
				Lines:        g.buildLineReports(file),
			})
			totalFiles++
		}

		packages = append(packages, PackageReport{
			Name:         name,
			Percentage:   pkg.Percentage,
			TotalLines:   pkg.TotalLines,
			CoveredLines: pkg.CoveredLines,
			Files:        files,
			Status:       g.getStatusClass(pkg.Percentage),
		})
	}

	summary := Summary{
		TotalPercentage:  coverage.Percentage,
		TotalLines:       coverage.TotalLines,
		CoveredLines:     coverage.CoveredLines,
		UncoveredLines:   coverage.TotalLines - coverage.CoveredLines,
		PackageCount:     len(packages),
		FileCount:        totalFiles,
		ChangeStatus:     "stable", // TODO: calculate from history
		PreviousCoverage: 0.0,      // TODO: get from history
	}

	return &Data{
		Coverage:    coverage,
		Config:      config,
		GeneratedAt: time.Now(),
		Version:     "1.0.0",
		ProjectName: "Go Project", // TODO: extract from git
		BranchName:  "main",       // TODO: extract from git
		CommitSHA:   "",           // TODO: extract from git
		CommitURL:   "",           // TODO: build from git info
		BadgeURL:    "",           // TODO: build from config
		Summary:     summary,
		Packages:    packages,
	}
}

// buildLineReports creates line-by-line coverage reports (simplified for now)
func (g *Generator) buildLineReports(file *parser.FileCoverage) []LineReport {
	// For now, create basic line reports from statements
	// In a full implementation, we'd read the actual source file
	lines := make([]LineReport, 0)

	for _, stmt := range file.Statements {
		for line := stmt.StartLine; line <= stmt.EndLine; line++ {
			lines = append(lines, LineReport{
				Number:  line,
				Content: fmt.Sprintf("// Line %d content", line),
				Covered: stmt.Count > 0,
				Count:   stmt.Count,
				Class:   g.getLineClass(stmt.Count > 0),
			})
		}
	}

	// Sort by line number
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Number < lines[j].Number
	})

	return lines
}

// getStatusClass returns CSS class based on coverage percentage
func (g *Generator) getStatusClass(percentage float64) string {
	switch {
	case percentage >= 90:
		return "excellent"
	case percentage >= 80:
		return "good"
	case percentage >= 70:
		return "acceptable"
	case percentage >= 60:
		return "low"
	default:
		return "poor"
	}
}

// getLineClass returns CSS class for line coverage
func (g *Generator) getLineClass(covered bool) string {
	if covered {
		return "covered"
	}
	return "uncovered"
}

// extractFileName extracts the file name from a full path
func (g *Generator) extractFileName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// renderHTML generates the final HTML report using the modern template system
func (g *Generator) renderHTML(ctx context.Context, data *Data) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Convert internal Data to templates.ReportData format
	reportData := g.convertToTemplateData(data)

	// Use the template manager to render the modern coverage report
	htmlContent, err := g.templateManager.RenderReport(ctx, reportData)
	if err != nil {
		return nil, fmt.Errorf("failed to render modern coverage report: %w", err)
	}

	return []byte(htmlContent), nil
}

// convertToTemplateData converts internal Data structure to templates.ReportData
func (g *Generator) convertToTemplateData(data *Data) templates.ReportData {
	// Handle nil data
	if data == nil {
		return templates.ReportData{
			Title:           "Coverage Report",
			ProjectName:     "go-broadcast",
			Generated:       time.Now(),
			Branch:          "main",
			CommitSha:       "",
			OverallCoverage: 0.0,
			PackageStats:    []templates.PackageStats{},
			FileStats:       []templates.FileStats{},
			Theme:           "auto",
			ShowDetails:     false,
		}
	}

	// Convert package reports to template format
	packageStats := make([]templates.PackageStats, 0, len(data.Packages))
	fileStats := make([]templates.FileStats, 0)

	// Add nil checks for data fields
	if data.Config == nil {
		data.Config = &Config{
			Title:     "Coverage Report",
			Theme:     "auto",
			ShowFiles: true,
		}
	}

	for _, pkg := range data.Packages {
		if pkg.Files == nil {
			pkg.Files = []FileReport{}
		}

		packageStats = append(packageStats, templates.PackageStats{
			Name:         pkg.Name,
			Coverage:     pkg.Percentage,
			Files:        len(pkg.Files),
			Lines:        pkg.TotalLines,
			CoveredLines: pkg.CoveredLines,
		})

		// Add file stats for this package
		for _, file := range pkg.Files {
			fileStats = append(fileStats, templates.FileStats{
				Name:         file.Name,
				Package:      pkg.Name,
				Coverage:     file.Percentage,
				Lines:        file.TotalLines,
				CoveredLines: file.CoveredLines,
				Functions:    0, // Not available in current data structure
				CoveredFuncs: 0, // Not available in current data structure
			})
		}
	}

	return templates.ReportData{
		Title:           data.Config.Title,
		ProjectName:     "go-broadcast", // Set the correct project name
		Generated:       data.GeneratedAt,
		Branch:          data.BranchName,
		CommitSha:       data.CommitSHA,
		OverallCoverage: data.Summary.TotalPercentage,
		PackageStats:    packageStats,
		FileStats:       fileStats,
		Theme:           data.Config.Theme,
		ShowDetails:     data.Config.ShowFiles,
	}
}

// Option represents a configuration option for report generation
type Option func(*Config)

// WithTheme sets the report theme
func WithTheme(theme string) Option {
	return func(config *Config) {
		config.Theme = theme
	}
}

// WithTitle sets the report title
func WithTitle(title string) Option {
	return func(config *Config) {
		config.Title = title
	}
}

// WithPackages enables/disables package display
func WithPackages(show bool) Option {
	return func(config *Config) {
		config.ShowPackages = show
	}
}

// WithFiles enables/disables file display
func WithFiles(show bool) Option {
	return func(config *Config) {
		config.ShowFiles = show
	}
}

// WithMissing enables/disables missing line display
func WithMissing(show bool) Option {
	return func(config *Config) {
		config.ShowMissing = show
	}
}
