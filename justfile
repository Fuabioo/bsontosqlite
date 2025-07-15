# bsontosqlite justfile
#
# Common development tasks for the bsontosqlite project.
#
# Usage examples:
#   just build                    # Build cross-platform static binaries
#   just up --help               # Run application with go run
#   just up version              # Show version using go run
#   just dev                     # Full dev cycle: format, tidy, test, build
#   just check-static            # Verify Linux binary is statically linked
#
# For BSON conversion:
#   just up --bson data.bson --metadata meta.json --output out.db

# Default recipe to display available commands
default:
    @just --list

# Build using goreleaser for all platforms
build:
    goreleaser release --clean --snapshot

# Run the application directly with go run
up *args:
    go run . {{args}}

# Clean build artifacts
clean:
    rm -rf dist/
    go clean

# Run tests
test:
    go test ./...

# Run tests with verbose output
test-verbose:
    go test -v ./...

# Format code
fmt:
    go fmt ./...

# Run linter
lint:
    golangci-lint run

# Tidy go modules
tidy:
    go mod tidy

# Install dependencies
deps:
    go mod download

# Show version
version:
    go run . version

# Check if binary is static (Linux only)
check-static: build
    @echo "Checking if Linux binary is static..."
    @if ldd ./dist/bsontosqlite_linux_amd64_v1/bsontosqlite 2>&1 | grep -q "not a dynamic executable"; then \
        echo "✓ Binary is statically linked"; \
    else \
        echo "✗ Binary is not statically linked"; \
        ldd ./dist/bsontosqlite_linux_amd64_v1/bsontosqlite; \
    fi

# Run a full development cycle: format, tidy, test, build
dev: fmt tidy test build

# Show help for the application
help:
    go run . --help

# Run with example files (if they exist)
dev-run:
    #!/usr/bin/env bash
    if [ -f "examples/sample.bson" ] && [ -f "examples/metadata.json" ]; then \
        echo "Running with example files..."; \
        just up --bson examples/sample.bson --metadata examples/metadata.json --output examples/output.db -v; \
    else \
        echo "Example files not found. Create examples/sample.bson and examples/metadata.json to use this command."; \
        echo "Usage: just up --bson <bson_file> --metadata <metadata_file> --output <output_file>"; \
    fi
