.PHONY: help build test lint fmt check clean install uninstall

# Variables
BINARY_NAME=cockpit
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=.
COVERAGE_FILE=coverage.out

help:
	@echo "AICockpit - Available commands:"
	@echo ""
	@echo "  make build      - Build the binary"
	@echo "  make test       - Run tests with coverage"
	@echo "  make lint       - Run golangci-lint"
	@echo "  make fmt        - Format and organize imports"
	@echo "  make check      - Run all checks (fmt + lint + test + build)"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make install    - Install binary to $(GOPATH)/bin/"
	@echo "  make uninstall  - Remove installed binary"
	@echo "  make help       - Show this help message"
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
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(GOPATH)/bin
	@cp $(BINARY_PATH) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✓ Uninstalled"

.DEFAULT_GOAL := help
