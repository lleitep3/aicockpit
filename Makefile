.PHONY: help build test lint fmt check clean install install-global install-win uninstall

# Variables
BINARY_NAME=cockpit
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=.
COVERAGE_FILE=coverage.out
INSTALL_PATH=$(HOME)/.local/bin

help:
	@echo "AICockpit - Available commands:"
	@echo ""
	@echo "  make build           - Build the binary"
	@echo "  make test            - Run tests with coverage"
	@echo "  make lint            - Run linters (go vet)"
	@echo "  make fmt             - Format code"
	@echo "  make check           - Run all checks (fmt + lint + test + build)"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make install         - Install to ~/.local/bin (user-level, Linux/macOS)"
	@echo "  make install-global  - Install to /usr/local/bin (system-wide, Linux/macOS)"
	@echo "  make install-win     - Install binary (Windows PowerShell)"
	@echo "  make uninstall       - Remove installed binary"
	@echo "  make help            - Show this help message"
	@echo ""

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@go build -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "✓ Build successful: $(BINARY_PATH)"

test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=$(COVERAGE_FILE) ./...
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	@echo "✓ Tests completed"

lint:
	@echo "Running linters..."
	@go vet ./...
	@echo "✓ Linting completed"

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

check: fmt lint test build
	@echo ""
	@echo "✓ All checks passed!"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ $(COVERAGE_FILE)
	@echo "✓ Clean completed"

install: build
	@bash scripts/install.sh

install-global: build
	@bash scripts/install.sh --global

install-win: build
	@powershell -ExecutionPolicy Bypass -File scripts/install.ps1

uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Uninstalled from $(INSTALL_PATH)"

.DEFAULT_GOAL := help
