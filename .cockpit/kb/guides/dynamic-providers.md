---
title: "Dynamic Providers System"
description: "Guide for configuring and extending AI providers in AICockpit"
tags: ["providers", "configuration", "extensibility", "devin", "goose", "claude-code", "github-copilot"]
author: "AICockpit Team"
version: "1.0"
---

# Dynamic Providers System

## Overview

AICockpit uses a dynamic provider system that allows you to:
- Add new AI providers at any time
- Configure which features each provider supports
- Automatically install components based on provider capabilities
- Extend AICockpit with custom providers

All provider configuration is stored in a single YAML file: `~/.cockpit/providers.yaml`

## Providers Configuration File

### Location

```
~/.cockpit/providers.yaml
```

### Structure

```yaml
version: "1.0"
description: "AI Providers Configuration for AICockpit"

providers:
  provider-name:
    enabled: true
    name: "Display Name"
    description: "Provider description"
    workspace: "~/.provider-workspace"
    version: "1.0.0"
    
    features:
      agents:
        enabled: true
        path: "agents"
        description: "Autonomous agents"
      
      skills:
        enabled: true
        path: "skills"
        description: "Reusable capabilities"
      
      hooks:
        enabled: true
        path: "hooks"
        description: "Event handlers"
      
      workflows:
        enabled: true
        path: "workflows"
        description: "Automated workflows"
      
      memories:
        enabled: true
        path: "memories"
        description: "Memory management"
      
      kb:
        enabled: true
        path: "kb"
        description: "Knowledge base"

features:
  agents:
    description: "Autonomous AI agents for task execution"
    example: "cockpit-builder agent"
  
  skills:
    description: "Reusable capabilities that agents can leverage"
    example: "go-development skill"
  
  hooks:
    description: "Event handlers that execute in response to system events"
    example: "cockpit-first hook"
  
  workflows:
    description: "Automated workflows and automation patterns"
    example: "ci-cd workflow"
  
  memories:
    description: "Memory and context management for agents"
    example: "conversation memory"
  
  kb:
    description: "Knowledge base for documentation and learning"
    example: "guides and troubleshooting docs"
```

## Default Providers

AICockpit comes with 4 default providers pre-configured:

### 1. Devin

**Workspace**: `~/.cockpit`

**Supported Features**:
- ✅ Agents
- ✅ Skills
- ✅ Hooks
- ✅ Workflows
- ✅ Memories
- ✅ Knowledge Base

**Description**: Autonomous AI agent for software engineering

### 2. Goose

**Workspace**: `~/.goose`

**Supported Features**:
- ✅ Agents
- ✅ Skills
- ✅ Hooks
- ✅ Workflows
- ❌ Memories (not supported)
- ✅ Knowledge Base

**Description**: AI agent for code generation and automation

### 3. Claude Code

**Workspace**: `~/.claude-code`

**Supported Features**:
- ❌ Agents (not supported)
- ✅ Skills
- ✅ Hooks
- ❌ Workflows (not supported)
- ✅ Memories
- ✅ Knowledge Base

**Description**: Claude AI integrated with code editing

### 4. GitHub Copilot

**Workspace**: `~/.github-copilot`

**Supported Features**:
- ❌ Agents (not supported)
- ✅ Skills
- ❌ Hooks (not supported)
- ❌ Workflows (not supported)
- ✅ Memories
- ✅ Knowledge Base

**Description**: GitHub's AI pair programmer

## Adding a New Provider

### Step 1: Edit providers.yaml

Add your provider to `~/.cockpit/providers.yaml`:

```yaml
providers:
  my-provider:
    enabled: true
    name: "My Custom Provider"
    description: "Description of my provider"
    workspace: "~/.my-provider"
    version: "1.0.0"
    
    features:
      agents:
        enabled: true
        path: "agents"
        description: "Autonomous agents"
      
      skills:
        enabled: true
        path: "skills"
        description: "Reusable capabilities"
      
      hooks:
        enabled: true
        path: "hooks"
        description: "Event handlers"
      
      workflows:
        enabled: false
        path: "workflows"
        description: "Automated workflows (not supported)"
      
      memories:
        enabled: true
        path: "memories"
        description: "Memory management"
      
      kb:
        enabled: true
        path: "kb"
        description: "Knowledge base"
```

### Step 2: Run Setup

```bash
cockpit setup --provider my-provider
```

This will:
1. Create the workspace directory
2. Create directory structure for supported features
3. Copy components for supported features
4. Create configuration files

## Enabling/Disabling Providers

### Enable a Provider

```bash
cockpit providers enable devin
```

### Disable a Provider

```bash
cockpit providers disable goose
```

### List All Providers

```bash
cockpit providers list
```

### List Enabled Providers

```bash
cockpit providers list --enabled
```

## Feature Support Matrix

| Feature | Devin | Goose | Claude Code | GitHub Copilot |
|---------|-------|-------|-------------|----------------|
| Agents | ✅ | ✅ | ❌ | ❌ |
| Skills | ✅ | ✅ | ✅ | ✅ |
| Hooks | ✅ | ✅ | ✅ | ❌ |
| Workflows | ✅ | ✅ | ❌ | ❌ |
| Memories | ✅ | ❌ | ✅ | ✅ |
| Knowledge Base | ✅ | ✅ | ✅ | ✅ |

## Setup Process

When running `cockpit setup`, the system:

1. **Reads providers.yaml** to get all available providers
2. **Prompts for selection** of which providers to enable
3. **For each enabled provider**:
   - Creates workspace directory
   - Creates feature directories (based on enabled features)
   - Copies default components
   - Creates configuration files
   - Creates manifest file
4. **Validates installation** for each provider

### Setup Example

```bash
$ cockpit setup

Available Providers:
1. devin (Autonomous AI agent for software engineering)
2. goose (AI agent for code generation and automation)
3. claude-code (Claude AI integrated with code editing)
4. github-copilot (GitHub's AI pair programmer)

Select providers to enable (comma-separated, or 'all'): all

Setting up devin...
✓ Workspace created at ~/.cockpit
✓ Directories created
✓ Components copied
✓ Configuration created

Setting up goose...
✓ Workspace created at ~/.goose
✓ Directories created
✓ Components copied
✓ Configuration created

Setting up claude-code...
✓ Workspace created at ~/.claude-code
✓ Directories created
✓ Components copied
✓ Configuration created

Setting up github-copilot...
✓ Workspace created at ~/.github-copilot
✓ Directories created
✓ Components copied
✓ Configuration created

✓ All providers configured successfully!
```

## Programmatic Usage

### Load Providers Configuration

```go
import "github.com/lleite/aicockpit/internal/providers"

config, err := providers.LoadProvidersConfig("~/.cockpit/providers.yaml")
if err != nil {
    log.Fatal(err)
}
```

### Get Enabled Providers

```go
enabled := config.GetEnabledProviders()
for _, provider := range enabled {
    fmt.Printf("Provider: %s\n", provider.Name)
    fmt.Printf("Workspace: %s\n", provider.GetWorkspacePath())
    fmt.Printf("Features: %v\n", provider.GetSupportedFeatures())
}
```

### Check Feature Support

```go
devin := config.GetProvider("devin")
if devin.SupportsFeature("agents") {
    agentsPath := devin.GetFeaturePath("agents")
    fmt.Printf("Agents path: %s\n", agentsPath)
}
```

### Add New Provider

```go
newProvider := &providers.Provider{
    Enabled:     true,
    Name:        "My Provider",
    Description: "My custom provider",
    Workspace:   "~/.my-provider",
    Version:     "1.0.0",
    Features: map[string]*providers.FeatureConfig{
        "agents": {Enabled: true, Path: "agents"},
        "skills": {Enabled: true, Path: "skills"},
    },
}

config.AddProvider("my-provider", newProvider)
providers.SaveProvidersConfig("~/.cockpit/providers.yaml", config)
```

## Best Practices

### 1. Feature Consistency

Keep feature support consistent across providers when possible:

```yaml
# ✓ Good - Clear feature support
providers:
  provider1:
    features:
      agents: {enabled: true}
      skills: {enabled: true}
  
  provider2:
    features:
      agents: {enabled: true}
      skills: {enabled: true}

# ✗ Bad - Inconsistent features
providers:
  provider1:
    features:
      agents: {enabled: true}
      skills: {enabled: false}
  
  provider2:
    features:
      agents: {enabled: false}
      skills: {enabled: true}
```

### 2. Clear Documentation

Document why features are enabled/disabled:

```yaml
providers:
  my-provider:
    features:
      agents:
        enabled: false
        description: "Agents not supported in this provider"
      
      memories:
        enabled: true
        description: "Full memory support with context window"
```

### 3. Validate Configuration

Always validate your providers.yaml:

```bash
cockpit providers validate
```

### 4. Backup Configuration

Keep backups of your providers.yaml:

```bash
cp ~/.cockpit/providers.yaml ~/.cockpit/providers.yaml.backup
```

## Troubleshooting

### Provider Not Found

```bash
# List all available providers
cockpit providers list

# Check providers.yaml
cat ~/.cockpit/providers.yaml
```

### Feature Not Supported

Check the feature matrix in providers.yaml:

```bash
# View provider details
cockpit providers info devin

# Check supported features
cockpit providers features devin
```

### Setup Failed for Provider

Check logs:

```bash
tail -f ~/.cockpit/logs/cockpit-*.log
```

### Invalid Configuration

Validate the YAML syntax:

```bash
cockpit providers validate
```

## Examples

### Example 1: Add Custom Provider

```yaml
# Add to ~/.cockpit/providers.yaml
providers:
  my-ai:
    enabled: true
    name: "My AI Provider"
    description: "Custom AI provider"
    workspace: "~/.my-ai"
    version: "1.0.0"
    
    features:
      agents:
        enabled: true
        path: "agents"
      
      skills:
        enabled: true
        path: "skills"
      
      hooks:
        enabled: true
        path: "hooks"
      
      workflows:
        enabled: true
        path: "workflows"
      
      memories:
        enabled: true
        path: "memories"
      
      kb:
        enabled: true
        path: "kb"
```

### Example 2: Disable Unsupported Features

```yaml
# For a provider that doesn't support workflows
providers:
  my-provider:
    features:
      workflows:
        enabled: false
        description: "Workflows not supported in this version"
```

### Example 3: Custom Feature Paths

```yaml
# Use custom paths for features
providers:
  my-provider:
    features:
      agents:
        enabled: true
        path: "custom/agents"  # Custom path
      
      skills:
        enabled: true
        path: "lib/skills"     # Different path
```

## Migration from Static to Dynamic Providers

If you're upgrading from the static provider system:

1. **Backup your configuration**
   ```bash
   cp ~/.cockpit/config.yaml ~/.cockpit/config.yaml.backup
   ```

2. **Update to new version**
   ```bash
   cockpit update
   ```

3. **Migrate configuration**
   ```bash
   cockpit migrate-providers
   ```

4. **Verify setup**
   ```bash
   cockpit setup --verify --all-providers
   ```

## API Reference

### ProvidersConfig

```go
type ProvidersConfig struct {
    Version     string
    Description string
    Providers   map[string]*Provider
    Features    map[string]*Feature
}

// Methods
func (c *ProvidersConfig) GetProvider(name string) *Provider
func (c *ProvidersConfig) GetEnabledProviders() []*Provider
func (c *ProvidersConfig) GetProviderNames() []string
func (c *ProvidersConfig) EnableProvider(name string) error
func (c *ProvidersConfig) DisableProvider(name string) error
func (c *ProvidersConfig) IsProviderEnabled(name string) bool
func (c *ProvidersConfig) AddProvider(name string, provider *Provider) error
func (c *ProvidersConfig) RemoveProvider(name string) error
func (c *ProvidersConfig) ValidateConfig() error
```

### Provider

```go
type Provider struct {
    Enabled     bool
    Name        string
    Description string
    Workspace   string
    Version     string
    Features    map[string]*FeatureConfig
}

// Methods
func (p *Provider) GetSupportedFeatures() []string
func (p *Provider) SupportsFeature(feature string) bool
func (p *Provider) GetFeaturePath(feature string) string
func (p *Provider) GetWorkspacePath() string
func (p *Provider) Validate() error
```

### Functions

```go
func LoadProvidersConfig(configPath string) (*ProvidersConfig, error)
func SaveProvidersConfig(configPath string, config *ProvidersConfig) error
```
