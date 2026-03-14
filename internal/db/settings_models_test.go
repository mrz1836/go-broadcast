package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONAuditResultsScan_Error tests Scan error paths
func TestJSONAuditResultsScan_Error(t *testing.T) {
	var jar JSONAuditResults

	// Test nil value
	err := jar.Scan(nil)
	require.NoError(t, err)
	assert.Nil(t, jar)

	// Test invalid JSON
	err = jar.Scan([]byte(`{invalid json`))
	assert.Error(t, err)
}

// TestJSONAuditResultsScan_StringBranch tests Scan with string input
func TestJSONAuditResultsScan_StringBranch(t *testing.T) {
	t.Run("JSONAuditResults from string", func(t *testing.T) {
		var jar JSONAuditResults
		err := jar.Scan(`[{"setting":"has_issues","expected":"true","actual":"true","pass":true}]`)
		require.NoError(t, err)
		assert.Len(t, jar, 1)
		assert.Equal(t, "has_issues", jar[0].Setting)
		assert.Equal(t, "true", jar[0].Expected)
		assert.Equal(t, "true", jar[0].Actual)
		assert.True(t, jar[0].Pass)
	})

	t.Run("JSONAuditResults from []byte", func(t *testing.T) {
		var jar JSONAuditResults
		err := jar.Scan([]byte(`[{"setting":"has_wiki","expected":"false","actual":"true","pass":false}]`))
		require.NoError(t, err)
		assert.Len(t, jar, 1)
		assert.Equal(t, "has_wiki", jar[0].Setting)
		assert.False(t, jar[0].Pass)
	})

	t.Run("JSONAuditResults from unsupported type", func(t *testing.T) {
		var jar JSONAuditResults
		err := jar.Scan(12345)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidType)
	})
}

// TestJSONAuditResultsValue tests Value() method
func TestJSONAuditResultsValue(t *testing.T) {
	t.Run("JSONAuditResults Value", func(t *testing.T) {
		jar := JSONAuditResults{
			{Setting: "has_issues", Expected: "true", Actual: "true", Pass: true},
			{Setting: "has_wiki", Expected: "false", Actual: "true", Pass: false},
		}
		val, err := jar.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
	})

	t.Run("Nil JSONAuditResults Value", func(t *testing.T) {
		var jar JSONAuditResults
		val, err := jar.Value()
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("Empty JSONAuditResults Value", func(t *testing.T) {
		jar := JSONAuditResults{}
		val, err := jar.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
		assert.Equal(t, []byte("[]"), val)
	})
}

// TestSettingsModelsMigration verifies all 4 new tables exist after AutoMigrate
func TestSettingsModelsMigration(t *testing.T) {
	db := TestDB(t)

	tables := []string{
		"settings_presets",
		"settings_preset_labels",
		"settings_preset_rulesets",
		"repo_settings_audits",
	}
	for _, table := range tables {
		assert.True(t, db.Migrator().HasTable(table), "table %s should exist after migration", table)
	}
}

// TestSettingsPresetCRUD tests basic create and read operations for SettingsPreset
func TestSettingsPresetCRUD(t *testing.T) {
	db := TestDB(t)

	t.Run("CreateAndReadPreset", func(t *testing.T) {
		preset := &SettingsPreset{
			ExternalID:               "go-default",
			Name:                     "Go Default Settings",
			Description:              "Default settings for Go repositories",
			HasIssues:                true,
			HasWiki:                  false,
			HasProjects:              false,
			HasDiscussions:           false,
			AllowSquashMerge:         true,
			AllowMergeCommit:         false,
			AllowRebaseMerge:         false,
			DeleteBranchOnMerge:      true,
			AllowAutoMerge:           true,
			AllowUpdateBranch:        true,
			SquashMergeCommitTitle:   "PR_TITLE",
			SquashMergeCommitMessage: "BLANK",
		}
		preset.Metadata = Metadata{"category": "golang"}

		err := db.Create(preset).Error
		require.NoError(t, err)
		assert.NotZero(t, preset.ID)

		// Read back
		var loaded SettingsPreset
		err = db.First(&loaded, preset.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "go-default", loaded.ExternalID)
		assert.Equal(t, "Go Default Settings", loaded.Name)
		assert.True(t, loaded.HasIssues)
		assert.False(t, loaded.HasWiki)
		assert.True(t, loaded.AllowSquashMerge)
		assert.Equal(t, "PR_TITLE", loaded.SquashMergeCommitTitle)
	})

	t.Run("CreatePresetWithChildren", func(t *testing.T) {
		preset := &SettingsPreset{
			ExternalID:  "with-children",
			Name:        "Preset With Children",
			Description: "Preset with labels and rulesets",
		}
		require.NoError(t, db.Create(preset).Error)

		// Create labels
		label1 := &SettingsPresetLabel{
			SettingsPresetID: preset.ID,
			Name:             "bug",
			Color:            "d73a4a",
			Description:      "Something is broken",
		}
		label2 := &SettingsPresetLabel{
			SettingsPresetID: preset.ID,
			Name:             "enhancement",
			Color:            "a2eeef",
			Description:      "New feature or request",
		}
		require.NoError(t, db.Create(label1).Error)
		require.NoError(t, db.Create(label2).Error)

		// Create ruleset
		ruleset := &SettingsPresetRuleset{
			SettingsPresetID: preset.ID,
			Name:             "main-protection",
			Target:           "branch",
			Enforcement:      "active",
			Include:          JSONStringSlice{"~DEFAULT_BRANCH"},
			Exclude:          JSONStringSlice{"release/*"},
			Rules:            JSONStringSlice{"required_pull_request", "required_status_checks"},
		}
		require.NoError(t, db.Create(ruleset).Error)

		// Load with children
		var loaded SettingsPreset
		err := db.Preload("Labels").Preload("Rulesets").First(&loaded, preset.ID).Error
		require.NoError(t, err)
		assert.Len(t, loaded.Labels, 2)
		assert.Len(t, loaded.Rulesets, 1)
		assert.Equal(t, "bug", loaded.Labels[0].Name)
		assert.Equal(t, "main-protection", loaded.Rulesets[0].Name)
		assert.Equal(t, "branch", loaded.Rulesets[0].Target)
		assert.Len(t, loaded.Rulesets[0].Include, 1)
		assert.Equal(t, "~DEFAULT_BRANCH", loaded.Rulesets[0].Include[0])
	})

	t.Run("UniqueConstraintOnExternalID", func(t *testing.T) {
		preset1 := &SettingsPreset{
			ExternalID: "unique-preset",
			Name:       "First Preset",
		}
		require.NoError(t, db.Create(preset1).Error)

		preset2 := &SettingsPreset{
			ExternalID: "unique-preset",
			Name:       "Duplicate Preset",
		}
		err := db.Create(preset2).Error
		assert.Error(t, err) // Should fail on unique constraint
	})
}

// TestSettingsPresetForeignKeys tests FK relationships between models
func TestSettingsPresetForeignKeys(t *testing.T) {
	db := TestDB(t)

	t.Run("LabelBelongsToPreset", func(t *testing.T) {
		preset := &SettingsPreset{
			ExternalID: "fk-label-preset",
			Name:       "FK Label Test",
		}
		require.NoError(t, db.Create(preset).Error)

		label := &SettingsPresetLabel{
			SettingsPresetID: preset.ID,
			Name:             "documentation",
			Color:            "0075ca",
			Description:      "Improvements to documentation",
		}
		require.NoError(t, db.Create(label).Error)

		// Verify FK is set correctly
		var loaded SettingsPresetLabel
		err := db.First(&loaded, label.ID).Error
		require.NoError(t, err)
		assert.Equal(t, preset.ID, loaded.SettingsPresetID)
	})

	t.Run("RulesetBelongsToPreset", func(t *testing.T) {
		preset := &SettingsPreset{
			ExternalID: "fk-ruleset-preset",
			Name:       "FK Ruleset Test",
		}
		require.NoError(t, db.Create(preset).Error)

		ruleset := &SettingsPresetRuleset{
			SettingsPresetID: preset.ID,
			Name:             "tag-protection",
			Target:           "tag",
			Enforcement:      "active",
			Include:          JSONStringSlice{"v*"},
			Exclude:          JSONStringSlice{},
			Rules:            JSONStringSlice{"required_signatures"},
		}
		require.NoError(t, db.Create(ruleset).Error)

		// Verify FK is set correctly
		var loaded SettingsPresetRuleset
		err := db.First(&loaded, ruleset.ID).Error
		require.NoError(t, err)
		assert.Equal(t, preset.ID, loaded.SettingsPresetID)
		assert.Equal(t, "tag", loaded.Target)
	})

	t.Run("AuditForeignKeyToRepo", func(t *testing.T) {
		// Create client -> org -> repo chain
		client := &Client{Name: "audit-client"}
		require.NoError(t, db.Create(client).Error)

		org := &Organization{ClientID: client.ID, Name: "audit-org"}
		require.NoError(t, db.Create(org).Error)

		repo := &Repo{
			OrganizationID: org.ID,
			Name:           "audit-repo",
			FullNameStr:    "audit-org/audit-repo",
		}
		require.NoError(t, db.Create(repo).Error)

		preset := &SettingsPreset{
			ExternalID: "audit-preset",
			Name:       "Audit Preset",
		}
		require.NoError(t, db.Create(preset).Error)

		// Create audit
		audit := &RepoSettingsAudit{
			RepoID:           repo.ID,
			SettingsPresetID: preset.ID,
			Score:            80,
			Total:            10,
			Passed:           8,
			Results: JSONAuditResults{
				{Setting: "has_issues", Expected: "true", Actual: "true", Pass: true},
				{Setting: "has_wiki", Expected: "false", Actual: "true", Pass: false},
			},
		}
		require.NoError(t, db.Create(audit).Error)
		assert.NotZero(t, audit.ID)

		// Load with relationships
		var loaded RepoSettingsAudit
		err := db.Preload("Repo").Preload("Preset").First(&loaded, audit.ID).Error
		require.NoError(t, err)
		assert.Equal(t, repo.ID, loaded.RepoID)
		assert.Equal(t, preset.ID, loaded.SettingsPresetID)
		assert.Equal(t, 80, loaded.Score)
		assert.Equal(t, 10, loaded.Total)
		assert.Equal(t, 8, loaded.Passed)
		assert.Len(t, loaded.Results, 2)
		assert.Equal(t, "has_issues", loaded.Results[0].Setting)
		assert.True(t, loaded.Results[0].Pass)
		assert.False(t, loaded.Results[1].Pass)

		// Verify loaded relationships
		assert.Equal(t, "audit-repo", loaded.Repo.Name)
		assert.Equal(t, "Audit Preset", loaded.Preset.Name)
	})

	t.Run("AuditFKConstraintInvalidRepo", func(t *testing.T) {
		preset := &SettingsPreset{
			ExternalID: "fk-invalid-repo-preset",
			Name:       "FK Invalid Repo Test",
		}
		require.NoError(t, db.Create(preset).Error)

		// Attempt to create audit with non-existent repo ID
		audit := &RepoSettingsAudit{
			RepoID:           99999,
			SettingsPresetID: preset.ID,
			Score:            0,
			Total:            0,
			Passed:           0,
		}
		err := db.Create(audit).Error
		assert.Error(t, err) // Should fail on FK constraint
	})
}
