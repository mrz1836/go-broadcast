package git

import (
	"context"
	"crypto/rand"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestCloneFilterArgs(t *testing.T) {
	tests := []struct {
		name string
		opts *CloneOptions
		want []string
	}{
		{name: "nil options", opts: nil, want: nil},
		{name: "no filtering", opts: &CloneOptions{}, want: nil},
		{name: "blob limit disabled", opts: &CloneOptions{BlobSizeLimit: "0"}, want: nil},
		{name: "blob limit", opts: &CloneOptions{BlobSizeLimit: "10m"}, want: []string{"--filter=blob:limit=10m"}},
		{
			name: "sparse ignores blob limit",
			opts: &CloneOptions{BlobSizeLimit: "10m", SparsePaths: []string{"README.md"}},
			want: []string{"--depth", "1", "--filter=blob:none", "--no-checkout", "--single-branch", "--no-tags"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cloneFilterArgs(tt.opts))
		})
	}
}

func TestSparseCheckoutPatterns(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{name: "no paths", paths: nil, want: ""},
		{name: "root file", paths: []string{"README.md"}, want: "/README.md\n"},
		{name: "nested file", paths: []string{".github/workflows/ci.yml"}, want: "/.github/workflows/ci.yml\n"},
		{name: "duplicates collapsed", paths: []string{"a.txt", "./a.txt", "/a.txt", "a.txt"}, want: "/a.txt\n"},
		{name: "path normalized", paths: []string{"dir//sub/../file.txt"}, want: "/dir/file.txt\n"},
		{name: "space", paths: []string{"a b.txt"}, want: "/a\\ b.txt\n"},
		{name: "trailing space", paths: []string{"trail "}, want: "/trail\\ \n"},
		{name: "glob characters", paths: []string{"x[1]*?.md"}, want: "/x\\[1\\]\\*\\?.md\n"},
		{name: "comment and negation characters", paths: []string{"#hash", "!bang"}, want: "/\\#hash\n/\\!bang\n"},
		{name: "backslash", paths: []string{`back\slash`}, want: "/back\\\\slash\n"},
		{name: "order preserved", paths: []string{"b.txt", "a.txt"}, want: "/b.txt\n/a.txt\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sparseCheckoutPatterns(tt.paths)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSparseCheckoutPatterns_InvalidPaths(t *testing.T) {
	for _, p := range []string{"", ".", "/", "..", "../escape.txt", "dir/../../escape.txt", "line\nbreak", "carriage\rreturn"} {
		t.Run(strconv.Quote(p), func(t *testing.T) {
			_, err := sparseCheckoutPatterns([]string{"ok.txt", p})
			require.ErrorIs(t, err, ErrInvalidSparsePath)
		})
	}
}

// The tests below run real git against a local file:// server with partial
// clone support enabled, so they exercise exactly the commands a sync runs
// without any network access.

// sparseTestFiles is the content of the seeded server's master branch.
func sparseTestFiles() map[string]string {
	return map[string]string{
		"README.md":                "master readme\n",
		"keep.txt":                 "keep\n",
		"gone.txt":                 "to be deleted\n",
		"dir/a.txt":                "a\n",
		"dir/b.txt":                "b\n",
		".github/workflows/ci.yml": "name: ci\n",
		"a b.txt":                  "space\n",
		"x[1].md":                  "brackets\n",
		"star*.txt":                "star\n",
		"#hash.md":                 "hash\n",
		"!bang.md":                 "bang\n",
		"q?.txt":                   "question\n",
	}
}

// requireSparseCapableGit skips the test when git lacks
// `git sparse-checkout set --no-cone` (added in git 2.35).
func requireSparseCapableGit(t *testing.T) {
	t.Helper()

	out, err := exec.CommandContext(context.Background(), "git", "version").Output()
	require.NoError(t, err)

	m := regexp.MustCompile(`(\d+)\.(\d+)`).FindStringSubmatch(string(out))
	require.Len(t, m, 3, "unexpected git version output: %s", out)
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	if major < 2 || (major == 2 && minor < 35) {
		t.Skipf("git %d.%d does not support non-cone sparse checkout", major, minor)
	}
}

// isolateGitConfig keeps the developer's global and system git configuration
// (commit signing, hooks, default branch) out of the test.
func isolateGitConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test User")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test User")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

func runTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // test helper with controlled arguments
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), out)
	return strings.TrimSpace(string(out))
}

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
}

// newSparseTestServer creates a bare repository that serves partial clones,
// seeded with a master branch (sparseTestFiles plus a large random blob) and a
// development branch whose README differs. It returns the server path and its
// file:// URL.
func newSparseTestServer(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()

	server := filepath.Join(root, "server.git")
	runTestGit(t, root, "init", "--quiet", "--bare", "-b", "master", server)
	runTestGit(t, server, "config", "uploadpack.allowFilter", "true")
	runTestGit(t, server, "config", "uploadpack.allowAnySHA1InWant", "true")

	seed := filepath.Join(root, "seed")
	runTestGit(t, root, "init", "--quiet", "-b", "master", seed)
	for rel, content := range sparseTestFiles() {
		writeTestFile(t, seed, rel, content)
	}
	big := make([]byte, 256*1024)
	_, err := rand.Read(big)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(seed, "big.bin"), big, 0o600))

	runTestGit(t, seed, "add", ".")
	runTestGit(t, seed, "commit", "--quiet", "-m", "initial")
	writeTestFile(t, seed, "keep.txt", "keep v2\n") // give master some history
	runTestGit(t, seed, "commit", "--quiet", "-am", "second")

	runTestGit(t, seed, "checkout", "--quiet", "-b", "development")
	writeTestFile(t, seed, "README.md", "development readme\n")
	runTestGit(t, seed, "commit", "--quiet", "-am", "development")

	runTestGit(t, seed, "push", "--quiet", server, "master", "development")
	return server, "file://" + filepath.ToSlash(server)
}

func newSparseTestClient(t *testing.T) *gitClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	client, err := NewClient(logger, &logging.LogConfig{})
	require.NoError(t, err)
	return client.(*gitClient)
}

// worktreeFiles lists the files present on disk in a clone, excluding .git.
func worktreeFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			rel, relErr := filepath.Rel(root, p)
			if relErr != nil {
				return relErr
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	require.NoError(t, err)
	sort.Strings(files)
	return files
}

func TestGitClient_SparseClone_MaterializesOnlyRequestedPaths(t *testing.T) {
	requireSparseCapableGit(t)
	isolateGitConfig(t)
	ctx := context.Background()
	_, serverURL := newSparseTestServer(t)
	client := newSparseTestClient(t)
	repoPath := filepath.Join(t.TempDir(), "target")

	err := client.CloneWithBranch(ctx, serverURL, repoPath, "development", &CloneOptions{
		SparsePaths: []string{"README.md", "dir/a.txt", "new/not-yet-created.txt", "README.md"},
	})
	require.NoError(t, err)

	// Only the requested, existing files are on disk...
	assert.Equal(t, []string{"README.md", "dir/a.txt"}, worktreeFiles(t, repoPath))

	// ...from the requested branch, not the default branch
	readme, err := os.ReadFile(filepath.Join(repoPath, "README.md")) //nolint:gosec // test path
	require.NoError(t, err)
	assert.Equal(t, "development readme\n", string(readme))
	assert.Equal(t, "development", runTestGit(t, repoPath, "symbolic-ref", "--short", "HEAD"))

	// ...while the index still tracks the complete tree and nothing looks modified
	assert.Len(t, strings.Split(runTestGit(t, repoPath, "ls-files"), "\n"), len(sparseTestFiles())+1)
	assert.Empty(t, runTestGit(t, repoPath, "status", "--porcelain"))

	// The clone is shallow and never downloaded unrequested file contents
	assert.Equal(t, "true", runTestGit(t, repoPath, "rev-parse", "--is-shallow-repository"))
	bigOID := runTestGit(t, repoPath, "rev-parse", "HEAD:big.bin")
	assert.Contains(t, runTestGit(t, repoPath, "rev-list", "--objects", "--missing=print", "HEAD"), "?"+bigOID)
}

// TestGitClient_SparseClone_SyncWorkflow runs the same sequence of git
// operations as the sync engine's commit phase against a sparse clone.
func TestGitClient_SparseClone_SyncWorkflow(t *testing.T) {
	requireSparseCapableGit(t)
	isolateGitConfig(t)
	ctx := context.Background()
	serverPath, serverURL := newSparseTestServer(t)
	client := newSparseTestClient(t)
	repoPath := filepath.Join(t.TempDir(), "target")
	const syncBranch = "chore/sync-files-test"

	require.NoError(t, client.CloneWithBranch(ctx, serverURL, repoPath, "development", &CloneOptions{
		SparsePaths: []string{"README.md", "new/file.txt", "gone.txt"},
	}))
	require.NoError(t, client.CreateBranch(ctx, repoPath, syncBranch))

	// Modify, add, and delete files the way commitChanges does
	writeTestFile(t, repoPath, "README.md", "synced readme\n")
	writeTestFile(t, repoPath, "new/file.txt", "brand new\n")
	require.NoError(t, os.Remove(filepath.Join(repoPath, "gone.txt")))
	require.NoError(t, client.BatchRemoveFiles(ctx, repoPath, []string{"gone.txt"}, false))
	require.NoError(t, client.Add(ctx, repoPath, "."))

	// The staged diff is computed against the target branch content
	diff, err := client.DiffIgnoreWhitespace(ctx, repoPath, true)
	require.NoError(t, err)
	assert.Contains(t, diff, "-development readme")
	assert.Contains(t, diff, "+synced readme")
	assert.Contains(t, diff, "+brand new")
	assert.Contains(t, diff, "-to be deleted")
	assert.NotContains(t, diff, "big.bin")

	require.NoError(t, client.Commit(ctx, repoPath, "sync: update files"))

	changed, err := client.GetChangedFiles(ctx, repoPath)
	require.NoError(t, err)
	sort.Strings(changed)
	assert.Equal(t, []string{"README.md", "gone.txt", "new/file.txt"}, changed)

	require.NoError(t, client.Push(ctx, repoPath, "origin", syncBranch, false))

	// The pushed branch contains exactly the intended changes on top of the
	// target branch, and files that were never downloaded are intact.
	assert.Equal(t, "M\tREADME.md\nD\tgone.txt\nA\tnew/file.txt",
		runTestGit(t, serverPath, "diff", "--name-status", "development", syncBranch))
	assert.Equal(t,
		runTestGit(t, serverPath, "rev-parse", "development:big.bin"),
		runTestGit(t, serverPath, "rev-parse", syncBranch+":big.bin"))
	runTestGit(t, serverPath, "fsck", "--no-progress", "--no-dangling")
}

func TestGitClient_SparseClone_DefaultBranch(t *testing.T) {
	requireSparseCapableGit(t)
	isolateGitConfig(t)
	ctx := context.Background()
	_, serverURL := newSparseTestServer(t)
	client := newSparseTestClient(t)
	repoPath := filepath.Join(t.TempDir(), "target")

	require.NoError(t, client.Clone(ctx, serverURL, repoPath, &CloneOptions{SparsePaths: []string{"README.md"}}))

	assert.Equal(t, "master", runTestGit(t, repoPath, "symbolic-ref", "--short", "HEAD"))
	assert.Equal(t, []string{"README.md"}, worktreeFiles(t, repoPath))
	readme, err := os.ReadFile(filepath.Join(repoPath, "README.md")) //nolint:gosec // test path
	require.NoError(t, err)
	assert.Equal(t, "master readme\n", string(readme))
	assert.Empty(t, runTestGit(t, repoPath, "status", "--porcelain"))
}

func TestGitClient_SparseClone_SpecialCharacterPaths(t *testing.T) {
	requireSparseCapableGit(t)
	isolateGitConfig(t)
	ctx := context.Background()
	_, serverURL := newSparseTestServer(t)
	client := newSparseTestClient(t)
	repoPath := filepath.Join(t.TempDir(), "target")

	special := []string{"a b.txt", "x[1].md", "star*.txt", "#hash.md", "!bang.md", "q?.txt"}
	require.NoError(t, client.CloneWithBranch(ctx, serverURL, repoPath, "master", &CloneOptions{SparsePaths: special}))

	want := append([]string(nil), special...)
	sort.Strings(want)
	assert.Equal(t, want, worktreeFiles(t, repoPath), "each path must match literally, not as a glob")
	assert.Empty(t, runTestGit(t, repoPath, "status", "--porcelain"))
}

func TestGitClient_SparseClone_InvalidPathFailsBeforeCloning(t *testing.T) {
	ctx := context.Background()
	client := newSparseTestClient(t)
	repoPath := filepath.Join(t.TempDir(), "target")

	err := client.CloneWithBranch(ctx, "file:///nonexistent/repo.git", repoPath, "main",
		&CloneOptions{SparsePaths: []string{"ok.txt", "../escape.txt"}})
	require.ErrorIs(t, err, ErrInvalidSparsePath)
	assert.NoDirExists(t, repoPath)

	err = client.Clone(ctx, "file:///nonexistent/repo.git", repoPath, &CloneOptions{SparsePaths: []string{""}})
	require.ErrorIs(t, err, ErrInvalidSparsePath)
	assert.NoDirExists(t, repoPath)
}

func TestGitClient_ConfigureSparseCheckout_FailureRemovesClone(t *testing.T) {
	isolateGitConfig(t)
	client := newSparseTestClient(t)

	// A directory that is not a git repository makes the first step fail
	notARepo := filepath.Join(t.TempDir(), "not-a-repo")
	writeTestFile(t, notARepo, "file.txt", "x")
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(notARepo))

	err := client.configureSparseCheckout(context.Background(), notARepo, "/file.txt\n")
	require.ErrorIs(t, err, ErrSparseCheckout)
	assert.NoDirExists(t, notARepo, "a failed sparse setup must not leave a half-configured clone behind")
}
