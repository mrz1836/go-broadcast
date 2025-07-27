// Package badge provides PR-specific badge generation with unique naming and organization
package badge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PRBadgeManager handles PR-specific badge generation and management
type PRBadgeManager struct {
	generator *Generator
	config    *PRBadgeConfig
}

// PRBadgeConfig holds configuration for PR badge generation
type PRBadgeConfig struct {
	// Storage settings
	OutputBasePath       string // Base path for badge storage
	CreateDirectories    bool   // Auto-create directories
	DirectoryPermissions os.FileMode
	FilePermissions      os.FileMode
	
	// Naming patterns
	CoveragePattern      string // Pattern for coverage badges
	TrendPattern         string // Pattern for trend badges  
	StatusPattern        string // Pattern for status badges
	ComparisonPattern    string // Pattern for comparison badges
	
	// Badge settings
	Styles               []string // Available badge styles
	DefaultStyle         string   // Default badge style
	IncludeTimestamp     bool     // Include timestamp in badges
	IncludeBranch        bool     // Include branch name in badges
	
	// Cleanup settings
	EnableCleanup        bool     // Enable automatic cleanup
	MaxAge               time.Duration // Maximum age for PR badges
	CleanupOnMerge       bool     // Cleanup badges when PR is merged
	
	// Generation settings
	GenerateMultipleStyles bool   // Generate badges in multiple styles
	GenerateThumbnails     bool   // Generate thumbnail versions
	GenerateRetina         bool   // Generate retina (2x) versions
}

// PRBadgeType represents different types of PR badges
type PRBadgeType string

const (
	PRBadgeCoverage    PRBadgeType = "coverage"
	PRBadgeTrend       PRBadgeType = "trend"
	PRBadgeStatus      PRBadgeType = "status"
	PRBadgeComparison  PRBadgeType = "comparison"
	PRBadgeDiff        PRBadgeType = "diff"
	PRBadgeQuality     PRBadgeType = "quality"
)

// PRBadgeRequest represents a request to generate PR badges
type PRBadgeRequest struct {
	// PR information
	Repository   string
	Owner        string
	PRNumber     int
	Branch       string
	CommitSHA    string
	BaseBranch   string
	
	// Coverage data
	Coverage     float64
	BaseCoverage float64
	Trend        TrendDirection
	
	// Quality metrics
	QualityGrade string
	RiskLevel    string
	
	// Badge configuration
	Types        []PRBadgeType
	Styles       []string
	CustomLabels map[PRBadgeType]string
	
	// Metadata
	Timestamp    time.Time
	Author       string
}

// PRBadgeResult represents the result of PR badge generation
type PRBadgeResult struct {
	// Generated badges
	Badges       map[PRBadgeType][]BadgeInfo
	
	// URLs and paths
	BaseURL      string
	LocalPaths   map[PRBadgeType][]string
	PublicURLs   map[PRBadgeType][]string
	
	// Metadata
	GeneratedAt  time.Time
	TotalBadges  int
	Errors       []error
}

// BadgeInfo contains information about a generated badge
type BadgeInfo struct {
	Type         PRBadgeType
	Style        string
	FilePath     string
	PublicURL    string
	Size         int64
	Dimensions   BadgeDimensions
	Metadata     BadgeMetadata
}

// BadgeDimensions represents badge dimensions
type BadgeDimensions struct {
	Width  int
	Height int
}

// BadgeMetadata contains badge metadata
type BadgeMetadata struct {
	GeneratedAt   time.Time
	Version       string
	Coverage      float64
	BaseCoverage  float64
	Change        float64
	QualityGrade  string
	PRNumber      int
	Branch        string
	CommitSHA     string
}

// NewPRBadgeManager creates a new PR badge manager
func NewPRBadgeManager(generator *Generator, config *PRBadgeConfig) *PRBadgeManager {
	if config == nil {
		config = &PRBadgeConfig{
			OutputBasePath:         "./coverage-badges",
			CreateDirectories:      true,
			DirectoryPermissions:   0755,
			FilePermissions:        0644,
			CoveragePattern:        "badge-coverage-{style}.svg",
			TrendPattern:           "badge-trend-{style}.svg",
			StatusPattern:          "badge-status-{style}.svg",
			ComparisonPattern:      "badge-comparison-{style}.svg",
			Styles:                 []string{"flat", "flat-square", "for-the-badge"},
			DefaultStyle:           "flat",
			IncludeTimestamp:       false,
			IncludeBranch:          true,
			EnableCleanup:          true,
			MaxAge:                 30 * 24 * time.Hour, // 30 days
			CleanupOnMerge:         true,
			GenerateMultipleStyles: true,
			GenerateThumbnails:     false,
			GenerateRetina:         false,
		}
	}
	
	return &PRBadgeManager{
		generator: generator,
		config:    config,
	}
}

// GeneratePRBadges generates all requested PR badges
func (m *PRBadgeManager) GeneratePRBadges(ctx context.Context, request *PRBadgeRequest) (*PRBadgeResult, error) {
	result := &PRBadgeResult{
		Badges:     make(map[PRBadgeType][]BadgeInfo),
		LocalPaths: make(map[PRBadgeType][]string),
		PublicURLs: make(map[PRBadgeType][]string),
		BaseURL:    m.buildBaseURL(request.Owner, request.Repository, request.PRNumber),
		GeneratedAt: time.Now(),
	}
	
	// Create output directory
	outputDir := m.buildOutputDirectory(request)
	if m.config.CreateDirectories {
		if err := os.MkdirAll(outputDir, m.config.DirectoryPermissions); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}
	
	// Determine styles to generate
	styles := request.Styles
	if len(styles) == 0 {
		if m.config.GenerateMultipleStyles {
			styles = m.config.Styles
		} else {
			styles = []string{m.config.DefaultStyle}
		}
	}
	
	// Generate each requested badge type
	for _, badgeType := range request.Types {
		for _, style := range styles {
			badgeInfo, err := m.generateSingleBadge(ctx, request, badgeType, style, outputDir)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to generate %s badge in %s style: %w", badgeType, style, err))
				continue
			}
			
			result.Badges[badgeType] = append(result.Badges[badgeType], *badgeInfo)
			result.LocalPaths[badgeType] = append(result.LocalPaths[badgeType], badgeInfo.FilePath)
			result.PublicURLs[badgeType] = append(result.PublicURLs[badgeType], badgeInfo.PublicURL)
			result.TotalBadges++
		}
	}
	
	return result, nil
}

// generateSingleBadge generates a single badge
func (m *PRBadgeManager) generateSingleBadge(ctx context.Context, request *PRBadgeRequest, badgeType PRBadgeType, style, outputDir string) (*BadgeInfo, error) {
	// Generate badge content based on type
	var badgeData []byte
	var err error
	var label string
	var coverage float64
	
	switch badgeType {
	case PRBadgeCoverage:
		label = m.getCustomLabel(request, badgeType, "coverage")
		coverage = request.Coverage
		badgeData, err = m.generator.Generate(ctx, coverage, 
			WithStyle(style), 
			WithLabel(label))
			
	case PRBadgeTrend:
		label = m.getCustomLabel(request, badgeType, "trend")
		badgeData, err = m.generator.GenerateTrendBadge(ctx, request.Coverage, request.BaseCoverage,
			WithStyle(style),
			WithLabel(label))
			
	case PRBadgeStatus:
		label = m.getCustomLabel(request, badgeType, "status")
		badgeData, err = m.generateStatusBadge(ctx, request, style, label)
		coverage = request.Coverage
		
	case PRBadgeComparison:
		label = m.getCustomLabel(request, badgeType, "comparison")
		badgeData, err = m.generateComparisonBadge(ctx, request, style, label)
		coverage = request.Coverage
		
	case PRBadgeDiff:
		label = m.getCustomLabel(request, badgeType, "diff")
		badgeData, err = m.generateDiffBadge(ctx, request, style, label)
		coverage = request.Coverage
		
	case PRBadgeQuality:
		label = m.getCustomLabel(request, badgeType, "quality")
		badgeData, err = m.generateQualityBadge(ctx, request, style, label)
		coverage = request.Coverage
		
	default:
		return nil, fmt.Errorf("unsupported badge type: %s", badgeType)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to generate badge content: %w", err)
	}
	
	// Build file path
	fileName := m.buildFileName(badgeType, style, request)
	filePath := filepath.Join(outputDir, fileName)
	
	// Write badge to file
	if err := os.WriteFile(filePath, badgeData, m.config.FilePermissions); err != nil {
		return nil, fmt.Errorf("failed to write badge file: %w", err)
	}
	
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	// Build public URL
	publicURL := m.buildPublicURL(request.Owner, request.Repository, request.PRNumber, fileName)
	
	// Calculate dimensions (simplified)
	dimensions := m.calculateBadgeDimensions(label, fmt.Sprintf("%.1f%%", coverage), style)
	
	// Create badge metadata
	metadata := BadgeMetadata{
		GeneratedAt:  time.Now(),
		Version:      "2.0",
		Coverage:     request.Coverage,
		BaseCoverage: request.BaseCoverage,
		Change:       request.Coverage - request.BaseCoverage,
		QualityGrade: request.QualityGrade,
		PRNumber:     request.PRNumber,
		Branch:       request.Branch,
		CommitSHA:    request.CommitSHA,
	}
	
	return &BadgeInfo{
		Type:       badgeType,
		Style:      style,
		FilePath:   filePath,
		PublicURL:  publicURL,
		Size:       fileInfo.Size(),
		Dimensions: dimensions,
		Metadata:   metadata,
	}, nil
}

// generateStatusBadge generates a status badge
func (m *PRBadgeManager) generateStatusBadge(ctx context.Context, request *PRBadgeRequest, style, label string) ([]byte, error) {
	var message string
	var color string
	
	coverage := request.Coverage
	switch {
	case coverage >= 90:
		message = "excellent"
		color = "#3fb950"
	case coverage >= 80:
		message = "good"
		color = "#7c3aed"
	case coverage >= 70:
		message = "fair"
		color = "#d29922"
	case coverage >= 60:
		message = "poor"
		color = "#fb8500"
	default:
		message = "critical"
		color = "#f85149"
	}
	
	badgeData := BadgeData{
		Label:     label,
		Message:   message,
		Color:     color,
		Style:     style,
		AriaLabel: fmt.Sprintf("Coverage status: %s", message),
	}
	
	return m.generator.renderSVG(ctx, badgeData)
}

// generateComparisonBadge generates a comparison badge
func (m *PRBadgeManager) generateComparisonBadge(ctx context.Context, request *PRBadgeRequest, style, label string) ([]byte, error) {
	diff := request.Coverage - request.BaseCoverage
	var message string
	var color string
	
	if diff > 0.1 {
		message = fmt.Sprintf("+%.1f%%", diff)
		color = "#3fb950"
	} else if diff < -0.1 {
		message = fmt.Sprintf("%.1f%%", diff)
		color = "#f85149"
	} else {
		message = "±0.0%"
		color = "#8b949e"
	}
	
	badgeData := BadgeData{
		Label:     label,
		Message:   message,
		Color:     color,
		Style:     style,
		AriaLabel: fmt.Sprintf("Coverage comparison: %s", message),
	}
	
	return m.generator.renderSVG(ctx, badgeData)
}

// generateDiffBadge generates a diff badge showing change magnitude
func (m *PRBadgeManager) generateDiffBadge(ctx context.Context, request *PRBadgeRequest, style, label string) ([]byte, error) {
	diff := request.Coverage - request.BaseCoverage
	absDiff := diff
	if absDiff < 0 {
		absDiff = -absDiff
	}
	
	var message string
	var color string
	
	switch {
	case absDiff >= 5.0:
		message = "major"
		color = "#f85149"
	case absDiff >= 2.0:
		message = "moderate"
		color = "#fb8500"
	case absDiff >= 0.5:
		message = "minor"
		color = "#d29922"
	default:
		message = "stable"
		color = "#3fb950"
	}
	
	if diff > 0 && absDiff >= 0.5 {
		message = "+" + message
		color = "#3fb950"
	} else if diff < 0 && absDiff >= 0.5 {
		message = "-" + message
		color = "#f85149"
	}
	
	badgeData := BadgeData{
		Label:     label,
		Message:   message,
		Color:     color,
		Style:     style,
		AriaLabel: fmt.Sprintf("Coverage change: %s", message),
	}
	
	return m.generator.renderSVG(ctx, badgeData)
}

// generateQualityBadge generates a quality grade badge
func (m *PRBadgeManager) generateQualityBadge(ctx context.Context, request *PRBadgeRequest, style, label string) ([]byte, error) {
	grade := request.QualityGrade
	if grade == "" {
		// Calculate grade based on coverage
		coverage := request.Coverage
		switch {
		case coverage >= 95:
			grade = "A+"
		case coverage >= 90:
			grade = "A"
		case coverage >= 85:
			grade = "B+"
		case coverage >= 80:
			grade = "B"
		case coverage >= 70:
			grade = "C"
		case coverage >= 60:
			grade = "D"
		default:
			grade = "F"
		}
	}
	
	// Set color based on grade
	var color string
	switch grade {
	case "A+", "A":
		color = "#3fb950"
	case "B+", "B":
		color = "#7c3aed"
	case "C":
		color = "#d29922"
	case "D":
		color = "#fb8500"
	case "F":
		color = "#f85149"
	default:
		color = "#8b949e"
	}
	
	badgeData := BadgeData{
		Label:     label,
		Message:   grade,
		Color:     color,
		Style:     style,
		AriaLabel: fmt.Sprintf("Coverage quality grade: %s", grade),
	}
	
	return m.generator.renderSVG(ctx, badgeData)
}

// Helper methods

func (m *PRBadgeManager) buildOutputDirectory(request *PRBadgeRequest) string {
	return filepath.Join(m.config.OutputBasePath, "pr", fmt.Sprintf("%d", request.PRNumber))
}

func (m *PRBadgeManager) buildFileName(badgeType PRBadgeType, style string, request *PRBadgeRequest) string {
	var pattern string
	
	switch badgeType {
	case PRBadgeCoverage:
		pattern = m.config.CoveragePattern
	case PRBadgeTrend:
		pattern = m.config.TrendPattern
	case PRBadgeStatus:
		pattern = m.config.StatusPattern
	case PRBadgeComparison:
		pattern = m.config.ComparisonPattern
	default:
		pattern = fmt.Sprintf("badge-%s-{style}.svg", badgeType)
	}
	
	fileName := strings.ReplaceAll(pattern, "{style}", style)
	fileName = strings.ReplaceAll(fileName, "{type}", string(badgeType))
	fileName = strings.ReplaceAll(fileName, "{pr}", fmt.Sprintf("%d", request.PRNumber))
	fileName = strings.ReplaceAll(fileName, "{branch}", request.Branch)
	
	if m.config.IncludeTimestamp {
		timestamp := request.Timestamp.Format("20060102-150405")
		fileName = strings.ReplaceAll(fileName, "{timestamp}", timestamp)
	}
	
	return fileName
}

func (m *PRBadgeManager) buildBaseURL(owner, repository string, prNumber int) string {
	return fmt.Sprintf("https://%s.github.io/%s/coverage/pr/%d", owner, repository, prNumber)
}

func (m *PRBadgeManager) buildPublicURL(owner, repository string, prNumber int, fileName string) string {
	return fmt.Sprintf("%s/%s", m.buildBaseURL(owner, repository, prNumber), fileName)
}

func (m *PRBadgeManager) getCustomLabel(request *PRBadgeRequest, badgeType PRBadgeType, defaultLabel string) string {
	if request.CustomLabels != nil {
		if label, exists := request.CustomLabels[badgeType]; exists {
			return label
		}
	}
	return defaultLabel
}

func (m *PRBadgeManager) calculateBadgeDimensions(label, message, style string) BadgeDimensions {
	// Simplified calculation - in reality this would be more sophisticated
	labelWidth := len(label) * 6
	messageWidth := len(message) * 6
	totalWidth := labelWidth + messageWidth + 20 // padding
	
	height := 20
	if style == "for-the-badge" {
		height = 28
	}
	
	return BadgeDimensions{
		Width:  totalWidth,
		Height: height,
	}
}

// CleanupPRBadges removes old PR badges based on configuration
func (m *PRBadgeManager) CleanupPRBadges(ctx context.Context, owner, repository string, prNumber int) error {
	if !m.config.EnableCleanup {
		return nil
	}
	
	prDir := filepath.Join(m.config.OutputBasePath, "pr", fmt.Sprintf("%d", prNumber))
	
	// Check if directory exists
	if _, err := os.Stat(prDir); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}
	
	// Remove the entire PR directory
	if err := os.RemoveAll(prDir); err != nil {
		return fmt.Errorf("failed to remove PR badge directory: %w", err)
	}
	
	return nil
}

// GetPRBadgeInfo returns information about existing PR badges
func (m *PRBadgeManager) GetPRBadgeInfo(ctx context.Context, owner, repository string, prNumber int) (*PRBadgeResult, error) {
	prDir := filepath.Join(m.config.OutputBasePath, "pr", fmt.Sprintf("%d", prNumber))
	
	result := &PRBadgeResult{
		Badges:     make(map[PRBadgeType][]BadgeInfo),
		LocalPaths: make(map[PRBadgeType][]string),
		PublicURLs: make(map[PRBadgeType][]string),
		BaseURL:    m.buildBaseURL(owner, repository, prNumber),
	}
	
	// Check if directory exists
	if _, err := os.Stat(prDir); os.IsNotExist(err) {
		return result, nil // No badges exist
	}
	
	// Read directory contents
	entries, err := os.ReadDir(prDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read PR badge directory: %w", err)
	}
	
	// Process each badge file
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".svg") {
			continue
		}
		
		filePath := filepath.Join(prDir, entry.Name())
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}
		
		// Parse badge type and style from filename
		badgeType, style := m.parseBadgeFileName(entry.Name())
		if badgeType == "" {
			continue
		}
		
		publicURL := m.buildPublicURL(owner, repository, prNumber, entry.Name())
		
		badgeInfo := BadgeInfo{
			Type:      PRBadgeType(badgeType),
			Style:     style,
			FilePath:  filePath,
			PublicURL: publicURL,
			Size:      fileInfo.Size(),
			Metadata: BadgeMetadata{
				GeneratedAt: fileInfo.ModTime(),
				PRNumber:    prNumber,
			},
		}
		
		result.Badges[PRBadgeType(badgeType)] = append(result.Badges[PRBadgeType(badgeType)], badgeInfo)
		result.LocalPaths[PRBadgeType(badgeType)] = append(result.LocalPaths[PRBadgeType(badgeType)], filePath)
		result.PublicURLs[PRBadgeType(badgeType)] = append(result.PublicURLs[PRBadgeType(badgeType)], publicURL)
		result.TotalBadges++
	}
	
	return result, nil
}

// parseBadgeFileName parses badge type and style from filename
func (m *PRBadgeManager) parseBadgeFileName(fileName string) (string, string) {
	// Remove .svg extension
	name := strings.TrimSuffix(fileName, ".svg")
	
	// Extract style (last part after last dash)
	parts := strings.Split(name, "-")
	if len(parts) < 3 {
		return "", ""
	}
	
	style := parts[len(parts)-1]
	
	// Extract badge type
	if strings.HasPrefix(name, "badge-") {
		typeParts := parts[1 : len(parts)-1]
		badgeType := strings.Join(typeParts, "-")
		return badgeType, style
	}
	
	return "", ""
}

// GenerateStandardPRBadges generates a standard set of PR badges
func (m *PRBadgeManager) GenerateStandardPRBadges(ctx context.Context, request *PRBadgeRequest) (*PRBadgeResult, error) {
	// Set standard badge types
	request.Types = []PRBadgeType{
		PRBadgeCoverage,
		PRBadgeTrend,
		PRBadgeStatus,
		PRBadgeComparison,
	}
	
	// Add quality badge if we have a grade
	if request.QualityGrade != "" {
		request.Types = append(request.Types, PRBadgeQuality)
	}
	
	return m.GeneratePRBadges(ctx, request)
}