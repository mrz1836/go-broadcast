package sync

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleResolver_ResolveVersion(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(5*time.Minute, logger)
	resolver := NewModuleResolver(logger, cache)

	t.Run("resolves exact version without tags", func(t *testing.T) {
		version, err := resolver.ResolveVersion(context.Background(), "test-repo", "v1.2.3", false)
		require.NoError(t, err)
		assert.Equal(t, "v1.2.3", version)

		// Check it was cached
		cached, found := cache.Get("test-repo:v1.2.3")
		assert.True(t, found)
		assert.Equal(t, "v1.2.3", cached)
	})

	t.Run("returns error for non-exact version without tags", func(t *testing.T) {
		_, err := resolver.ResolveVersion(context.Background(), "test-repo", "~1.2", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot resolve version constraint without git tags")
	})

	t.Run("empty constraint treated as latest", func(t *testing.T) {
		// This will fail without actual git tags, but that's expected
		_, err := resolver.ResolveVersion(context.Background(), "test-repo", "", true)
		// We expect an error because we're not actually fetching git tags in tests
		require.Error(t, err)
	})
}

func TestModuleResolver_IsVersionConstraint(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(5*time.Minute, logger)
	resolver := NewModuleResolver(logger, cache)

	tests := []struct {
		name       string
		constraint string
		expected   bool
	}{
		{"exact version", "v1.2.3", true},
		{"latest keyword", "latest", true},
		{"empty string", "", true},
		{"tilde constraint", "~1.2", true},
		{"caret constraint", "^1.2", true},
		{"range constraint", ">=1.2.0", true},
		{"complex constraint", ">=1.2.0, <2.0.0", true},
		{"invalid version", "v1.2.3.4.5", false},
		{"random string", "foobar", false},
		{"partial version", "1.2", true}, // semver accepts partial versions
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.IsVersionConstraint(tt.constraint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModuleResolver_resolveLatest(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(5*time.Minute, logger)
	resolver := NewModuleResolver(logger, cache)

	t.Run("selects highest version", func(t *testing.T) {
		versions := []string{"v1.0.0", "v2.0.0", "v1.5.0", "v0.9.0"}
		latest, err := resolver.resolveLatest(versions)
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", latest)
	})

	t.Run("handles pre-release versions", func(t *testing.T) {
		versions := []string{"v1.0.0", "v2.0.0-alpha", "v1.5.0"}
		latest, err := resolver.resolveLatest(versions)
		require.NoError(t, err)
		// v2.0.0-alpha is actually the highest version even though it's pre-release
		// If we want to exclude pre-releases, we'd need to add that logic
		assert.Equal(t, "v2.0.0-alpha", latest)
	})

	t.Run("returns error for empty list", func(t *testing.T) {
		_, err := resolver.resolveLatest([]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no versions available")
	})

	t.Run("skips invalid versions", func(t *testing.T) {
		versions := []string{"v1.0.0", "invalid", "v2.0.0", "not-a-version"}
		latest, err := resolver.resolveLatest(versions)
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", latest)
	})

	t.Run("returns error when no valid versions", func(t *testing.T) {
		versions := []string{"invalid", "not-a-version"}
		_, err := resolver.resolveLatest(versions)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid semantic versions found")
	})
}

func TestModuleResolver_resolveExact(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(5*time.Minute, logger)
	resolver := NewModuleResolver(logger, cache)

	t.Run("finds exact match", func(t *testing.T) {
		versions := []string{"v1.0.0", "v1.2.3", "v2.0.0"}
		exact, err := resolver.resolveExact(versions, "v1.2.3")
		require.NoError(t, err)
		assert.Equal(t, "v1.2.3", exact)
	})

	t.Run("returns error when version not found", func(t *testing.T) {
		versions := []string{"v1.0.0", "v2.0.0"}
		_, err := resolver.resolveExact(versions, "v1.2.3")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "version not found: v1.2.3")
	})
}

func TestModuleResolver_resolveSemver(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(5*time.Minute, logger)
	resolver := NewModuleResolver(logger, cache)

	versions := []string{
		"v1.0.0", "v1.1.0", "v1.2.0", "v1.2.3", "v1.3.0",
		"v2.0.0", "v2.1.0", "v2.2.0",
	}

	tests := []struct {
		name       string
		constraint string
		expected   string
	}{
		{"tilde constraint", "~1.2", "v1.2.3"},
		{"caret constraint", "^1.2", "v1.3.0"},
		{"greater than or equal", ">=2.0.0", "v2.2.0"},
		{"less than", "<2.0.0", "v1.3.0"},
		{"range", ">=1.2.0, <2.0.0", "v1.3.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.resolveSemver(versions, tt.constraint)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("returns error for invalid constraint", func(t *testing.T) {
		_, err := resolver.resolveSemver(versions, "invalid-constraint")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid semver constraint")
	})

	t.Run("returns error when no version matches", func(t *testing.T) {
		_, err := resolver.resolveSemver(versions, ">=3.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no version matches constraint")
	})
}

func TestModuleResolver_CacheIntegration(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(5*time.Minute, logger)
	resolver := NewModuleResolver(logger, cache)

	t.Run("uses cached version on second call", func(t *testing.T) {
		// First call - should cache
		version1, err := resolver.ResolveVersion(context.Background(), "test-repo", "v1.2.3", false)
		require.NoError(t, err)
		assert.Equal(t, "v1.2.3", version1)

		// Verify it's in cache
		cacheKey := "test-repo:v1.2.3"
		cached, found := cache.Get(cacheKey)
		assert.True(t, found)
		assert.Equal(t, "v1.2.3", cached)

		// Second call - should use cache
		version2, err := resolver.ResolveVersion(context.Background(), "test-repo", "v1.2.3", false)
		require.NoError(t, err)
		assert.Equal(t, version1, version2)
	})

	t.Run("different constraints have different cache keys", func(t *testing.T) {
		// Resolve two different versions
		version1, err := resolver.ResolveVersion(context.Background(), "test-repo", "v1.0.0", false)
		require.NoError(t, err)
		assert.Equal(t, "v1.0.0", version1)

		version2, err := resolver.ResolveVersion(context.Background(), "test-repo", "v2.0.0", false)
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", version2)

		// Both should be cached separately
		cached1, found1 := cache.Get("test-repo:v1.0.0")
		assert.True(t, found1)
		assert.Equal(t, "v1.0.0", cached1)

		cached2, found2 := cache.Get("test-repo:v2.0.0")
		assert.True(t, found2)
		assert.Equal(t, "v2.0.0", cached2)
	})
}
