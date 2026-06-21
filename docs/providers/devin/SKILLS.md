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

## Exemplo de Skill

**Estrutura de Pastas:**
```text
skills/
└── web-scraper/
    └── SKILL.md
```

**Conteúdo do `SKILL.md`:**
```markdown
---
name: web-scraper
description: Use esta skill sempre que o usuário solicitar extração de dados estruturados ou tabelas de uma página web pública.
version: 1.0.0
---

# Web Scraper Skill

Você tem a capacidade de extrair dados de páginas web. Sempre siga estes passos:
1. Analise o DOM da URL usando `curl` com `cheerio` ou via `puppeteer` se a página usar Javascript.
2. Formate a saída final sempre em CSV ou JSON.
3. Não sobrecarregue o servidor alvo (adicione pequenos `sleeps`).
```

## Referências Oficiais
- [Skills Overview](https://docs.devin.ai/cli/extensibility/skills/overview)
- [Creating Skills](https://docs.devin.ai/cli/extensibility/skills/creating-skills)
