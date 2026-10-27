---
title: "Skill Best Practices"
description: "Guidelines for creating effective skills in AICockpit"
tags: ["skills", "best-practices", "development"]
author: "AICockpit Team"
version: "1.0"
---

# Skill Best Practices

> **Nota:** Este documento descreve **skills internas do AICockpit** (implementadas em Go). Para criar **skills do Devin** (arquivos SKILL.md para uso com o provider Devin), consulte a documentação específica em `docs/providers/devin/SKILLS.md`.

## Overview

Skills are reusable capabilities that agents and other components can leverage. Skills encapsulate specific functionality and can be composed together to create powerful workflows.

## Skill Structure

### Directory Layout

```
skills/
├── skill-name/
│   ├── manifest.yaml          # Skill metadata and configuration
│   ├── README.md              # Skill documentation
│   ├── skill.go               # Main skill implementation
│   ├── skill_test.go          # Skill tests
│   ├── handlers/              # Capability handlers
│   │   ├── handler1.go
│   │   └── handler2.go
│   └── config/                # Skill-specific configuration
│       └── config.yaml
```

## Core Principles

### 1. Single Capability

Each skill should provide one well-defined capability. Skills should be focused and composable.

```go
// ✓ Good - Single capability
type FileSearchSkill struct {
    // Searches files by pattern
}

// ✗ Bad - Multiple unrelated capabilities
type UtilitySkill struct {
    // Does file search, network requests, database queries, etc.
}
```

### 2. Clear Interface

Skills should have a clear, well-documented interface:

```go
type Skill interface {
    // Name returns the skill name
    Name() string
    
    // Description returns the skill description
    Description() string
    
    // Execute executes the skill with given parameters
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
    
    // Validate validates the input parameters
    Validate(params map[string]interface{}) error
}
```

### 3. Context Awareness

Skills should respect context for cancellation and timeouts:

```go
func (s *Skill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Check if context is already cancelled
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Use context for timeouts
    resultChan := make(chan interface{})
    go func() {
        resultChan <- s.process(params)
    }()
    
    select {
    case result := <-resultChan:
        return result, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### 4. Parameter Validation

Always validate input parameters:

```go
func (s *Skill) Validate(params map[string]interface{}) error {
    if params == nil {
        return fmt.Errorf("parameters cannot be nil")
    }
    
    query, ok := params["query"].(string)
    if !ok {
        return fmt.Errorf("query parameter must be a string")
    }
    
    if query == "" {
        return fmt.Errorf("query cannot be empty")
    }
    
    return nil
}
```

## Skill Manifest

Every skill must have a `manifest.yaml` file:

```yaml
name: "skill-name"
version: "1.0.0"
description: "Skill description"
author: "Author Name"
license: "MIT"

# Skill capabilities
capabilities:
  - name: "capability-1"
    description: "Description of capability 1"
    parameters:
      param1:
        type: "string"
        required: true
        description: "Parameter description"
      param2:
        type: "integer"
        required: false
        default: 10

# Configuration
config:
  timeout: 30
  max_retries: 3
  log_level: "info"

# Dependencies
dependencies:
  go: "1.26"
  cockpit: "0.2.0"
```

## Implementation Example

```go
package skill

import (
    "context"
    "fmt"
)

type FileSearchSkill struct {
    config Config
    logger Logger
}

func NewFileSearchSkill(cfg Config, log Logger) *FileSearchSkill {
    return &FileSearchSkill{
        config: cfg,
        logger: log,
    }
}

func (s *FileSearchSkill) Name() string {
    return "file-search"
}

func (s *FileSearchSkill) Description() string {
    return "Search files by pattern"
}

func (s *FileSearchSkill) Validate(params map[string]interface{}) error {
    pattern, ok := params["pattern"].(string)
    if !ok || pattern == "" {
        return fmt.Errorf("pattern parameter is required and must be a string")
    }
    return nil
}

func (s *FileSearchSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    if err := s.Validate(params); err != nil {
        return nil, err
    }
    
    pattern := params["pattern"].(string)
    
    s.logger.Info("searching files", map[string]interface{}{
        "pattern": pattern,
    })
    
    // Check context
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Execute search
    results, err := s.search(ctx, pattern)
    if err != nil {
        s.logger.Error("search failed", err)
        return nil, fmt.Errorf("search failed: %w", err)
    }
    
    s.logger.Info("search completed", map[string]interface{}{
        "results_count": len(results),
    })
    
    return results, nil
}

func (s *FileSearchSkill) search(ctx context.Context, pattern string) ([]string, error) {
    // Implementation
    return []string{}, nil
}
```

## Testing

Skills should have comprehensive tests:

```go
func TestFileSearchSkill_Execute(t *testing.T) {
    cfg := Config{Timeout: 30}
    logger := NewMockLogger()
    skill := NewFileSearchSkill(cfg, logger)
    
    tests := []struct {
        name    string
        params  map[string]interface{}
        wantErr bool
    }{
        {
            name:    "valid pattern",
            params:  map[string]interface{}{"pattern": "*.go"},
            wantErr: false,
        },
        {
            name:    "empty pattern",
            params:  map[string]interface{}{"pattern": ""},
            wantErr: true,
        },
        {
            name:    "missing pattern",
            params:  map[string]interface{}{},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            _, err := skill.Execute(ctx, tt.params)
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Skill Composition

Skills can be composed to create more complex workflows:

```go
type CompositeSkill struct {
    skills []Skill
    logger Logger
}

func (cs *CompositeSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    var result interface{}
    
    for _, skill := range cs.skills {
        output, err := skill.Execute(ctx, params)
        if err != nil {
            return nil, fmt.Errorf("skill %s failed: %w", skill.Name(), err)
        }
        result = output
    }
    
    return result, nil
}
```

## Best Practices Checklist

- [ ] Skill has a single, well-defined capability
- [ ] Skill has clear parameter validation
- [ ] Skill respects context for cancellation
- [ ] Skill has comprehensive error handling
- [ ] Skill has unit tests with >90% coverage
- [ ] Skill has a manifest.yaml file
- [ ] Skill is composable with other skills
- [ ] Skill logs important actions
- [ ] Skill has clear documentation
- [ ] Skill follows Go conventions

## Performance Optimization

- Use goroutines for concurrent operations
- Implement caching for expensive operations
- Validate parameters early
- Use connection pooling
- Monitor resource usage

## Security Considerations

- Validate all input parameters
- Don't log sensitive information
- Use secure communication channels
- Implement rate limiting
- Use authentication where needed
- Regularly update dependencies

## Common Patterns

### Caching Pattern

```go
type CachedSkill struct {
    skill Skill
    cache map[string]interface{}
    mu    sync.RWMutex
}

func (cs *CachedSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    key := hashParams(params)
    
    cs.mu.RLock()
    if cached, ok := cs.cache[key]; ok {
        cs.mu.RUnlock()
        return cached, nil
    }
    cs.mu.RUnlock()
    
    result, err := cs.skill.Execute(ctx, params)
    if err != nil {
        return nil, err
    }
    
    cs.mu.Lock()
    cs.cache[key] = result
    cs.mu.Unlock()
    
    return result, nil
}
```

### Retry Pattern

```go
func (s *Skill) ExecuteWithRetry(ctx context.Context, params map[string]interface{}, maxRetries int) (interface{}, error) {
    var lastErr error
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        result, err := s.Execute(ctx, params)
        if err == nil {
            return result, nil
        }
        lastErr = err
        
        // Exponential backoff
        backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
        select {
        case <-time.After(backoff):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    return nil, lastErr
}
```

## Troubleshooting

### Skill Not Executing

1. Check skill is registered
2. Verify manifest.yaml is valid
3. Check skill configuration
4. Verify parameters are correct
5. Check logs for errors

### Skill Timeout

1. Increase timeout in configuration
2. Optimize skill implementation
3. Check for blocking operations
4. Implement cancellation support

### Memory Leaks

1. Check for goroutine leaks
2. Verify caches are cleared
3. Monitor resource usage
4. Profile with pprof
