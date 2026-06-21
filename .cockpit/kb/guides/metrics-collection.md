---
title: "Metrics Collection and Monitoring"
description: "Understanding and using metrics in AICockpit"
tags: ["metrics", "monitoring", "performance", "tracking"]
created: "2026-06-20"
modified: "2026-06-20"
author: "AICockpit Team"
version: "1.0"
related: []
---

# Metrics Collection and Monitoring

This guide explains how to use the metrics system in AICockpit to monitor performance and track usage.

## Overview

AICockpit collects metrics about:

- Command execution time
- Command success/failure rates
- Token usage (for AI operations)
- System resource usage
- Error rates and types

## Accessing Metrics

### View Metrics

```bash
cockpit metrics
```

This displays:
- Total commands executed
- Average execution time
- Success rate
- Error summary

### Export Metrics

```bash
cockpit metrics --export json
cockpit metrics --export csv
```

## Metrics Data

Metrics are stored in `~/.cockpit/logs/metrics.json`:

```json
{
  "timestamp": "2026-06-20T10:30:00Z",
  "command": "info",
  "duration_ms": 125,
  "status": "success",
  "tokens_used": 0,
  "error": null
}
```

## Performance Optimization

Use metrics to identify:

1. **Slow Commands**: Commands taking > 1000ms
2. **Frequent Errors**: Commands failing > 10% of the time
3. **Resource Usage**: High token consumption
4. **Patterns**: Time-based usage patterns

## Best Practices

1. Review metrics weekly
2. Archive old metrics monthly
3. Monitor error rates
4. Track performance trends
5. Optimize slow operations

## Troubleshooting

If metrics are not being collected:

1. Check if metrics are enabled in config
2. Verify logs directory permissions
3. Run `cockpit doctor` to diagnose issues
