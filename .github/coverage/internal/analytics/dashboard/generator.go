package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Generator handles dashboard generation
type Generator struct {
	config   *GeneratorConfig
	renderer *Renderer
}

// GeneratorConfig contains configuration for dashboard generation
type GeneratorConfig struct {
	ProjectName      string
	RepositoryOwner  string
	RepositoryName   string
	TemplateDir      string
	OutputDir        string
	AssetsDir        string
	GeneratorVersion string
}

// NewGenerator creates a new dashboard generator
func NewGenerator(config *GeneratorConfig) *Generator {
	return &Generator{
		config:   config,
		renderer: NewRenderer(config.TemplateDir),
	}
}

// Generate creates the dashboard from coverage data
func (g *Generator) Generate(ctx context.Context, data *CoverageData) error {
	// Ensure output directory exists
	if err := os.MkdirAll(g.config.OutputDir, 0o750); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Generate dashboard HTML
	dashboardHTML, err := g.generateDashboardHTML(ctx, data)
	if err != nil {
		return fmt.Errorf("generating dashboard HTML: %w", err)
	}

	// Write dashboard HTML
	dashboardPath := filepath.Join(g.config.OutputDir, "index.html")
	if err := os.WriteFile(dashboardPath, []byte(dashboardHTML), 0o600); err != nil {
		return fmt.Errorf("writing dashboard HTML: %w", err)
	}

	// Generate and write data JSON
	if err := g.generateDataJSON(ctx, data); err != nil {
		return fmt.Errorf("generating data JSON: %w", err)
	}

	// Copy assets
	if err := g.copyAssets(ctx); err != nil {
		return fmt.Errorf("copying assets: %w", err)
	}

	return nil
}

// generateDashboardHTML generates the main dashboard HTML
func (g *Generator) generateDashboardHTML(ctx context.Context, data *CoverageData) (string, error) {
	// Prepare template data
	templateData := g.prepareTemplateData(data)

	// Render dashboard
	html, err := g.renderer.RenderDashboard(ctx, templateData)
	if err != nil {
		return "", fmt.Errorf("rendering dashboard: %w", err)
	}

	return html, nil
}

// prepareTemplateData prepares data for template rendering
func (g *Generator) prepareTemplateData(data *CoverageData) map[string]interface{} {
	// Calculate additional metrics
	coveragePercent := fmt.Sprintf("%.2f", data.TotalCoverage)
	filesPercent := float64(data.CoveredFiles) / float64(data.TotalFiles) * 100

	// Calculate trends
	hasHistory := data.TrendData != nil && len(data.History) > 1
	coverageTrend := "0"
	trendDirection := "stable"
	filesTrend := "0"
	linesToCoverTrend := data.MissedLines

	if hasHistory && data.TrendData != nil {
		coverageTrend = fmt.Sprintf("%.2f", data.TrendData.ChangePercent)
		trendDirection = data.TrendData.Direction
		if data.TrendData.Direction == "up" {
			filesTrend = fmt.Sprintf("+%d", data.TrendData.ChangeLines)
		} else {
			filesTrend = fmt.Sprintf("%d", data.TrendData.ChangeLines)
		}
	}

	// Prepare branch data
	branches := g.prepareBranchData(data)

	// Build commit URL
	commitURL := ""
	if data.RepositoryURL != "" && data.CommitSHA != "" {
		commitURL = fmt.Sprintf("%s/commit/%s", strings.TrimSuffix(data.RepositoryURL, ".git"), data.CommitSHA)
	}

	return map[string]interface{}{
		"ProjectName":       g.config.ProjectName,
		"RepositoryURL":     data.RepositoryURL,
		"Branch":            data.Branch,
		"CommitSHA":         g.formatCommitSHA(data.CommitSHA),
		"CommitURL":         commitURL,
		"PRNumber":          data.PRNumber,
		"Timestamp":         data.Timestamp.Format("2006-01-02 15:04:05 UTC"),
		"TotalCoverage":     coveragePercent,
		"CoverageTrend":     coverageTrend,
		"TrendDirection":    trendDirection,
		"CoveredFiles":      data.CoveredFiles,
		"TotalFiles":        data.TotalFiles,
		"FilesPercent":      fmt.Sprintf("%.1f", filesPercent),
		"FilesTrend":        filesTrend,
		"LinesToCover":      data.MissedLines,
		"LinesToCoverTrend": linesToCoverTrend,
		"PackagesTracked":   len(data.Packages),
		"Branches":          branches,
		"Packages":          g.preparePackageData(data.Packages),
		"HasHistory":        hasHistory,
		"HasAnyData":        len(data.History) > 0,
		"HistoryJSON":       g.prepareHistoryJSON(data.History),
	}
}

// formatCommitSHA formats commit SHA for display
func (g *Generator) formatCommitSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// prepareBranchData prepares branch information for display
func (g *Generator) prepareBranchData(data *CoverageData) []map[string]interface{} {
	// For now, return current branch info
	// In future, this could load from metadata
	branches := []map[string]interface{}{
		{
			"Name":         data.Branch,
			"Coverage":     data.TotalCoverage,
			"CoveredLines": data.CoveredLines,
			"TotalLines":   data.TotalLines,
			"Protected":    data.Branch == "master" || data.Branch == "main",
		},
	}
	return branches
}

// preparePackageData prepares package data for display
func (g *Generator) preparePackageData(packages []PackageCoverage) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(packages))
	for _, pkg := range packages {
		result = append(result, map[string]interface{}{
			"Name":         pkg.Name,
			"Path":         pkg.Path,
			"Coverage":     fmt.Sprintf("%.2f", pkg.Coverage),
			"CoveredLines": pkg.CoveredLines,
			"TotalLines":   pkg.TotalLines,
			"MissedLines":  pkg.MissedLines,
		})
	}
	return result
}

// prepareHistoryJSON prepares history data as JSON string
func (g *Generator) prepareHistoryJSON(history []HistoricalPoint) string {
	if len(history) == 0 {
		return "[]"
	}

	data, err := json.Marshal(history)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// generateDataJSON generates the data JSON file
func (g *Generator) generateDataJSON(_ context.Context, data *CoverageData) error {
	// Create data directory
	dataDir := filepath.Join(g.config.OutputDir, "data")
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	// Marshal coverage data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling coverage data: %w", err)
	}

	// Write coverage data
	coveragePath := filepath.Join(dataDir, "coverage.json")
	if writeErr := os.WriteFile(coveragePath, jsonData, 0o600); writeErr != nil {
		return fmt.Errorf("writing coverage data: %w", writeErr)
	}

	// Generate and write metadata
	metadata := &Metadata{
		GeneratedAt:      time.Now(),
		GeneratorVersion: g.config.GeneratorVersion,
		DataVersion:      "1.0",
		LastUpdated:      data.Timestamp,
	}

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	metadataPath := filepath.Join(dataDir, "metadata.json")
	if err := os.WriteFile(metadataPath, metadataJSON, 0o600); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}

	return nil
}

// copyAssets copies static assets to output directory
func (g *Generator) copyAssets(_ context.Context) error {
	// For now, we'll embed assets in the HTML
	// In future, this could copy CSS, JS, and image files
	return nil
}

// Renderer handles template rendering
type Renderer struct {
	templateDir string
}

// NewRenderer creates a new template renderer
func NewRenderer(templateDir string) *Renderer {
	return &Renderer{
		templateDir: templateDir,
	}
}

// RenderDashboard renders the dashboard template
func (r *Renderer) RenderDashboard(_ context.Context, data map[string]interface{}) (string, error) {
	// For now, use embedded template
	// In future, load from file
	tmpl := template.Must(template.New("dashboard").Parse(dashboardTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
