---
title: "Multi-Provider Setup Guide"
description: "Complete guide for setting up AICockpit for multiple AI providers"
tags: ["setup", "installation", "multi-provider", "devin", "goose", "claude-code", "github-copilot"]
author: "AICockpit Team"
version: "1.0"
---

# Multi-Provider Setup Guide

## Overview

This guide explains how to set up AICockpit for multiple AI providers simultaneously. AICockpit supports Devin, Goose, Claude Code, and GitHub Copilot, allowing you to use different AI agents with the same knowledge base and components.

## Quick Start

### Automated Setup (Recommended)

```bash
# Run the multi-provider setup script
bash scripts/setup-multi-provider.sh
```

This script will:
1. Create workspaces for all providers
2. Set up directory structure
3. Copy configuration files
4. Copy knowledge base documents
5. Copy example components
6. Verify installations

### Manual Setup

If you prefer to set up providers individually:

```bash
# Setup Devin
mkdir -p ~/.cockpit/{agents,skills,hooks,kb,logs,cache,backups}
cp config.yaml ~/.cockpit/
cp manifest.yaml ~/.cockpit/

# Setup Goose
mkdir -p ~/.goose/{agents,skills,hooks,kb,logs,cache,backups}
cp config.yaml ~/.goose/
cp manifest.yaml ~/.goose/

# Setup Claude Code
mkdir -p ~/.claude-code/{agents,skills,hooks,kb,logs,cache,backups}
cp config.yaml ~/.claude-code/
cp manifest.yaml ~/.claude-code/

# Setup GitHub Copilot
mkdir -p ~/.github-copilot/{agents,skills,hooks,kb,logs,cache,backups}
cp config.yaml ~/.github-copilot/
cp manifest.yaml ~/.github-copilot/
```

## Verification

### Automated Verification

```bash
# Verify all installations
bash scripts/validate-multi-provider.sh
```

This script will:
1. Check workspace existence
2. Verify directory structure
3. Validate configuration files
4. Check manifest integrity
5. Compare installations across providers
6. Generate detailed report

### Manual Verification

```bash
# Check Devin installation
ls -la ~/.cockpit/
cat ~/.cockpit/config.yaml

# Check Goose installation
ls -la ~/.goose/
cat ~/.goose/config.yaml

# Check Claude Code installation
ls -la ~/.claude-code/
cat ~/.claude-code/config.yaml

# Check GitHub Copilot installation
ls -la ~/.github-copilot/
cat ~/.github-copilot/config.yaml
```

## Directory Structure

After setup, you'll have:

```
Home Directory
├── .cockpit/                    # Devin
│   ├── agents/
│   │   ├── cockpit-builder/
│   │   └── ...
│   ├── skills/
│   │   ├── go-development/
│   │   └── ...
│   ├── hooks/
│   │   ├── cockpit-first/
│   │   └── ...
│   ├── kb/
│   │   ├── guides/
│   │   ├── examples/
│   │   └── troubleshooting/
│   ├── logs/
│   ├── cache/
│   ├── backups/
│   ├── config.yaml
│   └── manifest.yaml
│
├── .goose/                      # Goose
│   ├── agents/
│   ├── skills/
│   ├── hooks/
│   ├── kb/
│   ├── logs/
│   ├── cache/
│   ├── backups/
│   ├── config.yaml
│   └── manifest.yaml
│
├── .claude-code/                # Claude Code
│   ├── agents/
│   ├── skills/
│   ├── hooks/
│   ├── kb/
│   ├── logs/
│   ├── cache/
│   ├── backups/
│   ├── config.yaml
│   └── manifest.yaml
│
└── .github-copilot/             # GitHub Copilot
    ├── agents/
    ├── skills/
    ├── hooks/
    ├── kb/
    ├── logs/
    ├── cache/
    ├── backups/
    ├── config.yaml
    └── manifest.yaml
```

## Configuration Files

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

## Installation Manifest

Each provider has its own manifest file:

```yaml
# ~/.cockpit/manifest.yaml
version: "1.0"
cockpit_version: "0.2.0"
installed_at: "2026-06-20T14:00:00Z"
last_updated: "2026-06-20T14:00:00Z"

agents:
  - name: "cockpit-builder"
    version: "1.0.0"
    installed_path: "~/.cockpit/agents/cockpit-builder"
    files:
      - path: "manifest.yaml"
        checksum: "sha256:abc123..."
      - path: "README.md"
        checksum: "sha256:def456..."

skills:
  - name: "go-development"
    version: "1.0.0"
    installed_path: "~/.cockpit/skills/go-development"
    files:
      - path: "manifest.yaml"
        checksum: "sha256:ghi789..."

hooks:
  - name: "cockpit-first"
    version: "1.0.0"
    installed_path: "~/.cockpit/hooks/cockpit-first"
    files:
      - path: "manifest.yaml"
        checksum: "sha256:jkl012..."

modules: []

config:
  backup_dir: "~/.cockpit/backups"
  keep_backup: true
  log_operations: true
```

## Installing Components

### Install in All Providers

```bash
# Install agent in all providers
cockpit agent install cockpit-builder --all-providers

# Install skill in all providers
cockpit skill install go-development --all-providers

# Install hook in all providers
cockpit hook install cockpit-first --all-providers
```

### Install in Specific Provider

```bash
# Install only in Devin
cockpit agent install cockpit-builder --provider devin

# Install only in Goose
cockpit agent install cockpit-builder --provider goose

# Install in multiple specific providers
cockpit agent install cockpit-builder --providers devin,goose,claude-code
```

### Verify Component Installation

```bash
# List agents in all providers
cockpit agent list --all-providers

# List skills in all providers
cockpit skill list --all-providers

# List hooks in all providers
cockpit hook list --all-providers
```

## Using Different Providers

### Set Active Provider

```bash
# Set Devin as active
export COCKPIT_PROVIDER=devin

# Set Goose as active
export COCKPIT_PROVIDER=goose

# Set Claude Code as active
export COCKPIT_PROVIDER=claude-code

# Set GitHub Copilot as active
export COCKPIT_PROVIDER=github-copilot
```

### Run Commands with Specific Provider

```bash
# Search KB in Devin
COCKPIT_PROVIDER=devin cockpit kb search "logging"

# Search KB in Goose
COCKPIT_PROVIDER=goose cockpit kb search "logging"

# List documents in Claude Code
COCKPIT_PROVIDER=claude-code cockpit kb list

# List documents in GitHub Copilot
COCKPIT_PROVIDER=github-copilot cockpit kb list
```

## Synchronizing Across Providers

### Copy Configuration

```bash
# Copy Devin config to Goose
cp ~/.cockpit/config.yaml ~/.goose/config.yaml

# Update ai_provider in Goose config
sed -i 's/ai_provider: "devin"/ai_provider: "goose"/' ~/.goose/config.yaml
```

### Copy Components

```bash
# Copy agents from Devin to Goose
cp -r ~/.cockpit/agents/* ~/.goose/agents/

# Copy skills from Devin to Goose
cp -r ~/.cockpit/skills/* ~/.goose/skills/

# Copy hooks from Devin to Goose
cp -r ~/.cockpit/hooks/* ~/.goose/hooks/
```

### Share Knowledge Base

```bash
# Create shared KB directory
mkdir -p ~/.shared-kb

# Copy KB from Devin to shared location
cp -r ~/.cockpit/kb/* ~/.shared-kb/

# Update all configs to use shared KB
for provider in cockpit goose claude-code github-copilot; do
    sed -i 's|~/.'"$provider"'/kb|~/.shared-kb|' ~/."$provider"/config.yaml
done
```

## Troubleshooting

### Provider Not Found

```bash
# Check if provider workspace exists
ls -la ~/.cockpit
ls -la ~/.goose
ls -la ~/.claude-code
ls -la ~/.github-copilot

# Run setup script again
bash scripts/setup-multi-provider.sh
```

### Configuration Issues

```bash
# Verify configuration syntax
cat ~/.cockpit/config.yaml
cat ~/.goose/config.yaml

# Check for errors
grep -n "error\|Error" ~/.cockpit/logs/*.log
```

### Component Not Found

```bash
# List installed components
cockpit agent list --all-providers
cockpit skill list --all-providers
cockpit hook list --all-providers

# Reinstall components
cockpit agent install cockpit-builder --all-providers
```

### Manifest Corruption

```bash
# Validate manifest
cockpit manifest validate --all-providers

# Rebuild manifest
cockpit manifest rebuild --provider devin
cockpit manifest rebuild --provider goose
```

## Best Practices

### 1. Consistent Installation

Keep the same components installed across all providers:

```bash
# Install same components everywhere
cockpit agent install cockpit-builder --all-providers
cockpit skill install go-development --all-providers
cockpit hook install cockpit-first --all-providers
```

### 2. Regular Verification

Verify installations regularly:

```bash
# Weekly verification
bash scripts/validate-multi-provider.sh
```

### 3. Backup Before Changes

```bash
# Backup before making changes
cp -r ~/.cockpit ~/.cockpit.backup
cp -r ~/.goose ~/.goose.backup
```

### 4. Monitor Disk Space

```bash
# Check disk usage
du -sh ~/.cockpit ~/.goose ~/.claude-code ~/.github-copilot

# Total usage
du -sh ~/ | grep -E "cockpit|goose|claude|copilot"
```

## Performance Optimization

### Shared Knowledge Base

Save disk space by sharing the knowledge base:

```bash
# Create shared KB
mkdir -p ~/.shared-kb
cp -r ~/.cockpit/kb/* ~/.shared-kb/

# Update configs
for dir in ~/.cockpit ~/.goose ~/.claude-code ~/.github-copilot; do
    sed -i 's|'"$dir"'/kb|~/.shared-kb|' "$dir/config.yaml"
done
```

### Symlinked Components

Link components across providers:

```bash
# Link agents
ln -s ~/.cockpit/agents/cockpit-builder ~/.goose/agents/cockpit-builder

# Link skills
ln -s ~/.cockpit/skills/go-development ~/.goose/skills/go-development

# Link hooks
ln -s ~/.cockpit/hooks/cockpit-first ~/.goose/hooks/cockpit-first
```

## Storage Requirements

### Per Provider

- **Agents**: ~100MB
- **Skills**: ~50MB
- **Hooks**: ~25MB
- **Knowledge Base**: ~200MB
- **Logs**: ~50MB
- **Cache**: ~100MB
- **Total**: ~525MB per provider

### Total for All Providers

- **4 Providers**: ~2.1GB
- **With Shared KB**: ~1.5GB (saves ~600MB)

## Examples

### Example 1: Complete Multi-Provider Setup

```bash
# 1. Run setup script
bash scripts/setup-multi-provider.sh

# 2. Verify installation
bash scripts/validate-multi-provider.sh

# 3. Install components in all providers
cockpit agent install cockpit-builder --all-providers
cockpit skill install go-development --all-providers
cockpit hook install cockpit-first --all-providers

# 4. Verify components
cockpit agent list --all-providers
```

### Example 2: Provider-Specific Setup

```bash
# Setup only Devin
mkdir -p ~/.cockpit/{agents,skills,hooks,kb,logs,cache,backups}
cp config.yaml ~/.cockpit/
cp manifest.yaml ~/.cockpit/

# Setup only Goose
mkdir -p ~/.goose/{agents,skills,hooks,kb,logs,cache,backups}
cp config.yaml ~/.goose/
cp manifest.yaml ~/.goose/

# Verify each
COCKPIT_PROVIDER=devin cockpit kb list
COCKPIT_PROVIDER=goose cockpit kb list
```

### Example 3: Shared Knowledge Base

```bash
# Create shared KB
mkdir -p ~/.shared-kb
cp -r ~/.cockpit/kb/* ~/.shared-kb/

# Update all configs
for provider in cockpit goose claude-code github-copilot; do
    sed -i 's|~/.'"$provider"'/kb|~/.shared-kb|' ~/."$provider"/config.yaml
done

# Verify
cockpit kb list --all-providers
```

## Next Steps

1. Run the setup script: `bash scripts/setup-multi-provider.sh`
2. Verify installation: `bash scripts/validate-multi-provider.sh`
3. Install components: `cockpit agent install cockpit-builder --all-providers`
4. Start using AICockpit with your preferred provider
