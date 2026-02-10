package cli

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

//nolint:gochecknoglobals // Cobra commands and flags are designed to be global variables
var (
	dbDiffYAML   string
	dbDiffDetail bool
	dbDiffCmd    = &cobra.Command{
		Use:   "diff",
		Short: "Show differences between database and YAML",
		Long: `Compare the database configuration with a YAML file and show differences.

This command is useful for:
• Verifying round-trip import/export fidelity
• Checking what would change before importing
• Auditing configuration drift

The diff shows:
• Structural differences (counts of groups, targets, lists)
• Missing or extra entities
• Configuration value changes

Examples:
  # Compare database with sync.yaml
  go-broadcast db diff

  # Compare with specific YAML file
  go-broadcast db diff --yaml my-config.yaml

  # Show detailed field-level differences
  go-broadcast db diff --detail

  # Compare custom database
  go-broadcast db diff --yaml sync.yaml --db-path /tmp/test.db`,
		RunE: runDBDiff,
	}
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	dbDiffCmd.Flags().StringVar(&dbDiffYAML, "yaml", "sync.yaml", "Path to YAML configuration file")
	dbDiffCmd.Flags().BoolVar(&dbDiffDetail, "detail", false, "Show detailed field-level differences")
}

// runDBDiff executes the database diff command
func runDBDiff(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	path := getDBPath()

	// Load YAML configuration
	output.Info(fmt.Sprintf("Loading YAML: %s", dbDiffYAML))
	yamlCfg, err := config.Load(dbDiffYAML)
	if err != nil {
		return fmt.Errorf("failed to load YAML: %w", err)
	}

	// Open database
	output.Info(fmt.Sprintf("Loading database: %s", path))
	database, err := db.Open(db.OpenOptions{
		Path:     path,
		LogLevel: logger.Silent,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Export database to config
	converter := db.NewConverter(database.DB())

	// Find config with matching ID (or first if none match)
	var configExternalID string
	if yamlCfg.ID != "" {
		var dbConfig db.Config
		result := database.DB().Where("external_id = ?", yamlCfg.ID).First(&dbConfig)
		if result.Error == nil {
			configExternalID = dbConfig.ExternalID
		}
	}

	if configExternalID == "" {
		// Use first config in database
		var dbConfig db.Config
		if err := database.DB().First(&dbConfig).Error; err != nil {
			return fmt.Errorf("no configuration found in database: %w", err)
		}
		configExternalID = dbConfig.ExternalID
	}

	dbCfg, err := converter.ExportConfig(ctx, configExternalID)
	if err != nil {
		return fmt.Errorf("failed to export database: %w", err)
	}

	// Compare configurations
	output.Info("\nComparing configurations...")
	diffs := compareConfigs(yamlCfg, dbCfg, dbDiffDetail)

	if len(diffs) == 0 {
		output.Success("✓ No differences found - configurations match!")
		return nil
	}

	// Show differences
	output.Warn(fmt.Sprintf("Found %d difference(s):", len(diffs)))
	for _, diff := range diffs {
		fmt.Println(diff)
	}

	return nil
}

// Diff represents a single difference between configurations
type Diff struct {
	Path     string // Path to the differing field (e.g., "groups[0].name")
	YAMLVal  string // Value in YAML
	DBVal    string // Value in database
	Category string // Category: "count", "missing", "extra", "value"
}

// compareConfigs compares two config.Config instances and returns differences
func compareConfigs(yamlCfg, dbCfg *config.Config, detail bool) []string {
	var diffs []string

	// Compare basic fields
	if yamlCfg.Name != dbCfg.Name {
		diffs = append(diffs, fmt.Sprintf("  • Config name differs: YAML=%q, DB=%q", yamlCfg.Name, dbCfg.Name))
	}
	if yamlCfg.Version != dbCfg.Version {
		diffs = append(diffs, fmt.Sprintf("  • Config version differs: YAML=%d, DB=%d", yamlCfg.Version, dbCfg.Version))
	}

	// Compare counts
	if len(yamlCfg.Groups) != len(dbCfg.Groups) {
		diffs = append(diffs, fmt.Sprintf("  • Group count differs: YAML=%d, DB=%d", len(yamlCfg.Groups), len(dbCfg.Groups)))
	}
	if len(yamlCfg.FileLists) != len(dbCfg.FileLists) {
		diffs = append(diffs, fmt.Sprintf("  • FileList count differs: YAML=%d, DB=%d", len(yamlCfg.FileLists), len(dbCfg.FileLists)))
	}
	if len(yamlCfg.DirectoryLists) != len(dbCfg.DirectoryLists) {
		diffs = append(diffs, fmt.Sprintf("  • DirectoryList count differs: YAML=%d, DB=%d", len(yamlCfg.DirectoryLists), len(dbCfg.DirectoryLists)))
	}

	// Compare groups
	yamlGroupMap := make(map[string]config.Group)
	for _, g := range yamlCfg.Groups {
		yamlGroupMap[g.ID] = g
	}

	dbGroupMap := make(map[string]config.Group)
	for _, g := range dbCfg.Groups {
		dbGroupMap[g.ID] = g
	}

	// Find missing/extra groups
	for id := range yamlGroupMap {
		if _, exists := dbGroupMap[id]; !exists {
			diffs = append(diffs, fmt.Sprintf("  • Group missing in DB: %q", id))
		}
	}
	for id := range dbGroupMap {
		if _, exists := yamlGroupMap[id]; !exists {
			diffs = append(diffs, fmt.Sprintf("  • Extra group in DB: %q", id))
		}
	}

	// Compare matching groups in detail if requested
	if detail {
		for id, yamlGroup := range yamlGroupMap {
			if dbGroup, exists := dbGroupMap[id]; exists {
				groupDiffs := compareGroups(id, yamlGroup, dbGroup)
				diffs = append(diffs, groupDiffs...)
			}
		}
	}

	// Compare file lists
	yamlFileListMap := make(map[string]config.FileList)
	for _, fl := range yamlCfg.FileLists {
		yamlFileListMap[fl.ID] = fl
	}

	dbFileListMap := make(map[string]config.FileList)
	for _, fl := range dbCfg.FileLists {
		dbFileListMap[fl.ID] = fl
	}

	for id := range yamlFileListMap {
		if _, exists := dbFileListMap[id]; !exists {
			diffs = append(diffs, fmt.Sprintf("  • FileList missing in DB: %q", id))
		}
	}
	for id := range dbFileListMap {
		if _, exists := yamlFileListMap[id]; !exists {
			diffs = append(diffs, fmt.Sprintf("  • Extra FileList in DB: %q", id))
		}
	}

	return diffs
}

// compareGroups compares two groups and returns differences
func compareGroups(id string, yamlGroup, dbGroup config.Group) []string {
	var diffs []string
	prefix := fmt.Sprintf("  • Group[%s]", id)

	if yamlGroup.Name != dbGroup.Name {
		diffs = append(diffs, fmt.Sprintf("%s.name: YAML=%q, DB=%q", prefix, yamlGroup.Name, dbGroup.Name))
	}

	if yamlGroup.Description != dbGroup.Description {
		diffs = append(diffs, fmt.Sprintf("%s.description: YAML=%q, DB=%q", prefix, yamlGroup.Description, dbGroup.Description))
	}

	if yamlGroup.Priority != dbGroup.Priority {
		diffs = append(diffs, fmt.Sprintf("%s.priority: YAML=%d, DB=%d", prefix, yamlGroup.Priority, dbGroup.Priority))
	}

	// Compare enabled (handle nil)
	yamlEnabled := true
	if yamlGroup.Enabled != nil {
		yamlEnabled = *yamlGroup.Enabled
	}
	dbEnabled := true
	if dbGroup.Enabled != nil {
		dbEnabled = *dbGroup.Enabled
	}
	if yamlEnabled != dbEnabled {
		diffs = append(diffs, fmt.Sprintf("%s.enabled: YAML=%v, DB=%v", prefix, yamlEnabled, dbEnabled))
	}

	// Compare source
	if yamlGroup.Source.Repo != dbGroup.Source.Repo {
		diffs = append(diffs, fmt.Sprintf("%s.source.repo: YAML=%q, DB=%q", prefix, yamlGroup.Source.Repo, dbGroup.Source.Repo))
	}

	// Compare target counts
	if len(yamlGroup.Targets) != len(dbGroup.Targets) {
		diffs = append(diffs, fmt.Sprintf("%s target count: YAML=%d, DB=%d", prefix, len(yamlGroup.Targets), len(dbGroup.Targets)))
	}

	// Compare dependencies
	if !stringSlicesEqual(yamlGroup.DependsOn, dbGroup.DependsOn) {
		diffs = append(diffs, fmt.Sprintf("%s.depends_on: YAML=%v, DB=%v", prefix, yamlGroup.DependsOn, dbGroup.DependsOn))
	}

	return diffs
}

// stringSlicesEqual compares two string slices for equality
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// formatValue formats a value for display
func formatValue(v interface{}) string {
	if v == nil {
		return "<nil>"
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String:
		return fmt.Sprintf("%q", v)
	case reflect.Slice, reflect.Array:
		var parts []string
		for i := 0; i < val.Len(); i++ {
			parts = append(parts, fmt.Sprintf("%v", val.Index(i)))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}
