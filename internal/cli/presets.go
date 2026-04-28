package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// presetSourceDB indicates a preset stored in the local database.
const presetSourceDB = "db"

// presetSourceConfigPrefix is the prefix used for presets defined in a config file.
const presetSourceConfigPrefix = "config-file:"

// presetSourceBundled indicates a preset compiled into the binary.
const presetSourceBundled = "bundled-default"

// presetListEntry is a single row in `presets list` output.
type presetListEntry struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source"`
}

// presetSeedSummary is the JSON-shaped result of `presets seed`.
type presetSeedSummary struct {
	BundledSeeded int    `json:"bundled_seeded"`
	LoadedFromDir int    `json:"loaded_from_dir"`
	Overrides     int    `json:"overrides"`
	FromDir       string `json:"from_dir,omitempty"`
}

// newPresetsCmd creates the top-level "presets" command with discovery subcommands.
func newPresetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "presets",
		Short: "Discover and seed repository settings presets",
		Long: `List and seed repository settings presets across all resolution sources.

Resolution chain (highest priority first):
  1. Database
  2. Config file (--config sync.yaml)
  3. Bundled defaults shipped with the binary (mvp, go-lib)

The reserved id "default" always resolves to the hardcoded fallback preset.`,
	}

	cmd.AddCommand(newPresetsListCmd())
	cmd.AddCommand(newPresetsSeedCmd())

	return cmd
}

func newPresetsListCmd() *cobra.Command {
	var (
		jsonOutput bool
		showSource bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List presets across all sources",
		Long: `List every preset known to go-broadcast across the database, the configuration
file, and the bundled defaults.

With --show-source, each entry is annotated with its origin (db, config-file:<path>,
or bundled-default). When the same id appears in multiple sources, every occurrence
is listed so the resolution chain is fully visible.`,
		Example: `  # List preset ids only
  go-broadcast presets list

  # Annotate each entry with the resolution source
  go-broadcast presets list --show-source

  # JSON output for scripting
  go-broadcast presets list --json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPresetsList(cmd.Context(), showSource, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&showSource, "show-source", false, "Annotate each preset with its resolution source")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

// runPresetsList aggregates preset entries across DB, config file, and bundled defaults.
func runPresetsList(ctx context.Context, showSource, jsonOutput bool) error {
	entries := make([]presetListEntry, 0, 8)

	// Database (best-effort: an unopenable database is treated as empty)
	if database, err := openDatabase(); err == nil {
		presetRepo := db.NewSettingsPresetRepository(database.DB())
		dbPresets, listErr := presetRepo.List(ctx)
		_ = database.Close()
		if listErr != nil {
			return printErrorResponse("preset", "listed", listErr.Error(), "", jsonOutput)
		}
		for _, p := range dbPresets {
			entries = append(entries, presetListEntry{
				ID:     p.ExternalID,
				Name:   p.Name,
				Source: presetSourceDB,
			})
		}
	}

	// Config file (best-effort: missing file is silently skipped)
	configPath := globalFlags.ConfigFile
	if cfg, err := config.Load(configPath); err == nil {
		for i := range cfg.SettingsPresets {
			p := &cfg.SettingsPresets[i]
			entries = append(entries, presetListEntry{
				ID:     p.ID,
				Name:   p.Name,
				Source: presetSourceConfigPrefix + configPath,
			})
		}
	}

	// Bundled defaults (always present)
	for _, bp := range BundledPresets() {
		entries = append(entries, presetListEntry{
			ID:     bp.ID,
			Name:   bp.Name,
			Source: presetSourceBundled,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].ID != entries[j].ID {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].Source < entries[j].Source
	})

	resp := CLIResponse{
		Success: true,
		Action:  "listed",
		Type:    "preset",
		Data:    entries,
		Count:   len(entries),
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	output.Info(fmt.Sprintf("Found %d preset(s):", len(entries)))
	for _, e := range entries {
		if showSource {
			output.Info(fmt.Sprintf("  %s\t%s", e.ID, e.Source))
		} else {
			output.Info(fmt.Sprintf("  %s", e.ID))
		}
	}
	return nil
}

func newPresetsSeedCmd() *cobra.Command {
	var (
		jsonOutput bool
		fromDir    string
	)
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed bundled defaults (and optionally external preset files) into the database",
		Long: `Seed the bundled generic defaults (mvp, go-lib) into the database.

Always idempotent: existing rows with the same id are kept untouched. With
--from <dir>, every *.yaml/*.yml/*.json file in the supplied directory is also
parsed as a preset and upserted into the database. Conflicts with bundled
defaults override the existing row and emit an INFO log entry.`,
		Example: `  # Seed bundled defaults (idempotent)
  go-broadcast presets seed

  # Seed bundled defaults plus every preset file in a directory
  go-broadcast presets seed --from ./my-presets/

  # Verify what landed in the database afterwards
  go-broadcast presets list --show-source`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPresetsSeed(cmd.Context(), fromDir, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&fromDir, "from", "", "Directory containing preset YAML/JSON files to load")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func runPresetsSeed(ctx context.Context, fromDir string, jsonOutput bool) error {
	database, err := openDatabase()
	if err != nil {
		return printErrorResponse("preset", "seeded", err.Error(),
			"run 'go-broadcast db init' to create a database", jsonOutput)
	}
	defer func() { _ = database.Close() }()

	presetRepo := db.NewSettingsPresetRepository(database.DB())

	bundled := BundledPresets()
	dbPresets := make([]*db.SettingsPreset, 0, len(bundled))
	for i := range bundled {
		dbPresets = append(dbPresets, configPresetToDBPreset(&bundled[i]))
	}

	bundledSeeded, err := presetRepo.SeedIfMissing(ctx, dbPresets)
	if err != nil {
		return printErrorResponse("preset", "seeded", err.Error(), "", jsonOutput)
	}

	summary := presetSeedSummary{BundledSeeded: bundledSeeded}

	if fromDir != "" {
		summary.FromDir = fromDir
		loaded, overrides, loadErr := loadPresetsFromDir(ctx, presetRepo, fromDir)
		if loadErr != nil {
			return printErrorResponse("preset", "seeded", loadErr.Error(),
				"check the directory contents and YAML/JSON syntax", jsonOutput)
		}
		summary.LoadedFromDir = loaded
		summary.Overrides = overrides
	}

	resp := CLIResponse{
		Success: true,
		Action:  "seeded",
		Type:    "preset",
		Data:    summary,
	}

	if jsonOutput {
		return printResponse(resp, true)
	}

	if fromDir == "" {
		output.Info(fmt.Sprintf("Seeded %d bundled preset(s)", summary.BundledSeeded))
	} else {
		output.Info(fmt.Sprintf("Seeded %d bundled, loaded %d from %s (%d overrides)",
			summary.BundledSeeded, summary.LoadedFromDir, summary.FromDir, summary.Overrides))
	}
	return nil
}

// loadPresetsFromDir walks dir for *.yaml/*.yml/*.json files, parses each as a SettingsPreset,
// and upserts it into the database. Returns (loaded, overrides, error).
func loadPresetsFromDir(ctx context.Context, presetRepo db.SettingsPresetRepository, dir string) (int, int, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to stat directory: %w", err)
	}
	if !info.IsDir() {
		return 0, 0, fmt.Errorf("--from %q is not a directory", dir) //nolint:err113 // user-facing CLI error
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read directory: %w", err)
	}

	bundledIDs := map[string]struct{}{}
	for _, bp := range BundledPresets() {
		bundledIDs[bp.ID] = struct{}{}
	}

	loaded := 0
	overrides := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		preset, parseErr := parsePresetFile(path)
		if parseErr != nil {
			return loaded, overrides, fmt.Errorf("preset file %s: %w", path, parseErr)
		}
		if preset.ID == "" {
			return loaded, overrides, fmt.Errorf("preset file %s: missing required field 'id'", path) //nolint:err113 // user-facing CLI error
		}

		dbPreset := configPresetToDBPreset(preset)
		if _, lookupErr := presetRepo.GetByExternalID(ctx, preset.ID); lookupErr == nil {
			overrides++
			if _, isBundled := bundledIDs[preset.ID]; isBundled {
				logrus.WithField("preset", preset.ID).Info("overriding bundled default with preset from --from directory")
			} else {
				logrus.WithField("preset", preset.ID).Info("overriding existing preset with preset from --from directory")
			}
		} else if !errors.Is(lookupErr, db.ErrRecordNotFound) {
			return loaded, overrides, fmt.Errorf("failed to check preset %q: %w", preset.ID, lookupErr)
		}

		if importErr := presetRepo.ImportFromConfig(ctx, dbPreset); importErr != nil {
			return loaded, overrides, fmt.Errorf("failed to import preset %q: %w", preset.ID, importErr)
		}
		loaded++
	}

	logrus.WithFields(logrus.Fields{
		"loaded":    loaded,
		"overrides": overrides,
		"dir":       dir,
	}).Info("loaded presets from directory")

	return loaded, overrides, nil
}

// parsePresetFile decodes a single preset definition from a YAML or JSON file.
func parsePresetFile(path string) (*config.SettingsPreset, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-supplied directory of preset files
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}
	var preset config.SettingsPreset
	if err := yaml.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}
	return &preset, nil
}
