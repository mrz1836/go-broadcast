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
	ErrPRNoDescription     = errors.New("PR has no description")
	ErrPRNoMetadataBlock   = errors.New("no metadata block found in PR description")
	ErrPRMetadataNotClosed = errors.New("metadata block not properly closed")
)

// PRMetadata represents metadata stored in PR descriptions
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

// ExtractPRMetadata extracts sync metadata from a PR description
func ExtractPRMetadata(pr gh.PR) (*PRMetadata, error) {
	if pr.Body == "" {
		return nil, ErrPRNoDescription
	}

	// Look for metadata block in PR description
	// Format: <!-- go-broadcast:metadata
	// yaml content
	// -->
	startMarker := "<!-- go-broadcast:metadata"
	endMarker := "-->"

	startIdx := strings.Index(pr.Body, startMarker)
	if startIdx == -1 {
		return nil, ErrPRNoMetadataBlock
	}

	// Find the end of the metadata block
	metadataStart := startIdx + len(startMarker)

	endIdx := strings.Index(pr.Body[metadataStart:], endMarker)
	if endIdx == -1 {
		return nil, ErrPRMetadataNotClosed
	}

	// Extract the YAML content
	yamlContent := strings.TrimSpace(pr.Body[metadataStart : metadataStart+endIdx])

	// Parse the YAML
	var metadata PRMetadata
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata YAML: %w", err)
	}

	return &metadata, nil
}

// FormatPRMetadata formats metadata for inclusion in a PR description
func FormatPRMetadata(metadata *PRMetadata) string {
	yamlBytes, err := yaml.Marshal(metadata)
	if err != nil {
		// Fallback to a simple format if YAML marshaling fails
		return fmt.Sprintf("<!-- go-broadcast:metadata\nsource_commit: %s\n-->", metadata.SourceCommit)
	}

	return fmt.Sprintf("<!-- go-broadcast:metadata\n%s-->", string(yamlBytes))
}

// GeneratePRDescription generates a complete PR description with metadata
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
