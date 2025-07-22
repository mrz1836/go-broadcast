package config

// Config represents the complete sync configuration
type Config struct {
	Version  int            `yaml:"version"`
	Source   SourceConfig   `yaml:"source"`
	Defaults DefaultConfig  `yaml:"defaults,omitempty"`
	Targets  []TargetConfig `yaml:"targets"`
}

// SourceConfig defines the source repository settings
type SourceConfig struct {
	Repo   string `yaml:"repo"`   // Format: org/repo
	Branch string `yaml:"branch"` // Default: master
}

// DefaultConfig contains default settings applied to all targets
type DefaultConfig struct {
	BranchPrefix string   `yaml:"branch_prefix,omitempty"` // Default: sync/template
	PRLabels     []string `yaml:"pr_labels,omitempty"`     // Default: ["automated-sync"]
}

// TargetConfig defines a target repository and its file mappings
type TargetConfig struct {
	Repo      string        `yaml:"repo"`                // Format: org/repo
	Files     []FileMapping `yaml:"files"`               // Files to sync
	Transform Transform     `yaml:"transform,omitempty"` // Optional transformations
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
