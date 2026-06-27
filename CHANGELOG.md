# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Features
- calibrate Goose cockpit setup + Docs  (#55)
- improve changelog generation to work in PRs (#47)
- implement automated update checking and upgrade system (#46)
- implement comprehensive lock/unlock security system (#44)
- add KB assets sync and dynamic package commands
- proxy execution transparently and silence usage on package script errors
- implement package upgrades, safe deploy injections and gold rules
- comando cockpit rtk [on|off] para ativar/desativar prefixo global
- implement caveman command to toggle caveman mode
- implement KB Graph Search and Vault System

### Bug Fixes
- merge bot PRs without waiting for checks (#67)
- wait for PR checks and merge with admin bypass (#65)
- use admin merge to bypass signed commit requirement (#63)
- wait for PR mergeability instead of gh checks (#59)
- respect protected main with PR-based updates (#57)
- repair changelog and release pipelines (#56)
- correct syntax and structure of changelog and release pipelines
- use [skip ci] to prevent recursive workflow triggers
- prevent infinite loops in changelog and release workflows (#49)
- align adapters and documentation with official Devin CLI specs (#45)
- initialize cockpit folder using embedded assets

### Performance
No performance improvements

### Breaking Changes
No breaking changes

### Documentation
- update CHANGELOG.md for changes since v0.4.2 [skip ci] (#62)
- add CODEOWNERS and infrastructure protection guidance
- add contribution rules and CI validation workflow
- update registry docs to use packages/ directory
- add github cli 401 auth bypass troubleshooting guide (#43)
- add animated terminal quickstart gif
- add assets directory and hide broken gif link in README
- add Quick Installation guide for CLI users to README
- enrich PR template with versioning, evidence, and commands sections
- implement comprehensive architecture documentation trail
- restructure and cleanup documentation to reflect Canonical Compiler architecture

### Testing
- add coverage for global deployment marker injection

### Other Changes
- Merge pull request #38 from lleitep3/feat/package-command-execution-proxy
- Merge pull request #37 from lleitep3/feat/package-upgrades-and-safe-deploy
- Merge pull request #33 from lleitep3/chore/rename-module
- Merge pull request #32 from lleitep3/ci/workflows-improvements
- Merge branch 'main' into ci/workflows-improvements
- Merge pull request #31 from lleitep3/chore/gitignore-cleanup
- Merge branch 'main' into chore/gitignore-cleanup
- Merge branch 'main' into ci/workflows-improvements
- Merge pull request #29 from lleitep3/fix/setup-initialization
- Merge branch 'main' into fix/setup-initialization
- Merge pull request #28 from lleitep3/docs/providers-mapping
- Merge branch 'main' into docs/providers-mapping


## [0.1.0] - 2025-01-XX

### Initial Release
- Core CLI structure with Cobra
- Configuration management (YAML-based)
- Setup command for initial configuration
- Deploy command for asset deployment
- Vault security system with lock/unlock
- Knowledge Base (KB) integration
- Package management system
- Multiple AI provider support (Devin, Goose, Claude Code, GitHub Copilot)
- Metrics and logging infrastructure
- Internationalization (i18n) support