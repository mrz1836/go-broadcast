package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsModels(t *testing.T) {
	db := TestDB(t)

	// Create a test client (required for Organization)
	client := &Client{
		Name:        "test-client",
		Description: "Test client for analytics",
	}
	require.NoError(t, db.Create(client).Error)

	// Create a test organization
	org := &Organization{
		ClientID:    client.ID,
		Name:        "test-org",
		Description: "Test organization",
	}
	require.NoError(t, db.Create(org).Error)

	// Test Repo creation with analytics fields
	t.Run("CreateRepoWithAnalyticsFields", func(t *testing.T) {
		repo := &Repo{
			OrganizationID: org.ID,
			Name:           "test-repo",
			FullNameStr:    "test-org/test-repo",
			Description:    "Test repository",
			DefaultBranch:  "main",
			Language:       "Go",
			IsPrivate:      false,
			IsFork:         false,
			IsArchived:     false,
			HTMLURL:        "https://github.com/test-org/test-repo",
		}

		err := db.Create(repo).Error
		require.NoError(t, err)
		assert.NotZero(t, repo.ID)
		assert.Equal(t, "test-org/test-repo", repo.FullNameStr)

		// Verify we can load with organization
		var loaded Repo
		err = db.Preload("Organization").First(&loaded, repo.ID).Error
		require.NoError(t, err)
		assert.Equal(t, org.Name, loaded.Organization.Name)
	})

	// Test RepositorySnapshot creation
	t.Run("CreateRepositorySnapshot", func(t *testing.T) {
		repo := &Repo{
			OrganizationID: org.ID,
			Name:           "snapshot-repo",
			FullNameStr:    "test-org/snapshot-repo",
			DefaultBranch:  "main",
		}
		require.NoError(t, db.Create(repo).Error)

		now := time.Now()
		snapshot := &RepositorySnapshot{
			RepositoryID:  repo.ID,
			SnapshotAt:    now,
			Stars:         42,
			Forks:         12,
			Watchers:      8,
			OpenIssues:    3,
			OpenPRs:       1,
			BranchCount:   5,
			LatestRelease: "v1.2.3",
		}

		err := db.Create(snapshot).Error
		require.NoError(t, err)
		assert.NotZero(t, snapshot.ID)
		assert.Equal(t, 42, snapshot.Stars)
	})

	// Test SecurityAlert creation
	t.Run("CreateSecurityAlert", func(t *testing.T) {
		repo := &Repo{
			OrganizationID: org.ID,
			Name:           "alert-repo",
			FullNameStr:    "test-org/alert-repo",
			DefaultBranch:  "main",
		}
		require.NoError(t, db.Create(repo).Error)

		alert := &SecurityAlert{
			RepositoryID: repo.ID,
			AlertType:    "dependabot",
			AlertNumber:  1,
			State:        "open",
			Severity:     "high",
			Summary:      "Vulnerable dependency",
			Description:  "Package X has a known vulnerability",
			HTMLURL:      "https://github.com/test-org/alert-repo/security/dependabot/1",
			AlertData: Metadata{
				"package":    "example-pkg",
				"cve":        "CVE-2024-1234",
				"cvss_score": 7.5,
			},
		}

		err := db.Create(alert).Error
		require.NoError(t, err)
		assert.NotZero(t, alert.ID)
		assert.Equal(t, "dependabot", alert.AlertType)
		assert.Equal(t, "high", alert.Severity)

		// Verify alert data stored correctly
		var loaded SecurityAlert
		err = db.First(&loaded, alert.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "CVE-2024-1234", loaded.AlertData["cve"])
	})

	// Test SyncRun creation and update
	t.Run("CreateAndUpdateSyncRun", func(t *testing.T) {
		started := time.Now()
		syncRun := &SyncRun{
			StartedAt:        started,
			Status:           "running",
			SyncType:         "full",
			ReposProcessed:   0,
			ReposSkipped:     0,
			ReposFailed:      0,
			SnapshotsCreated: 0,
			AlertsUpserted:   0,
			APICallsMade:     0,
			DurationMs:       0,
		}

		// Create
		err := db.Create(syncRun).Error
		require.NoError(t, err)
		assert.NotZero(t, syncRun.ID)
		assert.Equal(t, "running", syncRun.Status)

		// Update with results
		completed := time.Now()
		syncRun.CompletedAt = &completed
		syncRun.Status = "completed"
		syncRun.ReposProcessed = 75
		syncRun.ReposSkipped = 60
		syncRun.SnapshotsCreated = 15
		syncRun.AlertsUpserted = 42
		syncRun.APICallsMade = 228
		syncRun.DurationMs = 12000
		syncRun.Errors = Metadata{
			"errors": []map[string]interface{}{
				{"repo": "test-org/failed-repo", "error": "rate limit"},
			},
		}

		err = db.Save(syncRun).Error
		require.NoError(t, err)

		// Verify update
		var loaded SyncRun
		err = db.First(&loaded, syncRun.ID).Error
		require.NoError(t, err)
		assert.Equal(t, "completed", loaded.Status)
		assert.Equal(t, 75, loaded.ReposProcessed)
		assert.Equal(t, 60, loaded.ReposSkipped)
		assert.NotNil(t, loaded.CompletedAt)
	})

	// Test unique constraint on FullNameStr
	t.Run("UniqueConstraintOnFullName", func(t *testing.T) {
		repo1 := &Repo{
			OrganizationID: org.ID,
			Name:           "unique-repo",
			FullNameStr:    "test-org/unique-repo",
			DefaultBranch:  "main",
		}
		require.NoError(t, db.Create(repo1).Error)

		// Try to create duplicate
		repo2 := &Repo{
			OrganizationID: org.ID,
			Name:           "unique-repo-dup",
			FullNameStr:    "test-org/unique-repo",
			DefaultBranch:  "main",
		}
		err := db.Create(repo2).Error
		assert.Error(t, err) // Should fail on unique constraint
	})

	// Test relationship loading
	t.Run("LoadRelationships", func(t *testing.T) {
		repo := &Repo{
			OrganizationID: org.ID,
			Name:           "rel-repo",
			FullNameStr:    "test-org/rel-repo",
			DefaultBranch:  "main",
		}
		require.NoError(t, db.Create(repo).Error)

		// Create snapshots
		snapshot1 := &RepositorySnapshot{
			RepositoryID: repo.ID,
			SnapshotAt:   time.Now().Add(-24 * time.Hour),
			Stars:        10,
		}
		snapshot2 := &RepositorySnapshot{
			RepositoryID: repo.ID,
			SnapshotAt:   time.Now(),
			Stars:        12,
		}
		require.NoError(t, db.Create(snapshot1).Error)
		require.NoError(t, db.Create(snapshot2).Error)

		// Create alert
		alert := &SecurityAlert{
			RepositoryID: repo.ID,
			AlertType:    "code_scanning",
			AlertNumber:  1,
			State:        "open",
			Severity:     "medium",
		}
		require.NoError(t, db.Create(alert).Error)

		// Load with relationships
		var loaded Repo
		err := db.Preload("Organization").Preload("Snapshots").Preload("Alerts").First(&loaded, repo.ID).Error
		require.NoError(t, err)
		assert.Equal(t, org.Name, loaded.Organization.Name)
		assert.Len(t, loaded.Snapshots, 2)
		assert.Len(t, loaded.Alerts, 1)
	})
}

func TestAnalyticsIndexes(t *testing.T) {
	db := TestDB(t)

	// Verify indexes exist
	t.Run("VerifyIndexes", func(t *testing.T) {
		// This is a smoke test - GORM should create the indexes
		// We verify by creating sample data and querying

		client := &Client{Name: "idx-client"}
		require.NoError(t, db.Create(client).Error)

		org := &Organization{ClientID: client.ID, Name: "idx-org"}
		require.NoError(t, db.Create(org).Error)

		repo := &Repo{
			OrganizationID: org.ID,
			Name:           "idx-repo",
			FullNameStr:    "idx-org/idx-repo",
		}
		require.NoError(t, db.Create(repo).Error)

		// Query by indexed fields (should be fast)
		var found Repo
		err := db.Where("full_name = ?", "idx-org/idx-repo").First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, repo.ID, found.ID)
	})
}
