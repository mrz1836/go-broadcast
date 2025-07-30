// Package report generates HTML coverage reports
package report

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	globalconfig "github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

const coverageReport = "Coverage Report"

// RepositoryInfo contains information extracted from a Git repository
type RepositoryInfo struct {
	Name     string // Repository name (e.g., "go-broadcast")
	Owner    string // Repository owner (e.g., "mrz1836")
	FullName string // Full repository name (e.g., "mrz1836/go-broadcast")
	URL      string // Repository URL
	IsGitHub bool   // Whether this is a GitHub repository
}

// CoverageRecord represents a single coverage measurement
type CoverageRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	CommitSHA    string    `json:"commit_sha"`
	Branch       string    `json:"branch"`
	Percentage   float64   `json:"percentage"`
	TotalLines   int       `json:"total_lines"`
	CoveredLines int       `json:"covered_lines"`
}

// Generator creates beautiful, interactive HTML coverage reports with cutting-edge UX
type Generator struct {
	config   *Config
	renderer *Renderer
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
	// GitHub integration for source links
	GitHubOwner      string
	GitHubRepository string
	GitHubBranch     string
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
	LatestTag   string
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
			Title:            coverageReport,
			ShowPackages:     true,
			ShowFiles:        true,
			ShowMissing:      true,
			DarkMode:         true,
			Responsive:       true,
			InteractiveTrees: true,
		},
		renderer: NewRenderer(),
	}
}

// NewWithConfig creates a new report generator with custom configuration
func NewWithConfig(config *Config) *Generator {
	return &Generator{
		config:   config,
		renderer: NewRenderer(),
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
	reportData := g.buildReportData(ctx, coverage, &config)

	// Generate HTML
	return g.renderHTML(ctx, reportData)
}

// getGitCommitSHA returns the current Git commit SHA
func getGitCommitSHA(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getLatestGitTag gets the latest git tag in the repository
func getLatestGitTag(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
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

// buildCoverageBadgeURL builds a coverage badge URL using shields.io
func buildCoverageBadgeURL(percentage float64) string {
	// Determine color based on percentage
	var color string
	switch {
	case percentage >= 90:
		color = "brightgreen"
	case percentage >= 80:
		color = "green"
	case percentage >= 70:
		color = "yellowgreen"
	case percentage >= 60:
		color = "yellow"
	case percentage >= 50:
		color = "orange"
	default:
		color = "red"
	}

	return fmt.Sprintf("https://img.shields.io/badge/coverage-%.1f%%25-%s", percentage, color)
}

// buildGitHubCommitURL builds a GitHub commit URL from repository info and commit SHA
func buildGitHubCommitURL(owner, repo, commitSHA string) string {
	if owner == "" || repo == "" || commitSHA == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s/commit/%s", owner, repo, commitSHA)
}

// buildReportData constructs the report data structure
func (g *Generator) buildReportData(ctx context.Context, coverage *parser.CoverageData, config *Config) *Data {
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

	// For now, use simple defaults for history (TODO: implement file-based history)
	changeStatus := "stable"
	previousCoverage := 0.0

	summary := Summary{
		TotalPercentage:  coverage.Percentage,
		TotalLines:       coverage.TotalLines,
		CoveredLines:     coverage.CoveredLines,
		UncoveredLines:   coverage.TotalLines - coverage.CoveredLines,
		PackageCount:     len(packages),
		FileCount:        totalFiles,
		ChangeStatus:     changeStatus,
		PreviousCoverage: previousCoverage,
	}

	// Load global config to get current branch
	globalConfig := globalconfig.Load()
	branchName := globalConfig.GetCurrentBranch() //nolint:contextcheck // Global config interface doesn't support context

	// Get Git information
	var projectName, commitSHA, commitURL, badgeURL string

	// Get repository info
	if repoInfo := getGitRepositoryInfo(ctx); repoInfo != nil {
		projectName = repoInfo.Name

		// Get current commit SHA
		if sha := getGitCommitSHA(ctx); sha != "" {
			commitSHA = sha

			// Build commit URL if it's a GitHub repository
			if repoInfo.IsGitHub {
				commitURL = buildGitHubCommitURL(repoInfo.Owner, repoInfo.Name, commitSHA)
			}
		}
	}

	// If we couldn't get the project name from Git, use a default
	if projectName == "" {
		projectName = "Go Project"
	}

	// Generate badge URL
	badgeURL = buildCoverageBadgeURL(coverage.Percentage)

	// Get latest tag
	latestTag := getLatestGitTag(ctx)

	return &Data{
		Coverage:    coverage,
		Config:      config,
		GeneratedAt: time.Now(),
		Version:     "1.0.0",
		ProjectName: projectName,
		BranchName:  branchName,
		CommitSHA:   commitSHA,
		CommitURL:   commitURL,
		BadgeURL:    badgeURL,
		Summary:     summary,
		Packages:    packages,
		LatestTag:   latestTag,
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

	// Convert internal Data to ReportData format
	reportData := g.convertToTemplateData(ctx, data)

	// Use the renderer to render the modern coverage report
	htmlContent, err := g.renderer.RenderReport(ctx, reportData)
	if err != nil {
		return nil, fmt.Errorf("failed to render modern coverage report: %w", err)
	}

	return []byte(htmlContent), nil
}

// convertToTemplateData converts internal Data structure to ReportData
func (g *Generator) convertToTemplateData(ctx context.Context, data *Data) ReportData {
	// Get dynamic repository information
	repoInfo := getGitRepositoryInfo(ctx)

	// Handle nil data
	if data == nil {
		// Try to get project name from Git, fallback to generic name
		projectName := coverageReport
		gitHubOwner := ""
		gitHubRepo := ""
		title := coverageReport

		if repoInfo != nil {
			projectName = repoInfo.Name
			gitHubOwner = repoInfo.Owner
			gitHubRepo = repoInfo.Name
			title = fmt.Sprintf("%s/%s Coverage Report", repoInfo.Owner, repoInfo.Name)
		}

		return ReportData{
			Title:            title,
			ProjectName:      projectName,
			Generated:        time.Now(),
			Branch:           "main",
			CommitSha:        "",
			OverallCoverage:  0.0,
			PackageStats:     []PackageStats{},
			FileStats:        []FileStats{},
			Theme:            "auto",
			ShowDetails:      false,
			GitHubOwner:      gitHubOwner,
			GitHubRepository: gitHubRepo,
			GitHubBranch:     "main",
			RepositoryOwner:  gitHubOwner,
			RepositoryName:   gitHubRepo,
		}
	}

	// Convert package reports to template format
	packageStats := make([]PackageStats, 0, len(data.Packages))
	fileStats := make([]FileStats, 0)

	// Add nil checks for data fields
	if data.Config == nil {
		data.Config = &Config{
			Title:     coverageReport,
			Theme:     "auto",
			ShowFiles: true,
		}
	}

	for _, pkg := range data.Packages {
		if pkg.Files == nil {
			pkg.Files = []FileReport{}
		}

		packageStats = append(packageStats, PackageStats{
			Name:         pkg.Name,
			Coverage:     pkg.Percentage,
			Files:        len(pkg.Files),
			Lines:        pkg.TotalLines,
			CoveredLines: pkg.CoveredLines,
		})

		// Add file stats for this package
		for _, file := range pkg.Files {
			fileStats = append(fileStats, FileStats{
				Name:         file.Name,
				Path:         file.Path,
				Package:      pkg.Name,
				Coverage:     file.Percentage,
				Lines:        file.TotalLines,
				CoveredLines: file.CoveredLines,
				Functions:    0, // Not available in current data structure
				CoveredFuncs: 0, // Not available in current data structure
			})
		}
	}

	// Use dynamic repository information, with config fallbacks
	gitHubOwner := data.Config.GitHubOwner
	gitHubRepo := data.Config.GitHubRepository
	gitHubBranch := data.Config.GitHubBranch
	title := data.Config.Title

	// Override with dynamic Git info if available
	if repoInfo != nil {
		gitHubOwner = repoInfo.Owner
		gitHubRepo = repoInfo.Name
		// Use repository-focused title only if no custom title was set (empty or using default constant)
		if title == "" || title == coverageReport {
			title = fmt.Sprintf("%s/%s Coverage Report", repoInfo.Owner, repoInfo.Name)
		}
	}

	// Use current branch if not specified in config
	if gitHubBranch == "" {
		gitHubBranch = data.BranchName
	}

	return ReportData{
		Title:            title,
		ProjectName:      data.ProjectName, // Use the dynamically determined project name
		Generated:        data.GeneratedAt,
		Branch:           data.BranchName,
		CommitSha:        data.CommitSHA,
		OverallCoverage:  data.Summary.TotalPercentage,
		PackageStats:     packageStats,
		FileStats:        fileStats,
		Theme:            data.Config.Theme,
		ShowDetails:      data.Config.ShowFiles,
		GitHubOwner:      gitHubOwner,
		GitHubRepository: gitHubRepo,
		GitHubBranch:     gitHubBranch,
		RepositoryOwner:  gitHubOwner, // Alias for template compatibility
		RepositoryName:   gitHubRepo,  // Alias for template compatibility
		LatestTag:        data.LatestTag,
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
