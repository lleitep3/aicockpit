# Internal - Core Logic
This directory contains the internal packages that form the engine of the AICockpit CLI. 
These are not exposed to external systems and represent the foundational logic of the application.

## Subdirectories:
- **assets/**: Contains default templates and static assets used during the initial setup of the cockpit environment.
- **config/**: Manages the configuration loading, saving, and validation (reading/writing `config.yaml`).
- **i18n/**: Handles internationalization and multi-language support for the CLI output.
- **kb/**: The Knowledge Base engine, responsible for indexing, searching, and managing project context and documentation.
- **logging/**: Manages the execution logs and metrics collection system.
- **packages/**: Handles the downloading, loading, and management of packages (the "plugin" system).
- **providers/**: The core "Canonical Compiler" logic. It translates standardized packages into specific configurations for AI providers (Goose, Devin, etc.).
- **vault/**: Manages secrets and sensitive configuration data.
- **version/**: Tracks the CLI versioning metadata.
