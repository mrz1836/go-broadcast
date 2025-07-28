// Package report generates HTML coverage reports
package report

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// Generator creates beautiful, interactive HTML coverage reports with cutting-edge UX
type Generator struct {
	config *Config
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
	}
}

// NewWithConfig creates a new report generator with custom configuration
func NewWithConfig(config *Config) *Generator {
	return &Generator{config: config}
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

// renderHTML generates the final HTML report
func (g *Generator) renderHTML(ctx context.Context, data *Data) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	tmpl := template.New("report").Funcs(template.FuncMap{
		"formatPercentage": func(p float64) string {
			return fmt.Sprintf("%.1f%%", p)
		},
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05 MST")
		},
		"add": func(a, b int) int {
			return a + b
		},
		"statusColor": func(status string) string {
			switch status {
			case "excellent":
				return "#3fb950"
			case "good":
				return "#7c3aed"
			case "acceptable":
				return "#d29922"
			case "low":
				return "#fb8500"
			default:
				return "#f85149"
			}
		},
		"progressWidth": func(covered, total int) float64 {
			if total == 0 {
				return 0
			}
			return float64(covered) / float64(total) * 100
		},
	})

	// Parse the HTML template
	tmpl, err := tmpl.Parse(g.getHTMLTemplate())
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

// getHTMLTemplate returns the HTML template for the report
func (g *Generator) getHTMLTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Config.Title}}</title>
    <style>
        :root {
            --bg-primary: #0d1117;
            --bg-secondary: #161b22;
            --bg-tertiary: #21262d;
            --border-primary: #30363d;
            --text-primary: #f0f6fc;
            --text-secondary: #8b949e;
            --text-muted: #656d76;
            --accent-emphasis: #1f6feb;
            --success-emphasis: #3fb950;
            --attention-emphasis: #d29922;
            --danger-emphasis: #f85149;
            --severe-emphasis: #fb8500;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif;
            background-color: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem;
        }

        .header {
            text-align: center;
            margin-bottom: 3rem;
            padding: 2rem;
            background: var(--bg-secondary);
            border-radius: 12px;
            border: 1px solid var(--border-primary);
        }

        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
            background: linear-gradient(135deg, var(--accent-emphasis), var(--success-emphasis));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .header .subtitle {
            color: var(--text-secondary);
            font-size: 1.1rem;
        }

        .summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1.5rem;
            margin-bottom: 3rem;
        }

        .summary-card {
            background: var(--bg-secondary);
            padding: 1.5rem;
            border-radius: 8px;
            border: 1px solid var(--border-primary);
            text-align: center;
        }

        .summary-card .value {
            font-size: 2rem;
            font-weight: bold;
            margin-bottom: 0.5rem;
        }

        .summary-card .label {
            color: var(--text-secondary);
            font-size: 0.9rem;
        }

        .coverage-bar {
            width: 100%;
            height: 8px;
            background: var(--bg-tertiary);
            border-radius: 4px;
            overflow: hidden;
            margin: 1rem 0;
        }

        .coverage-fill {
            height: 100%;
            background: linear-gradient(90deg, var(--success-emphasis), var(--accent-emphasis));
            transition: width 0.3s ease;
        }

        .packages {
            margin-bottom: 3rem;
        }

        .section-title {
            font-size: 1.5rem;
            margin-bottom: 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .package-item, .file-item {
            background: var(--bg-secondary);
            border: 1px solid var(--border-primary);
            border-radius: 8px;
            margin-bottom: 1rem;
            overflow: hidden;
        }

        .package-header, .file-header {
            padding: 1rem;
            display: flex;
            justify-content: between;
            align-items: center;
            cursor: pointer;
            transition: background-color 0.2s ease;
        }

        .package-header:hover, .file-header:hover {
            background: var(--bg-tertiary);
        }

        .package-name, .file-name {
            font-weight: 600;
            flex-grow: 1;
        }

        .coverage-badge {
            padding: 0.25rem 0.75rem;
            border-radius: 4px;
            font-size: 0.85rem;
            font-weight: 600;
            margin-left: 1rem;
        }

        .coverage-badge.excellent {
            background: var(--success-emphasis);
            color: white;
        }

        .coverage-badge.good {
            background: var(--accent-emphasis);
            color: white;
        }

        .coverage-badge.acceptable {
            background: var(--attention-emphasis);
            color: white;
        }

        .coverage-badge.low, .coverage-badge.poor {
            background: var(--danger-emphasis);
            color: white;
        }

        .stats {
            display: flex;
            gap: 1rem;
            margin-left: 1rem;
            color: var(--text-secondary);
            font-size: 0.9rem;
        }

        .files {
            background: var(--bg-tertiary);
            border-top: 1px solid var(--border-primary);
        }

        .footer {
            text-align: center;
            padding: 2rem;
            color: var(--text-muted);
            border-top: 1px solid var(--border-primary);
            margin-top: 3rem;
        }

        .footer a {
            color: var(--accent-emphasis);
            text-decoration: none;
        }

        .footer a:hover {
            text-decoration: underline;
        }

        @media (max-width: 768px) {
            .container {
                padding: 1rem;
            }
            
            .header h1 {
                font-size: 2rem;
            }
            
            .summary {
                grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
                gap: 1rem;
            }
            
            .stats {
                flex-direction: column;
                gap: 0.25rem;
            }
        }

        .expandable {
            display: none;
        }

        .expandable.expanded {
            display: block;
        }

        .expand-icon {
            transition: transform 0.2s ease;
        }

        .expand-icon.rotated {
            transform: rotate(90deg);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Config.Title}}</h1>
            <div class="subtitle">
                Generated on {{formatTime .GeneratedAt}} | 
                Branch: {{.BranchName}} | 
                {{if .CommitSHA}}Commit: {{.CommitSHA}}{{end}}
            </div>
        </div>

        <div class="summary">
            <div class="summary-card">
                <div class="value">
                    {{formatPercentage .Summary.TotalPercentage}}
                </div>
                <div class="label">Coverage</div>
                <div class="coverage-bar">
                    <div class="coverage-fill" style="width: {{.Summary.TotalPercentage}}%"></div>
                </div>
            </div>
            <div class="summary-card">
                <div class="value">{{.Summary.CoveredLines}}</div>
                <div class="label">Covered Lines</div>
            </div>
            <div class="summary-card">
                <div class="value">{{.Summary.UncoveredLines}}</div>
                <div class="label">Uncovered Lines</div>
            </div>
            <div class="summary-card">
                <div class="value">{{.Summary.PackageCount}}</div>
                <div class="label">Packages</div>
            </div>
            <div class="summary-card">
                <div class="value">{{.Summary.FileCount}}</div>
                <div class="label">Files</div>
            </div>
        </div>

        {{if .Config.ShowPackages}}
        <div class="packages">
            <h2 class="section-title">📦 Packages</h2>
            {{range .Packages}}
            <div class="package-item">
                <div class="package-header" onclick="toggleExpand('pkg-{{.Name}}')">
                    <span class="expand-icon" id="icon-pkg-{{.Name}}">▶</span>
                    <span class="package-name">{{.Name}}</span>
                    <div class="stats">
                        <span>{{.CoveredLines}}/{{.TotalLines}} lines</span>
                        <span>{{len .Files}} files</span>
                    </div>
                    <span class="coverage-badge {{.Status}}">{{formatPercentage .Percentage}}</span>
                </div>
                {{if $.Config.ShowFiles}}
                <div class="files expandable" id="pkg-{{.Name}}">
                    {{range .Files}}
                    <div class="file-item">
                        <div class="file-header">
                            <span class="file-name">{{.Name}}</span>
                            <div class="stats">
                                <span>{{.CoveredLines}}/{{.TotalLines}} lines</span>
                            </div>
                            <span class="coverage-badge {{.Status}}">{{formatPercentage .Percentage}}</span>
                        </div>
                    </div>
                    {{end}}
                </div>
                {{end}}
            </div>
            {{end}}
        </div>
        {{end}}

        <div class="footer">
            Generated by <a href="https://github.com/mrz1836/go-broadcast" target="_blank">GoFortress Coverage</a> v{{.Version}}
        </div>
    </div>

    <script>
        function toggleExpand(id) {
            const element = document.getElementById(id);
            const icon = document.getElementById('icon-' + id);
            
            if (element.classList.contains('expanded')) {
                element.classList.remove('expanded');
                icon.classList.remove('rotated');
            } else {
                element.classList.add('expanded');
                icon.classList.add('rotated');
            }
        }

        // Auto-expand packages with low coverage
        document.addEventListener('DOMContentLoaded', function() {
            const packages = document.querySelectorAll('.package-item');
            packages.forEach(pkg => {
                const badge = pkg.querySelector('.coverage-badge');
                if (badge && (badge.classList.contains('poor') || badge.classList.contains('low'))) {
                    const header = pkg.querySelector('.package-header');
                    if (header) {
                        header.click();
                    }
                }
            });
        });
    </script>
</body>
</html>`
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
