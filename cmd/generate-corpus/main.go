// Command generate-corpus generates the initial fuzz test corpus
package main

import (
	"log"
	"os"

	"github.com/mrz1836/go-broadcast/internal/fuzz"
)

func main() {
	// Use internal/fuzz as base directory
	baseDir := "internal/fuzz"

	// Check if directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		log.Fatalf("Directory %s does not exist", baseDir)
	}

	gen := fuzz.NewCorpusGenerator(baseDir)

	log.Println("Generating fuzz test corpus...")
	if err := gen.GenerateAll(); err != nil {
		log.Fatalf("Failed to generate corpus: %v", err)
	}

	log.Println("Corpus generation complete!")
	log.Printf("Corpus files created in: %s/corpus/\n", baseDir)
}
