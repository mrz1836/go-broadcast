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
		{
			name: "empty-diff meta-commentary rejected",
			input: `## What Changed
* The diff provided is empty - no actual code changes are visible in the truncated content

## Why It Was Necessary
* Unable to determine necessity from empty diff content

## Testing Performed
* Cannot verify testing from empty diff

## Impact / Risk
* Risk Level: Unknown - diff content not visible for assessment`,
			expected: "",
		},
		{
			name: "breaking changes cannot be determined without diff rejected",
			input: `## What Changed
* Modified 12 configuration files

## Impact / Risk
* Breaking Changes: Cannot be determined without viewing actual diff content`,
			expected: "",
		},
		{
			name: "file-list based description with headers accepted",
			input: `## What Changed
* Updated 12 GitHub Actions workflow and configuration files in .github/

## Why It Was Necessary
* Keeps the target repository aligned with its source repository`,
			expected: `## What Changed
* Updated 12 GitHub Actions workflow and configuration files in .github/

## Why It Was Necessary
* Keeps the target repository aligned with its source repository`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePRBody(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

//nolint:gosmopolitan // intentional unicode test data
func TestValidatePRBody_Unicode(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldBeValid bool
	}{
		{
			name: "Japanese content with headers",
			input: `## What Changed
* 日本語ファイルを更新しました
* 設定ファイルを同期

## Why It Was Necessary
* ソースリポジトリとの同期`,
			shouldBeValid: true,
		},
		{
			name: "Chinese content with headers",
			input: `## What Changed
* 更新了中文文档
* 同步配置文件

## Why It Was Necessary
* 保持仓库同步`,
			shouldBeValid: true,
		},
		{
			name: "Cyrillic content with headers",
			input: `## What Changed
* Обновлены файлы конфигурации
* Синхронизированы рабочие процессы

## Why It Was Necessary
* Необходима синхронизация`,
			shouldBeValid: true,
		},
		{
			name: "Emoji in PR body headers",
			input: `## What Changed 🔄
* Updated workflow files
* Modified CI configuration 🚀

## Why It Was Necessary ✨
* Keeps repository aligned`,
			shouldBeValid: true,
		},
		{
			name: "Mixed unicode and ASCII with headers",
			input: `## What Changed
* Updated файл.txt and 文件.md
* Modified café.go settings

## Why It Was Necessary
* Keep sync with αβγ-repo`,
			shouldBeValid: true,
		},
		{
			name: "Accented characters throughout",
			input: `## What Changed
* Mise à jour des fichiers
* Configuração atualizada

## Why It Was Necessary
* Synchronization nécessaire`,
			shouldBeValid: true,
		},
		{
			name: "Arabic content with headers",
			input: `## What Changed
* تحديث ملفات التكوين
* مزامنة سير العمل

## Why It Was Necessary
* الحفاظ على التزامن`,
			shouldBeValid: true,
		},
		{
			name:          "Unicode commit message rejected",
			input:         "sync: 更新文件\n\n这是描述",
			shouldBeValid: false,
		},
		{
			name:          "Unicode content without headers rejected",
			input:         "日本語の説明\n複数行ですが\nヘッダーがありません",
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePRBody(tt.input)
			if tt.shouldBeValid {
				assert.NotEmpty(t, result, "expected valid PR body to be accepted")
				assert.Contains(t, result, "##", "valid PR body should contain headers")
			} else {
				assert.Empty(t, result, "expected invalid PR body to be rejected")
			}
		})
	}
}

func TestValidatePRBody_EmojiEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldBeValid bool
	}{
		{
			name: "Emoji-only bullet points",
			input: `## What Changed
* 🔧 Fixed configuration
* 🚀 Updated deployment
* 📝 Modified docs

## Why It Was Necessary
* 🔄 Sync requirement`,
			shouldBeValid: true,
		},
		{
			name: "Complex emoji sequences",
			input: `## What Changed
* Updated files 👨‍👩‍👧‍👦
* Modified 🏳️‍🌈 settings

## Why It Was Necessary
* Keep sync 🇺🇸`,
			shouldBeValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePRBody(tt.input)
			if tt.shouldBeValid {
				assert.NotEmpty(t, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}
