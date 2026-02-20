package db

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/validation"
)

// BeforeCreate validates Client model before database insertion
func (c *Client) BeforeCreate(_ *gorm.DB) error {
	if err := validation.ValidateNonEmpty("name", c.Name); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates Client model before database update
func (c *Client) BeforeUpdate(_ *gorm.DB) error {
	return c.BeforeCreate(nil)
}

// BeforeCreate validates Organization model before database insertion
func (o *Organization) BeforeCreate(_ *gorm.DB) error {
	if o.ClientID == 0 {
		return fmt.Errorf("%w: client_id is required", ErrValidationFailed)
	}
	if err := validation.ValidateNonEmpty("name", o.Name); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates Organization model before database update
func (o *Organization) BeforeUpdate(_ *gorm.DB) error {
	return o.BeforeCreate(nil)
}

// BeforeCreate validates Repo model before database insertion
func (r *Repo) BeforeCreate(tx *gorm.DB) error {
	if r.OrganizationID == 0 {
		return fmt.Errorf("%w: organization_id is required", ErrValidationFailed)
	}
	if err := validation.ValidateRepoShortName(r.Name); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	// Auto-populate FullNameStr if not already set
	if r.FullNameStr == "" && tx != nil {
		var org Organization
		if err := tx.Select("name").First(&org, r.OrganizationID).Error; err == nil && org.Name != "" {
			r.FullNameStr = org.Name + "/" + r.Name
		}
	}
	return nil
}

// BeforeUpdate validates Repo model before database update
func (r *Repo) BeforeUpdate(tx *gorm.DB) error {
	return r.BeforeCreate(tx)
}

// BeforeCreate validates Source model before database insertion
func (s *Source) BeforeCreate(_ *gorm.DB) error {
	if s.RepoID == 0 {
		return fmt.Errorf("%w: repo_id is required", ErrValidationFailed)
	}
	if err := validation.ValidateBranchName(s.Branch); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	if err := validation.ValidateEmail(s.SecurityEmail, "security_email"); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	if err := validation.ValidateEmail(s.SupportEmail, "support_email"); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates Source model before database update
func (s *Source) BeforeUpdate(_ *gorm.DB) error {
	return s.BeforeCreate(nil)
}

// BeforeCreate validates Target model before database insertion
func (t *Target) BeforeCreate(_ *gorm.DB) error {
	if t.RepoID == 0 {
		return fmt.Errorf("%w: repo_id is required", ErrValidationFailed)
	}
	if t.Branch != "" {
		if err := validation.ValidateBranchName(t.Branch); err != nil {
			return fmt.Errorf("%w: %w", ErrValidationFailed, err)
		}
	}
	if err := validation.ValidateEmail(t.SecurityEmail, "security_email"); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	if err := validation.ValidateEmail(t.SupportEmail, "support_email"); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates Target model before database update
func (t *Target) BeforeUpdate(_ *gorm.DB) error {
	return t.BeforeCreate(nil)
}

// BeforeCreate validates Group model before database insertion
func (g *Group) BeforeCreate(_ *gorm.DB) error {
	if err := validation.ValidateNonEmpty("name", g.Name); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	if err := validation.ValidateNonEmpty("external_id", g.ExternalID); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates Group model before database update
func (g *Group) BeforeUpdate(_ *gorm.DB) error {
	return g.BeforeCreate(nil)
}

// BeforeCreate validates FileMapping model before database insertion
func (f *FileMapping) BeforeCreate(_ *gorm.DB) error {
	// For deletions, source path can be empty
	if !f.DeleteFlag {
		if err := validation.ValidateFilePath(f.Src, "source"); err != nil {
			return fmt.Errorf("%w: %w", ErrValidationFailed, err)
		}
	}
	if err := validation.ValidateFilePath(f.Dest, "destination"); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates FileMapping model before database update
func (f *FileMapping) BeforeUpdate(_ *gorm.DB) error {
	return f.BeforeCreate(nil)
}

// BeforeCreate validates DirectoryMapping model before database insertion
func (d *DirectoryMapping) BeforeCreate(_ *gorm.DB) error {
	// For deletions, source path can be empty
	if !d.DeleteFlag {
		if err := validation.ValidateFilePath(d.Src, "source"); err != nil {
			return fmt.Errorf("%w: %w", ErrValidationFailed, err)
		}
	}
	if err := validation.ValidateFilePath(d.Dest, "destination"); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates DirectoryMapping model before database update
func (d *DirectoryMapping) BeforeUpdate(_ *gorm.DB) error {
	return d.BeforeCreate(nil)
}

// BeforeCreate validates GroupDefault model before database insertion
func (g *GroupDefault) BeforeCreate(_ *gorm.DB) error {
	if err := validation.ValidateBranchPrefix(g.BranchPrefix); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates GroupDefault model before database update
func (g *GroupDefault) BeforeUpdate(_ *gorm.DB) error {
	return g.BeforeCreate(nil)
}

// BeforeCreate validates FileList model before database insertion
func (f *FileList) BeforeCreate(_ *gorm.DB) error {
	if err := validation.ValidateNonEmpty("name", f.Name); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	if err := validation.ValidateNonEmpty("external_id", f.ExternalID); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates FileList model before database update
func (f *FileList) BeforeUpdate(_ *gorm.DB) error {
	return f.BeforeCreate(nil)
}

// BeforeCreate validates DirectoryList model before database insertion
func (d *DirectoryList) BeforeCreate(_ *gorm.DB) error {
	if err := validation.ValidateNonEmpty("name", d.Name); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	if err := validation.ValidateNonEmpty("external_id", d.ExternalID); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}
	return nil
}

// BeforeUpdate validates DirectoryList model before database update
func (d *DirectoryList) BeforeUpdate(_ *gorm.DB) error {
	return d.BeforeCreate(nil)
}
