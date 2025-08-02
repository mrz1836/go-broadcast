#!/bin/bash
# Creates complex directory fixture with special characters, unicode, nested structures
# Based on existing complex fixture structure

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$SCRIPT_DIR/directories/complex"

echo "Creating complex directory fixture..."

# Clean and create base directory
rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# Create directory with special characters and spaces
mkdir -p "$BASE_DIR/special chars dir"
cat > "$BASE_DIR/special chars dir/file with spaces.txt" << 'EOF'
# {{SERVICE_NAME}} Special Characters Test
# File with spaces in both directory and filename
# Repository: {{REPO_NAME}}
# Environment: {{ENVIRONMENT}}

This file tests handling of:
- Directories with spaces
- Filenames with spaces  
- Special character encoding

Service: {{SERVICE_NAME}}
Repo: {{REPO_NAME}}
Env: {{ENVIRONMENT}}

Special characters test: !@#$%^&*()_+-=[]{}|;:,.<>?
EOF

# Create symbols directory with various special characters
mkdir -p "$BASE_DIR/symbols"
for symbol in "!" "@" "#" "$" "%" "^" "&" "*" "(" ")" "_" "+" "-" "=" "[" "]" "{" "}" "|" ";" ":" "," "." "<" ">" "?"; do
    # Create files with symbol names (escaped for filesystem)
    case "$symbol" in
        "/" | "\\") continue ;;  # Skip filesystem separators
        "<" | ">" | "|" | "?" | "*" | ":" | "\"") 
            # These are problematic on some filesystems, use safe alternatives
            safe_name=$(printf "%s" "$symbol" | xxd -p)
            echo "Symbol file: $symbol" > "$BASE_DIR/symbols/symbol_${safe_name}.txt"
            ;;
        *)
            echo "Symbol file: $symbol" > "$BASE_DIR/symbols/symbol_${symbol}.txt" 2>/dev/null || true
            ;;
    esac
done

# Create unicode directory and files
mkdir -p "$BASE_DIR/unicode_测试"
cat > "$BASE_DIR/unicode_测试/unicode_file_测试.txt" << 'EOF'
# {{SERVICE_NAME}} Unicode Test File
# 测试中文字符处理
# Repository: {{REPO_NAME}}
# Environment: {{ENVIRONMENT}}

Unicode character testing:
- 中文: 测试文件
- 日本語: テストファイル  
- Español: archivo de prueba
- Français: fichier de test
- Deutsch: Testdatei
- Русский: тестовый файл
- العربية: ملف اختبار
- עברית: קובץ בדיקה

Service: {{SERVICE_NAME}}
Repository: {{REPO_NAME}}
Environment: {{ENVIRONMENT}}

Emoji test: 🚀 🎯 ✅ ❌ 🔧 📁 📄 💻
EOF

# Create nested structure with mixed content
mkdir -p "$BASE_DIR/nested/level1/level2/level3"
cat > "$BASE_DIR/nested/level1/level2/level3/deep_file.txt" << 'EOF'
# {{SERVICE_NAME}} Deep Nested File
# Path: complex/nested/level1/level2/level3/deep_file.txt
# Repository: {{REPO_NAME}}
# Environment: {{ENVIRONMENT}}

This file tests deep directory nesting and path handling.

Nested level: 3
Full path: nested/level1/level2/level3/deep_file.txt

Service Configuration:
- Name: {{SERVICE_NAME}}
- Repository: {{REPO_NAME}}  
- Environment: {{ENVIRONMENT}}

Testing deep path synchronization and template replacement at depth.
EOF

# Create binary-like files (fake images)
echo -e '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\tpHYs\x00\x00\x0b\x13\x00\x00\x0b\x13\x01\x00\x9a\x9c\x18\x00\x00\x00\nIDATx\x9cc\xf8\x00\x00\x00\x01\x00\x01,\x1e\x2c\x08\x00\x00\x00\x00IEND\xaeB`\x82' > "$BASE_DIR/fake_image.png"

# Create fake JPEG header
printf '\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00\xff\xdb\x00C\x00\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t\x08\n\x0c\x14\r\x0c\x0b\x0b\x0c\x19\x12\x13\x0f\x14\x1d\x1a\x1f\x1e\x1d\x1a\x1c\x1c $." ",#\x1c\x1c(7),01444\x1f"9=82<.342\xff\xc0\x00\x11\x08\x00\x01\x00\x01\x01\x01\x11\x00\x02\x11\x01\x03\x11\x01\xff\xc4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\xff\xc4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xff\xda\x00\x08\x01\x01\x00\x00?\x00\x00\xff\xd9' > "$BASE_DIR/fake_jpeg.jpg"

# Create various edge case directories and files
mkdir -p "$BASE_DIR/edge_cases"

# Create file with very long name (truncated to reasonable length)
long_name="very_long_filename_that_tests_path_length_limits_and_edge_cases_for_sync_operations_$(date +%s)"
echo "Long filename test" > "$BASE_DIR/edge_cases/${long_name}.txt"

# Create hidden files (dot files)
echo "Hidden file content" > "$BASE_DIR/edge_cases/.hidden_file"
echo "Another hidden file" > "$BASE_DIR/edge_cases/.another_hidden"

# Create files with numbers and mixed cases
for i in {1..5}; do
    echo "Numbered file $i content" > "$BASE_DIR/edge_cases/File_${i}_MixedCase.txt"
done

echo "✓ Created complex directory fixture with special characters, unicode, and nested structures"