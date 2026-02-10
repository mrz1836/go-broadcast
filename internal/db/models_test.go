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
				Repo:    "mrz1836/test-repo",
				Branch:  "main",
			},
			expectErr: false,
		},
		{
			name: "invalid repo format",
			source: Source{
				GroupID: group.ID,
				Repo:    "invalid",
				Branch:  "main",
			},
			expectErr: true,
			errString: "invalid format",
		},
		{
			name: "empty repo",
			source: Source{
				GroupID: group.ID,
				Repo:    "",
				Branch:  "main",
			},
			expectErr: true,
			errString: "cannot be empty",
		},
		{
			name: "invalid email",
			source: Source{
				GroupID:       group.ID,
				Repo:          "mrz1836/test",
				Branch:        "main",
				SecurityEmail: "not-an-email",
			},
			expectErr: true,
			errString: "invalid field",
		},
		{
			name: "path traversal in repo",
			source: Source{
				GroupID: group.ID,
				Repo:    "org/../etc/passwd",
				Branch:  "main",
			},
			expectErr: true,
			errString: "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(&tt.source).Error

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				assert.NoError(t, err)
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
				Repo:    "mrz1836/target-repo",
			},
			expectErr: false,
		},
		{
			name: "invalid repo",
			target: Target{
				GroupID: group.ID,
				Repo:    "invalid-repo",
			},
			expectErr: true,
			errString: "invalid format",
		},
		{
			name: "invalid email",
			target: Target{
				GroupID:      group.ID,
				Repo:         "mrz1836/test",
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
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				assert.NoError(t, err)
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
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				assert.NoError(t, err)
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
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errString)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestMetadata_InvalidJSON tests error handling for invalid JSON
func TestMetadata_InvalidJSON(t *testing.T) {
	var m Metadata

	// Invalid JSON should error
	err := m.Scan([]byte("{invalid json}"))
	assert.Error(t, err)

	// Invalid type should error
	err = m.Scan(12345)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

// TestJSONStringSlice_InvalidJSON tests error handling
func TestJSONStringSlice_InvalidJSON(t *testing.T) {
	var j JSONStringSlice

	err := j.Scan([]byte("{not an array}"))
	assert.Error(t, err)

	err = j.Scan(12345)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

// TestJSONStringMap_InvalidJSON tests error handling
func TestJSONStringMap_InvalidJSON(t *testing.T) {
	var j JSONStringMap

	err := j.Scan([]byte("[not an object]"))
	assert.Error(t, err)

	err = j.Scan(12345)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

// TestJSONModuleConfig_InvalidJSON tests error handling
func TestJSONModuleConfig_InvalidJSON(t *testing.T) {
	var j JSONModuleConfig

	err := j.Scan([]byte("invalid"))
	assert.Error(t, err)

	err = j.Scan(12345)
	assert.Error(t, err)
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
