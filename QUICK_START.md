# AICockpit - Quick Start Guide

## 🚀 5-Minute Setup

### 1. Build the Project
```bash
cd /home/lleite/projects/aicockpit
make build
```

### 2. Run Setup
```bash
./bin/cockpit setup
# Follow the prompts to select language and AI provider
```

### 3. Verify Installation
```bash
./bin/cockpit doctor
```

### 4. View Configuration
```bash
./bin/cockpit info
```

## 📦 Available Commands

```bash
# Setup and configuration
./bin/cockpit setup          # Interactive setup
./bin/cockpit info           # Show configuration
./bin/cockpit doctor         # Health check
./bin/cockpit uninstall      # Remove AICockpit

# Global flags
./bin/cockpit --language pt-br info    # Use Portuguese
./bin/cockpit --log-level debug info   # Debug mode
./bin/cockpit --version                # Show version
./bin/cockpit --help                   # Show help
```

## 🛠️ Development Commands

```bash
# Build and test
make build      # Build binary
make test       # Run tests
make lint       # Check code quality
make fmt        # Format code
make check      # Run all checks (fmt + lint + test + build)
make clean      # Clean build artifacts

# Installation
make install    # Install to $GOPATH/bin
make uninstall  # Remove from $GOPATH/bin
```

## 📁 Important Directories

```
~/.cockpit/                 # AICockpit home directory
├── config.yaml            # Configuration file
├── logs/                  # Log files
├── cache/                 # Cache directory
├── packages/              # Installed packages
├── vault/                 # Secrets vault
├── agents/                # AI agents
├── skills/                # Skills
├── rules/                 # Rules
├── hooks/                 # Hooks
└── kb/                    # Knowledge bases
```

## 🔍 Checking Logs

```bash
# View latest log
tail -f ~/.cockpit/logs/cockpit-$(date +%Y-%m-%d).log

# View all logs
ls -la ~/.cockpit/logs/
```

## 🌍 Language Support

```bash
# English (default)
./bin/cockpit info

# Portuguese
./bin/cockpit --language pt-br info
```

## 📊 Test Coverage

```bash
# Run tests with coverage report
make test

# View detailed coverage
go tool cover -html=coverage.out
```

## 🐛 Troubleshooting

### Issue: Command not found
```bash
# Make sure you're in the right directory
cd /home/lleite/projects/aicockpit

# Build first
make build

# Then run
./bin/cockpit --help
```

### Issue: Permission denied
```bash
# Make binary executable
chmod +x bin/cockpit

# Or reinstall
make install
```

### Issue: Configuration not found
```bash
# Run setup to create configuration
./bin/cockpit setup

# Or check if ~/.cockpit exists
ls -la ~/.cockpit/
```

## 📚 Documentation

- **README.md** - Full user guide
- **SDLC.md** - Development standards
- **AGENTS.md** - Agent guidelines
- **IMPLEMENTATION_SUMMARY.md** - Project status
- **initial-spec.md** - Original specification

## 🎯 Next Steps

1. ✅ Explore the CLI commands
2. ✅ Review the code structure
3. ✅ Run tests to verify everything works
4. ⏳ Start implementing Phase 2 features (vault, packages, etc)

## 💡 Tips

- Use `make check` before committing code
- Check logs in `~/.cockpit/logs/` for debugging
- Run `cockpit doctor` to verify system health
- Use `--language pt-br` to test Portuguese translations

## 🤝 Contributing

1. Create a feature branch
2. Make your changes
3. Run `make check` to verify
4. Commit with clear messages
5. Push and create a pull request

---

**Happy coding! 🚀**
