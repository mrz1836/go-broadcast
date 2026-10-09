package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// errNoNetwork stands in for a failed network/command operation in tests so the
// cli test package never makes real outbound requests.
var errNoNetwork = errors.New("network access disabled in tests")

// TestMain forces the newGHClient seam to fail fast with gh.ErrGHNotFound for the
// entire cli test package. This guarantees tests never spin up the real `gh` CLI,
// which makes real network calls (`gh auth status`, `gh api`) and is slow and
// flaky under the concurrent race suite.
//
// It reproduces the exact condition CI sees on a machine without the gh CLI, so
// existing assertions (which already tolerate gh being unavailable) are unchanged.
// Tests that need a working client inject their own mock via the *WithClient
// helpers or the newReviewPRClient seam; the real-API integration tests are
// explicitly t.Skip'd.
func TestMain(m *testing.M) {
	// Keep AI credentials in the ambient environment from turning tests into live,
	// billable provider requests. See testutil.DisableAIEnv.
	restoreAIEnv := testutil.DisableAIEnv()

	newGHClient = func(context.Context, *logrus.Logger, *logging.LogConfig) (gh.Client, error) {
		return nil, gh.ErrGHNotFound
	}

	// Prevent `git ls-remote` from reaching real git hosts (e.g. github.com/golang/go).
	// All module-version tests tolerate a fetch failure; the success path is covered
	// by TestFetchGitTags_ParsesOutput, which overrides this seam locally.
	gitLsRemoteTags = func(context.Context, string) ([]byte, error) {
		return nil, errNoNetwork
	}

	// Never fall back to the developer's real database at db.DefaultPath().
	// Commands that open the database (sync metrics, settings presets, db CRUD)
	// run AutoMigrate on open, so a test reaching the default path would migrate
	// the real database. Tests that need a database point dbPath at their own
	// temporary file (see setupTestDB); everything else sees a missing database,
	// exactly as on CI.
	isolatedDBDir, err := os.MkdirTemp("", "go-broadcast-cli-test-db-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create isolated test database dir: %v\n", err)
		os.Exit(1)
	}
	dbPath = filepath.Join(isolatedDBDir, "broadcast.db")

	code := m.Run()
	restoreAIEnv()
	_ = os.RemoveAll(isolatedDBDir)

	os.Exit(code)
}
