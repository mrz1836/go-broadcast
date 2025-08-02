package dashboard

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCoverageData_JSON(t *testing.T) {
	data := &CoverageData{
		ProjectName:    "test-project",
		RepositoryURL:  "https://github.com/owner/repo",
		Branch:         "master",
		CommitSHA:      "abc123",
		PRNumber:       "42",
		BadgeURL:       "https://owner.github.io/repo/coverage.svg",
		Timestamp:      time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		TotalCoverage:  85.5,
		TotalLines:     1000,
		CoveredLines:   855,
		MissedLines:    145,
		TotalFiles:     10,
		CoveredFiles:   8,
		PartialFiles:   1,
		UncoveredFiles: 1,
		Packages: []PackageCoverage{
			{
				Name:         "github.com/owner/repo/pkg1",
				Path:         "pkg1",
				Coverage:     90.0,
				TotalLines:   100,
				CoveredLines: 90,
				MissedLines:  10,
				Files: []FileCoverage{
					{
						Name:         "file1.go",
						Path:         "pkg1/file1.go",
						Coverage:     95.0,
						TotalLines:   50,
						CoveredLines: 47,
						MissedLines:  3,
						GitHubURL:    "https://github.com/owner/repo/blob/main/pkg1/file1.go",
					},
				},
			},
		},
		TrendData: &TrendData{
			Direction:       "up",
			ChangePercent:   2.5,
			ChangeLines:     25,
			ComparedTo:      "branch",
			ComparedToValue: "develop",
		},
		History: []HistoricalPoint{
			{
				Timestamp:    time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
				CommitSHA:    "def456",
				Coverage:     83.0,
				TotalLines:   980,
				CoveredLines: 813,
			},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal CoverageData: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled CoverageData
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal CoverageData: %v", err)
	}

	// Verify key fields
	if unmarshaled.ProjectName != data.ProjectName {
		t.Errorf("ProjectName = %v, want %v", unmarshaled.ProjectName, data.ProjectName)
	}
	if unmarshaled.TotalCoverage != data.TotalCoverage {
		t.Errorf("TotalCoverage = %v, want %v", unmarshaled.TotalCoverage, data.TotalCoverage)
	}
	if len(unmarshaled.Packages) != len(data.Packages) {
		t.Errorf("Packages length = %v, want %v", len(unmarshaled.Packages), len(data.Packages))
	}
	if unmarshaled.TrendData == nil {
		t.Error("TrendData is nil after unmarshaling")
	}
	if len(unmarshaled.History) != len(data.History) {
		t.Errorf("History length = %v, want %v", len(unmarshaled.History), len(data.History))
	}
}

func TestPackageCoverage_Calculations(t *testing.T) {
	pkg := &PackageCoverage{
		Name:         "test-package",
		Path:         "pkg",
		TotalLines:   100,
		CoveredLines: 85,
		MissedLines:  15,
	}

	// Calculate coverage percentage
	expectedCoverage := float64(pkg.CoveredLines) / float64(pkg.TotalLines) * 100
	pkg.Coverage = expectedCoverage

	if pkg.Coverage != 85.0 {
		t.Errorf("Coverage = %v, want 85.0", pkg.Coverage)
	}

	// Verify line counts
	if pkg.CoveredLines+pkg.MissedLines != pkg.TotalLines {
		t.Error("CoveredLines + MissedLines should equal TotalLines")
	}
}

func TestBranchInfo_URLs(t *testing.T) {
	branch := &BranchInfo{
		Name:         "feature/test",
		IsProtected:  false,
		Coverage:     90.5,
		LastUpdate:   time.Now(),
		BadgeURL:     "/badges/feature-test.svg",
		ReportURL:    "/reports/feature-test/",
		CommitSHA:    "abc123def456",
		CommitURL:    "https://github.com/owner/repo/commit/abc123def456",
		TotalLines:   1000,
		CoveredLines: 905,
	}

	// Verify URL formatting
	if branch.BadgeURL != "/badges/feature-test.svg" {
		t.Errorf("BadgeURL = %v, want /badges/feature-test.svg", branch.BadgeURL)
	}
	if branch.ReportURL != "/reports/feature-test/" {
		t.Errorf("ReportURL = %v, want /reports/feature-test/", branch.ReportURL)
	}

	// Verify commit URL contains SHA
	if !containsStr(branch.CommitURL, branch.CommitSHA) {
		t.Error("CommitURL should contain CommitSHA")
	}
}

func TestQualityStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     QualityStatus
		shouldPass bool
	}{
		{
			name: "passing quality gate",
			status: QualityStatus{
				Passed:      true,
				Threshold:   80.0,
				ActualValue: 85.5,
				Message:     "Coverage exceeds threshold",
			},
			shouldPass: true,
		},
		{
			name: "failing quality gate",
			status: QualityStatus{
				Passed:      false,
				Threshold:   80.0,
				ActualValue: 75.0,
				Message:     "Coverage below threshold",
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status.Passed != tt.shouldPass {
				t.Errorf("Passed = %v, want %v", tt.status.Passed, tt.shouldPass)
			}
			if tt.shouldPass && tt.status.ActualValue < tt.status.Threshold {
				t.Error("Passing status should have ActualValue >= Threshold")
			}
			if !tt.shouldPass && tt.status.ActualValue >= tt.status.Threshold {
				t.Error("Failing status should have ActualValue < Threshold")
			}
		})
	}
}

func TestDashboardMetadata(t *testing.T) {
	metadata := &Metadata{
		GeneratedAt:      time.Now(),
		GeneratorVersion: "1.0.0",
		DataVersion:      "1.0",
		Branches: []BranchInfo{
			{
				Name:     "master",
				Coverage: 85.5,
			},
			{
				Name:     "develop",
				Coverage: 83.0,
			},
		},
		LastUpdated: time.Now(),
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal Metadata: %v", err)
	}

	// Verify JSON contains expected fields
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	expectedFields := []string{
		"generated_at",
		"generator_version",
		"data_version",
		"branches",
		"last_updated",
	}

	for _, field := range expectedFields {
		if _, ok := jsonMap[field]; !ok {
			t.Errorf("JSON missing expected field: %s", field)
		}
	}
}

// containsStr checks if a string contains a substring
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestChartData(t *testing.T) {
	chart := &ChartData{
		Type:   "line",
		Title:  "Coverage Trend",
		Labels: []string{"Jan", "Feb", "Mar"},
		Series: []DataSeries{
			{
				Name:   "Coverage %",
				Values: []float64{80.5, 82.0, 85.5},
				Color:  "#3fb950",
			},
		},
	}

	// Verify data consistency
	if len(chart.Labels) != len(chart.Series[0].Values) {
		t.Error("Labels and Values should have the same length")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(chart)
	if err != nil {
		t.Fatalf("Failed to marshal ChartData: %v", err)
	}

	// Verify JSON structure
	var unmarshaled ChartData
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ChartData: %v", err)
	}

	if unmarshaled.Type != chart.Type {
		t.Errorf("Type = %v, want %v", unmarshaled.Type, chart.Type)
	}
	if len(unmarshaled.Series) != len(chart.Series) {
		t.Errorf("Series length = %v, want %v", len(unmarshaled.Series), len(chart.Series))
	}
}
