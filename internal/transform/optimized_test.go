package transform

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
	"github.com/mrz1836/go-broadcast/internal/pool"
)

// BenchmarkRegexCache compares performance of cached vs uncached regex compilation
func BenchmarkRegexCache(b *testing.B) {
	patterns := []struct {
		name    string
		pattern string
		input   string
	}{
		{
			"GitHub_URL",
			`github\.com/([^/]+/[^/]+)`,
			"https://github.com/user/repo/blob/main/README.md",
		},
		{
			"Template_Variable",
			`\{\{([A-Z_][A-Z0-9_]*)\}\}`,
			"Welcome to {{SERVICE_NAME}} running on {{ENVIRONMENT}}",
		},
		{
			"GitHub_Token",
			`ghp_[a-zA-Z0-9]{4,}`,
			"Authorization: Bearer ghp_1234567890abcdefghijklmnopqrstu",
		},
		{
			"Branch_Pattern",
			`^(sync/template)-(\d{8})-(\d{6})-([a-fA-F0-9]+)$`,
			"sync/template-20240101-120000-abc123def456",
		},
	}

	for _, pattern := range patterns {
		b.Run(pattern.name+"_Without_Cache", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				re, _ := regexp.Compile(pattern.pattern)
				_ = re.FindStringSubmatch(pattern.input)
			}
		})

		b.Run(pattern.name+"_With_Cache", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				re, _ := CompileRegex(pattern.pattern)
				_ = re.FindStringSubmatch(pattern.input)
			}
		})
	}
}

// BenchmarkStringBuilding compares old concatenation methods with optimized builders
func BenchmarkStringBuilding(b *testing.B) {
	// Test data for different scenarios
	scenarios := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			"Path_Building",
			func(b *testing.B) {
				parts := []string{"github.com", "user", "repo", "blob", "main", "file.go"}

				b.Run("Concatenation", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						result := parts[0]
						for j := 1; j < len(parts); j++ {
							result += "/" + parts[j]
						}
						_ = result
					}
				})

				b.Run("StringBuilder", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = BuildPath("/", parts...)
					}
				})
			},
		},
		{
			"GitHub_URL_Building",
			func(b *testing.B) {
				repo := "user/repository"
				pathParts := []string{"blob", "main", "src", "internal", "service.go"}

				b.Run("Sprintf", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						result := fmt.Sprintf("https://github.com/%s", repo)
						for _, part := range pathParts {
							result += "/" + part
						}
						_ = result
					}
				})

				b.Run("OptimizedBuilder", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = BuildGitHubURL(repo, pathParts...)
					}
				})
			},
		},
		{
			"Branch_Name_Building",
			func(b *testing.B) {
				prefix := "sync/template"
				timestamp := "20240101-120000"
				commitSHA := "abc123def456"

				b.Run("Sprintf", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = fmt.Sprintf("%s-%s-%s", prefix, timestamp, commitSHA)
					}
				})

				b.Run("OptimizedBuilder", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = BuildBranchName(prefix, timestamp, commitSHA)
					}
				})
			},
		},
		{
			"Commit_Message_Building",
			func(b *testing.B) {
				action := "sync"
				subject := "update files from source repository"
				details := []string{
					"Modified: README.md, .github/workflows/ci.yml",
					"Added: docs/api.md",
					"Removed: deprecated/old.txt",
				}

				b.Run("Concatenation", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						result := action + ": " + subject + "\n\n"
						for j, detail := range details {
							result += detail
							if j < len(details)-1 {
								result += "\n"
							}
						}
						_ = result
					}
				})

				b.Run("OptimizedBuilder", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = BuildCommitMessage(action, subject, details...)
					}
				})
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, scenario.fn)
	}
}

// BenchmarkBufferPool compares new buffer allocation with pooled buffers
func BenchmarkBufferPool(b *testing.B) {
	testData := []byte("Some test data to write multiple times for buffer pool testing")

	sizes := []struct {
		name       string
		iterations int
		dataSize   int
	}{
		{"Small_10_Iterations", 10, len(testData)},
		{"Medium_100_Iterations", 100, len(testData)},
		{"Large_1000_Iterations", 1000, len(testData)},
	}

	for _, size := range sizes {
		b.Run(size.name+"_New_Buffer", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf := bytes.NewBuffer(nil)
				for j := 0; j < size.iterations; j++ {
					buf.Write(testData)
				}
				_ = buf.String()
			}
		})

		b.Run(size.name+"_Pooled_Buffer", func(b *testing.B) {
			expectedSize := size.iterations * size.dataSize
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool.WithBuffer(expectedSize, func(buf *bytes.Buffer) error {
					for j := 0; j < size.iterations; j++ {
						buf.Write(testData)
					}
					_ = buf.String()
					return nil
				})
			}
		})
	}
}

// BenchmarkLargeStringBuilding tests large string construction optimization
func BenchmarkLargeStringBuilding(b *testing.B) {
	lineCounts := []int{100, 1000, 5000}

	for _, lineCount := range lineCounts {
		b.Run(fmt.Sprintf("Lines_%d_StringBuilder", lineCount), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var sb strings.Builder
				for j := 0; j < lineCount; j++ {
					sb.WriteString(fmt.Sprintf("This is line number %d with some content\n", j))
				}
				_ = sb.String()
			}
		})

		b.Run(fmt.Sprintf("Lines_%d_OptimizedLarge", lineCount), func(b *testing.B) {
			estimatedSize := lineCount * 50 // ~50 chars per line
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = BuildLargeString(estimatedSize, func(buf *bytes.Buffer) error {
					for j := 0; j < lineCount; j++ {
						fmt.Fprintf(buf, "This is line number %d with some content\n", j)
					}
					return nil
				})
			}
		})
	}
}

// BenchmarkRealWorldScenarios tests performance in realistic usage patterns
func BenchmarkRealWorldScenarios(b *testing.B) {
	b.Run("Template_Transformation", func(b *testing.B) {
		content := `# {{SERVICE_NAME}}

This is a template for {{SERVICE_NAME}} running on {{ENVIRONMENT}}.

## Configuration

- Service: {{SERVICE_NAME}}
- Environment: {{ENVIRONMENT}}
- Repository: {{REPO_NAME}}
- Branch: {{BRANCH_NAME}}

Visit us at: https://github.com/{{ORG_NAME}}/{{REPO_NAME}}`

		variables := map[string]string{
			"SERVICE_NAME": "my-awesome-service",
			"ENVIRONMENT":  "production",
			"REPO_NAME":    "awesome-repo",
			"BRANCH_NAME":  "main",
			"ORG_NAME":     "awesome-org",
		}

		b.Run("Without_Cache", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := content
				for varName, value := range variables {
					pattern := fmt.Sprintf("{{%s}}", varName)
					escapedPattern := regexp.QuoteMeta(pattern)
					re := regexp.MustCompile(escapedPattern)
					result = re.ReplaceAllString(result, value)
				}
				_ = result
			}
		})

		b.Run("With_Cache", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := content
				for varName, value := range variables {
					pattern := fmt.Sprintf("{{%s}}", varName)
					escapedPattern := regexp.QuoteMeta(pattern)
					re, _ := CompileRegex(escapedPattern)
					result = re.ReplaceAllString(result, value)
				}
				_ = result
			}
		})
	})

	b.Run("Repository_Sync_Report", func(b *testing.B) {
		repos := []string{
			"org/repo-1", "org/repo-2", "org/repo-3", "org/repo-4", "org/repo-5",
			"org/repo-6", "org/repo-7", "org/repo-8", "org/repo-9", "org/repo-10",
		}

		files := []string{
			"README.md", ".github/workflows/ci.yml", "Makefile",
			"docker-compose.yml", "src/main.go", "docs/api.md",
		}

		b.Run("String_Concatenation", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				report := "Sync Summary:\n\n"
				report += "Repositories processed: " + fmt.Sprintf("%d", len(repos)) + "\n"
				report += "Files synchronized:\n"
				for _, file := range files {
					report += "  - " + file + "\n"
				}
				report += "\nRepository details:\n"
				for _, repo := range repos {
					report += "  " + repo + ": SUCCESS\n"
				}
				_ = report
			}
		})

		b.Run("Optimized_Builder", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var parts []string
				parts = append(parts, "Sync Summary:", "")
				parts = append(parts, BuildProgressMessage(len(repos), len(repos), "repositories processed"))
				parts = append(parts, "Files synchronized:")
				parts = append(parts, BuildFileList(files, "  - ", "\n"))
				parts = append(parts, "", "Repository details:")

				repoDetails := make(map[string]string)
				for _, repo := range repos {
					repoDetails[repo] = "SUCCESS"
				}
				parts = append(parts, BuildKeyValuePairs(repoDetails, ": ", "\n"))

				_ = strings.Join(parts, "\n")
			}
		})
	})

	b.Run("GitHub_API_URLs", func(b *testing.B) {
		repos := []string{"user/repo1", "user/repo2", "user/repo3", "user/repo4", "user/repo5"}
		endpoints := []string{"issues", "pulls", "commits", "branches", "releases"}

		b.Run("Sprintf_URLs", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				urls := make([]string, 0, len(repos)*len(endpoints))
				for _, repo := range repos {
					for _, endpoint := range endpoints {
						url := fmt.Sprintf("https://api.github.com/repos/%s/%s", repo, endpoint)
						urls = append(urls, url)
					}
				}
				_ = urls
			}
		})

		b.Run("Optimized_URLs", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				urls := make([]string, 0, len(repos)*len(endpoints))
				for _, repo := range repos {
					for _, endpoint := range endpoints {
						url := BuildPath("/", "https://api.github.com", "repos", repo, endpoint)
						urls = append(urls, url)
					}
				}
				_ = urls
			}
		})
	})
}

// BenchmarkMemoryEfficiency tests memory allocation patterns
func BenchmarkMemoryEfficiency(b *testing.B) {
	b.Run("Regex_Compilation_Memory", func(b *testing.B) {
		patterns := []string{
			`github\.com/([^/]+/[^/]+)`,
			`\{\{([A-Z_][A-Z0-9_]*)\}\}`,
			`ghp_[a-zA-Z0-9]{4,}`,
			`^[a-zA-Z0-9][\w.-]*/[a-zA-Z0-9][\w.-]*$`,
			`^(sync/template)-(\d{8})-(\d{6})-([a-fA-F0-9]+)$`,
		}

		b.Run("Without_Cache", func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, "RegexWithoutCache", func() {
				for _, pattern := range patterns {
					_, _ = regexp.Compile(pattern)
				}
			})
		})

		b.Run("With_Cache", func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, "RegexWithCache", func() {
				for _, pattern := range patterns {
					_, _ = CompileRegex(pattern)
				}
			})
		})
	})

	b.Run("String_Building_Memory", func(b *testing.B) {
		parts := []string{"part1", "part2", "part3", "part4", "part5"}

		b.Run("Concatenation", func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, "StringConcatenation", func() {
				result := ""
				for i, part := range parts {
					if i > 0 {
						result += "/"
					}
					result += part
				}
				_ = result
			})
		})

		b.Run("StringBuilder", func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, "StringBuilding", func() {
				_ = BuildPath("/", parts...)
			})
		})
	})

	b.Run("Buffer_Pool_Memory", func(b *testing.B) {
		data := []byte("test data for memory efficiency testing")

		b.Run("New_Buffers", func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, "NewBuffers", func() {
				for i := 0; i < 10; i++ {
					buf := bytes.NewBuffer(nil)
					buf.Write(data)
					_ = buf.String()
				}
			})
		})

		b.Run("Pooled_Buffers", func(b *testing.B) {
			benchmark.RunWithMemoryTracking(b, "PooledBuffers", func() {
				for i := 0; i < 10; i++ {
					_ = pool.WithBuffer(len(data), func(buf *bytes.Buffer) error {
						buf.Write(data)
						_ = buf.String()
						return nil
					})
				}
			})
		})
	})
}
