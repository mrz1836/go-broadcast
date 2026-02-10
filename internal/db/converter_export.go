package db

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/mrz1836/go-broadcast/internal/config"
	"gorm.io/gorm"
)

// ExportConfig exports the database configuration to config.Config
// Ordered by position fields, generates string refs from FKs
func (c *Converter) ExportConfig(ctx context.Context, externalID string) (*config.Config, error) {
	// Find config by external ID
	var dbConfig Config
	if err := c.db.WithContext(ctx).Where("external_id = ?", externalID).First(&dbConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: config %q not found", ErrRecordNotFound, externalID)
		}
		return nil, fmt.Errorf("%w: failed to find config: %w", ErrExportFailed, err)
	}

	cfg := &config.Config{
		Version: dbConfig.Version,
		Name:    dbConfig.Name,
		ID:      dbConfig.ExternalID,
	}

	// Export file lists
	fileLists, err := c.exportFileLists(ctx, dbConfig.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to export file lists: %w", err)
	}
	cfg.FileLists = fileLists

	// Export directory lists
	dirLists, err := c.exportDirectoryLists(ctx, dbConfig.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to export directory lists: %w", err)
	}
	cfg.DirectoryLists = dirLists

	// Build reference maps for reverse lookup (DB ID -> external ID)
	refs, err := c.buildRefMapsFromDB(ctx, dbConfig.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to build reference maps: %w", err)
	}
	reverseRefs := c.buildReverseRefMaps(refs)

	// Export groups
	groups, err := c.exportGroups(ctx, dbConfig.ID, reverseRefs)
	if err != nil {
		return nil, fmt.Errorf("failed to export groups: %w", err)
	}
	cfg.Groups = groups

	return cfg, nil
}

// reverseRefMaps holds database ID to external ID mappings for export
type reverseRefMaps struct {
	fileLists      map[uint]string // db ID -> external_id
	directoryLists map[uint]string // db ID -> external_id
	groups         map[uint]string // db ID -> external_id
}

// buildReverseRefMaps creates reverse lookup maps from refMap
func (c *Converter) buildReverseRefMaps(refs *refMap) *reverseRefMaps {
	reverse := &reverseRefMaps{
		fileLists:      make(map[uint]string),
		directoryLists: make(map[uint]string),
		groups:         make(map[uint]string),
	}

	for extID, dbID := range refs.fileLists {
		reverse.fileLists[dbID] = extID
	}

	for extID, dbID := range refs.directoryLists {
		reverse.directoryLists[dbID] = extID
	}

	for extID, dbID := range refs.groups {
		reverse.groups[dbID] = extID
	}

	return reverse
}

// exportFileLists exports all file lists for a config, ordered by position
func (c *Converter) exportFileLists(ctx context.Context, configID uint) ([]config.FileList, error) {
	var dbFileLists []FileList
	if err := c.db.WithContext(ctx).
		Where("config_id = ?", configID).
		Order("position ASC").
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Find(&dbFileLists).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to load file lists: %w", ErrExportFailed, err)
	}

	fileLists := make([]config.FileList, len(dbFileLists))
	for i, dbFL := range dbFileLists {
		fileLists[i] = config.FileList{
			ID:          dbFL.ExternalID,
			Name:        dbFL.Name,
			Description: dbFL.Description,
			Files:       c.exportFileMappings(dbFL.Files),
		}
	}

	return fileLists, nil
}

// exportDirectoryLists exports all directory lists for a config, ordered by position
func (c *Converter) exportDirectoryLists(ctx context.Context, configID uint) ([]config.DirectoryList, error) {
	var dbDirLists []DirectoryList
	if err := c.db.WithContext(ctx).
		Where("config_id = ?", configID).
		Order("position ASC").
		Preload("Directories", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Directories.Transform"). // Preload transforms for directory mappings
		Find(&dbDirLists).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to load directory lists: %w", ErrExportFailed, err)
	}

	dirLists := make([]config.DirectoryList, len(dbDirLists))
	for i, dbDL := range dbDirLists {
		dirLists[i] = config.DirectoryList{
			ID:          dbDL.ExternalID,
			Name:        dbDL.Name,
			Description: dbDL.Description,
			Directories: c.exportDirectoryMappings(dbDL.Directories),
		}
	}

	return dirLists, nil
}

// exportGroups exports all groups for a config, ordered by position
func (c *Converter) exportGroups(ctx context.Context, configID uint, reverseRefs *reverseRefMaps) ([]config.Group, error) {
	var dbGroups []Group
	if err := c.db.WithContext(ctx).
		Where("config_id = ?", configID).
		Order("position ASC").
		Preload("Source").
		Preload("GroupGlobal").
		Preload("GroupDefault").
		Preload("Dependencies", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Targets", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Targets.FileMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Targets.DirectoryMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Targets.DirectoryMappings.Transform").
		Preload("Targets.Transform").
		Preload("Targets.FileListRefs", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC").Preload("FileList")
		}).
		Preload("Targets.DirectoryListRefs", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC").Preload("DirectoryList")
		}).
		Find(&dbGroups).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to load groups: %w", ErrExportFailed, err)
	}

	groups := make([]config.Group, len(dbGroups))
	for i, dbGroup := range dbGroups {
		groups[i] = config.Group{
			Name:        dbGroup.Name,
			ID:          dbGroup.ExternalID,
			Description: dbGroup.Description,
			Priority:    dbGroup.Priority,
			DependsOn:   c.exportGroupDependencies(dbGroup.Dependencies),
			Enabled:     dbGroup.Enabled,
			Source:      c.exportSource(dbGroup.Source),
			Global:      c.exportGroupGlobal(dbGroup.GroupGlobal),
			Defaults:    c.exportGroupDefault(dbGroup.GroupDefault),
			Targets:     c.exportTargets(dbGroup.Targets, reverseRefs),
		}
	}

	return groups, nil
}

// exportSource converts a Source model to config.SourceConfig
func (c *Converter) exportSource(dbSource Source) config.SourceConfig {
	return config.SourceConfig{
		Repo:          dbSource.Repo,
		Branch:        dbSource.Branch,
		BlobSizeLimit: dbSource.BlobSizeLimit,
		SecurityEmail: dbSource.SecurityEmail,
		SupportEmail:  dbSource.SupportEmail,
	}
}

// exportGroupGlobal converts a GroupGlobal model to config.GlobalConfig
func (c *Converter) exportGroupGlobal(dbGlobal GroupGlobal) config.GlobalConfig {
	return config.GlobalConfig{
		PRLabels:        jsonToStringSlice(dbGlobal.PRLabels),
		PRAssignees:     jsonToStringSlice(dbGlobal.PRAssignees),
		PRReviewers:     jsonToStringSlice(dbGlobal.PRReviewers),
		PRTeamReviewers: jsonToStringSlice(dbGlobal.PRTeamReviewers),
	}
}

// exportGroupDefault converts a GroupDefault model to config.DefaultConfig
func (c *Converter) exportGroupDefault(dbDefault GroupDefault) config.DefaultConfig {
	return config.DefaultConfig{
		BranchPrefix:    dbDefault.BranchPrefix,
		PRLabels:        jsonToStringSlice(dbDefault.PRLabels),
		PRAssignees:     jsonToStringSlice(dbDefault.PRAssignees),
		PRReviewers:     jsonToStringSlice(dbDefault.PRReviewers),
		PRTeamReviewers: jsonToStringSlice(dbDefault.PRTeamReviewers),
	}
}

// exportGroupDependencies converts GroupDependency slice to []string
func (c *Converter) exportGroupDependencies(deps []GroupDependency) []string {
	if len(deps) == 0 {
		return nil
	}

	// Sort by position (already preloaded sorted, but ensure)
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].Position < deps[j].Position
	})

	result := make([]string, len(deps))
	for i, dep := range deps {
		result[i] = dep.DependsOnID
	}

	return result
}

// exportTargets converts Target slice to config.TargetConfig slice
func (c *Converter) exportTargets(dbTargets []Target, reverseRefs *reverseRefMaps) []config.TargetConfig {
	if len(dbTargets) == 0 {
		return nil
	}

	targets := make([]config.TargetConfig, len(dbTargets))
	for i, dbTarget := range dbTargets {
		targets[i] = config.TargetConfig{
			Repo:              dbTarget.Repo,
			Branch:            dbTarget.Branch,
			BlobSizeLimit:     dbTarget.BlobSizeLimit,
			SecurityEmail:     dbTarget.SecurityEmail,
			SupportEmail:      dbTarget.SupportEmail,
			Files:             c.exportFileMappings(dbTarget.FileMappings),
			Directories:       c.exportDirectoryMappings(dbTarget.DirectoryMappings),
			FileListRefs:      c.exportFileListRefs(dbTarget.FileListRefs, reverseRefs),
			DirectoryListRefs: c.exportDirectoryListRefs(dbTarget.DirectoryListRefs, reverseRefs),
			Transform:         c.exportTransform(dbTarget.Transform),
			PRLabels:          jsonToStringSlice(dbTarget.PRLabels),
			PRAssignees:       jsonToStringSlice(dbTarget.PRAssignees),
			PRReviewers:       jsonToStringSlice(dbTarget.PRReviewers),
			PRTeamReviewers:   jsonToStringSlice(dbTarget.PRTeamReviewers),
		}
	}

	return targets
}

// exportFileMappings converts FileMapping slice to config.FileMapping slice
func (c *Converter) exportFileMappings(dbFiles []FileMapping) []config.FileMapping {
	if len(dbFiles) == 0 {
		return nil
	}

	files := make([]config.FileMapping, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = config.FileMapping{
			Src:    dbFile.Src,
			Dest:   dbFile.Dest,
			Delete: dbFile.DeleteFlag,
		}
	}

	return files
}

// exportDirectoryMappings converts DirectoryMapping slice to config.DirectoryMapping slice
func (c *Converter) exportDirectoryMappings(dbDirs []DirectoryMapping) []config.DirectoryMapping {
	if len(dbDirs) == 0 {
		return nil
	}

	dirs := make([]config.DirectoryMapping, len(dbDirs))
	for i, dbDir := range dbDirs {
		dirs[i] = config.DirectoryMapping{
			Src:               dbDir.Src,
			Dest:              dbDir.Dest,
			Exclude:           jsonToStringSlice(dbDir.Exclude),
			IncludeOnly:       jsonToStringSlice(dbDir.IncludeOnly),
			PreserveStructure: dbDir.PreserveStructure,
			IncludeHidden:     dbDir.IncludeHidden,
			Delete:            dbDir.DeleteFlag,
			Module:            jsonToModuleConfig(dbDir.ModuleConfig),
			Transform:         c.exportTransform(dbDir.Transform),
		}
	}

	return dirs
}

// exportTransform converts Transform model to config.Transform
func (c *Converter) exportTransform(dbTransform Transform) config.Transform {
	// Return empty transform if nothing is set
	if !dbTransform.RepoName && len(dbTransform.Variables) == 0 {
		return config.Transform{}
	}

	return config.Transform{
		RepoName:  dbTransform.RepoName,
		Variables: jsonToStringMap(dbTransform.Variables),
	}
}

// exportFileListRefs converts TargetFileListRef slice to []string (external IDs)
func (c *Converter) exportFileListRefs(refs []TargetFileListRef, reverseRefs *reverseRefMaps) []string {
	if len(refs) == 0 {
		return nil
	}

	// Sort by position
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Position < refs[j].Position
	})

	result := make([]string, len(refs))
	for i, ref := range refs {
		// Use preloaded FileList if available, otherwise lookup in reverse map
		if ref.FileList.ID != 0 {
			result[i] = ref.FileList.ExternalID
		} else if extID, exists := reverseRefs.fileLists[ref.FileListID]; exists {
			result[i] = extID
		} else {
			// Fallback: shouldn't happen if references are valid
			result[i] = fmt.Sprintf("unknown-file-list-%d", ref.FileListID)
		}
	}

	return result
}

// exportDirectoryListRefs converts TargetDirectoryListRef slice to []string (external IDs)
func (c *Converter) exportDirectoryListRefs(refs []TargetDirectoryListRef, reverseRefs *reverseRefMaps) []string {
	if len(refs) == 0 {
		return nil
	}

	// Sort by position
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].Position < refs[j].Position
	})

	result := make([]string, len(refs))
	for i, ref := range refs {
		// Use preloaded DirectoryList if available, otherwise lookup in reverse map
		if ref.DirectoryList.ID != 0 {
			result[i] = ref.DirectoryList.ExternalID
		} else if extID, exists := reverseRefs.directoryLists[ref.DirectoryListID]; exists {
			result[i] = extID
		} else {
			// Fallback: shouldn't happen if references are valid
			result[i] = fmt.Sprintf("unknown-directory-list-%d", ref.DirectoryListID)
		}
	}

	return result
}

// ExportGroup exports a single group by external ID
func (c *Converter) ExportGroup(ctx context.Context, configID uint, groupExternalID string) (*config.Group, error) {
	var dbGroup Group
	if err := c.db.WithContext(ctx).
		Where("config_id = ? AND external_id = ?", configID, groupExternalID).
		Preload("Source").
		Preload("GroupGlobal").
		Preload("GroupDefault").
		Preload("Dependencies", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Targets", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Targets.FileMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Targets.DirectoryMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Targets.DirectoryMappings.Transform").
		Preload("Targets.Transform").
		Preload("Targets.FileListRefs", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC").Preload("FileList")
		}).
		Preload("Targets.DirectoryListRefs", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC").Preload("DirectoryList")
		}).
		First(&dbGroup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: group %q not found", ErrRecordNotFound, groupExternalID)
		}
		return nil, fmt.Errorf("%w: failed to load group: %w", ErrExportFailed, err)
	}

	// Build reverse ref maps
	refs, err := c.buildRefMapsFromDB(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("failed to build reference maps: %w", err)
	}
	reverseRefs := c.buildReverseRefMaps(refs)

	group := &config.Group{
		Name:        dbGroup.Name,
		ID:          dbGroup.ExternalID,
		Description: dbGroup.Description,
		Priority:    dbGroup.Priority,
		DependsOn:   c.exportGroupDependencies(dbGroup.Dependencies),
		Enabled:     dbGroup.Enabled,
		Source:      c.exportSource(dbGroup.Source),
		Global:      c.exportGroupGlobal(dbGroup.GroupGlobal),
		Defaults:    c.exportGroupDefault(dbGroup.GroupDefault),
		Targets:     c.exportTargets(dbGroup.Targets, reverseRefs),
	}

	return group, nil
}
