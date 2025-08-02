package transform

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/mrz1836/go-broadcast/internal/pool"
)

// BuildPath constructs a path from parts using strings.Builder with capacity pre-allocation.
//
// This function replaces inefficient string concatenation with "+" operator
// and provides optimized path building with minimal memory allocations.
//
// Parameters:
// - separator: String to use between path parts (e.g., "/", "-", "_")
// - parts: Variable number of string parts to join
//
// Returns:
// - Constructed path string
//
// Performance:
// - Pre-calculates total size to minimize reallocations
// - Uses strings.Builder for efficient construction
// - Optimized for common path building patterns
//
// Example:
//
//	path := BuildPath("/", "github.com", "user", "repo", "blob", "master", "README.md")
//	// Result: "github.com/user/repo/blob/main/README.md"
func BuildPath(separator string, parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	if len(parts) == 1 {
		return parts[0]
	}

	// Estimate total size to minimize allocations
	totalSize := len(separator) * (len(parts) - 1) // separators
	for _, part := range parts {
		totalSize += len(part)
	}

	var sb strings.Builder
	sb.Grow(totalSize)

	sb.WriteString(transformBranchName(parts[0]))
	for i := 1; i < len(parts); i++ {
		sb.WriteString(separator)
		sb.WriteString(transformBranchName(parts[i]))
	}

	return sb.String()
}

// BuildGitHubURL constructs a GitHub URL from repository and optional path components.
//
// Parameters:
// - repo: Repository in format "org/repo"
// - pathParts: Optional path components (e.g., "blob", "master", "README.md")
//
// Returns:
// - Complete GitHub URL
//
// Example:
//
//	url := BuildGitHubURL("user/repo", "blob", "master", "README.md")
//	// Result: "https://github.com/user/repo/blob/main/README.md"
func BuildGitHubURL(repo string, pathParts ...string) string {
	baseSize := len("https://github.com/") + len(repo)
	for _, part := range pathParts {
		baseSize += len("/") + len(part)
	}

	var sb strings.Builder
	sb.Grow(baseSize)

	sb.WriteString("https://github.com/")
	sb.WriteString(repo)

	for _, part := range pathParts {
		sb.WriteByte('/')
		sb.WriteString(transformBranchName(part))
	}

	return sb.String()
}

// transformBranchName transforms legacy branch names to their modern equivalents
func transformBranchName(name string) string {
	if name == "master" {
		return "main"
	}
	return name
}

// BuildBranchName constructs a sync branch name with timestamp and commit SHA.
//
// Parameters:
// - prefix: Branch prefix (e.g., "chore/sync-files")
// - timestamp: Timestamp string (e.g., "20240101-120000")
// - commitSHA: Short commit SHA (e.g., "abc123")
//
// Returns:
// - Formatted branch name
//
// Example:
//
//	branch := BuildBranchName("chore/sync-files", "20240101-120000", "abc123")
//	// Result: "chore/sync-files-20240101-120000-abc123"
func BuildBranchName(prefix, timestamp, commitSHA string) string {
	// Pre-calculate size: prefix + "-" + timestamp + "-" + commitSHA
	totalSize := len(prefix) + 1 + len(timestamp) + 1 + len(commitSHA)

	var sb strings.Builder
	sb.Grow(totalSize)

	sb.WriteString(prefix)
	sb.WriteByte('-')
	sb.WriteString(timestamp)
	sb.WriteByte('-')
	sb.WriteString(commitSHA)

	return sb.String()
}

// BuildCommitMessage constructs a commit message with optional details.
//
// Parameters:
// - action: Primary action (e.g., "sync", "update", "add")
// - subject: Subject of the action (e.g., "files from source repository")
// - details: Optional additional details
//
// Returns:
// - Formatted commit message
//
// Example:
//
//	msg := BuildCommitMessage("sync", "update files from source repository", "Modified: README.md, .github/workflows/ci.yml")
//	// Result: "sync: update files from source repository\n\nModified: README.md, .github/workflows/ci.yml"
func BuildCommitMessage(action, subject string, details ...string) string {
	// Base size: action + ": " + subject
	baseSize := len(action) + 2 + len(subject)

	if len(details) > 0 {
		baseSize += 2 // "\n\n"
		for _, detail := range details {
			baseSize += len(detail) + 1 // detail + "\n"
		}
	}

	var sb strings.Builder
	sb.Grow(baseSize)

	sb.WriteString(action)
	sb.WriteString(": ")
	sb.WriteString(subject)

	if len(details) > 0 {
		sb.WriteString("\n\n")
		for i, detail := range details {
			sb.WriteString(detail)
			if i < len(details)-1 {
				sb.WriteByte('\n')
			}
		}
	}

	return sb.String()
}

// BuildFileList constructs a formatted list of files with optional prefix.
//
// Parameters:
// - files: Slice of file paths
// - prefix: Optional prefix for each file (e.g., "- ", "  ")
// - separator: Separator between files (e.g., "\n", ", ")
//
// Returns:
// - Formatted file list string
//
// Example:
//
//	list := BuildFileList([]string{"README.md", "main.go"}, "- ", "\n")
//	// Result: "- README.md\n- main.go"
func BuildFileList(files []string, prefix, separator string) string {
	if len(files) == 0 {
		return ""
	}

	// Estimate total size
	totalSize := 0
	for _, file := range files {
		totalSize += len(prefix) + len(file)
	}
	totalSize += len(separator) * (len(files) - 1)

	var sb strings.Builder
	sb.Grow(totalSize)

	for i, file := range files {
		if i > 0 {
			sb.WriteString(separator)
		}
		sb.WriteString(prefix)
		sb.WriteString(file)
	}

	return sb.String()
}

// BuildKeyValuePairs constructs a formatted list of key-value pairs.
//
// Parameters:
// - pairs: Map of key-value pairs
// - keyValueSep: Separator between key and value (e.g., ": ", "=")
// - pairSep: Separator between pairs (e.g., "\n", ", ")
//
// Returns:
// - Formatted key-value string
//
// Example:
//
//	kvs := BuildKeyValuePairs(map[string]string{"repo": "user/repo", "branch": "master"}, ": ", "\n")
//	// Result: "repo: user/repo\nbranch: main"
func BuildKeyValuePairs(pairs map[string]string, keyValueSep, pairSep string) string {
	if len(pairs) == 0 {
		return ""
	}

	// Convert map to slice for consistent ordering
	keys := make([]string, 0, len(pairs))
	totalSize := 0

	for key := range pairs {
		keys = append(keys, key)
		totalSize += len(key) + len(keyValueSep) + len(pairs[key])
	}
	totalSize += len(pairSep) * (len(pairs) - 1)

	var sb strings.Builder
	sb.Grow(totalSize)

	for i, key := range keys {
		if i > 0 {
			sb.WriteString(pairSep)
		}
		sb.WriteString(key)
		sb.WriteString(keyValueSep)
		sb.WriteString(transformBranchName(pairs[key]))
	}

	return sb.String()
}

// BuildLargeString constructs large strings using buffer pools for optimal memory usage.
//
// This function is designed for scenarios where the resulting string is expected
// to be large (>8KB) and benefits from buffer pool allocation strategies.
//
// Parameters:
// - estimatedSize: Estimated final string size in bytes
// - fn: Function that builds the string using the provided buffer
//
// Returns:
// - Constructed string
// - Error from the building function
//
// Example:
//
//	result, err := BuildLargeString(50000, func(buf *bytes.Buffer) error {
//	    for i := 0; i < 1000; i++ {
//	        buf.WriteString(fmt.Sprintf("Line %d\n", i))
//	    }
//	    return nil
//	})
func BuildLargeString(estimatedSize int, fn func(buf *bytes.Buffer) error) (string, error) {
	// For very large strings, use buffer pool integration
	if estimatedSize > pool.LargeBufferThreshold {
		return pool.WithBufferResult(estimatedSize, func(buf *bytes.Buffer) (string, error) {
			if err := fn(buf); err != nil {
				return "", err
			}
			return buf.String(), nil
		})
	}

	// For smaller large strings, use strings.Builder directly but with buffer pool
	return pool.WithBufferResult(estimatedSize, func(buf *bytes.Buffer) (string, error) {
		if err := fn(buf); err != nil {
			return "", err
		}
		return buf.String(), nil
	})
}

// BuildURLWithParams constructs a URL with query parameters.
//
// Parameters:
// - baseURL: Base URL without parameters
// - params: Map of parameter names to values
//
// Returns:
// - Complete URL with encoded parameters
//
// Example:
//
//	url := BuildURLWithParams("https://api.github.com/repos/user/repo", map[string]string{
//	    "per_page": "100",
//	    "state": "open",
//	})
//	// Result: "https://api.github.com/repos/user/repo?per_page=100&state=open"
func BuildURLWithParams(baseURL string, params map[string]string) string {
	if len(params) == 0 {
		return baseURL
	}

	// Estimate size: baseURL + "?" + params
	totalSize := len(baseURL) + 1 // baseURL + "?"
	for key, value := range params {
		totalSize += len(key) + 1 + len(value) + 1 // key=value&
	}

	var sb strings.Builder
	sb.Grow(totalSize)

	sb.WriteString(baseURL)
	sb.WriteByte('?')

	first := true
	for key, value := range params {
		if !first {
			sb.WriteByte('&')
		}
		sb.WriteString(key)
		sb.WriteByte('=')
		sb.WriteString(value)
		first = false
	}

	return sb.String()
}

// BuildProgressMessage constructs a progress message with current/total counts.
//
// Parameters:
// - current: Current progress count
// - total: Total expected count
// - operation: Description of the operation
//
// Returns:
// - Formatted progress message
//
// Example:
//
//	msg := BuildProgressMessage(5, 10, "repositories processed")
//	// Result: "5/10 repositories processed"
func BuildProgressMessage(current, total int, operation string) string {
	currentStr := strconv.Itoa(current)
	totalStr := strconv.Itoa(total)

	// Size: current + "/" + total + " " + operation
	totalSize := len(currentStr) + 1 + len(totalStr) + 1 + len(operation)

	var sb strings.Builder
	sb.Grow(totalSize)

	sb.WriteString(currentStr)
	sb.WriteByte('/')
	sb.WriteString(totalStr)
	sb.WriteByte(' ')
	sb.WriteString(operation)

	return sb.String()
}
