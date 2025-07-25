package profiling

import (
	"context"
	"errors"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var errTestFuncError = errors.New("test function error")

// configureForTesting configures a ProfileSuite for testing without CPU profiling conflicts
func configureForTesting(suite *ProfileSuite) {
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
	tempDir := b.TempDir()
	suite := NewProfileSuite(tempDir)

	// Disable reports for performance
	config := suite.config
	config.GenerateReports = false
	suite.Configure(config)

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
