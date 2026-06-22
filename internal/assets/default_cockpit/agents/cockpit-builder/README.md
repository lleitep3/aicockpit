# Cockpit Builder Agent

## Overview

The Cockpit Builder Agent is an autonomous AI agent designed to help with building, testing, and managing AICockpit projects. It can perform tasks like project setup, code generation, testing, and documentation.

## Capabilities

### Project Setup
- Initialize new AICockpit projects
- Configure project structure
- Set up development environment
- Install dependencies

### Code Generation
- Generate CLI commands
- Generate agents, skills, and hooks
- Generate test files
- Generate documentation

### Testing
- Run unit tests
- Run integration tests
- Generate test coverage reports
- Identify untested code

### Documentation
- Generate API documentation
- Generate README files
- Generate changelog
- Update documentation

## Installation

```bash
cockpit agent install cockpit-builder
```

## Usage

### Initialize a Project

```bash
cockpit agent run cockpit-builder --task project-setup --name my-project
```

### Generate Code

```bash
cockpit agent run cockpit-builder --task code-generation --type command --name my-command
```

### Run Tests

```bash
cockpit agent run cockpit-builder --task testing --coverage
```

### Generate Documentation

```bash
cockpit agent run cockpit-builder --task documentation --format markdown
```

## Configuration

Edit `~/.cockpit/agents/cockpit-builder/config.yaml`:

```yaml
timeout: 300
max_retries: 3
log_level: "info"
parallel_tasks: 4
```

## Dependencies

- **go-development** skill: For Go code operations
- **file-management** skill: For file operations
- **git-operations** skill: For Git operations
- **cockpit-first** hook: For initialization

## Examples

### Example 1: Create a New Command

```bash
cockpit agent run cockpit-builder \
  --task code-generation \
  --type command \
  --name "my-command" \
  --description "My custom command"
```

### Example 2: Generate Tests

```bash
cockpit agent run cockpit-builder \
  --task testing \
  --coverage \
  --min-coverage 90
```

### Example 3: Create Documentation

```bash
cockpit agent run cockpit-builder \
  --task documentation \
  --format markdown \
  --output docs/
```

## Troubleshooting

### Agent Not Found

```bash
cockpit agent list
cockpit agent install cockpit-builder
```

### Configuration Issues

Check configuration file:

```bash
cat ~/.cockpit/agents/cockpit-builder/config.yaml
```

### Execution Failures

Check logs:

```bash
tail -f ~/.cockpit/logs/cockpit-*.log
```

## Contributing

To contribute to the Cockpit Builder Agent, please follow the Agent Best Practices guide in the knowledge base.

## License

MIT
