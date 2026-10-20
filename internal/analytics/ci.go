package analytics

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	// CIWorkerLimit is the max number of concurrent CI metric collection workers.
	// Kept low (2) so the shared rate limiter can govern throughput effectively.
	CIWorkerLimit = 2

	// GoFortressWorkflowName is the name of the GoFortress CI workflow
	GoFortressWorkflowName = "GoFortress"

	// maxWorkflowRunsToCheck is the number of recent successful runs to fetch.
	// If the latest run's artifacts are expired, older runs are tried as fallback.
	// This uses a single paginated API request regardless of count.
	maxWorkflowRunsToCheck = 5

	// Artifact names used by GoFortress
	artifactLOCStats         = "loc-stats"
	artifactStatistics       = "statistics-section"
	artifactCoverageInternal = "coverage-stats-internal"
	artifactCoverageCodecov  = "coverage-stats-codecov" // fallback: repos using codecov provider upload this name
	artifactTestsSection     = "tests-section"
	artifactBenchPrefix      = "bench-stats-"
	artifactCIResultsPrefix  = "ci-results-"
)

// CIMetrics holds parsed CI metrics for a single repository
type CIMetrics struct {
	WorkflowRunID      int64
	WorkflowRunAttempt int // Attempt of the run the metrics came from; re-running a run keeps its ID
	Branch             string
	CommitSHA          string
	GoFilesLOC         int
	TestFilesLOC       int
	GoFilesCount       int
	TestFilesCount     int
	TestCount          int
	BenchmarkCount     int
	Coverage           *float64

	// ArtifactsComplete reports that every artifact the collector tried to read
	// was downloaded, so no metric was left unset by a transient download failure.
	ArtifactsComplete bool

	// Reused reports that the metrics were carried forward from the previous
	// snapshot because the newest successful run had not changed since it.
	Reused bool
}

// CICollector handles concurrent CI metrics collection from GoFortress artifacts
type CICollector struct {
	ghClient gh.Client
	logger   *logrus.Logger
	throttle *Throttle
}

// NewCICollector creates a new CI metrics collector.
// throttle may be nil for unthrottled operation.
func NewCICollector(ghClient gh.Client, logger *logrus.Logger, throttle *Throttle) *CICollector {
	return &CICollector{
		ghClient: ghClient,
		logger:   logger,
		throttle: throttle,
	}
}

// CollectCIMetrics fetches CI metrics for multiple repositories concurrently.
// Returns a map of repo full name to CIMetrics. Repos without GoFortress are gracefully skipped.
func (c *CICollector) CollectCIMetrics(ctx context.Context, repos []gh.RepoInfo) (map[string]*CIMetrics, error) {
	return c.CollectCIMetricsWithBaselines(ctx, repos, nil)
}

// CollectCIMetricsWithBaselines fetches CI metrics like CollectCIMetrics, but
// reuses a repository's baseline (its previous metrics) instead of downloading
// artifacts again when the baseline came from the newest successful run.
//
// Parameters:
// - ctx: Context for cancellation of the API calls
// - repos: Repositories to collect metrics for
// - baselines: Previous metrics keyed by repository full name; missing or nil
// entries are collected in full
//
// Returns:
// - A map of repo full name to CIMetrics; reused entries have Reused set
// - An error only if the collection itself fails (per-repo failures are skipped)
//
// Notes:
// - A baseline is reused only when it is certain to equal a fresh collection;
// see reusableCIMetrics
func (c *CICollector) CollectCIMetricsWithBaselines(
	ctx context.Context,
	repos []gh.RepoInfo,
	baselines map[string]*CIMetrics,
) (map[string]*CIMetrics, error) {
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
			metrics, err := c.collectRepoCI(ctx, repo.FullName, baselines[repo.FullName])
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
// Tries multiple recent successful runs to handle expired artifacts, unless
// baseline already holds the metrics of the newest successful run.
func (c *CICollector) collectRepoCI(ctx context.Context, repo string, baseline *CIMetrics) (*CIMetrics, error) {
	// Step 1: Find GoFortress workflow
	var workflows []gh.Workflow
	err := c.doAPI(ctx, "list-workflows:"+repo, func() error {
		var apiErr error
		workflows, apiErr = c.ghClient.ListWorkflows(ctx, repo)
		return apiErr
	})
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

	// Step 2: Get recent successful runs (fetch several in case latest has expired artifacts)
	var runs []gh.WorkflowRun
	err = c.doAPI(ctx, "get-workflow-runs:"+repo, func() error {
		var apiErr error
		runs, apiErr = c.ghClient.GetWorkflowRuns(ctx, repo, fortressID, maxWorkflowRunsToCheck)
		return apiErr
	})
	if err != nil {
		return nil, fmt.Errorf("get workflow runs: %w", err)
	}
	if len(runs) == 0 {
		return nil, nil //nolint:nilnil // nil signals no successful runs — caller checks for nil
	}

	// Step 3: Reuse the previous metrics when they came from the newest successful run.
	if reused := reusableCIMetrics(baseline, runs[0]); reused != nil {
		if c.logger != nil {
			c.logger.WithFields(logrus.Fields{
				"repo":        repo,
				"run_id":      reused.WorkflowRunID,
				"run_attempt": reused.WorkflowRunAttempt,
			}).Debug("Newest successful GoFortress run is unchanged; reusing its CI metrics")
		}
		return reused, nil
	}

	// Step 4: Try runs in order until we find one with valid coverage artifacts.
	// Keep the first result as fallback in case no run has valid coverage.
	var fallbackMetrics *CIMetrics

	for _, run := range runs {
		metrics, hasCoverage := c.tryRunArtifacts(ctx, repo, run)
		if metrics == nil {
			continue
		}

		if hasCoverage {
			return metrics, nil // Found a run with valid coverage — use it
		}

		// Keep the first metrics as fallback (most recent run, best available)
		if fallbackMetrics == nil {
			fallbackMetrics = metrics
		}

		if c.logger != nil {
			c.logger.WithFields(logrus.Fields{
				"repo":   repo,
				"run_id": run.ID,
			}).Debug("No valid coverage artifacts in run, trying older run")
		}
	}

	// No run had valid coverage; return the best we have (latest run's metrics without coverage)
	return fallbackMetrics, nil
}

// reusableCIMetrics returns a copy of baseline marked as reused when collecting
// newest again would certainly reproduce it, or nil when it must be collected.
//
// The baseline is reused only when all of these hold:
// - It came from newest itself: same run ID and same run attempt. Re-running a
// run keeps its ID but increments the attempt and uploads new artifacts.
// - It has coverage, so the collector would stop at newest rather than search
// older runs, whose availability can change.
// - Every artifact it tried to read was downloaded, so no metric is missing
// because of a transient failure that a new download might not repeat.
//
// A completed attempt's artifacts never change, so under these conditions a new
// download yields the same metrics. Reuse also keeps the metrics of an unchanged
// newest run available after GitHub expires its artifacts.
func reusableCIMetrics(baseline *CIMetrics, newest gh.WorkflowRun) *CIMetrics {
	if baseline == nil || baseline.Coverage == nil || !baseline.ArtifactsComplete {
		return nil
	}
	if baseline.WorkflowRunAttempt < 1 ||
		baseline.WorkflowRunID != newest.ID ||
		baseline.WorkflowRunAttempt != newest.RunAttempt {
		return nil
	}

	reused := *baseline
	coverage := *baseline.Coverage
	reused.Coverage = &coverage
	reused.Reused = true
	return &reused
}

// tryRunArtifacts fetches and parses artifacts for a single workflow run.
// Returns the parsed metrics and whether valid (non-expired) coverage artifacts were available.
// Returns (nil, false) if artifacts could not be fetched.
func (c *CICollector) tryRunArtifacts(ctx context.Context, repo string, run gh.WorkflowRun) (*CIMetrics, bool) {
	// Fetch artifacts for this run
	var artifacts []gh.Artifact
	err := c.doAPI(ctx, fmt.Sprintf("get-run-artifacts:%s:%d", repo, run.ID), func() error {
		var apiErr error
		artifacts, apiErr = c.ghClient.GetRunArtifacts(ctx, repo, run.ID)
		return apiErr
	})
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"repo":   repo,
				"run_id": run.ID,
			}).Warn("Failed to get run artifacts")
		}
		return nil, false
	}

	// Build artifact name set, filtering out expired artifacts
	artifactNames := make(map[string]bool, len(artifacts))
	expiredCount := 0
	for _, a := range artifacts {
		if a.Expired {
			expiredCount++
			continue
		}
		artifactNames[a.Name] = true
	}

	// Check if valid coverage artifacts exist
	hasCoverage := artifactNames[artifactCoverageInternal] || artifactNames[artifactCoverageCodecov]

	// Log expired artifacts — only warn when coverage is affected, otherwise debug
	if expiredCount > 0 && c.logger != nil {
		entry := c.logger.WithFields(logrus.Fields{
			"repo":          repo,
			"run_id":        run.ID,
			"expired_count": expiredCount,
			"valid_count":   len(artifactNames),
		})
		if !hasCoverage {
			entry.Warn("Coverage artifact expired in workflow run")
		} else {
			entry.Debug("Skipping expired artifacts in workflow run (coverage still valid)")
		}
	}

	metrics := &CIMetrics{
		WorkflowRunID:      run.ID,
		WorkflowRunAttempt: run.RunAttempt,
		Branch:             run.HeadBranch,
		CommitSHA:          run.HeadSHA,
	}

	// Download and parse artifacts
	tmpDir, err := os.MkdirTemp("", "ci-metrics-*")
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to create temp dir for CI metrics")
		}
		return nil, false
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Track whether every attempted artifact was downloaded and read. Parse failures
	// are not tracked: they repeat identically for the same artifact, whereas a
	// download failure may be transient and leave a metric unset only this time.
	metrics.ArtifactsComplete = true
	track := func(read bool) {
		metrics.ArtifactsComplete = metrics.ArtifactsComplete && read
	}

	// Visit artifact names in sorted order so a run always yields the same metrics,
	// even when it has several ci-results artifacts (map iteration order is random).
	sortedNames := slices.Sorted(maps.Keys(artifactNames))

	// LOC: prefer JSON, fall back to markdown
	if artifactNames[artifactLOCStats] {
		track(c.parseLOCArtifact(ctx, repo, run.ID, artifactLOCStats, tmpDir, metrics))
	} else if artifactNames[artifactStatistics] {
		track(c.parseLOCFromMarkdownArtifact(ctx, repo, run.ID, tmpDir, metrics))
	}

	// Coverage: prefer internal provider artifact; fall back to codecov provider artifact.
	// Both upload the same JSON schema (coverage_percentage field). The provider used
	// depends on GO_COVERAGE_PROVIDER in the repo's GoFortress config.
	switch {
	case artifactNames[artifactCoverageInternal]:
		track(c.parseCoverageArtifact(ctx, repo, run.ID, artifactCoverageInternal, tmpDir, metrics))
	case artifactNames[artifactCoverageCodecov]:
		track(c.parseCoverageArtifact(ctx, repo, run.ID, artifactCoverageCodecov, tmpDir, metrics))
	}

	// Tests: prefer ci-results JSON, fall back to tests-section markdown
	ciResultsFound := false
	for _, name := range sortedNames {
		if strings.HasPrefix(name, artifactCIResultsPrefix) {
			track(c.parseCIResultsArtifact(ctx, repo, run.ID, name, tmpDir, metrics))
			ciResultsFound = true
			break
		}
	}
	if !ciResultsFound && artifactNames[artifactTestsSection] {
		track(c.parseTestsArtifact(ctx, repo, run.ID, tmpDir, metrics))
	}

	// Benchmarks: from bench-stats-* JSON files
	for _, name := range sortedNames {
		if strings.HasPrefix(name, artifactBenchPrefix) {
			track(c.parseBenchArtifact(ctx, repo, run.ID, name, tmpDir, metrics))
		}
	}

	// Artifact names only prove that files were advertised by GitHub, not that
	// they downloaded and parsed successfully. Treat an all-zero parse as an
	// unusable run so collectRepoCI can try an older successful workflow run
	// instead of persisting a fresh, bogus zero snapshot.
	if metrics.GoFilesLOC == 0 && metrics.TestFilesLOC == 0 &&
		metrics.GoFilesCount == 0 && metrics.TestFilesCount == 0 &&
		metrics.TestCount == 0 && metrics.BenchmarkCount == 0 && metrics.Coverage == nil {
		if c.logger != nil {
			c.logger.WithFields(logrus.Fields{
				"repo":   repo,
				"run_id": run.ID,
			}).Warn("CI artifacts produced no usable metrics; trying an older run")
		}
		return nil, false
	}

	return metrics, hasCoverage
}

// doAPI executes fn through the throttle if available, or directly if not
func (c *CICollector) doAPI(ctx context.Context, operation string, fn func() error) error {
	if c.throttle != nil {
		return c.throttle.DoWithRetry(ctx, operation, fn)
	}
	return fn()
}

// parseLOCArtifact downloads and parses the loc-stats JSON artifact.
// It reports whether the artifact was downloaded and read.
func (c *CICollector) parseLOCArtifact(ctx context.Context, repo string, runID int64, name, tmpDir string, metrics *CIMetrics) bool {
	artDir := filepath.Join(tmpDir, name)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, name, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to download LOC artifact")
		}
		return false
	}

	data, err := findAndReadJSON(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to read LOC JSON")
		}
		return false
	}

	loc := parseLOCJSON(data)
	if loc != nil {
		metrics.GoFilesLOC = loc.GoFilesLOC
		metrics.TestFilesLOC = loc.TestFilesLOC
		metrics.GoFilesCount = loc.GoFilesCount
		metrics.TestFilesCount = loc.TestFilesCount
	}

	return true
}

// parseLOCFromMarkdownArtifact downloads statistics-section and parses LOC from markdown.
// It reports whether the artifact was downloaded and read.
func (c *CICollector) parseLOCFromMarkdownArtifact(ctx context.Context, repo string, runID int64, tmpDir string, metrics *CIMetrics) bool {
	artDir := filepath.Join(tmpDir, artifactStatistics)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, artifactStatistics, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to download statistics-section artifact")
		}
		return false
	}

	data, err := findAndReadMarkdown(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to read statistics-section markdown")
		}
		return false
	}

	goLOC, testLOC, goFiles, testFiles := parseLOCFromMarkdown(data)
	metrics.GoFilesLOC = goLOC
	metrics.TestFilesLOC = testLOC
	metrics.GoFilesCount = goFiles
	metrics.TestFilesCount = testFiles

	return true
}

// parseCoverageArtifact downloads and parses a coverage JSON artifact by name.
// Both "coverage-stats-internal" and "coverage-stats-codecov" share the same JSON schema.
// It reports whether the artifact was downloaded and read.
func (c *CICollector) parseCoverageArtifact(ctx context.Context, repo string, runID int64, name, tmpDir string, metrics *CIMetrics) bool {
	artDir := filepath.Join(tmpDir, name)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, name, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to download coverage artifact")
		}
		return false
	}

	data, err := findAndReadJSON(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to read coverage JSON")
		}
		return false
	}

	cov := parseCoverageJSON(data)
	if cov != nil {
		metrics.Coverage = cov
	}

	return true
}

// parseTestsArtifact downloads and parses the tests-section markdown artifact.
// It reports whether the artifact was downloaded and read.
func (c *CICollector) parseTestsArtifact(ctx context.Context, repo string, runID int64, tmpDir string, metrics *CIMetrics) bool {
	artDir := filepath.Join(tmpDir, artifactTestsSection)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, artifactTestsSection, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to download tests-section artifact")
		}
		return false
	}

	data, err := findAndReadMarkdown(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).Debug("Failed to read tests-section markdown")
		}
		return false
	}

	metrics.TestCount = parseTestCountFromMarkdown(data)

	return true
}

// parseCIResultsArtifact downloads and parses the ci-results artifact with JSONL test data.
// It reports whether the artifact was downloaded and read.
func (c *CICollector) parseCIResultsArtifact(ctx context.Context, repo string, runID int64, name, tmpDir string, metrics *CIMetrics) bool {
	artDir := filepath.Join(tmpDir, name)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, name, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to download ci-results artifact")
		}
		return false
	}

	// Look for .mage-x/ci-results.jsonl
	jsonlPath := filepath.Join(artDir, ".mage-x", "ci-results.jsonl")
	data, err := os.ReadFile(filepath.Clean(jsonlPath))
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to read ci-results.jsonl")
		}
		return false
	}

	testCount := parseCIResultsJSONL(data)
	if testCount > 0 {
		metrics.TestCount = testCount
	}

	return true
}

// parseBenchArtifact downloads and parses a bench-stats JSON artifact.
// It reports whether the artifact was downloaded and read.
func (c *CICollector) parseBenchArtifact(ctx context.Context, repo string, runID int64, name, tmpDir string, metrics *CIMetrics) bool {
	artDir := filepath.Join(tmpDir, name)
	if err := c.ghClient.DownloadRunArtifact(ctx, repo, runID, name, artDir); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to download bench artifact")
		}
		return false
	}

	data, err := findAndReadJSON(artDir)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("artifact", name).Debug("Failed to read bench JSON")
		}
		return false
	}

	metrics.BenchmarkCount += parseBenchStatsJSON(data)

	return true
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
	var raw map[string]any
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
	var raw map[string]any
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

		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		// Look for the summary entry
		if entryType, ok := entry["type"].(string); ok && entryType == "summary" {
			if summary, ok := entry["summary"].(map[string]any); ok {
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
