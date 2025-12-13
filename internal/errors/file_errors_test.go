package errors //nolint:revive,nolintlint // internal test package, name conflict intentional

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test error variables for use in tests
var (
	errTestPermission = errors.New("permission denied")
	errTestFile       = errors.New("test error")
	errTestJSON       = errors.New("invalid JSON")
	errTestProcessing = errors.New("processing failed")
)

func TestFileOperationError(t *testing.T) {
	baseErr := errTestPermission

	tests := []struct {
		name      string
		operation string
		path      string
		err       error
		want      string
		wantNil   bool
	}{
		{
			name:      "read operation error",
			operation: "read",
			path:      "/path/to/file.txt",
			err:       baseErr,
			want:      "file operation failed: read '/path/to/file.txt': permission denied",
		},
		{
			name:      "write operation error",
			operation: "write",
			path:      "/path/to/file.txt",
			err:       baseErr,
			want:      "file operation failed: write '/path/to/file.txt': permission denied",
		},
		{
			name:      "nil error returns nil",
			operation: "read",
			path:      "/path/to/file.txt",
			err:       nil,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FileOperationError(tt.operation, tt.path, tt.err)
			if tt.wantNil {
				assert.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.want)
				require.ErrorIs(t, err, errFileOperationTemplate)
				require.ErrorIs(t, err, baseErr)
			}
		})
	}
}

func TestFileConvenienceFunctions(t *testing.T) {
	baseErr := errTestFile

	tests := []struct {
		name     string
		fn       func(string, error) error
		path     string
		err      error
		wantText string
	}{
		{
			name:     "FileReadError",
			fn:       FileReadError,
			path:     "/file.txt",
			err:      baseErr,
			wantText: "file operation failed: read '/file.txt': test error",
		},
		{
			name:     "FileWriteError",
			fn:       FileWriteError,
			path:     "/file.txt",
			err:      baseErr,
			wantText: "file operation failed: write '/file.txt': test error",
		},
		{
			name:     "FileOpenError",
			fn:       FileOpenError,
			path:     "/file.txt",
			err:      baseErr,
			wantText: "file operation failed: open '/file.txt': test error",
		},
		{
			name:     "FileCreateError",
			fn:       FileCreateError,
			path:     "/file.txt",
			err:      baseErr,
			wantText: "file operation failed: create '/file.txt': test error",
		},
		{
			name:     "FileDeleteError",
			fn:       FileDeleteError,
			path:     "/file.txt",
			err:      baseErr,
			wantText: "file operation failed: delete '/file.txt': test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.path, tt.err)
			require.EqualError(t, err, tt.wantText)
			require.ErrorIs(t, err, errFileOperationTemplate)
			require.ErrorIs(t, err, baseErr)
		})
	}
}

func TestDirectoryOperationError(t *testing.T) {
	baseErr := errTestPermission

	tests := []struct {
		name      string
		operation string
		path      string
		err       error
		want      string
		wantNil   bool
	}{
		{
			name:      "create directory error",
			operation: "create",
			path:      "/path/to/dir",
			err:       baseErr,
			want:      "directory operation failed: create '/path/to/dir': permission denied",
		},
		{
			name:      "walk directory error",
			operation: "walk",
			path:      "/path/to/dir",
			err:       baseErr,
			want:      "directory operation failed: walk '/path/to/dir': permission denied",
		},
		{
			name:      "nil error returns nil",
			operation: "create",
			path:      "/path/to/dir",
			err:       nil,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DirectoryOperationError(tt.operation, tt.path, tt.err)
			if tt.wantNil {
				assert.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.want)
				require.ErrorIs(t, err, errDirectoryOperationTemplate)
				require.ErrorIs(t, err, baseErr)
			}
		})
	}
}

func TestDirectoryConvenienceFunctions(t *testing.T) {
	baseErr := errTestFile

	tests := []struct {
		name     string
		fn       func(string, error) error
		path     string
		err      error
		wantText string
	}{
		{
			name:     "DirectoryCreateError",
			fn:       DirectoryCreateError,
			path:     "/dir",
			err:      baseErr,
			wantText: "directory operation failed: create '/dir': test error",
		},
		{
			name:     "DirectoryWalkError",
			fn:       DirectoryWalkError,
			path:     "/dir",
			err:      baseErr,
			wantText: "directory operation failed: walk '/dir': test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.path, tt.err)
			require.EqualError(t, err, tt.wantText)
			require.ErrorIs(t, err, errDirectoryOperationTemplate)
			require.ErrorIs(t, err, baseErr)
		})
	}
}

func TestJSONOperationError(t *testing.T) {
	baseErr := errTestJSON

	tests := []struct {
		name      string
		operation string
		context   string
		err       error
		want      string
		wantNil   bool
	}{
		{
			name:      "marshal error",
			operation: "marshal",
			context:   "user data",
			err:       baseErr,
			want:      "JSON operation failed: marshal 'user data': invalid JSON",
		},
		{
			name:      "unmarshal error",
			operation: "unmarshal",
			context:   "config file",
			err:       baseErr,
			want:      "JSON operation failed: unmarshal 'config file': invalid JSON",
		},
		{
			name:      "nil error returns nil",
			operation: "marshal",
			context:   "data",
			err:       nil,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := JSONOperationError(tt.operation, tt.context, tt.err)
			if tt.wantNil {
				assert.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.want)
				require.ErrorIs(t, err, errJSONOperationTemplate)
				require.ErrorIs(t, err, baseErr)
			}
		})
	}
}

func TestJSONConvenienceFunctions(t *testing.T) {
	baseErr := errTestFile

	tests := []struct {
		name     string
		fn       func(string, error) error
		context  string
		err      error
		wantText string
	}{
		{
			name:     "JSONMarshalError",
			fn:       JSONMarshalError,
			context:  "user data",
			err:      baseErr,
			wantText: "JSON operation failed: marshal 'user data': test error",
		},
		{
			name:     "JSONUnmarshalError",
			fn:       JSONUnmarshalError,
			context:  "config",
			err:      baseErr,
			wantText: "JSON operation failed: unmarshal 'config': test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.context, tt.err)
			require.EqualError(t, err, tt.wantText)
			require.ErrorIs(t, err, errJSONOperationTemplate)
			require.ErrorIs(t, err, baseErr)
		})
	}
}

func TestBatchOperationError(t *testing.T) {
	baseErr := errTestProcessing

	tests := []struct {
		name      string
		operation string
		start     int
		end       int
		err       error
		want      string
		wantNil   bool
	}{
		{
			name:      "batch process error",
			operation: "process",
			start:     0,
			end:       10,
			err:       baseErr,
			want:      "batch operation failed: process items 0-9: processing failed",
		},
		{
			name:      "batch validate error",
			operation: "validate",
			start:     5,
			end:       15,
			err:       baseErr,
			want:      "batch operation failed: validate items 5-14: processing failed",
		},
		{
			name:      "nil error returns nil",
			operation: "process",
			start:     0,
			end:       10,
			err:       nil,
			wantNil:   true,
		},
		{
			name:      "single item batch (start equals end-1)",
			operation: "process",
			start:     5,
			end:       6,
			err:       baseErr,
			want:      "batch operation failed: process items 5-5: processing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BatchOperationError(tt.operation, tt.start, tt.end, tt.err)
			if tt.wantNil {
				assert.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.want)
				require.ErrorIs(t, err, errBatchOperationTemplate)
				require.ErrorIs(t, err, baseErr)
			}
		})
	}
}

func TestBatchOperationError_InvalidRange(t *testing.T) {
	baseErr := errTestProcessing

	tests := []struct {
		name        string
		operation   string
		start       int
		end         int
		wantContain string
	}{
		{
			name:        "start greater than end",
			operation:   "process",
			start:       10,
			end:         5,
			wantContain: "invalid range [10, 5)",
		},
		{
			name:        "zero range (start equals end)",
			operation:   "process",
			start:       0,
			end:         0,
			wantContain: "invalid range [0, 0)",
		},
		{
			name:        "same non-zero values",
			operation:   "validate",
			start:       5,
			end:         5,
			wantContain: "invalid range [5, 5)",
		},
		{
			name:        "negative start",
			operation:   "process",
			start:       -1,
			end:         5,
			wantContain: "invalid range [-1, 5)",
		},
		{
			name:        "negative end",
			operation:   "process",
			start:       0,
			end:         -1,
			wantContain: "invalid range [0, -1)",
		},
		{
			name:        "both negative",
			operation:   "process",
			start:       -5,
			end:         -1,
			wantContain: "invalid range [-5, -1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BatchOperationError(tt.operation, tt.start, tt.end, baseErr)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantContain)
			// Should still wrap both template and base error
			require.ErrorIs(t, err, errBatchOperationTemplate)
			require.ErrorIs(t, err, baseErr)
		})
	}
}

func TestBatchOperationError_InvalidRangeNilError(t *testing.T) {
	// Even with invalid range, nil error should return nil
	err := BatchOperationError("process", -1, 5, nil)
	assert.NoError(t, err)
}
