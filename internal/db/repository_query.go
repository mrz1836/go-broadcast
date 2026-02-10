package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// queryRepository implements QueryRepository
type queryRepository struct {
	db *gorm.DB
}

// NewQueryRepository creates a new QueryRepository
func NewQueryRepository(db *gorm.DB) QueryRepository {
	return &queryRepository{db: db}
}

// FindByFile finds all targets that sync a specific file path
func (r *queryRepository) FindByFile(ctx context.Context, filePath string) ([]*Target, error) {
	var targets []*Target

	// Query targets that have this file in their inline file mappings
	if err := r.db.WithContext(ctx).
		Distinct("targets.*").
		Joins("JOIN file_mappings ON file_mappings.owner_id = targets.id AND file_mappings.owner_type = ?", "target").
		Where("file_mappings.dest = ? OR file_mappings.src = ?", filePath, filePath).
		Preload("FileMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("failed to find targets by file: %w", err)
	}

	// Also query targets that reference file lists containing this file
	var targetsViaLists []*Target
	if err := r.db.WithContext(ctx).
		Distinct("targets.*").
		Joins("JOIN target_file_list_refs ON target_file_list_refs.target_id = targets.id").
		Joins("JOIN file_lists ON file_lists.id = target_file_list_refs.file_list_id").
		Joins("JOIN file_mappings ON file_mappings.owner_id = file_lists.id AND file_mappings.owner_type = ?", "file_list").
		Where("file_mappings.dest = ? OR file_mappings.src = ?", filePath, filePath).
		Preload("FileListRefs.FileList.Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Find(&targetsViaLists).Error; err != nil {
		return nil, fmt.Errorf("failed to find targets by file via lists: %w", err)
	}

	// Merge results (deduplicate by ID)
	targetMap := make(map[uint]*Target)
	for _, t := range targets {
		targetMap[t.ID] = t
	}
	for _, t := range targetsViaLists {
		if _, exists := targetMap[t.ID]; !exists {
			targetMap[t.ID] = t
		}
	}

	result := make([]*Target, 0, len(targetMap))
	for _, t := range targetMap {
		result = append(result, t)
	}

	return result, nil
}

// FindByRepo finds all file/directory mappings for a specific repo
func (r *queryRepository) FindByRepo(ctx context.Context, repo string) (*Target, error) {
	var target Target
	if err := r.db.WithContext(ctx).
		Preload("FileMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("DirectoryMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("DirectoryMappings.Transform").
		Preload("Transform").
		Preload("FileListRefs.FileList.Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("DirectoryListRefs.DirectoryList.Directories", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Where("repo = ?", repo).
		First(&target).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to find target by repo: %w", err)
	}
	return &target, nil
}

// FindByFileList finds all targets that reference a specific file list
func (r *queryRepository) FindByFileList(ctx context.Context, fileListID uint) ([]*Target, error) {
	var targets []*Target
	if err := r.db.WithContext(ctx).
		Joins("JOIN target_file_list_refs ON target_file_list_refs.target_id = targets.id").
		Where("target_file_list_refs.file_list_id = ?", fileListID).
		Preload("FileListRefs.FileList").
		Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("failed to find targets by file list: %w", err)
	}
	return targets, nil
}

// FindByDirectoryList finds all targets that reference a specific directory list
func (r *queryRepository) FindByDirectoryList(ctx context.Context, directoryListID uint) ([]*Target, error) {
	var targets []*Target
	if err := r.db.WithContext(ctx).
		Joins("JOIN target_directory_list_refs ON target_directory_list_refs.target_id = targets.id").
		Where("target_directory_list_refs.directory_list_id = ?", directoryListID).
		Preload("DirectoryListRefs.DirectoryList").
		Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("failed to find targets by directory list: %w", err)
	}
	return targets, nil
}

// FindByPattern searches file paths matching a pattern (LIKE query)
func (r *queryRepository) FindByPattern(ctx context.Context, pattern string) ([]*FileMapping, error) {
	var fileMappings []*FileMapping
	likePattern := "%" + pattern + "%"
	if err := r.db.WithContext(ctx).
		Where("dest LIKE ? OR src LIKE ?", likePattern, likePattern).
		Order("dest ASC").
		Find(&fileMappings).Error; err != nil {
		return nil, fmt.Errorf("failed to find file mappings by pattern: %w", err)
	}
	return fileMappings, nil
}
