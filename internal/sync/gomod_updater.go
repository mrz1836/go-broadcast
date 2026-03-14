package sync

import (
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// GoModUpdater handles updating go.mod dependency versions
type GoModUpdater struct {
	logger *logrus.Logger
}

// NewGoModUpdater creates a new GoModUpdater
func NewGoModUpdater(logger *logrus.Logger) *GoModUpdater {
	return &GoModUpdater{
		logger: logger,
	}
}

// UpdateDependency updates a specific dependency version in go.mod content.
// It handles both single-line require statements and require blocks.
//
// Returns:
//   - Updated content
//   - Whether any change was made
//   - Error if parsing fails
func (u *GoModUpdater) UpdateDependency(content []byte, modulePath, newVersion string) ([]byte, bool, error) {
	if modulePath == "" || newVersion == "" {
		return content, false, nil
	}

	contentStr := string(content)
	modified := false

	// Ensure version has 'v' prefix
	if !strings.HasPrefix(newVersion, "v") {
		newVersion = "v" + newVersion
	}

	// Pattern for single-line require: require github.com/foo/bar v1.0.0
	// Also handles comments at end of line
	singleRequirePattern := regexp.MustCompile(
		`(?m)^(\s*require\s+)` + regexp.QuoteMeta(modulePath) + `(\s+)v[\d]+\.[\d]+\.[\d]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?(\s*(?://.*)?)?$`,
	)

	// Pattern for require block entry: \tgithub.com/foo/bar v1.0.0
	// Also handles indirect comment
	blockRequirePattern := regexp.MustCompile(
		`(?m)^(\s+)` + regexp.QuoteMeta(modulePath) + `(\s+)v[\d]+\.[\d]+\.[\d]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?(\s*(?://.*)?)?$`,
	)

	// Try to update single-line require first
	if singleRequirePattern.MatchString(contentStr) {
		contentStr = singleRequirePattern.ReplaceAllStringFunc(contentStr, func(match string) string {
			modified = true
			// Extract prefix and suffix
			submatches := singleRequirePattern.FindStringSubmatch(match)
			if len(submatches) >= 6 {
				prefix := submatches[1]  // "require "
				spacing := submatches[2] // whitespace before version
				suffix := submatches[5]  // trailing comment if any
				return prefix + modulePath + spacing + newVersion + suffix
			}
			return match
		})
	}

	// Try to update in require block
	if blockRequirePattern.MatchString(contentStr) {
		contentStr = blockRequirePattern.ReplaceAllStringFunc(contentStr, func(match string) string {
			modified = true
			// Extract prefix and suffix
			submatches := blockRequirePattern.FindStringSubmatch(match)
			if len(submatches) >= 6 {
				indent := submatches[1]  // leading whitespace
				spacing := submatches[2] // whitespace before version
				suffix := submatches[5]  // trailing comment if any
				return indent + modulePath + spacing + newVersion + suffix
			}
			return match
		})
	}

	if modified && u.logger != nil {
		u.logger.WithFields(logrus.Fields{
			"module":      modulePath,
			"new_version": newVersion,
		}).Debug("Updated go.mod dependency version")
	}

	return []byte(contentStr), modified, nil
}

// AddDependency adds a new dependency to go.mod content if it doesn't exist.
// It adds the require statement after the go directive or at the end of an existing require block.
//
// Returns:
//   - Updated content
//   - Whether any change was made
//   - Error if parsing fails
func (u *GoModUpdater) AddDependency(content []byte, modulePath, version string) ([]byte, bool, error) {
	if modulePath == "" || version == "" {
		return content, false, nil
	}

	contentStr := string(content)

	// Ensure version has 'v' prefix
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// Check if dependency already exists (in require block or single-line require)
	// Matches both:
	//   require github.com/foo/bar v1.0.0  (single-line)
	//   \tgithub.com/foo/bar v1.0.0        (in require block)
	existsInBlockPattern := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(modulePath) + `\s+v`)
	existsSingleLinePattern := regexp.MustCompile(`(?m)^require\s+` + regexp.QuoteMeta(modulePath) + `\s+v`)
	if existsInBlockPattern.MatchString(contentStr) || existsSingleLinePattern.MatchString(contentStr) {
		// Dependency exists, use UpdateDependency instead
		return u.UpdateDependency(content, modulePath, version)
	}

	// Find require block to add to
	requireBlockPattern := regexp.MustCompile(`(?m)^require\s*\(\s*\n((?:.*\n)*?)\)`)
	if matches := requireBlockPattern.FindStringIndex(contentStr); matches != nil {
		// Insert before the closing parenthesis
		insertPos := matches[1] - 1 // Position before ')'
		newLine := "\t" + modulePath + " " + version + "\n"
		contentStr = contentStr[:insertPos] + newLine + contentStr[insertPos:]

		if u.logger != nil {
			u.logger.WithFields(logrus.Fields{
				"module":  modulePath,
				"version": version,
			}).Debug("Added new go.mod dependency")
		}

		return []byte(contentStr), true, nil
	}

	// No require block, add a new require statement after go directive
	goDirectivePattern := regexp.MustCompile(`(?m)^go\s+[\d.]+\s*\n`)
	if matches := goDirectivePattern.FindStringIndex(contentStr); matches != nil {
		insertPos := matches[1]
		newLine := "\nrequire " + modulePath + " " + version + "\n"
		contentStr = contentStr[:insertPos] + newLine + contentStr[insertPos:]

		if u.logger != nil {
			u.logger.WithFields(logrus.Fields{
				"module":  modulePath,
				"version": version,
			}).Debug("Added new go.mod dependency (single require)")
		}

		return []byte(contentStr), true, nil
	}

	// Fallback: append to end of file
	contentStr += "\nrequire " + modulePath + " " + version + "\n"

	if u.logger != nil {
		u.logger.WithFields(logrus.Fields{
			"module":  modulePath,
			"version": version,
		}).Debug("Added new go.mod dependency (appended)")
	}

	return []byte(contentStr), true, nil
}
