package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// directoryListRepository implements DirectoryListRepository
type directoryListRepository struct {
	db *gorm.DB
}

// NewDirectoryListRepository creates a new DirectoryListRepository
func NewDirectoryListRepository(db *gorm.DB) DirectoryListRepository {
	return &directoryListRepository{db: db}
}

// Create creates a new directory list
func (r *directoryListRepository) Create(ctx context.Context, directoryList *DirectoryList) error {
	if err := r.db.WithContext(ctx).Create(directoryList).Error; err != nil {
		return fmt.Errorf("failed to create directory list: %w", err)
	}
	return nil
}

// GetByID retrieves a directory list by database ID
func (r *directoryListRepository) GetByID(ctx context.Context, id uint) (*DirectoryList, error) {
	var directoryList DirectoryList
	if err := r.db.WithContext(ctx).First(&directoryList, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get directory list by id: %w", err)
	}
	return &directoryList, nil
}

// GetByExternalID retrieves a directory list by external_id
func (r *directoryListRepository) GetByExternalID(ctx context.Context, externalID string) (*DirectoryList, error) {
	var directoryList DirectoryList
	if err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&directoryList).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get directory list by external_id: %w", err)
	}
	return &directoryList, nil
}

// Update updates an existing directory list
func (r *directoryListRepository) Update(ctx context.Context, directoryList *DirectoryList) error {
	if err := r.db.WithContext(ctx).Save(directoryList).Error; err != nil {
		return fmt.Errorf("failed to update directory list: %w", err)
	}
	return nil
}

// Delete soft-deletes or hard-deletes a directory list
func (r *directoryListRepository) Delete(ctx context.Context, id uint, hard bool) error {
	if hard {
		if err := r.db.WithContext(ctx).Unscoped().Delete(&DirectoryList{}, id).Error; err != nil {
			return fmt.Errorf("failed to hard delete directory list: %w", err)
		}
	} else {
		if err := r.db.WithContext(ctx).Delete(&DirectoryList{}, id).Error; err != nil {
			return fmt.Errorf("failed to soft delete directory list: %w", err)
		}
	}
	return nil
}

// List retrieves all directory lists for a config
func (r *directoryListRepository) List(ctx context.Context, configID uint) ([]*DirectoryList, error) {
	var directoryLists []*DirectoryList
	if err := r.db.WithContext(ctx).
		Where("config_id = ?", configID).
		Order("position ASC").
		Find(&directoryLists).Error; err != nil {
		return nil, fmt.Errorf("failed to list directory lists: %w", err)
	}
	return directoryLists, nil
}

// ListWithDirectories retrieves all directory lists with preloaded directory mappings
func (r *directoryListRepository) ListWithDirectories(ctx context.Context, configID uint) ([]*DirectoryList, error) {
	var directoryLists []*DirectoryList
	if err := r.db.WithContext(ctx).
		Preload("Directories", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Directories.Transform").
		Where("config_id = ?", configID).
		Order("position ASC").
		Find(&directoryLists).Error; err != nil {
		return nil, fmt.Errorf("failed to list directory lists with directories: %w", err)
	}
	return directoryLists, nil
}
