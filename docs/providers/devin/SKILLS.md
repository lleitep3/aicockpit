# Devin Skills

O Devin aderiu ao padrão aberto **Agent Skills**, tornando o formato de suas habilidades amplamente compatível com outros provedores. Skills no Devin são excelentes para codificar fluxos de trabalho repetitivos (como levantar um ambiente local) e podem ser referenciadas explicitamente em regras de projeto.

**Referência Oficial:** [Devin Skills Guide](https://docs.devin.ai/product-guides/skills)

## Diretórios de Skills

### Skills Globais (disponíveis em todos os projetos)
- **Linux/macOS:** `~/.config/devin/skills/` (padrão XDG)
- **Linux/macOS:** `~/.agents/skills/` (compatibilidade com padrão .agents)
- **Linux/macOS:** `~/.codeium/<channel>/skills/` (canal-dependente: windsurf, windsurf-next, windsurf-insiders)
- **Windows:** `%APPDATA%\devin\skills\` (tipicamente `C:\Users\<User>\AppData\Roaming\devin\skills\`)

**Nota:** Skills globais não são commitadas no git e ficam disponíveis em todos os projetos da sua máquina.

### Skills de Projeto (específicas do repositório)
- `.agents/skills/<name>/SKILL.md` (padrão .agents)
- `.devin/skills/<name>/SKILL.md` (padrão Devin)
- `.windsurf/skills/<name>/SKILL.md` (padrão Windsurf)

**Nota:** Skills de projeto são commitadas no git e compartilhadas com a equipe.

## Formato do Arquivo
Cada skill é representada por um diretório que contém um arquivo primário obrigatório chamado `SKILL.md`.

## Padrão e Schema (YAML Frontmatter)
O arquivo `SKILL.md` deve obrigatoriamente iniciar com um bloco de metadados YAML (frontmatter). Os campos obrigatórios são `name` e `description`.

### Campos Disponíveis no Frontmatter

| Campo | Tipo | Padrão | Descrição |
|-------|------|---------|-----------|
| `name` | string | nome do diretório | Nome da skill (usado para invocação via `/nome`) |
| `description` | string | none | Descrição mostrada no autocompletar |
| `argument-hint` | string | none | Dica após o comando (ex: `[arquivo]`) |
| `model` | string | modelo atual | Override do modelo (ex: `sonnet`, `opus`, `swe`) |
| `subagent` | boolean | `false` | Executar como subagente independente |
| `agent` | string | none | Perfil de subagente específico (ex: `subagent_explore`) |
| `allowed-tools` | list | todas | Restringir ferramentas (ex: `read`, `grep`, `glob`, `exec`) |
| `permissions` | object | herdado | Overrides de permissões (allow/deny/ask) |
| `triggers` | list | `[user, model]` | Como pode ser invocada (`user`, `model` ou ambos) |

### Exemplo Completo

```yaml
---
name: github-pr-reviewer
description: Analisa e faz o review de Pull Requests no GitHub
argument-hint: "[PR_NUMBER]"
model: sonnet
subagent: true
allowed-tools:
  - read
  - grep
  - glob
  - exec
permissions:
  allow:
    - Exec(git)
  deny:
    - Write(**)
triggers:
  - user
  - model
---

Conteúdo do prompt da skill...
```

## Conteúdo Dinâmico

O Devin suporta três tipos de conteúdo dinâmico no corpo do SKILL.md:

### 1. Argumentos
Interpole argumentos fornecidos pelo usuário:

```markdown
---
name: explain
argument-hint: "[arquivo]"
---

Explique o código em $1 em detalhes.
Todos os argumentos: $ARGUMENTS
```

- `$1`, `$2`, etc. — Argumentos posicionais individuais
- `$ARGUMENTS` — Todos os argumentos como uma string única

### 2. Inclusão de Arquivos
Inclua conteúdo de arquivos usando sintaxe `@` (relativo ao diretório da skill):

```markdown
---
name: style-check
---

Verifique o código contra nosso style guide:

@style-guide.md

Aplique estas regras ao arquivo atual.
```

### 3. Saída de Comando
Execute um comando shell e inclua sua saída:

```markdown
---
name: review-changes
---

Revise estas alterações:

!`git diff --staged`

Forneça feedback sobre qualidade e correção do código.
```

## Boas Práticas
1. **Checklists & Workflows:** Utilize skills para codificar fluxos de projeto repetíveis, como "como configurar o ambiente de desenvolvimento local" ou "como rodar o pipeline de deploy em staging".
2. **Rule Integration:** Referencie suas skills no `AGENTS.md` (ou `.devinrules`) da raiz do seu repositório, assim o Devin saberá explicitamente que deve buscar e invocar aquela skill sob determinadas condições.

## Anatomia de uma Skill Completa

Para skills robustas, a melhor prática é utilizar uma estrutura de diretório rica em vez de um único arquivo isolado. Isso permite encapsular dependências, templates e scripts, isolando a complexidade do contexto principal do agente.

**Estrutura de Diretório Recomendada:**
```text
skills/
└── github-pr-reviewer/           # Diretório raiz da skill
    ├── SKILL.md                  # Ponto de entrada (Obrigatório)
    ├── scripts/                  # Scripts e executáveis locais
    │   ├── fetch-pr.sh
    │   └── analyze-diff.py
    ├── examples/                 # Exemplos de uso e inputs
    │   └── payload.json
    ├── resources/                # Templates e assets auxiliares
    │   └── prompt_template.txt
    └── references/               # Documentação profunda para o LLM ler sob demanda
        └── github-api-specs.md
```

### Detalhamento dos Componentes

#### 1. `SKILL.md` (Obrigatório)
O cérebro da skill. Deve conter o *Frontmatter YAML* para indexação e o markdown com as instruções declarativas e o roteiro exato que o agente deve seguir. **Dica:** Mantenha as instruções sob 500 linhas; se precisar de mais, faça o agente ler arquivos na pasta `references/`.

**Nota:** Campos como `version` e `author` não são suportados oficialmente no frontmatter do Devin. Se precisar de metadados adicionais, coloque-os em um arquivo separado (ex: `manifest.yaml`) na pasta da skill.

```markdown
---
name: github-pr-reviewer
description: Analisa e faz o review de Pull Requests no GitHub, sugerindo melhorias de código e segurança. Acione quando o usuário pedir para revisar um PR.
---

# GitHub PR Reviewer

Você é responsável por conduzir a revisão de Pull Requests.

## Fluxo de Execução
1. Execute `./scripts/fetch-pr.sh <PR_NUMBER>` para baixar o diff do PR.
2. Formate o diff retornado.
3. Leia as regras de segurança em `references/github-api-specs.md` para basear seu review.
4. Execute o script `./scripts/analyze-diff.py` e forneça o diff pelo stdin.
5. Emita um relatório estruturado para o usuário.
```

#### 2. `scripts/` (Opcional, Fortemente Recomendado)
Onde residem utilitários (Python, Bash, Node, Go) que estendem o que o Devin consegue fazer, resolvendo tarefas muito complexas em poucas linhas de código determinístico, em vez de depender puramente do raciocínio do LLM.

#### 3. `references/` e `resources/`
- Use `references/` para manuais extensos. O Devin lerá esses arquivos dinamicamente via `view_file` apenas quando entrar no escopo desta skill.
- Use `resources/` para templates ou dados fixos, evitando poluir o `SKILL.md` com strings literais gigantescas.

## Referências Oficiais
- [Skills Overview](https://docs.devin.ai/cli/extensibility/skills/overview)
- [Creating Skills](https://docs.devin.ai/cli/extensibility/skills/creating-skills)
