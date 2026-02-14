package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// setupTestDB creates a test database with seed data and sets the global dbPath.
// Returns a cleanup function that restores the original dbPath.
func setupTestDB(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test-crud.db")

	oldDBPath := dbPath
	dbPath = tmpPath

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)

	// Seed data manually (we can't use TestDBWithSeed because that returns in-memory)
	gormDB := database.DB()
	config := &db.Config{ExternalID: "test-config", Name: "Test", Version: 1}
	require.NoError(t, gormDB.Create(config).Error)

	// Client -> Org -> Repos
	client := &db.Client{Name: "mrz1836"}
	require.NoError(t, gormDB.Create(client).Error)
	org := &db.Organization{ClientID: client.ID, Name: "mrz1836"}
	require.NoError(t, gormDB.Create(org).Error)
	sourceRepo := &db.Repo{OrganizationID: org.ID, Name: "go-broadcast"}
	require.NoError(t, gormDB.Create(sourceRepo).Error)
	targetRepo1 := &db.Repo{OrganizationID: org.ID, Name: "test-repo-1"}
	require.NoError(t, gormDB.Create(targetRepo1).Error)
	targetRepo2 := &db.Repo{OrganizationID: org.ID, Name: "test-repo-2"}
	require.NoError(t, gormDB.Create(targetRepo2).Error)

	// Group
	enabled := true
	group := &db.Group{
		ConfigID: config.ID, ExternalID: "mrz-tools", Name: "MrZ Tools",
		Enabled: &enabled, Position: 0,
	}
	require.NoError(t, gormDB.Create(group).Error)

	// Source
	source := &db.Source{GroupID: group.ID, RepoID: sourceRepo.ID, Branch: "master"}
	require.NoError(t, gormDB.Create(source).Error)

	// GroupGlobal + GroupDefault
	require.NoError(t, gormDB.Create(&db.GroupGlobal{GroupID: group.ID}).Error)
	require.NoError(t, gormDB.Create(&db.GroupDefault{GroupID: group.ID}).Error)

	// Targets
	target1 := &db.Target{GroupID: group.ID, RepoID: targetRepo1.ID, Branch: "main", Position: 0}
	require.NoError(t, gormDB.Create(target1).Error)
	target2 := &db.Target{GroupID: group.ID, RepoID: targetRepo2.ID, Branch: "develop", Position: 1}
	require.NoError(t, gormDB.Create(target2).Error)

	// File mappings on target1
	fm := &db.FileMapping{OwnerType: "target", OwnerID: target1.ID, Src: ".editorconfig", Dest: ".editorconfig", Position: 0}
	require.NoError(t, gormDB.Create(fm).Error)

	// File lists
	fl := &db.FileList{ConfigID: config.ID, ExternalID: "ai-files", Name: "AI Files", Position: 0}
	require.NoError(t, gormDB.Create(fl).Error)
	flFile := &db.FileMapping{OwnerType: "file_list", OwnerID: fl.ID, Src: ".cursorrules", Dest: ".cursorrules", Position: 0}
	require.NoError(t, gormDB.Create(flFile).Error)

	// Directory lists
	dl := &db.DirectoryList{ConfigID: config.ID, ExternalID: "github-workflows", Name: "Workflows", Position: 0}
	require.NoError(t, gormDB.Create(dl).Error)

	// Refs
	require.NoError(t, gormDB.Create(&db.TargetFileListRef{TargetID: target1.ID, FileListID: fl.ID, Position: 0}).Error)
	require.NoError(t, gormDB.Create(&db.TargetDirectoryListRef{TargetID: target2.ID, DirectoryListID: dl.ID, Position: 0}).Error)

	require.NoError(t, database.Close())

	return func() {
		dbPath = oldDBPath
	}
}

// captureJSON captures JSON output by temporarily redirecting stdout
func captureJSON(t *testing.T, fn func() error) (CLIResponse, error) {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	oldStdout := os.Stdout
	os.Stdout = w

	runErr := fn()

	os.Stdout = oldStdout
	require.NoError(t, w.Close())

	buf, readErr := io.ReadAll(r)
	require.NoError(t, r.Close())
	require.NoError(t, readErr)

	var resp CLIResponse
	if len(buf) > 0 {
		require.NoError(t, json.Unmarshal(buf, &resp), "failed to parse JSON: %s", string(buf))
	}
	return resp, runErr
}

// ====================
// Group CRUD tests
// ====================

func TestGroupList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("json output", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupList(true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "listed", resp.Action)
		assert.Equal(t, "group", resp.Type)
		assert.Equal(t, 1, resp.Count)
	})

	t.Run("human output", func(t *testing.T) {
		err := runGroupList(false)
		require.NoError(t, err)
	})
}

func TestGroupGet(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("existing group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupGet("mrz-tools", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "get", resp.Action)
	})

	t.Run("non-existent group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupGet("nonexistent", true)
		})
		require.NoError(t, err) // JSON mode returns via stdout, not error
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Error, "not found")
		assert.NotEmpty(t, resp.Hint)
	})

	t.Run("human output", func(t *testing.T) {
		err := runGroupGet("mrz-tools", false)
		require.NoError(t, err)
	})
}

func TestGroupCreate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("create new group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupCreate("new-group", "New Group", "mrz1836/go-broadcast", "main", "A new group", 5, false, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "created", resp.Action)
	})

	t.Run("duplicate group fails", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupCreate("mrz-tools", "Dup", "mrz1836/go-broadcast", "main", "", 0, false, true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Error, "already exists")
	})

	t.Run("create disabled group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupCreate("disabled-grp", "Disabled Group", "mrz1836/go-broadcast", "main", "", 0, true, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})
}

func TestGroupDelete(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("soft delete", func(t *testing.T) {
		// Create first, then delete
		_, err := captureJSON(t, func() error {
			return runGroupCreate("del-test", "Delete Test", "mrz1836/go-broadcast", "main", "", 0, false, true)
		})
		require.NoError(t, err)

		resp, err := captureJSON(t, func() error {
			return runGroupDelete("del-test", false, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "soft-deleted", resp.Action)
	})

	t.Run("hard delete", func(t *testing.T) {
		// Hard delete of a group that has Source/Global/Default associations
		// will fail due to FK constraints, so we soft-delete first which is
		// the supported workflow. Test that hard-delete with FK error gives
		// a clear error response.
		createResp, err := captureJSON(t, func() error {
			return runGroupCreate("del-hard", "Hard Delete", "mrz1836/go-broadcast", "main", "", 0, false, true)
		})
		require.NoError(t, err)
		require.True(t, createResp.Success, "create failed: %s (hint: %s)", createResp.Error, createResp.Hint)

		// Hard-delete fails because of FK constraints (expected behavior)
		resp, err := captureJSON(t, func() error {
			return runGroupDelete("del-hard", true, true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success) // FK constraint prevents hard delete of group with associations
		assert.Contains(t, resp.Error, "constraint")

		// But soft delete works
		resp, err = captureJSON(t, func() error {
			return runGroupDelete("del-hard", false, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "soft-deleted", resp.Action)
	})

	t.Run("nonexistent group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupDelete("nonexistent", false, true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})
}

func TestGroupEnableDisable(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("disable group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupSetEnabled("mrz-tools", false, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "disabled", resp.Action)
	})

	t.Run("enable group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupSetEnabled("mrz-tools", true, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "enabled", resp.Action)
	})

	t.Run("nonexistent group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupSetEnabled("nonexistent", true, true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})
}

func TestGroupUpdate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("update name and priority", func(t *testing.T) {
		// We need to use Cobra's flag tracking, so let's call runGroupUpdate
		// via the actual command
		cmd := newDBGroupUpdateCmd()
		cmd.SetArgs([]string{"mrz-tools", "--name", "Updated Name", "--priority", "10", "--json"})
		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("nonexistent group", func(t *testing.T) {
		cmd := newDBGroupUpdateCmd()
		cmd.SetArgs([]string{"nonexistent", "--name", "X", "--json"})
		// This should output error JSON but not return error (JSON mode)
		err := cmd.Execute()
		require.NoError(t, err)
	})
}

// ====================
// Target CRUD tests
// ====================

func TestTargetList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("json output", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetList("mrz-tools", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 2, resp.Count)
	})

	t.Run("human output", func(t *testing.T) {
		err := runTargetList("mrz-tools", false)
		require.NoError(t, err)
	})

	t.Run("nonexistent group", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetList("nonexistent", true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})
}

func TestTargetAdd(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("add new target", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetAdd("mrz-tools", "mrz1836/new-repo", "main", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "created", resp.Action)
	})

	t.Run("idempotent add", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetAdd("mrz-tools", "mrz1836/test-repo-1", "main", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "already_exists", resp.Action)
	})
}

func TestTargetRemove(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("remove existing target", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetRemove("mrz-tools", "mrz1836/test-repo-2", false, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("remove nonexistent target", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetRemove("mrz-tools", "mrz1836/nonexistent", false, true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})
}

func TestTargetUpdate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("update branch and labels", func(t *testing.T) {
		cmd := newDBTargetUpdateCmd()
		cmd.SetArgs([]string{
			"--group", "mrz-tools",
			"--repo", "mrz1836/test-repo-1",
			"--branch", "new-branch",
			"--pr-labels", "label1,label2",
			"--json",
		})
		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("nonexistent target", func(t *testing.T) {
		cmd := newDBTargetUpdateCmd()
		cmd.SetArgs([]string{
			"--group", "mrz-tools",
			"--repo", "mrz1836/nonexistent",
			"--branch", "x",
			"--json",
		})
		err := cmd.Execute()
		require.NoError(t, err) // JSON mode
	})
}

func TestTargetGet(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("existing target", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetGet("mrz-tools", "mrz1836/test-repo-1", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "get", resp.Action)
	})

	t.Run("nonexistent target", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetGet("mrz-tools", "mrz1836/nonexistent", true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})
}

// ====================
// File List CRUD tests
// ====================

func TestFileListList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("json output", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListList(true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, 1, resp.Count)
	})
}

func TestFileListGet(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("existing", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListGet("ai-files", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("nonexistent", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListGet("nonexistent", true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})
}

func TestFileListCreate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("create new", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListCreate("security-files", "Security Files", "Security related files", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "created", resp.Action)
	})

	t.Run("duplicate fails", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListCreate("ai-files", "Dup", "", true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Error, "already exists")
	})
}

func TestFileListDelete(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create then delete
	_, err := captureJSON(t, func() error {
		return runFileListCreate("to-delete", "To Delete", "", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runFileListDelete("to-delete", false, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestFileListAddFile(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("add file", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListAddFile("ai-files", "SECURITY.md", "SECURITY.md", false, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "created", resp.Action)
	})

	t.Run("duplicate dest fails", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListAddFile("ai-files", "x", ".cursorrules", false, true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Error, "already exists")
	})
}

func TestFileListRemoveFile(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("remove existing", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListRemoveFile("ai-files", ".cursorrules", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("remove nonexistent", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runFileListRemoveFile("ai-files", "nonexistent.txt", true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})
}

// ====================
// Directory List CRUD tests
// ====================

func TestDirListList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runDirListList(true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Count)
}

func TestDirListGet(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("existing", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runDirListGet("github-workflows", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})
}

func TestDirListCreate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runDirListCreate("new-dirs", "New Dirs", "A directory list", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "created", resp.Action)
}

func TestDirListDelete(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, err := captureJSON(t, func() error {
		return runDirListCreate("to-del-dl", "To Del", "", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runDirListDelete("to-del-dl", true, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestDirListAddDir(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runDirListAddDir("github-workflows", ".ci", ".ci", "", "", true, false, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestDirListRemoveDir(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Add then remove
	_, err := captureJSON(t, func() error {
		return runDirListAddDir("github-workflows", "tmp", "tmp", "", "", true, false, true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runDirListRemoveDir("github-workflows", "tmp", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

// ====================
// Inline file mapping tests
// ====================

func TestFileAdd(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runFileAdd("mrz-tools", "mrz1836/test-repo-1", "README.md", "README.md", false, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestFileRemove(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runFileRemove("mrz-tools", "mrz1836/test-repo-1", ".editorconfig", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestFileListMappings(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runFileListMappings("mrz-tools", "mrz1836/test-repo-1", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Count) // .editorconfig
}

// ====================
// Inline dir mapping tests
// ====================

func TestDirAdd(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runDirAdd("mrz-tools", "mrz1836/test-repo-1", ".github", ".github", "", "", true, false, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestDirRemove(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Add then remove
	_, err := captureJSON(t, func() error {
		return runDirAdd("mrz-tools", "mrz1836/test-repo-1", "tmp", "tmp", "", "", true, false, true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runDirRemove("mrz-tools", "mrz1836/test-repo-1", "tmp", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestDirListMappings(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runDirListMappings("mrz-tools", "mrz1836/test-repo-1", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 0, resp.Count) // no inline dir mappings on target1
}

// ====================
// Ref tests
// ====================

func TestRefAddFileList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("add ref to target without it", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runRefAddFileList("mrz-tools", "mrz1836/test-repo-2", "ai-files", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "attached", resp.Action)
	})

	t.Run("idempotent add", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runRefAddFileList("mrz-tools", "mrz1836/test-repo-1", "ai-files", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "already_attached", resp.Action)
	})
}

func TestRefRemoveFileList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runRefRemoveFileList("mrz-tools", "mrz1836/test-repo-1", "ai-files", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "detached", resp.Action)
}

func TestRefAddDirList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("add ref", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runRefAddDirList("mrz-tools", "mrz1836/test-repo-1", "github-workflows", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "attached", resp.Action)
	})

	t.Run("idempotent add", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runRefAddDirList("mrz-tools", "mrz1836/test-repo-2", "github-workflows", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "already_attached", resp.Action)
	})
}

func TestRefRemoveDirList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runRefRemoveDirList("mrz-tools", "mrz1836/test-repo-2", "github-workflows", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "detached", resp.Action)
}

// ====================
// Bulk operation tests
// ====================

func TestBulkAddFileList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a new file list first
	_, err := captureJSON(t, func() error {
		return runFileListCreate("bulk-fl", "Bulk FL", "", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runBulkAddFileList("mrz-tools", "bulk-fl", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "bulk-attached", resp.Action)
	assert.Equal(t, 2, resp.Count) // Both targets
}

func TestBulkRemoveFileList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Add to all first
	_, err := captureJSON(t, func() error {
		return runFileListCreate("bulk-rm-fl", "Bulk RM FL", "", true)
	})
	require.NoError(t, err)
	_, err = captureJSON(t, func() error {
		return runBulkAddFileList("mrz-tools", "bulk-rm-fl", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runBulkRemoveFileList("mrz-tools", "bulk-rm-fl", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "bulk-detached", resp.Action)
	assert.Equal(t, 2, resp.Count)
}

func TestBulkAddDirList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, err := captureJSON(t, func() error {
		return runDirListCreate("bulk-dl", "Bulk DL", "", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runBulkAddDirList("mrz-tools", "bulk-dl", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 2, resp.Count)
}

func TestBulkRemoveDirList(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, err := captureJSON(t, func() error {
		return runDirListCreate("bulk-rm-dl", "Bulk RM DL", "", true)
	})
	require.NoError(t, err)
	_, err = captureJSON(t, func() error {
		return runBulkAddDirList("mrz-tools", "bulk-rm-dl", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runBulkRemoveDirList("mrz-tools", "bulk-rm-dl", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 2, resp.Count)
}

// ====================
// Response envelope tests
// ====================

func TestCLIResponse_JSON_Structure(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runGroupList(true)
	})
	require.NoError(t, err)

	// Verify all fields present in successful response
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Action)
	assert.NotEmpty(t, resp.Type)
	assert.NotNil(t, resp.Data)
	assert.Empty(t, resp.Error)
}

func TestCLIResponse_Error_Structure(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runGroupGet("nonexistent", true)
	})
	require.NoError(t, err)

	assert.False(t, resp.Success)
	assert.NotEmpty(t, resp.Error)
	assert.NotEmpty(t, resp.Hint)
}

// ====================
// Helpers tests
// ====================

func TestSplitCSV(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := splitCSV("")
		assert.Nil(t, result)
	})

	t.Run("single", func(t *testing.T) {
		result := splitCSV("label1")
		assert.Equal(t, db.JSONStringSlice{"label1"}, result)
	})

	t.Run("multiple with spaces", func(t *testing.T) {
		result := splitCSV("label1, label2, label3")
		assert.Equal(t, db.JSONStringSlice{"label1", "label2", "label3"}, result)
	})

	t.Run("empty values filtered", func(t *testing.T) {
		result := splitCSV("label1,,label2, ,")
		assert.Equal(t, db.JSONStringSlice{"label1", "label2"}, result)
	})
}

func TestPrintErrorResponse_Human(t *testing.T) {
	err := printErrorResponse("group", "created", "something failed", "try again", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "something failed")
	assert.Contains(t, err.Error(), "try again")
}

func TestPrintErrorResponse_Human_NoHint(t *testing.T) {
	err := printErrorResponse("group", "created", "something failed", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "something failed")
	assert.NotContains(t, err.Error(), "hint")
}

func TestPrintResponse_Human(t *testing.T) {
	resp := CLIResponse{
		Success: true,
		Action:  "created",
		Type:    "group",
		Data:    map[string]string{"id": "test"},
	}
	err := printResponse(resp, false)
	require.NoError(t, err)
}

// ====================
// Database not found tests
// ====================

func TestGroupList_NoDB(t *testing.T) {
	oldDBPath := dbPath
	dbPath = "/tmp/nonexistent-path-for-test/db.sqlite"
	defer func() { dbPath = oldDBPath }()

	err := runGroupList(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database does not exist")
}

// ====================
// Integration workflow test
// ====================

func TestFullWorkflow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 1. List groups
	resp, err := captureJSON(t, func() error {
		return runGroupList(true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 1, resp.Count)

	// 2. Create a new file list
	resp, err = captureJSON(t, func() error {
		return runFileListCreate("security-files", "Security Files", "Security-related config", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// 3. Add files to the file list
	resp, err = captureJSON(t, func() error {
		return runFileListAddFile("security-files", "SECURITY.md", "SECURITY.md", false, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// 4. Add a new target
	resp, err = captureJSON(t, func() error {
		return runTargetAdd("mrz-tools", "mrz1836/new-project", "main", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// 5. Bulk add file list to all targets
	resp, err = captureJSON(t, func() error {
		return runBulkAddFileList("mrz-tools", "security-files", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 3, resp.Count) // 2 original + 1 new

	// 6. Add an inline file to the new target
	resp, err = captureJSON(t, func() error {
		return runFileAdd("mrz-tools", "mrz1836/new-project", "LICENSE", "LICENSE", false, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// 7. Verify target get shows everything
	resp, err = captureJSON(t, func() error {
		return runTargetGet("mrz-tools", "mrz1836/new-project", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)

	// 8. Disable the group
	resp, err = captureJSON(t, func() error {
		return runGroupSetEnabled("mrz-tools", false, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "disabled", resp.Action)

	// 9. Re-enable the group
	resp, err = captureJSON(t, func() error {
		return runGroupSetEnabled("mrz-tools", true, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "enabled", resp.Action)
}
