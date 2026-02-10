package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelsHooks_ValidationErrors tests validation failures in hooks
func TestModelsHooks_ValidationErrors(t *testing.T) {
	db := TestDB(t)

	t.Run("Group BeforeCreate missing external_id", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID: config.ID,
			// Missing ExternalID
			Name: "Test Group",
		}
		err = db.Create(group).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "external_id")
	})

	t.Run("FileList BeforeCreate missing external_id", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-2",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		fileList := &FileList{
			ConfigID: config.ID,
			// Missing ExternalID
			Name: "Test File List",
		}
		err = db.Create(fileList).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "external_id")
	})

	t.Run("DirectoryList BeforeCreate missing external_id", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-3",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		dirList := &DirectoryList{
			ConfigID: config.ID,
			// Missing ExternalID
			Name: "Test Dir List",
		}
		err = db.Create(dirList).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "external_id")
	})

	t.Run("Target BeforeCreate invalid repo", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-4",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group",
			Name:       "Test Group",
		}
		err = db.Create(group).Error
		require.NoError(t, err)

		target := &Target{
			GroupID: group.ID,
			Repo:    "invalid repo format", // Invalid repo name
		}
		err = db.Create(target).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo")
	})
}

// TestModelsHooks_BeforeUpdate tests update validation
func TestModelsHooks_BeforeUpdate(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	t.Run("Target BeforeUpdate validation", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group",
			Name:       "Test Group",
		}
		err = db.Create(group).Error
		require.NoError(t, err)

		target := &Target{
			GroupID: group.ID,
			Repo:    "mrz1836/test",
		}
		err = db.Create(target).Error
		require.NoError(t, err)

		// Try to update to invalid state
		target.Repo = "" // Invalid: empty repo
		err = db.WithContext(ctx).Save(target).Error
		assert.Error(t, err)
	})

	t.Run("FileMapping BeforeUpdate validation", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-fm-update",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group-fm-update",
			Name:       "Test Group",
		}
		err = db.Create(group).Error
		require.NoError(t, err)

		target := &Target{
			GroupID: group.ID,
			Repo:    "mrz1836/test",
		}
		err = db.Create(target).Error
		require.NoError(t, err)

		mapping := &FileMapping{
			OwnerType: "target",
			OwnerID:   target.ID,
			Src:       "src/file.txt",
			Dest:      "dest/file.txt",
		}
		err = db.Create(mapping).Error
		require.NoError(t, err)

		// Update should still validate
		mapping.DeleteFlag = true // Setting delete flag with src is valid
		err = db.WithContext(ctx).Save(mapping).Error
		require.NoError(t, err)
	})
}

// TestModelsHooks_DirectoryMappingBeforeUpdate tests DirectoryMapping BeforeUpdate
func TestModelsHooks_DirectoryMappingBeforeUpdate(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Setup
	cfg := &Config{
		ExternalID: "dirmap-update-test",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(cfg).Error
	require.NoError(t, err)

	group := &Group{
		ConfigID:   cfg.ID,
		ExternalID: "group-dirmap-update",
		Name:       "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	target := &Target{
		GroupID: group.ID,
		Repo:    "mrz1836/test",
	}
	err = db.Create(target).Error
	require.NoError(t, err)

	dirMapping := &DirectoryMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "valid/src",
		Dest:      "valid/dest",
	}
	err = db.Create(dirMapping).Error
	require.NoError(t, err)

	// Test update with valid data
	dirMapping.Src = "updated/src"
	err = db.WithContext(ctx).Save(dirMapping).Error
	require.NoError(t, err)

	// Test update with invalid data (empty dest for non-delete)
	dirMapping.Dest = ""
	err = db.WithContext(ctx).Save(dirMapping).Error
	assert.Error(t, err)
}

// TestModelsHooks_GroupDefaultBeforeCreate tests GroupDefault validation
func TestModelsHooks_GroupDefaultBeforeCreate(t *testing.T) {
	db := TestDB(t)

	cfg := &Config{
		ExternalID: "group-default-test",
		Name:       "Test",
		Version:    1,
	}
	err := db.Create(cfg).Error
	require.NoError(t, err)

	group := &Group{
		ConfigID:   cfg.ID,
		ExternalID: "group-gd-test",
		Name:       "Test Group",
	}
	err = db.Create(group).Error
	require.NoError(t, err)

	t.Run("Valid branch prefix", func(t *testing.T) {
		groupDefault := &GroupDefault{
			GroupID:      group.ID,
			BranchPrefix: "feature",
		}
		err := db.Create(groupDefault).Error
		require.NoError(t, err)
	})

	t.Run("Empty branch prefix", func(t *testing.T) {
		// Need a new group for this test
		group2 := &Group{
			ConfigID:   cfg.ID,
			ExternalID: "group-gd-test-2",
			Name:       "Test Group 2",
		}
		err := db.Create(group2).Error
		require.NoError(t, err)

		groupDefault := &GroupDefault{
			GroupID:      group2.ID,
			BranchPrefix: "", // Empty is valid
		}
		err = db.Create(groupDefault).Error
		require.NoError(t, err)
	})
}
