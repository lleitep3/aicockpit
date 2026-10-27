---
title: "AICockpit Package Manager"
description: "Guide for managing AICockpit packages"
tags: ["packages", "package-manager", "installation", "uninstallation", "dependencies"]
author: "AICockpit Team"
version: "1.0"
---

# AICockpit Package Manager

## Overview

The AICockpit Package Manager handles installation, uninstallation, and management of packages. Packages can contain agents, skills, CLI modules, knowledge base documents, and workflows.

## Installation

### Install a Package

```bash
# Install from local directory
cockpit package install ./my-package

# Install from URL
cockpit package install https://github.com/user/my-package

# Install with specific version
cockpit package install my-package@1.0.0

# Install multiple packages
cockpit package install package1 package2 package3

# Install with dependencies
cockpit package install my-package --with-dependencies
```

### Installation Process

1. **Validate** package manifest (cockpit-package.yml)
2. **Check** AICockpit version compatibility
3. **Resolve** dependencies
4. **Prompt** user for configuration
5. **Create** package directory
6. **Copy/Symlink** files to providers
7. **Update** manifest
8. **Run** post-install hooks
9. **Verify** installation

### Example Installation

```bash
$ cockpit package install html-document-build

Validating package...
✓ Package manifest is valid
✓ AICockpit version compatible (0.2.0 >= 0.2.0)

Checking dependencies...
✓ kb-manager@1.0.0 is installed

Configuring package...
? Output directory for HTML files: (articles) 
? Color theme: (light) dark
? Enable table of contents: (yes) 

Installing package...
✓ Package directory created
✓ Agents installed (1)
✓ Skills installed (2)
✓ CLI modules installed (1)
✓ KB documents installed (3)

Running post-install hooks...
✓ Setup default configuration

✓ Package installed successfully!

Package: html-document-build@1.0.0
Location: ~/.cockpit/packages/html-document-build
Providers: devin, goose
Features: agents, skills, modules, kb
```

## Uninstallation

### Uninstall a Package

```bash
# Uninstall package
cockpit package uninstall html-document-build

# Uninstall with confirmation
cockpit package uninstall html-document-build --force

# Uninstall multiple packages
cockpit package uninstall package1 package2
```

### Uninstallation Process

1. **Check** dependent packages
2. **Warn** about dependencies
3. **Create** backup
4. **Remove** symlinks/copies
5. **Update** manifest
6. **Run** pre-uninstall hooks
7. **Remove** package directory
8. **Verify** uninstallation

### Example Uninstallation

```bash
$ cockpit package uninstall html-document-build

Checking dependencies...
⚠ The following packages depend on this:
  - article-publisher@1.0.0

? Continue uninstallation? (yes/no) yes

Creating backup...
✓ Backup created at ~/.cockpit/backups/html-document-build_1.0.0_2026-06-20

Uninstalling package...
✓ Agents removed (1)
✓ Skills removed (2)
✓ CLI modules removed (1)
✓ KB documents removed (3)

Running pre-uninstall hooks...
✓ Cleanup completed

✓ Package uninstalled successfully!

Backup location: ~/.cockpit/backups/html-document-build_1.0.0_2026-06-20
```

## Package Management

### List Installed Packages

```bash
# List all packages
cockpit package list

# List packages with details
cockpit package list --detailed

# List packages by provider
cockpit package list --provider devin

# Search packages
cockpit package search documentation
```

### Example Output

```bash
$ cockpit package list --detailed

Installed Packages:
┌─────────────────────────┬─────────┬──────────────────────────────────┐
│ Name                    │ Version │ Description                      │
├─────────────────────────┼─────────┼──────────────────────────────────┤
│ html-document-build     │ 1.0.0   │ Builds HTML documentation        │
│ kb-manager              │ 1.0.0   │ Knowledge base management        │
│ text-processor          │ 2.1.0   │ Text processing utilities        │
└─────────────────────────┴─────────┴──────────────────────────────────┘

Total: 3 packages installed
```

### View Package Information

```bash
# Show package details
cockpit package info html-document-build

# Show package dependencies
cockpit package info html-document-build --dependencies

# Show package configuration
cockpit package info html-document-build --config
```

### Example Output

```bash
$ cockpit package info html-document-build

Package: html-document-build
Version: 1.0.0
Author: AICockpit Team
License: MIT
Description: Builds HTML documentation from knowledge base articles

Location: ~/.cockpit/packages/html-document-build
Installed: 2026-06-20 14:00:00

Features:
  ✓ Agents (1)
  ✓ Skills (2)
  ✓ CLI Modules (1)
  ✓ Knowledge Base (3)

Dependencies:
  - kb-manager@1.0.0

Providers:
  ✓ devin
  ✓ goose

Configuration:
  output_dir: articles
  theme: light
  enable_toc: true
```

## Dependency Management

### Check Dependencies

```bash
# Check package dependencies
cockpit package dependencies html-document-build

# Check if all dependencies are installed
cockpit package validate html-document-build

# Show dependency tree
cockpit package tree html-document-build
```

### Example Output

```bash
$ cockpit package tree html-document-build

html-document-build@1.0.0
├── kb-manager@1.0.0
│   └── (no dependencies)
└── (no other dependencies)

All dependencies satisfied ✓
```

### Install Dependencies

```bash
# Install missing dependencies
cockpit package install html-document-build --with-dependencies

# Install specific dependency
cockpit package install kb-manager@1.0.0
```

## Configuration Management

### Configure Package

```bash
# Configure installed package
cockpit package configure html-document-build

# Set specific configuration
cockpit package configure html-document-build --set output_dir=/custom/path

# Reset to defaults
cockpit package configure html-document-build --reset
```

### Example Configuration

```bash
$ cockpit package configure html-document-build

Current configuration:
  output_dir: articles
  theme: light
  enable_toc: true

? Output directory: (articles) /var/www/docs
? Theme: (light) dark
? Enable table of contents: (yes) no

✓ Configuration updated

New configuration:
  output_dir: /var/www/docs
  theme: dark
  enable_toc: false
```

## Update and Upgrade

### Update Package

```bash
# Check for updates
cockpit package check-updates

# Update specific package
cockpit package update html-document-build

# Update all packages
cockpit package update --all

# Update to specific version
cockpit package update html-document-build@2.0.0
```

### Example Update

```bash
$ cockpit package update html-document-build

Checking for updates...
✓ New version available: 2.0.0

Validating new version...
✓ Compatible with AICockpit 0.2.0

Creating backup...
✓ Backup created

Updating package...
✓ Files updated
✓ Configuration migrated

Running post-install hooks...
✓ Update completed

✓ Package updated successfully!
Old version: 1.0.0
New version: 2.0.0
```

## Backup and Recovery

### Backup Package

```bash
# Backup installed package
cockpit package backup html-document-build

# Backup all packages
cockpit package backup --all

# List backups
cockpit package backups list

# Restore from backup
cockpit package restore html-document-build@1.0.0
```

### Example Backup

```bash
$ cockpit package backup html-document-build

Creating backup...
✓ Backup created

Backup location: ~/.cockpit/backups/html-document-build_1.0.0_2026-06-20T14:00:00Z

$ cockpit package backups list

Available backups:
  html-document-build_1.0.0_2026-06-20T14:00:00Z
  html-document-build_1.0.0_2026-06-19T10:30:00Z
  text-processor_2.1.0_2026-06-18T15:45:00Z

$ cockpit package restore html-document-build@1.0.0_2026-06-19T10:30:00Z

Restoring backup...
✓ Package restored

Restored version: 1.0.0
Restored at: 2026-06-19 10:30:00
```

## Validation and Verification

### Validate Package

```bash
# Validate package manifest
cockpit package validate ./my-package

# Validate installed package
cockpit package validate html-document-build

# Verify package integrity
cockpit package verify html-document-build
```

### Example Validation

```bash
$ cockpit package validate html-document-build

Validating package...
✓ Manifest syntax is valid
✓ Required fields present
✓ Version format is valid (semver)
✓ Features exist
✓ Providers are supported
✓ Dependencies are resolvable

✓ Package is valid!
```

## Troubleshooting

### Installation Issues

**Problem**: Package validation fails

```bash
# Check manifest syntax
cockpit package validate ./my-package

# Check specific errors
cockpit package validate ./my-package --verbose
```

**Problem**: Dependency not found

```bash
# Check dependencies
cockpit package dependencies my-package

# Install missing dependency
cockpit package install missing-dependency
```

**Problem**: Version incompatibility

```bash
# Check AICockpit version
cockpit info

# Check package requirements
cockpit package info my-package
```

### Uninstallation Issues

**Problem**: Cannot uninstall due to dependencies

```bash
# Check dependent packages
cockpit package info my-package --dependents

# Uninstall dependent packages first
cockpit package uninstall dependent-package
```

**Problem**: Package directory locked

```bash
# Force uninstall
cockpit package uninstall my-package --force

# Check for running processes
lsof ~/.cockpit/packages/my-package
```

## Best Practices

### 1. Always Check Dependencies

```bash
# Before installing
cockpit package dependencies my-package

# Before uninstalling
cockpit package info my-package --dependents
```

### 2. Backup Before Updates

```bash
# Create backup
cockpit package backup my-package

# Then update
cockpit package update my-package
```

### 3. Validate Configuration

```bash
# After installation
cockpit package configure my-package

# Verify settings
cockpit package info my-package --config
```

### 4. Keep Backups

```bash
# List available backups
cockpit package backups list

# Keep important backups
cockpit package backups keep my-package@1.0.0_2026-06-20
```

### 5. Monitor Package Health

```bash
# Verify package integrity
cockpit package verify my-package

# Check for updates
cockpit package check-updates
```

## Command Reference

| Command | Description |
|---------|-------------|
| `cockpit package install <pkg>` | Install a package |
| `cockpit package uninstall <pkg>` | Uninstall a package |
| `cockpit package list` | List installed packages |
| `cockpit package info <pkg>` | Show package information |
| `cockpit package configure <pkg>` | Configure a package |
| `cockpit package update <pkg>` | Update a package |
| `cockpit package validate <pkg>` | Validate a package |
| `cockpit package verify <pkg>` | Verify package integrity |
| `cockpit package dependencies <pkg>` | Show dependencies |
| `cockpit package tree <pkg>` | Show dependency tree |
| `cockpit package backup <pkg>` | Backup a package |
| `cockpit package restore <pkg>` | Restore from backup |
| `cockpit package search <query>` | Search packages |

## See Also

- [Package Specification](./package-specification.md)
- [Creating Packages](./creating-packages.md)
- [Package Examples](./package-examples.md)
