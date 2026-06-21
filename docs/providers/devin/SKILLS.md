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
description: Extrai o conteúdo em texto de uma dada URL usando Python e BeautifulSoup. Use isso sempre que o usuário pedir para fazer scrape de um site ou resumir um artigo externo.
---

# Web Scraper Skill

Sempre que o usuário solicitar o scrape de uma página web, siga estes passos exatamente:

1. **Verify Environment:** Garanta que `beautifulsoup4` e `requests` estejam instalados. Caso contrário, instale-os usando `pip install beautifulsoup4 requests`.
2. **Execution:** Use um script Python para buscar a URL, extrair o texto da tag `<article>` (ou `<body>` se não existir article) e remover todas as marcações HTML.
3. **Output:** Retorne o texto puro para o usuário, prefixado com a frase: "Aqui está o conteúdo extraído:". Não retorne HTML puro.
```
