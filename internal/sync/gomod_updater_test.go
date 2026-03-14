package sync

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoModUpdater_UpdateDependency(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	updater := NewGoModUpdater(logger)

	t.Run("updates single-line require statement", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require github.com/sirupsen/logrus v1.8.0
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/sirupsen/logrus", "v1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "github.com/sirupsen/logrus v1.9.0")
		assert.NotContains(t, string(updated), "v1.8.0")
	})

	t.Run("updates dependency in require block", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require (
	github.com/sirupsen/logrus v1.8.0
	github.com/stretchr/testify v1.8.4
)
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/sirupsen/logrus", "v1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "github.com/sirupsen/logrus v1.9.0")
		// Other dependencies should be unchanged
		assert.Contains(t, string(updated), "github.com/stretchr/testify v1.8.4")
	})

	t.Run("preserves indirect comment", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require (
	github.com/sirupsen/logrus v1.8.0 // indirect
)
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/sirupsen/logrus", "v1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "github.com/sirupsen/logrus v1.9.0 // indirect")
	})

	t.Run("handles version with prerelease tag", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require github.com/example/pkg v1.0.0-beta.1
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/example/pkg", "v1.0.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "github.com/example/pkg v1.0.0")
		assert.NotContains(t, string(updated), "beta")
	})

	t.Run("adds v prefix if missing", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require github.com/sirupsen/logrus v1.8.0
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/sirupsen/logrus", "1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "github.com/sirupsen/logrus v1.9.0")
	})

	t.Run("returns unchanged if dependency not found", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require github.com/sirupsen/logrus v1.8.0
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/other/pkg", "v1.0.0")
		require.NoError(t, err)
		assert.False(t, modified)
		assert.Equal(t, content, updated)
	})

	t.Run("returns unchanged for empty module path", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21
`)
		updated, modified, err := updater.UpdateDependency(content, "", "v1.0.0")
		require.NoError(t, err)
		assert.False(t, modified)
		assert.Equal(t, content, updated)
	})

	t.Run("returns unchanged for empty version", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/foo/bar", "")
		require.NoError(t, err)
		assert.False(t, modified)
		assert.Equal(t, content, updated)
	})

	t.Run("handles module path with special characters", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require (
	github.com/go-playground/validator/v10 v10.15.0
)
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/go-playground/validator/v10", "v10.16.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "github.com/go-playground/validator/v10 v10.16.0")
	})

	t.Run("works with nil logger", func(t *testing.T) {
		updater := NewGoModUpdater(nil)
		content := []byte(`module example.com/test

go 1.21

require github.com/sirupsen/logrus v1.8.0
`)
		updated, modified, err := updater.UpdateDependency(content, "github.com/sirupsen/logrus", "v1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "v1.9.0")
	})
}

func TestGoModUpdater_AddDependency(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	updater := NewGoModUpdater(logger)

	t.Run("adds to existing require block", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require (
	github.com/sirupsen/logrus v1.8.0
)
`)
		updated, modified, err := updater.AddDependency(content, "github.com/stretchr/testify", "v1.8.4")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "github.com/stretchr/testify v1.8.4")
		// Original dependency should remain
		assert.Contains(t, string(updated), "github.com/sirupsen/logrus v1.8.0")
	})

	t.Run("creates new require after go directive", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21
`)
		updated, modified, err := updater.AddDependency(content, "github.com/sirupsen/logrus", "v1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "require github.com/sirupsen/logrus v1.9.0")
	})

	t.Run("updates if dependency already exists", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21

require github.com/sirupsen/logrus v1.8.0
`)
		updated, modified, err := updater.AddDependency(content, "github.com/sirupsen/logrus", "v1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "github.com/sirupsen/logrus v1.9.0")
		assert.NotContains(t, string(updated), "v1.8.0")
	})

	t.Run("adds v prefix if missing", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21
`)
		updated, modified, err := updater.AddDependency(content, "github.com/sirupsen/logrus", "1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "v1.9.0")
	})

	t.Run("appends to end if no go directive", func(t *testing.T) {
		content := []byte(`module example.com/test
`)
		updated, modified, err := updater.AddDependency(content, "github.com/sirupsen/logrus", "v1.9.0")
		require.NoError(t, err)
		assert.True(t, modified)
		assert.Contains(t, string(updated), "require github.com/sirupsen/logrus v1.9.0")
	})

	t.Run("returns unchanged for empty module path", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21
`)
		updated, modified, err := updater.AddDependency(content, "", "v1.0.0")
		require.NoError(t, err)
		assert.False(t, modified)
		assert.Equal(t, content, updated)
	})

	t.Run("returns unchanged for empty version", func(t *testing.T) {
		content := []byte(`module example.com/test

go 1.21
`)
		updated, modified, err := updater.AddDependency(content, "github.com/foo/bar", "")
		require.NoError(t, err)
		assert.False(t, modified)
		assert.Equal(t, content, updated)
	})
}

func TestSanitizeVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple version", "v1.0.0", "v1.0.0"},
		{"with slash", "feature/v1.0.0", "feature-v1.0.0"},
		{"with backslash", "v1.0.0\\beta", "v1.0.0-beta"},
		{"with colon", "v1:0:0", "v1-0-0"},
		{"with asterisk", "v1.0.*", "v1.0.-"},
		{"with question mark", "v1.0.?", "v1.0.-"},
		{"with quotes", "v1.0.0\"beta\"", "v1.0.0-beta-"},
		{"with angle brackets", "<v1.0.0>", "-v1.0.0-"},
		{"with pipe", "v1|v2", "v1-v2"},
		{"complex", "feature/v1.0.0<beta>", "feature-v1.0.0-beta-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
