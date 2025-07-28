# GoFortress Coverage System

A self-contained, Go-native coverage system built as a complete bolt-on solution for any Go repository.

## Architecture

This coverage system is designed as a portable, self-contained unit that lives entirely within the `.github/coverage/` directory. It can be copied to any repository without polluting the main codebase structure.

### Directory Structure

```
.github/coverage/
├── cmd/
│   └── gofortress-coverage/        # Main CLI tool
│       ├── main.go                 # Entry point
│       ├── go.mod                  # Separate Go module
│       └── cmd/                    # Command implementations
├── internal/                       # Internal packages
│   ├── parser/                     # Coverage parsing logic
│   ├── badge/                      # SVG badge generation
│   ├── report/                     # HTML report generation
│   ├── history/                    # Historical data tracking
│   └── github/                     # GitHub API integration
└── README.md                       # This file
```

## Features

- **Go-Native**: Single binary with no runtime dependencies
- **Bolt-On Architecture**: Complete encapsulation within `.github/coverage/`
- **Portable**: Can be copied to any repository as a complete unit
- **Zero External Dependencies**: No reliance on external services
- **Professional Quality**: GitHub-style badges and clean reports
- **CI/CD Integration**: Seamless integration with GitHub Actions

## Implementation Status

- **Phase 1**: ✅ Foundation & Configuration (Current)
- **Phase 2**: ⏳ Core Coverage Engine
- **Phase 3**: ⏳ Fortress Workflow Integration
- **Phase 4**: ⏳ GitHub Pages & Storage
- **Phase 5**: ⏳ Pull Request Integration

## Usage

The CLI tool will be built and used in GitHub Actions workflows:

```bash
# Parse coverage data
./gofortress-coverage parse --file coverage.out --output coverage.json

# Generate badge
./gofortress-coverage badge --coverage 85.5 --output badge.svg

# Generate report
./gofortress-coverage report --data coverage.json --output report.html

# Update history
./gofortress-coverage history --add coverage.json --branch main --commit abc123

# Create PR comment
./gofortress-coverage comment --pr 123 --coverage coverage.json
```

## Configuration

The system is configured through environment variables in `.github/.env.shared`. See the main repository documentation for complete configuration options.

## Development

This is a separate Go module with its own dependencies. To work on the coverage tool:

```bash
cd .github/coverage/cmd/gofortress-coverage
go mod tidy
go build -o gofortress-coverage
go test ./...
```

## License

This coverage system inherits the license from the parent repository.