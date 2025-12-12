package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePRBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only returns empty",
			input:    "   \n\t  ",
			expected: "",
		},
		{
			name:     "single line commit message rejected",
			input:    "sync: update files from source repository",
			expected: "",
		},
		{
			name:     "sync with scope commit message rejected",
			input:    "sync(ci): update GitHub Actions and mage-x to v1.8.15",
			expected: "",
		},
		{
			name:     "chore prefix rejected",
			input:    "chore: update dependencies\n\nsome body",
			expected: "",
		},
		{
			name:     "feat prefix rejected",
			input:    "feat: add new feature\n\nsome body",
			expected: "",
		},
		{
			name:     "fix prefix rejected",
			input:    "fix: resolve bug\n\nsome body",
			expected: "",
		},
		{
			name:     "docs prefix rejected",
			input:    "docs: update readme\n\nsome body",
			expected: "",
		},
		{
			name:     "multiline without headers rejected",
			input:    "This is a description\nwith multiple lines\nbut no headers",
			expected: "",
		},
		{
			name: "valid PR body with headers accepted",
			input: `## What Changed
* Updated workflow files
* Modified CI configuration

## Why It Was Necessary
* Keeps repository aligned with source`,
			expected: `## What Changed
* Updated workflow files
* Modified CI configuration

## Why It Was Necessary
* Keeps repository aligned with source`,
		},
		{
			name: "valid PR body with all four sections accepted",
			input: `## What Changed
* Updated 4 GitHub workflow files

## Why It Was Necessary
* Synchronization requirement

## Testing Performed
* Validated YAML syntax

## Impact / Risk
* Low risk standard update`,
			expected: `## What Changed
* Updated 4 GitHub workflow files

## Why It Was Necessary
* Synchronization requirement

## Testing Performed
* Validated YAML syntax

## Impact / Risk
* Low risk standard update`,
		},
		{
			name: "PR body with leading whitespace trimmed and accepted",
			input: `
## What Changed
* Updated files`,
			expected: `## What Changed
* Updated files`,
		},
		{
			name:     "case insensitive commit prefix rejection - uppercase SYNC",
			input:    "SYNC: update files\nmore content",
			expected: "",
		},
		{
			name:     "case insensitive commit prefix rejection - mixed case Sync",
			input:    "Sync(ci): update workflows\nmore content",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePRBody(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
