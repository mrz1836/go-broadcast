package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// blobNode returns the tree entry Git records for a regular file holding content.
func blobNode(filePath string, content []byte) gh.GitTreeNode {
	id, _ := gitBlobID(content, 40)
	return gh.GitTreeNode{Path: filePath, Mode: gitModeFile, Type: gitEntryTypeBlob, SHA: id}
}

// executableNode returns the tree entry Git records for an executable file holding content.
func executableNode(filePath string, content []byte) gh.GitTreeNode {
	node := blobNode(filePath, content)
	node.Mode = gitModeExecutable
	return node
}

// symlinkNode returns the tree entry Git records for a symlink whose target is linkTarget.
func symlinkNode(filePath, linkTarget string) gh.GitTreeNode {
	node := blobNode(filePath, []byte(linkTarget))
	node.Mode = "120000"
	return node
}

// dirNode returns the tree entry Git records for a directory.
func dirNode(dirPath string) gh.GitTreeNode {
	return gh.GitTreeNode{Path: dirPath, Mode: "040000", Type: gitEntryTypeTree, SHA: "4b825dc642cb6eb9a060e54bf8d69288fbee4904"}
}

// submoduleNode returns the tree entry Git records for a submodule.
func submoduleNode(modulePath string) gh.GitTreeNode {
	return gh.GitTreeNode{Path: modulePath, Mode: "160000", Type: "commit", SHA: "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"}
}

// errTargetTreeNotMocked is returned for target-tree requests in tests that exercise
// the per-file GetFile path.
var errTargetTreeNotMocked = errors.New("target tree not provided by this test")

// expectTargetTreeUnavailable answers the target-tree request a target sync makes by
// reporting the tree as unavailable. File checks then fall back to GetFile exactly
// as they did before the tree existed, so a test's GetFile expectations still apply.
func expectTargetTreeUnavailable(client *gh.MockClient) {
	client.On("GetGitTree", mock.Anything, mock.Anything, mock.Anything, true).
		Return(nil, errTargetTreeNotMocked).Maybe()
}

// newTreeTestClient returns a mock client whose unexpected calls fail the test
// rather than panic, so lookups made from worker goroutines report cleanly.
func newTreeTestClient(t *testing.T) *gh.MockClient {
	t.Helper()
	client := &gh.MockClient{}
	client.Test(t)
	return client
}

func TestGitBlobIDMatchesGit(t *testing.T) {
	// Expected IDs were produced by `git hash-object --stdin`, using a SHA-256
	// repository (`git init --object-format=sha256`) for the 64-digit cases.
	tests := []struct {
		name     string
		content  []byte
		idLength int
		want     string
	}{
		{name: "sha1 empty", content: []byte{}, idLength: 40, want: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"},
		{name: "sha1 nil is empty", content: nil, idLength: 40, want: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391"},
		{name: "sha1 line", content: []byte("hello\n"), idLength: 40, want: "ce013625030ba8dba906f756967f9e9ca394464a"},
		{
			name:     "sha1 multi-line",
			content:  []byte("package main\n\nfunc main() {}\n"),
			idLength: 40,
			want:     "38dd16da61accb1a8de6ac8709d2e65ef4a51a4a",
		},
		{name: "sha256 empty", content: []byte{}, idLength: 64, want: "473a0f4c3be8a93681a267e3b1e9a7dcda1185436fe141f7749120a303721813"},
		{name: "sha256 line", content: []byte("hello\n"), idLength: 64, want: "2cf8d83d9ee29543b34a87727421fdecb7e3f3a183d337639025de576db9ebb4"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, ok := gitBlobID(tc.content, tc.idLength)
			require.True(t, ok)
			assert.Equal(t, tc.want, id)
		})
	}

	t.Run("unknown object id length", func(t *testing.T) {
		id, ok := gitBlobID([]byte("hello\n"), 12)
		assert.False(t, ok)
		assert.Empty(t, id)
	})
}

func TestIsCanonicalTreePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "README.md", want: true},
		{path: ".github/workflows/ci.yml", want: true},
		{path: ".env.base", want: true},
		{path: "", want: false},
		{path: ".", want: false},
		{path: "..", want: false},
		{path: "/README.md", want: false},
		{path: "./README.md", want: false},
		{path: "../README.md", want: false},
		{path: "docs/../README.md", want: false},
		{path: "docs//README.md", want: false},
		{path: "docs/", want: false},
		{path: `docs\README.md`, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, isCanonicalTreePath(tc.path))
		})
	}
}

func TestTargetTreeExistingContentSettlesFromTree(t *testing.T) {
	ctx := context.Background()
	readme := []byte("# Example\n")
	script := []byte("#!/bin/sh\necho hi\n")

	tree := &gh.GitTree{Tree: []gh.GitTreeNode{
		blobNode("README.md", readme),
		dirNode(".github"),
		executableNode(".github/run.sh", script),
	}}

	tests := []struct {
		name      string
		path      string
		candidate []byte
		wantErr   error
		want      []byte
	}{
		{name: "unchanged regular file", path: "README.md", candidate: readme, want: readme},
		{name: "unchanged executable file", path: ".github/run.sh", candidate: script, want: script},
		{name: "missing file", path: "docs/new.md", candidate: []byte("new\n"), wantErr: gh.ErrFileNotFound},
		{name: "missing file for deletion", path: "docs/old.md", candidate: nil, wantErr: gh.ErrFileNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newTreeTestClient(t)
			client.On("GetGitTree", mock.Anything, "owner/repo", "main", true).Return(tree, nil).Once()

			tt := newTargetTree(client, "owner/repo", "main", nil)
			got, err := tt.existingContent(ctx, tc.path, tc.candidate)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
			assert.Equal(t, targetTreeStats{TreeRequests: 1, RequestsAvoided: 1}, tt.stats())
			client.AssertExpectations(t) // No GetFile expectation: the tree settled the check
		})
	}
}

func TestTargetTreeExistingContentFetchesWhenTreeCannotSettle(t *testing.T) {
	ctx := context.Background()
	readme := []byte("# Example\n")

	upperCaseID := blobNode("UPPER.md", readme)
	upperCaseID.SHA = "E2E9B1D2E1F7A64C1B7F7B3F2C9E3A1A0F9B8C7D"

	tree := &gh.GitTree{Tree: []gh.GitTreeNode{
		blobNode("README.md", readme),
		symlinkNode("link.md", "README.md"),
		dirNode("docs"),
		submoduleNode("vendor/lib"),
		symlinkNode("linked-dir", "docs"),
		{Path: "odd-id.md", Mode: gitModeFile, Type: gitEntryTypeBlob, SHA: "abc123"},
		upperCaseID,
	}}

	tests := []struct {
		name      string
		path      string
		candidate []byte
	}{
		{name: "changed content", path: "README.md", candidate: []byte("# Changed\n")},
		{name: "existing file with no candidate", path: "README.md", candidate: nil},
		{name: "symlink entry", path: "link.md", candidate: []byte("README.md")},
		{name: "directory entry", path: "docs", candidate: []byte("file\n")},
		{name: "submodule entry", path: "vendor/lib", candidate: []byte("file\n")},
		{name: "path below a submodule", path: "vendor/lib/main.go", candidate: []byte("package lib\n")},
		{name: "path below a symlink", path: "linked-dir/guide.md", candidate: []byte("guide\n")},
		{name: "unrecognized object id length", path: "odd-id.md", candidate: []byte("odd\n")},
		{name: "upper-case id with different content", path: "UPPER.md", candidate: []byte("other\n")},
		{name: "non-canonical path", path: "./README.md", candidate: readme},
		{name: "path with doubled slash", path: "docs//guide.md", candidate: []byte("guide\n")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newTreeTestClient(t)
			client.On("GetGitTree", mock.Anything, "owner/repo", "main", true).Return(tree, nil).Maybe()
			client.On("GetFile", mock.Anything, "owner/repo", tc.path, "main").
				Return(&gh.FileContent{Content: []byte("fetched")}, nil).Once()

			tt := newTargetTree(client, "owner/repo", "main", nil)
			got, err := tt.existingContent(ctx, tc.path, tc.candidate)

			require.NoError(t, err)
			assert.Equal(t, []byte("fetched"), got)
			assert.Equal(t, 1, tt.stats().ContentRequests)
			assert.Zero(t, tt.stats().RequestsAvoided)
			client.AssertExpectations(t)
		})
	}
}

func TestTargetTreeMatchesUpperCaseObjectID(t *testing.T) {
	content := []byte("hello\n")
	node := blobNode("README.md", content)
	node.SHA = "CE013625030BA8DBA906F756967F9E9CA394464A"

	client := newTreeTestClient(t)
	client.On("GetGitTree", mock.Anything, "owner/repo", "main", true).
		Return(&gh.GitTree{Tree: []gh.GitTreeNode{node}}, nil).Once()

	tt := newTargetTree(client, "owner/repo", "main", nil)
	got, err := tt.existingContent(context.Background(), "README.md", content)

	require.NoError(t, err)
	assert.Equal(t, content, got)
	client.AssertExpectations(t)
}

func TestTargetTreeFallsBackWhenTreeUnusable(t *testing.T) {
	tests := []struct {
		name    string
		tree    *gh.GitTree
		treeErr error
	}{
		{name: "tree request fails", treeErr: assert.AnError},
		{name: "empty tree response", tree: nil},
		{
			name: "truncated listing",
			tree: &gh.GitTree{Truncated: true, Tree: []gh.GitTreeNode{blobNode("README.md", []byte("same\n"))}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := newTreeTestClient(t)
			client.On("GetGitTree", mock.Anything, "owner/repo", "main", true).Return(tc.tree, tc.treeErr).Once()
			client.On("GetFile", mock.Anything, "owner/repo", "README.md", "main").
				Return(&gh.FileContent{Content: []byte("same\n")}, nil).Once()
			client.On("GetFile", mock.Anything, "owner/repo", "missing.md", "main").
				Return(nil, gh.ErrFileNotFound).Once()

			tt := newTargetTree(client, "owner/repo", "main", nil)

			// Every check is answered by GetFile, exactly as without a tree.
			got, err := tt.existingContent(ctx, "README.md", []byte("same\n"))
			require.NoError(t, err)
			assert.Equal(t, []byte("same\n"), got)

			_, err = tt.existingContent(ctx, "missing.md", []byte("new\n"))
			require.ErrorIs(t, err, gh.ErrFileNotFound)

			// The tree is only requested once, even when it is unusable.
			assert.Equal(t, targetTreeStats{TreeRequests: 1, ContentRequests: 2}, tt.stats())

			_, err = tt.filesUnder(ctx, "docs")
			require.ErrorIs(t, err, ErrTargetTreeUnavailable)

			client.AssertExpectations(t)
		})
	}
}

func TestTargetTreeGetFileErrorIsReturned(t *testing.T) {
	client := newTreeTestClient(t)
	client.On("GetGitTree", mock.Anything, "owner/repo", "main", true).
		Return(&gh.GitTree{Tree: []gh.GitTreeNode{blobNode("README.md", []byte("old\n"))}}, nil).Once()
	client.On("GetFile", mock.Anything, "owner/repo", "README.md", "main").Return(nil, assert.AnError).Once()

	tt := newTargetTree(client, "owner/repo", "main", nil)
	got, err := tt.existingContent(context.Background(), "README.md", []byte("new\n"))

	require.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, got)
	client.AssertExpectations(t)
}

func TestTargetTreeDefaultBranchUsesHEAD(t *testing.T) {
	client := newTreeTestClient(t)
	client.On("GetGitTree", mock.Anything, "owner/repo", defaultBranchRef, true).
		Return(&gh.GitTree{Tree: []gh.GitTreeNode{blobNode("README.md", []byte("old\n"))}}, nil).Once()
	// Changed files are still fetched with the configured (empty) ref, as before.
	client.On("GetFile", mock.Anything, "owner/repo", "README.md", "").
		Return(&gh.FileContent{Content: []byte("old\n")}, nil).Once()

	tt := newTargetTree(client, "owner/repo", "", nil)
	got, err := tt.existingContent(context.Background(), "README.md", []byte("new\n"))

	require.NoError(t, err)
	assert.Equal(t, []byte("old\n"), got)
	client.AssertExpectations(t)
}

func TestTargetTreeFetchesTreeOnceUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	const fileCount = 50

	paths := make([]string, fileCount)
	contents := make([][]byte, fileCount)
	nodes := make([]gh.GitTreeNode, fileCount)
	for i := range fileCount {
		paths[i] = fmt.Sprintf("docs/file-%02d.md", i)
		contents[i] = fmt.Appendf(nil, "content %d\n", i)
		nodes[i] = blobNode(paths[i], contents[i])
	}

	client := newTreeTestClient(t)
	client.On("GetGitTree", mock.Anything, "owner/repo", "main", true).Return(&gh.GitTree{Tree: nodes}, nil).Once()

	tt := newTargetTree(client, "owner/repo", "main", nil)

	var wg sync.WaitGroup
	errs := make([]error, fileCount)
	for i := range fileCount {
		wg.Go(func() {
			_, errs[i] = tt.existingContent(ctx, paths[i], contents[i])
		})
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "lookup %d", i)
	}
	assert.Equal(t, targetTreeStats{TreeRequests: 1, RequestsAvoided: fileCount}, tt.stats())
	client.AssertExpectations(t)
}

func TestTargetTreeFilesUnder(t *testing.T) {
	ctx := context.Background()
	tree := &gh.GitTree{Tree: []gh.GitTreeNode{
		dirNode("docs"),
		blobNode("docs/guide.md", []byte("guide\n")),
		dirNode("docs/api"),
		blobNode("docs/api/endpoints.md", []byte("endpoints\n")),
		symlinkNode("docs/latest.md", "guide.md"),
		submoduleNode("docs/vendor"),
		blobNode("docsite/index.md", []byte("index\n")),
		blobNode("README.md", []byte("readme\n")),
	}}

	want := []string{"docs/api/endpoints.md", "docs/guide.md", "docs/latest.md"}

	tests := []struct {
		name string
		dir  string
		want []string
	}{
		{name: "plain directory", dir: "docs", want: want},
		{name: "trailing slash", dir: "docs/", want: want},
		{name: "leading slash", dir: "/docs", want: want},
		{name: "nested directory", dir: "docs/api", want: []string{"docs/api/endpoints.md"}},
		{name: "missing directory", dir: "missing", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newTreeTestClient(t)
			client.On("GetGitTree", mock.Anything, "owner/repo", "main", true).Return(tree, nil).Once()

			tt := newTargetTree(client, "owner/repo", "main", nil)
			got, err := tt.filesUnder(ctx, tc.dir)

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
			client.AssertExpectations(t)
		})
	}

	roots := []struct {
		name string
		dir  string
	}{
		{name: "empty", dir: ""},
		{name: "dot", dir: "."},
		{name: "slash", dir: "/"},
	}
	for _, tc := range roots {
		t.Run("refuses repository root "+tc.name, func(t *testing.T) {
			client := newTreeTestClient(t) // No expectations: the tree must not be requested

			tt := newTargetTree(client, "owner/repo", "main", nil)
			got, err := tt.filesUnder(ctx, tc.dir)

			require.ErrorIs(t, err, ErrDirectoryDeletionRoot)
			assert.Nil(t, got)
		})
	}
}

func TestTargetTreeServes(t *testing.T) {
	tt := newTargetTree(nil, "owner/repo", "main", nil)

	assert.True(t, tt.serves(config.TargetConfig{Repo: "owner/repo", Branch: "main"}))
	assert.False(t, tt.serves(config.TargetConfig{Repo: "owner/other", Branch: "main"}))
	assert.False(t, tt.serves(config.TargetConfig{Repo: "owner/repo", Branch: "develop"}))
	assert.False(t, tt.serves(config.TargetConfig{Repo: "owner/repo"}))

	var nilTree *targetTree
	assert.False(t, nilTree.serves(config.TargetConfig{Repo: "owner/repo", Branch: "main"}))
}

// TestBatchProcessorUsesTargetTree verifies that only files whose content differs
// are downloaded, while unchanged and new files are settled by the tree.
func TestBatchProcessorUsesTargetTree(t *testing.T) {
	ctx := context.Background()
	sourceDir := t.TempDir()

	files := map[string][]byte{
		"same.txt":    []byte("unchanged text\n"),
		"changed.txt": []byte("new text\n"),
		"new.txt":     []byte("brand new\n"),
		"image.png":   {0x89, 0x50, 0x4e, 0x47, 0x00, 0x01, 0x02, 0x03},
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(sourceDir, name), content, 0o600))
	}

	client := newTreeTestClient(t)
	client.On("GetGitTree", mock.Anything, "owner/target", "develop", true).Return(&gh.GitTree{Tree: []gh.GitTreeNode{
		blobNode("same.txt", files["same.txt"]),
		blobNode("changed.txt", []byte("old text\n")),
		blobNode("image.png", files["image.png"]),
	}}, nil).Once()
	client.On("GetFile", mock.Anything, "owner/target", "changed.txt", "develop").
		Return(&gh.FileContent{Content: []byte("old text\n")}, nil).Once()

	target := config.TargetConfig{Repo: "owner/target", Branch: "develop"}
	engine := &Engine{gh: client} // No transforms are configured, so no chain is needed
	sourceState := &state.SourceState{Repo: "owner/source", Branch: "main", LatestCommit: "abc123"}
	logger := logrus.NewEntry(logrus.New())

	processor := NewBatchProcessor(engine, target, sourceState, logger, 4)
	processor.targetTree = newTargetTree(client, target.Repo, target.Branch, logger)

	jobs := []FileJob{
		NewFileJob("same.txt", "same.txt", config.Transform{}),
		NewFileJob("changed.txt", "changed.txt", config.Transform{}),
		NewFileJob("new.txt", "new.txt", config.Transform{}),
		NewFileJob("image.png", "image.png", config.Transform{}),
	}

	changes, err := processor.ProcessFiles(ctx, sourceDir, jobs)
	require.NoError(t, err)

	byPath := make(map[string]FileChange, len(changes))
	for _, change := range changes {
		byPath[change.Path] = change
	}
	require.Len(t, byPath, 2, "only the changed and the new file should produce changes")

	changed := byPath["changed.txt"]
	assert.False(t, changed.IsNew)
	assert.Equal(t, []byte("new text\n"), changed.Content)
	assert.Equal(t, []byte("old text\n"), changed.OriginalContent, "the diff base must be the real target content")

	created := byPath["new.txt"]
	assert.True(t, created.IsNew)
	assert.Nil(t, created.OriginalContent)

	assert.Equal(t, targetTreeStats{TreeRequests: 1, ContentRequests: 1, RequestsAvoided: 3}, processor.targetTree.stats())
	client.AssertExpectations(t)
}

// TestRepositorySyncExecuteUnchangedTargetMakesOneTreeRequest verifies the common
// case end to end: a target that already matches the source is checked with a
// single tree request, without downloading any file or cloning the target.
func TestRepositorySyncExecuteUnchangedTargetMakesOneTreeRequest(t *testing.T) {
	sourceFiles := map[string]string{
		"README.md":      "# Example\n",
		"LICENSE":        "MIT\n",
		".editorconfig":  "root = true\n",
		"docs/guide.md":  "guide\n",
		"docs/api/v1.md": "v1\n",
	}

	treeNodes := make([]gh.GitTreeNode, 0, 2+len(sourceFiles))
	treeNodes = append(treeNodes, dirNode("docs"), dirNode("docs/api"))
	for filePath, content := range sourceFiles {
		treeNodes = append(treeNodes, blobNode(filePath, []byte(content)))
	}

	ghClient := newTreeTestClient(t)
	ghClient.On("ListBranches", mock.Anything, "org/target").Return([]gh.Branch{}, nil).Once()
	ghClient.On("GetGitTree", mock.Anything, "org/target", "develop", true).
		Return(&gh.GitTree{Tree: treeNodes}, nil).Once()
	// No GetFile expectation: every file is settled by the tree.

	gitClient := &git.MockClient{}
	gitClient.Test(t)
	gitClient.On("Clone", mock.Anything, "https://github.com/org/template.git", mock.Anything, mock.Anything).
		Return(nil).Run(func(args mock.Arguments) {
		destPath := args.String(2)
		for filePath, content := range sourceFiles {
			fullPath := filepath.Join(destPath, filepath.FromSlash(filePath))
			require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o750))
			require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
		}
	}).Once()
	gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil).Once()

	target := config.TargetConfig{
		Repo:   "org/target",
		Branch: "develop",
		Files: []config.FileMapping{
			{Src: "README.md", Dest: "README.md"},
			{Src: "LICENSE", Dest: "LICENSE"},
			{Src: ".editorconfig", Dest: ".editorconfig"},
		},
		Directories: []config.DirectoryMapping{
			{Src: "docs", Dest: "docs"},
		},
	}

	engine := &Engine{
		config:  &config.Config{Groups: []config.Group{{Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"}}}},
		gh:      ghClient,
		git:     gitClient,
		options: DefaultOptions(),
		logger:  logrus.New(),
	}
	repoSync := &RepositorySync{
		engine:      engine,
		target:      target,
		sourceState: &state.SourceState{Repo: "org/template", Branch: "master", LatestCommit: "abc123"},
		logger:      logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, repoSync.Execute(context.Background()))

	assert.Equal(t, targetTreeStats{TreeRequests: 1, RequestsAvoided: len(sourceFiles)}, repoSync.targetTree.stats())
	assert.Equal(t, 1, repoSync.syncMetrics.TotalAPIRequests, "only the tree request should reach GitHub")
	assert.Equal(t, len(sourceFiles)-1, repoSync.syncMetrics.APICallsSaved)
	ghClient.AssertExpectations(t)
	gitClient.AssertExpectations(t) // The target is never cloned and nothing is committed
}

// TestRepositorySyncProcessFileUsesTargetTree verifies file mappings are settled by
// the target tree and that recordTargetTreeStats reports the savings.
func TestRepositorySyncProcessFileUsesTargetTree(t *testing.T) {
	ctx := context.Background()
	sourceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("# Same\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "LICENSE"), []byte("MIT\n"), 0o600))

	client := newTreeTestClient(t)
	client.On("GetGitTree", mock.Anything, "owner/target", defaultBranchRef, true).Return(&gh.GitTree{Tree: []gh.GitTreeNode{
		blobNode("README.md", []byte("# Same\n")),
	}}, nil).Once()

	logger := logrus.NewEntry(logrus.New())
	rs := &RepositorySync{
		engine:      &Engine{gh: client},
		target:      config.TargetConfig{Repo: "owner/target"},
		sourceState: &state.SourceState{Repo: "owner/source"},
		logger:      logger,
		syncMetrics: &PerformanceMetrics{DirectoryMetrics: make(map[string]DirectoryMetrics)},
		targetTree:  newTargetTree(client, "owner/target", "", logger),
	}

	change, err := rs.processFile(ctx, sourceDir, config.FileMapping{Src: "README.md", Dest: "README.md"})
	require.ErrorIs(t, err, internalerrors.ErrTransformNotFound, "unchanged files are reported as skipped")
	assert.Nil(t, change)

	change, err = rs.processFile(ctx, sourceDir, config.FileMapping{Src: "LICENSE", Dest: "LICENSE"})
	require.NoError(t, err)
	require.NotNil(t, change)
	assert.True(t, change.IsNew)
	assert.Equal(t, []byte("MIT\n"), change.OriginalContent, "new files keep using the source content as their diff base")

	rs.recordTargetTreeStats()
	assert.Equal(t, 1, rs.syncMetrics.TotalAPIRequests, "only the tree request was made")
	assert.Equal(t, 1, rs.syncMetrics.APICallsSaved, "two avoided file requests minus the tree request")
	client.AssertExpectations(t)
}

// TestRepositorySyncProcessDirectoriesSharesTargetTree verifies the directory path
// without metrics also settles files from the target sync's tree.
func TestRepositorySyncProcessDirectoriesSharesTargetTree(t *testing.T) {
	tempDir := t.TempDir()
	docsDir := filepath.Join(tempDir, "source", "docs")
	require.NoError(t, os.MkdirAll(docsDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("guide\n"), 0o600))

	client := newTreeTestClient(t)
	client.On("GetGitTree", mock.Anything, "owner/target", defaultBranchRef, true).
		Return(&gh.GitTree{Tree: []gh.GitTreeNode{blobNode("docs/guide.md", []byte("guide\n"))}}, nil).Once()
	// No GetFile expectation: the unchanged file is settled by the shared tree.

	logger := logrus.NewEntry(logrus.New())
	rs := &RepositorySync{
		engine: &Engine{gh: client, git: &git.MockClient{}},
		target: config.TargetConfig{
			Repo:        "owner/target",
			Directories: []config.DirectoryMapping{{Src: "docs", Dest: "docs"}},
		},
		sourceState: &state.SourceState{Repo: "owner/source"},
		logger:      logger,
		tempDir:     tempDir,
	}
	rs.targetTree = newTargetTree(client, rs.target.Repo, rs.target.Branch, logger)

	changes, err := rs.processDirectories(context.Background())

	require.NoError(t, err)
	assert.Empty(t, changes)
	assert.Equal(t, targetTreeStats{TreeRequests: 1, RequestsAvoided: 1}, rs.targetTree.stats())
	client.AssertExpectations(t)
}

// TestRepositorySyncRecordTargetTreeStatsWithoutChecks verifies nothing is recorded
// when there is no tree or the tree was never consulted.
func TestRepositorySyncRecordTargetTreeStatsWithoutChecks(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name string
		tree *targetTree
	}{
		{name: "no tree", tree: nil},
		{name: "tree never consulted", tree: newTargetTree(nil, "owner/target", "", logger)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rs := &RepositorySync{
				logger:      logger,
				syncMetrics: &PerformanceMetrics{DirectoryMetrics: make(map[string]DirectoryMetrics)},
				targetTree:  tc.tree,
			}

			rs.recordTargetTreeStats()

			assert.Zero(t, rs.syncMetrics.TotalAPIRequests)
			assert.Zero(t, rs.syncMetrics.APICallsSaved)
		})
	}
}
