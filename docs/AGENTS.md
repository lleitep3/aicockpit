# AICockpit - Agent Guidelines

## Project Overview

**AICockpit** is a harness engineering tool for AI systems that enables autonomous evolution and efficiency. It's a CLI-based system that helps AI models optimize their operations, save tokens, and improve performance over time.

## Technology Stack

- **Language**: Go 1.26+ (required for development)
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Config Format**: YAML (gopkg.in/yaml.v3)
- **Testing**: Go's standard `testing` package
- **Linting**: go vet (primary), golangci-lint (v4)
- **CI/CD**: GitHub Actions with automated versioning
- **Coverage**: Minimum 90% required for all PRs

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
make build           # Build the binary
make test            # Run tests with coverage
make lint            # Run linters (go vet)
make fmt             # Format code
make check           # Run all checks (fmt + lint + test + build)
make clean           # Clean build artifacts
make install         # Install binary to ~/.local/bin (user-level)
make install-global  # Install binary to /usr/local/bin (system-wide)
make install-hooks   # Install git pre-commit hooks
make uninstall       # Remove installed binary
```

## Version Management

- **Current Version**: Read from `VERSION` file
- **Version File**: `VERSION` (simple text file)
- **Go Constant**: `internal/version/Version`
- **Automatic Updates**: Version bumped on merge to `main` based on commit type
- **Semantic Versioning**: MAJOR.MINOR.PATCH

### Version Bump Rules

- `feat(...)!:` → MAJOR version bump
- `feat(...)` → MINOR version bump
- `fix(...)` → PATCH version bump
- Other commits → PATCH version bump

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
- **Minimum 90% coverage required** for all PRs
- Run tests with: `make test`
- Coverage is validated in CI/CD pipeline
- PRs with coverage < 90% will be rejected

### Coverage Validation

```bash
# Check coverage locally
go test -v -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total

# View coverage report in browser
go tool cover -html=coverage.out
```

## Code Style

- Follow Go conventions (PascalCase for exported, camelCase for private)
- Use `go fmt` before committing
- Run `go vet` for static analysis
- Keep functions small and focused
- Add comments for exported functions and complex logic

## CI/CD Pipeline

### Workflows

1. **PR Check** (`pr-check.yml`)
   - Runs on pull requests to `main` or `develop`
   - Tests on Go 1.26 and 1.25
   - Validates 90% coverage requirement
   - Runs linting with golangci-lint
   - Does NOT update version

2. **Build** (`build.yml`)
   - Runs on push to `main` or `develop`
   - Builds on Linux, macOS, Windows
   - Uploads artifacts

3. **Test** (`test.yml`)
   - Runs on push to `main` or `develop`
   - Tests on Go 1.26 and 1.25
   - Runs linting
   - Uploads coverage

4. **Release** (`release.yml`)
   - Runs on push to `main` (after PR merge)
   - Automatically bumps version
   - Creates git tag
   - Creates GitHub release

### Important Notes

- **Version is ONLY updated on merge to main**, not on PRs
- **Coverage must be >= 90%** for all PRs
- **All commits must follow Conventional Commits** format
- **PR titles must include [MAJOR], [MINOR], or [PATCH]**

## Next Steps for Development

1. Implement vault system with OS keyring integration
2. Create package manifest system (cockpit-package.yaml)
3. Implement package management commands
4. Add command execution with logging
5. Create agents, skills, rules, hooks, KB management
6. Implement knowledge base search functionality
7. Add integration tests for CLI commands
8. Increase test coverage to 90%+ across all packages

## Additional Important Notes

- All commands are logged to `~/.cockpit/logs/`
- Configuration is auto-created on first run
- The tool is designed to be extensible via packages
- Each package can contain CLI commands, skills, rules, agents, and knowledge bases
- Metrics are automatically tracked for all command executions
- Daily log rotation is automatic
