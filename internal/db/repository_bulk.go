package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// BulkRepository provides bulk operations across multiple targets
type BulkRepository interface {
	AddFileListToAllTargets(ctx context.Context, groupID, fileListID uint) (affected int, err error)
	RemoveFileListFromAllTargets(ctx context.Context, groupID, fileListID uint) (affected int, err error)
	AddDirectoryListToAllTargets(ctx context.Context, groupID, directoryListID uint) (affected int, err error)
	RemoveDirectoryListFromAllTargets(ctx context.Context, groupID, directoryListID uint) (affected int, err error)
}

type bulkRepository struct {
	db *gorm.DB
}

// NewBulkRepository creates a new BulkRepository
func NewBulkRepository(db *gorm.DB) BulkRepository {
	return &bulkRepository{db: db}
}

// AddFileListToAllTargets adds a file list reference to all targets in a group.
// Skips targets that already have the reference. Returns the count of newly added references.
func (r *bulkRepository) AddFileListToAllTargets(ctx context.Context, groupID, fileListID uint) (int, error) {
	var affected int

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get all targets in group
		var targets []Target
		if err := tx.Where("group_id = ?", groupID).Find(&targets).Error; err != nil {
			return fmt.Errorf("failed to list targets: %w", err)
		}

		for _, target := range targets {
			// Check if ref already exists
			var existing TargetFileListRef
			err := tx.Where("target_id = ? AND file_list_id = ?", target.ID, fileListID).First(&existing).Error
			if err == nil {
				continue // Already has this ref
			}

			// Find max position for this target's refs
			var maxPos int
			tx.Model(&TargetFileListRef{}).Where("target_id = ?", target.ID).
				Select("COALESCE(MAX(position), -1)").Scan(&maxPos)

			ref := &TargetFileListRef{
				TargetID:   target.ID,
				FileListID: fileListID,
				Position:   maxPos + 1,
			}
			if err := tx.Create(ref).Error; err != nil {
				return fmt.Errorf("failed to add file list ref to target %d: %w", target.ID, err)
			}
			affected++
		}
		return nil
	})

	return affected, err
}

// RemoveFileListFromAllTargets removes a file list reference from all targets in a group.
// Returns the count of removed references.
func (r *bulkRepository) RemoveFileListFromAllTargets(ctx context.Context, groupID, fileListID uint) (int, error) {
	var affected int

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get all targets in group
		var targets []Target
		if err := tx.Where("group_id = ?", groupID).Find(&targets).Error; err != nil {
			return fmt.Errorf("failed to list targets: %w", err)
		}

		for _, target := range targets {
			result := tx.Where("target_id = ? AND file_list_id = ?", target.ID, fileListID).
				Delete(&TargetFileListRef{})
			if result.Error != nil {
				return fmt.Errorf("failed to remove file list ref from target %d: %w", target.ID, result.Error)
			}
			affected += int(result.RowsAffected)
		}
		return nil
	})

	return affected, err
}

// AddDirectoryListToAllTargets adds a directory list reference to all targets in a group.
// Skips targets that already have the reference. Returns the count of newly added references.
func (r *bulkRepository) AddDirectoryListToAllTargets(ctx context.Context, groupID, directoryListID uint) (int, error) {
	var affected int

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var targets []Target
		if err := tx.Where("group_id = ?", groupID).Find(&targets).Error; err != nil {
			return fmt.Errorf("failed to list targets: %w", err)
		}

		for _, target := range targets {
			// Check if ref already exists
			var existing TargetDirectoryListRef
			err := tx.Where("target_id = ? AND directory_list_id = ?", target.ID, directoryListID).First(&existing).Error
			if err == nil {
				continue
			}

			var maxPos int
			tx.Model(&TargetDirectoryListRef{}).Where("target_id = ?", target.ID).
				Select("COALESCE(MAX(position), -1)").Scan(&maxPos)

			ref := &TargetDirectoryListRef{
				TargetID:        target.ID,
				DirectoryListID: directoryListID,
				Position:        maxPos + 1,
			}
			if err := tx.Create(ref).Error; err != nil {
				return fmt.Errorf("failed to add directory list ref to target %d: %w", target.ID, err)
			}
			affected++
		}
		return nil
	})

	return affected, err
}

// RemoveDirectoryListFromAllTargets removes a directory list reference from all targets in a group.
// Returns the count of removed references.
func (r *bulkRepository) RemoveDirectoryListFromAllTargets(ctx context.Context, groupID, directoryListID uint) (int, error) {
	var affected int

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var targets []Target
		if err := tx.Where("group_id = ?", groupID).Find(&targets).Error; err != nil {
			return fmt.Errorf("failed to list targets: %w", err)
		}

		for _, target := range targets {
			result := tx.Where("target_id = ? AND directory_list_id = ?", target.ID, directoryListID).
				Delete(&TargetDirectoryListRef{})
			if result.Error != nil {
				return fmt.Errorf("failed to remove directory list ref from target %d: %w", target.ID, result.Error)
			}
			affected += int(result.RowsAffected)
		}
		return nil
	})

	return affected, err
}
