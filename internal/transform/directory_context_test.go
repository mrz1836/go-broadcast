package transform

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestNewDirectoryTransformContext(t *testing.T) {
	baseCtx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "/path/to/file.txt",
		Variables:  map[string]string{"key": "value"},
		LogConfig:  &logging.LogConfig{Debug: logging.DebugFlags{Transform: true}},
	}

	dirMapping := &config.DirectoryMapping{
		Src:  "source/dir",
		Dest: "dest/dir",
	}

	relativePath := "subdir/file.txt"
	fileIndex := 5
	totalFiles := 10

	// Capture time before creation for duration verification
	beforeCreation := time.Now()

	ctx := NewDirectoryTransformContext(baseCtx, dirMapping, relativePath, fileIndex, totalFiles)

	// Verify all fields are set correctly
	require.NotNil(t, ctx)
	assert.Equal(t, baseCtx.SourceRepo, ctx.SourceRepo)
	assert.Equal(t, baseCtx.TargetRepo, ctx.TargetRepo)
	assert.Equal(t, baseCtx.FilePath, ctx.FilePath)
	assert.Equal(t, baseCtx.Variables, ctx.Variables)
	assert.Equal(t, baseCtx.LogConfig, ctx.LogConfig)

	assert.True(t, ctx.IsFromDirectory)
	assert.Equal(t, dirMapping, ctx.DirectoryMapping)
	assert.Equal(t, relativePath, ctx.RelativePath)
	assert.Equal(t, fileIndex, ctx.FileIndex)
	assert.Equal(t, totalFiles, ctx.TotalFiles)

	// Verify TransformStartTime is set and reasonable
	assert.False(t, ctx.TransformStartTime.IsZero())
	assert.True(t, ctx.TransformStartTime.After(beforeCreation) || ctx.TransformStartTime.Equal(beforeCreation))
	assert.True(t, ctx.TransformStartTime.Before(time.Now().Add(time.Second)))
}

func TestNewDirectoryTransformContextWithNilMapping(t *testing.T) {
	baseCtx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "/path/to/file.txt",
	}

	ctx := NewDirectoryTransformContext(baseCtx, nil, "file.txt", 0, 1)

	require.NotNil(t, ctx)
	assert.True(t, ctx.IsFromDirectory)
	assert.Nil(t, ctx.DirectoryMapping)
	assert.Equal(t, "file.txt", ctx.RelativePath)
	assert.Equal(t, 0, ctx.FileIndex)
	assert.Equal(t, 1, ctx.TotalFiles)
	assert.False(t, ctx.TransformStartTime.IsZero())
}

func TestGetTransformDuration(t *testing.T) {
	baseCtx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "/path/to/file.txt",
	}

	dirMapping := &config.DirectoryMapping{
		Src:  "source/dir",
		Dest: "dest/dir",
	}

	ctx := NewDirectoryTransformContext(baseCtx, dirMapping, "file.txt", 0, 1)

	// Initial duration should be very small but positive
	duration1 := ctx.GetTransformDuration()
	assert.GreaterOrEqual(t, duration1, time.Duration(0))

	// Sleep a bit and verify duration increases
	time.Sleep(10 * time.Millisecond)
	duration2 := ctx.GetTransformDuration()
	assert.Greater(t, duration2, duration1)
	assert.GreaterOrEqual(t, duration2, 10*time.Millisecond)
}

func TestDirectoryTransformContextStringFromDirectory(t *testing.T) {
	baseCtx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "/path/to/file.txt",
	}

	dirMapping := &config.DirectoryMapping{
		Src:  "source/dir",
		Dest: "dest/dir",
	}

	ctx := NewDirectoryTransformContext(baseCtx, dirMapping, "subdir/file.txt", 2, 5)

	result := ctx.String()

	// Verify the string contains all expected information
	assert.Contains(t, result, "DirectoryTransformContext{")
	assert.Contains(t, result, "SourceRepo: org/source")
	assert.Contains(t, result, "TargetRepo: org/target")
	assert.Contains(t, result, "FilePath: /path/to/file.txt")
	assert.Contains(t, result, "RelativePath: subdir/file.txt")
	assert.Contains(t, result, "Progress: 3/5") // FileIndex+1 for 1-based display
	assert.Contains(t, result, "DirectoryMapping: source/dir->dest/dir")
	assert.Contains(t, result, "Duration:")

	// Verify it's a well-formed string
	assert.True(t, strings.HasPrefix(result, "DirectoryTransformContext{"))
	assert.True(t, strings.HasSuffix(result, "}"))
}

func TestDirectoryTransformContextStringNotFromDirectory(t *testing.T) {
	ctx := &DirectoryTransformContext{
		Context: Context{
			SourceRepo: "org/source",
			TargetRepo: "org/target",
			FilePath:   "/path/to/file.txt",
		},
		IsFromDirectory: false, // Explicitly set to false
	}

	result := ctx.String()

	// When not from directory, should use simplified format
	expected := "DirectoryTransformContext{FilePath: /path/to/file.txt, IsFromDirectory: false}"
	assert.Equal(t, expected, result)
}

func TestDirectoryTransformContextStringWithNilDirectoryMapping(t *testing.T) {
	ctx := &DirectoryTransformContext{
		Context: Context{
			SourceRepo: "org/source",
			TargetRepo: "org/target",
			FilePath:   "/path/to/file.txt",
		},
		IsFromDirectory:    true,
		DirectoryMapping:   nil, // Nil mapping
		RelativePath:       "file.txt",
		FileIndex:          0,
		TotalFiles:         1,
		TransformStartTime: time.Now(),
	}

	// String() should handle nil DirectoryMapping gracefully without panic
	assert.NotPanics(t, func() {
		result := ctx.String()
		// Verify the output indicates nil DirectoryMapping
		assert.Contains(t, result, "DirectoryTransformContext{")
		assert.Contains(t, result, "FilePath: /path/to/file.txt")
		assert.Contains(t, result, "IsFromDirectory: true")
		assert.Contains(t, result, "DirectoryMapping: <nil>")
	}, "String() should not panic when DirectoryMapping is nil")
}

func TestDirectoryTransformContextStringProgress(t *testing.T) {
	tests := []struct {
		name       string
		fileIndex  int
		totalFiles int
		expected   string
	}{
		{
			name:       "first file",
			fileIndex:  0,
			totalFiles: 10,
			expected:   "Progress: 1/10",
		},
		{
			name:       "middle file",
			fileIndex:  4,
			totalFiles: 10,
			expected:   "Progress: 5/10",
		},
		{
			name:       "last file",
			fileIndex:  9,
			totalFiles: 10,
			expected:   "Progress: 10/10",
		},
		{
			name:       "single file",
			fileIndex:  0,
			totalFiles: 1,
			expected:   "Progress: 1/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseCtx := Context{
				SourceRepo: "org/source",
				TargetRepo: "org/target",
				FilePath:   "/path/to/file.txt",
			}

			dirMapping := &config.DirectoryMapping{
				Src:  "src",
				Dest: "dest",
			}

			ctx := NewDirectoryTransformContext(baseCtx, dirMapping, "file.txt", tt.fileIndex, tt.totalFiles)
			result := ctx.String()

			assert.Contains(t, result, tt.expected)
		})
	}
}
