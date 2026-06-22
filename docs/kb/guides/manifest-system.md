---
title: "AICockpit Manifest System"
description: "Understanding and managing the AICockpit installation manifest"
tags: ["manifest", "installation", "uninstallation", "tracking"]
author: "AICockpit Team"
version: "1.0"
---

# AICockpit Manifest System

## Overview

The AICockpit Manifest System is a tracking mechanism that records all installed components (agents, skills, hooks, modules) and their locations. This system is essential for proper uninstallation, updates, and system maintenance.

## Manifest File

### Location

```
~/.cockpit/manifest.yaml
```

### Purpose

- Track installed components
- Record installation dates and versions
- Store file locations for uninstallation
- Maintain installation history
- Enable safe updates and rollbacks

## Manifest Structure

### Root Level

```yaml
version: "1.0"                    # Manifest format version
cockpit_version: "0.2.0"          # AICockpit version when installed
installed_at: "2026-06-20T14:00:00Z"  # Initial installation time
last_updated: "2026-06-20T14:00:00Z"  # Last update time
```

### Agents Section

```yaml
agents:
  - name: "cockpit-builder"
    version: "1.0.0"
    description: "Agent for building AICockpit projects"
    source: "ai-assets/examples/agents/cockpit-builder"
    installed_path: "~/.cockpit/agents/cockpit-builder"
    installed_at: "2026-06-20T14:00:00Z"
    files:
      - path: "manifest.yaml"
        size: 512
        checksum: "abc123"
      - path: "README.md"
        size: 2048
        checksum: "def456"
      - path: "agent.go"
        size: 4096
        checksum: "ghi789"
    dependencies:
      - "go-development"
      - "file-management"
```

### Skills Section

```yaml
skills:
  - name: "go-development"
    version: "1.0.0"
    description: "Skill for Go development operations"
    source: "ai-assets/examples/skills/go-development"
    installed_path: "~/.cockpit/skills/go-development"
    installed_at: "2026-06-20T14:00:00Z"
    files:
      - path: "manifest.yaml"
        size: 512
        checksum: "abc123"
      - path: "README.md"
        size: 2048
        checksum: "def456"
      - path: "skill.go"
        size: 3072
        checksum: "jkl012"
```

### Hooks Section

```yaml
hooks:
  - name: "cockpit-first"
    version: "1.0.0"
    description: "Hook for AICockpit initialization"
    source: "ai-assets/examples/hooks/cockpit-first"
    installed_path: "~/.cockpit/hooks/cockpit-first"
    installed_at: "2026-06-20T14:00:00Z"
    files:
      - path: "manifest.yaml"
        size: 512
        checksum: "abc123"
      - path: "README.md"
        size: 1024
        checksum: "mno345"
      - path: "hook.go"
        size: 2048
        checksum: "pqr678"
```

### Modules Section

```yaml
modules:
  - name: "kb"
    version: "1.0.0"
    description: "Knowledge Base module"
    installed_at: "2026-06-20T14:00:00Z"
    files:
      - path: "cmd/kb.go"
        size: 5120
        checksum: "stu901"
```

### Configuration Section

```yaml
config:
  # Backup directory for uninstallation
  backup_dir: "~/.cockpit/backups"
  
  # Keep backup after uninstall
  keep_backup: true
  
  # Log installation/uninstallation
  log_operations: true
  
  # Verify checksums on load
  verify_checksums: true
  
  # Auto-backup before uninstall
  auto_backup: true
```

## Installation Tracking

### When Installing

1. **Create Entry**: Add component to manifest
2. **Record Files**: List all files with paths and checksums
3. **Record Dependencies**: List required dependencies
4. **Record Metadata**: Installation date, version, source
5. **Create Backup**: Optionally backup existing files

### Example Installation Entry

```yaml
agents:
  - name: "cockpit-builder"
    version: "1.0.0"
    description: "Agent for building AICockpit projects"
    source: "ai-assets/examples/agents/cockpit-builder"
    installed_path: "~/.cockpit/agents/cockpit-builder"
    installed_at: "2026-06-20T14:00:00Z"
    files:
      - path: "manifest.yaml"
        size: 512
        checksum: "sha256:abc123..."
      - path: "README.md"
        size: 2048
        checksum: "sha256:def456..."
      - path: "agent.go"
        size: 4096
        checksum: "sha256:ghi789..."
      - path: "agent_test.go"
        size: 3072
        checksum: "sha256:jkl012..."
      - path: "config/config.yaml"
        size: 256
        checksum: "sha256:mno345..."
    dependencies:
      - "go-development"
      - "file-management"
```

## Uninstallation Tracking

### When Uninstalling

1. **Verify Entry**: Check manifest for component
2. **Create Backup**: Back up all files before deletion
3. **Delete Files**: Remove all listed files
4. **Remove Entry**: Remove component from manifest
5. **Update Manifest**: Save updated manifest

### Backup Structure

```
~/.cockpit/backups/
├── cockpit-builder_1.0.0_2026-06-20T14:00:00Z/
│   ├── manifest.yaml
│   ├── README.md
│   ├── agent.go
│   ├── agent_test.go
│   └── config/
│       └── config.yaml
└── go-development_1.0.0_2026-06-20T14:00:00Z/
    ├── manifest.yaml
    ├── README.md
    ├── skill.go
    ├── skill_test.go
    └── config/
        └── config.yaml
```

## Manifest Operations

### List Installed Components

```bash
cockpit manifest list
```

### Add Component

```bash
cockpit manifest add agent cockpit-builder
cockpit manifest add skill go-development
cockpit manifest add hook cockpit-first
```

### Remove Component

```bash
cockpit manifest remove agent cockpit-builder
```

### Verify Installation

```bash
cockpit manifest verify
```

### Update Manifest

```bash
cockpit manifest update
```

### View Manifest

```bash
cat ~/.cockpit/manifest.yaml
```

## Checksum Verification

### Purpose

Checksums verify that files haven't been modified or corrupted.

### Checksum Format

```
sha256:abc123def456ghi789jkl012mno345pqr678stu901vwx234yz
```

### Verification Process

1. Read checksum from manifest
2. Calculate current file checksum
3. Compare checksums
4. Report if mismatch

### Verify All Files

```bash
cockpit manifest verify --checksums
```

## Dependency Management

### Tracking Dependencies

Each component can list its dependencies:

```yaml
agents:
  - name: "cockpit-builder"
    dependencies:
      - "go-development"      # Required skill
      - "file-management"     # Required skill
      - "cockpit-first"       # Required hook
```

### Dependency Resolution

When installing a component:

1. Check dependencies
2. Verify dependencies are installed
3. Report missing dependencies
4. Optionally auto-install dependencies

### Dependency Validation

```bash
cockpit manifest validate-dependencies
```

## Backup and Recovery

### Automatic Backup

Before uninstalling, automatically back up:

```bash
cockpit manifest backup agent cockpit-builder
```

### Manual Backup

```bash
cockpit manifest backup --all
```

### Restore from Backup

```bash
cockpit manifest restore agent cockpit-builder
```

### List Backups

```bash
cockpit manifest backups list
```

## Best Practices

1. **Always Update Manifest**: When installing/uninstalling, always update manifest
2. **Verify Checksums**: Regularly verify file checksums
3. **Keep Backups**: Keep backups for at least 30 days
4. **Review Manifest**: Periodically review manifest for consistency
5. **Document Changes**: Log all installation/uninstallation operations
6. **Test Uninstallation**: Test uninstallation before deploying

## Troubleshooting

### Manifest Corrupted

1. Check manifest syntax
2. Restore from backup
3. Manually rebuild manifest

### Missing Files

1. Check manifest for file list
2. Verify files exist
3. Restore from backup
4. Reinstall component

### Checksum Mismatch

1. Verify file hasn't been modified
2. Check for corruption
3. Restore from backup
4. Reinstall component

### Dependency Issues

1. Check manifest for dependencies
2. Verify dependencies are installed
3. Install missing dependencies
4. Update manifest

## Example Workflow

### Installation

```bash
# 1. Copy files
cp -r ai-assets/examples/agents/cockpit-builder ~/.cockpit/agents/

# 2. Update manifest
cockpit manifest add agent cockpit-builder

# 3. Verify installation
cockpit manifest verify

# 4. Test component
cockpit agent run cockpit-builder --help
```

### Uninstallation

```bash
# 1. Create backup
cockpit manifest backup agent cockpit-builder

# 2. Remove from manifest
cockpit manifest remove agent cockpit-builder

# 3. Delete files
rm -rf ~/.cockpit/agents/cockpit-builder

# 4. Verify removal
cockpit manifest verify
```

## Security Considerations

- Protect manifest file (read-only for users)
- Verify checksums regularly
- Keep backups secure
- Log all operations
- Monitor for unauthorized changes
