#!/bin/bash
# Creates GitHub-like repository structure fixture
# Based on typical Go project layout

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$SCRIPT_DIR/directories/github"

echo "Creating GitHub-like repository structure fixture..."

# Clean and create base directory
rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# Create standard Go project structure
mkdir -p "$BASE_DIR/cmd/server"
mkdir -p "$BASE_DIR/cmd/client"
mkdir -p "$BASE_DIR/internal/config"
mkdir -p "$BASE_DIR/internal/handlers"
mkdir -p "$BASE_DIR/internal/models"
mkdir -p "$BASE_DIR/internal/services"
mkdir -p "$BASE_DIR/pkg/utils"
mkdir -p "$BASE_DIR/pkg/client"
mkdir -p "$BASE_DIR/docs"
mkdir -p "$BASE_DIR/scripts"
mkdir -p "$BASE_DIR/deployments/docker"
mkdir -p "$BASE_DIR/deployments/k8s"

# Create main.go files
cat > "$BASE_DIR/cmd/server/main.go" << 'EOF'
package main

import (
    "fmt"
    "log"
    "net/http"
    
    "{{REPO_NAME}}/internal/config"
    "{{REPO_NAME}}/internal/handlers"
)

func main() {
    cfg := config.Load()
    
    mux := http.NewServeMux()
    handlers.RegisterRoutes(mux)
    
    fmt.Printf("Starting {{SERVICE_NAME}} server on port %s\n", cfg.Port)
    fmt.Printf("Environment: {{ENVIRONMENT}}\n")
    fmt.Printf("Repository: {{REPO_NAME}}\n")
    
    log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
EOF

cat > "$BASE_DIR/cmd/client/main.go" << 'EOF'
package main

import (
    "fmt"
    "{{REPO_NAME}}/pkg/client"
)

func main() {
    fmt.Println("{{SERVICE_NAME}} Client")
    fmt.Println("Repository: {{REPO_NAME}}")
    fmt.Println("Environment: {{ENVIRONMENT}}")
    
    c := client.New("http://localhost:8080")
    if err := c.Connect(); err != nil {
        fmt.Printf("Connection failed: %v\n", err)
        return
    }
    
    fmt.Println("Connected successfully")
}
EOF

# Create internal packages
cat > "$BASE_DIR/internal/config/config.go" << 'EOF'
package config

import (
    "os"
)

type Config struct {
    ServiceName string
    Environment string
    Repository  string
    Port        string
    Debug       bool
}

func Load() *Config {
    return &Config{
        ServiceName: getEnv("SERVICE_NAME", "{{SERVICE_NAME}}"),
        Environment: getEnv("ENVIRONMENT", "{{ENVIRONMENT}}"),
        Repository:  getEnv("REPO_NAME", "{{REPO_NAME}}"),
        Port:        getEnv("PORT", "8080"),
        Debug:       getEnv("DEBUG", "false") == "true",
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
EOF

cat > "$BASE_DIR/internal/handlers/handlers.go" << 'EOF'
package handlers

import (
    "encoding/json"
    "net/http"
    "{{REPO_NAME}}/internal/config"
)

func RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/health", healthHandler)
    mux.HandleFunc("/info", infoHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
    cfg := config.Load()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "service":     cfg.ServiceName,
        "environment": cfg.Environment,
        "repository":  cfg.Repository,
    })
}
EOF

cat > "$BASE_DIR/internal/models/models.go" << 'EOF'
package models

type Service struct {
    Name        string `json:"name"`
    Environment string `json:"environment"`
    Repository  string `json:"repository"`
    Version     string `json:"version"`
}

type Response struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}
EOF

cat > "$BASE_DIR/internal/services/service.go" << 'EOF'
package services

import (
    "{{REPO_NAME}}/internal/models"
)

type ServiceManager struct {
    config *models.Service
}

func New() *ServiceManager {
    return &ServiceManager{
        config: &models.Service{
            Name:        "{{SERVICE_NAME}}",
            Environment: "{{ENVIRONMENT}}",
            Repository:  "{{REPO_NAME}}",
            Version:     "1.0.0",
        },
    }
}

func (s *ServiceManager) GetInfo() *models.Service {
    return s.config
}
EOF

# Create pkg packages
cat > "$BASE_DIR/pkg/utils/utils.go" << 'EOF'
package utils

import (
    "fmt"
    "strings"
)

func FormatServiceName(name string) string {
    return strings.ToUpper(name)
}

func BuildServiceInfo(name, env, repo string) string {
    return fmt.Sprintf("Service: %s, Environment: %s, Repository: %s", name, env, repo)
}

func ReplaceTemplateVars(content, serviceName, environment, repoName string) string {
    content = strings.ReplaceAll(content, "{{SERVICE_NAME}}", serviceName)
    content = strings.ReplaceAll(content, "{{ENVIRONMENT}}", environment)
    content = strings.ReplaceAll(content, "{{REPO_NAME}}", repoName)
    return content
}
EOF

cat > "$BASE_DIR/pkg/client/client.go" << 'EOF'
package client

import (
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func New(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        client:  &http.Client{},
    }
}

func (c *Client) Connect() error {
    resp, err := c.client.Get(c.baseURL + "/health")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("health check failed: %d", resp.StatusCode)
    }
    
    return nil
}

func (c *Client) GetInfo() (map[string]interface{}, error) {
    resp, err := c.client.Get(c.baseURL + "/info")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var info map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
        return nil, err
    }
    
    return info, nil
}
EOF

# Create documentation
cat > "$BASE_DIR/docs/README.md" << 'EOF'
# {{SERVICE_NAME}} Documentation

## Overview
{{SERVICE_NAME}} is a service for testing directory synchronization and template replacement.

- Repository: {{REPO_NAME}}
- Environment: {{ENVIRONMENT}}

## Architecture
This is a standard Go project structure with:
- `cmd/` - Application entry points
- `internal/` - Private application code
- `pkg/` - Public library code
- `docs/` - Documentation
- `scripts/` - Build and deployment scripts
- `deployments/` - Deployment configurations

## API Endpoints
- `GET /health` - Health check
- `GET /info` - Service information

## Configuration
Environment variables:
- `SERVICE_NAME` - Name of the service (default: {{SERVICE_NAME}})
- `ENVIRONMENT` - Deployment environment (default: {{ENVIRONMENT}})
- `REPO_NAME` - Repository name (default: {{REPO_NAME}})
- `PORT` - Server port (default: 8080)
- `DEBUG` - Enable debug mode (default: false)
EOF

cat > "$BASE_DIR/docs/api.md" << 'EOF'
# {{SERVICE_NAME}} API Reference

## Base URL
```
http://localhost:8080
```

## Endpoints

### Health Check
Check if the service is running.

**Request:**
```
GET /health
```

**Response:**
```json
{
  "status": "ok"
}
```

### Service Information
Get service configuration details.

**Request:**
```
GET /info
```

**Response:**
```json
{
  "service": "{{SERVICE_NAME}}",
  "environment": "{{ENVIRONMENT}}",
  "repository": "{{REPO_NAME}}"
}
```

## Authentication
This service does not require authentication for testing endpoints.

## Environment Variables
- `SERVICE_NAME`: {{SERVICE_NAME}}
- `ENVIRONMENT`: {{ENVIRONMENT}}
- `REPO_NAME`: {{REPO_NAME}}
EOF

# Create scripts
cat > "$BASE_DIR/scripts/build.sh" << 'EOF'
#!/bin/bash
# Build script for {{SERVICE_NAME}}
# Repository: {{REPO_NAME}}

set -e

echo "Building {{SERVICE_NAME}} for {{ENVIRONMENT}}"
echo "Repository: {{REPO_NAME}}"

# Build server
echo "Building server..."
go build -o bin/server ./cmd/server

# Build client  
echo "Building client..."
go build -o bin/client ./cmd/client

echo "Build complete!"
EOF

cat > "$BASE_DIR/scripts/deploy.sh" << 'EOF'
#!/bin/bash
# Deploy script for {{SERVICE_NAME}}
# Repository: {{REPO_NAME}}
# Environment: {{ENVIRONMENT}}

set -e

echo "Deploying {{SERVICE_NAME}} to {{ENVIRONMENT}}"
echo "Repository: {{REPO_NAME}}"

# Run deployment based on environment
case "{{ENVIRONMENT}}" in
    "development")
        echo "Deploying to development environment"
        docker-compose -f deployments/docker/docker-compose.dev.yml up -d
        ;;
    "staging")
        echo "Deploying to staging environment"
        kubectl apply -f deployments/k8s/staging/
        ;;
    "production")
        echo "Deploying to production environment"
        kubectl apply -f deployments/k8s/production/
        ;;
    *)
        echo "Unknown environment: {{ENVIRONMENT}}"
        exit 1
        ;;
esac

echo "Deployment complete!"
EOF

# Create deployment files
cat > "$BASE_DIR/deployments/docker/Dockerfile" << 'EOF'
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/server .

EXPOSE 8080
CMD ["./server"]
EOF

cat > "$BASE_DIR/deployments/docker/docker-compose.dev.yml" << 'EOF'
version: "3.8"

services:
  "{{SERVICE_NAME}}":
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile
    container_name: "{{SERVICE_NAME}}-dev"
    ports:
      - "8080:8080"
    environment:
      - "SERVICE_NAME={{SERVICE_NAME}}"
      - "ENVIRONMENT={{ENVIRONMENT}}"
      - "REPO_NAME={{REPO_NAME}}"
      - "DEBUG=true"
    restart: unless-stopped
EOF

cat > "$BASE_DIR/deployments/k8s/deployment.yaml" << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "{{SERVICE_NAME}}"
  labels:
    app: "{{SERVICE_NAME}}"
    environment: "{{ENVIRONMENT}}"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: "{{SERVICE_NAME}}"
  template:
    metadata:
      labels:
        app: "{{SERVICE_NAME}}"
    spec:
      containers:
      - name: "{{SERVICE_NAME}}"
        image: "{{REPO_NAME}}:latest"
        ports:
        - containerPort: 8080
        env:
        - name: SERVICE_NAME
          value: "{{SERVICE_NAME}}"
        - name: ENVIRONMENT
          value: "{{ENVIRONMENT}}"
        - name: REPO_NAME
          value: "{{REPO_NAME}}"
        - name: PORT
          value: "8080"
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
EOF

cat > "$BASE_DIR/deployments/k8s/service.yaml" << 'EOF'
apiVersion: v1
kind: Service
metadata:
  name: "{{SERVICE_NAME}}-service"
  labels:
    app: "{{SERVICE_NAME}}"
    environment: "{{ENVIRONMENT}}"
spec:
  selector:
    app: "{{SERVICE_NAME}}"
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
EOF

# Make scripts executable
chmod +x "$BASE_DIR/scripts/build.sh"
chmod +x "$BASE_DIR/scripts/deploy.sh"

echo "✓ Created GitHub-like Go project structure fixture"