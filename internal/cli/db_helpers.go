package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// openDatabase opens the database at the given path (or default) and returns it.
// The caller must close the returned database.
func openDatabase() (db.Database, error) {
	path := getDBPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("database does not exist: %s (run 'go-broadcast db init' to create)", path) //nolint:err113 // user-facing CLI error
	}
	database, err := db.Open(db.OpenOptions{
		Path:     path,
		LogLevel: logger.Silent,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return database, nil
}

// getDefaultConfig finds the first (and typically only) config in the database.
func getDefaultConfig(ctx context.Context, gormDB *gorm.DB) (*db.Config, error) {
	var cfg db.Config
	if err := gormDB.WithContext(ctx).First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("no configuration found (run 'go-broadcast db import' first)") //nolint:err113 // user-facing CLI error
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	return &cfg, nil
}

// resolveGroup looks up a group by external ID with a helpful hint on failure.
func resolveGroup(ctx context.Context, gormDB *gorm.DB, externalID string) (*db.Group, error) {
	groupRepo := db.NewGroupRepository(gormDB)
	group, err := groupRepo.GetByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, fmt.Errorf("group %q not found", externalID) //nolint:err113 // user-facing CLI error
		}
		return nil, fmt.Errorf("failed to resolve group: %w", err)
	}
	return group, nil
}

// resolveTarget looks up a target by group ID and full repo name with a helpful hint on failure.
func resolveTarget(ctx context.Context, gormDB *gorm.DB, groupID uint, repoFullName string) (*db.Target, error) {
	targetRepo := db.NewTargetRepository(gormDB)
	target, err := targetRepo.GetByRepoName(ctx, groupID, repoFullName)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, fmt.Errorf("target %q not found in group", repoFullName) //nolint:err113 // user-facing CLI error
		}
		return nil, fmt.Errorf("failed to resolve target: %w", err)
	}
	return target, nil
}

// resolveFileList looks up a file list by external ID with a helpful hint on failure.
func resolveFileList(ctx context.Context, gormDB *gorm.DB, externalID string) (*db.FileList, error) {
	flRepo := db.NewFileListRepository(gormDB)
	fl, err := flRepo.GetByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, fmt.Errorf("file list %q not found", externalID) //nolint:err113 // user-facing CLI error
		}
		return nil, fmt.Errorf("failed to resolve file list: %w", err)
	}
	return fl, nil
}

// resolveDirectoryList looks up a directory list by external ID with a helpful hint on failure.
func resolveDirectoryList(ctx context.Context, gormDB *gorm.DB, externalID string) (*db.DirectoryList, error) {
	dlRepo := db.NewDirectoryListRepository(gormDB)
	dl, err := dlRepo.GetByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, fmt.Errorf("directory list %q not found", externalID) //nolint:err113 // user-facing CLI error
		}
		return nil, fmt.Errorf("failed to resolve directory list: %w", err)
	}
	return dl, nil
}
