# AICockpit - Agent Guidelines

## Project Overview

**AICockpit** is a harness engineering tool for AI systems that enables autonomous evolution and efficiency. It's a CLI-based system that helps AI models optimize their operations, save tokens, and improve performance over time.

## Technology Stack

- **Language**: Go 1.22.4+
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Config Format**: YAML (gopkg.in/yaml.v3)
- **Testing**: Go's standard `testing` package
- **Linting**: go vet (primary), golangci-lint (available)

## Project Structure

```
aicockpit/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command
│   ├── setup.go           # Setup command
│   ├── info.go            # Info command
│   ├── doctor.go          # Doctor command
│   └── uninstall.go       # Uninstall command
├── internal/              # Internal packages (not exported)
│   ├── config/            # Configuration management
│   ├── logger/            # Logging system
│   └── i18n/              # Internationalization
├── main.go               # Entry point
├── Makefile              # Build automation
├── SDLC.md              # Development lifecycle
├── AGENTS.md            # This file
├── go.mod
├── go.sum
└── .golangci.yml        # Linter configuration
```

## Build Commands

```bash
make build      # Build the binary
make test       # Run tests with coverage
make lint       # Run linters (go vet)
make fmt        # Format code
make check      # Run all checks (fmt + lint + test + build)
make clean      # Clean build artifacts
make install    # Install binary to $GOPATH/bin
make uninstall  # Remove installed binary
```

## Current Implementation Status

### ✅ Completed
- [x] Project structure and Go module setup
- [x] Configuration system (config.yaml in ~/.cockpit)
- [x] Logging system with file output
- [x] Internationalization (i18n) - English and Portuguese
- [x] `cockpit setup` command
- [x] `cockpit info` command
- [x] `cockpit doctor` command
- [x] `cockpit uninstall` command
- [x] Unit tests for config, logger, and i18n
- [x] Build automation with Makefile
- [x] SDLC documentation

### ⏳ Pending
- [ ] Vault system (keyring integration)
- [ ] Package management commands (pkg list, install, remove, etc)
- [ ] Agents, skills, rules, hooks, KB commands
- [ ] Command execution with logging
- [ ] Package manifest system (cockpit-package.yaml)

## Key Design Decisions

1. **Singleton Pattern**: Logger and Translator use singleton pattern for global access
2. **Separation of Concerns**: Clear separation between CLI commands, config, and core logic
3. **Internationalization**: Full i18n support from the start (en-us, pt-br)
4. **Testing**: Unit tests for all core packages with >50% coverage target
5. **Error Handling**: Explicit error handling with proper error wrapping

## Configuration

Config file location: `~/.cockpit/config.yaml`

```yaml
version: "0.1.0"
language: "en-us"
log_level: "info"
ai_provider: "claude"
```

## Logging

- **Location**: `~/.cockpit/logs/cockpit-YYYY-MM-DD.log`
- **Format**: Text format with timestamp and level
- **Output**: Both console and file

## Testing Guidelines

- Use Go's standard `testing` package
- Create `*_test.go` files in the same package
- Target minimum 50% coverage for new code
- Run tests with: `make test`

## Code Style

- Follow Go conventions (PascalCase for exported, camelCase for private)
- Use `go fmt` before committing
- Run `go vet` for static analysis
- Keep functions small and focused
- Add comments for exported functions and complex logic

## Next Steps for Development

1. Implement vault system with OS keyring integration
2. Create package manifest system (cockpit-package.yaml)
3. Implement package management commands
4. Add command execution with logging
5. Create agents, skills, rules, hooks, KB management
6. Implement knowledge base search functionality
7. Add integration tests for CLI commands

## Important Notes

- All commands are logged to `~/.cockpit/logs/`
- Configuration is auto-created on first run
- The tool is designed to be extensible via packages
- Each package can contain CLI commands, skills, rules, agents, and knowledge bases
