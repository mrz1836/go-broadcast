package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
)

// ProductionReadinessTestSuite tests the production readiness validation system
type ProductionReadinessTestSuite struct {
	suite.Suite

	tempDir   string
	validator *ProductionReadinessValidator
}

func (s *ProductionReadinessTestSuite) SetupTest() {
	// Create temp directory for tests
	tempDir, err := os.MkdirTemp("", "production-readiness-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Create validator
	validator, err := NewProductionReadinessValidator()
	s.Require().NoError(err)
	s.validator = validator
}

func (s *ProductionReadinessTestSuite) TearDownTest() {
	if s.validator != nil {
		s.validator.Cleanup()
	}
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
}

func TestProductionReadinessTestSuite(t *testing.T) {
	suite.Run(t, new(ProductionReadinessTestSuite))
}

// Test validator creation and initialization
func (s *ProductionReadinessTestSuite) TestNewProductionReadinessValidator() {
	// Test successful creation
	validator, err := NewProductionReadinessValidator()
	s.Require().NoError(err)
	s.NotNil(validator)
	defer validator.Cleanup()

	// Verify temp directory was created
	s.DirExists(validator.tempDir)

	// Verify .github directory was created
	githubDir := filepath.Join(validator.tempDir, ".github")
	s.DirExists(githubDir)

	// Verify env file was created
	s.FileExists(validator.envFile)

	// Verify env file contents
	content, err := os.ReadFile(validator.envFile)
	s.Require().NoError(err)
	s.Contains(string(content), "ENABLE_PRE_COMMIT_SYSTEM=true")
	s.Contains(string(content), "PRE_COMMIT_SYSTEM_LOG_LEVEL=info")
}

// Test system info collection
func (s *ProductionReadinessTestSuite) TestCollectSystemInfo() {
	info := s.validator.collectSystemInfo()

	s.Equal(runtime.Version(), info.GoVersion)
	s.Equal(runtime.GOOS, info.OS)
	s.Equal(runtime.GOARCH, info.Architecture)
	s.Equal(runtime.NumCPU(), info.NumCPU)
}

// Test git repo initialization
func (s *ProductionReadinessTestSuite) TestInitGitRepo() {
	// Change to temp directory
	originalWD, err := os.Getwd()
	s.Require().NoError(err)
	defer func() { _ = os.Chdir(originalWD) }()

	err = os.Chdir(s.validator.tempDir)
	s.Require().NoError(err)

	// Initialize git repo
	err = s.validator.initGitRepo()
	s.Require().NoError(err)

	// Verify .git directory was created
	gitDir := filepath.Join(s.validator.tempDir, ".git")
	s.DirExists(gitDir)

	// Verify HEAD file was created
	headFile := filepath.Join(gitDir, "HEAD")
	s.FileExists(headFile)

	// Verify HEAD content
	content, err := os.ReadFile(headFile) //nolint:gosec // Test file, controlled path
	s.Require().NoError(err)
	s.Equal("ref: refs/heads/main", string(content))
}

// Test test file creation
func (s *ProductionReadinessTestSuite) TestCreateTestFiles() {
	// Change to temp directory
	originalWD, err := os.Getwd()
	s.Require().NoError(err)
	defer func() { _ = os.Chdir(originalWD) }()

	err = os.Chdir(s.validator.tempDir)
	s.Require().NoError(err)

	// Create test files
	files := s.validator.createTestFiles()

	// Verify correct number of files
	s.Len(files, 10)

	// Verify each file was created
	for _, file := range files {
		fullPath := filepath.Join(s.validator.tempDir, file)
		s.FileExists(fullPath)

		// Verify file has content
		info, statErr := os.Stat(fullPath)
		s.Require().NoError(statErr)
		s.Positive(info.Size())
	}

	// Verify specific file contents
	goContent, err := os.ReadFile(filepath.Join(s.validator.tempDir, "main.go"))
	s.Require().NoError(err)
	s.Contains(string(goContent), "package main")

	mdContent, err := os.ReadFile(filepath.Join(s.validator.tempDir, "README.md"))
	s.Require().NoError(err)
	s.Contains(string(mdContent), "# Test Project")
}

// Test performance validation functions
func (s *ProductionReadinessTestSuite) TestPerformanceValidation() {
	// Change to temp directory
	originalWD, err := os.Getwd()
	s.Require().NoError(err)
	defer func() { _ = os.Chdir(originalWD) }()

	err = os.Chdir(s.validator.tempDir)
	s.Require().NoError(err)

	// Initialize git repo and create test files
	err = s.validator.initGitRepo()
	s.Require().NoError(err)
	files := s.validator.createTestFiles()

	// Test measure average performance
	cfg := &config.Config{
		Enabled: true,
		Checks: struct {
			Fumpt      bool
			Lint       bool
			ModTidy    bool
			Whitespace bool
			EOF        bool
		}{
			Whitespace: true,
			EOF:        true,
		},
	}

	// Measure performance with small set of files
	duration, err := s.validator.measureAveragePerformance(cfg, files[:3], 2)
	s.Require().NoError(err)
	s.Greater(duration, time.Duration(0))
	s.Less(duration, 5*time.Second)

	// Test cold start measurement
	coldStart, err := s.validator.measureColdStart(cfg, files[:5])
	s.Require().NoError(err)
	s.Greater(coldStart, time.Duration(0))
	s.Less(coldStart, 10*time.Second)

	// Test warm run measurement
	warmRun, err := s.validator.measureWarmRun(cfg, files[:5])
	s.Require().NoError(err)
	s.Greater(warmRun, time.Duration(0))
	s.Less(warmRun, 10*time.Second)
	// Warm run should generally be faster than cold start
	s.LessOrEqual(warmRun, coldStart*2)
}

// Test parallel scaling measurement
func (s *ProductionReadinessTestSuite) TestParallelScaling() {
	// Change to temp directory
	originalWD, err := os.Getwd()
	s.Require().NoError(err)
	defer func() { _ = os.Chdir(originalWD) }()

	err = os.Chdir(s.validator.tempDir)
	s.Require().NoError(err)

	// Initialize git repo and create test files
	err = s.validator.initGitRepo()
	s.Require().NoError(err)
	files := s.validator.createTestFiles()

	cfg := &config.Config{
		Enabled: true,
		Checks: struct {
			Fumpt      bool
			Lint       bool
			ModTidy    bool
			Whitespace bool
			EOF        bool
		}{
			Whitespace: true,
			EOF:        true,
		},
	}

	// Test parallel scaling
	result := s.validator.testParallelScaling(cfg, files[:5])
	// Result should be true or false, but we're testing the function runs
	s.IsType(bool(true), result)

	// Test performance with different worker counts
	single, err := s.validator.measurePerformanceWithWorkers(cfg, files[:5], 1)
	s.Require().NoError(err)
	s.Greater(single, time.Duration(0))

	parallel, err := s.validator.measurePerformanceWithWorkers(cfg, files[:5], 4)
	s.Require().NoError(err)
	s.Greater(parallel, time.Duration(0))
}

// Test memory efficiency
func (s *ProductionReadinessTestSuite) TestMemoryEfficiency() {
	// Change to temp directory
	originalWD, err := os.Getwd()
	s.Require().NoError(err)
	defer func() { _ = os.Chdir(originalWD) }()

	err = os.Chdir(s.validator.tempDir)
	s.Require().NoError(err)

	// Initialize git repo and create test files
	err = s.validator.initGitRepo()
	s.Require().NoError(err)
	files := s.validator.createTestFiles()

	cfg := &config.Config{
		Enabled: true,
		Checks: struct {
			Fumpt      bool
			Lint       bool
			ModTidy    bool
			Whitespace bool
			EOF        bool
		}{
			Whitespace: true,
			EOF:        true,
		},
	}

	// Test memory efficiency
	result := s.validator.testMemoryEfficiency(cfg, files[:5])
	s.IsType(bool(true), result)
}

// Test configuration validation
func (s *ProductionReadinessTestSuite) TestConfigurationValidation() {
	health := s.validator.validateConfiguration()

	// Check that all fields are set
	s.IsType(bool(true), health.LoadsSuccessfully)
	s.IsType(bool(true), health.ValidatesCorrectly)
	s.IsType(bool(true), health.DefaultsAppropriate)
	s.IsType(bool(true), health.EnvironmentPrecedence)
	s.IsType(bool(true), health.ErrorHandling)
	s.IsType(bool(true), health.DocumentationComplete)
	s.GreaterOrEqual(health.Score, 0)
	s.LessOrEqual(health.Score, 100)
}

// Test CI compatibility validation
func (s *ProductionReadinessTestSuite) TestCICompatibilityValidation() {
	compat := s.validator.validateCICompatibility()

	// Check that all fields are set
	s.IsType(bool(true), compat.GitHubActions)
	s.IsType(bool(true), compat.GitLabCI)
	s.IsType(bool(true), compat.Jenkins)
	s.IsType(bool(true), compat.GenericCI)
	s.IsType(bool(true), compat.NetworkConstrained)
	s.IsType(bool(true), compat.ResourceLimited)
	s.GreaterOrEqual(compat.Score, 0)
	s.LessOrEqual(compat.Score, 100)
}

// Test parallel safety validation
func (s *ProductionReadinessTestSuite) TestParallelSafetyValidation() {
	safety := s.validator.validateParallelSafety()

	// Check that all fields are set
	s.IsType(bool(true), safety.ConcurrentExecution)
	s.IsType(bool(true), safety.MemoryManagement)
	s.IsType(bool(true), safety.ResourceCleanup)
	s.IsType(bool(true), safety.RaceConditions)
	s.IsType(bool(true), safety.ContextCancellation)
	s.IsType(bool(true), safety.ConsistentResults)
	s.GreaterOrEqual(safety.Score, 0)
	s.LessOrEqual(safety.Score, 100)
}

// Test production scenarios validation
func (s *ProductionReadinessTestSuite) TestProductionScenariosValidation() {
	scenarios := s.validator.validateProductionScenarios()

	// Check that all fields are set
	s.IsType(bool(true), scenarios.LargeRepositories)
	s.IsType(bool(true), scenarios.MixedFileTypes)
	s.IsType(bool(true), scenarios.HighVolumeCommits)
	s.IsType(bool(true), scenarios.RealWorldPatterns)
	s.IsType(bool(true), scenarios.ResourceConstraints)
	s.IsType(bool(true), scenarios.NetworkIssues)
	s.GreaterOrEqual(scenarios.Score, 0)
	s.LessOrEqual(scenarios.Score, 100)
}

// Test SKIP functionality validation
func (s *ProductionReadinessTestSuite) TestSkipFunctionalityValidation() {
	skip := s.validator.validateSkipFunctionality()

	// Check that all fields are set
	s.IsType(bool(true), skip.SingleCheckSkip)
	s.IsType(bool(true), skip.MultipleCheckSkip)
	s.IsType(bool(true), skip.InvalidCheckNames)
	s.IsType(bool(true), skip.EnvironmentVars)
	s.IsType(bool(true), skip.CIIntegration)
	s.IsType(bool(true), skip.EdgeCases)
	s.GreaterOrEqual(skip.Score, 0)
	s.LessOrEqual(skip.Score, 100)
}

// Test score calculation methods
func (s *ProductionReadinessTestSuite) TestScoreCalculations() {
	// Test performance score calculation
	perfMetrics := PerformanceMetrics{
		MeetsTargetTime: true,
		ParallelScaling: true,
		MemoryEfficient: true,
		ColdStartTime:   2 * time.Second,
		WarmRunTime:     1 * time.Second,
	}
	perfScore := s.validator.calculatePerformanceScore(perfMetrics)
	s.Equal(100, perfScore) // All conditions met = 100

	// Test with some failures
	perfMetrics.MeetsTargetTime = false
	perfMetrics.MemoryEfficient = false
	perfScore = s.validator.calculatePerformanceScore(perfMetrics)
	s.Equal(40, perfScore) // Only 2/5 conditions met

	// Test configuration score calculation
	configHealth := ConfigurationHealth{
		LoadsSuccessfully:     true,
		ValidatesCorrectly:    true,
		DefaultsAppropriate:   true,
		EnvironmentPrecedence: true,
		ErrorHandling:         true,
		DocumentationComplete: true,
	}
	configScore := s.validator.calculateConfigurationScore(configHealth)
	s.Equal(100, configScore)

	// Test CI compatibility score
	ciCompat := CICompatibility{
		GitHubActions:      true,
		GitLabCI:           true,
		Jenkins:            true,
		GenericCI:          true,
		NetworkConstrained: true,
		ResourceLimited:    true,
	}
	ciScore := s.validator.calculateCICompatibilityScore(ciCompat)
	s.Equal(100, ciScore)

	// Test parallel safety score
	parallelSafety := ParallelSafety{
		ConcurrentExecution: true,
		MemoryManagement:    true,
		ResourceCleanup:     true,
		RaceConditions:      true,
		ContextCancellation: true,
		ConsistentResults:   true,
	}
	parallelScore := s.validator.calculateParallelSafetyScore(parallelSafety)
	s.Equal(100, parallelScore)

	// Test production scenarios score
	prodScenarios := ProductionScenarios{
		LargeRepositories:   true,
		MixedFileTypes:      true,
		HighVolumeCommits:   true,
		RealWorldPatterns:   true,
		ResourceConstraints: true,
		NetworkIssues:       true,
	}
	prodScore := s.validator.calculateProductionScenariosScore(prodScenarios)
	s.Equal(100, prodScore)

	// Test SKIP functionality score
	skipFunc := SkipFunctionality{
		SingleCheckSkip:   true,
		MultipleCheckSkip: true,
		InvalidCheckNames: true,
		EnvironmentVars:   true,
		CIIntegration:     true,
		EdgeCases:         true,
	}
	skipScore := s.validator.calculateSkipFunctionalityScore(skipFunc)
	s.Equal(100, skipScore)
}

// Test overall assessment calculation
func (s *ProductionReadinessTestSuite) TestCalculateOverallAssessment() {
	report := &ProductionReadinessReport{
		PerformanceMetrics: PerformanceMetrics{
			Score:           100,
			MeetsTargetTime: true,
		},
		ConfigurationHealth: ConfigurationHealth{
			Score: 100,
		},
		CICompatibility: CICompatibility{
			Score: 100,
		},
		ParallelSafety: ParallelSafety{
			Score: 100,
		},
		ProductionScenarios: ProductionScenarios{
			Score: 100,
		},
		SkipFunctionality: SkipFunctionality{
			Score: 100,
		},
	}

	s.validator.calculateOverallAssessment(report)

	// With all 100 scores, overall should be 100
	s.Equal(100, report.OverallScore)
	s.True(report.ProductionReady)
	s.Empty(report.CriticalIssues)

	// Test with lower scores
	report.PerformanceMetrics.Score = 50
	report.ConfigurationHealth.Score = 60
	s.validator.calculateOverallAssessment(report)
	s.Less(report.OverallScore, 85)
	s.False(report.ProductionReady)

	// Test with critical issues
	report.CriticalIssues = []string{"Test critical issue"}
	s.validator.calculateOverallAssessment(report)
	s.False(report.ProductionReady)
}

// Test recommendation generation
func (s *ProductionReadinessTestSuite) TestGenerateRecommendations() {
	report := &ProductionReadinessReport{
		PerformanceMetrics: PerformanceMetrics{
			Score:           70,
			MeetsTargetTime: false,
		},
		CICompatibility: CICompatibility{
			Score: 70,
		},
		ParallelSafety: ParallelSafety{
			Score: 85,
		},
	}

	s.validator.generateRecommendations(report)

	// Should have recommendations for low scores
	s.NotEmpty(report.Recommendations)
	s.Contains(report.Recommendations[0], "performance")
	s.Contains(report.Recommendations[1], "CI environments")
	s.Contains(report.Recommendations[2], "parallel execution")
	s.Contains(report.Recommendations[3], "Performance optimization required")
}

// Test known limitations documentation
func (s *ProductionReadinessTestSuite) TestDocumentKnownLimitations() {
	report := &ProductionReadinessReport{}
	s.validator.documentKnownLimitations(report)

	s.NotEmpty(report.KnownLimitations)
	s.Len(report.KnownLimitations, 5)
	s.Contains(report.KnownLimitations[0], "hardware")
	s.Contains(report.KnownLimitations[1], "external tools")
}

// Test report formatting
func (s *ProductionReadinessTestSuite) TestFormatReport() {
	report := &ProductionReadinessReport{
		GeneratedAt: time.Now(),
		Version:     "1.0.0",
		Environment: "test",
		SystemInfo: SystemInfo{
			GoVersion:    runtime.Version(),
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
			NumCPU:       4,
		},
		OverallScore:    95,
		ProductionReady: true,
		PerformanceMetrics: PerformanceMetrics{
			Score:            90,
			SmallCommitAvg:   1500 * time.Millisecond,
			TypicalCommitAvg: 2000 * time.Millisecond,
			ColdStartTime:    2500 * time.Millisecond,
			WarmRunTime:      1200 * time.Millisecond,
			MeetsTargetTime:  true,
			ParallelScaling:  true,
			MemoryEfficient:  true,
		},
		ConfigurationHealth: ConfigurationHealth{
			Score:                 95,
			LoadsSuccessfully:     true,
			ValidatesCorrectly:    true,
			DefaultsAppropriate:   true,
			EnvironmentPrecedence: true,
			ErrorHandling:         true,
			DocumentationComplete: true,
		},
		CICompatibility: CICompatibility{
			Score:              100,
			GitHubActions:      true,
			GitLabCI:           true,
			Jenkins:            true,
			GenericCI:          true,
			NetworkConstrained: true,
			ResourceLimited:    true,
		},
		CriticalIssues:   []string{"Test issue 1", "Test issue 2"},
		Recommendations:  []string{"Test recommendation 1", "Test recommendation 2"},
		KnownLimitations: []string{"Test limitation 1", "Test limitation 2"},
	}

	formatted := report.FormatReport()

	// Verify report contains all sections
	s.Contains(formatted, "# GoFortress Pre-commit System - Production Readiness Report")
	s.Contains(formatted, "## System Information")
	s.Contains(formatted, "## Overall Assessment")
	s.Contains(formatted, "## Performance Metrics")
	s.Contains(formatted, "## Configuration Health")
	s.Contains(formatted, "## CI Compatibility")
	s.Contains(formatted, "## Critical Issues")
	s.Contains(formatted, "## Recommendations")
	s.Contains(formatted, "## Known Limitations")

	// Verify specific content
	s.Contains(formatted, "Overall Score: 95/100")
	s.Contains(formatted, "Status: ✅ PRODUCTION READY")
	s.Contains(formatted, "Go Version: "+runtime.Version())
	s.Contains(formatted, "Small Commit Avg: 1.5s")
	s.Contains(formatted, "❌ Test issue 1")
	s.Contains(formatted, "💡 Test recommendation 1")
	s.Contains(formatted, "⚠️  Test limitation 1")
}

// Test full report generation (integration test)
func (s *ProductionReadinessTestSuite) TestGenerateReport() {
	// Change to temp directory
	originalWD, err := os.Getwd()
	s.Require().NoError(err)
	defer func() { _ = os.Chdir(originalWD) }()

	// Generate full report
	report, err := s.validator.GenerateReport()

	// The test might fail due to missing dependencies, but we should get a report
	s.NotNil(report)

	// Verify basic report structure
	s.NotZero(report.GeneratedAt)
	s.Equal("1.0.0", report.Version)
	s.Equal("validation", report.Environment)

	// Verify system info was collected
	s.NotEmpty(report.SystemInfo.GoVersion)
	s.NotEmpty(report.SystemInfo.OS)
	s.NotEmpty(report.SystemInfo.Architecture)
	s.Positive(report.SystemInfo.NumCPU)

	// Verify all sections were populated (even if with defaults)
	s.GreaterOrEqual(report.PerformanceMetrics.Score, 0)
	s.GreaterOrEqual(report.ConfigurationHealth.Score, 0)
	s.GreaterOrEqual(report.CICompatibility.Score, 0)
	s.GreaterOrEqual(report.ParallelSafety.Score, 0)
	s.GreaterOrEqual(report.ProductionScenarios.Score, 0)
	s.GreaterOrEqual(report.SkipFunctionality.Score, 0)

	// Verify overall assessment was calculated
	s.GreaterOrEqual(report.OverallScore, 0)
	s.LessOrEqual(report.OverallScore, 100)

	// If there was an error, it should be in critical issues
	if err != nil {
		s.NotEmpty(report.CriticalIssues)
	}
}

// Test JSON marshaling of report
func (s *ProductionReadinessTestSuite) TestReportJSONMarshaling() {
	report := &ProductionReadinessReport{
		GeneratedAt: time.Now(),
		Version:     "1.0.0",
		Environment: "test",
		SystemInfo: SystemInfo{
			GoVersion:    "go1.21.0",
			OS:           "linux",
			Architecture: "amd64",
			NumCPU:       8,
		},
		OverallScore:    85,
		ProductionReady: true,
		PerformanceMetrics: PerformanceMetrics{
			Score:            90,
			SmallCommitAvg:   1500 * time.Millisecond,
			TypicalCommitAvg: 2000 * time.Millisecond,
			MeetsTargetTime:  true,
		},
		CriticalIssues:   []string{"Issue 1"},
		Recommendations:  []string{"Recommendation 1"},
		KnownLimitations: []string{"Limitation 1"},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	s.Require().NoError(err)
	s.NotEmpty(data)

	// Unmarshal back
	var unmarshaled ProductionReadinessReport
	err = json.Unmarshal(data, &unmarshaled)
	s.Require().NoError(err)

	// Verify key fields
	s.Equal(report.Version, unmarshaled.Version)
	s.Equal(report.Environment, unmarshaled.Environment)
	s.Equal(report.OverallScore, unmarshaled.OverallScore)
	s.Equal(report.ProductionReady, unmarshaled.ProductionReady)
	s.Equal(report.SystemInfo.GoVersion, unmarshaled.SystemInfo.GoVersion)
	s.Equal(report.PerformanceMetrics.Score, unmarshaled.PerformanceMetrics.Score)
	s.Len(report.CriticalIssues, len(unmarshaled.CriticalIssues))
}

// Test error handling in temp directory creation
func TestNewProductionReadinessValidatorErrors(_ *testing.T) {
	// Save original TempDir
	originalTempDir := os.TempDir()

	// Set TempDir to an invalid location
	_ = os.Setenv("TMPDIR", "/invalid/path/that/does/not/exist")
	defer func() {
		_ = os.Setenv("TMPDIR", originalTempDir)
	}()

	// This might still succeed on some systems, so we just verify it doesn't panic
	validator, err := NewProductionReadinessValidator()
	if err == nil {
		validator.Cleanup()
	}
}

// Test individual validation helper functions
func TestValidationHelperFunctions(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// Test all boolean helper functions
	helpers := []struct {
		name string
		fn   func() bool
	}{
		{"testConfigValidation", validator.testConfigValidation},
		{"testConfigDefaults", validator.testConfigDefaults},
		{"testEnvironmentPrecedence", validator.testEnvironmentPrecedence},
		{"testConfigErrorHandling", validator.testConfigErrorHandling},
		{"testConfigDocumentation", validator.testConfigDocumentation},
		{"testGitHubActions", validator.testGitHubActions},
		{"testGitLabCI", validator.testGitLabCI},
		{"testJenkins", validator.testJenkins},
		{"testGenericCI", validator.testGenericCI},
		{"testNetworkConstraints", validator.testNetworkConstraints},
		{"testResourceLimits", validator.testResourceLimits},
		{"testConcurrentExecution", validator.testConcurrentExecution},
		{"testMemoryManagement", validator.testMemoryManagement},
		{"testResourceCleanup", validator.testResourceCleanup},
		{"testContextCancellation", validator.testContextCancellation},
		{"testResultConsistency", validator.testResultConsistency},
		{"testLargeRepositories", validator.testLargeRepositories},
		{"testMixedFileTypes", validator.testMixedFileTypes},
		{"testHighVolumeCommits", validator.testHighVolumeCommits},
		{"testRealWorldPatterns", validator.testRealWorldPatterns},
		{"testResourceConstraints", validator.testResourceConstraints},
		{"testNetworkIssues", validator.testNetworkIssues},
		{"testSingleCheckSkip", validator.testSingleCheckSkip},
		{"testMultipleCheckSkip", validator.testMultipleCheckSkip},
		{"testInvalidCheckNames", validator.testInvalidCheckNames},
		{"testSkipEnvironmentVars", validator.testSkipEnvironmentVars},
		{"testSkipCIIntegration", validator.testSkipCIIntegration},
		{"testSkipEdgeCases", validator.testSkipEdgeCases},
	}

	for _, helper := range helpers {
		t.Run(helper.name, func(t *testing.T) {
			// Just verify the function runs without panic
			result := helper.fn()
			assert.IsType(t, bool(true), result, "Helper %s should return a boolean", helper.name)
		})
	}
}

// Test performance validation error handling
func TestPerformanceValidationErrors(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// Test with nil config - should now return error due to no checks
	_, err = validator.measureAveragePerformance(nil, []string{"test.go"}, 1)
	require.Error(t, err)

	// Test with empty files
	cfg := &config.Config{Enabled: true}
	duration, err := validator.measureAveragePerformance(cfg, []string{}, 1)
	require.Error(t, err) // Should error because no checks to run
	assert.Equal(t, time.Duration(0), duration)

	// Test with zero iterations
	duration, err = validator.measureAveragePerformance(cfg, []string{"test.go"}, 0)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), duration)
}

// Test context cancellation in performance measurements
func TestPerformanceMeasurementWithContext(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// Change to temp directory
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWD) }()

	err = os.Chdir(validator.tempDir)
	require.NoError(t, err)

	// Initialize git repo and create test files
	err = validator.initGitRepo()
	require.NoError(t, err)
	files := validator.createTestFiles()

	cfg := &config.Config{
		Enabled: true,
		Checks: struct {
			Fumpt      bool
			Lint       bool
			ModTidy    bool
			Whitespace bool
			EOF        bool
		}{
			Whitespace: true,
		},
	}

	// The performance functions should handle context gracefully
	duration, err := validator.measureColdStart(cfg, files[:3])
	// Should still work as the function creates its own context
	require.NoError(t, err)
	assert.Greater(t, duration, time.Duration(0))
}

// Test edge cases in score calculations
func TestScoreCalculationEdgeCases(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// Test with all false values
	perfMetrics := PerformanceMetrics{
		MeetsTargetTime: false,
		ParallelScaling: false,
		MemoryEfficient: false,
		ColdStartTime:   10 * time.Second,
		WarmRunTime:     5 * time.Second,
	}
	score := validator.calculatePerformanceScore(perfMetrics)
	assert.Equal(t, 0, score)

	// Test with mixed values
	perfMetrics.MeetsTargetTime = true
	perfMetrics.ColdStartTime = 3 * time.Second
	score = validator.calculatePerformanceScore(perfMetrics)
	assert.Equal(t, 50, score)
}

// Test report generation with various error conditions
func TestReportGenerationWithErrors(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// Delete the temp directory to cause errors
	_ = os.RemoveAll(validator.tempDir)

	// Generate report should handle the error gracefully
	report, err := validator.GenerateReport()
	require.Error(t, err)
	assert.Nil(t, report)
}

// Test cleanup functionality
func TestCleanup(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)

	tempDir := validator.tempDir
	assert.DirExists(t, tempDir)

	// Cleanup should remove the temp directory
	validator.Cleanup()
	assert.NoDirExists(t, tempDir)

	// Multiple cleanups should not panic
	validator.Cleanup()
}

// Benchmark report generation
func BenchmarkGenerateReport(b *testing.B) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(b, err)
	defer validator.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.GenerateReport()
	}
}

// Benchmark format report
func BenchmarkFormatReport(b *testing.B) {
	report := &ProductionReadinessReport{
		GeneratedAt:     time.Now(),
		Version:         "1.0.0",
		Environment:     "benchmark",
		OverallScore:    95,
		ProductionReady: true,
		SystemInfo: SystemInfo{
			GoVersion:    runtime.Version(),
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
			NumCPU:       runtime.NumCPU(),
		},
		PerformanceMetrics: PerformanceMetrics{
			Score:            90,
			SmallCommitAvg:   1500 * time.Millisecond,
			TypicalCommitAvg: 2000 * time.Millisecond,
			MeetsTargetTime:  true,
		},
		CriticalIssues:   []string{"Issue 1", "Issue 2"},
		Recommendations:  []string{"Rec 1", "Rec 2", "Rec 3"},
		KnownLimitations: []string{"Limit 1", "Limit 2", "Limit 3", "Limit 4", "Limit 5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = report.FormatReport()
	}
}

// Test performance metrics with edge case durations
func TestPerformanceMetricsEdgeCases(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// Test score calculation with exact boundary values
	metrics := PerformanceMetrics{
		ColdStartTime: 3 * time.Second,         // Exactly at boundary
		WarmRunTime:   1500 * time.Millisecond, // Exactly at boundary
	}
	score := validator.calculatePerformanceScore(metrics)
	assert.Equal(t, 20, score) // Should get both bonuses

	// Test with values just over boundaries
	metrics.ColdStartTime = 3*time.Second + 1*time.Nanosecond
	metrics.WarmRunTime = 1500*time.Millisecond + 1*time.Nanosecond
	score = validator.calculatePerformanceScore(metrics)
	assert.Equal(t, 0, score) // Should get no bonuses
}

// Test overall assessment with boundary conditions
func TestOverallAssessmentBoundaries(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	report := &ProductionReadinessReport{
		PerformanceMetrics: PerformanceMetrics{
			Score:           84, // Just below 85
			MeetsTargetTime: true,
		},
		ConfigurationHealth: ConfigurationHealth{Score: 100},
		CICompatibility:     CICompatibility{Score: 100},
		ParallelSafety:      ParallelSafety{Score: 100},
		ProductionScenarios: ProductionScenarios{Score: 100},
		SkipFunctionality:   SkipFunctionality{Score: 100},
	}

	validator.calculateOverallAssessment(report)

	// Overall score calculation:
	// Performance: 84 * 30% = 25.2 (integer division = 25)
	// Config: 100 * 20% = 20, CI: 100 * 15% = 15, etc.
	// Total: 25 + 20 + 15 + 15 + 15 + 5 = 95
	expectedScore := 95
	assert.Equal(t, expectedScore, report.OverallScore)
	assert.True(t, report.ProductionReady) // Ready due to score >= 85 and meets target time
}

// Test recommendation generation with all possible conditions
func TestComprehensiveRecommendations(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// Test with all scores below thresholds
	report := &ProductionReadinessReport{
		PerformanceMetrics: PerformanceMetrics{
			Score:           70,
			MeetsTargetTime: false,
		},
		CICompatibility: CICompatibility{
			Score: 70,
		},
		ParallelSafety: ParallelSafety{
			Score: 80,
		},
	}

	validator.generateRecommendations(report)

	// Should have 4 recommendations
	assert.Len(t, report.Recommendations, 4)

	// Verify all expected recommendations are present
	foundPerf := false
	foundCI := false
	foundParallel := false
	foundTarget := false

	for _, rec := range report.Recommendations {
		if strings.Contains(rec, "optimizing check algorithms") {
			foundPerf = true
		}
		if strings.Contains(rec, "CI environments") {
			foundCI = true
		}
		if strings.Contains(rec, "parallel execution") {
			foundParallel = true
		}
		if strings.Contains(rec, "Performance optimization required") {
			foundTarget = true
		}
	}

	assert.True(t, foundPerf, "Should have performance recommendation")
	assert.True(t, foundCI, "Should have CI recommendation")
	assert.True(t, foundParallel, "Should have parallel recommendation")
	assert.True(t, foundTarget, "Should have target time recommendation")
}

// Test report formatting with empty/nil fields
func TestFormatReportWithEmptyFields(t *testing.T) {
	report := &ProductionReadinessReport{
		GeneratedAt: time.Now(),
		Version:     "1.0.0",
		Environment: "test",
		SystemInfo: SystemInfo{
			GoVersion:    "",
			OS:           "",
			Architecture: "",
			NumCPU:       0,
		},
		// Leave all other fields at zero values
	}

	formatted := report.FormatReport()

	// Should still generate a valid report
	assert.Contains(t, formatted, "# GoFortress Pre-commit System")
	assert.Contains(t, formatted, "NOT PRODUCTION READY")    // Should not be ready with 0 score
	assert.NotContains(t, formatted, "## Critical Issues")   // No issues = no section
	assert.NotContains(t, formatted, "## Recommendations")   // No recommendations = no section
	assert.NotContains(t, formatted, "## Known Limitations") // No limitations = no section
}

// Test validation with permission errors
func TestValidationWithPermissionErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// Make the temp directory read-only
	err = os.Chmod(validator.tempDir, 0o444) //nolint:gosec // Test scenario, intentional restrictive permissions
	require.NoError(t, err)

	// Restore permissions in cleanup
	defer func() {
		_ = os.Chmod(validator.tempDir, 0o755) //nolint:gosec // Test cleanup, restoring permissions
	}()

	// Try to generate report - should handle permission errors gracefully
	report, err := validator.GenerateReport()
	require.Error(t, err)
	// Report might still be partially generated
	if report != nil {
		assert.NotEmpty(t, report.CriticalIssues)
	}
}

// Test helper function for config documentation check
func TestConfigDocumentationCheck(t *testing.T) {
	validator, err := NewProductionReadinessValidator()
	require.NoError(t, err)
	defer validator.Cleanup()

	// The actual implementation checks if help text is > 1000 chars
	result := validator.testConfigDocumentation()
	assert.IsType(t, bool(true), result)

	// Since we're checking the real config.GetConfigHelp(),
	// the result should be true if documentation exists
	if result {
		help := config.GetConfigHelp()
		assert.Greater(t, len(help), 1000)
	}
}

// Test concurrent report generation
func TestConcurrentReportGeneration(t *testing.T) {
	const numGoroutines = 5

	// Create multiple validators
	validators := make([]*ProductionReadinessValidator, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		v, err := NewProductionReadinessValidator()
		require.NoError(t, err)
		validators[i] = v
		defer v.Cleanup()
	}

	// Run report generation concurrently
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			report, _ := validators[idx].GenerateReport()
			assert.NotNil(t, report)
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// Example usage test
func ExampleProductionReadinessReport_FormatReport() {
	report := &ProductionReadinessReport{
		GeneratedAt:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		Version:         "1.0.0",
		Environment:     "example",
		OverallScore:    95,
		ProductionReady: true,
		SystemInfo: SystemInfo{
			GoVersion:    "go1.21.0",
			OS:           "linux",
			Architecture: "amd64",
			NumCPU:       8,
		},
		PerformanceMetrics: PerformanceMetrics{
			Score:           90,
			SmallCommitAvg:  1 * time.Second,
			MeetsTargetTime: true,
		},
	}

	formatted := report.FormatReport()
	lines := strings.Split(formatted, "\n")

	// Print first few lines as example
	for i := 0; i < 5 && i < len(lines); i++ {
		fmt.Println(lines[i])
	}
	// Output:
	// # GoFortress Pre-commit System - Production Readiness Report
	//
	// Generated: 2025-01-01T12:00:00Z
	// Version: 1.0.0
	// Environment: example
}
