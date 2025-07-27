package impact

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/github"
)

func TestNewPRImpactAnalyzer(t *testing.T) {
	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{
			QualityGates: config.QualityGatesConfig{
				MinCoverage:     80.0,
				MaxRegression:   -2.0,
				RequiredBranches: []string{"main", "develop"},
			},
		},
	}
	
	analyzer := NewPRImpactAnalyzer(cfg)
	if analyzer == nil {
		t.Fatal("NewPRImpactAnalyzer returned nil")
	}
	if analyzer.config != cfg {
		t.Error("PR impact analyzer config not set correctly")
	}
}

func TestAnalyzePRImpact(t *testing.T) {
	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{
			QualityGates: config.QualityGatesConfig{
				MinCoverage:   80.0,
				MaxRegression: -2.0,
			},
		},
	}
	analyzer := NewPRImpactAnalyzer(cfg)
	
	tests := []struct {
		name     string
		prData   PRData
		expected string
	}{
		{
			name: "positive impact PR",
			prData: PRData{
				Number:      123,
				Title:       "Add comprehensive tests",
				Author:      "developer1",
				Branch:      "feature/add-tests",
				BaseBranch:  "main",
				Files: []FileChange{
					{
						Path:         "src/module.go",
						LinesAdded:   50,
						LinesRemoved: 10,
						Status:       "modified",
						CoverageChange: &CoverageChange{
							Before: 75.0,
							After:  85.0,
							Delta:  10.0,
						},
					},
					{
						Path:         "src/module_test.go",
						LinesAdded:   100,
						LinesRemoved: 0,
						Status:       "added",
						CoverageChange: &CoverageChange{
							Before: 0.0,
							After:  95.0,
							Delta:  95.0,
						},
					},
				},
			},
			expected: "positive",
		},
		{
			name: "negative impact PR",
			prData: PRData{
				Number:     456,
				Title:      "Remove deprecated code",
				Author:     "developer2",
				Branch:     "feature/cleanup",
				BaseBranch: "main",
				Files: []FileChange{
					{
						Path:         "src/legacy.go",
						LinesAdded:   0,
						LinesRemoved: 200,
						Status:       "removed",
						CoverageChange: &CoverageChange{
							Before: 90.0,
							After:  0.0,
							Delta:  -90.0,
						},
					},
					{
						Path:         "src/main.go",
						LinesAdded:   20,
						LinesRemoved: 50,
						Status:       "modified",
						CoverageChange: &CoverageChange{
							Before: 80.0,
							After:  75.0,
							Delta:  -5.0,
						},
					},
				},
			},
			expected: "negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.AnalyzePRImpact(context.Background(), tt.prData)
			if err != nil {
				t.Fatalf("AnalyzePRImpact() error = %v", err)
			}
			
			if result.OverallImpact != tt.expected {
				t.Errorf("AnalyzePRImpact() impact = %v, expected %v", result.OverallImpact, tt.expected)
			}
			
			// Verify result structure
			if result.PRNumber != tt.prData.Number {
				t.Error("PR number mismatch in result")
			}
			if len(result.FileImpacts) == 0 {
				t.Error("File impacts should not be empty")
			}
			if result.RiskScore < 0 || result.RiskScore > 100 {
				t.Errorf("Risk score should be between 0 and 100, got %v", result.RiskScore)
			}
		})
	}
}

func TestCalculateFileImpact(t *testing.T) {
	analyzer := NewPRImpactAnalyzer(&config.Config{})
	
	tests := []struct {
		name     string
		change   FileChange
		expected string
	}{
		{
			name: "significant positive impact",
			change: FileChange{
				Path:         "important.go",
				LinesAdded:   100,
				LinesRemoved: 0,
				Status:       "added",
				CoverageChange: &CoverageChange{
					Before: 0.0,
					After:  90.0,
					Delta:  90.0,
				},
			},
			expected: "high_positive",
		},
		{
			name: "moderate negative impact",
			change: FileChange{
				Path:         "feature.go",
				LinesAdded:   20,
				LinesRemoved: 50,
				Status:       "modified",
				CoverageChange: &CoverageChange{
					Before: 85.0,
					After:  80.0,
					Delta:  -5.0,
				},
			},
			expected: "moderate_negative",
		},
		{
			name: "no coverage impact",
			change: FileChange{
				Path:         "docs.md",
				LinesAdded:   10,
				LinesRemoved: 5,
				Status:       "modified",
				CoverageChange: nil,
			},
			expected: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impact := analyzer.calculateFileImpact(tt.change)
			
			if impact.ImpactLevel != tt.expected {
				t.Errorf("calculateFileImpact() level = %v, expected %v", impact.ImpactLevel, tt.expected)
			}
			
			if impact.Path != tt.change.Path {
				t.Error("File path mismatch in impact")
			}
		})
	}
}

func TestRiskAssessment(t *testing.T) {
	analyzer := NewPRImpactAnalyzer(&config.Config{})
	
	prData := PRData{
		Number:     789,
		Title:      "Refactor core module",
		Author:     "senior-dev",
		Branch:     "refactor/core",
		BaseBranch: "main",
		Files: []FileChange{
			{
				Path:         "core/engine.go",
				LinesAdded:   200,
				LinesRemoved: 150,
				Status:       "modified",
				CoverageChange: &CoverageChange{
					Before: 85.0,
					After:  82.0,
					Delta:  -3.0,
				},
			},
		},
		Metadata: map[string]interface{}{
			"has_tests":      true,
			"reviewer_count": 2,
			"complexity":     "high",
		},
	}
	
	assessment, err := analyzer.AssessRisk(context.Background(), prData)
	if err != nil {
		t.Fatalf("AssessRisk() error = %v", err)
	}
	
	if assessment.OverallRisk == "" {
		t.Error("Overall risk should not be empty")
	}
	
	if assessment.RiskScore < 0 || assessment.RiskScore > 100 {
		t.Errorf("Risk score should be between 0 and 100, got %v", assessment.RiskScore)
	}
	
	if len(assessment.RiskFactors) == 0 {
		t.Error("Risk factors should not be empty")
	}
	
	// Verify risk factors structure
	for _, factor := range assessment.RiskFactors {
		if factor.Factor == "" {
			t.Error("Risk factor name should not be empty")
		}
		if factor.Weight < 0 || factor.Weight > 1 {
			t.Errorf("Risk factor weight should be between 0 and 1, got %v", factor.Weight)
		}
	}
}

func TestQualityGateEvaluation(t *testing.T) {
	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{
			QualityGates: config.QualityGatesConfig{
				MinCoverage:      80.0,
				MaxRegression:    -2.0,
				RequiredBranches: []string{"main"},
				MaxComplexity:    10,
			},
		},
	}
	analyzer := NewPRImpactAnalyzer(cfg)
	
	tests := []struct {
		name       string
		prData     PRData
		expectPass bool
	}{
		{
			name: "passing quality gates",
			prData: PRData{
				Number:     100,
				BaseBranch: "main",
				Files: []FileChange{
					{
						Path: "good.go",
						CoverageChange: &CoverageChange{
							Before: 80.0,
							After:  85.0,
							Delta:  5.0,
						},
					},
				},
			},
			expectPass: true,
		},
		{
			name: "failing coverage gate",
			prData: PRData{
				Number:     101,
				BaseBranch: "main",
				Files: []FileChange{
					{
						Path: "bad.go",
						CoverageChange: &CoverageChange{
							Before: 85.0,
							After:  75.0, // Below minimum
							Delta:  -10.0,
						},
					},
				},
			},
			expectPass: false,
		},
		{
			name: "failing regression gate",
			prData: PRData{
				Number:     102,
				BaseBranch: "main",
				Files: []FileChange{
					{
						Path: "regression.go",
						CoverageChange: &CoverageChange{
							Before: 90.0,
							After:  85.0,
							Delta:  -5.0, // Exceeds max regression
						},
					},
				},
			},
			expectPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.EvaluateQualityGates(context.Background(), tt.prData)
			if err != nil {
				t.Fatalf("EvaluateQualityGates() error = %v", err)
			}
			
			if result.Passed != tt.expectPass {
				t.Errorf("EvaluateQualityGates() passed = %v, expected %v", result.Passed, tt.expectPass)
			}
			
			// Verify gates were evaluated
			if len(result.GateResults) == 0 {
				t.Error("Gate results should not be empty")
			}
			
			for _, gate := range result.GateResults {
				if gate.Name == "" {
					t.Error("Gate name should not be empty")
				}
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	analyzer := NewPRImpactAnalyzer(&config.Config{})
	
	analysis := PRImpactAnalysis{
		PRNumber:      123,
		OverallImpact: "negative",
		RiskScore:     75.0,
		FileImpacts: []FileImpact{
			{
				Path:         "critical.go",
				ImpactLevel:  "high_negative",
				CoverageDelta: -10.0,
			},
		},
		QualityGates: &QualityGateResult{
			Passed: false,
			GateResults: []GateResult{
				{Name: "min_coverage", Passed: false, Message: "Coverage below 80%"},
			},
		},
	}
	
	recommendations, err := analyzer.GenerateRecommendations(context.Background(), analysis)
	if err != nil {
		t.Fatalf("GenerateRecommendations() error = %v", err)
	}
	
	if len(recommendations) == 0 {
		t.Error("Should provide at least one recommendation")
	}
	
	// Verify recommendation structure
	for _, rec := range recommendations {
		if rec.Type == "" {
			t.Error("Recommendation type should not be empty")
		}
		if rec.Priority == "" {
			t.Error("Recommendation priority should not be empty")
		}
		if rec.Description == "" {
			t.Error("Recommendation description should not be empty")
		}
	}
}

func TestPredictPRMergeImpact(t *testing.T) {
	analyzer := NewPRImpactAnalyzer(&config.Config{})
	
	prData := PRData{
		Number:     456,
		Title:      "Feature implementation",
		Branch:     "feature/new-feature",
		BaseBranch: "main",
		Files: []FileChange{
			{
				Path:         "feature.go",
				LinesAdded:   150,
				LinesRemoved: 20,
				Status:       "added",
				CoverageChange: &CoverageChange{
					Before: 0.0,
					After:  80.0,
					Delta:  80.0,
				},
			},
		},
	}
	
	// Mock historical data
	historicalData := []PRHistoricalData{
		{
			PRNumber:        100,
			LinesChanged:    100,
			FilesChanged:    2,
			CoverageImpact:  5.0,
			PostMergeIssues: 0,
		},
		{
			PRNumber:        200,
			LinesChanged:    200,
			FilesChanged:    5,
			CoverageImpact:  -2.0,
			PostMergeIssues: 1,
		},
	}
	
	prediction, err := analyzer.PredictMergeImpact(context.Background(), prData, historicalData)
	if err != nil {
		t.Fatalf("PredictMergeImpact() error = %v", err)
	}
	
	if prediction.PredictedCoverageChange == 0 {
		t.Error("Should predict some coverage change")
	}
	
	if prediction.Confidence < 0 || prediction.Confidence > 1 {
		t.Errorf("Confidence should be between 0 and 1, got %v", prediction.Confidence)
	}
	
	if len(prediction.SimilarPRs) == 0 {
		t.Error("Should identify similar PRs")
	}
}

func TestComplexityAnalysis(t *testing.T) {
	analyzer := NewPRImpactAnalyzer(&config.Config{})
	
	prData := PRData{
		Files: []FileChange{
			{
				Path:         "simple.go",
				LinesAdded:   10,
				LinesRemoved: 5,
				Status:       "modified",
			},
			{
				Path:         "complex.go",
				LinesAdded:   500,
				LinesRemoved: 200,
				Status:       "modified",
			},
		},
	}
	
	complexity, err := analyzer.AnalyzeComplexity(context.Background(), prData)
	if err != nil {
		t.Fatalf("AnalyzeComplexity() error = %v", err)
	}
	
	if complexity.OverallComplexity == "" {
		t.Error("Overall complexity should not be empty")
	}
	
	if complexity.ComplexityScore < 0 {
		t.Error("Complexity score should not be negative")
	}
	
	if len(complexity.FileComplexities) != len(prData.Files) {
		t.Error("Should analyze complexity for all files")
	}
}

func TestGenerateImpactReport(t *testing.T) {
	analyzer := NewPRImpactAnalyzer(&config.Config{})
	
	analysis := PRImpactAnalysis{
		PRNumber:      789,
		Title:         "Test PR",
		OverallImpact: "positive",
		RiskScore:     25.0,
		FileImpacts: []FileImpact{
			{Path: "test.go", ImpactLevel: "high_positive"},
		},
	}
	
	report, err := analyzer.GenerateImpactReport(context.Background(), analysis)
	if err != nil {
		t.Fatalf("GenerateImpactReport() error = %v", err)
	}
	
	if report.Summary == "" {
		t.Error("Report summary should not be empty")
	}
	
	if report.ReportType == "" {
		t.Error("Report type should not be empty")
	}
	
	if len(report.Sections) == 0 {
		t.Error("Report should have sections")
	}
}

func BenchmarkAnalyzePRImpact(b *testing.B) {
	cfg := &config.Config{
		Analytics: config.AnalyticsConfig{
			QualityGates: config.QualityGatesConfig{
				MinCoverage:   80.0,
				MaxRegression: -2.0,
			},
		},
	}
	analyzer := NewPRImpactAnalyzer(cfg)
	
	// Generate large PR data
	files := make([]FileChange, 50)
	for i := 0; i < 50; i++ {
		files[i] = FileChange{
			Path:         fmt.Sprintf("file%d.go", i),
			LinesAdded:   100 + i*10,
			LinesRemoved: 20 + i*2,
			Status:       "modified",
			CoverageChange: &CoverageChange{
				Before: 80.0,
				After:  82.0 + float64(i)*0.5,
				Delta:  2.0 + float64(i)*0.5,
			},
		}
	}
	
	prData := PRData{
		Number:     999,
		Title:      "Large PR",
		Branch:     "feature/large",
		BaseBranch: "main",
		Files:      files,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzePRImpact(context.Background(), prData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRiskAssessment(b *testing.B) {
	analyzer := NewPRImpactAnalyzer(&config.Config{})
	
	prData := PRData{
		Number:     888,
		Title:      "Risk assessment test",
		Branch:     "test/risk",
		BaseBranch: "main",
		Files: []FileChange{
			{
				Path:         "risky.go",
				LinesAdded:   300,
				LinesRemoved: 100,
				Status:       "modified",
				CoverageChange: &CoverageChange{
					Before: 85.0,
					After:  80.0,
					Delta:  -5.0,
				},
			},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AssessRisk(context.Background(), prData)
		if err != nil {
			b.Fatal(err)
		}
	}
}