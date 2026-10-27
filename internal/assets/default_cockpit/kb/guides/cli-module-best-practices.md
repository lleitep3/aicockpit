---
title: "CLI Module Best Practices"
description: "Guidelines for creating CLI modules and commands in AICockpit"
tags: ["cli", "modules", "best-practices", "development"]
author: "AICockpit Team"
version: "1.0"
---

# CLI Module Best Practices

## Overview

CLI modules extend AICockpit's command-line interface with new functionality. Modules are organized collections of related commands that provide specific features.

## Module Structure

### Directory Layout

```
cmd/
├── module-name.go             # Module commands
├── module-name_test.go        # Module tests
└── subcommand/
    ├── subcommand1.go
    └── subcommand2.go

modules/
└── module-name/
    ├── manifest.yaml          # Module metadata
    ├── README.md              # Module documentation
    └── config/
        └── config.yaml
```

## Core Principles

### 1. Command Hierarchy

Commands should be organized hierarchically for clarity:

```
cockpit
├── kb                    # Knowledge Base module
│   ├── search           # Search documents
│   ├── list             # List documents
│   ├── root             # Manage roots
│   │   ├── add
│   │   ├── remove
│   │   └── list
│   └── rebuild-cache    # Rebuild index
├── agent                # Agent module
│   ├── list
│   ├── run
│   └── config
└── skill                # Skill module
    ├── list
    ├── install
    └── remove
```

### 2. Consistent Interface

All commands should follow the same patterns:

```go
// ✓ Good - Consistent pattern
func NewKBCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "kb",
        Short: "Knowledge base operations",
        Long:  "Manage and search the knowledge base",
    }
    
    cmd.AddCommand(NewKBSearchCommand(log, cfg, t))
    cmd.AddCommand(NewKBListCommand(log, cfg, t))
    
    return cmd
}

func NewKBSearchCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
    var format string
    var limit int
    
    cmd := &cobra.Command{
        Use:   "search <query>",
        Short: "Search knowledge base",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
            return nil
        },
    }
    
    cmd.Flags().StringVar(&format, "format", "default", "Output format")
    cmd.Flags().IntVar(&limit, "limit", 10, "Result limit")
    
    return cmd
}
```

### 3. Error Handling

Commands should provide clear error messages:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("query is required")
    }
    
    query := args[0]
    if query == "" {
        return fmt.Errorf("query cannot be empty")
    }
    
    // Execute command
    if err := executeSearch(query); err != nil {
        return fmt.Errorf("search failed: %w", err)
    }
    
    return nil
}
```

### 4. Output Formatting

Commands should support multiple output formats:

```go
// Support different output formats
var format string
cmd.Flags().StringVar(&format, "format", "default", "Output format (default, json, table)")

// In RunE:
switch format {
case "json":
    return outputJSON(results)
case "table":
    return outputTable(results)
default:
    return outputDefault(results)
}
```

## Command Implementation

### Basic Command Structure

```go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/lleite/aicockpit/internal/config"
    "github.com/lleite/aicockpit/internal/logging"
    "github.com/lleite/aicockpit/internal/i18n"
)

func NewMyCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
    var verbose bool
    
    cmd := &cobra.Command{
        Use:   "mycommand",
        Short: "Short description",
        Long:  "Longer description of what the command does",
        Example: "cockpit mycommand --verbose",
        RunE: func(cmd *cobra.Command, args []string) error {
            if verbose {
                log.Info("executing mycommand", map[string]interface{}{
                    "args": args,
                })
            }
            
            // Validate input
            if len(args) == 0 {
                return fmt.Errorf("argument required")
            }
            
            // Execute command
            result, err := execute(args[0])
            if err != nil {
                return fmt.Errorf("execution failed: %w", err)
            }
            
            // Output result
            fmt.Println(result)
            
            return nil
        },
    }
    
    cmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose output")
    
    return cmd
}

func execute(arg string) (string, error) {
    // Implementation
    return "result", nil
}
```

### Subcommand Structure

```go
func NewMyCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mycommand",
        Short: "My command module",
        Long:  "A module with multiple subcommands",
    }
    
    // Add subcommands
    cmd.AddCommand(NewSubcommand1(log, cfg, t))
    cmd.AddCommand(NewSubcommand2(log, cfg, t))
    cmd.AddCommand(NewSubcommand3(log, cfg, t))
    
    return cmd
}

func NewSubcommand1(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
    return &cobra.Command{
        Use:   "subcommand1",
        Short: "First subcommand",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
            return nil
        },
    }
}
```

## Testing

Commands should have comprehensive tests:

```go
func TestMyCommand(t *testing.T) {
    log, _ := logging.NewManager("")
    cfg := &config.Config{Version: "0.1.0"}
    translator := i18n.New("en-us")
    
    cmd := NewMyCommand(log, cfg, translator)
    
    if cmd == nil {
        t.Error("NewMyCommand() returned nil")
    }
    
    if cmd.Use != "mycommand" {
        t.Errorf("Use = %s, want mycommand", cmd.Use)
    }
}

func TestMyCommand_Execute(t *testing.T) {
    log, _ := logging.NewManager("")
    cfg := &config.Config{Version: "0.1.0"}
    translator := i18n.New("en-us")
    
    cmd := NewMyCommand(log, cfg, translator)
    
    // Set arguments
    cmd.SetArgs([]string{"test-arg"})
    
    // Execute
    err := cmd.Execute()
    if err != nil {
        t.Errorf("Execute() error = %v", err)
    }
}
```

## Module Manifest

Every module should have a `manifest.yaml`:

```yaml
name: "module-name"
version: "1.0.0"
description: "Module description"
author: "Author Name"
license: "MIT"

# Commands provided by this module
commands:
  - name: "command1"
    description: "Command 1 description"
  - name: "command2"
    description: "Command 2 description"

# Required skills
skills:
  - "skill-name-1"
  - "skill-name-2"

# Required hooks
hooks:
  - "hook-name-1"

# Configuration
config:
  log_level: "info"
  timeout: 30

# Dependencies
dependencies:
  go: "1.26"
  cockpit: "0.2.0"
```

## Best Practices Checklist

- [ ] Command has clear, descriptive name
- [ ] Command has short and long descriptions
- [ ] Command has examples
- [ ] Command validates input arguments
- [ ] Command has comprehensive error handling
- [ ] Command supports multiple output formats
- [ ] Command has unit tests with >90% coverage
- [ ] Command follows Cobra conventions
- [ ] Command has clear documentation
- [ ] Command follows Go conventions

## Output Formatting

### Default Format

```go
func outputDefault(results []Result) error {
    fmt.Println("Results:")
    for i, r := range results {
        fmt.Printf("%d. %s\n", i+1, r.Name)
        fmt.Printf("   Description: %s\n", r.Description)
    }
    return nil
}
```

### JSON Format

```go
func outputJSON(results []Result) error {
    data, err := json.MarshalIndent(results, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal JSON: %w", err)
    }
    fmt.Println(string(data))
    return nil
}
```

### Table Format

```go
func outputTable(results []Result) error {
    fmt.Printf("%-30s | %-50s\n", "Name", "Description")
    fmt.Println(string(make([]byte, 85)))
    
    for _, r := range results {
        name := r.Name
        if len(name) > 30 {
            name = name[:27] + "..."
        }
        
        desc := r.Description
        if len(desc) > 50 {
            desc = desc[:47] + "..."
        }
        
        fmt.Printf("%-30s | %-50s\n", name, desc)
    }
    
    return nil
}
```

## Common Patterns

### Flag Validation

```go
RunE: func(cmd *cobra.Command, args []string) error {
    // Validate required flags
    if output == "" {
        return fmt.Errorf("--output flag is required")
    }
    
    // Validate flag values
    if format != "json" && format != "table" && format != "default" {
        return fmt.Errorf("invalid format: %s", format)
    }
    
    // Validate numeric flags
    if limit < 0 {
        return fmt.Errorf("limit must be positive")
    }
    
    return nil
}
```

### Logging Integration

```go
RunE: func(cmd *cobra.Command, args []string) error {
    log.Info("command executing", map[string]interface{}{
        "command": cmd.Name(),
        "args": args,
    })
    
    start := time.Now()
    
    // Execute command
    result, err := execute(args)
    
    duration := time.Since(start)
    
    if err != nil {
        log.Error("command failed", err)
        return err
    }
    
    log.Info("command completed", map[string]interface{}{
        "duration_ms": duration.Milliseconds(),
    })
    
    return nil
}
```

## Performance Considerations

- Minimize startup time
- Cache expensive operations
- Use goroutines for concurrent operations
- Monitor memory usage
- Optimize output formatting

## Security Considerations

- Validate all input arguments
- Don't log sensitive information
- Use secure communication channels
- Implement rate limiting
- Regularly update dependencies

## Troubleshooting

### Command Not Found

1. Check command is registered in root.go
2. Verify command name is correct
3. Check for typos in command path
4. Verify module is installed

### Command Fails

1. Check command arguments
2. Verify configuration is valid
3. Check logs for error details
4. Verify required dependencies

### Slow Command Execution

1. Profile command execution
2. Optimize expensive operations
3. Implement caching
4. Use goroutines for concurrent tasks
