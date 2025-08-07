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
	app := NewTestCleanupApp()
	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}
}

// TestCleanupApp represents the main application
type TestCleanupApp struct {
	flagParser  FlagParser
	fileWalker  FileWalker
	logger      Logger
	fileRemover FileRemover
}

// FlagParser defines interface for parsing command line flags
type FlagParser interface {
	ParseFlags(args []string) (TestCleanupConfig, error)
}

// FileWalker defines interface for walking directory trees
type FileWalker interface {
	Walk(root string, walkFunc WalkFunc) error
}

// WalkFunc is the function signature for file walking
type WalkFunc func(path string, info os.FileInfo, err error) error

// Logger defines interface for logging operations
type Logger interface {
	Printf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

// FileRemover defines interface for file removal operations
type FileRemover interface {
	Remove(filename string) error
	Stat(filename string) (os.FileInfo, error)
}

// DefaultFlagParser implements FlagParser using the flag package
type DefaultFlagParser struct{}

func (d *DefaultFlagParser) ParseFlags(args []string) (TestCleanupConfig, error) {
	var config TestCleanupConfig

	// Create new flag set for this instance
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.StringVar(&config.RootDir, "root", ".", "Root directory to clean (default: current directory)")
	fs.BoolVar(&config.DryRun, "dry-run", false, "Show what would be deleted without actually deleting")
	fs.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	patternsFlag := fs.String("patterns", "*.test,*.out,*.prof", "Comma-separated list of file patterns to clean")
	excludeFlag := fs.String("exclude-dirs", ".git,vendor,node_modules", "Comma-separated list of directories to exclude")

	if err := fs.Parse(args[1:]); err != nil {
		return config, err
	}

	// Parse patterns and exclude directories
	config.Patterns = strings.Split(*patternsFlag, ",")
	config.ExcludeDirs = strings.Split(*excludeFlag, ",")

	return config, nil
}

// DefaultFileWalker implements FileWalker using filepath.Walk
type DefaultFileWalker struct{}

func (d *DefaultFileWalker) Walk(root string, walkFunc WalkFunc) error {
	return filepath.Walk(root, filepath.WalkFunc(walkFunc))
}

// DefaultLogger implements Logger using the log package
type DefaultLogger struct{}

func (d *DefaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (d *DefaultLogger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// DefaultFileRemover implements FileRemover using the os package
type DefaultFileRemover struct{}

func (d *DefaultFileRemover) Remove(filename string) error {
	return os.Remove(filename)
}

func (d *DefaultFileRemover) Stat(filename string) (os.FileInfo, error) {
	return os.Stat(filename)
}

// NewTestCleanupApp creates a new TestCleanupApp with default implementations
func NewTestCleanupApp() *TestCleanupApp {
	return &TestCleanupApp{
		flagParser:  &DefaultFlagParser{},
		fileWalker:  &DefaultFileWalker{},
		logger:      &DefaultLogger{},
		fileRemover: &DefaultFileRemover{},
	}
}

// NewTestCleanupAppWithDependencies creates a new TestCleanupApp with injectable dependencies
func NewTestCleanupAppWithDependencies(flagParser FlagParser, fileWalker FileWalker, logger Logger, fileRemover FileRemover) *TestCleanupApp {
	return &TestCleanupApp{
		flagParser:  flagParser,
		fileWalker:  fileWalker,
		logger:      logger,
		fileRemover: fileRemover,
	}
}

// Run executes the test cleanup application
func (app *TestCleanupApp) Run(args []string) error {
	// Parse command line flags
	config, err := app.flagParser.ParseFlags(args)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if config.Verbose {
		app.logger.Printf("Test Cleanup Utility")
		app.logger.Printf("Root directory: %s", config.RootDir)
		app.logger.Printf("Patterns: %v", config.Patterns)
		app.logger.Printf("Exclude directories: %v", config.ExcludeDirs)
		app.logger.Printf("Dry run: %v", config.DryRun)
		app.logger.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}

	return app.cleanupTestFiles(config)
}

// cleanupTestFiles performs the actual file cleanup operation
func (app *TestCleanupApp) cleanupTestFiles(config TestCleanupConfig) error {
	var deletedFiles []string
	var totalSize int64

	err := app.fileWalker.Walk(config.RootDir, func(path string, info os.FileInfo, err error) error {
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
					app.logger.Printf("Found: %s (size: %d bytes)", path, info.Size())
				}

				if !config.DryRun {
					if err := app.fileRemover.Remove(path); err != nil {
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
		app.logger.Printf("\n🔍 Dry run completed")
		app.logger.Printf("Would delete %d files (total: %s)", len(deletedFiles), formatBytes(totalSize))
	} else {
		app.logger.Printf("\n✅ Cleanup completed")
		app.logger.Printf("Deleted %d files (total: %s)", len(deletedFiles), formatBytes(totalSize))
	}

	if config.Verbose && len(deletedFiles) > 0 {
		app.logger.Printf("\nFiles %s:", map[bool]string{true: "that would be deleted", false: "deleted"}[config.DryRun])
		for _, file := range deletedFiles {
			app.logger.Printf("  - %s", file)
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
