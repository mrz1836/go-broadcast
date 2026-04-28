package db

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// settingsPresetRepository implements SettingsPresetRepository
type settingsPresetRepository struct {
	db *gorm.DB
}

// NewSettingsPresetRepository creates a new SettingsPresetRepository
func NewSettingsPresetRepository(db *gorm.DB) SettingsPresetRepository {
	return &settingsPresetRepository{db: db}
}

// Create creates a new settings preset
func (r *settingsPresetRepository) Create(ctx context.Context, preset *SettingsPreset) error {
	if err := r.db.WithContext(ctx).Create(preset).Error; err != nil {
		return fmt.Errorf("failed to create settings preset: %w", err)
	}
	return nil
}

// GetByID retrieves a preset by database ID with children
func (r *settingsPresetRepository) GetByID(ctx context.Context, id uint) (*SettingsPreset, error) {
	var preset SettingsPreset
	if err := r.db.WithContext(ctx).
		Preload("Labels").
		Preload("Rulesets").
		First(&preset, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get settings preset by id: %w", err)
	}
	return &preset, nil
}

// GetByExternalID retrieves a preset by external_id with children
func (r *settingsPresetRepository) GetByExternalID(ctx context.Context, externalID string) (*SettingsPreset, error) {
	var preset SettingsPreset
	if err := r.db.WithContext(ctx).
		Preload("Labels").
		Preload("Rulesets").
		Where("external_id = ?", externalID).
		First(&preset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get settings preset by external_id: %w", err)
	}
	return &preset, nil
}

// Update updates an existing preset
func (r *settingsPresetRepository) Update(ctx context.Context, preset *SettingsPreset) error {
	if err := r.db.WithContext(ctx).Save(preset).Error; err != nil {
		return fmt.Errorf("failed to update settings preset: %w", err)
	}
	return nil
}

// Delete soft-deletes or hard-deletes a preset
func (r *settingsPresetRepository) Delete(ctx context.Context, id uint, hard bool) error {
	if hard {
		// Hard delete children first, then preset
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Unscoped().Where("settings_preset_id = ?", id).Delete(&SettingsPresetLabel{}).Error; err != nil {
				return fmt.Errorf("failed to hard delete preset labels: %w", err)
			}
			if err := tx.Unscoped().Where("settings_preset_id = ?", id).Delete(&SettingsPresetRuleset{}).Error; err != nil {
				return fmt.Errorf("failed to hard delete preset rulesets: %w", err)
			}
			if err := tx.Unscoped().Delete(&SettingsPreset{}, id).Error; err != nil {
				return fmt.Errorf("failed to hard delete settings preset: %w", err)
			}
			return nil
		})
	}
	if err := r.db.WithContext(ctx).Delete(&SettingsPreset{}, id).Error; err != nil {
		return fmt.Errorf("failed to soft delete settings preset: %w", err)
	}
	return nil
}

// List retrieves all presets with children
func (r *settingsPresetRepository) List(ctx context.Context) ([]*SettingsPreset, error) {
	var presets []*SettingsPreset
	if err := r.db.WithContext(ctx).
		Preload("Labels").
		Preload("Rulesets").
		Order("external_id ASC").
		Find(&presets).Error; err != nil {
		return nil, fmt.Errorf("failed to list settings presets: %w", err)
	}
	return presets, nil
}

// ImportFromConfig upserts a preset by ExternalID, replacing children
func (r *settingsPresetRepository) ImportFromConfig(ctx context.Context, preset *SettingsPreset) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find existing by external_id
		var existing SettingsPreset
		err := tx.Where("external_id = ?", preset.ExternalID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to lookup preset: %w", err)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new
			if createErr := tx.Create(preset).Error; createErr != nil {
				return fmt.Errorf("failed to create preset: %w", createErr)
			}
			return nil
		}

		// Update existing fields
		existing.Name = preset.Name
		existing.Description = preset.Description
		existing.HasIssues = preset.HasIssues
		existing.HasWiki = preset.HasWiki
		existing.HasProjects = preset.HasProjects
		existing.HasDiscussions = preset.HasDiscussions
		existing.AllowSquashMerge = preset.AllowSquashMerge
		existing.AllowMergeCommit = preset.AllowMergeCommit
		existing.AllowRebaseMerge = preset.AllowRebaseMerge
		existing.DeleteBranchOnMerge = preset.DeleteBranchOnMerge
		existing.AllowAutoMerge = preset.AllowAutoMerge
		existing.AllowUpdateBranch = preset.AllowUpdateBranch
		existing.SquashMergeCommitTitle = preset.SquashMergeCommitTitle
		existing.SquashMergeCommitMessage = preset.SquashMergeCommitMessage

		if saveErr := tx.Save(&existing).Error; saveErr != nil {
			return fmt.Errorf("failed to update preset: %w", saveErr)
		}

		// Replace labels: delete old, insert new
		if delErr := tx.Where("settings_preset_id = ?", existing.ID).Delete(&SettingsPresetLabel{}).Error; delErr != nil {
			return fmt.Errorf("failed to delete old labels: %w", delErr)
		}
		for i := range preset.Labels {
			preset.Labels[i].SettingsPresetID = existing.ID
			preset.Labels[i].ID = 0
			if createErr := tx.Create(&preset.Labels[i]).Error; createErr != nil {
				return fmt.Errorf("failed to create label: %w", createErr)
			}
		}

		// Replace rulesets: delete old, insert new
		if delErr := tx.Where("settings_preset_id = ?", existing.ID).Delete(&SettingsPresetRuleset{}).Error; delErr != nil {
			return fmt.Errorf("failed to delete old rulesets: %w", delErr)
		}
		for i := range preset.Rulesets {
			preset.Rulesets[i].SettingsPresetID = existing.ID
			preset.Rulesets[i].ID = 0
			if createErr := tx.Create(&preset.Rulesets[i]).Error; createErr != nil {
				return fmt.Errorf("failed to create ruleset: %w", createErr)
			}
		}

		// Update the preset ID for caller
		preset.ID = existing.ID
		return nil
	})
}

// AssignPresetToRepo sets Repo.SettingsPresetID
func (r *settingsPresetRepository) AssignPresetToRepo(ctx context.Context, repoID, presetID uint) error {
	if err := r.db.WithContext(ctx).
		Session(&gorm.Session{SkipHooks: true}).
		Model(&Repo{}).Where("id = ?", repoID).
		Update("settings_preset_id", presetID).Error; err != nil {
		return fmt.Errorf("failed to assign preset to repo: %w", err)
	}
	return nil
}

// SeedIfMissing creates any preset whose ExternalID is not already present in the DB.
// Returns the count of newly-seeded presets. Idempotent.
func (r *settingsPresetRepository) SeedIfMissing(ctx context.Context, presets []*SettingsPreset) (int, error) {
	seeded := 0
	for _, preset := range presets {
		if preset == nil {
			continue
		}
		_, err := r.GetByExternalID(ctx, preset.ExternalID)
		if err == nil {
			continue // already exists, skip
		}
		if !errors.Is(err, ErrRecordNotFound) {
			return seeded, fmt.Errorf("failed to check preset %q: %w", preset.ExternalID, err)
		}
		// Reset IDs so GORM creates a fresh row (callers may pass reusable templates)
		preset.ID = 0
		for i := range preset.Labels {
			preset.Labels[i].ID = 0
			preset.Labels[i].SettingsPresetID = 0
		}
		for i := range preset.Rulesets {
			preset.Rulesets[i].ID = 0
			preset.Rulesets[i].SettingsPresetID = 0
		}
		if err := r.Create(ctx, preset); err != nil {
			return seeded, fmt.Errorf("failed to seed preset %q: %w", preset.ExternalID, err)
		}
		seeded++
	}
	return seeded, nil
}

// GetPresetForRepo returns the assigned preset or nil
func (r *settingsPresetRepository) GetPresetForRepo(ctx context.Context, repoID uint) (*SettingsPreset, error) {
	var repo Repo
	if err := r.db.WithContext(ctx).First(&repo, repoID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf("failed to get repo: %w", err)
	}

	if repo.SettingsPresetID == nil {
		return nil, nil //nolint:nilnil // nil preset means no preset assigned
	}

	return r.GetByID(ctx, *repo.SettingsPresetID)
}
