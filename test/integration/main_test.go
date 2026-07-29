package integration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/ai"
	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// TestMain strips AI credentials from the environment before any test runs, so the
// integration suite never issues live provider requests. See testutil.DisableAIEnv.
//
// These tests build real sync engines, and engine construction reads AI settings
// straight from the ambient environment - so without this a developer machine with
// a real key turns the suite into live, billable traffic.
func TestMain(m *testing.M) {
	restore := testutil.DisableAIEnv()
	code := m.Run()
	restore()

	os.Exit(code)
}

// TestAIIsDisabledDuringTests guards the TestMain above against silent removal
func TestAIIsDisabledDuringTests(t *testing.T) {
	cfg := ai.LoadConfig()

	assert.False(t, cfg.IsEnabled(),
		"AI must be disabled in tests; a real key in the environment would make live provider calls")
	assert.Empty(t, cfg.APIKey, "no API key should be visible to tests")
}
