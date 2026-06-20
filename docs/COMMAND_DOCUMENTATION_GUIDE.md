# Command Documentation Guide

Guide for creating and maintaining command documentation in AICockpit.

## Overview

Every AICockpit command must have comprehensive documentation. Documentation is stored in `docs/commands/` and follows a consistent structure.

## Why Document Commands?

- **User Experience**: Users can understand how to use commands
- **Discoverability**: Help system and documentation are complete
- **Maintainability**: Future developers understand command behavior
- **Quality**: Ensures commands are well-designed
- **Consistency**: All commands follow the same patterns

## Documentation Structure

### File Organization

```
docs/commands/
├── README.md                    # Command index and overview
├── COMMAND_TEMPLATE.md          # Template for new commands
├── setup.md                     # Documentation for setup command
├── info.md                      # Documentation for info command
├── doctor.md                    # Documentation for doctor command
├── metrics.md                   # Documentation for metrics command
└── uninstall.md                 # Documentation for uninstall command
```

### File Naming

- Use lowercase command name
- Use `.md` extension
- Example: `setup.md`, `my-command.md`

## Creating Command Documentation

### Step 1: Use the Template

Start with `COMMAND_TEMPLATE.md`:

```bash
cp docs/commands/COMMAND_TEMPLATE.md docs/commands/mycommand.md
```

### Step 2: Fill in Required Sections

Every command documentation must include:

1. **Header**
   ```markdown
   # Command: `cockpit <command-name>`
   
   > **Short description**
   ```

2. **Overview**
   - What the command does
   - When to use it
   - Key features

3. **Usage**
   ```bash
   cockpit <command-name> [flags] [arguments]
   ```

4. **Description**
   - Detailed explanation
   - How it works
   - What it does

5. **Flags**
   - Global flags table
   - Command-specific flags table
   - Flag descriptions

6. **Arguments**
   - Argument table
   - Required/optional indicators
   - Descriptions

7. **Examples**
   - Basic usage
   - With flags
   - Advanced usage
   - Real-world scenarios

8. **Output**
   - Success output example
   - Error output example
   - Output explanation

9. **Exit Codes**
   - Code meanings
   - When each code is returned

10. **Related Commands**
    - Links to related commands
    - Cross-references

11. **Troubleshooting**
    - Common problems
    - Solutions
    - Debugging tips

### Step 3: Add Real Examples

Examples should be:
- **Realistic**: Show actual use cases
- **Complete**: Include all necessary flags
- **Tested**: Verify examples work
- **Explained**: Describe what happens

```markdown
### Example: Filter by Status

```bash
cockpit metrics filter --status error
```

Shows all failed command executions.
```

### Step 4: Update the Index

Add your command to `docs/commands/README.md`:

```markdown
| [`cockpit mycommand`](mycommand.md) | Description | STABLE |
```

### Step 5: Validate Documentation

Run the validation script:

```bash
bash scripts/generate-command-docs.sh
```

This checks:
- All commands have documentation
- All required sections are present
- Documentation is complete

## Documentation Standards

### Writing Style

- **Clear**: Use simple, direct language
- **Concise**: Be brief but complete
- **Consistent**: Use same terminology throughout
- **Professional**: Maintain professional tone

### Code Formatting

Use proper markdown code blocks:

```markdown
# Inline code
Use `cockpit setup` to initialize.

# Code block
```bash
cockpit setup --language pt-br
```

# Output block
```
Example output here
```
```

### Tables

Use markdown tables for structured information:

```markdown
| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Value 1  | Value 2  | Value 3  |
```

### Links

Link to related documentation:

```markdown
- [Related Command](../INSTALLATION.md)
- [Configuration Guide](../CONFIGURATION.md)
```

## Keeping Documentation in Sync

### When to Update Documentation

Update documentation when:

1. **Adding a new flag**
   - Add to flags table
   - Add example using the flag
   - Update description

2. **Changing command behavior**
   - Update description
   - Update examples
   - Update output examples

3. **Adding a new subcommand**
   - Add to subcommands section
   - Document flags and arguments
   - Add examples

4. **Fixing a bug**
   - Update troubleshooting section
   - Update examples if affected
   - Note the fix

### Update Checklist

When modifying a command:

- [ ] Update command documentation
- [ ] Update examples (if behavior changed)
- [ ] Update flags table (if flags changed)
- [ ] Update output examples (if output changed)
- [ ] Run validation script
- [ ] Update AGENTS.md if needed
- [ ] Commit with documentation changes

## Command Documentation Checklist

Before submitting documentation, verify:

- [ ] File is named correctly (lowercase, .md extension)
- [ ] Header is present with command name
- [ ] Overview section explains what command does
- [ ] Usage section shows correct syntax
- [ ] Description is detailed and clear
- [ ] All flags are documented in table
- [ ] All arguments are documented in table
- [ ] At least 3 examples are provided
- [ ] Examples are tested and work
- [ ] Output examples are accurate
- [ ] Exit codes are documented
- [ ] Related commands are linked
- [ ] Troubleshooting section has 2+ issues
- [ ] Last updated date is current
- [ ] Status is set (STABLE, BETA, EXPERIMENTAL)
- [ ] Command is added to README.md index
- [ ] Validation script passes

## Example: Complete Command Documentation

Here's a complete example of command documentation:

```markdown
# Command: `cockpit setup`

> **Interactive Setup Wizard**  
> Initialize and configure AICockpit for first-time use.

## Overview

The `setup` command provides an interactive wizard to configure AICockpit...

## Usage

```bash
cockpit setup [flags]
```

## Description

The setup command initializes AICockpit by:
1. Creating the `.cockpit` directory
2. Creating the configuration file
3. Setting up logging directory
...

## Flags

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--language` | string | `en-us` | Set language |

## Examples

### Basic Setup

```bash
cockpit setup
```

Launches the interactive setup wizard.

## Output

### Success Output

```
✓ AICockpit setup completed successfully!
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Setup completed successfully |
| 1 | Setup failed |

## Troubleshooting

### Problem: Permission denied

**Solution**: Check home directory permissions.

---

**Last Updated**: June 20, 2026  
**Command Version**: 0.2.3  
**Status**: STABLE
```

## Validation Script

Run the validation script to check documentation:

```bash
bash scripts/generate-command-docs.sh
```

This script:
- Scans all command files
- Checks for documentation
- Validates required sections
- Reports missing documentation
- Provides summary

## Best Practices

### DO

- ✅ Keep documentation up-to-date
- ✅ Use clear, simple language
- ✅ Provide realistic examples
- ✅ Test all examples
- ✅ Link to related documentation
- ✅ Include troubleshooting
- ✅ Update when command changes
- ✅ Use consistent formatting

### DON'T

- ❌ Leave documentation outdated
- ❌ Use overly technical language
- ❌ Provide untested examples
- ❌ Skip troubleshooting section
- ❌ Forget to update index
- ❌ Use inconsistent formatting
- ❌ Document unimplemented features
- ❌ Ignore validation errors

## Contributing Documentation

To contribute command documentation:

1. **Create documentation** using template
2. **Fill all sections** completely
3. **Test all examples** thoroughly
4. **Update README.md** index
5. **Run validation** script
6. **Submit PR** with documentation

## Related Documentation

- [AGENTS.md](../AGENTS.md) - AI development guide
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
- [docs/commands/README.md](commands/README.md) - Command index
- [docs/commands/COMMAND_TEMPLATE.md](commands/COMMAND_TEMPLATE.md) - Template

## Questions?

If you have questions about command documentation:

1. Check existing command documentation
2. Review this guide
3. Check COMMAND_TEMPLATE.md
4. Ask in GitHub discussions

---

**Last Updated**: June 20, 2026  
**Version**: 0.2.3  
**Status**: ACTIVE
