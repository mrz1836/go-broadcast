package config

// Config represents the complete sync configuration
type Config struct {
	Version  int            `yaml:"version"`
	Source   SourceConfig   `yaml:"source"`
	Global   GlobalConfig   `yaml:"global,omitempty"`
	Defaults DefaultConfig  `yaml:"defaults,omitempty"`
	Targets  []TargetConfig `yaml:"targets"`
}

// SourceConfig defines the source repository settings
type SourceConfig struct {
	Repo   string `yaml:"repo"`   // Format: org/repo
	Branch string `yaml:"branch"` // Default: master
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
	Repo            string        `yaml:"repo"`                        // Format: org/repo
	Files           []FileMapping `yaml:"files"`                       // Files to sync
	Transform       Transform     `yaml:"transform,omitempty"`         // Optional transformations
	PRLabels        []string      `yaml:"pr_labels,omitempty"`         // Override default PR labels
	PRAssignees     []string      `yaml:"pr_assignees,omitempty"`      // Override default PR assignees
	PRReviewers     []string      `yaml:"pr_reviewers,omitempty"`      // Override default PR reviewers
	PRTeamReviewers []string      `yaml:"pr_team_reviewers,omitempty"` // Override default PR team reviewers
}

// FileMapping defines source to destination file mapping
type FileMapping struct {
	Src  string `yaml:"src"`  // Source file path
	Dest string `yaml:"dest"` // Destination file path
}

// Transform defines transformation settings
type Transform struct {
	RepoName  bool              `yaml:"repo_name,omitempty"` // Replace repository names
	Variables map[string]string `yaml:"variables,omitempty"` // Template variables
}
