package integration

import (
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
