//go:build race

package pool

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBufferPoolRace is an aggressive concurrent test specifically for race detection.
// Run with: go test -race ./internal/pool/...
func TestBufferPoolRace(t *testing.T) {
	bp := NewBufferPool()

	const numGoroutines = 50
	const numOperations = 500

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Goroutines doing Get/Put cycles
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				size := (id*100 + j*10) % 70000
				buf := bp.GetBuffer(size)
				buf.WriteString("race test data")
				_ = buf.String()
				bp.PutBuffer(buf)
			}
		}(i)
	}

	// Goroutines reading stats concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				stats := bp.GetStats()
				_ = stats.SmallPool.ReturnRate()
				_ = stats.MediumPool.ReturnRate()
				_ = stats.LargePool.ReturnRate()
			}
		}()
	}

	// Goroutines resetting stats (less frequently)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations/10; j++ {
				bp.ResetStats()
			}
		}()
	}

	wg.Wait()
}

// TestDefaultPoolRace tests race conditions with the package-level default pool
func TestDefaultPoolRace(t *testing.T) {
	const numGoroutines = 20
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Goroutines using default pool
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				size := (id*50 + j) % 50000
				buf := GetBuffer(size)
				buf.WriteString("default pool race test")
				PutBuffer(buf)
			}
		}(i)
	}

	// Goroutines reading/resetting stats
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				if j%10 == 0 {
					ResetStats()
				}
				_ = GetStats()
			}
		}()
	}

	wg.Wait()
}

// TestWithBufferRace tests race conditions in the WithBuffer helper
func TestWithBufferRace(t *testing.T) {
	const numGoroutines = 30
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				size := (id*100 + j) % 60000
				err := WithBuffer(size, func(buf *bytes.Buffer) error {
					buf.WriteString("concurrent WithBuffer test")
					return nil
				})
				require.NoError(t, err)
			}
		}(i)
	}

	wg.Wait()
}

// TestEstimateBufferSizeRace tests concurrent calls to EstimateBufferSize
func TestEstimateBufferSizeRace(t *testing.T) {
	operations := []string{
		"json_marshal",
		"string_concat",
		"template_transform",
		"file_content",
		"git_diff",
		"unknown",
	}

	const numGoroutines = 20
	const numOperations = 200

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				op := operations[(id+j)%len(operations)]
				size := (id*100 + j) % 100000
				result := EstimateBufferSize(op, size)
				require.Positive(t, result)
			}
		}(i)
	}

	wg.Wait()
}

// TestMixedOperationsRace tests a realistic mix of operations
func TestMixedOperationsRace(t *testing.T) {
	bp := NewBufferPool()

	const numGoroutines = 25
	const numOperations = 150

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Mix of operations based on id and iteration
				switch (id + j) % 5 {
				case 0:
					// Small buffer operations
					buf := bp.GetBuffer(100)
					buf.WriteString("small")
					bp.PutBuffer(buf)
				case 1:
					// Medium buffer operations
					buf := bp.GetBuffer(5000)
					buf.WriteString("medium buffer data")
					bp.PutBuffer(buf)
				case 2:
					// Large buffer operations
					buf := bp.GetBuffer(50000)
					buf.WriteString("large buffer data content")
					bp.PutBuffer(buf)
				case 3:
					// Stats reading
					stats := bp.GetStats()
					_ = stats.Resets
				case 4:
					// Nil buffer put (edge case)
					bp.PutBuffer(nil)
				}
			}
		}(i)
	}

	wg.Wait()

	// Final stats should be consistent
	stats := bp.GetStats()
	require.GreaterOrEqual(t, stats.SmallPool.Gets, int64(0))
	require.GreaterOrEqual(t, stats.MediumPool.Gets, int64(0))
	require.GreaterOrEqual(t, stats.LargePool.Gets, int64(0))
}
