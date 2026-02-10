package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// configRepository implements ConfigRepository
type configRepository struct {
	db *gorm.DB
}

// NewConfigRepository creates a new ConfigRepository
func NewConfigRepository(db *gorm.DB) ConfigRepository {
	return &configRepository{db: db}
}

// Create creates a new config
func (r *configRepository) Create(ctx context.Context, config *Config) error {
	if err := r.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	return nil
}

// GetByID retrieves a config by database ID
func (r *configRepository) GetByID(ctx context.Context, id uint) (*Config, error) {
	var config Config
	if err := r.db.WithContext(ctx).First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get config by id: %w", err)
	}
	return &config, nil
}

// GetByExternalID retrieves a config by external_id
func (r *configRepository) GetByExternalID(ctx context.Context, externalID string) (*Config, error) {
	var config Config
	if err := r.db.WithContext(ctx).Where("external_id = ?", externalID).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get config by external_id: %w", err)
	}
	return &config, nil
}

// Update updates an existing config
func (r *configRepository) Update(ctx context.Context, config *Config) error {
	if err := r.db.WithContext(ctx).Save(config).Error; err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	return nil
}

// Delete soft-deletes a config
func (r *configRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&Config{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}
	return nil
}

// List retrieves all configs
func (r *configRepository) List(ctx context.Context) ([]*Config, error) {
	var configs []*Config
	if err := r.db.WithContext(ctx).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}
	return configs, nil
}
