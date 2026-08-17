package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// missingDBPath points the global dbPath at a nonexistent file so openDatabase
// fails, exercising the printErrorResponse / error branches of CRUD commands.
func missingDBPath(t *testing.T) func() {
	t.Helper()
	old := dbPath
	dbPath = filepath.Join(t.TempDir(), "nonexistent.db")
	return func() { dbPath = old }
}

// TestDBQuery_AllModesJSON runs every db query mode in both JSON and human
// output against the rich setupTestDB seed to exercise the detail-printing and
// JSON branches of the queryBy* helpers.
func TestDBQuery_AllModesJSON(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	cleanup := setupTestDB(t)
	defer cleanup()

	oldFile, oldRepo, oldFL, oldContains, oldJSON := dbQueryFile, dbQueryRepo, dbQueryFileList, dbQueryContains, dbQueryJSON
	t.Cleanup(func() {
		dbQueryFile, dbQueryRepo, dbQueryFileList, dbQueryContains, dbQueryJSON = oldFile, oldRepo, oldFL, oldContains, oldJSON
	})

	reset := func() { dbQueryFile, dbQueryRepo, dbQueryFileList, dbQueryContains = "", "", "", "" }

	run := func(jsonMode bool) {
		dbQueryJSON = jsonMode
		reset()
		dbQueryFile = ".editorconfig"
		require.NoError(t, runDBQuery(nil, nil))

		reset()
		dbQueryRepo = "acme/test-repo-1"
		require.NoError(t, runDBQuery(nil, nil))

		reset()
		dbQueryFileList = "ai-files"
		require.NoError(t, runDBQuery(nil, nil))

		reset()
		dbQueryContains = "cursorrules"
		require.NoError(t, runDBQuery(nil, nil))
	}

	t.Run("human mode", func(t *testing.T) { run(false) })
	t.Run("json mode", func(t *testing.T) { run(true) })
}

// TestDirListCRUD_JSONAndErrors exercises JSON-mode success paths plus the
// not-found and no-database error branches for the directory-list commands.
func TestDirListCRUD_JSONAndErrors(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("list json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runDirListList(context.Background(), true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "directory_list", resp.Type)
	})

	t.Run("list human", func(t *testing.T) {
		require.NoError(t, runDirListList(context.Background(), false))
	})

	t.Run("get json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runDirListGet(context.Background(), "github-workflows", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("get human", func(t *testing.T) {
		require.NoError(t, runDirListGet(context.Background(), "github-workflows", false))
	})

	t.Run("get not found json", func(t *testing.T) {
		_, err := captureJSON(t, func() error { return runDirListGet(context.Background(), "nope", true) })
		require.NoError(t, err) // JSON error responses return nil
	})

	t.Run("create json then add and remove dir", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runDirListCreate(context.Background(), "new-dl", "New DL", "desc", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)

		_, err = captureJSON(t, func() error {
			return runDirListAddDir(context.Background(), "new-dl", "src/", "dst/", "", "", false, false, true)
		})
		require.NoError(t, err)

		_, err = captureJSON(t, func() error { return runDirListRemoveDir(context.Background(), "new-dl", "dst/", true) })
		require.NoError(t, err)

		_, err = captureJSON(t, func() error { return runDirListDelete(context.Background(), "new-dl", false, true) })
		require.NoError(t, err)
	})

	t.Run("create duplicate errors (human)", func(t *testing.T) {
		// "github-workflows" already exists in the seed.
		err := runDirListCreate(context.Background(), "github-workflows", "dup", "", false)
		require.Error(t, err)
	})

	t.Run("create human + delete human", func(t *testing.T) {
		require.NoError(t, runDirListCreate(context.Background(), "dl-human", "DL Human", "", false))
		require.NoError(t, runDirListAddDir(context.Background(), "dl-human", "s/", "d/", "x", "y", true, false, false))
		require.NoError(t, runDirListRemoveDir(context.Background(), "dl-human", "d/", false))
		require.NoError(t, runDirListDelete(context.Background(), "dl-human", true, false))
	})

	t.Run("create dry-run", func(t *testing.T) {
		withDryRunEnabled(t)
		_, err := captureJSON(t, func() error { return runDirListCreate(context.Background(), "dl-dry", "Dry", "", true) })
		require.NoError(t, err)
	})

	t.Run("no database errors", func(t *testing.T) {
		restore := missingDBPath(t)
		defer restore()
		require.Error(t, runDirListList(context.Background(), false))
		require.Error(t, runDirListGet(context.Background(), "x", false))
		require.Error(t, runDirListCreate(context.Background(), "x", "x", "", false))
		require.Error(t, runDirListDelete(context.Background(), "x", false, false))
		require.Error(t, runDirListAddDir(context.Background(), "x", "a", "b", "", "", false, false, false))
		require.Error(t, runDirListRemoveDir(context.Background(), "x", "b", false))
	})
}

// TestFileListCRUD_JSONAndErrors mirrors the directory-list coverage for file lists.
func TestFileListCRUD_JSONAndErrors(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("list json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runFileListList(context.Background(), true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("list human", func(t *testing.T) {
		require.NoError(t, runFileListList(context.Background(), false))
	})

	t.Run("get json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runFileListGet(context.Background(), "ai-files", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("get human", func(t *testing.T) {
		require.NoError(t, runFileListGet(context.Background(), "ai-files", false))
	})

	t.Run("get not found", func(t *testing.T) {
		_, err := captureJSON(t, func() error { return runFileListGet(context.Background(), "nope", true) })
		require.NoError(t, err)
	})

	t.Run("create add remove delete json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runFileListCreate(context.Background(), "new-fl", "New FL", "desc", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)

		_, err = captureJSON(t, func() error { return runFileListAddFile(context.Background(), "new-fl", "a.txt", "a.txt", false, true) })
		require.NoError(t, err)

		_, err = captureJSON(t, func() error { return runFileListRemoveFile(context.Background(), "new-fl", "a.txt", true) })
		require.NoError(t, err)

		_, err = captureJSON(t, func() error { return runFileListDelete(context.Background(), "new-fl", false, true) })
		require.NoError(t, err)
	})

	t.Run("create human + add + remove + delete human", func(t *testing.T) {
		require.NoError(t, runFileListCreate(context.Background(), "fl-human", "FL Human", "", false))
		require.NoError(t, runFileListAddFile(context.Background(), "fl-human", "a.txt", "a.txt", false, false))
		require.NoError(t, runFileListRemoveFile(context.Background(), "fl-human", "a.txt", false))
		require.NoError(t, runFileListDelete(context.Background(), "fl-human", true, false))
	})

	t.Run("create duplicate errors", func(t *testing.T) {
		err := runFileListCreate(context.Background(), "ai-files", "dup", "", false)
		require.Error(t, err)
	})

	t.Run("create dry-run", func(t *testing.T) {
		withDryRunEnabled(t)
		_, err := captureJSON(t, func() error { return runFileListCreate(context.Background(), "fl-dry", "Dry", "", true) })
		require.NoError(t, err)
	})

	t.Run("no database errors", func(t *testing.T) {
		restore := missingDBPath(t)
		defer restore()
		require.Error(t, runFileListList(context.Background(), false))
		require.Error(t, runFileListGet(context.Background(), "x", false))
		require.Error(t, runFileListCreate(context.Background(), "x", "x", "", false))
		require.Error(t, runFileListDelete(context.Background(), "x", false, false))
		require.Error(t, runFileListAddFile(context.Background(), "x", "a", "b", false, false))
		require.Error(t, runFileListRemoveFile(context.Background(), "x", "b", false))
	})
}

// TestFileDirMappingCRUD_JSONAndErrors covers the inline target file/dir mapping commands.
func TestFileDirMappingCRUD_JSONAndErrors(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("file add list remove json", func(t *testing.T) {
		_, err := captureJSON(t, func() error {
			return runFileAdd(context.Background(), "my-tools", "acme/test-repo-1", "x.txt", "x.txt", false, true)
		})
		require.NoError(t, err)

		resp, err := captureJSON(t, func() error { return runFileListMappings(context.Background(), "my-tools", "acme/test-repo-1", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)

		_, err = captureJSON(t, func() error {
			return runFileRemove(context.Background(), "my-tools", "acme/test-repo-1", "x.txt", true)
		})
		require.NoError(t, err)
	})

	t.Run("dir add list remove json", func(t *testing.T) {
		_, err := captureJSON(t, func() error {
			return runDirAdd(context.Background(), "my-tools", "acme/test-repo-1", "s/", "d/", "", "", false, false, true)
		})
		require.NoError(t, err)

		resp, err := captureJSON(t, func() error { return runDirListMappings(context.Background(), "my-tools", "acme/test-repo-1", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)

		_, err = captureJSON(t, func() error { return runDirRemove(context.Background(), "my-tools", "acme/test-repo-1", "d/", true) })
		require.NoError(t, err)
	})

	t.Run("human output", func(t *testing.T) {
		require.NoError(t, runFileListMappings(context.Background(), "my-tools", "acme/test-repo-1", false))
		require.NoError(t, runDirListMappings(context.Background(), "my-tools", "acme/test-repo-1", false))
	})

	t.Run("add human + remove human", func(t *testing.T) {
		require.NoError(t, runFileAdd(context.Background(), "my-tools", "acme/test-repo-1", "h.txt", "h.txt", false, false))
		require.NoError(t, runFileRemove(context.Background(), "my-tools", "acme/test-repo-1", "h.txt", false))
		require.NoError(t, runDirAdd(context.Background(), "my-tools", "acme/test-repo-1", "hs/", "hd/", "", "", false, false, false))
		require.NoError(t, runDirRemove(context.Background(), "my-tools", "acme/test-repo-1", "hd/", false))
	})

	t.Run("add dry-run", func(t *testing.T) {
		withDryRunEnabled(t)
		_, err := captureJSON(t, func() error {
			return runFileAdd(context.Background(), "my-tools", "acme/test-repo-1", "dry.txt", "dry.txt", false, true)
		})
		require.NoError(t, err)
		_, err = captureJSON(t, func() error {
			return runDirAdd(context.Background(), "my-tools", "acme/test-repo-1", "ds/", "dd/", "", "", false, false, true)
		})
		require.NoError(t, err)
	})

	t.Run("no database errors", func(t *testing.T) {
		restore := missingDBPath(t)
		defer restore()
		require.Error(t, runFileAdd(context.Background(), "g", "o/r", "a", "b", false, false))
		require.Error(t, runFileRemove(context.Background(), "g", "o/r", "b", false))
		require.Error(t, runFileListMappings(context.Background(), "g", "o/r", false))
		require.Error(t, runDirAdd(context.Background(), "g", "o/r", "a", "b", "", "", false, false, false))
		require.Error(t, runDirRemove(context.Background(), "g", "o/r", "b", false))
		require.Error(t, runDirListMappings(context.Background(), "g", "o/r", false))
	})
}

// TestTargetGroupRead_JSON covers target/group read commands in JSON mode.
func TestTargetGroupRead_JSON(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("target list json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runTargetList(context.Background(), "my-tools", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("target get json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runTargetGet(context.Background(), "my-tools", "acme/test-repo-1", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("target get human", func(t *testing.T) {
		require.NoError(t, runTargetGet(context.Background(), "my-tools", "acme/test-repo-1", false))
	})

	t.Run("group get json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error { return runGroupGet(context.Background(), "my-tools", true) })
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("not found paths", func(t *testing.T) {
		_, err := captureJSON(t, func() error { return runTargetGet(context.Background(), "my-tools", "acme/ghost", true) })
		require.NoError(t, err)
		_, err = captureJSON(t, func() error { return runGroupGet(context.Background(), "ghost", true) })
		require.NoError(t, err)
	})

	t.Run("no database errors", func(t *testing.T) {
		restore := missingDBPath(t)
		defer restore()
		require.Error(t, runTargetList(context.Background(), "g", false))
		require.Error(t, runTargetGet(context.Background(), "g", "o/r", false))
		require.Error(t, runGroupGet(context.Background(), "g", false))
		require.Error(t, runGroupList(context.Background(), false))
	})
}

// TestGroupTargetWrite_JSON covers group/target create, update, clone, remove.
func TestGroupTargetWrite_JSON(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("group create json", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runGroupCreate(context.Background(), "grp-new", "Group New", "acme/go-broadcast", "master", "desc", 5, false, true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("group update json", func(t *testing.T) {
		cmd := newDBGroupUpdateCmd()
		cmd.SetContext(context.Background())
		require.NoError(t, cmd.Flags().Set("name", "Renamed"))
		_, err := captureJSON(t, func() error {
			return runGroupUpdate(cmd, "my-tools", "Renamed", "new desc", 9, false, false, true)
		})
		require.NoError(t, err)
	})

	t.Run("target update json", func(t *testing.T) {
		cmd := newDBTargetUpdateCmd()
		cmd.SetContext(context.Background())
		require.NoError(t, cmd.Flags().Set("branch", "release"))
		_, err := captureJSON(t, func() error {
			return runTargetUpdate(cmd, "my-tools", "acme/test-repo-1", "release", "", "", "", "", "", true)
		})
		require.NoError(t, err)
	})

	t.Run("target clone json", func(t *testing.T) {
		cmd := newDBTargetCloneCmd()
		cmd.SetContext(context.Background())
		_, err := captureJSON(t, func() error {
			return runTargetClone(cmd, "my-tools", "acme/test-repo-1", "acme/cloned-repo", "", "", "", "", "", "", "", true)
		})
		require.NoError(t, err)
	})

	t.Run("target clone human with overrides", func(t *testing.T) {
		cmd := newDBTargetCloneCmd()
		cmd.SetContext(context.Background())
		err := runTargetClone(cmd, "my-tools", "acme/test-repo-1", "acme/cloned-repo-2",
			"release", "lbl1,lbl2", "asg1", "rev1", "team1", "", "", false)
		require.NoError(t, err)
	})

	t.Run("target clone duplicate destination errors", func(t *testing.T) {
		cmd := newDBTargetCloneCmd()
		cmd.SetContext(context.Background())
		// acme/cloned-repo-2 was just created above.
		err := runTargetClone(cmd, "my-tools", "acme/test-repo-1", "acme/cloned-repo-2", "", "", "", "", "", "", "", false)
		require.Error(t, err)
	})

	t.Run("target clone missing source errors", func(t *testing.T) {
		cmd := newDBTargetCloneCmd()
		cmd.SetContext(context.Background())
		err := runTargetClone(cmd, "my-tools", "acme/ghost-source", "acme/new-dest", "", "", "", "", "", "", "", false)
		require.Error(t, err)
	})

	t.Run("target update human", func(t *testing.T) {
		cmd := newDBTargetUpdateCmd()
		cmd.SetContext(context.Background())
		require.NoError(t, cmd.Flags().Set("branch", "stable"))
		require.NoError(t, runTargetUpdate(cmd, "my-tools", "acme/test-repo-1", "stable", "", "", "", "", "", false))
	})

	t.Run("target remove json", func(t *testing.T) {
		_, err := captureJSON(t, func() error {
			return runTargetRemove(context.Background(), "my-tools", "acme/test-repo-2", false, true)
		})
		require.NoError(t, err)
	})

	t.Run("group delete json", func(t *testing.T) {
		_, err := captureJSON(t, func() error { return runGroupDelete(context.Background(), "grp-new", false, true) })
		require.NoError(t, err)
	})

	t.Run("dry-run create/update/clone/remove", func(t *testing.T) {
		withDryRunEnabled(t)
		_, err := captureJSON(t, func() error {
			return runGroupCreate(context.Background(), "grp-dry", "Dry", "acme/go-broadcast", "master", "", 1, false, true)
		})
		require.NoError(t, err)

		cmdU := newDBGroupUpdateCmd()
		cmdU.SetContext(context.Background())
		require.NoError(t, cmdU.Flags().Set("name", "X"))
		_, err = captureJSON(t, func() error { return runGroupUpdate(cmdU, "my-tools", "X", "", 0, false, false, true) })
		require.NoError(t, err)

		cmdC := newDBTargetCloneCmd()
		cmdC.SetContext(context.Background())
		_, err = captureJSON(t, func() error {
			return runTargetClone(cmdC, "my-tools", "acme/test-repo-1", "acme/dry-clone", "", "", "", "", "", "", "", true)
		})
		require.NoError(t, err)

		_, err = captureJSON(t, func() error {
			return runTargetRemove(context.Background(), "my-tools", "acme/test-repo-1", false, true)
		})
		require.NoError(t, err)

		_, err = captureJSON(t, func() error { return runGroupDelete(context.Background(), "my-tools", false, true) })
		require.NoError(t, err)
	})

	t.Run("no database errors", func(t *testing.T) {
		restore := missingDBPath(t)
		defer restore()
		require.Error(t, runGroupCreate(context.Background(), "x", "X", "o/r", "main", "", 0, false, false))
		require.Error(t, runGroupDelete(context.Background(), "x", false, false))
		require.Error(t, runTargetRemove(context.Background(), "g", "o/r", false, false))

		cmd := newDBTargetCloneCmd()
		cmd.SetContext(context.Background())
		require.Error(t, runTargetClone(cmd, "g", "o/a", "o/b", "", "", "", "", "", "", "", false))
	})
}

// TestRefCRUD_JSON covers the file-list / dir-list ref attach/detach commands.
func TestRefCRUD_JSON(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("attach and detach file list ref", func(t *testing.T) {
		// test-repo-2 has no file-list ref yet.
		_, err := captureJSON(t, func() error {
			return runRefAddFileList(context.Background(), "my-tools", "acme/test-repo-2", "ai-files", true)
		})
		require.NoError(t, err)

		_, err = captureJSON(t, func() error {
			return runRefRemoveFileList(context.Background(), "my-tools", "acme/test-repo-2", "ai-files", true)
		})
		require.NoError(t, err)
	})

	t.Run("attach and detach dir list ref", func(t *testing.T) {
		// test-repo-1 has no dir-list ref yet.
		_, err := captureJSON(t, func() error {
			return runRefAddDirList(context.Background(), "my-tools", "acme/test-repo-1", "github-workflows", true)
		})
		require.NoError(t, err)

		_, err = captureJSON(t, func() error {
			return runRefRemoveDirList(context.Background(), "my-tools", "acme/test-repo-1", "github-workflows", true)
		})
		require.NoError(t, err)
	})

	t.Run("dry-run attach/detach", func(t *testing.T) {
		withDryRunEnabled(t)
		_, err := captureJSON(t, func() error {
			return runRefAddFileList(context.Background(), "my-tools", "acme/test-repo-2", "ai-files", true)
		})
		require.NoError(t, err)
		_, err = captureJSON(t, func() error {
			return runRefAddDirList(context.Background(), "my-tools", "acme/test-repo-1", "github-workflows", true)
		})
		require.NoError(t, err)
	})

	t.Run("no database errors", func(t *testing.T) {
		restore := missingDBPath(t)
		defer restore()
		require.Error(t, runRefAddFileList(context.Background(), "g", "o/r", "fl", false))
		require.Error(t, runRefRemoveFileList(context.Background(), "g", "o/r", "fl", false))
		require.Error(t, runRefAddDirList(context.Background(), "g", "o/r", "dl", false))
		require.Error(t, runRefRemoveDirList(context.Background(), "g", "o/r", "dl", false))
	})
}
