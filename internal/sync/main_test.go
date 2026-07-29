package sync

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/ai"
	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// TestMain strips AI credentials from the environment before any test runs.
//
// Engine construction calls initializeAI, which reads the ambient environment. On a
// developer machine or CI runner that exports a real key, that turns unit tests into
// live, billable provider requests - the reason this package's tests could take
// minutes and fail on someone else's outage.
func TestMain(m *testing.M) {
	restore := testutil.DisableAIEnv()
	code := m.Run()
	restore()

	os.Exit(code)
}

// TestAIIsDisabledDuringTests fails loudly if the TestMain guard above is removed or
// stops working. Without it the failure mode is silent: tests still pass, they just
// quietly bill a real provider and depend on its uptime.
func TestAIIsDisabledDuringTests(t *testing.T) {
	cfg := ai.LoadConfig()

	assert.False(t, cfg.IsEnabled(),
		"AI must be disabled in tests; a real key in the environment would make live provider calls")
	assert.Empty(t, cfg.APIKey, "no API key should be visible to tests")
}
