package pages

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CleanupManager handles automatic cleanup and retention policies
type CleanupManager struct {
	BasePath string
	Config   *CleanupConfig
}

// CleanupConfig defines cleanup behavior and retention policies
type CleanupConfig struct {
	// Retention periods
	PRDataRetentionDays     int
	BranchDataRetentionDays int
	ReportRetentionDays     int

	// Size limits
	MaxTotalSizeMB  int64
	MaxFilesPerType int

	// Cleanup behavior
	PreserveMainBranch        bool
	PreserveProtectedBranches []string
	DryRunMode                bool

	// Patterns to match
	PRPattern     *regexp.Regexp
	BranchPattern *regexp.Regexp
}

// CleanupStats contains statistics about a cleanup operation
type CleanupStats struct {
	// Files processed
	TotalFilesScanned  int
	FilesDeleted       int
	DirectoriesDeleted int

	// Size metrics
	TotalSizeScanned int64
	SizeFreed        int64

	// Type breakdown
	PRFilesDeleted     int
	BranchFilesDeleted int
	ReportFilesDeleted int

	// Time metrics
	Duration time.Duration

	// Errors
	Errors []error
}

// ExpiredContent represents content that should be cleaned up
type ExpiredContent struct {
	Path         string
	Type         ContentType
	Age          time.Duration
	Size         int64
	PRNumber     string
	BranchName   string
	LastModified time.Time
}

// ContentType represents the type of content being cleaned
type ContentType string

const (
	ContentTypePR     ContentType = "pr"
	ContentTypeBranch ContentType = "branch"
	ContentTypeReport ContentType = "report"
	ContentTypeBadge  ContentType = "badge"
)

// NewCleanupManager creates a new cleanup manager with default configuration
func NewCleanupManager(basePath string) *CleanupManager {
	return &CleanupManager{
		BasePath: basePath,
		Config: &CleanupConfig{
			PRDataRetentionDays:       30,
			BranchDataRetentionDays:   90,
			ReportRetentionDays:       365,
			MaxTotalSizeMB:            500, // 500MB limit
			MaxFilesPerType:           1000,
			PreserveMainBranch:        true,
			PreserveProtectedBranches: []string{"main", "master", "develop"},
			DryRunMode:                false,
			PRPattern:                 regexp.MustCompile(`^pr/(\d+)$`),
			BranchPattern:             regexp.MustCompile(`^([^/]+)$`),
		},
	}
}

// PerformCleanup executes a comprehensive cleanup operation
func (cm *CleanupManager) PerformCleanup(ctx context.Context, maxAgeDays int) (*CleanupStats, error) {
	startTime := time.Now()

	stats := &CleanupStats{
		Errors: []error{},
	}

	// Override config max age if provided
	if maxAgeDays > 0 {
		cm.Config.PRDataRetentionDays = maxAgeDays
		cm.Config.BranchDataRetentionDays = maxAgeDays
	}

	// Scan for expired content
	expiredContent, err := cm.scanForExpiredContent(ctx)
	if err != nil {
		stats.Errors = append(stats.Errors, fmt.Errorf("failed to scan for expired content: %w", err))
		return stats, err
	}

	stats.TotalFilesScanned = len(expiredContent)

	// Calculate total size to be freed
	for _, content := range expiredContent {
		stats.TotalSizeScanned += content.Size
	}

	// Remove expired content
	if !cm.Config.DryRunMode {
		removedStats, err := cm.removeExpiredContent(ctx, expiredContent)
		if err != nil {
			stats.Errors = append(stats.Errors, err)
		}

		// Merge removal stats
		stats.FilesDeleted = removedStats.FilesDeleted
		stats.DirectoriesDeleted = removedStats.DirectoriesDeleted
		stats.SizeFreed = removedStats.SizeFreed
		stats.PRFilesDeleted = removedStats.PRFilesDeleted
		stats.BranchFilesDeleted = removedStats.BranchFilesDeleted
		stats.ReportFilesDeleted = removedStats.ReportFilesDeleted
	} else {
		// In dry run mode, just calculate what would be removed
		stats.SizeFreed = stats.TotalSizeScanned
		for _, content := range expiredContent {
			stats.FilesDeleted++
			switch content.Type {
			case ContentTypePR:
				stats.PRFilesDeleted++
			case ContentTypeBranch:
				stats.BranchFilesDeleted++
			case ContentTypeReport:
				stats.ReportFilesDeleted++
			}
		}
	}

	stats.Duration = time.Since(startTime)
	return stats, nil
}

// scanForExpiredContent scans the filesystem for content that should be cleaned up
func (cm *CleanupManager) scanForExpiredContent(ctx context.Context) ([]ExpiredContent, error) {
	var expired []ExpiredContent
	cutoffTime := time.Now()

	// Scan PR directories
	prExpired, err := cm.scanPRDirectories(ctx, cutoffTime.AddDate(0, 0, -cm.Config.PRDataRetentionDays))
	if err != nil {
		return nil, fmt.Errorf("failed to scan PR directories: %w", err)
	}
	expired = append(expired, prExpired...)

	// Scan branch directories
	branchExpired, err := cm.scanBranchDirectories(ctx, cutoffTime.AddDate(0, 0, -cm.Config.BranchDataRetentionDays))
	if err != nil {
		return nil, fmt.Errorf("failed to scan branch directories: %w", err)
	}
	expired = append(expired, branchExpired...)

	// Scan report directories
	reportExpired, err := cm.scanReportDirectories(ctx, cutoffTime.AddDate(0, 0, -cm.Config.ReportRetentionDays))
	if err != nil {
		return nil, fmt.Errorf("failed to scan report directories: %w", err)
	}
	expired = append(expired, reportExpired...)

	return expired, nil
}

// scanPRDirectories scans for expired PR-specific content
func (cm *CleanupManager) scanPRDirectories(ctx context.Context, cutoffTime time.Time) ([]ExpiredContent, error) {
	var expired []ExpiredContent

	// Scan PR badges
	prBadgesDir := filepath.Join(cm.BasePath, "badges", "pr")
	if err := cm.scanDirectory(prBadgesDir, ContentTypePR, cutoffTime, &expired); err != nil {
		return nil, fmt.Errorf("failed to scan PR badges: %w", err)
	}

	// Scan PR reports
	prReportsDir := filepath.Join(cm.BasePath, "reports", "pr")
	if err := cm.scanDirectory(prReportsDir, ContentTypePR, cutoffTime, &expired); err != nil {
		return nil, fmt.Errorf("failed to scan PR reports: %w", err)
	}

	return expired, nil
}

// scanBranchDirectories scans for expired branch-specific content
func (cm *CleanupManager) scanBranchDirectories(ctx context.Context, cutoffTime time.Time) ([]ExpiredContent, error) {
	var expired []ExpiredContent

	// Scan branch badges
	badgesDir := filepath.Join(cm.BasePath, "badges")
	entries, err := os.ReadDir(badgesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return expired, nil // Directory doesn't exist yet
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".svg") {
			continue
		}

		// Skip PR badges (handled separately)
		if entry.Name() == "pr" {
			continue
		}

		branchName := strings.TrimSuffix(entry.Name(), ".svg")

		// Skip protected branches
		if cm.shouldPreserveBranch(branchName) {
			continue
		}

		filePath := filepath.Join(badgesDir, entry.Name())
		if cm.isFileExpired(filePath, cutoffTime) {
			size := cm.getFileSize(filePath)
			expired = append(expired, ExpiredContent{
				Path:         filePath,
				Type:         ContentTypeBadge,
				BranchName:   branchName,
				Size:         size,
				LastModified: cm.getFileModTime(filePath),
			})
		}
	}

	// Scan branch reports
	reportsDir := filepath.Join(cm.BasePath, "reports")
	entries, err = os.ReadDir(reportsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return expired, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "pr" {
			continue
		}

		branchName := entry.Name()

		// Skip protected branches
		if cm.shouldPreserveBranch(branchName) {
			continue
		}

		dirPath := filepath.Join(reportsDir, entry.Name())
		if cm.isDirExpired(dirPath, cutoffTime) {
			size := cm.getDirSize(dirPath)
			expired = append(expired, ExpiredContent{
				Path:         dirPath,
				Type:         ContentTypeReport,
				BranchName:   branchName,
				Size:         size,
				LastModified: cm.getDirModTime(dirPath),
			})
		}
	}

	return expired, nil
}

// scanReportDirectories scans for expired report content
func (cm *CleanupManager) scanReportDirectories(ctx context.Context, cutoffTime time.Time) ([]ExpiredContent, error) {
	var expired []ExpiredContent

	// This would scan for very old reports that exceed the report retention period
	// For now, we'll focus on PR and branch cleanup

	return expired, nil
}

// scanDirectory scans a directory for expired files
func (cm *CleanupManager) scanDirectory(dirPath string, contentType ContentType, cutoffTime time.Time, expired *[]ExpiredContent) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist yet
		}
		return err
	}

	for _, entry := range entries {
		filePath := filepath.Join(dirPath, entry.Name())

		var prNumber, branchName string
		var size int64
		var lastMod time.Time

		if entry.IsDir() {
			if !cm.isDirExpired(filePath, cutoffTime) {
				continue
			}
			size = cm.getDirSize(filePath)
			lastMod = cm.getDirModTime(filePath)

			// Extract PR number or branch name from directory name
			if contentType == ContentTypePR {
				if matches := cm.Config.PRPattern.FindStringSubmatch(entry.Name()); len(matches) > 1 {
					prNumber = matches[1]
				}
			} else {
				branchName = entry.Name()
			}
		} else {
			if !cm.isFileExpired(filePath, cutoffTime) {
				continue
			}
			size = cm.getFileSize(filePath)
			lastMod = cm.getFileModTime(filePath)

			// Extract PR number from filename for badges
			if contentType == ContentTypePR && strings.HasSuffix(entry.Name(), ".svg") {
				prNumber = strings.TrimSuffix(entry.Name(), ".svg")
			}
		}

		*expired = append(*expired, ExpiredContent{
			Path:         filePath,
			Type:         contentType,
			PRNumber:     prNumber,
			BranchName:   branchName,
			Size:         size,
			LastModified: lastMod,
			Age:          time.Since(lastMod),
		})
	}

	return nil
}

// removeExpiredContent removes the identified expired content
func (cm *CleanupManager) removeExpiredContent(ctx context.Context, expired []ExpiredContent) (*CleanupStats, error) {
	stats := &CleanupStats{}

	for _, content := range expired {
		if err := cm.removeContent(content); err != nil {
			stats.Errors = append(stats.Errors, fmt.Errorf("failed to remove %s: %w", content.Path, err))
			continue
		}

		stats.FilesDeleted++
		stats.SizeFreed += content.Size

		switch content.Type {
		case ContentTypePR:
			stats.PRFilesDeleted++
		case ContentTypeBranch:
			stats.BranchFilesDeleted++
		case ContentTypeReport:
			stats.ReportFilesDeleted++
		}

		// Check if it's a directory
		if info, err := os.Stat(content.Path); err == nil && info.IsDir() {
			stats.DirectoriesDeleted++
		}
	}

	return stats, nil
}

// removeContent removes a single piece of content
func (cm *CleanupManager) removeContent(content ExpiredContent) error {
	return os.RemoveAll(content.Path)
}

// Helper methods

func (cm *CleanupManager) shouldPreserveBranch(branchName string) bool {
	if cm.Config.PreserveMainBranch && (branchName == "main" || branchName == "master") {
		return true
	}

	for _, protected := range cm.Config.PreserveProtectedBranches {
		if branchName == protected {
			return true
		}
	}

	return false
}

func (cm *CleanupManager) isFileExpired(filePath string, cutoffTime time.Time) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return info.ModTime().Before(cutoffTime)
}

func (cm *CleanupManager) isDirExpired(dirPath string, cutoffTime time.Time) bool {
	// For directories, check the most recent file modification time
	var mostRecent time.Time

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.ModTime().After(mostRecent) {
			mostRecent = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return false
	}

	return mostRecent.Before(cutoffTime)
}

func (cm *CleanupManager) getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (cm *CleanupManager) getDirSize(dirPath string) int64 {
	var totalSize int64

	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize
}

func (cm *CleanupManager) getFileModTime(filePath string) time.Time {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func (cm *CleanupManager) getDirModTime(dirPath string) time.Time {
	var mostRecent time.Time

	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.ModTime().After(mostRecent) {
			mostRecent = info.ModTime()
		}
		return nil
	})

	return mostRecent
}

// FormatCleanupStats formats cleanup statistics for display
func FormatCleanupStats(stats *CleanupStats, dryRun bool) string {
	action := "Removed"
	if dryRun {
		action = "Would remove"
	}

	var result strings.Builder

	if stats.FilesDeleted > 0 {
		result.WriteString(fmt.Sprintf("%s %d files (%s)\n",
			action, stats.FilesDeleted, formatBytes(stats.SizeFreed)))

		if stats.PRFilesDeleted > 0 {
			result.WriteString(fmt.Sprintf("  - %d PR-related files\n", stats.PRFilesDeleted))
		}
		if stats.BranchFilesDeleted > 0 {
			result.WriteString(fmt.Sprintf("  - %d branch-related files\n", stats.BranchFilesDeleted))
		}
		if stats.ReportFilesDeleted > 0 {
			result.WriteString(fmt.Sprintf("  - %d report files\n", stats.ReportFilesDeleted))
		}
		if stats.DirectoriesDeleted > 0 {
			result.WriteString(fmt.Sprintf("  - %d directories\n", stats.DirectoriesDeleted))
		}
	} else {
		result.WriteString("No expired content found\n")
	}

	if len(stats.Errors) > 0 {
		result.WriteString(fmt.Sprintf("\nErrors encountered: %d\n", len(stats.Errors)))
	}

	result.WriteString(fmt.Sprintf("Cleanup completed in %v\n", stats.Duration))

	return result.String()
}

// formatBytes formats byte counts in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
