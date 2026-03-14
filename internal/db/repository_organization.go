package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type organizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new OrganizationRepository
func NewOrganizationRepository(db *gorm.DB) OrganizationRepository {
	return &organizationRepository{db: db}
}

func (r *organizationRepository) Create(ctx context.Context, org *Organization) error {
	return r.db.WithContext(ctx).Create(org).Error
}

func (r *organizationRepository) GetByID(ctx context.Context, id uint) (*Organization, error) {
	var org Organization
	if err := r.db.WithContext(ctx).First(&org, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: organization id=%d", ErrRecordNotFound, id)
		}
		return nil, err
	}
	return &org, nil
}

func (r *organizationRepository) GetByName(ctx context.Context, name string) (*Organization, error) {
	var org Organization
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: organization name=%q", ErrRecordNotFound, name)
		}
		return nil, err
	}
	return &org, nil
}

func (r *organizationRepository) Update(ctx context.Context, org *Organization) error {
	return r.db.WithContext(ctx).Save(org).Error
}

func (r *organizationRepository) Delete(ctx context.Context, id uint, hard bool) error {
	db := r.db.WithContext(ctx)
	if hard {
		db = db.Unscoped()
	}
	result := db.Delete(&Organization{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: organization id=%d", ErrRecordNotFound, id)
	}
	return nil
}

func (r *organizationRepository) List(ctx context.Context, clientID uint) ([]*Organization, error) {
	var orgs []*Organization
	if err := r.db.WithContext(ctx).
		Where("client_id = ?", clientID).
		Order("name ASC").
		Find(&orgs).Error; err != nil {
		return nil, err
	}
	return orgs, nil
}

func (r *organizationRepository) ListWithRepos(ctx context.Context, clientID uint) ([]*Organization, error) {
	var orgs []*Organization
	if err := r.db.WithContext(ctx).
		Where("client_id = ?", clientID).
		Preload("Repos", func(db *gorm.DB) *gorm.DB {
			return db.Order("repos.name ASC")
		}).
		Order("name ASC").
		Find(&orgs).Error; err != nil {
		return nil, err
	}
	return orgs, nil
}

func (r *organizationRepository) FindOrCreate(ctx context.Context, name string, clientID uint) (*Organization, error) {
	var org Organization
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&org).Error
	if err == nil {
		return &org, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	org = Organization{
		ClientID: clientID,
		Name:     name,
	}
	if err := r.db.WithContext(ctx).Create(&org).Error; err != nil {
		return nil, err
	}
	return &org, nil
}
