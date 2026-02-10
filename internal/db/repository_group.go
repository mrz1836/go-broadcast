package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// groupRepository implements GroupRepository
type groupRepository struct {
	db *gorm.DB
}

// NewGroupRepository creates a new GroupRepository
func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

// Create creates a new group
func (r *groupRepository) Create(ctx context.Context, group *Group) error {
	if err := r.db.WithContext(ctx).Create(group).Error; err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	return nil
}

// GetByID retrieves a group by database ID
func (r *groupRepository) GetByID(ctx context.Context, id uint) (*Group, error) {
	var group Group
	if err := r.db.WithContext(ctx).First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get group by id: %w", err)
	}
	return &group, nil
}

// GetByExternalID retrieves a group by external_id
func (r *groupRepository) GetByExternalID(ctx context.Context, externalID string) (*Group, error) {
	var group Group
	if err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get group by external_id: %w", err)
	}
	return &group, nil
}

// Update updates an existing group
func (r *groupRepository) Update(ctx context.Context, group *Group) error {
	if err := r.db.WithContext(ctx).Save(group).Error; err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}
	return nil
}

// Delete soft-deletes or hard-deletes a group
func (r *groupRepository) Delete(ctx context.Context, id uint, hard bool) error {
	if hard {
		if err := r.db.WithContext(ctx).Unscoped().Delete(&Group{}, id).Error; err != nil {
			return fmt.Errorf("failed to hard delete group: %w", err)
		}
	} else {
		if err := r.db.WithContext(ctx).Delete(&Group{}, id).Error; err != nil {
			return fmt.Errorf("failed to soft delete group: %w", err)
		}
	}
	return nil
}

// List retrieves all groups for a config
func (r *groupRepository) List(ctx context.Context, configID uint) ([]*Group, error) {
	var groups []*Group
	if err := r.db.WithContext(ctx).
		Where("config_id = ?", configID).
		Order("position ASC").
		Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	return groups, nil
}

// ListWithAssociations retrieves all groups with full preloading
// Preloads: Source, GroupGlobal, GroupDefault, Targets, GroupDependencies
func (r *groupRepository) ListWithAssociations(ctx context.Context, configID uint) ([]*Group, error) {
	var groups []*Group
	if err := r.db.WithContext(ctx).
		Preload("Source").
		Preload("GroupGlobal").
		Preload("GroupDefault").
		Preload("Targets", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Dependencies", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Where("config_id = ?", configID).
		Order("position ASC").
		Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to list groups with associations: %w", err)
	}
	return groups, nil
}
