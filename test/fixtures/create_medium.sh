#!/bin/bash
# Creates medium directory fixture with 100 files across 10 directories
# Based on existing medium fixture structure

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$SCRIPT_DIR/directories/medium"

echo "Creating medium directory fixture..."

# Clean and create base directory
rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# Create 10 directories with 10 files each (100 total)
for i in {0..9}; do
    dir_path="$BASE_DIR/dir_$i"
    mkdir -p "$dir_path"

    # Create 10 files in each directory
    for j in {1..10}; do
        file_num=$((i * 10 + j))
        file_path="$dir_path/file_$file_num.txt"

        cat > "$file_path" << EOF
# {{SERVICE_NAME}} File $file_num
# Generated for testing directory sync functionality
# Directory: dir_$i
# Repository: {{REPO_NAME}}
# Environment: {{ENVIRONMENT}}

This is test file number $file_num in directory $i.
Used for testing medium-scale directory synchronization.

Metadata:
- File ID: $file_num
- Directory: $i
- Service: {{SERVICE_NAME}}
- Repository: {{REPO_NAME}}
- Environment: {{ENVIRONMENT}}
- Generated: $(date)

Content for testing transforms and synchronization.
This file contains template variables that should be replaced during sync operations.
EOF
    done
done

echo "✓ Created medium directory fixture: 100 files in 10 directories"
