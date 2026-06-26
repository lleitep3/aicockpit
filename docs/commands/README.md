# AICockpit Commands Documentation

Complete reference for all AICockpit CLI commands.

## Command Index

### Core Commands

| Command | Description | Status |
|---------|-------------|--------|
| [`cockpit setup`](setup.md) | Interactive setup wizard | STABLE |
| [`cockpit info`](info.md) | Display configuration information | STABLE |
| [`cockpit doctor`](doctor.md) | Health check and diagnostics | STABLE |
| [`cockpit metrics`](metrics.md) | View and analyze metrics | STABLE |
| [`cockpit update`](update.md) | Update to latest version | STABLE |
| [`cockpit uninstall`](uninstall.md) | Remove AICockpit | STABLE |

### Planned Commands

| Command | Description | Status |
|---------|-------------|--------|
| `cockpit vault` | Manage secrets and credentials | PLANNED |
| `cockpit pkg` | Package management | PLANNED |
| `cockpit agent` | Agent management | PLANNED |
| `cockpit skill` | Skill management | PLANNED |
| `cockpit rule` | Rule management | PLANNED |
| `cockpit hook` | Hook management | PLANNED |
| `cockpit kb` | Knowledge base management | PLANNED |

## Quick Reference

### Get Help

```bash
# Show all commands
cockpit --help

# Get help for specific command
cockpit <command> --help

# Get help for subcommand
cockpit <command> <subcommand> --help
```

### Global Flags

All commands support these global flags:

```bash
--language [en-us|pt-br]    # Set language
--log-level [debug|info|warn|error]  # Set log level
--help                      # Show help
--version                   # Show version
```

## Command Categories

### Setup & Configuration

- [`cockpit setup`](setup.md) - Initialize AICockpit
- [`cockpit info`](info.md) - View configuration
- [`cockpit doctor`](doctor.md) - Verify installation

### Monitoring & Analytics

- [`cockpit metrics`](metrics.md) - View metrics and statistics

### Maintenance

- [`cockpit update`](update.md) - Update to latest version
- [`cockpit uninstall`](uninstall.md) - Remove AICockpit

## Documentation Guidelines

### For Users

Each command documentation includes:
- **Overview** - What the command does
- **Usage** - How to run the command
- **Description** - Detailed explanation
- **Flags** - Available options
- **Arguments** - Required/optional inputs
- **Examples** - Common use cases
- **Output** - What to expect
- **Exit Codes** - Success/failure indicators
- **Troubleshooting** - Common issues and solutions

### For Developers

When adding a new command:

1. **Create the command file** in `cmd/`
   ```go
   // cmd/mycommand.go
   func NewMyCommand(...) *cobra.Command {
       return &cobra.Command{
           Use: "mycommand",
           Short: "Short description",
           RunE: func(cmd *cobra.Command, args []string) error {
               // implementation
           },
       }
   }
   ```

2. **Register in root command** (`cmd/root.go`)
   ```go
   rootCmd.AddCommand(NewMyCommand(log, cfg, t))
   ```

3. **Create documentation** (`docs/commands/mycommand.md`)
   - Use [COMMAND_TEMPLATE.md](COMMAND_TEMPLATE.md) as reference
   - Document all flags and arguments
   - Provide examples
   - Include troubleshooting

4. **Update this README** (`docs/commands/README.md`)
   - Add command to index
   - Update command categories

5. **Keep documentation in sync**
   - Update docs when command changes
   - Update examples if behavior changes
   - Document new flags/arguments

## Automatic Documentation Generation

A script is planned to automatically generate command documentation from code:

```bash
./scripts/generate-docs.sh
```

This will:
- Extract command metadata from Go code
- Generate markdown documentation
- Update command index
- Validate documentation completeness

## Command Naming Conventions

### Command Names

- Use lowercase
- Use single words when possible
- Use hyphens for multi-word commands (e.g., `my-command`)
- Keep names short and descriptive

### Flag Names

- Use lowercase
- Use hyphens for multi-word flags (e.g., `--my-flag`)
- Use short flags for common options (e.g., `-f` for `--force`)
- Be consistent with common conventions

### Subcommand Names

- Use lowercase
- Use verbs when appropriate (e.g., `list`, `create`, `delete`)
- Keep consistent with similar commands

## Examples

### Simple Command

```bash
cockpit info
```

### Command with Flags

```bash
cockpit setup --language pt-br
```

### Command with Subcommand

```bash
cockpit metrics list --limit 50
```

### Command with Multiple Flags

```bash
cockpit metrics filter --command setup --status error --limit 10
```

## Help System

### Built-in Help

```bash
# Show all commands
cockpit --help

# Show specific command help
cockpit setup --help

# Show subcommand help
cockpit metrics list --help
```

### Documentation

- [Quick Start Guide](../QUICK_START.md)
- [Installation Guide](../INSTALLATION.md)
- [Configuration Guide](../CONFIGURATION.md)
- [Troubleshooting Guide](../TROUBLESHOOTING.md)

## Version Information

Commands are documented for AICockpit version **0.1.0**.

For version-specific information, run:

```bash
cockpit --version
cockpit info
```

## Contributing

To contribute command documentation:

1. Follow the [COMMAND_TEMPLATE.md](COMMAND_TEMPLATE.md)
2. Include all sections
3. Provide clear examples
4. Test all examples
5. Update the README index
6. Submit a PR

## Related Documentation

- [AGENTS.md](../../AGENTS.md) - AI development guide
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines
- [CI-CD.md](../CI-CD.md) - CI/CD pipeline

---

**Last Updated**: June 25, 2026  
**Version**: 0.1.0  
**Status**: ACTIVE
