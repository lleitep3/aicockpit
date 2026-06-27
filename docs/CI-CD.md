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

### 4. Update Changelog Workflow (`update-changelog.yml`)

**Trigger**: `push` to `main` or `workflow_dispatch`

**Steps**:
1. Checkout with `RELEASE_TOKEN` (PAT) so the bot PR can trigger checks
2. Run `scripts/update-changelog.sh --pr`
3. Generate a new `CHANGELOG.md` from Conventional Commits
4. Create PR from `chore/changelog-update-...`
5. Merge the PR with `gh pr merge --admin` (bypasses `required signed commits`)

**Note**: the PR commit has `[skip ci]`, so no CI checks run on it.

### 5. Release Workflow (`release.yml`)

**Trigger**: `workflow_run` after `Update Changelog` completes successfully

**Steps**:
1. Checkout with `RELEASE_TOKEN`
2. Run `scripts/bump-release.sh --pr`
3. Determine version bump from Conventional Commits
4. Update `VERSION`, `internal/version/version.go` and `CHANGELOG.md`
5. Create PR from `chore/release/bump-vX.Y.Z`
6. Merge the PR with `gh pr merge --admin`
7. Create git tag `vX.Y.Z`
8. Create GitHub release

**Version Bump Logic**:
- `feat(...)!:` or `BREAKING CHANGE` → MAJOR
- `feat(...)` → MINOR
- `fix(...)` → PATCH
- Other commits → PATCH

### 6. PR Validation Workflow (`pr-validation.yml`)

**Trigger**: Pull requests to `main` or `develop`

**Steps**:
- Validate PR description against the PR template using `scripts/validate-pr.sh`
- Enforce required sections, type of change, version impact and checklist

### 7. PR Changelog Generator (`pr-changelog.yml`)

**Trigger**: Pull requests to `main` or `develop`

**Steps**:
- Preview the changelog that would be generated for the PR


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

**Note**: Version is ONLY updated on merge to `main`, not on PRs. The release workflow runs after the changelog workflow and creates a release PR that is merged automatically.

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

The release workflow requires these GitHub Actions secrets:

- `GITHUB_TOKEN` - Automatically provided by GitHub Actions (used for release creation)
- `RELEASE_TOKEN` - Personal Access Token with `repo` (classic) or `contents:write` + `pull-requests:write` (fine-grained) used to create and merge bot PRs

In addition, the repository must have these settings enabled:

- Settings → Actions → General → `Read and write permissions`
- Settings → Actions → General → `Allow GitHub Actions to create and approve pull requests`

## Release Process

### Automatic Release

When a PR is merged to `main`:

1. `Update Changelog` runs, updates `CHANGELOG.md` and merges the changelog PR
2. `Release` runs, bumps the version, merges the release PR, creates the tag and GitHub Release

No manual version bump is required.

### Manual Release (if needed)

```bash
# 1. Dry-run to see what would happen
bash scripts/bump-release.sh --dry-run

# 2. Run the release pipeline manually
gh workflow run release.yml --ref main
```

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
4. **Use Conventional Commits** (`feat(scope):`, `fix(scope):`, etc.) — the release bump depende deles
5. **Keep PRs focused and small**
6. **Review CI/CD results before merging**
7. **Do not push directly to `main`** — use a feature branch and PR
8. **Keep `RELEASE_TOKEN` valid** — if it expires, the changelog/release workflows will fail
9. **Do not remove `[skip ci]` from automated commits** — it prevents unnecessary CI runs on bot PRs

## Further Reading

- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [Go Testing](https://golang.org/doc/effective_go#testing)
- [GitHub Actions](https://docs.github.com/en/actions)
