# Workflow: Release Process

## Implementação como Skill

### Estrutura de Diretório
```
.devin/skills/
└── release/
    └── SKILL.md
```

### Conteúdo do SKILL.md
```markdown
---
name: release
description: Execute the complete release process for the project
argument-hint: "[version]"
allowed-tools:
  - read
  - grep
  - glob
  - exec
permissions:
  allow:
    - Exec(git)
    - Exec(npm)
  deny:
    - Write(.env*)
---

Execute the release process for version $1:

## Pre-release Checks
1. Verify we're on the main branch: `git branch --show-current`
2. Ensure working directory is clean: `git status`
3. Pull latest changes: `git pull origin main`
4. Run full test suite: `npm test`
5. Run linter: `npm run lint`

## Version Update
1. Update version in package.json to $1
2. Update CHANGELOG.md with new version entry
3. Commit version changes: `git commit -m "chore: bump version to $1"`

## Tag and Push
1. Create git tag: `git tag v$1`
2. Push tag to remote: `git push origin v$1`
3. Push commits: `git push origin main`

## Post-release
1. Trigger CI/CD pipeline if needed
2. Verify deployment to production
3. Announce release to team

Report the status of each step. If anything fails, stop and explain what needs to be fixed before proceeding.
```

## Como Usar
Comando: `/release 1.2.3`

## Alternativa: Orquestração com Subagentes

Para releases complexos, você pode orquestrar múltiplos subagentes:

```markdown
---
name: release-orchestrated
description: Orchestrate release process using subagents for parallel tasks
---

Execute the release process for version $1:

1. Use /test-runner subagent to run full test suite
2. Use /security-audit subagent to check for vulnerabilities
3. Use /changelog-generator skill to update CHANGELOG.md
4. Wait for all subagents to complete
5. If all checks pass, proceed with version update and tagging
6. Report final status
```
