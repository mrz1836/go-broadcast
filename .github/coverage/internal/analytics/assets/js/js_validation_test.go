package js

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJavaScriptSyntaxValidation validates JavaScript files for basic syntax correctness
// This test performs basic structural validation without requiring external JS engines
func TestJavaScriptSyntaxValidation(t *testing.T) {
	jsFiles := []string{
		"coverage-time.js",
		"theme.js",
	}

	for _, filename := range jsFiles {
		t.Run(filename, func(t *testing.T) {
			validateJavaScriptFile(t, filename)
		})
	}
}

// validateJavaScriptFile performs comprehensive syntax validation on a JavaScript file
func validateJavaScriptFile(t *testing.T, filename string) {
	content, err := os.ReadFile(filename) // #nosec G304 - filename is from test constants, not user input
	require.NoError(t, err, "Failed to read JavaScript file: %s", filename)

	contentStr := string(content)
	require.NotEmpty(t, contentStr, "JavaScript file should not be empty: %s", filename)

	// Test 1: Basic bracket/brace/parentheses balance
	t.Run("balanced_brackets", func(t *testing.T) {
		validateBalancedBrackets(t, contentStr, filename)
	})

	// Test 2: String literal validation
	t.Run("string_literals", func(t *testing.T) {
		validateStringLiterals(t, contentStr, filename)
	})

	// Test 3: Function declaration syntax
	t.Run("function_syntax", func(t *testing.T) {
		validateFunctionSyntax(t, contentStr, filename)
	})

	// Test 4: Common syntax errors
	t.Run("common_syntax_errors", func(t *testing.T) {
		validateCommonSyntaxErrors(t, contentStr, filename)
	})

	// Test 5: IIFE (Immediately Invoked Function Expression) validation
	t.Run("iife_validation", func(t *testing.T) {
		validateIIFE(t, contentStr, filename)
	})

	// Test 6: Variable declarations
	t.Run("variable_declarations", func(t *testing.T) {
		validateVariableDeclarations(t, contentStr, filename)
	})

	t.Logf("✅ JavaScript file %s passed all syntax validation tests", filename)
}

// validateBalancedBrackets ensures all brackets, braces, and parentheses are properly balanced
func validateBalancedBrackets(t *testing.T, content, filename string) {
	type bracket struct {
		char string
		line int
	}

	var stack []bracket
	lines := strings.Split(content, "\n")
	inString := false
	inMultiLineComment := false
	stringChar := byte(0)

	for lineNum, line := range lines {
		lineNum++                    // 1-based line numbers
		inSingleLineComment := false // Reset at start of each line

		for i, char := range line {
			switch {
			case inMultiLineComment:
				if i < len(line)-1 && char == '*' && line[i+1] == '/' {
					inMultiLineComment = false
				}
				continue
			case inSingleLineComment:
				continue
			case inString:
				if byte(char) == stringChar && (i == 0 || line[i-1] != '\\') {
					inString = false
					stringChar = 0
				}
				continue
			case char == '"' || char == '\'' || char == '`':
				inString = true
				stringChar = byte(char)
				continue
			case i < len(line)-1 && char == '/' && line[i+1] == '/':
				inSingleLineComment = true
				continue
			case i < len(line)-1 && char == '/' && line[i+1] == '*':
				inMultiLineComment = true
				continue
			}

			switch char {
			case '(', '[', '{':
				stack = append(stack, bracket{string(char), lineNum})
			case ')', ']', '}':
				if len(stack) == 0 {
					t.Errorf("Unmatched closing bracket '%c' at line %d in %s", char, lineNum, filename)
					return
				}
				last := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				expected := map[string]string{")": "(", "]": "[", "}": "{"}
				if last.char != expected[string(char)] {
					t.Errorf("Mismatched brackets: expected '%s' but found '%c' at line %d in %s (opening at line %d)",
						expected[string(char)], char, lineNum, filename, last.line)
					return
				}
			}
		}
	}

	if len(stack) > 0 {
		for _, unclosed := range stack {
			t.Errorf("Unclosed bracket '%s' at line %d in %s", unclosed.char, unclosed.line, filename)
		}
	}
}

// validateStringLiterals checks for properly terminated string literals
func validateStringLiterals(t *testing.T, content, filename string) {
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		lineNum++ // 1-based line numbers
		inString := false
		stringChar := byte(0)
		inComment := false

		for i := 0; i < len(line); i++ {
			char := line[i]

			// Skip comments
			if !inString && i < len(line)-1 && char == '/' && line[i+1] == '/' {
				break
			}
			if !inString && i < len(line)-1 && char == '/' && line[i+1] == '*' {
				// Handle multi-line comments (basic)
				continue
			}

			if inComment {
				continue
			}

			if !inString && (char == '"' || char == '\'' || char == '`') {
				inString = true
				stringChar = char
			} else if inString && char == stringChar {
				// Check if it's escaped
				backslashCount := 0
				for j := i - 1; j >= 0 && line[j] == '\\'; j-- {
					backslashCount++
				}
				escaped := backslashCount%2 == 1

				if !escaped {
					inString = false
					stringChar = 0
				}
			}
		}

		// Check if string is left open at end of line (template literals can span lines)
		if inString && stringChar != '`' {
			t.Errorf("Unterminated string literal at line %d in %s", lineNum, filename)
		}
	}
}

// validateFunctionSyntax checks for basic function declaration syntax
func validateFunctionSyntax(t *testing.T, content, filename string) {
	// Pattern for function declarations and expressions
	functionPattern := regexp.MustCompile(`function\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*\(`)
	arrowFunctionPattern := regexp.MustCompile(`(?:const|let|var)?\s*([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=\s*\([^)]*\)\s*=>`)
	methodPattern := regexp.MustCompile(`([a-zA-Z_$][a-zA-Z0-9_$]*)\s*:\s*function\s*\(`)

	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		lineNum++ // 1-based line numbers

		// Skip comments and strings (basic check)
		if strings.Contains(line, "//") {
			commentIndex := strings.Index(line, "//")
			line = line[:commentIndex]
		}

		// Check for function syntax issues
		if functionPattern.MatchString(line) {
			matches := functionPattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				functionName := matches[1]
				assert.NotEmpty(t, functionName, "Empty function name at line %d in %s", lineNum, filename)

				// Ensure the line contains opening brace or is followed by one
				if !strings.Contains(line, "{") {
					// This might be a multi-line function, which is okay
					t.Logf("Multi-line function detected at line %d", lineNum)
				}
			}
		}

		// Check arrow functions
		if arrowFunctionPattern.MatchString(line) {
			// Basic validation passed for arrow function
			t.Logf("Arrow function detected at line %d", lineNum)
		}

		// Check method definitions
		if methodPattern.MatchString(line) {
			// Basic validation passed for method
			t.Logf("Method definition detected at line %d", lineNum)
		}
	}
}

// validateCommonSyntaxErrors checks for common JavaScript syntax errors
func validateCommonSyntaxErrors(t *testing.T, content, filename string) {
	lines := strings.Split(content, "\n")

	// Pre-compile regexes for better performance
	strayColonPattern := regexp.MustCompile(`\}\)\s*;\s*:`)
	iifeColonPattern := regexp.MustCompile(`\}\)\(\)\s*;\s*:`)

	for lineNum, line := range lines {
		lineNum++ // 1-based line numbers

		// Skip comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}

		// Check for stray colons (like the original bug)
		// Pattern: }); followed by : on the same line
		if strayColonPattern.MatchString(line) {
			t.Errorf("Suspicious syntax - stray colon after IIFE closure at line %d in %s: %s",
				lineNum, filename, strings.TrimSpace(line))
		}

		// Check for })(); followed by colon (more specific to our bug)
		if iifeColonPattern.MatchString(line) {
			t.Errorf("Invalid syntax - colon after IIFE closure at line %d in %s: %s",
				lineNum, filename, strings.TrimSpace(line))
		}

		// Check for consecutive semicolons
		if strings.Contains(line, ";;") {
			t.Errorf("Double semicolon found at line %d in %s", lineNum, filename)
		}

		// Check for missing semicolons after variable declarations (basic check)
		varDeclarationPattern := regexp.MustCompile(`^\s*(const|let|var)\s+[a-zA-Z_$][a-zA-Z0-9_$]*\s*=.*[^;]\s*$`)
		if varDeclarationPattern.MatchString(line) && !strings.Contains(line, "//") {
			// This is a variable declaration that doesn't end with semicolon
			// In modern JS this might be okay, but flag for review
			t.Logf("Variable declaration without semicolon at line %d in %s (this may be intentional)", lineNum, filename)
		}
	}
}

// validateIIFE validates Immediately Invoked Function Expression syntax
func validateIIFE(t *testing.T, content, filename string) {
	// Look for IIFE patterns
	iifePattern := regexp.MustCompile(`\(\s*function\s*\([^)]*\)\s*\{`)
	closurePattern := regexp.MustCompile(`\}\s*\)\s*\(\s*\)\s*;`)

	hasIIFE := iifePattern.MatchString(content)
	hasClosure := closurePattern.MatchString(content)

	if hasIIFE {
		assert.True(t, hasClosure, "IIFE found but missing proper closure })(); in %s", filename)

		// Count IIFE openings and closures
		iifeCount := len(iifePattern.FindAllString(content, -1))
		closureCount := len(closurePattern.FindAllString(content, -1))

		assert.Equal(t, iifeCount, closureCount,
			"Mismatch between IIFE openings (%d) and closures (%d) in %s",
			iifeCount, closureCount, filename)
	}
}

// validateVariableDeclarations checks for proper variable declaration syntax
func validateVariableDeclarations(t *testing.T, content, filename string) {
	lines := strings.Split(content, "\n")

	// Pattern for variable declarations
	varPattern := regexp.MustCompile(`^\s*(const|let|var)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=`)

	for lineNum, line := range lines {
		lineNum++ // 1-based line numbers

		if varPattern.MatchString(line) {
			matches := varPattern.FindStringSubmatch(line)
			if len(matches) >= 3 {
				varType := matches[1]
				varName := matches[2]

				assert.NotEmpty(t, varName, "Empty variable name at line %d in %s", lineNum, filename)

				// Check for reserved words (basic list)
				reservedWords := []string{"function", "return", "if", "else", "for", "while", "do", "switch", "case", "break", "continue"}
				for _, reserved := range reservedWords {
					assert.NotEqual(t, reserved, varName,
						"Variable name '%s' is a reserved word at line %d in %s",
						varName, lineNum, filename)
				}

				// Suggest const over let/var for apparent constants
				if varType != "const" && regexp.MustCompile(`^[A-Z_]+$`).MatchString(varName) {
					t.Logf("Consider using 'const' for apparent constant '%s' at line %d in %s",
						varName, lineNum, filename)
				}
			}
		}
	}
}

// TestSpecificJavaScriptFeatures tests for specific features used in our coverage JS files
func TestSpecificJavaScriptFeatures(t *testing.T) {
	t.Run("coverage-time.js_features", func(t *testing.T) {
		content, err := os.ReadFile("coverage-time.js") // #nosec G304 - test file path is hardcoded
		require.NoError(t, err)
		contentStr := string(content)

		// Test for expected functions
		expectedFunctions := []string{
			"formatFullTimestamp",
			"getRelativeTime",
			"updateTimestampElement",
			"updateAllTimestamps",
			"initializeTimestamps",
		}

		for _, funcName := range expectedFunctions {
			assert.Contains(t, contentStr, funcName,
				"Expected function '%s' not found in coverage-time.js", funcName)
		}

		// Test for proper window object assignment
		assert.Contains(t, contentStr, "window.coverageTime",
			"Missing window.coverageTime assignment")

		// Test for proper IIFE structure
		assert.Contains(t, contentStr, "(function() {",
			"Missing IIFE opening in coverage-time.js")
		assert.Contains(t, contentStr, "})();",
			"Missing IIFE closure in coverage-time.js")
	})

	t.Run("theme.js_features", func(t *testing.T) {
		content, err := os.ReadFile("theme.js") // #nosec G304 - test file path is hardcoded
		require.NoError(t, err)
		contentStr := string(content)

		// Test for expected functions
		expectedFunctions := []string{
			"toggleTheme",
			"togglePackage",
			"copyBadgeURL",
			"fetchLatestGitHubTag",
			"updateVersionDisplay",
		}

		for _, funcName := range expectedFunctions {
			assert.Contains(t, contentStr, funcName,
				"Expected function '%s' not found in theme.js", funcName)
		}
	})
}

// TestJavaScriptFileIntegrity ensures JS files haven't been corrupted
func TestJavaScriptFileIntegrity(t *testing.T) {
	jsFiles := []string{"coverage-time.js", "theme.js"}

	for _, filename := range jsFiles {
		t.Run(filename+"_integrity", func(t *testing.T) {
			info, err := os.Stat(filename)
			require.NoError(t, err, "JavaScript file should exist: %s", filename)

			assert.Greater(t, info.Size(), int64(500),
				"JavaScript file seems too small, possible corruption: %s", filename)

			content, err := os.ReadFile(filename) // #nosec G304 - filename is from test constants, not user input
			require.NoError(t, err)

			// Check for binary/non-text content
			for i, b := range content {
				if b == 0 {
					t.Errorf("Null byte found at position %d in %s - possible binary corruption", i, filename)
					break
				}
			}

			// Ensure file ends with newline
			if len(content) > 0 && content[len(content)-1] != '\n' {
				t.Logf("JavaScript file %s should end with newline", filename)
			}
		})
	}
}
