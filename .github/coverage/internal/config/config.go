// Package config provides configuration management for the coverage system
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Static error definitions
var (
	ErrInvalidCoverageThreshold = errors.New("coverage threshold must be between 0 and 100")
	ErrEmptyCoverageInput       = errors.New("coverage input file cannot be empty")
	ErrMissingGitHubToken       = errors.New("GitHub token is required for GitHub integration")
	ErrMissingGitHubOwner       = errors.New("GitHub repository owner is required")
	ErrMissingGitHubRepo        = errors.New("GitHub repository name is required")
	ErrInvalidBadgeStyle        = errors.New("invalid badge style")
	ErrInvalidReportTheme       = errors.New("invalid report theme")
	ErrInvalidRetentionDays     = errors.New("history retention days must be positive")
	ErrInvalidMaxEntries        = errors.New("history max entries must be positive")
)

// Config holds the main configuration for the coverage system
type Config struct {
	// Coverage settings
	Coverage CoverageConfig `json:"coverage"`
	// GitHub integration settings
	GitHub GitHubConfig `json:"github"`
	// Badge generation settings
	Badge BadgeConfig `json:"badge"`
	// Report generation settings
	Report ReportConfig `json:"report"`
	// History tracking settings
	History HistoryConfig `json:"history"`
	// Storage settings
	Storage StorageConfig `json:"storage"`
}

// CoverageConfig holds coverage analysis settings
type CoverageConfig struct {
	// Input coverage file path
	InputFile string `json:"input_file"`
	// Output directory for generated files
	OutputDir string `json:"output_dir"`
	// Minimum coverage threshold
	Threshold float64 `json:"threshold"`
	// Paths to exclude from coverage
	ExcludePaths []string `json:"exclude_paths"`
	// File patterns to exclude
	ExcludeFiles []string `json:"exclude_files"`
	// Whether to exclude test files
	ExcludeTests bool `json:"exclude_tests"`
	// Whether to exclude generated files
	ExcludeGenerated bool `json:"exclude_generated"`
}

// GitHubConfig holds GitHub integration settings
type GitHubConfig struct {
	// GitHub API token
	Token string `json:"token"`
	// Repository owner
	Owner string `json:"owner"`
	// Repository name
	Repository string `json:"repository"`
	// Pull request number (0 if not in PR context)
	PullRequest int `json:"pull_request"`
	// Commit SHA
	CommitSHA string `json:"commit_sha"`
	// Whether to post PR comments
	PostComments bool `json:"post_comments"`
	// Whether to create commit statuses
	CreateStatuses bool `json:"create_statuses"`
	// API timeout
	Timeout time.Duration `json:"timeout"`
}

// BadgeConfig holds badge generation settings
type BadgeConfig struct {
	// Badge style (flat, flat-square, for-the-badge)
	Style string `json:"style"`
	// Label text
	Label string `json:"label"`
	// Logo URL
	Logo string `json:"logo"`
	// Logo color
	LogoColor string `json:"logo_color"`
	// Output file path
	OutputFile string `json:"output_file"`
	// Whether to generate trend badge
	IncludeTrend bool `json:"include_trend"`
}

// ReportConfig holds HTML report generation settings
type ReportConfig struct {
	// Output file path
	OutputFile string `json:"output_file"`
	// Report title
	Title string `json:"title"`
	// Theme (github-dark, light, etc.)
	Theme string `json:"theme"`
	// Whether to show package breakdown
	ShowPackages bool `json:"show_packages"`
	// Whether to show file breakdown
	ShowFiles bool `json:"show_files"`
	// Whether to show missing lines
	ShowMissing bool `json:"show_missing"`
	// Whether to enable responsive design
	Responsive bool `json:"responsive"`
	// Whether to include interactive features
	Interactive bool `json:"interactive"`
}

// HistoryConfig holds history tracking settings
type HistoryConfig struct {
	// Whether to enable history tracking
	Enabled bool `json:"enabled"`
	// Storage path for history files
	StoragePath string `json:"storage_path"`
	// Number of days to retain history
	RetentionDays int `json:"retention_days"`
	// Maximum number of entries to keep
	MaxEntries int `json:"max_entries"`
	// Whether to enable automatic cleanup
	AutoCleanup bool `json:"auto_cleanup"`
	// Whether to enable detailed metrics
	MetricsEnabled bool `json:"metrics_enabled"`
}

// StorageConfig holds storage settings
type StorageConfig struct {
	// Base directory for all coverage files
	BaseDir string `json:"base_dir"`
	// Whether to create directories automatically
	AutoCreate bool `json:"auto_create"`
	// File permissions for created files
	FileMode os.FileMode `json:"file_mode"`
	// Directory permissions for created directories
	DirMode os.FileMode `json:"dir_mode"`
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	config := &Config{
		Coverage: CoverageConfig{
			InputFile:        getEnvString("COVERAGE_INPUT_FILE", "coverage.txt"),
			OutputDir:        getEnvString("COVERAGE_OUTPUT_DIR", ".github/coverage"),
			Threshold:        getEnvFloat("COVERAGE_THRESHOLD", 80.0),
			ExcludePaths:     getEnvStringSlice("COVERAGE_EXCLUDE_PATHS", []string{"vendor/", "test/", "testdata/"}),
			ExcludeFiles:     getEnvStringSlice("COVERAGE_EXCLUDE_FILES", []string{"*_test.go", "*.pb.go"}),
			ExcludeTests:     getEnvBool("COVERAGE_EXCLUDE_TESTS", true),
			ExcludeGenerated: getEnvBool("COVERAGE_EXCLUDE_GENERATED", true),
		},
		GitHub: GitHubConfig{
			Token:          getEnvString("GITHUB_TOKEN", ""),
			Owner:          getEnvString("GITHUB_REPOSITORY_OWNER", ""),
			Repository:     getRepositoryFromEnv(),
			PullRequest:    getEnvInt("GITHUB_PR_NUMBER", 0),
			CommitSHA:      getEnvString("GITHUB_SHA", ""),
			PostComments:   getEnvBool("COVERAGE_POST_COMMENTS", true),
			CreateStatuses: getEnvBool("COVERAGE_CREATE_STATUSES", true),
			Timeout:        getEnvDuration("GITHUB_TIMEOUT", 30*time.Second),
		},
		Badge: BadgeConfig{
			Style:        getEnvString("COVERAGE_BADGE_STYLE", "flat"),
			Label:        getEnvString("COVERAGE_BADGE_LABEL", "coverage"),
			Logo:         getEnvString("COVERAGE_BADGE_LOGO", ""),
			LogoColor:    getEnvString("COVERAGE_BADGE_LOGO_COLOR", "white"),
			OutputFile:   getEnvString("COVERAGE_BADGE_OUTPUT", "coverage.svg"),
			IncludeTrend: getEnvBool("COVERAGE_BADGE_TREND", false),
		},
		Report: ReportConfig{
			OutputFile:   getEnvString("COVERAGE_REPORT_OUTPUT", "coverage.html"),
			Title:        getEnvString("COVERAGE_REPORT_TITLE", "Coverage Report"),
			Theme:        getEnvString("COVERAGE_REPORT_THEME", "github-dark"),
			ShowPackages: getEnvBool("COVERAGE_REPORT_PACKAGES", true),
			ShowFiles:    getEnvBool("COVERAGE_REPORT_FILES", true),
			ShowMissing:  getEnvBool("COVERAGE_REPORT_MISSING", true),
			Responsive:   getEnvBool("COVERAGE_REPORT_RESPONSIVE", true),
			Interactive:  getEnvBool("COVERAGE_REPORT_INTERACTIVE", true),
		},
		History: HistoryConfig{
			Enabled:        getEnvBool("COVERAGE_HISTORY_ENABLED", true),
			StoragePath:    getEnvString("COVERAGE_HISTORY_PATH", ".github/coverage/history"),
			RetentionDays:  getEnvInt("COVERAGE_HISTORY_RETENTION", 90),
			MaxEntries:     getEnvInt("COVERAGE_HISTORY_MAX_ENTRIES", 1000),
			AutoCleanup:    getEnvBool("COVERAGE_HISTORY_CLEANUP", true),
			MetricsEnabled: getEnvBool("COVERAGE_HISTORY_METRICS", true),
		},
		Storage: StorageConfig{
			BaseDir:    getEnvString("COVERAGE_BASE_DIR", ".github/coverage"),
			AutoCreate: getEnvBool("COVERAGE_AUTO_CREATE_DIRS", true),
			FileMode:   os.FileMode(getEnvIntBounded("COVERAGE_FILE_MODE", 0o644, 0, 0o777)),
			DirMode:    os.FileMode(getEnvIntBounded("COVERAGE_DIR_MODE", 0o755, 0, 0o777)),
		},
	}

	return config
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	// Validate coverage settings
	if c.Coverage.Threshold < 0 || c.Coverage.Threshold > 100 {
		return fmt.Errorf("%w, got: %.1f", ErrInvalidCoverageThreshold, c.Coverage.Threshold)
	}

	if c.Coverage.InputFile == "" {
		return ErrEmptyCoverageInput
	}

	// Validate GitHub settings if GitHub integration is enabled
	if c.GitHub.PostComments || c.GitHub.CreateStatuses {
		if c.GitHub.Token == "" {
			return ErrMissingGitHubToken
		}
		if c.GitHub.Owner == "" {
			return ErrMissingGitHubOwner
		}
		if c.GitHub.Repository == "" {
			return ErrMissingGitHubRepo
		}
	}

	// Validate badge settings
	validStyles := []string{"flat", "flat-square", "for-the-badge"}
	if !contains(validStyles, c.Badge.Style) {
		return fmt.Errorf("%w: %s, must be one of: %v", ErrInvalidBadgeStyle, c.Badge.Style, validStyles)
	}

	// Validate report settings
	validThemes := []string{"github-dark", "light", "github-light"}
	if !contains(validThemes, c.Report.Theme) {
		return fmt.Errorf("%w: %s, must be one of: %v", ErrInvalidReportTheme, c.Report.Theme, validThemes)
	}

	// Validate history settings
	if c.History.Enabled {
		if c.History.RetentionDays <= 0 {
			return fmt.Errorf("%w: got %d", ErrInvalidRetentionDays, c.History.RetentionDays)
		}
		if c.History.MaxEntries <= 0 {
			return fmt.Errorf("%w: got %d", ErrInvalidMaxEntries, c.History.MaxEntries)
		}
	}

	return nil
}

// IsGitHubContext returns true if running in a GitHub Actions context
func (c *Config) IsGitHubContext() bool {
	return c.GitHub.Owner != "" && c.GitHub.Repository != "" && c.GitHub.CommitSHA != ""
}

// IsPullRequestContext returns true if running in a pull request context
func (c *Config) IsPullRequestContext() bool {
	return c.IsGitHubContext() && c.GitHub.PullRequest > 0
}

// GetBadgeURL returns the URL for the coverage badge
func (c *Config) GetBadgeURL() string {
	if c.GitHub.Owner == "" || c.GitHub.Repository == "" {
		return ""
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s/%s",
		c.GitHub.Owner, c.GitHub.Repository, c.Storage.BaseDir, c.Badge.OutputFile)
}

// GetReportURL returns the URL for the coverage report
func (c *Config) GetReportURL() string {
	if c.GitHub.Owner == "" || c.GitHub.Repository == "" {
		return ""
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s/%s",
		c.GitHub.Owner, c.GitHub.Repository, c.Storage.BaseDir, c.Report.OutputFile)
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvIntBounded parses an integer from environment with min/max bounds
func getEnvIntBounded(key string, defaultValue, minValue, maxValue int) uint32 {
	// For file permissions, we know the valid range is 0-0777 (0-511 decimal)
	// Ensure bounds are reasonable for uint32
	const maxFileMode = 0o777

	// Validate and adjust bounds
	if minValue < 0 {
		minValue = 0
	}
	if maxValue < 0 || maxValue > maxFileMode {
		maxValue = maxFileMode
	}
	if minValue > maxValue {
		minValue = 0
		maxValue = maxFileMode
	}

	// Start with default value
	value := defaultValue

	// Parse environment variable if present
	if envValue := os.Getenv(key); envValue != "" {
		// Parse as uint to ensure non-negative
		if parsed, err := strconv.ParseUint(envValue, 0, 32); err == nil && parsed <= uint64(maxFileMode) {
			value = int(parsed)
		}
	}

	// Apply bounds checking
	if value < minValue {
		value = minValue
	} else if value > maxValue {
		value = maxValue
	}

	// At this point, value is guaranteed to be between 0 and maxFileMode (0o777 = 511)
	// which safely fits in uint32
	return uint32(value) //nolint:gosec // value is bounded between 0 and 511
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func getRepositoryFromEnv() string {
	// GitHub Actions provides GITHUB_REPOSITORY in "owner/repo" format
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
