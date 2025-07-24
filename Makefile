# Common makefile commands & variables between projects
include .make/common.mk

# Common Golang makefile commands & variables between projects
include .make/go.mk

## Set default repository details if not provided
REPO_NAME  ?= go-broadcast
REPO_OWNER ?= mrz1836

# Custom functions for this project

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
