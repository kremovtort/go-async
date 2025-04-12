# List all available recipes
default:
    @just --list

# Build the project
build:
    go build -v ./...

# Run all tests
test:
    go test -v ./...

# Run tests with race detection
test-race:
    go test -race -v ./...

# Run tests with coverage
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

# Run linter
lint:
    golangci-lint run

# Clean build artifacts
clean:
    go clean
    rm -f coverage.out

# Run all checks (build, test, lint)
check: build test lint

# Format code
fmt:
    go fmt ./...

# Run the example
example:
    go run examples/main.go

# Generate documentation
docs:
    pkgsite -http=:6060

# Create a new release
release VERSION:
    #!/usr/bin/env bash
    git tag -a v{{VERSION}} -m "Release v{{VERSION}}"
    git push origin v{{VERSION}}