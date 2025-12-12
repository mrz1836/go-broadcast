package ai

import (
	"os"
	"path/filepath"
)

const (
	defaultPRGuidelinesPath = ".github/tech-conventions/pull-request-guidelines.md"
)

// defaultPRGuidelinesTemplate is the fallback when guidelines file cannot be read.
// Based on the actual tech-conventions/pull-request-guidelines.md structure.
const defaultPRGuidelinesTemplate = `## PR Description Structure

Every PR must include the following four sections:

### 1. What Changed
A clear, bullet-pointed or paragraph-level summary of the technical changes.

### 2. Why It Was Necessary
Context or motivation behind the change. Reference related issues if applicable.

### 3. Testing Performed
Document:
- Test suites run
- Edge cases covered
- Manual steps that were taken (if any)

### 4. Impact / Risk
Call out:
- Breaking changes
- Regression risk
- Performance implications`

// LoadPRGuidelines loads PR guidelines from the repository.
// Returns embedded fallback if file doesn't exist or can't be read.
// Never returns an error - always provides usable guidelines.
func LoadPRGuidelines(repoPath string) string {
	guidelinesPath := filepath.Join(repoPath, defaultPRGuidelinesPath)

	content, err := os.ReadFile(guidelinesPath) //nolint:gosec // Path is constructed from repoPath parameter, not user input
	if err != nil {
		// Return fallback if file doesn't exist or can't be read
		return defaultPRGuidelinesTemplate
	}

	if len(content) == 0 {
		return defaultPRGuidelinesTemplate
	}

	return string(content)
}
