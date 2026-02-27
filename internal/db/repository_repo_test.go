package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoRepository_UpdateLastSyncTimestamp(t *testing.T) {
	t.Parallel()

	testDB := TestDB(t)
	repo := NewRepoRepository(testDB)
	ctx := context.Background()

	// Create a client + org + repo
	client := &Client{Name: "test-client-sync-ts"}
	require.NoError(t, testDB.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "test-org-sync-ts"}
	require.NoError(t, testDB.Create(org).Error)

	r := &Repo{OrganizationID: org.ID, Name: "test-repo-sync-ts"}
	require.NoError(t, repo.Create(ctx, r))
	require.Positive(t, r.ID)

	// Initially nil
	fetched, err := repo.GetByID(ctx, r.ID)
	require.NoError(t, err)
	assert.Nil(t, fetched.LastSyncAt)
	assert.Nil(t, fetched.LastSyncRunID)

	// Update timestamp
	syncAt := time.Now().Truncate(time.Second)
	var syncRunID uint = 42
	require.NoError(t, repo.UpdateLastSyncTimestamp(ctx, r.ID, syncAt, syncRunID))

	// Verify
	fetched, err = repo.GetByID(ctx, r.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.LastSyncAt)
	assert.WithinDuration(t, syncAt, *fetched.LastSyncAt, time.Second)
	require.NotNil(t, fetched.LastSyncRunID)
	assert.Equal(t, syncRunID, *fetched.LastSyncRunID)
}

func TestRepoRepository_UpdateLastBroadcastSyncTimestamp(t *testing.T) {
	t.Parallel()

	testDB := TestDB(t)
	repo := NewRepoRepository(testDB)
	ctx := context.Background()

	// Create a client + org + repo
	client := &Client{Name: "test-client-bcast-ts"}
	require.NoError(t, testDB.Create(client).Error)

	org := &Organization{ClientID: client.ID, Name: "test-org-bcast-ts"}
	require.NoError(t, testDB.Create(org).Error)

	r := &Repo{OrganizationID: org.ID, Name: "test-repo-bcast-ts"}
	require.NoError(t, repo.Create(ctx, r))
	require.Positive(t, r.ID)

	// Initially nil
	fetched, err := repo.GetByID(ctx, r.ID)
	require.NoError(t, err)
	assert.Nil(t, fetched.LastBroadcastSyncAt)
	assert.Nil(t, fetched.LastBroadcastSyncRunID)

	// Update timestamp
	syncAt := time.Now().Truncate(time.Second)
	var broadcastRunID uint = 99
	require.NoError(t, repo.UpdateLastBroadcastSyncTimestamp(ctx, r.ID, syncAt, broadcastRunID))

	// Verify
	fetched, err = repo.GetByID(ctx, r.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.LastBroadcastSyncAt)
	assert.WithinDuration(t, syncAt, *fetched.LastBroadcastSyncAt, time.Second)
	require.NotNil(t, fetched.LastBroadcastSyncRunID)
	assert.Equal(t, broadcastRunID, *fetched.LastBroadcastSyncRunID)

	// Verify analytics fields are untouched
	assert.Nil(t, fetched.LastSyncAt)
	assert.Nil(t, fetched.LastSyncRunID)
}
