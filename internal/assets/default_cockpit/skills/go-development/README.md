# Go Development Skill

## Overview

The Go Development Skill provides capabilities for Go development operations including code formatting, linting, testing, and building.

## Capabilities

### Format Code

Format Go code using `gofmt`.

```bash
cockpit skill execute go-development format-code --path ./cmd
```

### Lint Code

Lint Go code using `golangci-lint`.

```bash
cockpit skill execute go-development lint-code --path ./internal
```

### Run Tests

Run Go tests with optional coverage reporting.

```bash
cockpit skill execute go-development run-tests --path ./tests --coverage
```

### Build Binary

Build a Go binary from source.

```bash
cockpit skill execute go-development build-binary --path ./main.go --output ./bin/app
```

## Installation

```bash
cockpit skill install go-development
```

## Usage

### Format Code

```bash
cockpit skill execute go-development format-code \
  --path ./cmd
```

### Lint Code

```bash
cockpit skill execute go-development lint-code \
  --path ./internal
```

### Run Tests with Coverage

```bash
cockpit skill execute go-development run-tests \
  --path ./tests \
  --coverage
```

### Build Binary

```bash
cockpit skill execute go-development build-binary \
  --path ./main.go \
  --output ./bin/myapp
```

## Configuration

Edit `~/.cockpit/skills/go-development/config.yaml`:

```yaml
timeout: 60
max_retries: 2
log_level: "info"
go_version: "1.26"
```

## Requirements

- Go 1.26 or later
- golangci-lint 1.50 or later

## Examples

### Example 1: Format and Lint a Project

```bash
cockpit skill execute go-development format-code --path .
cockpit skill execute go-development lint-code --path .
```

### Example 2: Run Tests with Coverage

```bash
cockpit skill execute go-development run-tests \
  --path ./tests \
  --coverage
```

### Example 3: Build and Test

```bash
cockpit skill execute go-development run-tests --path ./tests
cockpit skill execute go-development build-binary --path ./main.go
```

## Troubleshooting

### Go Not Found

Ensure Go is installed and in your PATH:

```bash
go version
```

### golangci-lint Not Found

Install golangci-lint:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Tests Failing

Check test output:

```bash
cockpit skill execute go-development run-tests --path ./tests
```

## Contributing

To contribute to the Go Development Skill, please follow the Skill Best Practices guide in the knowledge base.

## License

MIT
