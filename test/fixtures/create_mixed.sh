#!/bin/bash
# Creates mixed binary/text files fixture for testing various file types
# Based on existing mixed fixture structure

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$SCRIPT_DIR/directories/mixed"

echo "Creating mixed binary/text files fixture..."

# Clean and create base directory
rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# Create text files with templates
for i in {1..5}; do
    cat > "$BASE_DIR/text_file_$i.txt" << EOF
# {{SERVICE_NAME}} Mixed Content Test File $i
# Repository: {{REPO_NAME}}
# Environment: {{ENVIRONMENT}}

This is text file number $i in the mixed content fixture.
Used for testing mixed file type synchronization.

File Properties:
- Type: Text
- Number: $i
- Service: {{SERVICE_NAME}}
- Repository: {{REPO_NAME}}
- Environment: {{ENVIRONMENT}}
- Content-Type: text/plain
- Generated: $(date)

Template Variables:
SERVICE_NAME={{SERVICE_NAME}}
REPO_NAME={{REPO_NAME}}
ENVIRONMENT={{ENVIRONMENT}}

This file contains both static content and template variables
that should be replaced during directory sync operations.
Ensuring text files are handled correctly alongside binary files.
EOF
done

# Create a PNG image file (minimal valid PNG)
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\nIDATx\x9cc\xf8\x00\x00\x00\x01\x00\x01,\x1e\x2c\x08\x00\x00\x00\x00IEND\xaeB`\x82' > "$BASE_DIR/image.png"

# Create a JPEG image file (minimal valid JPEG)
printf '\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x01\x00H\x00H\x00\x00\xff\xdb\x00C\x00\x08\x06\x06\x07\x06\x05\x08\x07\x07\x07\t\t\x08\n\x0c\x14\r\x0c\x0b\x0b\x0c\x19\x12\x13\x0f\x14\x1d\x1a\x1f\x1e\x1d\x1a\x1c\x1c $.\'"'"' ",#\x1c\x1c(7),01444\x1f\'"'"'9=82<.342\xff\xc0\x00\x11\x08\x00\x01\x00\x01\x01\x01\x11\x00\x02\x11\x01\x03\x11\x01\xff\xc4\x00\x14\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\xff\xc4\x00\x14\x10\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xff\xda\x00\x08\x01\x01\x00\x00?\x00\x00\xff\xd9' > "$BASE_DIR/photo.jpg"

# Create a shared library file (binary)
printf '\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00>\x00\x01\x00\x00\x00\x00\x10\x40\x00\x00\x00\x00\x00@\x00\x00\x00\x00\x00\x00\x00' > "$BASE_DIR/binary.so"

# Create a gzipped file
echo "This is compressed content for testing gzip handling in mixed file scenarios" | gzip > "$BASE_DIR/compressed.gz" 2>/dev/null || {
    # Fallback if gzip is not available
    printf '\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\x03' > "$BASE_DIR/compressed.gz"
    echo "Compressed content placeholder" >> "$BASE_DIR/compressed.gz"
}

# Create a ZIP archive (minimal)
# ZIP file header
printf 'PK\x03\x04\x14\x00\x00\x00\x08\x00' > "$BASE_DIR/archive.zip"
# Add timestamp (dummy)
printf '\x00\x00\x00\x00' >> "$BASE_DIR/archive.zip"
# CRC32, compressed size, uncompressed size (dummy values)
printf '\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00' >> "$BASE_DIR/archive.zip"
# Filename length (5 bytes for "test")
printf '\x04\x00' >> "$BASE_DIR/archive.zip"
# Extra field length
printf '\x00\x00' >> "$BASE_DIR/archive.zip"
# Filename
printf 'test' >> "$BASE_DIR/archive.zip"
# Central directory (end of file)
printf 'PK\x01\x02\x14\x00\x14\x00\x00\x00\x08\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x80\x01\x00\x00\x00\x00test' >> "$BASE_DIR/archive.zip"
# End of central directory
printf 'PK\x05\x06\x00\x00\x00\x00\x01\x00\x01\x00\x32\x00\x00\x00\x3C\x00\x00\x00\x00\x00' >> "$BASE_DIR/archive.zip"

# Create executable files with different content types
# Script file
cat > "$BASE_DIR/script.sh" << 'EOF'
#!/bin/bash
# {{SERVICE_NAME}} Test Script
# Repository: {{REPO_NAME}}
# Environment: {{ENVIRONMENT}}

echo "Running test script for {{SERVICE_NAME}}"
echo "Repository: {{REPO_NAME}}"
echo "Environment: {{ENVIRONMENT}}"

# This script is part of the mixed fixture testing
# It should be executable and contain template variables
EOF

# Make the script executable
chmod +x "$BASE_DIR/script.sh"

# Create configuration files of different types
cat > "$BASE_DIR/config.json" << 'EOF'
{
  "service": "{{SERVICE_NAME}}",
  "repository": "{{REPO_NAME}}",
  "environment": "{{ENVIRONMENT}}",
  "settings": {
    "debug": false,
    "port": 8080,
    "timeout": 30
  },
  "features": [
    "sync",
    "transform",
    "binary-handling",
    "mixed-content"
  ],
  "metadata": {
    "generated": "2024-01-01T00:00:00Z",
    "version": "1.0.0"
  }
}
EOF

cat > "$BASE_DIR/config.yaml" << 'EOF'
service: "{{SERVICE_NAME}}"
repository: "{{REPO_NAME}}"
environment: "{{ENVIRONMENT}}"

server:
  port: 8080
  host: "0.0.0.0"
  timeout: 30s

logging:
  level: info
  format: json

features:
  - sync
  - transform  
  - binary-handling
  - mixed-content

database:
  host: localhost
  port: 5432
  name: "{{SERVICE_NAME}}_db"

metadata:
  generated: "2024-01-01T00:00:00Z"
  version: "1.0.0"
EOF

cat > "$BASE_DIR/config.xml" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<configuration>
    <service>{{SERVICE_NAME}}</service>
    <repository>{{REPO_NAME}}</repository>
    <environment>{{ENVIRONMENT}}</environment>
    
    <server>
        <port>8080</port>
        <host>0.0.0.0</host>
        <timeout>30</timeout>
    </server>
    
    <logging>
        <level>info</level>
        <format>json</format>
    </logging>
    
    <features>
        <feature>sync</feature>
        <feature>transform</feature>
        <feature>binary-handling</feature>
        <feature>mixed-content</feature>
    </features>
    
    <metadata>
        <generated>2024-01-01T00:00:00Z</generated>
        <version>1.0.0</version>
    </metadata>
</configuration>
EOF

# Create data files
cat > "$BASE_DIR/data.csv" << 'EOF'
service,repository,environment,type,size
{{SERVICE_NAME}},{{REPO_NAME}},{{ENVIRONMENT}},text,small
{{SERVICE_NAME}},{{REPO_NAME}},{{ENVIRONMENT}},binary,medium
{{SERVICE_NAME}},{{REPO_NAME}},{{ENVIRONMENT}},compressed,small
{{SERVICE_NAME}},{{REPO_NAME}},{{ENVIRONMENT}},archive,large
{{SERVICE_NAME}},{{REPO_NAME}},{{ENVIRONMENT}},image,medium
EOF

# Create a log file with mixed content
cat > "$BASE_DIR/application.log" << 'EOF'
2024-01-01 00:00:00 INFO Starting {{SERVICE_NAME}}
2024-01-01 00:00:01 INFO Repository: {{REPO_NAME}}
2024-01-01 00:00:02 INFO Environment: {{ENVIRONMENT}}
2024-01-01 00:00:03 DEBUG Loading configuration
2024-01-01 00:00:04 INFO Mixed content fixture initialized
2024-01-01 00:00:05 WARN Testing binary file handling
2024-01-01 00:00:06 INFO Compressed files supported
2024-01-01 00:00:07 DEBUG Archive processing enabled
2024-01-01 00:00:08 INFO Image file detection active
2024-01-01 00:00:09 INFO {{SERVICE_NAME}} ready for testing
EOF

# Create a README for the mixed fixture
cat > "$BASE_DIR/README.md" << 'EOF'
# {{SERVICE_NAME}} Mixed Content Fixture

This fixture contains various file types for testing:

## File Types Included

### Text Files
- `text_file_*.txt` - Template-enabled text files
- `config.*` - Configuration files (JSON, YAML, XML)
- `data.csv` - CSV data file
- `application.log` - Log file
- `README.md` - This documentation file

### Binary Files
- `image.png` - PNG image file
- `photo.jpg` - JPEG image file  
- `binary.so` - Shared library file
- `archive.zip` - ZIP archive file
- `compressed.gz` - Gzipped file

### Executable Files
- `script.sh` - Bash script with executable permissions

## Template Variables
All text files contain these template variables:
- `{{SERVICE_NAME}}` - Service name
- `{{REPO_NAME}}` - Repository name
- `{{ENVIRONMENT}}` - Environment name

## Testing Purpose
This fixture tests:
- Mixed file type handling
- Binary file preservation
- Template variable replacement in text files only
- File permission preservation
- Various encoding and compression formats

Repository: {{REPO_NAME}}
Environment: {{ENVIRONMENT}}
EOF

# Count files by type
text_files=$(find "$BASE_DIR" -name "*.txt" -o -name "*.md" -o -name "*.json" -o -name "*.yaml" -o -name "*.xml" -o -name "*.csv" -o -name "*.log" -o -name "*.sh" | wc -l)
binary_files=$(find "$BASE_DIR" -name "*.png" -o -name "*.jpg" -o -name "*.so" -o -name "*.gz" -o -name "*.zip" | wc -l)
total_files=$(find "$BASE_DIR" -type f | wc -l)

echo "✓ Created mixed binary/text files fixture: $total_files total files ($text_files text, $binary_files binary)"