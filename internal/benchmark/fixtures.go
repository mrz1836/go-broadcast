// Package benchmark provides testing fixtures and utilities for performance benchmarking.
package benchmark

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Thread-safe random source for concurrent test execution
var (
	rng   = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec,gochecknoglobals // Using weak random for test fixtures
	rngMu sync.Mutex                                        //nolint:gochecknoglobals // Mutex for thread-safe access to rng
)

// TestRepo represents a test repository configuration
type TestRepo struct {
	Name  string
	Files []TestFile
	Size  string
}

// TestFile represents a test file with content
type TestFile struct {
	Path    string
	Content string
	Size    int
}

// GenerateYAMLConfig creates test YAML configuration data
func GenerateYAMLConfig(targetCount int) []byte {
	// Normalize negative values to 0
	if targetCount < 0 {
		targetCount = 0
	}

	var buf bytes.Buffer
	buf.WriteString(`version: 1
source:
  repo: "org/template-repo"
  branch: "master"
defaults:
  branch_prefix: "chore/sync-files"
  pr_labels: ["automated-sync", "chore"]
targets:`)

	for i := 0; i < targetCount; i++ {
		buf.WriteString(fmt.Sprintf(`
  - repo: "org/target-repo-%d"
    files:
      - src: ".github/workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
      - src: "Makefile"
        dest: "Makefile"
      - src: "README.md"
        dest: "README.md"
    transform:
      repo_name: true
      variables:
        SERVICE_NAME: "service-%d"
        ENVIRONMENT: "production"`, i, i))
	}

	return buf.Bytes()
}

// GenerateJSONResponse creates test JSON response data
func GenerateJSONResponse(itemCount int) []byte {
	// Normalize negative values to 0
	if itemCount < 0 {
		itemCount = 0
	}

	var buf bytes.Buffer
	buf.WriteString("[")

	for i := 0; i < itemCount; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf(`{
  "name": "item-%d",
  "sha": "%s",
  "commit": {
    "sha": "%s",
    "message": "Commit message %d"
  },
  "protected": %t
}`, i, generateSHA(), generateSHA(), i, i%2 == 0))
	}

	buf.WriteString("]")
	return buf.Bytes()
}

// GenerateBase64Content creates base64 encoded test content
func GenerateBase64Content(size int) string {
	// Normalize negative values to 0
	if size < 0 {
		size = 0
	}

	content := make([]byte, size)
	for i := range content {
		content[i] = byte(65 + (i % 26)) // A-Z pattern
	}

	// Simple base64-like encoding for testing
	encoded := make([]byte, (size*4+2)/3)
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	for i := 0; i < len(encoded); i++ {
		encoded[i] = chars[i%len(chars)]
	}

	return string(encoded)
}

// GenerateLogEntries creates test log entries with various patterns
func GenerateLogEntries(count int, withTokens bool) []string {
	// Normalize negative values to 0
	if count < 0 {
		count = 0
	}

	entries := make([]string, count)
	patterns := []string{
		"INFO Processing file: %s",
		"DEBUG Git command executed successfully",
		"ERROR Failed to clone repository: %s",
		"WARN Rate limit approaching: %d requests remaining",
		"INFO Successfully synchronized %d files",
	}

	for i := 0; i < count; i++ {
		pattern := patterns[i%len(patterns)]

		var entry string
		switch i % len(patterns) {
		case 0:
			entry = fmt.Sprintf(pattern, fmt.Sprintf("file-%d.txt", i))
		case 2:
			entry = fmt.Sprintf(pattern, fmt.Sprintf("repo-%d", i))
		case 3:
			entry = fmt.Sprintf(pattern, 1000-i)
		case 4:
			entry = fmt.Sprintf(pattern, i*10)
		default:
			entry = pattern
		}

		// Add tokens to some entries if requested
		if withTokens && i%5 == 0 {
			entry += " [token: ghp_" + generateToken() + "]"
		}

		entries[i] = entry
	}

	return entries
}

// GenerateGitDiff creates a realistic git diff output
func GenerateGitDiff(fileCount, linesPerFile int) string {
	// Normalize negative values to 0
	if fileCount < 0 {
		fileCount = 0
	}
	if linesPerFile < 0 {
		linesPerFile = 0
	}

	var buf strings.Builder

	for i := 0; i < fileCount; i++ {
		filename := fmt.Sprintf("file%d.txt", i)
		buf.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filename, filename))
		buf.WriteString(fmt.Sprintf("index %s..%s 100644\n", generateSHA()[:7], generateSHA()[:7]))
		buf.WriteString(fmt.Sprintf("--- a/%s\n", filename))
		buf.WriteString(fmt.Sprintf("+++ b/%s\n", filename))
		buf.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", linesPerFile, linesPerFile+5))

		for j := 0; j < linesPerFile; j++ {
			if j%3 == 0 {
				buf.WriteString(fmt.Sprintf("-old line %d\n", j))
				buf.WriteString(fmt.Sprintf("+new line %d\n", j))
			} else {
				buf.WriteString(fmt.Sprintf(" unchanged line %d\n", j))
			}
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// GenerateRepositoryList creates test repository data
func GenerateRepositoryList(count int) []TestRepo {
	// Normalize negative values to 0
	if count < 0 {
		count = 0
	}

	repos := make([]TestRepo, count)

	for i := 0; i < count; i++ {
		fileCount := 3 + (i % 10) // 3-12 files per repo
		files := make([]TestFile, fileCount)

		for j := 0; j < fileCount; j++ {
			files[j] = TestFile{
				Path:    fmt.Sprintf("file%d.txt", j),
				Content: fmt.Sprintf("Content for file %d in repo %d\n", j, i),
				Size:    50 + (j * 10),
			}
		}

		repos[i] = TestRepo{
			Name:  fmt.Sprintf("org/repo-%d", i),
			Files: files,
			Size:  getSizeCategory(len(files)),
		}
	}

	return repos
}

// Helper functions

func generateSHA() string {
	const chars = "abcdef0123456789"
	b := make([]byte, 40)
	rngMu.Lock()
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	rngMu.Unlock()
	return string(b)
}

func generateToken() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 20)
	rngMu.Lock()
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	rngMu.Unlock()
	return string(b)
}

func getSizeCategory(fileCount int) string {
	switch {
	case fileCount <= 5:
		return "small"
	case fileCount <= 15:
		return "medium"
	case fileCount <= 50:
		return "large"
	default:
		return "xlarge"
	}
}
