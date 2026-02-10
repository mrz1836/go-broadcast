package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestModelsHooks_DirectoryMappingBeforeUpdate tests DirectoryMapping BeforeUpdate (0% -> 100%)
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

// TestModelsHooks_GroupDefaultBeforeCreate tests GroupDefault validation (60-80% coverage)
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

// TestModelsScan_StringBranch tests Scan functions with string input (88.9% -> 100%)
func TestModelsScan_StringBranch(t *testing.T) {
	t.Run("JSONStringSlice from string", func(t *testing.T) {
		var jss JSONStringSlice
		err := jss.Scan(`["item1","item2"]`)
		require.NoError(t, err)
		assert.Len(t, jss, 2)
		assert.Equal(t, "item1", jss[0])
	})

	t.Run("JSONStringMap from string", func(t *testing.T) {
		var jsm JSONStringMap
		err := jsm.Scan(`{"key":"value"}`)
		require.NoError(t, err)
		assert.Equal(t, "value", jsm["key"])
	})

	t.Run("Metadata from string", func(t *testing.T) {
		var meta Metadata
		err := meta.Scan(`{"key":"value"}`)
		require.NoError(t, err)
		assert.Equal(t, "value", meta["key"])
	})

	t.Run("JSONModuleConfig from string", func(t *testing.T) {
		var jmc JSONModuleConfig
		err := jmc.Scan(`{"type":"go","version":"v1.0.0"}`)
		require.NoError(t, err)
		assert.Equal(t, "go", jmc.Type)
		assert.Equal(t, "v1.0.0", jmc.Version)
	})
}

// TestConverterImport_SourceEdgeCases tests Source validation edge cases (77.8% -> higher)
func TestConverterImport_SourceEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Source with all email fields", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "source-emails-test",
			Name:    "Source Emails Test",
			Groups: []config.Group{
				{
					ID:   "group-source-emails",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:          "mrz1836/test",
						Branch:        "main",
						SecurityEmail: "security@example.com",
						SupportEmail:  "support@example.com",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Equal(t, "security@example.com", exported.Groups[0].Source.SecurityEmail)
		assert.Equal(t, "support@example.com", exported.Groups[0].Source.SupportEmail)
	})

	t.Run("Source with blob size limit", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "source-blob-test",
			Name:    "Source Blob Test",
			Groups: []config.Group{
				{
					ID:   "group-source-blob",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:          "mrz1836/test",
						Branch:        "main",
						BlobSizeLimit: "50MB",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Equal(t, "50MB", exported.Groups[0].Source.BlobSizeLimit)
	})
}

// TestConverterImport_FileMappingsEdgeCases tests importFileMappings edge cases (80% -> higher)
func TestConverterImport_FileMappingsEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("File mapping with empty src (delete)", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "filemp-empty-src-test",
			Name:    "FileMapping Empty Src Test",
			Groups: []config.Group{
				{
					ID:   "group-filemp-empty",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "mrz1836/target",
							Files: []config.FileMapping{
								{
									Src:    "", // Empty src
									Dest:   "to-delete.txt",
									Delete: true,
								},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.True(t, exported.Groups[0].Targets[0].Files[0].Delete)
		assert.Empty(t, exported.Groups[0].Targets[0].Files[0].Src)
	})

	t.Run("Multiple file mappings in target", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "filemp-multiple-test",
			Name:    "FileMapping Multiple Test",
			Groups: []config.Group{
				{
					ID:   "group-filemp-multi",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "mrz1836/target",
							Files: []config.FileMapping{
								{Src: "file1.txt", Dest: "dest1.txt"},
								{Src: "file2.txt", Dest: "dest2.txt"},
								{Src: "file3.txt", Dest: "dest3.txt"},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Len(t, exported.Groups[0].Targets[0].Files, 3)
	})
}

// TestConverterImport_DirectoryMappingsEdgeCases tests importDirectoryMappings edge cases (75% -> higher)
func TestConverterImport_DirectoryMappingsEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Directory mapping with empty src (delete)", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "dirmp-empty-src-test",
			Name:    "DirMapping Empty Src Test",
			Groups: []config.Group{
				{
					ID:   "group-dirmp-empty",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "mrz1836/target",
							Directories: []config.DirectoryMapping{
								{
									Src:    "", // Empty src
									Dest:   "to-delete-dir",
									Delete: true,
								},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.True(t, exported.Groups[0].Targets[0].Directories[0].Delete)
		assert.Empty(t, exported.Groups[0].Targets[0].Directories[0].Src)
	})

	t.Run("Multiple directory mappings in target", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "dirmp-multiple-test",
			Name:    "DirMapping Multiple Test",
			Groups: []config.Group{
				{
					ID:   "group-dirmp-multi",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "mrz1836/target",
							Directories: []config.DirectoryMapping{
								{Src: "dir1", Dest: "dest-dir1"},
								{Src: "dir2", Dest: "dest-dir2"},
								{Src: "dir3", Dest: "dest-dir3"},
							},
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Len(t, exported.Groups[0].Targets[0].Directories, 3)
	})
}

// TestConverterImport_GroupDependenciesEdgeCases tests importGroupDependencies edge cases (80% -> higher)
func TestConverterImport_GroupDependenciesEdgeCases(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Group with no dependencies", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "group-nodep-test",
			Name:    "Group No Dep Test",
			Groups: []config.Group{
				{
					ID:        "group-nodep",
					Name:      "No Dependencies",
					DependsOn: []string{}, // Explicitly empty
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)
		assert.Empty(t, exported.Groups[0].DependsOn)
	})

	t.Run("Group with single dependency", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "group-singledep-test",
			Name:    "Group Single Dep Test",
			Groups: []config.Group{
				{
					ID:   "group-single-a",
					Name: "Group A",
					Source: config.SourceConfig{
						Repo:   "mrz1836/a",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target-a"},
					},
				},
				{
					ID:        "group-single-b",
					Name:      "Group B",
					DependsOn: []string{"group-single-a"}, // Single dependency
					Source: config.SourceConfig{
						Repo:   "mrz1836/b",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target-b"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		exported, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(t, err)

		// Find group-single-b
		var groupB *config.Group
		for i := range exported.Groups {
			if exported.Groups[i].ID == "group-single-b" {
				groupB = &exported.Groups[i]
				break
			}
		}
		require.NotNil(t, groupB)
		assert.Equal(t, []string{"group-single-a"}, groupB.DependsOn)
	})
}
