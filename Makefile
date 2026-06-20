.PHONY: help build test lint fmt check clean install uninstall

# Variables
BINARY_NAME=cockpit
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=.
COVERAGE_FILE=coverage.out
INSTALL_PATH=$(HOME)/.local/bin

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
	@mkdir -p $(INSTALL_PATH)
	@cp $(BINARY_PATH) $(INSTALL_PATH)/$(BINARY_NAME)
	@chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Installed to $(INSTALL_PATH)/$(BINARY_NAME)"
	@echo ""
	@echo "To use the command globally, add to your PATH:"
	@echo "  export PATH=\"$(INSTALL_PATH):$$PATH\""
	@echo ""
	@echo "Or add to your shell config (~/.bashrc, ~/.zshrc, etc):"
	@echo "  echo 'export PATH=\"$(INSTALL_PATH):$$PATH\"' >> ~/.bashrc"

uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Uninstalled from $(INSTALL_PATH)"

.DEFAULT_GOAL := help
