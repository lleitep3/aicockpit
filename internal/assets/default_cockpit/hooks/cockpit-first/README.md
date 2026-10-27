# Cockpit First Hook

## Overview

The Cockpit First Hook is executed during AICockpit initialization and setup. It performs essential setup tasks to ensure AICockpit is properly configured and ready for use.

## Functionality

### Directory Creation

Creates default directories in the AICockpit workspace:

- `~/.cockpit/agents/`
- `~/.cockpit/skills/`
- `~/.cockpit/hooks/`
- `~/.cockpit/kb/`
- `~/.cockpit/logs/`
- `~/.cockpit/cache/`

### Knowledge Base Initialization

Initializes the knowledge base with default documents and configuration.

### Agent Setup

Sets up default agents and configurations.

## Events

### cockpit-init

Triggered when AICockpit initializes for the first time.

### config-load

Triggered after configuration is loaded.

## Installation

```bash
cockpit hook install cockpit-first
```

## Configuration

Edit `~/.cockpit/hooks/cockpit-first/config.yaml`:

```yaml
enabled: true
log_level: "info"
retry_count: 3
create_default_dirs: true
initialize_kb: true
setup_agents: true
```

## Configuration Options

- **enabled**: Enable or disable the hook
- **log_level**: Logging level (debug, info, warn, error)
- **retry_count**: Number of retries on failure
- **create_default_dirs**: Create default directories
- **initialize_kb**: Initialize knowledge base
- **setup_agents**: Setup default agents

## Behavior

### On First Run

1. Creates default directory structure
2. Initializes knowledge base
3. Sets up default agents
4. Logs initialization completion

### On Subsequent Runs

1. Validates directory structure
2. Checks knowledge base integrity
3. Updates configurations if needed
4. Logs validation results

## Examples

### Example 1: Manual Trigger

```bash
cockpit hook trigger cockpit-first --event cockpit-init
```

### Example 2: Check Status

```bash
cockpit hook status cockpit-first
```

### Example 3: View Logs

```bash
tail -f ~/.cockpit/logs/cockpit-*.log | grep cockpit-first
```

## Troubleshooting

### Hook Not Executing

Check if hook is enabled:

```bash
cat ~/.cockpit/hooks/cockpit-first/config.yaml
```

### Directory Creation Failed

Check permissions:

```bash
ls -la ~/.cockpit/
```

### Knowledge Base Initialization Failed

Check KB configuration:

```bash
cockpit kb list
```

## Integration

The Cockpit First Hook integrates with:

- **cockpit-builder** agent: For project setup
- **go-development** skill: For code operations
- **Knowledge Base**: For documentation

## Contributing

To contribute to the Cockpit First Hook, please follow the Hook Best Practices guide in the knowledge base.

## License

MIT
