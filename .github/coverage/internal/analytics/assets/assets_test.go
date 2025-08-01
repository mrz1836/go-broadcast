package assets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AssetsTestSuite provides test suite for assets package
type AssetsTestSuite struct {
	suite.Suite

	tempDir string
}

// SetupTest creates temporary directory for each test
func (suite *AssetsTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "assets_test_*")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

// TearDownTest cleans up temporary directory after each test
func (suite *AssetsTestSuite) TearDownTest() {
	if suite.tempDir != "" {
		err := os.RemoveAll(suite.tempDir)
		suite.Require().NoError(err)
	}
}

// TestCopyAssetsToSuccess tests successful asset copying
func (suite *AssetsTestSuite) TestCopyAssetsToSuccess() {
	ctx := context.Background()
	_ = ctx // Context available for future use

	err := CopyAssetsTo(suite.tempDir)
	suite.Require().NoError(err)

	// Verify assets directory was created
	assetsDir := filepath.Join(suite.tempDir, "assets")
	stat, err := os.Stat(assetsDir)
	suite.Require().NoError(err)
	suite.True(stat.IsDir())

	// Verify some expected files exist
	expectedFiles := []string{
		"css/coverage.css",
		"images/favicon.ico",
		"images/favicon.svg",
		"images/favicon-16.svg",
		"site.webmanifest",
	}

	for _, expectedFile := range expectedFiles {
		filePath := filepath.Join(assetsDir, expectedFile)
		_, err := os.Stat(filePath)
		suite.NoError(err, "Expected file %s should exist", expectedFile)
	}
}

// TestCopyAssetsToInvalidDirectory tests copying to invalid directory
func (suite *AssetsTestSuite) TestCopyAssetsToInvalidDirectory() {
	// Try to copy to a file instead of directory
	tempFile := filepath.Join(suite.tempDir, "not_a_directory")
	err := os.WriteFile(tempFile, []byte("test"), 0o600)
	suite.Require().NoError(err)

	err = CopyAssetsTo(tempFile)
	suite.Require().Error(err)
	suite.Contains(err.Error(), "creating assets directory")
}

// TestCopyAssetsToPermissionDenied tests copying with permission issues
func (suite *AssetsTestSuite) TestCopyAssetsToPermissionDenied() {
	if os.Getuid() == 0 {
		suite.T().Skip("Skipping permission test when running as root")
	}

	// Create a directory with no write permissions
	restrictedDir := filepath.Join(suite.tempDir, "restricted")
	err := os.MkdirAll(restrictedDir, 0o400) // Read-only, more restrictive
	suite.Require().NoError(err)

	err = CopyAssetsTo(restrictedDir)
	suite.Error(err)
}

// TestCopyAssetsToExistingDirectory tests copying to existing directory
func (suite *AssetsTestSuite) TestCopyAssetsToExistingDirectory() {
	// Create assets directory first
	assetsDir := filepath.Join(suite.tempDir, "assets")
	err := os.MkdirAll(assetsDir, 0o750)
	suite.Require().NoError(err)

	// Copy should still work
	err = CopyAssetsTo(suite.tempDir)
	suite.NoError(err)
}

// TestGetAssetSuccess tests successful asset retrieval
func (suite *AssetsTestSuite) TestGetAssetSuccess() {
	testCases := []struct {
		name          string
		assetPath     string
		expectError   bool
		expectContent bool
	}{
		{
			name:          "CSS file",
			assetPath:     "css/coverage.css",
			expectError:   false,
			expectContent: true,
		},
		{
			name:          "Favicon ICO",
			assetPath:     "images/favicon.ico",
			expectError:   false,
			expectContent: true,
		},
		{
			name:          "Favicon SVG",
			assetPath:     "images/favicon.svg",
			expectError:   false,
			expectContent: true,
		},
		{
			name:          "Web manifest",
			assetPath:     "site.webmanifest",
			expectError:   false,
			expectContent: true,
		},
		{
			name:        "Non-existent file",
			assetPath:   "non-existent.txt",
			expectError: true,
		},
		{
			name:        "Empty path",
			assetPath:   "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			data, err := GetAsset(tc.assetPath)

			if tc.expectError {
				suite.Require().Error(err)
				suite.Nil(data)
				suite.Contains(err.Error(), "reading asset")
			} else {
				suite.NoError(err)
				if tc.expectContent {
					suite.NotEmpty(data, "Asset content should not be empty")
				}
			}
		})
	}
}

// TestAssetExistsSuccess tests asset existence checks
func (suite *AssetsTestSuite) TestAssetExistsSuccess() {
	testCases := []struct {
		name      string
		assetPath string
		exists    bool
	}{
		{
			name:      "CSS file exists",
			assetPath: "css/coverage.css",
			exists:    true,
		},
		{
			name:      "Favicon ICO exists",
			assetPath: "images/favicon.ico",
			exists:    true,
		},
		{
			name:      "Favicon SVG exists",
			assetPath: "images/favicon.svg",
			exists:    true,
		},
		{
			name:      "Non-existent file",
			assetPath: "non-existent.txt",
			exists:    false,
		},
		{
			name:      "Empty path",
			assetPath: "",
			exists:    false,
		},
		{
			name:      "Directory path",
			assetPath: "css",
			exists:    true,
		},
		{
			name:      "Root path",
			assetPath: ".",
			exists:    true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			exists := AssetExists(tc.assetPath)
			suite.Equal(tc.exists, exists, "Asset existence check failed for %s", tc.assetPath)
		})
	}
}

// TestListAssetsSuccess tests listing all assets
func (suite *AssetsTestSuite) TestListAssetsSuccess() {
	assets, err := ListAssets()
	suite.Require().NoError(err)
	suite.NotEmpty(assets, "Assets list should not be empty")

	// Verify expected assets are present
	expectedAssets := []string{
		"css/coverage.css",
		"images/favicon.ico",
		"images/favicon.svg",
		"images/favicon-16.svg",
		"site.webmanifest",
	}

	assetMap := make(map[string]bool)
	for _, asset := range assets {
		assetMap[asset] = true
	}

	for _, expected := range expectedAssets {
		suite.True(assetMap[expected], "Expected asset %s should be in list", expected)
	}

	// Verify no directories are included
	for _, asset := range assets {
		suite.False(strings.HasSuffix(asset, "/"), "Asset list should not contain directories: %s", asset)
	}
}

// TestListAssetsContent tests asset content validation
func (suite *AssetsTestSuite) TestListAssetsContent() {
	assets, err := ListAssets()
	suite.Require().NoError(err)

	// Verify each listed asset can be retrieved
	for _, asset := range assets {
		suite.Run("Asset_"+asset, func() {
			// Asset should exist
			suite.True(AssetExists(asset), "Listed asset %s should exist", asset)

			// Asset should be retrievable
			data, err := GetAsset(asset)
			suite.Require().NoError(err, "Listed asset %s should be retrievable", asset)
			suite.NotEmpty(data, "Listed asset %s should have content", asset)
		})
	}
}

// TestAssetContentValidation tests that assets have expected content types
func (suite *AssetsTestSuite) TestAssetContentValidation() {
	testCases := []struct {
		name         string
		assetPath    string
		contentCheck func([]byte) bool
		description  string
	}{
		{
			name:      "CSS file has CSS content",
			assetPath: "css/coverage.css",
			contentCheck: func(data []byte) bool {
				content := string(data)
				return strings.Contains(content, "body") || strings.Contains(content, ".") || strings.Contains(content, "{")
			},
			description: "CSS file should contain CSS syntax",
		},
		{
			name:      "Web manifest has JSON content",
			assetPath: "site.webmanifest",
			contentCheck: func(data []byte) bool {
				content := string(data)
				return strings.Contains(content, "{") && strings.Contains(content, "}")
			},
			description: "Web manifest should contain JSON syntax",
		},
		{
			name:      "SVG file has SVG content",
			assetPath: "images/favicon.svg",
			contentCheck: func(data []byte) bool {
				content := string(data)
				return strings.Contains(content, "<svg") || strings.Contains(content, "svg")
			},
			description: "SVG file should contain SVG markup",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			data, err := GetAsset(tc.assetPath)
			suite.Require().NoError(err)
			suite.Require().NotEmpty(data)

			suite.True(tc.contentCheck(data), tc.description)
		})
	}
}

// TestEmbedFSIntegrity tests that embedded FS is properly configured
func (suite *AssetsTestSuite) TestEmbedFSIntegrity() {
	// Test that FS is not nil and can be used
	suite.Require().NotNil(FS, "Embedded FS should not be nil")

	// Test that we can read from the FS directly
	entries, err := FS.ReadDir(".")
	suite.Require().NoError(err)
	suite.NotEmpty(entries, "Root directory should have entries")

	// Verify expected directories exist
	expectedDirs := []string{"css", "images"}
	foundDirs := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			foundDirs[entry.Name()] = true
		}
	}

	for _, expectedDir := range expectedDirs {
		suite.True(foundDirs[expectedDir], "Expected directory %s should exist", expectedDir)
	}
}

// TestConcurrentAccess tests concurrent access to assets
func (suite *AssetsTestSuite) TestConcurrentAccess() {
	const numGoroutines = 10
	const numOperations = 100

	errChan := make(chan error, numGoroutines*numOperations)
	doneChan := make(chan struct{}, numGoroutines)

	// Start multiple goroutines accessing assets concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { doneChan <- struct{}{} }()

			for j := 0; j < numOperations; j++ {
				// Test different operations
				switch j % 4 {
				case 0:
					_, err := GetAsset("css/coverage.css")
					if err != nil {
						errChan <- err
					}
				case 1:
					exists := AssetExists("images/favicon.ico")
					if !exists {
						errChan <- assert.AnError
					}
				case 2:
					_, err := ListAssets()
					if err != nil {
						errChan <- err
					}
				case 3:
					// Quick copy operation
					tmpDir, err := os.MkdirTemp("", "concurrent_test_*")
					if err != nil {
						errChan <- err
						continue
					}
					err = CopyAssetsTo(tmpDir)
					if err != nil {
						errChan <- err
					}
					_ = os.RemoveAll(tmpDir)
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-doneChan
	}

	// Check for errors
	close(errChan)
	for err := range errChan {
		suite.T().Errorf("Concurrent access error: %v", err)
	}
}

// TestAssetSizes tests that assets have reasonable sizes
func (suite *AssetsTestSuite) TestAssetSizes() {
	assets, err := ListAssets()
	suite.Require().NoError(err)

	for _, asset := range assets {
		suite.Run("Size_"+asset, func() {
			data, err := GetAsset(asset)
			suite.Require().NoError(err)

			// Assets should not be empty
			suite.NotEmpty(data, "Asset %s should not be empty", asset)

			// Assets should not be unreasonably large (1MB limit)
			suite.Less(len(data), 1024*1024, "Asset %s should not exceed 1MB", asset)
		})
	}
}

// TestRun runs the test suite
func TestAssetsTestSuite(t *testing.T) {
	suite.Run(t, new(AssetsTestSuite))
}

// Benchmark tests for performance validation
func BenchmarkGetAsset(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAsset("css/coverage.css")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAssetExists(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = AssetExists("css/coverage.css")
	}
}

func BenchmarkListAssets(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ListAssets()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCopyAssetsTo(b *testing.B) {
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		tempDir, err := os.MkdirTemp("", "benchmark_*")
		if err != nil {
			b.Fatal(err)
		}

		b.StartTimer()
		err = CopyAssetsTo(tempDir)
		b.StopTimer()

		if err != nil {
			b.Fatal(err)
		}

		_ = os.RemoveAll(tempDir)
	}
}
