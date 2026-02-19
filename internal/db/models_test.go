package db

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetadata_Roundtrip tests Metadata JSON serialization/deserialization
func TestMetadata_Roundtrip(t *testing.T) {
	db := TestDB(t)

	tests := []struct {
		name     string
		metadata Metadata
	}{
		{
			name:     "simple key-value",
			metadata: Metadata{"key": "value", "number": float64(42)},
		},
		{
			name:     "nested objects",
			metadata: Metadata{"nested": map[string]interface{}{"inner": "value"}},
		},
		{
			name:     "array values",
			metadata: Metadata{"tags": []interface{}{"tag1", "tag2"}},
		},
		{
			name:     "nil metadata",
			metadata: nil,
		},
		{
			name:     "empty metadata",
			metadata: Metadata{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				ExternalID: "test-" + tt.name,
				Name:       "Test Config",
				Version:    1,
			}
			config.Metadata = tt.metadata

			// Create
			err := db.Create(config).Error
			require.NoError(t, err)

			// Retrieve
			var retrieved Config
			err = db.First(&retrieved, config.ID).Error
			require.NoError(t, err)

			// Compare (handle nil vs empty cases)
			if tt.metadata == nil {
				assert.Nil(t, retrieved.Metadata)
			} else {
				assert.Equal(t, tt.metadata, retrieved.Metadata)
			}
		})
	}
}

// TestJSONStringSlice_Roundtrip tests JSONStringSlice serialization
func TestJSONStringSlice_Roundtrip(t *testing.T) {
	db := TestDB(t)

	tests := []struct {
		name  string
		slice JSONStringSlice
	}{
		{
			name:  "multiple values",
			slice: JSONStringSlice{"label1", "label2", "label3"},
		},
		{
			name:  "single value",
			slice: JSONStringSlice{"single"},
		},
		{
			name:  "empty slice",
			slice: JSONStringSlice{},
		},
		{
			name:  "nil slice",
			slice: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config and group for FK
			config := &Config{ExternalID: "test-config-" + tt.name, Name: "Test Config", Version: 1}
			require.NoError(t, db.Create(config).Error)

			group := &Group{
				ConfigID:   config.ID,
				ExternalID: "test-group-" + tt.name,
				Name:       "Test Group",
			}
			require.NoError(t, db.Create(group).Error)

			global := &GroupGlobal{
				GroupID:  group.ID,
				PRLabels: tt.slice,
			}

			// Create
			err := db.Create(global).Error
			require.NoError(t, err)

			// Retrieve
			var retrieved GroupGlobal
			err = db.First(&retrieved, global.ID).Error
			require.NoError(t, err)

			// Compare
			if tt.slice == nil {
				assert.Nil(t, retrieved.PRLabels)
			} else {
				assert.Equal(t, tt.slice, retrieved.PRLabels)
			}
		})
	}
}

// TestJSONStringMap_Roundtrip tests JSONStringMap serialization
func TestJSONStringMap_Roundtrip(t *testing.T) {
	db := TestDB(t)

	tests := []struct {
		name      string
		variables JSONStringMap
	}{
		{
			name:      "multiple variables",
			variables: JSONStringMap{"VAR1": "value1", "VAR2": "value2"},
		},
		{
			name:      "single variable",
			variables: JSONStringMap{"SINGLE": "value"},
		},
		{
			name:      "empty map",
			variables: JSONStringMap{},
		},
		{
			name:      "nil map",
			variables: nil,
		},
	}

	// Use a counter for unique owner IDs
	ownerID := uint(1)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transform := &Transform{
				OwnerType: "target",
				OwnerID:   ownerID,
				RepoName:  true,
				Variables: tt.variables,
			}
			ownerID++ // Increment for next test

			// Create
			err := db.Create(transform).Error
			require.NoError(t, err)

			// Retrieve
			var retrieved Transform
			err = db.First(&retrieved, transform.ID).Error
			require.NoError(t, err)

			// Compare
			if tt.variables == nil {
				assert.Nil(t, retrieved.Variables)
			} else {
				assert.Equal(t, tt.variables, retrieved.Variables)
			}
		})
	}
}

// TestJSONModuleConfig_Roundtrip tests JSONModuleConfig serialization
func TestJSONModuleConfig_Roundtrip(t *testing.T) {
	db := TestDB(t)

	tests := []struct {
		name         string
		moduleConfig *JSONModuleConfig
	}{
		{
			name: "full config",
			moduleConfig: &JSONModuleConfig{
				Type:       "go",
				Version:    "1.21",
				CheckTags:  true,
				UpdateRefs: true,
			},
		},
		{
			name: "partial config",
			moduleConfig: &JSONModuleConfig{
				Type: "npm",
			},
		},
		{
			name:         "nil config",
			moduleConfig: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirMapping := &DirectoryMapping{
				OwnerType:    "target",
				OwnerID:      1,                   // Dummy ID
				Src:          ".github/workflows", // Required for validation
				Dest:         ".github/workflows",
				ModuleConfig: tt.moduleConfig,
			}

			// Create
			err := db.Create(dirMapping).Error
			require.NoError(t, err)

			// Retrieve
			var retrieved DirectoryMapping
			err = db.First(&retrieved, dirMapping.ID).Error
			require.NoError(t, err)

			// Compare
			if tt.moduleConfig == nil {
				// When nil is stored, it might be retrieved as zero-value struct
				// This is acceptable behavior for JSON serialization
				if retrieved.ModuleConfig != nil {
					assert.Equal(t, JSONModuleConfig{}, *retrieved.ModuleConfig)
				}
			} else {
				require.NotNil(t, retrieved.ModuleConfig)
				assert.Equal(t, *tt.moduleConfig, *retrieved.ModuleConfig)
			}
		})
	}
}

// TestSource_ValidationHooks tests Source validation on create/update
func TestSource_ValidationHooks(t *testing.T) {
	db := TestDB(t)

	// Create config and group for FK
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	// Create Client -> Organization -> Repo chain
	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)
	org := &Organization{ClientID: client.ID, Name: "testorg"}
	require.NoError(t, db.Create(org).Error)
	repo := &Repo{OrganizationID: org.ID, Name: "testrepo"}
	require.NoError(t, db.Create(repo).Error)

	tests := []struct {
		name      string
		source    Source
		expectErr bool
		errString string
	}{
		{
			name: "valid source",
			source: Source{
				GroupID: group.ID,
				RepoID:  repo.ID,
				Branch:  "main",
			},
			expectErr: false,
		},
		{
			name: "missing repo_id",
			source: Source{
				GroupID: group.ID,
				RepoID:  0,
				Branch:  "main",
			},
			expectErr: true,
			errString: "repo_id is required",
		},
		{
			name: "invalid email",
			source: Source{
				GroupID:       group.ID,
				RepoID:        repo.ID,
				Branch:        "main",
				SecurityEmail: "not-an-email",
			},
			expectErr: true,
			errString: "invalid field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(&tt.source).Error

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				require.NoError(t, err)
				assert.Equal(t, repo.ID, tt.source.RepoID)
			}
		})
	}
}

// TestTarget_ValidationHooks tests Target validation
func TestTarget_ValidationHooks(t *testing.T) {
	db := TestDB(t)

	// Create config and group for FK
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	group := &Group{ConfigID: config.ID, ExternalID: "test-group", Name: "Test Group"}
	require.NoError(t, db.Create(group).Error)

	// Create Client -> Organization -> Repo chain
	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)
	org := &Organization{ClientID: client.ID, Name: "testorg"}
	require.NoError(t, db.Create(org).Error)
	repo := &Repo{OrganizationID: org.ID, Name: "target-repo"}
	require.NoError(t, db.Create(repo).Error)

	tests := []struct {
		name      string
		target    Target
		expectErr bool
		errString string
	}{
		{
			name: "valid target",
			target: Target{
				GroupID: group.ID,
				RepoID:  repo.ID,
			},
			expectErr: false,
		},
		{
			name: "missing repo_id",
			target: Target{
				GroupID: group.ID,
				RepoID:  0,
			},
			expectErr: true,
			errString: "repo_id is required",
		},
		{
			name: "invalid email",
			target: Target{
				GroupID:      group.ID,
				RepoID:       repo.ID,
				SupportEmail: "bad@email@com",
			},
			expectErr: true,
			errString: "invalid field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(&tt.target).Error

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				require.NoError(t, err)
				assert.Equal(t, repo.ID, tt.target.RepoID)
			}
		})
	}
}

// TestFileMapping_ValidationHooks tests FileMapping validation
func TestFileMapping_ValidationHooks(t *testing.T) {
	db := TestDB(t)

	tests := []struct {
		name      string
		mapping   FileMapping
		expectErr bool
		errString string
	}{
		{
			name: "valid mapping",
			mapping: FileMapping{
				OwnerType: "target",
				OwnerID:   1,
				Src:       ".cursorrules",
				Dest:      ".cursorrules",
			},
			expectErr: false,
		},
		{
			name: "delete flag allows empty src",
			mapping: FileMapping{
				OwnerType:  "target",
				OwnerID:    1,
				Src:        "",
				Dest:       ".obsolete",
				DeleteFlag: true,
			},
			expectErr: false,
		},
		{
			name: "empty src without delete flag",
			mapping: FileMapping{
				OwnerType: "target",
				OwnerID:   1,
				Src:       "",
				Dest:      "dest.txt",
			},
			expectErr: true,
			errString: "required",
		},
		{
			name: "path traversal in dest",
			mapping: FileMapping{
				OwnerType: "target",
				OwnerID:   1,
				Src:       "file.txt",
				Dest:      "../etc/passwd",
			},
			expectErr: true,
			errString: "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(&tt.mapping).Error

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGroup_ValidationHooks tests Group validation
func TestGroup_ValidationHooks(t *testing.T) {
	db := TestDB(t)

	// Create config for FK
	config := &Config{ExternalID: "test-config", Name: "Test Config", Version: 1}
	require.NoError(t, db.Create(config).Error)

	tests := []struct {
		name      string
		group     Group
		expectErr bool
		errString string
	}{
		{
			name: "valid group",
			group: Group{
				ConfigID:   config.ID,
				ExternalID: "valid-group",
				Name:       "Valid Group",
			},
			expectErr: false,
		},
		{
			name: "empty name",
			group: Group{
				ExternalID: "no-name-group",
				Name:       "",
			},
			expectErr: true,
			errString: "cannot be empty",
		},
		{
			name: "empty external_id",
			group: Group{
				ExternalID: "",
				Name:       "Test Group",
			},
			expectErr: true,
			errString: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(&tt.group).Error

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMetadata_InvalidJSON tests error handling for invalid JSON
func TestMetadata_InvalidJSON(t *testing.T) {
	var m Metadata

	// Invalid JSON should error
	err := m.Scan([]byte("{invalid json}"))
	require.Error(t, err)

	// Invalid type should error
	err = m.Scan(12345)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

// TestJSONStringSlice_InvalidJSON tests error handling
func TestJSONStringSlice_InvalidJSON(t *testing.T) {
	var j JSONStringSlice

	err := j.Scan([]byte("{not an array}"))
	require.Error(t, err)

	err = j.Scan(12345)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

// TestJSONStringMap_InvalidJSON tests error handling
func TestJSONStringMap_InvalidJSON(t *testing.T) {
	var j JSONStringMap

	err := j.Scan([]byte("[not an object]"))
	require.Error(t, err)

	err = j.Scan(12345)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

// TestJSONModuleConfig_InvalidJSON tests error handling
func TestJSONModuleConfig_InvalidJSON(t *testing.T) {
	var j JSONModuleConfig

	err := j.Scan([]byte("invalid"))
	require.Error(t, err)

	err = j.Scan(12345)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

// TestCustomTypes_Value tests driver.Valuer implementation
func TestCustomTypes_Value(t *testing.T) {
	t.Run("Metadata.Value", func(t *testing.T) {
		// Non-nil
		m := Metadata{"key": "value"}
		val, err := m.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)

		// Verify JSON encoding
		var decoded map[string]interface{}
		err = json.Unmarshal(val.([]byte), &decoded)
		require.NoError(t, err)
		assert.Equal(t, "value", decoded["key"])

		// Nil
		var nilM Metadata
		val, err = nilM.Value()
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("JSONStringSlice.Value", func(t *testing.T) {
		// Non-nil
		j := JSONStringSlice{"a", "b"}
		val, err := j.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)

		// Nil
		var nilJ JSONStringSlice
		val, err = nilJ.Value()
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("JSONStringMap.Value", func(t *testing.T) {
		// Non-nil
		j := JSONStringMap{"key": "value"}
		val, err := j.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)

		// Nil
		var nilJ JSONStringMap
		val, err = nilJ.Value()
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("JSONModuleConfig.Value", func(t *testing.T) {
		j := JSONModuleConfig{Type: "go", Version: "1.21"}
		val, err := j.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
	})
}

// TestRepo_FullName tests the Repo.FullName() method returns "org/repo" format
func TestRepo_FullName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		repo     Repo
		expected string
	}{
		{
			name: "standard org and repo",
			repo: Repo{
				Name:         "my-repo",
				Organization: Organization{Name: "my-org"},
			},
			expected: "my-org/my-repo",
		},
		{
			name: "empty org name",
			repo: Repo{
				Name:         "my-repo",
				Organization: Organization{Name: ""},
			},
			expected: "/my-repo",
		},
		{
			name: "empty repo name",
			repo: Repo{
				Name:         "",
				Organization: Organization{Name: "my-org"},
			},
			expected: "my-org/",
		},
		{
			name: "both empty",
			repo: Repo{
				Name:         "",
				Organization: Organization{},
			},
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.repo.FullName())
		})
	}
}

// TestRepoModel_AllFieldsPersist tests that all enhanced Repo fields persist correctly
func TestRepoModel_AllFieldsPersist(t *testing.T) {
	db := TestDB(t)

	// Create client and organization first
	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{
		ClientID: client.ID,
		Name:     "test-org",
	}
	require.NoError(t, db.Create(org).Error)

	// Create repo with ALL new fields populated
	now := TimePtr(2024, 2, 14, 12, 0, 0)
	repo := &Repo{
		OrganizationID:        org.ID,
		Name:                  "test-repo",
		Description:           "Test description",
		Language:              "Go",
		HomepageURL:           "https://example.com",
		Topics:                `["golang","testing","cli"]`,
		License:               "MIT",
		DiskUsageKB:           2048,
		IsPrivate:             false,
		IsArchived:            false,
		IsFork:                true,
		ForkParent:            "upstream/repo",
		DefaultBranch:         "main",
		HasIssuesEnabled:      true,
		HasWikiEnabled:        false,
		HasDiscussionsEnabled: true,
		HTMLURL:               "https://github.com/test/repo",
		SSHURL:                "git@github.com:test/repo.git",
		CloneURL:              "https://github.com/test/repo.git",
		GitHubCreatedAt:       now,
		LastPushedAt:          now,
		GitHubUpdatedAt:       now,
	}

	// Save to database
	err := db.Create(repo).Error
	require.NoError(t, err)
	require.NotZero(t, repo.ID, "Repo should have ID after create")

	// Reload from database
	var loaded Repo
	err = db.Preload("Organization").First(&loaded, repo.ID).Error
	require.NoError(t, err)

	// Verify ALL fields persisted correctly
	assert.Equal(t, "test-repo", loaded.Name)
	assert.Equal(t, "Test description", loaded.Description)
	assert.Equal(t, "Go", loaded.Language, "Language field not persisted")
	assert.Equal(t, "https://example.com", loaded.HomepageURL, "Homepage URL not persisted")
	assert.Equal(t, `["golang","testing","cli"]`, loaded.Topics, "Topics JSON not persisted")
	assert.Equal(t, "MIT", loaded.License, "License not persisted")
	assert.Equal(t, 2048, loaded.DiskUsageKB, "Disk usage not persisted")
	assert.False(t, loaded.IsPrivate)
	assert.False(t, loaded.IsArchived)
	assert.True(t, loaded.IsFork, "Fork flag not persisted")
	assert.Equal(t, "upstream/repo", loaded.ForkParent, "Fork parent not persisted")
	assert.Equal(t, "main", loaded.DefaultBranch, "Default branch not persisted")
	assert.True(t, loaded.HasIssuesEnabled, "Issues flag not persisted")
	assert.False(t, loaded.HasWikiEnabled, "Wiki flag not persisted")
	assert.True(t, loaded.HasDiscussionsEnabled, "Discussions flag not persisted")
	assert.Equal(t, "https://github.com/test/repo", loaded.HTMLURL, "HTML URL not persisted")
	assert.Equal(t, "git@github.com:test/repo.git", loaded.SSHURL, "SSH URL not persisted")
	assert.Equal(t, "https://github.com/test/repo.git", loaded.CloneURL, "Clone URL not persisted")
	assert.NotNil(t, loaded.GitHubCreatedAt, "GitHub created timestamp not persisted")
	assert.NotNil(t, loaded.LastPushedAt, "Last pushed timestamp not persisted")
	assert.NotNil(t, loaded.GitHubUpdatedAt, "GitHub updated timestamp not persisted")

	// Verify relationship loaded
	assert.Equal(t, "test-org", loaded.Organization.Name)
}

// TestRepoModel_QueryByLanguage tests querying repos by programming language
func TestRepoModel_QueryByLanguage(t *testing.T) {
	db := TestDB(t)

	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "test-org"}
	require.NoError(t, db.Create(org).Error)

	// Create repos with different languages
	repos := []Repo{
		{OrganizationID: org.ID, Name: "go-repo", Language: "Go"},
		{OrganizationID: org.ID, Name: "py-repo", Language: "Python"},
		{OrganizationID: org.ID, Name: "js-repo", Language: "JavaScript"},
		{OrganizationID: org.ID, Name: "go-repo2", Language: "Go"},
	}

	for _, r := range repos {
		require.NoError(t, db.Create(&r).Error)
	}

	// Query by language using index
	var goRepos []Repo
	err := db.Where("language = ?", "Go").Find(&goRepos).Error
	require.NoError(t, err)

	assert.Len(t, goRepos, 2, "Should find 2 Go repos")
	names := []string{goRepos[0].Name, goRepos[1].Name}
	assert.Contains(t, names, "go-repo")
	assert.Contains(t, names, "go-repo2")
}

// TestRepoModel_QueryByLicense tests querying repos by license
func TestRepoModel_QueryByLicense(t *testing.T) {
	db := TestDB(t)

	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "test-org"}
	require.NoError(t, db.Create(org).Error)

	// Create repos with different licenses
	repos := []Repo{
		{OrganizationID: org.ID, Name: "mit-repo", License: "MIT"},
		{OrganizationID: org.ID, Name: "apache-repo", License: "Apache-2.0"},
		{OrganizationID: org.ID, Name: "mit-repo2", License: "MIT"},
		{OrganizationID: org.ID, Name: "unlicensed-repo", License: ""},
	}

	for _, r := range repos {
		require.NoError(t, db.Create(&r).Error)
	}

	// Query by license
	var mitRepos []Repo
	err := db.Where("license = ?", "MIT").Find(&mitRepos).Error
	require.NoError(t, err)

	assert.Len(t, mitRepos, 2, "Should find 2 MIT repos")
}

// TestRepoModel_TopicsJSON tests JSON topics field handling
func TestRepoModel_TopicsJSON(t *testing.T) {
	db := TestDB(t)

	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "test-org"}
	require.NoError(t, db.Create(org).Error)

	repo := &Repo{
		OrganizationID: org.ID,
		Name:           "test-repo",
		Topics:         `["golang","testing","open-source"]`,
	}
	require.NoError(t, db.Create(repo).Error)

	// Query using JSON LIKE
	var result []Repo
	err := db.Where("topics LIKE ?", "%golang%").Find(&result).Error
	require.NoError(t, err)
	assert.Len(t, result, 1)

	// Parse topics JSON
	var topics []string
	err = json.Unmarshal([]byte(result[0].Topics), &topics)
	require.NoError(t, err)
	assert.Equal(t, []string{"golang", "testing", "open-source"}, topics)
}

// TestRepoModel_QueryByStatus tests filtering by repo status fields
func TestRepoModel_QueryByStatus(t *testing.T) {
	db := TestDB(t)

	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "test-org"}
	require.NoError(t, db.Create(org).Error)

	// Create repos with various statuses
	repos := []Repo{
		{OrganizationID: org.ID, Name: "active-public", IsPrivate: false, IsArchived: false},
		{OrganizationID: org.ID, Name: "active-private", IsPrivate: true, IsArchived: false},
		{OrganizationID: org.ID, Name: "archived-public", IsPrivate: false, IsArchived: true},
		{OrganizationID: org.ID, Name: "archived-private", IsPrivate: true, IsArchived: true},
		{OrganizationID: org.ID, Name: "fork-repo", IsFork: true, ForkParent: "upstream/repo"},
	}

	for _, r := range repos {
		require.NoError(t, db.Create(&r).Error)
	}

	// Test: Find active private repos
	var activePrivate []Repo
	err := db.Where("is_private = ? AND is_archived = ?", true, false).Find(&activePrivate).Error
	require.NoError(t, err)
	assert.Len(t, activePrivate, 1)
	assert.Equal(t, "active-private", activePrivate[0].Name)

	// Test: Find forks
	var forks []Repo
	err = db.Where("is_fork = ?", true).Find(&forks).Error
	require.NoError(t, err)
	assert.Len(t, forks, 1)
	assert.Equal(t, "upstream/repo", forks[0].ForkParent)

	// Test: Find all active (non-archived) repos
	var active []Repo
	err = db.Where("is_archived = ?", false).Find(&active).Error
	require.NoError(t, err)
	assert.Len(t, active, 3) // active-public, active-private, fork-repo
}

// TestRepoModel_NullableTimestamps tests nullable timestamp fields
func TestRepoModel_NullableTimestamps(t *testing.T) {
	db := TestDB(t)

	client := &Client{Name: "test-client"}
	require.NoError(t, db.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "test-org"}
	require.NoError(t, db.Create(org).Error)

	// Repo with no timestamps
	repo1 := &Repo{
		OrganizationID:  org.ID,
		Name:            "no-timestamps",
		GitHubCreatedAt: nil,
		LastPushedAt:    nil,
		GitHubUpdatedAt: nil,
	}
	require.NoError(t, db.Create(repo1).Error)

	// Repo with timestamps
	now := TimePtr(2024, 2, 14, 12, 0, 0)
	repo2 := &Repo{
		OrganizationID:  org.ID,
		Name:            "with-timestamps",
		GitHubCreatedAt: now,
		LastPushedAt:    now,
		GitHubUpdatedAt: now,
	}
	require.NoError(t, db.Create(repo2).Error)

	// Verify null timestamps
	var loaded1 Repo
	err := db.First(&loaded1, repo1.ID).Error
	require.NoError(t, err)
	assert.Nil(t, loaded1.GitHubCreatedAt)
	assert.Nil(t, loaded1.LastPushedAt)
	assert.Nil(t, loaded1.GitHubUpdatedAt)

	// Verify non-null timestamps
	var loaded2 Repo
	err = db.First(&loaded2, repo2.ID).Error
	require.NoError(t, err)
	assert.NotNil(t, loaded2.GitHubCreatedAt)
	assert.NotNil(t, loaded2.LastPushedAt)
	assert.NotNil(t, loaded2.GitHubUpdatedAt)
}
