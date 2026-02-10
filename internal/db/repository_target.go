package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// targetRepository implements TargetRepository
type targetRepository struct {
	db *gorm.DB
}

// NewTargetRepository creates a new TargetRepository
func NewTargetRepository(db *gorm.DB) TargetRepository {
	return &targetRepository{db: db}
}

// Create creates a new target
func (r *targetRepository) Create(ctx context.Context, target *Target) error {
	if err := r.db.WithContext(ctx).Create(target).Error; err != nil {
		return fmt.Errorf("failed to create target: %w", err)
	}
	return nil
}

// GetByID retrieves a target by database ID
func (r *targetRepository) GetByID(ctx context.Context, id uint) (*Target, error) {
	var target Target
	if err := r.db.WithContext(ctx).First(&target, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get target by id: %w", err)
	}
	return &target, nil
}

// GetByRepo retrieves a target by group ID and repo name
func (r *targetRepository) GetByRepo(ctx context.Context, groupID uint, repo string) (*Target, error) {
	var target Target
	if err := r.db.WithContext(ctx).
		Where("group_id = ? AND repo = ?", groupID, repo).
		First(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get target by repo: %w", err)
	}
	return &target, nil
}

// Update updates an existing target
func (r *targetRepository) Update(ctx context.Context, target *Target) error {
	if err := r.db.WithContext(ctx).Save(target).Error; err != nil {
		return fmt.Errorf("failed to update target: %w", err)
	}
	return nil
}

// Delete soft-deletes or hard-deletes a target
func (r *targetRepository) Delete(ctx context.Context, id uint, hard bool) error {
	if hard {
		if err := r.db.WithContext(ctx).Unscoped().Delete(&Target{}, id).Error; err != nil {
			return fmt.Errorf("failed to hard delete target: %w", err)
		}
	} else {
		if err := r.db.WithContext(ctx).Delete(&Target{}, id).Error; err != nil {
			return fmt.Errorf("failed to soft delete target: %w", err)
		}
	}
	return nil
}

// List retrieves all targets for a group
func (r *targetRepository) List(ctx context.Context, groupID uint) ([]*Target, error) {
	var targets []*Target
	if err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("position ASC").
		Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("failed to list targets: %w", err)
	}
	return targets, nil
}

// ListWithAssociations retrieves all targets with full preloading
// Preloads: FileMappings, DirectoryMappings, Transform, FileListRefs, DirectoryListRefs
func (r *targetRepository) ListWithAssociations(ctx context.Context, groupID uint) ([]*Target, error) {
	var targets []*Target
	if err := r.db.WithContext(ctx).
		Preload("FileMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("DirectoryMappings", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("DirectoryMappings.Transform").
		Preload("Transform").
		Preload("FileListRefs", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("FileListRefs.FileList").
		Preload("DirectoryListRefs", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("DirectoryListRefs.DirectoryList").
		Where("group_id = ?", groupID).
		Order("position ASC").
		Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("failed to list targets with associations: %w", err)
	}
	return targets, nil
}

// AddFileListRef adds a file list reference to a target
func (r *targetRepository) AddFileListRef(ctx context.Context, targetID, fileListID uint, position int) error {
	ref := &TargetFileListRef{
		TargetID:   targetID,
		FileListID: fileListID,
		Position:   position,
	}
	if err := r.db.WithContext(ctx).Create(ref).Error; err != nil {
		return fmt.Errorf("failed to add file list ref: %w", err)
	}
	return nil
}

// RemoveFileListRef removes a file list reference from a target
func (r *targetRepository) RemoveFileListRef(ctx context.Context, targetID, fileListID uint) error {
	if err := r.db.WithContext(ctx).
		Where("target_id = ? AND file_list_id = ?", targetID, fileListID).
		Delete(&TargetFileListRef{}).Error; err != nil {
		return fmt.Errorf("failed to remove file list ref: %w", err)
	}
	return nil
}

// AddDirectoryListRef adds a directory list reference to a target
func (r *targetRepository) AddDirectoryListRef(ctx context.Context, targetID, directoryListID uint, position int) error {
	ref := &TargetDirectoryListRef{
		TargetID:        targetID,
		DirectoryListID: directoryListID,
		Position:        position,
	}
	if err := r.db.WithContext(ctx).Create(ref).Error; err != nil {
		return fmt.Errorf("failed to add directory list ref: %w", err)
	}
	return nil
}

// RemoveDirectoryListRef removes a directory list reference from a target
func (r *targetRepository) RemoveDirectoryListRef(ctx context.Context, targetID, directoryListID uint) error {
	if err := r.db.WithContext(ctx).
		Where("target_id = ? AND directory_list_id = ?", targetID, directoryListID).
		Delete(&TargetDirectoryListRef{}).Error; err != nil {
		return fmt.Errorf("failed to remove directory list ref: %w", err)
	}
	return nil
}
