# Devin Skills

O Devin aderiu ao padrão aberto **Agent Skills**, tornando o formato de suas habilidades amplamente compatível com outros provedores. Skills no Devin são excelentes para codificar fluxos de trabalho repetitivos (como levantar um ambiente local) e podem ser referenciadas explicitamente em regras de projeto.

**Referência Oficial:** [Devin Skills Guide](https://docs.devin.ai/product-guides/skills)

## Diretórios de Skills
- **Global (macOS/Linux):** `~/.config/devin/skills/`
- **Global (Windows):** `%APPDATA%\devin\skills\`
- **Project-level:** `.devin/skills/`

## Formato do Arquivo
Cada skill é representada por um diretório que contém um arquivo primário obrigatório chamado `SKILL.md`.

## Padrão e Schema (YAML Frontmatter)
O arquivo `SKILL.md` deve obrigatoriamente iniciar com um bloco de metadados YAML (frontmatter). Os campos obrigatórios são `name` e `description`.

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

```markdown
---
name: github-pr-reviewer
description: Analisa e faz o review de Pull Requests no GitHub, sugerindo melhorias de código e segurança. Acione quando o usuário pedir para revisar um PR.
version: 1.2.0
author: Equipe de Platform
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
