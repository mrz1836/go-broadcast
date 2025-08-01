// Package report provides comprehensive coverage report generation with analytics and templating capabilities.
package report

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/analytics/assets"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// Generator creates HTML coverage reports
type Generator struct {
	config   *Config
	renderer *Renderer
}

// Config holds report generation configuration
type Config struct {
	OutputDir         string
	RepositoryOwner   string
	RepositoryName    string
	BranchName        string
	CommitSHA         string
	GoogleAnalyticsID string
}

// Data represents the complete data needed for report generation
type Data struct {
	Coverage          *parser.CoverageData
	GeneratedAt       time.Time
	Title             string
	ProjectName       string
	RepositoryOwner   string
	RepositoryName    string
	BranchName        string
	CommitSHA         string
	CommitURL         string
	BadgeURL          string
	Summary           Summary
	Packages          []PackageReport
	LatestTag         string
	GoogleAnalyticsID string
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

// PackageReport represents coverage data for a package
type PackageReport struct {
	Name         string
	Percentage   float64
	TotalLines   int
	CoveredLines int
	Files        []FileReport
}

// FileReport represents coverage data for a file
type FileReport struct {
	Name         string
	Path         string
	Percentage   float64
	TotalLines   int
	CoveredLines int
}

// NewGenerator creates a new report generator
func NewGenerator(config *Config) *Generator {
	return &Generator{
		config:   config,
		renderer: NewRenderer(),
	}
}

// Generate creates an HTML coverage report
func (g *Generator) Generate(ctx context.Context, coverage *parser.CoverageData) error {
	// Ensure output directory exists
	if err := os.MkdirAll(g.config.OutputDir, 0o750); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Build report data
	data := g.buildReportData(ctx, coverage)

	// Render report
	html, err := g.renderer.RenderReport(ctx, data)
	if err != nil {
		return fmt.Errorf("rendering report: %w", err)
	}

	// Write report HTML
	reportPath := filepath.Join(g.config.OutputDir, "coverage.html")
	if err := os.WriteFile(reportPath, html, 0o600); err != nil {
		return fmt.Errorf("writing report HTML: %w", err)
	}

	// Copy assets
	if err := assets.CopyAssetsTo(g.config.OutputDir); err != nil {
		return fmt.Errorf("copying assets: %w", err)
	}

	return nil
}

// buildReportData constructs the report data structure
func (g *Generator) buildReportData(_ context.Context, coverage *parser.CoverageData) *Data {
	var packages []PackageReport
	totalFiles := 0

	// Handle nil coverage
	if coverage != nil {
		packages = make([]PackageReport, 0, len(coverage.Packages))

		// Sort packages by name
		packageNames := make([]string, 0, len(coverage.Packages))
		for name := range coverage.Packages {
			packageNames = append(packageNames, name)
		}
		sort.Strings(packageNames)

		// Build package reports
		for _, name := range packageNames {
			pkg := coverage.Packages[name]
			files := make([]FileReport, 0, len(pkg.Files))

			// Sort files by name
			fileNames := make([]string, 0, len(pkg.Files))
			for fileName := range pkg.Files {
				fileNames = append(fileNames, fileName)
			}
			sort.Strings(fileNames)

			// Build file reports
			for _, fileName := range fileNames {
				file := pkg.Files[fileName]
				totalLines := 0
				coveredLines := 0

				for _, stmt := range file.Statements {
					lines := stmt.EndLine - stmt.StartLine + 1
					totalLines += lines
					if stmt.Count > 0 {
						coveredLines += lines
					}
				}

				var percentage float64
				if totalLines > 0 {
					percentage = float64(coveredLines) / float64(totalLines) * 100
				}

				files = append(files, FileReport{
					Name:         fileName,
					Path:         filepath.Join(name, fileName),
					Percentage:   percentage,
					TotalLines:   totalLines,
					CoveredLines: coveredLines,
				})
				totalFiles++
			}

			packages = append(packages, PackageReport{
				Name:         name,
				Percentage:   pkg.Percentage,
				TotalLines:   pkg.TotalLines,
				CoveredLines: pkg.CoveredLines,
				Files:        files,
			})
		}
	}

	// Calculate summary
	var summary Summary
	if coverage != nil {
		summary = Summary{
			TotalPercentage: coverage.Percentage,
			TotalLines:      coverage.TotalLines,
			CoveredLines:    coverage.CoveredLines,
			UncoveredLines:  coverage.TotalLines - coverage.CoveredLines,
			PackageCount:    len(packages),
			FileCount:       totalFiles,
		}
	} else {
		summary = Summary{
			TotalPercentage: 0.0,
			TotalLines:      0,
			CoveredLines:    0,
			UncoveredLines:  0,
			PackageCount:    0,
			FileCount:       0,
		}
	}

	// Get analytics config
	var googleAnalyticsID string
	if g.config != nil {
		googleAnalyticsID = g.config.GoogleAnalyticsID
	}

	// Build commit URL if we have GitHub info
	commitURL := ""
	if g.config != nil && g.config.RepositoryOwner != "" && g.config.RepositoryName != "" && g.config.CommitSHA != "" {
		commitURL = fmt.Sprintf("https://github.com/%s/%s/commit/%s",
			g.config.RepositoryOwner, g.config.RepositoryName, g.config.CommitSHA)
	}

	// Build title
	title := "Coverage Report"
	if g.config != nil {
		if g.config.RepositoryOwner != "" && g.config.RepositoryName != "" {
			title = fmt.Sprintf("%s/%s Coverage Report", g.config.RepositoryOwner, g.config.RepositoryName)
		} else if g.config.RepositoryName != "" {
			title = fmt.Sprintf("%s Coverage Report", g.config.RepositoryName)
		}
	}

	// Safely extract config values
	var repositoryOwner, repositoryName, branchName, commitSHA string
	if g.config != nil {
		repositoryOwner = g.config.RepositoryOwner
		repositoryName = g.config.RepositoryName
		branchName = g.config.BranchName
		commitSHA = g.config.CommitSHA
	}

	return &Data{
		Coverage:          coverage,
		GeneratedAt:       time.Now(),
		Title:             title,
		ProjectName:       repositoryName,
		RepositoryOwner:   repositoryOwner,
		RepositoryName:    repositoryName,
		BranchName:        branchName,
		CommitSHA:         commitSHA,
		CommitURL:         commitURL,
		BadgeURL:          "", // Badge is generated separately
		Summary:           summary,
		Packages:          packages,
		LatestTag:         "", // Could be fetched from git
		GoogleAnalyticsID: googleAnalyticsID,
	}
}
