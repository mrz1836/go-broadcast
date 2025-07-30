# Common makefile commands & variables between projects
include .make/common.mk

# Common Golang makefile commands & variables between projects
include .make/go.mk

## Set default repository details if not provided
REPO_NAME  ?= go-broadcast
REPO_OWNER ?= mrz1836

# Custom functions for this project

.PHONY: rebuild
rebuild: ## Clean and rebuild the project
	@echo "Cleaning build artifacts..."
	@go clean -i ./...
	@echo "Rebuilding project..."
	@go install ./cmd/go-broadcast

.PHONY: test-integration-complex
test-integration-complex: ## Run complex integration test scenarios (Phase 1)
	@echo "Running complex integration tests..."
	@go test -v ./test/integration -run "TestComplexSyncScenarios" \
		-timeout=10m \
		$(if $(VERBOSE),-v) \
		$(TAGS)

.PHONY: test-integration-advanced
test-integration-advanced: ## Run advanced workflow integration tests (Phase 2)
	@echo "Running advanced workflow integration tests..."
	@go test -v ./test/integration -run "TestAdvancedWorkflows" \
		-timeout=10m \
		$(if $(VERBOSE),-v) \
		$(TAGS)

.PHONY: test-integration-network
test-integration-network: ## Run network edge case integration tests (Phase 3)
	@echo "Running network edge case integration tests..."
	@go test -v ./test/integration -run "TestNetworkEdgeCases" \
		-timeout=15m \
		$(if $(VERBOSE),-v) \
		$(TAGS)

.PHONY: test-integration-all
test-integration-all: ## Run all integration test scenarios (All Phases)
	@echo "Running all integration test scenarios..."
	@$(MAKE) test-integration-complex
	@$(MAKE) test-integration-advanced
	@$(MAKE) test-integration-network

.PHONY: test-all-modules
test-all-modules: ## Run tests for main module and all submodules
	@echo "Testing main module..."
	@go test ./... \
		$(if $(VERBOSE),-v) \
		$(TAGS)
	@echo ""
	@echo "Finding and testing submodules..."
	@for dir in $$(find . -name go.mod -not -path "./go.mod" -not -path "./vendor/*" | xargs -n1 dirname); do \
		echo "Testing module in $$dir..."; \
		(cd $$dir && go test ./... $(if $(VERBOSE),-v) $(TAGS)) || exit 1; \
	done

.PHONY: test-all-modules-race
test-all-modules-race: ## Run tests for main module and all submodules with race detection
	@echo "Testing main module with race detection..."
	@go test -race ./... \
		$(if $(VERBOSE),-v) \
		$(TAGS)
	@echo ""
	@echo "Finding and testing submodules with race detection..."
	@for dir in $$(find . -name go.mod -not -path "./go.mod" -not -path "./vendor/*" | xargs -n1 dirname); do \
		echo "Testing module in $$dir with race detection..."; \
		(cd $$dir && go test -race ./... $(if $(VERBOSE),-v) $(TAGS)) || exit 1; \
	done

.PHONY: lint-all-modules
lint-all-modules: ## Run lint for main module and all submodules
	@echo "Linting main module..."
	@golangci-lint run --verbose
	@echo ""
	@echo "Finding and linting submodules..."
	@for dir in $$(find . -name go.mod -not -path "./go.mod" -not -path "./vendor/*" | xargs -n1 dirname); do \
		echo "Linting module in $$dir..."; \
		(cd $$dir && golangci-lint run --verbose) || exit 1; \
	done
