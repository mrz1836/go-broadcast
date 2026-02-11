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

	// Pre-create Client -> Organization -> Repo for FK-based tests
	client := &Client{Name: "Hook Test Client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "hook-test-org"}
	require.NoError(t, db.Create(org).Error)

	repo := &Repo{OrganizationID: org.ID, Name: "hook-test-repo"}
	require.NoError(t, db.Create(repo).Error)

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

	t.Run("Target BeforeCreate missing repo_id", func(t *testing.T) {
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
			RepoID:  0, // Missing repo_id
		}
		err = db.Create(target).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo_id")
	})

	t.Run("Source BeforeCreate missing repo_id", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-src",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group-src",
			Name:       "Test Group",
		}
		err = db.Create(group).Error
		require.NoError(t, err)

		source := &Source{
			GroupID: group.ID,
			RepoID:  0, // Missing repo_id
			Branch:  "main",
		}
		err = db.Create(source).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo_id")
	})

	t.Run("Client BeforeCreate missing name", func(t *testing.T) {
		c := &Client{
			Name: "", // Empty name
		}
		err := db.Create(c).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("Organization BeforeCreate missing client_id", func(t *testing.T) {
		o := &Organization{
			ClientID: 0, // Missing client_id
			Name:     "some-org",
		}
		err := db.Create(o).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client_id")
	})

	t.Run("Organization BeforeCreate missing name", func(t *testing.T) {
		o := &Organization{
			ClientID: client.ID,
			Name:     "", // Empty name
		}
		err := db.Create(o).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("Repo BeforeCreate missing organization_id", func(t *testing.T) {
		r := &Repo{
			OrganizationID: 0, // Missing organization_id
			Name:           "some-repo",
		}
		err := db.Create(r).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organization_id")
	})

	t.Run("Repo BeforeCreate empty name", func(t *testing.T) {
		r := &Repo{
			OrganizationID: org.ID,
			Name:           "", // Empty name
		}
		err := db.Create(r).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository name")
	})

	t.Run("Repo BeforeCreate name with slash", func(t *testing.T) {
		r := &Repo{
			OrganizationID: org.ID,
			Name:           "org/repo", // Slash not allowed in short name
		}
		err := db.Create(r).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
	})
}

// TestModelsHooks_BeforeUpdate tests update validation
func TestModelsHooks_BeforeUpdate(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Pre-create Client -> Organization -> Repo for FK-based tests
	client := &Client{Name: "Update Hook Client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "update-hook-org"}
	require.NoError(t, db.Create(org).Error)

	repo := &Repo{OrganizationID: org.ID, Name: "update-hook-repo"}
	require.NoError(t, db.Create(repo).Error)

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
			RepoID:  repo.ID,
		}
		err = db.Create(target).Error
		require.NoError(t, err)

		// Try to update to invalid state
		target.RepoID = 0 // Invalid: missing repo_id
		err = db.WithContext(ctx).Save(target).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo_id")
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
			RepoID:  repo.ID,
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

	t.Run("Client BeforeUpdate validation", func(t *testing.T) {
		c := &Client{Name: "client-update-test"}
		err := db.Create(c).Error
		require.NoError(t, err)

		c.Name = "" // Invalid: empty name
		err = db.WithContext(ctx).Save(c).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("Organization BeforeUpdate validation", func(t *testing.T) {
		o := &Organization{ClientID: client.ID, Name: "org-update-test"}
		err := db.Create(o).Error
		require.NoError(t, err)

		o.ClientID = 0 // Invalid: missing client_id
		err = db.WithContext(ctx).Save(o).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "client_id")
	})

	t.Run("Repo BeforeUpdate validation", func(t *testing.T) {
		r := &Repo{OrganizationID: org.ID, Name: "repo-update-test"}
		err := db.Create(r).Error
		require.NoError(t, err)

		r.OrganizationID = 0 // Invalid: missing organization_id
		err = db.WithContext(ctx).Save(r).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organization_id")
	})

	t.Run("Source BeforeUpdate validation", func(t *testing.T) {
		config := &Config{
			ExternalID: "test-config-src-update",
			Name:       "Test",
			Version:    1,
		}
		err := db.Create(config).Error
		require.NoError(t, err)

		group := &Group{
			ConfigID:   config.ID,
			ExternalID: "test-group-src-update",
			Name:       "Test Group",
		}
		err = db.Create(group).Error
		require.NoError(t, err)

		source := &Source{
			GroupID: group.ID,
			RepoID:  repo.ID,
			Branch:  "main",
		}
		err = db.Create(source).Error
		require.NoError(t, err)

		source.RepoID = 0 // Invalid: missing repo_id
		err = db.WithContext(ctx).Save(source).Error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo_id")
	})
}

// TestModelsHooks_DirectoryMappingBeforeUpdate tests DirectoryMapping BeforeUpdate
func TestModelsHooks_DirectoryMappingBeforeUpdate(t *testing.T) {
	db := TestDB(t)
	ctx := context.Background()

	// Pre-create Client -> Organization -> Repo for FK-based tests
	client := &Client{Name: "DirMap Update Client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "dirmap-update-org"}
	require.NoError(t, db.Create(org).Error)

	repo := &Repo{OrganizationID: org.ID, Name: "dirmap-update-repo"}
	require.NoError(t, db.Create(repo).Error)

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
		RepoID:  repo.ID,
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
	require.Error(t, err)
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

// TestModelsHooks_ClientBeforeCreate tests Client hook validation
func TestModelsHooks_ClientBeforeCreate(t *testing.T) {
	db := TestDB(t)

	t.Run("Valid client", func(t *testing.T) {
		c := &Client{Name: "valid-client"}
		err := db.Create(c).Error
		require.NoError(t, err)
		assert.NotZero(t, c.ID)
	})

	t.Run("Empty name", func(t *testing.T) {
		c := &Client{Name: ""}
		err := db.Create(c).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "name")
	})
}

// TestModelsHooks_OrganizationBeforeCreate tests Organization hook validation
func TestModelsHooks_OrganizationBeforeCreate(t *testing.T) {
	db := TestDB(t)

	client := &Client{Name: "org-hook-client"}
	require.NoError(t, db.Create(client).Error)

	t.Run("Valid organization", func(t *testing.T) {
		o := &Organization{ClientID: client.ID, Name: "valid-org"}
		err := db.Create(o).Error
		require.NoError(t, err)
		assert.NotZero(t, o.ID)
	})

	t.Run("Missing client_id", func(t *testing.T) {
		o := &Organization{ClientID: 0, Name: "no-client-org"}
		err := db.Create(o).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "client_id")
	})

	t.Run("Empty name", func(t *testing.T) {
		o := &Organization{ClientID: client.ID, Name: ""}
		err := db.Create(o).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "name")
	})
}

// TestModelsHooks_RepoBeforeCreate tests Repo hook validation
func TestModelsHooks_RepoBeforeCreate(t *testing.T) {
	db := TestDB(t)

	client := &Client{Name: "repo-hook-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "repo-hook-org"}
	require.NoError(t, db.Create(org).Error)

	t.Run("Valid repo", func(t *testing.T) {
		r := &Repo{OrganizationID: org.ID, Name: "valid-repo"}
		err := db.Create(r).Error
		require.NoError(t, err)
		assert.NotZero(t, r.ID)
	})

	t.Run("Missing organization_id", func(t *testing.T) {
		r := &Repo{OrganizationID: 0, Name: "no-org-repo"}
		err := db.Create(r).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "organization_id")
	})

	t.Run("Empty name", func(t *testing.T) {
		r := &Repo{OrganizationID: org.ID, Name: ""}
		err := db.Create(r).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "repository name")
	})

	t.Run("Name with slash rejected", func(t *testing.T) {
		r := &Repo{OrganizationID: org.ID, Name: "org/repo"}
		err := db.Create(r).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
	})
}

// TestModelsHooks_SourceBeforeCreate tests Source hook validation for repo_id
func TestModelsHooks_SourceBeforeCreate(t *testing.T) {
	db := TestDB(t)

	// Pre-create entity chain
	client := &Client{Name: "source-hook-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "source-hook-org"}
	require.NoError(t, db.Create(org).Error)

	repo := &Repo{OrganizationID: org.ID, Name: "source-hook-repo"}
	require.NoError(t, db.Create(repo).Error)

	cfg := &Config{ExternalID: "source-hook-cfg", Name: "Test", Version: 1}
	require.NoError(t, db.Create(cfg).Error)

	group := &Group{ConfigID: cfg.ID, ExternalID: "source-hook-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	t.Run("Valid source with repo_id", func(t *testing.T) {
		source := &Source{
			GroupID: group.ID,
			RepoID:  repo.ID,
			Branch:  "main",
		}
		err := db.Create(source).Error
		require.NoError(t, err)
		assert.NotZero(t, source.ID)
	})

	t.Run("Missing repo_id", func(t *testing.T) {
		group2 := &Group{ConfigID: cfg.ID, ExternalID: "source-hook-group-2", Name: "Test Group 2"}
		require.NoError(t, db.Create(group2).Error)

		source := &Source{
			GroupID: group2.ID,
			RepoID:  0, // Missing
			Branch:  "main",
		}
		err := db.Create(source).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "repo_id")
	})
}

// TestModelsHooks_TargetBeforeCreate tests Target hook validation for repo_id
func TestModelsHooks_TargetBeforeCreate(t *testing.T) {
	db := TestDB(t)

	// Pre-create entity chain
	client := &Client{Name: "target-hook-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "target-hook-org"}
	require.NoError(t, db.Create(org).Error)

	repo := &Repo{OrganizationID: org.ID, Name: "target-hook-repo"}
	require.NoError(t, db.Create(repo).Error)

	cfg := &Config{ExternalID: "target-hook-cfg", Name: "Test", Version: 1}
	require.NoError(t, db.Create(cfg).Error)

	group := &Group{ConfigID: cfg.ID, ExternalID: "target-hook-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	t.Run("Valid target with repo_id", func(t *testing.T) {
		target := &Target{
			GroupID: group.ID,
			RepoID:  repo.ID,
		}
		err := db.Create(target).Error
		require.NoError(t, err)
		assert.NotZero(t, target.ID)
	})

	t.Run("Missing repo_id", func(t *testing.T) {
		target := &Target{
			GroupID: group.ID,
			RepoID:  0, // Missing
		}
		err := db.Create(target).Error
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, err.Error(), "repo_id")
	})
}
