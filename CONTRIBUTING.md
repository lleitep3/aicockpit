# Contributing to AICockpit

Thank you for your interest in contributing to AICockpit! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Semantic Versioning](#semantic-versioning)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Code Style](#code-style)
- [Documentation](#documentation)

## Code of Conduct

Please be respectful and constructive in all interactions. We are committed to providing a welcoming and inclusive environment for all contributors.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone git@github.com:YOUR_USERNAME/aicockpit.git`
3. Add upstream remote: `git remote add upstream git@github.com:lleitep3/aicockpit.git`
4. Create a feature branch: `git checkout -b feature/your-feature-name`

## Development Setup

### Prerequisites

- **Go 1.26 or later** (required for development)
- Git
- Make

### Initial Setup

```bash
# Clone the repository
git clone git@github.com:lleitep3/aicockpit.git
cd aicockpit

# Install dependencies
go mod download

# Install pre-commit hooks
make install-hooks

# Run tests to verify setup
make test
```

## Semantic Versioning

AICockpit follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (X.0.0): Breaking changes, incompatible API changes
- **MINOR** (0.X.0): New features, backward compatible
- **PATCH** (0.0.X): Bug fixes, backward compatible

## Commit Guidelines

We use **Conventional Commits** for semantic versioning. All commits must follow this format:

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type

Must be one of the following:

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that don't affect code meaning (formatting, missing semicolons, etc)
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **chore**: Changes to build process, dependencies, or other non-code changes
- **ci**: Changes to CI configuration files and scripts

### Scope

Optional. Specify the section of the codebase affected:

- `config`
- `logging`
- `i18n`
- `cli`
- `metrics`
- `installation`
- `docs`

### Subject

- Use imperative mood ("add feature" not "added feature")
- Don't capitalize first letter
- No period (.) at the end
- Limit to 50 characters

### Body

Optional. Explain what and why, not how:

- Wrap at 72 characters
- Separate from subject with blank line
- Use bullet points for multiple changes

### Footer

Optional. Reference issues and breaking changes:

```
Closes #123
BREAKING CHANGE: description of breaking change
```

### Examples

```
feat(metrics): add command filtering by date

Add ability to filter metrics by specific date using --date flag.
This allows users to analyze metrics for specific days.

Closes #45
```

```
fix(logging): prevent duplicate log entries

Fix race condition in file logger that caused duplicate entries
when multiple commands executed simultaneously.
```

```
docs: update installation instructions for Windows
```

```
test(metrics): add tests for stats calculation
```

## Pull Request Process

### Before Creating a PR

1. **Update your branch**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run all checks**
   ```bash
   make check
   ```

3. **Ensure tests pass**
   ```bash
   make test
   ```

### Creating a PR

1. Push your branch to your fork
2. Create a Pull Request on GitHub
3. **PR Title Format**: `[TYPE] Description`
   - `[MAJOR]` for breaking changes
   - `[MINOR]` for new features
   - `[PATCH]` for bug fixes

### PR Title Examples

```
[MINOR] Add filtering by date to metrics command
[PATCH] Fix race condition in file logger
[MAJOR] Redesign configuration system
```

### PR Description Template

Use the provided PR template. Include:

- **Description**: What does this PR do?
- **Type of Change**: MAJOR/MINOR/PATCH
- **Related Issues**: Reference any related issues
- **Testing**: How was this tested?
- **Checklist**: Confirm all items

### PR Requirements

- ✅ All tests pass
- ✅ Code follows style guidelines
- ✅ Documentation is updated
- ✅ Commit messages follow conventions
- ✅ No breaking changes without [MAJOR] label
- ✅ PR title includes [MAJOR], [MINOR], or [PATCH]

### Review Process

1. At least one approval required
2. All CI checks must pass
3. No merge conflicts
4. Squash commits if requested

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test

# Run specific test
go test ./internal/logging -v
```

### Writing Tests

- Test files: `*_test.go`
- Test functions: `TestXxx(t *testing.T)`
- Use table-driven tests for multiple cases
- Aim for >80% coverage on core packages

### Example Test

```go
func TestMetricsCollector(t *testing.T) {
    tmpDir := t.TempDir()
    collector := NewMetricsCollector(tmpDir)

    metric := ExecutionMetric{
        Command:  "setup",
        Status:   "success",
        ExitCode: 0,
        Duration: 100.0,
    }

    if err := collector.RecordExecution(metric); err != nil {
        t.Fatalf("RecordExecution failed: %v", err)
    }

    metrics := collector.GetMetrics()
    if len(metrics) != 1 {
        t.Errorf("Expected 1 metric, got %d", len(metrics))
    }
}
```

## Code Style

### Go Style Guide

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Use `go vet` for static analysis
- Maximum line length: 100 characters

### Naming Conventions

- **Packages**: lowercase, single word
- **Functions**: CamelCase, exported functions start with uppercase
- **Variables**: camelCase
- **Constants**: UPPER_CASE

### Code Organization

```go
package mypackage

import (
    "fmt"
    "os"
)

// Exported function
func PublicFunction() {
    // implementation
}

// Unexported function
func privateFunction() {
    // implementation
}
```

### Comments

- Exported functions must have comments
- Comments should be clear and concise
- Use `//` for single-line comments
- Use `/* */` for multi-line comments

## Documentation

### Updating Documentation

1. Update relevant `.md` files in `docs/`
2. Update code comments if needed
3. Update README.md if adding new features
4. Keep documentation in sync with code

### Documentation Standards

- Use clear, concise language
- Include examples where appropriate
- Keep formatting consistent
- Update table of contents if needed

### Adding New Documentation

1. Create file in `docs/` directory
2. Add link to README.md
3. Follow existing documentation style
4. Include table of contents for long documents

## Pre-commit Hooks

Pre-commit hooks run automatically before each commit:

- ✅ Format code with `gofmt`
- ✅ Run `go vet`
- ✅ Run tests
- ✅ Check commit message format

### Installing Hooks

```bash
make install-hooks
```

### Skipping Hooks

```bash
git commit --no-verify
```

## CI/CD Pipeline

GitHub Actions automatically:

1. **On PR**: Run tests, lint, and build
2. **On main**: Run tests, lint, build, and deploy
3. **On develop**: Run tests and lint

### Workflow Files

- `.github/workflows/test.yml` - Test and lint
- `.github/workflows/build.yml` - Build verification

## Release Process

1. Create release branch: `git checkout -b release/v1.0.0`
2. Update version in code
3. Update CHANGELOG
4. Create PR with [MAJOR]/[MINOR]/[PATCH] label
5. After merge, create GitHub release
6. Tag commit: `git tag v1.0.0`

## Questions?

- Check existing issues and discussions
- Create a new issue for questions
- Join our community discussions

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

Thank you for contributing to AICockpit! 🚀
