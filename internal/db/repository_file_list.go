package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// fileListRepository implements FileListRepository
type fileListRepository struct {
	db *gorm.DB
}

// NewFileListRepository creates a new FileListRepository
func NewFileListRepository(db *gorm.DB) FileListRepository {
	return &fileListRepository{db: db}
}

// Create creates a new file list
func (r *fileListRepository) Create(ctx context.Context, fileList *FileList) error {
	if err := r.db.WithContext(ctx).Create(fileList).Error; err != nil {
		return fmt.Errorf("failed to create file list: %w", err)
	}
	return nil
}

// GetByID retrieves a file list by database ID
func (r *fileListRepository) GetByID(ctx context.Context, id uint) (*FileList, error) {
	var fileList FileList
	if err := r.db.WithContext(ctx).First(&fileList, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get file list by id: %w", err)
	}
	return &fileList, nil
}

// GetByExternalID retrieves a file list by external_id
func (r *fileListRepository) GetByExternalID(ctx context.Context, externalID string) (*FileList, error) {
	var fileList FileList
	if err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&fileList).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get file list by external_id: %w", err)
	}
	return &fileList, nil
}

// Update updates an existing file list
func (r *fileListRepository) Update(ctx context.Context, fileList *FileList) error {
	if err := r.db.WithContext(ctx).Save(fileList).Error; err != nil {
		return fmt.Errorf("failed to update file list: %w", err)
	}
	return nil
}

// Delete soft-deletes or hard-deletes a file list
func (r *fileListRepository) Delete(ctx context.Context, id uint, hard bool) error {
	if hard {
		if err := r.db.WithContext(ctx).Unscoped().Delete(&FileList{}, id).Error; err != nil {
			return fmt.Errorf("failed to hard delete file list: %w", err)
		}
	} else {
		if err := r.db.WithContext(ctx).Delete(&FileList{}, id).Error; err != nil {
			return fmt.Errorf("failed to soft delete file list: %w", err)
		}
	}
	return nil
}

// List retrieves all file lists for a config
func (r *fileListRepository) List(ctx context.Context, configID uint) ([]*FileList, error) {
	var fileLists []*FileList
	if err := r.db.WithContext(ctx).
		Where("config_id = ?", configID).
		Order("position ASC").
		Find(&fileLists).Error; err != nil {
		return nil, fmt.Errorf("failed to list file lists: %w", err)
	}
	return fileLists, nil
}

// ListWithFiles retrieves all file lists with preloaded file mappings
func (r *fileListRepository) ListWithFiles(ctx context.Context, configID uint) ([]*FileList, error) {
	var fileLists []*FileList
	if err := r.db.WithContext(ctx).
		Preload("Files", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Where("config_id = ?", configID).
		Order("position ASC").
		Find(&fileLists).Error; err != nil {
		return nil, fmt.Errorf("failed to list file lists with files: %w", err)
	}
	return fileLists, nil
}
