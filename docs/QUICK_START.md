# AICockpit - Quick Start Guide

## 🚀 5-Minute Setup

### 1. Build and Install
```bash
cd /home/lleite/projects/aicockpit
make install
```

### 2. Add to PATH (if needed)
```bash
export PATH="$HOME/.local/bin:$PATH"
```

### 3. Run Setup
```bash
cockpit setup
# Follow the prompts to select language and AI provider
```

### 4. Verify Installation
```bash
cockpit doctor
```

### 5. View Configuration
```bash
cockpit info
```

## 📦 Available Commands

```bash
# Setup and configuration
cockpit setup                          # Interactive setup
cockpit info                           # Show configuration
cockpit doctor                         # Health check
cockpit uninstall                      # Remove AICockpit

# Global flags
cockpit --language pt-br info          # Use Portuguese
cockpit --log-level debug info         # Debug mode
cockpit --version                      # Show version
cockpit --help                         # Show help
```

## 🛠️ Development Commands

```bash
# Build and test
make build      # Build binary to bin/cockpit
make test       # Run tests
make lint       # Check code quality
make fmt        # Format code
make check      # Run all checks (fmt + lint + test + build)
make clean      # Clean build artifacts

# Installation
make install    # Install to ~/.local/bin/cockpit
make uninstall  # Remove from ~/.local/bin/cockpit
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
# Make sure ~/.local/bin is in your PATH
export PATH="$HOME/.local/bin:$PATH"

# Or install again
cd /home/lleite/projects/aicockpit
make install

# Then run
cockpit --help
```

### Issue: Permission denied
```bash
# Reinstall (make install handles permissions)
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
