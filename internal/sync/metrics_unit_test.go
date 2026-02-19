package sync

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSyncRunExternalID(t *testing.T) {
	t.Parallel()

	// Pattern: SR-YYYYMMDD-6hexchars
	idPattern := regexp.MustCompile(`^SR-\d{8}-[0-9a-f]{6}$`)

	t.Run("format matches SR-YYYYMMDD-hexchars", func(t *testing.T) {
		t.Parallel()

		id := GenerateSyncRunExternalID()
		assert.Regexp(t, idPattern, id, "ID %q should match SR-YYYYMMDD-6hexchars format", id)
	})

	t.Run("date portion matches today UTC", func(t *testing.T) {
		t.Parallel()

		id := GenerateSyncRunExternalID()
		require.Regexp(t, idPattern, id)

		// Extract the date portion (positions 3..11)
		dateStr := id[3:11]
		expectedDate := time.Now().UTC().Format("20060102")
		assert.Equal(t, expectedDate, dateStr, "date portion should match today's UTC date")
	})

	t.Run("multiple calls produce unique IDs", func(t *testing.T) {
		t.Parallel()

		const n = 100
		seen := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			id := GenerateSyncRunExternalID()
			require.Regexp(t, idPattern, id)
			_, duplicate := seen[id]
			assert.False(t, duplicate, "duplicate ID detected: %s", id)
			seen[id] = struct{}{}
		}
		assert.Len(t, seen, n, "all %d IDs should be unique", n)
	})
}

func TestDetermineTrigger(t *testing.T) {
	t.Parallel()

	t.Run("nil options returns manual", func(t *testing.T) {
		t.Parallel()

		trigger := DetermineTrigger(nil)
		assert.Equal(t, TriggerManual, trigger)
	})

	t.Run("default options returns manual", func(t *testing.T) {
		t.Parallel()

		opts := DefaultOptions()
		trigger := DetermineTrigger(opts)
		assert.Equal(t, TriggerManual, trigger)
	})

	t.Run("empty options struct returns manual", func(t *testing.T) {
		t.Parallel()

		trigger := DetermineTrigger(&Options{})
		assert.Equal(t, TriggerManual, trigger)
	})
}

func TestIsRunningInCI(t *testing.T) {
	t.Parallel()

	t.Run("returns false", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isRunningInCI(), "isRunningInCI should return false (current implementation)")
	})
}
