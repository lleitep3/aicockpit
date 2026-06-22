---
title: "AICockpit Package Specification"
description: "Complete specification for AICockpit packages (cockpit-package.yml)"
tags: ["packages", "specification", "cockpit-package.yml", "installation", "dependencies"]
author: "AICockpit Team"
version: "1.0"
---

# AICockpit Package Specification

## Overview

AICockpit Packages are self-contained units that can contain:
- Knowledge Base documents
- Agents
- Skills
- CLI Modules
- Workflows
- Configurations

Each package is defined by a `cockpit-package.yml` file in its root directory.

## Package Structure

```
my-package/
├── cockpit-package.yml          # Package manifest
├── README.md                    # Package documentation
├── agents/                      # Optional: Agent implementations
│   ├── my-agent.go
│   └── my-agent_test.go
├── skills/                      # Optional: Skill implementations
│   ├── my-skill.go
│   └── my-skill_test.go
├── modules/                     # Optional: CLI modules
│   ├── cmd.go
│   └── cmd_test.go
├── kb/                          # Optional: Knowledge base documents
│   ├── guides/
│   │   └── guide.md
│   ├── examples/
│   │   └── example.md
│   └── troubleshooting/
│       └── issue.md
├── workflows/                   # Optional: Workflow definitions
│   └── workflow.yaml
├── config/                      # Optional: Default configurations
│   └── config.yaml
├── go.mod                       # Optional: Go module (if package has Go code)
├── go.sum                       # Optional: Go dependencies
├── package.json                 # Optional: Node.js module (if package has JS code)
├── package-lock.json            # Optional: Node.js dependencies
└── dependencies/                # Optional: External dependencies
    ├── go/
    │   └── requirements.txt
    └── node/
        └── requirements.txt
```

## cockpit-package.yml Specification

### Complete Example

```yaml
# Package metadata
name: "html-document-build"
version: "1.0.0"
description: "Builds HTML documentation from knowledge base articles"
author: "AICockpit Team"
license: "MIT"
homepage: "https://github.com/lleite/aicockpit"
repository: "https://github.com/lleite/aicockpit"

# Package type and category
type: "utility"  # utility, agent, skill, module, workflow, library
category: "documentation"

# Minimum required versions
requirements:
  cockpit: "0.2.0"
  go: "1.26"
  node: "22.0"  # Optional, only if package has Node.js code

# Package dependencies (other packages)
dependencies:
  - name: "kb-manager"
    version: ">=1.0.0"
    optional: false
  - name: "html-generator"
    version: ">=2.0.0"
    optional: true

# External dependencies (system libraries, npm packages, etc)
external-dependencies:
  go:
    - "github.com/some/package@v1.0.0"
  node:
    - "express@^4.18.0"
    - "ejs@^3.1.0"
  system:
    - "pandoc>=2.0"

# Features provided by this package
features:
  agents:
    - path: "agents/html-builder.go"
      name: "html-builder"
      description: "Builds HTML documentation"
  
  skills:
    - path: "skills/html-generator.go"
      name: "html-generator"
      description: "Generates HTML from markdown"
    - path: "skills/template-engine.go"
      name: "template-engine"
      description: "Processes HTML templates"
  
  modules:
    - path: "modules/cmd.go"
      name: "html-build"
      description: "CLI command for building HTML docs"
  
  kb:
    - path: "kb/guides/getting-started.md"
      type: "guide"
    - path: "kb/examples/basic-usage.md"
      type: "example"
    - path: "kb/troubleshooting/common-issues.md"
      type: "troubleshooting"
  
  workflows:
    - path: "workflows/build-and-deploy.yaml"
      name: "build-and-deploy"
      description: "Build and deploy documentation"

# Configuration
configuration:
  # Default configuration values
  defaults:
    output_dir: "articles"
    template: "default"
    theme: "light"
  
  # Configurable options
  options:
    - name: "output_dir"
      type: "string"
      description: "Directory where HTML articles are saved"
      default: "articles"
      required: false
    
    - name: "template"
      type: "string"
      description: "HTML template to use"
      default: "default"
      required: false
      options: ["default", "minimal", "full"]
    
    - name: "theme"
      type: "string"
      description: "Color theme for HTML output"
      default: "light"
      required: false
      options: ["light", "dark", "auto"]
    
    - name: "enable_toc"
      type: "boolean"
      description: "Enable table of contents"
      default: true
      required: false

# Installation configuration
installation:
  # Which providers support this package
  supported_providers:
    - devin
    - goose
  
  # Features to install per provider
  provider_features:
    devin:
      - agents
      - skills
      - modules
      - kb
      - workflows
    goose:
      - skills
      - modules
      - kb
  
  # Installation method
  method: "symlink"  # copy or symlink
  
  # Pre-installation hooks
  pre_install:
    - script: "scripts/pre-install.sh"
      description: "Validate system requirements"
  
  # Post-installation hooks
  post_install:
    - script: "scripts/post-install.sh"
      description: "Setup default configuration"

# Permissions and security
permissions:
  - "read:kb"
  - "write:articles"
  - "execute:workflows"

# Metadata
metadata:
  tags:
    - "documentation"
    - "html"
    - "generation"
  
  keywords:
    - "html"
    - "documentation"
    - "build"
    - "articles"
  
  maintainers:
    - name: "AICockpit Team"
      email: "team@aicockpit.dev"
  
  changelog: "CHANGELOG.md"
  
  # Package status
  status: "stable"  # alpha, beta, stable, deprecated
  
  # Support information
  support:
    issues: "https://github.com/lleite/aicockpit/issues"
    discussions: "https://github.com/lleite/aicockpit/discussions"
    documentation: "https://docs.aicockpit.dev"
```

## Field Descriptions

### Metadata

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Package name (lowercase, hyphenated) |
| version | string | Yes | Package version (semver) |
| description | string | Yes | Short package description |
| author | string | Yes | Package author name |
| license | string | Yes | License type (MIT, Apache-2.0, etc) |
| homepage | string | No | Package homepage URL |
| repository | string | No | Repository URL |

### Requirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| cockpit | string | Yes | Minimum AICockpit version |
| go | string | No | Minimum Go version (if package has Go code) |
| node | string | No | Minimum Node.js version (if package has JS code) |

### Dependencies

```yaml
dependencies:
  - name: "package-name"
    version: ">=1.0.0"  # semver range
    optional: false
```

### External Dependencies

```yaml
external-dependencies:
  go:
    - "github.com/user/package@v1.0.0"
  node:
    - "package-name@^1.0.0"
  system:
    - "pandoc>=2.0"
```

### Features

Each feature type has specific fields:

**Agents**:
```yaml
agents:
  - path: "agents/agent-name.go"
    name: "agent-name"
    description: "Agent description"
```

**Skills**:
```yaml
skills:
  - path: "skills/skill-name.go"
    name: "skill-name"
    description: "Skill description"
```

**Modules**:
```yaml
modules:
  - path: "modules/cmd.go"
    name: "command-name"
    description: "CLI command description"
```

**Knowledge Base**:
```yaml
kb:
  - path: "kb/guides/guide.md"
    type: "guide"  # guide, example, troubleshooting, reference
```

**Workflows**:
```yaml
workflows:
  - path: "workflows/workflow.yaml"
    name: "workflow-name"
    description: "Workflow description"
```

### Configuration

```yaml
configuration:
  defaults:
    key: "value"
  
  options:
    - name: "option-name"
      type: "string"  # string, boolean, integer, array, object
      description: "Option description"
      default: "default-value"
      required: false
      options: ["option1", "option2"]  # For enum-like options
```

### Installation

```yaml
installation:
  supported_providers:
    - devin
    - goose
  
  provider_features:
    devin:
      - agents
      - skills
      - modules
      - kb
    goose:
      - skills
      - modules
      - kb
  
  method: "symlink"  # copy or symlink
  
  pre_install:
    - script: "scripts/pre-install.sh"
      description: "Description"
  
  post_install:
    - script: "scripts/post-install.sh"
      description: "Description"
```

## Version Constraints

Package versions follow Semantic Versioning (semver):

```
MAJOR.MINOR.PATCH

Examples:
  1.0.0    - Exact version
  >=1.0.0  - Greater than or equal
  ^1.0.0   - Compatible with 1.x.x
  ~1.0.0   - Compatible with 1.0.x
  1.0.0 - 2.0.0  - Range
```

## Installation Tracking

When a package is installed, AICockpit creates an entry in `~/.cockpit/manifest.yaml`:

```yaml
packages:
  - name: "html-document-build"
    version: "1.0.0"
    source: "path/to/package"
    installed_at: "2026-06-20T14:00:00Z"
    installed_path: "~/.cockpit/packages/html-document-build"
    
    files:
      - source: "agents/html-builder.go"
        destination: "~/.cockpit/agents/html-builder.go"
        type: "symlink"  # or "copy"
        checksum: "sha256:abc123..."
      
      - source: "skills/html-generator.go"
        destination: "~/.cockpit/skills/html-generator.go"
        type: "symlink"
        checksum: "sha256:def456..."
    
    dependencies:
      - "kb-manager@1.0.0"
    
    configuration:
      output_dir: "articles"
      theme: "light"
```

## Package Validation

A valid package must have:

1. ✅ `cockpit-package.yml` in root directory
2. ✅ Valid YAML syntax
3. ✅ Required fields: name, version, description, author, license, cockpit version
4. ✅ Valid semver version
5. ✅ At least one feature (agents, skills, modules, kb, or workflows)
6. ✅ Valid feature paths (files must exist)
7. ✅ Valid provider support
8. ✅ Valid dependency versions

## Installation Process

When installing a package:

1. **Validate** package structure and manifest
2. **Check** AICockpit version compatibility
3. **Resolve** dependencies (install if needed)
4. **Prompt** user for configuration
5. **Create** package directory in `~/.cockpit/packages/<package-name>`
6. **Copy/Symlink** files to appropriate locations per provider
7. **Update** manifest with installation tracking
8. **Run** post-install hooks
9. **Verify** installation

## Uninstallation Process

When uninstalling a package:

1. **Check** if other packages depend on this package
2. **Warn** user about dependent packages
3. **Create** backup of package directory
4. **Remove** symlinks/copies from provider directories
5. **Update** manifest
6. **Run** pre-uninstall hooks
7. **Remove** package directory
8. **Verify** uninstallation

## Best Practices

### 1. Clear Dependencies

Always specify all dependencies:

```yaml
dependencies:
  - name: "kb-manager"
    version: ">=1.0.0"
    optional: false
```

### 2. Provider Support

Be explicit about provider support:

```yaml
installation:
  supported_providers:
    - devin
    - goose
  
  provider_features:
    devin:
      - agents
      - skills
      - modules
    goose:
      - skills
      - modules
```

### 3. Configuration Defaults

Provide sensible defaults:

```yaml
configuration:
  defaults:
    output_dir: "articles"
    theme: "light"
```

### 4. Documentation

Include comprehensive documentation:

```
- README.md with usage examples
- CHANGELOG.md with version history
- KB documents with guides and troubleshooting
```

### 5. Testing

Include tests for all components:

```
agents/agent_test.go
skills/skill_test.go
modules/cmd_test.go
```

## Examples

### Example 1: Simple Skill Package

```yaml
name: "text-processor"
version: "1.0.0"
description: "Text processing utilities"
author: "AICockpit Team"
license: "MIT"

requirements:
  cockpit: "0.2.0"

features:
  skills:
    - path: "skills/text-cleaner.go"
      name: "text-cleaner"
      description: "Cleans and normalizes text"
    - path: "skills/text-analyzer.go"
      name: "text-analyzer"
      description: "Analyzes text properties"

installation:
  supported_providers:
    - devin
    - goose
    - claude-code
    - github-copilot
  
  provider_features:
    devin:
      - skills
    goose:
      - skills
    claude-code:
      - skills
    github-copilot:
      - skills
```

### Example 2: Complex Package with Dependencies

```yaml
name: "html-document-build"
version: "1.0.0"
description: "Builds HTML documentation"
author: "AICockpit Team"
license: "MIT"

requirements:
  cockpit: "0.2.0"
  go: "1.26"

dependencies:
  - name: "kb-manager"
    version: ">=1.0.0"
    optional: false

external-dependencies:
  go:
    - "github.com/go-echarts/go-echarts@v2.2.4"

features:
  agents:
    - path: "agents/html-builder.go"
      name: "html-builder"
  
  skills:
    - path: "skills/html-generator.go"
      name: "html-generator"
  
  modules:
    - path: "modules/cmd.go"
      name: "html-build"
  
  kb:
    - path: "kb/guides/getting-started.md"
      type: "guide"

installation:
  supported_providers:
    - devin
    - goose
  
  provider_features:
    devin:
      - agents
      - skills
      - modules
      - kb
    goose:
      - skills
      - modules
      - kb
  
  method: "symlink"
  
  post_install:
    - script: "scripts/post-install.sh"
      description: "Setup default configuration"

configuration:
  defaults:
    output_dir: "articles"
    theme: "light"
  
  options:
    - name: "output_dir"
      type: "string"
      description: "Output directory for HTML files"
      default: "articles"
```

## Migration Guide

### From Old System to Packages

If you have existing components:

1. Create package directory structure
2. Create `cockpit-package.yml` with metadata
3. Move components to appropriate directories
4. Add tests and documentation
5. Validate package structure
6. Install package using new system

## Troubleshooting

### Package Validation Fails

Check:
- YAML syntax is valid
- All required fields are present
- Feature paths exist
- Version format is valid (semver)

### Installation Fails

Check:
- AICockpit version is compatible
- Dependencies are installed
- Provider is supported
- Disk space is available

### Dependency Resolution Fails

Check:
- Dependency package is installed
- Dependency version is compatible
- No circular dependencies

## See Also

- [Package Manager Guide](./package-manager.md)
- [Package Installation](./package-installation.md)
- [Creating Packages](./creating-packages.md)
