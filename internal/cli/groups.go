// Package cli provides command-line interface functionality for go-broadcast.
package cli

import (
	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// FilterConfigByGroups filters the configuration to only include specified groups.
// Matches groups by Name or ID. Skip patterns take precedence over include patterns.
// If no filters are specified, returns the original config unchanged.
func FilterConfigByGroups(cfg *config.Config, groupFilter, skipGroups []string) *config.Config {
	// If no filters specified, return original config
	if len(groupFilter) == 0 && len(skipGroups) == 0 {
		return cfg
	}

	// Create a copy of the config with filtered groups
	filteredCfg := &config.Config{
		Version:        cfg.Version,
		Name:           cfg.Name,
		ID:             cfg.ID,
		Groups:         []config.Group{},
		FileLists:      cfg.FileLists,
		DirectoryLists: cfg.DirectoryLists,
	}

	for _, group := range cfg.Groups {
		// Check if group should be skipped
		shouldSkip := false
		for _, skipPattern := range skipGroups {
			if group.Name == skipPattern || group.ID == skipPattern {
				logrus.WithFields(logrus.Fields{
					"group_name": group.Name,
					"group_id":   group.ID,
				}).Debug("Group matches skip pattern, excluding")
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			continue
		}

		// Check if group matches filter (if filter is specified)
		if len(groupFilter) > 0 {
			matchesFilter := false
			for _, filterPattern := range groupFilter {
				if group.Name == filterPattern || group.ID == filterPattern {
					matchesFilter = true
					break
				}
			}
			if !matchesFilter {
				logrus.WithFields(logrus.Fields{
					"group_name": group.Name,
					"group_id":   group.ID,
				}).Debug("Group doesn't match filter pattern, excluding")
				continue
			}
		}

		filteredCfg.Groups = append(filteredCfg.Groups, group)
	}

	return filteredCfg
}
