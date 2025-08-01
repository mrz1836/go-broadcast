package templates

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatCommitSHA(t *testing.T) {
	engine := NewPRTemplateEngine(nil)

	tests := []struct {
		name     string
		sha      string
		expected string
	}{
		{
			name:     "long SHA",
			sha:      "abc123def456789",
			expected: "abc123d",
		},
		{
			name:     "short SHA",
			sha:      "abc123",
			expected: "abc123",
		},
		{
			name:     "empty SHA",
			sha:      "",
			expected: "",
		},
		{
			name:     "exactly 7 chars",
			sha:      "abc1234",
			expected: "abc1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.formatCommitSHA(tt.sha)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	engine := NewPRTemplateEngine(&TemplateConfig{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	testTime := time.Date(2023, 12, 25, 14, 30, 45, 0, time.UTC)
	result := engine.formatTimestamp(testTime)
	require.Equal(t, "2023-12-25 14:30:45", result)
}

func TestStatusEmoji(t *testing.T) {
	tests := []struct {
		name          string
		includeEmojis bool
		status        string
		expected      string
	}{
		{
			name:          "emojis disabled",
			includeEmojis: false,
			status:        "excellent",
			expected:      "",
		},
		{
			name:          "excellent status",
			includeEmojis: true,
			status:        "excellent",
			expected:      "🟢",
		},
		{
			name:          "good status",
			includeEmojis: true,
			status:        "good",
			expected:      "🟡",
		},
		{
			name:          "warning status",
			includeEmojis: true,
			status:        "warning",
			expected:      "🟠",
		},
		{
			name:          "critical status",
			includeEmojis: true,
			status:        "critical",
			expected:      "🔴",
		},
		{
			name:          "default status",
			includeEmojis: true,
			status:        "unknown",
			expected:      "⚪",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				IncludeEmojis: tt.includeEmojis,
			})
			result := engine.statusEmoji(tt.status)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTrendEmoji(t *testing.T) {
	tests := []struct {
		name          string
		includeEmojis bool
		direction     string
		expected      string
	}{
		{
			name:          "emojis disabled",
			includeEmojis: false,
			direction:     "improved",
			expected:      "",
		},
		{
			name:          "improved direction",
			includeEmojis: true,
			direction:     "improved",
			expected:      "📈",
		},
		{
			name:          "up direction",
			includeEmojis: true,
			direction:     "up",
			expected:      "📈",
		},
		{
			name:          "upward direction",
			includeEmojis: true,
			direction:     "upward",
			expected:      "📈",
		},
		{
			name:          "degraded direction",
			includeEmojis: true,
			direction:     "degraded",
			expected:      "📉",
		},
		{
			name:          "down direction",
			includeEmojis: true,
			direction:     "down",
			expected:      "📉",
		},
		{
			name:          "downward direction",
			includeEmojis: true,
			direction:     "downward",
			expected:      "📉",
		},
		{
			name:          "stable direction",
			includeEmojis: true,
			direction:     "stable",
			expected:      "📊",
		},
		{
			name:          "volatile direction",
			includeEmojis: true,
			direction:     "volatile",
			expected:      "📊",
		},
		{
			name:          "unknown direction",
			includeEmojis: true,
			direction:     "unknown",
			expected:      "📊",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				IncludeEmojis: tt.includeEmojis,
			})
			result := engine.trendEmoji(tt.direction)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestRiskEmoji(t *testing.T) {
	tests := []struct {
		name          string
		includeEmojis bool
		risk          string
		expected      string
	}{
		{
			name:          "emojis disabled",
			includeEmojis: false,
			risk:          "high",
			expected:      "",
		},
		{
			name:          "high risk",
			includeEmojis: true,
			risk:          "high",
			expected:      "🚨",
		},
		{
			name:          "critical risk",
			includeEmojis: true,
			risk:          "critical",
			expected:      "🚨",
		},
		{
			name:          "medium risk",
			includeEmojis: true,
			risk:          "medium",
			expected:      "⚠️",
		},
		{
			name:          "low risk",
			includeEmojis: true,
			risk:          "low",
			expected:      "✅",
		},
		{
			name:          "unknown risk",
			includeEmojis: true,
			risk:          "unknown",
			expected:      "ℹ️",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				IncludeEmojis: tt.includeEmojis,
			})
			result := engine.riskEmoji(tt.risk)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGradeEmoji(t *testing.T) {
	tests := []struct {
		name          string
		includeEmojis bool
		grade         string
		expected      string
	}{
		{
			name:          "emojis disabled",
			includeEmojis: false,
			grade:         "A+",
			expected:      "",
		},
		{
			name:          "A+ grade",
			includeEmojis: true,
			grade:         "A+",
			expected:      "🏆",
		},
		{
			name:          "A grade",
			includeEmojis: true,
			grade:         "A",
			expected:      "🥇",
		},
		{
			name:          "B+ grade",
			includeEmojis: true,
			grade:         "B+",
			expected:      "🥈",
		},
		{
			name:          "B grade",
			includeEmojis: true,
			grade:         "B",
			expected:      "🥈",
		},
		{
			name:          "C grade",
			includeEmojis: true,
			grade:         "C",
			expected:      "🥉",
		},
		{
			name:          "D grade",
			includeEmojis: true,
			grade:         "D",
			expected:      "⚠️",
		},
		{
			name:          "F grade",
			includeEmojis: true,
			grade:         "F",
			expected:      "🚨",
		},
		{
			name:          "unknown grade",
			includeEmojis: true,
			grade:         "X",
			expected:      "📊",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				IncludeEmojis: tt.includeEmojis,
			})
			result := engine.gradeEmoji(tt.grade)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPriorityEmoji(t *testing.T) {
	tests := []struct {
		name          string
		includeEmojis bool
		priority      string
		expected      string
	}{
		{
			name:          "emojis disabled",
			includeEmojis: false,
			priority:      "high",
			expected:      "",
		},
		{
			name:          "high priority",
			includeEmojis: true,
			priority:      "high",
			expected:      "🔥",
		},
		{
			name:          "medium priority",
			includeEmojis: true,
			priority:      "medium",
			expected:      "📌",
		},
		{
			name:          "low priority",
			includeEmojis: true,
			priority:      "low",
			expected:      "💡",
		},
		{
			name:          "unknown priority",
			includeEmojis: true,
			priority:      "unknown",
			expected:      "ℹ️",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				IncludeEmojis: tt.includeEmojis,
			})
			result := engine.priorityEmoji(tt.priority)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestProgressBarComprehensive(t *testing.T) {
	tests := []struct {
		name                string
		includeProgressBars bool
		value               float64
		maxValue            float64
		width               int
		expectedContains    []string
	}{
		{
			name:                "progress bars disabled",
			includeProgressBars: false,
			value:               50,
			maxValue:            100,
			width:               10,
			expectedContains:    []string{""},
		},
		{
			name:                "full progress",
			includeProgressBars: true,
			value:               100,
			maxValue:            100,
			width:               10,
			expectedContains:    []string{"██████████", "100.0%"},
		},
		{
			name:                "half progress",
			includeProgressBars: true,
			value:               50,
			maxValue:            100,
			width:               10,
			expectedContains:    []string{"█████░░░░░", "50.0%"},
		},
		{
			name:                "zero progress",
			includeProgressBars: true,
			value:               0,
			maxValue:            100,
			width:               10,
			expectedContains:    []string{"░░░░░░░░░░", "0.0%"},
		},
		{
			name:                "over max value",
			includeProgressBars: true,
			value:               150,
			maxValue:            100,
			width:               10,
			expectedContains:    []string{"██████████", "150.0%"},
		},
		{
			name:                "negative value",
			includeProgressBars: true,
			value:               -10,
			maxValue:            100,
			width:               10,
			expectedContains:    []string{"░░░░░░░░░░", "-10.0%"},
		},
		{
			name:                "zero width defaults to 20",
			includeProgressBars: true,
			value:               50,
			maxValue:            100,
			width:               0,
			expectedContains:    []string{"50.0%"},
		},
		{
			name:                "negative width defaults to 20",
			includeProgressBars: true,
			value:               50,
			maxValue:            100,
			width:               -5,
			expectedContains:    []string{"50.0%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				IncludeProgressBars: tt.includeProgressBars,
			})
			result := engine.progressBar(tt.value, tt.maxValue, tt.width)

			if !tt.includeProgressBars {
				require.Empty(t, result)
				return
			}

			for _, expected := range tt.expectedContains {
				require.Contains(t, result, expected)
			}
		})
	}
}

func TestTrendChart(t *testing.T) {
	tests := []struct {
		name          string
		includeCharts bool
		value         interface{}
		expected      string
	}{
		{
			name:          "charts disabled",
			includeCharts: false,
			value:         95.0,
			expected:      "",
		},
		{
			name:          "single value high",
			includeCharts: true,
			value:         95.0,
			expected:      "📈",
		},
		{
			name:          "single value medium",
			includeCharts: true,
			value:         75.0,
			expected:      "📊",
		},
		{
			name:          "single value low",
			includeCharts: true,
			value:         65.0,
			expected:      "📉",
		},
		{
			name:          "empty slice",
			includeCharts: true,
			value:         []float64{},
			expected:      "",
		},
		{
			name:          "invalid type",
			includeCharts: true,
			value:         "invalid",
			expected:      "",
		},
		{
			name:          "slice of values",
			includeCharts: true,
			value:         []float64{80.0, 85.0, 90.0},
			expected:      "", // Should contain ASCII chart, but we just check it's not empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				IncludeCharts: tt.includeCharts,
			})
			result := engine.trendChart(tt.value)

			if tt.name == "slice of values" && tt.includeCharts {
				require.NotEmpty(t, result)
			} else {
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFilterFiles(t *testing.T) {
	tests := []struct {
		name            string
		hideStableFiles bool
		maxFileChanges  int
		files           []FileCoverageData
		expectedLen     int
		expectedFiles   []string
	}{
		{
			name:            "no filtering",
			hideStableFiles: false,
			maxFileChanges:  10,
			files: []FileCoverageData{
				{Filename: "file1.go", Status: "changed", Change: 5.0},
				{Filename: "file2.go", Status: "stable", Change: 0.5},
				{Filename: "file3.go", Status: "new", Change: 10.0},
			},
			expectedLen:   3,
			expectedFiles: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:            "hide stable files",
			hideStableFiles: true,
			maxFileChanges:  10,
			files: []FileCoverageData{
				{Filename: "file1.go", Status: "changed", Change: 5.0},
				{Filename: "file2.go", Status: "stable", Change: 0.5},
				{Filename: "file3.go", Status: "stable", Change: 2.0},
				{Filename: "file4.go", Status: "new", Change: 10.0},
			},
			expectedLen:   3,
			expectedFiles: []string{"file1.go", "file3.go", "file4.go"},
		},
		{
			name:            "limit max files",
			hideStableFiles: false,
			maxFileChanges:  2,
			files: []FileCoverageData{
				{Filename: "file1.go", Status: "changed", Change: 5.0},
				{Filename: "file2.go", Status: "stable", Change: 0.5},
				{Filename: "file3.go", Status: "new", Change: 10.0},
			},
			expectedLen:   2,
			expectedFiles: []string{"file1.go", "file2.go"},
		},
		{
			name:            "hide stable and limit",
			hideStableFiles: true,
			maxFileChanges:  1,
			files: []FileCoverageData{
				{Filename: "file1.go", Status: "changed", Change: 5.0},
				{Filename: "file2.go", Status: "stable", Change: 0.5},
				{Filename: "file3.go", Status: "new", Change: 10.0},
			},
			expectedLen:   1,
			expectedFiles: []string{"file1.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				HideStableFiles: tt.hideStableFiles,
				MaxFileChanges:  tt.maxFileChanges,
			})

			result := engine.filterFiles(tt.files)
			require.Len(t, result, tt.expectedLen)

			for i, expectedFile := range tt.expectedFiles {
				if i < len(result) {
					require.Equal(t, expectedFile, result[i].Filename)
				}
			}
		})
	}
}

func TestFilterPackages(t *testing.T) {
	tests := []struct {
		name              string
		hideStableFiles   bool
		maxPackageChanges int
		packages          []PackageCoverageData
		expectedLen       int
		expectedPackages  []string
	}{
		{
			name:              "no filtering",
			hideStableFiles:   false,
			maxPackageChanges: 10,
			packages: []PackageCoverageData{
				{Package: "pkg1", Status: "changed", Change: 5.0},
				{Package: "pkg2", Status: "stable", Change: 0.5},
				{Package: "pkg3", Status: "new", Change: 10.0},
			},
			expectedLen:      3,
			expectedPackages: []string{"pkg1", "pkg2", "pkg3"},
		},
		{
			name:              "hide stable packages",
			hideStableFiles:   true,
			maxPackageChanges: 10,
			packages: []PackageCoverageData{
				{Package: "pkg1", Status: "changed", Change: 5.0},
				{Package: "pkg2", Status: "stable", Change: 0.5},
				{Package: "pkg3", Status: "stable", Change: 2.0},
				{Package: "pkg4", Status: "new", Change: 10.0},
			},
			expectedLen:      3,
			expectedPackages: []string{"pkg1", "pkg3", "pkg4"},
		},
		{
			name:              "limit max packages",
			hideStableFiles:   false,
			maxPackageChanges: 2,
			packages: []PackageCoverageData{
				{Package: "pkg1", Status: "changed", Change: 5.0},
				{Package: "pkg2", Status: "stable", Change: 0.5},
				{Package: "pkg3", Status: "new", Change: 10.0},
			},
			expectedLen:      2,
			expectedPackages: []string{"pkg1", "pkg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewPRTemplateEngine(&TemplateConfig{
				HideStableFiles:   tt.hideStableFiles,
				MaxPackageChanges: tt.maxPackageChanges,
			})

			result := engine.filterPackages(tt.packages)
			require.Len(t, result, tt.expectedLen)

			for i, expectedPkg := range tt.expectedPackages {
				if i < len(result) {
					require.Equal(t, expectedPkg, result[i].Package)
				}
			}
		})
	}
}
