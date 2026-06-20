# Command: `cockpit metrics`

> **Metrics and Analytics**  
> View and analyze command execution metrics.

## Overview

The `metrics` command provides access to AICockpit's metrics and analytics system. It allows you to view execution history, performance statistics, and usage patterns.

## Usage

```bash
cockpit metrics <subcommand> [flags]
```

## Description

The metrics command tracks:

1. Command execution history
2. Success/failure rates
3. Execution duration
4. Error types and frequencies
5. Usage patterns by command
6. Performance statistics

## Subcommands

### `list`

List all recorded metrics.

```bash
cockpit metrics list [flags]
```

**Flags:**
- `--limit N` - Show last N metrics (default: 100)
- `--format [json|text]` - Output format (default: text)

**Example:**
```bash
cockpit metrics list --limit 50
cockpit metrics list --format json
```

### `stats`

Show statistics about command execution.

```bash
cockpit metrics stats [flags]
```

**Flags:**
- `--by [command|status|date]` - Group statistics by (default: command)

**Example:**
```bash
cockpit metrics stats --by command
cockpit metrics stats --by status
cockpit metrics stats --by date
```

### `filter`

Filter metrics by criteria.

```bash
cockpit metrics filter [flags]
```

**Flags:**
- `--command NAME` - Filter by command name
- `--status [success|error]` - Filter by status
- `--date YYYY-MM-DD` - Filter by date
- `--limit N` - Maximum results

**Example:**
```bash
cockpit metrics filter --command setup
cockpit metrics filter --status error
cockpit metrics filter --date 2026-06-20
```

### `clear`

Clear all metrics data.

```bash
cockpit metrics clear [flags]
```

**Flags:**
- `--confirm` - Skip confirmation prompt

**Example:**
```bash
cockpit metrics clear --confirm
```

## Flags

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--language` | string | `en-us` | Set language (en-us, pt-br) |
| `--log-level` | string | `info` | Set log level (debug, info, warn, error) |

## Examples

### View Recent Metrics

```bash
cockpit metrics list
```

Shows the last 100 metrics.

### View Statistics by Command

```bash
cockpit metrics stats --by command
```

Shows execution statistics grouped by command.

### Filter by Status

```bash
cockpit metrics filter --status error
```

Shows all failed command executions.

### View Metrics as JSON

```bash
cockpit metrics list --format json
```

Outputs metrics in JSON format for processing.

## Output

### List Output (Text)

```
Metrics List
============

ID  | Timestamp           | Command | Status  | Duration | Error
----|---------------------|---------|---------|----------|-------
1   | 2026-06-20 10:30:45 | setup   | success | 2.34s    | -
2   | 2026-06-20 10:35:12 | info    | success | 0.12s    | -
3   | 2026-06-20 10:40:00 | doctor  | success | 0.45s    | -
```

### Stats Output

```
Statistics by Command
====================

Command    | Executions | Success | Failed | Avg Duration
-----------|------------|---------|--------|-------------
setup      | 1          | 1       | 0      | 2.34s
info       | 5          | 5       | 0      | 0.15s
doctor     | 3          | 3       | 0      | 0.42s
metrics    | 2          | 2       | 0      | 0.08s
```

### List Output (JSON)

```json
[
  {
    "id": 1,
    "timestamp": "2026-06-20T10:30:45Z",
    "command": "setup",
    "status": "success",
    "duration_ms": 2340,
    "error": null
  },
  {
    "id": 2,
    "timestamp": "2026-06-20T10:35:12Z",
    "command": "info",
    "status": "success",
    "duration_ms": 120,
    "error": null
  }
]
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Metrics not found |
| 2 | Invalid arguments |

## Metrics Data

Metrics are stored in `~/.cockpit/metrics.json`:

```json
{
  "metrics": [
    {
      "timestamp": "2026-06-20T10:30:45Z",
      "command": "setup",
      "args": [],
      "status": "success",
      "exit_code": 0,
      "duration_ms": 2340,
      "user": "username",
      "version": "0.2.3",
      "language": "en-us",
      "error": null,
      "error_type": null
    }
  ]
}
```

## Related Commands

- `cockpit info` - Display configuration
- `cockpit doctor` - Health check

## See Also

- [Logging & Metrics Guide](../LOGGING_AND_METRICS.md)
- [Metrics Command Verification](../METRICS_COMMAND_VERIFICATION.md)

## Notes

- Metrics are automatically collected for all commands
- Data is stored locally in `~/.cockpit/metrics.json`
- Metrics can be cleared at any time
- No data is sent to external servers
- Metrics help track usage patterns and identify issues

## Troubleshooting

### Problem: "Metrics not found"

**Solution**: Run some commands first to generate metrics.

```bash
cockpit setup
cockpit info
cockpit metrics list
```

### Problem: Metrics file is corrupted

**Solution**: Clear and regenerate metrics.

```bash
cockpit metrics clear --confirm
cockpit info  # Generate new metrics
```

### Problem: Metrics file is too large

**Solution**: Clear old metrics.

```bash
cockpit metrics clear --confirm
```

---

**Last Updated**: June 20, 2026  
**Command Version**: 0.2.3  
**Status**: STABLE
