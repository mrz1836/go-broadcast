#!/bin/bash
# Creates large directory fixture with 1000+ files in nested subdirectories
# Based on existing large fixture structure with 10 main dirs, each with 10 subdirs, each with 10 files

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$SCRIPT_DIR/directories/large"

echo "Creating large directory fixture..."

# Clean and create base directory
rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# Create 10 main directories (dir_0 to dir_9)
for i in {0..9}; do
    main_dir="$BASE_DIR/dir_$i"
    mkdir -p "$main_dir"

    # Create 10 subdirectories in each main directory (subdir_0 to subdir_9)
    for j in {0..9}; do
        sub_dir="$main_dir/subdir_$j"
        mkdir -p "$sub_dir"

        # Create 10 files in each subdirectory
        for k in {1..10}; do
            file_num=$((i * 100 + j * 10 + k))
            file_path="$sub_dir/file_$file_num.txt"

            cat > "$file_path" << EOF
# {{SERVICE_NAME}} Large Test File $file_num
# Directory Structure: dir_$i/subdir_$j/file_$file_num.txt
# Repository: {{REPO_NAME}}
# Environment: {{ENVIRONMENT}}

Large scale directory sync test file number $file_num.
Located in directory structure: dir_$i -> subdir_$j

This file is part of a large fixture designed to test:
- Deep directory nesting
- Large number of files
- Performance under scale
- Bulk synchronization operations

File Metadata:
- File Number: $file_num
- Main Directory: $i
- Sub Directory: $j
- File Index: $k
- Service Name: {{SERVICE_NAME}}
- Repository: {{REPO_NAME}}
- Environment: {{ENVIRONMENT}}
- Generated: $(date)
- Path: dir_$i/subdir_$j/file_$file_num.txt

Template Variables Test:
SERVICE_NAME={{SERVICE_NAME}}
REPO_NAME={{REPO_NAME}}
ENVIRONMENT={{ENVIRONMENT}}

Content for stress testing directory synchronization with large file counts.
This fixture helps verify performance and correctness with 1000+ files.
EOF
        done
    done
done

# Count total files created
total_files=$(find "$BASE_DIR" -type f | wc -l)
echo "✓ Created large directory fixture: $total_files files in nested structure (10 dirs × 10 subdirs × 10 files)"
