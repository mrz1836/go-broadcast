package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// BaseModel contains common columns for all tables following GORM conventions
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Metadata  Metadata       `gorm:"type:text" json:"metadata,omitempty"`
}

// Metadata is a JSON key/value map stored as TEXT, provides extensibility on every table
//
//nolint:recvcheck // mixed receivers required by driver.Valuer/sql.Scanner interface
type Metadata map[string]interface{}

// Value implements driver.Valuer for database storage
func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil //nolint:nilnil // database/sql pattern for NULL values
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner for database retrieval
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("%w for Metadata", ErrInvalidType)
	}

	return json.Unmarshal(bytes, m)
}

// JSONStringSlice stores []string as JSON TEXT (for pr_labels, exclude, include_only, etc.)
//
//nolint:recvcheck // mixed receivers required by driver.Valuer/sql.Scanner interface
type JSONStringSlice []string

// Value implements driver.Valuer
func (j JSONStringSlice) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil //nolint:nilnil // database/sql pattern for NULL values
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner
func (j *JSONStringSlice) Scan(value interface{}) error {
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
		return fmt.Errorf("%w for JSONStringSlice", ErrInvalidType)
	}

	return json.Unmarshal(bytes, j)
}

// JSONStringMap stores map[string]string as JSON TEXT (for Transform.Variables)
//
//nolint:recvcheck // mixed receivers required by driver.Valuer/sql.Scanner interface
type JSONStringMap map[string]string

// Value implements driver.Valuer
func (j JSONStringMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil //nolint:nilnil // database/sql pattern for NULL values
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner
func (j *JSONStringMap) Scan(value interface{}) error {
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
		return fmt.Errorf("%w for JSONStringMap", ErrInvalidType)
	}

	return json.Unmarshal(bytes, j)
}

// JSONModuleConfig stores ModuleConfig as JSON TEXT
type JSONModuleConfig struct {
	Type       string `json:"type,omitempty"`        // "go", "npm", "python", etc.
	Version    string `json:"version,omitempty"`     // Semantic version string
	CheckTags  bool   `json:"check_tags,omitempty"`  // Check for newer tags
	UpdateRefs bool   `json:"update_refs,omitempty"` // Update version references in go.mod
}

// Value implements driver.Valuer
func (j *JSONModuleConfig) Value() (driver.Value, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*j)
}

// Scan implements sql.Scanner
func (j *JSONModuleConfig) Scan(value interface{}) error {
	if value == nil {
		*j = JSONModuleConfig{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("%w for JSONModuleConfig", ErrInvalidType)
	}

	return json.Unmarshal(bytes, j)
}

// =====================
// Main Tables (15 models)
// =====================

// Config represents the root configuration (maps to config.Config)
type Config struct {
	BaseModel

	ExternalID     string          `gorm:"uniqueIndex;type:text" json:"external_id"`
	Name           string          `gorm:"type:text" json:"name"`
	Version        int             `gorm:"not null;default:1" json:"version"`
	Groups         []Group         `gorm:"foreignKey:ConfigID" json:"groups,omitempty"`
	FileLists      []FileList      `gorm:"foreignKey:ConfigID" json:"file_lists,omitempty"`
	DirectoryLists []DirectoryList `gorm:"foreignKey:ConfigID" json:"directory_lists,omitempty"`
}

// Group represents a sync group (maps to config.Group)
type Group struct {
	BaseModel

	ConfigID    uint   `gorm:"index" json:"config_id"`
	ExternalID  string `gorm:"uniqueIndex;type:text;not null" json:"external_id"`
	Name        string `gorm:"type:text;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Priority    int    `gorm:"default:0" json:"priority"`
	Enabled     *bool  `gorm:"default:true" json:"enabled"`
	Position    int    `gorm:"default:0" json:"position"`

	// Relationships
	Source       Source            `gorm:"foreignKey:GroupID" json:"source,omitempty"`
	GroupGlobal  GroupGlobal       `gorm:"foreignKey:GroupID" json:"global,omitempty"`
	GroupDefault GroupDefault      `gorm:"foreignKey:GroupID" json:"defaults,omitempty"`
	Targets      []Target          `gorm:"foreignKey:GroupID" json:"targets,omitempty"`
	Dependencies []GroupDependency `gorm:"foreignKey:GroupID" json:"dependencies,omitempty"`
}

// GroupDependency stores group dependencies (DependsOn field)
type GroupDependency struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	GroupID     uint      `gorm:"index;not null" json:"group_id"`
	DependsOnID string    `gorm:"type:text;not null" json:"depends_on_id"` // External ID of dependency
	Position    int       `gorm:"default:0" json:"position"`
	Metadata    Metadata  `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Source represents source repository config (maps to config.SourceConfig)
type Source struct {
	BaseModel

	GroupID       uint   `gorm:"uniqueIndex;not null" json:"group_id"` // 1:1 relationship
	Repo          string `gorm:"type:text;not null;index" json:"repo"`
	Branch        string `gorm:"type:text;not null" json:"branch"`
	BlobSizeLimit string `gorm:"type:text" json:"blob_size_limit"`
	SecurityEmail string `gorm:"type:text" json:"security_email"`
	SupportEmail  string `gorm:"type:text" json:"support_email"`
}

// GroupGlobal represents group-level global config (maps to config.GlobalConfig)
type GroupGlobal struct {
	BaseModel

	GroupID         uint            `gorm:"uniqueIndex;not null" json:"group_id"` // 1:1 relationship
	PRLabels        JSONStringSlice `gorm:"type:text" json:"pr_labels"`
	PRAssignees     JSONStringSlice `gorm:"type:text" json:"pr_assignees"`
	PRReviewers     JSONStringSlice `gorm:"type:text" json:"pr_reviewers"`
	PRTeamReviewers JSONStringSlice `gorm:"type:text" json:"pr_team_reviewers"`
}

// GroupDefault represents group-level default config (maps to config.DefaultConfig)
type GroupDefault struct {
	BaseModel

	GroupID         uint            `gorm:"uniqueIndex;not null" json:"group_id"` // 1:1 relationship
	BranchPrefix    string          `gorm:"type:text" json:"branch_prefix"`
	PRLabels        JSONStringSlice `gorm:"type:text" json:"pr_labels"`
	PRAssignees     JSONStringSlice `gorm:"type:text" json:"pr_assignees"`
	PRReviewers     JSONStringSlice `gorm:"type:text" json:"pr_reviewers"`
	PRTeamReviewers JSONStringSlice `gorm:"type:text" json:"pr_team_reviewers"`
}

// Target represents a target repository (maps to config.TargetConfig)
type Target struct {
	BaseModel

	GroupID         uint            `gorm:"index;not null" json:"group_id"`
	Repo            string          `gorm:"type:text;not null;index" json:"repo"`
	Branch          string          `gorm:"type:text" json:"branch"`
	BlobSizeLimit   string          `gorm:"type:text" json:"blob_size_limit"`
	SecurityEmail   string          `gorm:"type:text" json:"security_email"`
	SupportEmail    string          `gorm:"type:text" json:"support_email"`
	PRLabels        JSONStringSlice `gorm:"type:text" json:"pr_labels"`
	PRAssignees     JSONStringSlice `gorm:"type:text" json:"pr_assignees"`
	PRReviewers     JSONStringSlice `gorm:"type:text" json:"pr_reviewers"`
	PRTeamReviewers JSONStringSlice `gorm:"type:text" json:"pr_team_reviewers"`
	Position        int             `gorm:"default:0" json:"position"`

	// Polymorphic relationships
	FileMappings      []FileMapping      `gorm:"polymorphic:Owner;polymorphicValue:target" json:"files,omitempty"`
	DirectoryMappings []DirectoryMapping `gorm:"polymorphic:Owner;polymorphicValue:target" json:"directories,omitempty"`
	Transform         Transform          `gorm:"polymorphic:Owner;polymorphicValue:target" json:"transform,omitempty"`

	// M2M relationships via join tables
	FileListRefs      []TargetFileListRef      `gorm:"foreignKey:TargetID" json:"file_list_refs,omitempty"`
	DirectoryListRefs []TargetDirectoryListRef `gorm:"foreignKey:TargetID" json:"directory_list_refs,omitempty"`
}

// FileList represents a reusable file list (maps to config.FileList)
type FileList struct {
	BaseModel

	ConfigID    uint          `gorm:"index;not null" json:"config_id"`
	ExternalID  string        `gorm:"uniqueIndex;type:text;not null" json:"external_id"`
	Name        string        `gorm:"type:text;not null" json:"name"`
	Description string        `gorm:"type:text" json:"description"`
	Position    int           `gorm:"default:0" json:"position"`
	Files       []FileMapping `gorm:"polymorphic:Owner;polymorphicValue:file_list" json:"files,omitempty"`
}

// DirectoryList represents a reusable directory list (maps to config.DirectoryList)
type DirectoryList struct {
	BaseModel

	ConfigID    uint               `gorm:"index;not null" json:"config_id"`
	ExternalID  string             `gorm:"uniqueIndex;type:text;not null" json:"external_id"`
	Name        string             `gorm:"type:text;not null" json:"name"`
	Description string             `gorm:"type:text" json:"description"`
	Position    int                `gorm:"default:0" json:"position"`
	Directories []DirectoryMapping `gorm:"polymorphic:Owner;polymorphicValue:directory_list" json:"directories,omitempty"`
}

// FileMapping represents a file mapping (polymorphic: Target or FileList)
type FileMapping struct {
	BaseModel

	OwnerType  string `gorm:"type:text;not null;index:idx_file_mapping_owner" json:"owner_type"` // "target" or "file_list"
	OwnerID    uint   `gorm:"not null;index:idx_file_mapping_owner" json:"owner_id"`
	Src        string `gorm:"type:text" json:"src"`
	Dest       string `gorm:"type:text;not null;index" json:"dest"`
	DeleteFlag bool   `gorm:"default:false" json:"delete"`
	Position   int    `gorm:"default:0" json:"position"`
}

// DirectoryMapping represents a directory mapping (polymorphic: Target or DirectoryList)
type DirectoryMapping struct {
	BaseModel

	OwnerType         string            `gorm:"type:text;not null;index:idx_dir_mapping_owner" json:"owner_type"` // "target" or "directory_list"
	OwnerID           uint              `gorm:"not null;index:idx_dir_mapping_owner" json:"owner_id"`
	Src               string            `gorm:"type:text" json:"src"`
	Dest              string            `gorm:"type:text;not null" json:"dest"`
	Exclude           JSONStringSlice   `gorm:"type:text" json:"exclude"`
	IncludeOnly       JSONStringSlice   `gorm:"type:text" json:"include_only"`
	PreserveStructure *bool             `gorm:"default:true" json:"preserve_structure"`
	IncludeHidden     *bool             `gorm:"default:true" json:"include_hidden"`
	DeleteFlag        bool              `gorm:"default:false" json:"delete"`
	ModuleConfig      *JSONModuleConfig `gorm:"type:text" json:"module_config"`
	Position          int               `gorm:"default:0" json:"position"`
	Transform         Transform         `gorm:"polymorphic:Owner;polymorphicValue:directory_mapping" json:"transform,omitempty"`
}

// Transform represents transformation settings (polymorphic: Target or DirectoryMapping)
type Transform struct {
	BaseModel

	OwnerType string        `gorm:"type:text;not null;uniqueIndex:idx_owner_transform" json:"owner_type"` // "target" or "directory_mapping"
	OwnerID   uint          `gorm:"not null;uniqueIndex:idx_owner_transform" json:"owner_id"`
	RepoName  bool          `gorm:"default:false" json:"repo_name"`
	Variables JSONStringMap `gorm:"type:text" json:"variables"`
}

// TargetFileListRef is the join table for Target <-> FileList M2M
type TargetFileListRef struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	TargetID   uint      `gorm:"uniqueIndex:idx_target_file_list;index;not null" json:"target_id"`
	FileListID uint      `gorm:"uniqueIndex:idx_target_file_list;index;not null" json:"file_list_id"`
	Position   int       `gorm:"default:0" json:"position"`
	Metadata   Metadata  `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relationships for preloading
	FileList FileList `gorm:"foreignKey:FileListID" json:"file_list,omitempty"`
}

// TargetDirectoryListRef is the join table for Target <-> DirectoryList M2M
type TargetDirectoryListRef struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	TargetID        uint      `gorm:"uniqueIndex:idx_target_dir_list;index;not null" json:"target_id"`
	DirectoryListID uint      `gorm:"uniqueIndex:idx_target_dir_list;index;not null" json:"directory_list_id"`
	Position        int       `gorm:"default:0" json:"position"`
	Metadata        Metadata  `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships for preloading
	DirectoryList DirectoryList `gorm:"foreignKey:DirectoryListID" json:"directory_list,omitempty"`
}

// SchemaMigration tracks schema versions for migration management
type SchemaMigration struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Version     string    `gorm:"uniqueIndex;type:text;not null" json:"version"`
	AppliedAt   time.Time `gorm:"not null" json:"applied_at"`
	Description string    `gorm:"type:text" json:"description"`
	Checksum    string    `gorm:"type:text" json:"checksum"` // For integrity verification
}
