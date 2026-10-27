# Command: `cockpit update`

> **Update AICockpit to Latest Version**  
> Check for updates and upgrade AICockpit to the latest version automatically.

## Overview

The `update` command checks for available updates and upgrades AICockpit to the latest version. It provides interactive prompts, displays changelog information, and automatically re-runs setup after a successful update.

## Usage

```bash
cockpit update [flags]
```

## Description

The update command performs the following operations:

1. **Checks for updates** - Queries GitHub Releases API for the latest version
2. **Displays version information** - Shows current and latest available versions
3. **Provides changelog link** - Direct link to release notes on GitHub
4. **Interactive confirmation** - Prompts user before proceeding with update
5. **Performs update** - Git-based upgrade process:
   - Fetches latest changes from repository
   - Checks out the new version tag
   - Rebuilds the application
   - Installs the updated binary
6. **Setup re-run** - Offers to automatically run setup after update

## Automatic Update Checking

AICockpit also includes automatic update checking that runs before every command:

- **Frequency**: Once per day (24-hour cache)
- **Trigger**: Runs before any command (except `update` and `setup`)
- **Configurable**: Can be disabled via `auto_update_check` in config
- **Non-blocking**: Doesn't prevent command execution if check fails

## Flags

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--language` | string | `en-us` | Set language (en-us, pt-br) |
| `--log-level` | string | `info` | Set log level (debug, info, warn, error) |

## Arguments

None. The update command is interactive and prompts for confirmation.

## Examples

### Manual Update Check

```bash
cockpit update
```

Checks for updates and prompts for confirmation if a new version is available.

### Update with Debug Logging

```bash
cockpit update --log-level debug
```

Performs update check with detailed debug logging.

### Update with Portuguese Language

```bash
cockpit update --language pt-br
```

Displays update messages in Portuguese.

## Output

The update command displays:

1. Check progress message
2. Version comparison (current vs latest)
3. Changelog link
4. Confirmation prompt
5. Update progress
6. Success/failure message
7. Setup re-run prompt

### Success Output

```bash
Checking for updates...
A new version of AICockpit is available: 0.2.0 (current: 0.1.0)
View changelog: https://github.com/lleitep3/aicockpit/releases/tag/v0.2.0
Would you like to update now? (y/n): y
Updating AICockpit to version 0.2.0...
Fetching latest changes...
Checking out version 0.2.0...
Pulling latest changes...
Rebuilding AICockpit...
Installing AICockpit...
AICockpit updated successfully to version 0.2.0
Would you like to run setup now? (y/n): y
Running setup...
```

### No Updates Available

```bash
Checking for updates...
✓ AICockpit is already up to date (version 0.1.0)
```

### Error Output

```bash
Checking for updates...
Failed to check for updates: connection timeout
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Update completed successfully or no updates available |
| 1 | Update failed (git error, build error, etc.) |
| 2 | Invalid arguments |
| 3 | User cancelled update |

## Configuration

Update behavior can be configured in `~/.cockpit/config.yaml`:

```yaml
# Enable/disable automatic update checking
auto_update_check: true

# Timestamp of last update check (automatically managed)
last_update_check: "2026-06-25T22:00:00Z"
```

### Disable Automatic Checks

To disable automatic update checking:

```yaml
auto_update_check: false
```

You can still manually check for updates using `cockpit update`.

## Requirements

For automatic updates to work:

1. **Git repository** - Must be run from within a git repository
2. **Network access** - Must be able to reach GitHub API
3. **Build tools** - Go compiler and build tools must be installed
4. **Write permissions** - Must have write access to installation directory

## Automatic Update Behavior

Automatic update checking:

- **Runs**: Before every command (except `update` and `setup`)
- **Frequency**: Maximum once per day
- **Cache**: Uses `last_update_check` timestamp
- **Failure handling**: Silently fails if check fails (doesn't block command)
- **User prompt**: Interactive prompt when update is available
- **Update action**: Directs user to run `cockpit update` for full update

## Changelog System

AICockpit includes automated changelog generation:

- **Script**: `scripts/generate-changelog.sh`
- **Source**: Conventional commits from git history
- **Categories**: Features, Bug Fixes, Performance, Breaking Changes, etc.
- **Integration**: Automatically used in GitHub Releases workflow

### Generate Changelog Manually

```bash
# Generate changelog for current version
bash scripts/generate-changelog.sh

# Generate for specific version
bash scripts/generate-changelog.sh 0.2.0
```

## Troubleshooting

### Problem: "Not in a git repository" error

**Solution**: Ensure you're running the command from within the AICockpit git repository.

```bash
cd /path/to/aicockpit
cockpit update
```

### Problem: "Failed to fetch" error

**Solution**: Check your internet connection and GitHub accessibility.

```bash
# Test GitHub connectivity
curl -I https://api.github.com

# Check git remote
git remote -v
```

### Problem: "Failed to build" error

**Solution**: Ensure Go and build tools are properly installed.

```bash
# Check Go installation
go version

# Verify build tools
make build
```

### Problem: Update check runs too frequently

**Solution**: Check the `last_update_check` timestamp in your config.

```bash
# View current config
cat ~/.cockpit/config.yaml

# Manually set last check time (optional)
# Edit config.yaml and set last_update_check to recent timestamp
```

### Problem: Want to disable automatic checks

**Solution**: Set `auto_update_check` to false in config.

```bash
# Edit config.yaml
vim ~/.cockpit/config.yaml

# Set:
auto_update_check: false
```

## Related Commands

- `cockpit setup` - Run setup after update
- `cockpit info` - Display current version
- `cockpit doctor` - Verify installation health

## See Also

- [Installation Guide](../INSTALLATION.md)
- [Configuration Guide](../CONFIGURATION.md)
- [CHANGELOG.md](../../CHANGELOG.md)
- [GitHub Releases](https://github.com/lleitep3/aicockpit/releases)

## Notes

- Automatic updates require git repository access
- Update process modifies local git state (checkout, pull)
- Setup re-run is recommended after update to ensure compatibility
- Changelog links provide detailed information about changes
- Update checks are non-blocking and don't prevent normal operation

## Best Practices

1. **Review changelog** before updating to understand changes
2. **Backup configuration** before major version updates
3. **Run setup** after update to ensure compatibility
4. **Test critical workflows** after updating
5. **Keep automatic checks enabled** for security and bug fixes

---

**Last Updated**: June 25, 2026  
**Command Version**: 0.1.0  
**Status**: STABLE