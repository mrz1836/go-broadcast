package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestConverterFullRoundTrip tests a comprehensive round-trip with all features
func TestConverterFullRoundTrip(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	enabled := true
	preserveStructure := false
	includeHidden := true
	checkTags := true

	cfg := &config.Config{
		Version: 1,
		ID:      "comprehensive-test",
		Name:    "Comprehensive Test",
		FileLists: []config.FileList{
			{
				ID:          "comprehensive-filelist",
				Name:        "Comprehensive File List",
				Description: "All file features",
				Files: []config.FileMapping{
					{Src: "file1.txt", Dest: "dest1.txt", Delete: false},
					{Dest: "delete-me.txt", Delete: true},
				},
			},
		},
		DirectoryLists: []config.DirectoryList{
			{
				ID:          "comprehensive-dirlist",
				Name:        "Comprehensive Dir List",
				Description: "All directory features",
				Directories: []config.DirectoryMapping{
					{
						Src:               "src/dir1",
						Dest:              "dest/dir1",
						Exclude:           []string{"*.log", "tmp/*"},
						IncludeOnly:       []string{"*.go", "*.md"},
						PreserveStructure: &preserveStructure,
						IncludeHidden:     &includeHidden,
						Module: &config.ModuleConfig{
							Type:       "go",
							Version:    "v1.0.0",
							CheckTags:  &checkTags,
							UpdateRefs: true,
						},
						Transform: config.Transform{
							RepoName: true,
							Variables: map[string]string{
								"PROJECT": "myproject",
								"VERSION": "1.0",
							},
						},
					},
					{Dest: "delete-dir", Delete: true},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:          "comprehensive-group-1",
				Name:        "Comprehensive Group 1",
				Description: "First group with all features",
				Priority:    100,
				Enabled:     &enabled,
				Source: config.SourceConfig{
					Repo:          "mrz1836/source",
					Branch:        "main",
					BlobSizeLimit: "100MB",
					SecurityEmail: "security@source.com",
					SupportEmail:  "support@source.com",
				},
				Global: config.GlobalConfig{
					PRLabels:        []string{"global-label1", "global-label2"},
					PRAssignees:     []string{"global-assignee"},
					PRReviewers:     []string{"global-reviewer1", "global-reviewer2"},
					PRTeamReviewers: []string{"global-team"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix:    "feature",
					PRLabels:        []string{"default-label"},
					PRAssignees:     []string{"default-assignee"},
					PRReviewers:     []string{"default-reviewer"},
					PRTeamReviewers: []string{"default-team"},
				},
				Targets: []config.TargetConfig{
					{
						Repo:              "mrz1836/target1",
						Branch:            "develop",
						BlobSizeLimit:     "50MB",
						SecurityEmail:     "sec@target1.com",
						SupportEmail:      "sup@target1.com",
						PRLabels:          []string{"target-label1", "target-label2"},
						PRAssignees:       []string{"target-assignee"},
						PRReviewers:       []string{"target-reviewer"},
						PRTeamReviewers:   []string{"target-team1", "target-team2"},
						FileListRefs:      []string{"comprehensive-filelist"},
						DirectoryListRefs: []string{"comprehensive-dirlist"},
						Files: []config.FileMapping{
							{Src: "inline.txt", Dest: "inline-dest.txt"},
						},
						Directories: []config.DirectoryMapping{
							{
								Src:  "inline/dir",
								Dest: "inline-dest/dir",
								Transform: config.Transform{
									RepoName: true,
									Variables: map[string]string{
										"INLINE_VAR": "inline_val",
									},
								},
							},
						},
						Transform: config.Transform{
							RepoName: true,
							Variables: map[string]string{
								"TARGET_VAR": "target_val",
							},
						},
					},
					{
						Repo:   "mrz1836/target2",
						Branch: "staging",
					},
				},
			},
			{
				ID:        "comprehensive-group-2",
				Name:      "Comprehensive Group 2",
				DependsOn: []string{"comprehensive-group-1"},
				Source: config.SourceConfig{
					Repo:   "mrz1836/source2",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target3"},
				},
			},
		},
	}

	// Import
	importedConfig, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)
	assert.NotNil(t, importedConfig)

	// Export
	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)

	// Verify all features preserved
	assert.Equal(t, cfg.ID, exported.ID)
	assert.Equal(t, cfg.Name, exported.Name)
	assert.Len(t, exported.FileLists, 1)
	assert.Len(t, exported.DirectoryLists, 1)
	assert.Len(t, exported.Groups, 2)

	// Verify file list
	fileList := exported.FileLists[0]
	assert.Equal(t, "comprehensive-filelist", fileList.ID)
	assert.Len(t, fileList.Files, 2)
	assert.False(t, fileList.Files[0].Delete)
	assert.True(t, fileList.Files[1].Delete)

	// Verify directory list
	dirList := exported.DirectoryLists[0]
	assert.Equal(t, "comprehensive-dirlist", dirList.ID)
	assert.Len(t, dirList.Directories, 2)
	dir := dirList.Directories[0]
	assert.Len(t, dir.Exclude, 2)
	assert.Len(t, dir.IncludeOnly, 2)
	assert.NotNil(t, dir.PreserveStructure)
	assert.False(t, *dir.PreserveStructure)
	assert.NotNil(t, dir.Module)
	assert.Equal(t, "go", dir.Module.Type)

	// Verify group 1
	group1 := exported.Groups[0]
	assert.Equal(t, "Comprehensive Group 1", group1.Name)
	assert.Equal(t, 100, group1.Priority)
	assert.NotNil(t, group1.Enabled)
	assert.True(t, *group1.Enabled)
	assert.Equal(t, "100MB", group1.Source.BlobSizeLimit)
	assert.Len(t, group1.Global.PRLabels, 2)
	assert.Len(t, group1.Targets, 2)

	// Verify target 1
	target1 := group1.Targets[0]
	assert.Equal(t, "mrz1836/target1", target1.Repo)
	assert.Equal(t, "develop", target1.Branch)
	assert.Len(t, target1.FileListRefs, 1)
	assert.Len(t, target1.DirectoryListRefs, 1)
	assert.Len(t, target1.Files, 1)
	assert.Len(t, target1.Directories, 1)
	assert.True(t, target1.Transform.RepoName)

	// Verify group 2
	group2 := exported.Groups[1]
	assert.Equal(t, "Comprehensive Group 2", group2.Name)
	assert.Len(t, group2.DependsOn, 1)
	assert.Equal(t, "comprehensive-group-1", group2.DependsOn[0])
}
