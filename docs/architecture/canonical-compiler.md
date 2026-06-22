# Providers Architecture & Canonical Implementation

The `providers` package is responsible for integrating `aicockpit` with various AI agents/tools (Devin, Goose, Antigravity, etc.).

## Canonical Implementation (The "Cockpit" Way)

AICockpit utilizes a unified **Canonical Representation** for configuring AI assistants. This means you define your rules, skills, permissions, and workflows once, and the compiler automatically translates them into the specific format required by each provider.

The canonical mapping parses the project's `.cockpit/` directory and `ai-assets/` and transforms it:

- **Rules**: Standalone Markdown rules merged into the respective agent's global instructional file.
- **Skills**: Tools/Skills (with metadata) translated into MCP tools, subagent definitions, or CLI hooks.
- **Workflows**: Multi-step orchestrated plans mapped to subagents or prompt sequences.
- **Permissions**: Allowed/Denied commands mapped to safety config formats (e.g., `.goosehints` or `.devin/config.yaml`).

### Providers Feature Matrix

| Feature             | Antigravity (Gemini) | Devin (Cognition) | Goose (Block/AI) |
|---------------------|----------------------|-------------------|------------------|
| **Entrypoint**      | `~/.gemini/config/AGENTS.md` | `~/.devin/AGENTS.md` | `~/.config/goose/.goosehints` |
| **Rules**           | `AGENTS.md`          | `AGENTS.md`       | `.goosehints`    |
| **Skills**          | `skills/`            | `skills/`         | *MCP Extensions* |
| **Workflows**       | Subagents            | Linked Plans      | *Not Supported*  |
| **Permissions**     | Handled in CLI/RTK   | `config.yaml`     | *Through RTK*    |
| **Hooks**           | Yes                  | Yes               | Yes              |

## Compiler Interface

Each provider implements the `Compiler` interface (see `compiler.go`), which ensures that the deployment of these configurations is done natively for each AI system without loss of fidelity.

1. **`CompileEntrypoint()`**: Injects the global instructions (like using `rtk`).
2. **`CompileRules()`**: Appends project-specific rules to the provider's rule list.
3. **`CompileSkills()`**: Copies or links the required skills to the provider's skill directory.
4. **`CompilePermissions()`**: Updates the provider's sandboxing/safety settings.
5. **`CompileWorkflows()`**: Configures available sub-agents or tasks.

## Generating the Environment

When a user runs `cockpit deploy` or when a provider environment is initialized, the `Manager` loads the active configuration from `config.yaml` and invokes the compiler for every enabled provider, effectively creating a synchronized, unified behavior across all AI agents interacting with the project.
