package state

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"gopkg.in/yaml.v3"
)

// PR metadata errors
var (
	ErrPRNoDescription       = errors.New("PR has no description")
	ErrPRNoMetadataBlock     = errors.New("no metadata block found in PR description")
	ErrPRMetadataNotClosed   = errors.New("metadata block not properly closed")
	ErrPRMissingSyncMetadata = errors.New("enhanced metadata missing sync_metadata section")
)

// PRMetadata represents metadata stored in PR descriptions (legacy format)
type PRMetadata struct {
	// SourceCommit is the commit SHA this PR was created from
	SourceCommit string `yaml:"source_commit"`

	// SourceRepo is the source repository
	SourceRepo string `yaml:"source_repo"`

	// SourceBranch is the source branch
	SourceBranch string `yaml:"source_branch"`

	// CreatedAt is when the sync was initiated
	CreatedAt time.Time `yaml:"created_at"`

	// Files is the list of files being synced
	Files []string `yaml:"files"`

	// TransformsApplied lists which transforms were applied
	TransformsApplied []string `yaml:"transforms_applied,omitempty"`
}

// EnhancedPRMetadata represents the new enhanced metadata format with directory sync support
type EnhancedPRMetadata struct {
	// SyncMetadata contains core sync information
	SyncMetadata *SyncMetadataInfo `yaml:"sync_metadata"`

	// Files contains individual file mappings
	Files []FileMapping `yaml:"files,omitempty"`

	// Directories contains directory mappings with detailed metrics
	Directories []DirectoryMapping `yaml:"directories,omitempty"`

	// Performance contains aggregate performance metrics
	Performance *PerformanceInfo `yaml:"performance,omitempty"`
}

// SyncMetadataInfo contains core synchronization metadata
type SyncMetadataInfo struct {
	// SourceRepo is the source repository
	SourceRepo string `yaml:"source_repo"`

	// SourceCommit is the source commit SHA
	SourceCommit string `yaml:"source_commit"`

	// TargetRepo is the target repository
	TargetRepo string `yaml:"target_repo"`

	// SyncCommit is the commit SHA of the sync operation
	SyncCommit string `yaml:"sync_commit,omitempty"`

	// SyncTime is when the sync was performed
	SyncTime time.Time `yaml:"sync_time"`
}

// FileMapping represents an individual file sync mapping
type FileMapping struct {
	// Source is the source file path
	Source string `yaml:"src"`

	// Destination is the destination file path
	Destination string `yaml:"dest"`

	// From indicates the source type ("file" for individual files)
	From string `yaml:"from"`
}

// DirectoryMapping represents a directory sync mapping with metrics
type DirectoryMapping struct {
	// Source is the source directory path
	Source string `yaml:"src"`

	// Destination is the destination directory path
	Destination string `yaml:"dest"`

	// Excluded contains glob patterns that were excluded
	Excluded []string `yaml:"excluded,omitempty"`

	// FilesSynced is the number of files successfully synced
	FilesSynced int `yaml:"files_synced"`

	// FilesExcluded is the number of files excluded by patterns
	FilesExcluded int `yaml:"files_excluded"`

	// ProcessingTimeMs is the time taken to process this directory in milliseconds
	ProcessingTimeMs int64 `yaml:"processing_time_ms"`
}

// PerformanceInfo contains aggregate performance metrics
type PerformanceInfo struct {
	// TotalSyncTimeMs is the total sync time in milliseconds
	TotalSyncTimeMs int64 `yaml:"total_sync_time_ms"`

	// TotalFilesProcessed is the total number of files processed
	TotalFilesProcessed int `yaml:"total_files_processed"`

	// TotalFilesChanged is the number of files that were changed
	TotalFilesChanged int `yaml:"total_files_changed"`

	// TotalFilesSkipped is the number of files skipped
	TotalFilesSkipped int `yaml:"total_files_skipped"`

	// APICallsSaved is the number of API calls saved through optimizations
	APICallsSaved int `yaml:"api_calls_saved"`

	// CacheHits is the number of cache hits
	CacheHits int `yaml:"cache_hits"`

	// CacheMisses is the number of cache misses
	CacheMisses int `yaml:"cache_misses"`
}

// ExtractPRMetadata extracts sync metadata from a PR description (supports both old and new formats)
func ExtractPRMetadata(pr gh.PR) (*PRMetadata, error) {
	if pr.Body == "" {
		return nil, ErrPRNoDescription
	}

	// Try to extract enhanced metadata first
	enhanced, err := ExtractEnhancedPRMetadata(pr)
	if err == nil {
		// Convert enhanced metadata to legacy format for backward compatibility
		return convertEnhancedToLegacy(enhanced), nil
	}

	// Fall back to legacy format extraction
	yamlContent, err := extractMetadataYAML(pr.Body, "<!-- go-broadcast:metadata")
	if err != nil {
		return nil, err
	}

	// Parse the YAML as legacy format
	var metadata PRMetadata
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata YAML: %w", err)
	}

	return &metadata, nil
}

// ExtractEnhancedPRMetadata extracts enhanced metadata from a PR description
func ExtractEnhancedPRMetadata(pr gh.PR) (*EnhancedPRMetadata, error) {
	if pr.Body == "" {
		return nil, ErrPRNoDescription
	}

	// Try new format first
	yamlContent, err := extractMetadataYAML(pr.Body, "<!-- go-broadcast-metadata")
	if err != nil {
		// Try alternative new format marker
		yamlContent, err = extractMetadataYAML(pr.Body, "<!-- go-broadcast:metadata")
		if err != nil {
			return nil, err
		}
	}

	// Parse as enhanced format
	var enhanced EnhancedPRMetadata
	if err := yaml.Unmarshal([]byte(yamlContent), &enhanced); err != nil {
		return nil, fmt.Errorf("failed to parse enhanced metadata YAML: %w", err)
	}

	// Validate required fields
	if enhanced.SyncMetadata == nil {
		return nil, ErrPRMissingSyncMetadata
	}

	return &enhanced, nil
}

// extractMetadataYAML extracts YAML content from metadata blocks
func extractMetadataYAML(body, startMarker string) (string, error) {
	endMarker := "-->"

	startIdx := strings.Index(body, startMarker)
	if startIdx == -1 {
		return "", ErrPRNoMetadataBlock
	}

	// Find the end of the metadata block
	metadataStart := startIdx + len(startMarker)

	endIdx := strings.Index(body[metadataStart:], endMarker)
	if endIdx == -1 {
		return "", ErrPRMetadataNotClosed
	}

	// Extract the YAML content
	yamlContent := strings.TrimSpace(body[metadataStart : metadataStart+endIdx])
	return yamlContent, nil
}

// convertEnhancedToLegacy converts enhanced metadata to legacy format for backward compatibility
func convertEnhancedToLegacy(enhanced *EnhancedPRMetadata) *PRMetadata {
	legacy := &PRMetadata{
		SourceCommit: enhanced.SyncMetadata.SourceCommit,
		SourceRepo:   enhanced.SyncMetadata.SourceRepo,
		CreatedAt:    enhanced.SyncMetadata.SyncTime,
		// SourceBranch is not available in enhanced format, leave empty
		SourceBranch: "",
	}

	// Collect all files from both individual files and directories
	expectedSize := len(enhanced.Files) + len(enhanced.Directories)
	allFiles := make([]string, 0, expectedSize)

	// Add individual files
	for _, file := range enhanced.Files {
		allFiles = append(allFiles, file.Destination)
	}

	// Add directory files (approximation - we don't have individual file lists)
	for _, dir := range enhanced.Directories {
		// Add the directory itself as a placeholder
		allFiles = append(allFiles, dir.Destination+"/*")
	}

	legacy.Files = allFiles

	// TransformsApplied is not tracked in enhanced format
	legacy.TransformsApplied = []string{}

	return legacy
}

// FormatPRMetadata formats metadata for inclusion in a PR description (legacy format)
func FormatPRMetadata(metadata *PRMetadata) string {
	yamlBytes, err := yaml.Marshal(metadata)
	if err != nil {
		// Fallback to a simple format if YAML marshaling fails
		return fmt.Sprintf("<!-- go-broadcast:metadata\nsource_commit: %s\n-->", metadata.SourceCommit)
	}

	return fmt.Sprintf("<!-- go-broadcast:metadata\n%s-->", string(yamlBytes))
}

// FormatEnhancedPRMetadata formats enhanced metadata for inclusion in a PR description
func FormatEnhancedPRMetadata(metadata *EnhancedPRMetadata) string {
	yamlBytes, err := yaml.Marshal(metadata)
	if err != nil {
		// Fallback to a simple format if YAML marshaling fails
		if metadata.SyncMetadata != nil {
			return fmt.Sprintf("<!-- go-broadcast-metadata\nsync_metadata:\n  source_commit: %s\n-->", metadata.SyncMetadata.SourceCommit)
		}
		return "<!-- go-broadcast-metadata\nsync_metadata: {}\n-->"
	}

	return fmt.Sprintf("<!-- go-broadcast-metadata\n%s-->", string(yamlBytes))
}

// GeneratePRDescription generates a complete PR description with metadata (legacy format)
func GeneratePRDescription(metadata *PRMetadata, summary string) string {
	var builder strings.Builder

	// Add the summary first
	builder.WriteString(summary)
	builder.WriteString("\n\n")

	// Add sync details
	builder.WriteString("## Sync Details\n\n")
	builder.WriteString(fmt.Sprintf("- **Source**: `%s` @ `%s`\n", metadata.SourceRepo, metadata.SourceBranch))
	builder.WriteString(fmt.Sprintf("- **Commit**: `%s`\n", metadata.SourceCommit))
	builder.WriteString(fmt.Sprintf("- **Created**: %s\n", metadata.CreatedAt.Format(time.RFC3339)))

	if len(metadata.Files) > 0 {
		builder.WriteString("\n### Files Synced\n")

		for _, file := range metadata.Files {
			builder.WriteString(fmt.Sprintf("- `%s`\n", file))
		}
	}

	if len(metadata.TransformsApplied) > 0 {
		builder.WriteString("\n### Transforms Applied\n")

		for _, transform := range metadata.TransformsApplied {
			builder.WriteString(fmt.Sprintf("- %s\n", transform))
		}
	}

	// Add metadata block at the end
	builder.WriteString("\n\n")
	builder.WriteString(FormatPRMetadata(metadata))

	return builder.String()
}

// GenerateEnhancedPRDescription generates a complete PR description with enhanced metadata
func GenerateEnhancedPRDescription(metadata *EnhancedPRMetadata, summary string) string {
	var builder strings.Builder

	// Add the summary first
	builder.WriteString(summary)
	builder.WriteString("\n\n")

	// Add sync details
	builder.WriteString("## Sync Details\n\n")
	if metadata.SyncMetadata != nil {
		builder.WriteString(fmt.Sprintf("- **Source**: `%s` @ `%s`\n", metadata.SyncMetadata.SourceRepo, metadata.SyncMetadata.SourceCommit))
		builder.WriteString(fmt.Sprintf("- **Target**: `%s`\n", metadata.SyncMetadata.TargetRepo))
		builder.WriteString(fmt.Sprintf("- **Sync Time**: %s\n", metadata.SyncMetadata.SyncTime.Format(time.RFC3339)))
		if metadata.SyncMetadata.SyncCommit != "" {
			builder.WriteString(fmt.Sprintf("- **Sync Commit**: `%s`\n", metadata.SyncMetadata.SyncCommit))
		}
	}

	// Add individual files if any
	if len(metadata.Files) > 0 {
		builder.WriteString("\n### Individual Files Synced\n")
		for _, file := range metadata.Files {
			if file.Source != file.Destination {
				builder.WriteString(fmt.Sprintf("- `%s` → `%s`\n", file.Source, file.Destination))
			} else {
				builder.WriteString(fmt.Sprintf("- `%s`\n", file.Destination))
			}
		}
	}

	// Add directory mappings if any
	if len(metadata.Directories) > 0 {
		builder.WriteString("\n### Directory Mappings\n")
		for _, dir := range metadata.Directories {
			if dir.Source != dir.Destination {
				builder.WriteString(fmt.Sprintf("- `%s` → `%s`\n", dir.Source, dir.Destination))
			} else {
				builder.WriteString(fmt.Sprintf("- `%s`\n", dir.Destination))
			}
			builder.WriteString(fmt.Sprintf("  - **Files Synced**: %d\n", dir.FilesSynced))
			if dir.FilesExcluded > 0 {
				builder.WriteString(fmt.Sprintf("  - **Files Excluded**: %d\n", dir.FilesExcluded))
			}
			if len(dir.Excluded) > 0 {
				builder.WriteString(fmt.Sprintf("  - **Exclusion Patterns**: %s\n", strings.Join(dir.Excluded, ", ")))
			}
			if dir.ProcessingTimeMs > 0 {
				builder.WriteString(fmt.Sprintf("  - **Processing Time**: %dms\n", dir.ProcessingTimeMs))
			}
		}
	}

	// Add performance metrics if available
	if metadata.Performance != nil {
		builder.WriteString("\n### Performance Metrics\n")
		if metadata.Performance.TotalSyncTimeMs > 0 {
			builder.WriteString(fmt.Sprintf("- **Total Sync Time**: %dms\n", metadata.Performance.TotalSyncTimeMs))
		}
		if metadata.Performance.TotalFilesProcessed > 0 {
			builder.WriteString(fmt.Sprintf("- **Files Processed**: %d\n", metadata.Performance.TotalFilesProcessed))
		}
		if metadata.Performance.TotalFilesChanged > 0 {
			builder.WriteString(fmt.Sprintf("- **Files Changed**: %d\n", metadata.Performance.TotalFilesChanged))
		}
		if metadata.Performance.TotalFilesSkipped > 0 {
			builder.WriteString(fmt.Sprintf("- **Files Skipped**: %d\n", metadata.Performance.TotalFilesSkipped))
		}
		if metadata.Performance.APICallsSaved > 0 {
			builder.WriteString(fmt.Sprintf("- **API Calls Saved**: %d\n", metadata.Performance.APICallsSaved))
		}
		if metadata.Performance.CacheHits > 0 || metadata.Performance.CacheMisses > 0 {
			total := metadata.Performance.CacheHits + metadata.Performance.CacheMisses
			if total > 0 {
				hitRate := float64(metadata.Performance.CacheHits) / float64(total) * 100
				builder.WriteString(fmt.Sprintf("- **Cache Hit Rate**: %.1f%% (%d hits, %d misses)\n", hitRate, metadata.Performance.CacheHits, metadata.Performance.CacheMisses))
			}
		}
	}

	// Add metadata block at the end
	builder.WriteString("\n\n")
	builder.WriteString(FormatEnhancedPRMetadata(metadata))

	return builder.String()
}

// ConvertToDirectorySyncInfo converts enhanced PR metadata to DirectorySyncInfo
func ConvertToDirectorySyncInfo(enhanced *EnhancedPRMetadata) *DirectorySyncInfo {
	if enhanced == nil {
		return nil
	}

	info := &DirectorySyncInfo{}

	// Convert directory mappings
	if len(enhanced.Directories) > 0 {
		for _, dir := range enhanced.Directories {
			mapping := DirectoryMappingInfo{
				Source:            dir.Source,
				Destination:       dir.Destination,
				ExcludePatterns:   dir.Excluded,
				PreserveStructure: true,  // Assumed for directory sync
				IncludeHidden:     false, // Default value, not tracked in PR metadata
				TransformApplied:  false, // Default value, not tracked in PR metadata
			}
			info.DirectoryMappings = append(info.DirectoryMappings, mapping)
		}
	}

	// Convert synced files info
	if len(enhanced.Files) > 0 || len(enhanced.Directories) > 0 {
		syncedFiles := &SyncedFilesInfo{}

		// Add individual files
		for _, file := range enhanced.Files {
			syncedFiles.IndividualSyncedFiles = append(syncedFiles.IndividualSyncedFiles, file.Destination)
		}
		syncedFiles.IndividualFileCount = len(enhanced.Files)

		// Add directory files (approximation)
		var dirFileCount int
		for _, dir := range enhanced.Directories {
			dirFileCount += dir.FilesSynced
		}
		syncedFiles.DirectoryFileCount = dirFileCount
		syncedFiles.TotalFiles = syncedFiles.IndividualFileCount + syncedFiles.DirectoryFileCount

		info.SyncedFiles = syncedFiles
	}

	// Convert performance metrics
	if enhanced.Performance != nil {
		perfMetrics := &DirectoryPerformanceMetrics{
			ExtractedFromPR: true,
		}

		// Create overall metrics
		if enhanced.Performance.TotalSyncTimeMs > 0 || enhanced.Performance.TotalFilesProcessed > 0 {
			overall := &OverallSyncMetrics{
				Duration:            time.Duration(enhanced.Performance.TotalSyncTimeMs) * time.Millisecond,
				ProcessingTimeMs:    enhanced.Performance.TotalSyncTimeMs,
				TotalFilesProcessed: enhanced.Performance.TotalFilesProcessed,
				TotalFilesChanged:   enhanced.Performance.TotalFilesChanged,
				TotalFilesSkipped:   enhanced.Performance.TotalFilesSkipped,
			}

			// Set start/end times if sync metadata is available
			if enhanced.SyncMetadata != nil {
				overall.EndTime = enhanced.SyncMetadata.SyncTime
				overall.StartTime = overall.EndTime.Add(-overall.Duration)
			}

			perfMetrics.OverallMetrics = overall
		}

		// Create API metrics
		if enhanced.Performance.APICallsSaved > 0 || enhanced.Performance.CacheHits > 0 || enhanced.Performance.CacheMisses > 0 {
			apiMetrics := &APISyncMetrics{
				APICallsSaved: enhanced.Performance.APICallsSaved,
				CacheHits:     enhanced.Performance.CacheHits,
				CacheMisses:   enhanced.Performance.CacheMisses,
			}

			// Calculate cache hit ratio
			total := apiMetrics.CacheHits + apiMetrics.CacheMisses
			if total > 0 {
				apiMetrics.CacheHitRatio = float64(apiMetrics.CacheHits) / float64(total)
			}

			perfMetrics.APIMetrics = apiMetrics
		}

		// Create directory-specific metrics
		if len(enhanced.Directories) > 0 {
			dirMetrics := make(map[string]*DirectoryProcessingMetrics)
			for _, dir := range enhanced.Directories {
				metrics := &DirectoryProcessingMetrics{
					FilesProcessed:     dir.FilesSynced,
					FilesExcluded:      dir.FilesExcluded,
					ProcessingDuration: time.Duration(dir.ProcessingTimeMs) * time.Millisecond,
				}
				dirMetrics[dir.Source] = metrics
			}
			perfMetrics.DirectoryMetrics = dirMetrics
		}

		info.PerformanceMetrics = perfMetrics
	}

	// Set last directory sync time
	if enhanced.SyncMetadata != nil {
		info.LastDirectorySync = &enhanced.SyncMetadata.SyncTime
	}

	return info
}

// CreateEnhancedMetadataFromDirectorySync creates enhanced PR metadata from DirectorySyncInfo
func CreateEnhancedMetadataFromDirectorySync(sourceRepo, targetRepo, sourceCommit string, syncInfo *DirectorySyncInfo) *EnhancedPRMetadata {
	enhanced := &EnhancedPRMetadata{
		SyncMetadata: &SyncMetadataInfo{
			SourceRepo:   sourceRepo,
			SourceCommit: sourceCommit,
			TargetRepo:   targetRepo,
			SyncTime:     time.Now().UTC(),
		},
	}

	if syncInfo == nil {
		return enhanced
	}

	// Convert directory mappings
	if len(syncInfo.DirectoryMappings) > 0 {
		for _, mapping := range syncInfo.DirectoryMappings {
			dir := DirectoryMapping{
				Source:      mapping.Source,
				Destination: mapping.Destination,
				Excluded:    mapping.ExcludePatterns,
			}

			// Try to get metrics from performance data
			if syncInfo.PerformanceMetrics != nil && syncInfo.PerformanceMetrics.DirectoryMetrics != nil {
				if metrics, exists := syncInfo.PerformanceMetrics.DirectoryMetrics[mapping.Source]; exists {
					dir.FilesSynced = metrics.FilesProcessed
					dir.FilesExcluded = metrics.FilesExcluded
					dir.ProcessingTimeMs = int64(metrics.ProcessingDuration / time.Millisecond)
				}
			}

			enhanced.Directories = append(enhanced.Directories, dir)
		}
	}

	// Convert individual files from synced files info
	if syncInfo.SyncedFiles != nil {
		for _, file := range syncInfo.SyncedFiles.IndividualSyncedFiles {
			enhanced.Files = append(enhanced.Files, FileMapping{
				Source:      file,
				Destination: file,
				From:        "file",
			})
		}
	}

	// Convert performance metrics
	if syncInfo.PerformanceMetrics != nil {
		perf := &PerformanceInfo{}

		if syncInfo.PerformanceMetrics.OverallMetrics != nil {
			overall := syncInfo.PerformanceMetrics.OverallMetrics
			perf.TotalSyncTimeMs = overall.ProcessingTimeMs
			perf.TotalFilesProcessed = overall.TotalFilesProcessed
			perf.TotalFilesChanged = overall.TotalFilesChanged
			perf.TotalFilesSkipped = overall.TotalFilesSkipped
		}

		if syncInfo.PerformanceMetrics.APIMetrics != nil {
			api := syncInfo.PerformanceMetrics.APIMetrics
			perf.APICallsSaved = api.APICallsSaved
			perf.CacheHits = api.CacheHits
			perf.CacheMisses = api.CacheMisses
		}

		enhanced.Performance = perf
	}

	return enhanced
}
