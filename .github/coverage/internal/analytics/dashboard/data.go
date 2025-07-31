package dashboard

import (
	"time"
)

// CoverageData represents the complete coverage data for dashboard generation
type CoverageData struct {
	// Project information
	ProjectName      string    `json:"project_name"`
	RepositoryURL    string    `json:"repository_url"`
	Branch           string    `json:"branch"`
	CommitSHA        string    `json:"commit_sha"`
	PRNumber         string    `json:"pr_number,omitempty"`
	PRTitle          string    `json:"pr_title,omitempty"`
	BaselineCoverage float64   `json:"baseline_coverage,omitempty"`
	Timestamp        time.Time `json:"timestamp"`

	// Overall metrics
	TotalCoverage float64 `json:"total_coverage"`
	TotalLines    int     `json:"total_lines"`
	CoveredLines  int     `json:"covered_lines"`
	MissedLines   int     `json:"missed_lines"`

	// File metrics
	TotalFiles     int `json:"total_files"`
	CoveredFiles   int `json:"covered_files"`
	PartialFiles   int `json:"partial_files"`
	UncoveredFiles int `json:"uncovered_files"`

	// Package metrics
	Packages []PackageCoverage `json:"packages"`

	// Trend data
	TrendData *TrendData `json:"trend_data,omitempty"`

	// Historical data
	History []HistoricalPoint `json:"history,omitempty"`

	// Build status information
	BuildStatus *BuildStatus `json:"build_status,omitempty"`

	// Workflow run context
	WorkflowRunNumber int  `json:"workflow_run_number,omitempty"`
	IsFirstRun        bool `json:"is_first_run,omitempty"`
	HasPreviousRuns   bool `json:"has_previous_runs,omitempty"`
}

// PackageCoverage represents coverage data for a single package
type PackageCoverage struct {
	Name         string             `json:"name"`
	Path         string             `json:"path"`
	Coverage     float64            `json:"coverage"`
	TotalLines   int                `json:"total_lines"`
	CoveredLines int                `json:"covered_lines"`
	MissedLines  int                `json:"missed_lines"`
	GitHubURL    string             `json:"github_url,omitempty"`
	Files        []FileCoverage     `json:"files"`
	Functions    []FunctionCoverage `json:"functions,omitempty"`
}

// FileCoverage represents coverage data for a single file
type FileCoverage struct {
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	Coverage     float64     `json:"coverage"`
	TotalLines   int         `json:"total_lines"`
	CoveredLines int         `json:"covered_lines"`
	MissedLines  int         `json:"missed_lines"`
	GitHubURL    string      `json:"github_url,omitempty"`
	LineHits     map[int]int `json:"line_hits,omitempty"` // Line number -> hit count
}

// FunctionCoverage represents coverage data for a single function
type FunctionCoverage struct {
	Name         string  `json:"name"`
	StartLine    int     `json:"start_line"`
	EndLine      int     `json:"end_line"`
	Coverage     float64 `json:"coverage"`
	CoveredLines int     `json:"covered_lines"`
	TotalLines   int     `json:"total_lines"`
}

// TrendData represents coverage trend information
type TrendData struct {
	Direction       string  `json:"direction"` // up, down, stable
	ChangePercent   float64 `json:"change_percent"`
	ChangeLines     int     `json:"change_lines"`
	ComparedTo      string  `json:"compared_to"` // branch or commit
	ComparedToValue string  `json:"compared_to_value"`
}

// HistoricalPoint represents a single point in coverage history
type HistoricalPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	CommitSHA    string    `json:"commit_sha"`
	Coverage     float64   `json:"coverage"`
	TotalLines   int       `json:"total_lines"`
	CoveredLines int       `json:"covered_lines"`
}

// BranchInfo represents information about a branch
type BranchInfo struct {
	Name         string    `json:"name"`
	IsProtected  bool      `json:"is_protected"`
	Coverage     float64   `json:"coverage"`
	LastUpdate   time.Time `json:"last_update"`
	BadgeURL     string    `json:"badge_url"`
	ReportURL    string    `json:"report_url"`
	CommitSHA    string    `json:"commit_sha"`
	CommitURL    string    `json:"commit_url"`
	TotalLines   int       `json:"total_lines"`
	CoveredLines int       `json:"covered_lines"`
}

// Metadata contains metadata for the dashboard
type Metadata struct {
	GeneratedAt      time.Time    `json:"generated_at"`
	GeneratorVersion string       `json:"generator_version"`
	DataVersion      string       `json:"data_version"`
	Branches         []BranchInfo `json:"branches"`
	LastUpdated      time.Time    `json:"last_updated"`
}

// QualityStatus represents the quality gate status
type QualityStatus struct {
	Passed      bool    `json:"passed"`
	Threshold   float64 `json:"threshold"`
	ActualValue float64 `json:"actual_value"`
	Message     string  `json:"message"`
}

// CoverageSummary provides a high-level summary
type CoverageSummary struct {
	Coverage      float64       `json:"coverage"`
	TotalLines    int           `json:"total_lines"`
	CoveredLines  int           `json:"covered_lines"`
	QualityStatus QualityStatus `json:"quality_status"`
	TopPackages   []PackageInfo `json:"top_packages"`
	LowPackages   []PackageInfo `json:"low_packages"`
}

// PackageInfo provides basic package information
type PackageInfo struct {
	Name     string  `json:"name"`
	Coverage float64 `json:"coverage"`
	Lines    int     `json:"lines"`
}

// ChartData represents data for chart visualization
type ChartData struct {
	Type   string       `json:"type"` // line, bar, pie, etc.
	Title  string       `json:"title"`
	Labels []string     `json:"labels"`
	Series []DataSeries `json:"series"`
}

// DataSeries represents a data series for charts
type DataSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"`
}

// BuildStatus represents the current build status for live updates
type BuildStatus struct {
	State        string    `json:"state"`           // "queued", "in_progress", "completed"
	Conclusion   string    `json:"conclusion"`      // "success", "failure", "canceled", "skipped", "timed_out", "action_required", "neutral"
	WorkflowName string    `json:"workflow_name"`   // Name of the workflow
	RunID        int64     `json:"run_id"`          // GitHub Actions run ID
	RunNumber    int       `json:"run_number"`      // Run number for display
	RunURL       string    `json:"run_url"`         // URL to the GitHub Actions run
	StartedAt    time.Time `json:"started_at"`      // When the run started
	UpdatedAt    time.Time `json:"updated_at"`      // Last update time
	Duration     string    `json:"duration"`        // Human-readable duration
	HeadSHA      string    `json:"head_sha"`        // Commit SHA being built
	HeadBranch   string    `json:"head_branch"`     // Branch being built
	Event        string    `json:"event"`           // Event that triggered the run (push, pull_request, etc.)
	DisplayTitle string    `json:"display_title"`   // Display title for the run
	Available    bool      `json:"available"`       // Whether build status is available
	Error        string    `json:"error,omitempty"` // Error message if status fetch failed
}
