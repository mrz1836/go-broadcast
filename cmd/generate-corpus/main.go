// Command generate-corpus generates the initial fuzz test corpus
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/mrz1836/go-broadcast/internal/fuzz"
)

// ErrDirectoryNotExist indicates the specified directory does not exist
var ErrDirectoryNotExist = errors.New("directory does not exist")

func main() {
	app := NewGenerateCorpusApp()
	if err := app.Run(); err != nil {
		log.Fatalf("Corpus generation failed: %v", err)
	}
}

// GenerateCorpusApp represents the main corpus generation application
type GenerateCorpusApp struct {
	logger                 Logger
	fileSystem             FileSystem
	corpusGeneratorFactory CorpusGeneratorFactory
}

// Logger defines the interface for logging operations
type Logger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

// FileSystem defines the interface for file system operations
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
}

// CorpusGeneratorFactory defines the interface for creating corpus generators
type CorpusGeneratorFactory interface {
	NewCorpusGenerator(baseDir string) CorpusGenerator
}

// CorpusGenerator defines the interface for generating corpus data
type CorpusGenerator interface {
	GenerateAll() error
}

// DefaultLogger implements Logger using the log package
type DefaultLogger struct{}

func (d *DefaultLogger) Println(v ...interface{}) {
	log.Println(v...)
}

func (d *DefaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (d *DefaultLogger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// DefaultFileSystem implements FileSystem using the os package
type DefaultFileSystem struct{}

func (d *DefaultFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// DefaultCorpusGeneratorFactory implements CorpusGeneratorFactory
type DefaultCorpusGeneratorFactory struct{}

func (d *DefaultCorpusGeneratorFactory) NewCorpusGenerator(baseDir string) CorpusGenerator {
	return &DefaultCorpusGeneratorWrapper{generator: fuzz.NewCorpusGenerator(baseDir)}
}

// DefaultCorpusGeneratorWrapper wraps the actual fuzz.CorpusGenerator
type DefaultCorpusGeneratorWrapper struct {
	generator interface {
		GenerateAll() error
	}
}

func (d *DefaultCorpusGeneratorWrapper) GenerateAll() error {
	return d.generator.GenerateAll()
}

// NewGenerateCorpusApp creates a new GenerateCorpusApp with default implementations
func NewGenerateCorpusApp() *GenerateCorpusApp {
	return &GenerateCorpusApp{
		logger:                 &DefaultLogger{},
		fileSystem:             &DefaultFileSystem{},
		corpusGeneratorFactory: &DefaultCorpusGeneratorFactory{},
	}
}

// NewGenerateCorpusAppWithDependencies creates a new GenerateCorpusApp with injectable dependencies
func NewGenerateCorpusAppWithDependencies(logger Logger, fileSystem FileSystem, corpusGeneratorFactory CorpusGeneratorFactory) *GenerateCorpusApp {
	return &GenerateCorpusApp{
		logger:                 logger,
		fileSystem:             fileSystem,
		corpusGeneratorFactory: corpusGeneratorFactory,
	}
}

// Run executes the corpus generation process
func (app *GenerateCorpusApp) Run() error {
	// Use internal/fuzz as base directory
	baseDir := "internal/fuzz"

	app.logger.Println("Starting fuzz corpus generation...")

	// Check if directory exists
	if _, err := app.fileSystem.Stat(baseDir); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrDirectoryNotExist, baseDir)
	} else if err != nil {
		return fmt.Errorf("failed to check directory %s: %w", baseDir, err)
	}

	gen := app.corpusGeneratorFactory.NewCorpusGenerator(baseDir)

	app.logger.Println("Generating fuzz test corpus...")
	if err := gen.GenerateAll(); err != nil {
		return fmt.Errorf("failed to generate corpus: %w", err)
	}

	app.logger.Println("Corpus generation complete!")
	app.logger.Printf("Corpus files created in: %s/corpus/\n", baseDir)

	return nil
}
