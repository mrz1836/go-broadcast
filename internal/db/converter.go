package db

import (
	"context"
	"fmt"

	"github.com/mrz1836/go-broadcast/internal/config"
	"gorm.io/gorm"
)

// Converter handles bidirectional conversion between config.Config (YAML) and DB models
type Converter struct {
	db *gorm.DB
}

// NewConverter creates a new converter instance
func NewConverter(db *gorm.DB) *Converter {
	return &Converter{db: db}
}

// refMap holds external ID to database ID mappings for reference resolution
type refMap struct {
	fileLists      map[string]uint // external_id -> db ID
	directoryLists map[string]uint // external_id -> db ID
	groups         map[string]uint // external_id -> db ID
}

// newRefMap creates an empty reference map
func newRefMap() *refMap {
	return &refMap{
		fileLists:      make(map[string]uint),
		directoryLists: make(map[string]uint),
		groups:         make(map[string]uint),
	}
}

// Helper functions for pointer types

// boolPtr creates a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// boolVal safely dereferences a bool pointer, returns default if nil
func boolVal(b *bool, defaultVal bool) bool {
	if b == nil {
		return defaultVal
	}
	return *b
}

// stringSliceToJSON converts []string to JSONStringSlice
func stringSliceToJSON(s []string) JSONStringSlice {
	if s == nil {
		return nil
	}
	return JSONStringSlice(s)
}

// jsonToStringSlice converts JSONStringSlice to []string
func jsonToStringSlice(j JSONStringSlice) []string {
	if j == nil {
		return nil
	}
	return []string(j)
}

// stringMapToJSON converts map[string]string to JSONStringMap
func stringMapToJSON(m map[string]string) JSONStringMap {
	if m == nil {
		return nil
	}
	return JSONStringMap(m)
}

// jsonToStringMap converts JSONStringMap to map[string]string
func jsonToStringMap(j JSONStringMap) map[string]string {
	if j == nil {
		return nil
	}
	return map[string]string(j)
}

// moduleConfigToJSON converts config.ModuleConfig to JSONModuleConfig
func moduleConfigToJSON(m *config.ModuleConfig) *JSONModuleConfig {
	if m == nil {
		return nil
	}
	return &JSONModuleConfig{
		Type:       m.Type,
		Version:    m.Version,
		CheckTags:  boolVal(m.CheckTags, true),
		UpdateRefs: m.UpdateRefs,
	}
}

// jsonToModuleConfig converts JSONModuleConfig to config.ModuleConfig
func jsonToModuleConfig(j *JSONModuleConfig) *config.ModuleConfig {
	if j == nil {
		return nil
	}
	checkTags := j.CheckTags
	return &config.ModuleConfig{
		Type:       j.Type,
		Version:    j.Version,
		CheckTags:  &checkTags,
		UpdateRefs: j.UpdateRefs,
	}
}

// validateReferences checks that all external ID references exist in the database
func (c *Converter) validateReferences(_ context.Context, cfg *config.Config, refs *refMap) error {
	// Validate group dependencies
	for _, group := range cfg.Groups {
		for _, depID := range group.DependsOn {
			if _, exists := refs.groups[depID]; !exists {
				return fmt.Errorf("%w: group dependency %q not found in config", ErrReferenceNotFound, depID)
			}
		}
	}

	// Validate file list references
	for _, group := range cfg.Groups {
		for _, target := range group.Targets {
			for _, fileListRef := range target.FileListRefs {
				if _, exists := refs.fileLists[fileListRef]; !exists {
					return fmt.Errorf("%w: file list %q not found in config", ErrReferenceNotFound, fileListRef)
				}
			}
		}
	}

	// Validate directory list references
	for _, group := range cfg.Groups {
		for _, target := range group.Targets {
			for _, dirListRef := range target.DirectoryListRefs {
				if _, exists := refs.directoryLists[dirListRef]; !exists {
					return fmt.Errorf("%w: directory list %q not found in config", ErrReferenceNotFound, dirListRef)
				}
			}
		}
	}

	return nil
}

// buildRefMapsFromConfig builds reference maps from config before import
func buildRefMapsFromConfig(cfg *config.Config) *refMap {
	refs := newRefMap()

	// Map groups by external ID (will be populated during import)
	for _, group := range cfg.Groups {
		refs.groups[group.ID] = 0 // Placeholder, will be set during import
	}

	// Map file lists
	for _, fileList := range cfg.FileLists {
		refs.fileLists[fileList.ID] = 0 // Placeholder
	}

	// Map directory lists
	for _, dirList := range cfg.DirectoryLists {
		refs.directoryLists[dirList.ID] = 0 // Placeholder
	}

	return refs
}

// buildRefMapsFromDB builds reference maps from existing database records
func (c *Converter) buildRefMapsFromDB(ctx context.Context, configID uint) (*refMap, error) {
	refs := newRefMap()

	// Load file lists
	var fileLists []FileList
	if err := c.db.WithContext(ctx).Where("config_id = ?", configID).Find(&fileLists).Error; err != nil {
		return nil, fmt.Errorf("failed to load file lists: %w", err)
	}
	for _, fl := range fileLists {
		refs.fileLists[fl.ExternalID] = fl.ID
	}

	// Load directory lists
	var dirLists []DirectoryList
	if err := c.db.WithContext(ctx).Where("config_id = ?", configID).Find(&dirLists).Error; err != nil {
		return nil, fmt.Errorf("failed to load directory lists: %w", err)
	}
	for _, dl := range dirLists {
		refs.directoryLists[dl.ExternalID] = dl.ID
	}

	// Load groups
	var groups []Group
	if err := c.db.WithContext(ctx).Where("config_id = ?", configID).Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to load groups: %w", err)
	}
	for _, g := range groups {
		refs.groups[g.ExternalID] = g.ID
	}

	return refs, nil
}
