# go-email Makefile

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint
BINARY_NAME=go-email
PACKAGE=github.com/go-email/go-email

# Version information
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X $(PACKAGE).GitCommit=$(COMMIT) -X $(PACKAGE).BuildDate=$(BUILD_DATE)"

.PHONY: all build clean test coverage fmt lint deps tidy help

# Default target
all: test build

# Build the project
build:
	@echo "Building..."
	$(GOBUILD) $(LDFLAGS) -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -f coverage.html

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOGET) -v ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Run all checks before commit
check: fmt lint test
	@echo "All checks passed!"

# Install the package
install:
	@echo "Installing..."
	$(GOCMD) install $(LDFLAGS) ./...

# Update dependencies
update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Generate documentation
doc:
	@echo "Starting documentation server..."
	@echo "Visit http://localhost:6060/pkg/$(PACKAGE)"
	godoc -http=:6060

# Run examples
examples:
	@echo "Running examples..."
	$(GOCMD) run examples/basic-usage.go

# Create a release
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is not set. Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Don't forget to push the tag: git push origin $(VERSION)"

# Show help
help:
	@echo "Available targets:"
	@echo "  all       - Run tests and build (default)"
	@echo "  build     - Build the project"
	@echo "  clean     - Clean build artifacts"
	@echo "  test      - Run tests"
	@echo "  coverage  - Run tests with coverage report"
	@echo "  fmt       - Format code"
	@echo "  lint      - Run linter"
	@echo "  deps      - Download dependencies"
	@echo "  tidy      - Tidy dependencies"
	@echo "  verify    - Verify dependencies"
	@echo "  check     - Run all checks (fmt, lint, test)"
	@echo "  install   - Install the package"
	@echo "  update    - Update dependencies"
	@echo "  doc       - Start documentation server"
	@echo "  examples  - Run example code"
	@echo "  release   - Create a release tag"
	@echo "  help      - Show this help message"
