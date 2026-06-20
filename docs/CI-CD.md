# CI/CD Pipeline Documentation

## Overview

AICockpit uses a comprehensive CI/CD pipeline with automated testing, linting, coverage validation, and semantic versioning.

## Workflows

### 1. PR Check Workflow (`pr-check.yml`)

**Trigger**: Pull requests to `main` or `develop` branches

**Steps**:
- ✅ Run tests on Go 1.26 and 1.25
- ✅ Run linting with golangci-lint
- ✅ Validate code coverage (minimum 90%)
- ✅ Upload coverage reports to Codecov

**Coverage Requirement**: 
- Minimum 90% code coverage required
- PRs with less than 90% coverage will fail
- Coverage is calculated using `go tool cover`

**Example**:
```bash
# Coverage check
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 90" | bc -l) )); then
  exit 1  # Fail if coverage < 90%
fi
```

### 2. Build Workflow (`build.yml`)

**Trigger**: Push to `main` or `develop` branches

**Steps**:
- ✅ Build on Linux, macOS, and Windows
- ✅ Upload artifacts for each platform
- ✅ Test on Go 1.26

**Artifacts**:
- `cockpit-linux` - Linux binary
- `cockpit-macos` - macOS binary
- `cockpit-windows` - Windows executable

### 3. Test Workflow (`test.yml`)

**Trigger**: Push to `main` or `develop` branches

**Steps**:
- ✅ Run tests on Go 1.26 and 1.25
- ✅ Run linting with golangci-lint
- ✅ Upload coverage to Codecov

### 4. Release Workflow (`release.yml`)

**Trigger**: Push to `main` branch (after PR merge)

**Steps**:
1. Determine version bump type from commit message
2. Bump version using semantic versioning
3. Create commit with new version
4. Create git tag
5. Push changes and tag
6. Create GitHub release

**Version Bump Logic**:
- `feat(...)!:` → MAJOR version bump
- `feat(...)` → MINOR version bump
- `fix(...)` → PATCH version bump
- Other commits → PATCH version bump

**Example**:
```bash
# MAJOR bump
feat(auth)!: redesign authentication system

# MINOR bump
feat(metrics): add new metrics endpoint

# PATCH bump
fix(logging): fix race condition
```

## Version Management

### Version Files

Version is managed in multiple places:

1. **VERSION** - Simple version file
   ```
   0.1.0
   ```

2. **internal/version/version.go** - Go constant
   ```go
   const Version = "0.1.0"
   ```

3. **config.yaml** - User configuration
   ```yaml
   version: 0.1.0
   ```

### Automatic Version Updates

The release workflow automatically:
1. Reads the latest commit message
2. Determines bump type (MAJOR/MINOR/PATCH)
3. Updates VERSION file
4. Updates internal/version/version.go
5. Creates a git tag
6. Creates a GitHub release

**Note**: Version is ONLY updated on merge to `main`, not on PRs.

## Coverage Requirements

### Minimum Coverage: 90%

All PRs must maintain at least 90% code coverage.

### Coverage Calculation

```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

### Coverage Report

Coverage reports are automatically uploaded to Codecov for tracking over time.

## Commit Message Format

All commits must follow Conventional Commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `style` - Code style
- `refactor` - Code refactoring
- `perf` - Performance improvement
- `test` - Test changes
- `chore` - Build/dependency changes
- `ci` - CI/CD changes

### Examples

```
feat(metrics): add filtering by date

fix(logging): prevent duplicate entries

docs: update installation guide

chore: update dependencies
```

## PR Requirements

### Before Creating a PR

1. **Run checks locally**:
   ```bash
   make check
   ```

2. **Verify coverage**:
   ```bash
   go test -v -race -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out | grep total
   ```

3. **Install pre-commit hooks**:
   ```bash
   make install-hooks
   ```

### PR Title Format

PR titles must include version bump indicator:

```
[MAJOR] Breaking change description
[MINOR] New feature description
[PATCH] Bug fix description
```

### PR Checklist

- [ ] Tests pass locally
- [ ] Coverage >= 90%
- [ ] Code follows style guidelines
- [ ] Commit messages follow Conventional Commits
- [ ] PR title includes [MAJOR], [MINOR], or [PATCH]
- [ ] Documentation updated

## Local Testing

### Run All Checks

```bash
make check
```

This runs:
1. `go fmt` - Format code
2. `go vet` - Lint code
3. `go test` - Run tests with coverage

### Run Specific Checks

```bash
# Format code
make fmt

# Lint code
make lint

# Run tests
make test

# Build binary
make build
```

## Troubleshooting

### Coverage Below 90%

If coverage is below 90%:

1. **Identify uncovered code**:
   ```bash
   go tool cover -html=coverage.out
   ```

2. **Add tests** for uncovered code

3. **Verify coverage**:
   ```bash
   go tool cover -func=coverage.out | grep total
   ```

### Linting Failures

If linting fails:

1. **Run golangci-lint locally**:
   ```bash
   golangci-lint run ./...
   ```

2. **Fix issues** according to linter output

3. **Format code**:
   ```bash
   go fmt ./...
   ```

### Build Failures

If build fails:

1. **Check Go version**:
   ```bash
   go version
   ```
   Minimum required: Go 1.26

2. **Download dependencies**:
   ```bash
   go mod download
   ```

3. **Build locally**:
   ```bash
   make build
   ```

## GitHub Actions Secrets

The release workflow requires GitHub Actions secrets:

- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

## Release Process

### Manual Release (if needed)

```bash
# 1. Update version
./scripts/bump-version.sh minor

# 2. Commit
git add VERSION internal/version/version.go
git commit -m "chore(release): bump version to X.Y.Z"

# 3. Tag
git tag -a vX.Y.Z -m "Release vX.Y.Z"

# 4. Push
git push origin main
git push origin vX.Y.Z
```

### Automatic Release

Releases are automatically created when commits are merged to `main`.

## Monitoring

### GitHub Actions

View workflow runs:
- https://github.com/lleitep3/aicockpit/actions

### Codecov

View coverage reports:
- https://codecov.io/gh/lleitep3/aicockpit

## Best Practices

1. **Always run `make check` before committing**
2. **Write tests for new features**
3. **Maintain 90%+ coverage**
4. **Use semantic commit messages**
5. **Keep PRs focused and small**
6. **Review CI/CD results before merging**

## Further Reading

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [Go Testing](https://golang.org/doc/effective_go#testing)
- [GitHub Actions](https://docs.github.com/en/actions)
