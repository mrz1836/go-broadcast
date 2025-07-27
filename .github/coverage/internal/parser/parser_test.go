package parser

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) { //nolint:revive // function naming
	parser := New()
	assert.NotNil(t, parser)
	assert.NotNil(t, parser.config)
	assert.Equal(t, []string{"test/", "vendor/", "examples/", "third_party/", "testdata/"}, parser.config.ExcludePaths)
	assert.Equal(t, []string{"*_test.go", "*.pb.go", "*_mock.go", "mock_*.go"}, parser.config.ExcludeFiles)
	assert.True(t, parser.config.ExcludeGenerated)
	assert.True(t, parser.config.ExcludeTestFiles)
	assert.Equal(t, 10, parser.config.MinFileLines)
}

func TestNewWithConfig(t *testing.T) { //nolint:revive // function naming
	config := &Config{
		ExcludePaths:     []string{"custom/"},
		ExcludeFiles:     []string{"*.custom"},
		ExcludeGenerated: false,
		ExcludeTestFiles: false,
		MinFileLines:     5,
	}

	parser := NewWithConfig(config)
	assert.NotNil(t, parser)
	assert.Equal(t, config, parser.config)
}

func TestParseFile(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx := context.Background()

	// Test valid coverage file
	coverage, err := parser.ParseFile(ctx, "testdata/coverage.txt")
	require.NoError(t, err)
	assert.NotNil(t, coverage)
	assert.Equal(t, "atomic", coverage.Mode)
	assert.True(t, coverage.Percentage >= 0 && coverage.Percentage <= 100)
	assert.Positive(t, coverage.TotalLines)
	assert.WithinDuration(t, time.Now(), coverage.Timestamp, 5*time.Second)
}

func TestParseFileNotExists(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx := context.Background()

	_, err := parser.ParseFile(ctx, "testdata/nonexistent.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open coverage file")
}

func TestParseValidCoverage(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx := context.Background()

	coverageData := `mode: atomic
github.com/example/pkg/file.go:10.1,12.2 2 1
github.com/example/pkg/file.go:15.1,17.16 2 0
github.com/example/pkg/other.go:20.1,22.2 1 1`

	reader := strings.NewReader(coverageData)
	coverage, err := parser.Parse(ctx, reader)

	require.NoError(t, err)
	assert.Equal(t, "atomic", coverage.Mode)
	assert.Equal(t, 5, coverage.TotalLines)            // 2 + 2 + 1 = 5 total statements
	assert.Equal(t, 3, coverage.CoveredLines)          // 2 + 0 + 1 = 3 covered statements
	assert.InDelta(t, 60.0, coverage.Percentage, 0.01) // 3/5 = 60%

	// Check packages
	assert.Len(t, coverage.Packages, 1)
	pkg, exists := coverage.Packages["pkg"]
	assert.True(t, exists)
	assert.Equal(t, "pkg", pkg.Name)
	assert.Len(t, pkg.Files, 2)
}

func TestParseInvalidMode(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx := context.Background()

	invalidData := `invalid mode line
github.com/example/pkg/file.go:10.1,12.2 2 1`

	reader := strings.NewReader(invalidData)
	_, err := parser.Parse(ctx, reader)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid coverage file: first line must specify mode")
}

func TestParseInvalidStatement(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx := context.Background()

	invalidData := `mode: atomic
invalid statement format`

	reader := strings.NewReader(invalidData)
	_, err := parser.Parse(ctx, reader)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse line")
}

func TestParseStatementValid(t *testing.T) { //nolint:revive // function naming
	parser := New()

	stmt, filename, err := parser.parseStatement("github.com/example/pkg/file.go:10.5,12.10 2 1")

	require.NoError(t, err)
	assert.Equal(t, "github.com/example/pkg/file.go", filename)
	assert.Equal(t, 10, stmt.StartLine)
	assert.Equal(t, 5, stmt.StartCol)
	assert.Equal(t, 12, stmt.EndLine)
	assert.Equal(t, 10, stmt.EndCol)
	assert.Equal(t, 2, stmt.NumStmt)
	assert.Equal(t, 1, stmt.Count)
}

func TestParseStatementInvalidFormat(t *testing.T) { //nolint:revive // function naming
	parser := New()

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "too few fields",
			line: "file.go:10.5,12.10",
			want: "invalid statement format: expected 3 fields",
		},
		{
			name: "missing colon",
			line: "file.go 10.5,12.10 2 1",
			want: "invalid statement format: expected 3 fields",
		},
		{
			name: "missing comma",
			line: "file.go:10.5 12.10 2 1",
			want: "invalid statement format: expected 3 fields",
		},
		{
			name: "invalid start position",
			line: "file.go:invalid,12.10 2 1",
			want: "invalid start position",
		},
		{
			name: "invalid numStmt",
			line: "file.go:10.5,12.10 invalid 1",
			want: "invalid numStmt",
		},
		{
			name: "invalid count",
			line: "file.go:10.5,12.10 2 invalid",
			want: "invalid count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parser.parseStatement(tt.line)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestParsePosition(t *testing.T) { //nolint:revive // function naming
	parser := New()

	line, col, err := parser.parsePosition("10.15")
	require.NoError(t, err)
	assert.Equal(t, 10, line)
	assert.Equal(t, 15, col)
}

func TestParsePositionInvalid(t *testing.T) { //nolint:revive // function naming
	parser := New()

	tests := []struct {
		name string
		pos  string
		want string
	}{
		{
			name: "missing dot",
			pos:  "1015",
			want: "invalid position format: missing dot",
		},
		{
			name: "invalid line",
			pos:  "invalid.15",
			want: "invalid line number",
		},
		{
			name: "invalid column",
			pos:  "10.invalid",
			want: "invalid column number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parser.parsePosition(tt.pos)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestShouldExcludeFile(t *testing.T) { //nolint:revive // function naming
	parser := New()

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "exclude test file",
			filename: "internal/config/config_test.go",
			want:     true,
		},
		{
			name:     "exclude vendor path",
			filename: "vendor/github.com/lib/pkg.go",
			want:     true,
		},
		{
			name:     "exclude testdata path",
			filename: "internal/testdata/file.go",
			want:     true,
		},
		{
			name:     "exclude protobuf file",
			filename: "internal/proto/service.pb.go",
			want:     true,
		},
		{
			name:     "exclude mock file",
			filename: "internal/mocks/mock_service.go",
			want:     true,
		},
		{
			name:     "include regular file",
			filename: "internal/config/config.go",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.shouldExcludeFile(tt.filename)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestShouldExcludeFileIncludeOnly(t *testing.T) { //nolint:revive // function naming
	config := &Config{
		IncludeOnlyPaths: []string{"internal/", "pkg/"},
	}
	parser := NewWithConfig(config)

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{
			name:     "include internal path",
			filename: "internal/config/config.go",
			want:     false,
		},
		{
			name:     "include pkg path",
			filename: "pkg/utils/helper.go",
			want:     false,
		},
		{
			name:     "exclude cmd path",
			filename: "cmd/main.go",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.shouldExcludeFile(tt.filename)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestExtractPackageName(t *testing.T) { //nolint:revive // function naming
	parser := New()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "nested package",
			filename: "github.com/example/internal/config/config.go",
			want:     "config",
		},
		{
			name:     "root package",
			filename: "main.go",
			want:     "main",
		},
		{
			name:     "single level",
			filename: "pkg/file.go",
			want:     "pkg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractPackageName(tt.filename)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCalculateFileCoverage(t *testing.T) { //nolint:revive // function naming
	parser := New()

	statements := []Statement{
		{StartLine: 10, NumStmt: 2, Count: 1},
		{StartLine: 15, NumStmt: 3, Count: 0},
		{StartLine: 20, NumStmt: 1, Count: 2},
	}

	fileCov := parser.calculateFileCoverage("test.go", statements)

	assert.Equal(t, "test.go", fileCov.Path)
	assert.Equal(t, 6, fileCov.TotalLines)
	assert.Equal(t, 3, fileCov.CoveredLines)
	assert.InDelta(t, 50.0, fileCov.Percentage, 0.01)
	assert.Len(t, fileCov.Statements, 3)

	// Check statements are sorted by line number
	assert.Equal(t, 10, fileCov.Statements[0].StartLine)
	assert.Equal(t, 15, fileCov.Statements[1].StartLine)
	assert.Equal(t, 20, fileCov.Statements[2].StartLine)
}

func TestParseContextCancellation(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	coverageData := `mode: atomic
github.com/example/pkg/file.go:10.1,12.2 2 1`

	reader := strings.NewReader(coverageData)
	_, err := parser.Parse(ctx, reader)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestParseComplexCoverage(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx := context.Background()

	// Test with complex coverage file
	coverage, err := parser.ParseFile(ctx, "testdata/complex.txt")
	require.NoError(t, err)

	assert.Equal(t, "count", coverage.Mode)
	assert.Positive(t, coverage.Percentage)
	assert.NotEmpty(t, coverage.Packages)

	// Verify vendor files are excluded
	for _, pkg := range coverage.Packages {
		for filename := range pkg.Files {
			assert.NotContains(t, filename, "vendor/")
			assert.NotContains(t, filename, "_test.go")
		}
	}
}

func TestIsGeneratedFile(t *testing.T) { //nolint:revive // function naming
	parser := New()

	// Create a temporary generated file
	tmpFile, err := os.CreateTemp("", "generated_test_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write generated file content
	_, err = tmpFile.WriteString("// Code generated by protoc-gen-go. DO NOT EDIT.\npackage test\n")
	require.NoError(t, err)
	_ = tmpFile.Close()

	assert.True(t, parser.isGeneratedFile(tmpFile.Name()))

	// Test with non-generated file
	tmpFile2, err := os.CreateTemp("", "regular_test_*.go")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile2.Name()) }()

	_, err = tmpFile2.WriteString("package test\n\nfunc main() {}\n")
	require.NoError(t, err)
	_ = tmpFile2.Close()

	assert.False(t, parser.isGeneratedFile(tmpFile2.Name()))
}

func TestParseEmptyFile(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx := context.Background()

	reader := strings.NewReader("")
	_, err := parser.Parse(ctx, reader)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid coverage file: missing mode declaration")
}

func TestParseOnlyMode(t *testing.T) { //nolint:revive // function naming
	parser := New()
	ctx := context.Background()

	reader := strings.NewReader("mode: atomic")
	coverage, err := parser.Parse(ctx, reader)

	require.NoError(t, err)
	assert.Equal(t, "atomic", coverage.Mode)
	assert.Equal(t, 0, coverage.TotalLines)
	assert.Equal(t, 0, coverage.CoveredLines)
	assert.InDelta(t, 0.0, coverage.Percentage, 0.001)
}
