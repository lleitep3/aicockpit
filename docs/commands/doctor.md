# Command: `cockpit doctor`

> **Health Check and Diagnostics**  
> Verify AICockpit installation and configuration health.

## Overview

The `doctor` command performs a comprehensive health check of your AICockpit installation. It verifies that all necessary files, directories, and configurations are in place and working correctly.

## Usage

```bash
cockpit doctor [flags]
```

## Description

The doctor command checks:

1. Configuration file existence and validity
2. Logs directory and permissions
3. Metrics file integrity
4. Required directories
5. File permissions
6. Configuration syntax
7. System compatibility

## Flags

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--language` | string | `en-us` | Set language (en-us, pt-br) |
| `--log-level` | string | `info` | Set log level (debug, info, warn, error) |

## Arguments

None.

## Examples

### Run Health Check

```bash
cockpit doctor
```

Performs a complete health check of the installation.

### Run with Verbose Output

```bash
cockpit doctor --log-level debug
```

Runs health check with detailed debug information.

## Output

### Success Output

```
AICockpit Health Check
=====================

✓ Configuration file exists: ~/.cockpit/config.yaml
✓ Configuration is valid YAML
✓ Logs directory exists: ~/.cockpit/logs
✓ Logs directory is writable
✓ Metrics file exists: ~/.cockpit/metrics.json
✓ All required directories present
✓ File permissions are correct
✓ System compatibility verified

Status: HEALTHY
All checks passed!
```

### Warning Output

```
AICockpit Health Check
=====================

✓ Configuration file exists: ~/.cockpit/config.yaml
⚠ Logs directory permissions could be improved
✓ Metrics file exists: ~/.cockpit/metrics.json
✓ All required directories present

Status: WARNING
Some checks need attention.
```

### Error Output

```
AICockpit Health Check
=====================

✗ Configuration file not found: ~/.cockpit/config.yaml
✗ Logs directory does not exist
✗ Metrics file is corrupted

Status: UNHEALTHY
Please run: cockpit setup
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All checks passed (HEALTHY) |
| 1 | Some checks failed (UNHEALTHY) |
| 2 | Invalid arguments |

## Checks Performed

### Configuration Checks
- File exists
- Valid YAML syntax
- Required fields present
- Correct values

### Directory Checks
- `.cockpit` directory exists
- `logs` directory exists
- Proper permissions (755)
- Writable by user

### File Checks
- `config.yaml` readable
- `metrics.json` valid JSON
- Log files accessible
- No corrupted files

### System Checks
- Go version compatible
- OS compatibility
- Required tools available

## Related Commands

- `cockpit setup` - Initialize configuration
- `cockpit info` - Display configuration
- `cockpit uninstall` - Remove AICockpit

## See Also

- [Installation Guide](../INSTALLATION.md)
- [Troubleshooting Guide](../TROUBLESHOOTING.md)

## Notes

- Safe to run at any time
- Does not modify configuration
- Useful for troubleshooting issues
- Can be run regularly to monitor health

## Troubleshooting

### Problem: "Configuration file not found"

**Solution**: Run setup to create configuration.

```bash
cockpit setup
```

### Problem: "Logs directory is not writable"

**Solution**: Fix directory permissions.

```bash
chmod 755 ~/.cockpit/logs
```

### Problem: "Metrics file is corrupted"

**Solution**: Delete the corrupted file (it will be recreated).

```bash
rm ~/.cockpit/metrics.json
cockpit metrics list
```

---

**Last Updated**: June 20, 2026  
**Command Version**: 0.2.3  
**Status**: STABLE
