---
title: "Installation Paths for AI Agents"
description: "Directory structure and installation paths for Devin and Antigravity"
tags: ["installation", "devin", "antigravity", "agents", "skills", "hooks"]
author: "AICockpit Team"
version: "1.0"
---

# Installation Paths for AI Agents

## Overview

This guide explains where to install agents, skills, and hooks for different AI agent platforms (Devin and Antigravity).

## Directory Structure

### AICockpit Workspace

```
~/.cockpit/
├── agents/              # Installed agents
│   ├── agent-name/
│   │   ├── manifest.yaml
│   │   ├── agent.go
│   │   └── config.yaml
├── skills/              # Installed skills
│   ├── skill-name/
│   │   ├── manifest.yaml
│   │   ├── skill.go
│   │   └── config.yaml
├── hooks/               # Installed hooks
│   ├── hook-name/
│   │   ├── manifest.yaml
│   │   ├── hook.go
│   │   └── config.yaml
├── kb/                  # Knowledge base
│   ├── guides/
│   ├── examples/
│   └── troubleshooting/
├── logs/                # Log files
├── cache/               # Cache files
├── config.yaml          # Main configuration
└── manifest.yaml        # Installation manifest
```

## Devin Installation Paths

### Agent Installation

**Source Location:**
```
ai-assets/examples/agents/agent-name/
```

**Installation Location:**
```
~/.cockpit/agents/agent-name/
```

**Files to Copy:**
```
ai-assets/examples/agents/agent-name/
├── manifest.yaml        → ~/.cockpit/agents/agent-name/manifest.yaml
├── README.md           → ~/.cockpit/agents/agent-name/README.md
├── agent.go            → ~/.cockpit/agents/agent-name/agent.go
├── agent_test.go       → ~/.cockpit/agents/agent-name/agent_test.go
└── config/
    └── config.yaml     → ~/.cockpit/agents/agent-name/config.yaml
```

### Skill Installation

**Source Location:**
```
ai-assets/examples/skills/skill-name/
```

**Installation Location:**
```
~/.cockpit/skills/skill-name/
```

**Files to Copy:**
```
ai-assets/examples/skills/skill-name/
├── manifest.yaml        → ~/.cockpit/skills/skill-name/manifest.yaml
├── README.md           → ~/.cockpit/skills/skill-name/README.md
├── skill.go            → ~/.cockpit/skills/skill-name/skill.go
├── skill_test.go       → ~/.cockpit/skills/skill-name/skill_test.go
└── config/
    └── config.yaml     → ~/.cockpit/skills/skill-name/config.yaml
```

### Hook Installation

**Source Location:**
```
ai-assets/examples/hooks/hook-name/
```

**Installation Location:**
```
~/.cockpit/hooks/hook-name/
```

**Files to Copy:**
```
ai-assets/examples/hooks/hook-name/
├── manifest.yaml        → ~/.cockpit/hooks/hook-name/manifest.yaml
├── README.md           → ~/.cockpit/hooks/hook-name/README.md
├── hook.go             → ~/.cockpit/hooks/hook-name/hook.go
├── hook_test.go        → ~/.cockpit/hooks/hook-name/hook_test.go
└── config/
    └── config.yaml     → ~/.cockpit/hooks/hook-name/config.yaml
```

## Antigravity Installation Paths

### Agent Installation

**Source Location:**
```
ai-assets/examples/agents/agent-name/
```

**Installation Location (Antigravity):**
```
~/.antigravity/agents/agent-name/
```

**Files to Copy:**
```
ai-assets/examples/agents/agent-name/
├── manifest.yaml        → ~/.antigravity/agents/agent-name/manifest.yaml
├── README.md           → ~/.antigravity/agents/agent-name/README.md
├── agent.go            → ~/.antigravity/agents/agent-name/agent.go
├── agent_test.go       → ~/.antigravity/agents/agent-name/agent_test.go
└── config/
    └── config.yaml     → ~/.antigravity/agents/agent-name/config.yaml
```

### Skill Installation

**Source Location:**
```
ai-assets/examples/skills/skill-name/
```

**Installation Location (Antigravity):**
```
~/.antigravity/skills/skill-name/
```

**Files to Copy:**
```
ai-assets/examples/skills/skill-name/
├── manifest.yaml        → ~/.antigravity/skills/skill-name/manifest.yaml
├── README.md           → ~/.antigravity/skills/skill-name/README.md
├── skill.go            → ~/.antigravity/skills/skill-name/skill.go
├── skill_test.go       → ~/.antigravity/skills/skill-name/skill_test.go
└── config/
    └── config.yaml     → ~/.antigravity/skills/skill-name/config.yaml
```

### Hook Installation

**Source Location:**
```
ai-assets/examples/hooks/hook-name/
```

**Installation Location (Antigravity):**
```
~/.antigravity/hooks/hook-name/
```

**Files to Copy:**
```
ai-assets/examples/hooks/hook-name/
├── manifest.yaml        → ~/.antigravity/hooks/hook-name/manifest.yaml
├── README.md           → ~/.antigravity/hooks/hook-name/README.md
├── hook.go             → ~/.antigravity/hooks/hook-name/hook.go
├── hook_test.go        → ~/.antigravity/hooks/hook-name/hook_test.go
└── config/
    └── config.yaml     → ~/.antigravity/hooks/hook-name/config.yaml
```

## Installation Manifest

### Purpose

The installation manifest (`~/.cockpit/manifest.yaml`) tracks all installed components and their locations. This is essential for proper uninstallation and updates.

### Format

```yaml
version: "1.0"
cockpit_version: "0.2.0"
installed_at: "2026-06-20T14:00:00Z"

agents:
  - name: "cockpit-builder"
    version: "1.0.0"
    source: "ai-assets/examples/agents/cockpit-builder"
    installed_path: "~/.cockpit/agents/cockpit-builder"
    files:
      - "manifest.yaml"
      - "README.md"
      - "agent.go"
      - "agent_test.go"
      - "config/config.yaml"

skills:
  - name: "go-development"
    version: "1.0.0"
    source: "ai-assets/examples/skills/go-development"
    installed_path: "~/.cockpit/skills/go-development"
    files:
      - "manifest.yaml"
      - "README.md"
      - "skill.go"
      - "skill_test.go"
      - "config/config.yaml"

hooks:
  - name: "cockpit-first"
    version: "1.0.0"
    source: "ai-assets/examples/hooks/cockpit-first"
    installed_path: "~/.cockpit/hooks/cockpit-first"
    files:
      - "manifest.yaml"
      - "README.md"
      - "hook.go"
      - "hook_test.go"
      - "config/config.yaml"
```

## Installation Process

### For Devin

1. **Copy Files**
   ```bash
   cp -r ai-assets/examples/agents/agent-name ~/.cockpit/agents/
   cp -r ai-assets/examples/skills/skill-name ~/.cockpit/skills/
   cp -r ai-assets/examples/hooks/hook-name ~/.cockpit/hooks/
   ```

2. **Update Manifest**
   ```bash
   cockpit manifest add agent cockpit-builder
   cockpit manifest add skill go-development
   cockpit manifest add hook cockpit-first
   ```

3. **Verify Installation**
   ```bash
   cockpit agent list
   cockpit skill list
   cockpit hook list
   ```

### For Antigravity

1. **Copy Files**
   ```bash
   cp -r ai-assets/examples/agents/agent-name ~/.antigravity/agents/
   cp -r ai-assets/examples/skills/skill-name ~/.antigravity/skills/
   cp -r ai-assets/examples/hooks/hook-name ~/.antigravity/hooks/
   ```

2. **Update Manifest**
   ```bash
   antigravity manifest add agent cockpit-builder
   antigravity manifest add skill go-development
   antigravity manifest add hook cockpit-first
   ```

3. **Verify Installation**
   ```bash
   antigravity agent list
   antigravity skill list
   antigravity hook list
   ```

## Uninstallation Process

### For Devin

```bash
# Remove from manifest
cockpit manifest remove agent cockpit-builder

# Remove files
rm -rf ~/.cockpit/agents/cockpit-builder
```

### For Antigravity

```bash
# Remove from manifest
antigravity manifest remove agent cockpit-builder

# Remove files
rm -rf ~/.antigravity/agents/cockpit-builder
```

## Best Practices

1. **Always Update Manifest**: When installing components, always update the manifest
2. **Verify Installation**: After installation, verify all files are in place
3. **Test Components**: Run tests to ensure components work correctly
4. **Document Changes**: Keep track of what was installed and when
5. **Backup Before Uninstall**: Back up configuration before uninstalling

## Troubleshooting

### Files Not Found After Installation

1. Check installation path
2. Verify files were copied correctly
3. Check file permissions
4. Verify manifest is updated

### Component Not Working

1. Check manifest for correct paths
2. Verify all required files are present
3. Check configuration files
4. Review logs for errors

### Uninstallation Issues

1. Check manifest for file list
2. Verify files exist before deletion
3. Check for file locks
4. Review logs for errors
