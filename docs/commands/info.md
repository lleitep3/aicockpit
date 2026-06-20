# Command: `cockpit info`

> **Display Configuration Information**  
> Show current AICockpit configuration and system information.

## Overview

The `info` command displays the current AICockpit configuration, version, and system information. Useful for debugging and verification.

## Usage

```bash
cockpit info [flags]
```

## Description

The info command displays:

1. AICockpit version
2. Configuration file location
3. Current language setting
4. Log level setting
5. AI provider configuration
6. Logs directory location
7. System information

## Flags

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--language` | string | `en-us` | Set language (en-us, pt-br) |
| `--log-level` | string | `info` | Set log level (debug, info, warn, error) |

## Arguments

None.

## Examples

### Display Configuration

```bash
cockpit info
```

Shows all configuration information.

### Display with Debug Logging

```bash
cockpit info --log-level debug
```

Shows configuration with debug logging enabled.

## Output

### Success Output

```
AICockpit Information
====================

Version: 0.2.3
Configuration File: ~/.cockpit/config.yaml
Logs Directory: ~/.cockpit/logs

Configuration:
  Language: en-us
  Log Level: info
  AI Provider: claude

System Information:
  Home Directory: /home/username
  OS: linux
  Architecture: amd64
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Configuration not found |
| 2 | Invalid arguments |

## Configuration Displayed

Shows the contents of `~/.cockpit/config.yaml`:

```yaml
version: "0.2.3"
language: "en-us"
log_level: "info"
ai_provider: "claude"
```

## Related Commands

- `cockpit setup` - Initialize configuration
- `cockpit doctor` - Verify installation health

## See Also

- [Configuration Guide](../CONFIGURATION.md)
- [Quick Start](../QUICK_START.md)

## Notes

- Does not modify any configuration
- Safe to run at any time
- Useful for troubleshooting

## Troubleshooting

### Problem: "Configuration not found" error

**Solution**: Run `cockpit setup` to create the configuration.

```bash
cockpit setup
```

### Problem: Shows incorrect information

**Solution**: Check the configuration file directly.

```bash
cat ~/.cockpit/config.yaml
```

---

**Last Updated**: June 20, 2026  
**Command Version**: 0.2.3  
**Status**: STABLE
