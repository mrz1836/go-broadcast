package sync

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestFilterGroupsByOptions tests the filterGroupsByOptions function
func TestFilterGroupsByOptions(t *testing.T) {
	// Create test groups
	groups := []config.Group{
		{
			Name: "group1",
			ID:   "g1",
		},
		{
			Name: "group2",
			ID:   "g2",
		},
		{
			Name: "special-group",
			ID:   "special",
		},
		{
			Name: "test-group",
			ID:   "test",
		},
	}

	t.Run("nil options returns all groups", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		filtered := o.filterGroupsByOptions(groups, nil)
		assert.Equal(t, groups, filtered)
		assert.Len(t, filtered, 4)
	})

	t.Run("empty filters returns all groups", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{},
			SkipGroups:  []string{},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Equal(t, groups, filtered)
		assert.Len(t, filtered, 4)
	})

	t.Run("filter by group name", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"group1", "group2"},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "group1", filtered[0].Name)
		assert.Equal(t, "group2", filtered[1].Name)
	})

	t.Run("filter by group ID", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"g1", "special"},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "g1", filtered[0].ID)
		assert.Equal(t, "special", filtered[1].ID)
	})

	t.Run("filter by mixed name and ID", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"group1", "special"}, // Name and ID mixed
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "group1", filtered[0].Name)
		assert.Equal(t, "special", filtered[1].ID)
	})

	t.Run("skip groups by name", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			SkipGroups: []string{"group1", "test-group"},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "group2", filtered[0].Name)
		assert.Equal(t, "special-group", filtered[1].Name)
	})

	t.Run("skip groups by ID", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			SkipGroups: []string{"g1", "test"},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "g2", filtered[0].ID)
		assert.Equal(t, "special", filtered[1].ID)
	})

	t.Run("filter and skip combined", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"group1", "group2", "special-group"},
			SkipGroups:  []string{"group2"},
		}

		// Should filter to group1, group2, special-group, then skip group2
		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "group1", filtered[0].Name)
		assert.Equal(t, "special-group", filtered[1].Name)
	})

	t.Run("filter with non-existent groups", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"nonexistent", "group1"},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "group1", filtered[0].Name)
	})

	t.Run("skip with non-existent groups", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			SkipGroups: []string{"nonexistent", "group1"},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 3)
		// group1 should be skipped
		for _, g := range filtered {
			assert.NotEqual(t, "group1", g.Name)
		}
	})

	t.Run("empty groups list", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"group1"},
		}

		filtered := o.filterGroupsByOptions([]config.Group{}, options)
		assert.Empty(t, filtered)
	})

	t.Run("filter all groups out", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"nonexistent"},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Empty(t, filtered)
	})

	t.Run("skip all groups", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			SkipGroups: []string{"group1", "group2", "special-group", "test-group"},
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Empty(t, filtered)
	})

	t.Run("case sensitivity", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"GROUP1"}, // Different case
		}

		// Test whether filtering is case-sensitive (it should be)
		filtered := o.filterGroupsByOptions(groups, options)
		assert.Empty(t, filtered, "Filtering should be case-sensitive")
	})

	t.Run("partial name matching", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"group"}, // Partial name
		}

		// Should not match partial names
		filtered := o.filterGroupsByOptions(groups, options)
		assert.Empty(t, filtered, "Should not match partial names")
	})

	t.Run("filter with duplicate entries", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			GroupFilter: []string{"group1", "group1", "g1"}, // Duplicates
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "group1", filtered[0].Name)
	})

	t.Run("skip with duplicate entries", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		options := &Options{
			SkipGroups: []string{"group1", "group1", "g1"}, // Duplicates
		}

		filtered := o.filterGroupsByOptions(groups, options)
		assert.Len(t, filtered, 3)
		// group1 should still be skipped only once
		for _, g := range filtered {
			assert.NotEqual(t, "group1", g.Name)
		}
	})

	t.Run("complex filtering scenario", func(t *testing.T) {
		o := &GroupOrchestrator{
			logger: logrus.New(),
		}

		// Create a more complex set of groups
		complexGroups := []config.Group{
			{Name: "production", ID: "prod"},
			{Name: "staging", ID: "stage"},
			{Name: "development", ID: "dev"},
			{Name: "testing", ID: "test"},
			{Name: "experimental", ID: "exp"},
		}

		options := &Options{
			GroupFilter: []string{"prod", "staging", "dev", "test"}, // Mix of IDs and names
			SkipGroups:  []string{"development", "test"},            // Skip some of the filtered
		}

		filtered := o.filterGroupsByOptions(complexGroups, options)
		assert.Len(t, filtered, 2)
		assert.Equal(t, "production", filtered[0].Name)
		assert.Equal(t, "staging", filtered[1].Name)
	})
}
