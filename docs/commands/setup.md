# Command: `cockpit setup`

> **Interactive Setup Wizard**  
> Initialize and configure AICockpit for first-time use.

## Overview

The `setup` command provides an interactive wizard to configure AICockpit. It guides you through the initial setup process, creating necessary directories and configuration files.

## Usage

```bash
cockpit setup [flags]
```

## Description

The setup command initializes AICockpit by:

1. Creating the `.cockpit` directory in your home folder
2. Creating the configuration file (`config.yaml`)
3. Setting up logging directory
4. Initializing metrics collection
5. Configuring language preferences
6. Setting up log levels

This command is typically run once when you first install AICockpit.

## Flags

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--language` | string | `en-us` | Set language (en-us, pt-br) |
| `--log-level` | string | `info` | Set log level (debug, info, warn, error) |

## Arguments

None. The setup command is interactive and prompts for input.

## Examples

### Basic Setup

```bash
cockpit setup
```

Launches the interactive setup wizard with default language (en-us).

### Setup with Portuguese

```bash
cockpit setup --language pt-br
```

Launches the setup wizard in Portuguese.

### Setup with Debug Logging

```bash
cockpit setup --log-level debug
```

Launches the setup wizard with debug logging enabled.

## Output

The setup command displays:

1. Welcome message
2. Configuration prompts
3. Confirmation of created files and directories
4. Summary of configuration

### Success Output

```
✓ AICockpit setup completed successfully!
✓ Configuration saved to: ~/.cockpit/config.yaml
✓ Logs directory created: ~/.cockpit/logs
✓ Ready to use AICockpit!
```

### Error Output

```
✗ Setup failed: Permission denied
Please check your home directory permissions.
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Setup completed successfully |
| 1 | Setup failed (permission error, etc.) |
| 2 | Invalid arguments |

## Configuration Created

The setup command creates:

```
~/.cockpit/
├── config.yaml          # Main configuration file
├── logs/                # Log directory
│   └── cockpit-YYYY-MM-DD.log
└── metrics.json         # Metrics file (created on first use)
```

### Default Configuration

```yaml
version: "0.2.3"
language: "en-us"
log_level: "info"
ai_provider: "claude"
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `COCKPIT_HOME` | Override default home directory (~/.cockpit) |
| `COCKPIT_LANGUAGE` | Override language setting |

## Related Commands

- `cockpit info` - Display current configuration
- `cockpit doctor` - Verify installation health
- `cockpit uninstall` - Remove AICockpit

## See Also

- [Installation Guide](../INSTALLATION.md)
- [Configuration Guide](../CONFIGURATION.md)
- [Quick Start](../QUICK_START.md)

## Notes

- Setup is idempotent - running it multiple times is safe
- Existing configuration will not be overwritten
- All files are created with appropriate permissions
- Logs are automatically rotated daily

## Troubleshooting

### Problem: "Permission denied" error

**Solution**: Check that you have write permissions in your home directory.

```bash
ls -la ~/ | grep cockpit
chmod 755 ~/.cockpit
```

### Problem: Setup hangs or doesn't respond

**Solution**: Press Ctrl+C to cancel and try again.

### Problem: Configuration not saved

**Solution**: Verify the `.cockpit` directory was created and has write permissions.

```bash
ls -la ~/.cockpit/
cat ~/.cockpit/config.yaml
```

---

**Last Updated**: June 20, 2026  
**Command Version**: 0.2.3  
**Status**: STABLE
