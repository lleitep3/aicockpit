---
title: "Logging Configuration Guide"
description: "How to configure and use the logging system in AICockpit"
tags: ["logging", "configuration", "setup", "guide"]
created: "2026-06-20"
modified: "2026-06-20"
author: "AICockpit Team"
version: "1.0"
related: ["troubleshooting-logging-issues"]
---

# Logging Configuration Guide

This guide explains how to configure and use the logging system in AICockpit.

## Overview

AICockpit provides a comprehensive logging system with the following features:

- Daily log rotation
- Multiple log levels (debug, info, warn, error)
- Structured logging with timestamps
- Metrics collection and tracking

## Configuration

### Basic Setup

Logging is configured in the `~/.cockpit/config.yaml` file:

```yaml
log_level: "info"
language: "en-us"
```

### Log Levels

AICockpit supports the following log levels:

- **debug**: Detailed information for debugging
- **info**: General informational messages
- **warn**: Warning messages for potential issues
- **error**: Error messages for failures

### Log Location

Logs are stored in `~/.cockpit/logs/` directory with daily rotation:

```
~/.cockpit/logs/
├── cockpit.log.2026-06-20
├── cockpit.log.2026-06-21
└── cockpit.log.2026-22
```

## Usage

### Setting Log Level via CLI

```bash
cockpit --log-level debug info
cockpit --log-level error doctor
```

### Viewing Logs

```bash
# View current log
tail -f ~/.cockpit/logs/cockpit.log

# View specific date
cat ~/.cockpit/logs/cockpit.log.2026-06-20
```

## Best Practices

1. Use `debug` level during development
2. Use `info` level in production
3. Use `warn` level for potential issues
4. Use `error` level for failures only
5. Regularly rotate logs to save disk space

## Troubleshooting

If you're having issues with logging, see the [Troubleshooting Logging Issues](../troubleshooting/logging-issues.md) guide.
