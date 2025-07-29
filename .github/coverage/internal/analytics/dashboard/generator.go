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
)

// Generator handles dashboard generation
type Generator struct {
	config       *GeneratorConfig
	renderer     *Renderer
	githubClient *github.Client
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

	// Build commit URL
	commitURL := ""
	if data.RepositoryURL != "" && data.CommitSHA != "" {
		commitURL = fmt.Sprintf("%s/commit/%s", strings.TrimSuffix(data.RepositoryURL, ".git"), data.CommitSHA)
	}

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

	// Get build status if available
	buildStatus := g.getBuildStatus(ctx, repoInfo)

	return map[string]interface{}{
		"ProjectName":       projectName,
		"RepositoryOwner":   repositoryOwner,
		"RepositoryName":    repositoryName,
		"RepositoryURL":     repositoryURL,
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

// getBuildStatus fetches the current build status from GitHub Actions
func (g *Generator) getBuildStatus(ctx context.Context, repoInfo *RepositoryInfo) *BuildStatus {
	// If GitHub client is not available or repository info is missing, return unavailable status
	if g.githubClient == nil || repoInfo == nil || !repoInfo.IsGitHub {
		return &BuildStatus{
			Available: false,
			Error:     "GitHub API access not configured or not a GitHub repository",
		}
	}

	// Get the latest workflow runs for the repository
	workflowRuns, err := g.githubClient.GetWorkflowRuns(ctx, repoInfo.Owner, repoInfo.Name, 5)
	if err != nil {
		return &BuildStatus{
			Available: false,
			Error:     fmt.Sprintf("Failed to fetch workflow runs: %v", err),
		}
	}

	// Find the most recent relevant workflow run
	var latestRun *github.WorkflowRun
	for _, run := range workflowRuns.WorkflowRuns {
		// Prioritize coverage-related workflows
		if strings.Contains(strings.ToLower(run.Name), "coverage") ||
			strings.Contains(strings.ToLower(run.Name), "fortress") ||
			strings.Contains(strings.ToLower(run.Path), "coverage") {
			latestRun = &run
			break
		}
		// Fallback to any recent workflow if no coverage workflow found
		if latestRun == nil {
			latestRun = &run
		}
	}

	// If no workflow runs found, return unavailable status
	if latestRun == nil {
		return &BuildStatus{
			Available: false,
			Error:     "No workflow runs found",
		}
	}

	// Calculate duration
	duration := g.formatDuration(latestRun.RunStartedAt, latestRun.UpdatedAt, latestRun.Status)

	return &BuildStatus{
		State:        latestRun.Status,
		Conclusion:   latestRun.Conclusion,
		WorkflowName: latestRun.Name,
		RunID:        latestRun.ID,
		RunNumber:    latestRun.RunNumber,
		RunURL:       latestRun.HTMLURL,
		StartedAt:    latestRun.RunStartedAt,
		UpdatedAt:    latestRun.UpdatedAt,
		Duration:     duration,
		HeadSHA:      latestRun.HeadSHA,
		HeadBranch:   latestRun.HeadBranch,
		Event:        latestRun.Event,
		DisplayTitle: latestRun.DisplayTitle,
		Available:    true,
	}
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
