package config

// Config represents the complete sync configuration
type Config struct {
	Version int `yaml:"version"`

	// Multi-source configuration mappings
	Mappings []SourceMapping `yaml:"mappings,omitempty"`

	// Shared configuration
	Global   GlobalConfig  `yaml:"global,omitempty"`
	Defaults DefaultConfig `yaml:"defaults,omitempty"`

	// Conflict resolution settings
	ConflictResolution *ConflictResolution `yaml:"conflict_resolution,omitempty"`
}

// SourceMapping represents a mapping from one source to multiple targets
type SourceMapping struct {
	Source  SourceConfig   `yaml:"source"`
	Targets []TargetConfig `yaml:"targets"`

	// Optional mapping-level defaults that override global defaults
	Defaults *DefaultConfig `yaml:"defaults,omitempty"`
}

// SourceConfig defines the source repository settings
type SourceConfig struct {
	Repo   string `yaml:"repo"`   // Format: org/repo
	Branch string `yaml:"branch"` // Default: master

	// Optional source identifier for conflict resolution
	ID string `yaml:"id,omitempty"`
}

// GlobalConfig contains global settings applied across all targets
// These settings are merged with target-specific settings rather than overridden
type GlobalConfig struct {
	PRLabels        []string `yaml:"pr_labels,omitempty"`         // Global PR labels to apply to all PRs
	PRAssignees     []string `yaml:"pr_assignees,omitempty"`      // Global GitHub usernames to assign to all PRs
	PRReviewers     []string `yaml:"pr_reviewers,omitempty"`      // Global GitHub usernames to request reviews from
	PRTeamReviewers []string `yaml:"pr_team_reviewers,omitempty"` // Global GitHub team slugs to request reviews from
}

// DefaultConfig contains default settings applied to all targets
type DefaultConfig struct {
	BranchPrefix    string   `yaml:"branch_prefix,omitempty"`     // Default: chore/sync-files
	PRLabels        []string `yaml:"pr_labels,omitempty"`         // Default: ["automated-sync"]
	PRAssignees     []string `yaml:"pr_assignees,omitempty"`      // GitHub usernames to assign to PRs
	PRReviewers     []string `yaml:"pr_reviewers,omitempty"`      // GitHub usernames to request reviews from
	PRTeamReviewers []string `yaml:"pr_team_reviewers,omitempty"` // GitHub team slugs to request reviews from
}

// TargetConfig defines a target repository and its file mappings
type TargetConfig struct {
	Repo            string             `yaml:"repo"`                        // Format: org/repo
	Files           []FileMapping      `yaml:"files"`                       // Files to sync
	Directories     []DirectoryMapping `yaml:"directories,omitempty"`       // Directories to sync
	Transform       Transform          `yaml:"transform,omitempty"`         // Optional transformations
	PRLabels        []string           `yaml:"pr_labels,omitempty"`         // Override default PR labels
	PRAssignees     []string           `yaml:"pr_assignees,omitempty"`      // Override default PR assignees
	PRReviewers     []string           `yaml:"pr_reviewers,omitempty"`      // Override default PR reviewers
	PRTeamReviewers []string           `yaml:"pr_team_reviewers,omitempty"` // Override default PR team reviewers
}

// FileMapping defines source to destination file mapping
type FileMapping struct {
	Src  string `yaml:"src"`  // Source file path
	Dest string `yaml:"dest"` // Destination file path
}

// DirectoryMapping defines source to destination directory mapping
type DirectoryMapping struct {
	Src               string    `yaml:"src"`                          // Source directory path
	Dest              string    `yaml:"dest"`                         // Destination directory path
	Exclude           []string  `yaml:"exclude,omitempty"`            // Glob patterns to exclude
	IncludeOnly       []string  `yaml:"include_only,omitempty"`       // Glob patterns to include (excludes everything else)
	Transform         Transform `yaml:"transform,omitempty"`          // Apply to all files
	PreserveStructure *bool     `yaml:"preserve_structure,omitempty"` // Keep nested structure (default: true)
	IncludeHidden     *bool     `yaml:"include_hidden,omitempty"`     // Include hidden files (default: true)
}

// Transform defines transformation settings
type Transform struct {
	RepoName  bool              `yaml:"repo_name,omitempty"` // Replace repository names
	Variables map[string]string `yaml:"variables,omitempty"` // Template variables
}

// ConflictResolution defines how to handle conflicts when multiple sources target the same file
type ConflictResolution struct {
	Strategy string   `yaml:"strategy"`           // "last-wins", "priority", "error"
	Priority []string `yaml:"priority,omitempty"` // Source IDs in priority order
}

// GetAllTargets returns all unique target repositories across all mappings
func (c *Config) GetAllTargets() map[string]bool {
	targets := make(map[string]bool)
	for _, mapping := range c.Mappings {
		for _, target := range mapping.Targets {
			targets[target.Repo] = true
		}
	}
	return targets
}

// GetTargetMappings returns all source-to-target mappings for a specific target repository
func (c *Config) GetTargetMappings(targetRepo string) []SourceTargetPair {
	var pairs []SourceTargetPair
	for _, mapping := range c.Mappings {
		for _, target := range mapping.Targets {
			if target.Repo == targetRepo {
				pairs = append(pairs, SourceTargetPair{
					Source:   mapping.Source,
					Target:   target,
					Defaults: mapping.Defaults,
				})
			}
		}
	}
	return pairs
}

// SourceTargetPair represents a single source-to-target relationship
type SourceTargetPair struct {
	Source   SourceConfig
	Target   TargetConfig
	Defaults *DefaultConfig
}
