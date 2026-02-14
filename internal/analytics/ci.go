package analytics

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// errFileNotFound is a sentinel error for missing artifact files
var errFileNotFound = errors.New("artifact file not found")

const (
	// CIWorkerLimit is the max number of concurrent CI metric collection workers
	CIWorkerLimit = 5

	// GoFortressWorkflowName is the name of the GoFortress CI workflow
	GoFortressWorkflowName = "GoFortress"

	// Artifact names used by GoFortress
	artifactLOCStats         = "loc-stats"
	artifactStatistics       = "statistics-section"
	artifactCoverageInternal = "coverage-stats-internal"
	artifactTestsSection     = "tests-section"
	artifactBenchPrefix      = "bench-stats-"
	artifactCIResultsPrefix  = "ci-results-"
)

// CIMetrics holds parsed CI metrics for a single repository
type CIMetrics struct {
	WorkflowRunID  int64
	Branch         string
	CommitSHA      string
	GoFilesLOC     int
	TestFilesLOC   int
	GoFilesCount   int
	TestFilesCount int
	TestCount      int
	BenchmarkCount int
	Coverage       *float64
}

// CICollector handles concurrent CI metrics collection from GoFortress artifacts
type CICollector struct {
	ghClient gh.Client
	logger   *logrus.Logger
}

// NewCICollector creates a new CI metrics collector
func NewCICollector(ghClient gh.Client, logger *logrus.Logger) *CICollector {
	return &CICollector{
		ghClient: ghClient,
		logger:   logger,
	}
}

// CollectCIMetrics fetches CI metrics for multiple repositories concurrently.
// Returns a map of repo full name to CIMetrics. Repos without GoFortress are gracefully skipped.
func (c *CICollector) CollectCIMetrics(ctx context.Context, repos []gh.RepoInfo) (map[string]*CIMetrics, error) {
	if len(repos) == 0 {
		return make(map[string]*CIMetrics), nil
	}

	if c.logger != nil {
		c.logger.WithField("repo_count", len(repos)).Info("Starting concurrent CI metrics collection")
	}

	results := make(map[string]*CIMetrics)
	var resultMu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(CIWorkerLimit)

	for _, repo := range repos {
		g.Go(func() error {
			metrics, err := c.collectRepoCI(ctx, repo.FullName)
			if err != nil {
				if c.logger != nil {
					c.logger.WithError(err).WithField("repo", repo.FullName).Warn("Failed to collect CI metrics")
				}
				return nil // Don't fail the entire operation
			}

			if metrics != nil {
				resultMu.Lock()
				results[repo.FullName] = metrics
				resultMu.Unlock()

				if c.logger != nil {
					c.logger.WithFields(logrus.Fields{
						"repo":       repo.FullName,
						"run_id":     metrics.WorkflowRunID,
						"go_loc":     metrics.GoFilesLOC,
						"test_count": metrics.TestCount,
					}).Debug("Collected CI metrics")
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("CI metrics collection failed: %w", err)
	}

	if c.logger != nil {
		c.logger.WithField("repos_with_metrics", len(results)).Info("CI metrics collection complete")
	}

	return results, nil
}

// collectRepoCI fetches CI metrics for a single repository.
// Returns nil if the repo doesn't have a GoFortress workflow.
func (c *CICollector) collectRepoCI(ctx context.Context, repo string) (*CIMetrics, error) {
	// Step 1: Find GoFortress workflow
	workflows, err := c.ghClient.ListWorkflows(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}

	var fortressID int64
	for _, wf := range workflows {
		if wf.Name == GoFortressWorkflowName {
			fortressID = wf.ID
			break
		}
	}
	if fortressID == 0 {
		return nil, nil //nolint:nilnil // nil signals no GoFortress workflow — caller checks for nil
	}

	// Step 2: Get latest successful run
	runs, err := c.ghClient.GetWorkflowRuns(ctx, repo, fortressID, 1)
	if err != nil {
		return nil, fmt.Errorf("get workflow runs: %w", err)
	}
	if len(runs) == 0 {
		return nil, nil //nolint:nilnil // nil signals no successful runs — caller checks for nil
	}
	latestRun := runs[0]

	// Step 3: Get artifacts for this run
	artifacts, err := c.ghClient.GetRunArtifacts(ctx, repo, latestRun.ID)
	if err != nil {
		return nil, fmt.Errorf("get run artifacts: %w", err)
	}

	// Build artifact name set for quick lookup
	artifactNames := make(map[string]bool, len(artifacts))
	for _, a := range artifacts {
		artifactNames[a.Name] = true
	}

	metrics := &CIMetrics{
		WorkflowRunID: latestRun.ID,
		Branch:        latestRun.HeadBranch,
		CommitSHA:     latestRun.HeadSHA,
	}

	// Step 4: Download and parse artifacts
	tmpDir, err := os.MkdirTemp("", "ci-metrics-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// LOC: prefer JSON, fall back to markdown
	if artifactNames[artifactLOCStats] {
		c.parseLOCArtifact(ctx, repo, latestRun.ID, artifactLOCStats, tmpDir, metrics)
	} else if artifactNames[artifactStatistics] {
		c.parseLOCFromMarkdownArtifact(ctx, repo, latestRun.ID, tmpDir, metrics)
	}

	// Coverage: from codecov JSON
	if artifactNames[artifactCoverageInternal] {
		c.parseCoverageArtifact(ctx, repo, latestRun.ID, tmpDir, metrics)
	}

	// Tests: prefer ci-results JSON, fall back to tests-section markdown
	ciResultsFound := false
	for name := range artifactNames {
		if strings.HasPrefix(name, artifactCIResultsPrefix) {
			c.parseCIResultsArtifact(ctx, repo, latestRun.ID, name, tmpDir, metrics)
			ciResultsFound = true
			break
		}
	}
	if !ciResultsFound && artifactNames[artifactTestsSection] {
		c.parseTestsArtifact(ctx, repo, latestRun.ID, tmpDir, metrics)
	}

	// Benchmarks: from bench-stats-* JSON files
	for name := range artifactNames {
		if strings.HasPrefix(name, artifactBenchPrefix) {
			c.parseBenchArtifact(ctx, repo, latestRun.ID, name, tmpDir, metrics)
		}
	}

	return metrics, nil
}

// parseLOCArtifact downloads and parses the loc-stats JSON artifact
func (c *CICollector) parseLOCArtifact(ctx context.Context, repo string, runID int64, name, tmpDir string, metrics *CIMetrics) {
	artDir := filepath.Join(tmpDir, name)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, name, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to download LOC artifact")
		}
		return
	}

	data, err := findAndReadJSON(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to read LOC JSON")
		}
		return
	}

	loc := parseLOCJSON(data)
	if loc != nil {
		metrics.GoFilesLOC = loc.GoFilesLOC
		metrics.TestFilesLOC = loc.TestFilesLOC
		metrics.GoFilesCount = loc.GoFilesCount
		metrics.TestFilesCount = loc.TestFilesCount
	}
}

// parseLOCFromMarkdownArtifact downloads statistics-section and parses LOC from markdown
func (c *CICollector) parseLOCFromMarkdownArtifact(ctx context.Context, repo string, runID int64, tmpDir string, metrics *CIMetrics) {
	artDir := filepath.Join(tmpDir, artifactStatistics)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, artifactStatistics, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to download statistics-section artifact")
		}
		return
	}

	data, err := findAndReadMarkdown(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to read statistics-section markdown")
		}
		return
	}

	goLOC, testLOC, goFiles, testFiles := parseLOCFromMarkdown(data)
	metrics.GoFilesLOC = goLOC
	metrics.TestFilesLOC = testLOC
	metrics.GoFilesCount = goFiles
	metrics.TestFilesCount = testFiles
}

// parseCoverageArtifact downloads and parses the coverage JSON artifact
func (c *CICollector) parseCoverageArtifact(ctx context.Context, repo string, runID int64, tmpDir string, metrics *CIMetrics) {
	artDir := filepath.Join(tmpDir, artifactCoverageInternal)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, artifactCoverageInternal, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to download coverage artifact")
		}
		return
	}

	data, err := findAndReadJSON(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to read coverage JSON")
		}
		return
	}

	cov := parseCoverageJSON(data)
	if cov != nil {
		metrics.Coverage = cov
	}
}

// parseTestsArtifact downloads and parses the tests-section markdown artifact
func (c *CICollector) parseTestsArtifact(ctx context.Context, repo string, runID int64, tmpDir string, metrics *CIMetrics) {
	artDir := filepath.Join(tmpDir, artifactTestsSection)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, artifactTestsSection, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to download tests-section artifact")
		}
		return
	}

	data, err := findAndReadMarkdown(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to read tests-section markdown")
		}
		return
	}

	metrics.TestCount = parseTestCountFromMarkdown(data)
}

// parseCIResultsArtifact downloads and parses the ci-results artifact with JSONL test data
func (c *CICollector) parseCIResultsArtifact(ctx context.Context, repo string, runID int64, name, tmpDir string, metrics *CIMetrics) {
	artDir := filepath.Join(tmpDir, name)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, name, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to download ci-results artifact")
		}
		return
	}

	// Look for .mage-x/ci-results.jsonl
	jsonlPath := filepath.Join(artDir, ".mage-x", "ci-results.jsonl")
	data, err := os.ReadFile(filepath.Clean(jsonlPath))
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to read ci-results.jsonl")
		}
		return
	}

	testCount := parseCIResultsJSONL(data)
	if testCount > 0 {
		metrics.TestCount = testCount
	}
}

// parseBenchArtifact downloads and parses a bench-stats JSON artifact
func (c *CICollector) parseBenchArtifact(ctx context.Context, repo string, runID int64, name, tmpDir string, metrics *CIMetrics) {
	artDir := filepath.Join(tmpDir, name)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, name, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to download bench artifact")
		}
		return
	}

	data, err := findAndReadJSON(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to read bench JSON")
		}
		return
	}

	metrics.BenchmarkCount += parseBenchStatsJSON(data)
}

// ============================================================
// Artifact parsing helpers
// ============================================================

// LOCData represents parsed LOC statistics from JSON
type LOCData struct {
	GoFilesLOC     int `json:"go_files_loc"`
	TestFilesLOC   int `json:"test_files_loc"`
	GoFilesCount   int `json:"go_files_count"`
	TestFilesCount int `json:"test_files_count"`
}

// parseLOCJSON parses the loc-stats JSON artifact
func parseLOCJSON(data []byte) *LOCData {
	var loc LOCData
	if err := json.Unmarshal(data, &loc); err != nil {
		return nil
	}
	// Validate we got meaningful data
	if loc.GoFilesLOC == 0 && loc.TestFilesLOC == 0 && loc.GoFilesCount == 0 && loc.TestFilesCount == 0 {
		return nil
	}
	return &loc
}

// parseLOCFromMarkdown extracts LOC data from the statistics-section markdown table.
// Expects a table with rows like "| Test Files | 1,234 | 56 | ..." and "| Go Files | 5,678 | 90 | ..."
func parseLOCFromMarkdown(data []byte) (goLOC, testLOC, goFiles, testFiles int) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "|") {
			continue
		}

		cols := strings.Split(line, "|")
		if len(cols) < 4 {
			continue
		}

		rowType := strings.TrimSpace(cols[1])
		locStr := strings.TrimSpace(cols[2])
		filesStr := strings.TrimSpace(cols[3])

		switch {
		case strings.Contains(rowType, "Test Files"):
			testLOC = parseFormattedInt(locStr)
			testFiles = parseFormattedInt(filesStr)
		case strings.Contains(rowType, "Go Files"):
			goLOC = parseFormattedInt(locStr)
			goFiles = parseFormattedInt(filesStr)
		}
	}
	return goLOC, testLOC, goFiles, testFiles
}

// parseCoverageJSON extracts coverage_percentage from codecov JSON
func parseCoverageJSON(data []byte) *float64 {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	// Try both field names: coverage_percentage and coverage_percent
	for _, key := range []string{"coverage_percentage", "coverage_percent"} {
		if val, ok := raw[key]; ok {
			switch v := val.(type) {
			case float64:
				return &v
			case string:
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					return &f
				}
			}
		}
	}

	return nil
}

// parseBenchStatsJSON extracts benchmark_count from a bench-stats JSON file
func parseBenchStatsJSON(data []byte) int {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0
	}

	if count, ok := raw["benchmark_count"]; ok {
		switch v := count.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}

	return 0
}

// parseCIResultsJSONL extracts test count from ci-results.jsonl (JSONL format with summary line)
func parseCIResultsJSONL(data []byte) int {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Bytes()

		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		// Look for the summary entry
		if entryType, ok := entry["type"].(string); ok && entryType == "summary" {
			if summary, ok := entry["summary"].(map[string]interface{}); ok {
				// Prefer unique_total (actual unique test functions)
				if uniqueTotal, ok := summary["unique_total"]; ok {
					switch v := uniqueTotal.(type) {
					case float64:
						return int(v)
					case int:
						return v
					}
				}
				// Fallback to total if unique_total not present
				if total, ok := summary["total"]; ok {
					switch v := total.(type) {
					case float64:
						return int(v)
					case int:
						return v
					}
				}
			}
		}
	}
	return 0
}

// parseTestCountFromMarkdown extracts the total test count from the tests-section markdown.
// Looks for patterns like "X tests" or "X total" in markdown tables.
var testCountRegex = regexp.MustCompile(`(\d[\d,]*)\s+(?:tests?|total)`)

func parseTestCountFromMarkdown(data []byte) int {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	totalTests := 0
	for scanner.Scan() {
		line := scanner.Text()
		matches := testCountRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			n := parseFormattedInt(matches[1])
			if n > totalTests {
				totalTests = n
			}
		}
	}
	return totalTests
}

// ============================================================
// File helpers
// ============================================================

// findAndReadJSON finds and reads the first JSON file in a directory
func findAndReadJSON(dir string) ([]byte, error) {
	return findAndReadFile(dir, ".json")
}

// findAndReadMarkdown finds and reads the first markdown file in a directory
func findAndReadMarkdown(dir string) ([]byte, error) {
	return findAndReadFile(dir, ".md")
}

// findAndReadFile finds and reads the first file with the given extension in a directory
func findAndReadFile(dir, ext string) ([]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ext) {
			filePath := filepath.Join(dir, entry.Name())
			cleanPath := filepath.Clean(filePath)

			return os.ReadFile(cleanPath)
		}
	}

	return nil, fmt.Errorf("%w: no %s file in %s", errFileNotFound, ext, dir)
}

// parseFormattedInt parses an integer that may contain commas (e.g., "1,234")
func parseFormattedInt(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	// Strip bold markdown formatting
	s = strings.ReplaceAll(s, "**", "")
	n, _ := strconv.Atoi(s)
	return n
}
