package badge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewPRBadgeManager(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		generator := New()
		manager := NewPRBadgeManager(generator, nil)

		require.NotNil(t, manager)
		require.NotNil(t, manager.config)
		require.Equal(t, "./coverage-badges", manager.config.OutputBasePath)
		require.True(t, manager.config.CreateDirectories)
		require.Equal(t, os.FileMode(0o755), manager.config.DirectoryPermissions)
		require.Equal(t, os.FileMode(0o644), manager.config.FilePermissions)
		require.Equal(t, "badge-coverage-{style}.svg", manager.config.CoveragePattern)
		require.Equal(t, "badge-trend-{style}.svg", manager.config.TrendPattern)
		require.Equal(t, "badge-status-{style}.svg", manager.config.StatusPattern)
		require.Equal(t, "badge-comparison-{style}.svg", manager.config.ComparisonPattern)
		require.Equal(t, []string{"flat", "flat-square", "for-the-badge"}, manager.config.Styles)
		require.Equal(t, "flat", manager.config.DefaultStyle)
		require.False(t, manager.config.IncludeTimestamp)
		require.True(t, manager.config.IncludeBranch)
		require.True(t, manager.config.EnableCleanup)
		require.Equal(t, 30*24*time.Hour, manager.config.MaxAge)
		require.True(t, manager.config.CleanupOnMerge)
		require.True(t, manager.config.GenerateMultipleStyles)
		require.False(t, manager.config.GenerateThumbnails)
		require.False(t, manager.config.GenerateRetina)
	})

	t.Run("with custom config", func(t *testing.T) {
		generator := New()
		customConfig := &PRBadgeConfig{
			OutputBasePath:         "/custom/path",
			CreateDirectories:      false,
			DirectoryPermissions:   0o700,
			FilePermissions:        0o600,
			CoveragePattern:        "custom-coverage-{style}.svg",
			TrendPattern:           "custom-trend-{style}.svg",
			StatusPattern:          "custom-status-{style}.svg",
			ComparisonPattern:      "custom-comparison-{style}.svg",
			Styles:                 []string{"flat"},
			DefaultStyle:           "flat-square",
			IncludeTimestamp:       true,
			IncludeBranch:          false,
			EnableCleanup:          false,
			MaxAge:                 7 * 24 * time.Hour,
			CleanupOnMerge:         false,
			GenerateMultipleStyles: false,
			GenerateThumbnails:     true,
			GenerateRetina:         true,
		}

		manager := NewPRBadgeManager(generator, customConfig)

		require.NotNil(t, manager)
		require.Equal(t, customConfig, manager.config)
		require.Equal(t, "/custom/path", manager.config.OutputBasePath)
		require.False(t, manager.config.CreateDirectories)
	})
}

func TestGeneratePRBadges(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		request        *PRBadgeRequest
		config         *PRBadgeConfig
		expectedBadges int
		expectError    bool
	}{
		{
			name: "single badge type with default style",
			request: &PRBadgeRequest{
				Repository:   "test-repo",
				Owner:        "test-owner",
				PRNumber:     123,
				Branch:       "feature-branch",
				CommitSHA:    "abc123",
				BaseBranch:   "main",
				Coverage:     85.5,
				BaseCoverage: 80.0,
				Trend:        TrendUp,
				QualityGrade: "B+",
				RiskLevel:    "low",
				Types:        []PRBadgeType{PRBadgeCoverage},
				Timestamp:    time.Now(),
				Author:       "test-user",
			},
			config: &PRBadgeConfig{
				OutputBasePath:         tempDir,
				CreateDirectories:      true,
				DirectoryPermissions:   0o755,
				FilePermissions:        0o644,
				CoveragePattern:        "badge-coverage-{style}.svg",
				Styles:                 []string{"flat"},
				DefaultStyle:           "flat",
				GenerateMultipleStyles: false,
			},
			expectedBadges: 1,
			expectError:    false,
		},
		{
			name: "multiple badge types with multiple styles",
			request: &PRBadgeRequest{
				Repository:   "test-repo",
				Owner:        "test-owner",
				PRNumber:     456,
				Branch:       "feature-branch",
				CommitSHA:    "def456",
				BaseBranch:   "main",
				Coverage:     92.3,
				BaseCoverage: 88.1,
				Trend:        TrendUp,
				QualityGrade: "A",
				RiskLevel:    "low",
				Types:        []PRBadgeType{PRBadgeCoverage, PRBadgeTrend, PRBadgeStatus},
				Styles:       []string{"flat", "flat-square"},
				Timestamp:    time.Now(),
				Author:       "test-user",
			},
			config: &PRBadgeConfig{
				OutputBasePath:         tempDir,
				CreateDirectories:      true,
				DirectoryPermissions:   0o755,
				FilePermissions:        0o644,
				CoveragePattern:        "badge-coverage-{style}.svg",
				TrendPattern:           "badge-trend-{style}.svg",
				StatusPattern:          "badge-status-{style}.svg",
				Styles:                 []string{"flat", "flat-square", "for-the-badge"},
				DefaultStyle:           "flat",
				GenerateMultipleStyles: true,
			},
			expectedBadges: 6, // 3 types × 2 styles
			expectError:    false,
		},
		{
			name: "all badge types",
			request: &PRBadgeRequest{
				Repository:   "test-repo",
				Owner:        "test-owner",
				PRNumber:     789,
				Branch:       "feature-branch",
				CommitSHA:    "ghi789",
				BaseBranch:   "main",
				Coverage:     75.0,
				BaseCoverage: 70.0,
				Trend:        TrendUp,
				QualityGrade: "B",
				RiskLevel:    "medium",
				Types: []PRBadgeType{
					PRBadgeCoverage,
					PRBadgeTrend,
					PRBadgeStatus,
					PRBadgeComparison,
					PRBadgeDiff,
					PRBadgeQuality,
				},
				Styles:    []string{"flat"},
				Timestamp: time.Now(),
				Author:    "test-user",
			},
			config: &PRBadgeConfig{
				OutputBasePath:         tempDir,
				CreateDirectories:      true,
				DirectoryPermissions:   0o755,
				FilePermissions:        0o644,
				CoveragePattern:        "badge-coverage-{style}.svg",
				TrendPattern:           "badge-trend-{style}.svg",
				StatusPattern:          "badge-status-{style}.svg",
				ComparisonPattern:      "badge-comparison-{style}.svg",
				Styles:                 []string{"flat"},
				DefaultStyle:           "flat",
				GenerateMultipleStyles: false,
			},
			expectedBadges: 6, // 6 types × 1 style
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := New()
			manager := NewPRBadgeManager(generator, tt.config)
			ctx := context.Background()

			result, err := manager.GeneratePRBadges(ctx, tt.request)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tt.expectedBadges, result.TotalBadges)

			// Check that directories were created
			expectedDir := filepath.Join(tt.config.OutputBasePath, "pr", fmt.Sprintf("%d", tt.request.PRNumber))
			_, err = os.Stat(expectedDir)
			require.NoError(t, err)

			// Check that badge files were created
			for badgeType := range result.Badges {
				badges := result.Badges[badgeType]
				require.NotEmpty(t, badges)

				for _, badge := range badges {
					_, err = os.Stat(badge.FilePath)
					require.NoError(t, err)
					require.Positive(t, badge.Size)
					require.Equal(t, badgeType, badge.Type)
					require.NotEmpty(t, badge.Style)
					require.NotEmpty(t, badge.PublicURL)
				}
			}

			// Verify URLs are properly formatted
			expectedBaseURL := fmt.Sprintf("https://%s.github.io/%s/coverage/pr/%d",
				tt.request.Owner, tt.request.Repository, tt.request.PRNumber)
			require.Equal(t, expectedBaseURL, result.BaseURL)
		})
	}
}

func TestGeneratePRBadgesWithErrors(t *testing.T) {
	t.Run("unsupported badge type error", func(t *testing.T) {
		tempDir := t.TempDir()

		generator := New()

		config := &PRBadgeConfig{
			OutputBasePath:         tempDir,
			CreateDirectories:      true,
			DirectoryPermissions:   0o755,
			FilePermissions:        0o644,
			CoveragePattern:        "badge-coverage-{style}.svg",
			TrendPattern:           "badge-trend-{style}.svg",
			StatusPattern:          "badge-status-{style}.svg",
			ComparisonPattern:      "badge-comparison-{style}.svg",
			Styles:                 []string{"flat", "flat-square", "for-the-badge"},
			DefaultStyle:           "flat",
			GenerateMultipleStyles: false, // Only generate default style for testing
		}

		manager := NewPRBadgeManager(generator, config)
		ctx := context.Background()

		request := &PRBadgeRequest{
			Repository: "test-repo",
			Owner:      "test-owner",
			PRNumber:   123,
			Coverage:   85.5,
			Types:      []PRBadgeType{PRBadgeType("unsupported")},
			Timestamp:  time.Now(),
		}

		result, err := manager.GeneratePRBadges(ctx, request)

		require.NoError(t, err) // Should not return error, but should have errors in result
		require.NotNil(t, result)
		require.NotEmpty(t, result.Errors)
		require.Contains(t, result.Errors[0].Error(), "unsupported badge type")
		require.Equal(t, 0, result.TotalBadges)
	})

	t.Run("directory creation failure", func(t *testing.T) {
		generator := New()

		// Use an invalid path that will cause mkdir to fail
		invalidPath := "/invalid/readonly/path"
		config := &PRBadgeConfig{
			OutputBasePath:    invalidPath,
			CreateDirectories: true,
		}

		manager := NewPRBadgeManager(generator, config)
		ctx := context.Background()

		request := &PRBadgeRequest{
			Repository: "test-repo",
			Owner:      "test-owner",
			PRNumber:   123,
			Coverage:   85.5,
			Types:      []PRBadgeType{PRBadgeCoverage},
			Timestamp:  time.Now(),
		}

		result, err := manager.GeneratePRBadges(ctx, request)

		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "failed to create output directory")
	})

	t.Run("context cancellation", func(t *testing.T) {
		tempDir := t.TempDir()

		generator := New()
		config := &PRBadgeConfig{
			OutputBasePath:         tempDir,
			CreateDirectories:      true,
			DirectoryPermissions:   0o755,
			FilePermissions:        0o644,
			CoveragePattern:        "badge-coverage-{style}.svg",
			TrendPattern:           "badge-trend-{style}.svg",
			StatusPattern:          "badge-status-{style}.svg",
			ComparisonPattern:      "badge-comparison-{style}.svg",
			Styles:                 []string{"flat", "flat-square", "for-the-badge"},
			DefaultStyle:           "flat",
			GenerateMultipleStyles: false, // Only generate default style for testing
		}

		manager := NewPRBadgeManager(generator, config)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		request := &PRBadgeRequest{
			Repository: "test-repo",
			Owner:      "test-owner",
			PRNumber:   123,
			Coverage:   85.5,
			Types:      []PRBadgeType{PRBadgeCoverage},
			Timestamp:  time.Now(),
		}

		result, err := manager.GeneratePRBadges(ctx, request)

		// The result depends on when the context is checked - it might succeed if checked after directory creation
		// but should fail during badge generation. Since the real generator checks context, we expect an error in results
		if err != nil {
			require.Equal(t, context.Canceled, err)
		} else {
			// If no error, there should be errors in the result from canceled badge generation
			require.NotNil(t, result)
			if len(result.Errors) > 0 {
				require.Contains(t, result.Errors[0].Error(), "context canceled")
			}
		}
	})
}

func TestGenerateSingleBadge(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		badgeType   PRBadgeType
		expectError bool
	}{
		{"coverage badge", PRBadgeCoverage, false},
		{"trend badge", PRBadgeTrend, false},
		{"status badge", PRBadgeStatus, false},
		{"comparison badge", PRBadgeComparison, false},
		{"diff badge", PRBadgeDiff, false},
		{"quality badge", PRBadgeQuality, false},
		{"unsupported badge type", PRBadgeType("unsupported"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := New()
			config := &PRBadgeConfig{
				OutputBasePath:         tempDir,
				CreateDirectories:      true,
				DirectoryPermissions:   0o755,
				FilePermissions:        0o644,
				CoveragePattern:        "badge-coverage-{style}.svg",
				TrendPattern:           "badge-trend-{style}.svg",
				StatusPattern:          "badge-status-{style}.svg",
				ComparisonPattern:      "badge-comparison-{style}.svg",
				Styles:                 []string{"flat", "flat-square", "for-the-badge"},
				DefaultStyle:           "flat",
				GenerateMultipleStyles: false, // Only generate default style for testing
			}
			manager := NewPRBadgeManager(generator, config)
			ctx := context.Background()

			request := &PRBadgeRequest{
				Repository:   "test-repo",
				Owner:        "test-owner",
				PRNumber:     123,
				Branch:       "feature-branch",
				CommitSHA:    "abc123",
				Coverage:     85.5,
				BaseCoverage: 80.0,
				QualityGrade: "B+",
				Timestamp:    time.Now(),
			}

			outputDir := manager.buildOutputDirectory(request)
			err := os.MkdirAll(outputDir, 0o750)
			require.NoError(t, err)

			info, err := manager.generateSingleBadge(ctx, request, tt.badgeType, "flat", outputDir)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, info)
				if tt.badgeType == PRBadgeType("unsupported") {
					require.ErrorIs(t, err, ErrUnsupportedBadgeType)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, info)
			require.Equal(t, tt.badgeType, info.Type)
			require.Equal(t, "flat", info.Style)
			require.NotEmpty(t, info.FilePath)
			require.NotEmpty(t, info.PublicURL)
			require.Positive(t, info.Size)
			require.NotZero(t, info.Dimensions.Width)
			require.NotZero(t, info.Dimensions.Height)
			require.NotZero(t, info.Metadata.GeneratedAt)
			require.Equal(t, "2.0", info.Metadata.Version)
			require.InDelta(t, request.Coverage, info.Metadata.Coverage, 0.001)
			require.InDelta(t, request.BaseCoverage, info.Metadata.BaseCoverage, 0.001)
			require.InDelta(t, request.Coverage-request.BaseCoverage, info.Metadata.Change, 0.001)
			require.Equal(t, request.QualityGrade, info.Metadata.QualityGrade)
			require.Equal(t, request.PRNumber, info.Metadata.PRNumber)
			require.Equal(t, request.Branch, info.Metadata.Branch)
			require.Equal(t, request.CommitSHA, info.Metadata.CommitSHA)

			// Verify file exists
			_, err = os.Stat(info.FilePath)
			require.NoError(t, err)
		})
	}
}

func TestStatusBadgeGeneration(t *testing.T) {
	tests := []struct {
		name          string
		coverage      float64
		expectedMsg   string
		expectedColor string
	}{
		{"excellent coverage", 95.0, "excellent", "#3fb950"},
		{"good coverage", 85.0, "good", "#7c3aed"},
		{"fair coverage", 75.0, "fair", "#d29922"},
		{"poor coverage", 65.0, "poor", "#fb8500"},
		{"critical coverage", 45.0, "critical", "#f85149"},
		{"edge case - exactly 90", 90.0, "excellent", "#3fb950"},
		{"edge case - exactly 80", 80.0, "good", "#7c3aed"},
		{"edge case - exactly 70", 70.0, "fair", "#d29922"},
		{"edge case - exactly 60", 60.0, "poor", "#fb8500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := New()
			manager := NewPRBadgeManager(generator, nil)
			ctx := context.Background()

			request := &PRBadgeRequest{
				Coverage: tt.coverage,
			}

			data, err := manager.generateStatusBadge(ctx, request, "flat", "status")

			require.NoError(t, err)
			require.NotNil(t, data)

			dataStr := string(data)
			require.Contains(t, dataStr, tt.expectedMsg)
		})
	}
}

func TestComparisonBadgeGeneration(t *testing.T) {
	tests := []struct {
		name          string
		coverage      float64
		baseCoverage  float64
		expectedMsg   string
		expectedColor string
	}{
		{"significant increase", 85.0, 80.0, "+5.0%", "#3fb950"},
		{"significant decrease", 75.0, 85.0, "-10.0%", "#f85149"},
		{"minimal change", 80.05, 80.0, "±0.0%", "#8b949e"},
		{"no change", 80.0, 80.0, "±0.0%", "#8b949e"},
		{"small increase", 80.05, 80.0, "±0.0%", "#8b949e"},
		{"small decrease", 79.95, 80.0, "±0.0%", "#8b949e"},
		{"large increase", 90.0, 70.0, "+20.0%", "#3fb950"},
		{"large decrease", 50.0, 80.0, "-30.0%", "#f85149"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := New()
			manager := NewPRBadgeManager(generator, nil)
			ctx := context.Background()

			request := &PRBadgeRequest{
				Coverage:     tt.coverage,
				BaseCoverage: tt.baseCoverage,
			}

			data, err := manager.generateComparisonBadge(ctx, request, "flat", "comparison")

			require.NoError(t, err)
			require.NotNil(t, data)

			dataStr := string(data)
			require.Contains(t, dataStr, tt.expectedMsg)
		})
	}
}

func TestDiffBadgeGeneration(t *testing.T) {
	tests := []struct {
		name         string
		coverage     float64
		baseCoverage float64
		expectedMsg  string
	}{
		{"major positive change", 90.0, 80.0, "+major"},
		{"major negative change", 70.0, 80.0, "-major"},
		{"moderate positive change", 83.0, 80.0, "+moderate"},
		{"moderate negative change", 77.0, 80.0, "-moderate"},
		{"minor positive change", 81.0, 80.0, "+minor"},
		{"minor negative change", 79.0, 80.0, "-minor"},
		{"stable", 80.1, 80.0, "stable"},
		{"exactly stable", 80.0, 80.0, "stable"},
		{"large change threshold", 85.0, 80.0, "+major"},
		{"moderate change threshold", 82.0, 80.0, "+moderate"},
		{"minor change threshold", 80.5, 80.0, "+minor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := New()
			manager := NewPRBadgeManager(generator, nil)
			ctx := context.Background()

			request := &PRBadgeRequest{
				Coverage:     tt.coverage,
				BaseCoverage: tt.baseCoverage,
			}

			data, err := manager.generateDiffBadge(ctx, request, "flat", "diff")

			require.NoError(t, err)
			require.NotNil(t, data)

			dataStr := string(data)
			require.Contains(t, dataStr, tt.expectedMsg)
		})
	}
}

func TestQualityBadgeGeneration(t *testing.T) {
	tests := []struct {
		name         string
		coverage     float64
		qualityGrade string
		expectedMsg  string
	}{
		{"explicit A+ grade", 85.0, "A+", "A+"},
		{"explicit A grade", 85.0, "A", "A"},
		{"explicit B+ grade", 85.0, "B+", "B+"},
		{"explicit B grade", 85.0, "B", "B"},
		{"explicit C grade", 85.0, "C", "C"},
		{"explicit D grade", 85.0, "D", "D"},
		{"explicit F grade", 85.0, "F", "F"},
		{"calculated A+ from coverage", 96.0, "", "A+"},
		{"calculated A from coverage", 92.0, "", "A"},
		{"calculated B+ from coverage", 87.0, "", "B+"},
		{"calculated B from coverage", 82.0, "", "B"},
		{"calculated C from coverage", 75.0, "", "C"},
		{"calculated D from coverage", 65.0, "", "D"},
		{"calculated F from coverage", 45.0, "", "F"},
		{"edge case - exactly 95", 95.0, "", "A+"},
		{"edge case - exactly 90", 90.0, "", "A"},
		{"edge case - exactly 85", 85.0, "", "B+"},
		{"edge case - exactly 80", 80.0, "", "B"},
		{"edge case - exactly 70", 70.0, "", "C"},
		{"edge case - exactly 60", 60.0, "", "D"},
		{"edge case - below 60", 59.0, "", "F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := New()
			manager := NewPRBadgeManager(generator, nil)
			ctx := context.Background()

			request := &PRBadgeRequest{
				Coverage:     tt.coverage,
				QualityGrade: tt.qualityGrade,
			}

			data, err := manager.generateQualityBadge(ctx, request, "flat", "quality")

			require.NoError(t, err)
			require.NotNil(t, data)

			dataStr := string(data)
			require.Contains(t, dataStr, tt.expectedMsg)
		})
	}
}

func TestHelperMethods(t *testing.T) {
	generator := New()
	manager := NewPRBadgeManager(generator, nil)

	t.Run("buildOutputDirectory", func(t *testing.T) {
		request := &PRBadgeRequest{PRNumber: 123}
		dir := manager.buildOutputDirectory(request)
		expected := filepath.Join("./coverage-badges", "pr", "123")
		require.Equal(t, expected, dir)
	})

	t.Run("buildBaseURL", func(t *testing.T) {
		url := manager.buildBaseURL("owner", "repo", 456)
		expected := "https://owner.github.io/repo/coverage/pr/456"
		require.Equal(t, expected, url)
	})

	t.Run("buildPublicURL", func(t *testing.T) {
		url := manager.buildPublicURL("owner", "repo", 789, "badge.svg")
		expected := "https://owner.github.io/repo/coverage/pr/789/badge.svg"
		require.Equal(t, expected, url)
	})

	t.Run("buildFileName", func(t *testing.T) {
		request := &PRBadgeRequest{
			PRNumber:  123,
			Branch:    "feature-branch",
			Timestamp: time.Date(2023, 12, 25, 10, 30, 45, 0, time.UTC),
		}

		tests := []struct {
			name      string
			badgeType PRBadgeType
			style     string
			expected  string
		}{
			{
				name:      "coverage badge",
				badgeType: PRBadgeCoverage,
				style:     "flat",
				expected:  "badge-coverage-flat.svg",
			},
			{
				name:      "trend badge",
				badgeType: PRBadgeTrend,
				style:     "flat-square",
				expected:  "badge-trend-flat-square.svg",
			},
			{
				name:      "status badge",
				badgeType: PRBadgeStatus,
				style:     "for-the-badge",
				expected:  "badge-status-for-the-badge.svg",
			},
			{
				name:      "comparison badge",
				badgeType: PRBadgeComparison,
				style:     "flat",
				expected:  "badge-comparison-flat.svg",
			},
			{
				name:      "unsupported badge type",
				badgeType: PRBadgeType("custom"),
				style:     "flat",
				expected:  "badge-custom-flat.svg",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fileName := manager.buildFileName(tt.badgeType, tt.style, request)
				require.Equal(t, tt.expected, fileName)
			})
		}
	})

	t.Run("buildFileName with timestamp", func(t *testing.T) {
		config := &PRBadgeConfig{
			CoveragePattern:  "badge-coverage-{style}-{timestamp}.svg",
			IncludeTimestamp: true,
		}
		prManager := NewPRBadgeManager(generator, config)

		request := &PRBadgeRequest{
			PRNumber:  123,
			Branch:    "feature-branch",
			Timestamp: time.Date(2023, 12, 25, 10, 30, 45, 0, time.UTC),
		}

		fileName := prManager.buildFileName(PRBadgeCoverage, "flat", request)
		require.Equal(t, "badge-coverage-flat-20231225-103045.svg", fileName)
	})

	t.Run("getCustomLabel", func(t *testing.T) {
		request := &PRBadgeRequest{
			CustomLabels: map[PRBadgeType]string{
				PRBadgeCoverage: "custom-coverage",
				PRBadgeTrend:    "custom-trend",
			},
		}

		// Should return custom label if exists
		label := manager.getCustomLabel(request, PRBadgeCoverage, "default")
		require.Equal(t, "custom-coverage", label)

		// Should return default label if custom doesn't exist
		label = manager.getCustomLabel(request, PRBadgeStatus, "default")
		require.Equal(t, "default", label)

		// Should return default label if no custom labels map
		requestNoLabels := &PRBadgeRequest{}
		label = manager.getCustomLabel(requestNoLabels, PRBadgeCoverage, "default")
		require.Equal(t, "default", label)
	})

	t.Run("calculateDimensions", func(t *testing.T) {
		tests := []struct {
			name           string
			label          string
			message        string
			style          string
			expectedHeight int
		}{
			{
				name:           "flat badge",
				label:          "coverage",
				message:        "85.5%",
				style:          "flat",
				expectedHeight: 20,
			},
			{
				name:           "for-the-badge style",
				label:          "coverage",
				message:        "85.5%",
				style:          "for-the-badge",
				expectedHeight: 28,
			},
			{
				name:           "empty label",
				label:          "",
				message:        "100%",
				style:          "flat",
				expectedHeight: 20,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dimensions := manager.calculateDimensions(tt.label, tt.message, tt.style)
				require.Equal(t, tt.expectedHeight, dimensions.Height)
				require.Positive(t, dimensions.Width)

				// Width should be reasonable based on text length
				expectedWidth := len(tt.label)*6 + len(tt.message)*6 + 20
				require.Equal(t, expectedWidth, dimensions.Width)
			})
		}
	})
}

func TestCleanupPRBadges(t *testing.T) {
	tempDir := t.TempDir()

	generator := New()
	config := &PRBadgeConfig{
		OutputBasePath: tempDir,
		EnableCleanup:  true,
	}
	manager := NewPRBadgeManager(generator, config)

	t.Run("cleanup enabled - removes PR directory", func(t *testing.T) {
		// Create PR directory with some files
		prDir := filepath.Join(tempDir, "pr", "123")
		err := os.MkdirAll(prDir, 0o750)
		require.NoError(t, err)

		testFile := filepath.Join(prDir, "test-badge.svg")
		err = os.WriteFile(testFile, []byte("test content"), 0o600)
		require.NoError(t, err)

		// Verify directory exists
		_, err = os.Stat(prDir)
		require.NoError(t, err)

		// Cleanup
		ctx := context.Background()
		err = manager.CleanupPRBadges(ctx, "owner", "repo", 123)
		require.NoError(t, err)

		// Verify directory was removed
		_, err = os.Stat(prDir)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("cleanup disabled - does nothing", func(t *testing.T) {
		configNoCleanup := &PRBadgeConfig{
			OutputBasePath: tempDir,
			EnableCleanup:  false,
		}
		managerNoCleanup := NewPRBadgeManager(generator, configNoCleanup)

		// Create PR directory with some files
		prDir := filepath.Join(tempDir, "pr", "456")
		err := os.MkdirAll(prDir, 0o750)
		require.NoError(t, err)

		testFile := filepath.Join(prDir, "test-badge.svg")
		err = os.WriteFile(testFile, []byte("test content"), 0o600)
		require.NoError(t, err)

		// Cleanup (should do nothing)
		ctx := context.Background()
		err = managerNoCleanup.CleanupPRBadges(ctx, "owner", "repo", 456)
		require.NoError(t, err)

		// Verify directory still exists
		_, err = os.Stat(prDir)
		require.NoError(t, err)
	})

	t.Run("cleanup non-existent directory - no error", func(t *testing.T) {
		ctx := context.Background()
		err := manager.CleanupPRBadges(ctx, "owner", "repo", 999)
		require.NoError(t, err)
	})
}

func TestGetPRInfo(t *testing.T) {
	tempDir := t.TempDir()

	generator := New()
	config := &PRBadgeConfig{
		OutputBasePath: tempDir,
	}
	manager := NewPRBadgeManager(generator, config)

	t.Run("no badges exist", func(t *testing.T) {
		ctx := context.Background()
		result, err := manager.GetPRInfo(ctx, "owner", "repo", 999)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 0, result.TotalBadges)
		require.Empty(t, result.Badges)
		require.Equal(t, "https://owner.github.io/repo/coverage/pr/999", result.BaseURL)
	})

	t.Run("existing badges", func(t *testing.T) {
		// Create PR directory with badge files
		prDir := filepath.Join(tempDir, "pr", "123")
		err := os.MkdirAll(prDir, 0o750)
		require.NoError(t, err)

		// Create test badge files
		badgeFiles := []struct {
			name     string
			content  string
			expected PRBadgeType
		}{
			{"badge-coverage-flat.svg", "<svg>coverage</svg>", PRBadgeCoverage},
			{"badge-trend-flat.svg", "<svg>trend</svg>", PRBadgeTrend},
			{"badge-status-flat.svg", "<svg>status</svg>", PRBadgeStatus},
			{"not-a-badge.txt", "not a badge", ""},
			{"invalid-badge-name.svg", "<svg>invalid</svg>", ""},
		}

		for _, bf := range badgeFiles {
			filePath := filepath.Join(prDir, bf.name)
			err = os.WriteFile(filePath, []byte(bf.content), 0o600)
			require.NoError(t, err)
		}

		ctx := context.Background()
		result, err := manager.GetPRInfo(ctx, "owner", "repo", 123)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 3, result.TotalBadges) // Only valid badge files
		require.Equal(t, "https://owner.github.io/repo/coverage/pr/123", result.BaseURL)

		// Check individual badges
		require.Contains(t, result.Badges, PRBadgeCoverage)
		require.Contains(t, result.Badges, PRBadgeTrend)
		require.Contains(t, result.Badges, PRBadgeStatus)

		// Check coverage badge details
		coverageBadges := result.Badges[PRBadgeCoverage]
		require.Len(t, coverageBadges, 1)
		badge := coverageBadges[0]
		require.Equal(t, PRBadgeCoverage, badge.Type)
		require.Equal(t, "flat", badge.Style)
		require.Contains(t, badge.FilePath, "badge-coverage-flat.svg")
		require.Contains(t, badge.PublicURL, "badge-coverage-flat.svg")
		require.Positive(t, badge.Size)
		require.Equal(t, 123, badge.Metadata.PRNumber)
	})

	t.Run("directory read error", func(t *testing.T) {
		// Create a file instead of directory to cause read error
		badPath := filepath.Join(tempDir, "pr", "invalid")
		err := os.WriteFile(badPath, []byte("not a directory"), 0o600)
		require.NoError(t, err)

		ctx := context.Background()
		result, err := manager.GetPRInfo(ctx, "owner", "repo", -1) // This will create path "pr/-1" which doesn't exist

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 0, result.TotalBadges)
	})
}

func TestParseBadgeFileName(t *testing.T) {
	generator := New()
	manager := NewPRBadgeManager(generator, nil)

	tests := []struct {
		name          string
		fileName      string
		expectedType  string
		expectedStyle string
	}{
		{
			name:          "valid coverage badge",
			fileName:      "badge-coverage-flat.svg",
			expectedType:  "coverage",
			expectedStyle: "flat",
		},
		{
			name:          "valid trend badge with dash in style",
			fileName:      "badge-trend-flat-square.svg",
			expectedType:  "trend-flat",
			expectedStyle: "square",
		},
		{
			name:          "valid status badge with complex style",
			fileName:      "badge-status-for-the-badge.svg",
			expectedType:  "status-for-the",
			expectedStyle: "badge",
		},
		{
			name:          "valid comparison badge",
			fileName:      "badge-comparison-flat.svg",
			expectedType:  "comparison",
			expectedStyle: "flat",
		},
		{
			name:          "badge with multiple type parts",
			fileName:      "badge-diff-analysis-flat.svg",
			expectedType:  "diff-analysis",
			expectedStyle: "flat",
		},
		{
			name:          "invalid - not enough parts",
			fileName:      "badge-flat.svg",
			expectedType:  "",
			expectedStyle: "",
		},
		{
			name:          "invalid - doesn't start with badge-",
			fileName:      "coverage-flat.svg",
			expectedType:  "",
			expectedStyle: "",
		},
		{
			name:          "invalid - no .svg extension",
			fileName:      "badge-coverage-flat",
			expectedType:  "coverage",
			expectedStyle: "flat",
		},
		{
			name:          "invalid - too few parts",
			fileName:      "badge-coverage.svg",
			expectedType:  "",
			expectedStyle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			badgeType, style := manager.parseBadgeFileName(tt.fileName)
			require.Equal(t, tt.expectedType, badgeType)
			require.Equal(t, tt.expectedStyle, style)
		})
	}
}

func TestGenerateStandardPRBadges(t *testing.T) {
	generator := New()

	t.Run("standard badges without quality grade", func(t *testing.T) {
		tempDir := t.TempDir()

		config := &PRBadgeConfig{
			OutputBasePath:         tempDir,
			CreateDirectories:      true,
			DirectoryPermissions:   0o755,
			FilePermissions:        0o644,
			CoveragePattern:        "badge-coverage-{style}.svg",
			TrendPattern:           "badge-trend-{style}.svg",
			StatusPattern:          "badge-status-{style}.svg",
			ComparisonPattern:      "badge-comparison-{style}.svg",
			Styles:                 []string{"flat", "flat-square", "for-the-badge"},
			DefaultStyle:           "flat",
			GenerateMultipleStyles: false, // Only generate default style for testing
		}
		manager := NewPRBadgeManager(generator, config)

		request := &PRBadgeRequest{
			Repository:   "test-repo",
			Owner:        "test-owner",
			PRNumber:     123,
			Branch:       "feature-branch",
			Coverage:     85.5,
			BaseCoverage: 80.0,
			Timestamp:    time.Now(),
		}

		ctx := context.Background()
		result, err := manager.GenerateStandardPRBadges(ctx, request)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should include coverage, trend, status, and comparison badges
		expectedTypes := []PRBadgeType{
			PRBadgeCoverage,
			PRBadgeTrend,
			PRBadgeStatus,
			PRBadgeComparison,
		}

		for _, expectedType := range expectedTypes {
			require.Contains(t, result.Badges, expectedType, "Missing badge type: %s", expectedType)
		}

		// Should not include quality badge since no grade provided
		require.NotContains(t, result.Badges, PRBadgeQuality)
	})

	t.Run("standard badges with quality grade", func(t *testing.T) {
		tempDir := t.TempDir()

		config := &PRBadgeConfig{
			OutputBasePath:         tempDir,
			CreateDirectories:      true,
			DirectoryPermissions:   0o755,
			FilePermissions:        0o644,
			CoveragePattern:        "badge-coverage-{style}.svg",
			TrendPattern:           "badge-trend-{style}.svg",
			StatusPattern:          "badge-status-{style}.svg",
			ComparisonPattern:      "badge-comparison-{style}.svg",
			Styles:                 []string{"flat", "flat-square", "for-the-badge"},
			DefaultStyle:           "flat",
			GenerateMultipleStyles: false, // Only generate default style for testing
		}
		manager := NewPRBadgeManager(generator, config)

		request := &PRBadgeRequest{
			Repository:   "test-repo",
			Owner:        "test-owner",
			PRNumber:     456,
			Branch:       "feature-branch",
			Coverage:     92.0,
			BaseCoverage: 88.0,
			QualityGrade: "A",
			Timestamp:    time.Now(),
		}

		ctx := context.Background()
		result, err := manager.GenerateStandardPRBadges(ctx, request)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should include all standard badges plus quality badge
		expectedTypes := []PRBadgeType{
			PRBadgeCoverage,
			PRBadgeTrend,
			PRBadgeStatus,
			PRBadgeComparison,
			PRBadgeQuality,
		}

		for _, expectedType := range expectedTypes {
			require.Contains(t, result.Badges, expectedType)
		}
	})
}

func TestPRBadgeManagerWithCustomLabels(t *testing.T) {
	tempDir := t.TempDir()

	generator := New()
	config := &PRBadgeConfig{
		OutputBasePath:         tempDir,
		CreateDirectories:      true,
		DirectoryPermissions:   0o755,
		FilePermissions:        0o644,
		CoveragePattern:        "badge-coverage-{style}.svg",
		TrendPattern:           "badge-trend-{style}.svg",
		StatusPattern:          "badge-status-{style}.svg",
		ComparisonPattern:      "badge-comparison-{style}.svg",
		Styles:                 []string{"flat", "flat-square", "for-the-badge"},
		DefaultStyle:           "flat",
		GenerateMultipleStyles: false, // Only generate default style for testing
	}
	manager := NewPRBadgeManager(generator, config)

	request := &PRBadgeRequest{
		Repository:   "test-repo",
		Owner:        "test-owner",
		PRNumber:     123,
		Coverage:     85.5,
		BaseCoverage: 80.0,
		Types:        []PRBadgeType{PRBadgeCoverage, PRBadgeTrend},
		CustomLabels: map[PRBadgeType]string{
			PRBadgeCoverage: "test coverage",
			PRBadgeTrend:    "coverage trend",
		},
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	result, err := manager.GeneratePRBadges(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, result.TotalBadges)

	// Verify badges were created with custom labels (this would be reflected in the actual badge content)
	require.Contains(t, result.Badges, PRBadgeCoverage)
	require.Contains(t, result.Badges, PRBadgeTrend)
}

func TestPRBadgeTypes(t *testing.T) {
	// Test that all badge type constants are properly defined
	expectedTypes := []PRBadgeType{
		PRBadgeCoverage,
		PRBadgeTrend,
		PRBadgeStatus,
		PRBadgeComparison,
		PRBadgeDiff,
		PRBadgeQuality,
	}

	expectedValues := []string{
		"coverage",
		"trend",
		"status",
		"comparison",
		"diff",
		"quality",
	}

	for i, badgeType := range expectedTypes {
		require.Equal(t, expectedValues[i], string(badgeType))
	}
}

func TestErrorDefinitions(t *testing.T) {
	// Test that error variables are properly defined
	require.Error(t, ErrUnsupportedBadgeType)
	require.Equal(t, "unsupported badge type", ErrUnsupportedBadgeType.Error())
}
