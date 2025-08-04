package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPRDiff(t *testing.T) {
	tests := []struct {
		name          string
		responseCode  int
		responseBody  string
		expectedError bool
		expectedFiles int
	}{
		{
			name:         "successful diff fetch",
			responseCode: 200,
			responseBody: `[
				{
					"filename": "internal/parser/parser.go",
					"status": "modified",
					"additions": 10,
					"deletions": 5,
					"changes": 15,
					"blob_url": "https://github.com/owner/repo/blob/abc123/internal/parser/parser.go",
					"raw_url": "https://github.com/owner/repo/raw/abc123/internal/parser/parser.go"
				},
				{
					"filename": "README.md",
					"status": "modified",
					"additions": 3,
					"deletions": 1,
					"changes": 4,
					"blob_url": "https://github.com/owner/repo/blob/abc123/README.md",
					"raw_url": "https://github.com/owner/repo/raw/abc123/README.md"
				}
			]`,
			expectedError: false,
			expectedFiles: 2,
		},
		{
			name:          "API error",
			responseCode:  404,
			responseBody:  `{"message": "Not Found"}`,
			expectedError: true,
			expectedFiles: 0,
		},
		{
			name:          "invalid JSON",
			responseCode:  200,
			responseBody:  `invalid json`,
			expectedError: true,
			expectedFiles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/repos/owner/repo/pulls/123/files", r.URL.Path)
				assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))

				w.WriteHeader(tt.responseCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := &Client{
				token:      "test-token",
				baseURL:    server.URL,
				httpClient: &http.Client{},
				config: &Config{
					UserAgent: "test-agent",
				},
			}

			diff, err := client.GetPRDiff(context.Background(), "owner", "repo", 123)

			if tt.expectedError {
				require.Error(t, err)
				assert.Nil(t, diff)
			} else {
				require.NoError(t, err)
				require.NotNil(t, diff)
				assert.Len(t, diff.Files, tt.expectedFiles)

				if tt.expectedFiles > 0 {
					assert.Equal(t, "internal/parser/parser.go", diff.Files[0].Filename)
					assert.Equal(t, "modified", diff.Files[0].Status)
					assert.Equal(t, 10, diff.Files[0].Additions)
					assert.Equal(t, 5, diff.Files[0].Deletions)
					assert.Equal(t, 15, diff.Files[0].Changes)
				}
			}
		})
	}
}

func TestCategorizeFile(t *testing.T) {
	tests := []struct {
		filename     string
		expectedType FileType
	}{
		// Go files
		{"internal/parser/parser.go", FileTypeGo},
		{"cmd/main.go", FileTypeGo},
		{"pkg/utils/helper.go", FileTypeGo},

		// Test files
		{"internal/parser/parser_test.go", FileTypeTest},
		{"cmd/main_test.go", FileTypeTest},
		{"pkg/utils/helper_test.go", FileTypeTest},

		// Generated files
		{"internal/proto/service.pb.go", FileTypeGenerated},
		{"internal/mocks/mock_service.gen.go", FileTypeGenerated},
		{"generated_types.go", FileTypeGenerated},
		{"vendor/github.com/lib/pkg.go", FileTypeGenerated},

		// Config files
		{".github/workflows/test.yml", FileTypeConfig},
		{".github/workflows/test.yaml", FileTypeConfig},
		{"Makefile", FileTypeConfig},
		{"Dockerfile", FileTypeConfig},
		{"go.mod", FileTypeConfig},
		{"go.sum", FileTypeConfig},
		{"package.json", FileTypeConfig},
		{".golangci.yml", FileTypeConfig},
		{".gitignore", FileTypeConfig},
		{"config/app.json", FileTypeConfig},
		{".vscode/settings.json", FileTypeConfig},

		// Documentation files
		{"README.md", FileTypeDocumentation},
		{"CHANGELOG.md", FileTypeDocumentation},
		{"LICENSE", FileTypeDocumentation},
		{"docs/api.md", FileTypeDocumentation},
		{"documentation/guide.rst", FileTypeDocumentation},
		{"examples/sample.txt", FileTypeDocumentation},

		// Other files
		{"main.py", FileTypeOther},
		{"script.sh", FileTypeOther},
		{"image.png", FileTypeOther},
		{"data.csv", FileTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := categorizeFile(tt.filename)
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestAnalyzePRFiles(t *testing.T) {
	prDiff := &PRDiff{
		Files: []PRFile{
			{Filename: "internal/parser/parser.go", Status: "modified", Additions: 10, Deletions: 5},
			{Filename: "internal/parser/parser_test.go", Status: "modified", Additions: 8, Deletions: 2},
			{Filename: "README.md", Status: "modified", Additions: 3, Deletions: 1},
			{Filename: ".github/workflows/test.yml", Status: "added", Additions: 50, Deletions: 0},
			{Filename: "cmd/new_tool.go", Status: "added", Additions: 100, Deletions: 0},
			{Filename: "vendor/lib/pkg.go", Status: "modified", Additions: 0, Deletions: 0},
		},
	}

	analysis := AnalyzePRFiles(prDiff)

	// Check summary totals
	assert.Equal(t, 6, analysis.Summary.TotalFiles)
	assert.Equal(t, 171, analysis.Summary.TotalAdditions)
	assert.Equal(t, 8, analysis.Summary.TotalDeletions)

	// Check Go files
	assert.Len(t, analysis.GoFiles, 2)
	assert.Equal(t, 2, analysis.Summary.GoFilesCount)
	assert.True(t, analysis.Summary.HasGoChanges)
	assert.Equal(t, 110, analysis.Summary.GoAdditions) // 10 + 100
	assert.Equal(t, 5, analysis.Summary.GoDeletions)   // 5 + 0

	// Check test files
	assert.Len(t, analysis.TestFiles, 1)
	assert.Equal(t, 1, analysis.Summary.TestFilesCount)
	assert.True(t, analysis.Summary.HasTestChanges)

	// Check config files
	assert.Len(t, analysis.ConfigFiles, 1)
	assert.Equal(t, 1, analysis.Summary.ConfigFilesCount)
	assert.True(t, analysis.Summary.HasConfigChanges)

	// Check documentation files
	assert.Len(t, analysis.DocumentationFiles, 1)
	assert.Equal(t, 1, analysis.Summary.DocumentationCount)

	// Check generated files
	assert.Len(t, analysis.GeneratedFiles, 1)
	assert.Equal(t, 1, analysis.Summary.GeneratedFilesCount)

	// Check other files
	assert.Empty(t, analysis.OtherFiles)
	assert.Equal(t, 0, analysis.Summary.OtherFilesCount)
}

func TestPRFileSummary_GetSummaryText(t *testing.T) {
	tests := []struct {
		name     string
		summary  PRFileSummary
		expected string
	}{
		{
			name:     "no files",
			summary:  PRFileSummary{},
			expected: "No files changed",
		},
		{
			name: "single Go file",
			summary: PRFileSummary{
				TotalFiles:   1,
				GoFilesCount: 1,
			},
			expected: "1 Go file",
		},
		{
			name: "multiple Go files",
			summary: PRFileSummary{
				TotalFiles:   3,
				GoFilesCount: 3,
			},
			expected: "3 Go files",
		},
		{
			name: "Go and test files",
			summary: PRFileSummary{
				TotalFiles:     2,
				GoFilesCount:   1,
				TestFilesCount: 1,
			},
			expected: "1 Go file and 1 test file",
		},
		{
			name: "multiple file types",
			summary: PRFileSummary{
				TotalFiles:         5,
				GoFilesCount:       2,
				TestFilesCount:     1,
				ConfigFilesCount:   1,
				DocumentationCount: 1,
			},
			expected: "2 Go files, 1 test file, 1 config file, and 1 documentation file",
		},
		{
			name: "only config files",
			summary: PRFileSummary{
				TotalFiles:       3,
				ConfigFilesCount: 3,
			},
			expected: "3 config files",
		},
		{
			name: "mixed with generated files",
			summary: PRFileSummary{
				TotalFiles:          4,
				GoFilesCount:        1,
				GeneratedFilesCount: 2,
				OtherFilesCount:     1,
			},
			expected: "1 Go file, 2 generated files, and 1 other file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.summary.GetSummaryText()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsConfigFile(t *testing.T) {
	tests := []struct {
		basename string
		ext      string
		dir      string
		expected bool
	}{
		// Extensions
		{"config.yml", ".yml", ".", true},
		{"config.yaml", ".yaml", ".", true},
		{"config.json", ".json", ".", true},
		{"config.toml", ".toml", ".", true},
		{"config.xml", ".xml", ".", true},

		// Names
		{"Makefile", "", ".", true},
		{"Dockerfile", "", ".", true},
		{"go.mod", ".mod", ".", true},
		{"go.sum", ".sum", ".", true},
		{".gitignore", "", ".", true},
		{".golangci.json", ".json", ".", true},

		// Directories
		{"anything.txt", ".txt", ".github", true},
		{"settings.json", ".json", ".vscode", true},
		{"app.yaml", ".yaml", "config", true},

		// Non-config files
		{"main.go", ".go", ".", false},
		{"README.md", ".md", ".", false},
		{"test.py", ".py", ".", false},
	}

	for _, tt := range tests {
		t.Run(tt.basename, func(t *testing.T) {
			result := isConfigFile(tt.basename, tt.ext, tt.dir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDocumentationFile(t *testing.T) {
	tests := []struct {
		basename string
		ext      string
		dir      string
		expected bool
	}{
		// Extensions
		{"readme.md", ".md", ".", true},
		{"guide.rst", ".rst", ".", true},
		{"notes.txt", ".txt", ".", true},
		{"manual.adoc", ".adoc", ".", true},

		// Names
		{"README", "", ".", true},
		{"CHANGELOG", "", ".", true},
		{"LICENSE", "", ".", true},
		{"AUTHORS", "", ".", true},

		// Directories
		{"api.md", ".md", "docs", true},
		{"guide.txt", ".txt", "documentation", true},
		{"sample.go", ".go", "examples", true},

		// Non-documentation files
		{"main.go", ".go", ".", false},
		{"config.json", ".json", ".", false},
		{"test.py", ".py", "src", false},
	}

	for _, tt := range tests {
		t.Run(tt.basename, func(t *testing.T) {
			result := isDocumentationFile(tt.basename, tt.ext, tt.dir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGeneratedFile(t *testing.T) {
	tests := []struct {
		basename string
		fullPath string
		expected bool
	}{
		// Patterns
		{"service.pb.go", "internal/proto/service.pb.go", true},
		{"types.gen.go", "internal/types.gen.go", true},
		{"mock_service.go", "mocks/mock_service.go", false}, // Not generated pattern
		{"generated_types.go", "internal/generated_types.go", true},
		{"file.generated", "internal/file.generated", true},

		// Directories
		{"anything.go", "vendor/github.com/lib/anything.go", true},
		{"file.js", "node_modules/lib/file.js", true},
		{"output.bin", "dist/output.bin", true},
		{"types.go", "generated/types.go", true},

		// Non-generated files
		{"main.go", "cmd/main.go", false},
		{"parser.go", "internal/parser.go", false},
		{"helper.go", "pkg/helper.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.basename, func(t *testing.T) {
			result := isGeneratedFile(tt.basename, tt.fullPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		count    int
		expected string
	}{
		{0, "s"},
		{1, ""},
		{2, "s"},
		{10, "s"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("count_%d", tt.count), func(t *testing.T) {
			result := pluralize(tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}
