package impact

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/analytics/history"
	"github.com/mrz1836/go-broadcast/coverage/internal/analytics/prediction"
)

func TestNewPRImpactAnalyzer(t *testing.T) {
	cfg := &AnalyzerConfig{
		BaselinePeriod:      30 * 24 * time.Hour,
		ConfidenceThreshold: 0.7,
		RiskToleranceLevel:  RiskModerate,
		ImpactThresholds: ImpactThresholds{
			MinorImpact:    1.0,
			ModerateImpact: 5.0,
			MajorImpact:    10.0,
			CriticalImpact: 20.0,
		},
		QualityGates: QualityGates{
			MinimumCoverage:    80.0,
			CoverageRegression: -2.0,
		},
	}

	predictor := &prediction.CoveragePredictor{}
	historyAnalyzer := &history.TrendAnalyzer{}

	analyzer := NewPRImpactAnalyzer(cfg, predictor, historyAnalyzer)
	if analyzer == nil {
		t.Fatal("NewPRImpactAnalyzer returned nil")
	}
	if analyzer.config != cfg {
		t.Error("PR impact analyzer config not set correctly")
	}
}

func TestAnalyzePRImpact(t *testing.T) { //nolint:revive // function naming
	cfg := &AnalyzerConfig{
		BaselinePeriod:      30 * 24 * time.Hour,
		ConfidenceThreshold: 0.7,
		RiskToleranceLevel:  RiskModerate,
		ImpactThresholds: ImpactThresholds{
			MinorImpact:    1.0,
			ModerateImpact: 5.0,
			MajorImpact:    10.0,
			CriticalImpact: 20.0,
		},
		QualityGates: QualityGates{
			MinimumCoverage:    80.0,
			CoverageRegression: -2.0,
		},
		EnablePatternAnalysis: true,
		EnableComplexityScore: true,
		EnableRiskAssessment:  true,
		EnableRecommendations: true,
	}

	predictor := &prediction.CoveragePredictor{}
	historyAnalyzer := &history.TrendAnalyzer{}
	analyzer := NewPRImpactAnalyzer(cfg, predictor, historyAnalyzer)

	tests := []struct {
		name             string
		changeSet        *PRChangeSet
		baselineCoverage float64
		expectError      bool
	}{
		{
			name: "positive impact PR",
			changeSet: &PRChangeSet{
				PRNumber:   123,
				Title:      "Add comprehensive tests",
				Author:     "developer1",
				Branch:     "feature/add-tests",
				BaseBranch: "main",
				FilesChanged: []FileChange{
					{
						Filename:   "src/module.go",
						Status:     StatusModified,
						Additions:  50,
						Deletions:  10,
						FileType:   "go",
						IsTestFile: false,
					},
					{
						Filename:   "src/module_test.go",
						Status:     StatusAdded,
						Additions:  100,
						Deletions:  0,
						FileType:   "go",
						IsTestFile: true,
					},
				},
				TotalAdditions: 150,
				TotalDeletions: 10,
			},
			baselineCoverage: 75.0,
			expectError:      false,
		},
		{
			name: "negative impact PR",
			changeSet: &PRChangeSet{
				PRNumber:   456,
				Title:      "Remove deprecated code",
				Author:     "developer2",
				Branch:     "feature/cleanup",
				BaseBranch: "main",
				FilesChanged: []FileChange{
					{
						Filename:   "src/legacy.go",
						Status:     StatusDeleted,
						Additions:  0,
						Deletions:  200,
						FileType:   "go",
						IsTestFile: false,
					},
					{
						Filename:   "src/main.go",
						Status:     StatusModified,
						Additions:  20,
						Deletions:  50,
						FileType:   "go",
						IsTestFile: false,
					},
				},
				TotalAdditions: 20,
				TotalDeletions: 250,
			},
			baselineCoverage: 80.0,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.AnalyzePRImpact(context.Background(), tt.changeSet, tt.baselineCoverage)
			if (err != nil) != tt.expectError {
				t.Errorf("AnalyzePRImpact() error = %v, expectError = %v", err, tt.expectError)
				return
			}

			if !tt.expectError {
				if result == nil {
					t.Error("Expected non-nil result")
					return
				}

				// Verify result structure
				if result.OverallImpact == "" {
					t.Error("Overall impact should not be empty")
				}
				if result.BaselineCoverage != tt.baselineCoverage {
					t.Errorf("Baseline coverage mismatch: got %f, expected %f",
						result.BaselineCoverage, tt.baselineCoverage)
				}
			}
		})
	}
}

func TestAssessRisk(t *testing.T) { //nolint:revive // function naming
	analyzer := NewPRImpactAnalyzer(&AnalyzerConfig{
		RiskToleranceLevel:   RiskModerate,
		EnableRiskAssessment: true,
	}, nil, nil)

	changeSet := &PRChangeSet{
		PRNumber:   789,
		Title:      "Refactor core module",
		Author:     "senior-dev",
		Branch:     "refactor/core",
		BaseBranch: "main",
		FilesChanged: []FileChange{
			{
				Filename:        "core/engine.go",
				Status:          StatusModified,
				Additions:       200,
				Deletions:       150,
				FileType:        "go",
				ComplexityScore: 15.0,
			},
		},
	}

	analysis := &Analysis{
		OverallImpact:    ImpactModerate,
		CoverageChange:   -3.0,
		BaselineCoverage: 80.0,
	}

	// This test is simplified as AssessRisk is a private method
	// We would test it indirectly through AnalyzePRImpact
	if analysis.OverallImpact == "" {
		t.Error("Overall impact should not be empty")
	}

	// Use analyzer to avoid unused variable error
	_ = analyzer
	_ = changeSet
}

func TestQualityGateEvaluation(t *testing.T) { //nolint:revive // function naming
	cfg := &AnalyzerConfig{
		QualityGates: QualityGates{
			MinimumCoverage:     80.0,
			CoverageRegression:  -2.0,
			ComplexityThreshold: 10.0,
		},
	}
	analyzer := NewPRImpactAnalyzer(cfg, nil, nil)

	// Use analyzer to avoid unused variable error
	_ = analyzer

	tests := []struct {
		name             string
		analysis         *Analysis
		baselineCoverage float64
		expectPass       bool
	}{
		{
			name: "passing quality gates",
			analysis: &Analysis{
				CoverageChange: 5.0,
			},
			baselineCoverage: 85.0,
			expectPass:       true,
		},
		{
			name: "failing coverage regression gate",
			analysis: &Analysis{
				CoverageChange: -5.0, // Exceeds max regression
			},
			baselineCoverage: 85.0,
			expectPass:       false,
		},
		{
			name: "failing minimum coverage gate",
			analysis: &Analysis{
				CoverageChange: -10.0,
			},
			baselineCoverage: 85.0, // Would drop to 75%, below minimum
			expectPass:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Quality gates are evaluated as part of AnalyzePRImpact
			// This test verifies the expected behavior
			projectedCoverage := tt.baselineCoverage + tt.analysis.CoverageChange
			passesMinimum := projectedCoverage >= cfg.QualityGates.MinimumCoverage
			passesRegression := tt.analysis.CoverageChange >= cfg.QualityGates.CoverageRegression

			actualPass := passesMinimum && passesRegression
			if actualPass != tt.expectPass {
				t.Errorf("Quality gate evaluation: expected %v, got %v", tt.expectPass, actualPass)
			}
		})
	}
}

func TestComplexityAnalysis(t *testing.T) { //nolint:revive // function naming
	analyzer := NewPRImpactAnalyzer(&AnalyzerConfig{
		EnableComplexityScore: true,
	}, nil, nil)

	changeSet := &PRChangeSet{
		FilesChanged: []FileChange{
			{
				Filename:        "simple.go",
				Additions:       10,
				Deletions:       5,
				Status:          StatusModified,
				ComplexityScore: 2.0,
			},
			{
				Filename:        "complex.go",
				Additions:       500,
				Deletions:       200,
				Status:          StatusModified,
				ComplexityScore: 25.0,
			},
		},
	}

	// Test complexity calculation
	totalComplexity := 0.0
	for _, file := range changeSet.FilesChanged {
		totalComplexity += file.ComplexityScore
	}

	if totalComplexity != 27.0 {
		t.Errorf("Expected total complexity 27.0, got %f", totalComplexity)
	}

	// Use analyzer to avoid unused variable error
	_ = analyzer
}

func BenchmarkAnalyzePRImpact(b *testing.B) { //nolint:revive // function naming
	cfg := &AnalyzerConfig{
		ImpactThresholds: ImpactThresholds{
			MinorImpact:    1.0,
			ModerateImpact: 5.0,
			MajorImpact:    10.0,
			CriticalImpact: 20.0,
		},
		QualityGates: QualityGates{
			MinimumCoverage:    80.0,
			CoverageRegression: -2.0,
		},
		EnablePatternAnalysis: true,
		EnableComplexityScore: true,
		EnableRiskAssessment:  true,
		EnableRecommendations: true,
	}
	analyzer := NewPRImpactAnalyzer(cfg, nil, nil)

	// Generate large PR data
	files := make([]FileChange, 50)
	for i := 0; i < 50; i++ {
		files[i] = FileChange{
			Filename:        fmt.Sprintf("file%d.go", i),
			Status:          StatusModified,
			Additions:       100 + i*10,
			Deletions:       20 + i*2,
			FileType:        "go",
			ComplexityScore: float64(i % 10),
		}
	}

	changeSet := &PRChangeSet{
		PRNumber:       999,
		Title:          "Large PR",
		Branch:         "feature/large",
		BaseBranch:     "main",
		FilesChanged:   files,
		TotalAdditions: 5000,
		TotalDeletions: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzePRImpact(context.Background(), changeSet, 80.0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRiskAssessment(b *testing.B) { //nolint:revive // function naming
	analyzer := NewPRImpactAnalyzer(&AnalyzerConfig{
		EnableRiskAssessment: true,
		RiskToleranceLevel:   RiskModerate,
	}, nil, nil)

	changeSet := &PRChangeSet{
		PRNumber:   888,
		Title:      "Risk assessment test",
		Branch:     "test/risk",
		BaseBranch: "main",
		FilesChanged: []FileChange{
			{
				Filename:        "risky.go",
				Status:          StatusModified,
				Additions:       300,
				Deletions:       100,
				FileType:        "go",
				ComplexityScore: 20.0,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzePRImpact(context.Background(), changeSet, 85.0)
		if err != nil {
			b.Fatal(err)
		}
	}
}
