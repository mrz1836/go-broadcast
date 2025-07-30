package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/github"
	"github.com/mrz1836/go-broadcast/coverage/internal/templates"
)

// Generator handles dashboard generation
type Generator struct {
	config           *GeneratorConfig
	renderer         *Renderer
	githubClient     *github.Client
	lastTemplateData map[string]interface{} // Store template data for build status generation
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
	GitHubToken      string // GitHub token for API access (optional)
}

// RepositoryInfo contains information extracted from a Git repository
type RepositoryInfo struct {
	Name     string // Repository name (e.g., "go-broadcast")
	Owner    string // Repository owner (e.g., "mrz1836")
	FullName string // Full repository name (e.g., "mrz1836/go-broadcast")
	URL      string // Repository URL
	IsGitHub bool   // Whether this is a GitHub repository
}

// NewGenerator creates a new dashboard generator
func NewGenerator(config *GeneratorConfig) *Generator {
	var githubClient *github.Client
	if config.GitHubToken != "" {
		githubClient = github.New(config.GitHubToken)
	}

	return &Generator{
		config:       config,
		renderer:     NewRenderer(config.TemplateDir),
		githubClient: githubClient,
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
	templateData := g.prepareTemplateData(ctx, data)

	// Store template data for later use (e.g., build status generation)
	g.lastTemplateData = templateData

	// Render dashboard
	html, err := g.renderer.RenderDashboard(ctx, templateData)
	if err != nil {
		return "", fmt.Errorf("rendering dashboard: %w", err)
	}

	return html, nil
}

// getGitRepositoryInfo extracts repository information from Git remote
func getGitRepositoryInfo(ctx context.Context) *RepositoryInfo {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	remoteURL := strings.TrimSpace(string(output))
	return parseRepositoryURL(remoteURL)
}

// parseRepositoryURL parses a Git remote URL and extracts repository information
func parseRepositoryURL(remoteURL string) *RepositoryInfo {
	// Handle SSH URLs (git@github.com:owner/repo.git)
	sshPattern := regexp.MustCompile(`^git@([^:]+):([^/]+)/(.+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(remoteURL); len(matches) == 4 {
		host := matches[1]
		owner := matches[2]
		repo := matches[3]

		return &RepositoryInfo{
			Name:     repo,
			Owner:    owner,
			FullName: fmt.Sprintf("%s/%s", owner, repo),
			URL:      remoteURL,
			IsGitHub: host == "github.com",
		}
	}

	// Handle HTTPS URLs
	parsedURL, err := url.Parse(remoteURL)
	if err != nil {
		return nil
	}

	// Extract owner and repository from path
	path := strings.Trim(parsedURL.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return nil
	}

	owner := parts[0]
	repo := parts[1]

	return &RepositoryInfo{
		Name:     repo,
		Owner:    owner,
		FullName: fmt.Sprintf("%s/%s", owner, repo),
		URL:      remoteURL,
		IsGitHub: parsedURL.Host == "github.com",
	}
}

// getLatestGitTag gets the latest git tag in the repository
func getLatestGitTag(ctx context.Context) string {
	// Try to get the latest tag
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		// If no tags exist, return empty string
		return ""
	}
	return strings.TrimSpace(string(output))
}

// prepareTemplateData prepares data for template rendering
func (g *Generator) prepareTemplateData(ctx context.Context, data *CoverageData) map[string]interface{} {
	// Get dynamic repository information
	repoInfo := getGitRepositoryInfo(ctx)

	// Calculate additional metrics
	filesPercent := float64(data.CoveredFiles) / float64(data.TotalFiles) * 100

	// Calculate trends
	hasHistory := data.TrendData != nil && len(data.History) > 0
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
	branches := g.prepareBranchData(ctx, data)

	// Use dynamic repository information, with config fallbacks
	repositoryOwner := g.config.RepositoryOwner
	repositoryName := g.config.RepositoryName
	projectName := g.config.ProjectName

	// Override with dynamic Git info if available
	if repoInfo != nil {
		repositoryOwner = repoInfo.Owner
		repositoryName = repoInfo.Name
		if projectName == "" {
			projectName = repoInfo.Name
		}
	}

	// Build repository URL if not provided
	repositoryURL := data.RepositoryURL
	if repositoryURL == "" && repositoryOwner != "" && repositoryName != "" {
		repositoryURL = fmt.Sprintf("https://github.com/%s/%s", repositoryOwner, repositoryName)
	}

	// Build commit URL
	commitURL := ""
	if repositoryURL != "" && data.CommitSHA != "" {
		commitURL = fmt.Sprintf("%s/commit/%s", strings.TrimSuffix(repositoryURL, ".git"), data.CommitSHA)
	} else if repositoryOwner != "" && repositoryName != "" && data.CommitSHA != "" {
		// Fallback: build from owner/name if repositoryURL is empty
		commitURL = fmt.Sprintf("https://github.com/%s/%s/commit/%s", repositoryOwner, repositoryName, data.CommitSHA)
	}

	// Build owner and branch URLs
	ownerURL := ""
	branchURL := ""
	if repositoryOwner != "" {
		ownerURL = fmt.Sprintf("https://github.com/%s", repositoryOwner)
	}
	if repositoryURL != "" && data.Branch != "" {
		branchURL = fmt.Sprintf("%s/tree/%s", strings.TrimSuffix(repositoryURL, ".git"), data.Branch)
	} else if repositoryOwner != "" && repositoryName != "" && data.Branch != "" {
		// Fallback: build from owner/name if repositoryURL is empty
		branchURL = fmt.Sprintf("https://github.com/%s/%s/tree/%s", repositoryOwner, repositoryName, data.Branch)
	}

	// Build status is not generated for static deployments
	// to avoid showing stale information on GitHub Pages
	var buildStatus *BuildStatus

	// Get the latest git tag
	latestTag := getLatestGitTag(ctx)

	return map[string]interface{}{
		"ProjectName":       projectName,
		"RepositoryOwner":   repositoryOwner,
		"RepositoryName":    repositoryName,
		"RepositoryURL":     repositoryURL,
		"OwnerURL":          ownerURL,
		"BranchURL":         branchURL,
		"DefaultBranch":     data.Branch,
		"Branch":            data.Branch,
		"CommitSHA":         g.formatCommitSHA(data.CommitSHA),
		"CommitURL":         commitURL,
		"PRNumber":          data.PRNumber,
		"Timestamp":         data.Timestamp.Format("2006-01-02 15:04:05 UTC"),
		"TotalCoverage":     roundToDecimals(data.TotalCoverage, 2),
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
		"BuildStatus":       buildStatus,
		"LatestTag":         latestTag,
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
func (g *Generator) prepareBranchData(ctx context.Context, data *CoverageData) []map[string]interface{} {
	// Get dynamic repository information
	repoInfo := getGitRepositoryInfo(ctx)

	// Use dynamic repository info with config fallbacks
	repositoryOwner := g.config.RepositoryOwner
	repositoryName := g.config.RepositoryName

	if repoInfo != nil {
		repositoryOwner = repoInfo.Owner
		repositoryName = repoInfo.Name
	}

	// Generate GitHub URL for branch if repository info is available
	var githubURL string
	if repositoryOwner != "" && repositoryName != "" && data.Branch != "" {
		githubURL = fmt.Sprintf("https://github.com/%s/%s/tree/%s",
			repositoryOwner, repositoryName, data.Branch)
	}

	// For now, return current branch info
	// In future, this could load from metadata
	branches := []map[string]interface{}{
		{
			"Name":         data.Branch,
			"Coverage":     data.TotalCoverage,
			"CoveredLines": data.CoveredLines,
			"TotalLines":   data.TotalLines,
			"Protected":    data.Branch == "master" || data.Branch == "main",
			"GitHubURL":    githubURL,
		},
	}
	return branches
}

// roundToDecimals rounds a float64 to the specified number of decimal places
func roundToDecimals(value float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(value*multiplier) / multiplier
}

// preparePackageData prepares package data for display
func (g *Generator) preparePackageData(packages []PackageCoverage) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(packages))
	for _, pkg := range packages {
		// Prepare files data
		files := make([]map[string]interface{}, 0, len(pkg.Files))
		for _, file := range pkg.Files {
			files = append(files, map[string]interface{}{
				"Name":      file.Name,
				"Coverage":  roundToDecimals(file.Coverage, 2),
				"GitHubURL": file.GitHubURL,
			})
		}

		result = append(result, map[string]interface{}{
			"Name":         pkg.Name,
			"Path":         pkg.Path,
			"Coverage":     roundToDecimals(pkg.Coverage, 2),
			"CoveredLines": pkg.CoveredLines,
			"TotalLines":   pkg.TotalLines,
			"MissedLines":  pkg.MissedLines,
			"GitHubURL":    pkg.GitHubURL,
			"Files":        files,
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

// formatDuration formats the duration of a workflow run
func (g *Generator) formatDuration(startedAt, updatedAt time.Time, status string) string {
	if startedAt.IsZero() {
		return "Unknown"
	}

	var endTime time.Time
	if status == "completed" && !updatedAt.IsZero() {
		endTime = updatedAt
	} else {
		endTime = time.Now()
	}

	duration := endTime.Sub(startedAt)

	// Format duration in human-readable format
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm %ds", int(duration.Minutes()), int(duration.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(duration.Hours()), int(duration.Minutes())%60)
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
	if writeErr := os.WriteFile(metadataPath, metadataJSON, 0o600); writeErr != nil {
		return fmt.Errorf("writing metadata: %w", writeErr)
	}

	// Build status JSON is not generated for static deployments.
	// The build status shown in the dashboard is embedded at generation time
	// to avoid showing stale "in_progress" status on GitHub Pages.

	return nil
}

// copyAssets copies static assets to output directory
func (g *Generator) copyAssets(_ context.Context) error {
	tm, err := templates.NewTemplateManager()
	if err != nil {
		return fmt.Errorf("creating template manager: %w", err)
	}

	// Copy favicon.ico from embedded templates
	faviconData, err := tm.GetEmbeddedFile("favicon.ico")
	if err != nil {
		// Favicon is optional, so we don't fail if it's missing
		return nil //nolint:nilerr // Favicon is optional, we ignore the error intentionally
	}
	faviconPath := filepath.Join(g.config.OutputDir, "favicon.ico")
	if writeErr := os.WriteFile(faviconPath, faviconData, 0o600); writeErr != nil {
		return fmt.Errorf("writing favicon.ico: %w", writeErr)
	}

	// Copy favicon.svg from embedded templates
	faviconSVGData, err := tm.GetEmbeddedFile("favicon.svg")
	if err != nil {
		// Favicon SVG is optional, continue if missing
		return nil //nolint:nilerr // Favicon SVG is optional, we ignore the error intentionally
	}
	faviconSVGPath := filepath.Join(g.config.OutputDir, "favicon.svg")
	if writeErr := os.WriteFile(faviconSVGPath, faviconSVGData, 0o600); writeErr != nil {
		return fmt.Errorf("writing favicon.svg: %w", writeErr)
	}

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
