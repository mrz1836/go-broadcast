package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
)

// setupAuditDB creates a file-based DB (so openDatabase finds it via the global
// dbPath) seeded with an organization and two repositories. It returns the
// gorm DB for further seeding.
func setupAuditDB(t *testing.T) db.Database {
	t.Helper()

	tmpPath := filepath.Join(t.TempDir(), "audit.db")
	withDBPath(t, tmpPath)

	database, err := db.Open(db.OpenOptions{Path: tmpPath, AutoMigrate: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	gormDB := database.DB()
	client := &db.Client{Name: "acme"}
	require.NoError(t, gormDB.Create(client).Error)
	org := &db.Organization{ClientID: client.ID, Name: "acme"}
	require.NoError(t, gormDB.Create(org).Error)
	require.NoError(t, gormDB.Create(&db.Repo{OrganizationID: org.ID, Name: "repo-a", FullNameStr: "acme/repo-a"}).Error)
	require.NoError(t, gormDB.Create(&db.Repo{OrganizationID: org.ID, Name: "repo-b", FullNameStr: "acme/repo-b"}).Error)

	return database
}

func TestResolveOrgRepos(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	database := setupAuditDB(t)
	_ = database
	ctx := context.Background()

	t.Run("returns repos for org", func(t *testing.T) {
		repos, err := resolveOrgRepos(ctx, "acme")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"acme/repo-a", "acme/repo-b"}, repos)
	})

	t.Run("empty for unknown org", func(t *testing.T) {
		repos, err := resolveOrgRepos(ctx, "nobody")
		require.NoError(t, err)
		assert.Empty(t, repos)
	})

	t.Run("db error when missing", func(t *testing.T) {
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))
		_, err := resolveOrgRepos(ctx, "acme")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database required")
	})
}

func TestResolveAllDBRepos(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	setupAuditDB(t)
	ctx := context.Background()

	t.Run("returns all repos", func(t *testing.T) {
		repos, err := resolveAllDBRepos(ctx)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"acme/repo-a", "acme/repo-b"}, repos)
	})

	t.Run("db error when missing", func(t *testing.T) {
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))
		_, err := resolveAllDBRepos(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database required")
	})
}

func TestSaveAuditResults(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	database := setupAuditDB(t)
	ctx := context.Background()

	// Seed a preset that the results reference.
	presetRepo := db.NewSettingsPresetRepository(database.DB())
	require.NoError(t, presetRepo.Create(ctx, &db.SettingsPreset{ExternalID: "mvp", Name: "MVP"}))

	results := []auditResult{
		{
			Repo:   "acme/repo-a",
			Preset: "mvp",
			Score:  100,
			Total:  2,
			Passed: 2,
			Checks: []auditCheckResult{
				{Setting: "has_issues", Expected: "true", Actual: "true", Pass: true},
			},
		},
		{Repo: "acme/repo-a", Error: "boom"},            // skipped due to error
		{Repo: "bad-format", Preset: "mvp"},             // skipped, no slash
		{Repo: "acme/unknown", Preset: "mvp"},           // skipped, repo not found
		{Repo: "acme/repo-b", Preset: "no-such-preset"}, // skipped, preset not found
	}

	// Should not panic and should persist exactly one audit row.
	saveAuditResults(ctx, results)

	var cnt int64
	database.DB().Model(&db.RepoSettingsAudit{}).Count(&cnt)
	assert.Equal(t, int64(1), cnt)
}

func TestSaveAuditResults_NoDatabase(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))
	// Must not panic when DB is unavailable.
	saveAuditResults(context.Background(), []auditResult{{Repo: "acme/repo-a", Preset: "mvp"}})
}

func TestUpdateRepoSettingsInDB(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	database := setupAuditDB(t)
	ctx := context.Background()

	preset := &config.SettingsPreset{
		ID:                       "mvp",
		AllowSquashMerge:         true,
		DeleteBranchOnMerge:      true,
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "PR_BODY",
	}

	t.Run("updates existing repo", func(t *testing.T) {
		updateRepoSettingsInDB(ctx, "acme/repo-a", preset)

		var repo db.Repo
		require.NoError(t, database.DB().Where("full_name = ?", "acme/repo-a").First(&repo).Error)
		assert.True(t, repo.AllowSquashMerge)
		assert.True(t, repo.DeleteBranchOnMerge)
		assert.Equal(t, "PR_TITLE", repo.SquashMergeCommitTitle)
	})

	t.Run("no-op for unknown repo", func(t *testing.T) {
		updateRepoSettingsInDB(ctx, "acme/missing", preset)
	})

	t.Run("no-op for bad format", func(t *testing.T) {
		updateRepoSettingsInDB(ctx, "no-slash", preset)
	})

	t.Run("no-op without database", func(t *testing.T) {
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))
		updateRepoSettingsInDB(ctx, "acme/repo-a", preset)
	})
}
