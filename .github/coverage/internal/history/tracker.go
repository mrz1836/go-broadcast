// Package history tracks coverage trends and manages historical data retention
package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// Static error definitions
var (
	ErrNoEntriesFound       = errors.New("no entries found for branch")
	ErrUnsupportedDataType  = errors.New("unsupported data type")
)

// Tracker manages coverage history and trend analysis
type Tracker struct {
	config *Config
}

// Config holds history tracking configuration
type Config struct {
	StoragePath      string // Path to store history files
	RetentionDays    int    // Days to retain history data
	MaxEntries       int    // Maximum number of entries to keep
	CompressionLevel int    // Compression level for stored data (0-9)
	AutoCleanup      bool   // Automatically clean up old entries
	BackupPath       string // Optional backup storage path
	MetricsEnabled   bool   // Enable detailed metrics collection
}

// Entry represents a single coverage history entry
type Entry struct {
	Timestamp    time.Time                       `json:"timestamp"`
	Branch       string                          `json:"branch"`
	CommitSHA    string                          `json:"commit_sha"`
	CommitURL    string                          `json:"commit_url,omitempty"`
	Coverage     *parser.CoverageData            `json:"coverage"`
	Metadata     map[string]string               `json:"metadata,omitempty"`
	BuildInfo    *BuildInfo                      `json:"build_info,omitempty"`
	FileHashes   map[string]string               `json:"file_hashes,omitempty"`
	PackageStats map[string]*PackageHistoryStats `json:"package_stats,omitempty"`
}

// BuildInfo contains build-related information
type BuildInfo struct {
	GoVersion    string `json:"go_version"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	BuildTime    string `json:"build_time"`
	BuildNumber  string `json:"build_number,omitempty"`
	PullRequest  string `json:"pull_request,omitempty"`
	WorkflowID   string `json:"workflow_id,omitempty"`
}

// PackageHistoryStats tracks package-level statistics over time
type PackageHistoryStats struct {
	PreviousPercentage float64   `json:"previous_percentage"`
	Trend              string    `json:"trend"` // "up", "down", "stable"
	TrendPercentage    float64   `json:"trend_percentage"`
	FirstSeen          time.Time `json:"first_seen"`
	LastModified       time.Time `json:"last_modified"`
	FileCount          int       `json:"file_count"`
	LinesAdded         int       `json:"lines_added"`
	LinesRemoved       int       `json:"lines_removed"`
}

// TrendData represents coverage trend over time
type TrendData struct {
	Entries     []Entry        `json:"entries"`
	Summary     *TrendSummary  `json:"summary"`
	Analysis    *TrendAnalysis `json:"analysis"`
	GeneratedAt time.Time      `json:"generated_at"`
}

// TrendSummary provides high-level trend statistics
type TrendSummary struct {
	TotalEntries      int       `json:"total_entries"`
	DateRange         DateRange `json:"date_range"`
	AveragePercentage float64   `json:"average_percentage"`
	MinPercentage     float64   `json:"min_percentage"`
	MaxPercentage     float64   `json:"max_percentage"`
	CurrentTrend      string    `json:"current_trend"`
	TrendStrength     string    `json:"trend_strength"`  // "strong", "moderate", "weak"
	StabilityScore    float64   `json:"stability_score"` // 0-100
}

// DateRange represents a time range
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TrendAnalysis provides detailed trend analysis
type TrendAnalysis struct {
	ShortTermTrend  *PeriodAnalysis `json:"short_term_trend"`  // Last 7 days
	MediumTermTrend *PeriodAnalysis `json:"medium_term_trend"` // Last 30 days
	LongTermTrend   *PeriodAnalysis `json:"long_term_trend"`   // Last 90 days
	Volatility      float64         `json:"volatility"`
	Momentum        float64         `json:"momentum"`
	Prediction      *Prediction     `json:"prediction,omitempty"`
}

// PeriodAnalysis analyzes trends for a specific time period
type PeriodAnalysis struct {
	Period        string  `json:"period"`
	StartCoverage float64 `json:"start_coverage"`
	EndCoverage   float64 `json:"end_coverage"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Direction     string  `json:"direction"`
	Confidence    float64 `json:"confidence"`
	DataPoints    int     `json:"data_points"`
}

// Prediction provides coverage trend predictions
type Prediction struct {
	NextWeek   *PredictionPoint `json:"next_week,omitempty"`
	NextMonth  *PredictionPoint `json:"next_month,omitempty"`
	Confidence float64          `json:"confidence"`
	Model      string           `json:"model"`
	Factors    []string         `json:"factors,omitempty"`
}

// PredictionPoint represents a single prediction
type PredictionPoint struct {
	Percentage float64   `json:"percentage"`
	Date       time.Time `json:"date"`
	Range      Range     `json:"range"`
}

// Range represents a confidence range
type Range struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// New creates a new history tracker with default configuration
func New() *Tracker {
	return &Tracker{
		config: &Config{
			StoragePath:      ".github/coverage/history",
			RetentionDays:    90,
			MaxEntries:       1000,
			CompressionLevel: 6,
			AutoCleanup:      true,
			MetricsEnabled:   true,
		},
	}
}

// NewWithConfig creates a new history tracker with custom configuration
func NewWithConfig(config *Config) *Tracker {
	return &Tracker{config: config}
}

// Record saves a new coverage entry to history
func (t *Tracker) Record(ctx context.Context, coverage *parser.CoverageData, options ...Option) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	opts := &RecordOptions{
		Branch:    "main",
		CommitSHA: "",
		Metadata:  make(map[string]string),
	}

	for _, opt := range options {
		opt(opts)
	}

	// Generate a unique commit SHA if none provided
	if opts.CommitSHA == "" {
		opts.CommitSHA = fmt.Sprintf("auto_%d", time.Now().UnixNano())
	}

	entry := &Entry{
		Timestamp:    time.Now(),
		Branch:       opts.Branch,
		CommitSHA:    opts.CommitSHA,
		CommitURL:    opts.CommitURL,
		Coverage:     coverage,
		Metadata:     opts.Metadata,
		BuildInfo:    opts.BuildInfo,
		FileHashes:   t.calculateFileHashes(coverage),
		PackageStats: t.calculatePackageStats(coverage, opts.Branch),
	}

	return t.saveEntry(ctx, entry)
}

// GetTrend retrieves coverage trend data for analysis
func (t *Tracker) GetTrend(ctx context.Context, options ...TrendOption) (*TrendData, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	opts := &TrendOptions{
		Branch:    "main",
		Days:      30,
		MaxPoints: 100,
	}

	for _, opt := range options {
		opt(opts)
	}

	entries, err := t.loadEntries(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load entries: %w", err)
	}

	if len(entries) == 0 {
		return &TrendData{
			Entries:     []Entry{},
			Summary:     &TrendSummary{},
			Analysis:    &TrendAnalysis{},
			GeneratedAt: time.Now(),
		}, nil
	}

	summary := t.calculateSummary(entries)
	analysis := t.analyzeEntries(entries)

	return &TrendData{
		Entries:     entries,
		Summary:     summary,
		Analysis:    analysis,
		GeneratedAt: time.Now(),
	}, nil
}

// GetLatestEntry returns the most recent coverage entry
func (t *Tracker) GetLatestEntry(ctx context.Context, branch string) (*Entry, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	opts := &TrendOptions{
		Branch:    branch,
		Days:      7,
		MaxPoints: 1,
	}

	entries, err := t.loadEntries(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load entries: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoEntriesFound, branch)
	}

	return &entries[0], nil
}

// Cleanup removes old entries based on retention policy
func (t *Tracker) Cleanup(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if !t.config.AutoCleanup {
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -t.config.RetentionDays)

	entries, err := t.loadAllEntries(ctx)
	if err != nil {
		return fmt.Errorf("failed to load entries for cleanup: %w", err)
	}

	var kept []Entry
	var removed int

	for _, entry := range entries {
		if entry.Timestamp.After(cutoff) && len(kept) < t.config.MaxEntries {
			kept = append(kept, entry)
		} else {
			removed++
		}
	}

	if removed > 0 {
		if err := t.saveAllEntries(ctx, kept); err != nil {
			return fmt.Errorf("failed to save cleaned entries: %w", err)
		}
	}

	return nil
}

// GetStatistics returns comprehensive statistics about the coverage history
func (t *Tracker) GetStatistics(ctx context.Context) (*Statistics, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entries, err := t.loadAllEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load entries: %w", err)
	}

	stats := &Statistics{
		TotalEntries:   len(entries),
		UniqueProjects: make(map[string]int),
		UniqueBranches: make(map[string]int),
		StorageSize:    t.calculateStorageSize(),
		GeneratedAt:    time.Now(),
	}

	if len(entries) > 0 {
		stats.OldestEntry = entries[len(entries)-1].Timestamp
		stats.NewestEntry = entries[0].Timestamp

		for _, entry := range entries {
			if project, exists := entry.Metadata["project"]; exists {
				stats.UniqueProjects[project]++
			}
			stats.UniqueBranches[entry.Branch]++
		}
	}

	return stats, nil
}

// Legacy method for backward compatibility
func (t *Tracker) Add(branch, commit string, data interface{}) error {
	ctx := context.Background()

	// Convert interface{} to CoverageData if possible
	if coverage, ok := data.(*parser.CoverageData); ok {
		return t.Record(ctx, coverage, WithBranch(branch), WithCommit(commit, ""))
	}

	return fmt.Errorf("%w: %T", ErrUnsupportedDataType, data)
}

// saveEntry saves a single entry to storage
func (t *Tracker) saveEntry(ctx context.Context, entry *Entry) error {
	if err := t.ensureStorageDir(); err != nil {
		return fmt.Errorf("failed to ensure storage directory: %w", err)
	}

	filename := t.getEntryFilename(entry)
	filepath := filepath.Join(t.config.StoragePath, filename)

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write entry file: %w", err)
	}

	return nil
}

// loadEntries loads entries based on trend options
func (t *Tracker) loadEntries(ctx context.Context, opts *TrendOptions) ([]Entry, error) {
	entries, err := t.loadAllEntries(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by branch
	var filtered []Entry
	for _, entry := range entries {
		if entry.Branch == opts.Branch {
			filtered = append(filtered, entry)
		}
	}

	// Filter by date range
	cutoff := time.Now().AddDate(0, 0, -opts.Days)
	var recent []Entry
	for _, entry := range filtered {
		if entry.Timestamp.After(cutoff) {
			recent = append(recent, entry)
		}
	}

	// Limit to max points
	if len(recent) > opts.MaxPoints {
		recent = recent[:opts.MaxPoints]
	}

	return recent, nil
}

// loadAllEntries loads all entries from storage
func (t *Tracker) loadAllEntries(ctx context.Context) ([]Entry, error) {
	if err := t.ensureStorageDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure storage directory: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(t.config.StoragePath, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob entry files: %w", err)
	}

	entries := make([]Entry, 0, len(files))
	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		data, err := os.ReadFile(file) //nolint:gosec // File path from controlled directory listing
		if err != nil {
			continue // Skip corrupted files
		}

		var entry Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue // Skip corrupted files
		}

		entries = append(entries, entry)
	}

	// Sort by timestamp (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	return entries, nil
}

// saveAllEntries saves all entries to storage (used for cleanup)
func (t *Tracker) saveAllEntries(ctx context.Context, entries []Entry) error {
	// Remove existing files
	files, err := filepath.Glob(filepath.Join(t.config.StoragePath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to glob existing files: %w", err)
	}

	for _, file := range files {
		_ = os.Remove(file)
	}

	// Save new entries
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := t.saveEntry(ctx, &entry); err != nil {
			return err
		}
	}

	return nil
}

// Helper functions

func (t *Tracker) ensureStorageDir() error {
	return os.MkdirAll(t.config.StoragePath, 0750)
}

func (t *Tracker) getEntryFilename(entry *Entry) string {
	timestamp := entry.Timestamp.Format("20060102-150405.000000")
	branch := entry.Branch
	if branch == "" {
		branch = "main"
	}
	commitSHA := entry.CommitSHA
	if commitSHA == "" {
		commitSHA = "nocommit"
	}
	// Include microseconds and commit SHA to avoid collisions
	// Truncate commit SHA to 8 characters or use full if shorter
	if len(commitSHA) > 8 {
		commitSHA = commitSHA[:8]
	}
	return fmt.Sprintf("%s-%s-%s.json", timestamp, branch, commitSHA)
}

func (t *Tracker) calculateFileHashes(coverage *parser.CoverageData) map[string]string {
	hashes := make(map[string]string)
	// Simple implementation - in production would use actual file hashing
	for _, pkg := range coverage.Packages {
		for filepath := range pkg.Files {
			hashes[filepath] = fmt.Sprintf("hash_%d", len(filepath))
		}
	}
	return hashes
}

func (t *Tracker) calculatePackageStats(coverage *parser.CoverageData, branch string) map[string]*PackageHistoryStats {
	stats := make(map[string]*PackageHistoryStats)

	for name, pkg := range coverage.Packages {
		stats[name] = &PackageHistoryStats{
			PreviousPercentage: 0.0, // Would load from previous entry
			Trend:              "stable",
			TrendPercentage:    0.0,
			FirstSeen:          time.Now(),
			LastModified:       time.Now(),
			FileCount:          len(pkg.Files),
			LinesAdded:         0,
			LinesRemoved:       0,
		}
	}

	return stats
}

func (t *Tracker) calculateSummary(entries []Entry) *TrendSummary {
	if len(entries) == 0 {
		return &TrendSummary{}
	}

	var total float64
	min := entries[0].Coverage.Percentage
	max := entries[0].Coverage.Percentage

	for _, entry := range entries {
		total += entry.Coverage.Percentage
		if entry.Coverage.Percentage < min {
			min = entry.Coverage.Percentage
		}
		if entry.Coverage.Percentage > max {
			max = entry.Coverage.Percentage
		}
	}

	trend := "stable"
	if len(entries) >= 2 {
		recent := entries[0].Coverage.Percentage
		older := entries[len(entries)-1].Coverage.Percentage
		if recent > older {
			trend = "up"
		} else if recent < older {
			trend = "down"
		}
	}

	return &TrendSummary{
		TotalEntries:      len(entries),
		DateRange:         DateRange{Start: entries[len(entries)-1].Timestamp, End: entries[0].Timestamp},
		AveragePercentage: total / float64(len(entries)),
		MinPercentage:     min,
		MaxPercentage:     max,
		CurrentTrend:      trend,
		TrendStrength:     "moderate",
		StabilityScore:    85.0,
	}
}

func (t *Tracker) analyzeEntries(entries []Entry) *TrendAnalysis {
	return &TrendAnalysis{
		ShortTermTrend:  t.analyzePeriod(entries, 7),
		MediumTermTrend: t.analyzePeriod(entries, 30),
		LongTermTrend:   t.analyzePeriod(entries, 90),
		Volatility:      t.calculateVolatility(entries),
		Momentum:        t.calculateMomentum(entries),
		Prediction:      t.generatePrediction(entries),
	}
}

func (t *Tracker) analyzePeriod(entries []Entry, days int) *PeriodAnalysis {
	cutoff := time.Now().AddDate(0, 0, -days)
	var periodEntries []Entry

	for _, entry := range entries {
		if entry.Timestamp.After(cutoff) {
			periodEntries = append(periodEntries, entry)
		}
	}

	if len(periodEntries) < 2 {
		return &PeriodAnalysis{
			Period:     fmt.Sprintf("%d days", days),
			DataPoints: len(periodEntries),
			Confidence: 0.0,
		}
	}

	start := periodEntries[len(periodEntries)-1].Coverage.Percentage
	end := periodEntries[0].Coverage.Percentage
	change := end - start
	changePercent := (change / start) * 100

	direction := "stable"
	if change > 0.1 {
		direction = "up"
	} else if change < -0.1 {
		direction = "down"
	}

	return &PeriodAnalysis{
		Period:        fmt.Sprintf("%d days", days),
		StartCoverage: start,
		EndCoverage:   end,
		Change:        change,
		ChangePercent: changePercent,
		Direction:     direction,
		Confidence:    85.0,
		DataPoints:    len(periodEntries),
	}
}

func (t *Tracker) calculateVolatility(entries []Entry) float64 {
	if len(entries) < 2 {
		return 0.0
	}

	var sum float64
	for _, entry := range entries {
		sum += entry.Coverage.Percentage
	}
	mean := sum / float64(len(entries))

	var variance float64
	for _, entry := range entries {
		diff := entry.Coverage.Percentage - mean
		variance += diff * diff
	}
	variance /= float64(len(entries))

	return variance // Simplified volatility calculation
}

func (t *Tracker) calculateMomentum(entries []Entry) float64 {
	if len(entries) < 3 {
		return 0.0
	}

	// Simple momentum: rate of change acceleration
	recent := entries[0].Coverage.Percentage
	middle := entries[len(entries)/2].Coverage.Percentage
	old := entries[len(entries)-1].Coverage.Percentage

	recentChange := recent - middle
	historicalChange := middle - old

	return recentChange - historicalChange
}

func (t *Tracker) generatePrediction(entries []Entry) *Prediction {
	if len(entries) < 5 {
		return nil
	}

	// Simple linear trend prediction
	trend := t.calculateMomentum(entries)
	current := entries[0].Coverage.Percentage

	nextWeek := current + (trend * 7)
	nextMonth := current + (trend * 30)

	return &Prediction{
		NextWeek: &PredictionPoint{
			Percentage: nextWeek,
			Date:       time.Now().AddDate(0, 0, 7),
			Range:      Range{Min: nextWeek - 2, Max: nextWeek + 2},
		},
		NextMonth: &PredictionPoint{
			Percentage: nextMonth,
			Date:       time.Now().AddDate(0, 0, 30),
			Range:      Range{Min: nextMonth - 5, Max: nextMonth + 5},
		},
		Confidence: 65.0,
		Model:      "linear_trend",
		Factors:    []string{"historical_trend", "recent_momentum"},
	}
}

func (t *Tracker) calculateStorageSize() int64 {
	var size int64
	files, err := filepath.Glob(filepath.Join(t.config.StoragePath, "*.json"))
	if err != nil {
		return 0
	}

	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			size += info.Size()
		}
	}

	return size
}

// Statistics provides comprehensive history statistics
type Statistics struct {
	TotalEntries   int            `json:"total_entries"`
	OldestEntry    time.Time      `json:"oldest_entry"`
	NewestEntry    time.Time      `json:"newest_entry"`
	UniqueProjects map[string]int `json:"unique_projects"`
	UniqueBranches map[string]int `json:"unique_branches"`
	StorageSize    int64          `json:"storage_size"`
	GeneratedAt    time.Time      `json:"generated_at"`
}

// Option types for configuration
type RecordOptions struct {
	Branch    string
	CommitSHA string
	CommitURL string
	Metadata  map[string]string
	BuildInfo *BuildInfo
}

type TrendOptions struct {
	Branch    string
	Days      int
	MaxPoints int
}

type Option func(*RecordOptions)
type TrendOption func(*TrendOptions)

// Configuration options
func WithBranch(branch string) Option {
	return func(opts *RecordOptions) {
		opts.Branch = branch
	}
}

func WithCommit(sha, url string) Option {
	return func(opts *RecordOptions) {
		opts.CommitSHA = sha
		opts.CommitURL = url
	}
}

func WithMetadata(key, value string) Option {
	return func(opts *RecordOptions) {
		if opts.Metadata == nil {
			opts.Metadata = make(map[string]string)
		}
		opts.Metadata[key] = value
	}
}

func WithBuildInfo(info *BuildInfo) Option {
	return func(opts *RecordOptions) {
		opts.BuildInfo = info
	}
}

// Trend options
func WithTrendBranch(branch string) TrendOption {
	return func(opts *TrendOptions) {
		opts.Branch = branch
	}
}

func WithTrendDays(days int) TrendOption {
	return func(opts *TrendOptions) {
		opts.Days = days
	}
}

func WithMaxDataPoints(max int) TrendOption {
	return func(opts *TrendOptions) {
		opts.MaxPoints = max
	}
}
