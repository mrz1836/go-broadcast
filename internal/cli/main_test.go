package cli

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
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
	newGHClient = func(context.Context, *logrus.Logger, *logging.LogConfig) (gh.Client, error) {
		return nil, gh.ErrGHNotFound
	}

	// Prevent `git ls-remote` from reaching real git hosts (e.g. github.com/golang/go).
	// All module-version tests tolerate a fetch failure; the success path is covered
	// by TestFetchGitTags_ParsesOutput, which overrides this seam locally.
	gitLsRemoteTags = func(context.Context, string) ([]byte, error) {
		return nil, errNoNetwork
	}

	os.Exit(m.Run())
}
