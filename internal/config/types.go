package config

// Config represents the complete sync configuration
type Config struct {
	Version         int              `yaml:"version"`                    // Config version (1)
	Name            string           `yaml:"name,omitempty"`             // Optional config name
	ID              string           `yaml:"id,omitempty"`               // Optional config ID
	FileLists       []FileList       `yaml:"file_lists,omitempty"`       // Reusable file lists
	DirectoryLists  []DirectoryList  `yaml:"directory_lists,omitempty"`  // Reusable directory lists
	Groups          []Group          `yaml:"groups"`                     // List of sync groups
	SettingsPresets []SettingsPreset `yaml:"settings_presets,omitempty"` // Repository settings presets
}

// GetPreset returns a settings preset by ID, or nil if not found
func (c *Config) GetPreset(id string) *SettingsPreset {
	for i := range c.SettingsPresets {
		if c.SettingsPresets[i].ID == id {
			return &c.SettingsPresets[i]
		}
	}
	return nil
}

// DefaultPreset returns a hardcoded default preset (mvp) for use when no config is loaded
func DefaultPreset() SettingsPreset {
	return SettingsPreset{
		ID:                       "mvp",
		Name:                     "MVP",
		Description:              "Default preset for new repositories",
		HasIssues:                true,
		HasWiki:                  false,
		HasProjects:              false,
		HasDiscussions:           false,
		AllowSquashMerge:         true,
		AllowMergeCommit:         false,
		AllowRebaseMerge:         false,
		DeleteBranchOnMerge:      true,
		AllowAutoMerge:           true,
		AllowUpdateBranch:        true,
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "COMMIT_MESSAGES",
		Rulesets: []RulesetConfig{
			{
				Name:        "branch-protection",
				Target:      "branch",
				Enforcement: "active",
				Include:     []string{"~DEFAULT_BRANCH"},
				Rules:       []string{"deletion", "pull_request"},
			},
			{
				Name:        "tag-protection",
				Target:      "tag",
				Enforcement: "active",
				Include:     []string{"~ALL"},
				Rules:       []string{"deletion", "update"},
			},
		},
		Labels: []LabelSpec{
			{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
			{Name: "enhancement", Color: "a2eeef", Description: "New feature or request"},
			{Name: "documentation", Color: "0075ca", Description: "Documentation improvements"},
			{Name: "good first issue", Color: "7057ff", Description: "Good for newcomers"},
			{Name: "help wanted", Color: "008672", Description: "Extra attention needed"},
			{Name: "priority: high", Color: "b60205", Description: "High priority"},
			{Name: "priority: low", Color: "c5def5", Description: "Low priority"},
			{Name: "wontfix", Color: "ffffff", Description: "Won't be addressed"},
		},
	}
}

// SourceConfig defines the source repository settings
type SourceConfig struct {
	Repo          string `yaml:"repo"`                      // Format: org/repo
	Branch        string `yaml:"branch"`                    // Default: master
	BlobSizeLimit string `yaml:"blob_size_limit,omitempty"` // Max blob size for partial clone (e.g., "10m"), "0" to disable
	SecurityEmail string `yaml:"security_email,omitempty"`  // Security contact email address (for transformation)
	SupportEmail  string `yaml:"support_email,omitempty"`   // Support/contact email address (for transformation)
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
	Repo              string             `yaml:"repo"`                          // Format: org/repo
	Branch            string             `yaml:"branch,omitempty"`              // Target branch for PR base (defaults to repo's default branch)
	BlobSizeLimit     string             `yaml:"blob_size_limit,omitempty"`     // Override source blob size limit for partial clone
	Files             []FileMapping      `yaml:"files,omitempty"`               // Files to sync
	Directories       []DirectoryMapping `yaml:"directories,omitempty"`         // Directories to sync
	FileListRefs      []string           `yaml:"file_list_refs,omitempty"`      // References to file lists by ID
	DirectoryListRefs []string           `yaml:"directory_list_refs,omitempty"` // References to directory lists by ID
	Transform         Transform          `yaml:"transform,omitempty"`           // Optional transformations
	SecurityEmail     string             `yaml:"security_email,omitempty"`      // Override security contact email (defaults to source security_email)
	SupportEmail      string             `yaml:"support_email,omitempty"`       // Override support contact email (defaults to source support_email)
	PRLabels          []string           `yaml:"pr_labels,omitempty"`           // Override default PR labels
	PRAssignees       []string           `yaml:"pr_assignees,omitempty"`        // Override default PR assignees
	PRReviewers       []string           `yaml:"pr_reviewers,omitempty"`        // Override default PR reviewers
	PRTeamReviewers   []string           `yaml:"pr_team_reviewers,omitempty"`   // Override default PR team reviewers
}

// FileMapping defines source to destination file mapping
type FileMapping struct {
	Src    string `yaml:"src"`              // Source file path
	Dest   string `yaml:"dest"`             // Destination file path
	Delete bool   `yaml:"delete,omitempty"` // Delete the destination file instead of syncing
}

// DirectoryMapping defines source to destination directory mapping
type DirectoryMapping struct {
	Src               string        `yaml:"src"`                          // Source directory path
	Dest              string        `yaml:"dest"`                         // Destination directory path
	Exclude           []string      `yaml:"exclude,omitempty"`            // Glob patterns to exclude
	IncludeOnly       []string      `yaml:"include_only,omitempty"`       // Glob patterns to include (excludes everything else)
	Transform         Transform     `yaml:"transform,omitempty"`          // Apply to all files
	PreserveStructure *bool         `yaml:"preserve_structure,omitempty"` // Keep nested structure (default: true)
	IncludeHidden     *bool         `yaml:"include_hidden,omitempty"`     // Include hidden files (default: true)
	Module            *ModuleConfig `yaml:"module,omitempty"`             // Module-aware sync settings
	Delete            bool          `yaml:"delete,omitempty"`             // Delete the destination directory instead of syncing
}

// Transform defines transformation settings
type Transform struct {
	RepoName  bool              `yaml:"repo_name,omitempty"` // Replace repository names
	Variables map[string]string `yaml:"variables,omitempty"` // Template variables
}

// Group represents a sync group with its own source and targets
type Group struct {
	Name        string         `yaml:"name"`                  // Friendly name
	ID          string         `yaml:"id"`                    // Unique identifier
	Description string         `yaml:"description,omitempty"` // Optional description
	Priority    int            `yaml:"priority,omitempty"`    // Execution order (default: 0)
	DependsOn   []string       `yaml:"depends_on,omitempty"`  // Group IDs this group depends on
	Enabled     *bool          `yaml:"enabled,omitempty"`     // Toggle on/off (default: true)
	Source      SourceConfig   `yaml:"source"`                // Source repository
	Global      GlobalConfig   `yaml:"global,omitempty"`      // Group-level globals
	Defaults    DefaultConfig  `yaml:"defaults,omitempty"`    // Group-level defaults
	Targets     []TargetConfig `yaml:"targets"`               // Target repositories
}

// ModuleConfig defines module-aware sync settings
type ModuleConfig struct {
	Type       string `yaml:"type,omitempty"`        // Module type: "go" (future: "npm", "python")
	Version    string `yaml:"version"`               // Version constraint (exact, latest, or semver)
	CheckTags  *bool  `yaml:"check_tags,omitempty"`  // Use git tags for versions (default: true)
	UpdateRefs bool   `yaml:"update_refs,omitempty"` // Update go.mod references
}

// FileList represents a reusable list of file mappings
type FileList struct {
	ID          string        `yaml:"id"`                    // Unique identifier for this list
	Name        string        `yaml:"name"`                  // Friendly name for the list
	Description string        `yaml:"description,omitempty"` // Optional description of the list contents
	Files       []FileMapping `yaml:"files"`                 // File mappings in this list
}

// DirectoryList represents a reusable list of directory mappings
type DirectoryList struct {
	ID          string             `yaml:"id"`                    // Unique identifier for this list
	Name        string             `yaml:"name"`                  // Friendly name for the list
	Description string             `yaml:"description,omitempty"` // Optional description of the list contents
	Directories []DirectoryMapping `yaml:"directories"`           // Directory mappings in this list
}

// SettingsPreset defines a reusable set of repository settings
type SettingsPreset struct {
	ID          string `yaml:"id"`                    // Unique identifier (e.g., "mvp", "go-lib")
	Name        string `yaml:"name"`                  // Friendly name
	Description string `yaml:"description,omitempty"` // Optional description

	// Repository feature flags
	HasIssues      bool `yaml:"has_issues"`
	HasWiki        bool `yaml:"has_wiki"`
	HasProjects    bool `yaml:"has_projects"`
	HasDiscussions bool `yaml:"has_discussions"`

	// Merge settings
	AllowSquashMerge    bool `yaml:"allow_squash_merge"`
	AllowMergeCommit    bool `yaml:"allow_merge_commit"`
	AllowRebaseMerge    bool `yaml:"allow_rebase_merge"`
	DeleteBranchOnMerge bool `yaml:"delete_branch_on_merge"`
	AllowAutoMerge      bool `yaml:"allow_auto_merge"`
	AllowUpdateBranch   bool `yaml:"allow_update_branch"`

	// Squash merge commit format
	SquashMergeCommitTitle   string `yaml:"squash_merge_commit_title,omitempty"`   // PR_TITLE or COMMIT_OR_PR_TITLE
	SquashMergeCommitMessage string `yaml:"squash_merge_commit_message,omitempty"` // COMMIT_MESSAGES, PR_BODY, or BLANK

	// Rulesets
	Rulesets []RulesetConfig `yaml:"rulesets,omitempty"`

	// Labels
	Labels []LabelSpec `yaml:"labels,omitempty"`
}

// RulesetConfig defines a repository ruleset
type RulesetConfig struct {
	Name        string   `yaml:"name"`                  // Ruleset name (e.g., "branch-protection")
	Target      string   `yaml:"target"`                // "branch" or "tag"
	Enforcement string   `yaml:"enforcement,omitempty"` // "active", "disabled", "evaluate" (default: "active")
	Include     []string `yaml:"include"`               // Ref patterns (e.g., "~DEFAULT_BRANCH", "~ALL")
	Exclude     []string `yaml:"exclude,omitempty"`     // Ref patterns to exclude
	Rules       []string `yaml:"rules"`                 // Rule types: "deletion", "update", "pull_request"
}

// LabelSpec defines a repository label
type LabelSpec struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description,omitempty"`
}

// boolPtr is a helper function to create a pointer to a boolean value.
// This is used for optional boolean fields with default values.
func boolPtr(b bool) *bool {
	return &b
}
