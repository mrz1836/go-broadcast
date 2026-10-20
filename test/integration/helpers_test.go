package integration

import (
	"errors"

	"github.com/stretchr/testify/mock"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// boolPtr returns a pointer to the given bool value
func boolPtr(b bool) *bool {
	return &b
}

// healthyRateLimit returns a GetRateLimit response with a generous primary
// budget so the rate-limit preflight gate (enabled by sync.DefaultOptions) lets
// the run proceed during integration tests.
func healthyRateLimit() *gh.RateLimitResponse {
	resp := &gh.RateLimitResponse{}
	resp.Resources.Core.Limit = 5000
	resp.Resources.Core.Remaining = 5000
	resp.Resources.Core.Reset = 1_900_000_000
	return resp
}

// expectRateLimitProbe registers a permissive GetRateLimit expectation on a mock
// GitHub client so the sync preflight probe succeeds with ample budget. It is a
// no-op-safe Maybe() so tests that disable the preflight still pass.
func expectRateLimitProbe(m *gh.MockClient) {
	m.On("GetRateLimit", mock.Anything).Return(healthyRateLimit(), nil).Maybe()
}

// errTargetTreeNotMocked is returned for target-tree requests; see expectTargetTreeUnavailable.
var errTargetTreeNotMocked = errors.New("target tree not provided by this test")

// expectTargetTreeUnavailable answers the target-tree request each target sync makes
// by reporting the tree as unavailable, so file checks fall back to GetFile exactly
// as they did before the tree existed and the test's GetFile expectations still apply.
func expectTargetTreeUnavailable(m *gh.MockClient) {
	m.On("GetGitTree", mock.Anything, mock.Anything, mock.Anything, true).
		Return(nil, errTargetTreeNotMocked).Maybe()
}
