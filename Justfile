_:
    @just help

# List available commands
help:
    @just --list

# Run all tests
test:
    go test ./...

# Run tests and display coverage
coverage:
    #!/usr/bin/env sh
    set -e
    coverage_file=$(mktemp)
    trap 'rm -f "$coverage_file"' EXIT
    go test -coverprofile="$coverage_file" ./...
    go tool cover -func="$coverage_file"

# Update Go module files
tidy:
    go mod tidy

# Check Go module files without modifying them
mod-check:
    go mod tidy -diff

# Format Go source files
format:
    golangci-lint fmt

# Apply automatic fixes and format code
fix:
    golangci-lint run --fix
    golangci-lint fmt

# Run golangci-lint without --fix
lint:
    golangci-lint run

# Run all non-mutating quality checks
check: mod-check test lint
    golangci-lint fmt --diff

# Build the binary
build:
    go build -o urfave-help .

# Install the binary
install:
    go install .

# Remove build artifacts
clean:
    rm -rf urfave-help urfave-help.exe dist/

# Build with version info (local dev)
build-dev:
    #!/usr/bin/env sh
    set -e
    commit=$(git rev-parse --short HEAD 2>/dev/null || echo none)
    build_date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    go build -ldflags "-X github.com/zigai/urfave-help/internal/cli.version=dev -X github.com/zigai/urfave-help/internal/cli.commit=${commit} -X github.com/zigai/urfave-help/internal/cli.date=${build_date}" -o urfave-help .


alias cov := coverage
alias fmt := format
