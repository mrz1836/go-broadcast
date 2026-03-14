package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// DirectoryMappingRepository manages DirectoryMapping CRUD operations
type DirectoryMappingRepository interface {
	Create(ctx context.Context, mapping *DirectoryMapping) error
	Delete(ctx context.Context, id uint, hard bool) error
	ListByOwner(ctx context.Context, ownerType string, ownerID uint) ([]*DirectoryMapping, error)
	FindByDest(ctx context.Context, ownerType string, ownerID uint, dest string) (*DirectoryMapping, error)
}

type directoryMappingRepository struct {
	db *gorm.DB
}

// NewDirectoryMappingRepository creates a new DirectoryMappingRepository
func NewDirectoryMappingRepository(db *gorm.DB) DirectoryMappingRepository {
	return &directoryMappingRepository{db: db}
}

// Create creates a new directory mapping
func (r *directoryMappingRepository) Create(ctx context.Context, mapping *DirectoryMapping) error {
	if err := r.db.WithContext(ctx).Create(mapping).Error; err != nil {
		return fmt.Errorf("failed to create directory mapping: %w", err)
	}
	return nil
}

// Delete soft-deletes or hard-deletes a directory mapping
func (r *directoryMappingRepository) Delete(ctx context.Context, id uint, hard bool) error {
	if hard {
		if err := r.db.WithContext(ctx).Unscoped().Delete(&DirectoryMapping{}, id).Error; err != nil {
			return fmt.Errorf("failed to hard delete directory mapping: %w", err)
		}
	} else {
		if err := r.db.WithContext(ctx).Delete(&DirectoryMapping{}, id).Error; err != nil {
			return fmt.Errorf("failed to soft delete directory mapping: %w", err)
		}
	}
	return nil
}

// ListByOwner retrieves all directory mappings for a given owner
func (r *directoryMappingRepository) ListByOwner(ctx context.Context, ownerType string, ownerID uint) ([]*DirectoryMapping, error) {
	var mappings []*DirectoryMapping
	if err := r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Order("position ASC").
		Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to list directory mappings: %w", err)
	}
	return mappings, nil
}

// FindByDest finds a directory mapping by destination path within an owner
func (r *directoryMappingRepository) FindByDest(ctx context.Context, ownerType string, ownerID uint, dest string) (*DirectoryMapping, error) {
	var mapping DirectoryMapping
	if err := r.db.WithContext(ctx).
		Where("owner_type = ? AND owner_id = ? AND dest = ?", ownerType, ownerID, dest).
		First(&mapping).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to find directory mapping by dest: %w", err)
	}
	return &mapping, nil
}
