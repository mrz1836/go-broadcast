// Package github provides GitHub API integration for coverage reporting
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// PRFile represents a file in a PR diff
type PRFile struct {
	Filename         string `json:"filename"`
	Status           string `json:"status"` // "added", "removed", "modified", "renamed"
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Changes          int    `json:"changes"`
	Patch            string `json:"patch,omitempty"` // Diff content
	BlobURL          string `json:"blob_url"`
	RawURL           string `json:"raw_url"`
	PreviousFilename string `json:"previous_filename,omitempty"` // For renamed files
}

// PRDiff represents the diff of a pull request
type PRDiff struct {
	Files []PRFile `json:"files"`
}

// FileType represents the type/category of a file
type FileType string

// File type constants for categorizing PR files
const (
	FileTypeGo            FileType = "go"
	FileTypeTest          FileType = "test"
	FileTypeConfig        FileType = "config"
	FileTypeDocumentation FileType = "documentation"
	FileTypeGenerated     FileType = "generated"
	FileTypeOther         FileType = "other"
)

// PRFileAnalysis contains analyzed information about PR files
type PRFileAnalysis struct {
	GoFiles            []PRFile
	TestFiles          []PRFile
	ConfigFiles        []PRFile
	DocumentationFiles []PRFile
	GeneratedFiles     []PRFile
	OtherFiles         []PRFile
	Summary            PRFileSummary
}

// PRFileSummary provides summary statistics about the PR files
type PRFileSummary struct {
	TotalFiles          int
	GoFilesCount        int
	TestFilesCount      int
	ConfigFilesCount    int
	DocumentationCount  int
	GeneratedFilesCount int
	OtherFilesCount     int
	HasGoChanges        bool
	HasTestChanges      bool
	HasConfigChanges    bool
	TotalAdditions      int
	TotalDeletions      int
	GoAdditions         int
	GoDeletions         int
}

// GetPRDiff retrieves the diff for a pull request
func (c *Client) GetPRDiff(ctx context.Context, owner, repo string, pr int) (*PRDiff, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files", c.baseURL, owner, repo, pr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR diff: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var files []PRFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to decode PR diff: %w", err)
	}

	return &PRDiff{Files: files}, nil
}

// AnalyzePRFiles categorizes and analyzes the files in a PR
func AnalyzePRFiles(prDiff *PRDiff) *PRFileAnalysis {
	analysis := &PRFileAnalysis{
		Summary: PRFileSummary{},
	}

	for _, file := range prDiff.Files {
		fileType := categorizeFile(file.Filename)

		// Update summary totals
		analysis.Summary.TotalFiles++
		analysis.Summary.TotalAdditions += file.Additions
		analysis.Summary.TotalDeletions += file.Deletions

		// Categorize files
		switch fileType {
		case FileTypeGo:
			analysis.GoFiles = append(analysis.GoFiles, file)
			analysis.Summary.GoFilesCount++
			analysis.Summary.HasGoChanges = true
			analysis.Summary.GoAdditions += file.Additions
			analysis.Summary.GoDeletions += file.Deletions

		case FileTypeTest:
			analysis.TestFiles = append(analysis.TestFiles, file)
			analysis.Summary.TestFilesCount++
			analysis.Summary.HasTestChanges = true

		case FileTypeConfig:
			analysis.ConfigFiles = append(analysis.ConfigFiles, file)
			analysis.Summary.ConfigFilesCount++
			analysis.Summary.HasConfigChanges = true

		case FileTypeDocumentation:
			analysis.DocumentationFiles = append(analysis.DocumentationFiles, file)
			analysis.Summary.DocumentationCount++

		case FileTypeGenerated:
			analysis.GeneratedFiles = append(analysis.GeneratedFiles, file)
			analysis.Summary.GeneratedFilesCount++

		default:
			analysis.OtherFiles = append(analysis.OtherFiles, file)
			analysis.Summary.OtherFilesCount++
		}
	}

	return analysis
}

// categorizeFile determines the type/category of a file based on its path and extension
func categorizeFile(filename string) FileType {
	basename := filepath.Base(filename)
	ext := filepath.Ext(filename)
	dir := filepath.Dir(filename)

	// Check for Go files (excluding tests)
	if ext == ".go" && !strings.HasSuffix(basename, "_test.go") {
		// Check if it's a generated file
		if isGeneratedFile(basename, filename) {
			return FileTypeGenerated
		}
		return FileTypeGo
	}

	// Check for test files
	if ext == ".go" && strings.HasSuffix(basename, "_test.go") {
		return FileTypeTest
	}

	// Check for configuration files
	if isConfigFile(basename, ext, dir) {
		return FileTypeConfig
	}

	// Check for documentation files
	if isDocumentationFile(basename, ext, dir) {
		return FileTypeDocumentation
	}

	// Check for generated files
	if isGeneratedFile(basename, filename) {
		return FileTypeGenerated
	}

	return FileTypeOther
}

// isConfigFile determines if a file is a configuration file
func isConfigFile(basename, ext, dir string) bool {
	// Common config file extensions
	configExts := []string{".yml", ".yaml", ".json", ".toml", ".xml", ".ini", ".cfg", ".conf"}
	for _, configExt := range configExts {
		if ext == configExt {
			return true
		}
	}

	// Common config file names
	configNames := []string{
		"Makefile", "makefile", "Dockerfile", "dockerfile", ".dockerignore",
		".gitignore", ".gitattributes", ".gitmodules",
		"go.mod", "go.sum", "package.json", "package-lock.json",
		".eslintrc", ".prettierrc", ".golangci.json", ".golangci.yml",
	}
	for _, configName := range configNames {
		if basename == configName {
			return true
		}
	}

	// Files in config directories
	configDirs := []string{".github", ".vscode", ".idea", "config", "configs", "deployment"}
	for _, configDir := range configDirs {
		if strings.HasPrefix(dir, configDir) {
			return true
		}
	}

	return false
}

// isDocumentationFile determines if a file is documentation
func isDocumentationFile(basename, ext, dir string) bool {
	// Documentation extensions
	docExts := []string{".md", ".rst", ".txt", ".adoc", ".asciidoc"}
	for _, docExt := range docExts {
		if ext == docExt {
			return true
		}
	}

	// Documentation directories
	docDirs := []string{"docs", "doc", "documentation", "man", "examples", "sample"}
	for _, docDir := range docDirs {
		if strings.HasPrefix(dir, docDir) || strings.Contains(dir, "/"+docDir+"/") {
			return true
		}
	}

	// Common documentation files
	docNames := []string{
		"README", "CHANGELOG", "CHANGES", "LICENSE", "COPYING",
		"AUTHORS", "CONTRIBUTORS", "MAINTAINERS", "NOTICE",
	}
	nameWithoutExt := strings.TrimSuffix(basename, ext)
	for _, docName := range docNames {
		if strings.EqualFold(nameWithoutExt, docName) {
			return true
		}
	}

	return false
}

// isGeneratedFile determines if a file is generated
func isGeneratedFile(basename, fullPath string) bool {
	// Common generated file patterns
	generatedPatterns := []string{
		".pb.go",     // Protocol buffers
		".gen.go",    // General generated Go files
		"_gen.go",    // General generated Go files
		"generated_", // Files starting with generated_
		".generated", // Files ending with .generated
	}

	for _, pattern := range generatedPatterns {
		if strings.Contains(basename, pattern) {
			return true
		}
	}

	// Generated directories
	generatedDirs := []string{"vendor", "node_modules", "dist", "build", "generated", "gen"}
	for _, genDir := range generatedDirs {
		if strings.HasPrefix(fullPath, genDir+"/") || strings.Contains(fullPath, "/"+genDir+"/") {
			return true
		}
	}

	return false
}

// GetSummaryText generates a human-readable summary of PR files
func (s *PRFileSummary) GetSummaryText() string {
	if s.TotalFiles == 0 {
		return "No files changed"
	}

	var parts []string

	if s.GoFilesCount > 0 {
		parts = append(parts, fmt.Sprintf("%d Go file%s", s.GoFilesCount, pluralize(s.GoFilesCount)))
	}

	if s.TestFilesCount > 0 {
		parts = append(parts, fmt.Sprintf("%d test file%s", s.TestFilesCount, pluralize(s.TestFilesCount)))
	}

	if s.ConfigFilesCount > 0 {
		parts = append(parts, fmt.Sprintf("%d config file%s", s.ConfigFilesCount, pluralize(s.ConfigFilesCount)))
	}

	if s.DocumentationCount > 0 {
		parts = append(parts, fmt.Sprintf("%d documentation file%s", s.DocumentationCount, pluralize(s.DocumentationCount)))
	}

	if s.GeneratedFilesCount > 0 {
		parts = append(parts, fmt.Sprintf("%d generated file%s", s.GeneratedFilesCount, pluralize(s.GeneratedFilesCount)))
	}

	if s.OtherFilesCount > 0 {
		parts = append(parts, fmt.Sprintf("%d other file%s", s.OtherFilesCount, pluralize(s.OtherFilesCount)))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%d file%s", s.TotalFiles, pluralize(s.TotalFiles))
	}

	if len(parts) == 1 {
		return parts[0]
	}

	if len(parts) == 2 {
		return parts[0] + " and " + parts[1]
	}

	// More than 2 parts
	lastPart := parts[len(parts)-1]
	otherParts := strings.Join(parts[:len(parts)-1], ", ")
	return otherParts + ", and " + lastPart
}

// pluralize returns "s" if count != 1, empty string otherwise
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
