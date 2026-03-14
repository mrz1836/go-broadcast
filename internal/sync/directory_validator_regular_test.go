package sync

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

func TestNewDirectoryValidator(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)

	assert.NotNil(t, validator)
	assert.Equal(t, logger, validator.logger)
	assert.Equal(t, 1000, validator.performanceThresholds.MaxAPICalls)
	assert.InDelta(t, 0.8, validator.performanceThresholds.MinCacheHitRate, 0.001)
	assert.Equal(t, int64(500), validator.performanceThresholds.MaxMemoryMB)
	assert.Equal(t, 10*time.Minute, validator.performanceThresholds.MaxProcessingTime)
	assert.InDelta(t, 1.0, validator.performanceThresholds.MinThroughputMBps, 0.001)
}

func TestDirectoryValidator_SetPerformanceThresholds(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)

	newThresholds := PerformanceThresholds{
		MaxAPICalls:       2000,
		MinCacheHitRate:   0.9,
		MaxMemoryMB:       1000,
		MaxProcessingTime: 5 * time.Minute,
		MinThroughputMBps: 2.0,
	}

	validator.SetPerformanceThresholds(newThresholds)

	assert.Equal(t, newThresholds, validator.performanceThresholds)
}

func TestDirectoryValidator_ValidateAPIEfficiency(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	// Test API calls within limit
	result, err := validator.ValidateAPIEfficiency(ctx, 50, 100)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Test API calls exceeding limit
	result, err = validator.ValidateAPIEfficiency(ctx, 150, 100)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "API call limit exceeded")
}

func TestDirectoryValidator_ValidateCacheUtilization(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	// Test good cache hit rate
	result, err := validator.ValidateCacheUtilization(ctx, 80, 20, 0.75)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Test poor cache hit rate
	result, err = validator.ValidateCacheUtilization(ctx, 20, 80, 0.75)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Cache hit rate")

	// Test with zero total requests (treated as 0% hit rate)
	result, err = validator.ValidateCacheUtilization(ctx, 0, 0, 0.75)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "Cache hit rate 0.00% is below expected 75.00%")
}

func TestDirectoryValidator_ValidateMemoryUsage(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	// Test memory usage within limit
	result, err := validator.ValidateMemoryUsage(ctx, 100*1024*1024, 500*1024*1024) // 100MB used, 500MB limit
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Test memory usage exceeding limit
	result, err = validator.ValidateMemoryUsage(ctx, 600*1024*1024, 500*1024*1024) // 600MB used, 500MB limit
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Memory usage")
}

func TestDirectoryValidator_ValidateProgressReporting(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	// Test matching counts
	progressUpdates := []string{"Processing file 1", "Processing file 2", "Processing file 3"}
	result, err := validator.ValidateProgressReporting(ctx, 3, 3, progressUpdates)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)

	// Test mismatched counts
	result, err = validator.ValidateProgressReporting(ctx, 5, 3, progressUpdates)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Progress reporting mismatch")

	// Test missing progress updates
	result, err = validator.ValidateProgressReporting(ctx, 3, 3, []string{})
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Contains(t, result.Errors[0], "No progress updates were generated")
}

func TestDirectoryValidator_ValidateValidationPerformanceMetrics(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	opts := ValidationOptions{
		PerformanceThresholds: &PerformanceThresholds{
			MaxAPICalls:       100,
			MinCacheHitRate:   0.8,
			MaxMemoryMB:       500,
			MaxProcessingTime: time.Minute,
			MinThroughputMBps: 1.0,
		},
	}

	// Test good performance metrics
	goodMetrics := ValidationPerformanceMetrics{
		APICalls:       50,
		CacheHits:      80,
		CacheMisses:    20,
		CacheHitRate:   0.8,
		MemoryUsage:    100 * 1024 * 1024, // 100MB
		ProcessingTime: 30 * time.Second,
		FilesProcessed: 100,
		ThroughputMBps: 2.0,
	}

	result, err := validator.ValidateValidationPerformanceMetrics(ctx, goodMetrics, opts)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.True(t, result.APICallsOptimized)
	assert.True(t, result.CacheHitRateGood)
	assert.True(t, result.MemoryUsageAcceptable)
	assert.True(t, result.ProcessingTimeGood)

	// Test poor performance metrics
	poorMetrics := ValidationPerformanceMetrics{
		APICalls:       150, // Exceeds limit
		CacheHits:      20,
		CacheMisses:    80,
		CacheHitRate:   0.2,               // Below threshold
		MemoryUsage:    600 * 1024 * 1024, // 600MB, exceeds limit
		ProcessingTime: 2 * time.Minute,   // Exceeds limit
		FilesProcessed: 10,
		ThroughputMBps: 0.5, // Below threshold
	}

	result, err = validator.ValidateValidationPerformanceMetrics(ctx, poorMetrics, opts)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.False(t, result.APICallsOptimized)
	assert.False(t, result.CacheHitRateGood)
	assert.False(t, result.MemoryUsageAcceptable)
	assert.False(t, result.ProcessingTimeGood)
	assert.NotEmpty(t, result.Recommendations)
}

func TestDirectoryValidator_ValidateTransformApplication(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	originalFiles := map[string]string{
		"file1.txt": "original content 1",
		"file2.txt": "original content 2",
	}

	transformedFiles := map[string]string{
		"file1.txt": "transformed content 1",
		"file2.txt": "transformed content 2",
	}

	transform := config.Transform{
		RepoName: true,
		Variables: map[string]string{
			"original": "transformed",
		},
	}

	opts := ValidationOptions{
		CheckTransforms: true,
	}

	result, err := validator.ValidateTransformApplication(ctx, originalFiles, transformedFiles, transform, opts)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 2, result.Summary.TotalFiles)
	assert.Equal(t, 2, result.Summary.ValidFiles)
}

func TestDirectoryValidator_ValidateFileIntegrity(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	sourceFiles := map[string]string{
		"file1.txt": "content 1",
		"file2.txt": "content 2",
		"file3.txt": "content 3",
	}

	// Destination files with matching content
	destFiles := map[string]string{
		"file1.txt": "content 1",
		"file2.txt": "content 2",
		"file3.txt": "content 3",
	}

	opts := ValidationOptions{
		CheckIntegrity: true,
		CheckContent:   true,
		MaxConcurrency: 2,
	}

	result, err := validator.ValidateFileIntegrity(ctx, sourceFiles, destFiles, opts)
	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, 3, result.Summary.TotalFiles)
	assert.Equal(t, 3, result.Summary.ValidFiles)
	assert.Equal(t, 0, result.Summary.InvalidFiles)

	// Test with mismatched content
	destFilesMismatch := map[string]string{
		"file1.txt": "different content 1",
		"file2.txt": "content 2",
		"file3.txt": "content 3",
	}

	result, err = validator.ValidateFileIntegrity(ctx, sourceFiles, destFilesMismatch, opts)
	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.Equal(t, 3, result.Summary.TotalFiles)
	assert.Equal(t, 2, result.Summary.ValidFiles)
	assert.Equal(t, 1, result.Summary.InvalidFiles)
	assert.Equal(t, 1, result.Summary.ContentMismatches)
}

func TestDirectoryValidator_CalculateFileChecksum(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)

	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "test content for checksum"

	err := os.WriteFile(testFile, []byte(content), 0o600)
	require.NoError(t, err)

	// Calculate checksum
	checksum, err := validator.calculateFileChecksum(testFile)
	require.NoError(t, err)

	// Verify checksum is correct
	expectedSum := sha256.Sum256([]byte(content))
	expectedChecksum := fmt.Sprintf("%x", expectedSum)
	assert.Equal(t, expectedChecksum, checksum)

	// Test with non-existent file
	_, err = validator.calculateFileChecksum(filepath.Join(tempDir, "nonexistent.txt"))
	require.Error(t, err)
}

func TestDirectoryValidator_IsHidden(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"regular file", "/path/to/file.txt", false},
		{"hidden file in root", "/.hidden", true},
		{"hidden file in subdirectory", "/path/to/.hidden", true},
		{"file in hidden directory", "/path/.hidden/file.txt", true},
		{"multiple hidden components", "/.config/.local/file.txt", true},
		{"windows-style hidden", "/path/to/file.txt", false}, // This test may need OS-specific logic
		{"dot file", "/home/user/.bashrc", true},
		{"double dot (parent dir)", "/path/../file.txt", false},
		{"current dir", "/path/./file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isHidden(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirectoryValidator_ValidateTransformResult(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)

	tests := []struct {
		name        string
		original    string
		transformed string
		transform   config.Transform
		expectError bool
	}{
		{
			name:        "valid variable transform with placeholders",
			original:    "Hello {{world}}",
			transformed: "Hello Universe",
			transform: config.Transform{
				Variables: map[string]string{
					"world": "Universe",
				},
			},
			expectError: false,
		},
		{
			name:        "invalid variable transform - placeholder not substituted",
			original:    "Hello {{world}}",
			transformed: "Hello {{world}}", // Placeholder still present
			transform: config.Transform{
				Variables: map[string]string{
					"world": "Universe",
				},
			},
			expectError: true,
		},
		{
			name:        "valid repo name transform",
			original:    "myrepo/file.txt",
			transformed: "transformedrepo/file.txt",
			transform: config.Transform{
				RepoName: true,
			},
			expectError: false,
		},
		{
			name:        "invalid repo name transform - no change",
			original:    "myrepo/file.txt",
			transformed: "myrepo/file.txt", // Content unchanged despite RepoName=true
			transform: config.Transform{
				RepoName: true,
			},
			expectError: true,
		},
		{
			name:        "valid transform without variables or repo name",
			original:    "Hello World",
			transformed: "Hello Universe",
			transform: config.Transform{
				Variables: map[string]string{},
			},
			expectError: false, // No validation rules to violate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateTransformResult(tt.original, tt.transformed, tt.transform, "test-file.txt")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultValidationOptions(t *testing.T) {
	opts := DefaultValidationOptions()

	assert.True(t, opts.CheckContent)
	assert.True(t, opts.CheckStructure)
	assert.True(t, opts.CheckExclusions)
	assert.True(t, opts.CheckTransforms) // Default is true in the implementation
	assert.False(t, opts.CheckPerformance)
	assert.True(t, opts.CheckIntegrity)
	assert.False(t, opts.IgnoreHiddenFiles)
	assert.Equal(t, runtime.NumCPU(), opts.MaxConcurrency) // Uses runtime.NumCPU() by default
	assert.True(t, opts.DetailedErrorReporting)
	assert.Nil(t, opts.PerformanceThresholds)
}

func TestValidationResult_Summary(t *testing.T) {
	result := ValidationResult{
		Valid:  true,
		Errors: []string{},
		Summary: ValidationSummary{
			TotalFiles:        10,
			ValidFiles:        8,
			InvalidFiles:      2,
			MissingFiles:      1,
			ExtraFiles:        0,
			ContentMismatches: 1,
			TransformErrors:   0,
			StructureErrors:   1,
			ExclusionErrors:   0,
			Duration:          time.Second * 30,
			BytesValidated:    1024 * 1024, // 1MB
		},
	}

	assert.True(t, result.Valid)
	assert.Equal(t, 10, result.Summary.TotalFiles)
	assert.Equal(t, 8, result.Summary.ValidFiles)
	assert.Equal(t, 2, result.Summary.InvalidFiles)
	assert.Equal(t, time.Second*30, result.Summary.Duration)
	assert.Equal(t, int64(1024*1024), result.Summary.BytesValidated)
}

func TestPerformanceValidationResult_Metrics(t *testing.T) {
	result := PerformanceValidationResult{
		Valid:                 true,
		APICallsOptimized:     true,
		CacheHitRateGood:      true,
		MemoryUsageAcceptable: true,
		ProcessingTimeGood:    true,
		Metrics: ValidationPerformanceMetrics{
			APICalls:       50,
			CacheHits:      40,
			CacheMisses:    10,
			CacheHitRate:   0.8,
			MemoryUsage:    100 * 1024 * 1024,
			ProcessingTime: time.Second * 30,
			FilesProcessed: 100,
			ThroughputMBps: 3.33,
		},
		Thresholds: PerformanceThresholds{
			MaxAPICalls:       100,
			MinCacheHitRate:   0.75,
			MaxMemoryMB:       500,
			MaxProcessingTime: time.Minute,
			MinThroughputMBps: 2.0,
		},
		Recommendations: []string{},
	}

	assert.True(t, result.Valid)
	assert.Equal(t, 50, result.Metrics.APICalls)
	assert.InDelta(t, 0.8, result.Metrics.CacheHitRate, 0.001)
	assert.Equal(t, int64(100*1024*1024), result.Metrics.MemoryUsage)
	assert.Equal(t, time.Second*30, result.Metrics.ProcessingTime)
	assert.Equal(t, 100, result.Thresholds.MaxAPICalls)
	assert.Empty(t, result.Recommendations)
}

func TestFileValidationError_Details(t *testing.T) {
	err := FileValidationError{
		FilePath:  "/path/to/file.txt",
		ErrorType: "content_mismatch",
		Expected:  "expected content",
		Actual:    "actual content",
		Details:   "Content hash mismatch detected",
		Suggestions: []string{
			"Check if transform was applied correctly",
			"Verify file encoding matches expected format",
		},
	}

	assert.Equal(t, "/path/to/file.txt", err.FilePath)
	assert.Equal(t, "content_mismatch", err.ErrorType)
	assert.Equal(t, "expected content", err.Expected)
	assert.Equal(t, "actual content", err.Actual)
	assert.Equal(t, "Content hash mismatch detected", err.Details)
	assert.Len(t, err.Suggestions, 2)
}

// Test helper functions for creating test scenarios

func TestDirectoryValidator_Threading(_ *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)

	// Test concurrent access to performance thresholds
	done := make(chan bool)

	// Goroutine 1: Reading thresholds
	go func() {
		for i := 0; i < 100; i++ {
			validator.SetPerformanceThresholds(PerformanceThresholds{
				MaxAPICalls: i,
			})
		}
		done <- true
	}()

	// Goroutine 2: Reading thresholds
	go func() {
		for i := 0; i < 100; i++ {
			_ = validator.GetPerformanceThresholds().MaxAPICalls
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Test should complete without data races - no assertions needed
}

func BenchmarkDirectoryValidator_ValidateFileIntegrity(b *testing.B) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	// Create test data
	sourceFiles := make(map[string]string)
	destFiles := make(map[string]string)

	for i := 0; i < 1000; i++ {
		filename := fmt.Sprintf("file_%d.txt", i)
		content := fmt.Sprintf("content for file %d", i)
		sourceFiles[filename] = content
		destFiles[filename] = content
	}

	opts := ValidationOptions{
		CheckIntegrity: true,
		CheckContent:   true,
		MaxConcurrency: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateFileIntegrity(ctx, sourceFiles, destFiles, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestDirectoryValidator_ValidateSyncResults tests the main sync validation function
func TestDirectoryValidator_ValidateSyncResults(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	t.Run("matching directories", func(t *testing.T) {
		// Create temp directories
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create matching files
		err := os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("content1"), 0o600)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(destDir, "file1.txt"), []byte("content1"), 0o600)
		require.NoError(t, err)

		dirMapping := config.DirectoryMapping{
			Src:  sourceDir,
			Dest: destDir,
		}

		opts := ValidationOptions{
			CheckContent:   true,
			CheckStructure: true,
		}

		result, err := validator.ValidateSyncResults(ctx, sourceDir, destDir, dirMapping, opts)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("missing file in destination", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create file only in source
		err := os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("content1"), 0o600)
		require.NoError(t, err)

		dirMapping := config.DirectoryMapping{
			Src:  sourceDir,
			Dest: destDir,
		}

		opts := ValidationOptions{
			CheckStructure: true,
		}

		result, err := validator.ValidateSyncResults(ctx, sourceDir, destDir, dirMapping, opts)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("content mismatch", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create files with different content
		err := os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("content1"), 0o600)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(destDir, "file1.txt"), []byte("different"), 0o600)
		require.NoError(t, err)

		dirMapping := config.DirectoryMapping{
			Src:  sourceDir,
			Dest: destDir,
		}

		opts := ValidationOptions{
			CheckContent:   true,
			CheckStructure: true,
		}

		result, err := validator.ValidateSyncResults(ctx, sourceDir, destDir, dirMapping, opts)
		require.NoError(t, err)
		assert.False(t, result.Valid)
	})

	t.Run("source directory not found", func(t *testing.T) {
		destDir := t.TempDir()

		dirMapping := config.DirectoryMapping{
			Src:  "/nonexistent/source",
			Dest: destDir,
		}

		opts := ValidationOptions{}

		result, err := validator.ValidateSyncResults(ctx, "/nonexistent/source", destDir, dirMapping, opts)
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("destination directory not found", func(t *testing.T) {
		sourceDir := t.TempDir()

		dirMapping := config.DirectoryMapping{
			Src:  sourceDir,
			Dest: "/nonexistent/dest",
		}

		opts := ValidationOptions{}

		result, err := validator.ValidateSyncResults(ctx, sourceDir, "/nonexistent/dest", dirMapping, opts)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestDirectoryValidator_ValidateExclusionCompliance tests exclusion validation
func TestDirectoryValidator_ValidateExclusionCompliance(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	t.Run("no excluded files in destination", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create file in destination that is NOT excluded
		err := os.WriteFile(filepath.Join(destDir, "allowed.txt"), []byte("content"), 0o600)
		require.NoError(t, err)

		dirMapping := config.DirectoryMapping{
			Src:     sourceDir,
			Dest:    destDir,
			Exclude: []string{"*.log", "*.tmp"},
		}

		opts := ValidationOptions{}

		result, err := validator.ValidateExclusionCompliance(ctx, sourceDir, destDir, dirMapping, opts)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("excluded file found in destination", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create excluded file in destination
		err := os.WriteFile(filepath.Join(destDir, "debug.log"), []byte("log content"), 0o600)
		require.NoError(t, err)

		dirMapping := config.DirectoryMapping{
			Src:     sourceDir,
			Dest:    destDir,
			Exclude: []string{"*.log"},
		}

		opts := ValidationOptions{}

		result, err := validator.ValidateExclusionCompliance(ctx, sourceDir, destDir, dirMapping, opts)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Positive(t, result.Summary.ExclusionErrors)
	})

	t.Run("destination directory not found", func(t *testing.T) {
		sourceDir := t.TempDir()

		dirMapping := config.DirectoryMapping{
			Src:  sourceDir,
			Dest: "/nonexistent/dest",
		}

		opts := ValidationOptions{}

		result, err := validator.ValidateExclusionCompliance(ctx, sourceDir, "/nonexistent/dest", dirMapping, opts)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestDirectoryValidator_ValidateDirectoryStructure tests structure validation
func TestDirectoryValidator_ValidateDirectoryStructure(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	t.Run("preserved structure - matching", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create nested directory structure in source
		err := os.MkdirAll(filepath.Join(sourceDir, "sub"), 0o750)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(sourceDir, "sub", "file.txt"), []byte("content"), 0o600)
		require.NoError(t, err)

		// Create same structure in destination
		err = os.MkdirAll(filepath.Join(destDir, "sub"), 0o750)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(destDir, "sub", "file.txt"), []byte("content"), 0o600)
		require.NoError(t, err)

		preserveStructure := true
		dirMapping := config.DirectoryMapping{
			Src:               sourceDir,
			Dest:              destDir,
			PreserveStructure: &preserveStructure,
		}

		opts := ValidationOptions{
			CheckStructure: true,
		}

		result, err := validator.ValidateDirectoryStructure(ctx, sourceDir, destDir, dirMapping, opts)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("flattened structure", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create nested directory structure in source
		err := os.MkdirAll(filepath.Join(sourceDir, "sub"), 0o750)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(sourceDir, "sub", "file.txt"), []byte("content"), 0o600)
		require.NoError(t, err)

		// Create flattened structure in destination
		err = os.WriteFile(filepath.Join(destDir, "file.txt"), []byte("content"), 0o600)
		require.NoError(t, err)

		preserveStructure := false
		dirMapping := config.DirectoryMapping{
			Src:               sourceDir,
			Dest:              destDir,
			PreserveStructure: &preserveStructure,
		}

		opts := ValidationOptions{
			CheckStructure: true,
		}

		result, err := validator.ValidateDirectoryStructure(ctx, sourceDir, destDir, dirMapping, opts)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("source not found", func(t *testing.T) {
		destDir := t.TempDir()

		dirMapping := config.DirectoryMapping{
			Src:  "/nonexistent/source",
			Dest: destDir,
		}

		opts := ValidationOptions{}

		result, err := validator.ValidateDirectoryStructure(ctx, "/nonexistent/source", destDir, dirMapping, opts)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestDirectoryValidator_ValidateAllAspects tests comprehensive validation
func TestDirectoryValidator_ValidateAllAspects(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)
	ctx := context.Background()

	t.Run("all validations pass", func(t *testing.T) {
		sourceDir := t.TempDir()
		destDir := t.TempDir()

		// Create matching files
		err := os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("content"), 0o600)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(destDir, "file.txt"), []byte("content"), 0o600)
		require.NoError(t, err)

		dirMapping := config.DirectoryMapping{
			Src:  sourceDir,
			Dest: destDir,
		}

		opts := ValidationOptions{
			CheckContent:    true,
			CheckStructure:  true,
			CheckExclusions: true,
		}

		result, err := validator.ValidateAllAspects(ctx, sourceDir, destDir, dirMapping, opts)
		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("source directory not found", func(t *testing.T) {
		destDir := t.TempDir()

		dirMapping := config.DirectoryMapping{
			Src:  "/nonexistent/source",
			Dest: destDir,
		}

		opts := ValidationOptions{}

		result, err := validator.ValidateAllAspects(ctx, "/nonexistent/source", destDir, dirMapping, opts)
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

// TestDirectoryValidator_isHidden tests hidden file detection
func TestDirectoryValidator_isHidden(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)

	tests := []struct {
		path     string
		expected bool
	}{
		{"file.txt", false},
		{".hidden", true},
		{".git/config", true},
		{"dir/.hidden", true},
		{"dir/file.txt", false},
		{".github/workflows/test.yml", true},
		{"normal/path/.file", true},
		{".", false},  // Current directory is not hidden
		{"..", false}, // Parent directory is not hidden
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := validator.isHidden(tt.path)
			assert.Equal(t, tt.expected, result, "Path: %s", tt.path)
		})
	}
}

// TestDirectoryValidator_calculateFileChecksum tests checksum calculation
func TestDirectoryValidator_calculateFileChecksum(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	validator := NewDirectoryValidator(logger)

	t.Run("valid file checksum", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		content := []byte("test content for checksum")
		err := os.WriteFile(filePath, content, 0o600)
		require.NoError(t, err)

		checksum, err := validator.calculateFileChecksum(filePath)
		require.NoError(t, err)
		assert.NotEmpty(t, checksum)

		// Verify checksum matches expected SHA256
		expectedHash := fmt.Sprintf("%x", sha256.Sum256(content))
		assert.Equal(t, expectedHash, checksum)
	})

	t.Run("file not found", func(t *testing.T) {
		checksum, err := validator.calculateFileChecksum("/nonexistent/file.txt")
		require.Error(t, err)
		assert.Empty(t, checksum)
	})

	t.Run("consistent checksums for same content", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create two files with same content
		file1 := filepath.Join(tempDir, "file1.txt")
		file2 := filepath.Join(tempDir, "file2.txt")
		content := []byte("identical content")

		err := os.WriteFile(file1, content, 0o600)
		require.NoError(t, err)
		err = os.WriteFile(file2, content, 0o600)
		require.NoError(t, err)

		checksum1, err := validator.calculateFileChecksum(file1)
		require.NoError(t, err)
		checksum2, err := validator.calculateFileChecksum(file2)
		require.NoError(t, err)

		assert.Equal(t, checksum1, checksum2)
	})
}
