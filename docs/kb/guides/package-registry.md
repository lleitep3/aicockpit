---
title: "AICockpit Package Registry"
description: "Guide for managing package registries in AICockpit"
tags: ["packages", "registry", "repositories", "package-index.yaml", "installation"]
author: "AICockpit Team"
version: "1.0"
---

# AICockpit Package Registry

## Overview

A Package Registry is a Git repository that contains AICockpit packages and a package index. Registries allow users to discover, search, and install packages from multiple sources.

## Registry Structure

### Git Repository Layout

```
cockpit-packages/
├── package-index.yaml          # Package registry index
├── README.md                   # Registry documentation
├── packages/                   # Package directory root
│   ├── html-document-build/    # Package directory
│   │   ├── cockpit-package.yml # Package manifest
│   │   ├── README.md           # Package documentation
│   │   ├── agents/
│   │   ├── .cockpit/skills/
│   │   ├── modules/
│   │   ├── kb/
│   │   └── workflows/
│   ├── text-processor/         # Another package
│   │   ├── cockpit-package.yml
│   │   ├── README.md
│   │   └── ...
│   └── kb-manager/             # Another package
│       ├── cockpit-package.yml
│       ├── README.md
│       └── ...
└── docs/                       # Optional registry documentation
    └── ...
```

## package-index.yaml

The `package-index.yaml` file is the registry index that lists all available packages.

### Complete Example

```yaml
# Package Registry Index
version: "1.0"
name: "AICockpit Official Packages"
description: "Official package registry for AICockpit"
url: "https://github.com/lleite/cockpit-packages"
maintainer: "AICockpit Team"
maintainer_email: "team@aicockpit.dev"

# Last update timestamp
updated_at: "2026-06-20T14:00:00Z"

# Registry metadata
metadata:
  total_packages: 3
  categories:
    - documentation
    - utilities
    - text-processing

# Packages in this registry
packages:
  - name: "html-document-build"
    version: "1.0.0"
    description: "Builds HTML documentation from knowledge base articles"
    author: "AICockpit Team"
    license: "MIT"
    category: "documentation"
    tags:
      - documentation
      - html
      - generation
    path: "packages/html-document-build"
    url: "https://github.com/lleite/cockpit-packages/tree/main/packages/html-document-build"
    homepage: "https://docs.aicockpit.dev/packages/html-document-build"
    repository: "https://github.com/lleite/cockpit-packages"
    
    # Supported providers
    supported_providers:
      - devin
      - goose
    
    # Features provided
    features:
      - agents
      - skills
      - modules
      - kb
    
    # Requirements
    requirements:
      cockpit: ">=0.2.0"
      go: ">=1.26"
    
    # Dependencies
    dependencies:
      - name: "kb-manager"
        version: ">=1.0.0"
    
    # Installation method
    installation_method: "symlink"
    
    # Checksum for integrity verification
    checksum: "sha256:abc123def456..."
    
    # Download size
    size_bytes: 1024000
    
    # Package status
    status: "stable"  # alpha, beta, stable, deprecated
    
    # Release date
    released_at: "2026-06-20T10:00:00Z"

  - name: "text-processor"
    version: "2.1.0"
    description: "Text processing utilities"
    author: "AICockpit Team"
    license: "MIT"
    category: "utilities"
    tags:
      - text
      - processing
      - utilities
    path: "packages/text-processor"
    url: "https://github.com/lleite/cockpit-packages/tree/main/packages/text-processor"
    supported_providers:
      - devin
      - goose
      - claude-code
      - github-copilot
    features:
      - skills
      - modules
    requirements:
      cockpit: ">=0.2.0"
    status: "stable"
    released_at: "2026-06-15T10:00:00Z"

  - name: "kb-manager"
    version: "1.0.0"
    description: "Knowledge base management utilities"
    author: "AICockpit Team"
    license: "MIT"
    category: "utilities"
    tags:
      - kb
      - knowledge-base
      - management
    path: "packages/kb-manager"
    url: "https://github.com/lleite/cockpit-packages/tree/main/packages/kb-manager"
    supported_providers:
      - devin
      - goose
    features:
      - skills
      - modules
    requirements:
      cockpit: ">=0.2.0"
    status: "stable"
    released_at: "2026-06-10T10:00:00Z"
```

## Registry Configuration

### Adding a Registry

Registries are configured in `~/.cockpit/config.yaml`:

```yaml
package_registries:
  - name: "official"
    url: "https://github.com/lleite/cockpit-packages"
    branch: "main"
    enabled: true
    priority: 1  # Higher priority = searched first
    
  - name: "community"
    url: "https://github.com/community/cockpit-packages"
    branch: "main"
    enabled: true
    priority: 2
    
  - name: "private"
    url: "git@github.com:myorg/private-packages.git"
    branch: "main"
    enabled: false
    priority: 3
```

### Registry Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Registry name (must be unique) |
| url | string | Yes | Git repository URL (HTTPS or SSH) |
| branch | string | Yes | Git branch to use (default: main) |
| enabled | boolean | No | Whether registry is enabled (default: true) |
| priority | integer | No | Search priority (higher = first) |

## Package Search

### Search Command

```bash
# Search in all enabled registries
cockpit pkg search html

# Search in specific registry
cockpit pkg search html --source official

# Search with detailed output
cockpit pkg search html --detailed

# Search by category
cockpit pkg search --category documentation

# Search by tag
cockpit pkg search --tag html
```

### Search Results

```bash
$ cockpit pkg search html

Found 2 packages:

1. html-document-build (1.0.0)
   Author: AICockpit Team
   Description: Builds HTML documentation from knowledge base articles
   Category: documentation
   Status: stable
   Providers: devin, goose
   Registry: official

2. html-generator (1.5.0)
   Author: Community
   Description: Simple HTML generator
   Category: utilities
   Status: beta
   Providers: devin, goose, claude-code
   Registry: community
```

## Package Installation

### Install Command

```bash
# Install from default registry
cockpit pkg install html-document-build

# Install specific version
cockpit pkg install html-document-build@1.0.0

# Install from specific registry
cockpit pkg install html-document-build --source official

# Install with dependencies
cockpit pkg install html-document-build --with-dependencies

# Install multiple packages
cockpit pkg install package1 package2 package3

# Install and configure
cockpit pkg install html-document-build --interactive
```

### Installation Process

1. **Search** package in registries (by priority)
2. **Validate** package manifest
3. **Check** AICockpit version compatibility
4. **Resolve** dependencies
5. **Clone/Download** package from registry
6. **Prompt** user for configuration
7. **Install** package using PackageManager
8. **Update** manifest with installation tracking
9. **Verify** installation

### Example Installation

```bash
$ cockpit pkg install html-document-build

Searching in registries...
✓ Found in 'official' registry

Validating package...
✓ Package manifest is valid
✓ AICockpit version compatible (0.2.0 >= 0.2.0)

Checking dependencies...
✓ kb-manager@1.0.0 is installed

Configuring package...
? Output directory for HTML files: (articles) 
? Color theme: (light) dark
? Enable table of contents: (yes) 

Downloading package...
✓ Package downloaded from official registry

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
Registry: official
Providers: devin, goose
Features: agents, skills, modules, kb
```

## Package Uninstallation

### Uninstall Command

```bash
# Uninstall package
cockpit pkg uninstall html-document-build

# Uninstall with confirmation
cockpit pkg uninstall html-document-build --force

# Uninstall multiple packages
cockpit pkg uninstall package1 package2
```

### Uninstallation Process

1. **Check** if package exists
2. **Check** dependent packages
3. **Warn** about dependencies
4. **Create** backup
5. **Remove** package files
6. **Update** manifest
7. **Verify** uninstallation

## Registry Management

### List Registries

```bash
# List all registries
cockpit pkg registries list

# List enabled registries
cockpit pkg registries list --enabled

# Show registry details
cockpit pkg registries info official
```

### Add Registry

```bash
# Add new registry
cockpit pkg registries add my-registry https://github.com/user/packages

# Add with specific branch
cockpit pkg registries add my-registry https://github.com/user/packages --branch develop

# Add with priority
cockpit pkg registries add my-registry https://github.com/user/packages --priority 2
```

### Remove Registry

```bash
# Remove registry
cockpit pkg registries remove my-registry

# Remove with confirmation
cockpit pkg registries remove my-registry --force
```

### Enable/Disable Registry

```bash
# Enable registry
cockpit pkg registries enable my-registry

# Disable registry
cockpit pkg registries disable my-registry
```

### Update Registry Cache

```bash
# Update all registries
cockpit pkg registries update

# Update specific registry
cockpit pkg registries update official

# Force refresh
cockpit pkg registries update --force
```

## Default Registry

By default, AICockpit points to the official registry:

```yaml
package_registries:
  - name: "official"
    url: "https://github.com/lleite/cockpit-packages"
    branch: "main"
    enabled: true
    priority: 1
```

This is automatically configured during `cockpit setup`.

## Creating a Package Registry

### Step 1: Create Git Repository

```bash
# Create repository
git clone https://github.com/user/cockpit-packages.git
cd cockpit-packages

# Initialize structure (packages under packages/)
mkdir -p packages
touch package-index.yaml
touch README.md
```

### Step 2: Create package-index.yaml

```yaml
version: "1.0"
name: "My Package Registry"
description: "My custom package registry"
url: "https://github.com/user/cockpit-packages"
maintainer: "Your Name"
maintainer_email: "your@email.com"

metadata:
  total_packages: 0
  categories: []

packages: []
```

### Step 3: Add Packages

```bash
# Copy package to registry under packages/
cp -r /path/to/my-package ./packages/my-package

# Update package-index.yaml with package info
# (Add entry to packages list with path: "packages/my-package")

# Commit and push
git add .
git commit -m "feat(packages): add my-package"
git push origin main
```

### Step 4: Register with AICockpit

```bash
# Add registry to AICockpit
cockpit pkg registries add my-registry https://github.com/user/cockpit-packages

# Verify
cockpit pkg registries list
```

## Registry Caching

AICockpit caches registry indexes locally:

```
~/.cockpit/cache/
├── registries/
│   ├── official/
│   │   ├── package-index.yaml
│   │   └── metadata.json
│   ├── community/
│   │   ├── package-index.yaml
│   │   └── metadata.json
│   └── private/
│       ├── package-index.yaml
│       └── metadata.json
```

Cache is automatically updated:
- On first search
- When explicitly requested with `--force`
- Daily (configurable)

## Package Integrity

### Checksum Verification

Each package in the registry includes a SHA256 checksum:

```yaml
packages:
  - name: "html-document-build"
    checksum: "sha256:abc123def456..."
```

AICockpit verifies checksums after downloading packages.

### GPG Signing (Future)

Future versions will support GPG signing for package authenticity.

## Best Practices

### 1. Keep Index Updated

Always update `package-index.yaml` when adding/removing packages:

```bash
# After adding package
git add package-index.yaml
git commit -m "Update package index"
git push origin main
```

### 2. Use Semantic Versioning

Follow semver for package versions:

```
MAJOR.MINOR.PATCH

Examples:
  1.0.0    - Initial release
  1.1.0    - New features
  1.1.1    - Bug fixes
  2.0.0    - Breaking changes
```

### 3. Document Packages

Include comprehensive README.md in each package:

```markdown
# Package Name

## Description

## Installation

## Usage

## Configuration

## Dependencies

## Support
```

### 4. Test Before Publishing

Always test packages before adding to registry:

```bash
# Test installation
cockpit pkg install ./my-package

# Test functionality
# ... run tests ...

# Uninstall
cockpit pkg uninstall my-package
```

### 5. Maintain Backwards Compatibility

Keep older versions available when possible:

```
my-package/
├── v1.0.0/
│   └── cockpit-package.yml
├── v1.1.0/
│   └── cockpit-package.yml
└── v2.0.0/
    └── cockpit-package.yml
```

## Troubleshooting

### Registry Not Found

```bash
# Check registered registries
cockpit pkg registries list

# Add registry
cockpit pkg registries add official https://github.com/lleite/cockpit-packages
```

### Package Not Found

```bash
# Search in all registries
cockpit pkg search package-name

# Search in specific registry
cockpit pkg search package-name --source official

# Update registry cache
cockpit pkg registries update --force
```

### Installation Fails

```bash
# Validate package manifest
cockpit pkg validate ./package-name

# Check dependencies
cockpit pkg dependencies package-name

# Check AICockpit version
cockpit info
```

### Authentication Issues

For private registries using SSH:

```bash
# Ensure SSH key is configured
ssh-add ~/.ssh/id_rsa

# Test connection
ssh -T git@github.com

# Add registry with SSH URL
cockpit pkg registries add private git@github.com:org/packages.git
```

## Examples

### Example 1: Search and Install

```bash
# Search for documentation packages
$ cockpit pkg search --category documentation

Found 1 package:
  html-document-build (1.0.0)
  Builds HTML documentation from knowledge base articles

# Install the package
$ cockpit pkg install html-document-build

✓ Package installed successfully!
```

### Example 2: Add Custom Registry

```bash
# Add community registry
$ cockpit pkg registries add community https://github.com/community/packages

# Search in community registry
$ cockpit pkg search --source community

# Install from community
$ cockpit pkg install community-package --source community
```

### Example 3: Manage Multiple Registries

```bash
# List all registries
$ cockpit pkg registries list

Registered Registries:
  1. official (priority: 1) - enabled
  2. community (priority: 2) - enabled
  3. private (priority: 3) - disabled

# Disable community registry
$ cockpit pkg registries disable community

# Update official registry
$ cockpit pkg registries update official

# Re-enable community
$ cockpit pkg registries enable community
```

## Contributing to a Registry

When submitting packages or updates to a registry via pull request, the following rules apply and are enforced by CI:

### Rules

1. **One package per PR.** Each pull request must add or modify exactly one package. Bundling multiple packages in a single PR will cause CI validation to fail.

2. **Version bump required.** Every PR that touches a package must include a version increment in both the package's `cockpit-package.yml` manifest and the corresponding entry in `package-index.yaml`. PRs without a version bump are rejected.

3. **Bump must match PR commits.** The version increment must reflect the nature of the commits in the PR:
   - `PATCH` bump for bug fixes and non-functional changes
   - `MINOR` bump for new features, backward compatible
   - `MAJOR` bump for breaking changes

4. **CI validates these rules.** The GitHub Actions workflow (`.github/workflows/validate-packages.yml`) automatically checks all of the above on every PR. A PR cannot be merged until all validation steps pass.

### Contribution Workflow

```bash
# 1. Fork and clone the registry
git clone https://github.com/user/cockpit-packages.git
cd cockpit-packages

# 2. Create a feature branch for exactly one package
git checkout -b feature/my-package

# 3. Add or update the package
cp -r /path/to/my-package ./packages/my-package

# 4. Bump the version in cockpit-package.yml
#    (edit version field, e.g. 1.0.0 -> 1.1.0)

# 5. Update package-index.yaml with the same version bump
#    and any changed metadata

# 6. Commit using conventional commits
git add packages/my-package package-index.yaml
git commit -m "feat(packages): add my-package v1.1.0"

# 7. Open a PR — CI will validate the rules above automatically
git push origin feature/my-package
```

## See Also

- [Package Specification](./package-specification.md)
- [Package Manager](./package-manager.md)
- [Creating Packages](./creating-packages.md)
