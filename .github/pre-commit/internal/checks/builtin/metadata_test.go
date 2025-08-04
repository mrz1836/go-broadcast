package builtin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test EOF check metadata
func TestEOFCheck_Metadata(t *testing.T) {
	check := NewEOFCheck()
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "eof", metadata.Name)
	assert.Equal(t, "Ensure text files end with a newline character", metadata.Description)
	assert.Contains(t, metadata.FilePatterns, "*.go")
	assert.Equal(t, 1*time.Second, metadata.EstimatedDuration)
	assert.Empty(t, metadata.Dependencies)
	assert.Equal(t, 30*time.Second, metadata.DefaultTimeout)
	assert.Equal(t, "formatting", metadata.Category)
	assert.True(t, metadata.RequiresFiles)
}

// Test EOF check with custom timeout metadata
func TestEOFCheckWithTimeout_Metadata(t *testing.T) {
	customTimeout := 60 * time.Second
	check := NewEOFCheckWithTimeout(customTimeout)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "eof", metadata.Name)
	assert.Equal(t, "Ensure text files end with a newline character", metadata.Description)
	assert.Contains(t, metadata.FilePatterns, "*.go")
	assert.Equal(t, 1*time.Second, metadata.EstimatedDuration)
	assert.Empty(t, metadata.Dependencies)
	assert.Equal(t, customTimeout, metadata.DefaultTimeout)
	assert.Equal(t, "formatting", metadata.Category)
	assert.True(t, metadata.RequiresFiles)
}

// Test Whitespace check metadata
func TestWhitespaceCheck_Metadata(t *testing.T) {
	check := NewWhitespaceCheck()
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "whitespace", metadata.Name)
	assert.Equal(t, "Remove trailing whitespace from text files", metadata.Description)
	assert.Contains(t, metadata.FilePatterns, "*.go")
	assert.Equal(t, 1*time.Second, metadata.EstimatedDuration)
	assert.Empty(t, metadata.Dependencies)
	assert.Equal(t, 30*time.Second, metadata.DefaultTimeout)
	assert.Equal(t, "formatting", metadata.Category)
	assert.True(t, metadata.RequiresFiles)
}

// Test Whitespace check with custom timeout metadata
func TestWhitespaceCheckWithTimeout_Metadata(t *testing.T) {
	customTimeout := 45 * time.Second
	check := NewWhitespaceCheckWithTimeout(customTimeout)
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "whitespace", metadata.Name)
	assert.Equal(t, "Remove trailing whitespace from text files", metadata.Description)
	assert.Contains(t, metadata.FilePatterns, "*.go")
	assert.Equal(t, 1*time.Second, metadata.EstimatedDuration)
	assert.Empty(t, metadata.Dependencies)
	assert.Equal(t, customTimeout, metadata.DefaultTimeout)
	assert.Equal(t, "formatting", metadata.Category)
	assert.True(t, metadata.RequiresFiles)
}

// Test that metadata fields are properly set
func TestCheckMetadata_FieldValidation(t *testing.T) {
	tests := []struct {
		name     string
		check    interface{ Metadata() interface{} }
		validate func(t *testing.T, metadata CheckMetadata)
	}{
		{
			name:  "EOF check default",
			check: NewEOFCheck(),
			validate: func(t *testing.T, metadata CheckMetadata) {
				assert.NotEmpty(t, metadata.Name)
				assert.NotEmpty(t, metadata.Description)
				assert.NotEmpty(t, metadata.FilePatterns)
				assert.Greater(t, metadata.EstimatedDuration, time.Duration(0))
				assert.Greater(t, metadata.DefaultTimeout, time.Duration(0))
				assert.NotEmpty(t, metadata.Category)
			},
		},
		{
			name:  "Whitespace check default",
			check: NewWhitespaceCheck(),
			validate: func(t *testing.T, metadata CheckMetadata) {
				assert.NotEmpty(t, metadata.Name)
				assert.NotEmpty(t, metadata.Description)
				assert.NotEmpty(t, metadata.FilePatterns)
				assert.Greater(t, metadata.EstimatedDuration, time.Duration(0))
				assert.Greater(t, metadata.DefaultTimeout, time.Duration(0))
				assert.NotEmpty(t, metadata.Category)
			},
		},
		{
			name:  "EOF check with 2 minute timeout",
			check: NewEOFCheckWithTimeout(2 * time.Minute),
			validate: func(t *testing.T, metadata CheckMetadata) {
				assert.Equal(t, 2*time.Minute, metadata.DefaultTimeout)
			},
		},
		{
			name:  "Whitespace check with 90 second timeout",
			check: NewWhitespaceCheckWithTimeout(90 * time.Second),
			validate: func(t *testing.T, metadata CheckMetadata) {
				assert.Equal(t, 90*time.Second, metadata.DefaultTimeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadataInterface := tt.check.Metadata()
			// Type assert to CheckMetadata
			metadata, ok := metadataInterface.(CheckMetadata)
			require.True(t, ok, "Metadata should be of type CheckMetadata")
			tt.validate(t, metadata)
		})
	}
}

// Test metadata consistency across different instances
func TestMetadata_Consistency(t *testing.T) {
	// Create multiple instances and verify metadata is consistent
	eof1 := NewEOFCheck()
	eof2 := NewEOFCheck()
	metadata1Interface := eof1.Metadata()
	metadata2Interface := eof2.Metadata()

	// Type assert both
	metadata1, ok := metadata1Interface.(CheckMetadata)
	require.True(t, ok)
	metadata2, ok := metadata2Interface.(CheckMetadata)
	require.True(t, ok)

	assert.Equal(t, metadata1, metadata2, "Metadata should be consistent across instances")

	ws1 := NewWhitespaceCheck()
	ws2 := NewWhitespaceCheck()
	wsMetadata1Interface := ws1.Metadata()
	wsMetadata2Interface := ws2.Metadata()

	// Type assert both
	wsMetadata1, ok := wsMetadata1Interface.(CheckMetadata)
	require.True(t, ok)
	wsMetadata2, ok := wsMetadata2Interface.(CheckMetadata)
	require.True(t, ok)

	assert.Equal(t, wsMetadata1, wsMetadata2, "Metadata should be consistent across instances")
}

// Benchmark metadata retrieval
func BenchmarkEOFCheck_Metadata(b *testing.B) {
	check := NewEOFCheck()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Metadata()
	}
}

func BenchmarkWhitespaceCheck_Metadata(b *testing.B) {
	check := NewWhitespaceCheck()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Metadata()
	}
}

// Example of using check metadata
func ExampleEOFCheck_Metadata() {
	check := NewEOFCheck()
	metadataInterface := check.Metadata()

	// Type assert to CheckMetadata
	metadata, ok := metadataInterface.(CheckMetadata)
	if !ok {
		println("Failed to get metadata")
		return
	}

	// Use metadata to display check information
	println("Check:", metadata.Name)
	println("Description:", metadata.Description)
	println("Category:", metadata.Category)
	println("Estimated duration:", metadata.EstimatedDuration.String())
	// Output would show:
	// Check: eof
	// Description: Ensure text files end with a newline character
	// Category: formatting
	// Estimated duration: 1s
}
