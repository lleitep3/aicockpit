---
name: tracking-audit
description: "Guides the AI agent on how to audit package tracking headers in the Cockpit workspace."
---

# Habilidade: tracking-audit

Esta habilidade ensina as IAs a auditarem o rastreamento de proveniência dos arquivos sincronizados pelo AICockpit.

## Quando Usar

Ative esta habilidade sempre que o desenvolvedor solicitar:

- Verificar se arquivos no workspace possuem tracking header correto.
- Auditar a proveniência de skills, rules, agents ou workflows instalados.
- Corrigir headers ausentes, desatualizados ou inconsistentes.
- Gerar relatório de conformidade dos trackings.

## Pré-requisitos

- Pacotes instalados em `~/.cockpit/packages/`.
- Arquivos canônicos em `~/.cockpit/skills`, `~/.cockpit/rules`, `~/.cockpit/agents` ou `~/.cockpit/workflows`.
- Conhecimento do formato de tracking header descrito em `docs/tracking.md`.

## Passo a Passo

1. **Coletar contexto**:
   - Liste os arquivos a auditar (`find`, `glob` ou `ls`).
   - Identifique o pacote de origem de cada arquivo.

2. **Validar presença do header**:
   - Leia as primeiras linhas de cada arquivo.
   - Verifique se a primeira linha casa com `// package:<name> version:<version> created:<ts> updated:<ts>`.

3. **Cruzar com o manifesto do pacote**:
   - Leia `~/.cockpit/packages/<package>/cockpit-package.yml`.
   - Compare `name`, `version`, `metadata.creation_date` e `metadata.last_modified`.

4. **Reportar inconsistências**:
   - Header ausente.
   - `package` divergente.
   - `version` divergente.
   - Timestamps fora do formato ISO-8601.

5. **Corrigir (se solicitado)**:
   - Use `internal/tracking.GenerateHeader(pkg)` para gerar o header correto.
   - Use `internal/tracking.InjectHeader(filePath, pkg)` para atualizar o arquivo.
   - Revalide o arquivo após a correção.

## Referências Úteis

- Documentação do tracking: `docs/tracking.md`
- Guia de auditoria: `ai-assets/knowledge-base/guides/tracking-audit.md`
