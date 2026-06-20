# Command: `cockpit uninstall`

> **Uninstall AICockpit**  
> Remove AICockpit from your system.

## Overview

The `uninstall` command removes AICockpit from your system. It can optionally remove configuration and data files.

## Usage

```bash
cockpit uninstall [flags]
```

## Description

The uninstall command:

1. Removes the AICockpit binary
2. Optionally removes configuration files
3. Optionally removes logs and metrics
4. Cleans up git hooks
5. Provides confirmation prompts

## Flags

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--language` | string | `en-us` | Set language (en-us, pt-br) |
| `--log-level` | string | `info` | Set log level (debug, info, warn, error) |

### Command-Specific Flags

| Flag | Short | Type | Boolean | Description |
|------|-------|------|---------|-------------|
| `--remove-config` | `-c` | - | Yes | Remove configuration files |
| `--remove-data` | `-d` | - | Yes | Remove logs and metrics |
| `--force` | `-f` | - | Yes | Skip confirmation prompts |

## Arguments

None.

## Examples

### Interactive Uninstall

```bash
cockpit uninstall
```

Removes the binary with confirmation prompts.

### Remove Everything

```bash
cockpit uninstall --remove-config --remove-data
```

Removes binary, configuration, logs, and metrics.

### Force Uninstall

```bash
cockpit uninstall --force --remove-config --remove-data
```

Removes everything without confirmation.

## Output

### Interactive Output

```
AICockpit Uninstall
===================

This will remove AICockpit from your system.

Remove configuration files? (y/n): y
Remove logs and metrics? (y/n): y

Removing AICockpit...
✓ Binary removed
✓ Configuration removed
✓ Logs and metrics removed
✓ Git hooks cleaned up

AICockpit has been uninstalled.
```

### Force Uninstall Output

```
AICockpit Uninstall
===================

Removing AICockpit...
✓ Binary removed
✓ Configuration removed
✓ Logs and metrics removed
✓ Git hooks cleaned up

AICockpit has been uninstalled.
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Uninstall completed successfully |
| 1 | Uninstall failed (permission error, etc.) |
| 2 | Invalid arguments |
| 3 | User cancelled uninstall |

## Files Removed

### Always Removed
- AICockpit binary
- Git hooks

### With `--remove-config`
- `~/.cockpit/config.yaml`
- `~/.cockpit/` directory structure

### With `--remove-data`
- `~/.cockpit/logs/` directory
- `~/.cockpit/metrics.json`

## Related Commands

- `cockpit setup` - Initialize AICockpit
- `cockpit info` - Display configuration

## See Also

- [Installation Guide](../INSTALLATION.md)
- [Quick Start](../QUICK_START.md)

## Notes

- Configuration and data are preserved by default
- Use `--remove-config` to remove configuration
- Use `--remove-data` to remove logs and metrics
- Use `--force` to skip confirmation prompts
- Uninstall is reversible by reinstalling

## Troubleshooting

### Problem: "Permission denied" error

**Solution**: Check that you have permission to remove the binary.

```bash
which cockpit
ls -la $(which cockpit)
```

### Problem: Cannot remove configuration

**Solution**: Check directory permissions.

```bash
ls -la ~/.cockpit/
chmod 755 ~/.cockpit
```

### Problem: Uninstall hangs

**Solution**: Press Ctrl+C to cancel and try with `--force` flag.

```bash
cockpit uninstall --force
```

---

**Last Updated**: June 20, 2026  
**Command Version**: 0.2.3  
**Status**: STABLE
