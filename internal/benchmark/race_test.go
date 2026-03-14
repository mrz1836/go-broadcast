package benchmark

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateSHAConcurrent tests that generateSHA is thread-safe
// Run with: go test -race -run TestGenerateSHAConcurrent
func TestGenerateSHAConcurrent(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Use a channel to collect results and verify they're valid
	results := make(chan string, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sha := generateSHA()
				results <- sha
			}
		}()
	}

	wg.Wait()
	close(results)

	// Verify all generated SHAs are valid
	count := 0
	for sha := range results {
		require.Len(t, sha, 40, "SHA should be 40 characters")
		// Verify only valid hex characters
		for _, c := range sha {
			require.Contains(t, "abcdef0123456789", string(c), "SHA should only contain hex characters")
		}
		count++
	}
	require.Equal(t, goroutines*iterations, count)
}

// TestGenerateTokenConcurrent tests that generateToken is thread-safe
// Run with: go test -race -run TestGenerateTokenConcurrent
func TestGenerateTokenConcurrent(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	results := make(chan string, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				token := generateToken()
				results <- token
			}
		}()
	}

	wg.Wait()
	close(results)

	// Verify all generated tokens are valid
	count := 0
	validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for token := range results {
		require.Len(t, token, 20, "Token should be 20 characters")
		for _, c := range token {
			require.Contains(t, validChars, string(c), "Token should only contain alphanumeric characters")
		}
		count++
	}
	require.Equal(t, goroutines*iterations, count)
}

// TestGenerateJSONResponseConcurrent tests that GenerateJSONResponse is thread-safe
// Run with: go test -race -run TestGenerateJSONResponseConcurrent
func TestGenerateJSONResponseConcurrent(t *testing.T) {
	const goroutines = 50
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				result := GenerateJSONResponse(5)
				assert.NotEmpty(t, result)
				assert.Equal(t, byte('['), result[0])
				assert.Equal(t, byte(']'), result[len(result)-1])
			}
		}()
	}

	wg.Wait()
}

// TestGenerateGitDiffConcurrent tests that GenerateGitDiff is thread-safe
// Run with: go test -race -run TestGenerateGitDiffConcurrent
func TestGenerateGitDiffConcurrent(t *testing.T) {
	const goroutines = 50
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				result := GenerateGitDiff(3, 5)
				assert.NotEmpty(t, result)
				assert.Contains(t, result, "diff --git")
			}
		}()
	}

	wg.Wait()
}

// TestCompareWithBaselineConcurrent tests that CompareWithBaseline is thread-safe for reads
// Run with: go test -race -run TestCompareWithBaselineConcurrent
func TestCompareWithBaselineConcurrent(t *testing.T) {
	const goroutines = 50

	current := BaselineReport{
		Benchmarks: map[string]Metrics{
			"bench1": {Name: "bench1", NsPerOp: 800, BytesPerOp: 200},
			"bench2": {Name: "bench2", NsPerOp: 1200, BytesPerOp: 400},
		},
	}
	baseline := BaselineReport{
		Benchmarks: map[string]Metrics{
			"bench1": {Name: "bench1", NsPerOp: 1000, BytesPerOp: 250},
			"bench2": {Name: "bench2", NsPerOp: 1000, BytesPerOp: 250},
		},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			result := CompareWithBaseline(current, baseline)
			assert.NotNil(t, result.Improvements)
			assert.NotNil(t, result.Regressions)
			assert.Equal(t, 2, result.Summary.TotalBenchmarks)
		}()
	}

	wg.Wait()
}

// TestGenerateTextReportConcurrent tests that GenerateTextReport is thread-safe
// Run with: go test -race -run TestGenerateTextReportConcurrent
func TestGenerateTextReportConcurrent(t *testing.T) {
	const goroutines = 50

	comparison := ComparisonReport{
		BaselineReport: BaselineReport{
			GOOS:   "linux",
			GOARCH: "amd64",
			Benchmarks: map[string]Metrics{
				"test1": {Name: "test1", NsPerOp: 1000, BytesPerOp: 250, AllocsPerOp: 5},
			},
		},
		CurrentReport: BaselineReport{
			GOOS:   "linux",
			GOARCH: "amd64",
			Benchmarks: map[string]Metrics{
				"test1": {Name: "test1", NsPerOp: 800, BytesPerOp: 200, AllocsPerOp: 4},
			},
		},
		Improvements: map[string]float64{"test1_speed": 20.0},
		Regressions:  map[string]float64{},
		Summary: ComparisonSummary{
			TotalBenchmarks: 1,
			Improved:        1,
		},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Store first result for comparison
	expected := GenerateTextReport(comparison)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			result := GenerateTextReport(comparison)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, "Performance Comparison Report")
			// All goroutines should produce identical output (deterministic)
			assert.Equal(t, expected, result, "Report generation should be deterministic")
		}()
	}

	wg.Wait()
}
