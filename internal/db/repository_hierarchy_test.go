package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientRepository_Create tests creating a client
func TestClientRepository_Create(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	client := &Client{
		Name:        "Test Client",
		Description: "Test description",
	}
	client.Metadata = Metadata{"key": "value"}

	err := repo.Create(ctx, client)
	require.NoError(t, err)
	assert.NotZero(t, client.ID)
	assert.NotZero(t, client.CreatedAt)
	assert.NotZero(t, client.UpdatedAt)
}

// TestClientRepository_GetByID tests retrieving a client by ID
func TestClientRepository_GetByID(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	// Create client
	client := &Client{Name: "Test Client"}
	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Get by ID
	fetched, err := repo.GetByID(ctx, client.ID)
	require.NoError(t, err)
	assert.Equal(t, client.ID, fetched.ID)
	assert.Equal(t, "Test Client", fetched.Name)
}

// TestClientRepository_GetByID_NotFound tests error when client not found
func TestClientRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	_, err := repo.GetByID(ctx, 99999)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRecordNotFound)
	assert.Contains(t, err.Error(), "client id=99999")
}

// TestClientRepository_GetByName tests retrieving a client by name
func TestClientRepository_GetByName(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	// Create client
	client := &Client{Name: "Unique Client Name"}
	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Get by name
	fetched, err := repo.GetByName(ctx, "Unique Client Name")
	require.NoError(t, err)
	assert.Equal(t, client.ID, fetched.ID)
	assert.Equal(t, "Unique Client Name", fetched.Name)
}

// TestClientRepository_GetByName_NotFound tests error when client name not found
func TestClientRepository_GetByName_NotFound(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	_, err := repo.GetByName(ctx, "Non-existent Client")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRecordNotFound)
	assert.Contains(t, err.Error(), "Non-existent Client")
}

// TestClientRepository_Update tests updating a client
func TestClientRepository_Update(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	// Create client
	client := &Client{Name: "original-name"}
	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Update
	client.Name = "updated-name"
	client.Description = "New description"
	err = repo.Update(ctx, client)
	require.NoError(t, err)

	// Verify
	fetched, err := repo.GetByID(ctx, client.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-name", fetched.Name)
	assert.Equal(t, "New description", fetched.Description)
}

// TestClientRepository_Delete_Soft tests soft deletion
func TestClientRepository_Delete_Soft(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	// Create client
	client := &Client{Name: "To Be Soft Deleted"}
	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Soft delete
	err = repo.Delete(ctx, client.ID, false)
	require.NoError(t, err)

	// Should not be found
	_, err = repo.GetByID(ctx, client.ID)
	require.ErrorIs(t, err, ErrRecordNotFound)

	// But should exist with Unscoped
	var count int64
	err = database.Unscoped().Model(&Client{}).Where("id = ?", client.ID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// TestClientRepository_Delete_Hard tests hard deletion
func TestClientRepository_Delete_Hard(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	// Create client
	client := &Client{Name: "To Be Hard Deleted"}
	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// Hard delete
	err = repo.Delete(ctx, client.ID, true)
	require.NoError(t, err)

	// Should not exist even with Unscoped
	var count int64
	countErr := database.Unscoped().Model(&Client{}).Where("id = ?", client.ID).Count(&count).Error
	require.NoError(t, countErr)
	assert.Equal(t, int64(0), count)
}

// TestClientRepository_Delete_NotFound tests delete error when client doesn't exist
func TestClientRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	err := repo.Delete(ctx, 99999, false)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

// TestClientRepository_List tests listing all clients
func TestClientRepository_List(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	// Create multiple clients
	clients := []*Client{
		{Name: "Client C"},
		{Name: "Client A"},
		{Name: "Client B"},
	}
	for _, c := range clients {
		err := repo.Create(ctx, c)
		require.NoError(t, err)
	}

	// List
	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// Verify ordering (should be by name ASC)
	assert.Equal(t, "Client A", list[0].Name)
	assert.Equal(t, "Client B", list[1].Name)
	assert.Equal(t, "Client C", list[2].Name)
}

// TestClientRepository_ListWithOrganizations tests preloading organizations
func TestClientRepository_ListWithOrganizations(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewClientRepository(database)

	// Create client with organizations
	client := &Client{Name: "Client with Orgs"}
	err := repo.Create(ctx, client)
	require.NoError(t, err)

	org1 := &Organization{ClientID: client.ID, Name: "Org B"}
	err = database.Create(org1).Error
	require.NoError(t, err)

	org2 := &Organization{ClientID: client.ID, Name: "Org A"}
	err = database.Create(org2).Error
	require.NoError(t, err)

	// List with organizations
	list, err := repo.ListWithOrganizations(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	// Verify organizations are preloaded and ordered
	assert.Len(t, list[0].Organizations, 2)
	assert.Equal(t, "Org A", list[0].Organizations[0].Name)
	assert.Equal(t, "Org B", list[0].Organizations[1].Name)
}

// ========== Organization Repository Tests ==========

// TestOrganizationRepository_Create tests creating an organization
func TestOrganizationRepository_Create(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create client first
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	// Create organization
	org := &Organization{
		ClientID:    client.ID,
		Name:        "Test Org",
		Description: "Test description",
	}
	org.Metadata = Metadata{"key": "value"}

	err = repo.Create(ctx, org)
	require.NoError(t, err)
	assert.NotZero(t, org.ID)
	assert.NotZero(t, org.CreatedAt)
}

// TestOrganizationRepository_GetByID tests retrieving an organization by ID
func TestOrganizationRepository_GetByID(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create client and organization
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "Test Org"}
	err = repo.Create(ctx, org)
	require.NoError(t, err)

	// Get by ID
	fetched, err := repo.GetByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, org.ID, fetched.ID)
	assert.Equal(t, "Test Org", fetched.Name)
}

// TestOrganizationRepository_GetByID_NotFound tests error when org not found
func TestOrganizationRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	_, err := repo.GetByID(ctx, 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

// TestOrganizationRepository_GetByName tests retrieving an organization by name
func TestOrganizationRepository_GetByName(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create client and organization
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "Unique Org"}
	err = repo.Create(ctx, org)
	require.NoError(t, err)

	// Get by name
	fetched, err := repo.GetByName(ctx, "Unique Org")
	require.NoError(t, err)
	assert.Equal(t, org.ID, fetched.ID)
}

// TestOrganizationRepository_GetByName_NotFound tests error when org name not found
func TestOrganizationRepository_GetByName_NotFound(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	_, err := repo.GetByName(ctx, "Non-existent Org")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

// TestOrganizationRepository_Update tests updating an organization
func TestOrganizationRepository_Update(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create client and organization
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "Original Org"}
	err = repo.Create(ctx, org)
	require.NoError(t, err)

	// Update
	org.Name = "Updated Org"
	org.Description = "New description"
	err = repo.Update(ctx, org)
	require.NoError(t, err)

	// Verify
	fetched, err := repo.GetByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Org", fetched.Name)
	assert.Equal(t, "New description", fetched.Description)
}

// TestOrganizationRepository_Delete tests deletion
func TestOrganizationRepository_Delete(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create client and organization
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "to-delete"}
	err = repo.Create(ctx, org)
	require.NoError(t, err)

	// Soft delete
	err = repo.Delete(ctx, org.ID, false)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, org.ID)
	require.ErrorIs(t, err, ErrRecordNotFound)
}

// TestOrganizationRepository_List tests listing organizations for a client
func TestOrganizationRepository_List(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create clients
	client1 := &Client{Name: "Client 1"}
	err := database.Create(client1).Error
	require.NoError(t, err)

	client2 := &Client{Name: "Client 2"}
	err = database.Create(client2).Error
	require.NoError(t, err)

	// Create organizations
	orgs := []*Organization{
		{ClientID: client1.ID, Name: "Org C"},
		{ClientID: client1.ID, Name: "Org A"},
		{ClientID: client2.ID, Name: "Org B"}, // Different client
	}
	for _, o := range orgs {
		createErr := repo.Create(ctx, o)
		require.NoError(t, createErr)
	}

	// List for client1
	list, err := repo.List(ctx, client1.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Verify ordering
	assert.Equal(t, "Org A", list[0].Name)
	assert.Equal(t, "Org C", list[1].Name)
}

// TestOrganizationRepository_ListWithRepos tests preloading repos
func TestOrganizationRepository_ListWithRepos(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create client and organization
	client := &Client{Name: "Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "Org with Repos"}
	err = repo.Create(ctx, org)
	require.NoError(t, err)

	// Create repos
	repo1 := &Repo{OrganizationID: org.ID, Name: "repo-b"}
	err = database.Create(repo1).Error
	require.NoError(t, err)

	repo2 := &Repo{OrganizationID: org.ID, Name: "repo-a"}
	err = database.Create(repo2).Error
	require.NoError(t, err)

	// List with repos
	list, err := repo.ListWithRepos(ctx, client.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	// Verify repos are preloaded and ordered
	assert.Len(t, list[0].Repos, 2)
	assert.Equal(t, "repo-a", list[0].Repos[0].Name)
	assert.Equal(t, "repo-b", list[0].Repos[1].Name)
}

// TestOrganizationRepository_FindOrCreate_Exists tests FindOrCreate when org exists
func TestOrganizationRepository_FindOrCreate_Exists(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create client and organization
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	existingOrg := &Organization{ClientID: client.ID, Name: "Existing Org"}
	err = repo.Create(ctx, existingOrg)
	require.NoError(t, err)

	// FindOrCreate should return existing
	org, err := repo.FindOrCreate(ctx, "Existing Org", client.ID)
	require.NoError(t, err)
	assert.Equal(t, existingOrg.ID, org.ID)
	assert.Equal(t, "Existing Org", org.Name)
}

// TestOrganizationRepository_FindOrCreate_Creates tests FindOrCreate when org doesn't exist
func TestOrganizationRepository_FindOrCreate_Creates(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewOrganizationRepository(database)

	// Create client
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	// FindOrCreate should create new
	org, err := repo.FindOrCreate(ctx, "New Org", client.ID)
	require.NoError(t, err)
	assert.NotZero(t, org.ID)
	assert.Equal(t, "New Org", org.Name)
	assert.Equal(t, client.ID, org.ClientID)
}

// ========== Repo Repository Tests ==========

// TestRepoRepository_Create tests creating a repo
func TestRepoRepository_Create(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client and organization
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "Test Org"}
	err = database.Create(org).Error
	require.NoError(t, err)

	// Create repo
	testRepo := &Repo{
		OrganizationID: org.ID,
		Name:           "test-repo",
		Description:    "Test description",
	}
	testRepo.Metadata = Metadata{"key": "value"}

	err = repo.Create(ctx, testRepo)
	require.NoError(t, err)
	assert.NotZero(t, testRepo.ID)
}

// TestRepoRepository_GetByID tests retrieving a repo by ID
func TestRepoRepository_GetByID(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client, organization, and repo
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "Test Org"}
	err = database.Create(org).Error
	require.NoError(t, err)

	testRepo := &Repo{OrganizationID: org.ID, Name: "test-repo"}
	err = repo.Create(ctx, testRepo)
	require.NoError(t, err)

	// Get by ID
	fetched, err := repo.GetByID(ctx, testRepo.ID)
	require.NoError(t, err)
	assert.Equal(t, testRepo.ID, fetched.ID)
	assert.Equal(t, "test-repo", fetched.Name)

	// Verify organization is preloaded
	assert.NotNil(t, fetched.Organization)
	assert.Equal(t, "Test Org", fetched.Organization.Name)
}

// TestRepoRepository_GetByID_NotFound tests error when repo not found
func TestRepoRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	_, err := repo.GetByID(ctx, 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

// TestRepoRepository_GetByFullName tests retrieving a repo by org/repo format
func TestRepoRepository_GetByFullName(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client, organization, and repo
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "mrz1836"}
	err = database.Create(org).Error
	require.NoError(t, err)

	testRepo := &Repo{OrganizationID: org.ID, Name: "go-broadcast"}
	err = repo.Create(ctx, testRepo)
	require.NoError(t, err)

	// Get by full name
	fetched, err := repo.GetByFullName(ctx, "mrz1836", "go-broadcast")
	require.NoError(t, err)
	assert.Equal(t, testRepo.ID, fetched.ID)
	assert.Equal(t, "go-broadcast", fetched.Name)

	// Verify organization is preloaded
	assert.NotNil(t, fetched.Organization)
	assert.Equal(t, "mrz1836", fetched.Organization.Name)
}

// TestRepoRepository_GetByFullName_NotFound tests error when repo not found
func TestRepoRepository_GetByFullName_NotFound(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	_, err := repo.GetByFullName(ctx, "nonexistent", "repo")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRecordNotFound)
	assert.Contains(t, err.Error(), "nonexistent/repo")
}

// TestRepoRepository_Update tests updating a repo
func TestRepoRepository_Update(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client, organization, and repo
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "Test Org"}
	err = database.Create(org).Error
	require.NoError(t, err)

	testRepo := &Repo{OrganizationID: org.ID, Name: "original-name"}
	err = repo.Create(ctx, testRepo)
	require.NoError(t, err)

	// Update
	testRepo.Name = "updated-name"
	testRepo.Description = "New description"
	err = repo.Update(ctx, testRepo)
	require.NoError(t, err)

	// Verify
	fetched, err := repo.GetByID(ctx, testRepo.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-name", fetched.Name)
	assert.Equal(t, "New description", fetched.Description)
}

// TestRepoRepository_Delete tests deletion
func TestRepoRepository_Delete(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client, organization, and repo
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "Test Org"}
	orgErr := database.Create(org).Error
	require.NoError(t, orgErr)

	testRepo := &Repo{OrganizationID: org.ID, Name: "to-delete"}
	err = repo.Create(ctx, testRepo)
	require.NoError(t, err)

	// Soft delete
	err = repo.Delete(ctx, testRepo.ID, false)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, testRepo.ID)
	require.ErrorIs(t, err, ErrRecordNotFound)
}

// TestRepoRepository_List tests listing repos for an organization
func TestRepoRepository_List(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client and organizations
	client := &Client{Name: "Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org1 := &Organization{ClientID: client.ID, Name: "Org 1"}
	err = database.Create(org1).Error
	require.NoError(t, err)

	org2 := &Organization{ClientID: client.ID, Name: "Org 2"}
	err = database.Create(org2).Error
	require.NoError(t, err)

	// Create repos
	repos := []*Repo{
		{OrganizationID: org1.ID, Name: "repo-c"},
		{OrganizationID: org1.ID, Name: "repo-a"},
		{OrganizationID: org2.ID, Name: "repo-b"}, // Different org
	}
	for _, r := range repos {
		createErr := repo.Create(ctx, r)
		require.NoError(t, createErr)
	}

	// List for org1
	list, err := repo.List(ctx, org1.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Verify ordering
	assert.Equal(t, "repo-a", list[0].Name)
	assert.Equal(t, "repo-c", list[1].Name)
}

// TestRepoRepository_FindOrCreateFromFullName_Exists tests when repo exists
func TestRepoRepository_FindOrCreateFromFullName_Exists(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client, organization, and repo
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "mrz1836"}
	err = database.Create(org).Error
	require.NoError(t, err)

	existingRepo := &Repo{OrganizationID: org.ID, Name: "existing-repo"}
	err = repo.Create(ctx, existingRepo)
	require.NoError(t, err)

	// FindOrCreate should return existing
	fetched, err := repo.FindOrCreateFromFullName(ctx, "mrz1836/existing-repo", client.ID)
	require.NoError(t, err)
	assert.Equal(t, existingRepo.ID, fetched.ID)
	assert.Equal(t, "existing-repo", fetched.Name)
	assert.NotNil(t, fetched.Organization)
}

// TestRepoRepository_FindOrCreateFromFullName_CreatesRepoOnly tests creating repo when org exists
func TestRepoRepository_FindOrCreateFromFullName_CreatesRepoOnly(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client and organization
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	org := &Organization{ClientID: client.ID, Name: "existing-org"}
	err = database.Create(org).Error
	require.NoError(t, err)

	// FindOrCreate should create only the repo
	fetched, err := repo.FindOrCreateFromFullName(ctx, "existing-org/new-repo", client.ID)
	require.NoError(t, err)
	assert.NotZero(t, fetched.ID)
	assert.Equal(t, "new-repo", fetched.Name)
	assert.Equal(t, org.ID, fetched.Organization.ID)
}

// TestRepoRepository_FindOrCreateFromFullName_CreatesOrgAndRepo tests cascade creation
func TestRepoRepository_FindOrCreateFromFullName_CreatesOrgAndRepo(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// Create client
	client := &Client{Name: "Test Client"}
	err := database.Create(client).Error
	require.NoError(t, err)

	// FindOrCreate should create both org and repo
	fetched, err := repo.FindOrCreateFromFullName(ctx, "new-org/new-repo", client.ID)
	require.NoError(t, err)
	assert.NotZero(t, fetched.ID)
	assert.Equal(t, "new-repo", fetched.Name)
	assert.NotNil(t, fetched.Organization)
	assert.Equal(t, "new-org", fetched.Organization.Name)
	assert.Equal(t, client.ID, fetched.Organization.ClientID)
}

// TestRepoRepository_FindOrCreateFromFullName_CreatesClient tests creating client when needed
func TestRepoRepository_FindOrCreateFromFullName_CreatesClient(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	// No client exists, pass invalid client ID
	fetched, err := repo.FindOrCreateFromFullName(ctx, "brand-new-org/brand-new-repo", 99999)
	require.NoError(t, err)
	assert.NotZero(t, fetched.ID)
	assert.Equal(t, "brand-new-repo", fetched.Name)
	assert.NotNil(t, fetched.Organization)
	assert.Equal(t, "brand-new-org", fetched.Organization.Name)

	// Verify client was created with org name
	var client Client
	err = database.Where("id = ?", fetched.Organization.ClientID).First(&client).Error
	require.NoError(t, err)
	assert.Equal(t, "brand-new-org", client.Name)
}

// TestRepoRepository_FindOrCreateFromFullName_InvalidFormat tests error handling
func TestRepoRepository_FindOrCreateFromFullName_InvalidFormat(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	ctx := context.Background()
	repo := NewRepoRepository(database)

	testCases := []struct {
		name     string
		fullName string
	}{
		{
			name:     "missing slash",
			fullName: "org-repo",
		},
		{
			name:     "empty org",
			fullName: "/repo",
		},
		{
			name:     "empty repo",
			fullName: "org/",
		},
		{
			name:     "empty string",
			fullName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.FindOrCreateFromFullName(ctx, tc.fullName, 1)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidRepoFormat)
		})
	}
}
