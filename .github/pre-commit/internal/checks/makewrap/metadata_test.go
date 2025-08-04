package makewrap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/shared"
)

// Test Fumpt check metadata
func TestFumptCheck_Metadata(t *testing.T) {
	sharedCtx := shared.NewContext()
	check := NewFumptCheckWithSharedContext(sharedCtx)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "fumpt", metadata.Name)
	assert.Equal(t, "Format Go code with gofumpt (stricter gofmt)", metadata.Description)
	assert.Equal(t, []string{"*.go"}, metadata.FilePatterns)
	assert.Equal(t, 3*time.Second, metadata.EstimatedDuration)
	assert.Equal(t, []string{"fumpt"}, metadata.Dependencies)
	assert.Equal(t, 30*time.Second, metadata.DefaultTimeout)
	assert.Equal(t, "formatting", metadata.Category)
	assert.True(t, metadata.RequiresFiles)
}

// Test Fumpt check with custom timeout metadata
func TestFumptCheckWithConfig_Metadata(t *testing.T) {
	sharedCtx := shared.NewContext()
	customTimeout := 60 * time.Second
	check := NewFumptCheckWithConfig(sharedCtx, customTimeout)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "fumpt", metadata.Name)
	assert.Equal(t, "Format Go code with gofumpt (stricter gofmt)", metadata.Description)
	assert.Equal(t, []string{"*.go"}, metadata.FilePatterns)
	assert.Equal(t, 3*time.Second, metadata.EstimatedDuration)
	assert.Equal(t, []string{"fumpt"}, metadata.Dependencies)
	assert.Equal(t, customTimeout, metadata.DefaultTimeout)
	assert.Equal(t, "formatting", metadata.Category)
	assert.True(t, metadata.RequiresFiles)
}

// Test Lint check metadata
func TestLintCheck_Metadata(t *testing.T) {
	sharedCtx := shared.NewContext()
	check := NewLintCheckWithSharedContext(sharedCtx)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "lint", metadata.Name)
	assert.Equal(t, "Run golangci-lint to check code quality and style", metadata.Description)
	assert.Equal(t, []string{"*.go"}, metadata.FilePatterns)
	assert.Equal(t, 10*time.Second, metadata.EstimatedDuration)
	assert.Equal(t, []string{"lint"}, metadata.Dependencies)
	assert.Equal(t, 60*time.Second, metadata.DefaultTimeout)
	assert.Equal(t, "linting", metadata.Category)
	assert.True(t, metadata.RequiresFiles)
}

// Test Lint check with custom timeout metadata
func TestLintCheckWithConfig_Metadata(t *testing.T) {
	sharedCtx := shared.NewContext()
	customTimeout := 120 * time.Second
	check := NewLintCheckWithConfig(sharedCtx, customTimeout)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "lint", metadata.Name)
	assert.Equal(t, "Run golangci-lint to check code quality and style", metadata.Description)
	assert.Equal(t, []string{"*.go"}, metadata.FilePatterns)
	assert.Equal(t, 10*time.Second, metadata.EstimatedDuration)
	assert.Equal(t, []string{"lint"}, metadata.Dependencies)
	assert.Equal(t, customTimeout, metadata.DefaultTimeout)
	assert.Equal(t, "linting", metadata.Category)
	assert.True(t, metadata.RequiresFiles)
}

// Test ModTidy check metadata
func TestModTidyCheck_Metadata(t *testing.T) {
	sharedCtx := shared.NewContext()
	check := NewModTidyCheckWithSharedContext(sharedCtx)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "mod-tidy", metadata.Name)
	assert.Equal(t, "Ensure go.mod and go.sum are up to date and tidy", metadata.Description)
	assert.Equal(t, []string{"*.go", "go.mod", "go.sum"}, metadata.FilePatterns)
	assert.Equal(t, 5*time.Second, metadata.EstimatedDuration)
	assert.Equal(t, []string{"mod-tidy"}, metadata.Dependencies)
	assert.Equal(t, 30*time.Second, metadata.DefaultTimeout)
	assert.Equal(t, "dependencies", metadata.Category)
	assert.False(t, metadata.RequiresFiles) // mod-tidy doesn't require specific files to be staged
}

// Test ModTidy check with custom timeout metadata
func TestModTidyCheckWithConfig_Metadata(t *testing.T) {
	sharedCtx := shared.NewContext()
	customTimeout := 45 * time.Second
	check := NewModTidyCheckWithConfig(sharedCtx, customTimeout)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "mod-tidy", metadata.Name)
	assert.Equal(t, "Ensure go.mod and go.sum are up to date and tidy", metadata.Description)
	assert.Equal(t, []string{"*.go", "go.mod", "go.sum"}, metadata.FilePatterns)
	assert.Equal(t, 5*time.Second, metadata.EstimatedDuration)
	assert.Equal(t, []string{"mod-tidy"}, metadata.Dependencies)
	assert.Equal(t, customTimeout, metadata.DefaultTimeout)
	assert.Equal(t, "dependencies", metadata.Category)
	assert.False(t, metadata.RequiresFiles)
}

// Test all makewrap checks have proper metadata
func TestAllMakewrapChecks_Metadata(t *testing.T) {
	sharedCtx := shared.NewContext()

	tests := []struct {
		name      string
		check     interface{ Metadata() interface{} }
		checkName string
		category  string
		hasFiles  bool
	}{
		{
			name:      "fumpt check",
			check:     NewFumptCheckWithSharedContext(sharedCtx),
			checkName: "fumpt",
			category:  "formatting",
			hasFiles:  true,
		},
		{
			name:      "lint check",
			check:     NewLintCheckWithSharedContext(sharedCtx),
			checkName: "lint",
			category:  "linting",
			hasFiles:  true,
		},
		{
			name:      "mod-tidy check",
			check:     NewModTidyCheckWithSharedContext(sharedCtx),
			checkName: "mod-tidy",
			category:  "dependencies",
			hasFiles:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadataInterface := tt.check.Metadata()
			// Type assert to CheckMetadata
			metadata, ok := metadataInterface.(CheckMetadata)
			require.True(t, ok, "Metadata should be of type CheckMetadata")

			assert.Equal(t, tt.checkName, metadata.Name)
			assert.NotEmpty(t, metadata.Description)
			assert.NotEmpty(t, metadata.FilePatterns)
			assert.Greater(t, metadata.EstimatedDuration, time.Duration(0))
			assert.NotEmpty(t, metadata.Dependencies)
			assert.Len(t, metadata.Dependencies, 1)
			assert.Equal(t, tt.checkName, metadata.Dependencies[0])
			assert.Greater(t, metadata.DefaultTimeout, time.Duration(0))
			assert.Equal(t, tt.category, metadata.Category)
			assert.Equal(t, tt.hasFiles, metadata.RequiresFiles)
		})
	}
}

// Test metadata with different timeout values
func TestMakewrapChecks_TimeoutVariations(t *testing.T) {
	sharedCtx := shared.NewContext()

	timeouts := []time.Duration{
		10 * time.Second,
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
		5 * time.Minute,
	}

	for _, timeout := range timeouts {
		t.Run(timeout.String(), func(t *testing.T) {
			fumpt := NewFumptCheckWithConfig(sharedCtx, timeout)
			fumptMetadata, ok := fumpt.Metadata().(CheckMetadata)
			require.True(t, ok)
			assert.Equal(t, timeout, fumptMetadata.DefaultTimeout)

			lint := NewLintCheckWithConfig(sharedCtx, timeout)
			lintMetadata, ok := lint.Metadata().(CheckMetadata)
			require.True(t, ok)
			assert.Equal(t, timeout, lintMetadata.DefaultTimeout)

			modTidy := NewModTidyCheckWithConfig(sharedCtx, timeout)
			modTidyMetadata, ok := modTidy.Metadata().(CheckMetadata)
			require.True(t, ok)
			assert.Equal(t, timeout, modTidyMetadata.DefaultTimeout)
		})
	}
}

// Benchmark metadata retrieval for makewrap checks
func BenchmarkFumptCheck_Metadata(b *testing.B) {
	sharedCtx := shared.NewContext()
	check := NewFumptCheckWithSharedContext(sharedCtx)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Metadata()
	}
}

func BenchmarkLintCheck_Metadata(b *testing.B) {
	sharedCtx := shared.NewContext()
	check := NewLintCheckWithSharedContext(sharedCtx)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Metadata()
	}
}

func BenchmarkModTidyCheck_Metadata(b *testing.B) {
	sharedCtx := shared.NewContext()
	check := NewModTidyCheckWithSharedContext(sharedCtx)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Metadata()
	}
}

// Example of using makewrap check metadata
func ExampleFumptCheck_Metadata() {
	sharedCtx := shared.NewContext()
	check := NewFumptCheckWithSharedContext(sharedCtx)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	if !ok {
		println("Failed to get metadata")
		return
	}

	// Display check information
	println("Check:", metadata.Name)
	println("Description:", metadata.Description)
	println("Category:", metadata.Category)
	println("Dependencies:", metadata.Dependencies[0])
	println("Requires files:", metadata.RequiresFiles)
	// Output would show:
	// Check: fumpt
	// Description: Format Go code with gofumpt (stricter gofmt)
	// Category: formatting
	// Dependencies: fumpt
	// Requires files: true
}

// Test metadata fields are immutable
func TestMetadata_Immutability(t *testing.T) {
	sharedCtx := shared.NewContext()
	check := NewFumptCheckWithSharedContext(sharedCtx)

	// Get metadata twice
	metadata1Interface := check.Metadata()
	metadata2Interface := check.Metadata()

	// Type assert both
	metadata1, ok := metadata1Interface.(CheckMetadata)
	require.True(t, ok)
	metadata2, ok := metadata2Interface.(CheckMetadata)
	require.True(t, ok)

	// Verify they're equal
	assert.Equal(t, metadata1, metadata2)

	// Modify the first metadata's slice
	originalPatterns := metadata1.FilePatterns
	metadata1.FilePatterns = append(metadata1.FilePatterns, "*.test")

	// Get metadata again and verify it wasn't affected
	metadata3Interface := check.Metadata()
	metadata3, ok := metadata3Interface.(CheckMetadata)
	require.True(t, ok)
	assert.Equal(t, originalPatterns, metadata3.FilePatterns)
}
