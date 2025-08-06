// Package main provides a test cleanup utility for removing test artifacts and temporary files.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// TestCleanupConfig holds configuration for cleanup operations
type TestCleanupConfig struct {
	RootDir     string
	DryRun      bool
	Verbose     bool
	Patterns    []string
	ExcludeDirs []string
}

func main() {
	var config TestCleanupConfig

	// Parse command line flags
	flag.StringVar(&config.RootDir, "root", ".", "Root directory to clean (default: current directory)")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be deleted without actually deleting")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	patternsFlag := flag.String("patterns", "*.test,*.out,*.prof", "Comma-separated list of file patterns to clean")
	excludeFlag := flag.String("exclude-dirs", ".git,vendor,node_modules", "Comma-separated list of directories to exclude")
	flag.Parse()

	// Parse patterns and exclude directories
	config.Patterns = strings.Split(*patternsFlag, ",")
	config.ExcludeDirs = strings.Split(*excludeFlag, ",")

	if config.Verbose {
		log.Printf("Test Cleanup Utility")
		log.Printf("Root directory: %s", config.RootDir)
		log.Printf("Patterns: %v", config.Patterns)
		log.Printf("Exclude directories: %v", config.ExcludeDirs)
		log.Printf("Dry run: %v", config.DryRun)
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}

	if err := cleanupTestFiles(config); err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}
}

func cleanupTestFiles(config TestCleanupConfig) error {
	var deletedFiles []string
	var totalSize int64

	err := filepath.Walk(config.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories
		if info.IsDir() {
			for _, excludeDir := range config.ExcludeDirs {
				if strings.Contains(path, excludeDir) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file matches any of the cleanup patterns
		fileName := info.Name()
		for _, pattern := range config.Patterns {
			pattern = strings.TrimSpace(pattern)
			if matched, err := filepath.Match(pattern, fileName); err == nil && matched {
				if config.Verbose {
					log.Printf("Found: %s (size: %d bytes)", path, info.Size())
				}

				if !config.DryRun {
					if err := os.Remove(path); err != nil {
						return fmt.Errorf("failed to remove %s: %w", path, err)
					}
				}

				deletedFiles = append(deletedFiles, path)
				totalSize += info.Size()
				break
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory tree: %w", err)
	}

	// Print summary
	if config.DryRun {
		log.Printf("\n🔍 Dry run completed")
		log.Printf("Would delete %d files (total: %s)", len(deletedFiles), formatBytes(totalSize))
	} else {
		log.Printf("\n✅ Cleanup completed")
		log.Printf("Deleted %d files (total: %s)", len(deletedFiles), formatBytes(totalSize))
	}

	if config.Verbose && len(deletedFiles) > 0 {
		log.Printf("\nFiles %s:", map[bool]string{true: "that would be deleted", false: "deleted"}[config.DryRun])
		for _, file := range deletedFiles {
			log.Printf("  - %s", file)
		}
	}

	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
