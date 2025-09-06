//go:build mage

package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test errors
var (
	ErrBenchmarkFailed        = errors.New("benchmark failed")
	ErrQuickBenchmarkFailed   = errors.New("quick benchmark failed")
	ErrQuickBenchmarksFailed  = errors.New("quick benchmarks failed")
	ErrHeavyBenchmarksFailed  = errors.New("heavy benchmarks failed")
	ErrTestsFailed            = errors.New("tests failed")
	ErrPerformanceTestsFailed = errors.New("performance tests failed")
	ErrQuickTestsFailed       = errors.New("quick tests failed")
)

// MockCommander for testing
type MockCommander struct {
	calls        [][]string
	returnError  error
	callIndex    int
	returnErrors []error // For different return values on subsequent calls
}

// RunV implements Commander interface for mocking
func (m *MockCommander) RunV(cmd string, args ...string) error {
	call := append([]string{cmd}, args...)
	m.calls = append(m.calls, call)

	// Use specific error for this call if available
	if m.callIndex < len(m.returnErrors) {
		err := m.returnErrors[m.callIndex]
		m.callIndex++
		return err
	}

	// Otherwise use default error
	m.callIndex++
	return m.returnError
}

// GetCalls returns all recorded calls
func (m *MockCommander) GetCalls() [][]string {
	return m.calls
}

// Reset clears all recorded calls and resets call index
func (m *MockCommander) Reset() {
	m.calls = nil
	m.callIndex = 0
	m.returnError = nil
	m.returnErrors = nil
}

// TestBenchHeavy tests the BenchHeavy function
func TestBenchHeavy(t *testing.T) {
	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	t.Run("Success", func(t *testing.T) {
		mock := &MockCommander{}
		setCommander(mock)

		err := BenchHeavy()

		require.NoError(t, err)
		calls := mock.GetCalls()
		require.Len(t, calls, 1)

		expectedCall := []string{
			"go", "test", "-bench=.", "-benchmem",
			"-tags=bench_heavy", "-benchtime=1s", "-timeout=60m", "./...",
		}
		assert.Equal(t, expectedCall, calls[0])
	})

	t.Run("CommandFails", func(t *testing.T) {
		mock := &MockCommander{returnError: ErrBenchmarkFailed}
		setCommander(mock)

		err := BenchHeavy()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "benchmark failed")
	})
}

// TestBenchQuick tests the BenchQuick function
func TestBenchQuick(t *testing.T) {
	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	t.Run("Success", func(t *testing.T) {
		mock := &MockCommander{}
		setCommander(mock)

		err := BenchQuick()

		require.NoError(t, err)
		calls := mock.GetCalls()
		require.Len(t, calls, 1)

		expectedCall := []string{
			"go", "test", "-bench=.", "-benchmem",
			"-benchtime=100ms", "-timeout=20m", "./...",
		}
		assert.Equal(t, expectedCall, calls[0])
	})

	t.Run("CommandFails", func(t *testing.T) {
		mock := &MockCommander{returnError: ErrQuickBenchmarkFailed}
		setCommander(mock)

		err := BenchQuick()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "quick benchmark failed")
	})
}

// TestBenchAll tests the BenchAll function
func TestBenchAll(t *testing.T) {
	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	t.Run("Success", func(t *testing.T) {
		mock := &MockCommander{}
		setCommander(mock)

		err := BenchAll()

		require.NoError(t, err)
		calls := mock.GetCalls()
		require.Len(t, calls, 2)

		// First call should be quick benchmarks
		expectedQuickCall := []string{
			"go", "test", "-bench=.", "-benchmem",
			"-benchtime=100ms", "-timeout=20m", "./...",
		}
		assert.Equal(t, expectedQuickCall, calls[0])

		// Second call should be heavy benchmarks
		expectedHeavyCall := []string{
			"go", "test", "-bench=.", "-benchmem",
			"-tags=bench_heavy", "-benchtime=1s", "-timeout=60m", "./...",
		}
		assert.Equal(t, expectedHeavyCall, calls[1])
	})

	t.Run("QuickBenchmarksFail", func(t *testing.T) {
		quickError := ErrQuickBenchmarksFailed
		mock := &MockCommander{returnErrors: []error{quickError}}
		setCommander(mock)

		err := BenchAll()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "quick benchmarks failed")

		// Should only have one call since first one failed
		calls := mock.GetCalls()
		assert.Len(t, calls, 1)
	})

	t.Run("HeavyBenchmarksFail", func(t *testing.T) {
		heavyError := ErrHeavyBenchmarksFailed
		mock := &MockCommander{returnErrors: []error{nil, heavyError}}
		setCommander(mock)

		err := BenchAll()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "heavy benchmarks failed")

		// Should have both calls
		calls := mock.GetCalls()
		assert.Len(t, calls, 2)
	})
}

// TestTestQuick tests the TestQuick function
func TestTestQuick(t *testing.T) {
	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	t.Run("Success", func(t *testing.T) {
		mock := &MockCommander{}
		setCommander(mock)

		err := TestQuick()

		require.NoError(t, err)
		calls := mock.GetCalls()
		require.Len(t, calls, 1)

		expectedCall := []string{"go", "test", "-short", "./..."}
		assert.Equal(t, expectedCall, calls[0])
	})

	t.Run("CommandFails", func(t *testing.T) {
		mock := &MockCommander{returnError: ErrTestsFailed}
		setCommander(mock)

		err := TestQuick()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tests failed")
	})
}

// TestTestPerf tests the TestPerf function
func TestTestPerf(t *testing.T) {
	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	t.Run("Success", func(t *testing.T) {
		mock := &MockCommander{}
		setCommander(mock)

		err := TestPerf()

		require.NoError(t, err)
		calls := mock.GetCalls()
		require.Len(t, calls, 1)

		expectedCall := []string{"go", "test", "-tags=performance", "-timeout=30m", "./test/integration"}
		assert.Equal(t, expectedCall, calls[0])
	})

	t.Run("CommandFails", func(t *testing.T) {
		mock := &MockCommander{returnError: ErrPerformanceTestsFailed}
		setCommander(mock)

		err := TestPerf()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "performance tests failed")
	})
}

// TestTestAll tests the TestAll function
func TestTestAll(t *testing.T) {
	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	t.Run("Success", func(t *testing.T) {
		mock := &MockCommander{}
		setCommander(mock)

		err := TestAll()

		require.NoError(t, err)
		calls := mock.GetCalls()
		require.Len(t, calls, 2)

		// First call should be quick tests
		expectedQuickCall := []string{"go", "test", "-short", "./..."}
		assert.Equal(t, expectedQuickCall, calls[0])

		// Second call should be performance tests
		expectedPerfCall := []string{"go", "test", "-tags=performance", "-timeout=30m", "./test/integration"}
		assert.Equal(t, expectedPerfCall, calls[1])
	})

	t.Run("QuickTestsFail", func(t *testing.T) {
		quickError := ErrQuickTestsFailed
		mock := &MockCommander{returnErrors: []error{quickError}}
		setCommander(mock)

		err := TestAll()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "quick tests failed")

		// Should only have one call since first one failed
		calls := mock.GetCalls()
		assert.Len(t, calls, 1)
	})

	t.Run("PerformanceTestsFail", func(t *testing.T) {
		perfError := ErrPerformanceTestsFailed
		mock := &MockCommander{returnErrors: []error{nil, perfError}}
		setCommander(mock)

		err := TestAll()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "performance tests failed")

		// Should have both calls
		calls := mock.GetCalls()
		assert.Len(t, calls, 2)
	})
}

// TestShCommander tests the production ShCommander implementation
func TestShCommander(t *testing.T) {
	t.Run("ImplementsCommanderInterface", func(t *testing.T) {
		var cmd Commander = ShCommander{}
		assert.NotNil(t, cmd)
	})

	// We don't test actual command execution here as it would
	// require running real commands, which could be slow or flaky.
	// Integration tests handle actual command execution.
}

// Benchmark tests for the mock system itself
func BenchmarkMockCommander(b *testing.B) {
	mock := &MockCommander{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.Reset()
		_ = mock.RunV("go", "test", "./...")
	}
}

// Table-driven test for all mage functions
func TestAllMageFunctions(t *testing.T) {
	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	testCases := []struct {
		name          string
		function      func() error
		expectedCalls [][]string
		expectError   bool
		mockError     error
	}{
		{
			name:     "BenchHeavy",
			function: BenchHeavy,
			expectedCalls: [][]string{
				{"go", "test", "-bench=.", "-benchmem", "-tags=bench_heavy", "-benchtime=1s", "-timeout=60m", "./..."},
			},
		},
		{
			name:     "BenchQuick",
			function: BenchQuick,
			expectedCalls: [][]string{
				{"go", "test", "-bench=.", "-benchmem", "-benchtime=100ms", "-timeout=20m", "./..."},
			},
		},
		{
			name:     "TestQuick",
			function: TestQuick,
			expectedCalls: [][]string{
				{"go", "test", "-short", "./..."},
			},
		},
		{
			name:     "TestPerf",
			function: TestPerf,
			expectedCalls: [][]string{
				{"go", "test", "-tags=performance", "-timeout=30m", "./test/integration"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockCommander{returnError: tc.mockError}
			setCommander(mock)

			err := tc.function()

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			calls := mock.GetCalls()
			assert.Equal(t, tc.expectedCalls, calls)
		})
	}
}
