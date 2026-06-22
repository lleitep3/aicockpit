---
title: "AI Providers Configuration"
description: "Configure AICockpit for multiple AI providers (Devin, Goose, Claude Code, GitHub Copilot)"
tags: ["providers", "configuration", "devin", "goose", "claude-code", "github-copilot"]
author: "AICockpit Team"
version: "1.0"
---

# AI Providers Configuration

## Overview

AICockpit supports multiple AI providers, allowing you to use different AI agents simultaneously. Each provider has its own workspace and configuration, but they all share the same AICockpit core functionality.

## Supported Providers

### 1. Devin

**Description**: Autonomous AI agent for software engineering

**Workspace**: `~/.cockpit/`

**Installation Path**: `~/.cockpit/agents/`, `~/.cockpit/skills/`, `~/.cockpit/hooks/`

**Configuration**: `~/.cockpit/config.yaml`

**Features**:
- Full autonomy for code tasks
- Direct file system access
- Git integration
- Command execution

### 2. Goose

**Description**: AI agent for code generation and automation

**Workspace**: `~/.goose/`

**Installation Path**: `~/.goose/agents/`, `~/.goose/skills/`, `~/.goose/hooks/`

**Configuration**: `~/.goose/config.yaml`

**Features**:
- Code generation
- Automation workflows
- Template support
- Plugin system

### 3. Claude Code

**Description**: Claude AI integrated with code editing

**Workspace**: `~/.claude-code/`

**Installation Path**: `~/.claude-code/agents/`, `~/.claude-code/skills/`, `~/.claude-code/hooks/`

**Configuration**: `~/.claude-code/config.yaml`

**Features**:
- Advanced code understanding
- Multi-file editing
- Context awareness
- Integration with IDEs

### 4. GitHub Copilot

**Description**: GitHub's AI pair programmer

**Workspace**: `~/.github-copilot/`

**Installation Path**: `~/.github-copilot/agents/`, `~/.github-copilot/skills/`, `~/.github-copilot/hooks/`

**Configuration**: `~/.github-copilot/config.yaml`

**Features**:
- Code completion
- Suggestion generation
- Documentation generation
- Test generation

## Installation Paths by Provider

### Directory Structure

```
Home Directory
├── .cockpit/                    # Devin
│   ├── agents/
│   ├── skills/
│   ├── hooks/
│   ├── kb/
│   ├── logs/
│   ├── config.yaml
│   └── manifest.yaml
├── .goose/                      # Goose
│   ├── agents/
│   ├── skills/
│   ├── hooks/
│   ├── kb/
│   ├── logs/
│   ├── config.yaml
│   └── manifest.yaml
├── .claude-code/                # Claude Code
│   ├── agents/
│   ├── skills/
│   ├── hooks/
│   ├── kb/
│   ├── logs/
│   ├── config.yaml
│   └── manifest.yaml
└── .github-copilot/             # GitHub Copilot
    ├── agents/
    ├── skills/
    ├── hooks/
    ├── kb/
    ├── logs/
    ├── config.yaml
    └── manifest.yaml
```

## Installation Matrix

### Component Installation Locations

| Component | Devin | Goose | Claude Code | GitHub Copilot |
|-----------|-------|-------|-------------|----------------|
| cockpit-builder agent | ~/.cockpit/agents/ | ~/.goose/agents/ | ~/.claude-code/agents/ | ~/.github-copilot/agents/ |
| go-development skill | ~/.cockpit/skills/ | ~/.goose/skills/ | ~/.claude-code/skills/ | ~/.github-copilot/skills/ |
| cockpit-first hook | ~/.cockpit/hooks/ | ~/.goose/hooks/ | ~/.claude-code/hooks/ | ~/.github-copilot/hooks/ |
| KB documents | ~/.cockpit/kb/ | ~/.goose/kb/ | ~/.claude-code/kb/ | ~/.github-copilot/kb/ |
| Configuration | ~/.cockpit/config.yaml | ~/.goose/config.yaml | ~/.claude-code/config.yaml | ~/.github-copilot/config.yaml |
| Manifest | ~/.cockpit/manifest.yaml | ~/.goose/manifest.yaml | ~/.claude-code/manifest.yaml | ~/.github-copilot/manifest.yaml |

## Configuration

### Devin Configuration

```yaml
# ~/.cockpit/config.yaml
version: "0.2.0"
language: "en-us"
log_level: "info"
ai_provider: "devin"

kb:
  roots:
    - ~/.cockpit/kb

agents:
  enabled: true
  
skills:
  enabled: true
  
hooks:
  enabled: true
```

### Goose Configuration

```yaml
# ~/.goose/config.yaml
version: "0.2.0"
language: "en-us"
log_level: "info"
ai_provider: "goose"

kb:
  roots:
    - ~/.goose/kb

agents:
  enabled: true
  
skills:
  enabled: true
  
hooks:
  enabled: true
```

### Claude Code Configuration

```yaml
# ~/.claude-code/config.yaml
version: "0.2.0"
language: "en-us"
log_level: "info"
ai_provider: "claude-code"

kb:
  roots:
    - ~/.claude-code/kb

agents:
  enabled: true
  
skills:
  enabled: true
  
hooks:
  enabled: true
```

### GitHub Copilot Configuration

```yaml
# ~/.github-copilot/config.yaml
version: "0.2.0"
language: "en-us"
log_level: "info"
ai_provider: "github-copilot"

kb:
  roots:
    - ~/.github-copilot/kb

agents:
  enabled: true
  
skills:
  enabled: true
  
hooks:
  enabled: true
```

## Multi-Provider Setup

### Installation for All Providers

```bash
# 1. Install for Devin
cockpit setup --provider devin

# 2. Install for Goose
cockpit setup --provider goose

# 3. Install for Claude Code
cockpit setup --provider claude-code

# 4. Install for GitHub Copilot
cockpit setup --provider github-copilot

# Or install for all at once
cockpit setup --all-providers
```

### Shared Components

Some components can be shared across providers:

```bash
# Install agent in all providers
cockpit agent install cockpit-builder --all-providers

# Install skill in all providers
cockpit skill install go-development --all-providers

# Install hook in all providers
cockpit hook install cockpit-first --all-providers
```

### Provider-Specific Installation

```bash
# Install only for Devin
cockpit agent install cockpit-builder --provider devin

# Install only for Goose
cockpit agent install cockpit-builder --provider goose

# Install for multiple specific providers
cockpit agent install cockpit-builder --providers devin,goose,claude-code
```

## Verification

### Check Installation for All Providers

```bash
# Check Devin installation
cockpit setup --verify --provider devin

# Check Goose installation
cockpit setup --verify --provider goose

# Check Claude Code installation
cockpit setup --verify --provider claude-code

# Check GitHub Copilot installation
cockpit setup --verify --provider github-copilot

# Check all installations
cockpit setup --verify --all-providers
```

### Verify Components

```bash
# List agents in all providers
cockpit agent list --all-providers

# List skills in all providers
cockpit skill list --all-providers

# List hooks in all providers
cockpit hook list --all-providers

# List KB documents in all providers
cockpit kb list --all-providers
```

## Provider Detection

AICockpit automatically detects which providers are available:

```bash
# Detect available providers
cockpit providers detect

# Show available providers
cockpit providers list

# Show active provider
cockpit providers current
```

## Switching Providers

```bash
# Switch to Devin
cockpit providers switch devin

# Switch to Goose
cockpit providers switch goose

# Switch to Claude Code
cockpit providers switch claude-code

# Switch to GitHub Copilot
cockpit providers switch github-copilot
```

## Manifest Management

Each provider has its own manifest:

```bash
# Manage Devin manifest
cockpit manifest list --provider devin
cockpit manifest add agent cockpit-builder --provider devin

# Manage Goose manifest
cockpit manifest list --provider goose
cockpit manifest add agent cockpit-builder --provider goose

# Manage all manifests
cockpit manifest list --all-providers
```

## Uninstallation

### Remove from Specific Provider

```bash
# Remove from Devin only
cockpit agent uninstall cockpit-builder --provider devin

# Remove from Goose only
cockpit agent uninstall cockpit-builder --provider goose
```

### Remove from All Providers

```bash
# Remove from all providers
cockpit agent uninstall cockpit-builder --all-providers
```

### Complete Provider Cleanup

```bash
# Remove all components from Devin
cockpit setup --cleanup --provider devin

# Remove all components from all providers
cockpit setup --cleanup --all-providers
```

## Best Practices

### 1. Consistent Installation

Keep the same components installed across all providers:

```bash
# Install same agent in all providers
cockpit agent install cockpit-builder --all-providers

# Install same skill in all providers
cockpit skill install go-development --all-providers
```

### 2. Configuration Sync

Keep configurations synchronized:

```bash
# Sync configuration across providers
cockpit config sync --all-providers
```

### 3. Knowledge Base Sharing

Share knowledge base across providers:

```yaml
# ~/.cockpit/config.yaml
kb:
  roots:
    - ~/.cockpit/kb
    - ~/.shared-kb  # Shared location

# ~/.goose/config.yaml
kb:
  roots:
    - ~/.goose/kb
    - ~/.shared-kb  # Same shared location
```

### 4. Regular Verification

Verify installations regularly:

```bash
# Weekly verification
cockpit setup --verify --all-providers

# Check for missing components
cockpit manifest validate --all-providers
```

## Troubleshooting

### Provider Not Found

```bash
# Detect available providers
cockpit providers detect

# List installed providers
cockpit providers list
```

### Installation Failed

```bash
# Check installation logs
tail -f ~/.cockpit/logs/cockpit-*.log
tail -f ~/.goose/logs/goose-*.log

# Verify installation
cockpit setup --verify --provider devin
```

### Component Not Available

```bash
# Check component installation
cockpit agent list --all-providers
cockpit skill list --all-providers

# Reinstall component
cockpit agent install cockpit-builder --all-providers
```

### Manifest Corruption

```bash
# Validate manifest
cockpit manifest validate --provider devin

# Rebuild manifest
cockpit manifest rebuild --provider devin
```

## Performance Considerations

### Storage Usage

Each provider requires separate storage:

- **Per Provider**: ~500MB (agents, skills, hooks, KB)
- **Total for 4 Providers**: ~2GB

### Optimization

```bash
# Share knowledge base to save space
cockpit config set kb.shared-root ~/.shared-kb --all-providers

# Symlink agents across providers
ln -s ~/.cockpit/agents/cockpit-builder ~/.goose/agents/cockpit-builder
```

## Security Considerations

### Isolation

Each provider is isolated:

```
~/.cockpit/       # Devin's workspace
~/.goose/         # Goose's workspace
~/.claude-code/   # Claude Code's workspace
~/.github-copilot/ # GitHub Copilot's workspace
```

### Access Control

```bash
# Restrict provider access
chmod 700 ~/.cockpit
chmod 700 ~/.goose
chmod 700 ~/.claude-code
chmod 700 ~/.github-copilot
```

### Credential Management

Store credentials securely:

```bash
# Use environment variables
export DEVIN_API_KEY="..."
export GOOSE_API_KEY="..."
export CLAUDE_CODE_API_KEY="..."
export GITHUB_COPILOT_TOKEN="..."
```

## Migration Between Providers

### Export Configuration

```bash
# Export from Devin
cockpit config export --provider devin > devin-config.yaml

# Export manifest
cockpit manifest export --provider devin > devin-manifest.yaml
```

### Import Configuration

```bash
# Import to Goose
cockpit config import devin-config.yaml --provider goose

# Import manifest
cockpit manifest import devin-manifest.yaml --provider goose
```

## Examples

### Example 1: Install for All Providers

```bash
# Install AICockpit for all providers
cockpit setup --all-providers

# Verify installation
cockpit setup --verify --all-providers

# List installed components
cockpit agent list --all-providers
cockpit skill list --all-providers
cockpit hook list --all-providers
```

### Example 2: Install Specific Component Everywhere

```bash
# Install cockpit-builder agent in all providers
cockpit agent install cockpit-builder --all-providers

# Verify installation
cockpit agent list --all-providers
```

### Example 3: Provider-Specific Setup

```bash
# Setup only Devin
cockpit setup --provider devin

# Setup only Goose
cockpit setup --provider goose

# Verify each
cockpit setup --verify --provider devin
cockpit setup --verify --provider goose
```

### Example 4: Sync Across Providers

```bash
# Install in Devin
cockpit agent install cockpit-builder --provider devin

# Sync to other providers
cockpit agent sync cockpit-builder --from devin --to goose,claude-code,github-copilot
```
