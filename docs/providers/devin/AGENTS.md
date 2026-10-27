# Devin Subagents

O Devin suporta a delegação de tarefas complexas para subagentes independentes. Subagentes permitem que o agente principal mantenha seu contexto focado enquanto descarrega tarefas secundárias, demoradas ou de pesquisa para instâncias isoladas.

**Referência Oficial:** [Devin Subagents](https://docs.devin.ai/cli/subagents)

## Como Funcionam

Subagentes no Devin operam em dois modos:

### Modos de Execução

| Modo | Comportamento |
|------|---------------|
| **Foreground** | Executa inline na sessão. O agente principal pausa e espera o subagente terminar antes de continuar. Você pode aprovar ou negar chamadas de ferramentas conforme elas aparecem. |
| **Background** | Executa em paralelo enquanto o agente principal continua trabalhando. O agente principal é automaticamente notificado quando o subagente conclui. Ferramentas não aprovadas são automaticamente negadas. |

### Perfis de Subagentes

Cada subagente executa com um perfil específico que determina suas capacidades:

| Perfil | Descrição | Acesso a Ferramentas |
|--------|-----------|---------------------|
| `subagent_explore` | Exploração e pesquisa de codebase apenas leitura | Ferramentas de codebase apenas leitura mais web search; não pode editar arquivos ou buscar URLs arbitrárias |
| `subagent_general` | Tarefas de propósito geral incluindo mudanças de código | Acesso completo a ferramentas (foreground) ou apenas ferramentas pré-aprovadas (background) |

O agente escolhe automaticamente o perfil apropriado baseado na tarefa. Você também pode pedir ao agente para usar um perfil específico por nome (ex: "review this code using the reviewer subagent").

### Capacidades

Subagentes no Devin são capazes de:
- Rodar em **background**, permitindo que o Devin principal continue interagindo com o usuário e realizando outras tarefas concorrentemente.
- Utilizar ferramentas idênticas ao agente principal (leitura de arquivos, terminal, navegação web).
- Reportar os resultados de forma resumida para o contexto do Devin principal quando concluírem a tarefa.
- **Trocar entre foreground e background** durante a execução (Ctrl+B para background, tecla 'f' no painel para foreground).
- **Ser cancelados e resumidos** posteriormente com um novo prompt.

## Permissões de Ferramentas

O comportamento de permissões depende se o subagente está rodando em foreground ou background:

- **Subagentes foreground** comportam-se como o agente principal — você é solicitado a aprovar ou negar chamadas de ferramentas como usual.
- **Subagentes background** herdam quaisquer permissões de ferramenta que você já tenha concedido durante a sessão atual. Qualquer ferramenta que não tenha sido pré-aprovada é automaticamente negada. Subagentes background não podem solicitar novas permissões.

## Subagentes Customizados

Além dos perfis built-in, você pode definir seus próprios perfis de subagente customizados. Subagentes customizados são definidos como arquivos `AGENT.md` dentro de um diretório nomeado sob `agents/`.

### Estrutura de Diretório

**Específico do Projeto:**
```
.devin/agents/
└── reviewer/
    └── AGENT.md
```

Também suportado:
```
.agents/agents/
└── reviewer/
    └── AGENT.md
```

**Global:**
```
# Linux/macOS
~/.config/devin/agents/
└── reviewer/
    └── AGENT.md

# Windows
%APPDATA%\devin\agents\
└── reviewer\
    └── AGENT.md
```

### Formato AGENT.md

Um arquivo `AGENT.md` usa o mesmo frontmatter YAML que skills, seguido pelo system prompt do subagente:

```markdown
---
name: reviewer
description: Reviews code changes for correctness and style
model: sonnet
allowed-tools:
  - read
  - grep
  - glob
  - exec
permissions:
  allow:
    - Exec(git diff)
    - Exec(git log)
  deny:
    - write
    - edit
max-nesting: 2
---

You are a code review subagent. Your job is to review code changes
thoroughly and report findings back to the parent agent.

Focus on:
1. Correctness — logic errors, edge cases, off-by-one mistakes
2. Security — potential vulnerabilities
3. Style — consistency with the rest of the codebase
4. Performance — obvious inefficiencies

Always cite specific file paths and line numbers in your findings.
```

### Campos do Frontmatter

| Campo | Tipo | Padrão | Descrição |
|-------|------|---------|-----------|
| `name` | string | nome do diretório | Identificador para o perfil (não pode conflitar com perfis built-in) |
| `description` | string | none | Mostrado ao agente ao selecionar um perfil |
| `model` | string | modelo de subagente padrão | Override do modelo usado por este subagente |
| `allowed-tools` | list | todas as ferramentas | Restringe quais ferramentas o subagente pode usar |
| `permissions` | object | herdar | Overrides de permissões (allow, deny, ask) |
| `max-nesting` | integer | none | Override da profundidade máxima de aninhamento, permitindo este subagente spawnar seus próprios subagentes |

## Aninhamento

Por padrão, subagentes não podem spawnar seus próprios subagentes — apenas o agente root pode. Ferramentas de subagente (`run_subagent` e `read_subagent`) são desabilitadas dentro de um subagente para evitar aninhamento ilimitado.

No entanto, **perfis de subagente customizados** podem optar por aninhamento aninhado definindo o campo `max-nesting` em seu frontmatter. Este valor override a profundidade máxima padrão, permitindo que subagentes spawnem filhos desde que a árvore permaneça dentro desse limite.

## Casos de Uso Comuns
- **Pesquisa Extensiva:** Delegar a leitura de uma grande documentação ou a análise de muitos arquivos de log.
- **Testes em Segundo Plano:** Rodar suítes de testes demoradas enquanto o Devin continua escrevendo código em outro módulo.
- **Refatorações Isoladas:** Pedir a um subagente para refatorar um arquivo específico baseado em um novo padrão, revisando o resultado apenas no final.
- **Code Review Especializado:** Usar um subagente customizado com perfil específico para revisões de código.
- **Análise de Arquitetura:** Delegar análise de arquitetura para subagente read-only enquanto mantém o contexto principal limpo.
