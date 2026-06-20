# AICockpit - AI Agent Development Guide

## 🤖 Purpose

This document provides comprehensive guidance for AI agents (like Claude, GPT, Devin) working on the AICockpit project. It contains essential information about the project structure, development workflow, quality standards, and best practices.

---

## 📋 Table of Contents

1. [Project Overview](#project-overview)
2. [Technology Stack](#technology-stack)
3. [Project Structure](#project-structure)
4. [Development Workflow](#development-workflow)
5. [Code Quality Standards](#code-quality-standards)
6. [Testing Requirements](#testing-requirements)
7. [CI/CD Pipeline](#cicd-pipeline)
8. [Versioning & Releases](#versioning--releases)
9. [Common Tasks](#common-tasks)
10. [Troubleshooting](#troubleshooting)

---

## Project Overview

### What is AICockpit?

AICockpit is a **harness engineering tool for AI systems** that enables autonomous evolution and efficiency. It's a CLI-based system that helps AI models:

- Operate more efficiently
- Save tokens through intelligent optimization
- Learn and improve from each interaction
- Manage knowledge bases and skills
- Execute commands with full audit trails
- Track metrics and performance

### Current Phase

**Phase 1 - Core CLI (Complete)**
- ✅ Core CLI structure
- ✅ Configuration system (YAML)
- ✅ Logging with daily rotation
- ✅ Internationalization (EN/PT)
- ✅ Metrics tracking
- ✅ Installation scripts (user-level & system-wide)
- ✅ CI/CD with automated versioning

**Phase 2 - Knowledge Base & Search (In Progress)**
- ✅ KB system with metadata headers
- ✅ Keyword-based search
- ✅ Document repository (file-based)
- ✅ Scoring system (0-1 probability)
- ✅ `cockpit kb` command (search, list, add, remove)
- [ ] Semantic search with embeddings
- [ ] Skills for KB integration
- [ ] Hooks for automatic KB search

**Phase 3 - Vault & Packages (Next)**
- [ ] Vault system (keyring integration)
- [ ] Package management
- [ ] Command execution framework
- [ ] Extended commands

---

## Technology Stack

### Required

- **Language**: Go 1.26+ (MANDATORY)
- **CLI Framework**: Cobra (github.com/spf13/cobra)
- **Config Format**: YAML (gopkg.in/yaml.v3)
- **Testing**: Go's standard `testing` package

### Development Tools

- **Linting**: go vet (primary), golangci-lint v4
- **Formatting**: gofmt
- **Build**: Make
- **CI/CD**: GitHub Actions
- **Version Control**: Git with Conventional Commits

### Key Packages

```go
github.com/spf13/cobra      // CLI framework
gopkg.in/yaml.v3            // YAML parsing
```

---

## Project Structure

```
aicockpit/
├── .github/
│   ├── workflows/
│   │   ├── pr-check.yml          # PR validation (90% coverage required)
│   │   ├── build.yml             # Cross-platform build
│   │   ├── test.yml              # Tests and linting
│   │   └── release.yml           # Automatic versioning & release
│   ├── PULL_REQUEST_TEMPLATE.md  # PR template
│   └── pull_request_template.md  # PR template
├── ai-assets/                    # AI assets (separate from CLI)
│   ├── knowledge-base/           # Knowledge base documents
│   │   ├── guides/               # How-to guides
│   │   ├── references/           # Technical references
│   │   ├── examples/             # Code examples
│   │   ├── troubleshooting/      # Problem solutions
│   │   └── best-practices/       # Best practices
│   ├── skills/                   # Skills for IAs
│   │   └── kb-search/            # KB search skill (planned)
│   └── hooks/                    # Hooks for automation
│       └── auto-kb-search/       # Auto KB search hook (planned)
├── cmd/                          # CLI commands
│   ├── root.go                   # Root command
│   ├── setup.go                  # Setup wizard
│   ├── info.go                   # Display info
│   ├── doctor.go                 # Health check
│   ├── uninstall.go              # Uninstall
│   ├── metrics.go                # Metrics command
│   ├── kb.go                     # Knowledge base command
│   └── pkg.go                    # Package management (planned)
├── internal/                     # Internal packages (not exported)
│   ├── config/                   # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── logger/                   # Legacy logging (deprecated)
│   │   ├── logger.go
│   │   └── logger_test.go
│   ├── logging/                  # New logging & metrics system
│   │   ├── file_logger.go        # Daily log rotation
│   │   ├── file_logger_test.go
│   │   ├── metrics.go            # Metrics collection
│   │   ├── metrics_test.go
│   │   ├── manager.go            # Unified logging interface
│   │   └── manager_test.go
│   ├── i18n/                     # Internationalization
│   │   ├── i18n.go
│   │   └── i18n_test.go
│   ├── kb/                       # Knowledge base system
│   │   ├── kb.go                 # Types and interfaces
│   │   ├── kb_test.go
│   │   ├── metadata.go           # Metadata parsing
│   │   ├── metadata_test.go
│   │   ├── search.go             # Keyword search
│   │   ├── search_test.go
│   │   ├── semantic.go           # Semantic search (planned)
│   │   ├── semantic_test.go
│   │   ├── repository.go         # File-based repository
│   │   ├── repository_test.go
│   │   └── scorer.go             # Scoring system
│   ├── version/                  # Version management
│   │   └── version.go
├── scripts/
│   ├── install.sh                # Linux/macOS installer
│   ├── install.ps1               # Windows installer
│   ├── bump-version.sh           # Automatic version bumping
│   └── README.md
├── docs/                         # Documentation
│   ├── QUICK_START.md
│   ├── INSTALLATION.md
│   ├── FEATURES.md
│   ├── LOGGING_AND_METRICS.md
│   ├── KNOWLEDGE_BASE.md         # KB system documentation
│   ├── CI-CD.md
│   ├── SDLC.md
│   └── ... (other docs)
├── .git/
│   └── hooks/
│       ├── pre-commit            # Code quality validation
│       └── commit-msg            # Commit message validation
├── .golangci.yml                 # Linter configuration
├── go.mod                        # Go module definition
├── go.sum                        # Go dependencies lock
├── Makefile                      # Build automation
├── CONTRIBUTING.md               # Contribution guidelines
├── README.md                     # Project README
├── AGENTS.md                     # This file
├── VERSION                       # Current version (0.3.1)
└── main.go                       # Entry point
```

---

## Development Workflow

### 1. Setup Development Environment

```bash
# Clone the repository
git clone git@github.com:lleitep3/aicockpit.git
cd aicockpit

# Verify Go version (must be 1.26+)
go version

# Download dependencies
go mod download

# Install pre-commit hooks
make install-hooks

# Run initial checks
make check
```

### 2. Create Feature Branch

```bash
# Create feature branch with descriptive name
git checkout -b feature/your-feature-name

# Or for bug fixes
git checkout -b fix/your-bug-fix

# Or for documentation
git checkout -b docs/your-doc-update
```

### 3. Make Changes

```bash
# Make your code changes
# Follow code style guidelines (see below)
# Write tests for new functionality

# Format code
make fmt

# Run linting
make lint

# Run tests
make test

# Run all checks
make check
```

### 4. Commit Changes

```bash
# Commit with semantic message
git commit -m "feat(scope): description of feature"

# Pre-commit hooks will:
# ✓ Format code with gofmt
# ✓ Run go vet
# ✓ Run tests
# ✓ Validate commit message format
```

### 5. Create Pull Request

```bash
# Push to your fork
git push origin feature/your-feature-name

# Create PR on GitHub with:
# - Title: [MAJOR|MINOR|PATCH] Description
# - Description: What, why, how
# - Link related issues
```

### 6. PR Validation

PR Check workflow will:
- ✅ Test on Go 1.26 and 1.25
- ✅ Validate coverage >= 90%
- ✅ Run linting
- ✅ Upload to Codecov
- ✅ **NOT update version**

### 7. Merge & Release

```bash
# After approval, merge PR to main
# Release workflow will automatically:
# ✓ Detect bump type from commit
# ✓ Update VERSION file
# ✓ Update internal/version/version.go
# ✓ Create git tag
# ✓ Create GitHub release
```

---

## Code Quality Standards

### Must Follow

1. **Go Conventions**
   - PascalCase for exported functions/types
   - camelCase for private functions/variables
   - Use `go fmt` for formatting
   - Run `go vet` for static analysis

2. **Code Style**
   ```go
   // ✓ Good
   func ProcessMetrics(data []byte) error {
       // implementation
   }
   
   // ✗ Bad
   func process_metrics(data []byte) error {
       // implementation
   }
   ```

3. **Error Handling**
   ```go
   // ✓ Good - explicit error handling
   if err != nil {
       return fmt.Errorf("failed to load config: %w", err)
   }
   
   // ✗ Bad - ignoring errors
   _ = someFunction()
   ```

4. **Comments**
   ```go
   // ✓ Good - exported function has comment
   // ProcessMetrics processes execution metrics and returns statistics
   func ProcessMetrics(data []byte) error {
       // implementation
   }
   
   // ✗ Bad - no comment on exported function
   func ProcessMetrics(data []byte) error {
       // implementation
   }
   ```

5. **Testing**
   ```go
   // ✓ Good - table-driven tests
   func TestProcessMetrics(t *testing.T) {
       tests := []struct {
           name    string
           input   []byte
           wantErr bool
       }{
           {"valid", []byte("{}"), false},
           {"invalid", []byte(""), true},
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               err := ProcessMetrics(tt.input)
               if (err != nil) != tt.wantErr {
                   t.Errorf("got %v, want error %v", err, tt.wantErr)
               }
           })
       }
   }
   ```

### Linting Rules

```bash
# Run linting locally
make lint

# Or with golangci-lint
golangci-lint run ./...

# Enabled linters:
# - gosimple: Simplify code
# - govet: Standard analysis
# - staticcheck: Advanced analysis
# - typecheck: Type checking
# - unused: Detect unused code
# - gofmt: Format checking
# - goimports: Import checking
# - misspell: Spelling
# - revive: Configurable linter
# - errorlint: Error handling
```

---

## Testing Requirements

### Minimum Coverage: 90%

**This is MANDATORY for all PRs**

### Coverage Validation

```bash
# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Check coverage percentage
go tool cover -func=coverage.out | grep total

# View coverage in browser
go tool cover -html=coverage.out
```

### Coverage Calculation

```
Total coverage = (lines covered / total lines) * 100

Example:
  Total lines: 1000
  Covered lines: 950
  Coverage: 95% ✓ (passes 90% requirement)
```

### Writing Tests

```go
// ✓ Good test structure
func TestNewMetricsCollector(t *testing.T) {
    tmpDir := t.TempDir()
    
    collector := NewMetricsCollector(tmpDir)
    
    if collector == nil {
        t.Fatal("expected non-nil collector")
    }
}

// ✓ Test error cases
func TestMetricsCollectorInvalidPath(t *testing.T) {
    collector := NewMetricsCollector("/invalid/path/that/does/not/exist")
    
    metric := ExecutionMetric{Command: "test"}
    err := collector.RecordExecution(metric)
    
    if err == nil {
        t.Error("expected error for invalid path")
    }
}
```

### Test File Naming

```
source_test.go       # Tests for source.go
source_integration_test.go  # Integration tests
```

---

## CI/CD Pipeline

### Workflows Overview

#### 1. PR Check (pr-check.yml)

**Trigger**: Pull requests to `main` or `develop`

**Steps**:
- Test on Go 1.26 and 1.25
- Validate coverage >= 90%
- Run linting
- Upload to Codecov
- **Does NOT update version**

**Failure Conditions**:
- Coverage < 90%
- Tests fail
- Linting fails

#### 2. Build (build.yml)

**Trigger**: Push to `main` or `develop`

**Steps**:
- Build on Linux, macOS, Windows
- Upload artifacts

#### 3. Test (test.yml)

**Trigger**: Push to `main` or `develop`

**Steps**:
- Test on Go 1.26 and 1.25
- Run linting
- Upload coverage

#### 4. Release (release.yml)

**Trigger**: Push to `main` (after PR merge)

**Steps**:
1. Detect bump type from commit message
2. Update VERSION file
3. Update internal/version/version.go
4. Create commit
5. Create git tag
6. Create GitHub release

**Version Bump Rules**:
- `feat(...)!:` → MAJOR
- `feat(...)` → MINOR
- `fix(...)` → PATCH
- Other → PATCH

### Monitoring Workflows

```bash
# View workflow runs
gh run list --limit 10

# View specific run
gh run view <run-id> --log

# View latest run
gh run view -w pr-check
```

---

## Versioning & Releases

### Version Format

```
MAJOR.MINOR.PATCH

Example: 0.2.0
  0 = MAJOR (breaking changes)
  2 = MINOR (new features)
  0 = PATCH (bug fixes)
```

### Version Files

```
VERSION                           # Simple text file
internal/version/version.go       # Go constant
config.yaml                       # User config
```

### Automatic Version Updates

**ONLY happens on merge to main**

```bash
# Example workflow:
# 1. Commit: "feat(metrics): add filtering"
# 2. PR created with [MINOR] label
# 3. PR merged to main
# 4. Release workflow runs:
#    - Detects "feat" → MINOR bump
#    - Updates 0.1.0 → 0.2.0
#    - Creates tag v0.2.0
#    - Creates GitHub release
```

### Manual Version Bump (if needed)

```bash
# Bump version manually
./scripts/bump-version.sh minor

# Commit
git add VERSION internal/version/version.go
git commit -m "chore(release): bump version to X.Y.Z"

# Tag
git tag -a vX.Y.Z -m "Release vX.Y.Z"

# Push
git push origin main
git push origin vX.Y.Z
```

---

## Common Tasks

### Adding a New Command

1. **Create command file** in `cmd/`
   ```go
   // cmd/mycommand.go
   package cmd
   
   import (
       "github.com/spf13/cobra"
   )
   
   func NewMyCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
       return &cobra.Command{
           Use:   "mycommand",
           Short: "Description",
           RunE: func(cmd *cobra.Command, args []string) error {
               // implementation
               return nil
           },
       }
   }
   ```

2. **Register in root command** (`cmd/root.go`)
   ```go
   rootCmd.AddCommand(NewMyCommand(log, cfg, t))
   ```

3. **Add tests** (`cmd/mycommand_test.go`)
   ```go
   func TestMyCommand(t *testing.T) {
       // test implementation
   }
   ```

4. **Update documentation** in `docs/`

### Adding a New Package

1. **Create package directory** in `internal/`
   ```bash
   mkdir -p internal/mypackage
   ```

2. **Create main file** (`internal/mypackage/mypackage.go`)
   ```go
   package mypackage
   
   // MyType represents...
   type MyType struct {
       // fields
   }
   
   // NewMyType creates a new MyType
   func NewMyType() *MyType {
       return &MyType{}
   }
   ```

3. **Create tests** (`internal/mypackage/mypackage_test.go`)
   ```go
   func TestNewMyType(t *testing.T) {
       // test implementation
   }
   ```

4. **Ensure 90% coverage**
   ```bash
   go test -v -race -coverprofile=coverage.out ./internal/mypackage
   go tool cover -func=coverage.out
   ```

### Updating Documentation

1. **Update relevant docs** in `docs/`
2. **Update README.md** if needed
3. **Update AGENTS.md** if adding new guidelines
4. **Commit with `docs:` prefix**
   ```bash
   git commit -m "docs: update installation guide"
   ```

### Fixing a Bug

1. **Create fix branch**
   ```bash
   git checkout -b fix/bug-description
   ```

2. **Write failing test first**
   ```go
   func TestBugFix(t *testing.T) {
       // test that fails before fix
   }
   ```

3. **Implement fix**
   ```go
   // fix implementation
   ```

4. **Verify test passes**
   ```bash
   make test
   ```

5. **Commit with `fix:` prefix**
   ```bash
   git commit -m "fix(scope): description of fix"
   ```

---

## Troubleshooting

### Coverage Below 90%

**Problem**: PR rejected due to low coverage

**Solution**:
```bash
# 1. Identify uncovered code
go tool cover -html=coverage.out

# 2. Write tests for uncovered code
# 3. Verify coverage
go tool cover -func=coverage.out | grep total

# 4. Ensure >= 90%
```

### Linting Failures

**Problem**: `make lint` fails

**Solution**:
```bash
# 1. Run golangci-lint
golangci-lint run ./...

# 2. Fix issues according to output
# 3. Format code
make fmt

# 4. Verify
make lint
```

### Build Failures

**Problem**: `make build` fails

**Solution**:
```bash
# 1. Check Go version
go version  # Must be 1.26+

# 2. Download dependencies
go mod download

# 3. Clean and rebuild
make clean
make build
```

### Test Failures

**Problem**: `make test` fails

**Solution**:
```bash
# 1. Run tests with verbose output
go test -v ./...

# 2. Run specific test
go test -v -run TestName ./package

# 3. Debug with print statements
# 4. Fix implementation
# 5. Verify
make test
```

### Pre-commit Hook Issues

**Problem**: Pre-commit hook fails

**Solution**:
```bash
# 1. Reinstall hooks
make install-hooks

# 2. Or skip hooks (not recommended)
git commit --no-verify

# 3. Or fix issues and retry
make check
git commit -m "..."
```

### Version Mismatch

**Problem**: VERSION file doesn't match version.go

**Solution**:
```bash
# 1. Use bump-version.sh
./scripts/bump-version.sh patch

# 2. Or manually update both files
# 3. Commit
git add VERSION internal/version/version.go
git commit -m "chore: update version"
```

---

## Best Practices

### ✅ DO

- ✅ Run `make check` before committing
- ✅ Write tests for new features
- ✅ Maintain 90%+ coverage
- ✅ Use semantic commit messages
- ✅ Keep PRs focused and small
- ✅ Review CI/CD results before merging
- ✅ Update documentation
- ✅ Use descriptive branch names
- ✅ Test locally before pushing
- ✅ Read CONTRIBUTING.md before starting

### ❌ DON'T

- ❌ Commit without running `make check`
- ❌ Add code with < 90% coverage
- ❌ Use non-semantic commit messages
- ❌ Create large PRs with multiple features
- ❌ Ignore linting warnings
- ❌ Skip tests
- ❌ Update version manually (let CI/CD do it)
- ❌ Merge PRs with failing CI/CD
- ❌ Ignore code review comments
- ❌ Commit directly to main

---

## Useful Commands

```bash
# Build and test
make check              # Run all checks
make build              # Build binary
make test               # Run tests
make lint               # Run linting
make fmt                # Format code

# Installation
make install            # Install user-level
make install-global     # Install system-wide
make install-hooks      # Install git hooks

# Development
make clean              # Clean artifacts
go mod tidy             # Tidy dependencies
go mod download         # Download dependencies

# Testing
go test -v ./...                              # Verbose tests
go test -v -race ./...                        # Race detector
go test -v -coverprofile=coverage.out ./...   # With coverage
go tool cover -html=coverage.out              # View coverage

# Git
git log --oneline                             # View commits
git status                                    # Check status
git diff                                      # View changes
gh run list                                   # View workflows
```

---

## Knowledge Base System

### Overview

The Knowledge Base (KB) system allows organizing and searching documentation with:

- **Metadata Headers**: YAML headers with title, description, tags, etc.
- **Keyword Search**: Fast search based on titles, tags, descriptions, content
- **Scoring**: Probabilistic scoring (0-1) for result ranking
- **File-Based Storage**: Markdown documents in `~/.cockpit/kb/`

### Document Format

```markdown
---
title: "Document Title"
description: "Brief description"
tags: ["tag1", "tag2"]
author: "Author Name"
version: "1.0"
related: ["doc-id-1"]
---

# Content here
```

### Using KB

```bash
# Search documents
cockpit kb search "logging configuration"

# List all documents
cockpit kb list

# Add document
cockpit kb add /path/to/doc.md

# Remove document
cockpit kb remove doc-id
```

### Implementation Details

- **Package**: `internal/kb/`
- **Types**: `Document`, `Metadata`, `SearchResult`, `SearchResults`
- **Searcher**: `KeywordSearcher` with `DefaultScorer`
- **Repository**: `FileRepository` for file operations
- **Coverage**: 90.7% test coverage

### Adding KB Documents

1. Create file in `ai-assets/knowledge-base/{category}/`
2. Add metadata header with `---` delimiters
3. Write content in Markdown
4. Test with `cockpit kb search`

### Future Enhancements

- Semantic search with embeddings
- Full-text indexing
- Document versioning
- Integration with AI agents via skills and hooks

See [docs/KNOWLEDGE_BASE.md](docs/KNOWLEDGE_BASE.md) for detailed documentation.

---

## Resources

### Documentation
- [README.md](README.md) - Project overview
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [docs/CI-CD.md](docs/CI-CD.md) - CI/CD pipeline details
- [docs/QUICK_START.md](docs/QUICK_START.md) - Quick start guide
- [docs/SDLC.md](docs/SDLC.md) - Development lifecycle
- [docs/KNOWLEDGE_BASE.md](docs/KNOWLEDGE_BASE.md) - KB system documentation

### External Resources
- [Go Documentation](https://golang.org/doc/)
- [Cobra Documentation](https://cobra.dev/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Actions](https://docs.github.com/en/actions)

---

## Contact & Support

- **Issues**: [GitHub Issues](https://github.com/lleitep3/aicockpit/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lleitep3/aicockpit/discussions)
- **Repository**: [lleitep3/aicockpit](https://github.com/lleitep3/aicockpit)

---

## Summary

**Key Points for AI Agents**:

1. **Go 1.26+** is REQUIRED
2. **90% coverage** is MANDATORY for all PRs
3. **Semantic commits** are REQUIRED
4. **Version updates** are AUTOMATIC (don't do manually)
5. **Pre-commit hooks** validate code quality
6. **CI/CD** enforces all standards
7. **Tests** must pass before merging
8. **Documentation** must be updated

**Remember**: This project is designed for AI systems to evolve autonomously. Follow these guidelines to maintain code quality and enable continuous improvement.

---

**Last Updated**: June 20, 2026  
**Version**: 0.2.0  
**Status**: Production Ready
