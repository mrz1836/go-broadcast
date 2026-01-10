//go:build mage

// Magefile for go-broadcast specific tasks
package main

import (
	"fmt"
	"sync"

	"github.com/magefile/mage/sh"
)

// Commander interface allows for dependency injection in tests
type Commander interface {
	RunV(cmd string, args ...string) error
}

// ShCommander wraps sh.RunV for production use
type ShCommander struct{}

// RunV implements Commander interface
func (s ShCommander) RunV(cmd string, args ...string) error {
	return sh.RunV(cmd, args...)
}

// CommanderManager manages the current commander instance
type CommanderManager struct {
	mu        sync.RWMutex
	commander Commander
	once      sync.Once
}

// defaultManager is the package-level manager
var defaultManager = &CommanderManager{} //nolint:gochecknoglobals // Required for mage pattern

// initCommander initializes the default commander
func (cm *CommanderManager) initCommander() {
	cm.once.Do(func() {
		cm.commander = ShCommander{}
	})
}

// setCommander allows setting the commander for testing
func setCommander(c Commander) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()
	defaultManager.commander = c
}

// getCommander returns the current commander
func getCommander() Commander {
	defaultManager.mu.RLock()
	defer defaultManager.mu.RUnlock()

	if defaultManager.commander == nil {
		defaultManager.mu.RUnlock()
		defaultManager.initCommander()
		defaultManager.mu.RLock()
	}

	return defaultManager.commander
}

// BenchHeavy runs intensive benchmarks excluded from CI
// This may take 10-30 minutes and includes:
// - Worker pool stress tests (1000+ concurrent tasks)
// - Large directory sync simulations
// - Memory efficiency tests with large datasets
// - Real-world scenario simulations
func BenchHeavy() error {
	return getCommander().RunV("go", "test", "-bench=.", "-benchmem",
		"-tags=bench_heavy", "-benchtime=1s", "-timeout=60m", "./...")
}

// BenchAll runs all benchmarks including heavy ones
// This may take 30-60 minutes total
func BenchAll() error {
	// Run default benchmarks first
	if err := getCommander().RunV("go", "test", "-bench=.", "-benchmem",
		"-benchtime=100ms", "-timeout=20m", "./..."); err != nil {
		return fmt.Errorf("quick benchmarks failed: %w", err)
	}

	// Then run heavy benchmarks
	return BenchHeavy()
}

// BenchQuick runs only the quick benchmarks (same as magex bench)
func BenchQuick() error {
	return getCommander().RunV("go", "test", "-bench=.", "-benchmem",
		"-benchtime=100ms", "-timeout=20m", "./...")
}

// TestQuick runs fast unit tests excluding performance tests
func TestQuick() error {
	return getCommander().RunV("go", "test", "-short", "./...")
}

// TestPerf runs performance regression tests with build tag
// These tests are excluded from regular runs due to long execution time
func TestPerf() error {
	return getCommander().RunV("go", "test", "-tags=performance", "-timeout=30m", "./test/integration")
}

// TestAll runs all tests including performance tests
func TestAll() error {
	if err := TestQuick(); err != nil {
		return fmt.Errorf("quick tests failed: %w", err)
	}
	return TestPerf()
}

// UpdateToolVersions checks and updates tool versions in .github/.env.base
// This command checks GitHub releases for the latest versions of all tools
// defined in .env.base and updates them if newer versions are available.
//
// By default, runs in dry-run mode (no changes).
// Set UPDATE_VERSIONS=true environment variable to apply updates.
//
// Major version upgrades (e.g., v1.x.x to v2.x.x) are skipped by default.
// Set ALLOW_MAJOR_UPGRADES=true to include major version upgrades.
//
// Usage:
//
//	mage updateToolVersions              # Dry run (no changes)
//	UPDATE_VERSIONS=true mage updateToolVersions  # Apply minor/patch updates only
//	UPDATE_VERSIONS=true ALLOW_MAJOR_UPGRADES=true mage updateToolVersions  # Apply all updates
//
// The command includes rate limiting (2s delay between checks) to avoid
// GitHub API rate limits and will use gh CLI if available for higher limits.
func UpdateToolVersions() error {
	// Check if we should actually update or just dry run
	dryRun := true
	updateEnv, _ := sh.Output("sh", "-c", "echo $UPDATE_VERSIONS")
	if updateEnv == "true" {
		dryRun = false
	}

	// Check if major upgrades are allowed (default: false)
	allowMajorUpgrades := false
	majorEnv, _ := sh.Output("sh", "-c", "echo $ALLOW_MAJOR_UPGRADES")
	if majorEnv == "true" {
		allowMajorUpgrades = true
	}

	return RunVersionUpdate(dryRun, allowMajorUpgrades)
}
