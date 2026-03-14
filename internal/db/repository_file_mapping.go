package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// FileMappingRepository manages FileMapping CRUD operations
type FileMappingRepository interface {
	Create(ctx context.Context, mapping *FileMapping) error
	Delete(ctx context.Context, id uint, hard bool) error
	ListByOwner(ctx context.Context, ownerType string, ownerID uint) ([]*FileMapping, error)
	FindByDest(ctx context.Context, ownerType string, ownerID uint, dest string) (*FileMapping, error)
}

type fileMappingRepository struct {
	db *gorm.DB
}

// NewFileMappingRepository creates a new FileMappingRepository
func NewFileMappingRepository(db *gorm.DB) FileMappingRepository {
	return &fileMappingRepository{db: db}
}

// Create creates a new file mapping
func (r *fileMappingRepository) Create(ctx context.Context, mapping *FileMapping) error {
	if err := r.db.WithContext(ctx).Create(mapping).Error; err != nil {
		return fmt.Errorf("failed to create file mapping: %w", err)
	}
	return nil
}

// Delete soft-deletes or hard-deletes a file mapping
func (r *fileMappingRepository) Delete(ctx context.Context, id uint, hard bool) error {
	if hard {
		if err := r.db.WithContext(ctx).Unscoped().Delete(&FileMapping{}, id).Error; err != nil {
			return fmt.Errorf("failed to hard delete file mapping: %w", err)
		}
	} else {
		if err := r.db.WithContext(ctx).Delete(&FileMapping{}, id).Error; err != nil {
			return fmt.Errorf("failed to soft delete file mapping: %w", err)
		}
	}
	return nil
}

// ListByOwner retrieves all file mappings for a given owner
func (r *fileMappingRepository) ListByOwner(ctx context.Context, ownerType string, ownerID uint) ([]*FileMapping, error) {
	var mappings []*FileMapping
	if err := r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Order("position ASC").
		Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to list file mappings: %w", err)
	}
	return mappings, nil
}

// FindByDest finds a file mapping by destination path within an owner
func (r *fileMappingRepository) FindByDest(ctx context.Context, ownerType string, ownerID uint, dest string) (*FileMapping, error) {
	var mapping FileMapping
	if err := r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ? AND dest = ?", ownerType, ownerID, dest).
		First(&mapping).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to find file mapping by dest: %w", err)
	}
	return &mapping, nil
}
