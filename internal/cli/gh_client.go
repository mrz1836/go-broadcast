package cli

import "github.com/mrz1836/go-broadcast/internal/gh"

// newGHClient constructs the GitHub client used by commands that talk to GitHub
// (status, validate, cancel, ...). It is a package-level seam so tests can inject
// a substitute and avoid spinning up the real `gh` CLI, which makes real network
// calls (e.g. `gh auth status`, `gh api`) and is slow/flaky under concurrent runs.
//
// The review-pr command uses its own equivalent seam (newReviewPRClient).
//
//nolint:gochecknoglobals // test injection seam
var newGHClient = gh.NewClient
