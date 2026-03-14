//go:build bench_heavy

package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// BenchmarkImportSmallConfig benchmarks importing a small configuration
func BenchmarkImportSmallConfig(b *testing.B) {
	ctx := context.Background()

	// Create a small config (1 group, 5 targets)
	cfg := &config.Config{
		ID:      "test-config",
		Name:    "Benchmark Config",
		Version: 1,
		Groups: []config.Group{
			{
				ID:          "bench-group",
				Name:        "Benchmark Group",
				Description: "Group for benchmarking",
				Priority:    1,
				Enabled:     ptrBool(true),
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: make([]config.TargetConfig, 5),
			},
		},
	}

	// Create 5 targets
	for i := 0; i < 5; i++ {
		cfg.Groups[0].Targets[i] = config.TargetConfig{
			Repo:   fmt.Sprintf("org/target-%d", i),
			Branch: "main",
			Files: []config.FileMapping{
				{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
				{Src: "README.md", Dest: "README.md"},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		db := TestDB(b)
		converter := NewConverter(db)
		b.StartTimer()

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(b, err)

		b.StopTimer()
		sqlDB, _ := db.DB()
		sqlDB.Close()
		b.StartTimer()
	}
}

// BenchmarkImportMediumConfig benchmarks importing a medium configuration
func BenchmarkImportMediumConfig(b *testing.B) {
	ctx := context.Background()

	// Create a medium config (3 groups, 50 targets total)
	cfg := &config.Config{
		ID:      "test-config",
		Name:    "Benchmark Config",
		Version: 1,
		Groups:  make([]config.Group, 3),
	}

	for g := 0; g < 3; g++ {
		cfg.Groups[g] = config.Group{
			ID:          fmt.Sprintf("bench-group-%d", g),
			Name:        fmt.Sprintf("Benchmark Group %d", g),
			Description: "Group for benchmarking",
			Priority:    g,
			Enabled:     ptrBool(true),
			Source: config.SourceConfig{
				Repo:   fmt.Sprintf("org/source-%d", g),
				Branch: "main",
			},
			Targets: make([]config.TargetConfig, 17),
		}

		for i := 0; i < 17; i++ {
			cfg.Groups[g].Targets[i] = config.TargetConfig{
				Repo:   fmt.Sprintf("org/target-%d-%d", g, i),
				Branch: "main",
				Files: []config.FileMapping{
					{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
					{Src: ".github/workflows/codeql.yml", Dest: ".github/workflows/codeql.yml"},
					{Src: "README.md", Dest: "README.md"},
				},
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		db := TestDB(b)
		converter := NewConverter(db)
		b.StartTimer()

		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(b, err)

		b.StopTimer()
		sqlDB, _ := db.DB()
		sqlDB.Close()
		b.StartTimer()
	}
}

// BenchmarkImportRealConfig benchmarks importing the real sync.yaml
func BenchmarkImportRealConfig(b *testing.B) {
	ctx := context.Background()

	// Load real sync.yaml
	syncYAML := os.Getenv("HOME") + "/projects/go-broadcast/sync.yaml"
	data, err := os.ReadFile(syncYAML)
	if err != nil {
		b.Skipf("sync.yaml not found at %s: %v", syncYAML, err)
	}

	var cfg config.Config
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		db := TestDB(b)
		converter := NewConverter(db)
		b.StartTimer()

		_, err := converter.ImportConfig(ctx, &cfg)
		require.NoError(b, err)

		b.StopTimer()
		sqlDB, _ := db.DB()
		sqlDB.Close()
		b.StartTimer()
	}
}

// BenchmarkExportSmallConfig benchmarks exporting a small configuration
func BenchmarkExportSmallConfig(b *testing.B) {
	ctx := context.Background()

	// Create and import a small config
	cfg := &config.Config{
		ID:      "test-config",
		Name:    "Benchmark Config",
		Version: 1,
		Groups: []config.Group{
			{
				ID:          "bench-group",
				Name:        "Benchmark Group",
				Description: "Group for benchmarking",
				Priority:    1,
				Enabled:     ptrBool(true),
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: make([]config.TargetConfig, 5),
			},
		},
	}

	for i := 0; i < 5; i++ {
		cfg.Groups[0].Targets[i] = config.TargetConfig{
			Repo:   fmt.Sprintf("org/target-%d", i),
			Branch: "main",
			Files: []config.FileMapping{
				{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
				{Src: "README.md", Dest: "README.md"},
			},
		}
	}

	db := TestDB(b)
	converter := NewConverter(db)

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ExportConfig(ctx, "test-config")
		require.NoError(b, err)
	}

	// Cleanup
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// BenchmarkExportRealConfig benchmarks exporting the real sync.yaml
func BenchmarkExportRealConfig(b *testing.B) {
	ctx := context.Background()

	// Load and import real sync.yaml
	syncYAML := os.Getenv("HOME") + "/projects/go-broadcast/sync.yaml"
	data, err := os.ReadFile(syncYAML)
	if err != nil {
		b.Skipf("sync.yaml not found at %s: %v", syncYAML, err)
	}

	var cfg config.Config
	err = yaml.Unmarshal(data, &cfg)
	require.NoError(b, err)

	db := TestDB(b)
	converter := NewConverter(db)

	_, err = converter.ImportConfig(ctx, &cfg)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ExportConfig(ctx, cfg.ID)
		require.NoError(b, err)
	}

	// Cleanup
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// BenchmarkQueryByFile benchmarks querying targets by file path
func BenchmarkQueryByFile(b *testing.B) {
	ctx := context.Background()

	// Create and import a config with many targets
	cfg := &config.Config{
		ID:      "test-config",
		Name:    "Benchmark Config",
		Version: 1,
		Groups: []config.Group{
			{
				ID:          "bench-group",
				Name:        "Benchmark Group",
				Description: "Group for benchmarking",
				Priority:    1,
				Enabled:     ptrBool(true),
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: make([]config.TargetConfig, 100),
			},
		},
	}

	// Create 100 targets with the same file
	for i := 0; i < 100; i++ {
		cfg.Groups[0].Targets[i] = config.TargetConfig{
			Repo:   fmt.Sprintf("org/target-%d", i),
			Branch: "main",
			Files: []config.FileMapping{
				{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
				{Src: fmt.Sprintf("file-%d.txt", i), Dest: fmt.Sprintf("file-%d.txt", i)},
			},
		}
	}

	db := TestDB(b)
	converter := NewConverter(db)

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(b, err)

	queryRepo := NewQueryRepository(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		targets, err := queryRepo.FindByFile(ctx, ".github/workflows/ci.yml")
		require.NoError(b, err)
		require.Len(b, targets, 100)
	}

	// Cleanup
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// BenchmarkQueryByRepo benchmarks querying file mappings by repository
func BenchmarkQueryByRepo(b *testing.B) {
	ctx := context.Background()

	// Create and import a config
	cfg := &config.Config{
		ID:      "test-config",
		Name:    "Benchmark Config",
		Version: 1,
		Groups: []config.Group{
			{
				ID:          "bench-group",
				Name:        "Benchmark Group",
				Description: "Group for benchmarking",
				Priority:    1,
				Enabled:     ptrBool(true),
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: make([]config.TargetConfig, 50),
			},
		},
	}

	for i := 0; i < 50; i++ {
		cfg.Groups[0].Targets[i] = config.TargetConfig{
			Repo:   fmt.Sprintf("org/target-%d", i),
			Branch: "main",
			Files: []config.FileMapping{
				{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
				{Src: ".github/workflows/codeql.yml", Dest: ".github/workflows/codeql.yml"},
				{Src: "README.md", Dest: "README.md"},
			},
		}
	}

	db := TestDB(b)
	converter := NewConverter(db)

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(b, err)

	queryRepo := NewQueryRepository(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		target, err := queryRepo.FindByRepo(ctx, "org/target-25")
		require.NoError(b, err)
		require.NotNil(b, target)
	}

	// Cleanup
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// BenchmarkCreateTarget benchmarks creating a single target
func BenchmarkCreateTarget(b *testing.B) {
	ctx := context.Background()

	// Create a config and group first
	db := TestDB(b)

	cfg := &Config{
		ExternalID: "test-config",
		Name:       "Benchmark Config",
		Version:    1,
	}
	err := db.Create(cfg).Error
	require.NoError(b, err)

	group := &Group{
		ConfigID:    cfg.ID,
		ExternalID:  "bench-group",
		Name:        "Benchmark Group",
		Description: "Group for benchmarking",
		Priority:    1,
		Enabled:     ptrBool(true),
	}
	err = db.Create(group).Error
	require.NoError(b, err)

	// Create source
	source := &Source{
		GroupID: group.ID,
		Repo:    "org/source",
		Branch:  "main",
	}
	err = db.Create(source).Error
	require.NoError(b, err)

	targetRepo := NewTargetRepository(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		target := &Target{
			GroupID: group.ID,
			Repo:    fmt.Sprintf("org/target-%d", i),
			Branch:  "main",
		}
		err := targetRepo.Create(ctx, target)
		require.NoError(b, err)
	}

	// Cleanup
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// BenchmarkListGroups benchmarks listing all groups
func BenchmarkListGroups(b *testing.B) {
	ctx := context.Background()

	// Create a config with many groups
	cfg := &config.Config{
		ID:      "test-config",
		Name:    "Benchmark Config",
		Version: 1,
		Groups:  make([]config.Group, 20),
	}

	for i := 0; i < 20; i++ {
		cfg.Groups[i] = config.Group{
			ID:          fmt.Sprintf("bench-group-%d", i),
			Name:        fmt.Sprintf("Benchmark Group %d", i),
			Description: "Group for benchmarking",
			Priority:    i,
			Enabled:     ptrBool(true),
			Source: config.SourceConfig{
				Repo:   fmt.Sprintf("org/source-%d", i),
				Branch: "main",
			},
			Targets: []config.TargetConfig{
				{Repo: fmt.Sprintf("org/target-%d", i), Branch: "main"},
			},
		}
	}

	db := TestDB(b)
	converter := NewConverter(db)

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(b, err)

	groupRepo := NewGroupRepository(db)

	// Get the config ID
	var dbCfg Config
	err = db.Where("external_id = ?", "test-config").First(&dbCfg).Error
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		groups, err := groupRepo.List(ctx, dbCfg.ID)
		require.NoError(b, err)
		require.Len(b, groups, 20)
	}

	// Cleanup
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// BenchmarkRoundTrip benchmarks a complete import → export cycle
func BenchmarkRoundTrip(b *testing.B) {
	ctx := context.Background()

	// Create a medium config
	cfg := &config.Config{
		ID:      "test-config",
		Name:    "Benchmark Config",
		Version: 1,
		Groups: []config.Group{
			{
				ID:          "bench-group",
				Name:        "Benchmark Group",
				Description: "Group for benchmarking",
				Priority:    1,
				Enabled:     ptrBool(true),
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: make([]config.TargetConfig, 20),
			},
		},
	}

	for i := 0; i < 20; i++ {
		cfg.Groups[0].Targets[i] = config.TargetConfig{
			Repo:   fmt.Sprintf("org/target-%d", i),
			Branch: "main",
			Files: []config.FileMapping{
				{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
				{Src: "README.md", Dest: "README.md"},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		db := TestDB(b)
		converter := NewConverter(db)
		b.StartTimer()

		// Import
		_, err := converter.ImportConfig(ctx, cfg)
		require.NoError(b, err)

		// Export
		_, err = converter.ExportConfig(ctx, "test-config")
		require.NoError(b, err)

		b.StopTimer()
		sqlDB, _ := db.DB()
		sqlDB.Close()
		b.StartTimer()
	}
}

// BenchmarkConcurrentReads benchmarks concurrent read operations
func BenchmarkConcurrentReads(b *testing.B) {
	ctx := context.Background()

	// Create and import a config
	cfg := &config.Config{
		ID:      "test-config",
		Name:    "Benchmark Config",
		Version: 1,
		Groups: []config.Group{
			{
				ID:          "bench-group",
				Name:        "Benchmark Group",
				Description: "Group for benchmarking",
				Priority:    1,
				Enabled:     ptrBool(true),
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: make([]config.TargetConfig, 10),
			},
		},
	}

	for i := 0; i < 10; i++ {
		cfg.Groups[0].Targets[i] = config.TargetConfig{
			Repo:   fmt.Sprintf("org/target-%d", i),
			Branch: "main",
			Files: []config.FileMapping{
				{Src: ".github/workflows/ci.yml", Dest: ".github/workflows/ci.yml"},
			},
		}
	}

	db := TestDB(b)
	converter := NewConverter(db)

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(b, err)

	groupRepo := NewGroupRepository(db)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := groupRepo.GetByExternalID(ctx, "bench-group")
			require.NoError(b, err)
		}
	})

	// Cleanup
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// ptrBool returns a pointer to the given bool value (avoid name conflict with boolPtr in converter.go)
func ptrBool(b bool) *bool {
	return &b
}
