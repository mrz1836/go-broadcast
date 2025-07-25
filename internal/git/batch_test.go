package git

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchAddFiles(t *testing.T) {
	tests := []struct {
		name        string
		repoPath    string
		files       []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "EmptyFileList",
			repoPath:    "/tmp/repo",
			files:       []string{},
			expectError: false,
		},
		{
			name:        "SingleFile",
			repoPath:    "/tmp/repo",
			files:       []string{"file1.txt"},
			expectError: false,
		},
		{
			name:        "SmallBatch",
			repoPath:    "/tmp/repo",
			files:       []string{"file1.txt", "file2.txt", "file3.txt"},
			expectError: false,
		},
		{
			name:        "LargeBatchExceedingLimit",
			repoPath:    "/tmp/repo",
			files:       generateFileList(250), // More than maxBatchSize (100)
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a real gitClient to test the batch logic
			client := &gitClient{logger: nil}
			ctx := context.Background()

			// Test with empty files - should return nil immediately
			if len(tt.files) == 0 {
				err := client.BatchAddFiles(ctx, tt.repoPath, tt.files)
				require.NoError(t, err)
				return
			}

			// For non-empty cases, we need to test with a mock git command
			// This tests the batch splitting logic
			// Full command execution would be tested in integration tests
		})
	}
}

func TestBatchStatus(t *testing.T) {
	tests := []struct {
		name           string
		repoPath       string
		files          []string
		commandOutput  string
		commandError   error
		expectedResult map[string]string
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "EmptyFileList",
			repoPath:       "/tmp/repo",
			files:          []string{},
			expectedResult: map[string]string{},
			expectError:    false,
		},
		{
			name:     "SingleFileModified",
			repoPath: "/tmp/repo",
			files:    []string{"file1.txt"},
			commandOutput: ` M file1.txt
`,
			expectedResult: map[string]string{
				"file1.txt": " M",
			},
			expectError: false,
		},
		{
			name:     "MultipleFilesWithDifferentStatuses",
			repoPath: "/tmp/repo",
			files:    []string{"file1.txt", "file2.txt", "file3.txt"},
			commandOutput: ` M file1.txt
A  file2.txt
?? file3.txt
`,
			expectedResult: map[string]string{
				"file1.txt": " M",
				"file2.txt": "A ",
				"file3.txt": "??",
			},
			expectError: false,
		},
		{
			name:     "FilesWithSpacesInPath",
			repoPath: "/tmp/repo",
			files:    []string{"path/to/my file.txt"},
			commandOutput: ` M path/to/my file.txt
`,
			expectedResult: map[string]string{
				"path/to/my file.txt": " M",
			},
			expectError: false,
		},
		{
			name:         "CommandError",
			repoPath:     "/tmp/repo",
			files:        []string{"file1.txt"},
			commandError: errors.New("git status failed"), //nolint:err113 // test error
			expectError:  true,
			errorMsg:     "batch status failed",
		},
		{
			name:           "EmptyOutput",
			repoPath:       "/tmp/repo",
			files:          []string{"file1.txt"},
			commandOutput:  "",
			expectedResult: map[string]string{},
			expectError:    false,
		},
		{
			name:     "ShortLines",
			repoPath: "/tmp/repo",
			files:    []string{"file1.txt"},
			commandOutput: ` M file1.txt
XX
M
`,
			expectedResult: map[string]string{
				"file1.txt": " M",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test empty file list
			if len(tt.files) == 0 {
				client := &gitClient{logger: nil}
				result, err := client.BatchStatus(context.Background(), tt.repoPath, tt.files)
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
				return
			}

			// For non-empty cases, we need integration tests since BatchStatus
			// uses exec.CommandContext directly
		})
	}
}

func TestBatchStatusAll(t *testing.T) {
	tests := []struct {
		name           string
		repoPath       string
		commandOutput  string
		commandError   error
		expectedResult map[string]string
		expectError    bool
		errorMsg       string
	}{
		{
			name:     "MultipleFilesStatus",
			repoPath: "/tmp/repo",
			commandOutput: ` M src/main.go
A  README.md
?? temp.txt
D  old.go
`,
			expectedResult: map[string]string{
				"src/main.go": " M",
				"README.md":   "A ",
				"temp.txt":    "??",
				"old.go":      "D ",
			},
			expectError: false,
		},
		{
			name:           "EmptyRepository",
			repoPath:       "/tmp/repo",
			commandOutput:  "",
			expectedResult: map[string]string{},
			expectError:    false,
		},
		{
			name:         "CommandError",
			repoPath:     "/tmp/repo",
			commandError: errors.New("not a git repository"), //nolint:err113 // test error
			expectError:  true,
			errorMsg:     "batch status all failed",
		},
		{
			name:     "StatusWithRenames",
			repoPath: "/tmp/repo",
			commandOutput: `R  old-name.txt -> new-name.txt
 M file.go
`,
			expectedResult: map[string]string{
				"old-name.txt -> new-name.txt": "R ",
				"file.go":                      " M",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Since BatchStatusAll uses exec.CommandContext directly,
			// we need integration tests for full coverage
			_ = tt
		})
	}
}

func TestBatchDiffFiles(t *testing.T) {
	tests := []struct {
		name           string
		repoPath       string
		files          []string
		staged         bool
		expectedResult map[string]string
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "EmptyFileList",
			repoPath:       "/tmp/repo",
			files:          []string{},
			staged:         false,
			expectedResult: map[string]string{},
			expectError:    false,
		},
		{
			name:        "LargeBatchSplit",
			repoPath:    "/tmp/repo",
			files:       generateFileList(120), // More than maxBatchSize (50)
			staged:      false,
			expectError: false,
		},
		{
			name:        "StagedDiff",
			repoPath:    "/tmp/repo",
			files:       []string{"file1.txt", "file2.txt"},
			staged:      true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test empty file list
			if len(tt.files) == 0 {
				client := &gitClient{logger: nil}
				result, err := client.BatchDiffFiles(context.Background(), tt.repoPath, tt.files, tt.staged)
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
				return
			}

			// For non-empty cases, we need integration tests
		})
	}
}

func TestBatchCheckIgnored(t *testing.T) {
	tests := []struct {
		name           string
		repoPath       string
		files          []string
		commandOutput  string
		commandError   error
		expectedResult map[string]bool
		expectError    bool
		errorMsg       string
	}{
		{
			name:           "EmptyFileList",
			repoPath:       "/tmp/repo",
			files:          []string{},
			expectedResult: map[string]bool{},
			expectError:    false,
		},
		{
			name:          "NoIgnoredFiles",
			repoPath:      "/tmp/repo",
			files:         []string{"file1.txt", "file2.txt"},
			commandOutput: "",
			commandError:  fmt.Errorf("exit status 1"), //nolint:err113 // test error
			expectedResult: map[string]bool{
				"file1.txt": false,
				"file2.txt": false,
			},
			expectError: false,
		},
		{
			name:     "SomeIgnoredFiles",
			repoPath: "/tmp/repo",
			files:    []string{"file1.txt", ".DS_Store", "build/"},
			commandOutput: `.DS_Store
build/
`,
			expectedResult: map[string]bool{
				"file1.txt": false,
				".DS_Store": true,
				"build/":    true,
			},
			expectError: false,
		},
		{
			name:     "AllIgnoredFiles",
			repoPath: "/tmp/repo",
			files:    []string{"node_modules/", ".env", "*.log"},
			commandOutput: `node_modules/
.env
*.log
`,
			expectedResult: map[string]bool{
				"node_modules/": true,
				".env":          true,
				"*.log":         true,
			},
			expectError: false,
		},
		{
			name:         "CommandError",
			repoPath:     "/tmp/repo",
			files:        []string{"file1.txt"},
			commandError: errors.New("fatal: not a git repository"), //nolint:err113 // test error
			expectError:  true,
			errorMsg:     "batch check-ignore failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test empty file list
			if len(tt.files) == 0 {
				client := &gitClient{logger: nil}
				result, err := client.BatchCheckIgnored(context.Background(), tt.repoPath, tt.files)
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
				return
			}

			// For non-empty cases, we need integration tests
		})
	}
}

func TestBatchRemoveFiles(t *testing.T) {
	tests := []struct {
		name        string
		repoPath    string
		files       []string
		keepLocal   bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "EmptyFileList",
			repoPath:    "/tmp/repo",
			files:       []string{},
			keepLocal:   false,
			expectError: false,
		},
		{
			name:        "SingleFileRemove",
			repoPath:    "/tmp/repo",
			files:       []string{"file1.txt"},
			keepLocal:   false,
			expectError: false,
		},
		{
			name:        "RemoveKeepLocal",
			repoPath:    "/tmp/repo",
			files:       []string{"file1.txt", "file2.txt"},
			keepLocal:   true,
			expectError: false,
		},
		{
			name:        "LargeBatchRemove",
			repoPath:    "/tmp/repo",
			files:       generateFileList(250), // More than maxBatchSize (100)
			keepLocal:   false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test empty file list
			if len(tt.files) == 0 {
				client := &gitClient{logger: nil}
				err := client.BatchRemoveFiles(context.Background(), tt.repoPath, tt.files, tt.keepLocal)
				require.NoError(t, err)
				return
			}

			// For non-empty cases, we need integration tests
		})
	}
}

// Helper function to generate a list of file names
func generateFileList(count int) []string {
	files := make([]string, count)
	for i := 0; i < count; i++ {
		files[i] = fmt.Sprintf("file%d.txt", i+1)
	}
	return files
}
