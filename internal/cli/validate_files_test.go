package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// TestValidateSourceFilesExist tests the validateSourceFilesExist function
func TestValidateSourceFilesExist(t *testing.T) {
	// This function creates its own GitHub client internally,
	// so we can't easily mock it without refactoring.
	// These tests demonstrate what we would test if the function accepted a client parameter.

	baseConfig := &config.Config{
		Groups: []config.Group{{
			Source: config.SourceConfig{
				Repo:   "org/source-repo",
				Branch: "master",
			},
			Targets: []config.TargetConfig{
				{
					Repo: "org/target1",
					Files: []config.FileMapping{
						{Src: "README.md", Dest: "README.md"},
						{Src: "LICENSE", Dest: "LICENSE"},
					},
				},
				{
					Repo: "org/target2",
					Files: []config.FileMapping{
						{Src: "README.md", Dest: "docs/README.md"},
						{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
					},
				},
			},
		}},
	}

	testCases := []struct {
		name         string
		config       *config.Config
		setupMocks   func(*gh.MockClient)
		clientError  error
		verifyOutput func(*testing.T, string)
	}{
		{
			name: "No source files to validate",
			config: &config.Config{
				Groups: []config.Group{{
					Source: config.SourceConfig{
						Repo:   "org/source-repo",
						Branch: "master",
					},
					Targets: []config.TargetConfig{},
				}},
			},
			verifyOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "No source files to validate")
			},
		},
		{
			name:        "GitHub client unavailable",
			config:      baseConfig,
			clientError: fmt.Errorf("client creation failed"), //nolint:err113 // test-only error
			verifyOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "Skipping source file validation (GitHub client unavailable)")
			},
		},
		{
			name:   "All source files exist",
			config: baseConfig,
			setupMocks: func(mockClient *gh.MockClient) {
				// All files exist
				mockClient.On("GetFile", mock.Anything, "org/source-repo", "README.md", "master").
					Return(&gh.FileContent{Path: "README.md", Content: []byte("# README")}, nil)
				mockClient.On("GetFile", mock.Anything, "org/source-repo", "LICENSE", "master").
					Return(&gh.FileContent{Path: "LICENSE", Content: []byte("MIT License")}, nil)
				mockClient.On("GetFile", mock.Anything, "org/source-repo", ".github/workflows/ci.yml", "master").
					Return(&gh.FileContent{Path: ".github/workflows/ci.yml", Content: []byte("name: CI")}, nil)
			},
			verifyOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "All source files exist (3/3)")
			},
		},
		{
			name:   "Some source files missing",
			config: baseConfig,
			setupMocks: func(mockClient *gh.MockClient) {
				// README exists
				mockClient.On("GetFile", mock.Anything, "org/source-repo", "README.md", "master").
					Return(&gh.FileContent{Path: "README.md", Content: []byte("# README")}, nil)
				// LICENSE missing
				mockClient.On("GetFile", mock.Anything, "org/source-repo", "LICENSE", "master").
					Return(nil, fmt.Errorf("file not found")) //nolint:err113 // test-only error
				// CI workflow exists
				mockClient.On("GetFile", mock.Anything, "org/source-repo", ".github/workflows/ci.yml", "master").
					Return(&gh.FileContent{Path: ".github/workflows/ci.yml", Content: []byte("name: CI")}, nil)
			},
			verifyOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "Source file not found: LICENSE")
				assert.Contains(t, output, "Some source files missing (2/3 found)")
			},
		},
		{
			name:   "File check errors",
			config: baseConfig,
			setupMocks: func(mockClient *gh.MockClient) {
				// README exists
				mockClient.On("GetFile", mock.Anything, "org/source-repo", "README.md", "master").
					Return(&gh.FileContent{Path: "README.md", Content: []byte("# README")}, nil)
				// LICENSE - API error
				mockClient.On("GetFile", mock.Anything, "org/source-repo", "LICENSE", "master").
					Return(nil, fmt.Errorf("API rate limit exceeded")) //nolint:err113 // test-only error
				// CI workflow - permission error
				mockClient.On("GetFile", mock.Anything, "org/source-repo", ".github/workflows/ci.yml", "master").
					Return(nil, fmt.Errorf("permission denied")) //nolint:err113 // test-only error
			},
			verifyOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "Failed to check source file 'LICENSE': API rate limit exceeded")
				assert.Contains(t, output, "Failed to check source file '.github/workflows/ci.yml': permission denied")
				assert.Contains(t, output, "Some source files missing (1/3 found)")
			},
		},
		{
			name: "Duplicate source files across targets",
			config: &config.Config{
				Groups: []config.Group{{
					Name: "test-group",
					ID:   "test-group-1",
					Source: config.SourceConfig{
						Repo:   "org/source-repo",
						Branch: "master",
					},
					Targets: []config.TargetConfig{
						{
							Repo: "org/target1",
							Files: []config.FileMapping{
								{Src: "README.md", Dest: "README.md"},
								{Src: "README.md", Dest: "docs/README.md"}, // Same source, different dest
							},
						},
						{
							Repo: "org/target2",
							Files: []config.FileMapping{
								{Src: "README.md", Dest: "README.md"}, // Same source again
							},
						},
					},
				}},
			},
			setupMocks: func(mockClient *gh.MockClient) {
				// Should only check README.md once due to deduplication
				mockClient.On("GetFile", mock.Anything, "org/source-repo", "README.md", "master").
					Return(&gh.FileContent{Path: "README.md", Content: []byte("# README")}, nil).Once()
			},
			verifyOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "All source files exist (1/1)")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock client if needed
			var mockClient *gh.MockClient
			if tc.setupMocks != nil {
				mockClient = new(gh.MockClient)
				tc.setupMocks(mockClient)
			}

			// If client error is expected, handle the case where no client is available
			if tc.clientError != nil {
				// For this test case, we test the main function which handles client creation errors
				// Since we can't easily force client creation to fail in the test,
				// we'll skip this specific test case for now
				t.Skip("Cannot easily simulate client creation failure in test")
				return
			}

			// Capture output (thread-safe)
			scope := output.CaptureOutput()
			defer scope.Restore()

			// Call the function with mock client
			ctx := context.Background()
			validateSourceFilesExistWithClient(ctx, tc.config, mockClient)

			// Verify output - combine stdout and stderr
			outputStr := scope.Stdout.String() + scope.Stderr.String()
			tc.verifyOutput(t, outputStr)

			// Verify mock expectations
			if mockClient != nil {
				mockClient.AssertExpectations(t)
			}
		})
	}
}

// TestValidateSourceFilesExistEdgeCases tests edge cases
func TestValidateSourceFilesExistEdgeCases(t *testing.T) {
	t.Run("Empty file paths", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Name: "test-group",
				ID:   "test-group-1",
				Source: config.SourceConfig{
					Repo:   "org/source-repo",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "org/target1",
						Files: []config.FileMapping{
							{Src: "", Dest: "README.md"},              // Empty source path
							{Src: "valid-file.txt", Dest: "file.txt"}, // Valid file to ensure processing continues
						},
					},
				},
			}},
		}

		// Create mock client
		mockClient := gh.NewMockClient()

		// Setup mock expectations
		// Both empty path and valid file will be checked (current behavior)
		// Empty path will fail, but validation continues
		mockClient.On("GetFile", mock.Anything, "org/source-repo", "", "master").
			Return(nil, gh.ErrFileNotFound)
		mockClient.On("GetFile", mock.Anything, "org/source-repo", "valid-file.txt", "master").
			Return(&gh.FileContent{
				Path:    "valid-file.txt",
				Content: []byte("test content"),
			}, nil)

		// Capture output (thread-safe)
		scope := output.CaptureOutput()
		defer scope.Restore()

		// Run the validation
		ctx := context.Background()
		validateSourceFilesExistWithClient(ctx, cfg, mockClient)

		// Verify output - combine stdout and stderr
		outputStr := scope.Stdout.String() + scope.Stderr.String()

		// Verify that empty paths are reported as not found and validation continues
		assert.Contains(t, outputStr, "Source file not found: ", "Should report empty path as not found")
		assert.Contains(t, outputStr, "Some source files missing (1/2 found)", "Should report partial success")

		// Verify mock expectations - both paths should be checked
		mockClient.AssertExpectations(t)
	})

	t.Run("Special characters in file paths", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Name: "test-group",
				ID:   "test-group-1",
				Source: config.SourceConfig{
					Repo:   "org/source-repo",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "org/target1",
						Files: []config.FileMapping{
							{Src: "path/with spaces/file.txt", Dest: "file.txt"},
							{Src: "path/with-dashes/file.txt", Dest: "file.txt"},
							{Src: "path/with_underscores/file.txt", Dest: "file.txt"},
							{Src: "path/with.dots/file.txt", Dest: "file.txt"},
						},
					},
				},
			}},
		}

		// Create mock client
		mockClient := new(gh.MockClient)

		// Setup mock expectations - all special character paths should work
		mockClient.On("GetFile", mock.Anything, "org/source-repo", "path/with spaces/file.txt", "master").
			Return(&gh.FileContent{Path: "path/with spaces/file.txt", Content: []byte("content with spaces")}, nil)
		mockClient.On("GetFile", mock.Anything, "org/source-repo", "path/with-dashes/file.txt", "master").
			Return(&gh.FileContent{Path: "path/with-dashes/file.txt", Content: []byte("content with dashes")}, nil)
		mockClient.On("GetFile", mock.Anything, "org/source-repo", "path/with_underscores/file.txt", "master").
			Return(&gh.FileContent{Path: "path/with_underscores/file.txt", Content: []byte("content with underscores")}, nil)
		mockClient.On("GetFile", mock.Anything, "org/source-repo", "path/with.dots/file.txt", "master").
			Return(&gh.FileContent{Path: "path/with.dots/file.txt", Content: []byte("content with dots")}, nil)

		// Capture output (thread-safe)
		scope := output.CaptureOutput()
		defer scope.Restore()

		// Call the function with mock client
		ctx := context.Background()
		validateSourceFilesExistWithClient(ctx, cfg, mockClient)

		// Verify output - combine stdout and stderr
		outputStr := scope.Stdout.String() + scope.Stderr.String()

		// Verify that all files with special characters are found
		assert.Contains(t, outputStr, "All source files exist (4/4)", "Should successfully validate all files with special characters")

		// Verify mock expectations
		mockClient.AssertExpectations(t)
	})

	t.Run("Very long file paths", func(t *testing.T) {
		longPath := "very/deep/nested/directory/structure/that/goes/on/and/on/and/on/with/many/levels/file.txt"
		cfg := &config.Config{
			Groups: []config.Group{{
				Name: "test-group",
				ID:   "test-group-1",
				Source: config.SourceConfig{
					Repo:   "org/source-repo",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "org/target1",
						Files: []config.FileMapping{
							{Src: longPath, Dest: "file.txt"},
						},
					},
				},
			}},
		}

		// Create mock client
		mockClient := new(gh.MockClient)

		// Setup mock expectations - long path should work
		mockClient.On("GetFile", mock.Anything, "org/source-repo", longPath, "master").
			Return(&gh.FileContent{Path: longPath, Content: []byte("content from very long path")}, nil)

		// Capture output (thread-safe)
		scope := output.CaptureOutput()
		defer scope.Restore()

		// Call the function with mock client
		ctx := context.Background()
		validateSourceFilesExistWithClient(ctx, cfg, mockClient)

		// Verify output - combine stdout and stderr
		outputStr := scope.Stdout.String() + scope.Stderr.String()

		// Verify that long paths are handled correctly
		assert.Contains(t, outputStr, "All source files exist (1/1)", "Should successfully validate files with very long paths")

		// Verify mock expectations
		mockClient.AssertExpectations(t)
	})
}

// TestValidateCommandWithMockedGitHub demonstrates testing with proper mocking and dependency injection
func TestValidateCommandWithMockedGitHub(t *testing.T) {
	// This test demonstrates testing with dependency injection now that both validate functions
	// accept a GitHub client as a parameter.

	t.Run("Test with dependency injection", func(t *testing.T) {
		// Create test configuration
		cfg := &config.Config{
			Groups: []config.Group{{
				Name: "test-group",
				ID:   "test-group-1",
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{
						Repo: "org/target1",
						Files: []config.FileMapping{
							{Src: "README.md", Dest: "README.md"},
						},
					},
				},
			}},
		}

		// Create mock client
		mockClient := new(gh.MockClient)

		// Setup expectations for validateRepositoryAccessibilityWithClient
		mockClient.On("GetBranch", mock.Anything, "org/source", "master").
			Return(&gh.Branch{Name: "master", Commit: struct {
				SHA string `json:"sha"`
				URL string `json:"url"`
			}{SHA: "abc123"}}, nil)
		mockClient.On("ListBranches", mock.Anything, "org/target1").
			Return([]gh.Branch{{Name: "master", Commit: struct {
				SHA string `json:"sha"`
				URL string `json:"url"`
			}{SHA: "def456"}}}, nil)

		// Setup expectations for validateSourceFilesExistWithClient
		mockClient.On("GetFile", mock.Anything, "org/source", "README.md", "master").
			Return(&gh.FileContent{Path: "README.md", Content: []byte("# README")}, nil)

		// Capture output (thread-safe)
		scope := output.CaptureOutput()
		defer scope.Restore()

		ctx := context.Background()

		// Test validateRepositoryAccessibilityWithClient
		err := validateRepositoryAccessibilityWithClient(ctx, cfg, mockClient, false)
		require.NoError(t, err, "Repository accessibility validation should succeed")

		// Test validateSourceFilesExistWithClient
		validateSourceFilesExistWithClient(ctx, cfg, mockClient)

		// Verify output contains success messages
		outputStr := scope.Stdout.String() + scope.Stderr.String()
		assert.Contains(t, outputStr, "Source repository accessible: org/source", "Should report source repo as accessible")
		assert.Contains(t, outputStr, "Target repository accessible: org/target1", "Should report target repo as accessible")
		assert.Contains(t, outputStr, "All source files exist", "Should report all source files exist")

		// Verify expectations were met
		mockClient.AssertExpectations(t)
	})
}

// TestValidateIntegration tests the validate command integration
// This could be expanded to test with a real GitHub API in integration tests
func TestValidateIntegration(t *testing.T) {
	t.Run("Integration test with real API", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping integration test in short mode")
		}

		// This would require:
		// 1. Valid GitHub credentials
		// 2. Access to test repositories
		// 3. Proper test data setup

		t.Skip("Skipping integration test - requires GitHub API access")
	})
}
