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
	ErrNoEnvFiles     = errors.New("no env files found")
)

// VersionChecker defines the interface for checking latest tool versions from GitHub or Go proxy.
type VersionChecker interface {
	CheckLatestVersion(ctx context.Context, repoURL, goModulePath string) (string, error)
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
	EnvVars      []string // Multiple env vars may use the same tool
	RepoURL      string   // GitHub repository URL
	RepoOwner    string   // GitHub owner
	RepoName     string   // GitHub repository name
	GoModulePath string   // Go module path for proxy.golang.org lookup (optional, takes precedence over GitHub)
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

// GoRelease represents a Go release from go.dev/dl API.
type GoRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// GoProxyInfo represents a Go proxy API response for module version lookup.
type GoProxyInfo struct {
	Version string `json:"Version"`
	Time    string `json:"Time"`
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

// GoDevAPIURL is the URL for the official Go download API.
const GoDevAPIURL = "https://go.dev/dl/?mode=json"

// CheckLatestVersion checks the latest version from GitHub releases or Go proxy.
func (r *realVersionChecker) CheckLatestVersion(ctx context.Context, repoURL, goModulePath string) (string, error) {
	// Priority 1: Go proxy API for tools with GoModulePath
	if goModulePath != "" {
		return r.checkGoProxyVersion(ctx, goModulePath)
	}

	// Priority 2: go.dev API for Go itself
	if repoURL == "https://go.dev" || repoURL == "https://github.com/golang/go" {
		return r.checkGoVersion(ctx)
	}

	// Priority 3: gh CLI if available and preferred
	if r.useGHCLI {
		version, err := r.checkViaGHCLI(ctx, repoURL)
		if err == nil {
			return version, nil
		}
		// Fall through to API if gh CLI fails
	}

	// Priority 4: GitHub API
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

	resp, err := r.httpClient.Do(req) //nolint:gosec // G704: URL is constructed from trusted GitHub API constants
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

// ErrGoDevAPI is returned when the go.dev API fails.
var ErrGoDevAPI = errors.New("go.dev API error")

// ErrGoProxyAPI is returned when the Go proxy API fails.
var ErrGoProxyAPI = errors.New("go proxy API error")

// GoProxyAPIURL is the base URL for the Go proxy API.
const GoProxyAPIURL = "https://proxy.golang.org"

// checkGoVersion uses the official go.dev API to check the latest stable Go version.
func (r *realVersionChecker) checkGoVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", GoDevAPIURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req) //nolint:gosec // G704: URL is the trusted GoDevAPIURL constant
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status %d: %s", ErrGoDevAPI, resp.StatusCode, string(body))
	}

	var releases []GoRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse go.dev releases JSON: %w", err)
	}

	// Find the first stable release
	for _, release := range releases {
		if release.Stable {
			return release.Version, nil
		}
	}

	return "", fmt.Errorf("%w: no stable releases found", ErrGoDevAPI)
}

// checkGoProxyVersion uses the Go proxy API to check the latest version of a Go module.
func (r *realVersionChecker) checkGoProxyVersion(ctx context.Context, modulePath string) (string, error) {
	// Build the proxy URL: https://proxy.golang.org/{module}/@latest
	apiURL := fmt.Sprintf("%s/%s/@latest", GoProxyAPIURL, modulePath)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req) //nolint:gosec // G704: URL is constructed from the trusted GoProxyAPIURL constant
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status %d: %s", ErrGoProxyAPI, resp.StatusCode, string(body))
	}

	var info GoProxyInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("failed to parse Go proxy JSON: %w", err)
	}

	if info.Version == "" {
		return "", fmt.Errorf("%w: empty version in response", ErrGoProxyAPI)
	}

	return info.Version, nil
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
	checker            VersionChecker
	updater            FileUpdater
	logger             VersionLogger
	dryRun             bool
	allowMajorUpgrades bool
	delay              time.Duration
}

// NewVersionUpdateService creates a new version update service.
func NewVersionUpdateService(checker VersionChecker, updater FileUpdater, logger VersionLogger, dryRun, allowMajorUpgrades bool, delay time.Duration) *VersionUpdateService {
	return &VersionUpdateService{
		checker:            checker,
		updater:            updater,
		logger:             logger,
		dryRun:             dryRun,
		allowMajorUpgrades: allowMajorUpgrades,
		delay:              delay,
	}
}

// GetToolDefinitions returns the list of tools to check with their GitHub repos.
func GetToolDefinitions() map[string]*ToolInfo {
	tools := make(map[string]*ToolInfo)

	// Define unique tools with their GitHub repos (from .env.base comments)
	definitions := []struct {
		key          string
		envVars      []string
		repoOwner    string
		repoName     string
		goModulePath string // Go module path for proxy.golang.org lookup (optional)
	}{
		{"go-coverage", []string{"GO_COVERAGE_VERSION"}, "mrz1836", "go-coverage", ""},
		{"mage-x", []string{"MAGE_X_VERSION"}, "mrz1836", "mage-x", ""},
		{"gitleaks", []string{"MAGE_X_GITLEAKS_VERSION", "GITLEAKS_VERSION", "GO_PRE_COMMIT_GITLEAKS_VERSION"}, "gitleaks", "gitleaks", ""},
		{"gofumpt", []string{"MAGE_X_GOFUMPT_VERSION", "GO_PRE_COMMIT_FUMPT_VERSION"}, "mvdan", "gofumpt", ""},
		{"golangci-lint", []string{"MAGE_X_GOLANGCI_LINT_VERSION", "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION"}, "golangci", "golangci-lint", ""},
		{"goreleaser", []string{"MAGE_X_GORELEASER_VERSION"}, "goreleaser", "goreleaser", ""},
		{"govulncheck", []string{"MAGE_X_GOVULNCHECK_VERSION", "GOVULNCHECK_VERSION"}, "golang", "vuln", ""},
		{"mockgen", []string{"MAGE_X_MOCKGEN_VERSION"}, "uber-go", "mock", ""},
		{"nancy", []string{"MAGE_X_NANCY_VERSION", "NANCY_VERSION"}, "sonatype-nexus-community", "nancy", ""},
		{"staticcheck", []string{"MAGE_X_STATICCHECK_VERSION"}, "dominikh", "go-tools", ""},
		{"swag", []string{"MAGE_X_SWAG_VERSION"}, "swaggo", "swag", ""},
		{"yamlfmt", []string{"MAGE_X_YAMLFMT_VERSION"}, "google", "yamlfmt", ""},
		{"go-pre-commit", []string{"GO_PRE_COMMIT_VERSION"}, "mrz1836", "go-pre-commit", ""},
		{"mage", []string{"MAGE_X_MAGE_VERSION"}, "magefile", "mage", ""},
		// Go proxy-based tools (use pseudo-versions like v0.0.0-YYYYMMDDHHMMSS-commitSHA)
		{"benchstat", []string{"MAGE_X_BENCHSTAT_VERSION"}, "", "", "golang.org/x/perf"},
		// Guardian CI tools
		{"act", []string{"GUARDIAN_ACT_VERSION"}, "nektos", "act", ""},
		{"actionlint", []string{"GUARDIAN_ACTIONLINT_VERSION"}, "rhysd", "actionlint", ""},
		{"go-sarif", []string{"GUARDIAN_GO_SARIF_VERSION"}, "owenrumney", "go-sarif", ""},
	}

	for _, def := range definitions {
		var repoURL string
		if def.repoOwner != "" && def.repoName != "" {
			repoURL = fmt.Sprintf("https://github.com/%s/%s", def.repoOwner, def.repoName)
		}
		tools[def.key] = &ToolInfo{
			EnvVars:      def.envVars,
			RepoURL:      repoURL,
			RepoOwner:    def.repoOwner,
			RepoName:     def.repoName,
			GoModulePath: def.goModulePath,
		}
	}

	// Special case: Go itself uses go.dev API instead of GitHub releases
	tools["go"] = &ToolInfo{
		EnvVars:   []string{"GOVULNCHECK_GO_VERSION"},
		RepoURL:   "https://go.dev", // Triggers special handling in CheckLatestVersion
		RepoOwner: "golang",
		RepoName:  "go",
	}

	return tools
}

// Run executes the version update process across the given env files.
func (s *VersionUpdateService) Run(ctx context.Context, envFiles []string) error {
	// Log the execution mode immediately
	if s.dryRun {
		s.logger.Info("🔍 Running in DRY RUN mode - no changes will be applied")
	} else {
		s.logger.Info("✏️  Running in UPDATE mode - changes will be applied")
	}
	s.logger.Info("")

	// Read all env files and combine content for version extraction
	fileContents := make(map[string][]byte, len(envFiles))
	var combinedContent []byte
	for _, filePath := range envFiles {
		content, err := s.updater.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		fileContents[filePath] = content
		combinedContent = append(combinedContent, content...)
		combinedContent = append(combinedContent, '\n')
	}

	s.logger.Info(fmt.Sprintf("📂 Loaded %d env files for version checking", len(envFiles)))

	// Get tool definitions
	tools := GetToolDefinitions()

	// Calculate estimated time
	toolCount := len(tools)
	// Estimate: (n-1) delays between n requests + avg API response time per tool (~2 seconds)
	estimatedSeconds := int((time.Duration(toolCount-1) * s.delay).Seconds()) + (toolCount * 2)

	s.logger.Info(fmt.Sprintf("🔍 Checking %d tools for updates (estimated time: ~%d seconds)...", toolCount, estimatedSeconds))
	s.logger.Info("")

	// Extract current versions from combined content
	currentVersions := s.extractVersions(combinedContent, tools)

	// Check latest versions
	results := s.checkVersions(ctx, tools, currentVersions)

	// Add blank line before results table
	s.logger.Info("")

	// Display results
	s.displayResults(results)

	// Update files if needed
	if !s.dryRun && s.hasUpdates(results) {
		return s.updateFiles(fileContents, results)
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

		// Show progress
		s.logger.Info(fmt.Sprintf("[%d/%d] Checking %s...", i+1, len(toolKeys), toolKey))

		currentVersion := currentVersions[toolKey]
		latestVersion, err := s.checker.CheckLatestVersion(ctx, tool.RepoURL, tool.GoModulePath)

		result := CheckResult{
			Tool:           toolKey,
			EnvVars:        tool.EnvVars,
			CurrentVersion: currentVersion,
			LatestVersion:  latestVersion,
		}

		if err != nil {
			result.Status = "error"
			result.Error = err
		} else if currentVersion == "latest" {
			// Special case: "latest" resolves to actual version - recommend pinning for reproducibility
			result.Status = "pin-recommended"
		} else if s.normalizeVersion(currentVersion) == s.normalizeVersion(latestVersion) {
			result.Status = "up-to-date"
		} else if s.isMajorUpgrade(currentVersion, latestVersion) && !s.allowMajorUpgrades {
			// Major upgrade detected but not allowed - skip with notification
			result.Status = "major-skipped"
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
	version = strings.TrimPrefix(version, "v")
	// Remove 'go' prefix if present (for Go version comparison)
	version = strings.TrimPrefix(version, "go")
	return version
}

// extractMajorVersion extracts the major version number from a version string.
// Returns the major version as a string and a boolean indicating success.
// Examples: "v1.2.3" -> "1", "2.0.0-rc5" -> "2", "go1.25.5" -> "1"
func (s *VersionUpdateService) extractMajorVersion(version string) (string, bool) {
	normalized := s.normalizeVersion(version)
	if normalized == "" {
		return "", false
	}

	// Find the first dot or end of string
	dotIdx := strings.Index(normalized, ".")
	if dotIdx == -1 {
		// No dot found, check if the entire string is a number
		if _, err := fmt.Sscanf(normalized, "%d", new(int)); err == nil {
			return normalized, true
		}
		return "", false
	}

	majorPart := normalized[:dotIdx]
	// Verify it's a valid number
	if _, err := fmt.Sscanf(majorPart, "%d", new(int)); err == nil {
		return majorPart, true
	}
	return "", false
}

// isMajorUpgrade checks if the latest version is a major upgrade from the current version.
// A major upgrade is when the major version number increases (e.g., v1.x.x -> v2.x.x).
func (s *VersionUpdateService) isMajorUpgrade(current, latest string) bool {
	currentMajor, currentOk := s.extractMajorVersion(current)
	latestMajor, latestOk := s.extractMajorVersion(latest)

	if !currentOk || !latestOk {
		return false
	}

	var currentNum, latestNum int
	if _, err := fmt.Sscanf(currentMajor, "%d", &currentNum); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(latestMajor, "%d", &latestNum); err != nil {
		return false
	}

	return latestNum > currentNum
}

// truncateVersion truncates long version strings with middle ellipsis.
// Versions longer than maxLen become "start...end" format.
func truncateVersion(version string, maxLen int) string {
	if len(version) <= maxLen {
		return version
	}
	// Account for 3 chars for "..."
	available := maxLen - 3
	startLen := (available * 2) / 3 // Keep more from start (prefix matters)
	endLen := available - startLen
	return version[:startLen] + "..." + version[len(version)-endLen:]
}

// displayResults displays the check results in a formatted table.
func (s *VersionUpdateService) displayResults(results []CheckResult) {
	// Print header
	header := fmt.Sprintf("%-20s %-22s %-22s %s\n", "Tool", "Current", "Latest", "Status")
	_, _ = os.Stdout.WriteString(header)
	_, _ = os.Stdout.WriteString(strings.Repeat("─", 85) + "\n")

	// Track statistics
	upToDate := 0
	updates := 0
	majorSkipped := 0
	pinRecommended := 0
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
		case "major-skipped":
			// Extract major versions for display
			currentMajor, _ := s.extractMajorVersion(result.CurrentVersion)
			latestMajor, _ := s.extractMajorVersion(result.LatestVersion)
			statusIcon = fmt.Sprintf("⏭ Major upgrade skipped (v%s→v%s)", currentMajor, latestMajor)
			majorSkipped++
		case "pin-recommended":
			statusIcon = "📌 Pin recommended"
			pinRecommended++
		case "error":
			statusIcon = fmt.Sprintf("✗ Error: %v", result.Error)
			errors++
		}

		line := fmt.Sprintf("%-20s %-22s %-22s %s\n",
			result.Tool,
			truncateVersion(result.CurrentVersion, 20),
			truncateVersion(result.LatestVersion, 20),
			statusIcon,
		)
		_, _ = os.Stdout.WriteString(line)
	}

	// Print summary
	_, _ = os.Stdout.WriteString("\n")
	_, _ = os.Stdout.WriteString("Summary:\n")
	_, _ = fmt.Fprintf(os.Stdout, "✓ %d tools up to date\n", upToDate)
	_, _ = fmt.Fprintf(os.Stdout, "⬆ %d tools with updates available\n", updates)
	if majorSkipped > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "⏭ %d major upgrades skipped (use ALLOW_MAJOR_UPGRADES=true to apply)\n", majorSkipped)
	}
	if pinRecommended > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "📌 %d tools recommend version pinning\n", pinRecommended)
	}
	_, _ = fmt.Fprintf(os.Stdout, "✗ %d tools failed to check\n", errors)
	_, _ = os.Stdout.WriteString("\n")

	if s.dryRun && (updates > 0 || pinRecommended > 0) {
		s.logger.Info("[DRY RUN] No changes made. Set UPDATE_VERSIONS=true to apply updates.")
	}
}

// hasUpdates checks if any updates are available or pinning is recommended.
func (s *VersionUpdateService) hasUpdates(results []CheckResult) bool {
	for _, result := range results {
		if result.Status == "update-available" || result.Status == "pin-recommended" {
			return true
		}
	}
	return false
}

// updateFiles updates env files that have matching env vars with new versions.
// Only files with actual changes are backed up and written.
func (s *VersionUpdateService) updateFiles(fileContents map[string][]byte, results []CheckResult) error {
	s.logger.Info("📝 Updating env files...")

	updatedCount := 0

	// Process each file
	for filePath, originalContent := range fileContents {
		newContent := make([]byte, len(originalContent))
		copy(newContent, originalContent)

		for _, result := range results {
			if result.Status != "update-available" {
				continue
			}

			for _, envVar := range result.EnvVars {
				// Match the env var and capture its current version to detect format
				pattern := regexp.MustCompile(fmt.Sprintf(`(?m)^(%s=)([^#\s]+)(\s|$)`,
					regexp.QuoteMeta(envVar)))

				newContent = pattern.ReplaceAllFunc(newContent, func(match []byte) []byte {
					submatches := pattern.FindSubmatch(match)
					if len(submatches) < 4 {
						return match
					}
					prefix := submatches[1]     // "ENV_VAR="
					currentVal := submatches[2] // current version value
					suffix := submatches[3]     // trailing whitespace or EOL

					// Preserve the prefix format of the original value
					hasVPrefix := strings.HasPrefix(string(currentVal), "v")
					hasGoPrefix := strings.HasPrefix(string(currentVal), "go")
					newVersion := result.LatestVersion

					// Handle 'v' prefix
					if hasVPrefix && !strings.HasPrefix(newVersion, "v") {
						newVersion = "v" + newVersion
					} else if !hasVPrefix && strings.HasPrefix(newVersion, "v") {
						newVersion = strings.TrimPrefix(newVersion, "v")
					}

					// Handle 'go' prefix (for Go version)
					if hasGoPrefix && !strings.HasPrefix(newVersion, "go") {
						newVersion = "go" + newVersion
					} else if !hasGoPrefix && strings.HasPrefix(newVersion, "go") {
						newVersion = strings.TrimPrefix(newVersion, "go")
					}

					return append(append(prefix, []byte(newVersion)...), suffix...)
				})
			}
		}

		// Only backup and write if content actually changed
		if !bytes.Equal(originalContent, newContent) {
			if err := s.updater.BackupFile(filePath); err != nil {
				return fmt.Errorf("failed to create backup for %s: %w", filePath, err)
			}
			s.logger.Info("✓ Backup created: " + filePath + ".backup")

			if err := s.updater.WriteFile(filePath, newContent, 0o600); err != nil {
				return fmt.Errorf("failed to write file %s: %w", filePath, err)
			}
			updatedCount++
			s.logger.Info("✓ Updated: " + filepath.Base(filePath))
		}
	}

	s.logger.Info(fmt.Sprintf("✓ %d file(s) updated successfully", updatedCount))
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
func getVersionUpdateService(dryRun, allowMajorUpgrades bool) *VersionUpdateService {
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
		versionUpdateService = NewVersionUpdateService(checker, updater, logger, dryRun, allowMajorUpgrades, delay)
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

// discoverEnvFiles scans a directory for eligible .env files.
// It returns sorted file paths, excluding files prefixed with 90- or 99-
// (project overrides and local development) and non-.env files.
func discoverEnvFiles(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read env directory %s: %w", dirPath, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Only process .env files
		if !strings.HasSuffix(name, ".env") {
			continue
		}

		// Exclude project overrides (90-*) and local development (99-*)
		if strings.HasPrefix(name, "90-") || strings.HasPrefix(name, "99-") {
			continue
		}

		files = append(files, filepath.Join(dirPath, name))
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("%w in %s", ErrNoEnvFiles, dirPath)
	}

	sort.Strings(files)
	return files, nil
}

// RunVersionUpdate runs the version update process.
func RunVersionUpdate(dryRun, allowMajorUpgrades bool) error {
	ctx := context.Background()

	// Discover env files in .github/env/
	envDirPath := filepath.Join(".github", "env")
	envFiles, err := discoverEnvFiles(envDirPath)
	if err != nil {
		return fmt.Errorf("failed to discover env files: %w", err)
	}

	service := getVersionUpdateService(dryRun, allowMajorUpgrades)
	return service.Run(ctx, envFiles)
}
