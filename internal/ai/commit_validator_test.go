package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateCommitMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Basic validation
		{
			name:  "already valid sync message",
			input: "sync: update README.md from source",
			want:  "sync: update README.md from source",
		},
		{
			name:  "sync with scope",
			input: "sync(docs): update documentation files",
			want:  "sync(docs): update documentation files",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},

		// Whitespace handling
		{
			name:  "leading whitespace",
			input: "   sync: update files",
			want:  "sync: update files",
		},
		{
			name:  "trailing whitespace",
			input: "sync: update files   ",
			want:  "sync: update files",
		},
		{
			name:  "leading and trailing whitespace",
			input: "  sync: update README.md  ",
			want:  "sync: update README.md",
		},

		// Multi-line handling
		{
			name:  "multi-line takes first line only",
			input: "sync: update files\n\nBody text here",
			want:  "sync: update files",
		},
		{
			name:  "multi-line with body",
			input: "sync: short subject\n\nThis is a detailed body explaining the changes.",
			want:  "sync: short subject",
		},
		{
			name:  "first line with trailing newline",
			input: "sync: update files\n",
			want:  "sync: update files",
		},

		// Prefix conversion
		{
			name:  "adds sync prefix if missing",
			input: "update README.md from source",
			want:  "sync: update README.md from source",
		},
		{
			name:  "converts chore to sync",
			input: "chore: update README.md",
			want:  "sync: update README.md",
		},
		{
			name:  "converts chore(sync) to sync",
			input: "chore(sync): update files",
			want:  "sync: update files",
		},
		{
			name:  "converts feat to sync",
			input: "feat: add new feature",
			want:  "sync: add new feature",
		},
		{
			name:  "converts fix to sync",
			input: "fix: correct typo",
			want:  "sync: correct typo",
		},
		{
			name:  "converts docs to sync",
			input: "docs: update documentation",
			want:  "sync: update documentation",
		},
		{
			name:  "converts refactor to sync",
			input: "refactor: improve code structure",
			want:  "sync: improve code structure",
		},
		{
			name:  "converts test to sync",
			input: "test: add unit tests",
			want:  "sync: add unit tests",
		},
		{
			name:  "converts build to sync",
			input: "build: update dependencies",
			want:  "sync: update dependencies",
		},
		{
			name:  "converts ci to sync",
			input: "ci: update workflow",
			want:  "sync: update workflow",
		},

		// Trailing period removal
		{
			name:  "removes trailing period",
			input: "sync: update files.",
			want:  "sync: update files",
		},
		{
			name:  "removes trailing period after conversion",
			input: "chore: update files.",
			want:  "sync: update files",
		},

		// Markdown formatting removal
		{
			name:  "removes backtick wrapping",
			input: "`sync: update files`",
			want:  "sync: update files",
		},
		{
			name:  "removes code block markers",
			input: "```sync: update files```",
			want:  "sync: update files",
		},
		{
			name:  "removes leading backtick only",
			input: "`sync: update files",
			want:  "sync: update files",
		},
		{
			name:  "removes trailing backtick only",
			input: "sync: update files`",
			want:  "sync: update files",
		},

		// Length truncation
		{
			name:  "truncates long messages at word boundary",
			input: "sync: this is a very long commit message that exceeds the seventy two character limit and should be truncated at a word boundary",
			want:  "sync: this is a very long commit message that exceeds the seventy...",
		},
		{
			name:  "exactly 72 chars unchanged",
			input: "sync: " + strings.Repeat("a", 66), // 6 + 66 = 72
			want:  "sync: " + strings.Repeat("a", 66),
		},

		// Combined scenarios
		{
			name:  "multi-line with period and whitespace",
			input: "  chore: update files.\n\nBody text  ",
			want:  "sync: update files",
		},
		{
			name:  "backticks with chore prefix",
			input: "`chore: update README.md`",
			want:  "sync: update README.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCommitMessage(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateCommitMessage_LengthLimit(t *testing.T) {
	// Test that messages are properly truncated to 72 characters
	longMsg := "sync: " + strings.Repeat("word ", 20) // Much longer than 72

	result := ValidateCommitMessage(longMsg)

	assert.LessOrEqual(t, len(result), maxCommitMessageLength,
		"result should not exceed %d chars", maxCommitMessageLength)
	assert.True(t, strings.HasSuffix(result, "..."),
		"truncated result should end with ...")
}

func TestRemoveMarkdownFormatting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no formatting", "plain text", "plain text"},
		{"single backticks", "`code`", "code"},
		{"triple backticks", "```code```", "code"},
		{"only leading backtick", "`code", "code"},
		{"only trailing backtick", "code`", "code"},
		{"backticks at start end removed", "`spaced`", "spaced"},
		{"empty string", "", ""},
		{"only backticks", "```", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeMarkdownFormatting(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnsureSyncPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Already correct
		{"sync: prefix unchanged", "sync: message", "sync: message"},
		{"sync(scope): prefix unchanged", "sync(docs): message", "sync(docs): message"},

		// Conversions
		{"converts chore(sync):", "chore(sync): message", "sync: message"},
		{"converts chore:", "chore: message", "sync: message"},
		{"converts feat:", "feat: message", "sync: message"},
		{"converts fix:", "fix: message", "sync: message"},
		{"converts docs:", "docs: message", "sync: message"},
		{"converts refactor:", "refactor: message", "sync: message"},
		{"converts test:", "test: message", "sync: message"},
		{"converts build:", "build: message", "sync: message"},
		{"converts ci:", "ci: message", "sync: message"},

		// No prefix
		{"adds sync: when no prefix", "update files", "sync: update files"},
		{"adds sync: to plain text", "just a message", "sync: just a message"},

		// Edge cases
		{"chore without colon not converted", "chore update files", "sync: chore update files"},
		{"feat without space not converted", "feat:message", "sync: feat:message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureSyncPrefix(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncateAtWordBoundary(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length unchanged",
			input:  "exact",
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "truncates at space",
			input:  "hello world test",
			maxLen: 12,
			want:   "hello world",
		},
		{
			name:   "truncates at best space",
			input:  "this is a long sentence",
			maxLen: 15,
			want:   "this is a long",
		},
		{
			name:   "hard truncate when no good space",
			input:  "verylongwordwithoutspaces",
			maxLen: 10,
			want:   "verylongwo",
		},
		{
			name:   "hard truncate when space too early",
			input:  "a verylongword",
			maxLen: 10,
			want:   "a verylong", // space at index 1 is before maxLen/2 (5), so hard truncate to 10 chars
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "single word longer than max",
			input:  "supercalifragilisticexpialidocious",
			maxLen: 10,
			want:   "supercalif",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateAtWordBoundary(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, len(got), tt.maxLen)
		})
	}
}

func TestValidateCommitMessage_Idempotent(t *testing.T) {
	// Validation should be idempotent - running twice should give same result.
	// We exclude long messages that get truncated with "..." since
	// the trailing period removal would affect the result on second pass.
	tests := []struct {
		name  string
		input string
	}{
		{"sync prefix", "sync: update files"},
		{"chore prefix", "chore: update files"},
		{"feat with period", "  feat: add feature.  "},
		{"backtick wrap", "`sync: message`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := ValidateCommitMessage(tt.input)
			second := ValidateCommitMessage(first)
			assert.Equal(t, first, second, "validation should be idempotent")
		})
	}
}

//nolint:gosmopolitan // intentional unicode test data
func TestValidateCommitMessage_Unicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Japanese characters in message",
			input:    "sync: 日本語ファイルを更新",
			expected: "sync: 日本語ファイルを更新",
		},
		{
			name:     "Chinese characters in message",
			input:    "sync: 更新中文文件",
			expected: "sync: 更新中文文件",
		},
		{
			name:     "Korean characters in message",
			input:    "sync: 한국어 파일 업데이트",
			expected: "sync: 한국어 파일 업데이트",
		},
		{
			name:     "Arabic characters in message",
			input:    "sync: تحديث الملفات العربية",
			expected: "sync: تحديث الملفات العربية",
		},
		{
			name:     "Greek characters in message",
			input:    "sync: ενημέρωση ελληνικών αρχείων",
			expected: "sync: ενημέρωση ελληνικών αρχείων",
		},
		{
			name:     "Cyrillic characters in message",
			input:    "sync: обновление файлов",
			expected: "sync: обновление файлов",
		},
		{
			name:     "emoji in message preserved",
			input:    "sync: update README 🎉",
			expected: "sync: update README 🎉",
		},
		{
			name:     "multiple emojis in message",
			input:    "sync: 🚀 update CI workflows 🔧",
			expected: "sync: 🚀 update CI workflows 🔧",
		},
		{
			name:     "emoji-only file reference",
			input:    "sync: update 📄 files",
			expected: "sync: update 📄 files",
		},
		{
			name:     "mixed unicode and ASCII",
			input:    "sync: update файл.txt and 文件.md",
			expected: "sync: update файл.txt and 文件.md",
		},
		{
			name:     "unicode with chore prefix converted",
			input:    "chore: 更新配置文件",
			expected: "sync: 更新配置文件",
		},
		{
			name:     "unicode message needs sync prefix",
			input:    "日本語のメッセージ",
			expected: "sync: 日本語のメッセージ",
		},
		{
			name:     "accented characters",
			input:    "sync: update café.txt and naïve.md",
			expected: "sync: update café.txt and naïve.md",
		},
		{
			name:     "mathematical symbols",
			input:    "sync: update formula α + β = γ",
			expected: "sync: update formula α + β = γ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCommitMessage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

//nolint:gosmopolitan // intentional unicode test data
func TestValidateCommitMessage_UnicodeIdempotent(t *testing.T) {
	// Unicode messages should also be idempotent
	messages := []string{
		"sync: 日本語ファイルを更新",
		"sync: обновление файлов 🎉",
		"chore: 更新配置文件",
		"sync: update café.txt",
	}

	for _, msg := range messages {
		first := ValidateCommitMessage(msg)
		second := ValidateCommitMessage(first)
		assert.Equal(t, first, second, "validation should be idempotent for: %s", msg)
	}
}
