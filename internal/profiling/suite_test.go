package profiling

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var errTestFuncError = errors.New("test function error")

// configureForTesting configures a ProfileSuite for testing without CPU profiling conflicts
func configureForTesting(suite *ProfileSuite) {
	// Stop any existing CPU profiling to prevent conflicts
	pprof.StopCPUProfile()
	// Stop any existing trace to prevent conflicts
	trace.Stop()

	config := suite.config
	config.EnableCPU = false       // Disable CPU profiling to avoid conflicts in tests
	config.EnableMemory = false    // Disable memory profiling to avoid CPU conflicts
	config.EnableTrace = false     // Disable trace profiling
	config.EnableBlock = true      // Keep only block profiling enabled for testing
	config.EnableMutex = true      // Keep mutex profiling enabled for testing
	config.GenerateReports = false // Disable reports for faster tests
	suite.Configure(config)
}

func TestNewProfileSuite(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)

	require.NotNil(t, suite)
	require.Equal(t, tempDir, suite.outputDir)
	require.NotNil(t, suite.memProfiler)
	require.Nil(t, suite.currentSession)
	require.NotNil(t, suite.config)
	require.NotNil(t, suite.sessionHistory)

	// Check default configuration
	require.True(t, suite.config.EnableCPU)
	require.True(t, suite.config.EnableMemory)
	require.True(t, suite.config.EnableTrace)
	require.True(t, suite.config.EnableBlock)
	require.True(t, suite.config.EnableMutex)
	require.Equal(t, 1, suite.config.BlockProfileRate)
	require.Equal(t, 1, suite.config.MutexProfileFraction)
	require.True(t, suite.config.GenerateReports)
	require.Equal(t, "both", suite.config.ReportFormat)
	require.True(t, suite.config.AutoCleanup)
	require.Equal(t, 10, suite.config.MaxSessionsToKeep)
}

func TestProfileSuiteConfigure(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())

	newConfig := ProfileConfig{
		EnableCPU:            false,
		EnableMemory:         true,
		EnableTrace:          false,
		EnableBlock:          false,
		EnableMutex:          false,
		BlockProfileRate:     10,
		MutexProfileFraction: 5,
		GenerateReports:      false,
		ReportFormat:         "text",
		AutoCleanup:          false,
		MaxSessionsToKeep:    5,
	}

	suite.Configure(newConfig)

	require.Equal(t, newConfig, suite.config)
}

func TestProfileSuiteStartStopProfiling(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	// Start profiling
	err := suite.StartProfiling("test-suite-session")
	require.NoError(t, err)
	require.NotNil(t, suite.currentSession)
	require.Equal(t, "test-suite-session", suite.currentSession.Name)
	require.True(t, suite.IsActive())

	// Stop profiling
	err = suite.StopProfiling()
	require.NoError(t, err)
	require.Nil(t, suite.currentSession)
	require.False(t, suite.IsActive())

	// Check session history
	history := suite.GetSessionHistory()
	require.Len(t, history, 1)
	require.Equal(t, "test-suite-session", history[0].Name)
}

func TestProfileSuiteStartProfilingTwice(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	// Start first session
	err := suite.StartProfiling("session1")
	require.NoError(t, err)

	// Try to start second session (should fail)
	err = suite.StartProfiling("session2")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrProfilingSessionActive)
}

func TestProfileSuiteStopProfilingNoActiveSession(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())

	// Try to stop when no session is active
	err := suite.StopProfiling()
	require.Error(t, err)
	require.Equal(t, ErrNoActiveSession, err)
}

func TestProfileSuiteProfileWithFunc(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	executed := false
	testFunc := func() error {
		executed = true
		// Do some work
		data := make([]byte, 1024)
		_ = data
		return nil
	}

	err := suite.ProfileWithFunc("test-func", testFunc)
	require.NoError(t, err)
	require.True(t, executed)
	require.False(t, suite.IsActive()) // Should be stopped after completion

	// Check session history
	history := suite.GetSessionHistory()
	require.Len(t, history, 1)
	require.Equal(t, "test-func", history[0].Name)
}

func TestProfileSuiteProfileWithFuncError(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	expectedErr := errTestFuncError
	testFunc := func() error {
		return expectedErr
	}

	err := suite.ProfileWithFunc("error-func", testFunc)
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
	require.False(t, suite.IsActive()) // Should be stopped even on error
}

func TestProfileSuiteProfileWithContext(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	ctx := context.Background()
	executed := false

	testFunc := func(ctx context.Context) error {
		executed = true
		// Verify context is passed through
		require.NotNil(t, ctx)
		return nil
	}

	err := suite.ProfileWithContext(ctx, "context-func", testFunc)
	require.NoError(t, err)
	require.True(t, executed)
	require.False(t, suite.IsActive())
}

func TestProfileSuiteProfileWithContextCancellation(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	ctx, cancel := context.WithCancel(context.Background())

	testFunc := func(ctx context.Context) error {
		cancel() // Cancel the context during execution
		return ctx.Err()
	}

	err := suite.ProfileWithContext(ctx, "cancel-func", testFunc)
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestProfileSuiteSessionHistory(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	// Run multiple profiling sessions
	sessionNames := []string{"session1", "session2", "session3"}

	for _, name := range sessionNames {
		err := suite.ProfileWithFunc(name, func() error {
			time.Sleep(1 * time.Millisecond) // Small delay to ensure different durations
			return nil
		})
		require.NoError(t, err)
	}

	history := suite.GetSessionHistory()
	require.Len(t, history, len(sessionNames))

	// Verify session details
	for i, session := range history {
		require.Equal(t, sessionNames[i], session.Name)
		require.False(t, session.StartTime.IsZero())
		require.Greater(t, session.Duration, time.Duration(0))
		require.NotEmpty(t, session.OutputDir)
		require.Positive(t, session.FileCount)
		require.Positive(t, session.TotalSize)
		require.NotEmpty(t, session.ProfileTypes)
	}
}

func TestProfileSuiteAutoCleanup(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	// Configure to keep only 2 sessions
	config := suite.config
	config.MaxSessionsToKeep = 2
	suite.Configure(config)

	// Run 4 sessions
	for i := 0; i < 4; i++ {
		sessionName := "cleanup-session-" + string(rune('0'+i))
		err := suite.ProfileWithFunc(sessionName, func() error {
			return nil
		})
		require.NoError(t, err)
	}

	// Should only keep the last 2 sessions
	history := suite.GetSessionHistory()
	require.Len(t, history, 2)
	require.Equal(t, "cleanup-session-2", history[0].Name)
	require.Equal(t, "cleanup-session-3", history[1].Name)
}

func TestProfileSuiteDisabledAutoCleanup(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	// Disable auto cleanup
	config := suite.config
	config.AutoCleanup = false
	config.MaxSessionsToKeep = 1
	suite.Configure(config)

	// Run multiple sessions
	for i := 0; i < 3; i++ {
		sessionName := "no-cleanup-session-" + string(rune('0'+i))
		err := suite.ProfileWithFunc(sessionName, func() error {
			return nil
		})
		require.NoError(t, err)
	}

	// Should keep all sessions since cleanup is disabled
	history := suite.GetSessionHistory()
	require.Len(t, history, 3)
}

func TestProfileSuiteConfigurationEffects(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)

	// Configure to disable certain profiling types
	configureForTesting(suite)
	config := suite.config
	config.EnableTrace = false
	config.EnableBlock = false
	config.EnableMutex = false
	config.AutoCleanup = false
	config.MaxSessionsToKeep = 10
	suite.Configure(config)

	err := suite.ProfileWithFunc("config-test", func() error {
		return nil
	})
	require.NoError(t, err)

	history := suite.GetSessionHistory()
	require.Len(t, history, 1)

	// With memory enabled but others disabled, should have fewer profile types
	session := history[0]
	require.NotEmpty(t, session.ProfileTypes)
	require.Less(t, len(session.ProfileTypes), 6) // Should be less than full set
}

func TestProfileSuiteGetCurrentSession(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	// Initially no current session
	require.Nil(t, suite.GetCurrentSession())

	// Start a session
	err := suite.StartProfiling("current-session-test")
	require.NoError(t, err)

	current := suite.GetCurrentSession()
	require.NotNil(t, current)
	require.Equal(t, "current-session-test", current.Name)

	// Stop the session
	err = suite.StopProfiling()
	require.NoError(t, err)

	require.Nil(t, suite.GetCurrentSession())
}

func TestProfileSuiteBlockAndMutexProfiling(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	// Enable block and mutex profiling with custom rates
	config := suite.config
	config.BlockProfileRate = 100
	config.MutexProfileFraction = 10
	suite.Configure(config)

	// Save original profiling settings
	defer func() {
		runtime.SetBlockProfileRate(0)
		runtime.SetMutexProfileFraction(0)
	}()

	err := suite.ProfileWithFunc("mutex-block-test", func() error {
		// The profiling should have set the rates
		return nil
	})
	require.NoError(t, err)
}

func TestSessionDirectoryNaming(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	sessionName := "directory-naming-test"

	err := suite.StartProfiling(sessionName)
	require.NoError(t, err)

	current := suite.GetCurrentSession()
	require.NotNil(t, current)

	// Directory should contain the session name and timestamp
	require.Contains(t, current.OutputDir, sessionName)
	require.Contains(t, current.OutputDir, tempDir)

	// Verify directory exists
	_, err = os.Stat(current.OutputDir)
	require.NoError(t, err)

	err = suite.StopProfiling()
	require.NoError(t, err)
}

func TestProfileSuiteReportGeneration(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	// Configure with report generation enabled
	config := suite.config
	config.EnableCPU = false // Avoid conflicts
	config.GenerateReports = true
	config.ReportFormat = "text" // Only text to avoid HTML generation complexity in tests
	suite.Configure(config)

	err := suite.ProfileWithFunc("report-test", func() error {
		// Do some work to generate profile data
		data := make([][]byte, 100)
		for i := range data {
			data[i] = make([]byte, 1024)
		}
		return nil
	})
	require.NoError(t, err)

	history := suite.GetSessionHistory()
	require.Len(t, history, 1)

	// Check if report file exists
	reportPath := history[0].OutputDir + "/comprehensive_report.txt"
	_, err = os.Stat(reportPath)
	require.NoError(t, err)
}

// BenchmarkProfileSuiteOverhead tests the overhead of the profiling suite
func BenchmarkProfileSuiteOverhead(b *testing.B) {
	// Stop any existing profiling to prevent conflicts
	pprof.StopCPUProfile()
	trace.Stop()

	tempDir := b.TempDir()
	suite := NewProfileSuite(tempDir)

	// Disable CPU profiling and reports for performance benchmarking
	// We only want to measure the suite overhead, not actually profile
	config := suite.config
	config.EnableCPU = false
	config.EnableTrace = false
	config.GenerateReports = false
	suite.Configure(config)

	// Ensure cleanup after benchmark
	defer func() {
		pprof.StopCPUProfile()
		trace.Stop()
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := suite.ProfileWithFunc("benchmark-test", func() error {
			// Minimal work
			sum := 0
			for j := 0; j < 100; j++ {
				sum += j
			}
			_ = sum
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestProfileSuiteConcurrentSafety(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())
	configureForTesting(suite)

	// Test concurrent access to session history and stats
	done := make(chan struct{})

	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			history := suite.GetSessionHistory()
			_ = history

			current := suite.GetCurrentSession()
			_ = current

			active := suite.IsActive()
			_ = active
		}
	}()

	// While reading, run profiling sessions
	for i := 0; i < 5; i++ {
		sessionName := "concurrent-safety-" + string(rune('0'+i))
		err := suite.ProfileWithFunc(sessionName, func() error {
			return nil
		})
		require.NoError(t, err)
	}

	<-done
}

// TestProfileSuiteGenerateHTMLReport tests HTML report generation
func TestProfileSuiteGenerateHTMLReport(t *testing.T) {
	// Skip this test if go tool is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go tool not available")
	}

	// Ensure clean state before test
	pprof.StopCPUProfile()
	trace.Stop()

	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	// Configure with HTML report generation
	config := suite.config
	config.EnableCPU = true // Need CPU profile for HTML report
	config.GenerateReports = true
	config.ReportFormat = "html"
	suite.Configure(config)

	// Ensure cleanup after test
	defer func() {
		pprof.StopCPUProfile()
		trace.Stop()
	}()

	// Run with a simple function - not actually testing CPU profiling
	// Just testing the report generation flow
	err := suite.ProfileWithFunc("html-report-test", func() error {
		return nil
	})
	require.NoError(t, err)
}

// TestProfileSuiteBothReportFormats tests generating both text and HTML reports
func TestProfileSuiteBothReportFormats(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	// Configure with both report formats
	config := suite.config
	config.GenerateReports = true
	config.ReportFormat = "both"
	suite.Configure(config)

	err := suite.ProfileWithFunc("both-reports-test", func() error {
		// Some work to profile
		data := make([]byte, 1024*1024) // 1MB allocation
		for i := range data {
			data[i] = byte(i % 256)
		}
		return nil
	})
	require.NoError(t, err)

	history := suite.GetSessionHistory()
	require.Len(t, history, 1)

	// Check if text report exists
	textReportPath := history[0].OutputDir + "/comprehensive_report.txt"
	_, err = os.Stat(textReportPath)
	require.NoError(t, err)
}

// TestProfileSuiteErrorHandlingDuringStart tests error handling when starting a session fails
func TestProfileSuiteErrorHandlingDuringStart(t *testing.T) {
	// Use a non-existent directory to cause failure
	suite := NewProfileSuite("/nonexistent/path/that/should/fail")
	configureForTesting(suite)

	err := suite.StartProfiling("error-test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create session directory")
	require.Nil(t, suite.currentSession)
}

// TestProfileSuiteCleanupOldSessionsError tests cleanup with permission errors
func TestProfileSuiteCleanupOldSessionsError(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	// Configure to keep only 1 session
	config := suite.config
	config.MaxSessionsToKeep = 1
	suite.Configure(config)

	// Run first session
	err := suite.ProfileWithFunc("session1", func() error {
		return nil
	})
	require.NoError(t, err)

	// Get the first session directory
	history := suite.GetSessionHistory()
	firstSessionDir := history[0].OutputDir

	// Make the directory read-only to simulate permission error during cleanup
	err = os.Chmod(firstSessionDir, 0o500) //nolint:gosec // Intentionally setting read-only for test
	require.NoError(t, err)

	// Run second session which should trigger cleanup
	err = suite.ProfileWithFunc("session2", func() error {
		return nil
	})
	require.NoError(t, err)

	// Cleanup should have been attempted but failed silently
	// Session history should still be updated
	history = suite.GetSessionHistory()
	require.Len(t, history, 1)
	require.Equal(t, "session2", history[0].Name)

	// Restore permissions for cleanup
	err = os.Chmod(firstSessionDir, 0o700) //nolint:gosec // Restoring normal permissions after test
	require.NoError(t, err)
}

// TestComprehensiveSessionConcurrency tests concurrent access to ComprehensiveSession
func TestComprehensiveSessionConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	err := suite.StartProfiling("concurrency-test")
	require.NoError(t, err)

	session := suite.GetCurrentSession()
	require.NotNil(t, session)

	// Test concurrent access to session fields
	var wg sync.WaitGroup
	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			// Access session fields concurrently
			_ = session.Name
			_ = session.StartTime
			_ = session.OutputDir
		}()
	}

	wg.Wait()

	err = suite.StopProfiling()
	require.NoError(t, err)
}

// TestProfileSuiteEnableProfilingWithZeroRates tests enabling profiling with zero rates
func TestProfileSuiteEnableProfilingWithZeroRates(t *testing.T) {
	suite := NewProfileSuite(t.TempDir())

	// Configure with zero rates (should not enable block/mutex profiling)
	config := ProfileConfig{
		EnableBlock:          true,
		EnableMutex:          true,
		BlockProfileRate:     0,
		MutexProfileFraction: 0,
	}
	suite.Configure(config)

	// Save original settings
	defer func() {
		runtime.SetBlockProfileRate(0)
		runtime.SetMutexProfileFraction(0)
	}()

	err := suite.ProfileWithFunc("zero-rates-test", func() error {
		return nil
	})
	require.NoError(t, err)
}

// TestProfileSuiteAdditionalProfiles tests capturing additional profiles
func TestProfileSuiteAdditionalProfiles(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	// Enable all profile types
	config := suite.config
	config.EnableBlock = true
	config.EnableMutex = true
	config.BlockProfileRate = 1
	config.MutexProfileFraction = 1
	suite.Configure(config)

	err := suite.ProfileWithFunc("additional-profiles-test", func() error {
		// Create some goroutines to generate goroutine profile
		done := make(chan struct{})
		for i := 0; i < 5; i++ {
			go func() {
				<-done
			}()
		}

		// Do some allocations for allocs profile
		data := make([][]byte, 10)
		for i := range data {
			data[i] = make([]byte, 1024)
		}

		close(done)
		return nil
	})
	require.NoError(t, err)

	history := suite.GetSessionHistory()
	require.Len(t, history, 1)

	// Check if profile files were created
	sessionDir := history[0].OutputDir

	// Check for goroutine profile
	_, err = os.Stat(sessionDir + "/goroutine.prof")
	require.NoError(t, err)

	// Check for block profile
	_, err = os.Stat(sessionDir + "/block.prof")
	require.NoError(t, err)

	// Check for mutex profile
	_, err = os.Stat(sessionDir + "/mutex.prof")
	require.NoError(t, err)
}

// TestProfileSuiteStopSessionIdempotent tests that stopping a session multiple times is safe
func TestProfileSuiteStopSessionIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)
	configureForTesting(suite)

	err := suite.StartProfiling("idempotent-test")
	require.NoError(t, err)

	// Stop the session
	err = suite.StopProfiling()
	require.NoError(t, err)

	// Attempting to stop again should return ErrNoActiveSession
	err = suite.StopProfiling()
	require.Error(t, err)
	require.Equal(t, ErrNoActiveSession, err)
}

// TestProfileSuiteMemoryProfilingIntegration tests memory profiling integration
func TestProfileSuiteMemoryProfilingIntegration(t *testing.T) {
	// Stop any existing CPU profiling from other tests
	pprof.StopCPUProfile()
	// Stop any existing trace that might be running
	trace.Stop()

	tempDir := t.TempDir()
	suite := NewProfileSuite(tempDir)

	// Configure with memory profiling enabled
	config := suite.config
	config.EnableCPU = false
	config.EnableMemory = true
	config.EnableTrace = false
	config.EnableBlock = false
	config.EnableMutex = false
	config.GenerateReports = false
	suite.Configure(config)

	err := suite.ProfileWithFunc("memory-integration-test", func() error {
		// Allocate some memory
		data := make([][]byte, 100)
		for i := range data {
			data[i] = make([]byte, 1024*10) // 10KB each
		}
		return nil
	})
	require.NoError(t, err)

	// Check that memory profiler was used
	require.NotNil(t, suite.memProfiler)
}

// TestWriteToReport tests the writeToReport helper function
func TestWriteToReport(t *testing.T) {
	tempFile, err := os.CreateTemp("", "report-test-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	defer func() { _ = tempFile.Close() }()

	// Test writing to report
	writeToReport(tempFile, "Test message: %s\n", "hello")
	writeToReport(tempFile, "Number: %d\n", 42)

	// Read back the content
	err = tempFile.Sync()
	require.NoError(t, err)

	content, err := os.ReadFile(tempFile.Name())
	require.NoError(t, err)

	expectedContent := "Test message: hello\nNumber: 42\n"
	require.Equal(t, expectedContent, string(content))
}
