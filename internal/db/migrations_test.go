package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRunMigrations_ConsolidateAnalyticsRepositories(t *testing.T) {
	gormDB := TestDB(t)

	// Seed client + org + repo (repos table is the destination of the migration)
	client := &Client{Name: "github"}
	require.NoError(t, gormDB.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "bsv-blockchain"}
	require.NoError(t, gormDB.Create(org).Error)

	repo := &Repo{OrganizationID: org.ID, Name: "go-wallet-toolbox"}
	require.NoError(t, gormDB.Create(repo).Error)

	// Create a legacy analytics_repositories table (this should be dropped by migration)
	require.NoError(t, gormDB.Exec(`
		CREATE TABLE analytics_repositories (
			id INTEGER PRIMARY KEY,
			full_name TEXT,
			metadata_e_tag TEXT,
			security_e_tag TEXT,
			last_sync_at DATETIME,
			last_sync_run_id INTEGER
		);
	`).Error)

	now := time.Now().UTC().Truncate(time.Second)

	// Insert legacy analytics row
	require.NoError(t, gormDB.Exec(`
		INSERT INTO analytics_repositories (id, full_name, metadata_e_tag, security_e_tag, last_sync_at, last_sync_run_id)
		VALUES (?, ?, ?, ?, ?, ?);
	`, 123, "bsv-blockchain/go-wallet-toolbox", "m-etag", "s-etag", now, 77).Error)

	// Sanity: table exists before migration
	require.True(t, gormDB.Migrator().HasTable("analytics_repositories"))

	// Run migrations
	require.NoError(t, RunMigrations(gormDB))

	// Table should be dropped
	require.False(t, gormDB.Migrator().HasTable("analytics_repositories"))

	// Repo full_name should be populated and legacy fields copied over
	var got Repo
	require.NoError(t, gormDB.First(&got, repo.ID).Error)
	require.Equal(t, "bsv-blockchain/go-wallet-toolbox", got.FullNameStr)
	require.Equal(t, "m-etag", got.MetadataETag)
	require.Equal(t, "s-etag", got.SecurityETag)
	require.NotNil(t, got.LastSyncAt)
	require.Equal(t, now, got.LastSyncAt.UTC())
	require.NotNil(t, got.LastSyncRunID)
	require.Equal(t, uint(77), *got.LastSyncRunID)
}
