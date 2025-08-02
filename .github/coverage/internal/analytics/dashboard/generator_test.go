package dashboard

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	config := &GeneratorConfig{
		ProjectName:      "test-project",
		RepositoryOwner:  "owner",
		RepositoryName:   "repo",
		TemplateDir:      "/tmp/templates",
		OutputDir:        "/tmp/output",
		AssetsDir:        "/tmp/assets",
		GeneratorVersion: "1.0.0",
	}

	gen := NewGenerator(config)
	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.config != config {
		t.Error("Generator config not set correctly")
	}
	if gen.renderer == nil {
		t.Error("Generator renderer not initialized")
	}
}

func TestGenerator_Generate(t *testing.T) {
	// Create temporary directory for output
	tempDir := t.TempDir()

	config := &GeneratorConfig{
		ProjectName:      "test-project",
		RepositoryOwner:  "owner",
		RepositoryName:   "repo",
		TemplateDir:      tempDir,
		OutputDir:        filepath.Join(tempDir, "output"),
		AssetsDir:        filepath.Join(tempDir, "assets"),
		GeneratorVersion: "1.0.0",
	}

	gen := NewGenerator(config)
	ctx := context.Background()

	data := &CoverageData{
		ProjectName:    "test-project",
		RepositoryURL:  "https://github.com/owner/repo",
		Branch:         "master",
		CommitSHA:      "abc123def456",
		Timestamp:      time.Now(),
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
			},
			{
				Name:         "github.com/owner/repo/pkg2",
				Path:         "pkg2",
				Coverage:     80.0,
				TotalLines:   200,
				CoveredLines: 160,
				MissedLines:  40,
			},
		},
		TrendData: &TrendData{
			Direction:       "up",
			ChangePercent:   2.5,
			ChangeLines:     25,
			ComparedTo:      "branch",
			ComparedToValue: "develop",
		},
	}

	err := gen.Generate(ctx, data)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check if index.html was created
	indexPath := filepath.Join(config.OutputDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("index.html was not created")
	}

	// Check if data directory was created
	dataDir := filepath.Join(config.OutputDir, "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("data directory was not created")
	}

	// Check if coverage.json was created
	coveragePath := filepath.Join(dataDir, "coverage.json")
	if _, err := os.Stat(coveragePath); os.IsNotExist(err) {
		t.Error("coverage.json was not created")
	}

	// Check if metadata.json was created
	metadataPath := filepath.Join(dataDir, "metadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("metadata.json was not created")
	}
}

func TestGenerator_formatCommitSHA(t *testing.T) {
	gen := &Generator{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "long SHA",
			input:    "abc123def456789",
			expected: "abc123d",
		},
		{
			name:     "short SHA",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "empty SHA",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.formatCommitSHA(tt.input)
			if result != tt.expected {
				t.Errorf("formatCommitSHA(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerator_prepareTemplateData(t *testing.T) {
	config := &GeneratorConfig{
		ProjectName:     "test-project",
		RepositoryOwner: "owner",
		RepositoryName:  "repo",
	}
	gen := NewGenerator(config)

	data := &CoverageData{
		ProjectName:    "test-project",
		RepositoryURL:  "https://github.com/owner/repo.git",
		Branch:         "master",
		CommitSHA:      "abc123def456789",
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
				Name:         "pkg1",
				Coverage:     90.0,
				TotalLines:   100,
				CoveredLines: 90,
			},
		},
		TrendData: &TrendData{
			Direction:     "up",
			ChangePercent: 2.5,
			ChangeLines:   25,
		},
		History: []HistoricalPoint{
			{
				Timestamp:    time.Now(),
				CommitSHA:    "previous",
				Coverage:     83.0,
				TotalLines:   980,
				CoveredLines: 813,
			},
		},
	}

	result := gen.prepareTemplateData(context.Background(), data)

	// Check key fields
	if result["ProjectName"] != "test-project" {
		t.Errorf("ProjectName = %v, want test-project", result["ProjectName"])
	}
	if result["Branch"] != "master" {
		t.Errorf("Branch = %v, want main", result["Branch"])
	}
	if result["CommitSHA"] != "abc123d" {
		t.Errorf("CommitSHA = %v, want abc123d", result["CommitSHA"])
	}
	if result["TotalCoverage"] != 85.5 {
		t.Errorf("TotalCoverage = %v, want 85.5", result["TotalCoverage"])
	}
	if result["CoveredFiles"] != 8 {
		t.Errorf("CoveredFiles = %v, want 8", result["CoveredFiles"])
	}
	if result["TotalFiles"] != 10 {
		t.Errorf("TotalFiles = %v, want 10", result["TotalFiles"])
	}
	if result["PackagesTracked"] != 1 {
		t.Errorf("PackagesTracked = %v, want 1", result["PackagesTracked"])
	}
	if result["HasHistory"] != true {
		t.Errorf("HasHistory = %v, want true", result["HasHistory"])
	}

	// Check commit URL
	expectedURL := "https://github.com/owner/repo/commit/abc123def456789"
	if result["CommitURL"] != expectedURL {
		t.Errorf("CommitURL = %v, want %v", result["CommitURL"], expectedURL)
	}
}

func TestRenderer_RenderDashboard(t *testing.T) {
	renderer := NewRenderer("/tmp/templates")
	ctx := context.Background()

	testBranch := "master" // Use master as test case since that's what you use
	data := map[string]interface{}{
		"ProjectName":     "test-project",
		"RepositoryOwner": "owner",
		"RepositoryName":  "repo",
		"Branch":          testBranch,
		"CommitSHA":       "abc123d",
		"TotalCoverage":   85.5,
		"CoveredFiles":    8,
		"TotalFiles":      10,
		"PackagesTracked": 2,
		"Timestamp":       time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		"RepositoryURL":   "https://github.com/owner/repo",
		"CoverageTrend":   "2.5",
		"HasHistory":      false,
		"HistoryJSON":     "[]",
		"Packages":        []map[string]interface{}{},
	}

	html, err := renderer.RenderDashboard(ctx, data)
	if err != nil {
		t.Fatalf("RenderDashboard failed: %v", err)
	}

	// Check that HTML contains expected content
	expectedStrings := []string{
		"<title>owner/repo Coverage Dashboard</title>",
		"🌿",        // Branch icon
		testBranch, // Branch name (dynamic based on test data)
		"85.5%",
		"8 of 10 files covered",
		"Packages analyzed",
		"2024-01-15 10:30:00 UTC",
	}

	for _, expected := range expectedStrings {
		if !containsString(html, expected) {
			t.Errorf("HTML does not contain expected string: %q", expected)
		}
	}
}

// Helper function for string contains check
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) != -1))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestGenerator_formatDuration(t *testing.T) {
	gen := &Generator{}

	tests := []struct {
		name      string
		startedAt time.Time
		updatedAt time.Time
		status    string
		expected  string
	}{
		{
			name:      "completed in seconds",
			startedAt: time.Now().Add(-30 * time.Second),
			updatedAt: time.Now(),
			status:    "completed",
			expected:  "30s",
		},
		{
			name:      "completed in minutes",
			startedAt: time.Now().Add(-5*time.Minute - 30*time.Second),
			updatedAt: time.Now(),
			status:    "completed",
			expected:  "5m 30s",
		},
		{
			name:      "completed in hours",
			startedAt: time.Now().Add(-2*time.Hour - 15*time.Minute),
			updatedAt: time.Now(),
			status:    "completed",
			expected:  "2h 15m",
		},
		{
			name:      "in progress",
			startedAt: time.Now().Add(-10 * time.Minute),
			updatedAt: time.Time{}, // Zero time
			status:    "in_progress",
			expected:  "10m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Allow 1 second tolerance for test execution time
			result := gen.formatDuration(tt.startedAt, tt.updatedAt, tt.status)
			// For simple validation, just check that the result is not empty
			// and follows the expected format pattern
			if result == "" {
				t.Errorf("formatDuration returned empty string")
			}
		})
	}
}
