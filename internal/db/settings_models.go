package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// SettingsPreset stores a reusable repository settings preset
type SettingsPreset struct {
	BaseModel

	ExternalID  string `gorm:"uniqueIndex;type:text;not null" json:"external_id"`
	Name        string `gorm:"type:text;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`

	// Repository feature flags
	HasIssues      bool `gorm:"default:true" json:"has_issues"`
	HasWiki        bool `gorm:"default:false" json:"has_wiki"`
	HasProjects    bool `gorm:"default:false" json:"has_projects"`
	HasDiscussions bool `gorm:"default:false" json:"has_discussions"`

	// Merge settings
	AllowSquashMerge    bool `gorm:"default:true" json:"allow_squash_merge"`
	AllowMergeCommit    bool `gorm:"default:false" json:"allow_merge_commit"`
	AllowRebaseMerge    bool `gorm:"default:false" json:"allow_rebase_merge"`
	DeleteBranchOnMerge bool `gorm:"default:true" json:"delete_branch_on_merge"`
	AllowAutoMerge      bool `gorm:"default:true" json:"allow_auto_merge"`
	AllowUpdateBranch   bool `gorm:"default:true" json:"allow_update_branch"`

	// Squash merge commit format
	SquashMergeCommitTitle   string `gorm:"type:text" json:"squash_merge_commit_title"`
	SquashMergeCommitMessage string `gorm:"type:text" json:"squash_merge_commit_message"`

	// Child relationships
	Labels   []SettingsPresetLabel   `gorm:"foreignKey:SettingsPresetID" json:"labels,omitempty"`
	Rulesets []SettingsPresetRuleset `gorm:"foreignKey:SettingsPresetID" json:"rulesets,omitempty"`
}

// SettingsPresetLabel stores a label definition within a preset
type SettingsPresetLabel struct {
	BaseModel

	SettingsPresetID uint   `gorm:"index;not null" json:"settings_preset_id"`
	Name             string `gorm:"type:text;not null" json:"name"`
	Color            string `gorm:"type:text;not null" json:"color"`
	Description      string `gorm:"type:text" json:"description"`
}

// SettingsPresetRuleset stores a ruleset definition within a preset
type SettingsPresetRuleset struct {
	BaseModel

	SettingsPresetID uint            `gorm:"index;not null" json:"settings_preset_id"`
	Name             string          `gorm:"type:text;not null" json:"name"`
	Target           string          `gorm:"type:text;not null" json:"target"` // "branch" or "tag"
	Enforcement      string          `gorm:"type:text;default:active" json:"enforcement"`
	Include          JSONStringSlice `gorm:"type:text" json:"include"`
	Exclude          JSONStringSlice `gorm:"type:text" json:"exclude"`
	Rules            JSONStringSlice `gorm:"type:text" json:"rules"`
}

// AuditCheckResult represents a single audit check outcome
type AuditCheckResult struct {
	Setting  string `json:"setting"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Pass     bool   `json:"pass"`
}

// JSONAuditResults stores []AuditCheckResult as JSON TEXT
//
//nolint:recvcheck // mixed receivers required by driver.Valuer/sql.Scanner interface
type JSONAuditResults []AuditCheckResult

// Value implements driver.Valuer
func (j JSONAuditResults) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil //nolint:nilnil // database/sql pattern for NULL values
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner
func (j *JSONAuditResults) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("%w for JSONAuditResults", ErrInvalidType)
	}

	return json.Unmarshal(bytes, j)
}

// RepoSettingsAudit stores audit results for a repo against a preset
type RepoSettingsAudit struct {
	BaseModel

	RepoID           uint             `gorm:"index;not null" json:"repo_id"`
	SettingsPresetID uint             `gorm:"index;not null" json:"settings_preset_id"`
	Score            int              `gorm:"default:0" json:"score"`   // Percentage 0-100
	Total            int              `gorm:"default:0" json:"total"`   // Total checks
	Passed           int              `gorm:"default:0" json:"passed"`  // Passed checks
	Results          JSONAuditResults `gorm:"type:text" json:"results"` // Detailed results
	Repo             Repo             `gorm:"foreignKey:RepoID" json:"repo,omitempty"`
	Preset           SettingsPreset   `gorm:"foreignKey:SettingsPresetID" json:"preset,omitempty"`
}
