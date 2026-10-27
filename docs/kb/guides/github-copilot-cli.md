---
title: "GitHub Copilot CLI provider"
description: "How AICockpit deploys its instructions, skills, workflows, and agents to GitHub Copilot CLI"
tags: ["github-copilot", "copilot-cli", "providers", "skills", "agents"]
author: "AICockpit Team"
version: "1.0"
---

# GitHub Copilot CLI provider

O provider `github-copilot` do AICockpit integra os arquivos canônicos com o CLI oficial. A integração é global e grava em `~/.copilot`.

## Arquivos publicados

| Conteúdo canônico | Destino |
| --- | --- |
| Entrypoint e regras | `~/.copilot/copilot-instructions.md` |
| Skills | `~/.copilot/skills/<name>/SKILL.md` |
| Workflows | `~/.copilot/skills/cockpit-workflow-<name>/SKILL.md` |
| Agentes | `~/.copilot/agents/<name>.agent.md` |

O deploy troca somente o bloco `cockpit:global` no arquivo de instruções e mantém o texto manual fora desse bloco. Os scripts de skills acompanham cada `SKILL.md`. `AllowedTools` de agentes vira a lista `tools` do frontmatter; permissões canônicas e `MaxNesting` não são representáveis e causam erro.

Workflows canônicos são checklists publicados como skills invocáveis. Eles não são dynamic workflows do Copilot, que exigem uma extensão com código.

## Importar skills locais

O Cockpit pode importar uma skill existente do Copilot ou de outra pasta Agent Skills para sua biblioteca canônica:

```bash
cockpit skills import ~/.agents/skills --dry-run
cockpit skills import ~/.agents/skills --save-conflicts ~/.cockpit/skill-reviews/copilot-local
```

Passe uma pasta com `SKILL.md` para importar uma skill, ou uma pasta cujos subdiretórios imediatos contenham `SKILL.md` para importar uma biblioteca. O importador copia a árvore completa, preserva o frontmatter, arquivos auxiliares e permissões de scripts, e nunca sobrescreve skills existentes. Skills iguais são ignoradas; nomes iguais com conteúdo diferente são reportados como conflitos. `--save-conflicts <nova-pasta>` guarda cópias dos conflitos para revisão fora da biblioteca canônica; essa pasta não é distribuída por `cockpit deploy` e precisa ainda não existir. Use `--json` para receber um relatório estruturado. `cockpit deploy` distribui as skills importadas para os providers ativos.

## Limites do adapter

- Permissões canônicas `allow`, `deny` e `ask` não são compiladas. A feature `permissions` vem desabilitada e, se habilitada com política não vazia, o deploy retorna erro.
- Hooks, MCP, configurações e credenciais do Copilot CLI não são alterados pelo provider.
- O provider não autentica o Copilot. A autenticação continua sob controle do GitHub Copilot CLI.

## Deploy e validação

Rode `cockpit deploy` para publicar os arquivos nos providers ativos. Confira o que o Copilot CLI descobriu:

```bash
copilot instruction list
copilot skill list
```

Use `copilot --help` para as opções do CLI e [o guia detalhado do provider](../../providers/github-copilot/README.md) para os formatos suportados.
