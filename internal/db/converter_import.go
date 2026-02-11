package db

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// ImportOptions contains optional parameters for ImportConfig
type ImportOptions struct {
	SourcePath string
}

// ImportOption is a functional option for ImportConfig
type ImportOption func(*ImportOptions)

// WithSourcePath sets the source file path for enrichment
func WithSourcePath(path string) ImportOption {
	return func(o *ImportOptions) {
		o.SourcePath = path
	}
}

// enrichConfigFields enriches config Name and ID fields if they're empty
// Derives values from the source file path to make them more user-friendly
// Only enriches if a sourcePath is provided; empty sourcePath means no enrichment
func enrichConfigFields(cfg *config.Config, sourcePath string) (name, externalID string) {
	name = cfg.Name
	externalID = cfg.ID

	// Only enrich if sourcePath is provided and values are empty
	if sourcePath != "" && (name == "" || externalID == "") {
		// Derive from filename
		var baseName string
		if sourcePath == "-" {
			// stdin
			baseName = "stdin"
		} else {
			// Get filename without extension
			filename := filepath.Base(sourcePath)
			ext := filepath.Ext(filename)
			baseName = strings.TrimSuffix(filename, ext)

			// If baseName is empty after trimming, use timestamp
			if baseName == "" {
				baseName = fmt.Sprintf("config-%d", time.Now().Unix())
			}
		}

		// Apply defaults if empty
		if name == "" {
			name = baseName
		}
		if externalID == "" {
			externalID = baseName
		}
	}

	return name, externalID
}

// CalculateConfigMetrics analyzes a config and returns metrics and analysis
// Returns two maps: "metrics" with counts and "config_analysis" with feature detection
func CalculateConfigMetrics(cfg *config.Config) map[string]interface{} {
	result := make(map[string]interface{})

	// Initialize metrics
	metrics := make(map[string]interface{})
	analysis := make(map[string]interface{})

	// Basic counts
	metrics["groups_count"] = len(cfg.Groups)
	metrics["file_lists_count"] = len(cfg.FileLists)
	metrics["directory_lists_count"] = len(cfg.DirectoryLists)

	// Detailed group analysis
	var totalTargets, totalFiles, totalDirectories int
	var enabledGroups, disabledGroups int
	hasDependencies := false
	hasTransforms := false
	hasModuleConfigs := false
	sourceRepos := make(map[string]bool)
	targetReposMap := make(map[string]bool)

	// Analyze groups
	for _, group := range cfg.Groups {
		// Enabled is *bool, default is true when nil
		if group.Enabled == nil || *group.Enabled {
			enabledGroups++
		} else {
			disabledGroups++
		}

		// Track source repos
		if group.Source.Repo != "" {
			sourceRepos[group.Source.Repo] = true
		}

		// Check for dependencies
		if len(group.DependsOn) > 0 {
			hasDependencies = true
		}

		// Analyze targets
		totalTargets += len(group.Targets)
		for _, target := range group.Targets {
			// Track target repos
			if target.Repo != "" {
				targetReposMap[target.Repo] = true
			}

			// Count inline files and directories
			totalFiles += len(target.Files)
			totalDirectories += len(target.Directories)

			// Count file list refs
			totalFiles += len(target.FileListRefs)

			// Count directory list refs
			totalDirectories += len(target.DirectoryListRefs)

			// Check for target-level transforms
			if target.Transform.RepoName || len(target.Transform.Variables) > 0 {
				hasTransforms = true
			}

			// Check for directory-level transforms and module configs
			for _, dir := range target.Directories {
				if dir.Transform.RepoName || len(dir.Transform.Variables) > 0 {
					hasTransforms = true
				}
				if dir.Module != nil && (dir.Module.Version != "" || dir.Module.Type != "") {
					hasModuleConfigs = true
				}
			}
		}
	}

	// Add counts to metrics
	metrics["total_targets"] = totalTargets
	metrics["total_files"] = totalFiles
	metrics["total_directories"] = totalDirectories
	metrics["enabled_groups"] = enabledGroups
	metrics["disabled_groups"] = disabledGroups

	// Build source repos list
	sourceReposList := make([]string, 0, len(sourceRepos))
	for repo := range sourceRepos {
		sourceReposList = append(sourceReposList, repo)
	}

	// Add analysis data
	analysis["has_dependencies"] = hasDependencies
	analysis["has_transforms"] = hasTransforms
	analysis["has_module_configs"] = hasModuleConfigs
	analysis["source_repos"] = sourceReposList
	analysis["target_repos_count"] = len(targetReposMap)

	// Combine into result
	result["metrics"] = metrics
	result["config_analysis"] = analysis

	return result
}

// ImportConfig imports a config.Config into the database
// All operations are performed in a single transaction for atomicity
// Returns the created/updated Config record
func (c *Converter) ImportConfig(ctx context.Context, cfg *config.Config, opts ...ImportOption) (*Config, error) {
	var dbConfig *Config
	var refs *refMap

	// Apply options
	options := &ImportOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Enrich config fields if needed
	name, externalID := enrichConfigFields(cfg, options.SourcePath)

	// Perform all operations in a transaction
	err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: Upsert Config record
		dbConfig = &Config{
			ExternalID: externalID,
			Name:       name,
			Version:    cfg.Version,
		}

		var existing Config
		result := tx.Where("external_id = ?", externalID).First(&existing)
		switch {
		case result.Error == nil:
			dbConfig.ID = existing.ID
			dbConfig.CreatedAt = existing.CreatedAt
			if err := tx.Save(dbConfig).Error; err != nil {
				return fmt.Errorf("%w: failed to update config: %w", ErrImportFailed, err)
			}
		case errors.Is(result.Error, gorm.ErrRecordNotFound):
			if err := tx.Create(dbConfig).Error; err != nil {
				return fmt.Errorf("%w: failed to create config: %w", ErrImportFailed, err)
			}
		default:
			return fmt.Errorf("%w: failed to check existing config: %w", ErrImportFailed, result.Error)
		}

		// Step 2: Build reference maps
		refs = buildRefMapsFromConfig(cfg)

		// Step 3: Create all FileLists first (to build ID map)
		if err := c.importFileLists(tx, dbConfig.ID, cfg.FileLists, refs); err != nil {
			return err
		}

		// Step 4: Create all DirectoryLists (to build ID map)
		if err := c.importDirectoryLists(tx, dbConfig.ID, cfg.DirectoryLists, refs); err != nil {
			return err
		}

		// Step 5: Validate all references exist
		if err := c.validateReferences(ctx, cfg, refs); err != nil {
			return err
		}

		// Step 6: Import groups with dependencies check
		if err := c.importGroups(tx, dbConfig.ID, cfg.Groups, refs); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return dbConfig, nil
}

// importFileLists creates or updates file lists and populates the reference map
func (c *Converter) importFileLists(tx *gorm.DB, configID uint, fileLists []config.FileList, refs *refMap) error {
	for i, fl := range fileLists {
		dbFileList := &FileList{
			ConfigID:    configID,
			ExternalID:  fl.ID,
			Name:        fl.Name,
			Description: fl.Description,
			Position:    i,
		}

		// Check if already exists
		var existing FileList
		result := tx.Where("external_id = ?", fl.ID).First(&existing)
		switch {
		case result.Error == nil:
			dbFileList.ID = existing.ID
			dbFileList.CreatedAt = existing.CreatedAt
			if err := tx.Save(dbFileList).Error; err != nil {
				return fmt.Errorf("%w: failed to update file list %q: %w", ErrImportFailed, fl.ID, err)
			}
		case errors.Is(result.Error, gorm.ErrRecordNotFound):
			if err := tx.Create(dbFileList).Error; err != nil {
				return fmt.Errorf("%w: failed to create file list %q: %w", ErrImportFailed, fl.ID, err)
			}
		default:
			return fmt.Errorf("%w: failed to check existing file list %q: %w", ErrImportFailed, fl.ID, result.Error)
		}

		// Update reference map
		refs.fileLists[fl.ID] = dbFileList.ID

		// Import file mappings
		if err := c.importFileMappings(tx, "file_list", dbFileList.ID, fl.Files); err != nil {
			return fmt.Errorf("failed to import file mappings for list %q: %w", fl.ID, err)
		}
	}

	return nil
}

// importDirectoryLists creates or updates directory lists and populates the reference map
func (c *Converter) importDirectoryLists(tx *gorm.DB, configID uint, directoryLists []config.DirectoryList, refs *refMap) error {
	for i, dl := range directoryLists {
		dbDirList := &DirectoryList{
			ConfigID:    configID,
			ExternalID:  dl.ID,
			Name:        dl.Name,
			Description: dl.Description,
			Position:    i,
		}

		// Check if already exists
		var existing DirectoryList
		result := tx.Where("external_id = ?", dl.ID).First(&existing)
		switch {
		case result.Error == nil:
			dbDirList.ID = existing.ID
			dbDirList.CreatedAt = existing.CreatedAt
			if err := tx.Save(dbDirList).Error; err != nil {
				return fmt.Errorf("%w: failed to update directory list %q: %w", ErrImportFailed, dl.ID, err)
			}
		case errors.Is(result.Error, gorm.ErrRecordNotFound):
			if err := tx.Create(dbDirList).Error; err != nil {
				return fmt.Errorf("%w: failed to create directory list %q: %w", ErrImportFailed, dl.ID, err)
			}
		default:
			return fmt.Errorf("%w: failed to check existing directory list %q: %w", ErrImportFailed, dl.ID, result.Error)
		}

		// Update reference map
		refs.directoryLists[dl.ID] = dbDirList.ID

		// Import directory mappings
		if err := c.importDirectoryMappings(tx, "directory_list", dbDirList.ID, dl.Directories); err != nil {
			return fmt.Errorf("failed to import directory mappings for list %q: %w", dl.ID, err)
		}
	}

	return nil
}

// importGroups creates or updates groups with all associations
func (c *Converter) importGroups(tx *gorm.DB, configID uint, groups []config.Group, refs *refMap) error {
	// Detect circular dependencies before import
	if err := detectCircularDependencies(groups); err != nil {
		return err
	}

	for i, group := range groups {
		dbGroup := &Group{
			ConfigID:    configID,
			ExternalID:  group.ID,
			Name:        group.Name,
			Description: group.Description,
			Priority:    group.Priority,
			Enabled:     group.Enabled,
			Position:    i,
		}

		// Check if already exists
		var existing Group
		result := tx.Where("external_id = ?", group.ID).First(&existing)
		switch {
		case result.Error == nil:
			dbGroup.ID = existing.ID
			dbGroup.CreatedAt = existing.CreatedAt
			if err := tx.Save(dbGroup).Error; err != nil {
				return fmt.Errorf("%w: failed to update group %q: %w", ErrImportFailed, group.ID, err)
			}
			if err := c.deleteGroupAssociations(tx, dbGroup.ID); err != nil {
				return fmt.Errorf("failed to delete old associations for group %q: %w", group.ID, err)
			}
		case errors.Is(result.Error, gorm.ErrRecordNotFound):
			if err := tx.Create(dbGroup).Error; err != nil {
				return fmt.Errorf("%w: failed to create group %q: %w", ErrImportFailed, group.ID, err)
			}
		default:
			return fmt.Errorf("%w: failed to check existing group %q: %w", ErrImportFailed, group.ID, result.Error)
		}

		// Update reference map
		refs.groups[group.ID] = dbGroup.ID

		// Import source
		if err := c.importSource(tx, dbGroup.ID, &group.Source, refs); err != nil {
			return fmt.Errorf("failed to import source for group %q: %w", group.ID, err)
		}

		// Import global config
		if err := c.importGroupGlobal(tx, dbGroup.ID, &group.Global); err != nil {
			return fmt.Errorf("failed to import global config for group %q: %w", group.ID, err)
		}

		// Import default config
		if err := c.importGroupDefault(tx, dbGroup.ID, &group.Defaults); err != nil {
			return fmt.Errorf("failed to import default config for group %q: %w", group.ID, err)
		}

		// Import dependencies
		if err := c.importGroupDependencies(tx, dbGroup.ID, group.DependsOn); err != nil {
			return fmt.Errorf("failed to import dependencies for group %q: %w", group.ID, err)
		}

		// Import targets
		if err := c.importTargets(tx, dbGroup.ID, group.Targets, refs); err != nil {
			return fmt.Errorf("failed to import targets for group %q: %w", group.ID, err)
		}
	}

	return nil
}

// deleteGroupAssociations deletes all associations for a group (for update scenario)
func (c *Converter) deleteGroupAssociations(tx *gorm.DB, groupID uint) error {
	// Delete targets (cascade will handle file/dir mappings, transforms, refs)
	if err := tx.Unscoped().Where("group_id = ?", groupID).Delete(&Target{}).Error; err != nil {
		return fmt.Errorf("failed to delete targets: %w", err)
	}

	// Delete dependencies
	if err := tx.Unscoped().Where("group_id = ?", groupID).Delete(&GroupDependency{}).Error; err != nil {
		return fmt.Errorf("failed to delete dependencies: %w", err)
	}

	// Source, GroupGlobal, GroupDefault are 1:1 and will be updated/created

	return nil
}

// resolveRepoID resolves a "org/repo" string to a Repo.ID, creating Client/Org/Repo as needed
func (c *Converter) resolveRepoID(tx *gorm.DB, fullName string, refs *refMap) (uint, error) {
	// Check cache first
	if id, ok := refs.repos[fullName]; ok {
		return id, nil
	}

	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, fmt.Errorf("%w: expected org/repo, got %q", ErrInvalidRepoFormat, fullName)
	}
	orgName := parts[0]
	repoName := parts[1]

	// Find or create organization
	var orgID uint
	if id, ok := refs.organizations[orgName]; ok {
		orgID = id
	} else {
		var org Organization
		err := tx.Where("name = ?", orgName).First(&org).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Auto-create client with same name as org
			client := Client{Name: orgName}
			if cErr := tx.Create(&client).Error; cErr != nil {
				return 0, fmt.Errorf("failed to create client for org %q: %w", orgName, cErr)
			}
			org = Organization{ClientID: client.ID, Name: orgName}
			if err = tx.Create(&org).Error; err != nil {
				return 0, fmt.Errorf("failed to create organization %q: %w", orgName, err)
			}
		} else if err != nil {
			return 0, err
		}
		orgID = org.ID
		refs.organizations[orgName] = orgID
	}

	// Find or create repo
	var repo Repo
	err := tx.Where("organization_id = ? AND name = ?", orgID, repoName).First(&repo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		repo = Repo{OrganizationID: orgID, Name: repoName}
		if err = tx.Create(&repo).Error; err != nil {
			return 0, fmt.Errorf("failed to create repo %q: %w", fullName, err)
		}
	} else if err != nil {
		return 0, err
	}

	refs.repos[fullName] = repo.ID
	return repo.ID, nil
}

// importSource creates or updates the source config for a group
func (c *Converter) importSource(tx *gorm.DB, groupID uint, source *config.SourceConfig, refs *refMap) error {
	repoID, err := c.resolveRepoID(tx, source.Repo, refs)
	if err != nil {
		return fmt.Errorf("failed to resolve source repo %q: %w", source.Repo, err)
	}

	dbSource := &Source{
		GroupID:       groupID,
		RepoID:        repoID,
		Branch:        source.Branch,
		BlobSizeLimit: source.BlobSizeLimit,
		SecurityEmail: source.SecurityEmail,
		SupportEmail:  source.SupportEmail,
	}

	// Check if exists (1:1 relationship)
	var existing Source
	result := tx.Where("group_id = ?", groupID).First(&existing)
	if result.Error == nil {
		dbSource.ID = existing.ID
		dbSource.CreatedAt = existing.CreatedAt
		return tx.Save(dbSource).Error
	}

	return tx.Create(dbSource).Error
}

// importGroupGlobal creates or updates the global config for a group
func (c *Converter) importGroupGlobal(tx *gorm.DB, groupID uint, global *config.GlobalConfig) error {
	dbGlobal := &GroupGlobal{
		GroupID:         groupID,
		PRLabels:        stringSliceToJSON(global.PRLabels),
		PRAssignees:     stringSliceToJSON(global.PRAssignees),
		PRReviewers:     stringSliceToJSON(global.PRReviewers),
		PRTeamReviewers: stringSliceToJSON(global.PRTeamReviewers),
	}

	var existing GroupGlobal
	result := tx.Where("group_id = ?", groupID).First(&existing)
	if result.Error == nil {
		dbGlobal.ID = existing.ID
		dbGlobal.CreatedAt = existing.CreatedAt
		return tx.Save(dbGlobal).Error
	}

	return tx.Create(dbGlobal).Error
}

// importGroupDefault creates or updates the default config for a group
func (c *Converter) importGroupDefault(tx *gorm.DB, groupID uint, defaults *config.DefaultConfig) error {
	dbDefault := &GroupDefault{
		GroupID:         groupID,
		BranchPrefix:    defaults.BranchPrefix,
		PRLabels:        stringSliceToJSON(defaults.PRLabels),
		PRAssignees:     stringSliceToJSON(defaults.PRAssignees),
		PRReviewers:     stringSliceToJSON(defaults.PRReviewers),
		PRTeamReviewers: stringSliceToJSON(defaults.PRTeamReviewers),
	}

	var existing GroupDefault
	result := tx.Where("group_id = ?", groupID).First(&existing)
	if result.Error == nil {
		dbDefault.ID = existing.ID
		dbDefault.CreatedAt = existing.CreatedAt
		return tx.Save(dbDefault).Error
	}

	return tx.Create(dbDefault).Error
}

// importGroupDependencies creates group dependency records
func (c *Converter) importGroupDependencies(tx *gorm.DB, groupID uint, dependsOn []string) error {
	for i, depID := range dependsOn {
		dep := &GroupDependency{
			GroupID:     groupID,
			DependsOnID: depID,
			Position:    i,
		}
		if err := tx.Create(dep).Error; err != nil {
			return fmt.Errorf("failed to create dependency on %q: %w", depID, err)
		}
	}

	return nil
}

// importTargets creates or updates targets with all associations
func (c *Converter) importTargets(tx *gorm.DB, groupID uint, targets []config.TargetConfig, refs *refMap) error {
	for i, target := range targets {
		repoID, err := c.resolveRepoID(tx, target.Repo, refs)
		if err != nil {
			return fmt.Errorf("failed to resolve target repo %q: %w", target.Repo, err)
		}

		dbTarget := &Target{
			GroupID:         groupID,
			RepoID:          repoID,
			Branch:          target.Branch,
			BlobSizeLimit:   target.BlobSizeLimit,
			SecurityEmail:   target.SecurityEmail,
			SupportEmail:    target.SupportEmail,
			PRLabels:        stringSliceToJSON(target.PRLabels),
			PRAssignees:     stringSliceToJSON(target.PRAssignees),
			PRReviewers:     stringSliceToJSON(target.PRReviewers),
			PRTeamReviewers: stringSliceToJSON(target.PRTeamReviewers),
			Position:        i,
		}

		// Create target (we already deleted old ones in deleteGroupAssociations)
		if err := tx.Create(dbTarget).Error; err != nil {
			return fmt.Errorf("failed to create target %q: %w", target.Repo, err)
		}

		// Import inline file mappings
		if err := c.importFileMappings(tx, "target", dbTarget.ID, target.Files); err != nil {
			return fmt.Errorf("failed to import file mappings for target %q: %w", target.Repo, err)
		}

		// Import inline directory mappings
		if err := c.importDirectoryMappings(tx, "target", dbTarget.ID, target.Directories); err != nil {
			return fmt.Errorf("failed to import directory mappings for target %q: %w", target.Repo, err)
		}

		// Import target-level transform
		if target.Transform.RepoName || len(target.Transform.Variables) > 0 {
			if err := c.importTransform(tx, "target", dbTarget.ID, &target.Transform); err != nil {
				return fmt.Errorf("failed to import transform for target %q: %w", target.Repo, err)
			}
		}

		// Import file list references
		for j, fileListRef := range target.FileListRefs {
			fileListID, exists := refs.fileLists[fileListRef]
			if !exists {
				return fmt.Errorf("%w: file list %q not found", ErrReferenceNotFound, fileListRef)
			}

			ref := &TargetFileListRef{
				TargetID:   dbTarget.ID,
				FileListID: fileListID,
				Position:   j,
			}
			if err := tx.Create(ref).Error; err != nil {
				return fmt.Errorf("failed to create file list ref %q: %w", fileListRef, err)
			}
		}

		// Import directory list references
		for j, dirListRef := range target.DirectoryListRefs {
			dirListID, exists := refs.directoryLists[dirListRef]
			if !exists {
				return fmt.Errorf("%w: directory list %q not found", ErrReferenceNotFound, dirListRef)
			}

			ref := &TargetDirectoryListRef{
				TargetID:        dbTarget.ID,
				DirectoryListID: dirListID,
				Position:        j,
			}
			if err := tx.Create(ref).Error; err != nil {
				return fmt.Errorf("failed to create directory list ref %q: %w", dirListRef, err)
			}
		}
	}

	return nil
}

// importFileMappings creates file mapping records
func (c *Converter) importFileMappings(tx *gorm.DB, ownerType string, ownerID uint, files []config.FileMapping) error {
	for i, file := range files {
		dbFile := &FileMapping{
			OwnerType:  ownerType,
			OwnerID:    ownerID,
			Src:        file.Src,
			Dest:       file.Dest,
			DeleteFlag: file.Delete,
			Position:   i,
		}
		if err := tx.Create(dbFile).Error; err != nil {
			return fmt.Errorf("failed to create file mapping %q: %w", file.Dest, err)
		}
	}

	return nil
}

// importDirectoryMappings creates directory mapping records with transforms
func (c *Converter) importDirectoryMappings(tx *gorm.DB, ownerType string, ownerID uint, dirs []config.DirectoryMapping) error {
	for i, dir := range dirs {
		dbDir := &DirectoryMapping{
			OwnerType:         ownerType,
			OwnerID:           ownerID,
			Src:               dir.Src,
			Dest:              dir.Dest,
			Exclude:           stringSliceToJSON(dir.Exclude),
			IncludeOnly:       stringSliceToJSON(dir.IncludeOnly),
			PreserveStructure: dir.PreserveStructure,
			IncludeHidden:     dir.IncludeHidden,
			DeleteFlag:        dir.Delete,
			ModuleConfig:      moduleConfigToJSON(dir.Module),
			Position:          i,
		}
		if err := tx.Create(dbDir).Error; err != nil {
			return fmt.Errorf("failed to create directory mapping %q: %w", dir.Dest, err)
		}

		// Import directory-level transform
		if dir.Transform.RepoName || len(dir.Transform.Variables) > 0 {
			if err := c.importTransform(tx, "directory_mapping", dbDir.ID, &dir.Transform); err != nil {
				return fmt.Errorf("failed to import transform for directory %q: %w", dir.Dest, err)
			}
		}
	}

	return nil
}

// importTransform creates a transform record
func (c *Converter) importTransform(tx *gorm.DB, ownerType string, ownerID uint, transform *config.Transform) error {
	dbTransform := &Transform{
		OwnerType: ownerType,
		OwnerID:   ownerID,
		RepoName:  transform.RepoName,
		Variables: stringMapToJSON(transform.Variables),
	}

	return tx.Create(dbTransform).Error
}

// detectCircularDependencies checks for circular dependencies in group dependencies
// Uses a simplified approach since we don't have DB groups yet, just config
func detectCircularDependencies(groups []config.Group) error {
	// Build adjacency map: group.ID -> []DependsOn
	edges := make(map[string][]string)
	allIDs := make(map[string]bool)

	for _, group := range groups {
		allIDs[group.ID] = true
		edges[group.ID] = group.DependsOn
	}

	// Validate all dependencies exist
	for _, group := range groups {
		for _, dep := range group.DependsOn {
			if !allIDs[dep] {
				return fmt.Errorf("%w: group %q depends on non-existent group %q", ErrReferenceNotFound, group.ID, dep)
			}
		}
	}

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range edges[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found a cycle
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Check each component
	for id := range allIDs {
		if !visited[id] {
			if dfs(id) {
				return fmt.Errorf("%w: circular dependency detected in group dependencies", ErrCircularDependency)
			}
		}
	}

	return nil
}
