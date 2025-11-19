// Package state provides PR metadata handling for go-broadcast
package state

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// PR metadata extraction errors
var (
	ErrPRNoDescription       = errors.New("PR has no description")
	ErrPRNoMetadataBlock     = errors.New("no metadata block found in PR description")
	ErrPRMetadataNotClosed   = errors.New("metadata block not properly closed")
	ErrPRMissingSyncMetadata = errors.New("enhanced metadata missing sync_metadata section")
)

// EnhancedPRMetadata represents the metadata format with directory sync support
type EnhancedPRMetadata struct {
	// SyncMetadata contains core sync information
	SyncMetadata *SyncMetadataInfo `yaml:"sync_metadata"`

	// Files contains individual file mappings
	Files []FileMapping `yaml:"files,omitempty"`

	// Directories contains directory mappings
	Directories []DirectoryMapping `yaml:"directories,omitempty"`

	// Performance contains performance metrics
	Performance *PerformanceInfo `yaml:"performance,omitempty"`
}

// SyncMetadataInfo contains core sync metadata
type SyncMetadataInfo struct {
	SourceRepo   string    `yaml:"source_repo"`
	SourceCommit string    `yaml:"source_commit"`
	TargetRepo   string    `yaml:"target_repo"`
	SyncCommit   string    `yaml:"sync_commit"`
	SyncTime     time.Time `yaml:"sync_time"`
}

// FileMapping represents a file sync mapping
type FileMapping struct {
	Source      string `yaml:"src"`
	Destination string `yaml:"dest"`
}

// DirectoryMapping represents a directory sync mapping with metrics
type DirectoryMapping struct {
	Source         string   `yaml:"src"`
	Destination    string   `yaml:"dest"`
	Excluded       []string `yaml:"excluded,omitempty"`
	FilesSynced    int      `yaml:"files_synced,omitempty"`
	FilesExcluded  int      `yaml:"files_excluded,omitempty"`
	ProcessingTime int64    `yaml:"processing_time_ms,omitempty"`
}

// PerformanceInfo contains performance metrics for the sync operation
type PerformanceInfo struct {
	TotalFiles        int               `yaml:"total_files,omitempty"`
	APICallsSaved     int               `yaml:"api_calls_saved,omitempty"`
	TransformsApplied map[string]bool   `yaml:"transforms_applied,omitempty"`
	TimingInfo        map[string]string `yaml:"timing,omitempty"`
}

// ExtractEnhancedPRMetadata extracts enhanced metadata from a PR description
func ExtractEnhancedPRMetadata(pr gh.PR) (*EnhancedPRMetadata, error) {
	if pr.Body == "" {
		return nil, ErrPRNoDescription
	}

	// Extract YAML content from the enhanced metadata block
	yamlContent, err := extractMetadataYAML(pr.Body, "<!-- go-broadcast-metadata")
	if err != nil {
		// Try alternative marker for compatibility
		yamlContent, err = extractMetadataYAML(pr.Body, "<!-- go-broadcast metadata")
		if err != nil {
			return nil, err
		}
	}

	// Parse as enhanced format
	var enhanced EnhancedPRMetadata
	if err := yaml.Unmarshal([]byte(yamlContent), &enhanced); err != nil {
		return nil, fmt.Errorf("failed to parse enhanced metadata YAML: %w", err)
	}

	// Validate that we have the required sync_metadata
	if enhanced.SyncMetadata == nil {
		return nil, ErrPRMissingSyncMetadata
	}

	return &enhanced, nil
}

// extractMetadataYAML extracts the YAML content from a metadata block
func extractMetadataYAML(body, marker string) (string, error) {
	// Find the start of the metadata block
	startIdx := strings.Index(body, marker)
	if startIdx == -1 {
		return "", ErrPRNoMetadataBlock
	}

	// Find the end of the metadata block
	endIdx := strings.Index(body[startIdx:], "-->")
	if endIdx == -1 {
		return "", ErrPRMetadataNotClosed
	}

	// Extract just the YAML content between the markers
	content := body[startIdx : startIdx+endIdx]
	// Remove the marker line to get just the YAML
	lines := strings.Split(content, "\n")
	if len(lines) > 1 {
		return strings.Join(lines[1:], "\n"), nil
	}

	return "", ErrPRNoMetadataBlock
}

// FormatEnhancedPRMetadata formats enhanced metadata for inclusion in a PR description
func FormatEnhancedPRMetadata(metadata *EnhancedPRMetadata) string {
	yamlBytes, err := yaml.Marshal(metadata)
	if err != nil {
		// Fallback to a simple format if YAML marshaling fails
		return "<!-- go-broadcast-metadata\nerror: failed to marshal metadata\n-->"
	}

	return fmt.Sprintf("<!-- go-broadcast-metadata\n%s-->", string(yamlBytes))
}

// GenerateEnhancedPRDescription generates a complete PR description with enhanced metadata
func GenerateEnhancedPRDescription(metadata *EnhancedPRMetadata, summary string) string {
	var builder strings.Builder

	// Add summary
	builder.WriteString(summary)
	builder.WriteString("\n\n")

	// Add what changed section
	builder.WriteString("## What Changed\n")
	builder.WriteString(fmt.Sprintf("* Sync from %s (commit: %s)\n",
		metadata.SyncMetadata.SourceRepo,
		metadata.SyncMetadata.SourceCommit))

	// Count total files
	totalFiles := len(metadata.Files)
	if metadata.Performance != nil && metadata.Performance.TotalFiles > 0 {
		totalFiles = metadata.Performance.TotalFiles
	}

	if totalFiles > 0 {
		builder.WriteString(fmt.Sprintf("* Total files synchronized: %d\n", totalFiles))
	}

	// List directories if any
	if len(metadata.Directories) > 0 {
		builder.WriteString(fmt.Sprintf("* Directories synchronized: %d\n", len(metadata.Directories)))

		// Calculate total excluded files
		totalExcluded := 0
		for _, dir := range metadata.Directories {
			totalExcluded += dir.FilesExcluded
		}
		if totalExcluded > 0 {
			builder.WriteString(fmt.Sprintf("* Files excluded: %d\n", totalExcluded))
		}
	}

	// Show transforms if applied
	if metadata.Performance != nil && len(metadata.Performance.TransformsApplied) > 0 {
		var transforms []string
		for transform := range metadata.Performance.TransformsApplied {
			transforms = append(transforms, transform)
		}
		builder.WriteString(fmt.Sprintf("* Transforms applied: %s\n", strings.Join(transforms, ", ")))
	}

	// Show API optimization if significant
	if metadata.Performance != nil && metadata.Performance.APICallsSaved > 0 {
		builder.WriteString(fmt.Sprintf("* API calls optimized: %d calls saved\n", metadata.Performance.APICallsSaved))
	}

	builder.WriteString("\n")

	// Add file/directory details based on count
	if len(metadata.Files) > 0 && len(metadata.Files) <= 10 {
		builder.WriteString("## Files Synchronized\n")
		for _, file := range metadata.Files {
			builder.WriteString(fmt.Sprintf("* `%s` → `%s`\n", file.Source, file.Destination))
		}
		builder.WriteString("\n")
	}

	if len(metadata.Directories) > 0 && len(metadata.Directories) <= 5 {
		builder.WriteString("## Directories Synchronized\n")
		for _, dir := range metadata.Directories {
			builder.WriteString(fmt.Sprintf("* `%s` → `%s`", dir.Source, dir.Destination))
			if dir.FilesSynced > 0 {
				builder.WriteString(fmt.Sprintf(" (%d files", dir.FilesSynced))
				if dir.FilesExcluded > 0 {
					builder.WriteString(fmt.Sprintf(", %d excluded", dir.FilesExcluded))
				}
				builder.WriteString(")")
			}
			builder.WriteString("\n")

			// Show exclusion patterns if any
			if len(dir.Excluded) > 0 && len(dir.Excluded) <= 5 {
				builder.WriteString(fmt.Sprintf("  - Excluded: %s\n", strings.Join(dir.Excluded, ", ")))
			}
		}
		builder.WriteString("\n")
	} else if len(metadata.Directories) > 5 {
		builder.WriteString(fmt.Sprintf("## Directories Synchronized\n%d directories synchronized (too many to list)\n\n", len(metadata.Directories)))
	}

	// Add metadata block
	builder.WriteString("\n")
	builder.WriteString(FormatEnhancedPRMetadata(metadata))

	return builder.String()
}
