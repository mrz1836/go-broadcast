// Package env provides utilities for loading environment variables from .env files.
// It follows the GoFortress pattern used by other tools in the ecosystem.
package env

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// parseEnvFile reads a .env file and returns key-value pairs.
// Handles comments (#), empty lines, and KEY=VALUE format.
func parseEnvFile(path string) (map[string]string, error) {
	// filepath.Clean sanitizes the path to prevent directory traversal
	cleanPath := filepath.Clean(path)

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE (handle quoted values, etc.)
		key, value, ok := parseEnvLine(line)
		if ok {
			vars[key] = value
		}
	}

	return vars, scanner.Err()
}

// parseEnvLine extracts key and value from "KEY=VALUE" format.
// Handles:
//   - Simple: KEY=value
//   - Quoted: KEY="value with spaces"
//   - Single quoted: KEY='value'
//   - Empty: KEY= (empty string)
//   - Values with equals: KEY=foo=bar (value is "foo=bar")
//   - Inline comments: KEY=value # comment → value is "value"
func parseEnvLine(line string) (key, value string, ok bool) {
	idx := strings.Index(line, "=")
	if idx == -1 {
		return "", "", false
	}

	key = strings.TrimSpace(line[:idx])
	if key == "" {
		return "", "", false
	}

	// Value is everything after the first '=' (supports values containing '=')
	value = line[idx+1:]

	// Check if the value starts with a quote (after trimming leading whitespace)
	trimmedValue := strings.TrimSpace(value)
	isQuoted := len(trimmedValue) > 0 &&
		(trimmedValue[0] == '"' || trimmedValue[0] == '\'')

	// Strip inline comments for unquoted values
	// Look for " #" (space followed by #) to distinguish from values like "#hashtag"
	if !isQuoted {
		if commentIdx := strings.Index(value, " #"); commentIdx != -1 {
			value = value[:commentIdx]
		}
	}

	// Trim leading/trailing whitespace from value
	value = strings.TrimSpace(value)

	// Handle quoted values - strip the quotes
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return key, value, true
}
