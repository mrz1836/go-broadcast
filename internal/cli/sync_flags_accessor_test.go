package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSyncFlagAccessors covers the thread-safe getAutomerge / getClearModuleCache
// accessors. Serial because they read package-level flag globals.
func TestSyncFlagAccessors(t *testing.T) { //nolint:paralleltest // mutates package globals
	syncFlagsMu.Lock()
	oldAuto, oldClear := automerge, clearModuleCache
	automerge, clearModuleCache = true, true
	syncFlagsMu.Unlock()
	t.Cleanup(func() {
		syncFlagsMu.Lock()
		automerge, clearModuleCache = oldAuto, oldClear
		syncFlagsMu.Unlock()
	})

	assert.True(t, getAutomerge())
	assert.True(t, getClearModuleCache())

	syncFlagsMu.Lock()
	automerge, clearModuleCache = false, false
	syncFlagsMu.Unlock()

	assert.False(t, getAutomerge())
	assert.False(t, getClearModuleCache())
}
