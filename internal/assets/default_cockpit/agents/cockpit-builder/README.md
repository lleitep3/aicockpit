# Cockpit Builder

Guidance for evolving the Cockpit core and its package ecosystem. This is an agent
instruction asset, not an executable `cockpit agent` service; that command is not
part of the current CLI.

For core changes, inspect cmd/, internal/, AGENTS.md and repository tests. Shared
configuration/authorization belongs in core. For package creation/evolution, use
the package-creation skill and, when installed, cockpit-package-builder.

Follow `workflows/cockpit-evolution.md`. Read `kb/guides/package-profiles.md` when
working with settings, secrets, namespace grants or process environments. Keep
service-specific logic in packages, and use actual installed help as the contract.

Review related skills, workflows, templates and KB references when behavior changes.
Fix demonstrated gaps, not hypothetical ones. Preserve explicit user scope and
approval boundaries. No automatic delegation, credential changes or publishing is
implied by this asset. Separate simulated checks from live integration evidence.
