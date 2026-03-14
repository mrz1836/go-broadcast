package benchmark

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateYAMLConfig(t *testing.T) {
	tests := []struct {
		name        string
		targetCount int
		want        struct {
			contains []string
			length   int
		}
	}{
		{
			name:        "SingleTarget",
			targetCount: 1,
			want: struct {
				contains []string
				length   int
			}{
				contains: []string{"version: 1", "targets:", "repo: \"org/target-repo-0\"", "SERVICE_NAME: \"service-0\""},
				length:   400, // Approximate length check
			},
		},
		{
			name:        "MultipleTargets",
			targetCount: 3,
			want: struct {
				contains []string
				length   int
			}{
				contains: []string{"version: 1", "targets:", "repo: \"org/target-repo-0\"", "repo: \"org/target-repo-2\""},
				length:   800, // Approximate length check
			},
		},
		{
			name:        "ZeroTargets",
			targetCount: 0,
			want: struct {
				contains []string
				length   int
			}{
				contains: []string{"version: 1", "targets:"},
				length:   200, // Base YAML without targets
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateYAMLConfig(tt.targetCount)

			require.NotEmpty(t, result)
			require.Greater(t, len(result), tt.want.length-100) // Allow some variance

			resultStr := string(result)
			for _, expected := range tt.want.contains {
				require.Contains(t, resultStr, expected)
			}
		})
	}
}

func TestGenerateJSONResponse(t *testing.T) {
	tests := []struct {
		name      string
		itemCount int
		want      struct {
			contains []string
			isJSON   bool
		}
	}{
		{
			name:      "SingleItem",
			itemCount: 1,
			want: struct {
				contains []string
				isJSON   bool
			}{
				contains: []string{"\"name\": \"item-0\"", "\"protected\": true"},
				isJSON:   true,
			},
		},
		{
			name:      "MultipleItems",
			itemCount: 3,
			want: struct {
				contains []string
				isJSON   bool
			}{
				contains: []string{"\"name\": \"item-0\"", "\"name\": \"item-2\"", "\"protected\": false"},
				isJSON:   true,
			},
		},
		{
			name:      "ZeroItems",
			itemCount: 0,
			want: struct {
				contains []string
				isJSON   bool
			}{
				contains: []string{"[]"},
				isJSON:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateJSONResponse(tt.itemCount)

			require.NotEmpty(t, result)

			resultStr := string(result)
			for _, expected := range tt.want.contains {
				require.Contains(t, resultStr, expected)
			}

			// Validate JSON structure
			require.Equal(t, byte('['), resultStr[0])
			require.Equal(t, byte(']'), resultStr[len(resultStr)-1])
		})
	}
}

func TestGenerateBase64Content(t *testing.T) {
	tests := []struct {
		name string
		size int
		want struct {
			length    int
			onlyValid bool
		}
	}{
		{
			name: "SmallContent",
			size: 10,
			want: struct {
				length    int
				onlyValid bool
			}{
				length:    14, // (10*4+2)/3 = 14
				onlyValid: true,
			},
		},
		{
			name: "MediumContent",
			size: 100,
			want: struct {
				length    int
				onlyValid bool
			}{
				length:    134, // (100*4+2)/3 = 134
				onlyValid: true,
			},
		},
		{
			name: "ZeroSize",
			size: 0,
			want: struct {
				length    int
				onlyValid bool
			}{
				length:    0,
				onlyValid: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateBase64Content(tt.size)

			require.Len(t, result, tt.want.length)

			// Check that result contains only valid base64 characters
			validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
			for _, char := range result {
				require.Contains(t, validChars, string(char))
			}
		})
	}
}

func TestGenerateLogEntries(t *testing.T) {
	tests := []struct {
		name       string
		count      int
		withTokens bool
		want       struct {
			entryCount  int
			hasTokens   bool
			hasPatterns []string
		}
	}{
		{
			name:       "BasicEntries",
			count:      5,
			withTokens: false,
			want: struct {
				entryCount  int
				hasTokens   bool
				hasPatterns []string
			}{
				entryCount:  5,
				hasTokens:   false,
				hasPatterns: []string{"INFO Processing file:", "DEBUG Git command", "ERROR Failed to clone"},
			},
		},
		{
			name:       "EntriesWithTokens",
			count:      5,
			withTokens: true,
			want: struct {
				entryCount  int
				hasTokens   bool
				hasPatterns []string
			}{
				entryCount:  5,
				hasTokens:   true,
				hasPatterns: []string{"INFO Processing file:", "[token: ghp_"},
			},
		},
		{
			name:       "NoEntries",
			count:      0,
			withTokens: false,
			want: struct {
				entryCount  int
				hasTokens   bool
				hasPatterns []string
			}{
				entryCount: 0,
				hasTokens:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateLogEntries(tt.count, tt.withTokens)

			require.Len(t, result, tt.want.entryCount)

			if tt.want.entryCount > 0 {
				combined := ""
				for _, entry := range result {
					combined += entry
				}

				for _, pattern := range tt.want.hasPatterns {
					require.Contains(t, combined, pattern)
				}

				if tt.withTokens && tt.count > 0 {
					hasToken := false
					for _, entry := range result {
						if len(entry) > 0 && (entry)[0:1] == "I" { // INFO entry (index 0)
							hasToken = true
							require.Contains(t, entry, "[token: ghp_")
							break
						}
					}
					require.True(t, hasToken, "Expected to find token in INFO entries")
				}
			}
		})
	}
}

func TestGenerateGitDiff(t *testing.T) {
	tests := []struct {
		name         string
		fileCount    int
		linesPerFile int
		want         struct {
			hasHeaders bool
			hasChanges bool
			fileCount  int
		}
	}{
		{
			name:         "SingleFile",
			fileCount:    1,
			linesPerFile: 3,
			want: struct {
				hasHeaders bool
				hasChanges bool
				fileCount  int
			}{
				hasHeaders: true,
				hasChanges: true,
				fileCount:  1,
			},
		},
		{
			name:         "MultipleFiles",
			fileCount:    3,
			linesPerFile: 5,
			want: struct {
				hasHeaders bool
				hasChanges bool
				fileCount  int
			}{
				hasHeaders: true,
				hasChanges: true,
				fileCount:  3,
			},
		},
		{
			name:         "NoFiles",
			fileCount:    0,
			linesPerFile: 5,
			want: struct {
				hasHeaders bool
				hasChanges bool
				fileCount  int
			}{
				hasHeaders: false,
				hasChanges: false,
				fileCount:  0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateGitDiff(tt.fileCount, tt.linesPerFile)

			if tt.want.fileCount == 0 {
				require.Empty(t, result)
				return
			}

			require.NotEmpty(t, result)

			if tt.want.hasHeaders {
				require.Contains(t, result, "diff --git")
				require.Contains(t, result, "index ")
				require.Contains(t, result, "@@")
			}

			if tt.want.hasChanges {
				require.Contains(t, result, "-old line")
				require.Contains(t, result, "+new line")
				require.Contains(t, result, " unchanged line")
			}

			// Count file headers to verify file count
			fileHeaders := 0
			lines := []byte(result)
			for i := 0; i < len(lines)-10; i++ {
				if string(lines[i:i+10]) == "diff --git" {
					fileHeaders++
				}
			}
			require.Equal(t, tt.want.fileCount, fileHeaders)
		})
	}
}

func TestGenerateRepositoryList(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  struct {
			repoCount int
			hasFiles  bool
			hasNames  bool
		}
	}{
		{
			name:  "SingleRepo",
			count: 1,
			want: struct {
				repoCount int
				hasFiles  bool
				hasNames  bool
			}{
				repoCount: 1,
				hasFiles:  true,
				hasNames:  true,
			},
		},
		{
			name:  "MultipleRepos",
			count: 5,
			want: struct {
				repoCount int
				hasFiles  bool
				hasNames  bool
			}{
				repoCount: 5,
				hasFiles:  true,
				hasNames:  true,
			},
		},
		{
			name:  "NoRepos",
			count: 0,
			want: struct {
				repoCount int
				hasFiles  bool
				hasNames  bool
			}{
				repoCount: 0,
				hasFiles:  false,
				hasNames:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateRepositoryList(tt.count)

			require.Len(t, result, tt.want.repoCount)

			if tt.want.repoCount > 0 {
				// Check first repo
				repo := result[0]
				require.Equal(t, "org/repo-0", repo.Name)
				require.NotEmpty(t, repo.Files)
				require.NotEmpty(t, repo.Size)
				require.Contains(t, []string{"small", "medium", "large", "xlarge"}, repo.Size)

				// Check file structure
				if tt.want.hasFiles {
					require.NotEmpty(t, repo.Files)
					file := repo.Files[0]
					require.NotEmpty(t, file.Path)
					require.NotEmpty(t, file.Content)
					require.Positive(t, file.Size)
				}
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("GenerateSHA", func(t *testing.T) {
		// Since generateSHA is not exported, we test indirectly through functions that use it
		config := GenerateYAMLConfig(1)
		require.NotEmpty(t, config)

		json := GenerateJSONResponse(1)
		require.Contains(t, string(json), "\"sha\":")
	})

	t.Run("GetSizeCategory", func(t *testing.T) {
		// Test through GenerateRepositoryList which uses getSizeCategory
		repos := GenerateRepositoryList(1)
		require.Len(t, repos, 1)
		require.Contains(t, []string{"small", "medium", "large", "xlarge"}, repos[0].Size)
	})
}

// TestNegativeInputs verifies that all generator functions handle negative inputs gracefully
func TestNegativeInputs(t *testing.T) {
	t.Run("GenerateYAMLConfig_Negative", func(t *testing.T) {
		// Should not panic and should return valid YAML with no targets
		result := GenerateYAMLConfig(-5)
		require.NotEmpty(t, result)
		require.Contains(t, string(result), "version: 1")
		require.Contains(t, string(result), "targets:")
		// Should not contain any target repos
		require.NotContains(t, string(result), "target-repo-")
	})

	t.Run("GenerateJSONResponse_Negative", func(t *testing.T) {
		// Should not panic and should return empty array
		result := GenerateJSONResponse(-5)
		require.Equal(t, "[]", string(result))
	})

	t.Run("GenerateBase64Content_Negative", func(t *testing.T) {
		// Should not panic and should return empty string
		result := GenerateBase64Content(-5)
		require.Empty(t, result)
	})

	t.Run("GenerateLogEntries_Negative", func(t *testing.T) {
		// Should not panic and should return empty slice
		result := GenerateLogEntries(-5, false)
		require.Empty(t, result)
	})

	t.Run("GenerateGitDiff_NegativeFileCount", func(t *testing.T) {
		// Should not panic and should return empty string
		result := GenerateGitDiff(-5, 10)
		require.Empty(t, result)
	})

	t.Run("GenerateGitDiff_NegativeLinesPerFile", func(t *testing.T) {
		// Should not panic and should return diff with no line changes
		result := GenerateGitDiff(1, -5)
		require.NotEmpty(t, result)
		require.Contains(t, result, "diff --git")
	})

	t.Run("GenerateRepositoryList_Negative", func(t *testing.T) {
		// Should not panic and should return empty slice
		result := GenerateRepositoryList(-5)
		require.Empty(t, result)
	})
}
