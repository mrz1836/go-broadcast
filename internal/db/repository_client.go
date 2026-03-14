package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type clientRepository struct {
	db *gorm.DB
}

// NewClientRepository creates a new ClientRepository
func NewClientRepository(db *gorm.DB) ClientRepository {
	return &clientRepository{db: db}
}

func (r *clientRepository) Create(ctx context.Context, client *Client) error {
	return r.db.WithContext(ctx).Create(client).Error
}

func (r *clientRepository) GetByID(ctx context.Context, id uint) (*Client, error) {
	var client Client
	if err := r.db.WithContext(ctx).First(&client, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: client id=%d", ErrRecordNotFound, id)
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) GetByName(ctx context.Context, name string) (*Client, error) {
	var client Client
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: client name=%q", ErrRecordNotFound, name)
		}
		return nil, err
	}
	return &client, nil
}

func (r *clientRepository) Update(ctx context.Context, client *Client) error {
	return r.db.WithContext(ctx).Save(client).Error
}

func (r *clientRepository) Delete(ctx context.Context, id uint, hard bool) error {
	db := r.db.WithContext(ctx)
	if hard {
		db = db.Unscoped()
	}
	result := db.Delete(&Client{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: client id=%d", ErrRecordNotFound, id)
	}
	return nil
}

func (r *clientRepository) List(ctx context.Context) ([]*Client, error) {
	var clients []*Client
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

func (r *clientRepository) ListWithOrganizations(ctx context.Context) ([]*Client, error) {
	var clients []*Client
	if err := r.db.WithContext(ctx).
		Preload("Organizations", func(db *gorm.DB) *gorm.DB {
			return db.Order("organizations.name ASC")
		}).
		Order("name ASC").
		Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}
