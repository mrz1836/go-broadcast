package output

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInit(_ *testing.T) {
	Init()
	// Init should enable color output - hard to test the actual color behavior
	// but we can ensure the function runs without panicking
}

func TestSetAndGetStdout(t *testing.T) {
	// Save original
	original := stdout
	defer func() {
		stdout = original
	}()

	// Test setting custom writer
	buf := &bytes.Buffer{}
	SetStdout(buf)

	// Verify it was set
	assert.Equal(t, buf, Stdout())
}

func TestSetAndGetStderr(t *testing.T) {
	// Save original
	original := stderr
	defer func() {
		stderr = original
	}()

	// Test setting custom writer
	buf := &bytes.Buffer{}
	SetStderr(buf)

	// Verify it was set
	assert.Equal(t, buf, Stderr())
}

func TestSuccess(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	Success("test success message")

	output := buf.String()
	assert.Contains(t, output, "test success message")
	assert.Contains(t, output, "\n") // Should end with newline
}

func TestSuccessf(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	Successf("test %s message %d", "formatted", 123)

	output := buf.String()
	assert.Contains(t, output, "test formatted message 123")
}

func TestInfo(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	Info("test info message")

	output := buf.String()
	assert.Contains(t, output, "test info message")
	assert.Contains(t, output, "\n")
}

func TestInfof(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	Infof("info %s %d", "test", 456)

	output := buf.String()
	assert.Contains(t, output, "info test 456")
}

func TestWarn(t *testing.T) {
	// Capture stderr output
	buf := &bytes.Buffer{}
	SetStderr(buf)
	defer SetStderr(stderr)

	Warn("test warning message")

	output := buf.String()
	assert.Contains(t, output, "test warning message")
	assert.Contains(t, output, "\n")
}

func TestWarnf(t *testing.T) {
	// Capture stderr output
	buf := &bytes.Buffer{}
	SetStderr(buf)
	defer SetStderr(stderr)

	Warnf("warning %s %d", "formatted", 789)

	output := buf.String()
	assert.Contains(t, output, "warning formatted 789")
}

func TestError(t *testing.T) {
	// Capture stderr output
	buf := &bytes.Buffer{}
	SetStderr(buf)
	defer SetStderr(stderr)

	Error("test error message")

	output := buf.String()
	assert.Contains(t, output, "test error message")
	assert.Contains(t, output, "\n")
}

func TestErrorf(t *testing.T) {
	// Capture stderr output
	buf := &bytes.Buffer{}
	SetStderr(buf)
	defer SetStderr(stderr)

	Errorf("error %s %d", "formatted", 999)

	output := buf.String()
	assert.Contains(t, output, "error formatted 999")
}

func TestPlain(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	Plain("test plain message")

	output := buf.String()
	assert.Contains(t, output, "test plain message")
	assert.Contains(t, output, "\n")
}

func TestPlainf(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	Plainf("plain %s %d", "formatted", 111)

	output := buf.String()
	assert.Contains(t, output, "plain formatted 111")
}

func TestNewProgress(t *testing.T) {
	progress := NewProgress("test message")

	assert.NotNil(t, progress)
	assert.Equal(t, "test message", progress.message)
	assert.Equal(t, []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, progress.spinner)
	assert.Equal(t, 0, progress.index)
	assert.NotNil(t, progress.done)
}

func TestProgressStartStop(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	progress := NewProgress("testing progress")

	// Start progress
	progress.Start()

	// Let it spin a bit
	time.Sleep(50 * time.Millisecond)

	// Stop progress
	progress.Stop()

	// Verify some output was generated
	output := buf.String()
	assert.Contains(t, output, "testing progress")
}

func TestProgressStopWithSuccess(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	progress := NewProgress("test progress")
	progress.Start()
	time.Sleep(10 * time.Millisecond)
	progress.StopWithSuccess("Success message")

	output := buf.String()
	assert.Contains(t, output, "Success message")
}

func TestProgressStopWithError(t *testing.T) {
	// Capture stderr output for error
	stderrBuf := &bytes.Buffer{}
	SetStderr(stderrBuf)
	defer SetStderr(stderr)

	// Capture stdout for progress
	stdoutBuf := &bytes.Buffer{}
	SetStdout(stdoutBuf)
	defer SetStdout(stdout)

	progress := NewProgress("test progress")
	progress.Start()
	time.Sleep(10 * time.Millisecond)
	progress.StopWithError("Error message")

	errorOutput := stderrBuf.String()
	assert.Contains(t, errorOutput, "Error message")
}

func TestConcurrentOutputFunctions(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 5

	// Start multiple goroutines writing concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				Successf("concurrent message from goroutine %d iteration %d", id, j)
				Infof("info from goroutine %d iteration %d", id, j)
				Plainf("plain from goroutine %d iteration %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	output := buf.String()
	// Should contain output from all goroutines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedLines := numGoroutines * messagesPerGoroutine * 3 // 3 types of messages
	assert.Len(t, lines, expectedLines)
}

func TestConcurrentWriterChanges(t *testing.T) {
	// Test concurrent access to writer setters/getters
	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create a unique buffer for this goroutine
			buf := &bytes.Buffer{}

			// Set stdout to our buffer
			SetStdout(buf)

			// Get stdout (should be thread-safe)
			writer := Stdout()
			assert.NotNil(t, writer)

			// Write something
			Successf("message from goroutine %d", id)
		}(i)
	}

	wg.Wait()
}

func TestMultipleProgressIndicators(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	// Create multiple progress indicators
	progress1 := NewProgress("first progress")
	progress2 := NewProgress("second progress")

	// Start both
	progress1.Start()
	progress2.Start()

	// Let them run briefly
	time.Sleep(30 * time.Millisecond)

	// Stop both
	progress1.Stop()
	progress2.Stop()

	// Verify output contains both messages
	output := buf.String()
	assert.Contains(t, output, "first progress")
	assert.Contains(t, output, "second progress")
}

func TestOutputToCorrectStreams(t *testing.T) {
	// Capture both stdout and stderr
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	SetStdout(stdoutBuf)
	SetStderr(stderrBuf)
	defer func() {
		SetStdout(stdout)
		SetStderr(stderr)
	}()

	// Test stdout functions
	Success("success message")
	Info("info message")
	Plain("plain message")

	// Test stderr functions
	Warn("warning message")
	Error("error message")

	// Verify stdout contains success, info, and plain messages
	stdoutOutput := stdoutBuf.String()
	assert.Contains(t, stdoutOutput, "success message")
	assert.Contains(t, stdoutOutput, "info message")
	assert.Contains(t, stdoutOutput, "plain message")

	// Verify stderr contains warning and error messages
	stderrOutput := stderrBuf.String()
	assert.Contains(t, stderrOutput, "warning message")
	assert.Contains(t, stderrOutput, "error message")

	// Verify cross-contamination doesn't happen
	assert.NotContains(t, stdoutOutput, "warning message")
	assert.NotContains(t, stdoutOutput, "error message")
	assert.NotContains(t, stderrOutput, "success message")
	assert.NotContains(t, stderrOutput, "info message")
	assert.NotContains(t, stderrOutput, "plain message")
}

func TestProgressSpinnerAdvancement(t *testing.T) {
	progress := NewProgress("test spinner")

	// Check initial state
	assert.Equal(t, 0, progress.index)

	// We can't easily test the actual spinning without making the internal
	// spinner advancement public, but we can test the structure is correct
	assert.Len(t, progress.spinner, 10)
	assert.Equal(t, "⠋", progress.spinner[0])
	assert.Equal(t, "⠏", progress.spinner[9])
}

func BenchmarkSuccessOutput(b *testing.B) {
	// Use a discard writer to avoid I/O overhead in benchmark
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Success("benchmark message")
	}
}

func BenchmarkConcurrentOutput(b *testing.B) {
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(stdout)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Success("concurrent benchmark message")
		}
	})
}
