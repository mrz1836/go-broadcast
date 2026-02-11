package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type repoRepository struct {
	db *gorm.DB
}

// NewRepoRepository creates a new RepoRepository
func NewRepoRepository(db *gorm.DB) RepoRepository {
	return &repoRepository{db: db}
}

func (r *repoRepository) Create(ctx context.Context, repo *Repo) error {
	return r.db.WithContext(ctx).Create(repo).Error
}

func (r *repoRepository) GetByID(ctx context.Context, id uint) (*Repo, error) {
	var repo Repo
	if err := r.db.WithContext(ctx).Preload("Organization").First(&repo, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: repo id=%d", ErrRecordNotFound, id)
		}
		return nil, err
	}
	return &repo, nil
}

func (r *repoRepository) GetByFullName(ctx context.Context, orgName, repoName string) (*Repo, error) {
	var repo Repo
	if err := r.db.WithContext(ctx).
		Joins("JOIN organizations ON organizations.id = repos.organization_id AND organizations.deleted_at IS NULL").
		Where("organizations.name = ? AND repos.name = ?", orgName, repoName).
		Preload("Organization").
		First(&repo).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: repo %s/%s", ErrRecordNotFound, orgName, repoName)
		}
		return nil, err
	}
	return &repo, nil
}

func (r *repoRepository) Update(ctx context.Context, repo *Repo) error {
	return r.db.WithContext(ctx).Save(repo).Error
}

func (r *repoRepository) Delete(ctx context.Context, id uint, hard bool) error {
	db := r.db.WithContext(ctx)
	if hard {
		db = db.Unscoped()
	}
	result := db.Delete(&Repo{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: repo id=%d", ErrRecordNotFound, id)
	}
	return nil
}

func (r *repoRepository) List(ctx context.Context, organizationID uint) ([]*Repo, error) {
	var repos []*Repo
	if err := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("name ASC").
		Find(&repos).Error; err != nil {
		return nil, err
	}
	return repos, nil
}

func (r *repoRepository) FindOrCreateFromFullName(ctx context.Context, fullName string, defaultClientID uint) (*Repo, error) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("%w: expected org/repo, got %q", ErrInvalidRepoFormat, fullName)
	}
	orgName := parts[0]
	repoName := parts[1]

	// Try to find existing repo
	var repo Repo
	err := r.db.WithContext(ctx).
		Joins("JOIN organizations ON organizations.id = repos.organization_id AND organizations.deleted_at IS NULL").
		Where("organizations.name = ? AND repos.name = ?", orgName, repoName).
		Preload("Organization").
		First(&repo).Error
	if err == nil {
		return &repo, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Find or create organization
	var org Organization
	err = r.db.WithContext(ctx).Where("name = ?", orgName).First(&org).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Auto-create client with same name as org if needed
		var client Client
		cErr := r.db.WithContext(ctx).Where("id = ?", defaultClientID).First(&client).Error
		if cErr != nil {
			// Create a new client named after the org
			client = Client{Name: orgName}
			if cErr = r.db.WithContext(ctx).Create(&client).Error; cErr != nil {
				return nil, fmt.Errorf("failed to create client for org %q: %w", orgName, cErr)
			}
		}
		org = Organization{
			ClientID: client.ID,
			Name:     orgName,
		}
		if err = r.db.WithContext(ctx).Create(&org).Error; err != nil {
			return nil, fmt.Errorf("failed to create organization %q: %w", orgName, err)
		}
	} else if err != nil {
		return nil, err
	}

	// Create repo
	repo = Repo{
		OrganizationID: org.ID,
		Name:           repoName,
	}
	if err = r.db.WithContext(ctx).Create(&repo).Error; err != nil {
		return nil, fmt.Errorf("failed to create repo %q: %w", fullName, err)
	}
	repo.Organization = org
	return &repo, nil
}
