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
	// Save original using mutex-protected accessor
	original := Stdout()
	defer SetStdout(original)

	// Test setting custom writer
	buf := &bytes.Buffer{}
	SetStdout(buf)

	// Verify it was set
	assert.Equal(t, buf, Stdout())
}

func TestSetAndGetStderr(t *testing.T) {
	// Save original using mutex-protected accessor
	original := Stderr()
	defer SetStderr(original)

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
	defer SetStdout(Stdout())

	Success("test success message")

	output := buf.String()
	assert.Contains(t, output, "test success message")
	assert.Contains(t, output, "\n") // Should end with newline
}

func TestSuccessf(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

	Successf("test %s message %d", "formatted", 123)

	output := buf.String()
	assert.Contains(t, output, "test formatted message 123")
}

func TestInfo(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

	Info("test info message")

	output := buf.String()
	assert.Contains(t, output, "test info message")
	assert.Contains(t, output, "\n")
}

func TestInfof(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

	Infof("info %s %d", "test", 456)

	output := buf.String()
	assert.Contains(t, output, "info test 456")
}

func TestWarn(t *testing.T) {
	// Capture stderr output
	buf := &bytes.Buffer{}
	SetStderr(buf)
	defer SetStderr(Stderr())

	Warn("test warning message")

	output := buf.String()
	assert.Contains(t, output, "test warning message")
	assert.Contains(t, output, "\n")
}

func TestWarnf(t *testing.T) {
	// Capture stderr output
	buf := &bytes.Buffer{}
	SetStderr(buf)
	defer SetStderr(Stderr())

	Warnf("warning %s %d", "formatted", 789)

	output := buf.String()
	assert.Contains(t, output, "warning formatted 789")
}

func TestError(t *testing.T) {
	// Capture stderr output
	buf := &bytes.Buffer{}
	SetStderr(buf)
	defer SetStderr(Stderr())

	Error("test error message")

	output := buf.String()
	assert.Contains(t, output, "test error message")
	assert.Contains(t, output, "\n")
}

func TestErrorf(t *testing.T) {
	// Capture stderr output
	buf := &bytes.Buffer{}
	SetStderr(buf)
	defer SetStderr(Stderr())

	Errorf("error %s %d", "formatted", 999)

	output := buf.String()
	assert.Contains(t, output, "error formatted 999")
}

func TestPlain(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

	Plain("test plain message")

	output := buf.String()
	assert.Contains(t, output, "test plain message")
	assert.Contains(t, output, "\n")
}

func TestPlainf(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

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
	defer SetStdout(Stdout())

	progress := NewProgress("testing progress")

	// Start progress
	progress.Start()

	// Let it spin a bit (ticker interval is 100ms, wait for at least one tick)
	time.Sleep(150 * time.Millisecond)

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
	defer SetStdout(Stdout())

	progress := NewProgress("test progress")
	progress.Start()
	time.Sleep(150 * time.Millisecond) // Wait for at least one tick (100ms interval)
	progress.StopWithSuccess("Success message")

	output := buf.String()
	assert.Contains(t, output, "Success message")
}

func TestProgressStopWithError(t *testing.T) {
	// Capture stderr output for error
	stderrBuf := &bytes.Buffer{}
	SetStderr(stderrBuf)
	defer SetStderr(Stderr())

	// Capture stdout for progress
	stdoutBuf := &bytes.Buffer{}
	SetStdout(stdoutBuf)
	defer SetStdout(Stdout())

	progress := NewProgress("test progress")
	progress.Start()
	time.Sleep(150 * time.Millisecond) // Wait for at least one tick (100ms interval)
	progress.StopWithError("Error message")

	errorOutput := stderrBuf.String()
	assert.Contains(t, errorOutput, "Error message")
}

func TestConcurrentOutputFunctions(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

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
	defer SetStdout(Stdout())

	// Create multiple progress indicators
	progress1 := NewProgress("first progress")
	progress2 := NewProgress("second progress")

	// Start both
	progress1.Start()
	progress2.Start()

	// Let them run briefly (ticker interval is 100ms)
	time.Sleep(150 * time.Millisecond)

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

	originalStdout := Stdout()
	originalStderr := Stderr()
	SetStdout(stdoutBuf)
	SetStderr(stderrBuf)
	defer func() {
		SetStdout(originalStdout)
		SetStderr(originalStderr)
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
	defer SetStdout(Stdout())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Success("benchmark message")
	}
}

func BenchmarkConcurrentOutput(b *testing.B) {
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Success("concurrent benchmark message")
		}
	})
}

// Tests for the new ColoredWriter interface
func TestColoredWriter(t *testing.T) {
	t.Run("Success messages", func(t *testing.T) {
		stdoutBuf := &bytes.Buffer{}
		stderrBuf := &bytes.Buffer{}

		writer := NewColoredWriter(stdoutBuf, stderrBuf)

		writer.Success("Success message")
		writer.Successf("Formatted success: %s", "test")

		content := stdoutBuf.String()
		assert.Contains(t, content, "Success message")
		assert.Contains(t, content, "Formatted success: test")
		assert.Empty(t, stderrBuf.String()) // Nothing should go to stderr
	})

	t.Run("Info messages", func(t *testing.T) {
		stdoutBuf := &bytes.Buffer{}
		stderrBuf := &bytes.Buffer{}

		writer := NewColoredWriter(stdoutBuf, stderrBuf)

		writer.Info("Info message")
		writer.Infof("Formatted info: %d", 42)

		content := stdoutBuf.String()
		assert.Contains(t, content, "Info message")
		assert.Contains(t, content, "Formatted info: 42")
		assert.Empty(t, stderrBuf.String())
	})

	t.Run("Warning messages", func(t *testing.T) {
		stdoutBuf := &bytes.Buffer{}
		stderrBuf := &bytes.Buffer{}

		writer := NewColoredWriter(stdoutBuf, stderrBuf)

		writer.Warn("Warning message")
		writer.Warnf("Formatted warning: %v", true)

		content := stderrBuf.String()
		assert.Contains(t, content, "Warning message")
		assert.Contains(t, content, "Formatted warning: true")
		assert.Empty(t, stdoutBuf.String()) // Nothing should go to stdout
	})

	t.Run("Error messages", func(t *testing.T) {
		stdoutBuf := &bytes.Buffer{}
		stderrBuf := &bytes.Buffer{}

		writer := NewColoredWriter(stdoutBuf, stderrBuf)

		writer.Error("Error message")
		writer.Errorf("Formatted error: %s", "failed")

		content := stderrBuf.String()
		assert.Contains(t, content, "Error message")
		assert.Contains(t, content, "Formatted error: failed")
		assert.Empty(t, stdoutBuf.String())
	})

	t.Run("Plain messages", func(t *testing.T) {
		stdoutBuf := &bytes.Buffer{}
		stderrBuf := &bytes.Buffer{}

		writer := NewColoredWriter(stdoutBuf, stderrBuf)

		writer.Plain("Plain message")
		writer.Plainf("Formatted plain: %s", "text")

		content := stdoutBuf.String()
		assert.Contains(t, content, "Plain message")
		assert.Contains(t, content, "Formatted plain: text")
		assert.Empty(t, stderrBuf.String())
	})

	t.Run("Thread safety", func(t *testing.T) {
		stdoutBuf := &bytes.Buffer{}
		stderrBuf := &bytes.Buffer{}

		writer := NewColoredWriter(stdoutBuf, stderrBuf)

		// Run concurrent writes to test thread safety
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				writer.Successf("Message %d", id)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Check that all messages were written
		content := stdoutBuf.String()
		for i := 0; i < 10; i++ {
			assert.Contains(t, content, "Message")
		}

		// Count newlines to ensure all messages were written
		newlines := strings.Count(content, "\n")
		assert.Equal(t, 10, newlines)
	})
}

func TestNewColoredWriter(t *testing.T) {
	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	writer := NewColoredWriter(stdoutBuf, stderrBuf)

	assert.NotNil(t, writer)
	assert.Equal(t, stdoutBuf, writer.stdout)
	assert.Equal(t, stderrBuf, writer.stderr)
	assert.NotNil(t, writer.successColor)
	assert.NotNil(t, writer.infoColor)
	assert.NotNil(t, writer.warnColor)
	assert.NotNil(t, writer.errorColor)
}

// Edge case tests for bug fixes

func TestNewColoredWriter_NilWriters(t *testing.T) {
	// Test that nil writers don't cause panics - they should default to io.Discard
	writer := NewColoredWriter(nil, nil)

	assert.NotNil(t, writer)
	assert.NotNil(t, writer.stdout) // Should be io.Discard, not nil
	assert.NotNil(t, writer.stderr) // Should be io.Discard, not nil

	// These should not panic
	writer.Success("test")
	writer.Info("test")
	writer.Warn("test")
	writer.Error("test")
	writer.Plain("test")
}

func TestProgress_StopWithoutStart(t *testing.T) {
	// Calling Stop() without Start() should not deadlock or panic
	progress := NewProgress("test")

	// This should return immediately without blocking
	done := make(chan struct{})
	go func() {
		progress.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success - Stop() returned without blocking
	case <-time.After(1 * time.Second):
		t.Fatal("Stop() without Start() caused a deadlock")
	}
}

func TestProgress_DoubleStop(t *testing.T) {
	// Calling Stop() twice should not deadlock or panic
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

	progress := NewProgress("test")
	progress.Start()
	time.Sleep(150 * time.Millisecond) // Wait for at least one tick

	// First stop - should work normally
	progress.Stop()

	// Second stop - should be a no-op, not deadlock
	done := make(chan struct{})
	go func() {
		progress.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success - second Stop() returned without blocking
	case <-time.After(1 * time.Second):
		t.Fatal("Double Stop() caused a deadlock")
	}
}

func TestProgress_DoubleStart(t *testing.T) {
	// Calling Start() twice should not spawn multiple goroutines
	buf := &bytes.Buffer{}
	SetStdout(buf)
	defer SetStdout(Stdout())

	progress := NewProgress("test")

	// Start twice
	progress.Start()
	progress.Start() // Should be a no-op

	time.Sleep(150 * time.Millisecond)

	// Stop should work normally (only one goroutine to stop)
	done := make(chan struct{})
	go func() {
		progress.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Stop() after double Start() caused issues")
	}
}

func TestProgress_Idempotent(_ *testing.T) {
	// Test that Progress operations are safe to call in any order
	progress := NewProgress("test")

	// Stop before Start
	progress.Stop()
	progress.Stop()

	// Start after Stop
	progress.Start()

	// Start again (should be no-op since already started)
	progress.Start()

	time.Sleep(150 * time.Millisecond)

	// Multiple stops
	progress.Stop()
	progress.Stop()
	progress.Stop()

	// No panics or deadlocks = success
}

func TestCaptureOutput(t *testing.T) {
	t.Run("captures stdout and stderr", func(t *testing.T) {
		scope := CaptureOutput()
		defer scope.Restore()

		Success("hello from capture")
		Error("error from capture")

		assert.Contains(t, scope.Stdout.String(), "hello from capture")
		assert.Contains(t, scope.Stderr.String(), "error from capture")
	})

	t.Run("restore reverts to original writers", func(t *testing.T) {
		originalStdout := Stdout()
		originalStderr := Stderr()

		scope := CaptureOutput()
		assert.NotEqual(t, originalStdout, Stdout())

		scope.Restore()
		assert.Equal(t, originalStdout, Stdout())
		assert.Equal(t, originalStderr, Stderr())
	})
}
