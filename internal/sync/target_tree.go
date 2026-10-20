package sync

import (
	"context"
	"crypto/sha1" //nolint:gosec // G505: Git names blobs by their SHA-1 content hash; not a security use
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

// Sentinel errors for target tree lookups
var (
	// ErrTargetTreeUnavailable indicates the target branch's Git tree could not be
	// fetched in full, so it cannot be used to enumerate or compare files.
	ErrTargetTreeUnavailable = errors.New("target repository tree unavailable")

	// ErrDirectoryDeletionRoot indicates a directory deletion resolved to the
	// repository root, which is never deleted wholesale.
	ErrDirectoryDeletionRoot = errors.New("refusing to delete the repository root")
)

const (
	// gitEntryTypeBlob and gitEntryTypeTree are Git tree entry types.
	gitEntryTypeBlob = "blob"
	gitEntryTypeTree = "tree"

	// gitModeFile and gitModeExecutable are the modes of regular files in a Git tree.
	// Other blob modes (such as 120000 for symlinks) are not compared by content.
	gitModeFile       = "100644"
	gitModeExecutable = "100755"

	// defaultBranchRef names the repository's default branch in Git Trees API requests.
	defaultBranchRef = "HEAD"
)

// treeVerdict is what a target tree can say about one path.
type treeVerdict int

const (
	// treeVerdictUnknown means the tree cannot settle the question; the file must be fetched.
	treeVerdictUnknown treeVerdict = iota

	// treeVerdictAbsent means no file exists at the path on the target branch.
	treeVerdictAbsent

	// treeVerdictMatches means the target file already holds exactly the candidate content.
	treeVerdictMatches

	// treeVerdictDiffers means the target file exists with different content.
	treeVerdictDiffers
)

// targetTreeStats summarizes how a target's file checks were answered.
type targetTreeStats struct {
	TreeRequests    int // Git Trees API requests made (at most one per target)
	ContentRequests int // Contents API requests made for files the tree could not settle
	RequestsAvoided int // File checks answered from the tree without a request
}

// targetTree is a lazily fetched snapshot of a target branch's Git tree. It lets
// a sync decide whether each target file needs to change without downloading it.
//
// Git names every blob by a hash of its content, so hashing the content a sync
// intends to write and comparing it with the blob ID recorded in the target tree
// proves the target already holds that exact content. A single recursive tree
// request therefore replaces one Contents API request for every unchanged file.
// Files whose content differs are still fetched (once each), because their
// existing bytes feed the PR diff and the line statistics.
//
// Every question the tree cannot answer with certainty is answered with GetFile,
// exactly as it was before the tree existed: the tree request failed, GitHub
// truncated the listing, the path is not spelled the way Git stores paths, or the
// entry is not a regular file (symlinks, submodules, directories). The tree can
// only remove requests; it never changes what a sync decides.
//
// A targetTree is safe for concurrent use and is bound to one repository and
// branch for the lifetime of one target sync, so it never serves a stale tree to
// a later sync.
type targetTree struct {
	client gh.Client
	repo   string
	branch string // As configured; empty means the repository's default branch
	logger *logrus.Entry

	loadOnce sync.Once
	entries  map[string]gh.GitTreeNode // Every entry by path; nil until a complete tree loads
	loadErr  error                     // Why the tree is unusable, when entries is nil

	treeRequests    atomic.Int64
	contentRequests atomic.Int64
	requestsAvoided atomic.Int64
}

// newTargetTree returns a targetTree for the given repository and branch. The tree
// is not fetched until the first lookup needs it.
func newTargetTree(client gh.Client, repo, branch string, logger *logrus.Entry) *targetTree {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &targetTree{
		client: client,
		repo:   repo,
		branch: branch,
		logger: logger,
	}
}

// serves reports whether t describes the branch that target syncs against.
// It is false for a nil tree.
func (t *targetTree) serves(target config.TargetConfig) bool {
	return t != nil && t.repo == target.Repo && t.branch == target.Branch
}

// existingContent returns the content filePath currently holds on the target
// branch, answering from the tree whenever it can.
//
// Parameters:
// - ctx: Context for the tree and content requests
// - filePath: Repository-relative path of the target file
// - candidate: Content the sync intends to write, or nil when there is none (deletions)
//
// Returns:
// - candidate itself when the tree shows the target file already holds exactly that content
// - gh.ErrFileNotFound when the tree shows no file exists at filePath
// - Otherwise the content (or error) returned by GetFile for filePath on the target branch
//
// Side Effects:
// - Fetches the target tree on first use (one Git Trees API request per target)
// - Calls GetFile only when the tree cannot settle the check
func (t *targetTree) existingContent(ctx context.Context, filePath string, candidate []byte) ([]byte, error) {
	switch t.verdict(ctx, filePath, candidate) {
	case treeVerdictMatches:
		t.requestsAvoided.Add(1)
		return candidate, nil
	case treeVerdictAbsent:
		t.requestsAvoided.Add(1)
		return nil, gh.ErrFileNotFound
	default:
		// The content differs or the tree cannot tell; fetch the file for the diff.
	}

	t.contentRequests.Add(1)
	fileContent, err := t.client.GetFile(ctx, t.repo, filePath, t.branch)
	if err != nil {
		return nil, err
	}
	return fileContent.Content, nil
}

// filesUnder returns, in sorted order, the path of every file stored below dir on
// the target branch.
//
// Parameters:
// - ctx: Context for the tree request
// - dir: Repository-relative directory; leading and trailing slashes are ignored
//
// Returns:
// - The file paths below dir (empty when the directory does not exist)
// - ErrDirectoryDeletionRoot when dir names the repository root
// - An ErrTargetTreeUnavailable error when the tree could not be fetched in full
//
// Side Effects:
// - Fetches the target tree on first use (one Git Trees API request per target)
//
// Notes:
// - A partial listing would delete only part of the directory, so a truncated tree is an error
func (t *targetTree) filesUnder(ctx context.Context, dir string) ([]string, error) {
	cleanDir := strings.TrimPrefix(strings.TrimSuffix(dir, "/"), "/")
	if cleanDir == "" || cleanDir == "." {
		return nil, fmt.Errorf("%w: %q", ErrDirectoryDeletionRoot, dir)
	}

	t.load(ctx)
	if t.entries == nil {
		return nil, t.loadErr
	}

	prefix := cleanDir + "/"
	var files []string
	for entryPath, node := range t.entries {
		if node.Type == gitEntryTypeBlob && strings.HasPrefix(entryPath, prefix) {
			files = append(files, entryPath)
		}
	}
	sort.Strings(files)

	return files, nil
}

// stats returns the request accounting for this target so far.
func (t *targetTree) stats() targetTreeStats {
	return targetTreeStats{
		TreeRequests:    int(t.treeRequests.Load()),
		ContentRequests: int(t.contentRequests.Load()),
		RequestsAvoided: int(t.requestsAvoided.Load()),
	}
}

// verdict reports what the tree knows about filePath relative to candidate.
// A nil candidate is never treated as matching, so its content is always fetched.
func (t *targetTree) verdict(ctx context.Context, filePath string, candidate []byte) treeVerdict {
	if !isCanonicalTreePath(filePath) {
		return treeVerdictUnknown
	}

	t.load(ctx)
	if t.entries == nil {
		return treeVerdictUnknown
	}

	node, found := t.entries[filePath]
	if !found {
		if t.hasNonDirectoryAncestor(filePath) {
			return treeVerdictUnknown
		}
		return treeVerdictAbsent
	}

	if candidate == nil || !isRegularFile(node) {
		return treeVerdictUnknown
	}

	blobID, ok := gitBlobID(candidate, len(node.SHA))
	if !ok {
		return treeVerdictUnknown
	}
	if strings.EqualFold(blobID, node.SHA) {
		return treeVerdictMatches
	}
	return treeVerdictDiffers
}

// load fetches the target tree the first time it is needed. A failed or truncated
// fetch leaves the tree unusable, and every lookup then falls back to GetFile.
func (t *targetTree) load(ctx context.Context) {
	t.loadOnce.Do(func() {
		ref := t.branch
		if ref == "" {
			ref = defaultBranchRef
		}

		t.treeRequests.Add(1)
		tree, err := t.client.GetGitTree(ctx, t.repo, ref, true)
		switch {
		case err != nil:
			t.loadErr = fmt.Errorf("%w: %w", ErrTargetTreeUnavailable, err)
		case tree == nil:
			t.loadErr = fmt.Errorf("%w: empty response", ErrTargetTreeUnavailable)
		case tree.Truncated:
			// GitHub truncates recursive listings of very large trees; an
			// incomplete listing cannot prove that a path is absent.
			t.loadErr = fmt.Errorf("%w: listing truncated by GitHub", ErrTargetTreeUnavailable)
		default:
			entries := make(map[string]gh.GitTreeNode, len(tree.Tree))
			for _, node := range tree.Tree {
				entries[node.Path] = node
			}
			t.entries = entries
		}

		log := t.logger.WithFields(logrus.Fields{
			"target_repo": t.repo,
			"ref":         ref,
		})
		if t.loadErr != nil {
			log.WithError(t.loadErr).Warn("Target tree unavailable; comparing target files individually")
			return
		}
		log.WithField("entries", len(t.entries)).Debug("Loaded target tree for change detection")
	})
}

// hasNonDirectoryAncestor reports whether any parent of filePath is a tree entry
// other than a directory, such as a symlink or submodule. The Contents API may
// resolve such paths in ways a tree listing cannot predict.
func (t *targetTree) hasNonDirectoryAncestor(filePath string) bool {
	for dir := path.Dir(filePath); dir != "."; dir = path.Dir(dir) {
		if node, ok := t.entries[dir]; ok && node.Type != gitEntryTypeTree {
			return true
		}
	}
	return false
}

// isRegularFile reports whether node is a regular (optionally executable) file.
func isRegularFile(node gh.GitTreeNode) bool {
	return node.Type == gitEntryTypeBlob && (node.Mode == gitModeFile || node.Mode == gitModeExecutable)
}

// isCanonicalTreePath reports whether p is spelled exactly as Git tree entries
// are: relative, slash-separated, and free of "." and ".." segments, repeated
// slashes, and trailing slashes. Other spellings are left to GetFile so the tree
// never reinterprets a path the Contents API might resolve differently.
func isCanonicalTreePath(p string) bool {
	return p != "" && p != "." && p != ".." &&
		!strings.HasPrefix(p, "/") &&
		!strings.HasPrefix(p, "../") &&
		!strings.Contains(p, `\`) &&
		path.Clean(p) == p
}

// gitBlobID returns the object ID Git assigns to content stored as a blob, using
// the hash algorithm implied by the length of the repository's object IDs: 40 hex
// digits for SHA-1 and 64 for SHA-256. ok is false for any other length.
func gitBlobID(content []byte, objectIDLength int) (id string, ok bool) {
	var hasher hash.Hash
	switch objectIDLength {
	case 2 * sha1.Size:
		hasher = sha1.New() //nolint:gosec // G401: Git object IDs are SHA-1 content hashes; not a security use
	case 2 * sha256.Size:
		hasher = sha256.New()
	default:
		return "", false
	}

	// Git hashes a blob as the header "blob <size>\x00" followed by the raw content.
	_, _ = hasher.Write([]byte("blob " + strconv.Itoa(len(content)) + "\x00"))
	_, _ = hasher.Write(content)

	return hex.EncodeToString(hasher.Sum(nil)), true
}
