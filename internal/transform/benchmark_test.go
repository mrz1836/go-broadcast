package transform

import (
	"bytes"
	"context"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
	"github.com/sirupsen/logrus"
)

//nolint:gochecknoglobals // Test data
var (
	smallGoFile = []byte(`package main

import (
	"fmt"
	"github.com/org/template-repo/pkg/utils"
	"github.com/org/template-repo/internal/config"
)

func main() {
	fmt.Println("Hello from github.com/org/template-repo")
}`)

	largeGoFile = bytes.Repeat([]byte(`package service

import (
	"context"
	"fmt"
	"github.com/org/template-repo/pkg/utils"
	"github.com/org/template-repo/internal/config"
	"github.com/org/template-repo/internal/handler"
)

// Service implements the main service logic
type Service struct {
	config *config.Config
}

// NewService creates a new service instance
func NewService() *Service {
	return &Service{}
}

// Run starts the service
func (s *Service) Run(ctx context.Context) error {
	fmt.Println("Running service from github.com/org/template-repo")
	return nil
}
`), 50)  // ~14KB file

	smallMarkdown = []byte(`# Template Repository

[![Build Status](https://github.com/org/template-repo/workflows/CI/badge.svg)](https://github.com/org/template-repo/actions)

This is the template repository for org/template-repo.

## Installation

` + "```bash" + `
go get github.com/org/template-repo
` + "```" + `

Visit https://github.com/org/template-repo for more information.`)

	templateContent = []byte(`# {{SERVICE_NAME}} Service

Environment: ${ENVIRONMENT}
Port: {{SERVICE_PORT}}

This service ({{SERVICE_NAME}}) runs in ${ENVIRONMENT} mode.

Configuration:
- Database: {{DB_HOST}}:{{DB_PORT}}
- Cache: ${CACHE_HOST}:${CACHE_PORT}
- API Key: {{API_KEY}}

{{SERVICE_NAME}} is part of the {{PLATFORM_NAME}} platform.`)

	binaryContent = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
)

func BenchmarkTemplateTransform_Small(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	transformer := NewTemplateTransformer(logger, nil)
	ctx := Context{
		FilePath: "config.yaml",
		Variables: map[string]string{
			"SERVICE_NAME":  "my-service",
			"ENVIRONMENT":   "production",
			"SERVICE_PORT":  "8080",
			"DB_HOST":       "localhost",
			"DB_PORT":       "5432",
			"CACHE_HOST":    "redis",
			"CACHE_PORT":    "6379",
			"API_KEY":       "secret-key",
			"PLATFORM_NAME": "MyPlatform",
		},
	}

	benchmark.WithMemoryTracking(b, func() {
		result, err := transformer.Transform(templateContent, ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	})
}

func BenchmarkTemplateTransform_Large(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create large content with many templates
	largeTemplate := bytes.Repeat(templateContent, 100)

	transformer := NewTemplateTransformer(logger, nil)
	ctx := Context{
		FilePath: "large-config.yaml",
		Variables: map[string]string{
			"SERVICE_NAME":  "my-service",
			"ENVIRONMENT":   "production",
			"SERVICE_PORT":  "8080",
			"DB_HOST":       "localhost",
			"DB_PORT":       "5432",
			"CACHE_HOST":    "redis",
			"CACHE_PORT":    "6379",
			"API_KEY":       "secret-key",
			"PLATFORM_NAME": "MyPlatform",
		},
	}

	benchmark.WithMemoryTracking(b, func() {
		result, err := transformer.Transform(largeTemplate, ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	})
}

func BenchmarkRepoTransform_GoFile(b *testing.B) {
	transformer := NewRepoTransformer()
	ctx := Context{
		FilePath:   "main.go",
		SourceRepo: "org/template-repo",
		TargetRepo: "myorg/my-service",
	}

	benchmark.WithMemoryTracking(b, func() {
		result, err := transformer.Transform(smallGoFile, ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	})
}

func BenchmarkRepoTransform_GoFile_Large(b *testing.B) {
	transformer := NewRepoTransformer()
	ctx := Context{
		FilePath:   "service.go",
		SourceRepo: "org/template-repo",
		TargetRepo: "myorg/my-service",
	}

	benchmark.WithMemoryTracking(b, func() {
		result, err := transformer.Transform(largeGoFile, ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	})
}

func BenchmarkRepoTransform_Markdown(b *testing.B) {
	transformer := NewRepoTransformer()
	ctx := Context{
		FilePath:   "README.md",
		SourceRepo: "org/template-repo",
		TargetRepo: "myorg/my-service",
	}

	benchmark.WithMemoryTracking(b, func() {
		result, err := transformer.Transform(smallMarkdown, ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	})
}

func BenchmarkBinaryDetection(b *testing.B) {
	testCases := []struct {
		name     string
		filePath string
		content  []byte
	}{
		{"Text", "test.txt", []byte("This is plain text content")},
		{"Binary", "test.png", binaryContent},
		{"LargeText", "large.txt", bytes.Repeat([]byte("Hello World "), 1000)},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			benchmark.WithMemoryTracking(b, func() {
				result := IsBinary(tc.filePath, tc.content)
				_ = result
			})
		})
	}
}

func BenchmarkChainTransform(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create a chain with all transformers
	chain := NewChain(logger).
		Add(NewBinaryTransformer()).
		Add(NewRepoTransformer()).
		Add(NewTemplateTransformer(logger, nil))

	transformCtx := Context{
		FilePath:   "service/main.go",
		SourceRepo: "org/template-repo",
		TargetRepo: "myorg/my-service",
		Variables: map[string]string{
			"SERVICE_NAME": "my-service",
			"ENVIRONMENT":  "production",
		},
	}

	// Content with both repo references and template variables
	content := []byte(`package main

import (
	"fmt"
	"github.com/org/template-repo/pkg/utils"
)

func main() {
	fmt.Println("{{SERVICE_NAME}} running in ${ENVIRONMENT}")
	fmt.Println("Visit https://github.com/org/template-repo")
}`)

	benchmark.WithMemoryTracking(b, func() {
		result, err := chain.Transform(context.Background(), content, transformCtx)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	})
}

func BenchmarkChainTransform_Binary(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create a chain with all transformers
	chain := NewChain(logger).
		Add(NewBinaryTransformer()).
		Add(NewRepoTransformer()).
		Add(NewTemplateTransformer(logger, nil))

	transformCtx := Context{
		FilePath:   "image.png",
		SourceRepo: "org/template-repo",
		TargetRepo: "myorg/my-service",
	}

	benchmark.WithMemoryTracking(b, func() {
		result, err := chain.Transform(context.Background(), binaryContent, transformCtx)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	})
}
