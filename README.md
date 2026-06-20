# AICockpit - AI Harness Engineering Platform

[![Test and Lint](https://github.com/lleitep3/aicockpit/workflows/Test%20and%20Lint/badge.svg)](https://github.com/lleitep3/aicockpit/actions/workflows/test.yml)
[![Build](https://github.com/lleitep3/aicockpit/workflows/Build/badge.svg)](https://github.com/lleitep3/aicockpit/actions/workflows/build.yml)
[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A powerful CLI-based harness engineering tool that enables AI systems to evolve autonomously and operate more efficiently by optimizing token usage and improving performance over time.

## 🎯 Vision

AICockpit is designed to be the "cockpit" for your AI systems - a comprehensive control center that helps AI models:

- 🚀 Operate more efficiently
- 💰 Save tokens through intelligent optimization
- 📚 Learn and improve from each interaction
- 🧠 Manage knowledge bases and skills
- 📋 Execute commands with full audit trails
- 📊 Track metrics and performance

## 📚 Documentation

### Getting Started

- **[Quick Start Guide](docs/QUICK_START.md)** - Get up and running in 5 minutes
- **[Installation Guide](docs/INSTALLATION.md)** - Detailed installation instructions
- **[Installation Scripts](docs/INSTALLATION_SCRIPTS.md)** - How the installation scripts work
- **[Installation Options](docs/INSTALLATION_OPTIONS.md)** - User-level vs System-wide installation

### Features & Usage

- **[Features Overview](docs/FEATURES.md)** - Complete list of features and capabilities
- **[Logging & Metrics](docs/LOGGING_AND_METRICS.md)** - How to use the logging and metrics system
- **[Metrics Command Verification](docs/METRICS_COMMAND_VERIFICATION.md)** - Verify metrics command is working

### Development

- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to AICockpit
- **[AI Agent Guidelines](AGENTS.md)** - Comprehensive guide for AI agents developing on AICockpit
- **[CI/CD Pipeline](docs/CI-CD.md)** - CI/CD workflow and automation details
- **[SDLC Guidelines](docs/SDLC.md)** - Software Development Lifecycle standards

### Project Information

- **[Executive Summary](docs/EXECUTIVE_SUMMARY.md)** - High-level project overview
- **[Implementation Summary](docs/IMPLEMENTATION_SUMMARY.md)** - Phase 1 implementation details
- **[Session Summary](docs/SESSION_SUMMARY.md)** - Latest session summary
- **[Global Installation Verified](docs/GLOBAL_INSTALLATION_VERIFIED.md)** - Installation verification

## 🚀 Quick Start

### Prerequisites

- **Go 1.26 or later** (required for development)
- Linux, macOS, or Windows
- Git

### Installation

#### Linux/macOS

**Option 1: User-Level Installation (Recommended)**

```bash
# Clone the repository
git clone git@github.com:lleitep3/aicockpit.git
cd aicockpit

# Build and install (automatic PATH configuration)
make install

# Verify installation
cockpit --version
```

**Option 2: System-Wide Installation (Global)**

```bash
# Build and install globally (requires sudo)
make install-global

# Verify installation
cockpit --version
```

#### Windows

```powershell
# Build and install
make install-win

# Verify installation
cockpit --version
```

### Initial Setup

```bash
# Run setup wizard
cockpit setup

# Check system health
cockpit doctor

# View configuration
cockpit info
```

For more details, see [Quick Start Guide](docs/QUICK_START.md).

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

### Metrics & Analytics

- **`cockpit metrics list`** - View execution metrics
  - Filter by command, status, or date
  - Limit results
  - View all command executions

- **`cockpit metrics stats`** - Show execution statistics
  - Total executions and success rate
  - Average duration
  - Command frequency
  - Error types

- **`cockpit metrics logs`** - View log files
  - List all log files
  - View logs for specific date
  - Check log file details

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
├── metrics.json         # Execution metrics
├── logs/                # Log files (daily rotation)
│   ├── cockpit-2026-06-20.log
│   ├── cockpit-2026-06-21.log
│   └── ...
├── cache/               # Cache directory
├── packages/            # Installed packages
├── vault/               # Secrets vault
├── agents/              # AI agents
├── skills/              # Skills
├── rules/               # Rules
├── hooks/               # Hooks
└── kb/                  # Knowledge bases
```

## 🔧 Development

### Build Commands

```bash
make help              # Show all available commands
make build             # Build the binary
make test              # Run tests with coverage
make lint              # Run linters
make fmt               # Format code
make check             # Run all checks
make clean             # Clean build artifacts
make install-hooks     # Install git pre-commit hooks
```

### Project Structure

```
aicockpit/
├── cmd/                    # CLI commands
│   ├── setup.go
│   ├── info.go
│   ├── doctor.go
│   ├── uninstall.go
│   ├── metrics.go
│   └── root.go
├── internal/               # Internal packages
│   ├── config/            # Configuration management
│   ├── logger/            # Logging system (deprecated)
│   ├── i18n/              # Internationalization
│   └── logging/           # New logging & metrics system
├── scripts/                # Installation scripts
│   ├── install.sh         # Linux/macOS installer
│   └── install.ps1        # Windows installer
├── .github/                # GitHub configuration
│   ├── workflows/         # GitHub Actions
│   └── PULL_REQUEST_TEMPLATE.md
├── docs/                   # Documentation
├── main.go                 # Entry point
├── Makefile                # Build automation
├── CONTRIBUTING.md         # Contribution guidelines
└── README.md               # This file
```

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
- **Formatting**: `go fmt` compliance
- **Documentation**: Comprehensive guides and examples
- **CI/CD**: GitHub Actions for automated validation

## 🔐 Security

- Secrets are stored in OS keyrings (planned)
- All operations are logged
- Configuration is user-specific
- No sensitive data in logs
- Pre-commit hooks validate code quality

## 📊 Logging & Metrics

AICockpit automatically tracks all command executions with detailed metrics:

```bash
# View execution metrics
cockpit metrics list

# View statistics
cockpit metrics stats

# View log files
cockpit metrics logs

# Filter by command
cockpit metrics list --command setup

# Filter by status
cockpit metrics list --status error

# Filter by date
cockpit metrics list --date 2026-06-20
```

**Features:**
- Daily log rotation (cockpit-YYYY-MM-DD.log)
- JSON format for easy parsing
- Automatic metrics collection
- Success/failure tracking
- Performance metrics
- Error analysis

See [Logging & Metrics Documentation](docs/LOGGING_AND_METRICS.md) for details.

## 🤝 Contributing

We welcome contributions! Please follow these guidelines:

1. **Read the [Contributing Guide](CONTRIBUTING.md)** - Important guidelines and standards
2. **Semantic Commits** - Use conventional commit format
3. **PR Requirements** - Include [MAJOR], [MINOR], or [PATCH] in PR title
4. **Tests** - Write tests for new features
5. **Quality** - Run `make check` before committing

### Quick Contribution Steps

```bash
# 1. Fork and clone
git clone git@github.com:YOUR_USERNAME/aicockpit.git
cd aicockpit

# 2. Create feature branch
git checkout -b feature/your-feature

# 3. Install pre-commit hooks
make install-hooks

# 4. Make changes and commit
git add .
git commit -m "feat(scope): description"

# 5. Run checks
make check

# 6. Push and create PR
git push origin feature/your-feature
```

## 📄 License

[Add your license here]

## 🙋 Support

For issues, questions, or suggestions:

1. Check [existing issues](https://github.com/lleitep3/aicockpit/issues)
2. Create a [new issue](https://github.com/lleitep3/aicockpit/issues/new)
3. Join discussions

## 🗺️ Roadmap

### Phase 1 ✅ (Complete)

- ✅ Core CLI structure
- ✅ Configuration system
- ✅ Logging and i18n
- ✅ Metrics tracking
- ✅ Installation scripts (user-level & system-wide)

### Phase 2 (In Progress)

- [ ] Vault system (keyring integration)
- [ ] Package management
- [ ] Command execution with logging
- [ ] Knowledge base search

### Phase 3 (Planned)

- [ ] Agent management
- [ ] Skills and rules system
- [ ] Hooks system
- [ ] Advanced analytics

### Phase 4 (Vision)

- [ ] AI-powered optimization
- [ ] Token usage analytics
- [ ] Performance metrics
- [ ] Autonomous evolution

## 📈 Project Statistics

- **Total Commits**: 20+
- **Lines of Code**: 1500+
- **Lines of Documentation**: 1000+
- **Test Coverage**: 30.6%
- **Tests Passing**: 20/20 ✓

## 🎓 Learn More

### For Users

- [Quick Start Guide](docs/QUICK_START.md) - Get started quickly
- [Installation Guide](docs/INSTALLATION.md) - Detailed setup instructions
- [Features Overview](docs/FEATURES.md) - All available features
- [Logging & Metrics](docs/LOGGING_AND_METRICS.md) - How to use metrics

### For Developers

- [Contributing Guide](CONTRIBUTING.md) - How to contribute
- [SDLC Guidelines](docs/SDLC.md) - Development standards
- [AI Agent Guidelines](docs/AGENTS.md) - For AI agents working with AICockpit

### For Project Managers

- [Executive Summary](docs/EXECUTIVE_SUMMARY.md) - High-level overview
- [Implementation Summary](docs/IMPLEMENTATION_SUMMARY.md) - Phase 1 details
- [Session Summary](docs/SESSION_SUMMARY.md) - Latest work summary

## 🙌 Acknowledgments

AICockpit is built with:

- [Go](https://golang.org/) - Programming language
- [Cobra](https://cobra.dev/) - CLI framework
- [GitHub Actions](https://github.com/features/actions) - CI/CD

## 📞 Contact

- **GitHub**: [lleitep3/aicockpit](https://github.com/lleitep3/aicockpit)
- **Issues**: [GitHub Issues](https://github.com/lleitep3/aicockpit/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lleitep3/aicockpit/discussions)

---

**Made with ❤️ for AI systems everywhere**
