package reporting

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewPerformanceReporter(t *testing.T) {
	config := DefaultReportConfig()
	reporter := NewPerformanceReporter(config)

	require.NotNil(t, reporter)
	require.Equal(t, config, reporter.config)
	require.Nil(t, reporter.baseline)
}

func TestDefaultReportConfig(t *testing.T) {
	config := DefaultReportConfig()

	require.Equal(t, "./reports", config.OutputDirectory)
	require.Equal(t, "baseline.json", config.BaselineFile)
	require.True(t, config.IncludeProfiles)
	require.True(t, config.GenerateHTML)
	require.True(t, config.GenerateJSON)
	require.True(t, config.GenerateMarkdown)
	require.InDelta(t, 5.0, config.ComparisonThreshold, 0.001)
	require.Nil(t, config.CustomMetrics)
}

func TestPerformanceReporterLoadBaseline(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupFile   bool
		fileContent string
		wantErr     bool
		errContains string
	}{
		{
			name:        "ValidBaseline",
			setupFile:   true,
			fileContent: `{"timestamp":"2023-01-01T12:00:00Z","report_id":"123","version":"1.0","current_metrics":{"latency":100}}`,
			wantErr:     false,
		},
		{
			name:        "FileNotFound",
			setupFile:   false,
			wantErr:     true,
			errContains: "baseline file not found",
		},
		{
			name:        "InvalidJSON",
			setupFile:   true,
			fileContent: `invalid json content`,
			wantErr:     true,
			errContains: "failed to parse baseline file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := tempDir
			if tt.name == "FileNotFound" {
				outputDir = filepath.Join(tempDir, "nonexistent")
			}
			config := ReportConfig{
				OutputDirectory: outputDir,
				BaselineFile:    "test_baseline.json",
			}
			reporter := NewPerformanceReporter(config)

			if tt.setupFile {
				baselinePath := filepath.Join(outputDir, "test_baseline.json")
				err := os.MkdirAll(filepath.Dir(baselinePath), 0o750)
				require.NoError(t, err)
				err = os.WriteFile(baselinePath, []byte(tt.fileContent), 0o600)
				require.NoError(t, err)
			}

			err := reporter.LoadBaseline()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
				require.Nil(t, reporter.baseline)
			} else {
				require.NoError(t, err)
				require.NotNil(t, reporter.baseline)
				require.Equal(t, "123", reporter.baseline.ReportID)
			}
		})
	}
}

func TestPerformanceReporterSaveBaseline(t *testing.T) {
	tempDir := t.TempDir()

	config := ReportConfig{
		OutputDirectory: tempDir,
		BaselineFile:    "test_baseline.json",
	}
	reporter := NewPerformanceReporter(config)

	report := &PerformanceReport{
		Timestamp:      time.Now(),
		ReportID:       "save-test-123",
		Version:        "1.0",
		CurrentMetrics: map[string]float64{"latency": 150.5},
	}

	err := reporter.SaveBaseline(report)
	require.NoError(t, err)
	require.Equal(t, report, reporter.baseline)

	// Verify file was created with correct content
	baselinePath := filepath.Join(tempDir, "test_baseline.json")
	data, err := os.ReadFile(baselinePath) //nolint:gosec // Reading test file is safe
	require.NoError(t, err)

	var savedReport PerformanceReport
	err = json.Unmarshal(data, &savedReport)
	require.NoError(t, err)
	require.Equal(t, "save-test-123", savedReport.ReportID)
	require.InDelta(t, 150.5, savedReport.CurrentMetrics["latency"], 0.001)

	// Verify file permissions
	info, err := os.Stat(baselinePath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestPerformanceReporterGenerateReport(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		hasBaseline     bool
		currentMetrics  map[string]float64
		baselineMetrics map[string]float64
		testResults     []TestResult
		customMetrics   map[string]interface{}
		verify          func(t *testing.T, report *PerformanceReport)
	}{
		{
			name:           "WithoutBaseline",
			hasBaseline:    false,
			currentMetrics: map[string]float64{"latency": 100, "throughput": 1000},
			testResults: []TestResult{
				{Name: "test1", Success: true, Duration: time.Second},
				{Name: "test2", Success: false, Error: "timeout"},
			},
			verify: func(t *testing.T, report *PerformanceReport) {
				require.NotEmpty(t, report.ReportID)
				require.Equal(t, "1.0", report.Version)
				require.InDelta(t, 100.0, report.CurrentMetrics["latency"], 0.001)
				require.InDelta(t, 1000.0, report.CurrentMetrics["throughput"], 0.001)
				require.Equal(t, 2, report.TotalTests)
				require.Equal(t, 1, report.PassedTests)
				require.Equal(t, 1, report.FailedTests)
				require.Empty(t, report.BaselineMetrics)
				require.Empty(t, report.Improvements)
				require.Empty(t, report.Regressions)
			},
		},
		{
			name:            "WithBaseline",
			hasBaseline:     true,
			currentMetrics:  map[string]float64{"latency": 80, "throughput": 1200},
			baselineMetrics: map[string]float64{"latency": 100, "throughput": 1000},
			testResults:     []TestResult{{Name: "test1", Success: true}},
			verify: func(t *testing.T, report *PerformanceReport) {
				require.NotEmpty(t, report.BaselineMetrics)
				require.InDelta(t, 100.0, report.BaselineMetrics["latency"], 0.001)
				require.NotEmpty(t, report.Improvements)
				// Latency improvement (lower is better)
				require.Contains(t, report.Improvements, "latency")
				require.Contains(t, report.Improvements, "throughput")
			},
		},
		{
			name:           "WithCustomMetrics",
			hasBaseline:    false,
			currentMetrics: map[string]float64{"latency": 100},
			customMetrics:  map[string]interface{}{"custom_metric": 42.5, "invalid_metric": "string"},
			testResults:    []TestResult{},
			verify: func(t *testing.T, report *PerformanceReport) {
				require.InDelta(t, 42.5, report.CurrentMetrics["custom_metric"], 0.001)
				require.NotContains(t, report.CurrentMetrics, "invalid_metric")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ReportConfig{
				OutputDirectory:     tempDir,
				CustomMetrics:       tt.customMetrics,
				ComparisonThreshold: 5.0,
			}
			reporter := NewPerformanceReporter(config)

			if tt.hasBaseline {
				baseline := &PerformanceReport{
					CurrentMetrics: tt.baselineMetrics,
				}
				reporter.baseline = baseline
			}

			profileSummary := ProfileSummary{
				CPUProfile: ProfileInfo{Available: true, Size: 1024},
			}

			report, err := reporter.GenerateReport(tt.currentMetrics, tt.testResults, profileSummary)
			require.NoError(t, err)
			require.NotNil(t, report)

			// Common verifications
			require.False(t, report.Timestamp.IsZero())
			require.NotEmpty(t, report.ReportID)
			require.Equal(t, "1.0", report.Version)
			require.NotNil(t, report.SystemInfo)
			require.Equal(t, profileSummary, report.ProfileSummary)

			tt.verify(t, report)
		})
	}
}

func TestCalculatePerformanceChanges(t *testing.T) {
	config := ReportConfig{ComparisonThreshold: 10.0}
	reporter := NewPerformanceReporter(config)

	tests := []struct {
		name             string
		currentMetrics   map[string]float64
		baselineMetrics  map[string]float64
		wantImprovements map[string]float64
		wantRegressions  map[string]float64
	}{
		{
			name:             "Latency Improvement",
			currentMetrics:   map[string]float64{"latency": 80},
			baselineMetrics:  map[string]float64{"latency": 100},
			wantImprovements: map[string]float64{"latency": 20.0}, // 20% improvement
			wantRegressions:  map[string]float64{},
		},
		{
			name:             "Throughput Improvement",
			currentMetrics:   map[string]float64{"throughput": 120},
			baselineMetrics:  map[string]float64{"throughput": 100},
			wantImprovements: map[string]float64{"throughput": 20.0}, // 20% improvement
			wantRegressions:  map[string]float64{},
		},
		{
			name:             "Memory Regression",
			currentMetrics:   map[string]float64{"memory_usage_mb": 150},
			baselineMetrics:  map[string]float64{"memory_usage_mb": 100},
			wantImprovements: map[string]float64{},
			wantRegressions:  map[string]float64{"memory_usage_mb": 50.0}, // 50% regression
		},
		{
			name:             "Below Threshold Change",
			currentMetrics:   map[string]float64{"latency": 95},
			baselineMetrics:  map[string]float64{"latency": 100},
			wantImprovements: map[string]float64{}, // 5% below threshold
			wantRegressions:  map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &PerformanceReport{
				CurrentMetrics:  tt.currentMetrics,
				BaselineMetrics: tt.baselineMetrics,
				Improvements:    make(map[string]float64),
				Regressions:     make(map[string]float64),
			}

			reporter.calculatePerformanceChanges(report)

			require.Len(t, report.Improvements, len(tt.wantImprovements))
			require.Len(t, report.Regressions, len(tt.wantRegressions))

			for metric, expectedImprovement := range tt.wantImprovements {
				actualImprovement, exists := report.Improvements[metric]
				require.True(t, exists, "Expected improvement for %s", metric)
				require.InDelta(t, expectedImprovement, actualImprovement, 0.1)
			}

			for metric, expectedRegression := range tt.wantRegressions {
				actualRegression, exists := report.Regressions[metric]
				require.True(t, exists, "Expected regression for %s", metric)
				require.InDelta(t, expectedRegression, actualRegression, 0.1)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	config := DefaultReportConfig()
	reporter := NewPerformanceReporter(config)

	tests := []struct {
		name      string
		report    *PerformanceReport
		wantCount int
		wantTypes []string
	}{
		{
			name: "HighMemoryUsage",
			report: &PerformanceReport{
				CurrentMetrics: map[string]float64{"memory_usage_mb": 600},
				TestResults:    []TestResult{{Success: true}},
				Improvements:   map[string]float64{},
				Regressions:    map[string]float64{},
			},
			wantCount: 1,
			wantTypes: []string{"mem-high-usage"},
		},
		{
			name: "TestFailures",
			report: &PerformanceReport{
				CurrentMetrics: map[string]float64{},
				TotalTests:     5,
				FailedTests:    2,
				TestResults: []TestResult{
					{Name: "test1", Success: true},
					{Name: "test2", Success: false, Error: "timeout"},
					{Name: "test3", Success: false, Error: "memory error"},
				},
				Improvements: map[string]float64{},
				Regressions:  map[string]float64{},
			},
			wantCount: 1,
			wantTypes: []string{"test-failures"},
		},
		{
			name: "PerformanceRegressions",
			report: &PerformanceReport{
				CurrentMetrics: map[string]float64{},
				TestResults:    []TestResult{{Success: true}},
				Improvements:   map[string]float64{},
				Regressions:    map[string]float64{"response_time": 30.0, "throughput": 15.0},
			},
			wantCount: 2,
			wantTypes: []string{"regression-response-time", "regression-throughput"},
		},
		{
			name: "LargeProfiles",
			report: &PerformanceReport{
				CurrentMetrics: map[string]float64{},
				TestResults:    []TestResult{{Success: true}},
				Improvements:   map[string]float64{},
				Regressions:    map[string]float64{},
				ProfileSummary: ProfileSummary{TotalProfileSize: 200 * 1024 * 1024}, // 200MB
			},
			wantCount: 1,
			wantTypes: []string{"large-profiles"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := reporter.generateRecommendations(tt.report)

			require.Len(t, recommendations, tt.wantCount)

			for i, expectedType := range tt.wantTypes {
				require.Equal(t, expectedType, recommendations[i].ID)
				require.NotEmpty(t, recommendations[i].Title)
				require.NotEmpty(t, recommendations[i].Description)
				require.NotEmpty(t, recommendations[i].Action)
				require.NotEmpty(t, recommendations[i].Impact)
				require.NotEmpty(t, recommendations[i].Category)
			}
		})
	}
}

func TestAnalyzeMemoryMetrics(t *testing.T) {
	config := DefaultReportConfig()
	reporter := NewPerformanceReporter(config)

	tests := []struct {
		name        string
		report      *PerformanceReport
		hasBaseline bool
		wantCount   int
		wantIDs     []string
	}{
		{
			name: "HighMemoryUsage",
			report: &PerformanceReport{
				CurrentMetrics: map[string]float64{"memory_usage_mb": 600},
			},
			wantCount: 1,
			wantIDs:   []string{"mem-high-usage"},
		},
		{
			name: "MemoryGrowth",
			report: &PerformanceReport{
				CurrentMetrics:  map[string]float64{"memory_usage_mb": 150},
				BaselineMetrics: map[string]float64{"memory_usage_mb": 100},
			},
			hasBaseline: true,
			wantCount:   1,
			wantIDs:     []string{"mem-growth"},
		},
		{
			name: "AcceptableMemoryUsage",
			report: &PerformanceReport{
				CurrentMetrics: map[string]float64{"memory_usage_mb": 200},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hasBaseline {
				reporter.baseline = &PerformanceReport{
					CurrentMetrics: tt.report.BaselineMetrics,
				}
			} else {
				reporter.baseline = nil
			}

			recommendations := reporter.analyzeMemoryMetrics(tt.report)

			require.Len(t, recommendations, tt.wantCount)
			for i, expectedID := range tt.wantIDs {
				require.Equal(t, expectedID, recommendations[i].ID)
				require.Equal(t, "Memory", recommendations[i].Category)
			}
		})
	}
}

func TestAnalyzeRegressions(t *testing.T) {
	config := DefaultReportConfig()
	reporter := NewPerformanceReporter(config)

	report := &PerformanceReport{
		Regressions: map[string]float64{
			"high_regression":   30.0, // Should be high priority
			"medium_regression": 15.0, // Should be medium priority
			"low_regression":    5.0,  // Should be low priority
		},
	}

	recommendations := reporter.analyzeRegressions(report)

	require.Len(t, recommendations, 3)

	// Check priorities are assigned correctly
	highPriorityFound := false
	mediumPriorityFound := false
	lowPriorityFound := false

	for _, rec := range recommendations {
		require.Equal(t, "Performance Regression", rec.Category)
		require.Contains(t, rec.ID, "regression-")

		switch rec.Priority {
		case PriorityHigh:
			highPriorityFound = true
		case PriorityMedium:
			mediumPriorityFound = true
		case PriorityLow:
			lowPriorityFound = true
		}
	}

	require.True(t, highPriorityFound)
	require.True(t, mediumPriorityFound)
	require.True(t, lowPriorityFound)
}

func TestAnalyzeTestFailures(t *testing.T) {
	config := DefaultReportConfig()
	reporter := NewPerformanceReporter(config)

	tests := []struct {
		name         string
		report       *PerformanceReport
		wantCount    int
		wantPriority RecommendationPriority
	}{
		{
			name: "HighFailureRate",
			report: &PerformanceReport{
				TotalTests:  10,
				FailedTests: 3, // 30% failure rate
				TestResults: []TestResult{
					{Name: "test1", Success: false},
					{Name: "test2", Success: false},
					{Name: "test3", Success: false},
				},
			},
			wantCount:    1,
			wantPriority: PriorityHigh,
		},
		{
			name: "MediumFailureRate",
			report: &PerformanceReport{
				TotalTests:  10,
				FailedTests: 1, // 10% failure rate
				TestResults: []TestResult{
					{Name: "test1", Success: false},
				},
			},
			wantCount:    1,
			wantPriority: PriorityMedium,
		},
		{
			name: "NoFailures",
			report: &PerformanceReport{
				TotalTests:  10,
				FailedTests: 0,
				TestResults: []TestResult{},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations := reporter.analyzeTestFailures(tt.report)

			require.Len(t, recommendations, tt.wantCount)
			if tt.wantCount > 0 {
				require.Equal(t, "test-failures", recommendations[0].ID)
				require.Equal(t, "Test Failures", recommendations[0].Category)
				require.Equal(t, tt.wantPriority, recommendations[0].Priority)
			}
		})
	}
}

func TestSaveReport(t *testing.T) {
	tempDir := t.TempDir()

	config := ReportConfig{
		OutputDirectory:  tempDir,
		GenerateJSON:     true,
		GenerateMarkdown: true,
		GenerateHTML:     true,
	}
	reporter := NewPerformanceReporter(config)

	report := &PerformanceReport{
		ReportID:       "test-123",
		Version:        "1.0",
		Timestamp:      time.Now(),
		CurrentMetrics: map[string]float64{"latency": 100},
		SystemInfo:     getSystemInfo(),
		TestResults:    []TestResult{{Name: "test1", Success: true}},
		Recommendations: []Recommendation{
			{
				ID:          "test-rec",
				Priority:    PriorityMedium,
				Category:    "Test",
				Title:       "Test Recommendation",
				Description: "Test description",
				Action:      "Test action",
				Impact:      "Test impact",
			},
		},
	}

	err := reporter.SaveReport(report)
	require.NoError(t, err)

	// Verify files were created
	baseFileName := "performance_report_test-123"
	jsonFile := filepath.Join(tempDir, baseFileName+".json")
	mdFile := filepath.Join(tempDir, baseFileName+".md")
	htmlFile := filepath.Join(tempDir, baseFileName+".html")

	require.FileExists(t, jsonFile)
	require.FileExists(t, mdFile)
	require.FileExists(t, htmlFile)

	// Verify JSON content
	jsonData, err := os.ReadFile(jsonFile) //nolint:gosec // Reading test file is safe
	require.NoError(t, err)
	var savedReport PerformanceReport
	err = json.Unmarshal(jsonData, &savedReport)
	require.NoError(t, err)
	require.Equal(t, "test-123", savedReport.ReportID)

	// Verify Markdown content contains expected sections
	mdData, err := os.ReadFile(mdFile) //nolint:gosec // Reading test file is safe
	require.NoError(t, err)
	mdContent := string(mdData)
	require.Contains(t, mdContent, "# Performance Analysis Report")
	require.Contains(t, mdContent, "test-123")
	require.Contains(t, mdContent, "Test Recommendation")

	// Verify HTML content contains expected sections
	htmlData, err := os.ReadFile(htmlFile) //nolint:gosec // Reading test file is safe
	require.NoError(t, err)
	htmlContent := string(htmlData)
	require.Contains(t, htmlContent, "<!DOCTYPE html>")
	require.Contains(t, htmlContent, "test-123")
	require.Contains(t, htmlContent, "Test Recommendation")
}

func TestHelperFunctions(t *testing.T) {
	t.Run("GenerateReportID", func(t *testing.T) {
		id1 := generateReportID()
		time.Sleep(time.Millisecond) // Ensure different timestamp
		id2 := generateReportID()

		require.NotEmpty(t, id1)
		require.NotEmpty(t, id2)
		// IDs might be the same if generated within the same second, so we test they're numeric
		require.Regexp(t, `^\d+$`, id1)
		require.Regexp(t, `^\d+$`, id2)
	})

	t.Run("GetSystemInfo", func(t *testing.T) {
		sysInfo := getSystemInfo()

		require.NotEmpty(t, sysInfo.GoVersion)
		require.NotEmpty(t, sysInfo.GOOS)
		require.NotEmpty(t, sysInfo.GOARCH)
		require.Positive(t, sysInfo.NumCPU)
		require.Positive(t, sysInfo.GOMAXPROCS)
		require.False(t, sysInfo.Timestamp.IsZero())
	})

	t.Run("IsLowerBetterMetric", func(t *testing.T) {
		tests := []struct {
			metric   string
			expected bool
		}{
			{"latency", true},
			{"response_time", true},
			{"memory_usage", true},
			{"throughput", false},
			{"ops_per_sec", false},
			{"unknown_metric", false},
		}

		for _, tt := range tests {
			result := isLowerBetterMetric(tt.metric)
			require.Equal(t, tt.expected, result, "metric: %s", tt.metric)
		}
	})

	t.Run("IsHigherBetterMetric", func(t *testing.T) {
		tests := []struct {
			metric   string
			expected bool
		}{
			{"throughput", true},
			{"ops_per_sec", true},
			{"requests_per_sec", true},
			{"latency", false},
			{"memory_usage", false},
			{"unknown_metric", false},
		}

		for _, tt := range tests {
			result := isHigherBetterMetric(tt.metric)
			require.Equal(t, tt.expected, result, "metric: %s", tt.metric)
		}
	})

	t.Run("FormatMetricName", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"response_time", "Response Time"},
			{"memory_usage_mb", "Memory Usage Mb"},
			{"ops_per_sec", "Ops Per Sec"},
			{"single", "Single"},
			{"", ""},
		}

		for _, tt := range tests {
			result := formatMetricName(tt.input)
			require.Equal(t, tt.expected, result)
		}
	})

	t.Run("Abs", func(t *testing.T) {
		require.InDelta(t, 5.0, abs(5.0), 0.001)
		require.InDelta(t, 5.0, abs(-5.0), 0.001)
		require.InDelta(t, 0.0, abs(0.0), 0.001)
	})
}

func TestTemplateFunctions(t *testing.T) {
	config := DefaultReportConfig()
	reporter := NewPerformanceReporter(config)
	funcs := reporter.getTemplateFuncs()

	t.Run("FormatFloat", func(t *testing.T) {
		formatFloat := funcs["formatFloat"].(func(float64) string)
		require.Equal(t, "123.46", formatFloat(123.456))
		require.Equal(t, "0.00", formatFloat(0))
	})

	t.Run("FormatPercent", func(t *testing.T) {
		formatPercent := funcs["formatPercent"].(func(float64) string)
		require.Equal(t, "23.5%", formatPercent(23.456))
		require.Equal(t, "0.0%", formatPercent(0))
	})

	t.Run("FormatBytes", func(t *testing.T) {
		formatBytes := funcs["formatBytes"].(func(int64) string)
		require.Equal(t, "512 B", formatBytes(512))
		require.Equal(t, "1.0 KB", formatBytes(1024))
		require.Equal(t, "1.5 KB", formatBytes(1536))
		require.Equal(t, "1.0 MB", formatBytes(1024*1024))
	})

	t.Run("Title", func(t *testing.T) {
		title := funcs["title"].(func(interface{}) string)
		require.Equal(t, "Hello", title("hello"))
		require.Equal(t, "World", title("WORLD"))
		require.Empty(t, title(""))
		require.Equal(t, "High", title(PriorityHigh))
	})

	t.Run("PriorityClass", func(t *testing.T) {
		priorityClass := funcs["priorityClass"].(func(RecommendationPriority) string)
		require.Equal(t, "priority-high", priorityClass(PriorityHigh))
		require.Equal(t, "priority-medium", priorityClass(PriorityMedium))
		require.Equal(t, "priority-low", priorityClass(PriorityLow))
		require.Empty(t, priorityClass(RecommendationPriority("unknown")))
	})
}

func TestRecommendationPriorityString(t *testing.T) {
	// Test that RecommendationPriority constants are properly defined
	require.Equal(t, "high", string(PriorityHigh))
	require.Equal(t, "medium", string(PriorityMedium))
	require.Equal(t, "low", string(PriorityLow))
}

func TestPerformanceReportJSON(t *testing.T) {
	// Test JSON marshaling/unmarshaling of PerformanceReport
	original := PerformanceReport{
		Timestamp:      time.Now(),
		ReportID:       "json-test-123",
		Version:        "1.0",
		CurrentMetrics: map[string]float64{"latency": 100.5},
		Improvements:   map[string]float64{"throughput": 20.0},
		Regressions:    map[string]float64{"memory": 15.0},
		Recommendations: []Recommendation{
			{
				ID:          "test-rec",
				Priority:    PriorityHigh,
				Category:    "Test",
				Title:       "Test Title",
				Description: "Test Description",
				Action:      "Test Action",
				Impact:      "Test Impact",
				Evidence:    []string{"evidence1", "evidence2"},
				References:  []string{"ref1"},
			},
		},
		SystemInfo: SystemInfo{
			GoVersion: "go1.21.0",
			GOOS:      "linux",
			GOARCH:    "amd64",
		},
		TestResults: []TestResult{
			{
				Name:       "test1",
				Duration:   time.Second,
				Success:    true,
				Throughput: 1000.0,
				MemoryUsed: 50,
			},
		},
		ProfileSummary: ProfileSummary{
			CPUProfile: ProfileInfo{Available: true, Size: 1024, Path: "/tmp/cpu.prof"},
		},
		Duration:    5 * time.Second,
		TotalTests:  5,
		PassedTests: 4,
		FailedTests: 1,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Unmarshal from JSON
	var unmarshaled PerformanceReport
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	// Verify key fields
	require.Equal(t, original.ReportID, unmarshaled.ReportID)
	require.Equal(t, original.Version, unmarshaled.Version)
	require.Len(t, unmarshaled.CurrentMetrics, len(original.CurrentMetrics))
	require.InDelta(t, original.CurrentMetrics["latency"], unmarshaled.CurrentMetrics["latency"], 0.001)
	require.Len(t, unmarshaled.Recommendations, len(original.Recommendations))
	require.Equal(t, original.Recommendations[0].ID, unmarshaled.Recommendations[0].ID)
	require.Equal(t, original.TotalTests, unmarshaled.TotalTests)
}

// BenchmarkGenerateReport tests the performance of report generation
func BenchmarkGenerateReport(b *testing.B) {
	config := DefaultReportConfig()
	reporter := NewPerformanceReporter(config)

	currentMetrics := map[string]float64{
		"latency":     100.5,
		"throughput":  1000.0,
		"memory_mb":   256.0,
		"cpu_percent": 75.0,
	}

	testResults := []TestResult{
		{Name: "test1", Success: true, Duration: time.Second, Throughput: 1000},
		{Name: "test2", Success: false, Error: "timeout", Duration: 2 * time.Second},
		{Name: "test3", Success: true, Duration: 500 * time.Millisecond, Throughput: 2000},
	}

	profileSummary := ProfileSummary{
		CPUProfile:       ProfileInfo{Available: true, Size: 1024},
		MemoryProfile:    ProfileInfo{Available: true, Size: 2048},
		TotalProfileSize: 3072,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reporter.GenerateReport(currentMetrics, testResults, profileSummary)
		if err != nil {
			b.Fatal(err)
		}
	}
}
