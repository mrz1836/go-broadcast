package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/sh"
)

// Static errors for linting compliance (err113)
var (
	ErrInvalidRepoURL = errors.New("invalid repository URL")
	ErrGitHubAPI      = errors.New("GitHub API error")
)

// VersionChecker defines the interface for checking latest tool versions from GitHub.
type VersionChecker interface {
	CheckLatestVersion(ctx context.Context, repoURL string) (string, error)
}

// FileUpdater defines the interface for file operations.
type FileUpdater interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, content []byte, perm os.FileMode) error
	BackupFile(path string) error
}

// VersionLogger defines the interface for logging version check results.
type VersionLogger interface {
	Info(msg string)
	Error(msg string)
	Warn(msg string)
}

// ToolInfo represents a tool with its version configuration.
type ToolInfo struct {
	EnvVars   []string // Multiple env vars may use the same tool
	RepoURL   string   // GitHub repository URL
	RepoOwner string   // GitHub owner
	RepoName  string   // GitHub repository name
}

// CheckResult represents the result of a version check.
type CheckResult struct {
	Tool           string
	EnvVars        []string
	CurrentVersion string
	LatestVersion  string
	Status         string // "up-to-date", "update-available", "error"
	Error          error
}

// GitHubRelease represents a GitHub release response.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
}

// realVersionChecker implements VersionChecker using GitHub API.
type realVersionChecker struct {
	httpClient *http.Client
	useGHCLI   bool
}

// NewVersionChecker creates a new version checker.
func NewVersionChecker(useGHCLI bool) VersionChecker {
	return &realVersionChecker{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		useGHCLI:   useGHCLI,
	}
}

// CheckLatestVersion checks the latest version from GitHub releases.
func (r *realVersionChecker) CheckLatestVersion(ctx context.Context, repoURL string) (string, error) {
	// Try gh CLI first if available and preferred
	if r.useGHCLI {
		version, err := r.checkViaGHCLI(ctx, repoURL)
		if err == nil {
			return version, nil
		}
		// Fall through to API if gh CLI fails
	}

	// Use GitHub API
	return r.checkViaAPI(ctx, repoURL)
}

// checkViaGHCLI uses gh CLI to check latest version.
func (r *realVersionChecker) checkViaGHCLI(_ context.Context, repoURL string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(repoURL, "https://github.com/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("%w: %s", ErrInvalidRepoURL, repoURL)
	}

	repo := fmt.Sprintf("%s/%s", parts[0], parts[1])
	output, err := sh.Output("gh", "api", fmt.Sprintf("repos/%s/releases/latest", repo))
	if err != nil {
		return "", err
	}

	var release GitHubRelease
	if err := json.Unmarshal([]byte(output), &release); err != nil {
		return "", fmt.Errorf("failed to parse release JSON: %w", err)
	}

	return release.TagName, nil
}

// checkViaAPI uses direct GitHub API to check latest version.
func (r *realVersionChecker) checkViaAPI(ctx context.Context, repoURL string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(repoURL, "https://github.com/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("%w: %s", ErrInvalidRepoURL, repoURL)
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", parts[0], parts[1])

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't override the main return error
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status %d: %s", ErrGitHubAPI, resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

// realFileUpdater implements FileUpdater using os package.
type realFileUpdater struct{}

// NewFileUpdater creates a new file updater.
func NewFileUpdater() FileUpdater {
	return &realFileUpdater{}
}

// ReadFile reads a file from disk.
func (r *realFileUpdater) ReadFile(path string) ([]byte, error) {
	//nolint:gosec // Path is controlled by the application
	return os.ReadFile(path)
}

// WriteFile writes content to a file.
func (r *realFileUpdater) WriteFile(path string, content []byte, perm os.FileMode) error {
	return os.WriteFile(path, content, perm)
}

// BackupFile creates a backup of the file.
func (r *realFileUpdater) BackupFile(path string) error {
	//nolint:gosec // Path is controlled by the application
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	backupPath := path + ".backup"
	return os.WriteFile(backupPath, content, 0o600)
}

// consoleLogger implements VersionLogger using os.Stdout.
type consoleLogger struct{}

// NewConsoleLogger creates a new console logger.
func NewConsoleLogger() VersionLogger {
	return &consoleLogger{}
}

// Info logs an info message.
func (c *consoleLogger) Info(msg string) {
	_, _ = os.Stdout.WriteString(msg + "\n")
}

// Error logs an error message.
func (c *consoleLogger) Error(msg string) {
	_, _ = os.Stdout.WriteString("ERROR: " + msg + "\n")
}

// Warn logs a warning message.
func (c *consoleLogger) Warn(msg string) {
	_, _ = os.Stdout.WriteString("WARN: " + msg + "\n")
}

// VersionUpdateService orchestrates the version update process.
type VersionUpdateService struct {
	checker VersionChecker
	updater FileUpdater
	logger  VersionLogger
	dryRun  bool
	delay   time.Duration
}

// NewVersionUpdateService creates a new version update service.
func NewVersionUpdateService(checker VersionChecker, updater FileUpdater, logger VersionLogger, dryRun bool, delay time.Duration) *VersionUpdateService {
	return &VersionUpdateService{
		checker: checker,
		updater: updater,
		logger:  logger,
		dryRun:  dryRun,
		delay:   delay,
	}
}

// GetToolDefinitions returns the list of tools to check with their GitHub repos.
func GetToolDefinitions() map[string]*ToolInfo {
	tools := make(map[string]*ToolInfo)

	// Define unique tools with their GitHub repos (from .env.base comments)
	definitions := []struct {
		key       string
		envVars   []string
		repoOwner string
		repoName  string
	}{
		{"go-coverage", []string{"GO_COVERAGE_VERSION"}, "mrz1836", "go-coverage"},
		{"mage-x", []string{"MAGE_X_VERSION"}, "mrz1836", "mage-x"},
		{"gitleaks", []string{"MAGE_X_GITLEAKS_VERSION", "GITLEAKS_VERSION", "GO_PRE_COMMIT_GITLEAKS_VERSION"}, "gitleaks", "gitleaks"},
		{"gofumpt", []string{"MAGE_X_GOFUMPT_VERSION", "GO_PRE_COMMIT_FUMPT_VERSION"}, "mvdan", "gofumpt"},
		{"golangci-lint", []string{"MAGE_X_GOLANGCI_LINT_VERSION", "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION"}, "golangci", "golangci-lint"},
		{"goreleaser", []string{"MAGE_X_GORELEASER_VERSION"}, "goreleaser", "goreleaser"},
		{"govulncheck", []string{"MAGE_X_GOVULNCHECK_VERSION", "GOVULNCHECK_VERSION"}, "golang", "vuln"},
		{"mockgen", []string{"MAGE_X_MOCKGEN_VERSION"}, "uber-go", "mock"},
		{"nancy", []string{"MAGE_X_NANCY_VERSION", "NANCY_VERSION"}, "sonatype-nexus-community", "nancy"},
		{"staticcheck", []string{"MAGE_X_STATICCHECK_VERSION"}, "dominikh", "go-tools"},
		{"swag", []string{"MAGE_X_SWAG_VERSION"}, "swaggo", "swag"},
		{"yamlfmt", []string{"MAGE_X_YAMLFMT_VERSION"}, "google", "yamlfmt"},
		{"go-pre-commit", []string{"GO_PRE_COMMIT_VERSION"}, "mrz1836", "go-pre-commit"},
	}

	for _, def := range definitions {
		tools[def.key] = &ToolInfo{
			EnvVars:   def.envVars,
			RepoURL:   fmt.Sprintf("https://github.com/%s/%s", def.repoOwner, def.repoName),
			RepoOwner: def.repoOwner,
			RepoName:  def.repoName,
		}
	}

	return tools
}

// Run executes the version update process.
func (s *VersionUpdateService) Run(ctx context.Context, envFilePath string) error {
	// Log the execution mode immediately
	if s.dryRun {
		s.logger.Info("🔍 Running in DRY RUN mode - no changes will be applied")
	} else {
		s.logger.Info("✏️  Running in UPDATE mode - changes will be applied")
	}
	s.logger.Info("")

	// Read the .env.base file
	content, err := s.updater.ReadFile(envFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Get tool definitions
	tools := GetToolDefinitions()

	// Calculate estimated time
	toolCount := len(tools)
	// Estimate: (n-1) delays between n requests + avg API response time per tool (~2 seconds)
	estimatedSeconds := int((time.Duration(toolCount-1) * s.delay).Seconds()) + (toolCount * 2)

	s.logger.Info(fmt.Sprintf("🔍 Checking %d tools for updates (estimated time: ~%d seconds)...", toolCount, estimatedSeconds))
	s.logger.Info("")

	// Extract current versions from file
	currentVersions := s.extractVersions(content, tools)

	// Check latest versions
	results := s.checkVersions(ctx, tools, currentVersions)

	// Display results
	s.displayResults(results)

	// Update file if needed
	if !s.dryRun && s.hasUpdates(results) {
		return s.updateFile(envFilePath, content, results)
	}

	return nil
}

// extractVersions extracts current versions from the file content.
func (s *VersionUpdateService) extractVersions(content []byte, tools map[string]*ToolInfo) map[string]string {
	versions := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Create a map of env var to tool key
	envVarToTool := make(map[string]string)
	for toolKey, tool := range tools {
		for _, envVar := range tool.EnvVars {
			envVarToTool[envVar] = toolKey
		}
	}

	// Scan file line by line
	for scanner.Scan() {
		line := scanner.Text()
		for envVar, toolKey := range envVarToTool {
			pattern := regexp.MustCompile(fmt.Sprintf(`^%s=([^#\s]+)`, regexp.QuoteMeta(envVar)))
			if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
				// Keep the first version found for each tool to detect if any env var needs updating
				if _, exists := versions[toolKey]; !exists {
					versions[toolKey] = strings.TrimSpace(matches[1])
				}
				break
			}
		}
	}

	return versions
}

// checkVersions checks the latest versions for all tools.
func (s *VersionUpdateService) checkVersions(ctx context.Context, tools map[string]*ToolInfo, currentVersions map[string]string) []CheckResult {
	results := make([]CheckResult, 0, len(tools))

	// Sort tools by key for consistent output
	toolKeys := make([]string, 0, len(tools))
	for key := range tools {
		toolKeys = append(toolKeys, key)
	}
	sort.Strings(toolKeys)

	for i, toolKey := range toolKeys {
		tool := tools[toolKey]

		// Add delay between requests to avoid rate limiting
		if i > 0 {
			time.Sleep(s.delay)
		}

		currentVersion := currentVersions[toolKey]
		latestVersion, err := s.checker.CheckLatestVersion(ctx, tool.RepoURL)

		result := CheckResult{
			Tool:           toolKey,
			EnvVars:        tool.EnvVars,
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion,
		}

		if err != nil {
			result.Status = "error"
			result.Error = err
		} else if s.normalizeVersion(currentVersion) == s.normalizeVersion(latestVersion) {
			result.Status = "up-to-date"
		} else {
			result.Status = "update-available"
		}

		results = append(results, result)
	}

	return results
}

// normalizeVersion normalizes version strings for comparison.
func (s *VersionUpdateService) normalizeVersion(version string) string {
	// Remove 'v' prefix if present
	return strings.TrimPrefix(version, "v")
}

// displayResults displays the check results in a formatted table.
func (s *VersionUpdateService) displayResults(results []CheckResult) {
	// Print header
	header := fmt.Sprintf("%-25s %-15s %-15s %s\n", "Tool", "Current", "Latest", "Status")
	_, _ = os.Stdout.WriteString(header)
	_, _ = os.Stdout.WriteString(strings.Repeat("─", 80) + "\n")

	// Track statistics
	upToDate := 0
	updates := 0
	errors := 0

	// Print results
	for _, result := range results {
		var statusIcon string
		switch result.Status {
		case "up-to-date":
			statusIcon = "✓ Up to date"
			upToDate++
		case "update-available":
			statusIcon = "⬆ Update available"
			updates++
		case "error":
			statusIcon = fmt.Sprintf("✗ Error: %v", result.Error)
			errors++
		}

		line := fmt.Sprintf("%-25s %-15s %-15s %s\n",
			result.Tool,
			result.CurrentVersion,
			result.LatestVersion,
			statusIcon,
		)
		_, _ = os.Stdout.WriteString(line)
	}

	// Print summary
	_, _ = os.Stdout.WriteString("\n")
	_, _ = os.Stdout.WriteString("Summary:\n")
	_, _ = fmt.Fprintf(os.Stdout, "✓ %d tools up to date\n", upToDate)
	_, _ = fmt.Fprintf(os.Stdout, "⬆ %d tools with updates available\n", updates)
	_, _ = fmt.Fprintf(os.Stdout, "✗ %d tools failed to check\n", errors)
	_, _ = os.Stdout.WriteString("\n")

	if s.dryRun && updates > 0 {
		s.logger.Info("[DRY RUN] No changes made. Set UPDATE_VERSIONS=true to apply updates.")
	}
}

// hasUpdates checks if any updates are available.
func (s *VersionUpdateService) hasUpdates(results []CheckResult) bool {
	for _, result := range results {
		if result.Status == "update-available" {
			return true
		}
	}
	return false
}

// updateFile updates the .env.base file with new versions.
func (s *VersionUpdateService) updateFile(envFilePath string, content []byte, results []CheckResult) error {
	s.logger.Info("📝 Updating .env.base file...")

	// Create backup
	if err := s.updater.BackupFile(envFilePath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	s.logger.Info("✓ Backup created: " + envFilePath + ".backup")

	// Update content
	newContent := content
	for _, result := range results {
		if result.Status == "update-available" {
			for _, envVar := range result.EnvVars {
				// Match the env var and capture its current version to detect format
				pattern := regexp.MustCompile(fmt.Sprintf(`(?m)^(%s=)([^#\s]+)(\s|$)`,
					regexp.QuoteMeta(envVar)))

				newContent = pattern.ReplaceAllFunc(newContent, func(match []byte) []byte {
					submatches := pattern.FindSubmatch(match)
					if len(submatches) < 4 {
						return match
					}
					prefix := submatches[1]       // "ENV_VAR="
					currentVal := submatches[2]   // current version value
					suffix := submatches[3]       // trailing whitespace or EOL

					// Preserve the v-prefix format of the original value
					hasVPrefix := strings.HasPrefix(string(currentVal), "v")
					newVersion := result.LatestVersion

					if hasVPrefix && !strings.HasPrefix(newVersion, "v") {
						newVersion = "v" + newVersion
					} else if !hasVPrefix && strings.HasPrefix(newVersion, "v") {
						newVersion = strings.TrimPrefix(newVersion, "v")
					}

					return append(append(prefix, []byte(newVersion)...), suffix...)
				})
			}
		}
	}

	// Write updated content
	if err := s.updater.WriteFile(envFilePath, newContent, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	s.logger.Info("✓ File updated successfully")
	return nil
}

// Global singleton for dependency injection
//
//nolint:gochecknoglobals // Required for singleton pattern
var (
	versionUpdateService *VersionUpdateService
	versionServiceOnce   sync.Once
	versionServiceMutex  sync.RWMutex
)

// getVersionUpdateService returns the global version update service instance.
func getVersionUpdateService(dryRun bool) *VersionUpdateService {
	versionServiceMutex.RLock()
	if versionUpdateService != nil {
		versionServiceMutex.RUnlock()
		return versionUpdateService
	}
	versionServiceMutex.RUnlock()

	versionServiceOnce.Do(func() {
		// Check if gh CLI is available
		useGHCLI := false
		if _, err := sh.Output("gh", "version"); err == nil {
			useGHCLI = true
		}

		checker := NewVersionChecker(useGHCLI)
		updater := NewFileUpdater()
		logger := NewConsoleLogger()
		delay := 2 * time.Second

		versionServiceMutex.Lock()
		versionUpdateService = NewVersionUpdateService(checker, updater, logger, dryRun, delay)
		versionServiceMutex.Unlock()
	})

	return versionUpdateService
}

// setVersionUpdateService sets the version update service (for testing).
func setVersionUpdateService(service *VersionUpdateService) {
	versionServiceMutex.Lock()
	defer versionServiceMutex.Unlock()
	versionUpdateService = service
}

// resetVersionUpdateService resets the singleton (for testing).
func resetVersionUpdateService() {
	versionServiceMutex.Lock()
	defer versionServiceMutex.Unlock()
	versionUpdateService = nil
	versionServiceOnce = sync.Once{}
}

// RunVersionUpdate runs the version update process.
func RunVersionUpdate(dryRun bool) error {
	ctx := context.Background()

	// Get the path to .env.base
	envFilePath := filepath.Join(".github", ".env.base")

	service := getVersionUpdateService(dryRun)
	return service.Run(ctx, envFilePath)
}
