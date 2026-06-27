---
title: "AICockpit Package Registry Setup"
description: "Guide for setting up the AICockpit package registry repository"
tags: ["packages", "registry", "repository", "setup", "github"]
author: "AICockpit Team"
version: "1.0"
---

# AICockpit Package Registry Setup

## Overview

The AICockpit Package Registry is a separate Git repository that hosts all packages for AICockpit. This document explains how the registry is structured and how to set it up.

## Repository Structure

The official registry repository is located at: `https://github.com/lleite/cockpit-registry`

### Directory Layout

```
cockpit-registry/
├── package-index.yaml          # Registry index (lists all packages)
├── README.md                   # Registry documentation
├── LICENSE                     # Repository license (MIT)
├── .gitignore                  # Git ignore rules
├── .github/                    # GitHub workflows
│   └── workflows/
│       └── validate-packages.yml
├── packages/                   # Package directory root
│   ├── hello-world/            # First example package
│   │   ├── cockpit-package.yml # Package manifest
│   │   ├── README.md           # Package documentation
│   │   ├── LICENSE             # Package license
│   │   ├── modules/            # CLI modules
│   │   │   ├── cmd.go          # Hello command implementation
│   │   │   └── cmd_test.go     # Command tests
│   │   └── kb/                 # Knowledge base
│   │       └── guides/
│   │           └── usage.md    # Usage guide
│   └── [future-packages]/      # Additional packages
│       └── ...
└── docs/                       # Optional registry documentation
    └── ...
```

## Key Files

### package-index.yaml

The registry index file that lists all available packages. This file is read by AICockpit when searching for packages.

**Structure:**
```yaml
version: "1.0"
name: "AICockpit Official Packages"
description: "Official package registry for AICockpit"
url: "https://github.com/lleite/cockpit-registry"
maintainer: "AICockpit Team"
maintainer_email: "team@aicockpit.dev"
updated_at: "2026-06-20T14:00:00Z"

metadata:
  total_packages: 1
  categories:
    - examples

packages:
  - name: "hello-world"
    version: "1.0.0"
    description: "A simple hello-world package"
    # ... more fields
```

**Fields:**
- `version`: Registry index version (currently "1.0")
- `name`: Human-readable registry name
- `description`: Registry description
- `url`: Repository URL
- `maintainer`: Registry maintainer name
- `maintainer_email`: Maintainer email
- `updated_at`: Last update timestamp (ISO 8601)
- `metadata.total_packages`: Total number of packages
- `metadata.categories`: List of package categories
- `packages`: Array of package entries

### Package Entry Fields

Each package entry in the index contains:

```yaml
- name: "package-name"                    # Unique package name
  version: "1.0.0"                        # Semantic version
  description: "Package description"      # Short description
  author: "Author Name"                   # Package author
  license: "MIT"                          # License type
  category: "examples"                    # Package category
  tags:                                   # Search tags
    - tag1
    - tag2
  path: "packages/package-name"           # Directory path in registry (under packages/)
  url: "https://github.com/.../tree/main/packages/package-name"  # GitHub URL
  homepage: "https://..."                 # Package homepage
  repository: "https://..."               # Repository URL
  supported_providers:                    # Supported providers
    - devin
    - goose
  features:                               # Features provided
    - modules
    - kb
  requirements:                           # Version requirements
    cockpit: ">=0.2.0"
  installation_method: "symlink"          # Installation method
  status: "stable"                        # Package status
  released_at: "2026-06-20T10:00:00Z"    # Release date
```

### README.md

Comprehensive documentation for the registry including:
- Overview of the registry
- Available packages
- Installation instructions
- Package structure
- Contributing guidelines
- Support information

### LICENSE

MIT License for the registry repository. All packages should have their own LICENSE files as well.

### .gitignore

Standard Git ignore rules for Go projects and IDEs.

## Default Registry Configuration

When AICockpit is installed, it automatically configures the official registry:

```yaml
package_registries:
  - name: "official"
    url: "https://github.com/lleite/cockpit-registry"
    branch: "main"
    enabled: true
    priority: 1
```

This configuration is stored in `~/.cockpit/config.yaml`.

## First Package: hello-world

The registry includes a simple example package to demonstrate the structure:

### Package Manifest (cockpit-package.yml)

```yaml
name: "hello-world"
version: "1.0.0"
description: "A simple hello-world package that adds a hello command to AICockpit"
author: "AICockpit Team"
license: "MIT"

type: "utility"
category: "examples"

requirements:
  cockpit: ">=0.2.0"

features:
  modules:
    - path: "modules/cmd.go"
      name: "hello"
      description: "Simple hello-world command"
  
  kb:
    - path: "kb/guides/usage.md"
      type: "guide"

installation:
  supported_providers:
    - devin
    - goose
    - claude-code
    - github-copilot
  
  provider_features:
    devin:
      - modules
      - kb
    goose:
      - modules
      - kb
    claude-code:
      - modules
      - kb
    github-copilot:
      - modules
      - kb
  
  method: "symlink"

metadata:
  tags:
    - hello-world
    - example
    - simple
  
  status: "stable"
```

### CLI Module (modules/cmd.go)

```go
package modules

import (
	"fmt"
	"github.com/spf13/cobra"
)

func NewHelloCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "hello",
		Short: "Display hello world message",
		Long:  "A simple hello-world command that displays a greeting message",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("hello world")
			return nil
		},
	}
}
```

### Usage

```bash
# Install the package
cockpit pkg install hello-world

# Run the command
cockpit hello

# Output
hello world
```

## Adding a New Package

To add a new package to the registry:

### 1. Create Package Directory

```bash
mkdir -p packages/my-package/{modules,skills,agents,kb}
```

### 2. Create Package Manifest

Create `packages/my-package/cockpit-package.yml` with package metadata.

### 3. Implement Features

Add your agents, skills, modules, etc.

### 4. Add Documentation

Create `my-package/README.md` with usage instructions.

### 5. Add Tests

Add tests for all components (minimum 90% coverage).

### 6. Update Registry Index

Add your package entry to `package-index.yaml`:

```yaml
packages:
  - name: "my-package"
    version: "1.0.0"
    description: "My awesome package"
    path: "packages/my-package"
    url: "https://github.com/user/cockpit-registry/tree/main/packages/my-package"
    # ... other fields
```

### 7. Commit and Push

```bash
git add .
git commit -m "feat(packages): add my-package"
git push origin main
```

## Registry Caching

AICockpit caches registry indexes locally for performance:

```
~/.cockpit/cache/registries/
├── official/
│   ├── package-index.yaml
│   └── metadata.json
└── [other-registries]/
    └── ...
```

Cache is automatically updated:
- On first search
- When explicitly requested with `--force`
- Daily (configurable)

## Registry Commands

### Search Packages

```bash
# Search in all registries
cockpit pkg search hello

# Search in specific registry
cockpit pkg search hello --source official

# Search by category
cockpit pkg search --category examples

# Search by tag
cockpit pkg search --tag example
```

### List Packages

```bash
# List all packages
cockpit pkg list

# List with details
cockpit pkg list --detailed

# List from specific registry
cockpit pkg list --source official
```

### Install Package

```bash
# Install package
cockpit pkg install hello-world

# Install specific version
cockpit pkg install hello-world@1.0.0

# Install from specific registry
cockpit pkg install hello-world --source official
```

### Manage Registries

```bash
# List registries
cockpit pkg registries list

# Add registry
cockpit pkg registries add my-registry https://github.com/user/packages

# Remove registry
cockpit pkg registries remove my-registry

# Enable/disable registry
cockpit pkg registries enable official
cockpit pkg registries disable official

# Update registry cache
cockpit pkg registries update official
```

## Best Practices

### 1. Keep Index Updated

Always update `package-index.yaml` when adding or modifying packages.

### 2. Use Semantic Versioning

Follow semver for package versions (MAJOR.MINOR.PATCH).

### 3. Document Everything

Include comprehensive README.md and usage guides in each package.

### 4. Test Thoroughly

Ensure minimum 90% test coverage for all components.

### 5. Maintain Backwards Compatibility

Keep older versions available when possible.

### 6. Use Consistent Naming

Package names should be lowercase with hyphens (e.g., `hello-world`).

### 7. Provide Clear Examples

Include usage examples in documentation.

## Contributing to the Registry

To contribute a package:

1. **Fork** the repository
2. **Create** a feature branch
3. **Add** your package
4. **Update** `package-index.yaml`
5. **Test** thoroughly
6. **Document** comprehensively
7. **Submit** a pull request

## Support

- **Issues:** https://github.com/lleite/cockpit-registry/issues
- **Discussions:** https://github.com/lleite/cockpit-registry/discussions
- **Documentation:** https://docs.aicockpit.dev/packages

## See Also

- [Package Specification](./package-specification.md)
- [Package Manager](./package-manager.md)
- [Package Registry](./package-registry.md)
