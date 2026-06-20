# AICockpit - AI Harness Engineering Platform

A powerful CLI-based harness engineering tool that enables AI systems to evolve autonomously and operate more efficiently by optimizing token usage and improving performance over time.

## 🎯 Vision

AICockpit is designed to be the "cockpit" for your AI systems - a comprehensive control center that helps AI models:
- Operate more efficiently
- Save tokens through intelligent optimization
- Learn and improve from each interaction
- Manage knowledge bases and skills
- Execute commands with full audit trails

## 🚀 Quick Start

### Prerequisites
- Go 1.22.4 or later
- Linux, macOS, or Windows

### Installation

#### Linux/macOS

```bash
# Clone the repository
git clone https://github.com/lleite/aicockpit.git
cd aicockpit

# Build and install (automatic PATH configuration)
make install

# Verify installation
cockpit --version
```

The installation script automatically:
- Detects your shell (Bash, Zsh, Fish)
- Adds `~/.local/bin` to your PATH
- Creates shell config files if needed
- Verifies the installation

#### Windows

```powershell
# Clone the repository
git clone https://github.com/lleite/aicockpit.git
cd aicockpit

# Build and install (automatic PATH configuration)
make install-win

# Verify installation
cockpit --version
```

The installation script automatically:
- Adds `~/.local/bin` to your user PATH
- Updates current PowerShell session
- Verifies the installation

### Initial Setup

After installation, run the setup wizard:

```bash
# Run setup wizard
cockpit setup

# Check system health
cockpit doctor

# View configuration
cockpit info
```

For more details, see [INSTALLATION.md](INSTALLATION.md) and [QUICK_START.md](QUICK_START.md).

## 📋 Available Commands

### Core Commands

- **`cockpit setup`** - Interactive setup wizard
  - Select language (English, Portuguese)
  - Choose AI provider (Claude, GPT, Devin, Antigravity, Goose)
  - Initialize vault and configuration

- **`cockpit info`** - Display AICockpit information
  - Show current configuration
  - List installed packages
  - Display file locations

- **`cockpit doctor`** - Health check
  - Verify all components are properly configured
  - Check directory structure
  - Validate configuration files

- **`cockpit uninstall`** - Remove AICockpit
  - Safely uninstall and remove all data

### Planned Commands

- **`cockpit pkg`** - Package management
- **`cockpit vault`** - Secret management
- **`cockpit agents`** - Manage AI agents
- **`cockpit skills`** - Manage skills
- **`cockpit rules`** - Manage rules
- **`cockpit hooks`** - Manage hooks
- **`cockpit kb`** - Knowledge base management

## 🌍 Internationalization

AICockpit supports multiple languages:
- English (en-us)
- Portuguese Brazilian (pt-br)

Set language globally:
```bash
cockpit --language pt-br info
```

## 📁 Directory Structure

AICockpit creates the following structure in `~/.cockpit/`:

```
~/.cockpit/
├── config.yaml          # Configuration file
├── logs/               # Log files
├── cache/              # Cache directory
├── packages/           # Installed packages
├── vault/              # Secrets vault
├── agents/             # AI agents
├── skills/             # Skills
├── rules/              # Rules
├── hooks/              # Hooks
└── kb/                 # Knowledge bases
```

## 🔧 Development

### Build Commands

```bash
make help       # Show all available commands
make build      # Build the binary
make test       # Run tests with coverage
make lint       # Run linters
make fmt        # Format code
make check      # Run all checks
make clean      # Clean build artifacts
```

### Project Structure

- `cmd/` - CLI commands
- `internal/` - Internal packages
  - `config/` - Configuration management
  - `logger/` - Logging system
  - `i18n/` - Internationalization
- `main.go` - Entry point

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test -v ./internal/config

# View coverage report
go tool cover -html=coverage.out
```

## 📊 Code Quality

- **Linting**: `go vet` and golangci-lint
- **Testing**: Unit tests with >50% coverage target
- **Formatting**: `go fmt` and `goimports`
- **Documentation**: SDLC.md and AGENTS.md

## 🔐 Security

- Secrets are stored in OS keyrings (planned)
- All operations are logged
- Configuration is user-specific
- No sensitive data in logs

## 📝 Logging

All operations are logged to `~/.cockpit/logs/cockpit-YYYY-MM-DD.log`

Example log entry:
```
time=2026-06-20T11:30:02.021-03:00 level=INFO msg="Cockpit info displayed"
```

## 🤝 Contributing

1. Follow the SDLC guidelines in `SDLC.md`
2. Write tests for new features
3. Run `make check` before committing
4. Use clear commit messages

## 📄 License

[Add your license here]

## 🙋 Support

For issues, questions, or suggestions, please open an issue on GitHub.

## 🗺️ Roadmap

### Phase 1 (Current)
- ✅ Core CLI structure
- ✅ Configuration system
- ✅ Logging and i18n
- ⏳ Vault system

### Phase 2
- Package management
- Command execution with logging
- Knowledge base search

### Phase 3
- Agent management
- Skills and rules system
- Hooks system

### Phase 4
- AI-powered optimization
- Token usage analytics
- Performance metrics

## 🎓 Learn More

- See `SDLC.md` for development guidelines
- See `AGENTS.md` for agent-specific information
- See `initial-spec.md` for the original specification
