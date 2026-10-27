---
name: package-creation
description: Create, evolve, validate and publish Cockpit packages; use the repository workflow for core CLI changes.
---

# Package creation and evolution

Read the installed command help and package manifest before proposing commands.
Search `cockpit kb search` for prior decisions. Do not assume examples describe
features present in the user's installed version.

- New packages start in `~/.cockpit/local-registry/<name>`; preserve existing staging
  work. If installed, `cockpit cockpit-builder create/validate` supplies the scaffold.
- Core capabilities shared across packages (profiles, vault access, dispatch) belong
  in the Cockpit core. Service-specific adapters belong in packages.
- Plan the observable behavior, ownership, compatibility, dependencies, failure
  recovery and validation before implementation. Do not impose cloud planning on
  a local-only package.
- Use `cockpit config --help` to discover shared profile support. Public metadata
  and secret references use package namespaces; credentials remain in the vault.
  Read `kb/guides/package-profiles.md` for grants, explicit profiles and child-only
  environment injection. Namespace context is not an OS sandbox.
- CLI help must serve humans; structured output must serve automation. Noninteractive
  commands fail with an actionable message rather than waiting for hidden prompts.
  Browser login/MFA is a human step. Never bypass browser/admin restrictions.
- Test behavior, including failure/denial, retry after unknown remote outcomes,
  account selection, no secret output, and dependency absence. A schema validator
  or mocked browser is not proof of live integration. Claim only tested platforms.
- For local validation, copy the staged package and declared assets to the installed
  package/canonical directories, preserving unrelated files, then `cockpit deploy`.
  Do not edit provider-generated files. Avoid copying caches, credentials or runtimes.
- Publish to the actual index path, normally `packages/<name>`. Inspect registry CI:
  use one package per PR where required and align manifest/index versions. Core
  versions are handled by core CI; package version policy comes from its registry.
- Run package tests, registry validation and secret checks; inspect the diff before
  pushing a feature branch. Report local, simulated, live and CI evidence separately.

For the end-to-end checklist, read `workflows/cockpit-evolution.md` in the default
Cockpit assets. Save only verified reusable knowledge in the relevant KB context;
check for an existing entry before adding/updating.
