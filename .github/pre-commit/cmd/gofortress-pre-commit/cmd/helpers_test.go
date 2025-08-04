package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func TestPrintSuccess(t *testing.T) {
	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test with color disabled
	noColor = true
	printSuccess("Test %s", "message")

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "✓ Test message")
}

func TestPrintError(t *testing.T) {
	// Capture stderr output since printError outputs to stderr when noColor is true
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with color disabled
	noColor = true
	printError("Error: %s", "test error")

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "✗ Error: test error")
}

func TestPrintWarning(t *testing.T) {
	// Capture stderr output since printWarning outputs to stderr when noColor is true
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Test with color disabled
	noColor = true
	printWarning("Warning: %s", "test warning")

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "⚠ Warning: test warning")
}

func TestPrintInfo(t *testing.T) {
	// Capture output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test with color disabled
	noColor = true
	printInfo("Info: %s", "test info")

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "ℹ Info: test info")
}

func TestColorHandling(_ *testing.T) {
	// Save original state
	originalNoColor := color.NoColor
	defer func() {
		color.NoColor = originalNoColor
	}()

	// Test with color enabled
	noColor = false
	// This should set color.NoColor to false (color enabled)
	// In actual execution, this happens in init() but we can't test that directly

	// Test with color disabled
	noColor = true
	// This should set color.NoColor to true (color disabled)
}
