---
title: "Troubleshooting Logging Issues"
description: "Solutions for common logging problems in AICockpit"
tags: ["logging", "troubleshooting", "debugging", "issues"]
created: "2026-06-20"
modified: "2026-06-20"
author: "AICockpit Team"
version: "1.0"
related: ["logging-setup"]
---

# Troubleshooting Logging Issues

This guide provides solutions for common logging problems in AICockpit.

## Common Issues

### Logs Not Being Created

**Problem**: No log files appear in `~/.cockpit/logs/`

**Solution**:

1. Check if the logs directory exists:
   ```bash
   ls -la ~/.cockpit/logs/
   ```

2. If it doesn't exist, run setup:
   ```bash
   cockpit setup
   ```

3. Check directory permissions:
   ```bash
   chmod 755 ~/.cockpit/logs/
   ```

### Log File Too Large

**Problem**: Log files are consuming too much disk space

**Solution**:

1. Check current log size:
   ```bash
   du -sh ~/.cockpit/logs/
   ```

2. Archive old logs:
   ```bash
   gzip ~/.cockpit/logs/cockpit.log.2026-06-*
   ```

3. Delete old logs (keep last 30 days):
   ```bash
   find ~/.cockpit/logs/ -name "*.log.*" -mtime +30 -delete
   ```

### Log Level Not Changing

**Problem**: Setting log level via CLI doesn't work

**Solution**:

1. Verify the config file:
   ```bash
   cat ~/.cockpit/config.yaml
   ```

2. Update the config manually:
   ```yaml
   log_level: "debug"
   ```

3. Restart the application

### Permission Denied Errors

**Problem**: "Permission denied" when accessing logs

**Solution**:

1. Check file permissions:
   ```bash
   ls -la ~/.cockpit/logs/
   ```

2. Fix permissions:
   ```bash
   chmod 644 ~/.cockpit/logs/*
   chmod 755 ~/.cockpit/logs/
   ```

3. Check user ownership:
   ```bash
   chown -R $USER:$USER ~/.cockpit/
   ```

## Getting Help

If you're still having issues:

1. Check the [Logging Configuration Guide](../guides/logging-setup.md)
2. Run the doctor command:
   ```bash
   cockpit doctor
   ```
3. Enable debug logging:
   ```bash
   cockpit --log-level debug info
   ```
