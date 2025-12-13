package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultExclusions(t *testing.T) {
	exclusions := DefaultExclusions()

	expected := []string{
		"*.out",
		"*.test",
		"*.exe",
		"**/.DS_Store",
		"**/tmp/*",
		"**/.git",
	}

	assert.Equal(t, expected, exclusions)
}

func TestApplyDirectoryDefaults_NilPointer(_ *testing.T) {
	// Should not panic when called with nil
	ApplyDirectoryDefaults(nil)
}

func TestApplyDirectoryDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    DirectoryMapping
		expected DirectoryMapping
	}{
		{
			name: "empty directory mapping gets all defaults",
			input: DirectoryMapping{
				Src:  "src",
				Dest: "dest",
			},
			expected: DirectoryMapping{
				Src:               "src",
				Dest:              "dest",
				Exclude:           DefaultExclusions(),
				PreserveStructure: boolPtr(true),
				IncludeHidden:     boolPtr(true),
			},
		},
		{
			name: "existing exclusions are preserved",
			input: DirectoryMapping{
				Src:     "src",
				Dest:    "dest",
				Exclude: []string{"custom.txt"},
			},
			expected: DirectoryMapping{
				Src:               "src",
				Dest:              "dest",
				Exclude:           []string{"custom.txt"},
				PreserveStructure: boolPtr(true),
				IncludeHidden:     boolPtr(true),
			},
		},
		{
			name: "existing boolean values are preserved",
			input: DirectoryMapping{
				Src:               "src",
				Dest:              "dest",
				PreserveStructure: boolPtr(false),
				IncludeHidden:     boolPtr(false),
			},
			expected: DirectoryMapping{
				Src:               "src",
				Dest:              "dest",
				Exclude:           DefaultExclusions(),
				PreserveStructure: boolPtr(false),
				IncludeHidden:     boolPtr(false),
			},
		},
		{
			name: "transform is preserved",
			input: DirectoryMapping{
				Src:  "src",
				Dest: "dest",
				Transform: Transform{
					RepoName: true,
					Variables: map[string]string{
						"ENV": "prod",
					},
				},
			},
			expected: DirectoryMapping{
				Src:     "src",
				Dest:    "dest",
				Exclude: DefaultExclusions(),
				Transform: Transform{
					RepoName: true,
					Variables: map[string]string{
						"ENV": "prod",
					},
				},
				PreserveStructure: boolPtr(true),
				IncludeHidden:     boolPtr(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := tt.input
			ApplyDirectoryDefaults(&dm)
			assert.Equal(t, tt.expected, dm)
		})
	}
}
