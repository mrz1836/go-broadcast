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
	t.Run("nil options returns manual when no CI vars set", func(t *testing.T) {
		// Clear any CI env vars that might be set
		for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS", "JENKINS_URL", "BUILDKITE"} {
			t.Setenv(v, "")
		}

		trigger := DetermineTrigger(nil)
		assert.Equal(t, TriggerManual, trigger)
	})

	t.Run("default options returns manual when no CI vars set", func(t *testing.T) {
		for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS", "JENKINS_URL", "BUILDKITE"} {
			t.Setenv(v, "")
		}

		opts := DefaultOptions()
		trigger := DetermineTrigger(opts)
		assert.Equal(t, TriggerManual, trigger)
	})

	t.Run("returns ci when CI env var is set", func(t *testing.T) {
		t.Setenv("CI", "true")

		trigger := DetermineTrigger(nil)
		assert.Equal(t, TriggerCI, trigger)
	})

	t.Run("returns ci when GITHUB_ACTIONS is set", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")

		trigger := DetermineTrigger(&Options{})
		assert.Equal(t, TriggerCI, trigger)
	})
}

func TestIsRunningInCI(t *testing.T) {
	t.Run("returns false when no CI vars set", func(t *testing.T) {
		for _, v := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS", "JENKINS_URL", "BUILDKITE"} {
			t.Setenv(v, "")
		}

		assert.False(t, isRunningInCI())
	})

	t.Run("returns true when CI is set", func(t *testing.T) {
		t.Setenv("CI", "true")

		assert.True(t, isRunningInCI())
	})

	t.Run("returns true when GITHUB_ACTIONS is set", func(t *testing.T) {
		t.Setenv("GITHUB_ACTIONS", "true")

		assert.True(t, isRunningInCI())
	})

	t.Run("returns true when GITLAB_CI is set", func(t *testing.T) {
		t.Setenv("GITLAB_CI", "true")

		assert.True(t, isRunningInCI())
	})

	t.Run("returns true when CIRCLECI is set", func(t *testing.T) {
		t.Setenv("CIRCLECI", "true")

		assert.True(t, isRunningInCI())
	})

	t.Run("returns true when TRAVIS is set", func(t *testing.T) {
		t.Setenv("TRAVIS", "true")

		assert.True(t, isRunningInCI())
	})

	t.Run("returns true when JENKINS_URL is set", func(t *testing.T) {
		t.Setenv("JENKINS_URL", "http://jenkins.example.com")

		assert.True(t, isRunningInCI())
	})

	t.Run("returns true when BUILDKITE is set", func(t *testing.T) {
		t.Setenv("BUILDKITE", "true")

		assert.True(t, isRunningInCI())
	})
}
