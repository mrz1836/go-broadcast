#!/bin/bash
# Generator script for directory sync test fixtures
# This script creates all the directory structures needed for integration tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$SCRIPT_DIR/directories"

echo "Directory Sync Test Fixtures Generator"
echo "======================================"
echo "Base directory: $BASE_DIR"
echo ""

# Function to clean existing fixtures
clean_fixtures() {
    echo "Cleaning existing fixtures..."
    rm -rf "$BASE_DIR"
    mkdir -p "$BASE_DIR"
    echo "✓ Cleaned existing fixtures"
}

# Function to create small directory fixture
create_small() {
    echo "Creating small directory fixture (10 files)..."

    mkdir -p "$BASE_DIR/small/config" "$BASE_DIR/small/docs"

    # Create 10 files with transforms
    cat > "$BASE_DIR/small/README.md" << 'EOF'
# {{SERVICE_NAME}} Service

This is a template repository for {{SERVICE_NAME}}.

## Features
- Configuration management
- Documentation
- Basic transforms

Repository: {{REPO_NAME}}
Environment: {{ENVIRONMENT}}
EOF

    cat > "$BASE_DIR/small/config/app.yaml" << 'EOF'
app:
  name: "{{SERVICE_NAME}}"
  version: "1.0.0"
  environment: "{{ENVIRONMENT}}"
  repository: "{{REPO_NAME}}"

server:
  port: 8080
  host: "0.0.0.0"

logging:
  level: "info"
  format: "json"
EOF

    cat > "$BASE_DIR/small/config/database.yml" << 'EOF'
database:
  host: "localhost"
  port: 5432
  name: "{{SERVICE_NAME}}_db"
  user: "{{SERVICE_NAME}}_user"
  password: "changeme"
  ssl_mode: "require"
  max_connections: 100
EOF

    cat > "$BASE_DIR/small/config.txt" << 'EOF'
# Configuration file for {{SERVICE_NAME}}
SERVICE_NAME={{SERVICE_NAME}}
ENVIRONMENT={{ENVIRONMENT}}
REPO={{REPO_NAME}}
DEBUG=false
PORT=8080
EOF

    cat > "$BASE_DIR/small/docs/api.md" << 'EOF'
# {{SERVICE_NAME}} API Documentation

## Overview
API documentation for {{SERVICE_NAME}} service.

Repository: {{REPO_NAME}}

## Endpoints

### Health Check
```
GET /health
```

### Service Info
```
GET /info
```

### Configuration
```
GET /config
```

Environment: {{ENVIRONMENT}}
EOF

    cat > "$BASE_DIR/small/docs/deployment.md" << 'EOF'
# Deployment Guide for {{SERVICE_NAME}}

## Prerequisites
- Docker
- Kubernetes
- {{SERVICE_NAME}} configuration

## Steps
1. Build image for {{SERVICE_NAME}}
2. Deploy to {{ENVIRONMENT}}
3. Verify deployment

Repository: {{REPO_NAME}}
EOF

    cat > "$BASE_DIR/small/Makefile" << 'EOF'
# Makefile for {{SERVICE_NAME}}
# Repository: {{REPO_NAME}}

.PHONY: build test clean deploy

SERVICE_NAME := {{SERVICE_NAME}}
ENVIRONMENT := {{ENVIRONMENT}}

build:
	@echo "Building $(SERVICE_NAME) for $(ENVIRONMENT)"
	go build -o bin/$(SERVICE_NAME) ./cmd/$(SERVICE_NAME)

test:
	@echo "Testing $(SERVICE_NAME)"
	go test -v ./...

clean:
	@echo "Cleaning $(SERVICE_NAME)"
	rm -rf bin/

deploy:
	@echo "Deploying $(SERVICE_NAME) to $(ENVIRONMENT)"
	kubectl apply -f deployments/$(ENVIRONMENT)/
EOF

    cat > "$BASE_DIR/small/docker-compose.yaml" << 'EOF'
version: "3.8"

services:
  "{{SERVICE_NAME}}":
    build: .
    container_name: "{{SERVICE_NAME}}-container"
    ports:
      - "8080:8080"
    environment:
      - "SERVICE_NAME={{SERVICE_NAME}}"
      - "ENVIRONMENT={{ENVIRONMENT}}"
      - "REPO={{REPO_NAME}}"
    volumes:
      - "./config:/app/config"
    depends_on:
      - database

  database:
    image: "postgres:13"
    container_name: "{{SERVICE_NAME}}-db"
    environment:
      POSTGRES_DB: "{{SERVICE_NAME}}_db"
      POSTGRES_USER: "{{SERVICE_NAME}}_user"
      POSTGRES_PASSWORD: "changeme"
    ports:
      - "5432:5432"
EOF

    cat > "$BASE_DIR/small/temp.log" << 'EOF'
2024-01-01 00:00:00 INFO Starting {{SERVICE_NAME}}
2024-01-01 00:00:01 INFO Configuration loaded from {{REPO_NAME}}
2024-01-01 00:00:02 INFO Environment: {{ENVIRONMENT}}
2024-01-01 00:00:03 WARN This is a temporary log file
2024-01-01 00:00:04 INFO Service ready
EOF

    cat > "$BASE_DIR/small/.gitignore" << 'EOF'
# Logs
*.log
logs/

# Build artifacts
bin/
dist/
build/

# Dependencies
node_modules/
vendor/

# OS files
.DS_Store
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo

# Environment files
.env
.env.local

# Temporary files
*.tmp
*.temp
temp/
EOF

    echo "✓ Created small directory fixture (10 files)"
}

# Function to run medium fixture script
create_medium() {
    echo "Creating medium directory fixture (100 files)..."
    if [ -f "$SCRIPT_DIR/create_medium.sh" ]; then
        bash "$SCRIPT_DIR/create_medium.sh"
        echo "✓ Created medium directory fixture"
    else
        echo "⚠ Medium fixture script not found, skipping"
    fi
}

# Function to run large fixture script
create_large() {
    echo "Creating large directory fixture (1000 files)..."
    if [ -f "$SCRIPT_DIR/create_large.sh" ]; then
        bash "$SCRIPT_DIR/create_large.sh"
        echo "✓ Created large directory fixture"
    else
        echo "⚠ Large fixture script not found, skipping"
    fi
}

# Function to run complex fixture script
create_complex() {
    echo "Creating complex nested structure fixture..."
    if [ -f "$SCRIPT_DIR/create_complex.sh" ]; then
        bash "$SCRIPT_DIR/create_complex.sh"
        echo "✓ Created complex nested structure fixture"
    else
        echo "⚠ Complex fixture script not found, skipping"
    fi
}

# Function to run github fixture script
create_github() {
    echo "Creating GitHub-like structure fixture..."
    if [ -f "$SCRIPT_DIR/create_github.sh" ]; then
        bash "$SCRIPT_DIR/create_github.sh"
        echo "✓ Created GitHub-like structure fixture"
    else
        echo "⚠ GitHub fixture script not found, skipping"
    fi
}

# Function to run mixed fixture script
create_mixed() {
    echo "Creating mixed binary/text files fixture..."
    if [ -f "$SCRIPT_DIR/create_mixed.sh" ]; then
        bash "$SCRIPT_DIR/create_mixed.sh" 2>/dev/null || true
        echo "✓ Created mixed binary/text files fixture"
    else
        echo "⚠ Mixed fixture script not found, skipping"
    fi
}

# Function to display summary
show_summary() {
    echo ""
    echo "Fixture Generation Summary"
    echo "========================="

    if [ -d "$BASE_DIR/small" ]; then
        small_count=$(find "$BASE_DIR/small" -type f | wc -l)
        echo "Small fixture: $small_count files"
    fi

    if [ -d "$BASE_DIR/medium" ]; then
        medium_count=$(find "$BASE_DIR/medium" -type f | wc -l)
        echo "Medium fixture: $medium_count files"
    fi

    if [ -d "$BASE_DIR/large" ]; then
        large_count=$(find "$BASE_DIR/large" -type f | wc -l)
        echo "Large fixture: $large_count files"
    fi

    if [ -d "$BASE_DIR/complex" ]; then
        complex_count=$(find "$BASE_DIR/complex" -type f | wc -l)
        echo "Complex fixture: $complex_count files"
    fi

    if [ -d "$BASE_DIR/github" ]; then
        github_count=$(find "$BASE_DIR/github" -type f | wc -l)
        echo "GitHub fixture: $github_count files"
    fi

    if [ -d "$BASE_DIR/mixed" ]; then
        mixed_count=$(find "$BASE_DIR/mixed" -type f | wc -l)
        mixed_size=$(du -sh "$BASE_DIR/mixed" 2>/dev/null | cut -f1 || echo "unknown")
        echo "Mixed fixture: $mixed_count files ($mixed_size)"
    fi

    echo ""
    echo "All fixtures are ready for testing!"
    echo "Use these fixtures in your directory sync integration tests."
}

# Main execution
main() {
    case "${1:-all}" in
        "clean")
            clean_fixtures
            ;;
        "small")
            create_small
            ;;
        "medium")
            create_medium
            ;;
        "large")
            create_large
            ;;
        "complex")
            create_complex
            ;;
        "github")
            create_github
            ;;
        "mixed")
            create_mixed
            ;;
        "all")
            clean_fixtures
            create_small
            create_medium
            create_large
            create_complex
            create_github
            create_mixed
            show_summary
            ;;
        "help"|"-h"|"--help")
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  all     - Generate all fixtures (default)"
            echo "  clean   - Clean existing fixtures"
            echo "  small   - Generate small fixture only"
            echo "  medium  - Generate medium fixture only"
            echo "  large   - Generate large fixture only"
            echo "  complex - Generate complex fixture only"
            echo "  github  - Generate GitHub fixture only"
            echo "  mixed   - Generate mixed fixture only"
            echo "  help    - Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Generate all fixtures"
            echo "  $0 small              # Generate only small fixture"
            echo "  $0 clean              # Clean all fixtures"
            ;;
        *)
            echo "Unknown command: $1"
            echo "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
