package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandlerHonorsCanceledContext verifies that db command handlers propagate
// the context threaded from cmd.Context() into their database operations, so a
// canceled context (e.g., Ctrl-C/SIGTERM) aborts the handler promptly instead of
// running to completion against context.Background(). This guards the Batch 4
// context-propagation fix.
func TestHandlerHonorsCanceledContext(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("canceled context aborts the handler", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel before the handler runs

		// jsonOutput=false so printErrorResponse returns the error rather than
		// emitting JSON and returning nil.
		err := runGroupList(ctx, false)
		require.Error(t, err, "handler should fail fast when its context is already canceled")
		assert.Contains(t, err.Error(), context.Canceled.Error(),
			"the canceled context should propagate into the database operation")
	})

	t.Run("valid context succeeds", func(t *testing.T) {
		require.NoError(t, runGroupList(context.Background(), true))
	})
}
