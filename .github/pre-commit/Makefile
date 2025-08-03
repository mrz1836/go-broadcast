# GoFortress Pre-commit System Module Makefile
#
# Note: For common Go operations (lint, test, fumpt, mod-tidy, benchmarks), 
# use the root repository Makefile which provides standardized targets for all modules:
#   cd ../.. && make lint-all-modules
#   cd ../.. && make test-all-modules
#   cd ../.. && make bench
#
# This Makefile focuses on module-specific operations.

# Variables
BINARY_NAME := gofortress-pre-commit
BINARY_PATH := ./cmd/$(BINARY_NAME)/$(BINARY_NAME)
GO := go
MODULE := github.com/mrz1836/go-broadcast/pre-commit

# Build flags
LDFLAGS := -ldflags="-s -w"
BUILD_FLAGS := -trimpath

.PHONY: all build clean install uninstall help dev-install dev-test version

# Default target
all: build

## help: Display this help message
help:
	@echo "GoFortress Pre-commit System (Module-specific targets)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Module-specific targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
	@echo ""
	@echo "For common Go operations, use root repository Makefile:"
	@echo "  cd ../.. && make lint-all-modules    # Lint all modules"
	@echo "  cd ../.. && make test-all-modules    # Test all modules"
	@echo "  cd ../.. && make bench               # Run benchmarks"
	@echo "  cd ../.. && make fumpt               # Format code"

## build: Build the pre-commit binary
build:
	@echo "Building $(BINARY_NAME)..."
	@$(GO) build $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/$(BINARY_NAME)
	@echo "Binary built: $(BINARY_PATH)"

## clean: Clean build artifacts
clean:
	@echo "Cleaning $(BINARY_NAME) artifacts..."
	@rm -f $(BINARY_PATH)
	@$(GO) clean ./...
	@echo "Clean complete"

## install: Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@$(GO) install ./cmd/$(BINARY_NAME)
	@echo "Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

## uninstall: Remove the binary from GOPATH/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $$(go env GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME)"

# Development helpers
## dev-install: Quick build and install for development
dev-install:
	@$(GO) build -o $(BINARY_PATH) ./cmd/$(BINARY_NAME) && \
		cp $(BINARY_PATH) $$(go env GOPATH)/bin/ && \
		echo "Development build installed"

## dev-test: Run tests for a specific package
dev-test:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make dev-test PKG=./internal/config [TEST=TestFunctionName]"; \
	else \
		echo "Running tests for $(PKG)..."; \
		$(GO) test -v $(if $(TEST),-run $(TEST)) $(PKG); \
	fi

## version: Display version information
version:
	@echo "Module: $(MODULE)"
	@echo "Binary: $(BINARY_NAME)"
	@echo "Go version: $$(go version)"
	@echo "Build time: $$(date)"