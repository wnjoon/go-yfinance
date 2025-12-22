.PHONY: build test lint docs docs-serve docs-build clean help

# Default target
all: build

# Python virtual environment for MkDocs
VENV := .venv
PIP := $(VENV)/bin/pip
MKDOCS := $(VENV)/bin/mkdocs

# Build the project
build:
	go build ./...

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run --timeout=5m

# Generate API documentation using gomarkdoc
docs:
	@echo "Generating API documentation..."
	@mkdir -p docs
	gomarkdoc --output docs/API.md ./pkg/...
	@echo "Documentation generated at docs/API.md"

# Setup MkDocs virtual environment
$(MKDOCS):
	python3 -m venv $(VENV)
	$(PIP) install --upgrade pip
	$(PIP) install mkdocs-material

# Serve documentation locally (http://localhost:8000)
docs-serve: $(MKDOCS)
	$(MKDOCS) serve

# Build documentation site
docs-build: $(MKDOCS)
	$(MKDOCS) build

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the project"
	@echo "  test       - Run tests"
	@echo "  test-cover - Run tests with coverage report"
	@echo "  lint       - Run golangci-lint"
	@echo "  docs       - Generate API documentation (gomarkdoc)"
	@echo "  docs-serve - Serve documentation locally (MkDocs)"
	@echo "  docs-build - Build documentation site (MkDocs)"
	@echo "  clean      - Clean build artifacts"
	@echo "  help       - Show this help message"
