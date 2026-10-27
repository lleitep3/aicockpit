---
title: "Hook Best Practices"
description: "Guidelines for creating effective hooks in AICockpit"
tags: ["hooks", "best-practices", "development"]
author: "AICockpit Team"
version: "1.0"
---

# Hook Best Practices

## Overview

Hooks are event handlers that execute in response to specific events in the AICockpit lifecycle. Hooks enable reactive programming patterns and allow components to respond to system events.

## Hook Types

### Lifecycle Hooks

- `cockpit-init`: Runs when AICockpit initializes
- `cockpit-shutdown`: Runs when AICockpit shuts down
- `config-load`: Runs after configuration is loaded
- `config-save`: Runs before configuration is saved

### Command Hooks

- `pre-command`: Runs before any command execution
- `post-command`: Runs after any command execution
- `command-error`: Runs when a command fails

### KB Hooks

- `kb-index-rebuild`: Runs after KB index is rebuilt
- `kb-document-add`: Runs after a document is added
- `kb-document-remove`: Runs after a document is removed
- `kb-search`: Runs after a KB search

### Agent Hooks

- `agent-start`: Runs when an agent starts
- `agent-stop`: Runs when an agent stops
- `agent-task-start`: Runs before an agent executes a task
- `agent-task-complete`: Runs after an agent completes a task

## Hook Structure

### Directory Layout

```
hooks/
├── hook-name/
│   ├── manifest.yaml          # Hook metadata and configuration
│   ├── README.md              # Hook documentation
│   ├── hook.go                # Main hook implementation
│   ├── hook_test.go           # Hook tests
│   └── config/                # Hook-specific configuration
│       └── config.yaml
```

## Core Principles

### 1. Non-Blocking Execution

Hooks should execute quickly and not block the main application flow:

```go
// ✓ Good - Async execution
func (h *Hook) Execute(event Event) error {
    go h.processAsync(event)
    return nil
}

// ✗ Bad - Blocking execution
func (h *Hook) Execute(event Event) error {
    h.processSync(event)  // Blocks the main flow
    return nil
}
```

### 2. Error Handling

Hooks should handle errors gracefully without affecting the main application:

```go
func (h *Hook) Execute(event Event) error {
    defer func() {
        if r := recover(); r != nil {
            h.logger.Error("hook panicked", fmt.Errorf("%v", r))
        }
    }()
    
    if err := h.process(event); err != nil {
        h.logger.Error("hook execution failed", err)
        // Don't propagate error - hooks should not break the main flow
        return nil
    }
    
    return nil
}
```

### 3. Idempotency

Hooks should be idempotent - executing multiple times should produce the same result:

```go
// ✓ Good - Idempotent
func (h *Hook) Execute(event Event) error {
    // Check if already processed
    if h.isProcessed(event.ID) {
        return nil
    }
    
    h.process(event)
    h.markAsProcessed(event.ID)
    
    return nil
}
```

### 4. Logging

Hooks should log their execution for debugging:

```go
func (h *Hook) Execute(event Event) error {
    h.logger.Info("hook executing", map[string]interface{}{
        "hook": h.Name(),
        "event_type": event.Type,
        "event_id": event.ID,
    })
    
    start := time.Now()
    err := h.process(event)
    duration := time.Since(start)
    
    if err != nil {
        h.logger.Error("hook failed", err)
    } else {
        h.logger.Info("hook completed", map[string]interface{}{
            "duration_ms": duration.Milliseconds(),
        })
    }
    
    return nil
}
```

## Hook Manifest

Every hook must have a `manifest.yaml` file:

```yaml
name: "hook-name"
version: "1.0.0"
description: "Hook description"
author: "Author Name"
license: "MIT"

# Events this hook listens to
events:
  - "event-type-1"
  - "event-type-2"

# Execution mode
mode: "async"  # or "sync"

# Timeout in seconds
timeout: 10

# Configuration
config:
  enabled: true
  log_level: "info"
  retry_count: 3

# Dependencies
dependencies:
  go: "1.26"
  cockpit: "0.2.0"
```

## Implementation Example

```go
package hook

import (
    "context"
    "fmt"
    "time"
)

type Event struct {
    Type      string
    ID        string
    Timestamp time.Time
    Data      map[string]interface{}
}

type Hook struct {
    config Config
    logger Logger
}

func NewHook(cfg Config, log Logger) *Hook {
    return &Hook{
        config: cfg,
        logger: log,
    }
}

func (h *Hook) Name() string {
    return "hook-name"
}

func (h *Hook) Description() string {
    return "Hook description"
}

func (h *Hook) Events() []string {
    return []string{"event-type-1", "event-type-2"}
}

func (h *Hook) Execute(event Event) error {
    h.logger.Info("hook executing", map[string]interface{}{
        "hook": h.Name(),
        "event_type": event.Type,
    })
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.config.Timeout)*time.Second)
    defer cancel()
    
    // Execute with error handling
    if err := h.process(ctx, event); err != nil {
        h.logger.Error("hook processing failed", err)
        // Don't propagate - hooks shouldn't break main flow
        return nil
    }
    
    h.logger.Info("hook completed", map[string]interface{}{
        "event_type": event.Type,
    })
    
    return nil
}

func (h *Hook) process(ctx context.Context, event Event) error {
    select {
    case <-ctx.Done():
        return fmt.Errorf("hook execution timeout")
    default:
    }
    
    // Process the event
    // Implementation specific to the hook
    
    return nil
}
```

## Testing

Hooks should have comprehensive tests:

```go
func TestHook_Execute(t *testing.T) {
    cfg := Config{Timeout: 10}
    logger := NewMockLogger()
    hook := NewHook(cfg, logger)
    
    tests := []struct {
        name  string
        event Event
    }{
        {
            name: "valid event",
            event: Event{
                Type: "event-type-1",
                ID:   "event-1",
            },
        },
        {
            name: "another event",
            event: Event{
                Type: "event-type-2",
                ID:   "event-2",
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := hook.Execute(tt.event)
            if err != nil {
                t.Errorf("Execute() error = %v", err)
            }
        })
    }
}
```

## Hook Chaining

Multiple hooks can be chained together:

```go
type HookChain struct {
    hooks  []Hook
    logger Logger
}

func (hc *HookChain) Execute(event Event) error {
    for _, hook := range hc.hooks {
        if err := hook.Execute(event); err != nil {
            hc.logger.Error("hook in chain failed", err)
            // Continue with next hook - don't break the chain
        }
    }
    return nil
}
```

## Best Practices Checklist

- [ ] Hook executes asynchronously
- [ ] Hook has error handling
- [ ] Hook is idempotent
- [ ] Hook has comprehensive logging
- [ ] Hook has unit tests with >90% coverage
- [ ] Hook has a manifest.yaml file
- [ ] Hook doesn't block main application
- [ ] Hook has timeout configuration
- [ ] Hook has clear documentation
- [ ] Hook follows Go conventions

## Common Patterns

### Async Pattern with Goroutine

```go
func (h *Hook) Execute(event Event) error {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                h.logger.Error("hook panicked", fmt.Errorf("%v", r))
            }
        }()
        
        if err := h.process(event); err != nil {
            h.logger.Error("async hook failed", err)
        }
    }()
    
    return nil
}
```

### Retry Pattern

```go
func (h *Hook) ExecuteWithRetry(event Event, maxRetries int) error {
    var lastErr error
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        if err := h.process(event); err == nil {
            return nil
        } else {
            lastErr = err
            time.Sleep(time.Duration(math.Pow(2, float64(attempt))) * time.Second)
        }
    }
    
    h.logger.Error("hook failed after retries", lastErr)
    return nil  // Don't propagate error
}
```

### Conditional Execution

```go
func (h *Hook) Execute(event Event) error {
    // Only execute for specific event types
    if !h.shouldExecute(event) {
        return nil
    }
    
    return h.process(event)
}

func (h *Hook) shouldExecute(event Event) bool {
    for _, eventType := range h.Events() {
        if event.Type == eventType {
            return true
        }
    }
    return false
}
```

## Performance Considerations

- Use goroutines for async execution
- Implement timeouts to prevent hanging
- Monitor hook execution time
- Avoid blocking operations
- Use connection pooling

## Security Considerations

- Validate event data
- Don't log sensitive information
- Use secure communication channels
- Implement rate limiting
- Regularly update dependencies

## Troubleshooting

### Hook Not Executing

1. Check hook is registered
2. Verify manifest.yaml is valid
3. Check event type matches
4. Verify hook configuration
5. Check logs for errors

### Hook Timeout

1. Increase timeout in configuration
2. Optimize hook implementation
3. Check for blocking operations
4. Implement cancellation support

### Hook Blocking Main Flow

1. Ensure hook executes asynchronously
2. Implement proper error handling
3. Add timeout configuration
4. Monitor hook execution time
