package cli

import "github.com/mrz1836/go-broadcast/internal/config"

// MVPPreset returns a minimal generic preset suitable for any new repository.
//
// It enables issues, squash merging with branch deletion, and a small set of
// common labels. No rulesets are attached so the preset works on free GitHub
// plans without requiring branch protection.
func MVPPreset() config.SettingsPreset {
	return config.SettingsPreset{
		ID:                       "mvp",
		Name:                     "MVP",
		Description:              "Minimal generic preset for new repositories",
		HasIssues:                true,
		HasWiki:                  false,
		HasProjects:              false,
		HasDiscussions:           false,
		AllowSquashMerge:         true,
		AllowMergeCommit:         false,
		AllowRebaseMerge:         false,
		DeleteBranchOnMerge:      true,
		AllowAutoMerge:           true,
		AllowUpdateBranch:        true,
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "COMMIT_MESSAGES",
		Labels: []config.LabelSpec{
			{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
			{Name: "enhancement", Color: "a2eeef", Description: "New feature or request"},
			{Name: "documentation", Color: "0075ca", Description: "Documentation improvements"},
			{Name: "good first issue", Color: "7057ff", Description: "Good for newcomers"},
			{Name: "help wanted", Color: "008672", Description: "Extra attention needed"},
		},
	}
}

// GoLibPreset returns a preset tuned for an open-source Go library.
//
// It layers a typical Go OSS label set and a single branch-protection ruleset
// (default branch requires a pull request, deletion blocked) onto the MVP
// settings. Suitable for any community Go module.
func GoLibPreset() config.SettingsPreset {
	return config.SettingsPreset{
		ID:                       "go-lib",
		Name:                     "Go Library",
		Description:              "Standard preset for open-source Go libraries",
		HasIssues:                true,
		HasWiki:                  false,
		HasProjects:              false,
		HasDiscussions:           true,
		AllowSquashMerge:         true,
		AllowMergeCommit:         false,
		AllowRebaseMerge:         false,
		DeleteBranchOnMerge:      true,
		AllowAutoMerge:           true,
		AllowUpdateBranch:        true,
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "COMMIT_MESSAGES",
		Rulesets: []config.RulesetConfig{
			{
				Name:        "branch-protection",
				Target:      "branch",
				Enforcement: "active",
				Include:     []string{"~DEFAULT_BRANCH"},
				Rules:       []string{"deletion", "pull_request"},
			},
			{
				Name:        "tag-protection",
				Target:      "tag",
				Enforcement: "active",
				Include:     []string{"~ALL"},
				Rules:       []string{"deletion", "update"},
			},
		},
		Labels: []config.LabelSpec{
			{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
			{Name: "enhancement", Color: "a2eeef", Description: "New feature or request"},
			{Name: "documentation", Color: "0075ca", Description: "Documentation improvements"},
			{Name: "good first issue", Color: "7057ff", Description: "Good for newcomers"},
			{Name: "help wanted", Color: "008672", Description: "Extra attention needed"},
			{Name: "dependencies", Color: "0366d6", Description: "Pull requests that update a dependency file"},
			{Name: "go", Color: "00ADD8", Description: "Go-related changes"},
			{Name: "tests", Color: "fbca04", Description: "Test additions or fixes"},
			{Name: "ci", Color: "5319e7", Description: "Continuous integration changes"},
			{Name: "breaking-change", Color: "b60205", Description: "Backwards-incompatible change"},
		},
	}
}

// BundledPresets returns the set of generic presets shipped with go-broadcast.
//
// The slice order is the canonical seeding order: callers can rely on
// BundledPresets()[0] being the MVP preset and [1] being the Go library preset.
func BundledPresets() []config.SettingsPreset {
	return []config.SettingsPreset{
		MVPPreset(),
		GoLibPreset(),
	}
}
