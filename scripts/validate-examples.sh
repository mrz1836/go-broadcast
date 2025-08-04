#!/bin/bash

# validate-examples.sh
# Validation script for all go-broadcast example configurations
# This script validates all example configurations and tests documented commands

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counter for tracking results
TOTAL_EXAMPLES=0
VALID_EXAMPLES=0
INVALID_EXAMPLES=0

print_header() {
    echo -e "${BLUE}===============================================${NC}"
    echo -e "${BLUE}  go-broadcast Example Configuration Validation${NC}"
    echo -e "${BLUE}===============================================${NC}"
    echo ""
}

print_section() {
    echo -e "${YELLOW}--- $1 ---${NC}"
}

validate_config() {
    local config_file="$1"
    local description="$2"

    echo -e "${BLUE}Validating: ${config_file}${NC}"
    echo -e "Description: ${description}"

    TOTAL_EXAMPLES=$((TOTAL_EXAMPLES + 1))

    if ./go-broadcast validate --config "$config_file"; then
        echo -e "${GREEN}✅ VALID: ${config_file}${NC}"
        VALID_EXAMPLES=$((VALID_EXAMPLES + 1))
    else
        echo -e "${RED}❌ INVALID: ${config_file}${NC}"
        INVALID_EXAMPLES=$((INVALID_EXAMPLES + 1))
    fi
    echo ""
}

test_command() {
    local command="$1"
    local description="$2"

    echo -e "${BLUE}Testing: ${command}${NC}"
    echo -e "Description: ${description}"

    if eval "$command" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ COMMAND WORKS: ${command}${NC}"
    else
        echo -e "${RED}❌ COMMAND FAILED: ${command}${NC}"
    fi
    echo ""
}

print_summary() {
    echo -e "${BLUE}===============================================${NC}"
    echo -e "${BLUE}  VALIDATION SUMMARY${NC}"
    echo -e "${BLUE}===============================================${NC}"
    echo -e "Total examples tested: ${TOTAL_EXAMPLES}"
    echo -e "${GREEN}Valid configurations: ${VALID_EXAMPLES}${NC}"
    if [ $INVALID_EXAMPLES -gt 0 ]; then
        echo -e "${RED}Invalid configurations: ${INVALID_EXAMPLES}${NC}"
    else
        echo -e "${GREEN}Invalid configurations: ${INVALID_EXAMPLES}${NC}"
    fi
    echo ""

    if [ $INVALID_EXAMPLES -eq 0 ]; then
        echo -e "${GREEN}🎉 ALL EXAMPLES VALID!${NC}"
        return 0
    else
        echo -e "${RED}❌ Some examples failed validation${NC}"
        return 1
    fi
}

main() {
    print_header

    # Check if go-broadcast binary exists
    if [ ! -f "./go-broadcast" ]; then
        echo -e "${RED}Error: go-broadcast binary not found. Please build it first:${NC}"
        echo "  make build-go"
        exit 1
    fi

    # Validate existing file-only examples
    print_section "File Sync Examples"
    validate_config "examples/minimal.yaml" "Minimal configuration for simple file sync"
    validate_config "examples/sync.yaml" "Complete example with all features"
    validate_config "examples/microservices.yaml" "Microservices architecture sync"
    validate_config "examples/multi-language.yaml" "Multi-language project sync"
    validate_config "examples/ci-cd-only.yaml" "CI/CD pipeline synchronization"
    validate_config "examples/documentation.yaml" "Documentation template sync"

    # Validate directory sync examples
    print_section "Directory Sync Examples"
    validate_config "examples/directory-sync.yaml" "Comprehensive directory sync examples"
    validate_config "examples/github-workflows.yaml" "GitHub infrastructure sync"
    validate_config "examples/large-directories.yaml" "Large directory management"
    validate_config "examples/exclusion-patterns.yaml" "Exclusion pattern showcase"
    validate_config "examples/github-complete.yaml" "Complete GitHub directory sync"

    # Test documented commands
    print_section "Command Testing"
    test_command "./go-broadcast --version" "Version command"
    test_command "./go-broadcast --help" "Help command"
    test_command "./go-broadcast validate --help" "Validate help command"
    test_command "./go-broadcast sync --help" "Sync help command"
    test_command "./go-broadcast status --help" "Status help command"
    test_command "./go-broadcast diagnose --help" "Diagnose help command"
    test_command "./go-broadcast cancel --help" "Cancel help command"

    # Test dry-run mode with valid configuration
    print_section "Dry-Run Testing"
    echo -e "${BLUE}Testing dry-run mode with minimal configuration...${NC}"
    if ./go-broadcast sync --dry-run --config examples/minimal.yaml 2>/dev/null; then
        echo -e "${GREEN}✅ Dry-run mode works correctly${NC}"
    else
        echo -e "${YELLOW}⚠️  Dry-run requires valid repository access (expected)${NC}"
    fi
    echo ""

    print_summary
}

# Usage information
usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -v, --verbose  Enable verbose output"
    echo ""
    echo "This script validates all example configurations in the examples/ directory"
    echo "and tests documented commands to ensure they work correctly."
    echo ""
    echo "Prerequisites:"
    echo "  - go-broadcast binary must be built (run: make build-go)"
    echo "  - All example files must exist in examples/ directory"
    echo ""
    echo "Examples:"
    echo "  $0                    # Validate all examples"
    echo "  $0 --verbose          # Validate with verbose output"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--verbose)
            set -x  # Enable verbose mode
            shift
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Run main function
main
