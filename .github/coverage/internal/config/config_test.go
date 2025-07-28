package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Clear any existing environment variables
	clearEnvironment()
	defer clearEnvironment()

	config := Load()

	assert.NotNil(t, config)

	// Test default values
	assert.Equal(t, "coverage.txt", config.Coverage.InputFile)
	assert.Equal(t, ".github/coverage", config.Coverage.OutputDir)
	assert.InDelta(t, 80.0, config.Coverage.Threshold, 0.001)
	assert.Equal(t, []string{"vendor/", "test/", "testdata/"}, config.Coverage.ExcludePaths)
	assert.Equal(t, []string{"*_test.go", "*.pb.go"}, config.Coverage.ExcludeFiles)
	assert.True(t, config.Coverage.ExcludeTests)
	assert.True(t, config.Coverage.ExcludeGenerated)

	// Test GitHub defaults
	assert.Empty(t, config.GitHub.Token)
	assert.Empty(t, config.GitHub.Owner)
	assert.Empty(t, config.GitHub.Repository)
	assert.Equal(t, 0, config.GitHub.PullRequest)
	assert.Empty(t, config.GitHub.CommitSHA)
	assert.True(t, config.GitHub.PostComments)
	assert.True(t, config.GitHub.CreateStatuses)
	assert.Equal(t, 30*time.Second, config.GitHub.Timeout)

	// Test badge defaults
	assert.Equal(t, "flat", config.Badge.Style)
	assert.Equal(t, "coverage", config.Badge.Label)
	assert.Empty(t, config.Badge.Logo)
	assert.Equal(t, "white", config.Badge.LogoColor)
	assert.Equal(t, "coverage.svg", config.Badge.OutputFile)
	assert.False(t, config.Badge.IncludeTrend)

	// Test report defaults
	assert.Equal(t, "coverage.html", config.Report.OutputFile)
	assert.Equal(t, "Coverage Report", config.Report.Title)
	assert.Equal(t, "github-dark", config.Report.Theme)
	assert.True(t, config.Report.ShowPackages)
	assert.True(t, config.Report.ShowFiles)
	assert.True(t, config.Report.ShowMissing)
	assert.True(t, config.Report.Responsive)
	assert.True(t, config.Report.Interactive)

	// Test history defaults
	assert.True(t, config.History.Enabled)
	assert.Equal(t, ".github/coverage/history", config.History.StoragePath)
	assert.Equal(t, 90, config.History.RetentionDays)
	assert.Equal(t, 1000, config.History.MaxEntries)
	assert.True(t, config.History.AutoCleanup)
	assert.True(t, config.History.MetricsEnabled)

	// Test storage defaults
	assert.Equal(t, ".github/coverage", config.Storage.BaseDir)
	assert.True(t, config.Storage.AutoCreate)
	assert.Equal(t, os.FileMode(0o644), config.Storage.FileMode)
	assert.Equal(t, os.FileMode(0o755), config.Storage.DirMode)
}

func TestLoadWithEnvironmentVariables(t *testing.T) {
	clearEnvironment()
	defer clearEnvironment()

	// Set environment variables
	_ = os.Setenv("COVERAGE_INPUT_FILE", "custom-coverage.txt")
	_ = os.Setenv("COVERAGE_OUTPUT_DIR", "/tmp/coverage")
	_ = os.Setenv("COVERAGE_THRESHOLD", "85.5")
	_ = os.Setenv("COVERAGE_EXCLUDE_PATHS", "vendor/,build/,dist/")
	_ = os.Setenv("COVERAGE_EXCLUDE_FILES", "*.test.go,*.mock.go")
	_ = os.Setenv("COVERAGE_EXCLUDE_TESTS", "false")
	_ = os.Setenv("COVERAGE_EXCLUDE_GENERATED", "false")

	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	_ = os.Setenv("GITHUB_REPOSITORY_OWNER", "test-owner")
	_ = os.Setenv("GITHUB_REPOSITORY", "test-owner/test-repo")
	_ = os.Setenv("GITHUB_PR_NUMBER", "123")
	_ = os.Setenv("GITHUB_SHA", "abc123def456")
	_ = os.Setenv("COVERAGE_POST_COMMENTS", "false")
	_ = os.Setenv("COVERAGE_CREATE_STATUSES", "false")
	_ = os.Setenv("GITHUB_TIMEOUT", "60s")

	_ = os.Setenv("COVERAGE_BADGE_STYLE", "flat-square")
	_ = os.Setenv("COVERAGE_BADGE_LABEL", "test coverage")
	_ = os.Setenv("COVERAGE_BADGE_LOGO", "go")
	_ = os.Setenv("COVERAGE_BADGE_LOGO_COLOR", "blue")
	_ = os.Setenv("COVERAGE_BADGE_OUTPUT", "test-coverage.svg")
	_ = os.Setenv("COVERAGE_BADGE_TREND", "true")

	_ = os.Setenv("COVERAGE_REPORT_OUTPUT", "test-coverage.html")
	_ = os.Setenv("COVERAGE_REPORT_TITLE", "Test Coverage Report")
	_ = os.Setenv("COVERAGE_REPORT_THEME", "light")
	_ = os.Setenv("COVERAGE_REPORT_PACKAGES", "false")
	_ = os.Setenv("COVERAGE_REPORT_FILES", "false")
	_ = os.Setenv("COVERAGE_REPORT_MISSING", "false")
	_ = os.Setenv("COVERAGE_REPORT_RESPONSIVE", "false")
	_ = os.Setenv("COVERAGE_REPORT_INTERACTIVE", "false")

	_ = os.Setenv("COVERAGE_HISTORY_ENABLED", "false")
	_ = os.Setenv("COVERAGE_HISTORY_PATH", "/tmp/history")
	_ = os.Setenv("COVERAGE_HISTORY_RETENTION", "30")
	_ = os.Setenv("COVERAGE_HISTORY_MAX_ENTRIES", "500")
	_ = os.Setenv("COVERAGE_HISTORY_CLEANUP", "false")
	_ = os.Setenv("COVERAGE_HISTORY_METRICS", "false")

	_ = os.Setenv("COVERAGE_BASE_DIR", "/tmp/base")
	_ = os.Setenv("COVERAGE_AUTO_CREATE_DIRS", "false")
	_ = os.Setenv("COVERAGE_FILE_MODE", "420")
	_ = os.Setenv("COVERAGE_DIR_MODE", "493")

	config := Load()

	// Test coverage settings
	assert.Equal(t, "custom-coverage.txt", config.Coverage.InputFile)
	assert.Equal(t, "/tmp/coverage", config.Coverage.OutputDir)
	assert.InDelta(t, 85.5, config.Coverage.Threshold, 0.001)
	assert.Equal(t, []string{"vendor/", "build/", "dist/"}, config.Coverage.ExcludePaths)
	assert.Equal(t, []string{"*.test.go", "*.mock.go"}, config.Coverage.ExcludeFiles)
	assert.False(t, config.Coverage.ExcludeTests)
	assert.False(t, config.Coverage.ExcludeGenerated)

	// Test GitHub settings
	assert.Equal(t, "test-token", config.GitHub.Token)
	assert.Equal(t, "test-owner", config.GitHub.Owner)
	assert.Equal(t, "test-repo", config.GitHub.Repository)
	assert.Equal(t, 123, config.GitHub.PullRequest)
	assert.Equal(t, "abc123def456", config.GitHub.CommitSHA)
	assert.False(t, config.GitHub.PostComments)
	assert.False(t, config.GitHub.CreateStatuses)
	assert.Equal(t, 60*time.Second, config.GitHub.Timeout)

	// Test badge settings
	assert.Equal(t, "flat-square", config.Badge.Style)
	assert.Equal(t, "test coverage", config.Badge.Label)
	assert.Equal(t, "go", config.Badge.Logo)
	assert.Equal(t, "blue", config.Badge.LogoColor)
	assert.Equal(t, "test-coverage.svg", config.Badge.OutputFile)
	assert.True(t, config.Badge.IncludeTrend)

	// Test report settings
	assert.Equal(t, "test-coverage.html", config.Report.OutputFile)
	assert.Equal(t, "Test Coverage Report", config.Report.Title)
	assert.Equal(t, "light", config.Report.Theme)
	assert.False(t, config.Report.ShowPackages)
	assert.False(t, config.Report.ShowFiles)
	assert.False(t, config.Report.ShowMissing)
	assert.False(t, config.Report.Responsive)
	assert.False(t, config.Report.Interactive)

	// Test history settings
	assert.False(t, config.History.Enabled)
	assert.Equal(t, "/tmp/history", config.History.StoragePath)
	assert.Equal(t, 30, config.History.RetentionDays)
	assert.Equal(t, 500, config.History.MaxEntries)
	assert.False(t, config.History.AutoCleanup)
	assert.False(t, config.History.MetricsEnabled)

	// Test storage settings
	assert.Equal(t, "/tmp/base", config.Storage.BaseDir)
	assert.False(t, config.Storage.AutoCreate)
	assert.Equal(t, os.FileMode(0o644), config.Storage.FileMode)
	assert.Equal(t, os.FileMode(0o755), config.Storage.DirMode)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid default config",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				Badge: BadgeConfig{
					Style: "flat",
				},
				Report: ReportConfig{
					Theme: "github-dark",
				},
				History: HistoryConfig{
					Enabled:       false, // Disabled for this test
					RetentionDays: 90,
					MaxEntries:    1000,
				},
			},
			expectError: false,
		},
		{
			name: "invalid coverage threshold - too low",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: -1.0,
				},
			},
			expectError: true,
			errorMsg:    "coverage threshold must be between 0 and 100",
		},
		{
			name: "invalid coverage threshold - too high",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 101.0,
				},
			},
			expectError: true,
			errorMsg:    "coverage threshold must be between 0 and 100",
		},
		{
			name: "empty input file",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "",
					Threshold: 80.0,
				},
			},
			expectError: true,
			errorMsg:    "coverage input file cannot be empty",
		},
		{
			name: "GitHub integration missing token",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				GitHub: GitHubConfig{
					PostComments: true,
					Token:        "",
					Owner:        "test-owner",
					Repository:   "test-repo",
				},
			},
			expectError: true,
			errorMsg:    "GitHub token is required for GitHub integration",
		},
		{
			name: "GitHub integration missing owner",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				GitHub: GitHubConfig{
					PostComments: true,
					Token:        "test-token",
					Owner:        "",
					Repository:   "test-repo",
				},
			},
			expectError: true,
			errorMsg:    "GitHub repository owner is required",
		},
		{
			name: "GitHub integration missing repository",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				GitHub: GitHubConfig{
					PostComments: true,
					Token:        "test-token",
					Owner:        "test-owner",
					Repository:   "",
				},
			},
			expectError: true,
			errorMsg:    "GitHub repository name is required",
		},
		{
			name: "invalid badge style",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				Badge: BadgeConfig{
					Style: "invalid-style",
				},
				Report: ReportConfig{
					Theme: "github-dark",
				},
			},
			expectError: true,
			errorMsg:    "invalid badge style",
		},
		{
			name: "invalid report theme",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				Badge: BadgeConfig{
					Style: "flat",
				},
				Report: ReportConfig{
					Theme: "invalid-theme",
				},
			},
			expectError: true,
			errorMsg:    "invalid report theme",
		},
		{
			name: "invalid history retention days",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				Badge: BadgeConfig{
					Style: "flat",
				},
				Report: ReportConfig{
					Theme: "github-dark",
				},
				History: HistoryConfig{
					Enabled:       true,
					RetentionDays: -1,
					MaxEntries:    100,
				},
			},
			expectError: true,
			errorMsg:    "history retention days must be positive",
		},
		{
			name: "invalid history max entries",
			config: &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				Badge: BadgeConfig{
					Style: "flat",
				},
				Report: ReportConfig{
					Theme: "github-dark",
				},
				History: HistoryConfig{
					Enabled:       true,
					RetentionDays: 30,
					MaxEntries:    0,
				},
			},
			expectError: true,
			errorMsg:    "history max entries must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsGitHubContext(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "complete GitHub context",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "test-owner",
					Repository: "test-repo",
					CommitSHA:  "abc123",
				},
			},
			expected: true,
		},
		{
			name: "missing owner",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "",
					Repository: "test-repo",
					CommitSHA:  "abc123",
				},
			},
			expected: false,
		},
		{
			name: "missing repository",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "test-owner",
					Repository: "",
					CommitSHA:  "abc123",
				},
			},
			expected: false,
		},
		{
			name: "missing commit SHA",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "test-owner",
					Repository: "test-repo",
					CommitSHA:  "",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsGitHubContext()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPullRequestContext(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "complete PR context",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:       "test-owner",
					Repository:  "test-repo",
					CommitSHA:   "abc123",
					PullRequest: 123,
				},
			},
			expected: true,
		},
		{
			name: "GitHub context but no PR",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:       "test-owner",
					Repository:  "test-repo",
					CommitSHA:   "abc123",
					PullRequest: 0,
				},
			},
			expected: false,
		},
		{
			name: "incomplete GitHub context",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:       "",
					Repository:  "test-repo",
					CommitSHA:   "abc123",
					PullRequest: 123,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsPullRequestContext()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBadgeURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "complete configuration",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "test-owner",
					Repository: "test-repo",
				},
				Storage: StorageConfig{
					BaseDir: ".github/coverage",
				},
				Badge: BadgeConfig{
					OutputFile: "coverage.svg",
				},
			},
			expected: "https://test-owner.github.io/test-repo/coverage.svg",
		},
		{
			name: "missing owner",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "",
					Repository: "test-repo",
				},
			},
			expected: "",
		},
		{
			name: "missing repository",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "test-owner",
					Repository: "",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetBadgeURL()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetReportURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "complete configuration",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "test-owner",
					Repository: "test-repo",
				},
				Storage: StorageConfig{
					BaseDir: ".github/coverage",
				},
				Report: ReportConfig{
					OutputFile: "coverage.html",
				},
			},
			expected: "https://test-owner.github.io/test-repo/",
		},
		{
			name: "missing owner",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "",
					Repository: "test-repo",
				},
			},
			expected: "",
		},
		{
			name: "missing repository",
			config: &Config{
				GitHub: GitHubConfig{
					Owner:      "test-owner",
					Repository: "",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetReportURL()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvironmentHelpers(t *testing.T) {
	clearEnvironment()
	defer clearEnvironment()

	t.Run("getEnvString", func(t *testing.T) {
		assert.Equal(t, "default", getEnvString("TEST_STRING", "default"))

		_ = os.Setenv("TEST_STRING", "custom")
		assert.Equal(t, "custom", getEnvString("TEST_STRING", "default"))
	})

	t.Run("getEnvInt", func(t *testing.T) {
		assert.Equal(t, 42, getEnvInt("TEST_INT", 42))

		_ = os.Setenv("TEST_INT", "123")
		assert.Equal(t, 123, getEnvInt("TEST_INT", 42))

		_ = os.Setenv("TEST_INT", "invalid")
		assert.Equal(t, 42, getEnvInt("TEST_INT", 42))
	})

	t.Run("getEnvFloat", func(t *testing.T) {
		assert.InDelta(t, 3.14, getEnvFloat("TEST_FLOAT", 3.14), 0.001)

		_ = os.Setenv("TEST_FLOAT", "2.71")
		assert.InDelta(t, 2.71, getEnvFloat("TEST_FLOAT", 3.14), 0.001)

		_ = os.Setenv("TEST_FLOAT", "invalid")
		assert.InDelta(t, 3.14, getEnvFloat("TEST_FLOAT", 3.14), 0.001)
	})

	t.Run("getEnvBool", func(t *testing.T) {
		assert.True(t, getEnvBool("TEST_BOOL", true))

		// Test true values
		trueValues := []string{"true", "1", "yes", "on", "TRUE", "YES", "ON"}
		for _, val := range trueValues {
			_ = os.Setenv("TEST_BOOL", val)
			assert.True(t, getEnvBool("TEST_BOOL", false), "Value %s should be true", val)
		}

		// Test false values
		falseValues := []string{"false", "0", "no", "off", "FALSE", "NO", "OFF"}
		for _, val := range falseValues {
			_ = os.Setenv("TEST_BOOL", val)
			assert.False(t, getEnvBool("TEST_BOOL", true), "Value %s should be false", val)
		}

		// Test invalid value (should return default)
		_ = os.Setenv("TEST_BOOL", "invalid")
		assert.True(t, getEnvBool("TEST_BOOL", true))
	})

	t.Run("getEnvDuration", func(t *testing.T) {
		assert.Equal(t, 5*time.Second, getEnvDuration("TEST_DURATION", 5*time.Second))

		_ = os.Setenv("TEST_DURATION", "10s")
		assert.Equal(t, 10*time.Second, getEnvDuration("TEST_DURATION", 5*time.Second))

		_ = os.Setenv("TEST_DURATION", "invalid")
		assert.Equal(t, 5*time.Second, getEnvDuration("TEST_DURATION", 5*time.Second))
	})

	t.Run("getEnvStringSlice", func(t *testing.T) {
		defaultSlice := []string{"a", "b", "c"}
		assert.Equal(t, defaultSlice, getEnvStringSlice("TEST_SLICE", defaultSlice))

		_ = os.Setenv("TEST_SLICE", "x,y,z")
		assert.Equal(t, []string{"x", "y", "z"}, getEnvStringSlice("TEST_SLICE", defaultSlice))

		_ = os.Setenv("TEST_SLICE", "single")
		assert.Equal(t, []string{"single"}, getEnvStringSlice("TEST_SLICE", defaultSlice))
	})
}

func TestGetRepositoryFromEnv(t *testing.T) {
	clearEnvironment()
	defer clearEnvironment()

	t.Run("valid repository format", func(t *testing.T) {
		_ = os.Setenv("GITHUB_REPOSITORY", "owner/repository")
		assert.Equal(t, "repository", getRepositoryFromEnv())
	})

	t.Run("invalid repository format", func(t *testing.T) {
		_ = os.Setenv("GITHUB_REPOSITORY", "invalid-format")
		assert.Empty(t, getRepositoryFromEnv())
	})

	t.Run("empty repository", func(t *testing.T) {
		_ = os.Setenv("GITHUB_REPOSITORY", "")
		assert.Empty(t, getRepositoryFromEnv())
	})

	t.Run("missing repository", func(t *testing.T) {
		_ = os.Unsetenv("GITHUB_REPOSITORY")
		assert.Empty(t, getRepositoryFromEnv())
	})
}

func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item found",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "item not found",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
		{
			name:     "empty item",
			slice:    []string{"", "b", "c"},
			item:     "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitHubActionsIntegration(t *testing.T) {
	clearEnvironment()
	defer clearEnvironment()

	// Set GitHub Actions environment variables
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	_ = os.Setenv("GITHUB_REPOSITORY", "test-owner/test-repo")
	_ = os.Setenv("GITHUB_REPOSITORY_OWNER", "test-owner")
	_ = os.Setenv("GITHUB_SHA", "abc123def456")
	_ = os.Setenv("GITHUB_PR_NUMBER", "456")

	config := Load()

	assert.Equal(t, "test-token", config.GitHub.Token)
	assert.Equal(t, "test-owner", config.GitHub.Owner)
	assert.Equal(t, "test-repo", config.GitHub.Repository)
	assert.Equal(t, "abc123def456", config.GitHub.CommitSHA)
	assert.Equal(t, 456, config.GitHub.PullRequest)

	assert.True(t, config.IsGitHubContext())
	assert.True(t, config.IsPullRequestContext())

	// Test URL generation (in PR context, should return PR-specific URLs)
	expectedBadgeURL := "https://test-owner.github.io/test-repo/badges/pr/456/coverage.svg"
	assert.Equal(t, expectedBadgeURL, config.GetBadgeURL())

	expectedReportURL := "https://test-owner.github.io/test-repo/reports/pr/456/coverage.html"
	assert.Equal(t, expectedReportURL, config.GetReportURL())
}

func TestConfigurationEdgeCases(t *testing.T) {
	t.Run("all GitHub integration disabled", func(t *testing.T) {
		config := &Config{
			Coverage: CoverageConfig{
				InputFile: "coverage.txt",
				Threshold: 80.0,
			},
			Badge: BadgeConfig{
				Style: "flat",
			},
			Report: ReportConfig{
				Theme: "github-dark",
			},
			GitHub: GitHubConfig{
				PostComments:   false,
				CreateStatuses: false,
			},
		}

		err := config.Validate()
		assert.NoError(t, err, "Should not require GitHub settings when integration is disabled")
	})

	t.Run("history disabled", func(t *testing.T) {
		config := &Config{
			Coverage: CoverageConfig{
				InputFile: "coverage.txt",
				Threshold: 80.0,
			},
			Badge: BadgeConfig{
				Style: "flat",
			},
			Report: ReportConfig{
				Theme: "github-dark",
			},
			History: HistoryConfig{
				Enabled:       false,
				RetentionDays: -1, // Invalid but should be ignored
				MaxEntries:    0,  // Invalid but should be ignored
			},
		}

		err := config.Validate()
		assert.NoError(t, err, "Should not validate history settings when disabled")
	})

	t.Run("valid badge styles", func(t *testing.T) {
		validStyles := []string{"flat", "flat-square", "for-the-badge"}

		for _, style := range validStyles {
			config := &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				Badge: BadgeConfig{
					Style: style,
				},
				Report: ReportConfig{
					Theme: "github-dark",
				},
			}

			err := config.Validate()
			assert.NoError(t, err, "Style %s should be valid", style)
		}
	})

	t.Run("valid report themes", func(t *testing.T) {
		validThemes := []string{"github-dark", "light", "github-light"}

		for _, theme := range validThemes {
			config := &Config{
				Coverage: CoverageConfig{
					InputFile: "coverage.txt",
					Threshold: 80.0,
				},
				Badge: BadgeConfig{
					Style: "flat",
				},
				Report: ReportConfig{
					Theme: theme,
				},
			}

			err := config.Validate()
			assert.NoError(t, err, "Theme %s should be valid", theme)
		}
	})
}

// Helper function to clear environment variables
func clearEnvironment() {
	envVars := []string{
		"COVERAGE_INPUT_FILE", "COVERAGE_OUTPUT_DIR", "COVERAGE_THRESHOLD",
		"COVERAGE_EXCLUDE_PATHS", "COVERAGE_EXCLUDE_FILES", "COVERAGE_EXCLUDE_TESTS", "COVERAGE_EXCLUDE_GENERATED",
		"GITHUB_TOKEN", "GITHUB_REPOSITORY_OWNER", "GITHUB_REPOSITORY", "GITHUB_PR_NUMBER", "GITHUB_SHA",
		"COVERAGE_POST_COMMENTS", "COVERAGE_CREATE_STATUSES", "GITHUB_TIMEOUT",
		"COVERAGE_BADGE_STYLE", "COVERAGE_BADGE_LABEL", "COVERAGE_BADGE_LOGO", "COVERAGE_BADGE_LOGO_COLOR",
		"COVERAGE_BADGE_OUTPUT", "COVERAGE_BADGE_TREND",
		"COVERAGE_REPORT_OUTPUT", "COVERAGE_REPORT_TITLE", "COVERAGE_REPORT_THEME",
		"COVERAGE_REPORT_PACKAGES", "COVERAGE_REPORT_FILES", "COVERAGE_REPORT_MISSING",
		"COVERAGE_REPORT_RESPONSIVE", "COVERAGE_REPORT_INTERACTIVE",
		"COVERAGE_HISTORY_ENABLED", "COVERAGE_HISTORY_PATH", "COVERAGE_HISTORY_RETENTION",
		"COVERAGE_HISTORY_MAX_ENTRIES", "COVERAGE_HISTORY_CLEANUP", "COVERAGE_HISTORY_METRICS",
		"COVERAGE_BASE_DIR", "COVERAGE_AUTO_CREATE_DIRS", "COVERAGE_FILE_MODE", "COVERAGE_DIR_MODE",
		"TEST_STRING", "TEST_INT", "TEST_FLOAT", "TEST_BOOL", "TEST_DURATION", "TEST_SLICE",
	}

	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}
}
