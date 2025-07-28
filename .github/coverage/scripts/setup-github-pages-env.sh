#!/bin/bash
# ------------------------------------------------------------------------------------
#  GoFortress Coverage - GitHub Pages Environment Setup Script
#
#  Purpose: Automatically configure GitHub Pages environment settings for a repository
#  to allow deployments from the master branch. This script sets up the environment
#  protection rules that are required for the GoFortress coverage system to deploy
#  coverage reports and badges to GitHub Pages.
#
#  Usage: ./.github/coverage/scripts/setup-github-pages-env.sh [repository]
#  Example: ./.github/coverage/scripts/setup-github-pages-env.sh mrz1836/go-broadcast
#
#  Requirements:
#  - GitHub CLI (gh) installed and authenticated
#  - Repository admin permissions
#  - Personal Access Token with repo scope (for private repos)
#
#  Maintainer: @mrz1836
#
# ------------------------------------------------------------------------------------

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [repository]"
    echo ""
    echo "Arguments:"
    echo "  repository    GitHub repository in format 'owner/repo' (optional if run from repo directory)"
    echo ""
    echo "Examples:"
    echo "  $0                           # Use current repository"
    echo "  $0 mrz1836/go-broadcast      # Specify repository explicitly"
    echo ""
    echo "Requirements:"
    echo "  - GitHub CLI (gh) installed and authenticated"
    echo "  - Repository admin permissions"
    echo "  - Personal Access Token with repo scope (for private repos)"
}

# Function to validate GitHub CLI authentication
check_gh_auth() {
    print_status "Checking GitHub CLI authentication..."
    
    if ! command -v gh &> /dev/null; then
        print_error "GitHub CLI (gh) is not installed. Please install it first:"
        echo "  https://cli.github.com/"
        exit 1
    fi
    
    if ! gh auth status &> /dev/null; then
        print_error "GitHub CLI is not authenticated. Please run:"
        echo "  gh auth login"
        exit 1
    fi
    
    print_success "GitHub CLI is properly authenticated"
}

# Function to determine repository
get_repository() {
    local repo="$1"
    
    if [[ -z "$repo" ]]; then
        # Try to get repository from current directory
        if git remote get-url origin &> /dev/null; then
            repo=$(git remote get-url origin | sed -n 's#.*github\.com[:/]\([^/]*\)/\([^/]*\)\.git.*#\1/\2#p')
            if [[ -z "$repo" ]]; then
                print_error "Could not determine repository from current directory"
                echo "Please specify repository explicitly or run from a Git repository"
                show_usage
                exit 1
            fi
            print_status "Detected repository: $repo"
        else
            print_error "No repository specified and not in a Git repository"
            show_usage
            exit 1
        fi
    fi
    
    echo "$repo"
}

# Function to check if repository exists and is accessible
check_repository_access() {
    local repo="$1"
    
    print_status "Checking repository access: $repo"
    
    if ! gh repo view "$repo" &> /dev/null; then
        print_error "Cannot access repository '$repo'"
        echo "Please check:"
        echo "  - Repository exists and is spelled correctly"
        echo "  - You have access to the repository"
        echo "  - Your GitHub CLI authentication has the required permissions"
        exit 1
    fi
    
    print_success "Repository access confirmed"
}

# Function to create or update GitHub Pages environment
setup_pages_environment() {
    local repo="$1"
    
    print_status "Setting up GitHub Pages environment for $repo..."
    
    # Create or update the github-pages environment
    print_status "Creating/updating github-pages environment..."
    
    if gh api "repos/$repo/environments/github-pages" --method PUT \
        --field deployment_branch_policy='{"protected_branches":false,"custom_branch_policies":true}' \
        --silent; then
        print_success "GitHub Pages environment configured"
    else
        print_error "Failed to create/update github-pages environment"
        echo "This might be because:"
        echo "  - You don't have admin permissions to the repository"
        echo "  - The repository doesn't have GitHub Pages enabled"
        echo "  - Your token doesn't have sufficient permissions"
        exit 1
    fi
}

# Function to add deployment branch rules
setup_deployment_branches() {
    local repo="$1"
    
    print_status "Configuring deployment branch rules..."
    
    # Add master branch deployment rule
    print_status "Adding master branch deployment rule..."
    
    if gh api "repos/$repo/environments/github-pages/deployment-branch-policies" --method POST \
        --field name="master" \
        --field type="branch" \
        --silent 2>/dev/null; then
        print_success "Master branch deployment rule added"
    else
        print_warning "Master branch rule may already exist or failed to add"
    fi
    
    # Add gh-pages branch deployment rule (common for GitHub Pages)
    print_status "Adding gh-pages branch deployment rule..."
    
    if gh api "repos/$repo/environments/github-pages/deployment-branch-policies" --method POST \
        --field name="gh-pages" \
        --field type="branch" \
        --silent 2>/dev/null; then
        print_success "gh-pages branch deployment rule added"
    else
        print_warning "gh-pages branch rule may already exist or failed to add"
    fi
    
    # Add dependabot/* branch pattern deployment rule
    print_status "Adding dependabot/* branch pattern deployment rule..."
    
    if gh api "repos/$repo/environments/github-pages/deployment-branch-policies" --method POST \
        --field name="dependabot/*" \
        --field type="branch" \
        --silent 2>/dev/null; then
        print_success "dependabot/* branch pattern deployment rule added"
    else
        print_warning "dependabot/* branch pattern rule may already exist or failed to add"
    fi
}

# Function to verify the setup
verify_setup() {
    local repo="$1"
    
    print_status "Verifying GitHub Pages environment configuration..."
    
    # Get environment details
    if env_details=$(gh api "repos/$repo/environments/github-pages" 2>/dev/null); then
        print_success "GitHub Pages environment exists"
        
        # Check deployment branch policies
        if policies=$(gh api "repos/$repo/environments/github-pages/deployment-branch-policies" 2>/dev/null); then
            policy_count=$(echo "$policies" | jq '.branch_policies | length' 2>/dev/null || echo "0")
            print_success "Found $policy_count deployment branch policies"
            
            if [[ "$policy_count" -gt 0 ]]; then
                print_status "Configured branches:"
                echo "$policies" | jq -r '.branch_policies[] | "  - " + .name' 2>/dev/null || echo "  (Unable to parse branch names)"
            fi
        else
            print_warning "Could not retrieve deployment branch policies"
        fi
    else
        print_error "GitHub Pages environment verification failed"
        exit 1
    fi
}

# Function to show next steps
show_next_steps() {
    local repo="$1"
    
    echo ""
    print_success "GitHub Pages environment setup completed successfully!"
    echo ""
    echo "Next steps:"
    echo "  1. Your repository is now configured to deploy to GitHub Pages from:"
    echo "     - master branch (main deployments)"
    echo "     - gh-pages branch (GitHub Pages default)"
    echo "     - dependabot/* branches (automated dependency updates)"
    echo "  2. The GoFortress coverage workflow should now deploy successfully"
    echo "  3. Coverage reports will be available at: https://$(echo "$repo" | cut -d'/' -f1).github.io/$(echo "$repo" | cut -d'/' -f2)/"
    echo ""
    echo "To test the setup:"
    echo "  1. Push a commit to the master branch with coverage data"
    echo "  2. Check the 'Process Coverage' workflow in GitHub Actions"
    echo "  3. Verify deployment in the 'Environments' section of your repository settings"
    echo ""
    print_status "If you encounter issues, check the repository's Environments settings manually:"
    echo "  https://github.com/$repo/settings/environments"
}

# Main function
main() {
    local repo_arg="${1:-}"
    
    echo "🏰 GoFortress Coverage - GitHub Pages Environment Setup"
    echo "======================================================="
    echo ""
    
    # Check prerequisites
    check_gh_auth
    
    # Determine repository
    local repo
    repo=$(get_repository "$repo_arg")
    
    # Check repository access
    check_repository_access "$repo"
    
    # Setup environment
    setup_pages_environment "$repo"
    
    # Setup deployment branches
    setup_deployment_branches "$repo"
    
    # Verify setup
    verify_setup "$repo"
    
    # Show next steps
    show_next_steps "$repo"
}

# Handle help flag
if [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]]; then
    show_usage
    exit 0
fi

# Run main function
main "$@"