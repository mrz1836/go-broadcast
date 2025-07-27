// Package analysis provides coverage comparison and analysis capabilities for PR integration
package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Coverage change direction constants
const (
	DirectionImproved = "improved"
	DirectionDegraded = "degraded"
	DirectionStable   = "stable"
)

// ComparisonEngine handles coverage comparison between base and PR branches
type ComparisonEngine struct {
	config *ComparisonConfig
}

// ComparisonConfig holds configuration for coverage comparison
type ComparisonConfig struct {
	// Significance thresholds
	SignificantPercentageChange float64 // Threshold for significant percentage change
	SignificantLineChange       int     // Threshold for significant line count change

	// File analysis settings
	AnalyzeFileChanges bool // Whether to analyze individual file changes
	MaxFilesToAnalyze  int  // Maximum number of files to analyze in detail
	IgnoreTestFiles    bool // Whether to ignore test files in analysis

	// Trend analysis settings
	EnableTrendAnalysis bool // Whether to perform trend analysis
	TrendHistoryDays    int  // Number of days of history to consider for trends

	// Quality thresholds
	ExcellentCoverageThreshold  float64 // Threshold for excellent coverage
	GoodCoverageThreshold       float64 // Threshold for good coverage
	AcceptableCoverageThreshold float64 // Threshold for acceptable coverage
}

// CoverageSnapshot represents a coverage snapshot for comparison
type CoverageSnapshot struct {
	Branch          string                    `json:"branch"`
	CommitSHA       string                    `json:"commit_sha"`
	Timestamp       time.Time                 `json:"timestamp"`
	OverallCoverage CoverageMetrics           `json:"overall_coverage"`
	FileCoverage    map[string]FileMetrics    `json:"file_coverage"`
	PackageCoverage map[string]PackageMetrics `json:"package_coverage"`
	TestMetadata    TestMetadata              `json:"test_metadata"`
}

// CoverageMetrics represents overall coverage metrics
type CoverageMetrics struct {
	Percentage        float64 `json:"percentage"`
	TotalStatements   int     `json:"total_statements"`
	CoveredStatements int     `json:"covered_statements"`
	TotalLines        int     `json:"total_lines"`
	CoveredLines      int     `json:"covered_lines"`
	TotalFunctions    int     `json:"total_functions"`
	CoveredFunctions  int     `json:"covered_functions"`
}

// FileMetrics represents coverage metrics for a single file
type FileMetrics struct {
	Filename          string   `json:"filename"`
	Package           string   `json:"package"`
	Percentage        float64  `json:"percentage"`
	TotalStatements   int      `json:"total_statements"`
	CoveredStatements int      `json:"covered_statements"`
	UncoveredLines    []int    `json:"uncovered_lines"`
	Functions         []string `json:"functions"`
	IsTestFile        bool     `json:"is_test_file"`
	LinesAdded        int      `json:"lines_added"`   // For PR analysis
	LinesRemoved      int      `json:"lines_removed"` // For PR analysis
	IsNewFile         bool     `json:"is_new_file"`   // For PR analysis
	IsModified        bool     `json:"is_modified"`   // For PR analysis
}

// PackageMetrics represents coverage metrics for a package
type PackageMetrics struct {
	Package           string  `json:"package"`
	Percentage        float64 `json:"percentage"`
	TotalStatements   int     `json:"total_statements"`
	CoveredStatements int     `json:"covered_statements"`
	FileCount         int     `json:"file_count"`
}

// TestMetadata represents metadata about test execution
type TestMetadata struct {
	TestDuration   time.Duration `json:"test_duration"`
	TestCount      int           `json:"test_count"`
	FailedTests    int           `json:"failed_tests"`
	SkippedTests   int           `json:"skipped_tests"`
	BenchmarkCount int           `json:"benchmark_count"`
}

// ComparisonResult represents the result of comparing two coverage snapshots
type ComparisonResult struct {
	BaseSnapshot      CoverageSnapshot        `json:"base_snapshot"`
	PRSnapshot        CoverageSnapshot        `json:"pr_snapshot"`
	OverallChange     OverallChangeAnalysis   `json:"overall_change"`
	FileChanges       []FileChangeAnalysis    `json:"file_changes"`
	PackageChanges    []PackageChangeAnalysis `json:"package_changes"`
	TrendAnalysis     TrendAnalysis           `json:"trend_analysis"`
	QualityAssessment QualityAssessment       `json:"quality_assessment"`
	Recommendations   []Recommendation        `json:"recommendations"`
	Summary           ComparisonSummary       `json:"summary"`
}

// OverallChangeAnalysis represents analysis of overall coverage change
type OverallChangeAnalysis struct {
	PercentageChange       float64 `json:"percentage_change"`
	StatementChange        int     `json:"statement_change"`
	CoveredStatementChange int     `json:"covered_statement_change"`
	Direction              string  `json:"direction"` // DirectionImproved, DirectionDegraded, DirectionStable
	Magnitude              string  `json:"magnitude"` // "significant", "moderate", "minor", "negligible"
	IsSignificant          bool    `json:"is_significant"`
}

// FileChangeAnalysis represents analysis of file-level coverage changes
type FileChangeAnalysis struct {
	Filename               string  `json:"filename"`
	BasePercentage         float64 `json:"base_percentage"`
	PRPercentage           float64 `json:"pr_percentage"`
	PercentageChange       float64 `json:"percentage_change"`
	StatementChange        int     `json:"statement_change"`
	CoveredStatementChange int     `json:"covered_statement_change"`
	Direction              string  `json:"direction"`
	Magnitude              string  `json:"magnitude"`
	IsSignificant          bool    `json:"is_significant"`
	IsNewFile              bool    `json:"is_new_file"`
	IsDeleted              bool    `json:"is_deleted"`
	LinesAdded             int     `json:"lines_added"`
	LinesRemoved           int     `json:"lines_removed"`
	Risk                   string  `json:"risk"` // "high", "medium", "low"
}

// PackageChangeAnalysis represents analysis of package-level coverage changes
type PackageChangeAnalysis struct {
	Package          string  `json:"package"`
	BasePercentage   float64 `json:"base_percentage"`
	PRPercentage     float64 `json:"pr_percentage"`
	PercentageChange float64 `json:"percentage_change"`
	FileCount        int     `json:"file_count"`
	Direction        string  `json:"direction"`
	IsSignificant    bool    `json:"is_significant"`
}

// TrendAnalysis represents trend analysis based on historical data
type TrendAnalysis struct {
	Direction         string            `json:"direction"`  // "upward", "downward", DirectionStable, "volatile"
	Momentum          string            `json:"momentum"`   // "accelerating", "steady", "decelerating"
	Volatility        float64           `json:"volatility"` // Standard deviation of recent changes
	Prediction        Prediction        `json:"prediction"` // Predicted future trend
	HistoricalContext HistoricalContext `json:"historical_context"`
}

// Prediction represents predicted coverage trends
type Prediction struct {
	NextCoverage    float64 `json:"next_coverage"`
	Confidence      float64 `json:"confidence"`
	TimeHorizon     string  `json:"time_horizon"`
	PredictionModel string  `json:"prediction_model"`
}

// HistoricalContext provides context based on historical data
type HistoricalContext struct {
	AverageCoverage float64   `json:"average_coverage"`
	BestCoverage    float64   `json:"best_coverage"`
	WorstCoverage   float64   `json:"worst_coverage"`
	TrendStartDate  time.Time `json:"trend_start_date"`
	DataPoints      int       `json:"data_points"`
}

// QualityAssessment represents overall quality assessment
type QualityAssessment struct {
	OverallGrade  string   `json:"overall_grade"` // "A+", "A", "B+", "B", "C", "D", "F"
	CoverageGrade string   `json:"coverage_grade"`
	TrendGrade    string   `json:"trend_grade"`
	RiskLevel     string   `json:"risk_level"`    // "low", "medium", "high", "critical"
	QualityScore  float64  `json:"quality_score"` // 0-100
	Strengths     []string `json:"strengths"`
	Weaknesses    []string `json:"weaknesses"`
}

// Recommendation represents actionable recommendations
type Recommendation struct {
	Type            string   `json:"type"`     // "coverage", "testing", "refactoring", "process"
	Priority        string   `json:"priority"` // "high", "medium", "low"
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	ActionItems     []string `json:"action_items"`
	EstimatedEffort string   `json:"estimated_effort"` // "low", "medium", "high"
	ExpectedImpact  string   `json:"expected_impact"`  // "high", "medium", "low"
}

// ComparisonSummary provides a high-level summary
type ComparisonSummary struct {
	OverallImpact  string   `json:"overall_impact"` // "positive", "negative", "neutral"
	KeyChanges     []string `json:"key_changes"`
	CriticalIssues []string `json:"critical_issues"`
	Highlights     []string `json:"highlights"`
	NextSteps      []string `json:"next_steps"`
}

// NewComparisonEngine creates a new comparison engine with configuration
func NewComparisonEngine(config *ComparisonConfig) *ComparisonEngine {
	if config == nil {
		config = &ComparisonConfig{
			SignificantPercentageChange: 1.0,
			SignificantLineChange:       10,
			AnalyzeFileChanges:          true,
			MaxFilesToAnalyze:           50,
			IgnoreTestFiles:             false,
			EnableTrendAnalysis:         true,
			TrendHistoryDays:            30,
			ExcellentCoverageThreshold:  90.0,
			GoodCoverageThreshold:       80.0,
			AcceptableCoverageThreshold: 70.0,
		}
	}

	return &ComparisonEngine{
		config: config,
	}
}

// CompareCoverage compares coverage between base and PR snapshots
func (e *ComparisonEngine) CompareCoverage(_ context.Context, baseSnapshot, prSnapshot *CoverageSnapshot) (*ComparisonResult, error) {
	result := &ComparisonResult{
		BaseSnapshot: *baseSnapshot,
		PRSnapshot:   *prSnapshot,
	}

	// Analyze overall changes
	result.OverallChange = e.analyzeOverallChange(baseSnapshot, prSnapshot)

	// Analyze file-level changes
	if e.config.AnalyzeFileChanges {
		result.FileChanges = e.analyzeFileChanges(baseSnapshot, prSnapshot)
	}

	// Analyze package-level changes
	result.PackageChanges = e.analyzePackageChanges(baseSnapshot, prSnapshot)

	// Perform trend analysis if enabled
	if e.config.EnableTrendAnalysis {
		result.TrendAnalysis = e.analyzeTrends(baseSnapshot, prSnapshot)
	}

	// Generate quality assessment
	result.QualityAssessment = e.generateQualityAssessment(prSnapshot, &result.OverallChange, result.FileChanges)

	// Generate recommendations
	result.Recommendations = e.generateRecommendations(result)

	// Generate summary
	result.Summary = e.generateSummary(result)

	return result, nil
}

// analyzeOverallChange analyzes overall coverage changes
func (e *ComparisonEngine) analyzeOverallChange(base, pr *CoverageSnapshot) OverallChangeAnalysis {
	percentageChange := pr.OverallCoverage.Percentage - base.OverallCoverage.Percentage
	statementChange := pr.OverallCoverage.TotalStatements - base.OverallCoverage.TotalStatements
	coveredChange := pr.OverallCoverage.CoveredStatements - base.OverallCoverage.CoveredStatements

	direction := DirectionStable
	if percentageChange > 0.1 {
		direction = DirectionImproved
	} else if percentageChange < -0.1 {
		direction = DirectionDegraded
	}

	magnitude := e.calculateMagnitude(math.Abs(percentageChange))
	isSignificant := math.Abs(percentageChange) >= e.config.SignificantPercentageChange

	return OverallChangeAnalysis{
		PercentageChange:       percentageChange,
		StatementChange:        statementChange,
		CoveredStatementChange: coveredChange,
		Direction:              direction,
		Magnitude:              magnitude,
		IsSignificant:          isSignificant,
	}
}

// analyzeFileChanges analyzes coverage changes at the file level
func (e *ComparisonEngine) analyzeFileChanges(base, pr *CoverageSnapshot) []FileChangeAnalysis {
	changes := make([]FileChangeAnalysis, 0, len(base.FileCoverage)+len(pr.FileCoverage))

	// Create maps for efficient lookup
	baseFiles := make(map[string]FileMetrics)
	for filename, metrics := range base.FileCoverage {
		baseFiles[filename] = metrics
	}

	prFiles := make(map[string]FileMetrics)
	for filename, metrics := range pr.FileCoverage {
		prFiles[filename] = metrics
	}

	// Analyze all files present in either snapshot
	allFiles := make(map[string]bool)
	for filename := range baseFiles {
		allFiles[filename] = true
	}
	for filename := range prFiles {
		allFiles[filename] = true
	}

	for filename := range allFiles {
		if e.config.IgnoreTestFiles && e.isTestFile(filename) {
			continue
		}

		baseMetrics, existsInBase := baseFiles[filename]
		prMetrics, existsInPR := prFiles[filename]

		change := FileChangeAnalysis{
			Filename:  filename,
			IsNewFile: !existsInBase && existsInPR,
			IsDeleted: existsInBase && !existsInPR,
		}

		if change.IsNewFile {
			change.PRPercentage = prMetrics.Percentage
			change.PercentageChange = prMetrics.Percentage
			change.Direction = "new"
			change.StatementChange = prMetrics.TotalStatements
			change.CoveredStatementChange = prMetrics.CoveredStatements
			change.LinesAdded = prMetrics.LinesAdded
		} else if change.IsDeleted {
			change.BasePercentage = baseMetrics.Percentage
			change.PercentageChange = -baseMetrics.Percentage
			change.Direction = "deleted"
			change.StatementChange = -baseMetrics.TotalStatements
			change.CoveredStatementChange = -baseMetrics.CoveredStatements
			change.LinesRemoved = baseMetrics.TotalStatements // Approximation
		} else {
			// File exists in both snapshots
			change.BasePercentage = baseMetrics.Percentage
			change.PRPercentage = prMetrics.Percentage
			change.PercentageChange = prMetrics.Percentage - baseMetrics.Percentage
			change.StatementChange = prMetrics.TotalStatements - baseMetrics.TotalStatements
			change.CoveredStatementChange = prMetrics.CoveredStatements - baseMetrics.CoveredStatements
			change.LinesAdded = prMetrics.LinesAdded
			change.LinesRemoved = prMetrics.LinesRemoved

			if change.PercentageChange > 0.1 {
				change.Direction = DirectionImproved
			} else if change.PercentageChange < -0.1 {
				change.Direction = DirectionDegraded
			} else {
				change.Direction = DirectionStable
			}
		}

		change.Magnitude = e.calculateMagnitude(math.Abs(change.PercentageChange))
		change.IsSignificant = math.Abs(change.PercentageChange) >= e.config.SignificantPercentageChange ||
			math.Abs(float64(change.StatementChange)) >= float64(e.config.SignificantLineChange)

		change.Risk = e.calculateRisk(change)

		changes = append(changes, change)
	}

	// Sort by significance and percentage change
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].IsSignificant != changes[j].IsSignificant {
			return changes[i].IsSignificant
		}
		return math.Abs(changes[i].PercentageChange) > math.Abs(changes[j].PercentageChange)
	})

	// Limit the number of changes to analyze
	if len(changes) > e.config.MaxFilesToAnalyze {
		changes = changes[:e.config.MaxFilesToAnalyze]
	}

	return changes
}

// analyzePackageChanges analyzes coverage changes at the package level
func (e *ComparisonEngine) analyzePackageChanges(base, pr *CoverageSnapshot) []PackageChangeAnalysis {
	changes := make([]PackageChangeAnalysis, 0, len(base.PackageCoverage)+len(pr.PackageCoverage))

	// Create maps for efficient lookup
	basePackages := make(map[string]PackageMetrics)
	for packageName, metrics := range base.PackageCoverage {
		basePackages[packageName] = metrics
	}

	prPackages := make(map[string]PackageMetrics)
	for packageName, metrics := range pr.PackageCoverage {
		prPackages[packageName] = metrics
	}

	// Analyze all packages
	allPackages := make(map[string]bool)
	for packageName := range basePackages {
		allPackages[packageName] = true
	}
	for packageName := range prPackages {
		allPackages[packageName] = true
	}

	for packageName := range allPackages {
		baseMetrics, existsInBase := basePackages[packageName]
		prMetrics, existsInPR := prPackages[packageName]

		if !existsInBase || !existsInPR {
			continue // Skip packages that don't exist in both snapshots
		}

		percentageChange := prMetrics.Percentage - baseMetrics.Percentage

		direction := DirectionStable
		if percentageChange > 0.1 {
			direction = DirectionImproved
		} else if percentageChange < -0.1 {
			direction = DirectionDegraded
		}

		isSignificant := math.Abs(percentageChange) >= e.config.SignificantPercentageChange

		change := PackageChangeAnalysis{
			Package:          packageName,
			BasePercentage:   baseMetrics.Percentage,
			PRPercentage:     prMetrics.Percentage,
			PercentageChange: percentageChange,
			FileCount:        prMetrics.FileCount,
			Direction:        direction,
			IsSignificant:    isSignificant,
		}

		changes = append(changes, change)
	}

	// Sort by significance and percentage change
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].IsSignificant != changes[j].IsSignificant {
			return changes[i].IsSignificant
		}
		return math.Abs(changes[i].PercentageChange) > math.Abs(changes[j].PercentageChange)
	})

	return changes
}

// analyzeTrends analyzes coverage trends (placeholder for advanced trend analysis)
func (e *ComparisonEngine) analyzeTrends(base, pr *CoverageSnapshot) TrendAnalysis {
	// This is a simplified implementation
	// In a full implementation, this would analyze historical data

	percentageChange := pr.OverallCoverage.Percentage - base.OverallCoverage.Percentage

	direction := DirectionStable
	momentum := "steady"

	if percentageChange > 1.0 {
		direction = "upward"
		momentum = "accelerating"
	} else if percentageChange < -1.0 {
		direction = "downward"
		momentum = "accelerating"
	}

	volatility := math.Abs(percentageChange) // Simplified volatility calculation

	prediction := Prediction{
		NextCoverage:    pr.OverallCoverage.Percentage + (percentageChange * 0.5),
		Confidence:      0.7,
		TimeHorizon:     "1 week",
		PredictionModel: "linear",
	}

	historicalContext := HistoricalContext{
		AverageCoverage: (base.OverallCoverage.Percentage + pr.OverallCoverage.Percentage) / 2,
		BestCoverage:    math.Max(base.OverallCoverage.Percentage, pr.OverallCoverage.Percentage),
		WorstCoverage:   math.Min(base.OverallCoverage.Percentage, pr.OverallCoverage.Percentage),
		TrendStartDate:  base.Timestamp,
		DataPoints:      2,
	}

	return TrendAnalysis{
		Direction:         direction,
		Momentum:          momentum,
		Volatility:        volatility,
		Prediction:        prediction,
		HistoricalContext: historicalContext,
	}
}

// generateQualityAssessment generates a quality assessment
func (e *ComparisonEngine) generateQualityAssessment(pr *CoverageSnapshot, overallChange *OverallChangeAnalysis, fileChanges []FileChangeAnalysis) QualityAssessment {
	coverage := pr.OverallCoverage.Percentage

	// Calculate coverage grade
	coverageGrade := e.calculateCoverageGrade(coverage)

	// Calculate trend grade
	trendGrade := e.calculateTrendGrade(overallChange)

	// Calculate overall grade (weighted average)
	overallGrade := e.calculateOverallGrade(coverageGrade, trendGrade)

	// Calculate risk level
	riskLevel := e.calculateRiskLevel(coverage, overallChange, fileChanges)

	// Calculate quality score (0-100)
	qualityScore := e.calculateQualityScore(coverage, overallChange)

	// Identify strengths and weaknesses
	strengths, weaknesses := e.identifyStrengthsAndWeaknesses(coverage, overallChange, fileChanges)

	return QualityAssessment{
		OverallGrade:  overallGrade,
		CoverageGrade: coverageGrade,
		TrendGrade:    trendGrade,
		RiskLevel:     riskLevel,
		QualityScore:  qualityScore,
		Strengths:     strengths,
		Weaknesses:    weaknesses,
	}
}

// generateRecommendations generates actionable recommendations
func (e *ComparisonEngine) generateRecommendations(result *ComparisonResult) []Recommendation {
	var recommendations []Recommendation

	coverage := result.PRSnapshot.OverallCoverage.Percentage

	// Coverage recommendations
	if coverage < e.config.AcceptableCoverageThreshold {
		recommendations = append(recommendations, Recommendation{
			Type:        "coverage",
			Priority:    "high",
			Title:       "Improve Overall Coverage",
			Description: fmt.Sprintf("Current coverage %.1f%% is below acceptable threshold of %.1f%%", coverage, e.config.AcceptableCoverageThreshold),
			ActionItems: []string{
				"Add unit tests for uncovered functions",
				"Focus on files with lowest coverage first",
				"Consider integration tests for complex scenarios",
			},
			EstimatedEffort: "high",
			ExpectedImpact:  "high",
		})
	}

	// File-specific recommendations
	highRiskFiles := 0
	for _, fileChange := range result.FileChanges {
		if fileChange.Risk == "high" {
			highRiskFiles++
		}
	}

	if highRiskFiles > 0 {
		recommendations = append(recommendations, Recommendation{
			Type:        "testing",
			Priority:    "medium",
			Title:       "Address High-Risk Files",
			Description: fmt.Sprintf("%d files have high risk due to low coverage or significant changes", highRiskFiles),
			ActionItems: []string{
				"Review files with coverage below 50%",
				"Add tests for recently modified files",
				"Consider refactoring complex files",
			},
			EstimatedEffort: "medium",
			ExpectedImpact:  "medium",
		})
	}

	// Trend recommendations
	if result.OverallChange.Direction == DirectionDegraded && result.OverallChange.IsSignificant {
		recommendations = append(recommendations, Recommendation{
			Type:        "process",
			Priority:    "high",
			Title:       "Prevent Coverage Regression",
			Description: "Coverage has significantly decreased in this PR",
			ActionItems: []string{
				"Implement coverage threshold checks in CI",
				"Require coverage review for significant changes",
				"Add coverage gates to prevent merging low-coverage PRs",
			},
			EstimatedEffort: "low",
			ExpectedImpact:  "high",
		})
	}

	return recommendations
}

// generateSummary generates a high-level summary
func (e *ComparisonEngine) generateSummary(result *ComparisonResult) ComparisonSummary {
	var keyChanges []string
	var criticalIssues []string
	var highlights []string
	var nextSteps []string

	// Overall impact
	overallImpact := "neutral"
	if result.OverallChange.IsSignificant {
		switch result.OverallChange.Direction {
		case DirectionImproved:
			overallImpact = "positive"
		case DirectionDegraded:
			overallImpact = "negative"
		}
	}

	// Key changes
	if result.OverallChange.IsSignificant {
		keyChanges = append(keyChanges, fmt.Sprintf("Overall coverage %s by %.1f%%",
			result.OverallChange.Direction, math.Abs(result.OverallChange.PercentageChange)))
	}

	significantFileChanges := 0
	for _, fileChange := range result.FileChanges {
		if fileChange.IsSignificant {
			significantFileChanges++
		}
	}

	if significantFileChanges > 0 {
		keyChanges = append(keyChanges, fmt.Sprintf("%d files with significant coverage changes", significantFileChanges))
	}

	// Critical issues
	coverage := result.PRSnapshot.OverallCoverage.Percentage
	if coverage < e.config.AcceptableCoverageThreshold {
		criticalIssues = append(criticalIssues, fmt.Sprintf("Coverage %.1f%% below acceptable threshold", coverage))
	}

	if result.OverallChange.Direction == DirectionDegraded && result.OverallChange.IsSignificant {
		criticalIssues = append(criticalIssues, "Significant coverage regression detected")
	}

	// Highlights
	if coverage >= e.config.ExcellentCoverageThreshold {
		highlights = append(highlights, "Excellent overall coverage maintained")
	}

	if result.OverallChange.Direction == DirectionImproved && result.OverallChange.IsSignificant {
		highlights = append(highlights, "Significant coverage improvement achieved")
	}

	// Next steps
	for _, rec := range result.Recommendations {
		if rec.Priority == "high" {
			nextSteps = append(nextSteps, rec.Title)
		}
	}

	return ComparisonSummary{
		OverallImpact:  overallImpact,
		KeyChanges:     keyChanges,
		CriticalIssues: criticalIssues,
		Highlights:     highlights,
		NextSteps:      nextSteps,
	}
}

// LoadCoverageSnapshot loads a coverage snapshot from a file
func (e *ComparisonEngine) LoadCoverageSnapshot(_ context.Context, filePath string) (*CoverageSnapshot, error) {
	file, err := os.Open(filePath) //nolint:gosec // file path is controlled and validated by caller
	if err != nil {
		return nil, fmt.Errorf("failed to open coverage file: %w", err)
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read coverage file: %w", err)
	}

	var snapshot CoverageSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse coverage snapshot: %w", err)
	}

	return &snapshot, nil
}

// SaveComparisonResult saves a comparison result to a file
func (e *ComparisonEngine) SaveComparisonResult(_ context.Context, result *ComparisonResult, filePath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal comparison result: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write comparison result: %w", err)
	}

	return nil
}

// Helper methods

func (e *ComparisonEngine) calculateMagnitude(change float64) string {
	if change >= 5.0 {
		return "significant"
	} else if change >= 2.0 {
		return "moderate"
	} else if change >= 0.5 {
		return "minor"
	}
	return "negligible"
}

func (e *ComparisonEngine) calculateRisk(change FileChangeAnalysis) string {
	if change.PRPercentage < 50 && change.Direction == DirectionDegraded {
		return "high"
	} else if change.PRPercentage < 70 || math.Abs(change.PercentageChange) > 10 {
		return "medium"
	}
	return "low"
}

func (e *ComparisonEngine) calculateCoverageGrade(coverage float64) string {
	if coverage >= 95 {
		return "A+"
	} else if coverage >= 90 {
		return "A"
	} else if coverage >= 85 {
		return "B+"
	} else if coverage >= 80 {
		return "B"
	} else if coverage >= 70 {
		return "C"
	} else if coverage >= 60 {
		return "D"
	}
	return "F"
}

func (e *ComparisonEngine) calculateTrendGrade(change *OverallChangeAnalysis) string {
	if change.Direction == DirectionImproved && change.IsSignificant {
		return "A"
	} else if change.Direction == DirectionImproved {
		return "B"
	} else if change.Direction == DirectionStable {
		return "B"
	} else if change.Direction == DirectionDegraded && !change.IsSignificant {
		return "C"
	}
	return "D"
}

func (e *ComparisonEngine) calculateOverallGrade(coverageGrade, _ string) string {
	// Simplified overall grade calculation
	// In practice, this would use a more sophisticated weighting system
	return coverageGrade
}

func (e *ComparisonEngine) calculateRiskLevel(coverage float64, change *OverallChangeAnalysis, fileChanges []FileChangeAnalysis) string {
	riskScore := 0

	if coverage < 60 {
		riskScore += 3
	} else if coverage < 70 {
		riskScore += 2
	} else if coverage < 80 {
		riskScore++
	}

	if change.Direction == DirectionDegraded && change.IsSignificant {
		riskScore += 2
	}

	highRiskFiles := 0
	for _, fileChange := range fileChanges {
		if fileChange.Risk == "high" {
			highRiskFiles++
		}
	}

	if highRiskFiles > 5 {
		riskScore += 2
	} else if highRiskFiles > 2 {
		riskScore++
	}

	if riskScore >= 5 {
		return "critical"
	} else if riskScore >= 3 {
		return "high"
	} else if riskScore >= 1 {
		return "medium"
	}
	return "low"
}

func (e *ComparisonEngine) calculateQualityScore(coverage float64, change *OverallChangeAnalysis) float64 {
	score := coverage // Start with coverage percentage

	// Adjust based on trend
	if change.Direction == DirectionImproved && change.IsSignificant {
		score += 5
	} else if change.Direction == DirectionDegraded && change.IsSignificant {
		score -= 10
	}

	// Cap at 100
	if score > 100 {
		score = 100
	} else if score < 0 {
		score = 0
	}

	return score
}

func (e *ComparisonEngine) identifyStrengthsAndWeaknesses(coverage float64, change *OverallChangeAnalysis, fileChanges []FileChangeAnalysis) ([]string, []string) {
	var strengths []string
	var weaknesses []string

	// Strengths
	if coverage >= e.config.ExcellentCoverageThreshold {
		strengths = append(strengths, "Excellent overall coverage")
	}

	if change.Direction == DirectionImproved {
		strengths = append(strengths, "Coverage trend is improving")
	}

	goodFiles := 0
	for _, fileChange := range fileChanges {
		if fileChange.PRPercentage >= 80 {
			goodFiles++
		}
	}

	if goodFiles > len(fileChanges)/2 {
		strengths = append(strengths, "Majority of files have good coverage")
	}

	// Weaknesses
	if coverage < e.config.AcceptableCoverageThreshold {
		weaknesses = append(weaknesses, "Overall coverage below acceptable threshold")
	}

	if change.Direction == DirectionDegraded && change.IsSignificant {
		weaknesses = append(weaknesses, "Significant coverage regression")
	}

	poorFiles := 0
	for _, fileChange := range fileChanges {
		if fileChange.PRPercentage < 50 {
			poorFiles++
		}
	}

	if poorFiles > 0 {
		weaknesses = append(weaknesses, fmt.Sprintf("%d files with poor coverage", poorFiles))
	}

	return strengths, weaknesses
}

func (e *ComparisonEngine) isTestFile(filename string) bool {
	return strings.HasSuffix(filename, "_test.go") ||
		strings.Contains(filename, "/test/") ||
		strings.Contains(filename, "/tests/")
}
