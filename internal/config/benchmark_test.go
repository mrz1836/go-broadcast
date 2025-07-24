package config

import (
	"bytes"
	"context"
	"fmt"
	"testing"
)

//nolint:gochecknoglobals // Test data
var sampleYAML = []byte(`version: 1
source:
  repo: "org/template-repo"
  branch: "master"
defaults:
  branch_prefix: "sync/template"
  pr_labels: ["automated-sync", "template-update"]
targets:
  - repo: "org/target-repo-1"
    files:
      - src: ".github/workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
      - src: "Makefile"
        dest: "Makefile"
    transform:
      repo_name: true
      variables:
        SERVICE_NAME: "target-service-1"
        ENVIRONMENT: "production"
  - repo: "org/target-repo-2"
    files:
      - src: ".github/workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
      - src: "docker-compose.yml"
        dest: "docker-compose.yml"
    transform:
      repo_name: true
      variables:
        SERVICE_NAME: "target-service-2"
        ENVIRONMENT: "staging"`)

//nolint:gochecknoglobals // Test data
var largeYAML = func() []byte {
	// Create a large YAML with many targets
	var buf bytes.Buffer
	buf.WriteString(`version: 1
source:
  repo: "org/template-repo"
  branch: "master"
defaults:
  branch_prefix: "sync/template"
  pr_labels: ["automated-sync"]
targets:`)

	for i := 0; i < 100; i++ {
		buf.WriteString(`
  - repo: "org/target-repo-`)
		buf.WriteString(fmt.Sprintf("%d", i))
		buf.WriteString(`"
    files:
      - src: ".github/workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
      - src: "Makefile"
        dest: "Makefile"
      - src: "README.md"
        dest: "README.md"
    transform:
      repo_name: true
      variables:
        SERVICE_NAME: "service-`)
		buf.WriteString(fmt.Sprintf("%d", i))
		buf.WriteString(`"
        ENVIRONMENT: "production"`)
	}
	return buf.Bytes()
}()

//nolint:gochecknoglobals // Benchmark result storage
var testConfig *Config

func BenchmarkLoadFromReader(b *testing.B) {
	var cfg *Config
	var err error

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(sampleYAML)
		cfg, err = LoadFromReader(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
	testConfig = cfg
}

func BenchmarkLoadFromReader_Large(b *testing.B) {
	var cfg *Config
	var err error

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(largeYAML)
		cfg, err = LoadFromReader(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
	testConfig = cfg
}

func BenchmarkValidate(b *testing.B) {
	reader := bytes.NewReader(sampleYAML)
	cfg, err := LoadFromReader(reader)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = cfg.ValidateWithLogging(context.Background(), nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidate_Large(b *testing.B) {
	reader := bytes.NewReader(largeYAML)
	cfg, err := LoadFromReader(reader)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = cfg.ValidateWithLogging(context.Background(), nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadAndValidate(b *testing.B) {
	var cfg *Config
	var err error

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(sampleYAML)
		cfg, err = LoadFromReader(reader)
		if err != nil {
			b.Fatal(err)
		}
		err = cfg.ValidateWithLogging(context.Background(), nil)
		if err != nil {
			b.Fatal(err)
		}
	}
	testConfig = cfg
}

func BenchmarkLoadAndValidate_Large(b *testing.B) {
	var cfg *Config
	var err error

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(largeYAML)
		cfg, err = LoadFromReader(reader)
		if err != nil {
			b.Fatal(err)
		}
		err = cfg.ValidateWithLogging(context.Background(), nil)
		if err != nil {
			b.Fatal(err)
		}
	}
	testConfig = cfg
}
